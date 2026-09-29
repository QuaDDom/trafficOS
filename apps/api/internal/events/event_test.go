package events

import (
	"encoding/json"
	"testing"
	"time"
)

var encoded []byte

func BenchmarkEventEncode(b *testing.B) {
	e := Event{ID: "evt_benchmark", Type: "signal.phase.changed", Timestamp: time.Now(), Source: "signal-controller", CorrelationID: "corr_benchmark", Payload: map[string]any{"intersectionId": "INT-05", "phase": "EW_GREEN", "version": 42}}
	b.ReportAllocs()
	for b.Loop() {
		encoded, _ = json.Marshal(e)
	}
	b.SetBytes(int64(len(encoded)))
}
