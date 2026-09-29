// Package demand is shared by the external simulator and reproducible comparisons.
package demand

import "math"

type Arrival struct {
	Type      string `json:"type,omitempty"`
	Occupants int    `json:"occupants,omitempty"`
	Road      int    `json:"road"`
	Direction string `json:"direction"`
}
type Generator struct{ Seed uint64 }

func (g *Generator) Random() float64 {
	if g.Seed == 0 {
		g.Seed = 42
	}
	g.Seed ^= g.Seed << 13
	g.Seed ^= g.Seed >> 7
	g.Seed ^= g.Seed << 17
	return float64(g.Seed>>11) / float64(uint64(1)<<53)
}

// Rates are vehicles/hour per entry lane. Main avenue receives 1.5x demand.
func (g *Generator) Arrivals(profile string, dt float64) []Arrival {
	return g.ArrivalsFor(profile, dt, 3, 3)
}
func (g *Generator) ArrivalsFor(profile string, dt float64, rows, columns int) []Arrival {
	rate := 400.0
	switch profile {
	case "LOW":
		rate = 100
	case "RUSH_HOUR":
		rate = 900
	case "GRIDLOCK":
		rate = 1800
	}
	a := []Arrival{}
	for road := 0; road < max(rows, columns); road++ {
		for _, dir := range []string{"E", "W", "N", "S"} {
			if ((dir == "E" || dir == "W") && road >= rows) || ((dir == "N" || dir == "S") && road >= columns) {
				continue
			}
			r := rate
			if road == 1 && (dir == "E" || dir == "W") {
				r *= 1.5
			}
			limit := math.Exp(-r * dt / 3600)
			product := g.Random()
			for product > limit {
				kind, people := "CAR", 1+int(g.Random()*4)
				roll := g.Random()
				if roll < .12 {
					kind, people = "BUS", 8+int(g.Random()*48)
				} else if roll < .37 {
					kind, people = "SUV", 1+int(g.Random()*6)
				}
				a = append(a, Arrival{Road: road, Direction: dir, Type: kind, Occupants: people})
				product *= g.Random()
			}
		}
	}
	return a
}
