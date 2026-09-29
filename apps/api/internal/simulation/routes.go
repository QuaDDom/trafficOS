package simulation

import (
	"math"
	"sort"
	"trafficos/apps/api/internal/signals"
	"trafficos/packages/demand"
)

type routePoint struct{ X, Z, S float64 }

func direction(dx, dz float64) string {
	if math.Abs(dx) > math.Abs(dz) {
		if dx > 0 {
			return "E"
		}
		return "W"
	}
	if dz > 0 {
		return "S"
	}
	return "N"
}
func vector(dir string) (float64, float64) {
	switch dir {
	case "N":
		return 0, -1
	case "S":
		return 0, 1
	case "W":
		return -1, 0
	default:
		return 1, 0
	}
}
func opposite(dir string) string {
	switch dir {
	case "N":
		return "S"
	case "S":
		return "N"
	case "W":
		return "E"
	default:
		return "W"
	}
}
func (w *World) route(from, to string) []string {
	queue := [][]string{{from}}
	seen := map[string]bool{from: true}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		last := p[len(p)-1]
		if last == to {
			return p
		}
		for _, id := range w.Map.Neighbors(last) {
			if !seen[id] {
				seen[id] = true
				next := append(append([]string{}, p...), id)
				queue = append(queue, next)
			}
		}
	}
	return nil
}
func (w *World) plan(a demand.Arrival, kind string) ([]string, string) {
	from := w.Map.Entry(a.Road, a.Direction)
	if from == "" {
		return nil, ""
	}
	out := a.Direction
	road := a.Road
	if kind != "AMBULANCE" {
		dirs := []string{"N", "S", "E", "W"}
		out = dirs[int(w.random()*4)%4]
		count := len(w.Map.Rows)
		if axis(out) == "NS" {
			count = len(w.Map.Columns)
		}
		road = int(w.random()*float64(count)) % count
	}
	to := w.Map.Entry(road, opposite(out))
	if to == from {
		out = a.Direction
		to = w.Map.Entry(a.Road, opposite(out))
	}
	return w.route(from, to), out
}
func appendPoint(v *Vehicle, x, z float64) {
	s := -66.0
	if len(v.path) > 0 {
		p := v.path[len(v.path)-1]
		s = p.S + math.Hypot(x-p.X, z-p.Z)
		if s-p.S < 1e-8 {
			return
		}
	}
	v.path = append(v.path, routePoint{x, z, s})
}
func (w *World) buildPath(v *Vehicle, out string) {
	incoming := v.Direction
	for k, id := range v.Route {
		n := w.Map.Node(id)
		dx, dz := vector(incoming)
		outgoing := out
		if k+1 < len(v.Route) {
			next := w.Map.Node(v.Route[k+1])
			outgoing = direction(next.X-n.X, next.Z-n.Z)
		}
		ox, oz := vector(outgoing)
		if k == 0 {
			appendPoint(v, n.X-dx*26-dz*2, n.Z-dz*26+dx*2)
		}
		ex, ez := n.X-dx*7-dz*2, n.Z-dz*7+dx*2
		tx, tz := n.X+ox*7-oz*2, n.Z+oz*7+ox*2
		appendPoint(v, ex, ez)
		v.Stops = append(v.Stops, v.path[len(v.path)-1].S-max(0, (v.Length-3.2)/2))
		v.approaches = append(v.approaches, incoming)
		maneuver := "STRAIGHT"
		if incoming == outgoing {
			appendPoint(v, tx, tz)
		} else {
			if dx*oz-dz*ox > 0 {
				maneuver = "RIGHT"
			} else {
				maneuver = "LEFT"
			}
			// Quadratic lane-center arc, with tangents aligned to both connected roads.
			cx, cz := n.X-dz*2-oz*2, n.Z+dx*2+ox*2
			for j := 1; j <= 16; j++ {
				t := float64(j) / 16
				u := 1 - t
				appendPoint(v, u*u*ex+2*u*t*cx+t*t*tx, u*u*ez+2*u*t*cz+t*t*tz)
			}
		}
		v.turns = append(v.turns, maneuver)
		v.clears = append(v.clears, v.path[len(v.path)-1].S+max(0, (v.Length-3.2)/2))
		incoming = outgoing
	}
	n := w.Map.Node(v.Route[len(v.Route)-1])
	dx, dz := vector(out)
	appendPoint(v, n.X+dx*28-dz*2, n.Z+dz*28+dx*2)
	v.PathLength = v.path[len(v.path)-1].S
	v.exitDirection = out
	locate(v)
}
func sample(v *Vehicle, s float64) (float64, float64, float64) {
	for k := 1; k < len(v.path); k++ {
		b := v.path[k]
		if s <= b.S || k == len(v.path)-1 {
			a := v.path[k-1]
			t := max(0, min(1, (s-a.S)/(b.S-a.S)))
			return a.X + (b.X-a.X)*t, a.Z + (b.Z-a.Z)*t, math.Atan2(-(b.Z - a.Z), b.X-a.X)
		}
	}
	return v.X, v.Z, v.Heading
}
func locate(v *Vehicle) {
	v.X, v.Z, v.Heading = sample(v, v.Position)
	v.Maneuver = "EXIT"
	v.Direction = v.exitDirection
	if v.Next < len(v.Route) {
		v.Direction = v.approaches[v.Next]
		v.Maneuver = v.turns[v.Next]
	}
}
func (w *World) moveVehicles(dt float64) {
	// Each crossing is reserved until its vehicle clears it, including turns and
	// late clearance after a light change. New entrants also need downstream space.
	occupied := map[string]string{}
	for _, v := range w.Vehicles {
		if v.Next < len(v.Route) && v.Position > v.Stops[v.Next]+.001 {
			occupied[v.Route[v.Next]] = v.ID
		}
	}
	sort.SliceStable(w.Vehicles, func(i, j int) bool { return w.Vehicles[i].Born < w.Vehicles[j].Born })
	for idx := range w.Vehicles {
		v := &w.Vehicles[idx]
		limit := w.Map.SpeedLimit
		if v.Type == "BUS" {
			limit = min(limit, 8)
		}
		if v.Type == "AMBULANCE" {
			limit = 12
		}
		gap := math.Inf(1)
		waitReason := "QUEUE"
		dx, dz := math.Cos(v.Heading), -math.Sin(v.Heading)
		for oi := range w.Vehicles {
			other := &w.Vehicles[oi]
			if other.ID == v.ID {
				continue
			}
			rx, rz := other.X-v.X, other.Z-v.Z
			forward := rx*dx + rz*dz
			lateral := math.Abs(rx*dz - rz*dx)
			if forward > 0 && lateral < 1.6 && math.Cos(other.Heading-v.Heading) > .7 {
				gap = min(gap, max(0, forward-headway(*v, *other)))
			}
		}
		crossing := false
		if v.Next < len(v.Route) {
			inter := w.Intersection(v.Route[v.Next])
			stop := v.Stops[v.Next]
			crossing = v.Position > stop+.001
			if v.Maneuver != "STRAIGHT" && v.Position > stop-6 {
				limit = min(limit, 5.5)
			}
			canGo := signals.IsGreen(inter.Phase) && signals.Axis(inter.Phase) == axis(v.Direction)
			blockedBy := "SIGNAL"
			if owner := occupied[inter.ID]; owner != "" && owner != v.ID {
				canGo = false
				blockedBy = "JUNCTION_BUSY"
			}
			if !crossing {
				ex, ez, eh := sample(v, v.clears[v.Next]+max(3.8, v.Length))
				for _, other := range w.Vehicles {
					if other.ID != v.ID && overlapsExit(*v, ex, ez, eh, other) {
						canGo = false
						blockedBy = "EXIT_BLOCKED"
						break
					}
				}
				if !canGo {
					stopGap := max(0, stop-v.Position)
					if stopGap <= gap {
						gap = stopGap
						waitReason = blockedBy
					}
				}
			}
			for _, incident := range w.Incidents {
				if incident.Active && incident.IntersectionID == inter.ID && !crossing {
					limit = min(limit, 3)
				}
			}
		}
		desired := min(limit, math.Sqrt(2*4.5*gap))
		speed := min(desired, v.Speed+acceleration(v.Type)*dt)
		target := v.Position + min(speed*dt, gap)
		moved := max(0, target-v.Position)
		v.Speed = moved / dt
		if moved < .01 {
			if v.State != "WAITING" {
				v.StopCount++
				w.Metrics.TotalStops++
			}
			v.State = "WAITING"
			v.WaitReason = waitReason
			v.Wait += dt
			v.ApproachWait += dt
			w.Metrics.TotalPersonWait += dt * float64(v.Occupants)
			w.Metrics.TotalWait += dt
		} else {
			v.State = "MOVING"
			v.WaitReason = ""
			if crossing {
				v.State = "CROSSING"
			}
		}
		v.Position = target
		if v.Next < len(v.Route) && target > v.Stops[v.Next]+.001 {
			occupied[v.Route[v.Next]] = v.ID
		}
		if v.Next < len(v.Route) && target >= v.clears[v.Next] {
			if v.turns[v.Next] != "STRAIGHT" {
				w.Emit("traffic.vehicle.turned", "traffic-engine", v.ID, map[string]any{"vehicleId": v.ID, "intersectionId": v.Route[v.Next], "maneuver": v.turns[v.Next]})
			}
			v.Next++
			v.ApproachWait = 0
			v.State = "MOVING"
		}
		locate(v)
	}
}
