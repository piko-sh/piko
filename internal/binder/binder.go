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

package binder

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// defaultMaxSliceSize is the largest number of items allowed in a slice.
	defaultMaxSliceSize = 1_000

	// defaultMaxPathDepth is the maximum nesting level allowed when walking a path.
	defaultMaxPathDepth = 32

	// defaultMaxPathLength is the largest path length allowed, in bytes.
	defaultMaxPathLength = 4_096

	// defaultMaxFieldCount is the default limit for fields to process.
	defaultMaxFieldCount = 1_000

	// defaultMaxValueLength is the upper limit in bytes for field values.
	defaultMaxValueLength = 65_536

	// errFieldNotFound is the error message used when a struct field cannot be found.
	errFieldNotFound = "field not found"

	// initialMultiErrorCapacity is the initial capacity for MultiError maps.
	// Small since most binds succeed; grows if needed.
	initialMultiErrorCapacity = 4
)

var (
	// defaultBinder holds the lazily initialised singleton ASTBinder instance.
	defaultBinder *ASTBinder

	// binderOnce guards one-time initialisation of defaultBinder.
	binderOnce sync.Once

	// identifierLUT is a lookup table for valid Go identifier characters.
	// Using a lookup table replaces multiple comparisons with a single memory
	// access per byte, which is roughly twice as fast as the branch-based approach
	// for typical field names.
	identifierLUT = [256]bool{
		'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true,
		'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true,
		'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true,
		'y': true, 'z': true,
		'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true,
		'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true,
		'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true, 'X': true,
		'Y': true, 'Z': true,
		'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true,
		'8': true, '9': true,
		'_': true,
	}
)

// ASTBinder fills Go structs with form data using Piko path expressions.
// Field order is optimised for alignment (larger fields first, bools last).
type ASTBinder struct {
	// converters stores user-registered type converters. Uses sync.Map for
	// lock-free reads.
	converters sync.Map

	// astCache stores parsed path expressions, keyed by file path.
	astCache sync.Map

	// cache stores parsed struct metadata for faster repeated bindings.
	cache binderCache

	// maxSliceSize limits the maximum allowed slice index; 0 means no limit.
	maxSliceSize atomic.Int64

	// maxPathDepth limits how deeply paths can be nested; 0 means no limit.
	maxPathDepth atomic.Int64

	// maxPathLength limits how long a path string can be in characters.
	// A value of 0 means there is no limit.
	maxPathLength atomic.Int64

	// maxFieldCount stores the maximum number of fields allowed in a form
	// submission. A value of 0 means no limit.
	maxFieldCount atomic.Int64

	// maxValueLength stores the maximum allowed length for a field value in
	// characters. A value of 0 means no limit is applied.
	maxValueLength atomic.Int64

	// hasConverters tracks whether any custom converters are registered.
	// Used as a fast path to skip the map lookup when none exist.
	hasConverters atomic.Bool

	// ignoreUnknownKeys controls whether unknown struct fields are skipped
	// without error; false by default.
	ignoreUnknownKeys atomic.Bool
}

// NewASTBinder creates a new AST-powered binder with default settings.
//
// The returned binder is ready to use with all fields set to their defaults.
// All protection limits are enabled with sensible values. Unknown fields
// cause errors by default (strict mode).
//
// Returns *ASTBinder which is set up and ready for use.
func NewASTBinder() *ASTBinder {
	b := &ASTBinder{
		converters:        sync.Map{},
		hasConverters:     atomic.Bool{},
		cache:             binderCache{},
		astCache:          sync.Map{},
		ignoreUnknownKeys: atomic.Bool{},
		maxSliceSize:      atomic.Int64{},
		maxPathDepth:      atomic.Int64{},
		maxPathLength:     atomic.Int64{},
		maxFieldCount:     atomic.Int64{},
		maxValueLength:    atomic.Int64{},
	}
	b.hasConverters.Store(false)
	b.ignoreUnknownKeys.Store(false)
	b.maxSliceSize.Store(defaultMaxSliceSize)
	b.maxPathDepth.Store(defaultMaxPathDepth)
	b.maxPathLength.Store(defaultMaxPathLength)
	b.maxFieldCount.Store(defaultMaxFieldCount)
	b.maxValueLength.Store(defaultMaxValueLength)
	return b
}

// Bind populates the fields of the destination struct using data from the
// source map.
//
// Takes destination (any) which is the destination struct pointer to populate.
// Takes source (map[string][]string) which provides the source data for binding.
// Takes opts (...Option) which override global settings for this call.
//
// Returns error as a MultiError containing all binding errors, or nil if
// successful.
func (b *ASTBinder) Bind(ctx context.Context, destination any, source map[string][]string, opts ...Option) error {
	if err := validateBindTarget(destination); err != nil {
		return fmt.Errorf("validating bind target: %w", err)
	}

	var limits binderOptions
	if len(opts) == 0 {
		limits = b.loadDefaults()
	} else {
		bindOpts := &BindOptions{}
		for _, opt := range opts {
			opt(bindOpts)
		}
		limits = b.resolveOptions(bindOpts)
	}

	v := reflect.ValueOf(destination).Elem()

	if err := checkFieldCountLimit(source, limits.maxFieldCount); err != nil {
		return fmt.Errorf("checking field count limit: %w", err)
	}

	structMeta := b.cache.get(v.Type(), limits.maxPathDepth)

	multiErrors := b.bindFields(ctx, v, source, structMeta, limits)

	if multiErrors != nil {
		return multiErrors
	}
	return nil
}

// BindMap populates the fields of the destination struct using data from a
// map[string]any, typically produced by JSON decoding. It flattens the nested
// map into bracket-notation form data and delegates to the standard Bind
// pipeline.
//
// Takes destination (any) which is the destination struct pointer to populate.
// Takes source (map[string]any) which provides the source data for binding.
// Takes opts (...Option) which override global settings for this call.
//
// Returns error as a MultiError containing all binding errors, or nil if
// successful.
func (b *ASTBinder) BindMap(ctx context.Context, destination any, source map[string]any, opts ...Option) error {
	flattened := flattenMapToFormData(source)
	return b.Bind(ctx, destination, flattened, opts...)
}

// BindJSON populates the fields of the destination struct from raw JSON bytes.
// It decodes the JSON into a map[string]any, then delegates to BindMap.
//
// Takes destination (any) which is the destination struct pointer to populate.
// Takes source ([]byte) which contains the raw JSON bytes to decode.
// Takes opts (...Option) which override global settings for this call.
//
// Returns error when JSON decoding fails or binding errors occur.
func (b *ASTBinder) BindJSON(ctx context.Context, destination any, source []byte, opts ...Option) error {
	var m map[string]any
	if err := json.Unmarshal(source, &m); err != nil {
		return fmt.Errorf("decoding JSON for binding: %w", err)
	}
	return b.BindMap(ctx, destination, m, opts...)
}

// RegisterConverter registers a custom function to convert string values to a
// specific type. This takes precedence over all other conversion mechanisms
// and is safe for concurrent use.
//
// Takes typ (reflect.Type) which specifies the target type for conversion.
// Takes converter (ConverterFunc) which provides the conversion function.
func (b *ASTBinder) RegisterConverter(typ reflect.Type, converter ConverterFunc) {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	b.converters.Store(typ, converter)
	b.hasConverters.Store(true)
}

// SetMaxSliceSize sets the maximum allowed slice index for form binding.
//
// This prevents memory exhaustion attacks from malicious inputs like
// "items[9999999]". A value of 0 means no limit is enforced. This method
// is safe for concurrent use.
//
// Takes size (int) which specifies the maximum slice index allowed.
func (b *ASTBinder) SetMaxSliceSize(size int) {
	if size < 0 {
		size = 0
	}
	b.maxSliceSize.Store(int64(size))
}

// SetMaxPathDepth sets the maximum nesting depth for form paths, which
// prevents stack overflow from deeply nested paths like "a.b.c.d...".
//
// Takes depth (int) which specifies the maximum depth; a value of 0 or less
// means no limit is enforced.
func (b *ASTBinder) SetMaxPathDepth(depth int) {
	if depth < 0 {
		depth = 0
	}
	b.maxPathDepth.Store(int64(depth))
}

// SetMaxPathLength sets the maximum length of a form path string. This
// prevents CPU and memory exhaustion from extremely long path strings.
//
// Takes length (int) which specifies the maximum path length. A value of zero
// or less means no limit is enforced.
func (b *ASTBinder) SetMaxPathLength(length int) {
	if length < 0 {
		length = 0
	}
	b.maxPathLength.Store(int64(length))
}

// SetMaxFieldCount sets the maximum number of fields allowed in a form
// submission.
//
// Takes count (int) which specifies the field limit. A value of zero or less
// means no limit is enforced.
//
// This prevents hash-flooding DoS attacks from forms with thousands of keys.
func (b *ASTBinder) SetMaxFieldCount(count int) {
	if count < 0 {
		count = 0
	}
	b.maxFieldCount.Store(int64(count))
}

// SetMaxValueLength sets the maximum length of a field value string.
//
// Takes length (int) which specifies the maximum allowed length.
//
// This prevents CPU/memory exhaustion from malicious TextUnmarshaler
// implementations. A value of 0 means no limit is enforced. This method
// is safe for concurrent use.
func (b *ASTBinder) SetMaxValueLength(length int) {
	if length < 0 {
		length = 0
	}
	b.maxValueLength.Store(int64(length))
}

// SetIgnoreUnknownKeys sets the global default for ignoring unknown form fields.
// This method is safe for concurrent use.
//
// Takes ignore (bool) which controls whether unknown fields are silently
// ignored (true) or cause an error for each unknown key (false, the default).
func (b *ASTBinder) SetIgnoreUnknownKeys(ignore bool) {
	b.ignoreUnknownKeys.Store(ignore)
}

// loadDefaults gets all protection limits using the global defaults only.
// This is the fast path when no per-call options are given.
//
// Returns binderOptions which contains the current default limits.
func (b *ASTBinder) loadDefaults() binderOptions {
	return binderOptions{
		ignoreUnknownKeys: b.ignoreUnknownKeys.Load(),
		maxFieldCount:     int(b.maxFieldCount.Load()),
		maxPathLength:     int(b.maxPathLength.Load()),
		maxValueLength:    int(b.maxValueLength.Load()),
		maxPathDepth:      int(b.maxPathDepth.Load()),
		maxSliceSize:      int(b.maxSliceSize.Load()),
	}
}

// resolveOptions creates the final binding settings by merging global defaults
// with per-call overrides. Per-call options take priority over global settings.
//
// Takes opts (*BindOptions) which provides per-call overrides for limits.
//
// Returns binderOptions which contains the merged settings.
func (b *ASTBinder) resolveOptions(opts *BindOptions) binderOptions {
	limits := binderOptions{
		ignoreUnknownKeys: b.ignoreUnknownKeys.Load(),
		maxFieldCount:     int(b.maxFieldCount.Load()),
		maxPathLength:     int(b.maxPathLength.Load()),
		maxValueLength:    int(b.maxValueLength.Load()),
		maxPathDepth:      int(b.maxPathDepth.Load()),
		maxSliceSize:      int(b.maxSliceSize.Load()),
	}

	if opts.IgnoreUnknownKeys != nil {
		limits.ignoreUnknownKeys = *opts.IgnoreUnknownKeys
	}
	if opts.MaxFieldCount != nil {
		limits.maxFieldCount = *opts.MaxFieldCount
	}
	if opts.MaxPathLength != nil {
		limits.maxPathLength = *opts.MaxPathLength
	}
	if opts.MaxValueLength != nil {
		limits.maxValueLength = *opts.MaxValueLength
	}
	if opts.MaxPathDepth != nil {
		limits.maxPathDepth = *opts.MaxPathDepth
	}
	if opts.MaxSliceSize != nil {
		limits.maxSliceSize = *opts.MaxSliceSize
	}

	return limits
}

// bindFields processes all fields in the source map and populates the
// destination struct.
//
// Takes v (reflect.Value) which is the destination struct to populate.
// Takes src (map[string][]string) which contains field paths mapped to values.
// Takes structMeta (*structInfo) which provides metadata about the struct.
// Takes limits (binderOptions) which specifies validation constraints.
//
// Returns MultiError which is nil on success, or contains all binding errors.
// The MultiError is allocated lazily only when the first error occurs.
func (b *ASTBinder) bindFields(ctx context.Context, v reflect.Value, src map[string][]string, structMeta *structInfo, limits binderOptions) MultiError {
	var multiErrors MultiError

	for path, values := range src {
		if err := validatePathLength(path, limits.maxPathLength); err != nil {
			accumulateError(&multiErrors, path, err)
			continue
		}

		if len(values) == 0 {
			continue
		}
		value := values[len(values)-1]

		if err := validateValueLength(path, value, limits.maxValueLength); err != nil {
			accumulateError(&multiErrors, path, err)
			continue
		}

		if err := b.bindSingleField(ctx, v, path, value, structMeta, limits); err != nil {
			accumulateError(&multiErrors, path, err)
		}
	}

	return multiErrors
}

// bindSingleField attempts to bind a single field using fast path or slow
// path. Extracted method to separate fast/slow path logic.
//
// Takes v (reflect.Value) which is the struct value to bind the field on.
// Takes path (string) which is the field path expression to bind.
// Takes value (string) which is the value to assign to the field.
// Takes structMeta (*structInfo) which provides cached struct field metadata.
// Takes limits (binderOptions) which specifies binding constraints.
//
// Returns error when the path syntax is invalid or the value cannot be set.
func (b *ASTBinder) bindSingleField(ctx context.Context, v reflect.Value, path, value string, structMeta *structInfo, limits binderOptions) error {
	if isSimpleIdentifier(path) {
		if fieldMeta, ok := structMeta.Fields[path]; ok {
			if fieldMeta.CanDirect {
				return b.convertAndSetDirect(v, value, path, fieldMeta)
			}
			fieldVal := fieldByIndexSafe(v, fieldMeta.Index)
			return b.convertAndSet(fieldVal, value, path, fieldMeta)
		}
		return nil
	}

	pathAST, err := b.getOrParseAST(ctx, path)
	if err != nil {
		return fmt.Errorf("parsing AST for path %q: %w", path, err)
	}
	return b.setByAST(v, pathAST, value, path, limits)
}

// getOrParseAST retrieves a cached AST or parses a new one.
//
// Takes path (string) which specifies the expression path to parse.
//
// Returns ast_domain.Expression which is the parsed or cached AST.
// Returns error when the path is empty or contains items that are not
// supported, such as operators, literals, or function calls.
func (b *ASTBinder) getOrParseAST(ctx context.Context, path string) (ast_domain.Expression, error) {
	if cachedAST, ok := b.astCache.Load(path); ok {
		if pathAST, ok := cachedAST.(ast_domain.Expression); ok {
			return pathAST, nil
		}
	}

	parser := ast_domain.NewExpressionParser(ctx, path, "")
	parsed, diagnostics := parser.ParseExpression(ctx)
	if ast_domain.HasErrors(diagnostics) {
		return nil, errInvalidPath{path: path, err: diagnostics[0]}
	}

	if !isPathExpression(parsed) {
		return nil, errInvalidPath{path: path, err: errors.New("path cannot contain operators, literals, or function calls")}
	}

	b.astCache.Store(path, parsed)
	return parsed, nil
}

// binderOptions holds DoS protection limits and binding settings for a single
// Bind call. Values are loaded from atomics once at call start and passed
// through the stack to avoid repeated atomic loads in recursive functions.
type binderOptions struct {
	// ignoreUnknownKeys allows unknown field names to be silently
	// ignored during binding.
	ignoreUnknownKeys bool

	// maxFieldCount is the maximum number of fields allowed.
	// It provides protection against denial-of-service attacks.
	maxFieldCount int

	// maxPathLength is the maximum length allowed for a path; 0 means no limit.
	maxPathLength int

	// maxValueLength is the maximum allowed length for a single field value.
	maxValueLength int

	// maxPathDepth limits how deep nested paths can go; 0 means no limit.
	maxPathDepth int

	// maxSliceSize is the largest allowed slice index; 0 means no limit.
	maxSliceSize int
}

// GetBinder returns the shared binder instance used for data binding.
// The instance is safe for concurrent use and caches struct metadata for
// better performance.
//
// Returns *ASTBinder which is the shared binder instance.
func GetBinder() *ASTBinder {
	binderOnce.Do(func() {
		defaultBinder = NewASTBinder()
	})
	return defaultBinder
}

// accumulateError adds an error to the MultiError map.
//
// Creates the map on the first call if it does not exist yet.
//
// Takes multiErrors (*MultiError) which is the map to add the error to.
// Takes path (string) which is the key for the error in the map.
// Takes err (error) which is the error to store.
func accumulateError(multiErrors *MultiError, path string, err error) {
	if *multiErrors == nil {
		*multiErrors = make(MultiError, initialMultiErrorCapacity)
	}
	(*multiErrors)[path] = err
}

// validateBindTarget checks if the destination is a valid pointer to a struct.
//
// Takes destination (any) which is the value to check.
//
// Returns error when destination is nil or not a pointer to a struct.
func validateBindTarget(destination any) error {
	if destination == nil {
		return errInvalidTarget{targetType: "nil"}
	}
	v := reflect.ValueOf(destination)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		t := reflect.TypeOf(destination)
		return errInvalidTarget{targetType: t.String()}
	}
	return nil
}

// checkFieldCountLimit checks that the number of form fields does not exceed
// the allowed limit.
//
// Takes src (map[string][]string) which contains the form fields to check.
// Takes maxFieldCount (int) which sets the maximum number of fields allowed.
//
// Returns error when the field count exceeds the limit.
func checkFieldCountLimit(src map[string][]string, maxFieldCount int) error {
	if maxFieldCount > 0 && len(src) > maxFieldCount {
		return fmt.Errorf("binder: number of form fields (%d) exceeds maximum limit of %d", len(src), maxFieldCount)
	}
	return nil
}

// validatePathLength checks whether a path is longer than the allowed limit.
//
// Takes path (string) which is the file path to check.
// Takes maxPathLength (int) which is the maximum allowed path length.
//
// Returns error when the path length is greater than maxPathLength.
func validatePathLength(path string, maxPathLength int) error {
	if maxPathLength > 0 && len(path) > maxPathLength {
		return errInvalidPath{path: "...", err: fmt.Errorf("path length exceeds maximum limit of %d", maxPathLength)}
	}
	return nil
}

// validateValueLength checks if a value exceeds the allowed length limit.
//
// Takes path (string) which identifies the field location for error messages.
// Takes value (string) which is the value to check.
// Takes maxValueLength (int) which sets the maximum allowed length.
//
// Returns error when the value is longer than the maximum.
func validateValueLength(path, value string, maxValueLength int) error {
	if maxValueLength > 0 && len(value) > maxValueLength {
		return errSetField{err: fmt.Errorf("value length exceeds maximum limit of %d", maxValueLength), path: path, field: "", fieldType: ""}
	}
	return nil
}

// isSimpleIdentifier reports whether the path is a simple Go identifier
// that contains only letters, numbers, and underscores.
//
// This is a fast check that lets us skip AST parsing for simple form fields.
// It returns false for paths with operators, brackets, or spaces that need
// full parsing.
//
// Takes path (string) which is the path to check.
//
// Returns bool which is true if the path contains only identifier characters.
func isSimpleIdentifier(path string) bool {
	if len(path) == 0 {
		return false
	}
	for i := range len(path) {
		if !identifierLUT[path[i]] {
			return false
		}
	}
	return true
}
