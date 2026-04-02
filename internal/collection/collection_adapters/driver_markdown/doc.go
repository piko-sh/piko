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

// Package driver_markdown implements a static collection provider for
// markdown files.
//
// This adapter discovers, parses, and processes markdown content from the
// filesystem. It supports locale-aware path analysis, frontmatter
// extraction, and translation linking between localised versions of the
// same content. Content may be sourced from local directories or external
// Go modules via the resolver system.
//
// # Directory structure
//
// The provider expects content organised as:
//
//	content/
//	  blog/           -> collection "blog"
//	    en/post.md    -> locale "en", slug "post"
//	    fr/post.md    -> locale "fr", linked via translation key
//	  docs/           -> collection "docs"
//	    intro.md      -> default locale, slug "intro"
//
// # Thread safety
//
// Provider instances are safe for concurrent use. Each call to
// FetchStaticContent operates independently with no shared mutable
// state.
package driver_markdown
