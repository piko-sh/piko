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

package provider_otter

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/cache/cache_dto"
)

const (
	// initialTypeCacheCapacity is the starting size for the global type accessor
	// cache. Most applications use a small number of cached struct types.
	initialTypeCacheCapacity = 8

	// decimalBase is base 10 for formatting integers as decimal strings.
	decimalBase = 10
)

// fieldAccessor holds cached metadata for field access without memory
// allocation. Uses unsafe pointer arithmetic for direct memory access.
type fieldAccessor struct {
	// indexes is the field index path for navigating through nested structs.
	indexes []int

	// offset is the byte offset from the struct start for single-level fields.
	offset uintptr

	// kind is the cached reflect.Kind for fast type switching.
	kind reflect.Kind

	// isDirect reports whether direct unsafe access can be used (non-pointer,
	// single-level access only).
	isDirect bool
}

// typeAccessorCache stores field accessors for each struct type.
// It uses atomic.Pointer for lock-free reads.
type typeAccessorCache struct {
	// fast holds a read-only map of accessors for each type, accessed without
	// locks.
	fast atomic.Pointer[map[uintptr]map[string]*fieldAccessor]

	// mu guards updates to the cache.
	mu sync.Mutex
}

// globalTypeCache is the global cache for field accessors.
var globalTypeCache = &typeAccessorCache{}

// FieldExtractor extracts field values from cached values based on a schema.
// It uses zero-allocation unsafe field access after cache warmup.
type FieldExtractor[V any] struct {
	// schema specifies which fields to extract from search results.
	schema *cache_dto.SearchSchema

	// sortableFields maps field names to whether they can be used for sorting.
	sortableFields map[string]bool

	// fieldPathParts holds field paths split into parts for fast lookup without
	// memory allocation. Built once at startup for all schema fields.
	fieldPathParts map[string][]string

	// fieldPathInvalid tracks invalid field paths to avoid repeated failed
	// lookups. Presence in map means the path is invalid; uses struct{} for zero
	// memory.
	fieldPathInvalid map[string]struct{}

	// textFields holds the names of TEXT fields used for full-text search.
	textFields []string

	// tagFields contains tag field names used for exact matching.
	tagFields []string

	// numericFields holds the names of numeric fields used for range queries.
	numericFields []string

	// vectorFields holds the names of VECTOR fields used for similarity search.
	vectorFields []string

	// cacheMu guards access to the field path cache.
	cacheMu sync.RWMutex
}

// ExtractTextFields returns all text field values from the value.
// Used for full-text search indexing.
//
// Takes value (V) which is the cached value to extract from.
//
// Returns []string containing text values from TEXT fields.
func (fe *FieldExtractor[V]) ExtractTextFields(value V) []string {
	if fe == nil {
		return nil
	}

	result := make([]string, 0, len(fe.textFields))
	for _, fieldName := range fe.textFields {
		if text := fe.extractString(value, fieldName); text != "" {
			result = append(result, text)
		}
	}
	return result
}

// ExtractTagValue returns the value of a tag field from the cached value.
//
// Takes value (V) which is the cached value to extract from.
// Takes fieldName (string) which is the name of the field to extract.
//
// Returns string which contains the tag value, or an empty string if not found.
func (fe *FieldExtractor[V]) ExtractTagValue(value V, fieldName string) string {
	if fe == nil {
		return ""
	}
	return fe.extractString(value, fieldName)
}

// ExtractNumericValue returns the value of a NUMERIC field as float64.
//
// Takes value (V) which is the cached value to extract from.
// Takes fieldName (string) which is the field to extract.
//
// Returns float64 containing the numeric value.
// Returns bool indicating if extraction succeeded.
func (fe *FieldExtractor[V]) ExtractNumericValue(value V, fieldName string) (float64, bool) {
	if fe == nil {
		return 0, false
	}
	return fe.extractNumeric(value, fieldName)
}

// ExtractSortableValue returns the value of a sortable field.
//
// Takes value (V) which is the cached value to extract from.
// Takes fieldName (string) which is the field to extract.
//
// Returns any containing the field value for sorting.
// Returns bool indicating if extraction succeeded.
func (fe *FieldExtractor[V]) ExtractSortableValue(value V, fieldName string) (any, bool) {
	if fe == nil || !fe.sortableFields[fieldName] {
		return nil, false
	}
	return fe.extractAny(value, fieldName)
}

// ExtractVectorValue returns the value of a VECTOR field as []float32.
//
// Takes value (V) which is the cached value to extract from.
// Takes fieldName (string) which is the name of the vector field.
//
// Returns []float32 containing the vector data.
// Returns bool indicating if extraction succeeded.
func (fe *FieldExtractor[V]) ExtractVectorValue(value V, fieldName string) ([]float32, bool) {
	if fe == nil {
		return nil, false
	}
	extracted, ok := fe.extractAny(value, fieldName)
	if !ok {
		return nil, false
	}
	vec, ok := extracted.([]float32)
	return vec, ok
}

// GetVectorFields returns the names of all VECTOR fields in the schema.
//
// Returns []string which contains the vector field names, or nil if the
// receiver is nil.
func (fe *FieldExtractor[V]) GetVectorFields() []string {
	if fe == nil {
		return nil
	}
	return fe.vectorFields
}

// IsSortable returns true if the field is marked as sortable.
//
// Takes fieldName (string) which specifies the field to check.
//
// Returns bool which is true if the field can be used for sorting.
func (fe *FieldExtractor[V]) IsSortable(fieldName string) bool {
	if fe == nil {
		return false
	}
	return fe.sortableFields[fieldName]
}

// GetSortableFields returns the names of all fields that support sorting.
//
// Returns []string which contains the sortable field names, or nil if the
// receiver is nil.
func (fe *FieldExtractor[V]) GetSortableFields() []string {
	if fe == nil {
		return nil
	}
	result := make([]string, 0, len(fe.sortableFields))
	for name := range fe.sortableFields {
		result = append(result, name)
	}
	return result
}

// extractString extracts a string value from a field, supporting dot notation.
//
// Takes value (V) which is the source value to extract from.
// Takes fieldPath (string) which specifies the field path using dot notation.
//
// Returns string which is the extracted value, or an empty string if not found.
func (fe *FieldExtractor[V]) extractString(value V, fieldPath string) string {
	extracted, ok := fe.extractAny(value, fieldPath)
	if !ok {
		return ""
	}
	return toString(extracted)
}

// extractNumeric extracts a numeric value from a field.
//
// Takes value (V) which is the source value to extract from.
// Takes fieldPath (string) which specifies the path to the field.
//
// Returns float64 which is the extracted numeric value.
// Returns bool which indicates whether extraction succeeded.
func (fe *FieldExtractor[V]) extractNumeric(value V, fieldPath string) (float64, bool) {
	extracted, ok := fe.extractAny(value, fieldPath)
	if !ok {
		return 0, false
	}
	return toFloat64(extracted)
}

// extractAny extracts any value from a field using zero-allocation unsafe
// access. Supports dot notation for nested fields (e.g., "address.city") and
// uses binder-style reflection caching to eliminate reflect.ValueOf
// allocations.
//
// Takes value (V) which is the struct instance to extract from.
// Takes fieldPath (string) which specifies the field path using dot notation.
//
// Returns any which is the extracted field value.
// Returns bool which indicates whether the extraction was successful.
//
// Safe for concurrent use. Uses a read lock when checking the invalid field
// path cache.
func (fe *FieldExtractor[V]) extractAny(value V, fieldPath string) (any, bool) {
	fe.cacheMu.RLock()
	_, isInvalid := fe.fieldPathInvalid[fieldPath]
	fe.cacheMu.RUnlock()

	if isInvalid {
		return nil, false
	}

	t := reflect.TypeFor[V]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	tKey := typeKey(t)

	if fastMap := globalTypeCache.fast.Load(); fastMap != nil {
		if accessors, ok := (*fastMap)[tKey]; ok {
			if accessor, ok := accessors[fieldPath]; ok {
				return fe.extractWithAccessor(value, accessor)
			}
		}
	}

	return fe.extractAndCache(value, fieldPath, t, tKey)
}

// extractAndCache builds a field accessor and caches it for future
// zero-allocation access.
//
// Takes value (V) which is the struct value to extract a field from.
// Takes fieldPath (string) which specifies the dot-separated path to the field.
// Takes t (reflect.Type) which is the reflected type of the value.
// Takes tKey (uintptr) which is the cache key derived from the type.
//
// Returns any which is the extracted field value.
// Returns bool which indicates whether extraction succeeded.
//
// Safe for concurrent use; uses mutex when updating the cache.
func (fe *FieldExtractor[V]) extractAndCache(value V, fieldPath string, t reflect.Type, tKey uintptr) (any, bool) {
	v := reflect.ValueOf(value)
	v = dereferencePointers(v)

	if v.Kind() != reflect.Struct {
		fe.cacheMu.Lock()
		fe.fieldPathInvalid[fieldPath] = struct{}{}
		fe.cacheMu.Unlock()
		return nil, false
	}

	fe.cacheMu.RLock()
	parts, hasPreSplit := fe.fieldPathParts[fieldPath]
	fe.cacheMu.RUnlock()

	if !hasPreSplit {
		parts = strings.Split(fieldPath, ".")
		fe.cacheMu.Lock()
		fe.fieldPathParts[fieldPath] = parts
		fe.cacheMu.Unlock()
	}

	accessor, result, ok := fe.buildAccessor(v, t, parts)
	if !ok {
		fe.cacheMu.Lock()
		fe.fieldPathInvalid[fieldPath] = struct{}{}
		fe.cacheMu.Unlock()
		return nil, false
	}

	fe.cacheAccessor(tKey, fieldPath, accessor)

	return result, true
}

// buildAccessor creates a fieldAccessor for a given path.
//
// Takes v (reflect.Value) which is the value to start from.
// Takes t (reflect.Type) which is the type to traverse.
// Takes parts ([]string) which contains the field path segments.
//
// Returns *fieldAccessor which provides access to the target field.
// Returns any which is the current value at the field path.
// Returns bool which indicates whether the accessor was built.
func (*FieldExtractor[V]) buildAccessor(v reflect.Value, t reflect.Type, parts []string) (*fieldAccessor, any, bool) {
	indexes := make([]int, 0, len(parts))
	var offset uintptr
	currentType := t

	for i, part := range parts {
		field, fieldIndex, ok := findFieldByNameInType(currentType, part)
		if !ok {
			return nil, nil, false
		}

		indexes = append(indexes, fieldIndex)

		if i == 0 {
			offset = field.Offset
		}

		v = v.Field(fieldIndex)
		v = dereferencePointers(v)
		currentType = field.Type
		if currentType.Kind() == reflect.Pointer {
			currentType = currentType.Elem()
		}

		if v.Kind() != reflect.Struct && i < len(parts)-1 {
			return nil, nil, false
		}
	}

	if !v.CanInterface() {
		return nil, nil, false
	}

	result := v.Interface()

	accessor := &fieldAccessor{
		indexes:  indexes,
		offset:   offset,
		kind:     v.Kind(),
		isDirect: len(indexes) == 1 && v.Kind() != reflect.Pointer && isPrimitiveKind(v.Kind()),
	}

	return accessor, result, true
}

// cacheAccessor stores a field accessor in the global type cache.
//
// Takes tKey (uintptr) which identifies the type in the cache.
// Takes fieldPath (string) which specifies the path to the field.
// Takes accessor (*fieldAccessor) which provides the accessor to store.
//
// Safe for concurrent use. Uses mutex locking with atomic swap to update the
// cache without blocking reads from other goroutines.
func (*FieldExtractor[V]) cacheAccessor(tKey uintptr, fieldPath string, accessor *fieldAccessor) {
	globalTypeCache.mu.Lock()
	defer globalTypeCache.mu.Unlock()

	var newMap map[uintptr]map[string]*fieldAccessor
	if old := globalTypeCache.fast.Load(); old != nil {
		newMap = make(map[uintptr]map[string]*fieldAccessor, len(*old)+1)
		maps.Copy(newMap, *old)
	} else {
		newMap = make(map[uintptr]map[string]*fieldAccessor, initialTypeCacheCapacity)
	}

	oldInner := newMap[tKey]
	newInner := make(map[string]*fieldAccessor, len(oldInner)+1)
	maps.Copy(newInner, oldInner)
	newInner[fieldPath] = accessor
	newMap[tKey] = newInner

	globalTypeCache.fast.Store(&newMap)
}

// NewFieldExtractor creates a new field extractor for the given schema.
//
// Takes schema (*cache_dto.SearchSchema) which defines the fields
// to extract from cached values.
//
// Returns *FieldExtractor[V] which is the configured extractor,
// or nil if schema is nil.
func NewFieldExtractor[V any](schema *cache_dto.SearchSchema) *FieldExtractor[V] {
	if schema == nil {
		return nil
	}

	fe := &FieldExtractor[V]{
		schema:           schema,
		textFields:       make([]string, 0),
		tagFields:        make([]string, 0),
		numericFields:    make([]string, 0),
		vectorFields:     make([]string, 0),
		sortableFields:   make(map[string]bool),
		fieldPathParts:   make(map[string][]string),
		fieldPathInvalid: make(map[string]struct{}),
	}

	for _, field := range schema.Fields {
		fe.fieldPathParts[field.Name] = strings.Split(field.Name, ".")
	}

	for _, field := range schema.Fields {
		switch field.Type {
		case cache_dto.FieldTypeText:
			fe.textFields = append(fe.textFields, field.Name)
		case cache_dto.FieldTypeTag:
			fe.tagFields = append(fe.tagFields, field.Name)
		case cache_dto.FieldTypeNumeric:
			fe.numericFields = append(fe.numericFields, field.Name)
		case cache_dto.FieldTypeVector:
			fe.vectorFields = append(fe.vectorFields, field.Name)
		default:
		}
		if field.Sortable {
			fe.sortableFields[field.Name] = true
		}
	}

	return fe
}

// findFieldByNameInType finds a field by name in a struct type using
// case-insensitive matching.
//
// Takes t (reflect.Type) which is the struct type to search.
// Takes name (string) which is the field name to find.
//
// Returns reflect.StructField which is the matching field descriptor.
// Returns int which is the index of the field, or -1 if not found.
// Returns bool which is true when a matching field was found.
func findFieldByNameInType(t reflect.Type, name string) (reflect.StructField, int, bool) {
	for field := range t.Fields() {
		if strings.EqualFold(field.Name, name) {
			return field, field.Index[0], true
		}
	}
	return reflect.StructField{}, -1, false
}

// isPrimitiveKind reports whether the given kind is a primitive type.
//
// Takes k (reflect.Kind) which is the kind to check.
//
// Returns bool which is true for bool, integer, float, and string kinds.
func isPrimitiveKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

// dereferencePointers unwraps all pointer indirections from a reflect.Value.
//
// Takes v (reflect.Value) which is the value to dereference.
//
// Returns reflect.Value which is the underlying non-pointer value, or the
// original value if a nil pointer is encountered.
func dereferencePointers(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}

// toString converts a value of any type to its string form.
// Uses strconv for number types for better speed.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the string form of the value.
func toString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case *string:
		if v == nil {
			return ""
		}
		return *v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, decimalBase)
	case int32:
		return strconv.FormatInt(int64(v), decimalBase)
	case int16:
		return strconv.FormatInt(int64(v), decimalBase)
	case int8:
		return strconv.FormatInt(int64(v), decimalBase)
	case uint:
		return strconv.FormatUint(uint64(v), decimalBase)
	case uint64:
		return strconv.FormatUint(v, decimalBase)
	case uint32:
		return strconv.FormatUint(uint64(v), decimalBase)
	case uint16:
		return strconv.FormatUint(uint64(v), decimalBase)
	case uint8:
		return strconv.FormatUint(uint64(v), decimalBase)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

// toFloat64 converts any numeric value to float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value, or zero if conversion fails.
// Returns bool which indicates whether the conversion succeeded.
func toFloat64(value any) (float64, bool) {
	if value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case int16:
		return float64(v), true
	case int8:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint8:
		return float64(v), true
	default:
		return 0, false
	}
}
