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
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// buildSliceLiteral generates a Go AST slice literal from content items.
//
// This creates an ast.CompositeLit representing a slice of the target type,
// with each element populated from the ContentItem metadata.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes targetType (ast.Expr) which specifies the element type for the slice.
// Takes items ([]collection_dto.ContentItem) which provides the data for each
// slice element.
//
// Returns ast.Expr which is the constructed slice literal as a composite
// literal node.
// Returns error when building an individual item literal fails.
func (s *collectionService) buildSliceLiteral(
	ctx context.Context,
	targetType ast.Expr,
	items []collection_dto.ContentItem,
) (ast.Expr, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Building slice literal",
		logger_domain.Int(logKeyItemCount, len(items)))

	elements := make([]ast.Expr, len(items))

	for i := range items {
		itemLiteral, err := s.buildItemLiteral(ctx, &items[i])
		if err != nil {
			return nil, fmt.Errorf("building item literal for item %d: %w", i, err)
		}
		elements[i] = itemLiteral
	}

	sliceLiteral := &ast.CompositeLit{
		Type: &ast.ArrayType{
			Lbrack: 0,
			Len:    nil,
			Elt:    targetType,
		},
		Lbrace:     0,
		Elts:       elements,
		Rbrace:     0,
		Incomplete: false,
	}

	l.Trace("Built slice literal successfully",
		logger_domain.Int("element_count", len(elements)))

	return sliceLiteral, nil
}

// internalMetadataKeys contains metadata keys that are internal to the framework
// and should not be encoded into user-defined target structs.
var internalMetadataKeys = map[string]bool{
	collection_dto.MetaKeyDraft:      true,
	collection_dto.MetaKeyWordCount:  true,
	collection_dto.MetaKeySections:   true,
	collection_dto.MetaKeyNavigation: true,
}

// buildItemLiteral generates a composite literal for a single ContentItem.
//
// This maps the ContentItem's metadata to struct field initialisers.
// Field names are capitalised to match Go's exported field naming convention.
//
// The function encodes:
//   - User-defined metadata from the Metadata map (excluding internal keys)
//   - Slug from ContentItem if not already in metadata
//   - URL from ContentItem (always added as it's computed, not in frontmatter)
//
// Internal metadata keys (Draft, WordCount, Sections, Navigation) are excluded
// because they are framework-internal and not typically part of user-defined
// target structs.
//
// Takes item (*collection_dto.ContentItem) which provides the content metadata
// to convert.
//
// Returns ast.Expr which is the composite literal representing the item.
// Returns error when a metadata value cannot be converted to an AST expression.
func (s *collectionService) buildItemLiteral(ctx context.Context, item *collection_dto.ContentItem) (ast.Expr, error) {
	kvPairs := make([]ast.Expr, 0, len(item.Metadata)+3)

	addedFields := make(map[string]bool)

	metadataKeys := slices.Sorted(maps.Keys(item.Metadata))

	for _, key := range metadataKeys {
		if internalMetadataKeys[key] {
			continue
		}

		value := item.Metadata[key]

		fieldName := capitalise(key)

		valueLiteral, err := s.valueToASTExpr(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("converting value for field %q: %w", key, err)
		}

		kvPairs = append(kvPairs, createKeyValuePair(fieldName, valueLiteral))
		addedFields[fieldName] = true
	}

	if item.Slug != "" && !addedFields["Slug"] {
		kvPairs = append(kvPairs, createKeyValuePair("Slug", createStringLit(item.Slug)))
	}

	if item.URL != "" && !addedFields["URL"] {
		kvPairs = append(kvPairs, createKeyValuePair("URL", createStringLit(item.URL)))
	}

	return &ast.CompositeLit{
		Type:       nil,
		Lbrace:     0,
		Elts:       kvPairs,
		Rbrace:     0,
		Incomplete: false,
	}, nil
}

// valueToASTExpr converts a Go value to an AST expression.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes value (any) which is the Go value to convert.
//
// Returns ast.Expr which is the corresponding AST expression.
// Returns error when slice or map conversion fails.
func (s *collectionService) valueToASTExpr(ctx context.Context, value any) (ast.Expr, error) {
	if value == nil {
		return createNilIdent(), nil
	}

	switch v := value.(type) {
	case string:
		return createStringLit(v), nil
	case int, int8, int16, int32, int64:
		return createIntLit(v), nil
	case uint, uint8, uint16, uint32, uint64:
		return createIntLit(v), nil
	case float32, float64:
		return createFloatLit(v), nil
	case bool:
		return createBoolIdent(v), nil
	case []any:
		return s.createSliceLit(ctx, v)
	case map[string]any:
		return s.createMapLit(ctx, v)
	default:
		return s.createFallbackStringLit(ctx, v), nil
	}
}

// createSliceLit creates an AST slice literal.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes v ([]any) which contains the elements to include in the slice.
//
// Returns ast.Expr which is the composite literal representing the slice.
// Returns error when any element cannot be converted to an AST expression.
func (s *collectionService) createSliceLit(ctx context.Context, v []any) (ast.Expr, error) {
	elements := make([]ast.Expr, len(v))
	for i, element := range v {
		elemExpr, err := s.valueToASTExpr(ctx, element)
		if err != nil {
			return nil, fmt.Errorf("converting slice element %d: %w", i, err)
		}
		elements[i] = elemExpr
	}

	return &ast.CompositeLit{
		Type: &ast.ArrayType{
			Lbrack: 0,
			Len:    nil,
			Elt:    &ast.Ident{NamePos: 0, Name: "any", Obj: nil},
		},
		Lbrace:     0,
		Elts:       elements,
		Rbrace:     0,
		Incomplete: false,
	}, nil
}

// createMapLit creates an AST map literal.
//
// Keys are sorted for deterministic output order.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes v (map[string]any) which contains the key-value pairs to convert.
//
// Returns ast.Expr which is the constructed map composite literal.
// Returns error when a map value cannot be converted to an AST expression.
func (s *collectionService) createMapLit(ctx context.Context, v map[string]any) (ast.Expr, error) {
	keys := slices.Sorted(maps.Keys(v))

	kvPairs := make([]ast.Expr, 0, len(v))
	for _, key := range keys {
		value := v[key]
		valExpr, err := s.valueToASTExpr(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("converting map value for key %q: %w", key, err)
		}
		kvPairs = append(kvPairs, &ast.KeyValueExpr{
			Key:   createStringLit(key),
			Colon: 0,
			Value: valExpr,
		})
	}

	return &ast.CompositeLit{
		Type: &ast.MapType{
			Map:   0,
			Key:   &ast.Ident{NamePos: 0, Name: "string", Obj: nil},
			Value: &ast.Ident{NamePos: 0, Name: "any", Obj: nil},
		},
		Lbrace:     0,
		Elts:       kvPairs,
		Rbrace:     0,
		Incomplete: false,
	}, nil
}

// createFallbackStringLit creates a fallback string literal for types that
// cannot be directly converted to an AST node.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes v (any) which is the value to convert to a string literal.
//
// Returns *ast.BasicLit which contains the string form of the value.
func (*collectionService) createFallbackStringLit(ctx context.Context, v any) *ast.BasicLit {
	_, l := logger_domain.From(ctx, log)
	l.Warn("Unsupported value type in AST conversion, using string representation",
		logger_domain.String("type", fmt.Sprintf("%T", v)))
	return createStringLit(fmt.Sprintf("%v", v))
}

// createKeyValuePair creates an AST key-value expression.
//
// Takes fieldName (string) which specifies the key name for the expression.
// Takes value (ast.Expr) which provides the value for the expression.
//
// Returns *ast.KeyValueExpr which is the constructed key-value pair.
func createKeyValuePair(fieldName string, value ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key: &ast.Ident{
			NamePos: 0,
			Name:    fieldName,
			Obj:     nil,
		},
		Colon: 0,
		Value: value,
	}
}

// createNilIdent creates an AST identifier node for the nil keyword.
//
// Returns *ast.Ident which represents nil in the abstract syntax tree.
func createNilIdent() *ast.Ident {
	return &ast.Ident{NamePos: 0, Name: "nil", Obj: nil}
}

// createStringLit creates an AST string literal node.
//
// Takes v (string) which is the value to wrap as a quoted string.
//
// Returns *ast.BasicLit which is the AST node for the string.
func createStringLit(v string) *ast.BasicLit {
	return &ast.BasicLit{ValuePos: 0, Kind: token.STRING, Value: fmt.Sprintf("%q", v)}
}

// createIntLit creates an AST integer literal from the given value.
//
// Takes v (any) which provides the integer value to format.
//
// Returns *ast.BasicLit which is the integer literal node.
func createIntLit(v any) *ast.BasicLit {
	return &ast.BasicLit{ValuePos: 0, Kind: token.INT, Value: fmt.Sprintf("%d", v)}
}

// createFloatLit creates an AST float literal.
//
// Takes v (any) which provides the float value to format.
//
// Returns *ast.BasicLit which is the formatted float literal node.
func createFloatLit(v any) *ast.BasicLit {
	return &ast.BasicLit{ValuePos: 0, Kind: token.FLOAT, Value: fmt.Sprintf("%f", v)}
}

// createBoolIdent creates an AST identifier node for a boolean value.
//
// Takes v (bool) which is the boolean value to convert.
//
// Returns *ast.Ident which represents the boolean as an identifier node.
func createBoolIdent(v bool) *ast.Ident {
	return &ast.Ident{NamePos: 0, Name: fmt.Sprintf("%t", v), Obj: nil}
}

// capitalise converts the first letter of a string to uppercase.
// Converts metadata keys to Go exported field names.
//
// Takes str (string) which is the text to capitalise.
//
// Returns string which is the input with its first letter in uppercase.
func capitalise(str string) string {
	if str == "" {
		return ""
	}

	if len(str) == 1 {
		return strings.ToUpper(str)
	}

	return strings.ToUpper(str[:1]) + str[1:]
}
