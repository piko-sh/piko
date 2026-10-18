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

package events_provider_gochannel

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
)

var (
	benchMessagePayload = make([]byte, 512)
)

func BenchmarkBoundedPublishDeliver(b *testing.B) {
	provider, err := NewGoChannelProvider(DefaultConfig())
	if err != nil {
		b.Fatalf("creating provider: %v", err)
	}
	b.Cleanup(func() { _ = provider.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	b.Cleanup(cancel)
	messages, err := provider.Subscriber().Subscribe(ctx, "bench.bounded")
	if err != nil {
		b.Fatalf("subscribing: %v", err)
	}
	delivered := make(chan struct{})
	go func() {
		for msg := range messages {
			msg.Ack()
			delivered <- struct{}{}
		}
	}()

	b.ReportAllocs()
	for b.Loop() {
		if err := provider.Publisher().Publish("bench.bounded", message.NewMessage("bench", benchMessagePayload)); err != nil {
			b.Fatalf("publishing: %v", err)
		}
		<-delivered
	}
}
