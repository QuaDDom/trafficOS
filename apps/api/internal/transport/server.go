package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"trafficos/apps/api/internal/application"
	"trafficos/apps/api/internal/observability"
	"trafficos/apps/api/internal/simulation"
	"trafficos/apps/api/internal/storage"
	"trafficos/packages/network"
)

type RequestRecord struct {
	ID         string            `json:"id"`
	Method     string            `json:"method"`
	Route      string            `json:"route"`
	Status     int               `json:"status"`
	DurationMs float64           `json:"durationMs"`
	Timestamp  time.Time         `json:"timestamp"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Response   string            `json:"response"`
}
type visitor struct {
	tokens  float64
	updated time.Time
}
type Server struct {
	Service        *application.Service
	Deps           *storage.Dependencies
	Origins        []string
	SimulatorToken string
	mu             sync.Mutex
	requests       []RequestRecord
	visitors       map[string]visitor
	comparison     chan struct{}
}

func New(s *application.Service, d *storage.Dependencies, origins []string, token string) *Server {
	return &Server{Service: s, Deps: d, Origins: origins, SimulatorToken: token, requests: []RequestRecord{}, visitors: map[string]visitor{}, comparison: make(chan struct{}, 1)}
}
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer, s.guard, s.instrument)
	r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "alive"}) })
	r.Get("/health/ready", s.health)
	r.Get("/api/v1/health", s.health)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/ws", s.ws)
	r.Get("/api/v1/maps", func(w http.ResponseWriter, r *http.Request) { write(w, 200, network.Catalog()) })
	r.Get("/api/v1/snapshot", func(w http.ResponseWriter, r *http.Request) { write(w, 200, s.Service.Snapshot()) })
	r.Get("/api/v1/simulation/config", func(w http.ResponseWriter, r *http.Request) { write(w, 200, s.Service.Config()) })
	r.Get("/api/v1/intersections", func(w http.ResponseWriter, r *http.Request) { write(w, 200, s.Service.Snapshot().Intersections) })
	for _, path := range []string{"/api/v1/intersections/{id}", "/api/v1/intersections/{id}/state"} {
		r.Get(path, func(w http.ResponseWriter, r *http.Request) {
			i := s.Service.Snapshot().Intersection(chi.URLParam(r, "id"))
			if i == nil {
				problem(w, r, application.Fail("NOT_FOUND", "Intersection not found", 404))
				return
			}
			write(w, 200, i)
		})
	}
	r.Get("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		es, _ := s.Service.History()
		filtered := es[:0]
		for _, e := range es {
			if q := r.URL.Query().Get("correlationId"); q == "" || e.CorrelationID == q {
				filtered = append(filtered, e)
			}
		}
		write(w, 200, filtered)
	})
	r.Get("/api/v1/decisions", func(w http.ResponseWriter, r *http.Request) { _, ds := s.Service.History(); write(w, 200, ds) })
	r.Get("/api/v1/decisions/{id}", func(w http.ResponseWriter, r *http.Request) {
		d, err := s.Service.Decision(chi.URLParam(r, "id"))
		if err != nil {
			problem(w, r, application.Fail("NOT_FOUND", err.Error(), 404))
			return
		}
		write(w, 200, d)
	})
	r.Get("/api/v1/requests", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		write(w, 200, s.requests)
	})
	r.Get("/api/v1/traffic/metrics", s.metrics)
	r.Get("/api/v1/database/{table}", func(w http.ResponseWriter, r *http.Request) {
		table := chi.URLParam(r, "table")
		if _, ok := storage.Tables[table]; !ok {
			problem(w, r, application.Fail("INVALID_TABLE", "Table not allowed", 400))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		rows, err := s.Service.Store.Rows(ctx, table)
		if err != nil {
			problem(w, r, application.Fail("DATABASE_UNAVAILABLE", err.Error(), 503))
			return
		}
		write(w, 200, rows)
	})
	for path, kind := range map[string]string{"vehicles": "vehicle", "simulation/map": "map", "simulation/profile": "profile", "simulation/control": "control", "simulation/reset": "reset", "emergency-vehicles": "emergency", "incidents": "incident", "signals/commands": "signal", "chaos": "chaos", "incidents/resolve": "resolve"} {
		r.Post("/api/v1/"+path, s.command(kind))
	}
	r.Post("/api/v1/telemetry", s.telemetry)
	r.Post("/api/v1/comparisons", s.compare)
	r.Get("/api/openapi.json", func(w http.ResponseWriter, r *http.Request) { write(w, 200, OpenAPI()) })
	r.Get("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, `<!doctype html><html><head><title>TrafficOS API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:'/api/openapi.json',dom_id:'#swagger-ui'})</script></body></html>`)
	})
	return r
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Debug("response write failed", "error", err)
	}
}
func problem(w http.ResponseWriter, r *http.Request, err error) {
	code, message, status := "INTERNAL_ERROR", "Unexpected server error", 500
	var f *application.Fault
	if errors.As(err, &f) {
		code, message, status = f.Code, f.Message, f.Status
	} else {
		slog.Error("request failed", "error", err)
	}
	write(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": w.Header().Get("X-Request-ID")}})
}
func readBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, application.Fail("BODY_TOO_LARGE", "Request body exceeds 128 KiB", 413)
	}
	if !json.Valid(raw) || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil, application.Fail("INVALID_JSON", "Expected a JSON object", 400)
	}
	return raw, nil
}
func (s *Server) command(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := readBody(w, r)
		if err != nil {
			problem(w, r, err)
			return
		}
		out, status, err := s.Service.Execute(r.Context(), kind, r.Header.Get("Idempotency-Key"), raw, w.Header().Get("X-Request-ID"))
		if err != nil {
			problem(w, r, err)
			return
		}
		write(w, status, out)
	}
}
func (s *Server) telemetry(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.SimulatorToken)) != 1 {
		problem(w, r, application.Fail("UNAUTHORIZED", "Simulator token required", 401))
		return
	}
	raw, err := readBody(w, r)
	if err != nil {
		problem(w, r, err)
		return
	}
	var packet application.Telemetry
	if err = json.Unmarshal(raw, &packet); err != nil {
		problem(w, r, application.Fail("INVALID_TELEMETRY", err.Error(), 400))
		return
	}
	out, err := s.Service.Ingest(r.Context(), packet)
	if err != nil {
		problem(w, r, err)
		return
	}
	write(w, 200, out)
}
func (s *Server) compare(w http.ResponseWriter, r *http.Request) {
	select {
	case s.comparison <- struct{}{}:
		defer func() { <-s.comparison }()
	default:
		problem(w, r, application.Fail("COMPARISON_BUSY", "One comparison is already running", 429))
		return
	}
	raw, err := readBody(w, r)
	if err != nil {
		problem(w, r, err)
		return
	}
	var in struct {
		Seed         uint64  `json:"seed"`
		Duration     float64 `json:"duration"`
		Profile      string  `json:"profile"`
		MapID        string  `json:"mapId"`
		PriorityMode string  `json:"priorityMode"`
	}
	if json.Unmarshal(raw, &in) != nil || in.Seed == 0 || in.Duration < 30 || in.Duration > 600 || !simulation.ValidProfile(in.Profile) {
		problem(w, r, application.Fail("INVALID_COMPARISON", "Use seed > 0, duration 30–600 seconds and a valid profile", 400))
		return
	}
	if in.MapID == "" {
		in.MapID = s.Service.Snapshot().Map.ID
	}
	if !network.Valid(in.MapID) {
		problem(w, r, application.Fail("INVALID_MAP", "Unknown map", 400))
		return
	}
	if in.PriorityMode == "" {
		in.PriorityMode = s.Service.Snapshot().PriorityMode
	}
	if in.PriorityMode != "VEHICLES" && in.PriorityMode != "PEOPLE" {
		problem(w, r, application.Fail("INVALID_PRIORITY_MODE", "Use VEHICLES or PEOPLE", 400))
		return
	}
	write(w, 200, simulation.ComparePriority(in.Seed, in.Duration, in.Profile, in.MapID, in.PriorityMode))
}
func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	patterns := []string{}
	for _, origin := range s.Origins {
		if u, err := url.Parse(origin); err == nil {
			patterns = append(patterns, u.Host)
		}
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: patterns})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	ch, initial, unsubscribe := s.Service.Subscribe()
	defer unsubscribe()
	observability.WS.Inc()
	defer observability.WS.Dec()
	ctx = conn.CloseRead(ctx)
	send := func(raw []byte) error {
		writeCtx, done := context.WithTimeout(ctx, 3*time.Second)
		defer done()
		return conn.Write(writeCtx, websocket.MessageText, raw)
	}
	if send(initial) != nil {
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case raw, ok := <-ch:
			if !ok {
				conn.Close(websocket.StatusTryAgainLater, "slow consumer; reconnect for snapshot")
				return
			}
			if send(raw) != nil {
				return
			}
		case <-ticker.C:
			pingCtx, done := context.WithTimeout(ctx, 5*time.Second)
			err = conn.Ping(pingCtx)
			done()
			if err != nil {
				return
			}
		}
	}
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	states := map[string]string{"api": "ONLINE", "postgres": "DISABLED", "redis": "DISABLED", "nats": "DISABLED", "optimizer": "ONLINE", "simulator": "OFFLINE"}
	ready := true
	if s.Service.Store.Mode() == "postgres" {
		states["postgres"] = "ONLINE"
		if err := s.Service.Store.Ping(ctx); err != nil {
			states["postgres"] = "OFFLINE"
			ready = false
		}
	}
	if s.Deps != nil {
		states["redis"] = "ONLINE"
		if err := s.Deps.Redis.Ping(ctx).Err(); err != nil {
			states["redis"] = "OFFLINE"
			ready = false
		}
		states["nats"] = "ONLINE"
		if !s.Deps.NATS.IsConnected() || s.Deps.NATS.FlushTimeout(500*time.Millisecond) != nil {
			states["nats"] = "OFFLINE"
			ready = false
		}
	}
	if s.Service.SimulatorOnline() {
		states["simulator"] = "ONLINE"
	}
	if !s.Service.Snapshot().OptimizerOnline {
		states["optimizer"] = "OFFLINE"
	}
	status := 200
	if !ready {
		status = 503
	}
	write(w, status, map[string]any{"ready": ready, "storageMode": s.Service.Store.Mode(), "services": states, "checkedAt": time.Now().UTC()})
}
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	times := []float64{}
	for _, req := range s.requests {
		times = append(times, req.DurationMs)
	}
	s.mu.Unlock()
	sort.Float64s(times)
	var p50, p95 any
	if len(times) >= 10 {
		p50 = times[len(times)/2]
		p95 = times[int(float64(len(times)-1)*0.95)]
	}
	write(w, 200, map[string]any{"traffic": s.Service.Snapshot().Metrics, "apiLatencyP50": p50, "apiLatencyP95": p95, "requestSamples": len(times), "websocketClients": s.Service.Hub.Count()})
}
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := make([]byte, 12)
		rand.Read(id)
		w.Header().Set("X-Request-ID", "req_"+hex.EncodeToString(id))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		origin := r.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range s.Origins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				problem(w, r, application.Fail("ORIGIN_DENIED", "Origin is not allowed", 403))
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		s.mu.Lock()
		now := time.Now()
		v, ok := s.visitors[ip]
		if !ok {
			v = visitor{tokens: 60, updated: now}
		}
		v.tokens = min(60, v.tokens+now.Sub(v.updated).Seconds()*20)
		v.updated = now
		allowed := v.tokens >= 1
		if allowed {
			v.tokens--
		}
		s.visitors[ip] = v
		if len(s.visitors) > 1000 {
			for key, v := range s.visitors {
				if now.Sub(v.updated) > time.Minute {
					delete(s.visitors, key)
				}
			}
		}
		s.mu.Unlock()
		if !allowed {
			w.Header().Set("Retry-After", "1")
			problem(w, r, application.Fail("RATE_LIMITED", "Too many requests", 429))
			return
		}
		next.ServeHTTP(w, r)
	})
}

type capture struct {
	http.ResponseWriter
	status   int
	response bytes.Buffer
}

func (c *capture) WriteHeader(status int) { c.status = status; c.ResponseWriter.WriteHeader(status) }
func (c *capture) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.WriteHeader(200)
	}
	if c.response.Len() < 2048 {
		n := min(len(b), 2048-c.response.Len())
		c.response.Write(b[:n])
	}
	return c.ResponseWriter.Write(b)
}
func (c *capture) Unwrap() http.ResponseWriter { return c.ResponseWriter }
func (s *Server) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		var body string
		if r.Body != nil && r.Method == "POST" {
			raw, err := io.ReadAll(io.LimitReader(r.Body, 128*1024+1))
			if err != nil {
				problem(w, r, application.Fail("INVALID_BODY", "Cannot read body", 400))
				return
			}
			if len(raw) > 128*1024 {
				problem(w, r, application.Fail("BODY_TOO_LARGE", "Request body exceeds 128 KiB", 413))
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
			body = string(raw[:min(len(raw), 2048)])
		}
		c := &capture{ResponseWriter: w}
		next.ServeHTTP(c, r)
		if c.status == 0 {
			c.status = 200
		}
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		duration := time.Since(start)
		observability.HTTP.WithLabelValues(r.Method, route, strconv.Itoa(c.status)).Inc()
		observability.HTTPDuration.WithLabelValues(route).Observe(duration.Seconds())
		if strings.HasPrefix(r.URL.Path, "/api/v1/") && r.URL.Path != "/api/v1/requests" {
			record := RequestRecord{ID: w.Header().Get("X-Request-ID"), Method: r.Method, Route: r.URL.Path, Status: c.status, DurationMs: float64(duration.Microseconds()) / 1000, Timestamp: start, Headers: map[string]string{"Content-Type": r.Header.Get("Content-Type"), "Idempotency-Key": r.Header.Get("Idempotency-Key")}, Body: body, Response: c.response.String()}
			s.mu.Lock()
			s.requests = append(s.requests, record)
			if len(s.requests) > 200 {
				s.requests = append([]RequestRecord{}, s.requests[len(s.requests)-200:]...)
			}
			s.mu.Unlock()
		}
		slog.Info("http request", "method", r.Method, "route", route, "status", c.status, "durationMs", float64(duration.Microseconds())/1000, "requestId", w.Header().Get("X-Request-ID"))
	})
}
