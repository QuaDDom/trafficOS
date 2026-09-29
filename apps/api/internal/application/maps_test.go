package application

import (
	"context"
	"testing"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/storage"
	"trafficos/packages/demand"
)

func TestMapSwitchValidatesResetsAndRejectsStaleFrames(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	ctx := context.Background()
	old := s.Snapshot().Epoch
	if _, _, err := s.Execute(ctx, "map", "bad", []byte(`{"mapId":"unknown"}`), "test"); err == nil {
		t.Fatal("invalid map accepted")
	}
	if s.Snapshot().Epoch != old {
		t.Fatal("invalid command mutated world")
	}
	if _, _, err := s.Execute(ctx, "map", "avenue", []byte(`{"mapId":"avenida"}`), "test"); err != nil {
		t.Fatal(err)
	}
	next := s.Snapshot()
	if next.Epoch == old || len(next.Intersections) != 12 {
		t.Fatal("map not replaced")
	}
	stale := Telemetry{Epoch: old, Session: "map-test", Sequence: 1, Frames: []Frame{{DT: .2, Arrivals: []demand.Arrival{{Road: 0, Direction: "E"}}}}}
	if _, err := s.Ingest(ctx, stale); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Time != 0 {
		t.Fatal("old map telemetry advanced new world")
	}
	stale.Epoch = next.Epoch
	stale.Sequence = 2
	stale.Frames[0].Arrivals[0] = demand.Arrival{Road: 3, Direction: "S"}
	if _, err := s.Ingest(ctx, stale); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot().Vehicles) != 1 {
		t.Fatal("fourth avenue entry missing")
	}
	if _, _, err := s.Execute(ctx, "reset", "reset", []byte(`{}`), "test"); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Map.ID != "avenida" {
		t.Fatal("reset discarded map choice")
	}
}
func TestMapChangeRollsBackOnStorageFailure(t *testing.T) {
	s := New(failingStore{storage.NewMemory()}, realtime.New())
	before := s.Snapshot()
	if _, _, err := s.Execute(context.Background(), "map", "map", []byte(`{"mapId":"barrio"}`), "test"); err == nil {
		t.Fatal("expected failure")
	}
	if s.Snapshot().Epoch != before.Epoch || s.Snapshot().Map.ID != before.Map.ID {
		t.Fatal("uncommitted map became visible")
	}
}
