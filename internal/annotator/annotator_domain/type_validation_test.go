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
	goast "go/ast"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestValidateOrderingComparison_ValidIntComparison(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("int"))
	right := newSimpleTypeInfo(goast.NewIdent("int"))

	result := ValidateOrderingComparison(left, right, ast_domain.OpLt)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateOrderingComparison_InvalidStructComparison(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(&goast.SelectorExpr{
		X:   goast.NewIdent("time"),
		Sel: goast.NewIdent("Time"),
	})
	right := newSimpleTypeInfo(&goast.SelectorExpr{
		X:   goast.NewIdent("time"),
		Sel: goast.NewIdent("Time"),
	})

	result := ValidateOrderingComparison(left, right, ast_domain.OpLt)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "Invalid operation")
}

func TestValidateEqualityComparison_ValidStringComparison(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("string"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateEqualityComparison(ast_domain.OpEq, left, right, true)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateEqualityComparison_StrictMismatch(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("int"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateEqualityComparison(ast_domain.OpEq, left, right, true)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "strictly compare")
}

func TestValidateEqualityComparison_LooseNumericStringValid(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("int"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateEqualityComparison(ast_domain.OpLooseEq, left, right, false)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateEqualityComparison_LooseStructMismatch(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(&goast.SelectorExpr{
		X:   goast.NewIdent("time"),
		Sel: goast.NewIdent("Time"),
	})
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateEqualityComparison(ast_domain.OpLooseEq, left, right, false)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "cannot compare")
}

func TestValidateLogicalAndLeftOperand_ValidBool(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("bool"))

	result := ValidateLogicalAndLeftOperand(left)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateLogicalAndLeftOperand_InvalidString(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateLogicalAndLeftOperand(left)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "Left operand")
	assert.Contains(t, result.Message, "&&")
	assert.Contains(t, result.Message, "~")
}

func TestValidateLogicalAndRightOperand_ValidBool(t *testing.T) {
	t.Parallel()

	right := newSimpleTypeInfo(goast.NewIdent("bool"))

	result := ValidateLogicalAndRightOperand(right)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateLogicalAndRightOperand_InvalidInt(t *testing.T) {
	t.Parallel()

	right := newSimpleTypeInfo(goast.NewIdent("int"))

	result := ValidateLogicalAndRightOperand(right)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "Right operand")
	assert.Contains(t, result.Message, "&&")
}

func TestValidateOrOperator_ValidAssignable(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("string"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateOrOperator(left, right)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateOrOperator_ValidNilLeft(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("nil"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateOrOperator(left, right)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateOrOperator_InvalidIncompatibleTypes(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(&goast.SelectorExpr{
		X:   goast.NewIdent("time"),
		Sel: goast.NewIdent("Time"),
	})
	right := newSimpleTypeInfo(&goast.SelectorExpr{
		X:   goast.NewIdent("http"),
		Sel: goast.NewIdent("Request"),
	})

	result := ValidateOrOperator(left, right)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "||")
	assert.Contains(t, result.Message, "incompatible")
}

func TestValidateCoalesceOperator_ValidAssignable(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("string"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateCoalesceOperator(left, right)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateCoalesceOperator_InvalidIncompatible(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("int"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateCoalesceOperator(left, right)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "??")
	assert.Contains(t, result.Message, "incompatible")
}

func TestValidateBooleanCondition_ValidBool(t *testing.T) {
	t.Parallel()

	typeInfo := newSimpleTypeInfo(goast.NewIdent("bool"))

	result := ValidateBooleanCondition(typeInfo)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateBooleanCondition_ValidNil(t *testing.T) {
	t.Parallel()

	result := ValidateBooleanCondition(nil)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateBooleanCondition_InvalidString(t *testing.T) {
	t.Parallel()

	typeInfo := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateBooleanCondition(typeInfo)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "Condition must be a boolean")
	assert.Contains(t, result.Message, "~")
}

func TestValidateUnaryNot_ValidBool(t *testing.T) {
	t.Parallel()

	operand := newSimpleTypeInfo(goast.NewIdent("bool"))

	result := ValidateUnaryNot(operand)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateUnaryNot_InvalidString(t *testing.T) {
	t.Parallel()

	operand := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateUnaryNot(operand)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "!")
	assert.Contains(t, result.Message, "non-boolean")
}

func TestValidateUnaryNeg_ValidInt(t *testing.T) {
	t.Parallel()

	operand := newSimpleTypeInfo(goast.NewIdent("int"))

	result := ValidateUnaryNeg(operand)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateUnaryNeg_ValidFloat(t *testing.T) {
	t.Parallel()

	operand := newSimpleTypeInfo(goast.NewIdent("float64"))

	result := ValidateUnaryNeg(operand)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateUnaryNeg_InvalidString(t *testing.T) {
	t.Parallel()

	operand := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateUnaryNeg(operand)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "-")
	assert.Contains(t, result.Message, "non-arithmetic")
}

func TestValidateArithmeticOperator_ValidIntAddition(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("int"))
	right := newSimpleTypeInfo(goast.NewIdent("int"))

	result := ValidateArithmeticOperator(left, right, ast_domain.OpPlus)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateArithmeticOperator_ValidStringConcatenation(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("string"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateArithmeticOperator(left, right, ast_domain.OpPlus)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateArithmeticOperator_InvalidStringSubtraction(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("string"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateArithmeticOperator(left, right, ast_domain.OpMinus)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "not defined")
}

func TestValidateTernaryBranches_ValidSameType(t *testing.T) {
	t.Parallel()

	consequent := newSimpleTypeInfo(goast.NewIdent("string"))
	alternate := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateTernaryBranches(consequent, alternate)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateTernaryBranches_InvalidMismatch(t *testing.T) {
	t.Parallel()

	consequent := newSimpleTypeInfo(goast.NewIdent("string"))
	alternate := newSimpleTypeInfo(goast.NewIdent("int"))

	result := ValidateTernaryBranches(consequent, alternate)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "Ternary expression")
	assert.Contains(t, result.Message, "mismatched types")
}

func TestValidateMoneyAddSub_ValidMoneyMoney(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("Money"))
	right := newSimpleTypeInfo(goast.NewIdent("Money"))

	result := ValidateMoneyAddSub(left, right, true, true)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateMoneyAddSub_InvalidMoneyString(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("Money"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateMoneyAddSub(left, right, true, false)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "cannot add or subtract Money")
}

func TestValidateMoneyMulDiv_ValidMoneyByNumber(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("Money"))
	right := newSimpleTypeInfo(goast.NewIdent("int"))

	result := ValidateMoneyMulDiv(left, right, true, false, "money * 2")

	assert.True(t, result.Valid)
	assert.Empty(t, result.Message)
}

func TestValidateMoneyMulDiv_InvalidMoneyByMoney(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("Money"))
	right := newSimpleTypeInfo(goast.NewIdent("Money"))

	result := ValidateMoneyMulDiv(left, right, true, true, "money * money")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "Cannot multiply or divide Money by Money")
}

func TestValidateMoneyMulDiv_InvalidMoneyByString(t *testing.T) {
	t.Parallel()

	left := newSimpleTypeInfo(goast.NewIdent("Money"))
	right := newSimpleTypeInfo(goast.NewIdent("string"))

	result := ValidateMoneyMulDiv(left, right, true, false, "money * str")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "can only multiply or divide Money by a standard number")
}

func TestValidationResult_MessageContainsTypeNames(t *testing.T) {
	t.Parallel()

	timeType := &goast.SelectorExpr{X: goast.NewIdent("time"), Sel: goast.NewIdent("Time")}
	httpType := &goast.SelectorExpr{X: goast.NewIdent("http"), Sel: goast.NewIdent("Request")}

	testCases := []struct {
		name     string
		result   ValidationResult
		expected []string
	}{
		{
			name:     "ordering comparison shows types",
			result:   ValidateOrderingComparison(newSimpleTypeInfo(timeType), newSimpleTypeInfo(httpType), ast_domain.OpLt),
			expected: []string{"time.Time", "http.Request"},
		},
		{
			name:     "equality comparison shows types",
			result:   ValidateEqualityComparison(ast_domain.OpEq, newSimpleTypeInfo(timeType), newSimpleTypeInfo(httpType), true),
			expected: []string{"time.Time", "http.Request"},
		},
		{
			name:     "or operator shows types",
			result:   ValidateOrOperator(newSimpleTypeInfo(timeType), newSimpleTypeInfo(httpType)),
			expected: []string{"time.Time", "http.Request"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, tc.result.Valid)
			for _, typeName := range tc.expected {
				assert.True(t, strings.Contains(tc.result.Message, typeName), "expected message to contain %q, got %q", typeName, tc.result.Message)
			}
		})
	}
}
