//go:build integration

package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"trafficos/apps/api/internal/application"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/storage"
	"trafficos/packages/demand"
)

func setting(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}
func TestPostgresNATSWebSocketEndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	store, err := storage.Open(ctx, setting("DATABASE_URL", "postgres://trafficos:trafficos@localhost:5432/trafficos?sslmode=disable"))
	if err != nil {
		t.Fatal("Start real dependencies before this test: ", err)
	}
	defer store.Pool.Close()
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	hub := realtime.New()
	deps, err := storage.Connect(setting("REDIS_URL", "redis://localhost:6379/0"), setting("NATS_URL", "nats://localhost:4222"), hub)
	if err != nil {
		t.Fatal(err)
	}
	defer deps.Close()
	go deps.Relay(ctx, store)
	service := application.New(store, hub)
	service.NativeBus = true
	server := httptest.NewServer(New(service, deps, []string{"http://localhost:3000"}, "integration-token").Handler())
	defer server.Close()
	conn, _, err := websocket.Dial(ctx, strings.Replace(server.URL, "http", "ws", 1)+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(2 * 1024 * 1024)
	changes := make(chan bool, 1)
	go func() {
		for {
			_, raw, err := conn.Read(ctx)
			if err != nil {
				changes <- false
				return
			}
			var envelope struct {
				Type    string `json:"type"`
				Payload struct {
					Type string `json:"type"`
				} `json:"payload"`
			}
			if json.Unmarshal(raw, &envelope) == nil && envelope.Type == "event" && envelope.Payload.Type == "signal.phase.changed" {
				changes <- true
				return
			}
		}
	}()
	gen := demand.Generator{Seed: 42}
	session := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	for seq := int64(1); seq <= 25; seq++ {
		frames := []application.Frame{}
		for k := 0; k < 10; k++ {
			frames = append(frames, application.Frame{DT: .2, Arrivals: gen.Arrivals("GRIDLOCK", .2)})
		}
		raw, _ := json.Marshal(application.Telemetry{Session: session, Sequence: seq, Frames: frames})
		req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/api/v1/telemetry", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer integration-token")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("%d: %s", res.StatusCode, body)
		}
	}
	select {
	case ok := <-changes:
		if !ok {
			t.Fatal("WebSocket closed before relayed event")
		}
	case <-ctx.Done():
		t.Fatal("NATS event not delivered")
	}
	_, decisions := service.History()
	if len(decisions) == 0 {
		t.Fatal("no decisions")
	}
	var found int
	if err = store.Pool.QueryRow(ctx, "SELECT count(*) FROM optimization_decisions WHERE id=$1", decisions[0].ID).Scan(&found); err != nil || found != 1 {
		t.Fatalf("decision was not persisted: %v count=%d", err, found)
	}
	raw, _ := json.Marshal(service.Snapshot())
	cacheKey := "trafficos:integration:" + session
	if err = deps.Redis.Set(ctx, cacheKey, raw, 5*time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	cached, err := deps.Redis.Get(ctx, cacheKey).Bytes()
	if err != nil || !bytes.Equal(raw, cached) {
		t.Fatal("Redis snapshot roundtrip failed")
	}
	var count int
	err = store.Pool.QueryRow(ctx, "SELECT count(*) FROM events WHERE correlation_id=$1", decisions[0].CorrelationID).Scan(&count)
	if err != nil || count < 2 {
		t.Fatal("correlated audit missing")
	}
}
