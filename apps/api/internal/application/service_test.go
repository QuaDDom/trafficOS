package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/storage"
)

func TestConcurrentIdempotency(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	var wg sync.WaitGroup
	ids := make(chan string, 20)
	for k := 0; k < 20; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw, status, err := s.Execute(context.Background(), "emergency", "same", []byte(`{"priority":2}`), "req_test")
			if err != nil || status != 201 {
				t.Errorf("%d %v", status, err)
				return
			}
			var result struct {
				ID string `json:"id"`
			}
			json.Unmarshal(raw, &result)
			ids <- result.ID
		}()
	}
	wg.Wait()
	close(ids)
	id := ""
	for got := range ids {
		if id != "" && id != got {
			t.Fatal("retry duplicated emergency")
		}
		id = got
	}
	if len(s.Snapshot().Corridors) != 1 || len(s.Snapshot().Vehicles) != 1 {
		t.Fatal("duplicate side effect")
	}
	_, _, err := s.Execute(context.Background(), "emergency", "same", []byte(`{"priority":1}`), "req_conflict")
	var fault *Fault
	if !errors.As(err, &fault) || fault.Status != 409 {
		t.Fatalf("expected conflict: %v", err)
	}
}

type failingStore struct{ *storage.Memory }

func (f failingStore) Commit(context.Context, storage.Batch) error {
	return fmt.Errorf("database offline")
}
func TestFailedCommitRollsBackState(t *testing.T) {
	s := New(failingStore{storage.NewMemory()}, realtime.New())
	before := s.Snapshot()
	_, _, err := s.Execute(context.Background(), "profile", "", []byte(`{"profile":"GRIDLOCK"}`), "req_fail")
	if err == nil {
		t.Fatal("expected storage failure")
	}
	if s.Snapshot().Profile != before.Profile || s.Snapshot().Metrics.EventsTotal != before.Metrics.EventsTotal {
		t.Fatal("uncommitted state leaked")
	}
	es, _ := s.History()
	if len(es) != 0 {
		t.Fatal("uncommitted audit leaked")
	}
}
func TestTelemetryIdempotencyAndOrdering(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	p := Telemetry{Session: "test", Sequence: 1, Frames: []Frame{{DT: .2}}}
	if _, err := s.Ingest(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Ingest(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Time != .2 {
		t.Fatal("duplicate advanced simulation")
	}
	p.Sequence = 3
	if _, err := s.Ingest(context.Background(), p); err == nil {
		t.Fatal("sequence gap accepted")
	}
	p.Sequence = 2
	p.Session = "rival"
	if _, err := s.Ingest(context.Background(), p); err == nil {
		t.Fatal("second active producer accepted")
	}
}
func TestFallbackAndRecovery(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	for _, online := range []bool{false, true} {
		_, _, err := s.Execute(context.Background(), "chaos", "", []byte(fmt.Sprintf(`{"type":"optimizer","online":%t}`, online)), "req_chaos")
		if err != nil {
			t.Fatal(err)
		}
		for _, i := range s.Snapshot().Intersections {
			want := "SAFE_MODE"
			if online {
				want = "ADAPTIVE"
			}
			if i.ControllerMode != want {
				t.Fatalf("%s != %s", i.ControllerMode, want)
			}
		}
	}
}
func TestUnsafeCommandAudited(t *testing.T) {
	s := New(storage.NewMemory(), realtime.New())
	before := s.Snapshot().Intersections[0]
	_, _, err := s.Execute(context.Background(), "signal", "", []byte(`{"intersectionId":"INT-01","phase":"NS_GREEN","version":1}`), "request-unsafe")
	if err == nil {
		t.Fatal("unsafe command accepted")
	}
	after := s.Snapshot().Intersections[0]
	if before.Phase != after.Phase || before.Version != after.Version {
		t.Fatal("rejected command changed signal")
	}
	es, _ := s.History()
	if len(es) != 2 || es[1].Type != "signal.transition.rejected" || es[1].CorrelationID != "request-unsafe" {
		t.Fatal("missing correlated rejection")
	}
}
