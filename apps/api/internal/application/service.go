package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
	"trafficos/apps/api/internal/events"
	"trafficos/apps/api/internal/observability"
	"trafficos/apps/api/internal/optimization"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/signals"
	"trafficos/apps/api/internal/simulation"
	"trafficos/apps/api/internal/storage"
	"trafficos/packages/demand"
	"trafficos/packages/network"
)

type Fault struct {
	Code, Message string
	Status        int
}

func (e *Fault) Error() string                     { return e.Message }
func Fail(code, message string, status int) *Fault { return &Fault{code, message, status} }

type Frame struct {
	DT       float64          `json:"dt"`
	Arrivals []demand.Arrival `json:"arrivals"`
}
type Telemetry struct {
	Epoch    string  `json:"epoch"`
	Session  string  `json:"session"`
	Sequence int64   `json:"sequence"`
	Frames   []Frame `json:"frames"`
}
type Command struct {
	PriorityMode   string        `json:"priorityMode"`
	Occupants      int           `json:"occupants"`
	Road           int           `json:"road"`
	Direction      string        `json:"direction"`
	MapID          string        `json:"mapId"`
	Profile        string        `json:"profile"`
	Mode           string        `json:"mode"`
	Paused         *bool         `json:"paused"`
	Speed          *float64      `json:"speed"`
	Priority       int           `json:"priority"`
	IntersectionID string        `json:"intersectionId"`
	Type           string        `json:"type"`
	Phase          signals.Phase `json:"phase"`
	Version        int           `json:"version"`
	Online         *bool         `json:"online"`
	LatencyMs      int           `json:"latencyMs"`
}
type Service struct {
	mu               sync.Mutex
	World            *simulation.World
	Store            storage.Store
	Hub              *realtime.Hub
	Events           []events.Event
	Decisions        []optimization.Decision
	LastTelemetry    time.Time
	TelemetrySession string
	Sequence         int64
	latencyMs        int
	lastCheckpoint   float64
	NativeBus        bool
}

func New(store storage.Store, hub *realtime.Hub) *Service {
	return &Service{World: simulation.New(42), Store: store, Hub: hub, Events: []events.Event{}, Decisions: []optimization.Decision{}}
}
func (s *Service) Snapshot() *simulation.World {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.World.Clone()
}
func (s *Service) Config() map[string]any { s.mu.Lock(); defer s.mu.Unlock(); return s.config() }
func (s *Service) config() map[string]any {
	return map[string]any{"priorityMode": s.World.PriorityMode, "profile": s.World.Profile, "speed": s.World.Speed, "paused": s.World.Paused, "epoch": s.World.Epoch, "mapId": s.World.Map.ID, "rows": len(s.World.Map.Rows), "columns": len(s.World.Map.Columns)}
}
func (s *Service) SimulatorOnline() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.LastTelemetry.IsZero() && time.Since(s.LastTelemetry) < 3*time.Second
}
func (s *Service) Subscribe() (chan []byte, []byte, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, done := s.Hub.Subscribe()
	raw, _ := json.Marshal(map[string]any{"type": "snapshot", "payload": s.World, "timestamp": time.Now()})
	return ch, raw, done
}
func (s *Service) History() ([]events.Event, []optimization.Decision) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]events.Event{}, s.Events...), append([]optimization.Decision{}, s.Decisions...)
}
func (s *Service) commit(ctx context.Context, next *simulation.World, key *storage.Idempotency) error {
	checkpoint := next.Epoch != s.World.Epoch || next.Time-s.lastCheckpoint >= 10 || s.lastCheckpoint == 0
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := s.Store.Commit(dbCtx, storage.Batch{World: next, Events: next.PendingEvents, Decisions: next.PendingDecisions, Checkpoint: checkpoint, Idempotency: key}); err != nil {
		return Fail("STORAGE_UNAVAILABLE", "Audit transaction failed; command was not applied", 503)
	}
	observability.DB.Observe(time.Since(start).Seconds())
	if checkpoint {
		s.lastCheckpoint = next.Time
	}
	s.World = next
	for _, e := range next.PendingEvents {
		s.Events = append(s.Events, e)
		observability.Events.WithLabelValues(e.Type).Inc()
		if e.Type == "signal.phase.changed" {
			observability.Transitions.Inc()
		}
		slog.Debug("domain event", "type", e.Type, "correlationId", e.CorrelationID, "id", e.ID)
	}
	for _, d := range next.PendingDecisions {
		s.Decisions = append(s.Decisions, d)
		observability.Decisions.Observe(d.ProcessingTimeMs / 1000)
	}
	if len(s.Events) > 1500 {
		s.Events = append([]events.Event{}, s.Events[len(s.Events)-1500:]...)
	}
	if len(s.Decisions) > 500 {
		s.Decisions = append([]optimization.Decision{}, s.Decisions[len(s.Decisions)-500:]...)
	}
	publishedEvents := next.PendingEvents
	next.PendingEvents = nil
	next.PendingDecisions = nil
	online := 0
	for _, i := range next.Intersections {
		if i.Status == "ONLINE" {
			online++
		}
	}
	observability.Online.Set(float64(online))
	s.Hub.Broadcast(map[string]any{"type": "snapshot", "payload": next, "timestamp": time.Now()})
	if !s.NativeBus && len(publishedEvents) > 0 {
		s.Hub.Broadcast(map[string]any{"type": "events", "payload": publishedEvents, "epoch": next.Epoch})
	}
	return nil
}
func (s *Service) Ingest(ctx context.Context, t Telemetry) (map[string]any, error) {
	if len(t.Session) < 1 || len(t.Session) > 80 || t.Sequence < 1 || len(t.Frames) > 10 || len(t.Frames) == 0 {
		return nil, Fail("INVALID_TELEMETRY", "Session, sequence and 1–10 frames required", 400)
	}
	for _, f := range t.Frames {
		if math.IsNaN(f.DT) || f.DT <= 0 || f.DT > 0.21 || len(f.Arrivals) > 100 {
			return nil, Fail("INVALID_TELEMETRY", "Invalid frame duration or arrivals", 400)
		}
		for _, a := range f.Arrivals {
			if !simulation.ValidVehicle(a.Type, a.Occupants) {
				return nil, Fail("INVALID_TELEMETRY", "Use CAR/SUV/BUS and occupants within 1–4/6/60 (or 0 for default)", 400)
			}
			if a.Road < 0 || (a.Direction != "N" && a.Direction != "S" && a.Direction != "E" && a.Direction != "W") {
				return nil, Fail("INVALID_TELEMETRY", "Invalid entry lane", 400)
			}
		}
	}
	s.mu.Lock()
	delay := s.latencyMs
	s.mu.Unlock()
	if delay > 0 {
		select {
		case <-time.After(time.Duration(delay) * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.TelemetrySession == t.Session {
		if t.Sequence <= s.Sequence {
			return s.config(), nil
		}
		if t.Sequence != s.Sequence+1 {
			return nil, Fail("SEQUENCE_GAP", "Retry the missing telemetry sequence", 409)
		}
	} else if s.TelemetrySession != "" && time.Since(s.LastTelemetry) < 3*time.Second {
		return nil, Fail("SIMULATOR_ACTIVE", "Another simulator owns the telemetry lease", 409)
	}
	next := s.World.Clone()
	if t.Epoch == "" || t.Epoch == next.Epoch {
		for _, f := range t.Frames {
			for _, a := range f.Arrivals {
				if next.Map.Entry(a.Road, a.Direction) == "" {
					return nil, Fail("INVALID_TELEMETRY", "Entry lane is not on this map", 400)
				}
			}
			next.Advance(f.DT, f.Arrivals)
		}
	}
	if err := s.commit(ctx, next, nil); err != nil {
		return nil, err
	}
	s.Sequence = t.Sequence
	s.TelemetrySession = t.Session
	s.LastTelemetry = time.Now()
	return s.config(), nil
}
func (s *Service) Execute(ctx context.Context, kind, key string, raw []byte, corr string) (json.RawMessage, int, error) {
	var c Command
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, 0, Fail("INVALID_JSON", "Expected a JSON object", 400)
	}
	if len(key) > 128 {
		return nil, 0, Fail("INVALID_IDEMPOTENCY_KEY", "Key must be at most 128 characters", 400)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := sha256.Sum256(append([]byte(kind+":"), raw...))
	fingerprint := hex.EncodeToString(hash[:])
	scopedKey := kind + ":" + key
	if key != "" {
		old, err := s.Store.Lookup(ctx, scopedKey)
		if err != nil {
			return nil, 0, Fail("STORAGE_UNAVAILABLE", "Cannot verify idempotency key", 503)
		}
		if old != nil {
			if old.Fingerprint != fingerprint {
				return nil, 0, Fail("IDEMPOTENCY_CONFLICT", "Key already used with a different payload", 409)
			}
			return old.Response, old.Status, nil
		}
	}
	next := s.World.Clone()
	var result any
	status := 200
	var commandErr error
	switch kind {
	case "profile":
		if !simulation.ValidProfile(c.Profile) {
			return nil, 0, Fail("INVALID_PROFILE", "Use LOW, NORMAL, RUSH_HOUR or GRIDLOCK", 400)
		}
		next.Profile = c.Profile
		next.Emit("simulation.profile.changed", "simulation", corr, map[string]string{"profile": c.Profile})
	case "control":
		if c.PriorityMode != "" {
			if c.PriorityMode != "VEHICLES" && c.PriorityMode != "PEOPLE" {
				return nil, 0, Fail("INVALID_PRIORITY_MODE", "Use VEHICLES or PEOPLE", 400)
			}
			next.PriorityMode = c.PriorityMode
			for j := range next.Intersections {
				next.Intersections[j].LastCalculation = -2
				next.Intersections[j].LastDecision = nil
			}
			next.Emit("simulation.priority.changed", "simulation", corr, map[string]string{"priorityMode": c.PriorityMode})
		}
		if c.Mode != "" {
			if c.Mode != "FIXED" && c.Mode != "ADAPTIVE" {
				return nil, 0, Fail("INVALID_MODE", "Use FIXED or ADAPTIVE", 400)
			}
			next.Mode = c.Mode
		}
		if c.Speed != nil {
			if *c.Speed != 0.5 && *c.Speed != 1 && *c.Speed != 2 && *c.Speed != 5 && *c.Speed != 10 {
				return nil, 0, Fail("INVALID_SPEED", "Use 0.5, 1, 2, 5 or 10", 400)
			}
			next.Speed = *c.Speed
		}
		if c.Paused != nil {
			next.Paused = *c.Paused
		}
		next.Emit("simulation.control.changed", "simulation", corr, c)
	case "reset":
		next = simulation.NewMap(42, s.World.Map.ID)
		next.PriorityMode = s.World.PriorityMode
		next.Profile, next.Mode, next.Speed, next.Paused = s.World.Profile, s.World.Mode, s.World.Speed, s.World.Paused
		next.Emit("simulation.reset", "simulation", corr, map[string]int{"seed": 42})
	case "map":
		if !network.Valid(c.MapID) {
			return nil, 0, Fail("INVALID_MAP", "Choose a map ID from GET /api/v1/maps", 400)
		}
		next = simulation.NewMap(42, c.MapID)
		next.PriorityMode = s.World.PriorityMode
		next.Profile = s.World.Profile
		next.Speed = s.World.Speed
		next.Mode = s.World.Mode
		next.Paused = s.World.Paused
		next.Emit("simulation.map.changed", "simulation", corr, map[string]string{"mapId": c.MapID, "previousMap": s.World.Map.ID})
	case "vehicle":
		if (c.Type != "CAR" && c.Type != "SUV" && c.Type != "BUS") || !simulation.ValidVehicle(c.Type, c.Occupants) || c.Occupants < 1 {
			return nil, 0, Fail("INVALID_VEHICLE", "Use CAR/SUV/BUS with 1–4/6/60 occupants", 400)
		}
		if next.Map.Entry(c.Road, c.Direction) == "" || (c.Direction != "N" && c.Direction != "S" && c.Direction != "E" && c.Direction != "W") {
			return nil, 0, Fail("INVALID_ENTRY", "Entry not on selected map", 400)
		}
		v := next.SpawnVehicle(c.Type, c.Occupants, c.Road, c.Direction)
		if v == nil {
			return nil, 0, Fail("ENTRY_OCCUPIED", "Entry occupied; wait for space and try again", 409)
		}
		result = v
		status = 201
	case "emergency":
		if key == "" {
			return nil, 0, Fail("IDEMPOTENCY_REQUIRED", "Provide Idempotency-Key", 400)
		}
		if c.Priority == 0 {
			c.Priority = 1
		}
		if c.Priority < 1 || c.Priority > 3 {
			return nil, 0, Fail("INVALID_PRIORITY", "Priority must be 1–3", 400)
		}
		v, err := next.SpawnEmergency(c.Priority, corr)
		if err != nil {
			return nil, 0, Fail("ENTRY_OCCUPIED", err.Error(), 409)
		}
		result = v
		status = 201
	case "incident":
		if next.Intersection(c.IntersectionID) == nil {
			return nil, 0, Fail("NOT_FOUND", "Intersection does not exist", 404)
		}
		if c.Type != "ROADWORK" && c.Type != "COLLISION" {
			return nil, 0, Fail("INVALID_INCIDENT", "Use ROADWORK or COLLISION", 400)
		}
		i := simulation.Incident{ID: next.ID("incident"), IntersectionID: c.IntersectionID, Type: c.Type, Active: true, CreatedAt: next.Time}
		next.Incidents = append(next.Incidents, i)
		next.Emit("incident.created", "incidents", corr, i)
		result = i
		status = 201
	case "resolve":
		found := false
		for idx := range next.Incidents {
			if next.Incidents[idx].ID == c.IntersectionID {
				next.Incidents[idx].Active = false
				next.Emit("incident.resolved", "incidents", corr, next.Incidents[idx])
				found = true
			}
		}
		if !found {
			return nil, 0, Fail("NOT_FOUND", "Incident does not exist", 404)
		}
	case "signal":
		i := next.Intersection(c.IntersectionID)
		if i == nil {
			return nil, 0, Fail("NOT_FOUND", "Intersection does not exist", 404)
		}
		if c.Version != i.Version {
			return nil, 0, Fail("VERSION_CONFLICT", "Signal version changed; refresh before retrying", 409)
		}
		if err := next.Transition(i, c.Phase, corr); err != nil {
			commandErr = Fail("INVALID_SIGNAL_TRANSITION", err.Error(), 422)
		}
		result = i
	case "chaos":
		if c.Type == "optimizer" {
			if c.Online == nil {
				return nil, 0, Fail("INVALID_CHAOS", "online is required", 400)
			}
			next.SetOptimizer(*c.Online, corr)
		} else if c.Type == "intersection" {
			i := next.Intersection(c.IntersectionID)
			if i == nil || c.Online == nil {
				return nil, 0, Fail("INVALID_CHAOS", "Valid intersection and online are required", 400)
			}
			i.Status = "OFFLINE"
			if *c.Online {
				i.Status = "ONLINE"
			}
			next.Emit("system.node.status.changed", "chaos", corr, map[string]string{"intersectionId": i.ID, "status": i.Status})
		} else if c.Type == "latency" {
			if c.LatencyMs < 0 || c.LatencyMs > 1000 {
				return nil, 0, Fail("INVALID_LATENCY", "latencyMs must be 0–1000", 400)
			}
			next.Emit("system.telemetry.latency.changed", "chaos", corr, map[string]int{"latencyMs": c.LatencyMs})
		} else {
			return nil, 0, Fail("INVALID_CHAOS", "Use optimizer, intersection or latency", 400)
		}
	default:
		return nil, 0, Fail("NOT_FOUND", "Command not found", 404)
	}
	if result == nil {
		result = map[string]any{"ok": true, "correlationId": corr}
	}
	response, err := json.Marshal(result)
	if err != nil {
		return nil, 0, err
	}
	var idem *storage.Idempotency
	if key != "" && commandErr == nil {
		idem = &storage.Idempotency{Key: scopedKey, Fingerprint: fingerprint, Status: status, Response: response}
	}
	if err = s.commit(ctx, next, idem); err != nil {
		return nil, 0, err
	}
	if kind == "chaos" && c.Type == "latency" {
		s.latencyMs = c.LatencyMs
	}
	if kind == "reset" {
		s.lastCheckpoint = 0
	}
	if commandErr != nil {
		return nil, 0, commandErr
	}
	return response, status, nil
}
func (s *Service) Decision(id string) (optimization.Decision, error) {
	_, ds := s.History()
	for _, d := range ds {
		if d.ID == id {
			return d, nil
		}
	}
	return optimization.Decision{}, fmt.Errorf("decision not in recent history")
}
