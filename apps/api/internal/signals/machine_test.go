package signals

import (
	"math"
	"testing"
)

func TestNonfiniteDurationsRejected(t *testing.T) {
	for _, age := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1} {
		if Validate(NSGreen, NSYellow, age) == nil {
			t.Fatal("invalid duration accepted")
		}
	}
}

func TestEveryTransitionAndTiming(t *testing.T) {
	phases := []Phase{NSGreen, NSYellow, RedToEW, EWGreen, EWYellow, RedToNS}
	for _, from := range phases {
		for _, to := range phases {
			for _, age := range []float64{0, Minimum(from) - 0.01, Minimum(from), 100} {
				want := to == Next(from) && age >= Minimum(from)
				if got := Validate(from, to, age) == nil; got != want {
					t.Fatalf("%s -> %s at %f: %v", from, to, age, got)
				}
			}
		}
	}
}
func FuzzUnsafeTransition(f *testing.F) {
	f.Add(uint8(0), uint8(3), 8.0)
	f.Fuzz(func(t *testing.T, a, b uint8, age float64) {
		p := []Phase{NSGreen, NSYellow, RedToEW, EWGreen, EWYellow, RedToNS}
		from, to := p[int(a)%6], p[int(b)%6]
		if Validate(from, to, age) == nil && (Next(from) != to || age < Minimum(from)) {
			t.Fatal("unsafe transition accepted")
		}
	})
}
