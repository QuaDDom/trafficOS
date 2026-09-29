package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sync"
	"time"
	"trafficos/apps/api/internal/events"
	"trafficos/apps/api/internal/optimization"
	"trafficos/apps/api/internal/simulation"
	"trafficos/infrastructure/migrations"
)

type Idempotency struct {
	Key         string
	Fingerprint string
	Status      int
	Response    json.RawMessage
}
type Batch struct {
	Events      []events.Event
	Decisions   []optimization.Decision
	World       *simulation.World
	Checkpoint  bool
	Idempotency *Idempotency
}
type Store interface {
	Commit(context.Context, Batch) error
	Lookup(context.Context, string) (*Idempotency, error)
	Rows(context.Context, string) ([]json.RawMessage, error)
	Ping(context.Context) error
	Mode() string
}
type Memory struct {
	mu   sync.Mutex
	keys map[string]Idempotency
}

func NewMemory() *Memory { return &Memory{keys: map[string]Idempotency{}} }
func (m *Memory) Commit(_ context.Context, b Batch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b.Idempotency != nil {
		m.keys[b.Idempotency.Key] = *b.Idempotency
	}
	return nil
}
func (m *Memory) Lookup(_ context.Context, key string) (*Idempotency, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.keys[key]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (m *Memory) Rows(context.Context, string) ([]json.RawMessage, error) {
	return nil, errors.New("PostgreSQL is disabled in explicit memory mode; no persisted rows")
}
func (m *Memory) Ping(context.Context) error { return nil }
func (m *Memory) Mode() string               { return "memory" }

type Postgres struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Postgres, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &Postgres{Pool: p}, nil
}
func (p *Postgres) Migrate(ctx context.Context) error {
	_, err := p.Pool.Exec(ctx, migrations.Initial)
	return err
}
func (p *Postgres) Mode() string                   { return "postgres" }
func (p *Postgres) Ping(ctx context.Context) error { return p.Pool.Ping(ctx) }
func (p *Postgres) Lookup(ctx context.Context, key string) (*Idempotency, error) {
	v := &Idempotency{Key: key}
	err := p.Pool.QueryRow(ctx, "SELECT fingerprint,status,response FROM idempotency_keys WHERE key=$1", key).Scan(&v.Fingerprint, &v.Status, &v.Response)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return v, err
}
func (p *Postgres) Commit(ctx context.Context, b Batch) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, e := range b.Events {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO events(id,type,timestamp,correlation_id,payload) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", e.ID, e.Type, e.Timestamp, e.CorrelationID, raw); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO event_outbox(id,payload) VALUES($1,$2) ON CONFLICT DO NOTHING", e.ID, raw); err != nil {
			return err
		}
		if e.Type == "signal.phase.changed" {
			payload, _ := json.Marshal(e.Payload)
			var fields struct {
				IntersectionID string `json:"intersectionId"`
			}
			json.Unmarshal(payload, &fields)
			if _, err = tx.Exec(ctx, "INSERT INTO signal_transitions(id,intersection_id,created_at,payload) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", e.ID, fields.IntersectionID, e.Timestamp, payload); err != nil {
				return err
			}
		}
	}
	for _, d := range b.Decisions {
		raw, _ := json.Marshal(d)
		if _, err = tx.Exec(ctx, "INSERT INTO optimization_decisions(id,intersection_id,created_at,correlation_id,payload) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", d.ID, d.IntersectionID, d.CreatedAt, d.CorrelationID, raw); err != nil {
			return err
		}
	}
	if b.Checkpoint && b.World != nil {
		ids := make([]string, 0, len(b.World.Intersections))
		for _, i := range b.World.Intersections {
			ids = append(ids, i.ID)
		}
		// These tables describe the current map. Historical audit remains in events.
		for _, query := range []string{"DELETE FROM signal_states WHERE NOT(intersection_id=ANY($1))", "DELETE FROM lanes WHERE NOT(intersection_id=ANY($1))", "DELETE FROM intersections WHERE NOT(id=ANY($1))"} {
			if _, err = tx.Exec(ctx, query, ids); err != nil {
				return err
			}
		}
		for _, i := range b.World.Intersections {
			raw, _ := json.Marshal(i)
			if _, err = tx.Exec(ctx, "INSERT INTO intersections(id,name,x,z) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,x=EXCLUDED.x,z=EXCLUDED.z", i.ID, i.Name, i.X, i.Z); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "INSERT INTO signal_states(intersection_id,payload) VALUES($1,$2) ON CONFLICT(intersection_id) DO UPDATE SET payload=EXCLUDED.payload,updated_at=now()", i.ID, raw); err != nil {
				return err
			}
			for _, l := range i.Lanes {
				if _, err = tx.Exec(ctx, "INSERT INTO lanes(id,intersection_id,direction,capacity) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", l.ID, i.ID, l.Direction, l.Capacity); err != nil {
					return err
				}
			}
		}
		raw, _ := json.Marshal(b.World.Metrics)
		if _, err = tx.Exec(ctx, "INSERT INTO traffic_snapshots(payload) VALUES($1)", raw); err != nil {
			return err
		}
	}
	if b.World != nil {
		for _, i := range b.World.Incidents {
			raw, _ := json.Marshal(i)
			if _, err = tx.Exec(ctx, "INSERT INTO incidents(id,payload) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET payload=EXCLUDED.payload", i.ID, raw); err != nil {
				return err
			}
		}
		for _, c := range b.World.Corridors {
			raw, _ := json.Marshal(c)
			if _, err = tx.Exec(ctx, "INSERT INTO emergency_corridors(id,payload) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET payload=EXCLUDED.payload", c.ID, raw); err != nil {
				return err
			}
			for _, r := range c.Reservations {
				rr, _ := json.Marshal(r)
				if _, err = tx.Exec(ctx, "INSERT INTO emergency_reservations(corridor_id,intersection_id,payload) VALUES($1,$2,$3) ON CONFLICT(corridor_id,intersection_id) DO UPDATE SET payload=EXCLUDED.payload", c.ID, r.IntersectionID, rr); err != nil {
					return err
				}
			}
		}
		for _, v := range b.World.Vehicles {
			if v.Type == "AMBULANCE" {
				raw, _ := json.Marshal(v)
				if _, err = tx.Exec(ctx, "INSERT INTO emergency_vehicles(id,payload) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET payload=EXCLUDED.payload", v.ID, raw); err != nil {
					return err
				}
			}
		}
	}
	if b.Idempotency != nil {
		v := b.Idempotency
		if _, err = tx.Exec(ctx, "INSERT INTO idempotency_keys(key,fingerprint,status,response) VALUES($1,$2,$3,$4)", v.Key, v.Fingerprint, v.Status, v.Response); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

var Tables = map[string]string{"optimization_decisions": "created_at", "events": "timestamp", "incidents": "created_at", "emergency_corridors": "created_at", "traffic_snapshots": "created_at", "signal_transitions": "created_at"}

func (p *Postgres) Rows(ctx context.Context, table string) ([]json.RawMessage, error) {
	column, ok := Tables[table]
	if !ok {
		return nil, errors.New("table not allowed")
	}
	rows, err := p.Pool.Query(ctx, "SELECT payload FROM "+table+" ORDER BY "+column+" DESC LIMIT 20")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var raw json.RawMessage
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, rows.Err()
}

type OutboxRecord struct {
	ID      string
	Payload []byte
}

func (p *Postgres) Pending(ctx context.Context) ([]OutboxRecord, error) {
	rows, err := p.Pool.Query(ctx, "SELECT id,payload FROM event_outbox WHERE published_at IS NULL ORDER BY created_at,id LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OutboxRecord{}
	for rows.Next() {
		var r OutboxRecord
		if err = rows.Scan(&r.ID, &r.Payload); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (p *Postgres) Published(ctx context.Context, id string) error {
	_, err := p.Pool.Exec(ctx, "UPDATE event_outbox SET published_at=$2 WHERE id=$1", id, time.Now())
	return err
}
