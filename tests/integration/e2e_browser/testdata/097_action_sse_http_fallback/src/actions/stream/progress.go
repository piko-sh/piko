// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.
package stream

import (
	"time"

	"piko.sh/piko"
)

type ProgressInput struct {
	TaskID string `json:"taskId" validate:"required"`
}

type ProgressOutput struct {
	TaskID string `json:"taskId"`
	Status string `json:"status"`
}

type ProgressAction struct {
	piko.ActionMetadata
}

func (a *ProgressAction) Call(input ProgressInput) (ProgressOutput, error) {
	return ProgressOutput{
		TaskID: input.TaskID,
		Status: "completed",
	}, nil
}

func (a *ProgressAction) StreamProgress(stream *piko.SSEStream) error {
	for i := 1; i <= 3; i++ {
		if err := stream.Send("progress", map[string]any{
			"step":    i,
			"percent": i * 33,
		}); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return stream.SendComplete(map[string]string{"status": "done"})
}
