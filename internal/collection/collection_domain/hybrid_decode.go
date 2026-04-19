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

package collection_domain

import (
	"fmt"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/collection/collection_schema"
	coll_fb "piko.sh/piko/internal/collection/collection_schema/collection_schema_gen"
)

// DecodeCollectionBlob decodes a FlatBuffer collection blob into a typed
// slice. Each item's metadata JSON is unmarshalled into T.
//
// Called by generated hybrid collection getter functions to convert the cached
// FlatBuffer blob back into the user's typed slice.
//
// Takes blob ([]byte) which is the FlatBuffer-encoded collection data.
//
// Returns []T which contains the decoded items.
// Returns error when the blob cannot be unpacked or decoded.
func DecodeCollectionBlob[T any](blob []byte) ([]T, error) {
	if len(blob) == 0 {
		return nil, nil
	}

	payload, err := collection_schema.Unpack(blob)
	if err != nil {
		return nil, fmt.Errorf("decoding collection blob: %w", err)
	}

	coll := coll_fb.GetRootAsStaticCollectionFB(payload, 0)
	itemCount := coll.ItemsLength()

	result := make([]T, 0, itemCount)

	for i := range itemCount {
		item := &coll_fb.ContentItemFB{}
		if !coll.Items(item, i) {
			continue
		}

		metadataJSON := item.MetadataJsonBytes()
		if len(metadataJSON) == 0 {
			continue
		}

		var v T
		if err := cache_domain.CacheAPI.Unmarshal(metadataJSON, &v); err != nil {
			continue
		}

		result = append(result, v)
	}

	return result, nil
}
