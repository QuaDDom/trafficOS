// Package signals owns the safety envelope. No controller can bypass it.
package signals

import (
	"fmt"
	"math"
)

type Phase string

const (
	NSGreen  Phase = "NS_GREEN"
	NSYellow Phase = "NS_YELLOW"
	RedToEW  Phase = "ALL_RED_NS_TO_EW"
	EWGreen  Phase = "EW_GREEN"
	EWYellow Phase = "EW_YELLOW"
	RedToNS  Phase = "ALL_RED_EW_TO_NS"
)
const MinGreen = 8.0
const MaxGreen = 45.0
const Yellow = 3.0
const Clearance = 2.0

func Next(p Phase) Phase {
	switch p {
	case NSGreen:
		return NSYellow
	case NSYellow:
		return RedToEW
	case RedToEW:
		return EWGreen
	case EWGreen:
		return EWYellow
	case EWYellow:
		return RedToNS
	case RedToNS:
		return NSGreen
	}
	return ""
}
func IsGreen(p Phase) bool { return p == NSGreen || p == EWGreen }
func Axis(p Phase) string {
	if p == NSGreen || p == NSYellow || p == RedToEW {
		return "NS"
	}
	return "EW"
}
func Minimum(p Phase) float64 {
	if IsGreen(p) {
		return MinGreen
	}
	if p == NSYellow || p == EWYellow {
		return Yellow
	}
	return Clearance
}
func Validate(from, to Phase, age float64) error {
	if math.IsNaN(age) || math.IsInf(age, 0) || age < 0 {
		return fmt.Errorf("invalid_duration")
	}
	if Next(from) != to || to == "" {
		return fmt.Errorf("unsafe_transition: %s -> %s", from, to)
	}
	if age+1e-8 < Minimum(from) {
		return fmt.Errorf("minimum_duration: %.2fs < %.2fs", age, Minimum(from))
	}
	return nil
}
