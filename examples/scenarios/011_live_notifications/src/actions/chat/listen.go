package chat

import (
	"strconv"
	"time"

	"piko.sh/piko"
)

// ListenInput is empty - the listen stream takes no parameters.
type ListenInput struct{}

// ListenOutput is the non-streaming fallback response.
type ListenOutput struct {
	Active bool `json:"active"`
}

// ListenAction streams chat messages to the client via SSE.
// It subscribes to the hub and forwards messages in real time.
type ListenAction struct {
	piko.ActionMetadata
}

// Call is the non-streaming fallback.
func (a *ListenAction) Call(_ ListenInput) (ListenOutput, error) {
	return ListenOutput{Active: true}, nil
}

// StreamProgress subscribes to the chat hub and streams messages.
// On connect, it replays recent message history. The stream stays
// open until the client disconnects.
func (a *ListenAction) StreamProgress(stream *piko.SSEStream) error {
	// Subscribe BEFORE reading history to avoid missing messages
	// between the history read and subscription.
	msgCh, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	// Determine which messages to skip on reconnection.
	// The Last-Event-ID is the hub message ID of the last received message.
	var lastSeenID uint64
	if lastID := stream.LastEventID(); lastID != "" {
		if parsed, err := strconv.ParseUint(lastID, 10, 64); err == nil {
			lastSeenID = parsed
		}
	}

	// Replay recent history (skip messages the client already has).
	for _, message := range hub.History() {
		if message.ID <= lastSeenID {
			continue
		}
		if err := stream.SendWithID(strconv.FormatUint(message.ID, 10), "chat", message); err != nil {
			return err
		}
	}

	// Signal the client that the stream is ready (no event ID).
	if err := stream.Send("connected", map[string]string{"status": "ok"}); err != nil {
		return err
	}

	// Enter live mode: forward messages until the client disconnects.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-stream.Done():
			return nil
		case message, ok := <-msgCh:
			if !ok {
				return nil
			}
			if err := stream.SendWithID(strconv.FormatUint(message.ID, 10), "chat", message); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := stream.SendHeartbeat(); err != nil {
				return err
			}
		}
	}
}
