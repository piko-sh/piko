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

type NotifyInput struct{}

type NotifyOutput struct {
	Ready bool `json:"ready"`
}

type NotifyAction struct {
	piko.ActionMetadata
}

func (a *NotifyAction) Call(_ NotifyInput) (NotifyOutput, error) {
	return NotifyOutput{Ready: true}, nil
}

func (a *NotifyAction) StreamProgress(stream *piko.SSEStream) error {
	for i := 0; i < 2; i++ {
		if err := stream.Send("notification", map[string]string{
			"message": "Update available",
		}); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return stream.SendComplete(map[string]string{"status": "done"})
}
