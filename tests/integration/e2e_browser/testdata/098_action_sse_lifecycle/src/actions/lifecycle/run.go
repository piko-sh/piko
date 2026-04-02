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
package lifecycle

import (
	"time"

	"piko.sh/piko"
)

type RunInput struct{}

type RunOutput struct {
	Started bool `json:"started"`
}

type RunAction struct {
	piko.ActionMetadata
}

func (a *RunAction) Call(_ RunInput) (RunOutput, error) {
	return RunOutput{Started: true}, nil
}

func (a *RunAction) StreamProgress(stream *piko.SSEStream) error {
	if err := stream.Send("phase", map[string]string{"name": "starting"}); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		select {
		case <-stream.Done():
			return nil
		default:
		}
		if err := stream.Send("progress", map[string]int{"step": i}); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err := stream.SendData(map[string]string{"type": "data-only"}); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	if err := stream.SendHeartbeat(); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	return stream.SendComplete(map[string]string{"result": "all-phases-done"})
}
