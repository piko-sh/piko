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

package runtime

import (
	"context"
	"fmt"
	"reflect"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/generator/generator_helpers"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/markdown/markdown_dto"
	"piko.sh/piko/internal/search/search_adapters"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safeconv"
)

// SearchMode defines how much text analysis is applied during search.
type SearchMode string

type (
	// Annotation holds annotation data that is evaluated at runtime,
	// such as reactive bindings and directives that depend on component state.
	Annotation = ast_domain.RuntimeAnnotation

	// TemplateAST is the root of a compiled template tree.
	TemplateAST = ast_domain.TemplateAST

	// TemplateNode is the core building block of the runtime AST, representing an
	// element, text, or comment.
	TemplateNode = ast_domain.TemplateNode

	// HTMLAttribute represents a standard, static HTML attribute on a node.
	HTMLAttribute = ast_domain.HTMLAttribute

	// Directive contains the runtime information for a Piko directive (e.g., p-on,
	// p-model).
	Directive = ast_domain.Directive

	// NodeType defines the type of a node in the TemplateAST (e.g., NodeElement).
	NodeType = ast_domain.NodeType

	// InternalMetadata holds page-level metadata calculated during rendering.
	InternalMetadata = templater_dto.InternalMetadata

	// AssetRef represents a static asset needed by a component, used for
	// preloading.
	AssetRef = templater_dto.AssetRef

	// ActionPayload is the serialisable structure for a client-side action call.
	ActionPayload = templater_dto.ActionPayload

	// ActionArgument represents a single argument within an ActionPayload.
	ActionArgument = templater_dto.ActionArgument

	// RuntimeDiagnostic represents a warning or error that occurs during the
	// execution of the generated code (e.g., a nil-pointer access).
	RuntimeDiagnostic = generator_dto.RuntimeDiagnostic

	// Severity defines the level of a RuntimeDiagnostic (e.g., Warning, Error).
	Severity = generator_dto.Severity

	// DirectWriter holds structured writer parts for zero-allocation rendering.
	// Used by generated code to build hierarchical p-key values without string
	// concatenation.
	DirectWriter = ast_domain.DirectWriter

	// WriterPart represents one segment of a DirectWriter.
	WriterPart = ast_domain.WriterPart

	// WriterPartType discriminates between part types in a DirectWriter.
	WriterPartType = ast_domain.WriterPartType
)

const (
	// FilterOpEquals matches items where a field exactly equals the given value.
	FilterOpEquals = collection_dto.FilterOpEquals

	// FilterOpNotEquals matches items where the field does not equal the value.
	FilterOpNotEquals = collection_dto.FilterOpNotEquals

	// FilterOpGreaterThan matches items where the field is greater than the value.
	FilterOpGreaterThan = collection_dto.FilterOpGreaterThan

	// FilterOpGreaterEqual matches items where the field is greater than or equal
	// to the value.
	FilterOpGreaterEqual = collection_dto.FilterOpGreaterEqual

	// FilterOpLessThan matches items where the field value is less than the
	// specified comparison value.
	FilterOpLessThan = collection_dto.FilterOpLessThan

	// FilterOpLessEqual matches items where a field value is less than or equal to
	// the specified value.
	FilterOpLessEqual = collection_dto.FilterOpLessEqual

	// FilterOpContains matches items where a field contains the given substring.
	FilterOpContains = collection_dto.FilterOpContains

	// FilterOpStartsWith matches items where a field begins with the given prefix.
	FilterOpStartsWith = collection_dto.FilterOpStartsWith

	// FilterOpEndsWith matches items where the field value ends with a given
	// suffix.
	FilterOpEndsWith = collection_dto.FilterOpEndsWith

	// FilterOpIn is a filter operation that matches items where the field value is
	// in the provided array.
	FilterOpIn = collection_dto.FilterOpIn

	// FilterOpNotIn matches items where the field value is not in the provided
	// array.
	FilterOpNotIn = collection_dto.FilterOpNotIn

	// FilterOpExists matches items where the specified field exists or does not
	// exist, based on the boolean value provided.
	FilterOpExists = collection_dto.FilterOpExists

	// FilterOpFuzzyMatch performs fuzzy text matching that tolerates typos.
	FilterOpFuzzyMatch = collection_dto.FilterOpFuzzyMatch

	// SortAsc sorts in ascending order (A-Z, 0-9, oldest-newest).
	SortAsc = collection_dto.SortAsc

	// SortDesc sorts in descending order (Z-A, 9-0, newest to oldest).
	SortDesc = collection_dto.SortDesc

	// defaultFuzzyThreshold is the minimum similarity score for fuzzy matching.
	defaultFuzzyThreshold = 0.3

	// defaultQuickSearchLimit is the maximum number of results for quick searches.
	defaultQuickSearchLimit = 10

	// defaultBM25K1 is the term frequency saturation parameter for BM25 scoring.
	defaultBM25K1 = 1.2

	// defaultBM25B is the BM25 document length normalisation parameter.
	defaultBM25B = 0.75

	// defaultTitleWeight is the search weight applied to title field matches.
	defaultTitleWeight = 2.5

	// defaultContentWeight is the default search weight for content fields.
	defaultContentWeight = 1.0

	// SearchModeFast uses basic tokenisation and exact matching. It is designed
	// for speed with less than 10ms latency.
	SearchModeFast SearchMode = "fast"

	// SearchModeSmart uses stemming and phonetic encoding to handle misspellings
	// and language variations.
	SearchModeSmart SearchMode = "smart"

	// NodeElement is an alias for the AST domain NodeElement constant.
	NodeElement = ast_domain.NodeElement

	// NodeText is the node type for plain text content in the AST.
	NodeText = ast_domain.NodeText

	// NodeComment is the node type for comment nodes in the AST.
	NodeComment = ast_domain.NodeComment

	// NodeFragment is a node type representing a document fragment container.
	NodeFragment = ast_domain.NodeFragment

	// WriterPartString is the writer part type for string values.
	WriterPartString = ast_domain.WriterPartString

	// WriterPartInt is the writer part type for integer values.
	WriterPartInt = ast_domain.WriterPartInt

	// WriterPartUint is a writer part type for unsigned integer values.
	WriterPartUint = ast_domain.WriterPartUint

	// WriterPartFloat is the writer part type for float values.
	WriterPartFloat = ast_domain.WriterPartFloat

	// WriterPartBool is the writer component type for boolean values.
	WriterPartBool = ast_domain.WriterPartBool

	// WriterPartAny is a writer part type for values of any type.
	WriterPartAny = ast_domain.WriterPartAny

	// WriterPartEscapeString is a writer part type for escaped string output.
	WriterPartEscapeString = ast_domain.WriterPartEscapeString

	// WriterPartFNVString is the FNV hash writer part for string values.
	WriterPartFNVString = ast_domain.WriterPartFNVString

	// WriterPartFNVFloat is the writer part type for FNV-hashed float values.
	WriterPartFNVFloat = ast_domain.WriterPartFNVFloat

	// WriterPartFNVAny is a writer part that computes an FNV hash of any value.
	WriterPartFNVAny = ast_domain.WriterPartFNVAny

	// WriterPartBytes is the metric key for tracking bytes written per part.
	WriterPartBytes = ast_domain.WriterPartBytes

	// Debug is the debug log level from the generator DTO package.
	Debug = generator_dto.Debug

	// Info is an alias for the generator DTO info constant.
	Info = generator_dto.Info

	// Warning is the log level for warning messages that indicate potential
	// issues.
	Warning = generator_dto.Warning

	// Error is the generator error constant from the DTO package.
	Error = generator_dto.Error

	// httpStatusNotFound is the HTTP 404 status code returned for missing
	// collection items.
	httpStatusNotFound = 404
)

// RenderArena is a pooled container holding pre-allocated slabs for all AST
// types used during a single render request. Instead of ~807 individual pool
// Gets/Puts, the entire arena is obtained and released as a single unit.
type RenderArena = ast_domain.RenderArena

var (
	// GetDirectWriter retrieves a pooled DirectWriter for zero-allocation key
	// building. Use this in generated code instead of string concatenation for
	// p-key values.
	GetDirectWriter = ast_domain.GetDirectWriter

	// PutDirectWriter returns a DirectWriter to the pool after use. It is called
	// by TemplateNode.Reset when the node is returned to the pool.
	PutDirectWriter = ast_domain.PutDirectWriter

	// GetRuntimeAnnotation fetches a pooled RuntimeAnnotation.
	// Used by generated code to remove annotation allocations after warmup.
	GetRuntimeAnnotation = ast_domain.GetRuntimeAnnotation

	// GetTemplateAST retrieves a pooled TemplateAST for the root AST container.
	// Used by generated code to eliminate root AST allocations after warmup.
	GetTemplateAST = ast_domain.GetTemplateAST

	// GetRootNodesSlice retrieves a pooled slice for TemplateAST.RootNodes.
	// It rounds up to the nearest bucket size (1, 2, or 4) and falls back to
	// make() for capacities greater than 4.
	GetRootNodesSlice = ast_domain.GetRootNodesSlice

	// GetArena retrieves a pooled RenderArena for zero-allocation AST building.
	//
	// The arena provides all allocations for a single render request via
	// pre-allocated slabs, reducing pool operations from ~1,614 to just 2
	// (Get arena + Put arena). The arena must be attached to the TemplateAST
	// via SetArena so PutTree can release it automatically.
	GetArena = ast_domain.GetArena

	// PutArena returns a RenderArena to the pool after use. This is called
	// automatically by PutTree when the AST has an attached arena.
	PutArena = ast_domain.PutArena

	// EvaluateTruthiness provides JavaScript-like truthiness evaluation for any Go
	// type.
	EvaluateTruthiness = generator_helpers.EvaluateTruthiness

	// EvaluateStrictEquality provides Go-style strict equality comparison using
	// the == operator. It returns false if types do not match and uses optimised
	// type-specific comparisons.
	EvaluateStrictEquality = generator_helpers.EvaluateStrictEquality

	// EvaluateLooseEquality provides JavaScript-like loose equality (~=)
	// comparison. Compares by string representation, enabling type coercion like 0
	// ~= "0".
	EvaluateLooseEquality = generator_helpers.EvaluateLooseEquality

	// EvaluateOr implements JavaScript-like || operator semantics, returning the
	// first truthy value or the last value if none are truthy.
	EvaluateOr = generator_helpers.EvaluateOr

	// EvaluateCoalesce implements the JavaScript-like ?? (nullish
	// coalescing) operator, returning the first non-nil value or
	// the last value if all are nil, and unlike || treating only
	// nil as "nullish" (preserving "", 0, false).
	EvaluateCoalesce = generator_helpers.EvaluateCoalesce

	// EvaluateBinary performs a binary operation on two values at runtime.
	// Handles arithmetic (+, -, *, /, %) and comparison (>, <, >=, <=) for
	// maths.Decimal, maths.BigInt, and maths.Money, falling back to float64.
	EvaluateBinary = generator_helpers.EvaluateBinary

	// ValueToString provides a universal, reflection-based way to convert any
	// value to a string. The generator uses more optimised methods when possible,
	// but this is the final fallback.
	ValueToString = generator_helpers.ValueToString

	// F creates a locale-free FormatBuilder for formatting values
	// in templates with optional method chaining
	// (e.g. F(state.Price).Precision(2)).
	F = i18n_domain.F

	// ClassesFromString is an alias for generator_helpers.ClassesFromString.
	ClassesFromString = generator_helpers.ClassesFromString

	// ClassesFromSlice creates a class string from a dynamic slice of strings.
	ClassesFromSlice = generator_helpers.ClassesFromSlice

	// ClassesFromMapStringBool creates a CSS class string from a map of class
	// names to boolean values. Classes with true values are included in the
	// output.
	ClassesFromMapStringBool = generator_helpers.ClassesFromMapStringBool

	// MergeClasses merges multiple sources (static strings, dynamic values) into a
	// single class string.
	MergeClasses = generator_helpers.MergeClasses

	// ClassesFromStringBytes is a zero-allocation variant of ClassesFromString.
	// Returns a pooled buffer that must be passed to
	// DirectWriter.AppendPooledBytes().
	ClassesFromStringBytes = generator_helpers.ClassesFromStringBytes

	// ClassesFromSliceBytes is a zero-allocation variant of ClassesFromSlice.
	// Returns a pooled buffer that must be passed to
	// DirectWriter.AppendPooledBytes().
	ClassesFromSliceBytes = generator_helpers.ClassesFromSliceBytes

	// MergeClassesBytes is a zero-allocation variant of MergeClasses.
	// Returns a pooled buffer that must be passed to
	// DirectWriter.AppendPooledBytes().
	MergeClassesBytes = generator_helpers.MergeClassesBytes

	// BuildClassBytesV builds a class string from variadic string parts without
	// intermediate string concatenation allocations. Used by generated code for
	// template literals like `badge ${props.Size} ${props.Colour}`.
	//
	// In large functions, use fixed-arity variants to avoid heap escape.
	//
	// Returns a pooled buffer that must be passed to
	// DirectWriter.AppendPooledBytes().
	BuildClassBytesV = generator_helpers.BuildClassBytesV

	// BuildClassBytes2 builds a class string from exactly 2 parts without
	// allocation.
	BuildClassBytes2 = generator_helpers.BuildClassBytes2

	// BuildClassBytes4 builds a class string from exactly 4 parts without
	// allocation.
	BuildClassBytes4 = generator_helpers.BuildClassBytes4

	// BuildClassBytes6 builds a class string from exactly 6 parts without
	// allocation.
	BuildClassBytes6 = generator_helpers.BuildClassBytes6

	// BuildClassBytes8 builds a class string from exactly 8 parts without
	// allocation.
	BuildClassBytes8 = generator_helpers.BuildClassBytes8

	// StylesFromString is a variable that holds a function to create styles from
	// a string source.
	StylesFromString = generator_helpers.StylesFromString

	// StylesFromStringMap builds a style string from a map of string keys and
	// values.
	StylesFromStringMap = generator_helpers.StylesFromStringMap

	// MergeStyles merges multiple sources into a single, semicolon-delimited style
	// string.
	MergeStyles = generator_helpers.MergeStyles

	// StylesFromStringBytes is a zero-allocation variant of StylesFromString.
	// Returns a pooled buffer that must be passed to
	// DirectWriter.AppendEscapePooledBytes().
	StylesFromStringBytes = generator_helpers.StylesFromStringBytes

	// StylesFromStringMapBytes is a zero-allocation variant of
	// StylesFromStringMap.
	// Returns a pooled buffer that must be passed to
	// DirectWriter.AppendEscapePooledBytes().
	StylesFromStringMapBytes = generator_helpers.StylesFromStringMapBytes

	// MergeStylesBytes is a zero-allocation variant of MergeStyles. Returns a
	// pooled buffer that must be passed to DirectWriter.AppendEscapePooledBytes.
	MergeStylesBytes = generator_helpers.MergeStylesBytes

	// AppendHiddenToStyleBytes appends "display:none !important;" to existing
	// style bytes. Used for the p-show directive when the condition is false.
	AppendHiddenToStyleBytes = generator_helpers.AppendHiddenToStyleBytes

	// BuildStyleStringBytes2 is a fixed-arity style string builder to avoid string
	// concatenation allocation. These are used by generated code when building
	// styles from template literals.
	BuildStyleStringBytes2 = generator_helpers.BuildStyleStringBytes2

	// BuildStyleStringBytes3 is an alias for
	// generator_helpers.BuildStyleStringBytes3.
	BuildStyleStringBytes3 = generator_helpers.BuildStyleStringBytes3

	// BuildStyleStringBytes4 is a 4-byte variant of BuildStyleStringBytes.
	BuildStyleStringBytes4 = generator_helpers.BuildStyleStringBytes4

	// BuildStyleStringBytesV is an alias for
	// generator_helpers.BuildStyleStringBytesV.
	BuildStyleStringBytesV = generator_helpers.BuildStyleStringBytesV

	// ResolveModulePath resolves module alias (@/) paths at runtime.
	// Used by generated code when dynamic src attributes contain @ alias paths.
	ResolveModulePath = generator_helpers.ResolveModulePath

	// EncodeActionPayloadBytes encodes an ActionPayload to a base64 URL-safe
	// byte slice.
	//
	// It returns a pooled buffer that must be passed to
	// DirectWriter.AppendPooledBytes(). The buffer is automatically released
	// when the DirectWriter is reset.
	EncodeActionPayloadBytes = generator_helpers.EncodeActionPayloadBytes

	// EncodeActionPayloadBytes0 is a fixed-arity encoder variant to avoid slice
	// allocation for Args. These are used by generated code when the number of
	// arguments is known at compile time.
	EncodeActionPayloadBytes0 = generator_helpers.EncodeActionPayloadBytes0

	// EncodeActionPayloadBytes1 is a helper function for encoding action payloads
	// into bytes.
	EncodeActionPayloadBytes1 = generator_helpers.EncodeActionPayloadBytes1

	// EncodeActionPayloadBytes2 is an alias for the generator_helpers encoding
	// function.
	EncodeActionPayloadBytes2 = generator_helpers.EncodeActionPayloadBytes2

	// EncodeActionPayloadBytes3 is an alias for the generator helper function that
	// encodes action payloads to bytes.
	EncodeActionPayloadBytes3 = generator_helpers.EncodeActionPayloadBytes3

	// EncodeActionPayloadBytes4 encodes an action payload into a 4-byte slice.
	EncodeActionPayloadBytes4 = generator_helpers.EncodeActionPayloadBytes4

	// ClassesFromStringBytesArena is an arena-aware variant of
	// ClassesFromStringBytes.
	ClassesFromStringBytesArena = generator_helpers.ClassesFromStringBytesArena

	// ClassesFromSliceBytesArena is an arena-aware variant of
	// ClassesFromSliceBytes.
	ClassesFromSliceBytesArena = generator_helpers.ClassesFromSliceBytesArena

	// MergeClassesBytesArena is an arena-aware variant of MergeClassesBytes.
	MergeClassesBytesArena = generator_helpers.MergeClassesBytesArena

	// BuildClassBytes2Arena builds a class string from 2 parts using arena
	// allocation.
	BuildClassBytes2Arena = generator_helpers.BuildClassBytes2Arena

	// BuildClassBytes4Arena builds a class string from 4 parts using arena
	// allocation.
	BuildClassBytes4Arena = generator_helpers.BuildClassBytes4Arena

	// BuildClassBytes6Arena builds a class string from 6 parts using arena
	// allocation.
	BuildClassBytes6Arena = generator_helpers.BuildClassBytes6Arena

	// BuildClassBytes8Arena builds a class string from 8 parts using arena
	// allocation.
	BuildClassBytes8Arena = generator_helpers.BuildClassBytes8Arena

	// BuildClassBytesVArena builds a class string from variadic parts using arena
	// allocation.
	BuildClassBytesVArena = generator_helpers.BuildClassBytesVArena

	// StylesFromStringBytesArena is an arena-aware variant of
	// StylesFromStringBytes.
	StylesFromStringBytesArena = generator_helpers.StylesFromStringBytesArena

	// StylesFromStringMapBytesArena is an arena-aware variant of
	// StylesFromStringMapBytes.
	StylesFromStringMapBytesArena = generator_helpers.StylesFromStringMapBytesArena

	// MergeStylesBytesArena is an arena-aware variant of MergeStylesBytes.
	MergeStylesBytesArena = generator_helpers.MergeStylesBytesArena

	// BuildStyleStringBytes2Arena builds a style string from 2 parts using arena
	// allocation.
	BuildStyleStringBytes2Arena = generator_helpers.BuildStyleStringBytes2Arena

	// BuildStyleStringBytes3Arena builds a style string from three parts using
	// arena allocation.
	BuildStyleStringBytes3Arena = generator_helpers.BuildStyleStringBytes3Arena

	// BuildStyleStringBytes4Arena builds a style string from 4 parts using arena
	// allocation.
	BuildStyleStringBytes4Arena = generator_helpers.BuildStyleStringBytes4Arena

	// BuildStyleStringBytesVArena builds a style string from variadic parts using
	// arena allocation.
	BuildStyleStringBytesVArena = generator_helpers.BuildStyleStringBytesVArena

	// EncodeActionPayloadBytesArena encodes an ActionPayload using arena
	// allocation.
	EncodeActionPayloadBytesArena = generator_helpers.EncodeActionPayloadBytesArena

	// EncodeActionPayloadBytes0Arena encodes a 0-argument action using arena
	// allocation.
	EncodeActionPayloadBytes0Arena = generator_helpers.EncodeActionPayloadBytes0Arena

	// EncodeActionPayloadBytes1Arena encodes a 1-argument action using arena
	// allocation.
	EncodeActionPayloadBytes1Arena = generator_helpers.EncodeActionPayloadBytes1Arena

	// EncodeActionPayloadBytes2Arena encodes a 2-argument action using arena
	// allocation.
	EncodeActionPayloadBytes2Arena = generator_helpers.EncodeActionPayloadBytes2Arena

	// EncodeActionPayloadBytes3Arena encodes a 3-argument action using arena
	// allocation.
	EncodeActionPayloadBytes3Arena = generator_helpers.EncodeActionPayloadBytes3Arena

	// EncodeActionPayloadBytes4Arena encodes a 4-argument action using arena
	// allocation.
	EncodeActionPayloadBytes4Arena = generator_helpers.EncodeActionPayloadBytes4Arena

	// GetByteBuf retrieves a pooled byte buffer for encoding
	// operations, where the caller must release it via PutByteBuf
	// or track it via DirectWriter.AppendPooledBytes().
	GetByteBuf = ast_domain.GetByteBuf

	// PutByteBuf returns a byte buffer to the pool so it can be used again.
	PutByteBuf = ast_domain.PutByteBuf

	// AppendDiagnostic is the helper function used to report warnings or errors
	// during the running of the generated BuildAST function.
	AppendDiagnostic = func(
		diagnostics []*generator_dto.RuntimeDiagnostic,
		severity generator_dto.Severity,
		message, code, sourcePath, expression string,
		line, column int,
	) []*generator_dto.RuntimeDiagnostic {
		return generator_helpers.AppendDiagnostic(diagnostics, &generator_dto.RuntimeDiagnostic{
			Severity:   severity,
			Message:    message,
			Code:       code,
			SourcePath: sourcePath,
			Expression: expression,
			Line:       line,
			Column:     column,
		})
	}

	// ValidatePikoElementTagName checks that a dynamically resolved tag name
	// for <piko:element :is="..."> is valid. Returns "div" as a safe fallback
	// and appends a runtime diagnostic when the tag is empty or rejected.
	ValidatePikoElementTagName = generator_helpers.ValidatePikoElementTagName

	// GetContentAST extracts the contentAST from CollectionData for <piko:content
	// /> rendering.
	// Returns the TemplateAST containing the parsed markdown body, or nil if not
	// available.
	GetContentAST = generator_helpers.GetContentAST

	// CoerceToString converts any value to its string form.
	// It returns an empty string for nil values.
	CoerceToString = generator_helpers.CoerceToString

	// CoerceToInt converts any value to an int.
	// It returns 0 for values that cannot be converted.
	CoerceToInt = generator_helpers.CoerceToInt

	// CoerceToInt64 converts any value to an int64.
	// It returns 0 for values that cannot be converted.
	CoerceToInt64 = generator_helpers.CoerceToInt64

	// CoerceToInt32 converts any value to int32.
	// Returns 0 for invalid conversions.
	CoerceToInt32 = generator_helpers.CoerceToInt32

	// CoerceToInt16 converts any value to int16. It returns 0 for values that
	// cannot be converted.
	CoerceToInt16 = generator_helpers.CoerceToInt16

	// CoerceToFloat64 converts any value to float64.
	// It returns 0.0 when the value cannot be converted.
	CoerceToFloat64 = generator_helpers.CoerceToFloat64

	// CoerceToFloat32 converts any value to float32.
	// Returns 0.0 for invalid conversions.
	CoerceToFloat32 = generator_helpers.CoerceToFloat32

	// CoerceToBool converts any value to a bool.
	// It uses truthiness rules similar to JavaScript.
	CoerceToBool = generator_helpers.CoerceToBool

	// CoerceToDecimal converts any value to maths.Decimal.
	// Returns zero Decimal for invalid conversions.
	CoerceToDecimal = generator_helpers.CoerceToDecimal

	// CoerceToBigInt converts any value to a maths.BigInt.
	// It returns a zero BigInt for values that cannot be converted.
	CoerceToBigInt = generator_helpers.CoerceToBigInt
)

var (
	// DecodeAST converts FlatBuffers bytes back to a TemplateAST.
	// Used by collection pages to decode embedded content ASTs.
	DecodeAST = ast_adapters.DecodeAST

	// RegisterASTFunc registers a component's compiled BuildAST function with the
	// runtime.
	RegisterASTFunc = templater_domain.RegisterASTFunc

	// RegisterCachePolicyFunc registers a CachePolicy function for a component.
	RegisterCachePolicyFunc = templater_domain.RegisterCachePolicyFunc

	// RegisterMiddlewareFunc registers a component's Middlewares function.
	RegisterMiddlewareFunc = templater_domain.RegisterMiddlewareFunc

	// RegisterSupportedLocalesFunc registers a function that returns the locales
	// a component supports.
	RegisterSupportedLocalesFunc = templater_domain.RegisterSupportedLocalesFunc

	// RegisterPreviewFunc registers a component's Preview function for
	// dev-mode previewing with sample data.
	RegisterPreviewFunc = templater_domain.RegisterPreviewFunc

	// RegisterStaticCollectionBlob registers a binary blob for a static
	// collection. Generated code invokes RegisterStaticCollectionBlob in init()
	// using //go:embed directives.
	//
	// The blob is a FlatBuffer binary that holds all collection items. It is built
	// for zero-copy access and O(log n) lookups.
	//
	// Example generated code:
	//
	//	//go:embed data.bin
	//	var collectionBlob []byte
	//
	//	func init() {
	//	    pikoruntime.RegisterStaticCollectionBlob("docs", collectionBlob)
	//	}
	RegisterStaticCollectionBlob = collection_domain.RegisterStaticCollectionBlob
)

// I18nConfig contains the internationalisation configuration for generating
// SEO metadata. This public-facing type mirrors the internal config structure
// and is safe for use in generated code.
type I18nConfig struct {
	// DefaultLocale is the fallback locale used when no locale is specified or
	// matched. Example: "en".
	DefaultLocale string

	// Strategy defines how locale information is represented in URLs.
	// Supported values are "query-only", "prefix", and "prefix_except_default".
	Strategy string

	// Locales is the list of all supported locales.
	// Example: []string{"en", "fr", "de"}.
	Locales []string
}

// HybridConfig holds the settings for hybrid mode (ISR) collections.
type HybridConfig = collection_dto.HybridConfig

// These type aliases make collection types available to generated code and
// user initialisation code without exposing internal package structure.

type (
	// FetchOptions contains options for runtime collection fetching.
	FetchOptions = collection_dto.FetchOptions

	// Provider is the interface that dynamic providers must implement.
	Provider = collection_domain.RuntimeProvider

	// Section represents a heading in markdown content, used for building a
	// Table of Contents. It contains the heading title, slug (HTML ID), and
	// level (2-6).
	Section = markdown_dto.SectionData

	// Filter represents a single filtering condition for collection queries.
	// Used with WithFilter() to filter collection items by field values.
	Filter = collection_dto.Filter

	// FilterGroup represents a group of filters combined with AND/OR logic.
	// Used to build complex query conditions.
	FilterGroup = collection_dto.FilterGroup

	// FilterOperator defines the comparison operation for a filter (eq, contains,
	// etc).
	FilterOperator = collection_dto.FilterOperator

	// SortOrder defines the sort direction (ascending or descending).
	SortOrder = collection_dto.SortOrder

	// SortOption specifies a field and direction for sorting collection results.
	SortOption = collection_dto.SortOption

	// PaginationOptions specifies offset and limit for paginating collection
	// results.
	PaginationOptions = collection_dto.PaginationOptions
)

// These types are used for fuzzy text search on collections.

// SearchResult represents a search result with relevance scoring.
type SearchResult[T any] struct {
	// FieldScores holds the score from each field that was searched.
	FieldScores map[string]float64

	// Item is the content that matched the search query.
	Item T

	// Score is the relevance score ranging from 0.0 to 1.0, where 1.0 indicates a
	// perfect match and 0.0 indicates no match.
	Score float64
}

// SearchField specifies a field to search with optional weight.
type SearchField struct {
	// Name is the field name to search.
	Name string

	// Weight is the importance multiplier for the field; default is 1.0.
	// Higher values make matches in the field count for more.
	Weight float64
}

// SearchOption configures search behaviour using the functional options
// pattern.
type SearchOption func(*searchConfig)

// searchConfig holds settings for searching a collection.
type searchConfig struct {
	// searchMode specifies the search strategy; "fast" or "smart".
	searchMode string

	// fields specifies which fields to search and their weights.
	fields []SearchField

	// fuzzyThreshold is the minimum similarity score for fuzzy matching; 0 means
	// exact matches only, 1.0 means match anything.
	fuzzyThreshold float64

	// minScore is the minimum relevance score for search results.
	minScore float64

	// limit is the maximum number of results to return.
	limit int

	// offset is the number of results to skip for pagination.
	offset int

	// caseSensitive indicates whether the search should match case exactly.
	caseSensitive bool
}

// SectionNode represents a hierarchical section entry for table of contents.
// Unlike Section (which is flat), SectionNode contains nested children
// for building tree-structured navigation.
//
// This is a provider-agnostic type from the collection layer, allowing
// any content provider (markdown, headless CMS, etc.) to build ToC structures.
type SectionNode = collection_dto.SectionNode

// SectionTreeOption is a functional option for setting up GetSectionsTree.
type SectionTreeOption = collection_domain.SectionTreeOption

// These functions create FetchOptions for collection queries.
// They follow the functional options pattern for flexible, composable queries.

// CollectionOption is a function that changes how items are fetched.
type CollectionOption func(*FetchOptions)

// AdvancedSearchResult represents a search result from the inverted index.
type AdvancedSearchResult[T any] struct {
	// FieldScores shows the BM25 score contribution by field.
	FieldScores map[string]float64

	// Item is the matched content item.
	Item T

	// Score is the BM25 relevance score ranging from 0.0 to infinity.
	// Higher scores indicate better matches.
	Score float64

	// DocumentID is the internal document identifier.
	DocumentID uint32
}

// AdvancedSearchOption is a functional option that sets up advanced search.
type AdvancedSearchOption func(*advancedSearchConfig)

// advancedSearchConfig holds settings for advanced search queries.
type advancedSearchConfig struct {
	// mode specifies which search index to use.
	mode SearchMode

	// fields specifies which document fields to search; empty searches all fields.
	fields []search_dto.SearchField

	// limit is the maximum number of results to return.
	limit int

	// offset specifies the number of results to skip; 0 starts from the first
	// result.
	offset int

	// minScore is the minimum similarity score for results; 0 means no threshold.
	minScore float64

	// caseSensitive enables case-sensitive search matching.
	caseSensitive bool
}

// These types and functions enable hierarchical navigation generation from
// collections.

type (
	// NavigationGroups contains multiple named navigation structures, where each
	// group represents a distinct navigation UI component such as a sidebar or
	// footer.
	NavigationGroups = collection_dto.NavigationGroups

	// NavigationTree represents a hierarchical navigation structure for a specific
	// group and locale.
	NavigationTree = collection_dto.NavigationTree

	// NavigationNode represents a single node in the navigation hierarchy.
	NavigationNode = collection_dto.NavigationNode

	// NavigationConfig controls navigation tree building behaviour.
	NavigationConfig = collection_dto.NavigationConfig
)

// collectionNotFoundError is returned when a collection item lookup fails.
// It implements the ActionError interface (StatusCode() + ErrorCode()) so the
// rendering pipeline's extractErrorStatusCode returns HTTP 404, routing the
// error through the error page system.
type collectionNotFoundError struct {
	// cause is the underlying collection lookup error.
	cause error

	// collection is the name of the collection that was queried.
	collection string

	// route is the URL path that could not be found.
	route string
}

// Error implements the error interface.
//
// Returns string describing the missing collection item.
func (e *collectionNotFoundError) Error() string {
	return fmt.Sprintf("collection item not found: collection %q, route %q", e.collection, e.route)
}

// StatusCode returns HTTP 404 Not Found.
//
// Returns int which is always 404.
func (*collectionNotFoundError) StatusCode() int { return httpStatusNotFound }

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "COLLECTION_NOT_FOUND".
func (*collectionNotFoundError) ErrorCode() string { return "COLLECTION_NOT_FOUND" }

// Unwrap returns the underlying error.
//
// Returns error which is the original collection lookup error.
func (e *collectionNotFoundError) Unwrap() error { return e.cause }

// GetData retrieves the page data from CollectionData and converts it to type
// T. This provides type-safe access to collection data in Render functions,
// using a JSON round-trip for reliable map to struct conversion.
//
// Takes r (*templater_dto.RequestData) which contains the CollectionData to
// extract.
//
// Returns T which is the page data converted to the specified type, or the zero
// value if conversion fails.
//
//piko:link GetDataLink
func GetData[T any](r *templater_dto.RequestData) T {
	return generator_helpers.GetData[T](r)
}

// GetDataLink is the //piko:link sibling for GetData. The interpreter
// dispatches to GetDataLink when a .pk file calls GetData[T] with a
// user-defined T that has no compiled instantiation in the binary.
//
// Takes tType (reflect.Type) which is the instantiated type argument
// the user wrote in the brackets.
// Takes r (*templater_dto.RequestData) which contains the
// CollectionData to extract.
//
// Returns a reflect.Value of concrete type tType, either populated from
// the collection's "page" map or a zero value when conversion fails.
func GetDataLink(tType reflect.Type, r *templater_dto.RequestData) reflect.Value {
	value, _ := generator_helpers.GetDataReflect(r, tType)
	return value
}

// GenerateLocaleHead generates internationalisation SEO metadata for a page. It
// returns the current locale, canonical URL, and alternate hreflang links for
// all supported locales.
//
// Designed to be called from within a component's Render function to populate
// the Metadata.Language, Metadata.CanonicalUrl, and Metadata.AlternateLinks
// fields.
//
// Takes r (*templater_dto.RequestData) which provides the current request data.
// Takes i18nConfig (I18nConfig) which defines locales and URL strategy.
// Takes pagePath (string) which specifies the page's URL path.
// Takes supportedLocalesOverride ([]string) which optionally limits the locales
// used instead of the full config. Pass nil or empty slice to use all locales.
//
// Returns locale (string) which is the current request's locale from r.Locale.
// Returns canonicalURL (string) which is the canonical URL using default
// locale.
// Returns alternateLinks ([]map[string]string) which contains hreflang links
// for SEO.
func GenerateLocaleHead(
	r *templater_dto.RequestData,
	i18nConfig I18nConfig,
	pagePath string,
	supportedLocalesOverride []string,
) (locale string, canonicalURL string, alternateLinks []map[string]string) {
	internalConfig := &config.WebsiteConfig{
		I18n: config.I18nConfig{
			DefaultLocale: i18nConfig.DefaultLocale,
			Strategy:      i18nConfig.Strategy,
			Locales:       i18nConfig.Locales,
		},
	}

	return generator_helpers.GenerateLocaleHead(r, internalConfig, pagePath, supportedLocalesOverride)
}

// These functions support Incremental Static Regeneration for collections.

// GetHybridBlob retrieves the current FlatBuffer blob from the hybrid
// registry.
//
// Called by generated code to access hybrid collection data.
// Returns the current blob and whether background revalidation should be triggered.
//
// Takes ctx (context.Context) which controls cancellation and tracing.
// Takes providerName (string) which identifies the provider that owns this
// collection.
// Takes collectionName (string) which specifies the collection identifier.
//
// Returns []byte which contains the current FlatBuffer blob, or nil if not
// registered.
// Returns bool which indicates whether TTL has expired and revalidation
// should run.
func GetHybridBlob(ctx context.Context, providerName, collectionName string) ([]byte, bool) {
	return collection_domain.GetHybridBlob(ctx, providerName, collectionName)
}

// TriggerHybridRevalidation triggers background revalidation for a hybrid
// collection.
//
// Validates the ETag and updates the cache if the content has changed.
// Returns immediately.
//
// Takes providerName (string) which identifies the provider.
// Takes collectionName (string) which identifies the collection.
func TriggerHybridRevalidation(ctx context.Context, providerName, collectionName string) {
	collection_domain.TriggerHybridRevalidation(ctx, providerName, collectionName)
}

// DecodeCollectionBlob decodes a FlatBuffer collection blob into a typed
// slice. Each item's metadata JSON is unmarshalled into T.
//
// Called by generated hybrid collection getter functions to convert the cached
// FlatBuffer blob back into the user's typed slice.
//
// Takes blob ([]byte) which is the FlatBuffer-encoded collection data.
//
// Returns []T which contains the decoded items.
// Returns error when the blob cannot be unpacked or decoded.
func DecodeCollectionBlob[T any](blob []byte) ([]T, error) {
	return collection_domain.DecodeCollectionBlob[T](blob)
}

// RegisterHybridSnapshot registers a build-time snapshot for runtime use.
//
// Called from generated init() functions to register the embedded FlatBuffer
// blob and its ETag for hybrid mode operation.
//
// Takes ctx (context.Context) which controls cancellation and tracing.
// Takes providerName (string) which identifies the provider that generated
// this snapshot.
// Takes collectionName (string) which identifies the collection this snapshot
// belongs to.
// Takes blob ([]byte) which contains the FlatBuffer-serialised content.
// Takes etag (string) which is the content fingerprint at build time.
// Takes hybridConfig (HybridConfig) which specifies the hybrid mode configuration.
func RegisterHybridSnapshot(
	ctx context.Context,
	providerName, collectionName string,
	blob []byte,
	etag string,
	hybridConfig HybridConfig,
) {
	collection_domain.RegisterHybridSnapshot(ctx, providerName, collectionName, blob, etag, hybridConfig)
}

// HasHybridCollection checks if a hybrid collection is registered.
//
// This can be used to check if hybrid mode is active for a collection.
//
// Takes providerName (string) which identifies the provider.
// Takes collectionName (string) which identifies the collection.
//
// Returns bool which is true if the collection is registered in hybrid mode.
func HasHybridCollection(providerName, collectionName string) bool {
	return collection_domain.HasHybridCollection(providerName, collectionName)
}

// GetHybridETag returns the current ETag for a hybrid collection.
//
// This is primarily for debugging and monitoring purposes.
//
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection.
//
// Returns string which is the current ETag value for the hybrid collection.
func GetHybridETag(providerName, collectionName string) string {
	return collection_domain.GetHybridETag(providerName, collectionName)
}

// FetchCollection fetches dynamic collection data at runtime.
//
// Called by generated code when a component uses data.GetCollection() with a
// dynamic provider (e.g., headless CMS, database).
//
// Takes providerName (string) which specifies the provider to use.
// Takes collectionName (string) which specifies the collection to fetch.
// Takes options (*FetchOptions) which provides locale, filters, and cache
// config.
// Takes target (any) which is a pointer to a slice to populate.
//
// Returns error when the fetch fails or the provider is not found.
func FetchCollection(
	ctx context.Context,
	providerName string,
	collectionName string,
	options *FetchOptions,
	target any,
) error {
	return collection_domain.FetchCollection(ctx, providerName, collectionName, options, target)
}

// RegisterRuntimeProvider registers a provider for runtime data fetching.
//
// This should be called during application initialisation (in main.go) to
// make a dynamic provider available for runtime collection fetching.
//
// Takes provider (Provider) which is the runtime provider to register.
//
// Returns error when the provider name conflicts with an existing provider.
func RegisterRuntimeProvider(provider Provider) error {
	return collection_domain.RegisterProvider(provider)
}

// These functions provide a fluent API for building filter conditions.

// NewFilter creates a single filter condition.
//
// Takes field (string) which specifies the field name to filter on.
// Takes operator (FilterOperator) which defines the comparison operation.
// Takes value (any) which provides the value to compare against.
//
// Returns Filter which contains the configured filter condition.
//
// Example:
//
//	filter := pikoruntime.NewFilter("status", pikoruntime.FilterOpEquals, "published")
func NewFilter(field string, operator FilterOperator, value any) Filter {
	return collection_dto.Filter{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
}

// And combines multiple filters with AND logic (all must match).
//
// Takes filters (...Filter) which are the conditions that all must match.
//
// Returns FilterGroup which contains the filters combined with AND logic.
//
// Example:
//
//	filterGroup := pikoruntime.And(
//	    pikoruntime.NewFilter("status", pikoruntime.FilterOpEquals, "published"),
//	    pikoruntime.NewFilter("featured", pikoruntime.FilterOpEquals, true),
//	)
func And(filters ...Filter) FilterGroup {
	return collection_dto.FilterGroup{
		Filters: filters,
		Logic:   "AND",
	}
}

// Or combines multiple filters with OR logic where at least one must match.
//
// Takes filters (...Filter) which are the filter conditions to combine.
//
// Returns FilterGroup which contains the filters combined with OR logic.
//
// Example:
//
//	filterGroup := pikoruntime.Or(
//	    pikoruntime.NewFilter("category", pikoruntime.FilterOpEquals, "tech"),
//	    pikoruntime.NewFilter("category", pikoruntime.FilterOpEquals, "science"),
//	)
func Or(filters ...Filter) FilterGroup {
	return collection_dto.FilterGroup{
		Filters: filters,
		Logic:   "OR",
	}
}

// NewSortOption creates a sorting option for collection queries.
//
// Takes field (string) which specifies the field name to sort by.
// Takes order (SortOrder) which specifies the sort direction.
//
// Returns SortOption which contains the configured sorting parameters.
//
// Example:
//
//	sort := pikoruntime.NewSortOption("publishedAt", pikoruntime.SortDesc)
func NewSortOption(field string, order SortOrder) SortOption {
	return collection_dto.SortOption{
		Field: field,
		Order: order,
	}
}

// NewPaginationOptions creates pagination parameters.
//
// Takes limit (int) which specifies the maximum number of results to return.
// Takes offset (int) which specifies the number of results to skip.
//
// Returns PaginationOptions which contains the configured pagination settings.
//
// Example:
//
//	pagination := pikoruntime.NewPaginationOptions(10, 20)  // Limit 10, Offset 20
func NewPaginationOptions(limit, offset int) PaginationOptions {
	return collection_dto.PaginationOptions{
		Limit:  limit,
		Offset: offset,
	}
}

// WithMinLevel sets the minimum heading level to include (default: 2).
// Headings below this level are filtered out.
//
// Takes level (int) which specifies the minimum heading level.
//
// Returns SectionTreeOption which configures the section tree filtering.
//
// Example:
//
//	tree := piko.GetSectionsTree(r, piko.WithMinLevel(2)) // Start from h2
func WithMinLevel(level int) SectionTreeOption {
	return collection_domain.WithMinLevel(level)
}

// WithMaxLevel sets the maximum heading level to include (default: 4).
// Headings above this level are filtered out.
//
// Takes level (int) which specifies the maximum heading level to include.
//
// Returns SectionTreeOption which configures the section tree builder.
//
// Example:
//
//	tree := piko.GetSectionsTree(r, piko.WithMaxLevel(3)) // Only h2 and h3
func WithMaxLevel(level int) SectionTreeOption {
	return collection_domain.WithMaxLevel(level)
}

// GetSectionsTree extracts sections from collection data and builds a
// hierarchical tree. Unlike GetSections which returns a flat list, this
// returns nested SectionNode structures suitable for rendering a table of
// contents with proper nesting.
//
// Takes r (*templater_dto.RequestData) which contains the collection data.
// Takes opts (...SectionTreeOption) which provides optional configuration
// such as WithMinLevel and WithMaxLevel.
//
// Returns []SectionNode which contains top-level nodes with nested children.
func GetSectionsTree(r *templater_dto.RequestData, opts ...SectionTreeOption) []SectionNode {
	flatSections := GetSections(r)
	return collection_domain.BuildSectionTree(flatSections, opts...)
}

// GetSections extracts the table of contents (sections/headings) from
// collection data. Returns a list of headings found in markdown content,
// useful for building a ToC sidebar.
//
// Takes r (*templater_dto.RequestData) which contains the CollectionData to
// extract sections from.
//
// Returns []Section which contains heading titles, slugs, and levels. Returns
// nil when the collection data is missing, malformed, or has no sections.
func GetSections(r *templater_dto.RequestData) []Section {
	if r.CollectionData() == nil {
		return nil
	}

	rootMap, ok := r.CollectionData().(map[string]any)
	if !ok {
		return nil
	}

	pageData, exists := rootMap["page"]
	if !exists {
		return nil
	}

	pageMap, ok := pageData.(map[string]any)
	if !ok {
		return nil
	}

	sectionsRaw, exists := pageMap["Sections"]
	if !exists {
		return nil
	}

	if sections, ok := sectionsRaw.([]markdown_dto.SectionData); ok {
		return sections
	}

	if sectionsSlice, ok := sectionsRaw.([]any); ok {
		sections := make([]Section, 0, len(sectionsSlice))
		for _, item := range sectionsSlice {
			if sectionMap, ok := item.(map[string]any); ok {
				section := Section{
					Title: getString(sectionMap, "Title"),
					Slug:  getString(sectionMap, "Slug"),
					Level: getInt(sectionMap, "Level"),
				}
				sections = append(sections, section)
			}
		}
		return sections
	}

	return nil
}

// GetStaticCollectionItem retrieves a single item from a static collection by
// route.
//
// Performs an O(log n) binary search lookup in the embedded FlatBuffer blob
// and returns the metadata and ASTs for the requested route.
//
// Used by generated BuildAST code to populate r.CollectionData for collection
// pages.
//
// Takes ctx (context.Context) which controls cancellation and tracing.
// Takes collectionName (string) which is the name of the collection (e.g.,
// "docs", "blog").
// Takes route (string) which is the URL route to look up (e.g.,
// "/docs/actions").
//
// Returns metadata (map[string]any) which is the page metadata.
// Returns contentAST (*TemplateAST) which is the parsed content.
// Returns excerptAST (*TemplateAST) which is the parsed excerpt, or nil.
// Returns err (error) when the collection or route is not found.
//
// Example usage in generated code:
//
//	metadata, contentAST, excerptAST, err := pikoruntime.GetStaticCollectionItem(r.Context(), "docs", r.URL.Path)
//	if err == nil {
//	    r.CollectionData = map[string]interface{}{
//	        "page":       metadata,
//	        "contentAST": contentAST,
//	        "excerptAST": excerptAST,
//	    }
//	}
func GetStaticCollectionItem(ctx context.Context, collectionName, route string) (metadata map[string]any, contentAST *TemplateAST, excerptAST *TemplateAST, err error) {
	return collection_domain.GetStaticCollectionItem(ctx, collectionName, route)
}

// GetAllCollectionItems retrieves all items from a static collection.
//
// Retrieves metadata only, without ASTs. Use to build navigation, sitemaps, or
// RSS feeds where you need to iterate over all items in a collection but do
// not need the full content ASTs.
//
// Takes collectionName (string) which specifies the collection to retrieve
// (e.g., "docs", "blog").
//
// Returns []map[string]any which contains metadata maps, one per collection
// item.
// Returns error when the collection is not found.
func GetAllCollectionItems(collectionName string) ([]map[string]any, error) {
	return collection_domain.GetStaticCollectionItems(collectionName)
}

// GetCollectionNavigation returns a lazily initialised navigation tree for a
// static collection. The first call for a given collection and config pair
// builds the tree; all subsequent calls return the same instance.
//
// This combines GetAllCollectionItems and BuildNavigationFromMetadata into a
// single call that avoids per-request tree construction.
//
// Takes ctx (context.Context) which carries request-scoped values.
// Takes collectionName (string) which identifies the static collection.
// Takes navigationConfig (NavigationConfig) which controls how
// navigation is built.
//
// Returns *NavigationGroups which contains the navigation trees.
// Returns error when the collection is not found.
func GetCollectionNavigation(ctx context.Context, collectionName string, navigationConfig NavigationConfig) (*NavigationGroups, error) {
	return collection_domain.GetStaticCollectionNavigation(ctx, collectionName, navigationConfig)
}

// WithFilter applies a filter group to a collection query.
//
// Takes fg (FilterGroup) which specifies the filter conditions to apply.
//
// Returns CollectionOption which configures the query with the given filters.
//
// Example:
//
//	posts, err := FetchCollection(ctx, "headless-cms", "blog",
//	    WithFilter(And(
//	        NewFilter("status", FilterOpEquals, "published"),
//	        NewFilter("featured", FilterOpEquals, true),
//	    )),
//	    &posts)
func WithFilter(fg FilterGroup) CollectionOption {
	return func(opts *FetchOptions) {
		opts.FilterGroup = &fg
	}
}

// WithSort applies sorting to a collection query.
//
// Multiple sort options are applied in order.
//
// Takes sorts (...SortOption) which specifies the sort criteria to apply.
//
// Returns CollectionOption which configures the sort settings on fetch options.
//
// Example:
//
//	posts, err := FetchCollection(ctx, "headless-cms", "blog",
//	    WithSort(
//	        NewSortOption("featured", SortDesc),
//	        NewSortOption("publishedAt", SortDesc),
//	    ),
//	    &posts)
func WithSort(sorts ...SortOption) CollectionOption {
	return func(opts *FetchOptions) {
		opts.Sort = sorts
	}
}

// WithPagination applies pagination to a collection query.
//
// Takes pagination (PaginationOptions) which specifies the limit and offset
// for the query results.
//
// Returns CollectionOption which configures pagination on fetch options.
//
// Example:
//
//	posts, err := FetchCollection(ctx, "headless-cms", "blog",
//	    WithPagination(NewPaginationOptions(10, 20)),  // Limit 10, Offset 20
//	    &posts)
func WithPagination(pagination PaginationOptions) CollectionOption {
	return func(opts *FetchOptions) {
		opts.Pagination = &pagination
	}
}

// WithLimit sets only the pagination limit without an offset.
//
// Takes limit (int) which specifies the maximum number of items to return.
//
// Returns CollectionOption which configures the fetch pagination settings.
//
// Example:
//
//	posts, err := FetchCollection(ctx, "headless-cms", "blog",
//	    WithLimit(10),
//	    &posts)
func WithLimit(limit int) CollectionOption {
	return func(opts *FetchOptions) {
		opts.Pagination = &PaginationOptions{
			Limit: limit,
		}
	}
}

// ApplyCollectionOptions applies a list of options to FetchOptions.
//
// Takes opts (*FetchOptions) which receives the applied options.
// Takes options (...CollectionOption) which specifies the options to apply.
func ApplyCollectionOptions(opts *FetchOptions, options ...CollectionOption) {
	for _, opt := range options {
		opt(opts)
	}
}

// WithSearchFields specifies which fields to search with their weights.
//
// Takes fields (...SearchField) which defines the fields and their weights.
//
// Returns SearchOption which configures the search to use the given fields.
//
// Example:
//
//	results := SearchCollection[Post](r, "blog", "golang",
//	    WithSearchFields(
//	        SearchField{Name: "Title", Weight: 2.0},
//	        SearchField{Name: "Body", Weight: 1.0},
//	    ))
func WithSearchFields(fields ...SearchField) SearchOption {
	return func(searchConfig *searchConfig) {
		searchConfig.fields = fields
	}
}

// WithFuzzyThreshold sets the fuzzy matching threshold for search operations.
//
// A value of 0.0 means exact match only, whilst 1.0 means very fuzzy
// matching. The default value is 0.3.
//
// Takes threshold (float64) which specifies the matching tolerance level.
//
// Returns SearchOption which configures the search with the given threshold.
//
// Example:
//
//	results := SearchCollection[Post](r, "blog", "golang",
//	    WithFuzzyThreshold(0.5))  // More lenient matching
func WithFuzzyThreshold(threshold float64) SearchOption {
	return func(searchConfig *searchConfig) {
		searchConfig.fuzzyThreshold = threshold
	}
}

// WithSearchLimit limits the number of search results returned.
//
// Takes limit (int) which specifies the maximum number of results to return.
//
// Returns SearchOption which configures the search limit when applied.
//
// Example:
//
//	results := SearchCollection[Post](r, "blog", "golang",
//	    WithSearchLimit(10))  // Top 10 results
func WithSearchLimit(limit int) SearchOption {
	return func(searchConfig *searchConfig) {
		searchConfig.limit = limit
	}
}

// WithSearchOffset skips the first N results for pagination.
//
// Takes offset (int) which specifies the number of results to skip.
//
// Returns SearchOption which configures the search to skip the specified
// number of results.
//
// Example:
//
//	results := SearchCollection[Post](r, "blog", "golang",
//	    WithSearchLimit(10),
//	    WithSearchOffset(20))  // Results 21-30
func WithSearchOffset(offset int) SearchOption {
	return func(searchConfig *searchConfig) {
		searchConfig.offset = offset
	}
}

// WithMinScore filters out results below the specified score.
//
// Takes score (float64) which specifies the minimum score threshold.
//
// Returns SearchOption which configures the search to exclude results
// with scores below the threshold.
//
// Example:
//
//	results := SearchCollection[Post](r, "blog", "golang",
//	    WithMinScore(0.5))  // Only results with score >= 0.5
func WithMinScore(score float64) SearchOption {
	return func(searchConfig *searchConfig) {
		searchConfig.minScore = score
	}
}

// WithCaseSensitive enables or disables case-sensitive search matching.
// By default, search matching is case-insensitive.
//
// Takes sensitive (bool) which specifies whether matching is case-sensitive.
//
// Returns SearchOption which configures the search behaviour.
func WithCaseSensitive(sensitive bool) SearchOption {
	return func(searchConfig *searchConfig) {
		searchConfig.caseSensitive = sensitive
	}
}

// WithSearchMode sets the search mode to use.
// Valid values: "fast" (default) or "smart" (with stemming and phonetic
// matching).
//
// Takes mode (string) which specifies the search mode.
//
// Returns SearchOption which configures the search mode on the search config.
//
// Example:
//
//	results := SearchCollection[Post](r, "blog", "running",
//	    WithSearchMode("smart"))  // Matches "run", "runs", "running"
func WithSearchMode(mode string) SearchOption {
	return func(searchConfig *searchConfig) {
		searchConfig.searchMode = mode
	}
}

// RegisterSearchIndex registers a binary search index blob for runtime
// zero-copy access.
//
// This is called by generated code in init() functions (from //go:embed
// directives) to register the embedded search index binaries.
//
// Takes collectionName (string) which identifies the collection
// (e.g., "docs", "blog").
// Takes mode (string) which specifies the search mode ("fast" or "smart").
// Takes data ([]byte) which contains the FlatBuffer binary blob
// (embedded via //go:embed).
//
// Returns error when registration fails.
//
// Example generated code:
//
//	//go:embed search_fast.bin
//	var searchFastBlob []byte
//
//	//go:embed search_smart.bin
//	var searchSmartBlob []byte
//
//	func init() {
//	    pikoruntime.RegisterStaticCollectionBlob("docs", collectionBlob)
//	    pikoruntime.RegisterSearchIndex("docs", "fast", searchFastBlob)
//	    pikoruntime.RegisterSearchIndex("docs", "smart", searchSmartBlob)
//	}
func RegisterSearchIndex(collectionName, mode string, data []byte) error {
	return search_adapters.RegisterSearchIndex(collectionName, mode, data)
}

// WithMode sets the search mode (Fast or Smart).
//
// Takes mode (SearchMode) which specifies the search behaviour to use.
//
// Returns AdvancedSearchOption which configures the search mode.
//
// Example:
//
//	results := AdvancedSearch[Post](ctx, "blog", "golang",
//	    WithMode(SearchModeSmart))  // Use Smart mode with stemming
func WithMode(mode SearchMode) AdvancedSearchOption {
	return func(advancedSearchConfig *advancedSearchConfig) {
		advancedSearchConfig.mode = mode
	}
}

// WithAdvancedSearchFields specifies which fields to search with weights.
//
// Takes fields (...search_dto.SearchField) which defines the searchable fields
// and their relative importance.
//
// Returns AdvancedSearchOption which configures the search to use the specified
// fields.
//
// Example:
//
//	results := AdvancedSearch[Post](ctx, "blog", "golang",
//	    WithAdvancedSearchFields(
//	        search_dto.SearchField{Name: "title", Weight: 2.5},
//	        search_dto.SearchField{Name: "content", Weight: 1.0},
//	    ))
func WithAdvancedSearchFields(fields ...search_dto.SearchField) AdvancedSearchOption {
	return func(advancedSearchConfig *advancedSearchConfig) {
		advancedSearchConfig.fields = fields
	}
}

// WithAdvancedSearchLimit limits the number of results returned.
//
// Takes limit (int) which specifies the maximum number of results.
//
// Returns AdvancedSearchOption which configures the search limit.
func WithAdvancedSearchLimit(limit int) AdvancedSearchOption {
	return func(advancedSearchConfig *advancedSearchConfig) {
		advancedSearchConfig.limit = limit
	}
}

// WithAdvancedSearchOffset skips the first N results.
//
// Takes offset (int) which specifies the number of results to skip.
//
// Returns AdvancedSearchOption which configures the search offset.
func WithAdvancedSearchOffset(offset int) AdvancedSearchOption {
	return func(advancedSearchConfig *advancedSearchConfig) {
		advancedSearchConfig.offset = offset
	}
}

// WithAdvancedMinScore filters results below this BM25 score.
//
// Takes score (float64) which specifies the minimum BM25 score threshold.
//
// Returns AdvancedSearchOption which configures the minimum score filter.
func WithAdvancedMinScore(score float64) AdvancedSearchOption {
	return func(advancedSearchConfig *advancedSearchConfig) {
		advancedSearchConfig.minScore = score
	}
}

// AdvancedSearch performs full-text search using the inverted index and BM25
// scoring.
//
// Uses the zero-copy search index embedded in the binary for:
//   - O(log n) term lookups
//   - BM25 probabilistic ranking
//   - Optional stemming and phonetic matching (Smart mode)
//
// Takes ctx (context.Context) which controls cancellation.
// Takes collectionName (string) which identifies the collection to search
// (e.g., "docs").
// Takes query (string) which is the search query text.
// Takes opts (...AdvancedSearchOption) which provides optional configuration
// (mode, fields, limit, etc.).
//
// Returns []AdvancedSearchResult[T] which contains ranked search results
// with BM25 scores.
// Returns error when the index is not found or the search fails.
//
// Example:
//
//	results, err := pikoruntime.AdvancedSearch[Post](ctx, "blog", "golang tutorial",
//	    WithMode(SearchModeSmart),
//	    WithAdvancedSearchLimit(10),
//	    WithAdvancedMinScore(1.0),
//	)
func AdvancedSearch[T any](
	ctx context.Context,
	collectionName string,
	query string,
	opts ...AdvancedSearchOption,
) ([]AdvancedSearchResult[T], error) {
	advancedSearchConfig := applyAdvancedSearchOptions(opts...)

	reader, err := search_adapters.GetSearchIndex(collectionName, string(advancedSearchConfig.mode))
	if err != nil {
		return nil, fmt.Errorf("failed to get search index: %w", err)
	}

	scorer := search_domain.NewBM25Scorer(defaultBM25K1, defaultBM25B)
	processor := search_domain.NewQueryProcessorForIndex(reader)

	searchConfig := search_dto.SearchConfig{
		Query:         query,
		Fields:        advancedSearchConfig.fields,
		Limit:         advancedSearchConfig.limit,
		Offset:        advancedSearchConfig.offset,
		MinScore:      advancedSearchConfig.minScore,
		CaseSensitive: advancedSearchConfig.caseSensitive,
	}

	queryResults, err := processor.Search(ctx, query, reader, scorer, searchConfig)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return hydrateSearchResults[T](ctx, queryResults, reader, collectionName), nil
}

// GetSearchIndexMetadata returns metadata about a search index.
// Useful for debugging and understanding index characteristics.
//
// Takes collectionName (string) which identifies the collection.
// Takes mode (string) which specifies the search mode ("fast" or "smart").
//
// Returns map[string]any which contains index statistics such as total_docs
// and vocab_size.
// Returns error when the index is not found.
func GetSearchIndexMetadata(collectionName, mode string) (map[string]any, error) {
	return search_adapters.GetSearchIndexMetadata(collectionName, mode)
}

// ListSearchIndexes returns all available search indexes.
//
// Returns map[string][]string which maps collection names to their available
// modes.
//
// Example:
//
//	indexes := pikoruntime.ListSearchIndexes()
//	for collection, modes := range indexes {
//	    fmt.Printf("%s: %v\n", collection, modes)
//	}
func ListSearchIndexes() map[string][]string {
	return search_adapters.ListSearchIndexes()
}

// HasSearchIndex checks if a search index exists for a
// collection and mode.
//
// Example:
//
//	if pikoruntime.HasSearchIndex("docs", "smart") {
//	    // Use Smart mode search
//	} else {
//	    // Fall back to Fast mode or fuzzy search
//	}
//
// Takes collectionName (string) which identifies the
// collection.
// Takes mode (string) which specifies the search mode ("fast"
// or "smart").
//
// Returns bool which is true if the index exists.
func HasSearchIndex(collectionName, mode string) bool {
	return search_adapters.HasSearchIndex(collectionName, mode)
}

// DefaultNavigationConfig returns a NavigationConfig with sensible defaults.
//
// Returns NavigationConfig which contains sensible defaults for navigation.
func DefaultNavigationConfig() NavigationConfig {
	return collection_dto.DefaultNavigationConfig()
}

// BuildNavigationFromMetadata constructs hierarchical navigation from
// collection metadata maps.
//
// Takes metadata maps (from GetAllCollectionItems) and builds navigation trees
// based on the "Navigation" field in each item's metadata.
//
// Takes metadataItems ([]map[string]any) which provides metadata maps from a
// collection.
// Takes navigationConfig (NavigationConfig) which specifies
// navigation building options.
//
// Returns *NavigationGroups which contains all named navigation trees.
func BuildNavigationFromMetadata(ctx context.Context, metadataItems []map[string]any, navigationConfig NavigationConfig) *NavigationGroups {
	if groups, ok := collection_domain.TryGetCachedNavigation(ctx, metadataItems, navigationConfig); ok {
		return groups
	}

	contentItems := make([]collection_dto.ContentItem, 0, len(metadataItems))

	for _, metadata := range metadataItems {
		item := collection_dto.ContentItem{
			ID:       getString(metadata, "ID"),
			Slug:     getString(metadata, "Slug"),
			Locale:   getString(metadata, "Locale"),
			URL:      getString(metadata, "URL"),
			Metadata: metadata,
		}
		contentItems = append(contentItems, item)
	}

	builder := collection_domain.NewNavigationBuilder()
	return builder.BuildNavigationGroups(ctx, contentItems, navigationConfig)
}

// BuildNavigationFromContentItems builds navigation groups from a slice of
// content items. This is an internal helper used by generated code.
//
// Takes items ([]collection_dto.ContentItem) which contains the content to
// organise into navigation groups.
// Takes navigationConfig (NavigationConfig) which specifies how
// to group and order the navigation items.
//
// Returns *NavigationGroups which contains the organised navigation structure.
func BuildNavigationFromContentItems(ctx context.Context, items []collection_dto.ContentItem, navigationConfig NavigationConfig) *NavigationGroups {
	builder := collection_domain.NewNavigationBuilder()
	return builder.BuildNavigationGroups(ctx, items, navigationConfig)
}

// CollectionNotFound creates a 404 error for a missing collection item.
// This is used by generated code when a p-collection page's item lookup fails,
// routing the error through the error page system.
//
// Takes collection (string) which is the collection name (e.g., "blog").
// Takes route (string) which is the URL path that was looked up.
// Takes cause (error) which is the underlying lookup error.
//
// Returns error which implements ActionError with status code 404.
func CollectionNotFound(collection, route string, cause error) error {
	return &collectionNotFoundError{collection: collection, route: route, cause: cause}
}

// getString retrieves a string value from a map by key.
//
// Takes m (map[string]any) which is the map to search.
// Takes key (string) which is the key to look up.
//
// Returns string which is the value if found and is a string, or empty string
// otherwise.
func getString(m map[string]any, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

// getInt retrieves an integer value from a map by key.
//
// Takes m (map[string]any) which is the map to search.
// Takes key (string) which is the key to look up.
//
// Returns int which is the value if found, or zero if the key is missing or
// not a numeric type.
func getInt(m map[string]any, key string) int {
	switch value := m[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case int32:
		return int(value)
	case int16:
		return int(value)
	case int8:
		return int(value)
	case uint:
		return safeconv.Uint64ToInt(uint64(value))
	case uint64:
		return safeconv.Uint64ToInt(value)
	case uint32:
		return int(value)
	case uint16:
		return int(value)
	case uint8:
		return int(value)
	case float64:
		return int(value)
	case float32:
		return int(value)
	default:
		return 0
	}
}

// applySearchOptions applies search options to create a config.
//
// Takes opts (...SearchOption) which are functional options to customise the
// search configuration.
//
// Returns searchConfig which contains the merged settings with defaults for
// any options not specified.
func applySearchOptions(opts ...SearchOption) searchConfig {
	searchConfig := searchConfig{
		fuzzyThreshold: defaultFuzzyThreshold,
		limit:          0,
		offset:         0,
		minScore:       0.0,
		caseSensitive:  false,
		searchMode:     "fast",
	}

	for _, opt := range opts {
		opt(&searchConfig)
	}

	return searchConfig
}

// hydrateSearchResults converts raw search results to typed results with
// collection data.
//
// Takes queryResults ([]search_domain.QueryResult) which contains the raw
// search results to convert.
// Takes reader (search_domain.IndexReaderPort) which provides document
// metadata access.
// Takes collectionName (string) which identifies the collection for item
// lookup.
//
// Returns []AdvancedSearchResult[T] which contains the typed results with
// populated items.
func hydrateSearchResults[T any](
	ctx context.Context,
	queryResults []search_domain.QueryResult,
	reader search_domain.IndexReaderPort,
	collectionName string,
) []AdvancedSearchResult[T] {
	results := make([]AdvancedSearchResult[T], 0, len(queryResults))
	for _, queryResult := range queryResults {
		docMeta, err := reader.GetDocMetadata(queryResult.DocumentID)
		if err != nil {
			continue
		}
		metadata, _, _, err := collection_domain.GetStaticCollectionItem(ctx, collectionName, docMeta.Route)
		if err != nil {
			continue
		}
		var item T
		if err := collection_domain.ConvertSearchResultToType(metadata, &item); err != nil {
			continue
		}
		results = append(results, AdvancedSearchResult[T]{
			Item:        item,
			Score:       queryResult.Score,
			FieldScores: queryResult.FieldScores,
			DocumentID:  queryResult.DocumentID,
		})
	}
	return results
}

// applyAdvancedSearchOptions applies options to create config.
//
// Takes opts (...AdvancedSearchOption) which are functional options that
// modify the search configuration.
//
// Returns advancedSearchConfig which contains the configured search settings
// with sensible defaults applied.
func applyAdvancedSearchOptions(opts ...AdvancedSearchOption) advancedSearchConfig {
	advancedSearchConfig := advancedSearchConfig{
		mode: SearchModeFast,
		fields: []search_dto.SearchField{
			{Name: "title", Weight: defaultTitleWeight},
			{Name: "content", Weight: defaultContentWeight},
		},
		limit:         0,
		offset:        0,
		minScore:      0.0,
		caseSensitive: false,
	}

	for _, opt := range opts {
		opt(&advancedSearchConfig)
	}

	return advancedSearchConfig
}
