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

// Package dispatcher_domain defines the core port interfaces and domain
// types for the dispatcher hexagon.
//
// It declares the [DispatcherInspector] port, which provides read-only
// access to dispatcher state and dead letter queues across email and
// notification dispatchers. This port abstracts over concrete
// dispatcher implementations so that monitoring and admin tooling can
// query processing statistics without depending on adapter details.
package dispatcher_domain
