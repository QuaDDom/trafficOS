package observability

import "github.com/prometheus/client_golang/prometheus"

var HTTP = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "trafficos_http_requests_total", Help: "Completed HTTP requests by bounded route and status."}, []string{"method", "route", "status"})
var HTTPDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "trafficos_http_request_duration_seconds", Help: "HTTP handler duration."}, []string{"route"})
var Events = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "trafficos_events_total", Help: "Committed domain events."}, []string{"type"})
var Decisions = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "trafficos_optimizer_decision_duration_seconds", Help: "Pure optimizer execution duration.", Buckets: prometheus.ExponentialBuckets(0.0000001, 10, 7)})
var Transitions = prometheus.NewCounter(prometheus.CounterOpts{Name: "trafficos_signal_transitions_total", Help: "Accepted safe signal transitions."})
var DB = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "trafficos_database_duration_seconds", Help: "Audit transaction duration."})
var WS = prometheus.NewGauge(prometheus.GaugeOpts{Name: "trafficos_websocket_clients", Help: "Connected WebSocket clients."})
var Online = prometheus.NewGauge(prometheus.GaugeOpts{Name: "trafficos_intersections_online", Help: "Online simulated intersections."})
var OutboxFailures = prometheus.NewCounter(prometheus.CounterOpts{Name: "trafficos_outbox_failures_total", Help: "Failed outbox publish attempts."})

func init() {
	prometheus.MustRegister(HTTP, HTTPDuration, Events, Decisions, Transitions, DB, WS, Online, OutboxFailures)
}
