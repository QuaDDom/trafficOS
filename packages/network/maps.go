// Package network defines the same street graph used by traffic and rendering.
package network

import "fmt"

type Point struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
}
type Node struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Z    float64 `json:"z"`
}
type Road struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	A      Point  `json:"a"`
	B      Point  `json:"b"`
	Avenue bool   `json:"avenue"`
}
type Lot struct {
	X     float64 `json:"x"`
	Z     float64 `json:"z"`
	Width float64 `json:"width"`
	Depth float64 `json:"depth"`
	Kind  string  `json:"kind"`
}
type Map struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Columns     []float64 `json:"columns"`
	Rows        []float64 `json:"rows"`
	Nodes       []Node    `json:"nodes"`
	Roads       []Road    `json:"roads"`
	Lots        []Lot     `json:"lots"`
	Width       float64   `json:"width"`
	Depth       float64   `json:"depth"`
	SpeedLimit  float64   `json:"speedLimit"`
	nodeIndex   map[string]Node
	adjacency   map[string][]string
}

func IDs() []string { return []string{"centro", "barrio", "avenida", "ribera", "parque", "metropolis"} }
func Valid(id string) bool {
	for _, candidate := range IDs() {
		if candidate == id {
			return true
		}
	}
	return false
}
func Catalog() []Map {
	result := []Map{}
	for _, id := range IDs() {
		result = append(result, Get(id))
	}
	return result
}
func coordinates(count int, spacing float64) []float64 {
	values := make([]float64, count)
	for i := range values {
		values[i] = (float64(i) - float64(count-1)/2) * spacing
	}
	return values
}
func Get(id string) Map {
	m := Map{ID: id, Columns: []float64{-40, 0, 40}, Rows: []float64{-40, 0, 40}, SpeedLimit: 9}
	switch id {
	case "ribera":
		m.Name = "Distrito Ribera"
		m.Description = "Canal, bridges and waterfront neighborhoods / 25 junctions"
		m.Columns, m.Rows = coordinates(5, 62), coordinates(5, 56)
		m.SpeedLimit = 10
	case "parque":
		m.Name = "Ciudad del Parque"
		m.Description = "Green corridors, residential districts and a civic center / 49 junctions"
		m.Columns, m.Rows = coordinates(7, 58), coordinates(7, 58)
		m.SpeedLimit = 10
	case "metropolis":
		m.Name = "Metrópolis"
		m.Description = "Downtown towers, industrial blocks and residential neighborhoods / 100 junctions"
		m.Columns, m.Rows = coordinates(10, 64), coordinates(10, 60)
		m.SpeedLimit = 12
	case "barrio":
		m.Name = "Barrio Jardín"
		m.Description = "Quiet streets, T-junctions and neighborhood parks"
		m.Columns = []float64{-48, 0, 48}
		m.Rows = []float64{-44, 0, 44}
		m.SpeedLimit = 7
	case "avenida":
		m.Name = "Gran Avenida"
		m.Description = "A long boulevard, four cross streets and mixed traffic"
		m.Columns = []float64{-78, -26, 26, 78}
		m.Rows = []float64{-38, 0, 38}
		m.SpeedLimit = 12
	default:
		m.ID = "centro"
		m.Name = "Centro Urbano"
		m.Description = "Compact blocks, busy intersections and turning traffic"
	}
	cols := len(m.Columns)
	large := cols >= 5
	nodeID := func(r, c int) string { return fmt.Sprintf("INT-%02d", r*cols+c+1) }
	add := func(from, to string, a, b Point, avenue bool) {
		m.Roads = append(m.Roads, Road{fmt.Sprintf("RD-%02d", len(m.Roads)+1), from, to, a, b, avenue})
	}
	for r, z := range m.Rows {
		for c, x := range m.Columns {
			id := nodeID(r, c)
			m.Nodes = append(m.Nodes, Node{id, fmt.Sprintf("%s / %02d", m.Name, r*cols+c+1), x, z})
			// Canal crossings are limited to three bridges; other streets end at the water.
			horizontal := !(m.ID == "ribera" && c == 2 && r != 0 && r != 1 && r != 4)
			if c+1 < cols && horizontal {
				add(id, nodeID(r, c+1), Point{x, z}, Point{m.Columns[c+1], z}, r == 1 || (large && r == len(m.Rows)/2))
			}
			// Removed internal links make real T-junctions; routing must go around them.
			missing := (m.ID == "barrio" && ((r == 0 && c == 0) || (r == 1 && c == 2))) || (m.ID == "parque" && r == 3 && (c == 1 || c == 5)) || (m.ID == "metropolis" && r%3 == 2 && c%3 == 1)
			if r+1 < len(m.Rows) && !missing {
				add(id, nodeID(r+1, c), Point{x, z}, Point{x, m.Rows[r+1]}, large && (c == cols/2 || c == 1))
			}
			if c == 0 {
				add("", id, Point{x - 28, z}, Point{x, z}, r == 1)
			}
			if c == cols-1 {
				add(id, "", Point{x, z}, Point{x + 28, z}, r == 1)
			}
			if r == 0 {
				add("", id, Point{x, z - 28}, Point{x, z}, false)
			}
			if r == len(m.Rows)-1 {
				add(id, "", Point{x, z}, Point{x, z + 28}, false)
			}
		}
	}
	xs := append([]float64{m.Columns[0] - 40}, m.Columns...)
	xs = append(xs, m.Columns[cols-1]+40)
	zs := append([]float64{m.Rows[0] - 40}, m.Rows...)
	zs = append(zs, m.Rows[len(m.Rows)-1]+40)
	for r := 0; r < len(zs)-1; r++ {
		for c := 0; c < len(xs)-1; c++ {
			kind := "buildings"
			if large {
				kind = "residential"
				if c >= cols/2-1 && c <= cols/2+1 && r >= len(m.Rows)/2-1 && r <= len(m.Rows)/2+1 {
					kind = "towers"
				}
				if m.ID == "metropolis" && c >= cols-1 && r >= len(m.Rows)-2 {
					kind = "industrial"
				}
				if (r*3+c)%7 == 0 || (m.ID == "parque" && (c == 3 || r == 3)) {
					kind = "park"
				}
			}
			if (r == 1 && c == 2) || (m.ID == "barrio" && (r+c)%3 == 0) {
				kind = "park"
			}
			if r == 2 && c == len(xs)-2 {
				kind = "hospital"
			}
			if r == 2 && c == 0 {
				kind = "depot"
			}
			if m.ID == "ribera" && c == 3 {
				kind = "water"
			}
			m.Lots = append(m.Lots, Lot{(xs[c] + xs[c+1]) / 2, (zs[r] + zs[r+1]) / 2, xs[c+1] - xs[c] - 13, zs[r+1] - zs[r] - 13, kind})
		}
	}
	m.Width = xs[len(xs)-1] - xs[0]
	m.Depth = zs[len(zs)-1] - zs[0]
	// Immutable indexes are shared safely by world clones and stay out of JSON.
	m.nodeIndex, m.adjacency = map[string]Node{}, map[string][]string{}
	for _, n := range m.Nodes {
		m.nodeIndex[n.ID] = n
	}
	for _, road := range m.Roads {
		if road.From != "" && road.To != "" {
			m.adjacency[road.From] = append(m.adjacency[road.From], road.To)
			m.adjacency[road.To] = append(m.adjacency[road.To], road.From)
		}
	}
	return m
}
func (m Map) Node(id string) Node {
	if m.nodeIndex != nil {
		return m.nodeIndex[id]
	}
	for _, n := range m.Nodes {
		if n.ID == id {
			return n
		}
	}
	return Node{}
}
func (m Map) Neighbors(id string) []string {
	if m.adjacency != nil {
		return m.adjacency[id]
	}
	out := []string{}
	for _, r := range m.Roads {
		if r.From == id && r.To != "" {
			out = append(out, r.To)
		}
		if r.To == id && r.From != "" {
			out = append(out, r.From)
		}
	}
	return out
}
func (m Map) Entry(road int, dir string) string {
	r, c := road, 0
	switch dir {
	case "W":
		c = len(m.Columns) - 1
	case "S":
		r = 0
		c = road
	case "N":
		r = len(m.Rows) - 1
		c = road
	}
	if r < 0 || r >= len(m.Rows) || c < 0 || c >= len(m.Columns) {
		return ""
	}
	return fmt.Sprintf("INT-%02d", r*len(m.Columns)+c+1)
}
func (m Map) Exits(id string) []string {
	n := m.Node(id)
	out := []string{}
	if n.X == m.Columns[0] {
		out = append(out, "W")
	}
	if n.X == m.Columns[len(m.Columns)-1] {
		out = append(out, "E")
	}
	if n.Z == m.Rows[0] {
		out = append(out, "N")
	}
	if n.Z == m.Rows[len(m.Rows)-1] {
		out = append(out, "S")
	}
	return out
}
