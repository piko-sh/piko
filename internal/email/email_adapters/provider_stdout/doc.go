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

// Package provider_stdout implements an email provider that writes
// emails to stdout as human-readable blocks for development use.
//
// It is the default development provider, useful for verifying
// integration wiring without external email infrastructure. Each
// email is formatted with metadata, body content, and attachment
// summaries.
//
// All methods are safe for concurrent use.
package provider_stdout
