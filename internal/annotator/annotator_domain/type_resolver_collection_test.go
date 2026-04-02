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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

func TestIsGetCollectionMemberExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression *ast_domain.MemberExpression
		name       string
		expected   bool
	}{
		{
			name: "valid r.GetCollection expression",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "r"},
				Property: &ast_domain.Identifier{Name: "GetCollection"},
			},
			expected: true,
		},
		{
			name: "wrong base identifier",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "state"},
				Property: &ast_domain.Identifier{Name: "GetCollection"},
			},
			expected: false,
		},
		{
			name: "wrong property identifier",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "r"},
				Property: &ast_domain.Identifier{Name: "GetItem"},
			},
			expected: false,
		},
		{
			name: "base is not identifier",
			expression: &ast_domain.MemberExpression{
				Base: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "state"},
					Property: &ast_domain.Identifier{Name: "r"},
				},
				Property: &ast_domain.Identifier{Name: "GetCollection"},
			},
			expected: false,
		},
		{
			name: "property is not identifier",
			expression: &ast_domain.MemberExpression{
				Base: &ast_domain.Identifier{Name: "r"},
				Property: &ast_domain.StringLiteral{
					Value: "GetCollection",
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			result := h.Resolver.isGetCollectionMemberExpr(tc.expression)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractGetCollectionMemberExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		callExpr    *ast_domain.CallExpression
		name        string
		expectedNil bool
	}{
		{
			name: "valid r.GetCollection call",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "r"},
					Property: &ast_domain.Identifier{Name: "GetCollection"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "articles"},
				},
			},
			expectedNil: false,
		},
		{
			name: "generic r.GetCollection[T] call",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.IndexExpression{
					Base: &ast_domain.MemberExpression{
						Base:     &ast_domain.Identifier{Name: "r"},
						Property: &ast_domain.Identifier{Name: "GetCollection"},
					},
					Index: &ast_domain.Identifier{Name: "Article"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "articles"},
				},
			},
			expectedNil: false,
		},
		{
			name: "not a member expression",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "someFunc"},
				Args:   []ast_domain.Expression{},
			},
			expectedNil: true,
		},
		{
			name: "wrong method name",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "r"},
					Property: &ast_domain.Identifier{Name: "GetItem"},
				},
				Args: []ast_domain.Expression{},
			},
			expectedNil: true,
		},
		{
			name: "wrong base for generic",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.IndexExpression{
					Base: &ast_domain.MemberExpression{
						Base:     &ast_domain.Identifier{Name: "state"},
						Property: &ast_domain.Identifier{Name: "GetCollection"},
					},
					Index: &ast_domain.Identifier{Name: "Article"},
				},
				Args: []ast_domain.Expression{},
			},
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			result := h.Resolver.extractGetCollectionMemberExpr(tc.callExpr)

			if tc.expectedNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestExtractStringLiteralFromPikoAST(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		expression  ast_domain.Expression
		expected    string
		expectError bool
	}{
		{
			name:        "valid string literal",
			expression:  &ast_domain.StringLiteral{Value: "articles"},
			expected:    "articles",
			expectError: false,
		},
		{
			name:        "empty string literal",
			expression:  &ast_domain.StringLiteral{Value: ""},
			expected:    "",
			expectError: false,
		},
		{
			name:        "identifier is not string literal",
			expression:  &ast_domain.Identifier{Name: "myVar"},
			expected:    "",
			expectError: true,
		},
		{
			name:        "integer literal is not string literal",
			expression:  &ast_domain.IntegerLiteral{Value: 42},
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			result, err := h.Resolver.extractStringLiteralFromPikoAST(tc.expression)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestExtractTypeParameterFromMemberExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		memberExpr  *ast_domain.MemberExpression
		expectedTyp string
		expectError bool
	}{
		{
			name: "valid type parameter",
			memberExpr: &ast_domain.MemberExpression{
				Base: &ast_domain.Identifier{Name: "r"},
				Property: &ast_domain.IndexExpression{
					Base:  &ast_domain.Identifier{Name: "GetCollection"},
					Index: &ast_domain.Identifier{Name: "Article"},
				},
			},
			expectedTyp: "Article",
			expectError: false,
		},
		{
			name: "qualified type parameter",
			memberExpr: &ast_domain.MemberExpression{
				Base: &ast_domain.Identifier{Name: "r"},
				Property: &ast_domain.IndexExpression{
					Base: &ast_domain.Identifier{Name: "GetCollection"},
					Index: &ast_domain.MemberExpression{
						Base:     &ast_domain.Identifier{Name: "models"},
						Property: &ast_domain.Identifier{Name: "Article"},
					},
				},
			},
			expectedTyp: "models.Article",
			expectError: false,
		},
		{
			name: "no type parameter",
			memberExpr: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "r"},
				Property: &ast_domain.Identifier{Name: "GetCollection"},
			},
			expectedTyp: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			typeName, typeExpr, err := h.Resolver.extractTypeParameterFromMemberExpr(tc.memberExpr)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedTyp, typeName)
				assert.NotNil(t, typeExpr)
			}
		})
	}
}

func TestParseGetCollectionOptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		validate    func(*testing.T, any)
		name        string
		arguments   []ast_domain.Expression
		expectError bool
	}{
		{
			name:        "no options returns empty FetchOptions",
			arguments:   []ast_domain.Expression{},
			expectError: false,
			validate: func(t *testing.T, result any) {
				opts, ok := result.(collection_dto.FetchOptions)
				require.True(t, ok)
				assert.Empty(t, opts.ProviderName)
				assert.Empty(t, opts.Locale)
				assert.Empty(t, opts.Filters)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			result, err := h.Resolver.parseGetCollectionOptions(h.Context, tc.arguments)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tc.validate(t, result)
			}
		})
	}
}

func TestApplyWithProvider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		expectedProvider string
		arguments        []ast_domain.Expression
		expectError      bool
	}{
		{
			name: "valid provider name",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "contentful"},
			},
			expectedProvider: "contentful",
			expectError:      false,
		},
		{
			name:             "no arguments",
			arguments:        []ast_domain.Expression{},
			expectedProvider: "",
			expectError:      true,
		},
		{
			name: "too many arguments",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "contentful"},
				&ast_domain.StringLiteral{Value: "extra"},
			},
			expectedProvider: "",
			expectError:      true,
		},
		{
			name: "non-string argument",
			arguments: []ast_domain.Expression{
				&ast_domain.IntegerLiteral{Value: 123},
			},
			expectedProvider: "",
			expectError:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options := &collection_dto.FetchOptions{
				Filters: make(map[string]any),
			}
			scope := make(map[string]any)

			err := applyWithProvider(tc.arguments, options, scope)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedProvider, options.ProviderName)
			}
		})
	}
}

func TestApplyWithLocale(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		expectedLocale string
		arguments      []ast_domain.Expression
		expectError    bool
	}{
		{
			name: "valid locale",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "en-GB"},
			},
			expectedLocale: "en-GB",
			expectError:    false,
		},
		{
			name:           "no arguments",
			arguments:      []ast_domain.Expression{},
			expectedLocale: "",
			expectError:    true,
		},
		{
			name: "non-string argument",
			arguments: []ast_domain.Expression{
				&ast_domain.BooleanLiteral{Value: true},
			},
			expectedLocale: "",
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options := &collection_dto.FetchOptions{
				Filters: make(map[string]any),
			}
			scope := make(map[string]any)

			err := applyWithLocale(tc.arguments, options, scope)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedLocale, options.Locale)
			}
		})
	}
}

func TestApplyWithFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expectedValue any
		name          string
		expectedKey   string
		arguments     []ast_domain.Expression
		expectError   bool
	}{
		{
			name: "valid string filter",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "category"},
				&ast_domain.StringLiteral{Value: "tech"},
			},
			expectedKey:   "category",
			expectedValue: "tech",
			expectError:   false,
		},
		{
			name: "valid integer filter",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "minPrice"},
				&ast_domain.IntegerLiteral{Value: 100},
			},
			expectedKey:   "minPrice",
			expectedValue: float64(100),
			expectError:   false,
		},
		{
			name: "too few arguments",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "category"},
			},
			expectedKey:   "",
			expectedValue: nil,
			expectError:   true,
		},
		{
			name:          "no arguments",
			arguments:     []ast_domain.Expression{},
			expectedKey:   "",
			expectedValue: nil,
			expectError:   true,
		},
		{
			name: "non-string key",
			arguments: []ast_domain.Expression{
				&ast_domain.IntegerLiteral{Value: 123},
				&ast_domain.StringLiteral{Value: "value"},
			},
			expectedKey:   "",
			expectedValue: nil,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options := &collection_dto.FetchOptions{
				Filters: make(map[string]any),
			}
			scope := make(map[string]any)

			err := applyWithFilter(tc.arguments, options, scope)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedValue, options.Filters[tc.expectedKey])
			}
		})
	}
}

func TestApplyCollectionOption(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		validate    func(*testing.T, *collection_dto.FetchOptions)
		name        string
		optionName  string
		arguments   []ast_domain.Expression
		expectError bool
	}{
		{
			name:       "WithProvider option",
			optionName: "WithProvider",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "cms"},
			},
			expectError: false,
			validate: func(t *testing.T, opts *collection_dto.FetchOptions) {
				assert.Equal(t, "cms", opts.ProviderName)
			},
		},
		{
			name:       "WithLocale option",
			optionName: "WithLocale",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "fr-FR"},
			},
			expectError: false,
			validate: func(t *testing.T, opts *collection_dto.FetchOptions) {
				assert.Equal(t, "fr-FR", opts.Locale)
			},
		},
		{
			name:       "WithFilter option",
			optionName: "WithFilter",
			arguments: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "status"},
				&ast_domain.StringLiteral{Value: "published"},
			},
			expectError: false,
			validate: func(t *testing.T, opts *collection_dto.FetchOptions) {
				assert.Equal(t, "published", opts.Filters["status"])
			},
		},
		{
			name:        "unsupported option",
			optionName:  "WithUnknown",
			arguments:   []ast_domain.Expression{},
			expectError: true,
			validate:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options := &collection_dto.FetchOptions{
				Filters: make(map[string]any),
			}
			scope := make(map[string]any)

			err := applyCollectionOption(tc.optionName, tc.arguments, options, scope)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tc.validate(t, options)
			}
		})
	}
}

func TestParseCollectionOptionExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		optionExpr  ast_domain.Expression
		name        string
		expectError bool
	}{
		{
			name: "valid option expression",
			optionExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "data"},
					Property: &ast_domain.Identifier{Name: "WithProvider"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "contentful"},
				},
			},
			expectError: false,
		},
		{
			name:        "not a call expression",
			optionExpr:  &ast_domain.StringLiteral{Value: "invalid"},
			expectError: true,
		},
		{
			name: "callee is not member expression",
			optionExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "WithProvider"},
				Args:   []ast_domain.Expression{},
			},
			expectError: true,
		},
		{
			name: "property is not identifier",
			optionExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "data"},
					Property: &ast_domain.StringLiteral{Value: "WithProvider"},
				},
				Args: []ast_domain.Expression{},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options := &collection_dto.FetchOptions{
				Filters: make(map[string]any),
			}
			scope := make(map[string]any)

			err := parseCollectionOptionExpr(tc.optionExpr, options, scope)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTryResolveGetCollectionCall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		callExpr           *ast_domain.CallExpression
		name               string
		expectedHandled    bool
		expectedDiagnostic bool
	}{
		{
			name: "not a GetCollection call returns false",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "someFunc"},
				Args:   []ast_domain.Expression{},
			},
			expectedHandled:    false,
			expectedDiagnostic: false,
		},
		{
			name: "GetCollection without collectionService generates diagnostic",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "r"},
					Property: &ast_domain.Identifier{Name: "GetCollection"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "articles"},
				},
			},
			expectedHandled:    true,
			expectedDiagnostic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			_, handled := h.Resolver.tryResolveGetCollectionCall(
				h.Context,
				tc.callExpr,
				ast_domain.Location{Line: 1, Column: 1},
			)

			assert.Equal(t, tc.expectedHandled, handled)

			if tc.expectedDiagnostic {
				assert.True(t, h.HasDiagnostics())
			}
		})
	}
}

func TestHandleMissingCollectionService(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Args: []ast_domain.Expression{},
	}

	result := h.Resolver.handleMissingCollectionService(
		h.Context,
		callExpr,
		ast_domain.Location{Line: 5, Column: 10},
	)

	assert.Nil(t, result)
	require.True(t, h.HasDiagnostics())

	diagnostic := h.GetFirstDiagnostic()
	require.NotNil(t, diagnostic)
	assert.Contains(t, diagnostic.Message, "Collection system not initialised")
	assert.Equal(t, ast_domain.Error, diagnostic.Severity)
}

func TestHandleCollectionError(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "articles"},
		},
	}
	testErr := assert.AnError
	loc := ast_domain.Location{Line: 10, Column: 5}

	result := h.Resolver.handleCollectionError(
		h.Context,
		callExpr,
		loc,
		testErr,
		"Test log message",
		"Test diagnostic: %v",
	)

	assert.Nil(t, result)
	require.True(t, h.HasDiagnostics())

	diagnostic := h.GetFirstDiagnostic()
	require.NotNil(t, diagnostic)
	assert.Contains(t, diagnostic.Message, "Test diagnostic:")
	assert.Contains(t, diagnostic.Message, testErr.Error())
	assert.Equal(t, ast_domain.Error, diagnostic.Severity)
	assert.Equal(t, loc, diagnostic.Location)
}

func TestHandleCollectionSemanticsError(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "articles"},
		},
	}
	testErr := assert.AnError
	loc := ast_domain.Location{Line: 15, Column: 3}

	result := h.Resolver.handleCollectionSemanticsError(h.Context, callExpr, loc, testErr)

	assert.Nil(t, result)
	require.True(t, h.HasDiagnostics())

	diagnostic := h.GetFirstDiagnostic()
	require.NotNil(t, diagnostic)
	assert.Contains(t, diagnostic.Message, "Invalid GetCollection() call")
	assert.Equal(t, ast_domain.Error, diagnostic.Severity)
}

func TestHandleCollectionProcessError(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "products"},
		},
	}
	testErr := assert.AnError
	loc := ast_domain.Location{Line: 20, Column: 1}

	result := h.Resolver.handleCollectionProcessError(h.Context, callExpr, loc, testErr)

	assert.Nil(t, result)
	require.True(t, h.HasDiagnostics())

	diagnostic := h.GetFirstDiagnostic()
	require.NotNil(t, diagnostic)
	assert.Contains(t, diagnostic.Message, "Failed to process collection")
	assert.Equal(t, ast_domain.Error, diagnostic.Severity)
}

func TestHandleCollectionError_IncludesExpressionString(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "products"},
		},
	}
	loc := ast_domain.Location{Line: 1, Column: 1}

	h.Resolver.handleCollectionError(
		h.Context,
		callExpr,
		loc,
		assert.AnError,
		"log message",
		"diagnostic: %v",
	)

	require.True(t, h.HasDiagnostics())
	diagnostic := h.GetFirstDiagnostic()
	require.NotNil(t, diagnostic)
	assert.NotEmpty(t, diagnostic.Expression, "diagnostic should include the expression string")
}

func TestExtractGetCollectionSemantics_NoArgs(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "GetCollection"},
				Index: &ast_domain.Identifier{Name: "Article"},
			},
		},
		Args: []ast_domain.Expression{},
	}
	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	require.True(t, ok)

	_, err := h.Resolver.extractGetCollectionSemantics(h.Context, callExpr, memberExpr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 argument")
}

func TestExtractGetCollectionSemantics_NonStringFirstArg(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "GetCollection"},
				Index: &ast_domain.Identifier{Name: "Article"},
			},
		},
		Args: []ast_domain.Expression{
			&ast_domain.IntegerLiteral{Value: 42},
		},
	}
	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	require.True(t, ok)

	_, err := h.Resolver.extractGetCollectionSemantics(h.Context, callExpr, memberExpr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "first argument must be a string literal")
}

func TestExtractGetCollectionSemantics_NoTypeParameter(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "articles"},
		},
	}
	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	require.True(t, ok)

	_, err := h.Resolver.extractGetCollectionSemantics(h.Context, callExpr, memberExpr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracting type parameter")
}

func TestExtractGetCollectionSemantics_ValidCall(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "GetCollection"},
				Index: &ast_domain.Identifier{Name: "Article"},
			},
		},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "articles"},
		},
	}
	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	require.True(t, ok)

	semantics, err := h.Resolver.extractGetCollectionSemantics(h.Context, callExpr, memberExpr)

	require.NoError(t, err)
	require.NotNil(t, semantics)
	assert.Equal(t, "articles", semantics.CollectionName)
	assert.Equal(t, "Article", semantics.TargetTypeName)
	assert.NotNil(t, semantics.TargetTypeExpression)
}

func TestExtractGetCollectionSemantics_WithOptions(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "GetCollection"},
				Index: &ast_domain.Identifier{Name: "Product"},
			},
		},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "products"},
			&ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "data"},
					Property: &ast_domain.Identifier{Name: "WithProvider"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "cms"},
				},
			},
		},
	}
	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	require.True(t, ok)

	semantics, err := h.Resolver.extractGetCollectionSemantics(h.Context, callExpr, memberExpr)

	require.NoError(t, err)
	require.NotNil(t, semantics)
	assert.Equal(t, "products", semantics.CollectionName)
	assert.Equal(t, "Product", semantics.TargetTypeName)
	assert.NotNil(t, semantics.Options)

	opts, ok := semantics.Options.(collection_dto.FetchOptions)
	require.True(t, ok, "options should be FetchOptions")
	assert.Equal(t, "cms", opts.ProviderName)
}

func TestParseGetCollectionOptions_WithProviderAndLocale(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	arguments := []ast_domain.Expression{
		&ast_domain.CallExpression{
			Callee: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "data"},
				Property: &ast_domain.Identifier{Name: "WithProvider"},
			},
			Args: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "contentful"},
			},
		},
		&ast_domain.CallExpression{
			Callee: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "data"},
				Property: &ast_domain.Identifier{Name: "WithLocale"},
			},
			Args: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "en-GB"},
			},
		},
	}

	result, err := h.Resolver.parseGetCollectionOptions(h.Context, arguments)

	require.NoError(t, err)
	opts, ok := result.(collection_dto.FetchOptions)
	require.True(t, ok)
	assert.Equal(t, "contentful", opts.ProviderName)
	assert.Equal(t, "en-GB", opts.Locale)
}

func TestParseGetCollectionOptions_WithFilter(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	arguments := []ast_domain.Expression{
		&ast_domain.CallExpression{
			Callee: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "data"},
				Property: &ast_domain.Identifier{Name: "WithFilter"},
			},
			Args: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "status"},
				&ast_domain.StringLiteral{Value: "published"},
			},
		},
	}

	result, err := h.Resolver.parseGetCollectionOptions(h.Context, arguments)

	require.NoError(t, err)
	opts, ok := result.(collection_dto.FetchOptions)
	require.True(t, ok)
	assert.Equal(t, "published", opts.Filters["status"])
}

func TestParseGetCollectionOptions_InvalidOption(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	arguments := []ast_domain.Expression{
		&ast_domain.CallExpression{
			Callee: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "data"},
				Property: &ast_domain.Identifier{Name: "WithUnknown"},
			},
			Args: []ast_domain.Expression{},
		},
	}

	_, err := h.Resolver.parseGetCollectionOptions(h.Context, arguments)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported option")
}

func TestParseGetCollectionOptions_NonCallExprOption(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	arguments := []ast_domain.Expression{
		&ast_domain.StringLiteral{Value: "not a call"},
	}

	_, err := h.Resolver.parseGetCollectionOptions(h.Context, arguments)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "option must be a function call")
}

func TestExtractGenericGetCollectionMemberExpr_BaseNotMemberExpr(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	indexExpr := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "notAMemberExpr"},
		Index: &ast_domain.Identifier{Name: "Article"},
	}

	result := h.Resolver.extractGenericGetCollectionMemberExpr(indexExpr)

	assert.Nil(t, result, "should return nil when base is not a MemberExpr")
}

func TestExtractGenericGetCollectionMemberExpr_WrongPropertyName(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	indexExpr := &ast_domain.IndexExpression{
		Base: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetItem"},
		},
		Index: &ast_domain.Identifier{Name: "Article"},
	}

	result := h.Resolver.extractGenericGetCollectionMemberExpr(indexExpr)

	assert.Nil(t, result, "should return nil when property is not GetCollection")
}

func TestExtractGenericGetCollectionMemberExpr_PropertyNotIdentifier(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	indexExpr := &ast_domain.IndexExpression{
		Base: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.StringLiteral{Value: "GetCollection"},
		},
		Index: &ast_domain.Identifier{Name: "Article"},
	}

	result := h.Resolver.extractGenericGetCollectionMemberExpr(indexExpr)

	assert.Nil(t, result, "should return nil when property is not an identifier")
}

func TestExtractGenericGetCollectionMemberExpr_BaseIdentNotR(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	indexExpr := &ast_domain.IndexExpression{
		Base: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "state"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Index: &ast_domain.Identifier{Name: "Article"},
	}

	result := h.Resolver.extractGenericGetCollectionMemberExpr(indexExpr)

	assert.Nil(t, result, "should return nil when base identifier is not r")
}

func TestExtractGenericGetCollectionMemberExpr_BaseNotIdentifier(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	indexExpr := &ast_domain.IndexExpression{
		Base: &ast_domain.MemberExpression{
			Base: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "state"},
				Property: &ast_domain.Identifier{Name: "r"},
			},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Index: &ast_domain.Identifier{Name: "Article"},
	}

	result := h.Resolver.extractGenericGetCollectionMemberExpr(indexExpr)

	assert.Nil(t, result, "should return nil when base of MemberExpr is not an identifier")
}

func TestExtractGenericGetCollectionMemberExpr_ValidGenericCall(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	indexExpr := &ast_domain.IndexExpression{
		Base: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "r"},
			Property: &ast_domain.Identifier{Name: "GetCollection"},
		},
		Index: &ast_domain.Identifier{Name: "Product"},
	}

	result := h.Resolver.extractGenericGetCollectionMemberExpr(indexExpr)

	require.NotNil(t, result, "should return a MemberExpr for a valid generic GetCollection call")

	baseIdent, ok := result.Base.(*ast_domain.Identifier)
	require.True(t, ok)
	assert.Equal(t, "r", baseIdent.Name)

	propIndex, ok := result.Property.(*ast_domain.IndexExpression)
	require.True(t, ok)
	assert.NotNil(t, propIndex.Index)
}

func TestApplyWithLocale_TooManyArgs(t *testing.T) {
	t.Parallel()

	options := &collection_dto.FetchOptions{
		Filters: make(map[string]any),
	}
	scope := make(map[string]any)

	err := applyWithLocale([]ast_domain.Expression{
		&ast_domain.StringLiteral{Value: "en-GB"},
		&ast_domain.StringLiteral{Value: "extra"},
	}, options, scope)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithLocale expects 1 argument, got 2")
}

func TestApplyWithFilter_TooManyArgs(t *testing.T) {
	t.Parallel()

	options := &collection_dto.FetchOptions{
		Filters: make(map[string]any),
	}
	scope := make(map[string]any)

	err := applyWithFilter([]ast_domain.Expression{
		&ast_domain.StringLiteral{Value: "key"},
		&ast_domain.StringLiteral{Value: "value"},
		&ast_domain.StringLiteral{Value: "extra"},
	}, options, scope)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithFilter expects 2 arguments")
}

func TestApplyWithFilter_BooleanValue(t *testing.T) {
	t.Parallel()

	options := &collection_dto.FetchOptions{
		Filters: make(map[string]any),
	}
	scope := make(map[string]any)

	err := applyWithFilter([]ast_domain.Expression{
		&ast_domain.StringLiteral{Value: "active"},
		&ast_domain.BooleanLiteral{Value: true},
	}, options, scope)

	require.NoError(t, err)
	assert.Equal(t, true, options.Filters["active"])
}
