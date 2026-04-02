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

package collection

import (
	"context"
	"encoding/json"
	"errors"

	coll_fb "piko.sh/piko/internal/collection/collection_schema/collection_schema_gen"
	"piko.sh/piko/internal/logger/logger_domain"
)

// errFlatBufferParseFailed is returned when a FlatBuffer payload cannot be
// decoded.
var errFlatBufferParseFailed = errors.New("failed to parse FlatBuffer payload")

// StaticCollection is a JSON-serialisable representation of a compiled
// collection. Each item's metadata is included as decoded JSON, while AST
// payloads are represented as size information (since they are nested
// FlatBuffers that cannot be meaningfully displayed as text).
type StaticCollection struct {
	// Name is the collection name, if set.
	Name string `json:"name,omitempty"`

	// Items holds all content items in the collection.
	Items []ContentItem `json:"items"`
}

// ContentItem represents a single content item in a collection.
type ContentItem struct {
	// Metadata holds the decoded metadata map (originally stored as JSON).
	Metadata any `json:"metadata,omitempty"`

	// Route is the URL path for this item.
	Route string `json:"route"`

	// ContentSize is the byte size of the content AST payload.
	ContentSize int `json:"content_size,omitempty"`

	// ExcerptSize is the byte size of the excerpt AST payload.
	ExcerptSize int `json:"excerpt_size,omitempty"`

	// HasContent indicates whether a content AST is present.
	HasContent bool `json:"has_content"`

	// HasExcerpt indicates whether an excerpt AST is present.
	HasExcerpt bool `json:"has_excerpt"`
}

// ConvertCollection parses a raw FlatBuffer collection payload into a
// JSON-serialisable struct.
//
// Takes payload ([]byte) which is the raw FlatBuffer data after stripping the
// version header (use Unpack first).
//
// Returns *StaticCollection which contains all items with decoded metadata.
// Returns error when the payload cannot be parsed.
func ConvertCollection(payload []byte) (*StaticCollection, error) {
	fb := coll_fb.GetRootAsStaticCollectionFB(payload, 0)
	if fb == nil {
		return nil, errFlatBufferParseFailed
	}

	items := convertItems(fb)

	return &StaticCollection{
		Name:  string(fb.Name()),
		Items: items,
	}, nil
}

// convertItems extracts all content items from the FlatBuffer.
//
// Takes fb (*coll_fb.StaticCollectionFB) which is the source FlatBuffer to
// extract items from.
//
// Returns []ContentItem which contains the converted items, or nil if the
// FlatBuffer has no items.
func convertItems(fb *coll_fb.StaticCollectionFB) []ContentItem {
	length := fb.ItemsLength()
	if length == 0 {
		return nil
	}
	items := make([]ContentItem, length)
	var item coll_fb.ContentItemFB
	for i := range length {
		if fb.Items(&item, i) {
			items[i] = convertContentItem(&item)
		}
	}
	return items
}

// convertContentItem converts a single FlatBuffer content item.
//
// Takes fb (*coll_fb.ContentItemFB) which is the FlatBuffer content item to
// convert.
//
// Returns ContentItem which contains the extracted route, metadata, and size
// information.
func convertContentItem(fb *coll_fb.ContentItemFB) ContentItem {
	metadataJSON := fb.MetadataJsonBytes()
	contentAST := fb.ContentAstBytes()
	excerptAST := fb.ExcerptAstBytes()

	var metadata any
	if len(metadataJSON) > 0 {
		if unmarshalError := json.Unmarshal(metadataJSON, &metadata); unmarshalError != nil {
			_, warningLogger := logger_domain.From(context.Background(), nil)
			warningLogger.Warn("failed to unmarshal FlatBuffer metadata",
				logger_domain.String("route", string(fb.Route())),
				logger_domain.Error(unmarshalError))
		}
	}

	return ContentItem{
		Route:       string(fb.Route()),
		Metadata:    metadata,
		HasContent:  len(contentAST) > 0,
		ContentSize: len(contentAST),
		HasExcerpt:  len(excerptAST) > 0,
		ExcerptSize: len(excerptAST),
	}
}
