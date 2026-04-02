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

type LiveInput struct{}

type LiveOutput struct {
	Active bool `json:"active"`
}

type LiveAction struct {
	piko.ActionMetadata
}

func (a *LiveAction) Call(_ LiveInput) (LiveOutput, error) {
	return LiveOutput{Active: true}, nil
}

func (a *LiveAction) StreamProgress(stream *piko.SSEStream) error {
	for i := 0; i < 3; i++ {
		if err := stream.Send("tick", map[string]int{"count": i}); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return stream.SendComplete(map[string]string{"status": "finished"})
}
