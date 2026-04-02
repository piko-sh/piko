package chat

import (
	"time"

	"piko.sh/piko"
)

// SendInput defines the data submitted to send a chat message.
type SendInput struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

// SendResponse confirms the message was broadcast.
type SendResponse struct {
	ID uint64 `json:"id"`
}

// SendAction handles posting a new chat message. The message is
// broadcast to all active Listen streams via the hub.
type SendAction struct {
	piko.ActionMetadata
}

// Call broadcasts the message to all connected listeners.
func (a SendAction) Call(input SendInput) (SendResponse, error) {
	message := Message{
		Username: input.Username,
		Text:     input.Message,
		Time:     time.Now().Format("15:04:05"),
	}

	hub.Broadcast(message)

	return SendResponse{ID: message.ID}, nil
}
