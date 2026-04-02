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

// Package notification_domain coordinates the sending of notifications
// through a pluggable provider registry.
//
// It defines port interfaces ([Service], [NotificationProviderPort],
// [NotificationDispatcherPort]), a fluent builder for composing
// notifications, and an asynchronous dispatcher with batching,
// exponential-backoff retry, circuit breakers, and dead letter queue
// support.
//
// # Context handling
//
// All terminal operations ([NotificationBuilder.Do]) honour context
// cancellation and deadlines. If the context is already cancelled or has
// exceeded its deadline, the operation returns immediately with the
// context's error.
//
// # Thread safety
//
// The [Service] implementation and [NotificationDispatcher] are safe for
// concurrent use. Provider registration, dispatcher operations, and
// notification sending are all protected by appropriate mutexes.
package notification_domain
