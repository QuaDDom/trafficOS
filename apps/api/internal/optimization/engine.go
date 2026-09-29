package optimization

import (
	"trafficos/apps/api/internal/signals"
)

type Input struct {
	PriorityMode   string        `json:"priorityMode"`
	NSPeople       int           `json:"nsPeople"`
	EWPeople       int           `json:"ewPeople"`
	NSQueuedPeople int           `json:"nsQueuedPeople"`
	EWQueuedPeople int           `json:"ewQueuedPeople"`
	NSPersonWait   float64       `json:"nsPersonWait"`
	EWPersonWait   float64       `json:"ewPersonWait"`
	IntersectionID string        `json:"intersectionId"`
	Phase          signals.Phase `json:"phase"`
	PhaseDuration  float64       `json:"phaseDuration"`
	NSQueue        int           `json:"nsQueue"`
	EWQueue        int           `json:"ewQueue"`
	NSWait         float64       `json:"nsWait"`
	EWWait         float64       `json:"ewWait"`
	IncomingNS     int           `json:"incomingNS"`
	IncomingEW     int           `json:"incomingEW"`
}
type Reason struct {
	Basis            string  `json:"basis"`
	Type             string  `json:"type"`
	CurrentPressure  float64 `json:"currentPressure"`
	OpposingPressure float64 `json:"opposingPressure"`
	Threshold        float64 `json:"threshold"`
	MaxGreen         float64 `json:"maxGreen"`
}
type Decision struct {
	ID               string  `json:"id"`
	IntersectionID   string  `json:"intersectionId"`
	Input            Input   `json:"input"`
	Action           string  `json:"action"`
	Duration         float64 `json:"duration"`
	Reason           Reason  `json:"reason"`
	CreatedAt        string  `json:"createdAt"`
	ProcessingTimeMs float64 `json:"processingTimeMs"`
	CorrelationID    string  `json:"correlationId"`
}

// Decide is pure and deterministic. Waiting pressure prevents starvation;
// the state machine independently enforces all hard safety limits.
func Decide(in Input) Decision {
	current := float64(in.NSQueue)*2 + in.NSWait/8 + float64(in.IncomingNS)*0.25
	opposing := float64(in.EWQueue)*2 + in.EWWait/8 + float64(in.IncomingEW)*0.25
	basis := "VEHICLES"
	if in.PriorityMode == "PEOPLE" {
		basis = "PEOPLE"
		current = float64(in.NSQueuedPeople)*2 + in.NSPersonWait/8 + float64(max(0, in.NSPeople-in.NSQueuedPeople))*.25
		opposing = float64(in.EWQueuedPeople)*2 + in.EWPersonWait/8 + float64(max(0, in.EWPeople-in.EWQueuedPeople))*.25
	}
	if signals.Axis(in.Phase) == "EW" {
		current, opposing = opposing, current
	}
	d := Decision{IntersectionID: in.IntersectionID, Input: in, Action: "HOLD", Duration: 2, Reason: Reason{Basis: basis, Type: "BALANCED_DEMAND", CurrentPressure: current, OpposingPressure: opposing, Threshold: 1.4, MaxGreen: signals.MaxGreen}}
	switch {
	case !signals.IsGreen(in.Phase):
		d.Reason.Type = "CLEARANCE_IN_PROGRESS"
	case in.PhaseDuration < signals.MinGreen:
		d.Reason.Type = "MINIMUM_GREEN"
	case in.PhaseDuration >= signals.MaxGreen:
		d.Action = "NEXT_PHASE"
		d.Reason.Type = "MAXIMUM_GREEN"
	case opposing > current*1.4+2:
		d.Action = "SHORTEN_PHASE"
		d.Duration = 0
		d.Reason.Type = "OPPOSING_PRESSURE"
		if basis == "PEOPLE" {
			d.Reason.Type = "OPPOSING_PERSON_PRESSURE"
		}
	case current > opposing*1.4+2:
		d.Action = "EXTEND_PHASE"
		d.Duration = min(8, signals.MaxGreen-in.PhaseDuration)
		d.Reason.Type = "QUEUE_IMBALANCE"
		if basis == "PEOPLE" {
			d.Reason.Type = "PERSON_PRESSURE_EXTENSION"
		}
	case in.PhaseDuration >= 25:
		d.Action = "NEXT_PHASE"
		d.Reason.Type = "BALANCED_CYCLE"
	}
	return d
}
