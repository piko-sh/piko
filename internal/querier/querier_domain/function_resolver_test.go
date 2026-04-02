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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func sqlType(engineName string, category querier_dto.SQLTypeCategory) querier_dto.SQLType {
	return querier_dto.SQLType{
		EngineName: engineName,
		Category:   category,
	}
}

func funcArg(name string, typ querier_dto.SQLType) querier_dto.FunctionArgument {
	return querier_dto.FunctionArgument{
		Name: name,
		Type: typ,
	}
}

func funcSig(name string, returnType querier_dto.SQLType, args ...querier_dto.FunctionArgument) *querier_dto.FunctionSignature {
	return &querier_dto.FunctionSignature{
		Name:       name,
		ReturnType: returnType,
		Arguments:  args,
	}
}

var intType = sqlType("integer", querier_dto.TypeCategoryInteger)

var textType = sqlType("text", querier_dto.TypeCategoryText)

var floatType = sqlType("float8", querier_dto.TypeCategoryFloat)

var boolType = sqlType("boolean", querier_dto.TypeCategoryBoolean)

var unknownType = sqlType("", querier_dto.TypeCategoryUnknown)

func TestNewFunctionResolver(t *testing.T) {
	t.Parallel()

	t.Run("merges builtin functions from engine", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"lower": {
					funcSig("lower", textType, funcArg("input", textType)),
				},
				"abs": {
					funcSig("abs", intType, funcArg("n", intType)),
				},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		assert.Len(t, resolver.functions["lower"], 1, "expected one overload for lower")
		assert.Len(t, resolver.functions["abs"], 1, "expected one overload for abs")
	})

	t.Run("case-insensitive builtin keys", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"UPPER": {
					funcSig("UPPER", textType, funcArg("input", textType)),
				},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		assert.Len(t, resolver.functions["upper"], 1, "builtin key should be lowercased")
		assert.Empty(t, resolver.functions["UPPER"], "original-cased key should not exist")
	})

	t.Run("catalogue function appends new overload", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"myfunc": {
					funcSig("myfunc", intType, funcArg("a", intType)),
				},
			},
		}
		catalogue := newTestCatalogue("public")

		catalogue.Schemas["public"].Functions["myfunc"] = []*querier_dto.FunctionSignature{
			funcSig("myfunc", textType, funcArg("a", textType)),
		}
		engine := &mockEngine{}

		resolver := newFunctionResolver(builtins, catalogue, engine)

		require.Len(t, resolver.functions["myfunc"], 2, "expected two overloads for myfunc")
	})

	t.Run("catalogue function overrides builtin with same argument types", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"myfunc": {
					funcSig("myfunc", intType, funcArg("a", intType)),
				},
			},
		}

		catalogueSig := funcSig("myfunc", textType, funcArg("a", intType))
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Functions["myfunc"] = []*querier_dto.FunctionSignature{catalogueSig}
		engine := &mockEngine{}

		resolver := newFunctionResolver(builtins, catalogue, engine)

		require.Len(t, resolver.functions["myfunc"], 1, "duplicate should be replaced, not appended")
		assert.Equal(t, querier_dto.TypeCategoryText, resolver.functions["myfunc"][0].ReturnType.Category,
			"catalogue signature should override the builtin")
	})

	t.Run("nil builtins handled gracefully", func(t *testing.T) {
		t.Parallel()

		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Functions["myfunc"] = []*querier_dto.FunctionSignature{
			funcSig("myfunc", intType),
		}
		engine := &mockEngine{}

		resolver := newFunctionResolver(nil, catalogue, engine)

		assert.Len(t, resolver.functions["myfunc"], 1, "catalogue functions should still be present")
	})

	t.Run("nil catalogue handled gracefully", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"abs": {
					funcSig("abs", intType, funcArg("n", intType)),
				},
			},
		}
		engine := &mockEngine{}

		resolver := newFunctionResolver(builtins, nil, engine)

		assert.Len(t, resolver.functions["abs"], 1, "builtin functions should still be present")
	})
}

func TestFunctionResolver_Resolve(t *testing.T) {
	t.Parallel()

	t.Run("unknown function returns Q005 error", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("nonexistent", "", []querier_dto.SQLType{intType})

		assert.Nil(t, match, "unknown function should not produce a match")
		require.NotNil(t, srcErr, "unknown function should produce an error")
		assert.Equal(t, "Q005", srcErr.Code)
		assert.Contains(t, srcErr.Message, "nonexistent")
	})

	t.Run("exact arity match with single overload", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"abs": {
					funcSig("abs", intType, funcArg("n", intType)),
				},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("abs", "", []querier_dto.SQLType{intType})

		assert.Nil(t, srcErr, "should not produce an error")
		require.NotNil(t, match, "should produce a match")
		assert.Equal(t, querier_dto.TypeCategoryInteger, match.returnType.Category)
		assert.Equal(t, "integer", match.returnType.EngineName)
	})

	t.Run("case-insensitive function name lookup", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"lower": {
					funcSig("lower", textType, funcArg("input", textType)),
				},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("LOWER", "", []querier_dto.SQLType{textType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.Equal(t, querier_dto.TypeCategoryText, match.returnType.Category)
	})

	t.Run("best overload selected by score", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"myfunc": {
					funcSig("myfunc", textType, funcArg("a", textType)),
					funcSig("myfunc", intType, funcArg("a", intType)),
				},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("myfunc", "", []querier_dto.SQLType{intType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.Equal(t, querier_dto.TypeCategoryInteger, match.returnType.Category,
			"should pick the integer overload as an exact match")
	})

	t.Run("variadic function accepts extra arguments", func(t *testing.T) {
		t.Parallel()

		variadicSig := funcSig("concat", textType, funcArg("a", textType))
		variadicSig.IsVariadic = true

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"concat": {variadicSig},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("concat", "", []querier_dto.SQLType{textType, textType, textType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.Equal(t, querier_dto.TypeCategoryText, match.returnType.Category)
	})

	t.Run("schema filtering selects matching schema", func(t *testing.T) {
		t.Parallel()

		publicSig := funcSig("myfunc", intType, funcArg("a", intType))
		publicSig.Schema = "public"

		customSig := funcSig("myfunc", textType, funcArg("a", intType))
		customSig.Schema = "custom"

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"myfunc": {publicSig, customSig},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("myfunc", "custom", []querier_dto.SQLType{intType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.Equal(t, querier_dto.TypeCategoryText, match.returnType.Category,
			"should pick the custom-schema overload")
	})

	t.Run("no arity match returns Q005 error", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"abs": {
					funcSig("abs", intType, funcArg("n", intType)),
				},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("abs", "", []querier_dto.SQLType{intType, intType})

		assert.Nil(t, match, "wrong arity should not produce a match")
		require.NotNil(t, srcErr)
		assert.Equal(t, "Q005", srcErr.Code)
		assert.Contains(t, srcErr.Message, "no matching overload")
	})

	t.Run("zero-argument function resolves correctly", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"now": {
					funcSig("now", sqlType("timestamptz", querier_dto.TypeCategoryTemporal)),
				},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("now", "", nil)

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.Equal(t, querier_dto.TypeCategoryTemporal, match.returnType.Category)
	})

	t.Run("preserves aggregate and returnsSet flags", func(t *testing.T) {
		t.Parallel()

		aggSig := funcSig("count", intType, funcArg("col", intType))
		aggSig.IsAggregate = true
		aggSig.ReturnsSet = false
		aggSig.NullableBehaviour = querier_dto.FunctionNullableNeverNull
		aggSig.DataAccess = querier_dto.DataAccessReadOnly

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"count": {aggSig},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("count", "", []querier_dto.SQLType{intType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.True(t, match.isAggregate, "isAggregate should be propagated")
		assert.False(t, match.returnsSet, "returnsSet should be propagated")
		assert.Equal(t, querier_dto.FunctionNullableNeverNull, match.nullableBehaviour)
		assert.Equal(t, querier_dto.DataAccessReadOnly, match.dataAccess)
	})

	t.Run("implicit cast selects compatible overload", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"sqrt": {
					funcSig("sqrt", floatType, funcArg("n", floatType)),
				},
			},
		}
		engine := &mockEngine{
			canImplicitCastFn: func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
				return from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat
			},
		}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("sqrt", "", []querier_dto.SQLType{intType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.Equal(t, querier_dto.TypeCategoryFloat, match.returnType.Category)
	})

	t.Run("no implicit cast returns Q005 error", func(t *testing.T) {
		t.Parallel()

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"boolcheck": {
					funcSig("boolcheck", boolType, funcArg("b", boolType)),
				},
			},
		}
		engine := &mockEngine{
			canImplicitCastFn: func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
				return false
			},
		}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("boolcheck", "", []querier_dto.SQLType{textType})

		assert.Nil(t, match, "incompatible type should not produce a match")
		require.NotNil(t, srcErr)
		assert.Equal(t, "Q005", srcErr.Code)
	})

	t.Run("optional arguments allow fewer args than declared", func(t *testing.T) {
		t.Parallel()

		sig := funcSig("substr", textType,
			funcArg("str", textType),
			funcArg("start", intType),
			funcArg("length", intType),
		)

		sig.MinArguments = 2

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"substr": {sig},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("substr", "", []querier_dto.SQLType{textType, intType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
		assert.Equal(t, querier_dto.TypeCategoryText, match.returnType.Category)
	})

	t.Run("schema-qualified skips mismatched schema candidates", func(t *testing.T) {
		t.Parallel()

		sig := funcSig("revenue", floatType, funcArg("amount", floatType))
		sig.Schema = "analytics"

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"revenue": {sig},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("revenue", "sales", []querier_dto.SQLType{floatType})

		assert.Nil(t, match)
		require.NotNil(t, srcErr)
		assert.Equal(t, "Q005", srcErr.Code)
	})

	t.Run("empty schema on candidate matches any requested schema", func(t *testing.T) {
		t.Parallel()

		sig := funcSig("unscoped", intType, funcArg("n", intType))
		sig.Schema = ""

		builtins := &querier_dto.FunctionCatalogue{
			Functions: map[string][]*querier_dto.FunctionSignature{
				"unscoped": {sig},
			},
		}
		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")

		resolver := newFunctionResolver(builtins, catalogue, engine)

		match, srcErr := resolver.Resolve("unscoped", "anyschema", []querier_dto.SQLType{intType})

		assert.Nil(t, srcErr)
		require.NotNil(t, match)
	})
}

func TestScoreCandidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		candidate      *querier_dto.FunctionSignature
		argumentTypes  []querier_dto.SQLType
		canCastFn      func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool
		wantScore      int
		wantExactCount int
		wantViable     bool
	}{
		{
			name:           "exact match scores 3 per argument",
			candidate:      funcSig("f", intType, funcArg("a", intType), funcArg("b", intType)),
			argumentTypes:  []querier_dto.SQLType{intType, intType},
			wantScore:      6,
			wantExactCount: 2,
			wantViable:     true,
		},
		{
			name: "same category different engine name scores 2 per argument",
			candidate: funcSig("f", intType,
				funcArg("a", sqlType("int4", querier_dto.TypeCategoryInteger)),
			),
			argumentTypes:  []querier_dto.SQLType{sqlType("int8", querier_dto.TypeCategoryInteger)},
			wantScore:      2,
			wantExactCount: 0,
			wantViable:     true,
		},
		{
			name:          "implicit cast scores 1 per argument",
			candidate:     funcSig("f", floatType, funcArg("n", floatType)),
			argumentTypes: []querier_dto.SQLType{intType},
			canCastFn: func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
				return from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat
			},
			wantScore:      1,
			wantExactCount: 0,
			wantViable:     true,
		},
		{
			name:          "no match makes candidate not viable",
			candidate:     funcSig("f", boolType, funcArg("b", boolType)),
			argumentTypes: []querier_dto.SQLType{textType},
			canCastFn: func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
				return false
			},
			wantScore:      0,
			wantExactCount: 0,
			wantViable:     false,
		},
		{
			name:           "unknown actual type scores 2",
			candidate:      funcSig("f", intType, funcArg("a", intType)),
			argumentTypes:  []querier_dto.SQLType{unknownType},
			wantScore:      2,
			wantExactCount: 0,
			wantViable:     true,
		},
		{
			name: "unknown expected type scores 2",
			candidate: funcSig("f", intType,
				funcArg("a", unknownType),
			),
			argumentTypes:  []querier_dto.SQLType{intType},
			wantScore:      2,
			wantExactCount: 0,
			wantViable:     true,
		},
		{
			name: "variadic function with extra args applies penalty",
			candidate: func() *querier_dto.FunctionSignature {
				sig := funcSig("f", textType, funcArg("a", textType))
				sig.IsVariadic = true
				return sig
			}(),
			argumentTypes:  []querier_dto.SQLType{textType, textType, textType},
			wantScore:      8,
			wantExactCount: 3,
			wantViable:     true,
		},
		{
			name: "variadic function matching exactly has no penalty",
			candidate: func() *querier_dto.FunctionSignature {
				sig := funcSig("f", textType, funcArg("a", textType))
				sig.IsVariadic = true
				return sig
			}(),
			argumentTypes:  []querier_dto.SQLType{textType},
			wantScore:      3,
			wantExactCount: 1,
			wantViable:     true,
		},
		{
			name:           "zero-argument function scores 1",
			candidate:      funcSig("f", intType),
			argumentTypes:  nil,
			wantScore:      1,
			wantExactCount: 0,
			wantViable:     true,
		},
		{
			name:           "too few arguments makes candidate not viable",
			candidate:      funcSig("f", intType, funcArg("a", intType), funcArg("b", intType)),
			argumentTypes:  []querier_dto.SQLType{intType},
			wantScore:      0,
			wantExactCount: 0,
			wantViable:     false,
		},
		{
			name:           "too many arguments for non-variadic makes candidate not viable",
			candidate:      funcSig("f", intType, funcArg("a", intType)),
			argumentTypes:  []querier_dto.SQLType{intType, intType},
			wantScore:      0,
			wantExactCount: 0,
			wantViable:     false,
		},
		{
			name: "mixed scores across multiple arguments",
			candidate: funcSig("f", intType,
				funcArg("a", intType),
				funcArg("b", sqlType("int4", querier_dto.TypeCategoryInteger)),
			),
			argumentTypes: []querier_dto.SQLType{
				intType,
				sqlType("int8", querier_dto.TypeCategoryInteger),
			},
			wantScore:      5,
			wantExactCount: 1,
			wantViable:     true,
		},
		{
			name: "MinArguments allows fewer args than declared",
			candidate: func() *querier_dto.FunctionSignature {
				sig := funcSig("f", textType,
					funcArg("a", textType),
					funcArg("b", intType),
					funcArg("c", intType),
				)
				sig.MinArguments = 1
				return sig
			}(),
			argumentTypes:  []querier_dto.SQLType{textType, intType},
			wantScore:      6,
			wantExactCount: 2,
			wantViable:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine := &mockEngine{
				canImplicitCastFn: tt.canCastFn,
			}
			resolver := &functionResolver{
				functions: make(map[string][]*querier_dto.FunctionSignature),
				engine:    engine,
			}

			totalScore, exactCount, viable := resolver.scoreCandidate(tt.candidate, tt.argumentTypes)

			assert.Equal(t, tt.wantViable, viable, "viable mismatch")
			if viable {
				assert.Equal(t, tt.wantScore, totalScore, "totalScore mismatch")
				assert.Equal(t, tt.wantExactCount, exactCount, "exactCount mismatch")
			}
		})
	}
}

func TestMergeOrAppendSignature(t *testing.T) {
	t.Parallel()

	t.Run("appends new signature with different argument types", func(t *testing.T) {
		t.Parallel()

		existing := []*querier_dto.FunctionSignature{
			funcSig("myfunc", intType, funcArg("a", intType)),
		}
		newSig := funcSig("myfunc", textType, funcArg("a", textType))

		result := mergeOrAppendSignature(existing, newSig)

		require.Len(t, result, 2, "new signature with different args should be appended")
		assert.Equal(t, querier_dto.TypeCategoryInteger, result[0].ReturnType.Category)
		assert.Equal(t, querier_dto.TypeCategoryText, result[1].ReturnType.Category)
	})

	t.Run("replaces existing signature with matching argument types", func(t *testing.T) {
		t.Parallel()

		existing := []*querier_dto.FunctionSignature{
			funcSig("myfunc", intType, funcArg("a", intType)),
		}

		replacement := funcSig("myfunc", textType, funcArg("a", intType))

		result := mergeOrAppendSignature(existing, replacement)

		require.Len(t, result, 1, "matching signature should be replaced, not appended")
		assert.Equal(t, querier_dto.TypeCategoryText, result[0].ReturnType.Category,
			"return type should reflect the replacement")
	})

	t.Run("appends to empty list", func(t *testing.T) {
		t.Parallel()

		var existing []*querier_dto.FunctionSignature
		newSig := funcSig("myfunc", intType, funcArg("a", intType))

		result := mergeOrAppendSignature(existing, newSig)

		require.Len(t, result, 1)
		assert.Equal(t, querier_dto.TypeCategoryInteger, result[0].ReturnType.Category)
	})

	t.Run("replaces correct overload among multiple", func(t *testing.T) {
		t.Parallel()

		existing := []*querier_dto.FunctionSignature{
			funcSig("myfunc", intType, funcArg("a", intType)),
			funcSig("myfunc", textType, funcArg("a", textType)),
			funcSig("myfunc", floatType, funcArg("a", floatType)),
		}

		replacement := funcSig("myfunc", boolType, funcArg("a", textType))

		result := mergeOrAppendSignature(existing, replacement)

		require.Len(t, result, 3, "should replace, not append")
		assert.Equal(t, querier_dto.TypeCategoryInteger, result[0].ReturnType.Category, "first overload unchanged")
		assert.Equal(t, querier_dto.TypeCategoryBoolean, result[1].ReturnType.Category, "second overload replaced")
		assert.Equal(t, querier_dto.TypeCategoryFloat, result[2].ReturnType.Category, "third overload unchanged")
	})

	t.Run("different arity is treated as different signature", func(t *testing.T) {
		t.Parallel()

		existing := []*querier_dto.FunctionSignature{
			funcSig("myfunc", intType, funcArg("a", intType)),
		}

		twoArgSig := funcSig("myfunc", intType, funcArg("a", intType), funcArg("b", intType))

		result := mergeOrAppendSignature(existing, twoArgSig)

		require.Len(t, result, 2, "different arity should be treated as a distinct signature")
	})
}

func TestScoreArgument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expected  querier_dto.SQLType
		actual    querier_dto.SQLType
		canCastFn func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool
		wantScore int
	}{
		{
			name:      "exact match by category and engine name",
			expected:  intType,
			actual:    intType,
			wantScore: 3,
		},
		{
			name:      "same category different engine name",
			expected:  sqlType("int4", querier_dto.TypeCategoryInteger),
			actual:    sqlType("int8", querier_dto.TypeCategoryInteger),
			wantScore: 2,
		},
		{
			name:     "implicit cast available",
			expected: floatType,
			actual:   intType,
			canCastFn: func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
				return from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat
			},
			wantScore: 1,
		},
		{
			name:     "no match returns zero",
			expected: boolType,
			actual:   textType,
			canCastFn: func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
				return false
			},
			wantScore: 0,
		},
		{
			name:      "unknown actual type scores 2",
			expected:  intType,
			actual:    unknownType,
			wantScore: 2,
		},
		{
			name:      "unknown expected type scores 2",
			expected:  unknownType,
			actual:    intType,
			wantScore: 2,
		},
		{
			name:      "both unknown scores 2 via actual-unknown branch",
			expected:  unknownType,
			actual:    unknownType,
			wantScore: 2,
		},
		{
			name:      "case-insensitive engine name comparison for exact match",
			expected:  sqlType("INTEGER", querier_dto.TypeCategoryInteger),
			actual:    sqlType("integer", querier_dto.TypeCategoryInteger),
			wantScore: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine := &mockEngine{
				canImplicitCastFn: tt.canCastFn,
			}
			resolver := &functionResolver{
				functions: make(map[string][]*querier_dto.FunctionSignature),
				engine:    engine,
			}

			score := resolver.scoreArgument(tt.expected, tt.actual)

			assert.Equal(t, tt.wantScore, score)
		})
	}
}
