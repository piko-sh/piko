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

//go:build !safe && !(js && wasm)

package binder

import (
	"maps"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

// initialStructCacheCapacity is the starting size for the struct metadata cache.
// Most applications use only a small number of form struct types.
const initialStructCacheCapacity = 8

// binderCache stores struct metadata with thread-safe access.
// It uses sharded storage with uintptr keys for fast lookups.
type binderCache struct {
	// fast holds an atomic pointer to a read-only map for lock-free lookups.
	// The map is rebuilt on cache miss, which is rare after warmup.
	fast atomic.Pointer[map[uintptr]*structInfo]

	// slow holds struct info during cache warm-up.
	slow sync.Map

	// mu guards access to the fast map during rebuilds.
	mu sync.Mutex
}

// get retrieves or builds struct metadata for a type. Optimised for hot-path
// reads.
//
// Takes t (reflect.Type) which specifies the struct type to look up.
// Takes maxDepth (int) which limits the recursion depth for nested structs.
//
// Returns *structInfo which contains the cached or newly built metadata.
func (c *binderCache) get(t reflect.Type, maxDepth int) *structInfo {
	key := typeKey(t)

	if m := c.fast.Load(); m != nil {
		if info, ok := (*m)[key]; ok {
			return info
		}
	}

	return c.getSlow(t, key, maxDepth)
}

// getSlow handles cache misses with proper synchronisation.
//
// Takes t (reflect.Type) which specifies the type to look up or build.
// Takes key (uintptr) which is the cache key for the type.
// Takes maxDepth (int) which limits how deep to go when building metadata.
//
// Returns *structInfo which contains the type's binding metadata.
func (c *binderCache) getSlow(t reflect.Type, key uintptr, maxDepth int) *structInfo {
	if cached, ok := c.slow.Load(key); ok {
		if info, isStructInfo := cached.(*structInfo); isStructInfo {
			return info
		}
	}

	info := c.build(t, maxDepth)

	actual, _ := c.slow.LoadOrStore(key, info)
	if actualInfo, ok := actual.(*structInfo); ok {
		info = actualInfo
	}

	c.rebuildFastMap(key, info)

	return info
}

// rebuildFastMap rebuilds the fast map with a new entry.
//
// Takes key (uintptr) which identifies the struct type to cache.
// Takes info (*structInfo) which contains the cached binding metadata.
//
// Safe for concurrent use; acquires the mutex before changing the map.
func (c *binderCache) rebuildFastMap(key uintptr, info *structInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var newMap map[uintptr]*structInfo
	if old := c.fast.Load(); old != nil {
		newMap = make(map[uintptr]*structInfo, len(*old)+1)
		maps.Copy(newMap, *old)
	} else {
		newMap = make(map[uintptr]*structInfo, initialStructCacheCapacity)
	}
	newMap[key] = info

	c.fast.Store(&newMap)
}

// build performs one-time reflection to analyse a struct type and create its
// metadata.
//
// Takes t (reflect.Type) which specifies the struct type to analyse.
// Takes maxDepth (int) which limits how deep nested structs are processed.
//
// Returns *structInfo which contains the collected field metadata.
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
// Takes info (*structInfo) which collects details about each field.
// Takes parentIndex ([]int) which tracks the path of field positions.
// Takes parentPath (string) which tracks the path of field names.
// Takes depth (int) which is the current nesting level.
// Takes maxDepth (int) which limits nesting depth to prevent stack overflow.
func (c *binderCache) walk(t reflect.Type, info *structInfo, parentIndex []int, parentPath string, depth int, maxDepth int) {
	if maxDepth > 0 && depth > maxDepth {
		return
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	for field := range t.Fields() {
		c.processField(&field, field.Index[0], info, parentIndex, parentPath, depth, maxDepth)
	}
}

// processField handles a single struct field during the cache walk.
//
// Takes field (*reflect.StructField) which points to the field metadata; uses
// a pointer to satisfy the hugeParam linter.
// Takes index (int) which is the field's position within its parent struct.
// Takes info (*structInfo) which gathers the found field metadata.
// Takes parentIndex ([]int) which tracks the index path from the root struct.
// Takes parentPath (string) which is the dot-separated path to the parent.
// Takes depth (int) which is the current depth of nesting.
// Takes maxDepth (int) which limits how deep nested structs are processed.
func (c *binderCache) processField(field *reflect.StructField, index int, info *structInfo, parentIndex []int, parentPath string, depth int, maxDepth int) {
	if !field.IsExported() {
		return
	}

	currentPath, ignored := parseFieldPath(field, parentPath)
	if ignored {
		return
	}

	currentIndex := buildFieldIndex(parentIndex, index)
	fieldType := field.Type
	effectiveType := dereferenceType(fieldType)

	if field.Anonymous {
		if effectiveType.Kind() == reflect.Struct && !isCustomType(effectiveType) {
			c.walk(effectiveType, info, currentIndex, parentPath, depth+1, maxDepth)
		}
		return
	}

	fi := c.buildFieldInfo(field, fieldType, effectiveType, currentPath, currentIndex)
	info.Fields[currentPath] = fi

	if effectiveType.Kind() == reflect.Struct && !isCustomType(effectiveType) {
		c.walk(effectiveType, info, currentIndex, currentPath, depth+1, maxDepth)
	}
}

// buildFieldInfo creates a fieldInfo struct for a terminal field.
//
// Takes field (*reflect.StructField) which provides the struct field data.
// Takes fieldType (reflect.Type) which is the declared type of the field.
// Takes effectiveType (reflect.Type) which is the type used for unmarshalling.
// Takes path (string) which is the dot-separated path to the field.
// Takes index ([]int) which is the index path for nested field access.
//
// Returns *fieldInfo which contains the cached field data ready for binding.
func (*binderCache) buildFieldInfo(field *reflect.StructField, fieldType, effectiveType reflect.Type, path string, index []int) *fieldInfo {
	unmarshalerInstance, _ := implementsTextUnmarshaler(effectiveType)

	canDirect := canUseDirectAccess(fieldType, index, unmarshalerInstance)

	return &fieldInfo{
		Type:        fieldType,
		unmarshaler: unmarshalerInstance,
		Path:        path,
		Index:       index,
		Offset:      field.Offset,
		Kind:        fieldType.Kind(),
		CanDirect:   canDirect,
	}
}

// typeKey extracts a unique uintptr key from a reflect.Type for use as a map
// key.
//
// Takes t (reflect.Type) which is the type to extract a key from.
//
// Returns uintptr which is a unique key for the type.
//
// Uses unsafe to extract the data pointer from a reflect.Type interface. This
// is safe because:
//
//  1. Stable layout: Go's interface layout is (type, data) and this is stable
//     across Go versions. This is part of the runtime and used by the reflect
//     package itself.
//
//  2. Static type descriptors: The data pointer points to a type descriptor
//     (*rtype) which is a static, read-only part of the compiled binary. Type
//     descriptors are never moved by the garbage collector (they are not on
//     the heap), never freed (they exist for the lifetime of the program), and
//     are unique per type (the same type always has the same descriptor).
//
//  3. No pointer dereference: We only use the uintptr as a map key for
//     comparison. We never dereference it or convert it back to a pointer.
//
//  4. Checked by tests: See TestTypeKey_Consistency in unsafe_safety_test.go
//     which checks consistency, uniqueness, and concurrent safety.
//
// Using reflect.Type directly as a map key requires interface hashing, which
// involves comparing the type descriptor and possibly the underlying data.
// Using uintptr is a simple integer comparison. Profiling showed sync.Map.Load
// with reflect.Type keys used about 10% of CPU time due to this overhead.
func typeKey(t reflect.Type) uintptr {
	type iface struct {
		_    uintptr
		data uintptr
	}
	return (*iface)(unsafe.Pointer(&t)).data
}

// buildFieldIndex creates the full index path for a field by appending its
// position to the parent's index path.
//
// Takes parentIndex ([]int) which is the index path of the parent field.
// Takes index (int) which is the field's position within its parent.
//
// Returns []int which is the complete index path for the field.
func buildFieldIndex(parentIndex []int, index int) []int {
	currentIndex := make([]int, len(parentIndex)+1)
	copy(currentIndex, parentIndex)
	currentIndex[len(parentIndex)] = index
	return currentIndex
}

// dereferenceType returns the element type if t is a pointer, otherwise returns
// t unchanged.
//
// Takes t (reflect.Type) which is the type to check and dereference.
//
// Returns reflect.Type which is the element type for pointers, or the original
// type if t is not a pointer.
func dereferenceType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}

// canUseDirectAccess checks if a field can use unsafe direct memory access.
//
// Direct access requires all of the following:
//   - Single-level access (no nested traversal through embedded fields)
//   - Non-pointer type
//   - Primitive type (string, int, bool, float, etc.)
//   - No well-known converter (e.g. time.Duration is int64 but needs special
//     parsing)
//   - No TextUnmarshaler interface
//
// User-registered converters are checked at runtime since they are dynamic.
//
// Takes fieldType (reflect.Type) which is the type of the struct field.
// Takes index ([]int) which is the field index path for embedded fields.
// Takes unmarshaler (any) which is the TextUnmarshaler if the field has one.
//
// Returns bool which is true if the field can use direct memory access.
func canUseDirectAccess(fieldType reflect.Type, index []int, unmarshaler any) bool {
	return len(index) == 1 &&
		fieldType.Kind() != reflect.Pointer &&
		isPrimitiveKind(fieldType.Kind()) &&
		!hasWellKnownConverter(fieldType) &&
		unmarshaler == nil
}
