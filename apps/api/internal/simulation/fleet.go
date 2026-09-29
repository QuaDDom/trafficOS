package simulation

import (
	"math"
	"trafficos/packages/demand"
)

func vehicleWidth(kind string) float64 {
	if kind == "BUS" {
		return 1.9
	}
	if kind == "SUV" || kind == "AMBULANCE" {
		return 1.7
	}
	return 1.5
}

// Test the reserved exit slot as oriented rectangles. A radius based only on
// length incorrectly treats buses in the opposite lane as an occupied exit.
func overlapsExit(v Vehicle, x, z, heading float64, other Vehicle) bool {
	dx, dz := other.X-x, other.Z-z
	al, bl := v.Length, other.Length
	if al == 0 {
		al = VehicleLength(v.Type)
	}
	if bl == 0 {
		bl = VehicleLength(other.Type)
	}
	// Conservative broad phase: skip distant bodies before trigonometry.
	radius := (al+bl+vehicleWidth(v.Type)+vehicleWidth(other.Type))/2 + .9
	if dx*dx+dz*dz > radius*radius {
		return false
	}
	ax, az := math.Cos(heading), -math.Sin(heading)
	bx, bz := math.Cos(other.Heading), -math.Sin(other.Heading)
	for _, axis := range [][2]float64{{ax, az}, {-az, ax}, {bx, bz}, {-bz, bx}} {
		ux, uz := axis[0], axis[1]
		radiusA := math.Abs(ux*ax+uz*az)*(al/2+.3) + math.Abs(-ux*az+uz*ax)*(vehicleWidth(v.Type)/2+.15)
		radiusB := math.Abs(ux*bx+uz*bz)*(bl/2+.3) + math.Abs(-ux*bz+uz*bx)*(vehicleWidth(other.Type)/2+.15)
		if math.Abs(dx*ux+dz*uz) >= radiusA+radiusB {
			return false
		}
	}
	return true
}

func ValidVehicle(kind string, occupants int) bool {
	maximum := 4
	switch kind {
	case "", "CAR":
	case "SUV":
		maximum = 6
	case "BUS":
		maximum = 60
	default:
		return false
	}
	return occupants >= 0 && occupants <= maximum
}
func DefaultOccupants(kind string) int {
	if kind == "BUS" {
		return 35
	}
	if kind == "AMBULANCE" {
		return 3
	}
	return 2
}
func VehicleLength(kind string) float64 {
	switch kind {
	case "BUS":
		return 6.4
	case "SUV":
		return 3.34
	case "AMBULANCE":
		return 3.58
	default:
		return 3.2
	}
}
func headway(a, b Vehicle) float64 {
	al, bl := a.Length, b.Length
	if al == 0 {
		al = VehicleLength(a.Type)
	}
	if bl == 0 {
		bl = VehicleLength(b.Type)
	}
	return (al+bl)/2 + .6
}
func acceleration(kind string) float64 {
	if kind == "BUS" {
		return 1.4
	}
	return 2.8
}
func (w *World) SpawnVehicle(kind string, occupants, road int, dir string) *Vehicle {
	v := w.addVehicle(demand.Arrival{Type: kind, Occupants: occupants, Road: road, Direction: dir}, kind)
	if v != nil {
		w.measureLanes()
	}
	return v
}
