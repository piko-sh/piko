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

// Package deadletter_domain defines the port interfaces and entry
// constraints for a generic dead letter queue.
//
// A dead letter queue captures items that failed processing so they can
// be inspected, retried, or purged later. The package uses Go generics
// to remain agnostic of the concrete entry type, so any service
// (email, notification, storage, etc.) can reuse the same queue
// contract.
package deadletter_domain
