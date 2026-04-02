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

// Package seo_adapters implements the driven ports for the SEO hexagon.
//
// It implements the SEO domain ports: fetching dynamic URLs from HTTP
// endpoints, storing SEO artefacts (sitemap.xml, robots.txt) in the
// registry, and translating annotator domain objects into SEO-specific
// views via [ProjectViewTranslator].
//
// # Thread safety
//
// [HTTPSourceAdapter] is safe for concurrent use; the underlying HTTP
// client and circuit breaker handle synchronisation internally.
// [RegistryStorageAdapter] thread safety depends on the underlying
// registry service implementation.
package seo_adapters
