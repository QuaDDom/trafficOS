package simulation

import (
	"math"
	"testing"
	"trafficos/apps/api/internal/signals"
	"trafficos/packages/demand"
	"trafficos/packages/network"
)

func TestPersonDetectionIgnoresDistantAndCommittedVehicles(t *testing.T) {
	w := New(42)
	v := w.addVehicle(demand.Arrival{Road: 1, Direction: "E", Occupants: 40}, "BUS")
	id := v.Route[0]
	v.State = "WAITING"
	v.ApproachWait = 10
	v.Position = v.Stops[0] - 5
	locate(v)
	w.measureLanes()
	lane := func() Lane {
		for _, l := range w.Intersection(id).Lanes {
			if l.Direction == "E" {
				return l
			}
		}
		panic("lane")
	}
	if l := lane(); l.People != 40 || l.QueuedPeople != 40 || l.BusCount != 1 || l.PersonWait != 400 {
		t.Fatalf("occupancy not measured: %+v", l)
	}
	v.Position = v.Stops[0] - 41
	locate(v)
	w.measureLanes()
	if lane().People != 0 {
		t.Fatal("distant bus influenced crossing")
	}
	v.Position = v.Stops[0] + 1
	locate(v)
	w.measureLanes()
	if lane().People != 0 {
		t.Fatal("committed bus influenced its old approach")
	}
}
func TestBusClearanceAndFollowingDistance(t *testing.T) {
	w := New(42)
	bus := w.addVehicle(demand.Arrival{Road: 1, Direction: "E", Occupants: 40}, "BUS")
	route := append([]string{}, bus.Route...)
	stop, clear := bus.Stops[0], bus.clears[0]
	path := append([]routePoint{}, bus.path...)
	if bus.Length != 6.4 {
		t.Fatal("bus length missing")
	}
	bus.Position = -59
	locate(bus)
	car := w.addVehicle(demand.Arrival{Road: 1, Direction: "E", Occupants: 2}, "CAR")
	if car == nil {
		t.Fatal("entry should have room")
	}
	// Keep both on the same route to isolate longitudinal following.
	car.Route = route
	car.path = path
	car.Stops = append([]float64{}, w.Vehicles[0].Stops...)
	car.clears = append([]float64{}, w.Vehicles[0].clears...)
	car.approaches = append([]string{}, w.Vehicles[0].approaches...)
	car.turns = append([]string{}, w.Vehicles[0].turns...)
	car.exitDirection = w.Vehicles[0].exitDirection
	car.PathLength = w.Vehicles[0].PathLength
	locate(car)
	if headway(w.Vehicles[0], *car) < 5.3 {
		t.Fatal("bus uses car-sized headway")
	}
	i := w.Intersection(route[0])
	i.Phase = signals.NSGreen
	for k := 0; k < 50; k++ {
		w.moveVehicles(.2)
	}
	if w.Vehicles[0].Position > stop+.001 {
		t.Fatal("bus crossed red")
	}
	if d := math.Hypot(w.Vehicles[0].X-w.Vehicles[1].X, w.Vehicles[0].Z-w.Vehicles[1].Z); d+1e-6 < headway(w.Vehicles[0], w.Vehicles[1]) {
		t.Fatal("following car overlaps bus")
	}
	if clear-stop < 17 {
		t.Fatal("bus did not reserve extra clearance")
	}
}
func TestMixedFleetPeopleConservationAndSafetyAllMaps(t *testing.T) {
	for _, m := range network.Catalog() {
		t.Run(m.ID, func(t *testing.T) {
			w := NewMap(42, m.ID)
			w.PriorityMode = "PEOPLE"
			g := demand.Generator{Seed: 42}
			seen := map[string]bool{}
			for k := 0; k < 1000; k++ {
				w.Advance(.2, g.ArrivalsFor("NORMAL", .2, len(m.Rows), len(m.Columns)))
				cross := map[string]bool{}
				for _, v := range w.Vehicles {
					seen[v.Type] = true
					if v.Occupants < 1 {
						t.Fatal("unassigned occupants")
					}
					if v.Next < len(v.Route) && v.Position > v.Stops[v.Next]+.001 {
						if cross[v.Route[v.Next]] {
							t.Fatal("conflicting bus/car in intersection")
						}
						cross[v.Route[v.Next]] = true
					}
				}
				if w.Metrics.PeopleCreated != w.Metrics.PeopleCompleted+w.Metrics.ActivePeople {
					t.Fatal("people not conserved")
				}
				for _, i := range w.Intersections {
					if signals.IsGreen(i.Phase) && i.PhaseAge > signals.MaxGreen+.21 {
						t.Fatal("priority exceeded max green")
					}
				}
				w.PendingEvents = nil
				w.PendingDecisions = nil
			}
			if !seen["BUS"] || !seen["SUV"] || w.Metrics.PeopleCompleted == 0 {
				t.Fatalf("fleet did not progress: %v %+v", seen, w.Metrics)
			}
		})
	}
}
func TestEmergencyPrecedesPeoplePriority(t *testing.T) {
	w := New(42)
	w.PriorityMode = "PEOPLE"
	if _, err := w.SpawnEmergency(2, "test"); err != nil {
		t.Fatal(err)
	}
	if w.Metrics.ActivePeople != 3 || w.Metrics.ActiveVehicles != 1 {
		t.Fatal("emergency occupants must be visible before the next tick")
	}
	w.Advance(.2, nil)
	found := false
	for _, i := range w.Intersections {
		if i.ControllerMode == "EMERGENCY_OVERRIDE" {
			found = true
		}
	}
	if !found {
		t.Fatal("emergency lost precedence")
	}
	w.SetOptimizer(false, "test")
	w.Advance(.2, nil)
	for _, i := range w.Intersections {
		if i.ControllerMode != "SAFE_MODE" {
			t.Fatal("fallback lost")
		}
	}
}
