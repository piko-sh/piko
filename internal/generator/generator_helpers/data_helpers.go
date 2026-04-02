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

package generator_helpers

import (
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/templater/templater_dto"
)

// GetData retrieves the page data from CollectionData and converts it to type T.
//
// This function is called at runtime by generated code when using
// piko.GetData[T](r) in the Render function of collection page templates.
// It extracts the "page" key from the root collection data map and performs
// type-safe conversion.
//
// Architecture Note:
//   - CollectionData is populated by BuildAST via GetStaticCollectionItem()
//   - The data is lazily deserialised from embedded FlatBuffer blobs
//   - Only the requested page's data is loaded (O(log n) binary search)
//
// The CollectionData structure:
//
//	{
//	    "page": { Title: "...", Author: "...", ... },  // Extracted and returned
//	    "contentAST": *TemplateAST                     // Used by GetContentAST
//	}
//
// Takes r (*templater_dto.RequestData) which contains the CollectionData map.
//
// Returns T which is the page data converted to the requested type, or zero
// value if conversion fails.
func GetData[T any](r *templater_dto.RequestData) T {
	var zero T

	if r.CollectionData() == nil {
		return zero
	}

	rootMap, ok := r.CollectionData().(map[string]any)
	if !ok {
		return zero
	}

	pageData, exists := rootMap["page"]
	if !exists {
		return zero
	}

	if value, ok := pageData.(T); ok {
		return value
	}

	pageMap, ok := pageData.(map[string]any)
	if !ok {
		return zero
	}

	bytes, err := json.Marshal(pageMap)
	if err != nil {
		return zero
	}

	var result T
	if err := json.Unmarshal(bytes, &result); err != nil {
		return zero
	}

	return result
}
