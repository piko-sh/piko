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

package registry_domain

import (
	"strings"
	"sync"
)

const (
	// defaultMapCapacity is the initial capacity for pooled maps.
	defaultMapCapacity = 64

	// defaultSliceCapacity is the initial capacity for pooled string slices.
	defaultSliceCapacity = 128

	// defaultBuilderCapacity is the initial capacity in bytes for pooled string
	// builders.
	defaultBuilderCapacity = 128

	// defaultQueryArgsCapacity is the starting slice capacity for query arguments.
	defaultQueryArgsCapacity = 300

	// defaultTagMapCapacity is the initial capacity for tag maps in the pool.
	defaultTagMapCapacity = 8

	// defaultPayloadMapCapacity is the initial capacity for event payload maps.
	defaultPayloadMapCapacity = 8
)

var (
	// orderingMapPool provides reusable maps for orderArtefactsByIDs.
	// Uses any type to avoid importing registry_dto.
	orderingMapPool = sync.Pool{
		New: func() any {
			return make(map[string]any, defaultMapCapacity)
		},
	}

	// dedupeMapPool provides reusable maps for deduplicateAndSort.
	dedupeMapPool = sync.Pool{
		New: func() any {
			return make(map[string]struct{}, defaultMapCapacity)
		},
	}

	// queryArgsPool reuses query argument slices to reduce allocation pressure
	// during database query building.
	queryArgsPool = sync.Pool{
		New: func() any {
			return new(make([]any, 0, defaultQueryArgsCapacity))
		},
	}

	// stringBuilderPool provides reusable string builders for key/query construction.
	stringBuilderPool = sync.Pool{
		New: func() any {
			b := &strings.Builder{}
			b.Grow(defaultBuilderCapacity)
			return b
		},
	}

	// tagMapPool provides reusable maps for MetadataTags.
	tagMapPool = sync.Pool{
		New: func() any {
			return make(map[string]string, defaultTagMapCapacity)
		},
	}

	// eventPayloadPool provides reusable maps for publishEvent.
	eventPayloadPool = sync.Pool{
		New: func() any {
			return make(map[string]any, defaultPayloadMapCapacity)
		},
	}

	// stringSlicePool reuses string slices to reduce allocation pressure during
	// registry operations.
	stringSlicePool = sync.Pool{
		New: func() any {
			return new(make([]string, 0, defaultSliceCapacity))
		},
	}
)

// getOrderingMap gets a map from the pool for ordering artefacts.
//
// Returns map[string]any which is a recycled map from the pool or a new map
// with default capacity.
func getOrderingMap() map[string]any {
	if m, ok := orderingMapPool.Get().(map[string]any); ok {
		return m
	}
	return make(map[string]any, defaultMapCapacity)
}

// putOrderingMap returns a map to the pool after clearing it.
//
// Takes m (map[string]any) which is the map to clear and return to the pool.
func putOrderingMap(m map[string]any) {
	clear(m)
	orderingMapPool.Put(m)
}

// getDedupeMap gets a map from the pool for removing duplicates.
//
// Returns map[string]struct{} which is a reused map from the pool or a new map
// with default capacity.
func getDedupeMap() map[string]struct{} {
	if m, ok := dedupeMapPool.Get().(map[string]struct{}); ok {
		return m
	}
	return make(map[string]struct{}, defaultMapCapacity)
}

// putDedupeMap returns a map to the pool after clearing it.
//
// Takes m (map[string]struct{}) which is the map to return.
func putDedupeMap(m map[string]struct{}) {
	clear(m)
	dedupeMapPool.Put(m)
}

// getQueryArgs gets a slice from the pool for query arguments.
//
// Returns *[]any which is a pooled slice ready for use.
func getQueryArgs() *[]any {
	if s, ok := queryArgsPool.Get().(*[]any); ok {
		return s
	}
	return new(make([]any, 0, defaultQueryArgsCapacity))
}

// putQueryArgs returns a slice to the pool after resetting it.
//
// Takes s (*[]any) which is the slice to reset and return to the pool.
func putQueryArgs(s *[]any) {
	*s = (*s)[:0]
	queryArgsPool.Put(s)
}

// getStringBuilder gets a string builder from the pool.
//
// Returns *strings.Builder which is ready for use with pre-set capacity.
func getStringBuilder() *strings.Builder {
	if b, ok := stringBuilderPool.Get().(*strings.Builder); ok {
		return b
	}
	b := &strings.Builder{}
	b.Grow(defaultBuilderCapacity)
	return b
}

// putStringBuilder returns a string builder to the pool after resetting it.
//
// Takes b (*strings.Builder) which is the builder to reset and return to the
// pool.
func putStringBuilder(b *strings.Builder) {
	b.Reset()
	stringBuilderPool.Put(b)
}

// getTagMap gets a map from the pool for metadata tags.
//
// Returns map[string]string which is a reused map from the pool or a new map
// with default capacity.
func getTagMap() map[string]string {
	if m, ok := tagMapPool.Get().(map[string]string); ok {
		return m
	}
	return make(map[string]string, defaultTagMapCapacity)
}

// putTagMap returns a map to the pool after clearing it.
//
// Takes m (map[string]string) which is the tag map to clear and return.
func putTagMap(m map[string]string) {
	clear(m)
	tagMapPool.Put(m)
}

// getEventPayload gets a map from the pool for event payloads.
//
// Returns map[string]any which is a reused map from the pool or a new map with
// the default capacity.
func getEventPayload() map[string]any {
	if m, ok := eventPayloadPool.Get().(map[string]any); ok {
		return m
	}
	return make(map[string]any, defaultPayloadMapCapacity)
}

// putEventPayload returns a map to the pool after clearing it.
//
// Takes m (map[string]any) which is the map to clear and return to the pool.
func putEventPayload(m map[string]any) {
	clear(m)
	eventPayloadPool.Put(m)
}

// getStringSlice gets a string slice from the pool.
//
// Returns *[]string which is a pooled or newly created slice ready for use.
func getStringSlice() *[]string {
	if s, ok := stringSlicePool.Get().(*[]string); ok {
		return s
	}
	return new(make([]string, 0, defaultSliceCapacity))
}

// putStringSlice returns a string slice to the pool after resetting it.
//
// Takes s (*[]string) which is the slice to reset and return to the pool.
func putStringSlice(s *[]string) {
	*s = (*s)[:0]
	stringSlicePool.Put(s)
}
