//go:build integration

package storage

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
	"trafficos/apps/api/internal/optimization"
	"trafficos/apps/api/internal/signals"
	"trafficos/apps/api/internal/simulation"
)

func TestPostgresCommitAndIdempotency(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://trafficos:trafficos@localhost:5432/trafficos?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	p, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Pool.Close()
	if err = p.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	w := simulation.New(42)
	corr := w.ID("test")
	d := optimization.Decide(optimization.Input{IntersectionID: "INT-05", Phase: signals.NSGreen, PhaseDuration: 20, NSQueue: 1, EWQueue: 20})
	d.ID = w.ID("decision")
	d.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	d.CorrelationID = corr
	w.Emit("optimizer.decision.created", "optimizer", corr, d)
	c, err := w.SpawnEmergency(2, corr)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(c)
	idem := &Idempotency{Key: w.ID("key"), Fingerprint: "abc", Status: 201, Response: raw}
	if err = p.Commit(ctx, Batch{World: w, Checkpoint: true, Events: w.PendingEvents, Decisions: []optimization.Decision{d}, Idempotency: idem}); err != nil {
		t.Fatal(err)
	}
	found, err := p.Lookup(ctx, idem.Key)
	if err != nil || found == nil || found.Fingerprint != "abc" {
		t.Fatal("idempotency not persisted", err)
	}
	var actual map[string]any
	if err = json.Unmarshal(found.Response, &actual); err != nil || actual["id"] != c.ID {
		t.Fatal("response replay mismatch")
	}
	rows, err := p.Rows(ctx, "optimization_decisions")
	if err != nil || len(rows) == 0 {
		t.Fatal("decision not persisted", err)
	}
	var count int
	if err = p.Pool.QueryRow(ctx, "SELECT count(*) FROM event_outbox WHERE id=$1", w.PendingEvents[0].ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("transactional outbox missing", err)
	}
	// The duplicate idempotency insert must roll back the entire transaction.
	w.Emit("test.rollback", "integration", corr, map[string]string{"check": "rollback"})
	last := w.PendingEvents[len(w.PendingEvents)-1]
	if err = p.Commit(ctx, Batch{Events: w.PendingEvents, Idempotency: idem}); err == nil {
		t.Fatal("expected constraint violation")
	}
	if err = p.Pool.QueryRow(ctx, "SELECT count(*) FROM events WHERE id=$1", last.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial audit commit", err)
	}
}
func TestPostgresMapCheckpointReplacesCurrentGeometry(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://trafficos:trafficos@localhost:5432/trafficos?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	p, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Pool.Close()
	if err = p.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	avenue := simulation.NewMap(42, "avenida")
	avenue.Emit("simulation.map.changed", "integration", "map-test", map[string]string{"mapId": "avenida"})
	eventID := avenue.PendingEvents[0].ID
	if err = p.Commit(ctx, Batch{World: avenue, Checkpoint: true, Events: avenue.PendingEvents}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = p.Pool.QueryRow(ctx, "SELECT count(*) FROM intersections").Scan(&count); err != nil || count != 12 {
		t.Fatal("avenue not persisted", count, err)
	}
	barrio := simulation.NewMap(42, "barrio")
	if err = p.Commit(ctx, Batch{World: barrio, Checkpoint: true}); err != nil {
		t.Fatal(err)
	}
	if err = p.Pool.QueryRow(ctx, "SELECT count(*) FROM intersections").Scan(&count); err != nil || count != 9 {
		t.Fatal("stale map nodes remain", count, err)
	}
	var x float64
	if err = p.Pool.QueryRow(ctx, "SELECT x FROM intersections WHERE id='INT-01'").Scan(&x); err != nil || x != -48 {
		t.Fatal("map coordinates not updated", x, err)
	}
	if err = p.Pool.QueryRow(ctx, "SELECT count(*) FROM events WHERE id=$1", eventID).Scan(&count); err != nil || count != 1 {
		t.Fatal("map switch deleted historical audit", err)
	}
}
