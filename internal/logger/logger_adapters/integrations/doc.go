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

// Package integrations bridges the logging subsystem to external
// services by implementing the driven port interfaces defined in
// logger_domain.
//
// Each adapter translates logger-specific types into the target
// service's data transfer objects (e.g. notifications).
//
// # Thread safety
//
// [NotificationServiceAdapter] is safe for concurrent use provided the
// underlying [notification_domain.Service] is also safe for concurrent
// use.
package integrations
