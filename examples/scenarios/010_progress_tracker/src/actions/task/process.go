package task

import (
	"fmt"
	"time"

	"piko.sh/piko"
)

type ProcessInput struct {
	TaskName string `json:"taskName" validate:"required"`
}

type ProcessOutput struct {
	TaskName string `json:"taskName"`
	Status   string `json:"status"`
}

// ProcessAction handles long-running tasks with SSE progress streaming.
// When the client uses .withOnProgress().call(), Piko calls StreamProgress
// instead of Call. See https://piko.sh/docs/guide/server-actions
type ProcessAction struct {
	piko.ActionMetadata
}

func (a *ProcessAction) Call(input ProcessInput) (ProcessOutput, error) {
	return ProcessOutput{
		TaskName: input.TaskName,
		Status:   "completed",
	}, nil
}

// StreamProgress sends progress events via SSE. stream.Send() pushes events
// to the client; stream.SendComplete() sends the final event and closes.
func (a *ProcessAction) StreamProgress(stream *piko.SSEStream) error {
	steps := 5
	for i := 1; i <= steps; i++ {
		percent := i * (100 / steps)
		if err := stream.Send("progress", map[string]any{
			"step":    i,
			"total":   steps,
			"percent": percent,
			"message": fmt.Sprintf("Processing step %d of %d...", i, steps),
		}); err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}

	return stream.SendComplete(map[string]string{
		"status":  "done",
		"message": "All steps completed successfully.",
	})
}
