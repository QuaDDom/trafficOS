package storage

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"time"
	"trafficos/apps/api/internal/events"
	"trafficos/apps/api/internal/observability"
	"trafficos/apps/api/internal/realtime"
)

type Dependencies struct {
	Redis *redis.Client
	NATS  *nats.Conn
	JS    nats.JetStreamContext
}

func Connect(redisURL, natsURL string, hub *realtime.Hub) (*Dependencies, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	opt.DialTimeout = time.Second
	opt.ReadTimeout = time.Second
	opt.WriteTimeout = time.Second
	opt.MaxRetries = 2
	d := &Dependencies{Redis: redis.NewClient(opt)}
	d.NATS, err = nats.Connect(natsURL, nats.Timeout(2*time.Second), nats.MaxReconnects(-1), nats.ReconnectWait(time.Second), nats.DisconnectErrHandler(func(_ *nats.Conn, err error) { slog.Warn("NATS disconnected", "error", err) }))
	if err != nil {
		d.Redis.Close()
		return nil, err
	}
	d.JS, err = d.NATS.JetStream()
	if err != nil {
		d.Close()
		return nil, err
	}
	if _, err = d.JS.StreamInfo("TRAFFICOS"); err != nil {
		if _, err = d.JS.AddStream(&nats.StreamConfig{Name: "TRAFFICOS", Subjects: []string{"trafficos.events"}, Storage: nats.FileStorage, MaxAge: 24 * time.Hour, MaxBytes: 128 * 1024 * 1024, Duplicates: 10 * time.Minute}); err != nil {
			d.Close()
			return nil, err
		}
	}
	// A live observer subscription is intentionally ephemeral; PostgreSQL is the
	// durable audit trail. JetStream deduplicates retried outbox publications.
	_, err = d.NATS.Subscribe("trafficos.events", func(m *nats.Msg) {
		var e events.Event
		if json.Unmarshal(m.Data, &e) == nil {
			hub.Broadcast(map[string]any{"type": "event", "payload": e})
		}
	})
	if err != nil {
		d.Close()
		return nil, err
	}
	if err = d.NATS.FlushTimeout(time.Second); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}
func (d *Dependencies) Close() {
	if d.NATS != nil {
		d.NATS.Close()
	}
	if d.Redis != nil {
		d.Redis.Close()
	}
}
func (d *Dependencies) Relay(ctx context.Context, p *Postgres) {
	delay := 100 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		batchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		records, err := p.Pending(batchCtx)
		if err == nil {
			for _, r := range records {
				_, err = d.JS.Publish("trafficos.events", r.Payload, nats.MsgId(r.ID), nats.Context(batchCtx))
				if err != nil {
					break
				}
				if err = p.Published(batchCtx, r.ID); err != nil {
					break
				}
			}
		}
		cancel()
		if err != nil {
			observability.OutboxFailures.Inc()
			slog.Warn("outbox publish deferred", "error", err)
			delay = min(5*time.Second, delay*2)
		} else {
			delay = 100 * time.Millisecond
		}
	}
}
