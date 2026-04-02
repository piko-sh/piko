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

//go:build safe || (js && wasm)

package binder

import (
	"reflect"
	"sync"
)

// binderCache is a thread-safe cache for struct metadata. It uses sync.Map
// with reflect.Type keys for storage.
type binderCache struct {
	// cache stores struct metadata, keyed by reflect.Type.
	cache sync.Map

	// mu guards the cache during concurrent rebuilds.
	mu sync.Mutex
}

// get retrieves or builds struct metadata for a type.
//
// Takes t (reflect.Type) which specifies the type to get metadata for.
// Takes maxDepth (int) which limits how deep to go when building metadata.
//
// Returns *structInfo which contains the cached or newly built metadata.
//
// Safe for concurrent use. Uses double-checked locking to avoid doing the
// same work twice when multiple goroutines request the same type at once.
func (c *binderCache) get(t reflect.Type, maxDepth int) *structInfo {
	if cached, ok := c.cache.Load(t); ok {
		if info, isStructInfo := cached.(*structInfo); isStructInfo {
			return info
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.cache.Load(t); ok {
		if info, isStructInfo := cached.(*structInfo); isStructInfo {
			return info
		}
	}

	info := c.build(t, maxDepth)
	c.cache.Store(t, info)
	return info
}

// build performs one-time reflection to analyse a struct type and create its
// metadata.
//
// Takes t (reflect.Type) which specifies the struct type to analyse.
// Takes maxDepth (int) which limits how deep to walk nested structs.
//
// Returns *structInfo which contains the analysed field metadata.
func (c *binderCache) build(t reflect.Type, maxDepth int) *structInfo {
	info := &structInfo{
		Fields: make(map[string]*fieldInfo),
	}
	c.walk(t, info, nil, "", 0, maxDepth)
	return info
}

// walk goes through a struct's fields one by one, including nested structs.
//
// Takes t (reflect.Type) which is the struct type to examine.
// Takes info (*structInfo) which stores the field details found.
// Takes parentIndex ([]int) which tracks the path through embedded fields.
// Takes parentPath (string) which builds the dot-separated field name.
// Takes depth (int) which is how deep the current call is.
// Takes maxDepth (int) which limits nesting depth to prevent stack overflow.
func (c *binderCache) walk(t reflect.Type, info *structInfo, parentIndex []int, parentPath string, depth int, maxDepth int) {
	if maxDepth > 0 && depth > maxDepth {
		return
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for field := range t.Fields() {
		c.processField(&field, field.Index[0], info, parentIndex, parentPath, depth, maxDepth)
	}
}

// processField handles all logic for a single struct field during the cache
// walk. It takes a pointer to a `reflect.StructField` to satisfy the 'hugeParam'
// linter.
//
// Takes field (*reflect.StructField) which is the struct field to process.
// Takes index (int) which is the field's position within its parent struct.
// Takes info (*structInfo) which accumulates field metadata during the walk.
// Takes parentIndex ([]int) which tracks the index path from the root struct.
// Takes parentPath (string) which is the dot-separated path to the parent.
// Takes depth (int) which is the current recursion depth.
// Takes maxDepth (int) which limits recursion to prevent infinite loops.
func (c *binderCache) processField(field *reflect.StructField, index int, info *structInfo, parentIndex []int, parentPath string, depth int, maxDepth int) {
	if !field.IsExported() {
		return
	}

	currentPath, ignored := parseFieldPath(field, parentPath)
	if ignored {
		return
	}

	currentIndex := make([]int, len(parentIndex)+1)
	copy(currentIndex, parentIndex)
	currentIndex[len(parentIndex)] = index
	fieldType := field.Type
	effectiveType := fieldType
	if effectiveType.Kind() == reflect.Ptr {
		effectiveType = effectiveType.Elem()
	}

	if field.Anonymous {
		if effectiveType.Kind() == reflect.Struct && !isCustomType(effectiveType) {
			c.walk(effectiveType, info, currentIndex, parentPath, depth+1, maxDepth)
		}
		return
	}

	unmarshalerInstance, _ := implementsTextUnmarshaler(effectiveType)

	canDirect := len(currentIndex) == 1 &&
		fieldType.Kind() != reflect.Ptr &&
		isPrimitiveKind(fieldType.Kind()) &&
		!hasWellKnownConverter(fieldType) &&
		unmarshalerInstance == nil

	fi := &fieldInfo{
		Type:        fieldType,
		unmarshaler: unmarshalerInstance,
		Path:        currentPath,
		Index:       currentIndex,
		Offset:      field.Offset,
		Kind:        fieldType.Kind(),
		CanDirect:   canDirect,
	}

	info.Fields[currentPath] = fi

	if effectiveType.Kind() == reflect.Struct && !isCustomType(effectiveType) {
		c.walk(effectiveType, info, currentIndex, currentPath, depth+1, maxDepth)
	}
}
