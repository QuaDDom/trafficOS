package simulation

import (
	"encoding/json"
	"math"
	"testing"
	"trafficos/apps/api/internal/signals"
	"trafficos/packages/demand"
)

func TestLongRunSafetyAndVehicleConservation(t *testing.T) {
	w := New(42)
	g := demand.Generator{Seed: 42}
	previous := w.Clone()
	for step := 0; step < 3000; step++ {
		w.Advance(.2, g.Arrivals("GRIDLOCK", .2))
		for k, i := range w.Intersections {
			old := previous.Intersections[k]
			if i.Phase != old.Phase {
				if err := signals.Validate(old.Phase, i.Phase, old.PhaseAge+.2); err != nil {
					t.Fatal(err)
				}
			}
			if signals.IsGreen(i.Phase) && i.PhaseAge > 45.21 {
				t.Fatal("maximum green violated")
			}
		}
		for _, v := range w.Vehicles {
			if math.IsNaN(v.X) || v.Speed < 0 {
				t.Fatal("invalid vehicle state")
			}
		}
		if w.Metrics.VehiclesCreated != w.Metrics.VehiclesCompleted+len(w.Vehicles) {
			t.Fatal("vehicle conservation violated")
		}
		previous = w.Clone()
		w.PendingEvents = nil
		w.PendingDecisions = nil
	}
	if w.Metrics.VehiclesCompleted == 0 || w.Metrics.TotalWait <= 0 {
		t.Fatal("simulation did not progress")
	}
}
func TestRedStopsAndGreenReleases(t *testing.T) {
	w := New(42)
	w.Mode = "FIXED"
	v := w.addVehicle(demand.Arrival{Road: 0, Direction: "S"}, "CAR")
	id := v.ID
	for k := 0; k < 50; k++ {
		w.Advance(.2, nil)
	}
	for _, v := range w.Vehicles {
		if v.ID == id && v.Position > -47.0+1e-6 {
			t.Fatalf("crossed red: %+v", v)
		}
	}
	for k := 0; k < 200; k++ {
		w.Advance(.2, nil)
	}
	for _, v := range w.Vehicles {
		if v.ID == id && v.Position <= -47 {
			t.Fatal("never released on green")
		}
	}
}
func TestEmergencyReservationsReleaseAndRestore(t *testing.T) {
	w := New(42)
	c, err := w.SpawnEmergency(2, "emergency-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Reservations) != 3 {
		t.Fatal("missing route windows")
	}
	overridden := false
	for k := 0; k < 600; k++ {
		w.Advance(.2, nil)
		for _, i := range w.Intersections {
			if i.ControllerMode == "EMERGENCY_OVERRIDE" {
				overridden = true
			}
		}
		w.PendingEvents = nil
		w.PendingDecisions = nil
	}
	if !overridden {
		t.Fatal("no priority applied")
	}
	if w.Corridors[0].Status != "COMPLETED" {
		t.Fatal("ambulance failed to arrive")
	}
	for _, r := range w.Corridors[0].Reservations {
		if !r.Released {
			t.Fatal("reservation leaked")
		}
	}
	for _, i := range w.Intersections {
		if i.ControllerMode != "ADAPTIVE" {
			t.Fatal("controller not restored")
		}
	}
}
func TestEmergencyConflictOrdering(t *testing.T) {
	w := New(42)
	w.Time = 10
	w.Corridors = []Corridor{{ID: "low", Priority: 1, CreatedAt: 0, Status: "ACTIVE", Reservations: []Reservation{{IntersectionID: "INT-05", Axis: "NS", StartsAt: 0, EndsAt: 20}}}, {ID: "high", Priority: 3, CreatedAt: 2, Status: "ACTIVE", Reservations: []Reservation{{IntersectionID: "INT-05", Axis: "EW", StartsAt: 2, EndsAt: 20}}}}
	if w.desiredAxis("INT-05") != "EW" {
		t.Fatal("priority order ignored")
	}
	w.Corridors[1].Priority = 1
	if w.desiredAxis("INT-05") != "NS" {
		t.Fatal("ETA tie-break ignored")
	}
}
func TestComparisonRepeatability(t *testing.T) {
	a, b := Compare(42, 60, "RUSH_HOUR"), Compare(42, 60, "RUSH_HOUR")
	for k := range a.Results {
		aa, _ := json.Marshal(a.Results[k])
		bb, _ := json.Marshal(b.Results[k])
		if string(aa) != string(bb) {
			t.Fatal("comparison not deterministic")
		}
	}
}

var comparisonSink Comparison

func BenchmarkWorldTick(b *testing.B) {
	w := New(42)
	g := demand.Generator{Seed: 42}
	b.ResetTimer()
	for b.Loop() {
		w.Advance(.2, g.Arrivals("NORMAL", .2))
		w.PendingEvents = nil
		w.PendingDecisions = nil
	}
}
func BenchmarkComparison180s(b *testing.B) {
	for b.Loop() {
		comparisonSink = Compare(42, 180, "NORMAL")
	}
}
