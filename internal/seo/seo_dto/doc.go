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

// Package seo_dto defines data transfer objects for the SEO module.
//
// It contains XML sitemap structures, robots.txt content, and
// [ProjectView]/[ComponentView] types that decouple the SEO domain
// from the annotator with a read-only anti-corruption layer.
//
// # Predefined bot lists
//
// The package exports [AIBots] and [NonSEOBots] slices containing
// known crawler user-agents for blocking AI training bots and
// non-SEO scrapers in robots.txt generation.
package seo_dto
