package optimization

import (
	"testing"
	"trafficos/apps/api/internal/signals"
)

func TestDecisionRules(t *testing.T) {
	for _, tc := range []struct {
		age            float64
		ns, ew         int
		action, reason string
	}{
		{2, 0, 100, "HOLD", "MINIMUM_GREEN"}, {45, 100, 0, "NEXT_PHASE", "MAXIMUM_GREEN"}, {10, 20, 1, "EXTEND_PHASE", "QUEUE_IMBALANCE"}, {10, 1, 20, "SHORTEN_PHASE", "OPPOSING_PRESSURE"}, {25, 2, 2, "NEXT_PHASE", "BALANCED_CYCLE"},
	} {
		d := Decide(Input{Phase: signals.NSGreen, PhaseDuration: tc.age, NSQueue: tc.ns, EWQueue: tc.ew})
		if d.Action != tc.action || d.Reason.Type != tc.reason {
			t.Fatalf("%+v => %+v", tc, d)
		}
	}
}
func TestDeterminismAndRotation(t *testing.T) {
	a := Input{Phase: signals.NSGreen, PhaseDuration: 18, NSQueue: 20, EWQueue: 2}
	if Decide(a) != Decide(a) {
		t.Fatal("not deterministic")
	}
	b := a
	b.Phase = signals.EWGreen
	b.NSQueue, b.EWQueue = b.EWQueue, b.NSQueue
	if Decide(a).Action != Decide(b).Action {
		t.Fatal("axis bias")
	}
}

var decisionSink Decision

func BenchmarkDecide(b *testing.B) {
	in := Input{Phase: signals.NSGreen, PhaseDuration: 18, NSQueue: 20, EWQueue: 4}
	for b.Loop() {
		decisionSink = Decide(in)
	}
}
