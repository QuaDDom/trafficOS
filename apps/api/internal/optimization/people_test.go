package optimization

import (
	"testing"
	"trafficos/apps/api/internal/signals"
)

func TestPeoplePriorityChangesDecisionWithoutSkippingSafety(t *testing.T) {
	in := Input{Phase: signals.NSGreen, PhaseDuration: 12, NSQueue: 8, EWQueue: 1, IncomingNS: 8, IncomingEW: 1, NSPeople: 12, EWPeople: 40, NSQueuedPeople: 12, EWQueuedPeople: 40}
	if got := Decide(in); got.Action != "EXTEND_PHASE" {
		t.Fatalf("vehicle baseline: %+v", got)
	}
	in.PriorityMode = "PEOPLE"
	if got := Decide(in); got.Action != "SHORTEN_PHASE" || got.Reason.Type != "OPPOSING_PERSON_PRESSURE" || got.Reason.Basis != "PEOPLE" {
		t.Fatalf("full bus should win: %+v", got)
	}
	in.EWPeople, in.EWQueuedPeople = 1, 1
	if got := Decide(in); got.Action != "EXTEND_PHASE" {
		t.Fatalf("empty bus must not get a blanket bonus: %+v", got)
	}
	in.EWPeople, in.EWQueuedPeople = 600, 600
	in.PhaseDuration = 7.9
	if Decide(in).Action != "HOLD" {
		t.Fatal("priority violated minimum green")
	}
	in.Phase = signals.NSYellow
	in.PhaseDuration = 3
	if Decide(in).Action != "HOLD" {
		t.Fatal("optimizer must not skip clearance")
	}
	in.Phase = signals.EWGreen
	in.PhaseDuration = 45
	if Decide(in).Action != "NEXT_PHASE" {
		t.Fatal("busy bus street starved opposing traffic")
	}
}
func TestPeoplePriorityRotatesAxesAndAccountsForWait(t *testing.T) {
	a := Input{PriorityMode: "PEOPLE", Phase: signals.NSGreen, PhaseDuration: 12, NSPeople: 10, EWPeople: 10, NSQueuedPeople: 10, EWQueuedPeople: 10, EWPersonWait: 800}
	if Decide(a).Action != "SHORTEN_PHASE" {
		t.Fatal("person wait ignored")
	}
	b := a
	b.Phase = signals.EWGreen
	b.NSPersonWait, b.EWPersonWait = b.EWPersonWait, b.NSPersonWait
	if Decide(a).Action != Decide(b).Action {
		t.Fatal("axis bias")
	}
}
