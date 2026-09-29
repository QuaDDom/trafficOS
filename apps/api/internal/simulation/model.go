package simulation

import (
	"crypto/rand"
	"fmt"
	"time"
	"trafficos/apps/api/internal/events"
	"trafficos/apps/api/internal/optimization"
	"trafficos/apps/api/internal/signals"
	"trafficos/packages/network"
)

type Lane struct {
	People       int     `json:"people"`
	QueuedPeople int     `json:"queuedPeople"`
	PersonWait   float64 `json:"personWait"`
	BusCount     int     `json:"busCount"`
	ID           string  `json:"id"`
	Direction    string  `json:"direction"`
	Capacity     int     `json:"capacity"`
	VehicleCount int     `json:"vehicleCount"`
	QueueLength  int     `json:"queueLength"`
	AverageWait  float64 `json:"averageWait"`
	AverageSpeed float64 `json:"averageSpeed"`
}
type Intersection struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	X               float64                `json:"x"`
	Z               float64                `json:"z"`
	Status          string                 `json:"status"`
	ControllerMode  string                 `json:"controllerMode"`
	Phase           signals.Phase          `json:"phase"`
	PhaseStartedAt  float64                `json:"phaseStartedAt"`
	PhaseAge        float64                `json:"phaseAge"`
	Version         int                    `json:"version"`
	Lanes           []Lane                 `json:"lanes"`
	LastDecision    *optimization.Decision `json:"lastDecision,omitempty"`
	Heartbeat       float64                `json:"heartbeat"`
	LastCalculation float64                `json:"-"`
}
type Vehicle struct {
	Occupants     int       `json:"occupants"`
	Length        float64   `json:"length"`
	ApproachWait  float64   `json:"-"`
	Heading       float64   `json:"heading"`
	Stops         []float64 `json:"stops"`
	PathLength    float64   `json:"pathLength"`
	Maneuver      string    `json:"maneuver"`
	path          []routePoint
	clears        []float64
	approaches    []string
	turns         []string
	exitDirection string
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	Direction     string   `json:"direction"`
	Road          int      `json:"road"`
	Position      float64  `json:"position"`
	X             float64  `json:"x"`
	Z             float64  `json:"z"`
	Speed         float64  `json:"speed"`
	State         string   `json:"state"`
	WaitReason    string   `json:"waitReason,omitempty"`
	Route         []string `json:"route"`
	Next          int      `json:"next"`
	Wait          float64  `json:"wait"`
	Born          float64  `json:"born"`
	StopCount     int      `json:"stopCount"`
}
type Reservation struct {
	IntersectionID string  `json:"intersectionId"`
	Axis           string  `json:"axis"`
	StartsAt       float64 `json:"startsAt"`
	EndsAt         float64 `json:"endsAt"`
	Released       bool    `json:"released"`
}
type Corridor struct {
	ID           string        `json:"id"`
	VehicleID    string        `json:"vehicleId"`
	Priority     int           `json:"priority"`
	CreatedAt    float64       `json:"createdAt"`
	Status       string        `json:"status"`
	Reservations []Reservation `json:"reservations"`
}
type Incident struct {
	ID             string  `json:"id"`
	IntersectionID string  `json:"intersectionId"`
	Type           string  `json:"type"`
	Active         bool    `json:"active"`
	CreatedAt      float64 `json:"createdAt"`
}
type Metrics struct {
	PeopleCreated       int     `json:"peopleCreated"`
	PeopleCompleted     int     `json:"peopleCompleted"`
	ActivePeople        int     `json:"activePeople"`
	TotalPersonWait     float64 `json:"totalPersonWait"`
	AveragePersonWait   float64 `json:"averagePersonWait"`
	Cars                int     `json:"cars"`
	SUVs                int     `json:"suvs"`
	Buses               int     `json:"buses"`
	VehiclesCreated     int     `json:"vehiclesCreated"`
	VehiclesCompleted   int     `json:"vehiclesCompleted"`
	ActiveVehicles      int     `json:"activeVehicles"`
	QueueSize           int     `json:"queueSize"`
	MaxQueue            int     `json:"maxQueue"`
	AverageWait         float64 `json:"averageWait"`
	AverageStopDuration float64 `json:"averageStopDuration"`
	AverageSpeed        float64 `json:"averageSpeed"`
	TotalWait           float64 `json:"totalWait"`
	TotalStops          int     `json:"totalStops"`
	EventsTotal         int     `json:"eventsTotal"`
	DroppedArrivals     int     `json:"droppedArrivals"`
}
type World struct {
	PriorityMode     string                  `json:"priorityMode"`
	Map              network.Map             `json:"map"`
	Epoch            string                  `json:"epoch"`
	Time             float64                 `json:"time"`
	Profile          string                  `json:"profile"`
	Mode             string                  `json:"mode"`
	Speed            float64                 `json:"speed"`
	Paused           bool                    `json:"paused"`
	OptimizerOnline  bool                    `json:"optimizerOnline"`
	Intersections    []Intersection          `json:"intersections"`
	Vehicles         []Vehicle               `json:"vehicles"`
	Corridors        []Corridor              `json:"corridors"`
	Incidents        []Incident              `json:"incidents"`
	Metrics          Metrics                 `json:"metrics"`
	Seed             uint64                  `json:"seed"`
	PendingEvents    []events.Event          `json:"-"`
	PendingDecisions []optimization.Decision `json:"-"`
	sequence         int
}

func New(seed uint64) *World { return NewMap(seed, "centro") }
func NewMap(seed uint64, mapID string) *World {
	w := &World{Map: network.Get(mapID), Epoch: rand.Text(), Profile: "NORMAL", PriorityMode: "VEHICLES", Mode: "ADAPTIVE", Speed: 1, OptimizerOnline: true, Seed: seed, Vehicles: []Vehicle{}, Corridors: []Corridor{}, Incidents: []Incident{}}
	for index, n := range w.Map.Nodes {
		p := signals.NSGreen
		if (index/len(w.Map.Columns)+index%len(w.Map.Columns))%2 == 0 {
			p = signals.EWGreen
		}
		i := Intersection{ID: n.ID, Name: n.Name, X: n.X, Z: n.Z, Status: "ONLINE", ControllerMode: "ADAPTIVE", Phase: p, Version: 1}
		for _, d := range []string{"N", "S", "E", "W"} {
			i.Lanes = append(i.Lanes, Lane{ID: n.ID + "-" + d, Direction: d, Capacity: 16})
		}
		w.Intersections = append(w.Intersections, i)
	}
	return w
}
func (w *World) ID(prefix string) string {
	w.sequence++
	return fmt.Sprintf("%s_%s_%07d", prefix, w.Epoch, w.sequence)
}
func (w *World) Emit(kind, source, corr string, payload any) {
	w.Metrics.EventsTotal++
	w.PendingEvents = append(w.PendingEvents, events.Event{ID: w.ID("evt"), Type: kind, Timestamp: time.Now().UTC(), Source: source, CorrelationID: corr, Payload: payload})
}
func (w *World) Intersection(id string) *Intersection {
	for i := range w.Intersections {
		if w.Intersections[i].ID == id {
			return &w.Intersections[i]
		}
	}
	return nil
}
func ValidProfile(p string) bool {
	return p == "LOW" || p == "NORMAL" || p == "RUSH_HOUR" || p == "GRIDLOCK"
}

// Xorshift64 uses separate deterministic arrival entropy, never wall-clock input.
func (w *World) random() float64 {
	if w.Seed == 0 {
		w.Seed = 42
	}
	w.Seed ^= w.Seed << 13
	w.Seed ^= w.Seed >> 7
	w.Seed ^= w.Seed << 17
	return float64(w.Seed>>11) / float64(uint64(1)<<53)
}
