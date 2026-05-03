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
	"unicode/utf8"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/collection/collection_schema"
	coll_fb "piko.sh/piko/internal/collection/collection_schema/collection_schema_gen"
	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/json"
)

// MaxCollectionItems caps the number of items a single collection may contain.
// Acts as a DoS guardrail so a runaway content directory cannot exhaust memory
// while encoding the FlatBuffer blob.
const MaxCollectionItems = 100_000

// MaxSlugBytes caps the byte length of a single item's slug. Slugs become
// FlatBuffer keys, log fields and URL captures, so the cap acts as
// defence-in-depth against pathological filenames or hostile providers.
const MaxSlugBytes = 1024

// asciiControlMaxExclusive is the upper bound (exclusive) of the ASCII C0
// control range; runes below this are non-printable.
const asciiControlMaxExclusive = 0x20

// asciiDelete is the ASCII DEL character, the only control point above the
// printable range that must also be rejected from slugs.
const asciiDelete = 0x7F

// deterministicJSON encodes Metadata with sorted map keys so that the
// generated FlatBuffer blob is byte-stable across runs.
var deterministicJSON = json.Freeze(json.Config{SortMapKeys: true})

// ErrEmptyCollection is returned when EncodeCollection receives no items.
var ErrEmptyCollection = errors.New("cannot encode empty collection")

// ErrEmptySlug is returned when an item lacks the slug used as its lookup key.
var ErrEmptySlug = errors.New("collection item has empty slug")

// ErrDuplicateSlug is returned when two items share the same slug.
var ErrDuplicateSlug = errors.New("collection contains duplicate slug")

// ErrTooManyItems is returned when an encode request exceeds MaxCollectionItems.
var ErrTooManyItems = errors.New("collection exceeds maximum item count")

// ErrInvalidSlug is returned when a slug fails structural validation (length,
// UTF-8, control characters, traversal segments).
var ErrInvalidSlug = errors.New("collection item slug is invalid")

// ErrEmptyBlob is returned when DecodeCollectionItem is called with no payload.
var ErrEmptyBlob = errors.New("cannot decode from empty blob")

// ErrSlugNotInBlob is returned when DecodeCollectionItem cannot find the slug.
var ErrSlugNotInBlob = errors.New("slug not found in collection")

// ErrSchemaVersionMismatch is returned when the blob's schema version differs
// from the build's expected version.
var ErrSchemaVersionMismatch = errors.New("collection schema version mismatch")

// FlatBufferEncoder packs ContentItem slices into compact binary blobs and
// supports O(log n) lookup at runtime via FlatBuffer binary search keyed by
// item slug.
type FlatBufferEncoder struct{}

// NewFlatBufferEncoder constructs a FlatBufferEncoder ready for use.
//
// Returns *FlatBufferEncoder which can encode and decode collection blobs.
func NewFlatBufferEncoder() *FlatBufferEncoder {
	return &FlatBufferEncoder{}
}

// EncodeCollection packs items into a versioned FlatBuffer blob keyed by slug.
//
// Items are sorted by Slug so the FlatBuffer ItemsByKey binary search can
// locate them at lookup time. Validation runs before encoding: empty input,
// items with empty Slug, duplicate slugs, and counts above MaxCollectionItems
// are all rejected up front.
//
// Takes items ([]collection_dto.ContentItem) which is the slice to encode.
//
// Returns []byte which is the packed FlatBuffer blob.
// Returns error which is a sentinel (ErrEmptyCollection, ErrEmptySlug,
// ErrDuplicateSlug, or ErrTooManyItems) wrapped with context describing the
// offending item, or nil on success.
func (*FlatBufferEncoder) EncodeCollection(items []collection_dto.ContentItem) ([]byte, error) {
	if len(items) == 0 {
		return nil, ErrEmptyCollection
	}
	if len(items) > MaxCollectionItems {
		return nil, fmt.Errorf("%w: got %d items, max %d", ErrTooManyItems, len(items), MaxCollectionItems)
	}

	sortedItems := make([]collection_dto.ContentItem, len(items))
	copy(sortedItems, items)
	slices.SortFunc(sortedItems, func(a, b collection_dto.ContentItem) int {
		return cmp.Compare(a.Slug, b.Slug)
	})

	if err := validateSlugs(sortedItems); err != nil {
		return nil, err
	}

	builder := flatbuffers.NewBuilder(1024 * 64)

	itemOffsets := make([]flatbuffers.UOffsetT, len(sortedItems))
	for i := range sortedItems {
		offset, err := encodeContentItem(builder, &sortedItems[i])
		if err != nil {
			return nil, fmt.Errorf("encode item %q: %w", sortedItems[i].Slug, err)
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

// DecodeCollectionItem extracts a single item from a versioned blob by slug.
//
// The blob's FlatBuffer "route" field is named for legacy reasons but stores
// the item's slug, so the lookup key passed in must be a slug (e.g.
// "anthropic"), not a URL path. Performs an O(log n) binary search without
// decoding the entire collection and returns raw bytes for lazy decoding by the
// caller.
//
// Takes blob ([]byte) which is the versioned encoded collection produced by
// EncodeCollection.
// Takes slug (string) which is the item identifier to look up.
//
// Returns metadataJSON ([]byte) which is the raw JSON metadata for the item.
// Returns contentAST ([]byte) which is the encoded content AST.
// Returns excerptAST ([]byte) which is the encoded excerpt AST, or nil when
// the item has no excerpt.
// Returns err (error) which is ErrEmptyBlob if the blob is empty,
// ErrSchemaVersionMismatch if the schema version differs, or ErrSlugNotInBlob
// when the slug is absent.
func (*FlatBufferEncoder) DecodeCollectionItem(
	blob []byte,
	slug string,
) (metadataJSON, contentAST, excerptAST []byte, err error) {
	if len(blob) == 0 {
		return nil, nil, nil, ErrEmptyBlob
	}

	payload, err := collection_schema.Unpack(blob)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrSchemaVersionMismatch, err)
	}

	coll := coll_fb.GetRootAsStaticCollectionFB(payload, 0)

	item := &coll_fb.ContentItemFB{}
	found := coll.ItemsByKey(item, slug)
	if !found {
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrSlugNotInBlob, slug)
	}

	metadataJSON = item.MetadataJsonBytes()
	contentAST = item.ContentAstBytes()
	excerptAST = item.ExcerptAstBytes()

	return metadataJSON, contentAST, excerptAST, nil
}

// validateSlugs scans sorted items for empty, malformed or duplicate slug
// values.
//
// Items must already be sorted by Slug so duplicate detection is a single
// linear scan. validateSlug enforces structural rules (length, UTF-8, control
// characters, traversal segments) so a hostile provider cannot smuggle a slug
// containing log-corrupting bytes or path-traversal tokens into the runtime.
//
// Takes sorted ([]collection_dto.ContentItem) which is the slug-sorted item
// slice to validate.
//
// Returns error which is ErrEmptySlug, ErrInvalidSlug or ErrDuplicateSlug
// wrapped with the offending value, or nil when all slugs are valid and
// unique.
func validateSlugs(sorted []collection_dto.ContentItem) error {
	for i := range sorted {
		if sorted[i].Slug == "" {
			return fmt.Errorf("%w: item index %d has ID %q", ErrEmptySlug, i, sorted[i].ID)
		}
		if err := validateSlug(sorted[i].Slug); err != nil {
			return fmt.Errorf("%w: item index %d (%q): %w", ErrInvalidSlug, i, sorted[i].Slug, err)
		}
		if i > 0 && sorted[i].Slug == sorted[i-1].Slug {
			return fmt.Errorf("%w: %q appears more than once", ErrDuplicateSlug, sorted[i].Slug)
		}
	}
	return nil
}

// validateSlug enforces the structural contract for a single slug. Slugs must
// be valid UTF-8, fit MaxSlugBytes, contain no ASCII control characters or
// path-traversal segments, and not begin or end with a separator.
//
// Takes slug (string) which is the slug to validate.
//
// Returns error describing the rule violated, or nil when the slug is well
// formed.
func validateSlug(slug string) error {
	if len(slug) > MaxSlugBytes {
		return fmt.Errorf("length %d exceeds cap %d", len(slug), MaxSlugBytes)
	}
	if !utf8.ValidString(slug) {
		return errors.New("contains invalid UTF-8")
	}
	if slug[0] == '/' || slug[len(slug)-1] == '/' {
		return errors.New("must not begin or end with /")
	}
	for index := range slug {
		r := rune(slug[index])
		if r < asciiControlMaxExclusive || r == asciiDelete {
			return fmt.Errorf("contains control character 0x%02X at offset %d", r, index)
		}
	}
	for _, segment := range slugSegments(slug) {
		if segment == "" {
			return errors.New("contains empty path segment")
		}
		if segment == "." || segment == ".." {
			return fmt.Errorf("contains traversal segment %q", segment)
		}
	}
	return nil
}

// slugSegments splits a slug on the path separator. A single-segment slug
// without separators returns a one-element slice.
//
// Takes slug (string) which is the slug to split.
//
// Returns []string which are the path segments.
func slugSegments(slug string) []string {
	if slug == "" {
		return nil
	}
	count := 1
	for index := 0; index < len(slug); index++ {
		if slug[index] == '/' {
			count++
		}
	}
	segments := make([]string, 0, count)
	start := 0
	for index := 0; index < len(slug); index++ {
		if slug[index] == '/' {
			segments = append(segments, slug[start:index])
			start = index + 1
		}
	}
	segments = append(segments, slug[start:])
	return segments
}

// encodeContentItem encodes a single ContentItem into FlatBuffer format.
//
// The FlatBuffer "route" key is set from item.Slug; the field is named "route"
// for legacy schema reasons.
//
// Takes builder (*flatbuffers.Builder) which buffers the encoded bytes.
// Takes item (*collection_dto.ContentItem) which is the source content item.
//
// Returns flatbuffers.UOffsetT which is the offset of the encoded item table.
// Returns error which wraps any failure during metadata or AST encoding.
func encodeContentItem(
	builder *flatbuffers.Builder,
	item *collection_dto.ContentItem,
) (flatbuffers.UOffsetT, error) {
	routeOffset := builder.CreateString(item.Slug)

	metadataJSON, err := deterministicJSON.Marshal(item.Metadata)
	if err != nil {
		return 0, fmt.Errorf("marshal metadata: %w", err)
	}
	metadataOffset := builder.CreateByteVector(metadataJSON)

	var contentASTOffset flatbuffers.UOffsetT
	if item.ContentAST != nil {
		contentASTBytes, err := ast_adapters.EncodeAST(item.ContentAST)
		if err != nil {
			return 0, fmt.Errorf("encode content AST: %w", err)
		}
		contentASTOffset = builder.CreateByteVector(contentASTBytes)
	}

	var excerptASTOffset flatbuffers.UOffsetT
	if item.ExcerptAST != nil {
		excerptASTBytes, err := ast_adapters.EncodeAST(item.ExcerptAST)
		if err != nil {
			return 0, fmt.Errorf("encode excerpt AST: %w", err)
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
