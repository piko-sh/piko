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

// Package notification provides a provider-agnostic framework for
// sending notifications from a Piko application.
//
// It offers a fluent builder API for composing and dispatching
// notifications across one or more configured providers:
//
//	err := notification.NewNotificationBuilderFromDefault().
//	    Title("System Alert").
//	    Message("Server CPU usage is at 95%").
//	    Priority(notification.PriorityHigh).
//	    Do(ctx)
//
// The service supports automatic retry with exponential backoff,
// dead-letter queuing, batch operations, background dispatching,
// priority levels, and rich formatting.
package notification
