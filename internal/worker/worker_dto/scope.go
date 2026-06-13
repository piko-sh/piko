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

package worker_dto

// UniqueScope selects which fields of an enqueue that we hash for identity dedupe.
type UniqueScope string

const (
	// UniqueArgs dedupes on kind, queue and the json serialisation of the args.
	UniqueArgs UniqueScope = "args"

	// UniqueQueue dedupes on kind and queue.
	UniqueQueue UniqueScope = "queue"

	// UniqueKind dedupes on kind only.
	UniqueKind UniqueScope = "kind"
)
