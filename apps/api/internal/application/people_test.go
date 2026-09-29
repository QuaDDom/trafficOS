package application

import (
	"context"
	"testing"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/storage"
)

func TestPriorityAndFleetCommands(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	ctx := context.Background()
	for _, body := range []string{`{"priorityMode":"BUS_ALWAYS"}`, `{"priorityMode":"PEOPLE","speed":3}`} {
		if _, _, err := s.Execute(ctx, "control", "", []byte(body), "test"); err == nil {
			t.Fatal("invalid config accepted")
		}
		if s.World.PriorityMode != "VEHICLES" {
			t.Fatal("partial invalid change leaked")
		}
	}
	if _, _, err := s.Execute(ctx, "control", "priority", []byte(`{"priorityMode":"PEOPLE"}`), "test"); err != nil {
		t.Fatal(err)
	}
	raw, code, err := s.Execute(ctx, "vehicle", "bus", []byte(`{"type":"BUS","occupants":40,"road":1,"direction":"E"}`), "test")
	if err != nil || code != 201 {
		t.Fatalf("spawn failed %d %s %v", code, raw, err)
	}
	if _, _, err = s.Execute(ctx, "vehicle", "bus", []byte(`{"type":"BUS","occupants":40,"road":1,"direction":"E"}`), "test"); err != nil || len(s.World.Vehicles) != 1 {
		t.Fatal("idempotent spawn duplicated")
	}
	if _, _, err = s.Execute(ctx, "vehicle", "", []byte(`{"type":"SUV","occupants":60,"road":1,"direction":"W"}`), "test"); err == nil {
		t.Fatal("SUV overcapacity accepted")
	}
	for _, cmd := range []struct{ kind, body string }{{"map", `{"mapId":"avenida"}`}, {"reset", `{}`}} {
		if _, _, err = s.Execute(ctx, cmd.kind, "", []byte(cmd.body), "test"); err != nil {
			t.Fatal(err)
		}
		if s.World.PriorityMode != "PEOPLE" {
			t.Fatal("setting lost on map/reset")
		}
	}
}
func TestPriorityRollback(t *testing.T) {
	s := New(failingStore{storage.NewMemory()}, realtime.New())
	if _, _, err := s.Execute(context.Background(), "control", "", []byte(`{"priorityMode":"PEOPLE"}`), "test"); err == nil {
		t.Fatal("expected storage failure")
	}
	if s.World.PriorityMode != "VEHICLES" {
		t.Fatal("uncommitted policy became visible")
	}
}
