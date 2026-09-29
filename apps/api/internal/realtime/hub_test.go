package realtime

import (
	"sync"
	"testing"
)

func TestSlowConsumerIsBounded(t *testing.T) {
	h := New()
	ch, done := h.Subscribe()
	defer done()
	for i := 0; i < 40; i++ {
		h.Broadcast(map[string]int{"sequence": i})
	}
	if h.Count() != 0 {
		t.Fatal("slow client retained")
	}
	count := 0
	for range ch {
		count++
	}
	if count != 32 {
		t.Fatalf("unexpected buffer %d", count)
	}
}
func TestConcurrentSubscribePublish(t *testing.T) {
	h := New()
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, done := h.Subscribe()
			defer done()
			for j := 0; j < 10; j++ {
				h.Broadcast(map[string]int{"n": j})
			}
		}()
	}
	wg.Wait()
}
