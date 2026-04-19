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
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// classSpaceSeparator is the space used to join CSS class names together.
	classSpaceSeparator = " "

	// defaultStyleMapCapacity is the starting capacity for style maps. Most
	// elements have fewer than 8 style properties.
	defaultStyleMapCapacity = 8

	// defaultClassMapCapacity is the starting capacity for class deduplication
	// maps.
	defaultClassMapCapacity = 16

	// defaultClassBufCapacity is the starting size for the class name buffer.
	defaultClassBufCapacity = 256

	// stylePartsCount3 is the number of parts for BuildStyleStringBytes3.
	stylePartsCount3 = 3

	// stylePartsCount4 is the number of parts for BuildStyleStringBytes4.
	stylePartsCount4 = 4

	// hiddenStyleSuffix is the CSS style string used to hide an element.
	hiddenStyleSuffix = "display:none !important;"
)

// stringTokeniser provides zero-allocation whitespace tokenisation.
// It replaces strings.FieldsSeq to avoid iterator struct allocation on each
// call.
type stringTokeniser struct {
	// input is the input string to tokenise.
	input string

	// position is the current byte position in the input string.
	position int
}

// next returns the next token from the string, split by whitespace.
//
// Returns token (string) which is the next token with whitespace removed.
// Returns ok (bool) which is true if a token was found, false at end of input.
func (t *stringTokeniser) next() (token string, ok bool) {
	for t.position < len(t.input) && t.input[t.position] <= ' ' {
		t.position++
	}
	if t.position >= len(t.input) {
		return "", false
	}
	start := t.position
	for t.position < len(t.input) && t.input[t.position] > ' ' {
		t.position++
	}
	return t.input[start:t.position], true
}

// delimTokeniser provides zero-allocation single-byte delimiter tokenisation.
// It replaces strings.SplitSeq to avoid the iterator/coroutine state escaping
// to the heap (see GitHub issue golang/go#73524).
type delimTokeniser struct {
	// input is the input string to tokenise.
	input string

	// position is the current byte position in the input string.
	position int

	// delimiter is the single-byte delimiter to split on.
	delimiter byte
}

// next returns the next segment between delimiters.
//
// Returns segment (string) which is the raw text between delimiters (not
// trimmed).
// Returns ok (bool) which is true if a segment was found, false at end of
// input.
func (t *delimTokeniser) next() (segment string, ok bool) {
	if t.position > len(t.input) {
		return "", false
	}
	start := t.position
	index := strings.IndexByte(t.input[start:], t.delimiter)
	if index < 0 {
		t.position = len(t.input) + 1
		return t.input[start:], true
	}
	t.position = start + index + 1
	return t.input[start : start+index], true
}

// classBuilder holds pooled resources for building class strings. The map
// tracks classes already added to avoid duplicates, while the buffer builds
// the space-separated result.
type classBuilder struct {
	// seen tracks class names already added to prevent duplicates.
	seen map[string]struct{}

	// buffer holds the space-separated class names as they are built.
	buffer *bytes.Buffer
}

var (
	// classBuilderPool provides reusable classBuilder instances to reduce
	// allocations. After warmup, buildClassString makes only one allocation (the
	// final result string).
	classBuilderPool = sync.Pool{
		New: func() any {
			return &classBuilder{
				seen:   make(map[string]struct{}, defaultClassMapCapacity),
				buffer: bytes.NewBuffer(make([]byte, 0, defaultClassBufCapacity)),
			}
		},
	}

	// styleBuilderPool provides reusable styleBuilder instances.
	styleBuilderPool = sync.Pool{
		New: func() any {
			return &styleBuilder{
				stylesMap: make(map[string]string, defaultStyleMapCapacity),
				keys:      make([]string, 0, defaultStyleMapCapacity),
			}
		},
	}

	// stringSlicePool provides reusable []string slices for collecting class
	// strings without per-call allocations. Used by MergeClassesBytes.
	stringSlicePool = sync.Pool{
		New: func() any {
			slice := make([]string, 0, defaultClassMapCapacity)
			return &slice
		},
	}

	// styleConcatPool provides reusable byte buffers for concatenating style
	// string fragments. Avoids a make([]byte) allocation on every call to
	// parseStylePartsToMap2/3/4 and the BuildStyleStringBytesV fallback path.
	styleConcatPool = sync.Pool{
		New: func() any {
			buffer := make([]byte, 0, defaultClassBufCapacity)
			return &buffer
		},
	}
)

// styleBuilder holds pooled resources for building style strings.
// It avoids creating a new map and keys slice on every call.
type styleBuilder struct {
	// stylesMap holds CSS property-value pairs during style building.
	stylesMap map[string]string

	// keys holds style property names for sorted iteration.
	keys []string
}

// reset clears the style builder for reuse without deallocating.
func (builder *styleBuilder) reset() {
	clear(builder.stylesMap)
	builder.keys = builder.keys[:0]
}

// reset clears the builder state for reuse without freeing memory.
func (builder *classBuilder) reset() {
	clear(builder.seen)
	builder.buffer.Reset()
}

// ClassesFromString takes a raw class string and returns a clean, formatted
// version.
//
// Takes class (string) which is the raw class string to process.
//
// Returns string which is the cleaned and formatted class string.
func ClassesFromString(class string) string {
	return buildClassString(class)
}

// ClassesFromSlice creates a class string from a slice of class names.
//
// Takes classes ([]string) which contains the class names to combine.
//
// Returns string which is the combined class string, or empty if the slice
// is empty.
func ClassesFromSlice(classes []string) string {
	if len(classes) == 0 {
		return ""
	}
	return buildClassString(strings.Join(classes, classSpaceSeparator))
}

// ClassesFromMapStringBool creates a class string from a map of class names
// to boolean values. This is the direct helper for the most common format used
// with conditional classes.
//
// Takes classes (map[string]bool) which maps class names to whether they
// should be included.
//
// Returns string which contains the class names separated by spaces, sorted
// for consistent output. Only class names with a true value are included.
func ClassesFromMapStringBool(classes map[string]bool) string {
	if len(classes) == 0 {
		return ""
	}

	count := 0
	for _, include := range classes {
		if include {
			count++
		}
	}
	if count == 0 {
		return ""
	}

	included := make([]string, 0, count)
	for class, include := range classes {
		if include {
			included = append(included, class)
		}
	}

	slices.Sort(included)

	return strings.Join(included, classSpaceSeparator)
}

// MergeClasses combines class attributes from multiple values using reflection.
//
// The Piko emitter will try to call a more specific helper first (such as
// MergeClassesFromMapStringBool) if the type is known at compile time. This
// function is a fallback for dynamic types like any or interface{}.
//
// When values is empty, returns an empty string.
//
// Takes values (...any) which provides the class values to combine.
//
// Returns string which contains the combined class attributes.
func MergeClasses(values ...any) string {
	if len(values) == 0 {
		return ""
	}

	classStrings := make([]string, 0, len(values))
	for _, styleValue := range values {
		classStrings = append(classStrings, extractClassString(styleValue))
	}
	return buildClassString(classStrings...)
}

// ClassesFromStringBytes is a version of ClassesFromString that avoids memory
// allocation by returning a pooled byte buffer.
//
// The returned buffer must be passed to DirectWriter.AppendPooledBytes()
// which will track it and release it back to the pool when Reset() is called.
//
// Takes class (string) which is the raw class string to process.
//
// Returns *[]byte which is a pooled buffer containing the processed class
// string, or nil if the input is empty.
func ClassesFromStringBytes(class string) *[]byte {
	return buildClassBytes(class)
}

// BuildClassBytesV builds a class string from variadic string parts without
// intermediate string concatenation allocations. This is designed to replace
// patterns like ClassesFromStringBytes("a " + x + " b") which allocate for
// each + operator.
//
// Takes parts (...string) which are class string fragments to combine.
//
// Returns *[]byte which is a pooled buffer containing the combined class
// string with duplicates removed, or nil if empty.
//
// Note: In large functions (like generated BuildAST), the variadic slice may
// escape to heap. Use the fixed-arity variants (BuildClassBytes2,
// BuildClassBytes4, etc.) for truly zero-allocation code generation.
func BuildClassBytesV(parts ...string) *[]byte {
	if len(parts) == 0 {
		return nil
	}
	return buildClassBytes(parts...)
}

// BuildClassBytes2 builds a class string from exactly two parts without
// allocation. Uses inline processing to avoid variadic slice heap escape in
// large functions.
//
// Takes a (string) which is the first class string part.
// Takes b (string) which is the second class string part.
//
// Returns *[]byte which contains the combined class string.
func BuildClassBytes2(a, b string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	return releaseClassBuilderToBytes(builder)
}

// BuildClassBytes4 builds a class string from exactly four parts without
// allocation. Uses inline processing to avoid variadic slice heap escape in
// large functions.
//
// Takes a (string) which is the first class part.
// Takes b (string) which is the second class part.
// Takes c (string) which is the third class part.
// Takes d (string) which is the fourth class part.
//
// Returns *[]byte which contains the combined class string.
func BuildClassBytes4(a, b, c, d string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	processClassString(builder, c)
	processClassString(builder, d)
	return releaseClassBuilderToBytes(builder)
}

// BuildClassBytes6 builds a class string from exactly 6 parts without
// allocation. Uses inline processing to avoid variadic slice heap escape in
// large functions.
//
// Takes a (string) which is the first class part.
// Takes b (string) which is the second class part.
// Takes c (string) which is the third class part.
// Takes d (string) which is the fourth class part.
// Takes e (string) which is the fifth class part.
// Takes f (string) which is the sixth class part.
//
// Returns *[]byte which contains the combined class string as bytes.
func BuildClassBytes6(a, b, c, d, e, f string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	processClassString(builder, c)
	processClassString(builder, d)
	processClassString(builder, e)
	processClassString(builder, f)
	return releaseClassBuilderToBytes(builder)
}

// BuildClassBytes8 builds a class string from exactly 8 parts without
// allocation. Uses inline processing to avoid variadic slice heap escape in
// large functions.
//
// Takes a (string) which is the first class part.
// Takes b (string) which is the second class part.
// Takes c (string) which is the third class part.
// Takes d (string) which is the fourth class part.
// Takes e (string) which is the fifth class part.
// Takes f (string) which is the sixth class part.
// Takes g (string) which is the seventh class part.
// Takes h (string) which is the eighth class part.
//
// Returns *[]byte which contains the combined class string.
//
// revive:disable:argument-limit Fixed-arity for zero-allocation hot path
func BuildClassBytes8(a, b, c, d, e, f, g, h string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	processClassString(builder, c)
	processClassString(builder, d)
	processClassString(builder, e)
	processClassString(builder, f)
	processClassString(builder, g)
	processClassString(builder, h)
	return releaseClassBuilderToBytes(builder)
}

// ClassesFromSliceBytes returns a pooled byte buffer containing the combined
// class names. This is a zero-allocation variant of ClassesFromSlice.
//
// Takes classes ([]string) which contains the class names to combine.
//
// Returns *[]byte which is a pooled buffer, or nil if the slice is empty.
func ClassesFromSliceBytes(classes []string) *[]byte {
	if len(classes) == 0 {
		return nil
	}
	return buildClassBytes(strings.Join(classes, classSpaceSeparator))
}

// MergeClassesBytes is a zero-allocation version of MergeClasses that returns
// a pooled byte buffer.
//
// Takes values (...any) which provides the class values to merge.
//
// Returns *[]byte which is a pooled buffer, or nil if empty.
func MergeClassesBytes(values ...any) *[]byte {
	if len(values) == 0 {
		return nil
	}

	slicePtr := getStringSlice()
	for _, styleValue := range values {
		*slicePtr = append(*slicePtr, extractClassString(styleValue))
	}
	result := buildClassBytes(*slicePtr...)
	putStringSlice(slicePtr)
	return result
}

// StylesFromString parses a raw style string and returns it in a clean,
// consistent format.
//
// Takes style (string) which is the raw style string to parse.
//
// Returns string which is the cleaned and formatted style string.
func StylesFromString(style string) string {
	stylesMap := make(map[string]string)
	parseStyleStringToMap(style, stylesMap)
	return buildStyleString(stylesMap)
}

// StylesFromStringMap creates a CSS style string from a map of style names
// to values.
//
// Takes styles (map[string]string) which provides the style property-value
// pairs to convert.
//
// Returns string which contains the formatted style string with properties
// in kebab-case and sorted order.
func StylesFromStringMap(styles map[string]string) string {
	targetMap := make(map[string]string, len(styles))
	mergeStyleMapString(targetMap, styles)
	return buildStyleString(targetMap)
}

// MergeStyles merges multiple style values into a single style string.
// The Piko emitter will attempt to call a more specific, strongly-typed
// helper first.
//
// Takes values (...any) which are style values to merge. Each value can be
// a string, map[string]string, or map[string]any.
//
// Returns string which contains the merged style attributes.
func MergeStyles(values ...any) string {
	if len(values) == 0 {
		return ""
	}

	stylesMap := make(map[string]string)

	for _, dynamicValue := range values {
		switch v := dynamicValue.(type) {
		case string:
			parseStyleStringToMap(v, stylesMap)
		case map[string]string:
			mergeStyleMapString(stylesMap, v)
		case map[string]any:
			mergeStyleMapAny(stylesMap, v)
		}
	}

	return buildStyleString(stylesMap)
}

// StylesFromStringBytes is a zero-allocation version of StylesFromString
// that returns a pooled byte buffer.
//
// The returned buffer must be passed to DirectWriter.AppendPooledBytes()
// which will track it and release it back to the pool when Reset() is called.
//
// Takes style (string) which is the raw style string to parse and normalise.
//
// Returns *[]byte which is a pooled buffer containing the processed style
// string, or nil if the input is empty or produces an empty result.
func StylesFromStringBytes(style string) *[]byte {
	builder, ok := styleBuilderPool.Get().(*styleBuilder)
	if !ok {
		builder = &styleBuilder{
			stylesMap: make(map[string]string, defaultStyleMapCapacity),
			keys:      make([]string, 0, defaultStyleMapCapacity),
		}
	}

	parseStyleStringToMap(style, builder.stylesMap)
	result := buildStyleBytesFromBuilder(builder)

	builder.reset()
	styleBuilderPool.Put(builder)
	return result
}

// StylesFromStringMapBytes converts a map of style properties to a byte
// buffer. It uses a pooled buffer for better performance.
//
// Takes styles (map[string]string) which provides the style property-value
// pairs to convert.
//
// Returns *[]byte which is a pooled buffer containing the formatted style
// string, or nil if the map is empty.
func StylesFromStringMapBytes(styles map[string]string) *[]byte {
	builder, ok := styleBuilderPool.Get().(*styleBuilder)
	if !ok {
		builder = &styleBuilder{
			stylesMap: make(map[string]string, defaultStyleMapCapacity),
			keys:      make([]string, 0, defaultStyleMapCapacity),
		}
	}

	mergeStyleMapString(builder.stylesMap, styles)
	result := buildStyleBytesFromBuilder(builder)

	builder.reset()
	styleBuilderPool.Put(builder)
	return result
}

// MergeStylesBytes is a zero-allocation variant of MergeStyles
// that returns a pooled byte buffer.
//
// Takes values (...any) which are style values to merge. Each value can be
// a string, map[string]string, or map[string]any.
//
// Returns *[]byte which is a pooled buffer containing the merged styles,
// or nil if empty.
func MergeStylesBytes(values ...any) *[]byte {
	if len(values) == 0 {
		return nil
	}

	builder, ok := styleBuilderPool.Get().(*styleBuilder)
	if !ok {
		builder = &styleBuilder{
			stylesMap: make(map[string]string, defaultStyleMapCapacity),
			keys:      make([]string, 0, defaultStyleMapCapacity),
		}
	}

	for _, dynamicValue := range values {
		switch v := dynamicValue.(type) {
		case string:
			parseStyleStringToMap(v, builder.stylesMap)
		case map[string]string:
			mergeStyleMapString(builder.stylesMap, v)
		case map[string]any:
			mergeStyleMapAny(builder.stylesMap, v)
		}
	}

	result := buildStyleBytesFromBuilder(builder)

	builder.reset()
	styleBuilderPool.Put(builder)
	return result
}

// AppendHiddenToStyleBytes appends "display:none !important;" to style bytes.
//
// Used for the p-show directive when the condition is false. CSS "last wins"
// semantics ensure the element is hidden even if display was set earlier.
//
// Takes bufferPointer (*[]byte) which is the existing style bytes from a *Bytes
// helper.
//
// Returns *[]byte which is the modified buffer. Returns the same pointer if
// input was non-nil, or a new pooled buffer if input was nil.
func AppendHiddenToStyleBytes(bufferPointer *[]byte) *[]byte {
	if bufferPointer == nil {
		return StylesFromStringBytes(hiddenStyleSuffix)
	}
	*bufferPointer = append(*bufferPointer, hiddenStyleSuffix...)
	return bufferPointer
}

// BuildStyleStringBytes2 joins two string parts and parses them as a style.
// This avoids the extra string allocation that happens with + joining.
//
// Takes a (string) which is the first part to join.
// Takes b (string) which is the second part to join.
//
// Returns *[]byte which is a pooled buffer containing the parsed style.
func BuildStyleStringBytes2(a, b string) *[]byte {
	builder, ok := styleBuilderPool.Get().(*styleBuilder)
	if !ok {
		builder = &styleBuilder{
			stylesMap: make(map[string]string, defaultStyleMapCapacity),
			keys:      make([]string, 0, defaultStyleMapCapacity),
		}
	}

	parseStylePartsToMap2(a, b, builder.stylesMap)
	result := buildStyleBytesFromBuilder(builder)

	builder.reset()
	styleBuilderPool.Put(builder)
	return result
}

// BuildStyleStringBytes3 joins three string parts and parses them as a style.
// This avoids the extra string allocation that happens with + joining.
//
// Takes a (string) which is the first part to join.
// Takes b (string) which is the second part to join.
// Takes c (string) which is the third part to join.
//
// Returns *[]byte which is a pooled buffer holding the parsed style.
func BuildStyleStringBytes3(a, b, c string) *[]byte {
	builder, ok := styleBuilderPool.Get().(*styleBuilder)
	if !ok {
		builder = &styleBuilder{
			stylesMap: make(map[string]string, defaultStyleMapCapacity),
			keys:      make([]string, 0, defaultStyleMapCapacity),
		}
	}

	parseStylePartsToMap3(a, b, c, builder.stylesMap)
	result := buildStyleBytesFromBuilder(builder)

	builder.reset()
	styleBuilderPool.Put(builder)
	return result
}

// BuildStyleStringBytes4 joins four string parts and parses them as a style.
// This avoids the extra string allocation that happens with + joining.
//
// Takes a (string) which is the first part to join.
// Takes b (string) which is the second part to join.
// Takes c (string) which is the third part to join.
// Takes d (string) which is the fourth part to join.
//
// Returns *[]byte which is a pooled buffer containing the parsed style.
func BuildStyleStringBytes4(a, b, c, d string) *[]byte {
	builder, ok := styleBuilderPool.Get().(*styleBuilder)
	if !ok {
		builder = &styleBuilder{
			stylesMap: make(map[string]string, defaultStyleMapCapacity),
			keys:      make([]string, 0, defaultStyleMapCapacity),
		}
	}

	parseStylePartsToMap4(a, b, c, d, builder.stylesMap)
	result := buildStyleBytesFromBuilder(builder)

	builder.reset()
	styleBuilderPool.Put(builder)
	return result
}

// BuildStyleStringBytesV joins string parts and parses them as a style.
// Uses fixed-arity helpers for up to four parts, and falls back to string
// joining for more.
//
// Takes parts (...string) which are the string parts to join.
//
// Returns *[]byte which is a pooled buffer holding the parsed style.
func BuildStyleStringBytesV(parts ...string) *[]byte {
	if len(parts) == 0 {
		return nil
	}

	switch len(parts) {
	case 1:
		return StylesFromStringBytes(parts[0])
	case 2:
		return BuildStyleStringBytes2(parts[0], parts[1])
	case stylePartsCount3:
		return BuildStyleStringBytes3(parts[0], parts[1], parts[2])
	case stylePartsCount4:
		return BuildStyleStringBytes4(parts[0], parts[1], parts[2], parts[3])
	}

	totalLen := 0
	for _, p := range parts {
		totalLen += len(p)
	}

	ptr := acquireStyleConcatBuf(totalLen)
	for _, p := range parts {
		*ptr = append(*ptr, p...)
	}
	result := StylesFromStringBytes(string(*ptr))
	releaseStyleConcatBuf(ptr)
	return result
}

// MergeClassesBytesArena is the arena-aware version of MergeClassesBytes.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes values (...any) which contains the class values to merge.
//
// Returns *[]byte which contains the merged class bytes, or nil if values
// is empty.
func MergeClassesBytesArena(arena *ast_domain.RenderArena, values ...any) *[]byte {
	if len(values) == 0 {
		return nil
	}

	slicePtr := getStringSlice()
	for _, styleValue := range values {
		*slicePtr = append(*slicePtr, extractClassString(styleValue))
	}
	result := buildClassBytesArena(arena, *slicePtr...)
	putStringSlice(slicePtr)
	return result
}

// ClassesFromSliceBytesArena is the arena-aware version of
// ClassesFromSliceBytes.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes classes ([]string) which contains the CSS class names to join.
//
// Returns *[]byte which contains the joined classes, or nil if classes is
// empty.
func ClassesFromSliceBytesArena(arena *ast_domain.RenderArena, classes []string) *[]byte {
	if len(classes) == 0 {
		return nil
	}
	return buildClassBytesArena(arena, strings.Join(classes, classSpaceSeparator))
}

// ClassesFromStringBytesArena is the arena-aware version of
// ClassesFromStringBytes.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes class (string) which specifies the CSS class string to convert.
//
// Returns *[]byte which contains the class bytes, or nil if class is empty.
func ClassesFromStringBytesArena(arena *ast_domain.RenderArena, class string) *[]byte {
	if class == "" {
		return nil
	}
	return buildClassBytesArena(arena, class)
}

// BuildClassBytes2Arena is the arena-aware version of BuildClassBytes2.
//
// Takes arena (*ast_domain.RenderArena) which manages memory allocation.
// Takes a (string) which is the first class string to process.
// Takes b (string) which is the second class string to process.
//
// Returns *[]byte which contains the combined processed class bytes.
func BuildClassBytes2Arena(arena *ast_domain.RenderArena, a, b string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	return releaseClassBuilderToBytesArena(arena, builder)
}

// BuildClassBytes4Arena is the arena-aware version of BuildClassBytes4.
//
// Takes arena (*ast_domain.RenderArena) which manages memory allocation.
// Takes a (string) which is the first CSS class string to process.
// Takes b (string) which is the second CSS class string to process.
// Takes c (string) which is the third CSS class string to process.
// Takes d (string) which is the fourth CSS class string to process.
//
// Returns *[]byte which contains the combined and processed class bytes.
func BuildClassBytes4Arena(arena *ast_domain.RenderArena, a, b, c, d string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	processClassString(builder, c)
	processClassString(builder, d)
	return releaseClassBuilderToBytesArena(arena, builder)
}

// BuildClassBytes6Arena is the arena-aware version of BuildClassBytes6.
//
// Takes arena (*ast_domain.RenderArena) which manages memory allocation for
// the result.
// Takes a (string) which is the first CSS class value.
// Takes b (string) which is the second CSS class value.
// Takes c (string) which is the third CSS class value.
// Takes d (string) which is the fourth CSS class value.
// Takes e (string) which is the fifth CSS class value.
// Takes f (string) which is the sixth CSS class value.
//
// Returns *[]byte which contains the combined class string allocated from
// the arena.
func BuildClassBytes6Arena(arena *ast_domain.RenderArena, a, b, c, d, e, f string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	processClassString(builder, c)
	processClassString(builder, d)
	processClassString(builder, e)
	processClassString(builder, f)
	return releaseClassBuilderToBytesArena(arena, builder)
}

// BuildClassBytes8Arena is the arena-aware version of BuildClassBytes8.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes a (string) which is the first CSS class string to process.
// Takes b (string) which is the second CSS class string to process.
// Takes c (string) which is the third CSS class string to process.
// Takes d (string) which is the fourth CSS class string to process.
// Takes e (string) which is the fifth CSS class string to process.
// Takes f (string) which is the sixth CSS class string to process.
// Takes g (string) which is the seventh CSS class string to process.
// Takes h (string) which is the eighth CSS class string to process.
//
// Returns *[]byte which contains the combined class attribute bytes.
func BuildClassBytes8Arena(arena *ast_domain.RenderArena, a, b, c, d, e, f, g, h string) *[]byte {
	builder := acquireClassBuilder()
	processClassString(builder, a)
	processClassString(builder, b)
	processClassString(builder, c)
	processClassString(builder, d)
	processClassString(builder, e)
	processClassString(builder, f)
	processClassString(builder, g)
	processClassString(builder, h)
	return releaseClassBuilderToBytesArena(arena, builder)
}

// BuildClassBytesVArena is the arena-aware version of BuildClassBytesV.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes parts (...string) which contains the class name segments to join.
//
// Returns *[]byte which contains the joined class bytes, or nil if parts is
// empty.
func BuildClassBytesVArena(arena *ast_domain.RenderArena, parts ...string) *[]byte {
	if len(parts) == 0 {
		return nil
	}
	return buildClassBytesArena(arena, parts...)
}

// StylesFromStringBytesArena is the arena-aware version of
// StylesFromStringBytes.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes style (string) which contains the CSS style string to parse.
//
// Returns *[]byte which contains the rendered style bytes, or nil if style is
// empty.
func StylesFromStringBytesArena(arena *ast_domain.RenderArena, style string) *[]byte {
	if style == "" {
		return nil
	}

	builder := acquireStyleBuilder()
	parseStyleStringToMap(style, builder.stylesMap)
	result := buildStyleBytesFromBuilderArena(arena, builder)
	releaseStyleBuilder(builder)
	return result
}

// StylesFromStringMapBytesArena is the arena-aware version of
// StylesFromStringMapBytes.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes styles (map[string]string) which contains the style key-value pairs.
//
// Returns *[]byte which contains the formatted style bytes, or nil if styles
// is empty.
func StylesFromStringMapBytesArena(arena *ast_domain.RenderArena, styles map[string]string) *[]byte {
	if len(styles) == 0 {
		return nil
	}

	builder := acquireStyleBuilder()
	mergeStyleMapString(builder.stylesMap, styles)
	result := buildStyleBytesFromBuilderArena(arena, builder)
	releaseStyleBuilder(builder)
	return result
}

// MergeStylesBytesArena is the arena-aware version of MergeStylesBytes.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes values (...any) which contains style values to merge (strings or maps).
//
// Returns *[]byte which contains the merged style bytes, or nil if values is
// empty.
func MergeStylesBytesArena(arena *ast_domain.RenderArena, values ...any) *[]byte {
	if len(values) == 0 {
		return nil
	}

	builder := acquireStyleBuilder()

	for _, dynamicValue := range values {
		switch v := dynamicValue.(type) {
		case string:
			parseStyleStringToMap(v, builder.stylesMap)
		case map[string]string:
			mergeStyleMapString(builder.stylesMap, v)
		case map[string]any:
			mergeStyleMapAny(builder.stylesMap, v)
		}
	}

	result := buildStyleBytesFromBuilderArena(arena, builder)
	releaseStyleBuilder(builder)
	return result
}

// BuildStyleStringBytes2Arena is the arena-aware version of
// BuildStyleStringBytes2.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes a (string) which is the first style string to parse.
// Takes b (string) which is the second style string to parse.
//
// Returns *[]byte which contains the combined and formatted style bytes.
func BuildStyleStringBytes2Arena(arena *ast_domain.RenderArena, a, b string) *[]byte {
	builder := acquireStyleBuilder()
	parseStylePartsToMap2(a, b, builder.stylesMap)
	result := buildStyleBytesFromBuilderArena(arena, builder)
	releaseStyleBuilder(builder)
	return result
}

// BuildStyleStringBytes3Arena is the arena-aware version of
// BuildStyleStringBytes3.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes a (string) which is the first style part to parse.
// Takes b (string) which is the second style part to parse.
// Takes c (string) which is the third style part to parse.
//
// Returns *[]byte which contains the combined style bytes.
func BuildStyleStringBytes3Arena(arena *ast_domain.RenderArena, a, b, c string) *[]byte {
	builder := acquireStyleBuilder()
	parseStylePartsToMap3(a, b, c, builder.stylesMap)
	result := buildStyleBytesFromBuilderArena(arena, builder)
	releaseStyleBuilder(builder)
	return result
}

// BuildStyleStringBytes4Arena is the arena-aware version of
// BuildStyleStringBytes4.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes a (string) which is the first style part.
// Takes b (string) which is the second style part.
// Takes c (string) which is the third style part.
// Takes d (string) which is the fourth style part.
//
// Returns *[]byte which contains the combined style string as bytes.
func BuildStyleStringBytes4Arena(arena *ast_domain.RenderArena, a, b, c, d string) *[]byte {
	builder := acquireStyleBuilder()
	parseStylePartsToMap4(a, b, c, d, builder.stylesMap)
	result := buildStyleBytesFromBuilderArena(arena, builder)
	releaseStyleBuilder(builder)
	return result
}

// BuildStyleStringBytesVArena is the arena-aware version of
// BuildStyleStringBytesV.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes parts (...string) which contains the style strings to combine.
//
// Returns *[]byte which contains the combined style bytes, or nil if parts
// is empty.
func BuildStyleStringBytesVArena(arena *ast_domain.RenderArena, parts ...string) *[]byte {
	if len(parts) == 0 {
		return nil
	}

	switch len(parts) {
	case 1:
		return StylesFromStringBytesArena(arena, parts[0])
	case 2:
		return BuildStyleStringBytes2Arena(arena, parts[0], parts[1])
	case stylePartsCount3:
		return BuildStyleStringBytes3Arena(arena, parts[0], parts[1], parts[2]) //nolint:gosec // len==3 checked by switch
	case stylePartsCount4:
		return BuildStyleStringBytes4Arena(arena, parts[0], parts[1], parts[2], parts[3]) //nolint:gosec // len==4 checked by switch
	}

	totalLen := 0
	for _, p := range parts {
		totalLen += len(p)
	}

	ptr := acquireStyleConcatBuf(totalLen)
	for _, p := range parts {
		*ptr = append(*ptr, p...)
	}

	builder := acquireStyleBuilder()
	parseStyleStringToMap(string(*ptr), builder.stylesMap)
	releaseStyleConcatBuf(ptr)
	result := buildStyleBytesFromBuilderArena(arena, builder)
	releaseStyleBuilder(builder)
	return result
}

// getStringSlice retrieves a pooled string slice, reset to zero length.
//
// Returns *[]string which is a pointer to a reusable slice from the pool.
func getStringSlice() *[]string {
	ptr, ok := stringSlicePool.Get().(*[]string)
	if !ok {
		slice := make([]string, 0, defaultClassMapCapacity)
		return &slice
	}
	*ptr = (*ptr)[:0]
	return ptr
}

// putStringSlice returns a string slice to the pool after clearing references.
//
// Takes ptr (*[]string) which points to the slice to return to the pool.
func putStringSlice(ptr *[]string) {
	if ptr == nil {
		return
	}
	slice := *ptr
	for i := range slice {
		slice[i] = ""
	}
	*ptr = slice[:0]
	stringSlicePool.Put(ptr)
}

// acquireStyleConcatBuf retrieves a pooled byte buffer, grown to at least
// minCap capacity.
//
// Takes minCap (int) which is the minimum capacity the returned buffer must
// have.
//
// Returns *[]byte which is a pooled buffer with length zero and at least
// minCap capacity.
func acquireStyleConcatBuf(minCap int) *[]byte {
	ptr, ok := styleConcatPool.Get().(*[]byte)
	if !ok {
		buffer := make([]byte, 0, minCap)
		return &buffer
	}
	*ptr = (*ptr)[:0]
	if cap(*ptr) < minCap {
		*ptr = make([]byte, 0, minCap)
	}
	return ptr
}

// releaseStyleConcatBuf returns a byte buffer to the pool.
//
// Takes ptr (*[]byte) which is the buffer to return to the pool.
func releaseStyleConcatBuf(ptr *[]byte) {
	if ptr == nil {
		return
	}
	styleConcatPool.Put(ptr)
}

// acquireStyleBuilder retrieves a pooled styleBuilder, ready for use.
//
// Returns *styleBuilder which has a cleared map and keys slice.
func acquireStyleBuilder() *styleBuilder {
	builder, ok := styleBuilderPool.Get().(*styleBuilder)
	if !ok {
		builder = &styleBuilder{
			stylesMap: make(map[string]string, defaultStyleMapCapacity),
			keys:      make([]string, 0, defaultStyleMapCapacity),
		}
	}
	return builder
}

// releaseStyleBuilder returns a styleBuilder to the pool after clearing it.
//
// Takes builder (*styleBuilder) which is the builder to return to the pool.
func releaseStyleBuilder(builder *styleBuilder) {
	builder.reset()
	styleBuilderPool.Put(builder)
}

// extractClassString converts a value to a CSS class string.
//
// Takes styleValue (any) which is the value to convert to class names.
// Supported types are string, []string, map[string]bool, map[string]any, and
// []any. Other types are handled by the default converter.
//
// Returns string which contains space-separated CSS class names.
func extractClassString(styleValue any) string {
	switch v := styleValue.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, classSpaceSeparator)
	case map[string]bool:
		return ClassesFromMapStringBool(v)
	case map[string]any:
		return classesFromMapStringAny(v)
	case []any:
		return classesFromSliceAny(v)
	default:
		return classFromDefault(v)
	}
}

// classesFromMapStringAny converts a map of class names to a class string.
//
// Takes v (map[string]any) which maps class names to values checked for
// truthiness to decide if they should be included.
//
// Returns string which contains the class names with truthy values, joined by
// spaces and sorted for consistent output.
func classesFromMapStringAny(v map[string]any) string {
	if len(v) == 0 {
		return ""
	}

	included := make([]string, 0, len(v))
	for class, include := range v {
		if EvaluateTruthiness(include) {
			included = append(included, class)
		}
	}
	if len(included) == 0 {
		return ""
	}

	slices.Sort(included)

	return strings.Join(included, classSpaceSeparator)
}

// classesFromSliceAny converts a slice of any values to a class string.
//
// Takes v ([]any) which contains the items to convert to class names.
//
// Returns string which holds the space-separated class names from the slice.
func classesFromSliceAny(v []any) string {
	if len(v) == 0 {
		return ""
	}

	items := make([]string, 0, len(v))
	for _, item := range v {
		if strItem, ok := item.(string); ok && strItem != "" {
			items = append(items, strItem)
		}
	}
	if len(items) == 0 {
		return ""
	}

	return strings.Join(items, classSpaceSeparator)
}

// classFromDefault extracts a class string from a default value.
//
// Takes v (any) which is the default value to extract a class string from.
//
// Returns string which is the formatted default value, or empty if v is nil
// or formats to an empty or "<nil>" string.
func classFromDefault(v any) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

// buildClassString combines CSS class names from multiple input strings into a
// single string with duplicates removed.
//
// Takes inputs (...string) which are class strings that may contain duplicates
// and extra whitespace.
//
// Returns string which contains unique class names separated by spaces, keeping
// the order in which each class first appears.
func buildClassString(inputs ...string) string {
	builder, ok := classBuilderPool.Get().(*classBuilder)
	if !ok {
		builder = &classBuilder{
			seen: make(map[string]struct{}),
		}
	}

	for _, input := range inputs {
		tokeniser := stringTokeniser{input: input}
		for cn, ok := tokeniser.next(); ok; cn, ok = tokeniser.next() {
			if _, exists := builder.seen[cn]; !exists {
				builder.seen[cn] = struct{}{}
				if builder.buffer.Len() > 0 {
					_ = builder.buffer.WriteByte(' ')
				}
				_, _ = builder.buffer.WriteString(cn)
			}
		}
	}

	result := builder.buffer.String()
	builder.reset()
	classBuilderPool.Put(builder)

	return result
}

// acquireClassBuilder gets a classBuilder from the pool.
//
// Returns *classBuilder which is either a recycled instance from the pool or
// a newly allocated one if the pool is empty.
func acquireClassBuilder() *classBuilder {
	builder, ok := classBuilderPool.Get().(*classBuilder)
	if !ok {
		builder = &classBuilder{
			seen: make(map[string]struct{}),
		}
	}
	return builder
}

// processClassString tokenises a class string and adds unique classes to the
// builder.
//
// Takes builder (*classBuilder) which accumulates the unique class names.
// Takes input (string) which contains space-separated class names to process.
func processClassString(builder *classBuilder, input string) {
	tokeniser := stringTokeniser{input: input}
	for cn, ok := tokeniser.next(); ok; cn, ok = tokeniser.next() {
		if _, exists := builder.seen[cn]; !exists {
			builder.seen[cn] = struct{}{}
			if builder.buffer.Len() > 0 {
				_ = builder.buffer.WriteByte(' ')
			}
			_, _ = builder.buffer.WriteString(cn)
		}
	}
}

// releaseClassBuilderToBytes extracts the result and returns the builder to
// the pool.
//
// Takes builder (*classBuilder) which is the builder to release.
//
// Returns *[]byte which contains the copied buffer contents, or nil if the
// buffer was empty.
func releaseClassBuilderToBytes(builder *classBuilder) *[]byte {
	if builder.buffer.Len() == 0 {
		builder.reset()
		classBuilderPool.Put(builder)
		return nil
	}

	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = append((*bufferPointer)[:0], builder.buffer.Bytes()...)

	builder.reset()
	classBuilderPool.Put(builder)

	return bufferPointer
}

// buildClassBytes combines CSS class strings and removes duplicates.
//
// Takes inputs (...string) which are class strings that may contain duplicates
// and extra whitespace.
//
// Returns *[]byte which is a pooled buffer with the cleaned class string, or
// nil if the result is empty. The caller must pass this buffer to
// DirectWriter.AppendPooledBytes() for proper memory handling.
func buildClassBytes(inputs ...string) *[]byte {
	builder, ok := classBuilderPool.Get().(*classBuilder)
	if !ok {
		builder = &classBuilder{
			seen: make(map[string]struct{}),
		}
	}

	for _, input := range inputs {
		tokeniser := stringTokeniser{input: input}
		for cn, ok := tokeniser.next(); ok; cn, ok = tokeniser.next() {
			if _, exists := builder.seen[cn]; !exists {
				builder.seen[cn] = struct{}{}
				if builder.buffer.Len() > 0 {
					_ = builder.buffer.WriteByte(' ')
				}
				_, _ = builder.buffer.WriteString(cn)
			}
		}
	}

	if builder.buffer.Len() == 0 {
		builder.reset()
		classBuilderPool.Put(builder)
		return nil
	}

	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = append((*bufferPointer)[:0], builder.buffer.Bytes()...)

	builder.reset()
	classBuilderPool.Put(builder)

	return bufferPointer
}

// mergeStyleMapString copies entries from a source map into a target map.
//
// Keys are converted to kebab-case and values have leading and trailing spaces
// removed. Non-empty values replace any existing entries. Empty values remove
// the key from the target map.
//
// Takes targetMap (map[string]string) which is the map to update.
// Takes sourceMap (map[string]string) which provides the entries to copy.
func mergeStyleMapString(targetMap, sourceMap map[string]string) {
	for key, styleValue := range sourceMap {
		kebabKey := toKebabCase(key)
		strVal := strings.TrimSpace(styleValue)
		if kebabKey != "" && strVal != "" {
			targetMap[kebabKey] = strVal
		} else if kebabKey != "" {
			delete(targetMap, kebabKey)
		}
	}
}

// mergeStyleMapAny merges styles from a map with any-typed values into a
// string style map.
//
// Takes targetMap (map[string]string) which is the destination style map.
// Takes sourceMap (map[string]any) which provides the styles to merge.
func mergeStyleMapAny(targetMap map[string]string, sourceMap map[string]any) {
	for key, styleValue := range sourceMap {
		kebabKey := toKebabCase(key)
		strVal := ""
		if styleValue != nil {
			strVal = strings.TrimSpace(fmt.Sprintf("%v", styleValue))
		}

		if kebabKey != "" && strVal != "" && strVal != "<nil>" {
			targetMap[kebabKey] = strVal
		} else if kebabKey != "" {
			delete(targetMap, kebabKey)
		}
	}
}

// parseStyleStringToMap parses a CSS style string into a map of key-value
// pairs.
//
// Takes s (string) which contains style declarations separated by semicolons,
// where each declaration has a property and value separated by a colon
// (e.g. "color: red; font-size: 12px").
// Takes targetMap (map[string]string) which receives the parsed property-value
// pairs.
func parseStyleStringToMap(s string, targetMap map[string]string) {
	if s == "" {
		return
	}
	tokeniser := delimTokeniser{input: s, delimiter: ';'}
	for declaration, ok := tokeniser.next(); ok; declaration, ok = tokeniser.next() {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}
		key, styleValue, found := strings.Cut(declaration, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		styleValue = strings.TrimSpace(styleValue)
		if key != "" && styleValue != "" {
			targetMap[key] = styleValue
		}
	}
}

// buildStyleString builds a sorted style string from a map of CSS properties.
//
// Takes stylesMap (map[string]string) which holds CSS property-value pairs.
//
// Returns string which contains the style attribute value with entries joined
// by semicolons. The output is sorted by property name to give the same result
// each time.
func buildStyleString(stylesMap map[string]string) string {
	if len(stylesMap) == 0 {
		return ""
	}

	keys := slices.Sorted(maps.Keys(stylesMap))

	size := 0
	for _, k := range keys {
		size += len(k) + 1 + len(stylesMap[k]) + 1
	}

	buffer := make([]byte, 0, size)
	for _, k := range keys {
		buffer = append(buffer, k...)
		buffer = append(buffer, ':')
		buffer = append(buffer, stylesMap[k]...)
		buffer = append(buffer, ';')
	}

	return string(buffer)
}

// toKebabCase converts a string from camelCase to kebab-case. This is used
// to normalise style property names.
//
// Takes s (string) which is the input string to convert.
//
// Returns string which is the kebab-case version of the input.
func toKebabCase(s string) string {
	if s == "" {
		return ""
	}

	hasUpper := false
	for i := range len(s) {
		if s[i] >= 'A' && s[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return s
	}

	buffer := make([]byte, 0, len(s)+len(s)/2)

	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				buffer = append(buffer, '-')
			}
			buffer = append(buffer, c+32)
		} else {
			buffer = append(buffer, c)
		}
	}

	return string(buffer)
}

// populateStyleBytes builds a deterministic, sorted, semicolon-delimited style
// byte buffer from the given map.
//
// Takes bufferPointer (*[]byte) which is the target buffer to populate.
// Takes stylesMap (map[string]string) which contains CSS property-value pairs.
// Takes keys ([]string) which specifies the order of properties to write.
func populateStyleBytes(bufferPointer *[]byte, stylesMap map[string]string, keys []string) {
	size := 0
	for _, k := range keys {
		size += len(k) + 1 + len(stylesMap[k]) + 1
	}

	if cap(*bufferPointer) < size {
		*bufferPointer = make([]byte, 0, size)
	} else {
		*bufferPointer = (*bufferPointer)[:0]
	}

	for _, k := range keys {
		*bufferPointer = append(*bufferPointer, k...)
		*bufferPointer = append(*bufferPointer, ':')
		*bufferPointer = append(*bufferPointer, stylesMap[k]...)
		*bufferPointer = append(*bufferPointer, ';')
	}
}

// buildStyleBytesFromBuilder builds a style byte buffer using a pooled
// styleBuilder to avoid creating a new map and keys slice.
//
// Takes builder (*styleBuilder) which contains the styles map and reusable keys
// slice.
//
// Returns *[]byte which is a pooled buffer containing the formatted style
// string, or nil if the map is empty.
func buildStyleBytesFromBuilder(builder *styleBuilder) *[]byte {
	if len(builder.stylesMap) == 0 {
		return nil
	}

	builder.keys = builder.keys[:0]
	for k := range builder.stylesMap {
		builder.keys = append(builder.keys, k)
	}
	slices.Sort(builder.keys)

	bufferPointer := ast_domain.GetByteBuf()
	populateStyleBytes(bufferPointer, builder.stylesMap, builder.keys)
	return bufferPointer
}

// parseStylePartsToMap2 concatenates two parts and parses them as CSS into the
// map.
//
// Takes a (string) which is the first CSS style part.
// Takes b (string) which is the second CSS style part.
// Takes stylesMap (map[string]string) which receives the parsed style entries.
func parseStylePartsToMap2(a, b string, stylesMap map[string]string) {
	ptr := acquireStyleConcatBuf(len(a) + len(b))
	*ptr = append(*ptr, a...)
	*ptr = append(*ptr, b...)
	parseStyleStringToMap(string(*ptr), stylesMap)
	releaseStyleConcatBuf(ptr)
}

// parseStylePartsToMap3 concatenates three parts and parses them as CSS into
// the map.
//
// Takes a (string) which is the first CSS fragment.
// Takes b (string) which is the second CSS fragment.
// Takes c (string) which is the third CSS fragment.
// Takes stylesMap (map[string]string) which receives the parsed style pairs.
func parseStylePartsToMap3(a, b, c string, stylesMap map[string]string) {
	ptr := acquireStyleConcatBuf(len(a) + len(b) + len(c))
	*ptr = append(*ptr, a...)
	*ptr = append(*ptr, b...)
	*ptr = append(*ptr, c...)
	parseStyleStringToMap(string(*ptr), stylesMap)
	releaseStyleConcatBuf(ptr)
}

// parseStylePartsToMap4 concatenates four style parts and parses them as CSS
// into the given map.
//
// Takes a (string) which is the first style part.
// Takes b (string) which is the second style part.
// Takes c (string) which is the third style part.
// Takes d (string) which is the fourth style part.
// Takes stylesMap (map[string]string) which receives the parsed CSS properties.
func parseStylePartsToMap4(a, b, c, d string, stylesMap map[string]string) {
	ptr := acquireStyleConcatBuf(len(a) + len(b) + len(c) + len(d))
	*ptr = append(*ptr, a...)
	*ptr = append(*ptr, b...)
	*ptr = append(*ptr, c...)
	*ptr = append(*ptr, d...)
	parseStyleStringToMap(string(*ptr), stylesMap)
	releaseStyleConcatBuf(ptr)
}

// buildClassBytesArena combines CSS class strings using arena-managed buffers.
//
// Takes arena (*ast_domain.RenderArena) which provides memory management for
// the output buffer.
// Takes inputs (...string) which are the CSS class strings to combine.
//
// Returns *[]byte which contains the combined class string in arena memory.
func buildClassBytesArena(arena *ast_domain.RenderArena, inputs ...string) *[]byte {
	builder := acquireClassBuilder()

	for _, input := range inputs {
		processClassString(builder, input)
	}

	return releaseClassBuilderToBytesArena(arena, builder)
}

// releaseClassBuilderToBytesArena releases a classBuilder and returns its
// contents as bytes using the provided arena.
//
// Takes arena (*ast_domain.RenderArena) which provides memory allocation.
// Takes builder (*classBuilder) which contains the class bytes to release.
//
// Returns *[]byte which contains the class bytes, or nil if the buffer is
// empty.
func releaseClassBuilderToBytesArena(arena *ast_domain.RenderArena, builder *classBuilder) *[]byte {
	if builder.buffer.Len() == 0 {
		builder.reset()
		classBuilderPool.Put(builder)
		return nil
	}

	bufferPointer := arena.GetByteBuf()
	*bufferPointer = append((*bufferPointer)[:0], builder.buffer.Bytes()...)

	builder.reset()
	classBuilderPool.Put(builder)

	return bufferPointer
}

// buildStyleBytesFromBuilderArena extracts style bytes from a builder using
// an arena for memory allocation.
//
// Takes arena (*ast_domain.RenderArena) which provides pooled byte buffers.
// Takes builder (*styleBuilder) which contains the styles to extract.
//
// Returns *[]byte which contains the sorted style bytes, or nil if no styles
// exist.
func buildStyleBytesFromBuilderArena(arena *ast_domain.RenderArena, builder *styleBuilder) *[]byte {
	if len(builder.stylesMap) == 0 {
		return nil
	}

	builder.keys = builder.keys[:0]
	for k := range builder.stylesMap {
		builder.keys = append(builder.keys, k)
	}
	slices.Sort(builder.keys)

	bufferPointer := arena.GetByteBuf()
	populateStyleBytes(bufferPointer, builder.stylesMap, builder.keys)
	return bufferPointer
}
