// Package realtime fans out events to connected WebSocket clients.
package realtime

import "sync"

// Event is sent as JSON to the user's connected clients.
type Event struct {
	Type string `json:"type"` // notification | message | job
	Data any    `json:"data"`
}

type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan Event]struct{})}
}

// Subscribe registers a listener for userID; call the returned func to stop.
func (h *Hub) Subscribe(userID string) (<-chan Event, func()) {
	ch := make(chan Event, 32)
	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[chan Event]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[userID], ch)
		if len(h.subs[userID]) == 0 {
			delete(h.subs, userID)
		}
		h.mu.Unlock()
	}
}

// Publish delivers e to every connection of userID. Slow consumers drop
// events rather than block the sender; clients refetch on reconnect.
func (h *Hub) Publish(userID string, e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[userID] {
		select {
		case ch <- e:
		default:
		}
	}
}
