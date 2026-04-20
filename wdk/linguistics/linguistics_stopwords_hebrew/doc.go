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

// Package linguistics_stopwords_hebrew provides Hebrew stop words
// for text analysis and search indexing.
//
// The list contains base forms only; the Hebrew stemmer strips
// prefixed particles before stop-word filtering, so prefixed
// variants do not need to be enumerated here.
//
// It self-registers via an init function so that a blank import is
// sufficient to make Hebrew stop words available through the
// registry.
package linguistics_stopwords_hebrew
