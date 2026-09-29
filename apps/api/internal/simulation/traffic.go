package simulation

import (
	"math"
	"time"
	"trafficos/apps/api/internal/optimization"
	"trafficos/apps/api/internal/signals"
	"trafficos/packages/demand"
)

func axis(dir string) string {
	if dir == "N" || dir == "S" {
		return "NS"
	}
	return "EW"
}
func (w *World) addVehicle(a demand.Arrival, kind string) *Vehicle {
	route, out := w.plan(a, kind)
	if len(route) == 0 {
		w.Metrics.DroppedArrivals++
		return nil
	}
	people := a.Occupants
	if people == 0 {
		people = DefaultOccupants(kind)
	}
	v := Vehicle{Occupants: people, Length: VehicleLength(kind), Type: kind, Direction: a.Direction, Road: a.Road, Position: -66, State: "MOVING", Route: route, Born: w.Time}
	w.buildPath(&v, out)
	for _, other := range w.Vehicles {
		if math.Hypot(v.X-other.X, v.Z-other.Z) < headway(v, other) {
			w.Metrics.DroppedArrivals++
			return nil
		}
	}
	if len(w.Vehicles) >= 600 {
		w.Metrics.DroppedArrivals++
		return nil
	}
	v.ID = w.ID("veh")
	w.Vehicles = append(w.Vehicles, v)
	w.Metrics.VehiclesCreated++
	w.Metrics.PeopleCreated += v.Occupants
	w.Metrics.ActiveVehicles = len(w.Vehicles)
	w.Metrics.ActivePeople += v.Occupants
	switch kind {
	case "CAR":
		w.Metrics.Cars++
	case "SUV":
		w.Metrics.SUVs++
	case "BUS":
		w.Metrics.Buses++
	}
	w.Emit("traffic.vehicle.entered", "traffic-engine", v.ID, map[string]any{"vehicleId": v.ID, "type": kind, "occupants": v.Occupants, "road": a.Road, "direction": a.Direction, "route": route})
	return &w.Vehicles[len(w.Vehicles)-1]
}

func (w *World) Advance(dt float64, arrivals []demand.Arrival) {
	if w.Paused {
		return
	}
	w.Time += dt
	for _, a := range arrivals {
		kind := a.Type
		if kind == "" {
			kind = "CAR"
		}
		w.addVehicle(a, kind)
	}
	w.reservations()
	w.measureLanes()
	for idx := range w.Intersections {
		i := &w.Intersections[idx]
		i.PhaseAge = w.Time - i.PhaseStartedAt
		i.Heartbeat = w.Time
		desired := w.desiredAxis(i.ID)
		mode := w.Mode
		if !w.OptimizerOnline || i.Status == "OFFLINE" {
			mode = "SAFE_MODE"
		} else if desired != "" {
			mode = "EMERGENCY_OVERRIDE"
		}
		if mode != i.ControllerMode {
			i.ControllerMode = mode
			w.Emit("intersection.status.changed", "signal-controller", w.ID("corr"), map[string]any{"intersectionId": i.ID, "mode": mode})
		}
		next := false
		corr := w.ID("corr")
		if !signals.IsGreen(i.Phase) {
			next = i.PhaseAge+1e-8 >= signals.Minimum(i.Phase)
		} else if mode == "EMERGENCY_OVERRIDE" {
			next = (signals.Axis(i.Phase) != desired && i.PhaseAge >= signals.MinGreen) || i.PhaseAge >= signals.MaxGreen
		} else if mode == "FIXED" || mode == "SAFE_MODE" {
			next = i.PhaseAge >= 25
		} else if w.Time-i.LastCalculation >= 2 {
			i.LastCalculation = w.Time
			in := optimization.Input{IntersectionID: i.ID, Phase: i.Phase, PhaseDuration: i.PhaseAge, PriorityMode: w.PriorityMode}
			for _, l := range i.Lanes {
				if axis(l.Direction) == "NS" {
					in.NSPeople += l.People
					in.NSQueuedPeople += l.QueuedPeople
					in.NSPersonWait += l.PersonWait
					in.NSQueue += l.QueueLength
					in.NSWait += l.AverageWait
					in.IncomingNS += l.VehicleCount
				} else {
					in.EWPeople += l.People
					in.EWQueuedPeople += l.QueuedPeople
					in.EWPersonWait += l.PersonWait
					in.EWQueue += l.QueueLength
					in.EWWait += l.AverageWait
					in.IncomingEW += l.VehicleCount
				}
			}
			if in.NSQueue+in.EWQueue >= 4 {
				w.Emit("traffic.queue.threshold_exceeded", "traffic-engine", corr, map[string]any{"intersectionId": i.ID, "queue": in.NSQueue + in.EWQueue})
			}
			w.Emit("optimizer.calculation.started", "optimizer", corr, in)
			start := time.Now()
			d := optimization.Decide(in)
			d.ProcessingTimeMs = float64(time.Since(start).Nanoseconds()) / 1e6
			d.ID = w.ID("dec")
			d.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			d.CorrelationID = corr
			i.LastDecision = &d
			w.PendingDecisions = append(w.PendingDecisions, d)
			w.Emit("optimizer.decision.created", "optimizer", corr, d)
			next = d.Action == "NEXT_PHASE" || d.Action == "SHORTEN_PHASE"
		}
		if signals.IsGreen(i.Phase) && i.PhaseAge >= signals.MaxGreen {
			next = true
		}
		if next {
			w.Transition(i, signals.Next(i.Phase), corr)
		}
	}
	w.moveVehicles(dt)
	kept := w.Vehicles[:0]
	for _, v := range w.Vehicles {
		if v.Position >= v.PathLength {
			w.Metrics.VehiclesCompleted++
			w.Metrics.PeopleCompleted += v.Occupants
			w.Emit("traffic.vehicle.left", "traffic-engine", v.ID, map[string]any{"vehicleId": v.ID, "wait": v.Wait})
		} else {
			kept = append(kept, v)
		}
	}
	w.Vehicles = kept
	w.measureLanes()
	w.Metrics.ActiveVehicles = len(w.Vehicles)
	w.Metrics.QueueSize = 0
	w.Metrics.ActivePeople = 0
	w.Metrics.Cars, w.Metrics.SUVs, w.Metrics.Buses = 0, 0, 0
	speed := 0.0
	for _, v := range w.Vehicles {
		if v.State == "WAITING" {
			w.Metrics.QueueSize++
		}
		speed += v.Speed
		w.Metrics.ActivePeople += v.Occupants
		switch v.Type {
		case "CAR":
			w.Metrics.Cars++
		case "SUV":
			w.Metrics.SUVs++
		case "BUS":
			w.Metrics.Buses++
		}
	}
	w.Metrics.MaxQueue = max(w.Metrics.MaxQueue, w.Metrics.QueueSize)
	w.Metrics.AveragePersonWait = w.Metrics.TotalPersonWait / float64(max(1, w.Metrics.PeopleCreated))
	w.Metrics.AverageWait = w.Metrics.TotalWait / float64(max(1, w.Metrics.VehiclesCreated))
	w.Metrics.AverageStopDuration = w.Metrics.TotalWait / float64(max(1, w.Metrics.TotalStops))
	w.Metrics.AverageSpeed = speed / float64(max(1, len(w.Vehicles))) * 3.6
}

func (w *World) Transition(i *Intersection, to signals.Phase, corr string) error {
	w.Emit("signal.phase.change.requested", "signal-controller", corr, map[string]any{"intersectionId": i.ID, "requested": to, "current": i.Phase})
	if err := signals.Validate(i.Phase, to, i.PhaseAge); err != nil {
		w.Emit("signal.transition.rejected", "signal-controller", corr, map[string]any{"intersectionId": i.ID, "requested": to, "current": i.Phase, "reason": err.Error()})
		return err
	}
	old := i.Phase
	i.Phase = to
	i.PhaseStartedAt = w.Time
	i.PhaseAge = 0
	i.Version++
	w.Emit("signal.phase.changed", "signal-controller", corr, map[string]any{"intersectionId": i.ID, "from": old, "phase": to, "version": i.Version})
	return nil
}
func (w *World) measureLanes() {
	for k := range w.Intersections {
		for j := range w.Intersections[k].Lanes {
			l := &w.Intersections[k].Lanes[j]
			l.People, l.QueuedPeople, l.BusCount, l.PersonWait = 0, 0, 0, 0
			l.VehicleCount = 0
			l.QueueLength = 0
			l.AverageWait = 0
			l.AverageSpeed = 0
		}
	}
	for _, v := range w.Vehicles {
		if v.Next >= len(v.Route) {
			continue
		}
		i := w.Intersection(v.Route[v.Next])
		for j := range i.Lanes {
			l := &i.Lanes[j]
			if l.Direction != v.Direction {
				continue
			}
			distance := v.Stops[v.Next] - v.Position
			if distance < -.001 || distance > 40 {
				continue
			}
			l.VehicleCount++
			if distance >= -.001 && distance <= 40 {
				l.People += v.Occupants
				l.PersonWait += v.ApproachWait * float64(v.Occupants)
				if v.Type == "BUS" {
					l.BusCount++
				}
				if v.State == "WAITING" {
					l.QueuedPeople += v.Occupants
				}
			}
			if v.State == "WAITING" {
				l.QueueLength++
			}
			l.AverageWait += v.ApproachWait
			l.AverageSpeed += v.Speed * 3.6
		}
	}
	for k := range w.Intersections {
		for j := range w.Intersections[k].Lanes {
			l := &w.Intersections[k].Lanes[j]
			if l.VehicleCount > 0 {
				l.AverageWait /= float64(l.VehicleCount)
				l.AverageSpeed /= float64(l.VehicleCount)
			}
		}
	}
}
func (w *World) SetOptimizer(online bool, corr string) {
	w.OptimizerOnline = online
	kind := "optimizer.unavailable"
	fallback := "fallback.activated"
	if online {
		kind = "optimizer.recovered"
		fallback = "adaptive_control.restored"
	}
	w.Emit(kind, "chaos", corr, map[string]bool{"online": online})
	for k := range w.Intersections {
		if !online {
			w.Intersections[k].ControllerMode = "SAFE_MODE"
		} else {
			w.Intersections[k].ControllerMode = w.Mode
		}
	}
	w.Emit(fallback, "signal-controller", corr, map[string]bool{"optimizerOnline": online})
}

type ComparisonResult struct {
	Mode    string  `json:"mode"`
	Metrics Metrics `json:"metrics"`
}
type Comparison struct {
	Seed             uint64             `json:"seed"`
	Duration         float64            `json:"duration"`
	Profile          string             `json:"profile"`
	PriorityMode     string             `json:"priorityMode"`
	MapID            string             `json:"mapId"`
	Results          []ComparisonResult `json:"results"`
	ProcessingTimeMs float64            `json:"processingTimeMs"`
}

func Compare(seed uint64, duration float64, profile string) Comparison {
	return CompareMap(seed, duration, profile, "centro")
}
func CompareMap(seed uint64, duration float64, profile, mapID string) Comparison {
	return ComparePriority(seed, duration, profile, mapID, "VEHICLES")
}
func ComparePriority(seed uint64, duration float64, profile, mapID, priorityMode string) Comparison {
	start := time.Now()
	c := Comparison{PriorityMode: priorityMode, Seed: seed, Duration: duration, Profile: profile, MapID: mapID, Results: []ComparisonResult{}}
	for _, mode := range []string{"FIXED", "ADAPTIVE"} {
		w := NewMap(seed, mapID)
		w.Profile = profile
		w.PriorityMode = priorityMode
		w.Mode = mode
		g := demand.Generator{Seed: seed}
		for step := 0; step < int(math.Round(duration/0.2)); step++ {
			w.Advance(0.2, g.ArrivalsFor(profile, 0.2, len(w.Map.Rows), len(w.Map.Columns)))
			w.PendingEvents = nil
			w.PendingDecisions = nil
		}
		c.Results = append(c.Results, ComparisonResult{Mode: mode, Metrics: w.Metrics})
	}
	c.ProcessingTimeMs = float64(time.Since(start).Microseconds()) / 1000
	return c
}
