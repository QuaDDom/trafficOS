// The simulator is a separate telemetry producer. The API remains authoritative
// for all positions, queues, signal state and controller decisions.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
	"trafficos/packages/demand"
)

type config struct {
	Rows    int     `json:"rows"`
	Columns int     `json:"columns"`
	Profile string  `json:"profile"`
	Speed   float64 `json:"speed"`
	Paused  bool    `json:"paused"`
	Epoch   string  `json:"epoch"`
}
type frame struct {
	DT       float64          `json:"dt"`
	Arrivals []demand.Arrival `json:"arrivals"`
}
type packet struct {
	Epoch    string  `json:"epoch"`
	Session  string  `json:"session"`
	Sequence int64   `json:"sequence"`
	Frames   []frame `json:"frames"`
}

func env(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Second}
	base := env("API_URL", "http://localhost:8080")
	token := env("SIMULATOR_TOKEN", "local-simulator-change-me")
	cfg := config{Profile: "NORMAL", Speed: 1, Rows: 3, Columns: 3}
	g := demand.Generator{Seed: 42}
	p := packet{Session: fmt.Sprintf("sim_%d", time.Now().UnixNano())}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	delay := 200 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if len(p.Frames) == 0 {
			p.Sequence++
			p.Epoch = cfg.Epoch
			dt := 0.2 * cfg.Speed
			for dt > 0.0001 {
				step := min(dt, 0.2)
				arrivals := []demand.Arrival{}
				if !cfg.Paused {
					arrivals = g.ArrivalsFor(cfg.Profile, step, cfg.Rows, cfg.Columns)
				}
				p.Frames = append(p.Frames, frame{DT: step, Arrivals: arrivals})
				dt -= step
			}
		}
		raw, _ := json.Marshal(p)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/telemetry", bytes.NewReader(raw))
		if err != nil {
			slog.Error("invalid API URL", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := client.Do(req)
		if err == nil {
			if res.StatusCode == 200 {
				var next config
				err = json.NewDecoder(res.Body).Decode(&next)
				if err == nil {
					if cfg.Epoch != "" && next.Epoch != cfg.Epoch {
						g.Seed = 42
					}
					cfg = next
					p.Frames = nil
					delay = 200 * time.Millisecond
				}
			} else {
				err = fmt.Errorf("telemetry rejected: HTTP %d", res.StatusCode)
			}
			res.Body.Close()
		}
		if err != nil {
			slog.Warn("telemetry retry", "sequence", p.Sequence, "error", err, "backoff", delay.String())
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			delay = min(5*time.Second, delay*2)
		}
	}
}
