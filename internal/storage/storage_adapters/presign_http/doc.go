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

// Package presign_http implements HTTP handlers for presigned storage
// operations and public file serving.
//
// It handles file uploads via HMAC-signed presigned URLs, file
// downloads via presigned download tokens, and unauthenticated access
// to public repositories. The handlers validate tokens, enforce rate
// limits and size constraints, manage HTTP caching (ETag,
// If-None-Match, If-Modified-Since), and stream content to and from
// the underlying storage provider.
//
// All handler types are safe for concurrent use once constructed.
package presign_http
