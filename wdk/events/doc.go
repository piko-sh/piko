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

// Package events provides access to Piko's message bus infrastructure
// powered by Watermill.
//
// Unlike other Piko facades, this package intentionally does NOT wrap
// Watermill types. Users should import Watermill types directly for
// Message, Publisher, Subscriber, and Router:
//
//	import "github.com/ThreeDotsLabs/watermill/message"
//
// # Quick start
//
// After Piko is initialised, you can access the shared Watermill
// infrastructure:
//
//	import (
//	    "github.com/ThreeDotsLabs/watermill/message"
//	    "piko.sh/piko/wdk/events"
//	)
//
//	// Get the shared router, publisher, and subscriber
//	router := events.GetRouter()
//	pub := events.GetPublisher()
//	sub := events.GetSubscriber()
//
//	// Add your own handler to the router
//	router.AddNoPublisherHandler("my-handler", "my-topic", sub,
//	    func(msg *message.Message) error {
//	        // Process the message
//	        return nil // nil = Ack, error = Nack
//	    },
//	)
//
// # Shared infrastructure
//
// Piko and your application share the same Watermill Router, Publisher,
// and Subscriber. This gives you a single connection pool, consistent
// configuration, and unified shutdown handling.
//
// # Publishing messages
//
//	import "github.com/ThreeDotsLabs/watermill"
//
//	wmMessage := message.NewMessage(
//	    watermill.NewUUID(),
//	    []byte(`{"order_id": "123"}`),
//	)
//	err := events.GetPublisher().Publish("orders.created", wmMessage)
//
// # Back-pressure
//
// The default GoChannel provider has BlockPublishUntilSubscriberAck
// enabled. This means Publish() will block until all subscribers have
// acknowledged the message. This prevents message loss but may reduce
// throughput for high-volume scenarios.
//
// # Available providers
//
// Provider adapters are available in the events_provider_* sub-packages.
// The default is an in-memory GoChannel with back-pressure enabled.
package events
