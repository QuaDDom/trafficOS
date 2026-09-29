package demand

import (
	"reflect"
	"testing"
)

func TestMixedFleetReproducibleAndBounded(t *testing.T) {
	a, b := Generator{Seed: 42}, Generator{Seed: 42}
	seen := map[string]int{}
	for k := 0; k < 500; k++ {
		left, right := a.ArrivalsFor("NORMAL", .2, 3, 4), b.ArrivalsFor("NORMAL", .2, 3, 4)
		if !reflect.DeepEqual(left, right) {
			t.Fatal("demand is not reproducible")
		}
		for _, v := range left {
			seen[v.Type]++
			limit := 4
			if v.Type == "SUV" {
				limit = 6
			}
			if v.Type == "BUS" {
				limit = 60
			}
			if v.Occupants < 1 || v.Occupants > limit {
				t.Fatalf("invalid occupancy %+v", v)
			}
		}
	}
	for _, kind := range []string{"CAR", "SUV", "BUS"} {
		if seen[kind] < 3 {
			t.Fatalf("missing fleet category %s: %v", kind, seen)
		}
	}
}
