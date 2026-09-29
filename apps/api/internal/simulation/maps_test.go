package simulation

import (
	"math"
	"testing"
	"trafficos/packages/demand"
	"trafficos/packages/network"
)

func TestRoutesFollowEveryMapAndTurnWithinRoads(t *testing.T) {
	for _, m := range network.Catalog() {
		t.Run(m.ID, func(t *testing.T) {
			w := NewMap(42, m.ID)
			turns := 0
			for k := 0; k < 80; k++ {
				v := w.addVehicle(demand.Arrival{Road: k % 3, Direction: []string{"N", "S", "E", "W"}[k%4]}, "CAR")
				if v == nil {
					t.Fatal("empty isolated entry")
				}
				for j, id := range v.Route {
					if j > 0 {
						connected := false
						for _, n := range m.Neighbors(v.Route[j-1]) {
							connected = connected || n == id
						}
						if !connected {
							t.Fatal("route crosses a missing street")
						}
					}
					if v.turns[j] != "STRAIGHT" {
						turns++
					}
				}
				for j, p := range v.path {
					if j > 0 && p.S <= v.path[j-1].S {
						t.Fatal("non-increasing route distance")
					}
					nearest := math.Inf(1)
					for _, r := range m.Roads {
						dx, dz := r.B.X-r.A.X, r.B.Z-r.A.Z
						u := max(0, min(1, ((p.X-r.A.X)*dx+(p.Z-r.A.Z)*dz)/(dx*dx+dz*dz)))
						nearest = min(nearest, math.Hypot(p.X-r.A.X-u*dx, p.Z-r.A.Z-u*dz))
					}
					if nearest > 5.1 {
						t.Fatalf("route left road: %+v, distance %.2f", p, nearest)
					}
				}
				w.Vehicles = nil
			}
			if turns < 10 {
				t.Fatal("map is missing turning routes")
			}
		})
	}
}
func TestAllMapsProgressWithExclusiveCrossings(t *testing.T) {
	for _, m := range network.Catalog() {
		t.Run(m.ID, func(t *testing.T) {
			w := NewMap(42, m.ID)
			g := demand.Generator{Seed: 42}
			turned := 0
			for k := 0; k < 1500; k++ {
				w.Advance(.2, g.ArrivalsFor("NORMAL", .2, len(m.Rows), len(m.Columns)))
				crossings := map[string]string{}
				for _, v := range w.Vehicles {
					if v.Speed < 0 || math.IsNaN(v.Heading) {
						t.Fatal("invalid movement")
					}
					if v.Next < len(v.Route) && v.Position > v.Stops[v.Next]+.001 {
						if crossings[v.Route[v.Next]] != "" {
							t.Fatal("conflicting vehicles share a crossing")
						}
						crossings[v.Route[v.Next]] = v.ID
					}
				}
				for _, e := range w.PendingEvents {
					if e.Type == "traffic.vehicle.turned" {
						turned++
					}
				}
				if w.Metrics.VehiclesCreated != w.Metrics.VehiclesCompleted+len(w.Vehicles) {
					t.Fatal("vehicle conservation failed")
				}
				w.PendingEvents = nil
				w.PendingDecisions = nil
			}
			if w.Metrics.VehiclesCompleted < 10 || turned < 10 {
				t.Fatalf("traffic did not progress: completed %d turns %d", w.Metrics.VehiclesCompleted, turned)
			}
		})
	}
}
func TestEmergencyAcrossAllMaps(t *testing.T) {
	for _, m := range network.Catalog() {
		t.Run(m.ID, func(t *testing.T) {
			w := NewMap(42, m.ID)
			c, err := w.SpawnEmergency(2, "test")
			if err != nil {
				t.Fatal(err)
			}
			if len(c.Reservations) != len(m.Columns) {
				t.Fatal("route does not match avenue")
			}
			for k := 0; k < 900; k++ {
				w.Advance(.2, nil)
				w.PendingEvents = nil
				w.PendingDecisions = nil
			}
			if w.Corridors[0].Status != "COMPLETED" {
				t.Fatal("emergency did not finish")
			}
		})
	}
}
