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
	"context"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func newEnvelopeTask() *orchestrator_domain.Task {
	task := orchestrator_domain.NewTask("artefact.compiler", map[string]any{
		"artefactID": "assets/images/benchmark-artefact.svg",
		"profile":    "minified",
	})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task.CreatedAt, task.UpdatedAt, task.ExecuteAt = now, now, now
	task.DeduplicationKey = "compile:assets/images/benchmark-artefact.svg:minified"
	task.BuildTag = "build-1"
	return task
}

func BenchmarkEnvelope(b *testing.B) {
	dispatcher := &watermillTaskDispatcher{}
	ref := orchestrator_domain.RefOf(newEnvelopeTask())
	topic := orchestrator_domain.TopicTaskDispatchNormal

	b.Run("codec=binary/op=encode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			envelopeSink.payload = orchestrator_domain.AppendTaskRef(make([]byte, 0, orchestrator_domain.MaxTaskRefSize), ref)
		}
	})
	binary := orchestrator_domain.AppendTaskRef(nil, ref)
	b.Run("codec=binary/op=decode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			decoded, err := orchestrator_domain.DecodeTaskRef(binary)
			if err != nil {
				b.Fatalf("decoding: %v", err)
			}
			task := &orchestrator_domain.Task{ID: decoded.ID, Attempt: decoded.Attempt}
			task.Config.Priority = decoded.Priority
			envelopeSink.task = task
		}
	})

	b.Run("codec=json/op=encode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			event := orchestrator_domain.Event{
				Type:    orchestrator_domain.EventType(topic),
				Payload: refPayload(ref),
			}
			if _, err := json.Marshal(event); err != nil {
				b.Fatalf("encoding: %v", err)
			}
		}
	})

	encoded, err := json.Marshal(orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType(topic),
		Payload: refPayload(ref),
	})
	if err != nil {
		b.Fatalf("encoding: %v", err)
	}
	b.Run("codec=json/op=decode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var event orchestrator_domain.Event
			if err := json.Unmarshal(encoded, &event); err != nil {
				b.Fatalf("decoding: %v", err)
			}
			if _, err := dispatcher.taskFromPayload(event.Payload); err != nil {
				b.Fatalf("decoding task: %v", err)
			}
		}
	})
}

func BenchmarkBusPublishDeliver(b *testing.B) {
	ref := orchestrator_domain.RefOf(newEnvelopeTask())
	topic := orchestrator_domain.TopicTaskDispatchNormal

	b.Run("codec=binary", func(b *testing.B) {
		bus := newGoChannelBus(b)
		delivered := make(chan struct{})
		err := bus.SubscribeRaw(b.Context(), topic, func(context.Context, []byte) error {
			delivered <- struct{}{}
			return nil
		})
		if err != nil {
			b.Fatalf("subscribing: %v", err)
		}
		b.ReportAllocs()
		for b.Loop() {
			payload := orchestrator_domain.AppendTaskRef(make([]byte, 0, orchestrator_domain.MaxTaskRefSize), ref)
			if err := bus.PublishRaw(b.Context(), topic, payload); err != nil {
				b.Fatalf("publishing: %v", err)
			}
			<-delivered
		}
	})

	b.Run("codec=json", func(b *testing.B) {
		bus := newGoChannelBus(b)
		delivered := make(chan struct{})
		err := bus.SubscribeWithHandler(b.Context(), topic, func(context.Context, orchestrator_domain.Event) error {
			delivered <- struct{}{}
			return nil
		})
		if err != nil {
			b.Fatalf("subscribing: %v", err)
		}
		b.ReportAllocs()
		for b.Loop() {
			event := orchestrator_domain.Event{
				Type:    orchestrator_domain.EventType(topic),
				Payload: refPayload(ref),
			}
			if err := bus.Publish(b.Context(), topic, event); err != nil {
				b.Fatalf("publishing: %v", err)
			}
			<-delivered
		}
	})
}
