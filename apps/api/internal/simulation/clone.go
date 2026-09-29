package simulation

func (w *World) Clone() *World {
	c := *w
	c.Intersections = append([]Intersection{}, w.Intersections...)
	for k := range c.Intersections {
		c.Intersections[k].Lanes = append([]Lane{}, w.Intersections[k].Lanes...)
	}
	c.Vehicles = append([]Vehicle{}, w.Vehicles...)
	c.Incidents = append([]Incident{}, w.Incidents...)
	c.Corridors = append([]Corridor{}, w.Corridors...)
	for k := range c.Corridors {
		c.Corridors[k].Reservations = append([]Reservation{}, w.Corridors[k].Reservations...)
	}
	c.PendingEvents = nil
	c.PendingDecisions = nil
	return &c
}
