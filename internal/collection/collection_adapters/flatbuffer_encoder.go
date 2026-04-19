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

package collection_adapters

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"piko.sh/piko/internal/json"
	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/collection/collection_schema"
	coll_fb "piko.sh/piko/internal/collection/collection_schema/collection_schema_gen"
	"piko.sh/piko/internal/fbs"
)

// FlatBufferEncoder implements CollectionEncoderPort using FlatBuffers.
// It packs ContentItem slices into compact binary data and supports fast
// lookups at runtime through binary search.
type FlatBufferEncoder struct{}

// NewFlatBufferEncoder creates a new FlatBuffer encoder.
//
// Returns *FlatBufferEncoder which is ready for encoding collection data.
func NewFlatBufferEncoder() *FlatBufferEncoder {
	return &FlatBufferEncoder{}
}

// EncodeCollection packs a collection into a FlatBuffer binary blob.
//
// Takes items ([]collection_dto.ContentItem) which contains the content items
// to encode.
//
// Returns []byte which contains the finished FlatBuffer binary that can be
// directly embedded and read at runtime without decoding overhead.
// Returns error when items is empty or encoding of any item fails.
//
// Workflow:
//  1. Sort items by URL (route) to enable binary search
//  2. For each item:
//     - Encode metadata map to JSON bytes
//     - Encode ContentAST via ast_adapters.EncodeAST()
//     - Encode ExcerptAST (if present)
//  3. Build FlatBuffer using the collection_schema
//  4. Return finished bytes
func (*FlatBufferEncoder) EncodeCollection(items []collection_dto.ContentItem) ([]byte, error) {
	if len(items) == 0 {
		return nil, errors.New("cannot encode empty collection")
	}

	sortedItems := make([]collection_dto.ContentItem, len(items))
	copy(sortedItems, items)
	slices.SortFunc(sortedItems, func(a, b collection_dto.ContentItem) int {
		return cmp.Compare(a.URL, b.URL)
	})

	builder := flatbuffers.NewBuilder(1024 * 64)

	itemOffsets := make([]flatbuffers.UOffsetT, len(sortedItems))
	for i := range sortedItems {
		offset, err := encodeContentItem(builder, &sortedItems[i])
		if err != nil {
			return nil, fmt.Errorf("failed to encode item %s: %w", sortedItems[i].URL, err)
		}
		itemOffsets[i] = offset
	}

	coll_fb.StaticCollectionFBStartItemsVector(builder, len(itemOffsets))
	for i := len(itemOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(itemOffsets[i])
	}
	itemsVectorOffset := builder.EndVector(len(itemOffsets))

	coll_fb.StaticCollectionFBStart(builder)
	coll_fb.StaticCollectionFBAddItems(builder, itemsVectorOffset)
	rootOffset := coll_fb.StaticCollectionFBEnd(builder)

	builder.Finish(rootOffset)
	payload := builder.FinishedBytes()

	result := make([]byte, fbs.PackedSize(len(payload)))
	collection_schema.PackInto(result, payload)

	return result, nil
}

// DecodeCollectionItem extracts a single item from the versioned blob using
// binary search.
//
// Performs an O(log n) lookup in the sorted items vector without decoding the
// entire collection. Returns raw bytes that can be lazily decoded by the caller.
//
// Takes blob ([]byte) which is the versioned encoded collection produced
// by EncodeCollection.
// Takes route (string) which is the URL/route to look up (e.g.
// "/docs/actions").
//
// Returns metadataJSON ([]byte) which is the raw JSON bytes of the
// metadata map.
// Returns contentAST ([]byte) which is the raw versioned FlatBuffer
// bytes of the ContentAST.
// Returns excerptAST ([]byte) which is the raw versioned FlatBuffer
// bytes of the ExcerptAST, or nil if not present.
// Returns err (error) when the schema version does not match or the
// route is not found.
func (*FlatBufferEncoder) DecodeCollectionItem(
	blob []byte,
	route string,
) (metadataJSON, contentAST, excerptAST []byte, err error) {
	if len(blob) == 0 {
		return nil, nil, nil, errors.New("cannot decode from empty blob")
	}

	payload, err := collection_schema.Unpack(blob)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("collection schema version mismatch: %w", err)
	}

	coll := coll_fb.GetRootAsStaticCollectionFB(payload, 0)

	item := &coll_fb.ContentItemFB{}
	found := coll.ItemsByKey(item, route)
	if !found {
		return nil, nil, nil, fmt.Errorf("route not found: %s", route)
	}

	metadataJSON = item.MetadataJsonBytes()
	contentAST = item.ContentAstBytes()
	excerptAST = item.ExcerptAstBytes()

	return metadataJSON, contentAST, excerptAST, nil
}

// encodeContentItem encodes a single ContentItem into FlatBuffer format.
//
// This is a helper function that handles:
//   - JSON encoding of metadata
//   - AST encoding for content and excerpt
//   - Building the ContentItemFB table
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer builder to use.
// Takes item (*collection_dto.ContentItem) which is the content item to
// encode.
//
// Returns flatbuffers.UOffsetT which is the offset of the encoded item.
// Returns error when metadata marshalling or AST encoding fails.
func encodeContentItem(
	builder *flatbuffers.Builder,
	item *collection_dto.ContentItem,
) (flatbuffers.UOffsetT, error) {
	routeOffset := builder.CreateString(item.URL)

	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	metadataOffset := builder.CreateByteVector(metadataJSON)

	var contentASTOffset flatbuffers.UOffsetT
	if item.ContentAST != nil {
		contentASTBytes, err := ast_adapters.EncodeAST(item.ContentAST)
		if err != nil {
			return 0, fmt.Errorf("failed to encode content AST: %w", err)
		}
		contentASTOffset = builder.CreateByteVector(contentASTBytes)
	}

	var excerptASTOffset flatbuffers.UOffsetT
	if item.ExcerptAST != nil {
		excerptASTBytes, err := ast_adapters.EncodeAST(item.ExcerptAST)
		if err != nil {
			return 0, fmt.Errorf("failed to encode excerpt AST: %w", err)
		}
		excerptASTOffset = builder.CreateByteVector(excerptASTBytes)
	}

	coll_fb.ContentItemFBStart(builder)
	coll_fb.ContentItemFBAddRoute(builder, routeOffset)
	coll_fb.ContentItemFBAddMetadataJson(builder, metadataOffset)
	if item.ContentAST != nil {
		coll_fb.ContentItemFBAddContentAst(builder, contentASTOffset)
	}
	if item.ExcerptAST != nil {
		coll_fb.ContentItemFBAddExcerptAst(builder, excerptASTOffset)
	}

	return coll_fb.ContentItemFBEnd(builder), nil
}
