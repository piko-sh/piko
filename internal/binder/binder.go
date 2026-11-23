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
	"bytes"
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/runeutil"
	"piko.sh/piko/wdk/goroutine"
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

	// sliceElementBudgetFactor scales the slice element budget.
	sliceElementBudgetFactor = 4

	// minimumSliceElementBudget is the floor for the slice element budget.
	minimumSliceElementBudget = defaultMaxSliceSize

	// maxCachedPathExpressions bounds the parsed-path cache.
	maxCachedPathExpressions = 4_096

	// bindCancellationCheckInterval is how many fields are bound between cancellation
	// checks.
	bindCancellationCheckInterval = 256

	// DefaultMaxBindJSONBytes caps the raw payload BindJSON accepts.
	DefaultMaxBindJSONBytes int64 = 4 << 20

	// DefaultMaxFieldErrors caps the number of fields reported in binding errors.
	DefaultMaxFieldErrors int64 = 1_000

	// FieldErrorsTruncatedKey is the entry added to a field-error map that hit the cap, so a
	// client can tell a trimmed list from a complete one.
	FieldErrorsTruncatedKey = "_truncated"

	// initialMultiErrorCapacity is the initial capacity for MultiError maps.
	initialMultiErrorCapacity = 4
)

var (
	// errFieldNotFound distinguishes an unknown struct field from a malformed path or
	// invalid value while resolving nested form inputs.
	errFieldNotFound = errors.New("field not found")

	// ErrBindJSONTooLarge reports that a BindJSON payload exceeds the configured limit.
	ErrBindJSONTooLarge = errors.New("BindJSON input exceeds configured size limit")

	// ErrSliceElementBudgetExhausted is returned when a bind would over-allocate.
	ErrSliceElementBudgetExhausted = errors.New("slice element budget exhausted")

	// ErrTooManyFields is returned when a request holds more fields than the binder allows,
	// whether at the top level or inside a decoded subtree.
	ErrTooManyFields = errors.New("too many fields")

	// ErrValidationFailed reports that a bound destination violated its `validate:"..."`
	// tags.
	ErrValidationFailed = errors.New("validation failed")

	// log is the package-level logger for the binder package.
	log = logger_domain.GetLogger("piko/internal/binder")
)

var (
	// getBinder returns the lazily initialised singleton ASTBinder instance, building it on
	// first call via sync.OnceValue.
	getBinder = sync.OnceValue(NewASTBinder)

	// identifierLUT is a lookup table for valid Go identifier characters.
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
type ASTBinder struct {
	// converters stores user-registered type converters.
	converters sync.Map

	// astCache stores parsed path expressions, keyed by file path.
	astCache sync.Map

	// structValidator validates opted-in binds and is nil when validation is unavailable.
	structValidator atomic.Pointer[StructValidator]

	// cache stores parsed struct metadata for faster repeated bindings.
	cache binderCache

	// astCacheEntries counts cached paths to bound the memory used by submitted keys.
	astCacheEntries atomic.Int64

	// maxSliceSize limits the maximum slice index, with zero disabling the limit.
	maxSliceSize atomic.Int64

	// maxPathDepth limits path nesting, with zero disabling the limit.
	maxPathDepth atomic.Int64

	// maxPathLength limits the length of a field path in bytes, with zero disabling it.
	maxPathLength atomic.Int64

	// maxFieldCount limits fields in a submission, with zero disabling the limit.
	maxFieldCount atomic.Int64

	// maxValueLength limits the length of a field value in bytes, with zero disabling it.
	maxValueLength atomic.Int64

	// maxBindJSONBytes caps the raw JSON payload, with zero disabling the cap.
	maxBindJSONBytes atomic.Int64

	// maxFieldErrors caps per-field validation messages, with zero disabling the cap.
	maxFieldErrors atomic.Int64

	// hasConverters tracks whether any custom converters are registered.
	hasConverters atomic.Bool

	// ignoreUnknownKeys controls whether unknown struct fields are skipped without error.
	ignoreUnknownKeys atomic.Bool
}

// StructValidator validates a populated struct against its field tags.
type StructValidator interface {
	// Struct validates s and returns an error describing every failed constraint.
	//
	// Takes s (any) which is the populated struct to validate.
	//
	// Returns error when a constraint fails.
	Struct(s any) error
}

// FieldErrorReporter is an optional StructValidator extension that translates a
// validation failure into per-field messages.
type FieldErrorReporter interface {
	// FieldErrors maps err to messages keyed by the destination's form field name.
	//
	// Takes err (error) which is an error previously returned by Struct.
	// Takes destination (any) which is the struct that was validated, used to resolve Go
	// field names to their bind and json names.
	//
	// Returns map[string][]string keyed by form field name, or nil when err did not
	// originate from this validator.
	FieldErrors(err error, destination any) map[string][]string
}

// ValidationFailedError reports that binding succeeded but the destination violated its
// `validate:"..."` tags.
type ValidationFailedError struct {
	// Err is the underlying validator error.
	Err error

	// Fields holds per-field messages, keyed by form field name, when the validator could
	// supply them.
	Fields map[string][]string
}

// Error implements the error interface.
//
// Returns string which describes the validation failure.
func (e *ValidationFailedError) Error() string {
	if e.Err == nil {
		return "validating bound destination"
	}
	return "validating bound destination: " + e.Err.Error()
}

// Unwrap exposes the underlying validator error to errors.Is and errors.As.
//
// Returns error which is the validator's own error.
func (e *ValidationFailedError) Unwrap() error {
	return e.Err
}

// Is reports whether target is ErrValidationFailed, so callers can detect a constraint
// failure without importing the concrete type.
//
// Takes target (error) which is the error being compared against.
//
// Returns bool which is true when target is the validation sentinel.
func (*ValidationFailedError) Is(target error) bool {
	return target == ErrValidationFailed
}

// SafeMessage implements the framework's user-facing error contract.
//
// Returns string which names the failure without disclosing constraint internals.
func (*ValidationFailedError) SafeMessage() string {
	return "validation failed"
}

// StatusCode reports the HTTP status a validation failure answers with.
//
// Returns int which is 422.
func (*ValidationFailedError) StatusCode() int {
	return http.StatusUnprocessableEntity
}

// ErrorCode reports the machine-readable code a validation failure answers with.
//
// Returns string which is the stable error code clients switch on.
func (*ValidationFailedError) ErrorCode() string {
	return errorCodeValidationFailed
}

// NewASTBinder creates a new AST-powered binder with default settings.
//
// Returns *ASTBinder which is set up and ready for use.
func NewASTBinder() *ASTBinder {
	b := &ASTBinder{}
	b.hasConverters.Store(false)
	b.ignoreUnknownKeys.Store(false)
	b.maxSliceSize.Store(defaultMaxSliceSize)
	b.maxPathDepth.Store(defaultMaxPathDepth)
	b.maxPathLength.Store(defaultMaxPathLength)
	b.maxFieldCount.Store(defaultMaxFieldCount)
	b.maxValueLength.Store(defaultMaxValueLength)
	b.maxBindJSONBytes.Store(DefaultMaxBindJSONBytes)
	b.maxFieldErrors.Store(DefaultMaxFieldErrors)
	return b
}

// SetStructValidator installs the validator used by binds that opt in via WithValidation.
//
// Takes v (StructValidator) which validates bound destinations.
func (b *ASTBinder) SetStructValidator(v StructValidator) {
	if v == nil || holdsNilValue(v) {
		b.structValidator.Store(nil)
		return
	}
	b.structValidator.Store(&v)
}

// SetMaxBindJSONBytes sets the maximum payload size accepted by BindJSON.
//
// The limit can be changed while other calls bind values.
//
// Takes maxBytes (int64) which limits payload bytes, with nonpositive values disabling
// the limit.
func (b *ASTBinder) SetMaxBindJSONBytes(maxBytes int64) {
	if maxBytes < 0 {
		maxBytes = 0
	}
	b.maxBindJSONBytes.Store(maxBytes)
}

// SetMaxFieldErrors limits the number of fields reported in binding errors.
//
// The limit can be changed while other calls bind values.
//
// Takes maxErrors (int64) which caps reported fields, with nonpositive values disabling
// the cap.
func (b *ASTBinder) SetMaxFieldErrors(maxErrors int64) {
	if maxErrors < 0 {
		maxErrors = 0
	}
	b.maxFieldErrors.Store(maxErrors)
}

// MaxBindJSONBytes returns the active byte-size cap enforced by BindJSON.
//
// Returns int64 which is the current cap. A value of zero indicates no cap is enforced.
func (b *ASTBinder) MaxBindJSONBytes() int64 {
	return b.maxBindJSONBytes.Load()
}

// Bind populates the fields of the destination struct using data from the source map.
//
// When the call opts in via WithValidation and a validator is configured, the populated
// destination is validated once binding has succeeded.
//
// Takes destination (any) which is the destination struct pointer to populate.
// Takes source (map[string][]string) which provides the source data for binding.
// Takes opts (...Option) which override global settings for this call.
//
// Returns error which is nil on success. Per-field binding failures return a MultiError.
// Size limits, cancellation, invalid destinations, and enabled validation return other
// errors.
func (b *ASTBinder) Bind(ctx context.Context, destination any, source map[string][]string, opts ...Option) error {
	limits, err := b.bindWithoutValidation(ctx, destination, source, opts...)
	if settled := b.settleBindErrors(nil, err); settled != nil {
		return settled
	}
	return b.runValidation(ctx, limits, destination)
}

// BindMap populates a destination struct from decoded values.
//
// Use BindJSON to preserve raw JSON and integers that may lose precision on decoding.
//
// Takes destination (any) which is the destination struct pointer to populate.
// Takes source (map[string]any) which provides the source data for binding.
// Takes opts (...Option) which override global settings for this call.
//
// Returns error which is nil on success. Per-field binding failures return a MultiError.
// Size limits, cancellation, invalid destinations, and enabled validation return other
// errors.
func (b *ASTBinder) BindMap(ctx context.Context, destination any, source map[string]any, opts ...Option) error {
	limits := b.resolveLimits(opts)
	remaining, subtreeErrs := b.bindWholeSubtreeFields(destination, source, limits)
	flattened := flattenMapToFormData(remaining)
	_, bindErr := b.bindWithoutValidation(ctx, destination, flattened, opts...)
	if settled := b.settleBindErrors(subtreeErrs, bindErr); settled != nil {
		return settled
	}
	return b.runValidation(ctx, limits, destination)
}

// BindJSON populates a destination struct from raw JSON bytes.
//
// json.RawMessage fields retain their original bytes, and integers keep their precision.
// A top-level JSON null supplies no fields.
//
// Takes destination (any) which is the destination struct pointer to populate.
// Takes source ([]byte) which contains the raw JSON bytes to decode.
// Takes opts (...Option) which override global settings for this call.
//
// Returns error for binding, validation, cancellation or an invalid destination.
// Oversized input wraps ErrBindJSONTooLarge and maps to HTTP 413. Malformed JSON maps to
// HTTP 400 with a safe message.
func (b *ASTBinder) BindJSON(ctx context.Context, destination any, source []byte, opts ...Option) error {
	if maxBytes := b.maxBindJSONBytes.Load(); maxBytes > 0 && int64(len(source)) > maxBytes {
		return newLimitError(fmt.Errorf("BindJSON input %d bytes exceeds limit %d: %w", len(source), maxBytes, ErrBindJSONTooLarge))
	}
	var rawFields map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(source, &rawFields); err != nil {
		return &malformedJSONError{err: err}
	}
	limits := b.resolveLimits(opts)
	remaining, subtreeErrs := b.bindWholeSubtreeFieldsRaw(destination, rawFields, limits)
	flattened := flattenMapToFormData(remaining)
	_, bindErr := b.bindWithoutValidation(ctx, destination, flattened, opts...)
	if settled := b.settleBindErrors(subtreeErrs, bindErr); settled != nil {
		return settled
	}
	return b.runValidation(ctx, limits, destination)
}

// RegisterConverter registers a custom function to convert string values to a specific
// type.
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
// The limit can be changed while other calls bind values.
//
// Takes size (int) which is the limit, with zero or less disabling it.
func (b *ASTBinder) SetMaxSliceSize(size int) {
	if size < 0 {
		size = 0
	}
	b.maxSliceSize.Store(int64(size))
}

// SetMaxPathDepth sets the maximum nesting depth for form paths.
//
// The limit can be changed while other calls bind values.
//
// Takes depth (int) which caps nesting, with nonpositive values disabling the limit.
func (b *ASTBinder) SetMaxPathDepth(depth int) {
	if depth < 0 {
		depth = 0
	}
	b.maxPathDepth.Store(int64(depth))
}

// SetMaxPathLength sets the maximum length of a form path string.
//
// The limit can be changed while other calls bind values.
//
// Takes length (int) which caps path bytes, with nonpositive values disabling the limit.
func (b *ASTBinder) SetMaxPathLength(length int) {
	if length < 0 {
		length = 0
	}
	b.maxPathLength.Store(int64(length))
}

// SetMaxFieldCount sets the maximum number of fields allowed in a form submission.
//
// The limit can be changed while other calls bind values.
//
// Takes count (int) which caps the field count, with nonpositive values disabling it.
func (b *ASTBinder) SetMaxFieldCount(count int) {
	if count < 0 {
		count = 0
	}
	b.maxFieldCount.Store(int64(count))
}

// SetMaxValueLength sets the maximum byte length of a field value.
//
// The limit can be changed while other calls bind values.
//
// Takes length (int) which caps value bytes, with nonpositive values disabling the limit.
func (b *ASTBinder) SetMaxValueLength(length int) {
	if length < 0 {
		length = 0
	}
	b.maxValueLength.Store(int64(length))
}

// SetIgnoreUnknownKeys sets the global default for ignoring unknown form fields.
//
// Takes ignore (bool) which controls whether unknown fields are silently ignored (true)
// or cause an error for each unknown key (false, the default).
func (b *ASTBinder) SetIgnoreUnknownKeys(ignore bool) {
	b.ignoreUnknownKeys.Store(ignore)
}

// loadStructValidator returns the configured validator, or nil when none is set.
//
// Returns StructValidator which validates bound destinations, or nil.
func (b *ASTBinder) loadStructValidator() StructValidator {
	if stored := b.structValidator.Load(); stored != nil {
		return *stored
	}
	return nil
}

// runValidation validates destination when the call opted in and a validator is
// configured. Callers must invoke this only once every binding pass has succeeded, so a
// constraint never fires against a half-populated struct.
//
// Takes limits (binderOptions) which carry the already-resolved per-call settings.
// Takes destination (any) which is the freshly bound struct pointer.
//
// Returns error when a constraint fails, or nil.
func (b *ASTBinder) runValidation(ctx context.Context, limits binderOptions, destination any) error {
	if !limits.validate {
		return nil
	}

	validator := b.loadStructValidator()
	if validator == nil {
		return nil
	}

	if !isValidatableStruct(destination) {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("validating bound destination: %w", err)
	}

	var validatorErr error
	err := goroutine.SafeCall(ctx, "binder.validate", func() error {
		validatorErr = validator.Struct(destination)
		return validatorErr
	})
	if err == nil {
		return nil
	}

	if panicErr, ok := errors.AsType[*goroutine.PanicError](err); ok {
		return fmt.Errorf("validator panicked, which points at a malformed validate tag rather than the payload: %w", panicErr)
	}

	reporter, ok := validator.(FieldErrorReporter)
	if !ok {
		return &ValidationFailedError{Err: validatorErr, Fields: nil}
	}

	fields := reporter.FieldErrors(validatorErr, destination)
	if fields == nil {
		return fmt.Errorf("validating bound destination: %w", validatorErr)
	}
	return &ValidationFailedError{Err: validatorErr, Fields: b.capFieldErrors(fields)}
}

// capFieldErrors trims the reported messages to the configured ceiling.
//
// Takes fields (map[string][]string) which are the validator's per-field messages.
//
// Returns map[string][]string which holds at most the configured number of entries.
func (b *ASTBinder) capFieldErrors(fields map[string][]string) map[string][]string {
	limit := b.maxFieldErrors.Load()
	capped, trimmed := keepFirstFields(fields, limit)
	if trimmed {
		capped[FieldErrorsTruncatedKey] = []string{fieldErrorsTruncatedNotice(limit, len(fields))}
	}
	return capped
}

// capMultiError trims the binding failures to the configured ceiling.
//
// Takes bindingErrors (MultiError) which holds every per-field binding failure.
//
// Returns MultiError which holds at most the configured number of fields, plus
// FieldErrorsTruncatedKey when fields were dropped.
func (b *ASTBinder) capMultiError(bindingErrors MultiError) MultiError {
	limit := b.maxFieldErrors.Load()
	capped, trimmed := keepFirstFields(bindingErrors, limit)
	if trimmed {
		capped[FieldErrorsTruncatedKey] = errFieldErrorsTruncated{reported: limit, total: len(bindingErrors)}
	}
	return capped
}

// settleBindErrors turns the results of the binding passes into the single error a bind
// returns.
//
// Cancellation and size limit errors are returned without per-field failures. Other
// failures are trimmed to the configured ceiling.
//
// Takes subtreeErrs (MultiError) which holds the per-field subtree binding errors.
// Takes bindErr (error) which is the flatten-pass result, possibly nil.
//
// Returns error which is the error the bind returns, or nil when both passes succeeded.
func (b *ASTBinder) settleBindErrors(subtreeErrs MultiError, bindErr error) error {
	merged := mergeBindErrors(subtreeErrs, bindErr)
	if merged == nil {
		return nil
	}

	bindingErrors, ok := errors.AsType[MultiError](merged)
	if !ok {
		return merged
	}
	if limitErr := liftLimitError(bindingErrors); limitErr != nil {
		return limitErr
	}
	return b.capMultiError(bindingErrors)
}

// bindWithoutValidation performs the binding pass alone, leaving validation to the
// caller.
//
// Takes destination (any) which is the destination struct pointer to populate.
// Takes source (map[string][]string) which provides the source data for binding.
// Takes opts (...Option) which override global settings for this call.
//
// Returns binderOptions which are the resolved per-call settings, so the caller can run
// validation without resolving the options again.
// Returns error which is a MultiError holding every per-field binding failure, an error
// that answers with 413 when the source broke the field count limit, the cancellation
// when ctx ended during the bind, or nil.
func (b *ASTBinder) bindWithoutValidation(
	ctx context.Context,
	destination any,
	source map[string][]string,
	opts ...Option,
) (binderOptions, error) {
	if err := validateBindTarget(destination); err != nil {
		return binderOptions{}, fmt.Errorf("validating bind target: %w", err)
	}

	limits := b.resolveLimits(opts)
	limits.sliceElements = newSliceElementBudget(source)

	v := reflect.ValueOf(destination).Elem()

	if err := checkFieldCountLimit(source, limits.maxFieldCount); err != nil {
		return limits, fmt.Errorf("checking field count limit: %w", err)
	}

	structMeta := b.cache.get(v.Type(), limits.maxPathDepth)

	multiErrors, err := b.bindFields(ctx, v, source, structMeta, limits)
	if err != nil {
		return limits, err
	}
	if multiErrors != nil {
		return limits, multiErrors
	}
	return limits, nil
}

// resolveLimits resolves the effective binder options for a call, using the binder's
// global defaults when no per-call options are supplied.
//
// Takes opts ([]Option) which override the binder's global defaults for this call.
//
// Returns binderOptions which are the effective protection limits for the call.
func (b *ASTBinder) resolveLimits(opts []Option) binderOptions {
	if len(opts) == 0 {
		return b.loadDefaults()
	}
	bindOpts := &BindOptions{}
	for _, opt := range opts {
		opt(bindOpts)
	}
	return b.resolveOptions(bindOpts)
}

// bindWholeSubtreeFields binds each top-level opaque field directly from its decoded
// source subtree.
//
// Takes destination (any) which is the struct pointer being populated.
// Takes source (map[string]any) which is the decoded request body.
// Takes limits (binderOptions) which supply the DoS ceilings for the subtree.
//
// Returns map[string]any which holds the source keys still to be bound by the flatten
// pass.
// Returns MultiError which accumulates any per-field binding failures.
func (b *ASTBinder) bindWholeSubtreeFields(destination any, source map[string]any, limits binderOptions) (map[string]any, MultiError) {
	elem, ok := structPointerElem(destination)
	if !ok {
		return source, nil
	}

	var remaining map[string]any
	var errs MultiError

	present := func(key string) bool { _, present := source[key]; return present }
	for _, field := range reflect.VisibleFields(elem.Type()) {
		key, ok := b.subtreeFieldKey(field, present)
		if !ok {
			continue
		}
		target, err := elem.FieldByIndexErr(field.Index)
		if err != nil || !target.CanSet() {
			continue
		}

		if remaining == nil {
			remaining = maps.Clone(source)
		}
		delete(remaining, key)

		if err := bindSubtreeFieldValue(key, source[key], target, limits); err != nil {
			accumulateError(&errs, key, err)
		}
	}

	if remaining == nil {
		remaining = source
	}
	return remaining, errs
}

// bindWholeSubtreeFieldsRaw is the byte-preserving counterpart of bindWholeSubtreeFields
// used by BindJSON.
//
// Takes destination (any) which is the struct pointer being populated.
// Takes source (map[string]stdjson.RawMessage) which holds each top field's raw JSON.
// Takes limits (binderOptions) which supply the DoS ceilings for the subtree.
//
// Returns map[string]any which holds the decoded values left for the flatten pass.
// Returns MultiError which accumulates any per-field binding failures.
func (b *ASTBinder) bindWholeSubtreeFieldsRaw(destination any, source map[string]stdjson.RawMessage, limits binderOptions) (map[string]any, MultiError) {
	elem, ok := structPointerElem(destination)
	if !ok {
		return rawFieldsToAnyMap(source), nil
	}

	subtreeKeys := make(map[string]struct{})
	var errs MultiError

	present := func(key string) bool { _, present := source[key]; return present }
	for _, field := range reflect.VisibleFields(elem.Type()) {
		key, ok := b.subtreeFieldKey(field, present)
		if !ok {
			continue
		}
		target, err := elem.FieldByIndexErr(field.Index)
		if err != nil || !target.CanSet() {
			continue
		}
		subtreeKeys[key] = struct{}{}
		if err := bindSubtreeFieldRaw(key, source[key], target, limits); err != nil {
			accumulateError(&errs, key, err)
		}
	}

	remaining := make(map[string]any, len(source))
	for key, raw := range source {
		if _, isSubtree := subtreeKeys[key]; isSubtree {
			continue
		}

		value, err := decodeJSONUseNumber(raw)
		if err != nil {
			accumulateError(&errs, key, errInvalidPath{path: key, err: fmt.Errorf("decoding value: %w", err)})
			continue
		}
		remaining[key] = value
	}
	return remaining, errs
}

// subtreeFieldKey reports whether an exported struct field present in the source should
// be bound directly from its JSON subtree, returning the source key to bind from.
//
// Takes field (reflect.StructField) which is the destination struct field being
// considered.
// Takes present (func(string) bool) which reports whether the source holds a given key.
//
// Returns string which is the source key to bind from when the bool is true.
// Returns bool which is true when the field should be bound from its whole subtree.
func (b *ASTBinder) subtreeFieldKey(field reflect.StructField, present func(string) bool) (string, bool) {
	if field.PkgPath != "" || field.Anonymous {
		return "", false
	}
	key, ignored := parseFieldPath(&field, "")
	if ignored {
		return "", false
	}
	if !present(key) {
		return "", false
	}
	return key, b.isWholeSubtreeField(field.Type)
}

// subtreeStats summarises a decoded JSON value for DoS-limit checks.
type subtreeStats struct {
	// depth is the deepest object or array nesting level found.
	depth int

	// fieldCount is the total number of object keys and array elements.
	fieldCount int

	// maxArrayLen is the length of the longest array, in elements.
	maxArrayLen int

	// maxValueLen is the length of the longest string leaf, in bytes.
	maxValueLen int
}

// isWholeSubtreeField reports whether a field type must be bound by unmarshalling its
// JSON subtree directly rather than through the flatten-and-rebind path.
//
// Takes t (reflect.Type) which is the destination field type.
//
// Returns bool which is true when the field must be bound from its whole JSON subtree.
func (b *ASTBinder) isWholeSubtreeField(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeFor[stdjson.RawMessage]() {
		return true
	}
	if isCustomType(t) || hasWellKnownConverter(t) || b.hasRegisteredConverter(t) {
		return false
	}
	if _, ok := implementsTextUnmarshaler(t); ok {
		return false
	}
	switch t.Kind() {
	case reflect.Struct, reflect.Map, reflect.Interface:
		return true
	case reflect.Slice, reflect.Array:
		elem := t.Elem()
		for elem.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}

		if isCustomType(elem) || hasWellKnownConverter(elem) || b.hasRegisteredConverter(elem) {
			return false
		}
		if _, ok := implementsTextUnmarshaler(elem); ok {
			return false
		}
		switch elem.Kind() {
		case reflect.Struct, reflect.Map, reflect.Interface, reflect.Slice, reflect.Array:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// hasRegisteredConverter reports whether the type has a built-in or user-registered
// string converter, in which case it keeps the flatten path rather than a JSON
// round-trip.
//
// Takes t (reflect.Type) which is the type to inspect.
//
// Returns bool which is true when a built-in or user converter is registered for t.
func (b *ASTBinder) hasRegisteredConverter(t reflect.Type) bool {
	if _, ok := wellKnownTypeConverters[t]; ok {
		return true
	}
	return b.getUserConverter(t) != nil
}

// loadDefaults reads the binder's default limits.
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
		sliceElements:     nil,
		validate:          false,
	}
}

// resolveOptions creates the final binding settings by merging global defaults with
// per-call overrides.
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
		sliceElements:     nil,
		validate:          false,
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
	if opts.Validate != nil {
		limits.validate = *opts.Validate
	}

	return limits
}

// bindFields processes all fields in the source map and populates the destination struct.
//
// Takes v (reflect.Value) which is the destination struct to populate.
// Takes src (map[string][]string) which contains field paths mapped to values.
// Takes structMeta (*structInfo) which provides metadata about the struct.
// Takes limits (binderOptions) which specifies validation constraints.
//
// Returns MultiError which is nil on success, or contains the per-field binding errors.
// The MultiError is allocated lazily only when the first error occurs.
// Returns error which is the cancellation when ctx ends during the bind. It is kept out
// of the MultiError because it says nothing about the input.
func (b *ASTBinder) bindFields(
	ctx context.Context,
	v reflect.Value,
	src map[string][]string,
	structMeta *structInfo,
	limits binderOptions,
) (MultiError, error) {
	var multiErrors MultiError

	fieldsBound := 0
	for path, values := range src {
		fieldsBound++
		if fieldsBound%bindCancellationCheckInterval == 0 && ctx.Err() != nil {
			return multiErrors, bindCancelledError(ctx, fieldsBound)
		}
		if err := b.bindPathValues(ctx, v, path, values, structMeta, limits); err != nil {
			accumulateError(&multiErrors, path, err)
		}
	}

	return multiErrors, nil
}

// bindPathValues binds the values for a single form key onto the destination struct.
//
// A repeated bare key targeting a slice field fills the slice. Any other key uses the
// last value. An empty value list is skipped without error.
//
// Takes v (reflect.Value) which is the destination struct.
// Takes path (string) which is the form key.
// Takes values ([]string) which are the raw values for the key.
// Takes structMeta (*structInfo) which holds the destination field metadata.
// Takes limits (binderOptions) which specifies validation constraints.
//
// Returns error when validation fails or a value cannot be set.
func (b *ASTBinder) bindPathValues(ctx context.Context, v reflect.Value, path string, values []string, structMeta *structInfo, limits binderOptions) error {
	if err := validatePathLength(path, limits.maxPathLength); err != nil {
		return err
	}

	if len(values) == 0 {
		return nil
	}

	if len(values) > 1 {
		if fieldMeta, ok := coercibleSliceField(path, structMeta); ok {
			field := fieldByIndexSafe(v, fieldMeta.Index)
			return b.bindValuesToSliceField(field, values, path, limits)
		}
	}

	value := values[len(values)-1]

	if err := validateValueLength(path, value, limits.maxValueLength); err != nil {
		return err
	}

	return b.bindSingleField(ctx, v, path, value, structMeta, limits)
}

// coercibleSliceField reports whether a bare form key targets a coercible slice field.
//
// Takes path (string) which is the form key.
// Takes structMeta (*structInfo) which holds the destination field metadata.
//
// Returns *fieldInfo which describes the slice field, or nil when no such field matches.
// Returns bool which is true for a known slice field other than []byte after one
// dereference.
func coercibleSliceField(path string, structMeta *structInfo) (*fieldInfo, bool) {
	if !isSimpleIdentifier(path) {
		return nil, false
	}
	fieldMeta, ok := structMeta.Fields[path]
	if !ok {
		return nil, false
	}
	fieldType := fieldMeta.Type
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() != reflect.Slice || fieldType == reflect.TypeFor[[]byte]() {
		return nil, false
	}
	return fieldMeta, true
}

// bindValuesToSliceField converts repeated form values into the destination slice.
//
// Takes field (reflect.Value) which is the destination slice field (a nil *[]T is
// allocated).
// Takes values ([]string) which are the raw values to bind in order.
// Takes path (string) which identifies the field for error messages.
// Takes limits (binderOptions) which supplies maxValueLength and maxSliceSize.
//
// Returns error when a value is too long, the slice would exceed maxSliceSize, or an
// element cannot be converted.
func (b *ASTBinder) bindValuesToSliceField(field reflect.Value, values []string, path string, limits binderOptions) error {
	target := dereferencePointer(field)
	elementType := target.Type().Elem()
	elementInfo := newFieldInfoForType(path, elementType)

	for index, value := range values {
		if err := validateValueLength(path, value, limits.maxValueLength); err != nil {
			return err
		}
		if err := growSliceToFitIndex(target, index, limits.maxSliceSize, limits.sliceElements); err != nil {
			return errSetField{err: err, path: path, field: path, fieldType: target.Type().String()}
		}
		if err := b.convertAndSet(target.Index(index), value, path, elementInfo); err != nil {
			return err
		}
	}

	return nil
}

// bindSingleField attempts to bind a single field using fast path or slow path.
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
		log.Trace("Skipping unbindable form field",
			logger_domain.String("field", path),
			logger_domain.Error(err))
		return nil
	}
	return b.setByAST(v, pathAST, value, path, limits)
}

// getOrParseAST retrieves a cached AST or parses a new one.
//
// Takes path (string) which specifies the expression path to parse.
//
// Returns ast_domain.Expression which is the parsed or cached AST.
// Returns error when the path is empty or contains items that are not supported, such as
// operators, literals, or function calls.
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

	b.cachePathExpression(path, parsed)
	return parsed, nil
}

// cachePathExpression stores a parsed path expression when the cache has room.
//
// Takes path (string) which is the path expression that was parsed.
// Takes parsed (ast_domain.Expression) which is the parsed expression to cache.
func (b *ASTBinder) cachePathExpression(path string, parsed ast_domain.Expression) {
	if b.astCacheEntries.Load() >= maxCachedPathExpressions {
		return
	}

	if _, loaded := b.astCache.LoadOrStore(path, parsed); !loaded {
		b.astCacheEntries.Add(1)
	}
}

// binderOptions holds DoS protection limits and binding settings for a single Bind call.
type binderOptions struct {
	// sliceElements caps total materialised slice elements, with nil disabling the cap.
	sliceElements *sliceElementBudget

	// maxFieldCount is the maximum number of fields allowed.
	maxFieldCount int

	// maxPathLength is the maximum length allowed for a path. Zero means no limit.
	maxPathLength int

	// maxValueLength is the maximum allowed length for a single field value.
	maxValueLength int

	// maxPathDepth limits how deep nested paths can go. Zero means no limit.
	maxPathDepth int

	// maxSliceSize is the largest allowed slice index. Zero means no limit.
	maxSliceSize int

	// ignoreUnknownKeys allows unknown field names to be silently ignored during binding.
	ignoreUnknownKeys bool

	// validate runs the configured struct validator once binding has succeeded.
	validate bool
}

// sliceElementBudget caps the slice elements one bind call may materialise.
type sliceElementBudget struct {
	// remaining counts the elements still available to the call.
	remaining atomic.Int64

	// total is the budget the call started with, retained for error messages.
	total int64
}

// newSliceElementBudget builds the element budget for a bind call.
//
// Takes source (map[string][]string) which is the flattened source data for the call.
//
// Returns *sliceElementBudget which allows a multiple of the supplied value count, with a
// floor so a small source can still fill a slice up to the default size limit.
func newSliceElementBudget(source map[string][]string) *sliceElementBudget {
	valueCount := 0
	for _, values := range source {
		valueCount += len(values)
	}

	total := max(int64(valueCount)*sliceElementBudgetFactor, minimumSliceElementBudget)

	budget := &sliceElementBudget{remaining: atomic.Int64{}, total: total}
	budget.remaining.Store(total)
	return budget
}

// charge deducts materialised elements from the budget.
//
// Takes elements (int) which is the number of elements about to be allocated.
//
// Returns error which wraps ErrSliceElementBudgetExhausted and answers with 413 when the
// allocation exceeds the budget justified by the source data.
func (budget *sliceElementBudget) charge(elements int) error {
	if budget == nil || elements <= 0 {
		return nil
	}

	if budget.remaining.Add(-int64(elements)) < 0 {
		return newLimitError(fmt.Errorf(
			"growing a slice by %d elements exceeds the %d elements this input allows: %w",
			elements, budget.total, ErrSliceElementBudgetExhausted,
		))
	}
	return nil
}

// GetBinder returns the shared binder instance used for data binding.
//
// Returns *ASTBinder which is the shared binder instance.
func GetBinder() *ASTBinder {
	return getBinder()
}

// ActionInputSource selects the map an action input binds from, given the request's
// decoded arguments and that input's parameter name.
//
// Takes arguments (map[string]any) which are the request's decoded arguments.
// Takes key (string) which is the input's parameter name.
//
// Returns map[string]any which is the source to bind from.
func ActionInputSource(arguments map[string]any, key string) map[string]any {
	raw, present := arguments[key]
	if !present {
		return arguments
	}
	if nested, ok := raw.(map[string]any); ok {
		return nested
	}
	return map[string]any{}
}

// accumulateError adds an error to the MultiError map.
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

// holdsNilValue reports whether v carries a nil of a nilable kind.
//
// Takes v (StructValidator) which is the candidate validator.
//
// Returns bool which is true when v wraps a nil value.
func holdsNilValue(v StructValidator) bool {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Map, reflect.Chan, reflect.Slice, reflect.Interface:
		return value.IsNil()
	default:
		return !value.IsValid()
	}
}

// isValidatableStruct reports whether destination is something a struct validator can
// accept.
//
// Takes destination (any) which is the freshly bound value.
//
// Returns bool which is true when destination resolves to a struct.
func isValidatableStruct(destination any) bool {
	value := reflect.ValueOf(destination)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	return value.Kind() == reflect.Struct && value.Type() != reflect.TypeFor[time.Time]()
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

// checkFieldCountLimit checks that the number of form fields does not exceed the allowed
// limit.
//
// Takes src (map[string][]string) which contains the form fields to check.
// Takes maxFieldCount (int) which sets the maximum number of fields allowed.
//
// Returns error which wraps ErrTooManyFields and answers with 413 when the field count
// exceeds the limit.
func checkFieldCountLimit(src map[string][]string, maxFieldCount int) error {
	if maxFieldCount > 0 && len(src) > maxFieldCount {
		return newLimitError(fmt.Errorf("binder: number of form fields (%d) exceeds maximum limit of %d: %w",
			len(src), maxFieldCount, ErrTooManyFields))
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

// isSimpleIdentifier reports whether the path is a simple Go identifier that contains
// only letters, numbers, and underscores.
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

// rawFieldsToAnyMap decodes every raw field to a value on a best-effort basis so a
// non-struct destination still reaches the flatten pass, where the real target error is
// reported.
//
// Takes source (map[string]stdjson.RawMessage) which holds each field's raw JSON.
//
// Returns map[string]any which holds the successfully decoded field values.
func rawFieldsToAnyMap(source map[string]stdjson.RawMessage) map[string]any {
	out := make(map[string]any, len(source))
	for key, raw := range source {
		value, err := decodeJSONUseNumber(raw)
		if err != nil {
			continue
		}
		out[key] = value
	}
	return out
}

// structPointerElem returns the struct value that a non-nil pointer destination refers
// to.
//
// Takes destination (any) which should be a non-nil pointer to a struct.
//
// Returns reflect.Value which is the referenced struct value when the bool is true.
// Returns bool which is true when destination is a non-nil pointer to a struct.
func structPointerElem(destination any) (reflect.Value, bool) {
	rv := reflect.ValueOf(destination)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return reflect.Value{}, false
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	return elem, true
}

// bindSubtreeFieldValue enforces the DoS limits on an already-decoded JSON subtree and
// binds it into target.
//
// Takes key (string) which is the source key, used for error context.
// Takes raw (any) which is the already-decoded subtree value.
// Takes target (reflect.Value) which is the settable destination field.
// Takes limits (binderOptions) which supply the DoS ceilings for the subtree.
//
// Returns error which wraps any limit breach or decode failure, or nil on success.
func bindSubtreeFieldValue(key string, raw any, target reflect.Value, limits binderOptions) error {
	if err := checkSubtreeLimits(key, raw, limits); err != nil {
		return err
	}
	remapped := remapSubtreeKeys(raw, target.Type())
	data, err := stdjson.Marshal(remapped)
	if err != nil {
		return errSetField{path: key, field: key, fieldType: target.Type().String(), err: fmt.Errorf("re-encoding value for binding: %w", err)}
	}
	return decodeSubtreeBytes(key, data, target, !limits.ignoreUnknownKeys)
}

// bindSubtreeFieldRaw binds a whole-subtree field from its original JSON bytes,
// preserving a json.RawMessage field byte-for-byte and keeping integer precision for
// everything else.
//
// Takes key (string) which is the source key, used for error context.
// Takes raw (stdjson.RawMessage) which is the field's original JSON bytes.
// Takes target (reflect.Value) which is the settable destination field.
// Takes limits (binderOptions) which supply the DoS ceilings for the subtree.
//
// Returns error which wraps any limit breach or decode failure, or nil on success.
func bindSubtreeFieldRaw(key string, raw stdjson.RawMessage, target reflect.Value, limits binderOptions) error {
	decoded, err := decodeJSONUseNumber(raw)
	if err != nil {
		return errInvalidPath{path: key, err: fmt.Errorf("decoding value: %w", err)}
	}
	if err := checkSubtreeLimits(key, decoded, limits); err != nil {
		return err
	}
	if isRawMessageType(target.Type()) {
		return decodeSubtreeBytes(key, raw, target, false)
	}
	remapped := remapSubtreeKeys(decoded, target.Type())
	data, err := stdjson.Marshal(remapped)
	if err != nil {
		return errSetField{path: key, field: key, fieldType: target.Type().String(), err: fmt.Errorf("re-encoding value for binding: %w", err)}
	}
	return decodeSubtreeBytes(key, data, target, !limits.ignoreUnknownKeys)
}

// checkSubtreeLimits enforces binder limits on a JSON subtree.
//
// Too many entries produce a size limit error, as on the flattened path. Other limit
// breaches produce errInvalidPath so SafeMessage stays consistent.
//
// Takes key (string) which is the source key, used for error context.
// Takes raw (any) which is the decoded subtree value to inspect.
// Takes limits (binderOptions) which supply the DoS ceilings to enforce.
//
// Returns error which is a size limit breach or an errInvalidPath, or nil when within
// limits.
func checkSubtreeLimits(key string, raw any, limits binderOptions) error {
	stats := subtreeStatsOf(raw, limits.maxPathDepth)
	if limits.maxPathDepth > 0 && 1+stats.depth > limits.maxPathDepth {
		return errInvalidPath{path: key, err: fmt.Errorf("path depth exceeds maximum limit of %d", limits.maxPathDepth)}
	}
	if limits.maxFieldCount > 0 && stats.fieldCount > limits.maxFieldCount {
		return newLimitError(fmt.Errorf("binder: field '%s' holds %d entries, which exceeds maximum limit of %d: %w",
			runeutil.Truncate(key, maxQuotedValueRunes), stats.fieldCount, limits.maxFieldCount, ErrTooManyFields))
	}
	if limits.maxSliceSize > 0 && stats.maxArrayLen > limits.maxSliceSize {
		return errInvalidPath{path: key, err: fmt.Errorf("slice length (%d) exceeds maximum limit of %d", stats.maxArrayLen, limits.maxSliceSize)}
	}
	if limits.maxValueLength > 0 && stats.maxValueLen > limits.maxValueLength {
		return errInvalidPath{path: key, err: fmt.Errorf("value length exceeds maximum limit of %d", limits.maxValueLength)}
	}
	return nil
}

// subtreeStatsOf walks a decoded JSON value collecting its nesting depth, entry count,
// longest array and longest string value.
//
// Takes value (any) which is the decoded JSON value to summarise.
// Takes maxDepth (int) which caps the recursion depth, with zero disabling the bound.
//
// Returns subtreeStats which summarises the value for the DoS-limit checks.
func subtreeStatsOf(value any, maxDepth int) subtreeStats {
	var s subtreeStats
	walkSubtreeStats(value, 0, maxDepth, &s)
	return s
}

// walkSubtreeStats recursively accumulates a decoded JSON value's statistics into s,
// stopping once depth exceeds maxDepth.
//
// Takes value (any) which is the current decoded JSON node.
// Takes depth (int) which is the current nesting level.
// Takes maxDepth (int) which caps the recursion depth, with zero disabling the bound.
// Takes s (*subtreeStats) which accumulates the collected statistics.
func walkSubtreeStats(value any, depth, maxDepth int, s *subtreeStats) {
	if maxDepth > 0 && depth > maxDepth {
		s.depth = max(s.depth, depth)
		return
	}
	switch v := value.(type) {
	case map[string]any:
		s.depth = max(s.depth, depth+1)
		s.fieldCount += len(v)
		for _, child := range v {
			walkSubtreeStats(child, depth+1, maxDepth, s)
		}
	case []any:
		s.depth = max(s.depth, depth+1)
		s.fieldCount += len(v)
		s.maxArrayLen = max(s.maxArrayLen, len(v))
		for _, child := range v {
			walkSubtreeStats(child, depth+1, maxDepth, s)
		}
	case string:
		s.maxValueLen = max(s.maxValueLen, len(v))
	default:
	}
}

// decodeSubtreeBytes unmarshals JSON bytes into target, rejecting unknown struct fields
// in strict mode.
//
// Takes key (string) which is the source key, used for error context.
// Takes data ([]byte) which is the JSON to decode.
// Takes target (reflect.Value) which is the settable destination field.
// Takes disallowUnknown (bool) which rejects unknown struct fields when true.
//
// Returns error which is an errSetField on failure, or nil on success.
func decodeSubtreeBytes(key string, data []byte, target reflect.Value, disallowUnknown bool) error {
	decoder := stdjson.NewDecoder(bytes.NewReader(data))
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target.Addr().Interface()); err != nil {
		return errSetField{path: key, field: key, fieldType: target.Type().String(), err: fmt.Errorf("decoding value: %w", err)}
	}
	return nil
}

// decodeJSONUseNumber decodes JSON bytes into a value, keeping numbers as json.Number so
// a subsequent re-encode does not lose the integer precision a float64 intermediate
// would.
//
// Takes data ([]byte) which is the JSON to decode.
//
// Returns any which is the decoded value with numbers preserved as json.Number.
// Returns error which is non-nil when the bytes are not valid JSON.
func decodeJSONUseNumber(data []byte) (any, error) {
	decoder := stdjson.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// isRawMessageType reports whether t, after dereferencing pointers, is a json.RawMessage.
//
// Takes t (reflect.Type) which is the type to inspect.
//
// Returns bool which is true when t is a json.RawMessage or pointer to one.
func isRawMessageType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t == reflect.TypeFor[stdjson.RawMessage]()
}

// remapSubtreeKeys rewrites a decoded JSON value's keys so a later encoding/json decode
// into t honours the binder's bind and json tag precedence, recursing through pointer,
// slice, array and map element types.
//
// Takes value (any) which is the decoded JSON value to rewrite.
// Takes t (reflect.Type) which is the destination type the value will decode into.
//
// Returns any which is the value with struct keys remapped for encoding/json.
func remapSubtreeKeys(value any, t reflect.Type) any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		nested, ok := value.(map[string]any)
		if !ok || isJSONNativeType(t) {
			return value
		}
		return remapStructKeys(nested, t)
	case reflect.Slice, reflect.Array:
		elements, ok := value.([]any)
		if !ok {
			return value
		}
		elementType := t.Elem()
		out := make([]any, len(elements))
		for i, child := range elements {
			out[i] = remapSubtreeKeys(child, elementType)
		}
		return out
	case reflect.Map:
		nested, ok := value.(map[string]any)
		if !ok {
			return value
		}
		elementType := t.Elem()
		out := make(map[string]any, len(nested))
		for key, child := range nested {
			out[key] = remapSubtreeKeys(child, elementType)
		}
		return out
	default:
		return value
	}
}

// remapStructKeys re-keys a decoded object so encoding/json binds it into t while
// honouring bind and json tag precedence.
//
// Takes source (map[string]any) which is the decoded object to re-key.
// Takes t (reflect.Type) which is the destination struct type.
//
// Returns map[string]any which is the object with keys matched to encoding/json field
// names.
func remapStructKeys(source map[string]any, t reflect.Type) map[string]any {
	out := make(map[string]any, len(source))
	matched := make(map[string]struct{}, len(source))
	for _, field := range reflect.VisibleFields(t) {
		if field.Anonymous || field.PkgPath != "" {
			continue
		}
		key, ignored := parseFieldPath(&field, "")
		if ignored {
			continue
		}
		child, present := source[key]
		if !present {
			continue
		}
		matched[key] = struct{}{}
		out[jsonMatchKey(&field)] = remapSubtreeKeys(child, field.Type)
	}
	for key, child := range source {
		if _, ok := matched[key]; ok {
			continue
		}
		if _, exists := out[key]; exists {
			continue
		}
		out[key] = child
	}
	return out
}

// jsonMatchKey returns the object key that encoding/json matches when decoding into
// field, preferring the json tag name and falling back to the Go field name.
//
// Takes field (*reflect.StructField) which is the destination struct field.
//
// Returns string which is the object key encoding/json will match for the field.
func jsonMatchKey(field *reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag != "" && tag != "-" {
		if name, _, _ := strings.Cut(tag, ","); name != "" {
			return name
		}
	}
	return field.Name
}

// isJSONNativeType reports whether a struct type manages its own JSON representation, in
// which case its internal fields must not be remapped.
//
// Takes t (reflect.Type) which is the struct type to inspect.
//
// Returns bool which is true for a well-known converter type or a json.Unmarshaler.
func isJSONNativeType(t reflect.Type) bool {
	if isCustomType(t) || hasWellKnownConverter(t) {
		return true
	}
	return implementsJSONUnmarshaler(t)
}

// implementsJSONUnmarshaler reports whether a type or its pointer form implements
// json.Unmarshaler.
//
// Takes t (reflect.Type) which is the type to inspect.
//
// Returns bool which is true when t or *t implements json.Unmarshaler.
func implementsJSONUnmarshaler(t reflect.Type) bool {
	ptr := reflect.New(t)
	if _, ok := reflect.TypeAssert[stdjson.Unmarshaler](ptr); ok {
		return true
	}
	if _, ok := reflect.TypeAssert[stdjson.Unmarshaler](ptr.Elem()); ok {
		return true
	}
	return false
}

// mergeBindErrors combines the subtree binding errors with the flatten-pass error into a
// single error, preserving MultiError semantics.
//
// A flatten-pass error that is not a MultiError, such as a cancellation or a broken size
// limit, stopped the whole bind, so it is returned on its own rather than filed under a
// field.
//
// Takes subtreeErrs (MultiError) which holds the per-field subtree binding errors.
// Takes bindErr (error) which is the flatten-pass result, possibly nil.
//
// Returns error which merges both sources, or nil when neither reported a failure.
func mergeBindErrors(subtreeErrs MultiError, bindErr error) error {
	if bindErr == nil {
		if len(subtreeErrs) == 0 {
			return nil
		}
		return subtreeErrs
	}

	flat, ok := errors.AsType[MultiError](bindErr)
	if !ok || len(subtreeErrs) == 0 {
		return bindErr
	}
	maps.Copy(subtreeErrs, flat)
	return subtreeErrs
}

// liftLimitError finds a size limit breach recorded against a field.
//
// Takes bindingErrors (MultiError) which holds the per-field binding errors.
//
// Returns error which is the first breach in field name order, or nil when there is none.
func liftLimitError(bindingErrors MultiError) error {
	for _, name := range slices.Sorted(maps.Keys(bindingErrors)) {
		if limitErr, ok := errors.AsType[*limitError](bindingErrors[name]); ok {
			return limitErr
		}
	}
	return nil
}

// keepFirstFields trims a field-keyed map to its first limit entries in name order, so
// the same input always keeps the same fields.
//
// Takes fields (map[string]V) which is the map to trim.
// Takes limit (int64) which is the most entries to keep. Zero or less keeps them all.
//
// Returns map[string]V which is fields itself when nothing was dropped, or a trimmed
// copy.
// Returns bool which is true when entries were dropped.
func keepFirstFields[V any](fields map[string]V, limit int64) (map[string]V, bool) {
	if limit <= 0 || int64(len(fields)) <= limit {
		return fields, false
	}

	kept := make(map[string]V, limit+1)
	for _, name := range slices.Sorted(maps.Keys(fields))[:limit] {
		kept[name] = fields[name]
	}
	return kept, true
}

// bindCancelledError describes a bind that stopped because ctx ended.
//
// Takes fieldsBound (int) which is how many fields were reached before the stop.
//
// Returns error which wraps both ctx.Err() and the cancellation cause, so a caller can
// match context.Canceled or context.DeadlineExceeded as well as its own cause.
func bindCancelledError(ctx context.Context, fieldsBound int) error {
	ctxErr := ctx.Err()
	cause := context.Cause(ctx)
	if cause == nil || errors.Is(cause, ctxErr) {
		return fmt.Errorf("binding cancelled after %d fields: %w", fieldsBound, ctxErr)
	}
	return fmt.Errorf("binding cancelled after %d fields: %w: %w", fieldsBound, ctxErr, cause)
}
