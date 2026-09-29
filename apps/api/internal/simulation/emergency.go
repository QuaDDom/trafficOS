package simulation

import (
	"fmt"
	"sort"
	"trafficos/packages/demand"
)

func (w *World) SpawnEmergency(priority int, corr string) (*Corridor, error) {
	v := w.addVehicle(demand.Arrival{Road: 1, Direction: "E"}, "AMBULANCE")
	if v == nil {
		return nil, fmt.Errorf("entry lane occupied; retry with a new request")
	}
	c := Corridor{ID: w.ID("corridor"), VehicleID: v.ID, Priority: priority, CreatedAt: w.Time, Status: "ACTIVE", Reservations: []Reservation{}}
	for idx, id := range v.Route {
		eta := w.Time + (v.Stops[idx]+7-v.Position)/12
		c.Reservations = append(c.Reservations, Reservation{IntersectionID: id, Axis: "EW", StartsAt: max(w.Time, eta-15), EndsAt: eta + 12})
	}
	w.Corridors = append(w.Corridors, c)
	w.Emit("emergency.vehicle.detected", "emergency", corr, map[string]any{"vehicleId": v.ID, "priority": priority})
	w.Emit("emergency.route.calculated", "emergency", corr, map[string]any{"route": v.Route, "destination": "City Hospital"})
	w.Emit("emergency.corridor.created", "emergency", corr, c)
	return &w.Corridors[len(w.Corridors)-1], nil
}
func (w *World) reservations() {
	for ci := range w.Corridors {
		c := &w.Corridors[ci]
		if c.Status != "ACTIVE" {
			continue
		}
		var vehicle *Vehicle
		for vi := range w.Vehicles {
			if w.Vehicles[vi].ID == c.VehicleID {
				vehicle = &w.Vehicles[vi]
				break
			}
		}
		for ri := range c.Reservations {
			r := &c.Reservations[ri]
			if r.Released {
				continue
			}
			if vehicle == nil || vehicle.Next > ri {
				r.Released = true
				w.Emit("emergency.reservation.released", "corridor-controller", c.ID, *r)
			} else {
				eta := w.Time + max(0, (vehicle.Stops[ri]+7-vehicle.Position)/12)
				r.StartsAt = max(c.CreatedAt, eta-15)
				r.EndsAt = eta + 12
			}
		}
		if vehicle == nil {
			c.Status = "COMPLETED"
			w.Emit("emergency.corridor.completed", "corridor-controller", c.ID, map[string]string{"corridorId": c.ID})
		}
	}
}
func (w *World) desiredAxis(id string) string {
	type candidate struct {
		priority     int
		eta, created float64
		id, axis     string
	}
	candidates := []candidate{}
	for _, c := range w.Corridors {
		if c.Status != "ACTIVE" {
			continue
		}
		for _, r := range c.Reservations {
			if r.IntersectionID == id && !r.Released && w.Time >= r.StartsAt && w.Time <= r.EndsAt {
				candidates = append(candidates, candidate{c.Priority, r.StartsAt, c.CreatedAt, c.ID, r.Axis})
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.priority != b.priority {
			return a.priority > b.priority
		}
		if a.eta != b.eta {
			return a.eta < b.eta
		}
		if a.created != b.created {
			return a.created < b.created
		}
		return a.id < b.id
	})
	if len(candidates) > 0 {
		return candidates[0].axis
	}
	return ""
}
