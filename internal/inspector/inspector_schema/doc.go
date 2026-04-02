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

// Package inspector_schema manages versioned serialisation for the
// inspector's type data cache using FlatBuffers. It embeds
// type_data.fbs and computes a SHA-256 hash at init time; this hash
// prefixes every serialised payload so that the cache is
// automatically invalidated whenever the schema evolves.
//
// # Integration
//
// Consumers call [Pack] or [PackInto] before writing cache entries
// and [Unpack] when reading them back. A returned
// [fbs.ErrSchemaVersionMismatch] signals that the entry was
// produced by an older schema and must be regenerated.
package inspector_schema
