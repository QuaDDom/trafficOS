package network

import "testing"

func TestCatalogConnectivityAndEntries(t *testing.T) {
	expected := map[string]int{"centro": 9, "barrio": 9, "avenida": 12, "ribera": 25, "parque": 49, "metropolis": 100}
	for _, m := range Catalog() {
		t.Run(m.ID, func(t *testing.T) {
			if !Valid(m.ID) || len(m.Nodes) != expected[m.ID] {
				t.Fatal("invalid map size")
			}
			seen := map[string]bool{m.Nodes[0].ID: true}
			queue := []string{m.Nodes[0].ID}
			for len(queue) > 0 {
				id := queue[0]
				queue = queue[1:]
				for _, next := range m.Neighbors(id) {
					if !seen[next] {
						seen[next] = true
						queue = append(queue, next)
					}
				}
			}
			if len(seen) != len(m.Nodes) {
				t.Fatal("disconnected district")
			}
			for _, dir := range []string{"N", "S", "E", "W"} {
				count := len(m.Rows)
				if dir == "N" || dir == "S" {
					count = len(m.Columns)
				}
				for road := 0; road < count; road++ {
					if !seen[m.Entry(road, dir)] {
						t.Fatalf("unreachable entry %s/%d", dir, road)
					}
				}
				if m.Entry(count, dir) != "" {
					t.Fatal("out-of-range entry accepted")
				}
			}
			for _, r := range m.Roads {
				if r.From != "" {
					n := m.Node(r.From)
					if n.X != r.A.X || n.Z != r.A.Z {
						t.Fatal("road endpoint mismatch")
					}
				}
				if r.To != "" {
					n := m.Node(r.To)
					if n.X != r.B.X || n.Z != r.B.Z {
						t.Fatal("road endpoint mismatch")
					}
				}
			}
		})
	}
	if Valid("unknown") {
		t.Fatal("unknown map accepted")
	}
}

func TestRiberaHasOnlyThreeCanalBridges(t *testing.T) {
	m := Get("ribera")
	bridges := 0
	for _, r := range m.Roads {
		if r.A.X == m.Columns[2] && r.B.X == m.Columns[3] {
			bridges++
		}
	}
	if bridges != 3 {
		t.Fatalf("got %d bridges", bridges)
	}
}
