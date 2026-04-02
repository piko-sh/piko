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

// Package storage_provider_r2 provides a Cloudflare R2 storage
// adapter.
//
// It wraps the S3 provider with R2-specific defaults, automatically
// constructing the endpoint URL from the configured Cloudflare account
// ID. Because it delegates to the S3 provider, it inherits all of its
// capabilities (multipart uploads, byte-range reads, server-side copy,
// batch operations, rate limiting).
//
// All methods are safe for concurrent use.
package storage_provider_r2
