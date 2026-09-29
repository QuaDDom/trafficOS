package simulation

import (
	"testing"
	"trafficos/packages/demand"
	"trafficos/packages/network"
)

func TestLargeMapEveryBoundaryEntry(t *testing.T) {
	for _, id := range []string{"ribera", "parque", "metropolis"} {
		m := network.Get(id)
		for _, dir := range []string{"N", "S", "E", "W"} {
			count := len(m.Rows)
			if axis(dir) == "NS" {
				count = len(m.Columns)
			}
			for road := 0; road < count; road++ {
				w := NewMap(42, id)
				v := w.SpawnVehicle("BUS", 40, road, dir)
				if v == nil || len(v.Route) < 2 {
					t.Fatalf("missing bus route %s %s %d", id, dir, road)
				}
				if v.Route[0] != m.Entry(road, dir) {
					t.Fatal("wrong entry")
				}
				for k := 1; k < len(v.Route); k++ {
					found := false
					for _, next := range m.Neighbors(v.Route[k-1]) {
						found = found || next == v.Route[k]
					}
					if !found {
						t.Fatal("route crossed an absent link")
					}
				}
			}
		}
	}
}

func BenchmarkMetropolisTick(b *testing.B) {
	w := NewMap(42, "metropolis")
	w.PriorityMode = "PEOPLE"
	g := demand.Generator{Seed: 42}
	for k := 0; k < 1000; k++ {
		w.Advance(.2, g.ArrivalsFor("GRIDLOCK", .2, 10, 10))
		w.PendingEvents = nil
		w.PendingDecisions = nil
	}
	b.ReportAllocs()
	b.ResetTimer()
	for k := 0; k < b.N; k++ {
		copy := w.Clone()
		copy.Advance(.2, nil)
	}
	b.ReportMetric(float64(len(w.Vehicles)), "vehicles")
}
