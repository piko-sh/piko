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

// Package provider_disk implements the storage provider port using
// the local filesystem with atomic writes and safedisk sandboxing.
//
// Writes use a temporary file plus rename to prevent data corruption
// from partial writes, and all filesystem operations are sandboxed
// with safedisk to prevent path traversal attacks. The package
// supports both Unix and Windows platforms via platform-specific disk
// space monitoring.
//
// All methods are safe for concurrent use.
package provider_disk
