package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"trafficos/apps/api/internal/application"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/storage"
	"trafficos/packages/demand"
)

func testServer(t *testing.T) (*application.Service, *httptest.Server) {
	t.Helper()
	s := application.New(storage.NewMemory(), realtime.New())
	server := httptest.NewServer(New(s, nil, []string{"http://localhost:3000"}, "test-token").Handler())
	t.Cleanup(server.Close)
	return s, server
}
func TestHTTPValidationAndCORS(t *testing.T) {
	_, server := testServer(t)
	for _, tc := range []struct {
		path, body, origin string
		want               int
	}{{"simulation/profile", `{"profile":"NOT_REAL"}`, "", 400}, {"simulation/control", `{"speed":-1}`, "", 400}, {"incidents", `{"intersectionId":"missing","type":"ROADWORK"}`, "", 404}, {"simulation/profile", `{"profile":"LOW"}`, "https://evil.test", 403}, {"telemetry", `{}`, "", 401}, {"simulation/profile", `null`, "", 400}} {
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/"+tc.path, strings.NewReader(tc.body))
		req.Header.Set("Origin", tc.origin)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != tc.want {
			t.Errorf("%s: %d != %d", tc.path, res.StatusCode, tc.want)
		}
		res.Body.Close()
	}
}
func TestHTTPToWebSocketFlow(t *testing.T) {
	s, server := testServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(server.URL, "http", "ws", 1)+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(2 * 1024 * 1024)
	// Drain concurrently so backpressure is part of the tested real transport.
	seen := make(chan map[string]bool, 1)
	snapshots := make(chan struct{}, 1)
	go func() {
		found := map[string]bool{}
		reported := false
		for {
			_, raw, err := conn.Read(ctx)
			if err != nil {
				if !reported {
					seen <- found
				}
				return
			}
			var msg struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}
			if json.Unmarshal(raw, &msg) == nil {
				if msg.Type == "snapshot" {
					select {
					case snapshots <- struct{}{}:
					case <-ctx.Done():
						return
					}
				}
				if msg.Type == "event" {
					var e struct {
						Type string `json:"type"`
					}
					if json.Unmarshal(msg.Payload, &e) == nil {
						found[e.Type] = true
					}
				}
				if msg.Type == "events" {
					var batch []struct {
						Type string `json:"type"`
					}
					if json.Unmarshal(msg.Payload, &batch) == nil {
						for _, e := range batch {
							found[e.Type] = true
						}
					}
				}
				if !reported && found["signal.phase.changed"] && found["optimizer.decision.created"] && found["traffic.queue.threshold_exceeded"] {
					copyFound := map[string]bool{}
					for k, v := range found {
						copyFound[k] = v
					}
					seen <- copyFound
					reported = true
				}
			}
		}
	}()
	select {
	case <-snapshots:
	case <-ctx.Done():
		t.Fatal("initial snapshot missing")
	}
	gen := demand.Generator{Seed: 42}
	for seq := int64(1); seq <= 250; seq++ {
		_, err = s.Ingest(ctx, application.Telemetry{Session: "e2e", Sequence: seq, Frames: []application.Frame{{DT: .2, Arrivals: gen.Arrivals("GRIDLOCK", .2)}}})
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-snapshots:
		case <-ctx.Done():
			t.Fatal("telemetry snapshot missing")
		}
	}
	select {
	case found := <-seen:
		for _, kind := range []string{"traffic.queue.threshold_exceeded", "optimizer.calculation.started", "optimizer.decision.created", "signal.phase.change.requested", "signal.phase.changed"} {
			if !found[kind] {
				t.Errorf("missing WS event %s", kind)
			}
		}
	case <-ctx.Done():
		t.Fatal("WebSocket flow timed out")
	}
	es, ds := s.History()
	if len(es) == 0 || len(ds) == 0 {
		t.Fatal("audit not recorded")
	}
	corr := map[string]map[string]bool{}
	for _, e := range es {
		if corr[e.CorrelationID] == nil {
			corr[e.CorrelationID] = map[string]bool{}
		}
		corr[e.CorrelationID][e.Type] = true
	}
	linked := false
	for _, kinds := range corr {
		if kinds["traffic.queue.threshold_exceeded"] && kinds["optimizer.decision.created"] && kinds["signal.phase.changed"] {
			linked = true
		}
	}
	if !linked {
		t.Fatal("causal chain not correlated")
	}
}
func TestEmergencyHTTPReplay(t *testing.T) {
	_, server := testServer(t)
	responses := []string{}
	for k := 0; k < 2; k++ {
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/emergency-vehicles", bytes.NewBufferString(`{"priority":2}`))
		req.Header.Set("Idempotency-Key", "http-replay")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 201 {
			t.Fatal(res.Status)
		}
		var obj map[string]any
		json.NewDecoder(res.Body).Decode(&obj)
		res.Body.Close()
		responses = append(responses, obj["id"].(string))
	}
	if responses[0] != responses[1] {
		t.Fatal("duplicate HTTP emergency")
	}
}
