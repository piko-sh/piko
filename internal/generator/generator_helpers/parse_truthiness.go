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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"piko.sh/piko/wdk/maths"
)

const (
	// opAdd is the addition arithmetic operator symbol.
	opAdd = "+"

	// opSubtract is the subtraction arithmetic operator symbol.
	opSubtract = "-"

	// opMultiply is the multiplication arithmetic operator symbol.
	opMultiply = "*"

	// opDivide is the division arithmetic operator symbol.
	opDivide = "/"

	// opModulo is the modulo arithmetic operator symbol.
	opModulo = "%"

	// opGreaterThan is the greater-than comparison operator symbol.
	opGreaterThan = ">"

	// opLessThan is the less-than comparison operator symbol.
	opLessThan = "<"

	// opGreaterEqual is the greater-than-or-equal comparison operator symbol.
	opGreaterEqual = ">="

	// opLessEqual is the less-than-or-equal comparison operator symbol.
	opLessEqual = "<="
)

// arithmeticComparable constrains types that support arithmetic and comparison
// operations with the same return-type pattern (Decimal, BigInt).
//
// Takes T which is the self-referential numeric type.
type arithmeticComparable[T any] interface {
	// Add returns the sum of the receiver and the operand.
	Add(T) T

	// Subtract returns the difference of the receiver minus the operand.
	Subtract(T) T

	// Multiply returns the product of the receiver and the operand.
	Multiply(T) T

	// Divide returns the quotient of the receiver divided by the operand.
	Divide(T) T

	// Modulus returns the remainder of the receiver divided by the operand.
	Modulus(T) T

	// CheckGreaterThan reports whether the receiver is greater than the operand.
	CheckGreaterThan(T) bool

	// CheckLessThan reports whether the receiver is less than the operand.
	CheckLessThan(T) bool

	// CheckEquals reports whether the receiver is equal to the operand.
	CheckEquals(T) bool
}

// EvaluateTruthiness determines the boolean truth value of any Go value.
//
// It follows JavaScript-like semantics where empty strings, zero values,
// nil, and "false" (case-insensitive) are considered falsy. This is used
// by the template engine to evaluate p-if and p-show directives.
//
// Takes value (any) which is the value to evaluate for truthiness.
//
// Returns bool which is true if the value is truthy, false otherwise.
func EvaluateTruthiness(value any) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return evaluateStringTruthiness(v)
	case int, int8, int16, int32, int64:
		return evaluateSignedIntTruthiness(value)
	case uint, uint8, uint16, uint32, uint64:
		return evaluateUnsignedIntTruthiness(value)
	case float32, float64:
		return evaluateFloatTruthiness(value)
	}

	return evaluateReflectTruthiness(value)
}

// ConvertToFloat64 converts any numeric or string value to a float64.
//
// Non-numeric types return 0.0. Booleans return 1.0 for true and 0.0 for
// false. This is used by the template engine for numeric comparisons and
// arithmetic.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted numeric value.
func ConvertToFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int8:
		return float64(typed)
	case int16:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint8:
		return float64(typed)
	case uint16:
		return float64(typed)
	case uint32:
		return float64(typed)
	case uint64:
		return float64(typed)
	case string:
		f, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return f
		}
		return 0.0
	case bool:
		if typed {
			return 1.0
		}
		return 0.0
	default:
		return 0.0
	}
}

// EvaluateLooseEquality compares two values for loose equality using JS-style
// type coercion. Float64 values are compared directly; all other types are
// compared by their string representation, enabling comparisons like 0 ~= "0"
// to return true.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns bool which is true when the values are loosely equal.
func EvaluateLooseEquality(a, b any) bool {
	floatA, aIsFloat := a.(float64)
	if aIsFloat {
		floatB, bIsFloat := b.(float64)
		if bIsFloat {
			return floatA == floatB
		}
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// EvaluateStrictEquality compares two values for strict equality using Go-style
// == comparison.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns bool which is true when the values are strictly equal, or false if
// the types do not match.
//
// Uses optimised type-specific comparisons for common types, falling back to
// reflect.DeepEqual for complex types. A typed nil (e.g., (*Image)(nil)) is
// considered equal to untyped nil. This enables template expressions like
// `p-if="props.FloorPlan != nil"` to work correctly when FloorPlan is a nil
// pointer.
func EvaluateStrictEquality(a, b any) bool {
	if b == nil {
		if a == nil {
			return true
		}
		return isNilableValue(a)
	}
	if a == nil {
		return isNilableValue(b)
	}

	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}

	switch av := a.(type) {
	case bool:
		return av == b.(bool)
	case int:
		return av == b.(int)
	case int8:
		return av == b.(int8)
	case int16:
		return av == b.(int16)
	case int32:
		return av == b.(int32)
	case int64:
		return av == b.(int64)
	case uint:
		return av == b.(uint)
	case uint8:
		return av == b.(uint8)
	case uint16:
		return av == b.(uint16)
	case uint32:
		return av == b.(uint32)
	case uint64:
		return av == b.(uint64)
	case float32:
		return av == b.(float32)
	case float64:
		return av == b.(float64)
	case string:
		return av == b.(string)
	}

	return reflect.DeepEqual(a, b)
}

// EvaluateOr returns the first truthy value, or the last value if none are
// truthy. This uses JavaScript-style || operator rules, which differ from Go's
// || that always returns a boolean.
//
// Takes left (any) which is the first value to check for truthiness.
// Takes right (any) which is the value to use if left is falsy.
//
// Returns any which is left if it is truthy, otherwise right.
func EvaluateOr(left, right any) any {
	if EvaluateTruthiness(left) {
		return left
	}
	return right
}

// EvaluateCoalesce implements JavaScript-like nullish coalescing (??) operator.
//
// It returns left if it is not nil, otherwise right. Unlike the || operator,
// it does not treat empty strings, zero, or false as nullish.
//
// Takes left (any) which is the primary value to check for nil.
// Takes right (any) which is the fallback value returned if left is nil.
//
// Returns any which is left if non-nil, otherwise right.
func EvaluateCoalesce(left, right any) any {
	if left == nil {
		return right
	}
	v := reflect.ValueOf(left)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return right
	}
	return left
}

// EvaluateBinary performs a binary operation on two values at runtime.
//
// It handles arithmetic (+, -, *, /, %) and comparison (>, <, >=, <=) operators
// for maths types (maths.Decimal, maths.BigInt, maths.Money) and falls back to
// float64 arithmetic for primitive numeric types.
//
// Takes left (any) which is the left operand.
// Takes operator (string) which is the operator ("+", "-", "*", "/", "%", ">",
// "<", ">=", "<=").
// Takes right (any) which is the right operand.
//
// Returns any which is the result of the operation. Arithmetic operators return
// the same maths type as the operands; comparison operators return bool. For
// unsupported types or operators, returns nil.
//
//nolint:revive // dispatch table
func EvaluateBinary(left any, operator string, right any) any {
	if result, ok := evaluateBinaryDecimal(left, right, operator); ok {
		return result
	}

	if result, ok := evaluateBinaryBigInt(left, right, operator); ok {
		return result
	}

	if result, ok := evaluateBinaryMoney(left, right, operator); ok {
		return result
	}

	return evaluateBinaryFloat64(left, right, operator)
}

// evaluateStringTruthiness checks if a string value is truthy.
//
// Takes v (string) which is the value to check.
//
// Returns bool which is true unless v is empty, "0", or "false" (case does not
// matter).
func evaluateStringTruthiness(v string) bool {
	if v == "" || v == "0" || strings.EqualFold(v, "false") {
		return false
	}
	return true
}

// evaluateSignedIntTruthiness checks whether a signed integer value is truthy.
//
// Takes value (any) which is the value to check.
//
// Returns bool which is true if the value is not zero, or if the value is not
// a signed integer type.
func evaluateSignedIntTruthiness(value any) bool {
	switch v := value.(type) {
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	}
	return true
}

// evaluateUnsignedIntTruthiness checks if an unsigned integer value is truthy.
//
// Takes value (any) which is the value to check.
//
// Returns bool which is true if the value is not zero, or true if the value is
// not an unsigned integer type.
func evaluateUnsignedIntTruthiness(value any) bool {
	switch v := value.(type) {
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	}
	return true
}

// evaluateFloatTruthiness checks whether a float value is truthy.
//
// Takes value (any) which is the value to check.
//
// Returns bool which is true if the value is a non-zero float, or true if the
// value is not a float type.
func evaluateFloatTruthiness(value any) bool {
	switch v := value.(type) {
	case float32:
		return v != 0.0
	case float64:
		return v != 0.0
	}
	return true
}

// evaluateReflectTruthiness uses reflection to check if a value is truthy.
//
// Takes value (any) which is the value to check using reflection.
//
// Returns bool which is true if the value is truthy. For pointer, interface,
// map, slice, function, and channel types, returns true when the value is not
// nil. For arrays and structs, always returns true.
func evaluateReflectTruthiness(value any) bool {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return !v.IsNil()
	case reflect.Array, reflect.Struct:
		return true
	}
	return true
}

// isNilableValue checks whether a value is a nil pointer, interface, map,
// slice, function, or channel. This is used by EvaluateStrictEquality to
// compare typed nils with untyped nil correctly.
//
// Takes value (any) which is the value to check.
//
// Returns bool which is true if value is a nilable type that is currently nil.
func isNilableValue(value any) bool {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

// evaluateBinaryDecimal handles binary operations when either operand is
// maths.Decimal.
//
// Takes left (any) which is the left-hand operand.
// Takes right (any) which is the right-hand operand.
// Takes operator (string) which is the arithmetic or comparison operator.
//
// Returns any which is the operation result.
// Returns bool which is true when the left operand is a Decimal.
func evaluateBinaryDecimal(left, right any, operator string) (any, bool) {
	ld, ok := coerceLeftDecimal(left)
	if !ok {
		return nil, false
	}
	rd := coerceRightDecimal(right)
	return dispatchArithmeticOperation(ld, rd, operator)
}

// coerceLeftDecimal attempts to extract a maths.Decimal from the left operand.
//
// Takes left (any) which is the value to coerce.
//
// Returns maths.Decimal which is the extracted value.
// Returns bool which is true when the left operand is a Decimal.
func coerceLeftDecimal(left any) (maths.Decimal, bool) {
	if ld, ok := left.(maths.Decimal); ok {
		return ld, true
	}
	if ldp, ok := left.(*maths.Decimal); ok && ldp != nil {
		return *ldp, true
	}
	return maths.Decimal{}, false
}

// coerceRightDecimal extracts a maths.Decimal from the right operand, falling
// back to CoerceToDecimal for non-Decimal types.
//
// Takes right (any) which is the value to coerce.
//
// Returns maths.Decimal which is the coerced value.
func coerceRightDecimal(right any) maths.Decimal {
	if rd, ok := right.(maths.Decimal); ok {
		return rd
	}
	if rdp, ok := right.(*maths.Decimal); ok && rdp != nil {
		return *rdp
	}
	return CoerceToDecimal(right)
}

// dispatchArithmeticOperation applies the given operator to two values satisfying
// arithmeticComparable.
//
// Takes left (T) which is the left operand.
// Takes right (T) which is the right operand.
// Takes operator (string) which is the arithmetic or comparison operator.
//
// Returns any which is the operation result.
// Returns bool which is true for supported operators.
func dispatchArithmeticOperation[T arithmeticComparable[T]](left, right T, operator string) (any, bool) {
	switch operator {
	case opAdd:
		return left.Add(right), true
	case opSubtract:
		return left.Subtract(right), true
	case opMultiply:
		return left.Multiply(right), true
	case opDivide:
		return left.Divide(right), true
	case opModulo:
		return left.Modulus(right), true
	case opGreaterThan:
		return left.CheckGreaterThan(right), true
	case opLessThan:
		return left.CheckLessThan(right), true
	case opGreaterEqual:
		return left.CheckGreaterThan(right) || left.CheckEquals(right), true
	case opLessEqual:
		return left.CheckLessThan(right) || left.CheckEquals(right), true
	default:
		return nil, false
	}
}

// evaluateBinaryBigInt handles binary operations when either operand is
// maths.BigInt.
//
// Takes left (any) which is the left-hand operand.
// Takes right (any) which is the right-hand operand.
// Takes operator (string) which is the arithmetic or comparison operator.
//
// Returns any which is the operation result.
// Returns bool which is true when the left operand is a BigInt.
func evaluateBinaryBigInt(left, right any, operator string) (any, bool) {
	lb, ok := coerceLeftBigInt(left)
	if !ok {
		return nil, false
	}
	rb := coerceRightBigInt(right)
	return dispatchArithmeticOperation(lb, rb, operator)
}

// coerceLeftBigInt attempts to extract a maths.BigInt from the left operand.
//
// Takes left (any) which is the value to coerce.
//
// Returns maths.BigInt which is the extracted value.
// Returns bool which is true when the left operand is a BigInt.
func coerceLeftBigInt(left any) (maths.BigInt, bool) {
	if lb, ok := left.(maths.BigInt); ok {
		return lb, true
	}
	if lbp, ok := left.(*maths.BigInt); ok && lbp != nil {
		return *lbp, true
	}
	return maths.BigInt{}, false
}

// coerceRightBigInt extracts a maths.BigInt from the right operand, falling
// back to CoerceToBigInt for non-BigInt types.
//
// Takes right (any) which is the value to coerce.
//
// Returns maths.BigInt which is the coerced value.
func coerceRightBigInt(right any) maths.BigInt {
	if rb, ok := right.(maths.BigInt); ok {
		return rb
	}
	if rbp, ok := right.(*maths.BigInt); ok && rbp != nil {
		return *rbp
	}
	return CoerceToBigInt(right)
}

// evaluateBinaryMoney handles binary operations when either operand is
// maths.Money.
//
// Takes left (any) which is the left-hand operand.
// Takes right (any) which is the right-hand operand.
// Takes operator (string) which is the arithmetic or comparison operator.
//
// Returns any which is the operation result.
// Returns bool which is true when the left operand is a Money value.
func evaluateBinaryMoney(left, right any, operator string) (any, bool) {
	lm, leftIsMoney := left.(maths.Money)
	if !leftIsMoney {
		if lmp, ok := left.(*maths.Money); ok && lmp != nil {
			lm = *lmp
			leftIsMoney = true
		}
	}
	if !leftIsMoney {
		return nil, false
	}

	rm, rightIsMoney := right.(maths.Money)
	if !rightIsMoney {
		if rmp, ok := right.(*maths.Money); ok && rmp != nil {
			rm = *rmp
			rightIsMoney = true
		}
	}

	switch operator {
	case "+", "-":
		return evaluateMoneyAddSub(lm, rm, right, rightIsMoney, operator)
	case opMultiply:
		return lm.Multiply(CoerceToDecimal(right)), true
	case opDivide:
		return lm.Divide(CoerceToDecimal(right)), true
	case opModulo:
		return lm.Modulus(CoerceToDecimal(right)), true
	case ">", "<", ">=", "<=":
		return evaluateMoneyComparison(lm, rm, rightIsMoney, operator)
	default:
		return nil, false
	}
}

// evaluateMoneyAddSub handles addition and subtraction for Money values,
// dispatching to Money-Money or Money-Decimal operations.
//
// Takes lm (maths.Money) which is the left-hand money operand.
// Takes rm (maths.Money) which is the right-hand money operand (valid only
// when rightIsMoney is true).
// Takes right (any) which is the original right operand for decimal coercion.
// Takes rightIsMoney (bool) which indicates whether the right operand is Money.
// Takes operator (string) which is "+" or "-".
//
// Returns any which is the operation result.
// Returns bool which is always true.
func evaluateMoneyAddSub(lm, rm maths.Money, right any, rightIsMoney bool, operator string) (any, bool) {
	if operator == "+" {
		if rightIsMoney {
			return lm.Add(rm), true
		}
		return lm.AddDecimal(CoerceToDecimal(right)), true
	}
	if rightIsMoney {
		return lm.Subtract(rm), true
	}
	return lm.SubtractDecimal(CoerceToDecimal(right)), true
}

// evaluateMoneyComparison handles comparison operators for Money values.
// Comparisons require both operands to be Money.
//
// Takes lm (maths.Money) which is the left-hand money operand.
// Takes rm (maths.Money) which is the right-hand money operand.
// Takes rightIsMoney (bool) which indicates whether the right operand is Money.
// Takes operator (string) which is the comparison operator.
//
// Returns any which is the comparison result as a bool, or nil.
// Returns bool which is true when the left operand is Money.
func evaluateMoneyComparison(lm, rm maths.Money, rightIsMoney bool, operator string) (any, bool) {
	if !rightIsMoney {
		return nil, false
	}
	switch operator {
	case opGreaterThan:
		return lm.CheckGreaterThan(rm), true
	case opLessThan:
		return lm.CheckLessThan(rm), true
	case opGreaterEqual:
		return lm.CheckGreaterThan(rm) || lm.CheckEquals(rm), true
	case opLessEqual:
		return lm.CheckLessThan(rm) || lm.CheckEquals(rm), true
	default:
		return nil, false
	}
}

// evaluateBinaryFloat64 handles binary operations by converting both operands
// to float64. This is the fallback for primitive numeric types.
//
// Takes left (any) which is the left-hand operand.
// Takes right (any) which is the right-hand operand.
// Takes operator (string) which is the arithmetic or comparison operator.
//
// Returns any which is the operation result as a float64 or bool.
func evaluateBinaryFloat64(left, right any, operator string) any {
	lf := ConvertToFloat64(left)
	rf := ConvertToFloat64(right)

	switch operator {
	case opAdd:
		return lf + rf
	case opSubtract:
		return lf - rf
	case opMultiply:
		return lf * rf
	case opDivide:
		if rf == 0.0 {
			return 0.0
		}
		return lf / rf
	case opModulo:
		if rf == 0.0 {
			return 0.0
		}
		return float64(int64(lf) % int64(rf))
	case opGreaterThan:
		return lf > rf
	case opLessThan:
		return lf < rf
	case opGreaterEqual:
		return lf >= rf
	case opLessEqual:
		return lf <= rf
	default:
		return nil
	}
}
