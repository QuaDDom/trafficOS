package simulation

import (
	"math"
	"testing"
	"trafficos/packages/demand"
)

func TestExitClearanceRespectsLaneWidthAndOrientation(t *testing.T) {
	bus := Vehicle{Type: "BUS", Length: 6.4}
	tests := []struct {
		name          string
		x, z, heading float64
		want          bool
	}{
		{"opposite lane bus", 0, -4, math.Pi, false},
		{"same lane occupied", 2, 0, 0, true},
		{"perpendicular vehicle across exit", 0, 0, math.Pi / 2, true},
		{"perpendicular vehicle beyond exit", 0, 8, math.Pi / 2, false},
		{"diagonal body inside slot", 2, 1, math.Pi / 4, true},
		{"clear longitudinal gap", 8, 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			other := bus
			other.X, other.Z, other.Heading = tc.x, tc.z, tc.heading
			if got := overlapsExit(bus, 0, 0, 0, other); got != tc.want {
				t.Fatalf("blocked=%v want %v", got, tc.want)
			}
		})
	}
}

func TestNormalMixedFleetDoesNotFreezeBehindOppositeLaneBuses(t *testing.T) {
	w := New(42)
	g := demand.Generator{Seed: 42}
	completedAtSixMinutes := 0
	for k := 0; k < 2700; k++ {
		w.Advance(.2, g.ArrivalsFor("NORMAL", .2, 3, 3))
		w.PendingEvents = nil
		w.PendingDecisions = nil
		if k == 1799 {
			completedAtSixMinutes = w.Metrics.VehiclesCompleted
		}
	}
	if w.Metrics.VehiclesCompleted-completedAtSixMinutes < 20 {
		t.Fatalf("flow stopped: six minutes %d; nine minutes %d", completedAtSixMinutes, w.Metrics.VehiclesCompleted)
	}
}

func TestLaneWaitUsesCurrentApproachNotPreviousJunctions(t *testing.T) {
	w := New(42)
	v := w.SpawnVehicle("BUS", 40, 1, "E")
	v.Wait = 900
	v.ApproachWait = 7
	v.Position = v.Stops[0] - 4
	locate(v)
	w.measureLanes()
	for _, l := range w.Intersection(v.Route[0]).Lanes {
		if l.Direction == "E" && l.AverageWait != 7 {
			t.Fatal("old junction wait leaked into demand")
		}
	}
}
