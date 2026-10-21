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

package orchestrator_domain

import (
	"context"
)

// RawHandler handles the payload of one raw message.
//
// The payload is only valid until the handler returns. Returning an error leaves the
// message unacknowledged, so a transport that redelivers sends it again.
type RawHandler func(ctx context.Context, payload []byte) error

// RawEventBus carries opaque message payloads without the Event JSON envelope.
type RawEventBus interface {
	// PublishRaw sends payload to topic. The bus owns payload once called; the caller must
	// not modify it afterwards.
	//
	// Takes topic (string) which is the destination.
	// Takes payload ([]byte) which is the message body.
	//
	// Returns error when the bus is closed, the payload exceeds the bus's limit, or the
	// transport refuses the message.
	PublishRaw(ctx context.Context, topic string, payload []byte) error

	// SubscribeRaw delivers each message on topic to handler. When several processes
	// subscribe to the same topic, each message goes to one of them.
	//
	// Takes topic (string) which is the topic.
	// Takes handler (RawHandler) which receives each payload.
	//
	// Returns error when the subscription cannot be created.
	SubscribeRaw(ctx context.Context, topic string, handler RawHandler) error
}
