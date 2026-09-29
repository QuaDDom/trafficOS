package realtime

import (
	"encoding/json"
	"sync"
)

type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func New() *Hub { return &Hub{clients: map[chan []byte]struct{}{}} }
func (h *Hub) Subscribe() (chan []byte, func()) {
	ch := make(chan []byte, 32)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if _, ok := h.clients[ch]; ok {
			delete(h.clients, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}
func (h *Hub) Count() int { h.mu.Lock(); defer h.mu.Unlock(); return len(h.clients) }

// Slow clients are disconnected, never allowed to block the controller. Reconnect
// sends a full snapshot, so a dropped connection cannot corrupt client state.
func (h *Hub) Broadcast(v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.BroadcastRaw(raw)
}
func (h *Hub) BroadcastRaw(raw []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- raw:
		default:
			delete(h.clients, ch)
			close(ch)
		}
	}
}
