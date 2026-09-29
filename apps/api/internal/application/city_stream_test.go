package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/storage"
	"trafficos/packages/demand"
)

func TestMetropolisCommitDoesNotOverflowLiveSubscriber(t *testing.T) {
	hub := realtime.New()
	s := New(storage.NewMemory(), hub)
	ctx := context.Background()
	if _, _, err := s.Execute(ctx, "map", "", []byte(`{"mapId":"metropolis"}`), "test"); err != nil {
		t.Fatal(err)
	}
	s.World.Time = 1.9 // The next tick calculates decisions at all 100 intersections.
	ch, _, done := s.Subscribe()
	defer done()
	if _, err := s.Ingest(ctx, Telemetry{Epoch: s.World.Epoch, Session: "city", Sequence: 1, Frames: []Frame{{DT: .2}}}); err != nil {
		t.Fatal(err)
	}
	if hub.Count() != 1 {
		t.Fatal("one normal city tick disconnected the subscriber")
	}
	var first struct{ Type string }
	json.Unmarshal(readLive(t, ch), &first)
	if first.Type != "snapshot" {
		t.Fatal("snapshot must establish epoch before its events")
	}
	var batch struct {
		Type    string
		Payload []json.RawMessage
	}
	json.Unmarshal(readLive(t, ch), &batch)
	if batch.Type != "events" || len(batch.Payload) < 100 {
		t.Fatalf("expected complete event batch, got %s/%d", batch.Type, len(batch.Payload))
	}
}

func TestResetRetainsExperimentSettings(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	ctx := context.Background()
	s.World.Profile = "LOW"
	s.World.Mode = "FIXED"
	s.World.Speed = 2
	s.World.Paused = true
	if _, _, err := s.Execute(ctx, "reset", "", []byte(`{}`), "test"); err != nil {
		t.Fatal(err)
	}
	w := s.Snapshot()
	if w.Profile != "LOW" || w.Mode != "FIXED" || w.Speed != 2 || !w.Paused {
		t.Fatal("reset silently changed experiment configuration")
	}
}

func readLive(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	select {
	case raw := <-ch:
		return raw
	case <-time.After(time.Second):
		t.Fatal("missing live message")
		return nil
	}
}
func TestTelemetryAcceptsFarCityBoundaryAndRejectsOutsideMap(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	ctx := context.Background()
	if _, _, err := s.Execute(ctx, "map", "", []byte(`{"mapId":"metropolis"}`), "test"); err != nil {
		t.Fatal(err)
	}
	packet := Telemetry{Epoch: s.World.Epoch, Session: "large", Sequence: 1, Frames: []Frame{{DT: .2, Arrivals: []demand.Arrival{{Road: 9, Direction: "E", Type: "BUS", Occupants: 40}}}}}
	if _, err := s.Ingest(ctx, packet); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot().Vehicles) != 1 {
		t.Fatal("arrival lost")
	}
	packet.Sequence++
	packet.Frames[0].Arrivals[0].Road = 10
	if _, err := s.Ingest(ctx, packet); err == nil {
		t.Fatal("out-of-map arrival accepted")
	}
}
