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

package orchestrator_adapters

import (
	"testing"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/testutil/allocbudget"
)

var (
	envelopeSink struct {
		payload []byte
		task    *orchestrator_domain.Task
	}
)

func TestAllocBudget_Envelope(t *testing.T) {
	dispatcher := &watermillTaskDispatcher{}
	ref := orchestrator_domain.RefOf(newEnvelopeTask())

	t.Run("codec=binary/op=encode", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 1}, func() {
			envelopeSink.payload = orchestrator_domain.AppendTaskRef(make([]byte, 0, orchestrator_domain.MaxTaskRefSize), ref)
		})
	})
	binary := orchestrator_domain.AppendTaskRef(nil, ref)
	t.Run("codec=binary/op=decode", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 2}, func() {
			decoded, err := orchestrator_domain.DecodeTaskRef(binary)
			if err != nil {
				t.Fatalf("decoding: %v", err)
			}
			task := &orchestrator_domain.Task{ID: decoded.ID, Attempt: decoded.Attempt}
			task.Config.Priority = decoded.Priority
			envelopeSink.task = task
		})
	})

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType(orchestrator_domain.TopicTaskDispatchNormal),
		Payload: refPayload(ref),
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	t.Run("codec=json/op=encode", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 18}, func() {
			if _, err := json.Marshal(orchestrator_domain.Event{
				Type:    event.Type,
				Payload: refPayload(ref),
			}); err != nil {
				t.Fatalf("encoding: %v", err)
			}
		})
	})
	t.Run("codec=json/op=decode", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 14}, func() {
			var decoded orchestrator_domain.Event
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("decoding: %v", err)
			}
			if _, err := dispatcher.taskFromPayload(decoded.Payload); err != nil {
				t.Fatalf("decoding task: %v", err)
			}
		})
	})
}
