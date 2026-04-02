package chat

import (
	"sync"
)

// Message represents a chat message with a hub-assigned ID.
type Message struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Time     string `json:"time"`
}

const historySize = 100

// chatHub manages chat subscribers and message history.
type chatHub struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan Message
	nextSubID   uint64
	nextMessageID   uint64

	// Ring buffer for message history.
	history    [historySize]Message
	historyLen int // number of messages stored (max historySize)
	historyPos int // next write position
}

var hub = &chatHub{
	subscribers: make(map[uint64]chan Message),
}

// Subscribe registers a new listener. Returns a receive channel and
// an unsubscribe function. The channel is buffered to absorb bursts.
func (h *chatHub) Subscribe() (<-chan Message, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextSubID
	h.nextSubID++

	messageChannel := make(chan Message, 16)
	h.subscribers[id] = messageChannel

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, ok := h.subscribers[id]; ok {
			delete(h.subscribers, id)
			close(messageChannel)
		}
	}

	return messageChannel, unsubscribe
}

// Broadcast sends a message to all subscribers and adds it to history.
func (h *chatHub) Broadcast(message Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextMessageID++
	message.ID = h.nextMessageID

	// Add to ring buffer.
	h.history[h.historyPos] = message
	h.historyPos = (h.historyPos + 1) % historySize
	if h.historyLen < historySize {
		h.historyLen++
	}

	// Fan out to all subscribers (non-blocking).
	for _, subscriberChannel := range h.subscribers {
		select {
		case subscriberChannel <- message:
		default:
		}
	}
}

// History returns the buffered messages in chronological order.
func (h *chatHub) History() []Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]Message, 0, h.historyLen)
	if h.historyLen < historySize {
		for i := range h.historyLen {
			result = append(result, h.history[i])
		}
	} else {
		for i := range historySize {
			index := (h.historyPos + i) % historySize
			result = append(result, h.history[index])
		}
	}
	return result
}
