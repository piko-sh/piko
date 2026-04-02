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
	"sync/atomic"

	"piko.sh/piko/internal/collection/collection_dto"
)

// MockEncoder is a test double for CollectionEncoderPort that returns zero
// values from nil function fields and tracks call counts atomically.
type MockEncoder struct {
	// EncodeCollectionFunc is the function called by
	// EncodeCollection.
	EncodeCollectionFunc func(items []collection_dto.ContentItem) ([]byte, error)

	// DecodeCollectionItemFunc is the function called by
	// DecodeCollectionItem.
	DecodeCollectionItemFunc func(blob []byte, route string) (metadataJSON, contentAST, excerptAST []byte, err error)

	// EncodeCollectionCallCount tracks how many times
	// EncodeCollection was called.
	EncodeCollectionCallCount int64

	// DecodeCollectionItemCallCount tracks how many times
	// DecodeCollectionItem was called.
	DecodeCollectionItemCallCount int64
}

var _ CollectionEncoderPort = (*MockEncoder)(nil)

// EncodeCollection delegates to EncodeCollectionFunc if set.
//
// Takes items ([]collection_dto.ContentItem) which is the list of
// content items to encode.
//
// Returns (nil, nil) if EncodeCollectionFunc is nil.
func (m *MockEncoder) EncodeCollection(items []collection_dto.ContentItem) ([]byte, error) {
	atomic.AddInt64(&m.EncodeCollectionCallCount, 1)
	if m.EncodeCollectionFunc != nil {
		return m.EncodeCollectionFunc(items)
	}
	return nil, nil
}

// DecodeCollectionItem delegates to DecodeCollectionItemFunc if set.
//
// Takes blob ([]byte) which is the encoded blob to decode.
// Takes route (string) which identifies the route of the collection item.
//
// Returns (nil, nil, nil, nil) if DecodeCollectionItemFunc is nil.
func (m *MockEncoder) DecodeCollectionItem(blob []byte, route string) (metadataJSON, contentAST, excerptAST []byte, err error) {
	atomic.AddInt64(&m.DecodeCollectionItemCallCount, 1)
	if m.DecodeCollectionItemFunc != nil {
		return m.DecodeCollectionItemFunc(blob, route)
	}
	return nil, nil, nil, nil
}
