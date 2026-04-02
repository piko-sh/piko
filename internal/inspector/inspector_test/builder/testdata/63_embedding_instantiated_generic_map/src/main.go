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

package main

import "testcase_63_embedding_instantiated_generic_map/keys"

// Cache is a named type definition for a generic map.
type Cache[K comparable, V any] map[K]V

type Response struct {
	// Embed an instantiated version of our generic map type.
	// This tests if the serialiser can "see through" the embedding and the
	// type definition to find the PackagePath from the 'keys' package.
	Cache[keys.SessionKey, int]
}
