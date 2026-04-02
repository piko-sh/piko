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

package ast_domain

// Provides runtime evaluation of expression trees with variable scopes for computing values from template expressions.
// Handles literals, identifiers, member access, operators, function calls, and composite expressions with type coercion and reflection support.

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/wdk/maths"
)

// EvaluateExpression evaluates an expression tree with a given variable scope,
// returning the computed value.
//
// Takes expression (Expression) which is the expression tree to evaluate.
// Takes scope (map[string]any) which provides variable bindings for the
// evaluation.
//
// Returns any which is the computed result, or nil if the expression is nil
// or cannot be evaluated.
func EvaluateExpression(expression Expression, scope map[string]any) any {
	if expression == nil {
		return nil
	}
	if result, ok := evaluatePrimitiveLiteral(expression); ok {
		return result
	}
	if result, ok := evaluateAccessExpression(expression, scope); ok {
		return result
	}
	if result, ok := evaluateOperatorExpression(expression, scope); ok {
		return result
	}
	if result, ok := evaluateCompositeExpression(expression, scope); ok {
		return result
	}
	return nil
}

// evaluatePrimitiveLiteral converts a primitive literal expression to its
// Go value.
//
// Takes expression (Expression) which is the expression node to evaluate.
//
// Returns any which is the converted Go value of the primitive literal.
// Returns bool which is true if the expression was a primitive literal type.
func evaluatePrimitiveLiteral(expression Expression) (any, bool) {
	switch node := expression.(type) {
	case *StringLiteral:
		return node.Value, true
	case *IntegerLiteral:
		return float64(node.Value), true
	case *FloatLiteral:
		return node.Value, true
	case *BooleanLiteral:
		return node.Value, true
	case *NilLiteral:
		return nil, true
	case *RuneLiteral:
		return node.Value, true
	case *DecimalLiteral:
		return maths.NewDecimalFromString(node.Value), true
	case *BigIntLiteral:
		return maths.NewBigIntFromString(node.Value), true
	case *DateTimeLiteral:
		return evaluateDateTimeLiteral(node), true
	case *DateLiteral:
		return evaluateDateLiteral(node), true
	case *TimeLiteral:
		return evaluateTimeLiteral(node), true
	case *DurationLiteral:
		return evaluateDurationLiteral(node), true
	default:
		return nil, false
	}
}

// evaluateAccessExpression handles identifier and member access expressions.
//
// Takes expression (Expression) which is the expression node to evaluate.
// Takes scope (map[string]any) which provides the variable bindings.
//
// Returns any which is the evaluated value.
// Returns bool which indicates whether the expression type was recognised.
func evaluateAccessExpression(expression Expression, scope map[string]any) (any, bool) {
	switch node := expression.(type) {
	case *Identifier:
		return evaluateIdentifier(node, scope), true
	case *MemberExpression:
		return evaluateMemberExpr(node, scope), true
	case *IndexExpression:
		return evaluateIndexExpr(node, scope), true
	case *CallExpression:
		return evaluateCallExpr(node, scope), true
	default:
		return nil, false
	}
}

// evaluateOperatorExpression handles unary and binary operator expressions.
//
// Takes expression (Expression) which is the expression node to evaluate.
// Takes scope (map[string]any) which provides variable bindings.
//
// Returns any which is the result of the operator expression.
// Returns bool which is true if the expression was an operator type.
func evaluateOperatorExpression(expression Expression, scope map[string]any) (any, bool) {
	switch node := expression.(type) {
	case *UnaryExpression:
		value := EvaluateExpression(node.Right, scope)
		return evaluateUnary(node.Operator, value), true
	case *BinaryExpression:
		leftVal := EvaluateExpression(node.Left, scope)
		rightVal := EvaluateExpression(node.Right, scope)
		return evaluateBinary(node.Operator, leftVal, rightVal), true
	default:
		return nil, false
	}
}

// evaluateCompositeExpression handles composite literal expressions such as
// arrays, objects, and ternary expressions.
//
// Takes expression (Expression) which is the expression to evaluate.
// Takes scope (map[string]any) which provides the variable bindings.
//
// Returns any which is the evaluated result.
// Returns bool which indicates whether the expression type was handled.
func evaluateCompositeExpression(expression Expression, scope map[string]any) (any, bool) {
	switch node := expression.(type) {
	case *ArrayLiteral:
		return evaluateArrayLiteral(node, scope), true
	case *ObjectLiteral:
		return evaluateObjectLiteral(node, scope), true
	case *TernaryExpression:
		return evaluateTernaryExpr(node, scope), true
	default:
		return nil, false
	}
}

// evaluateIdentifier resolves an identifier node to its value in the scope.
//
// Takes node (*Identifier) which is the identifier to resolve.
// Takes scope (map[string]any) which contains variable bindings.
//
// Returns any which is the resolved value. For the special "now" identifier,
// returns time.Now if not set in scope.
func evaluateIdentifier(node *Identifier, scope map[string]any) any {
	if node.Name == "now" {
		if value, ok := scope["now"]; ok {
			return value
		}
		return time.Now
	}
	return scope[node.Name]
}

// evaluateMemberExpr retrieves a property value from a base object.
//
// Takes node (*MemberExpression) which is the member expression to evaluate.
// Takes scope (map[string]any) which provides the variable bindings.
//
// Returns any which is the property value, or nil if the property cannot be
// accessed.
func evaluateMemberExpr(node *MemberExpression, scope map[string]any) any {
	base := EvaluateExpression(node.Base, scope)
	prop, ok := node.Property.(*Identifier)
	if !ok {
		return nil
	}
	if baseMap, isMap := base.(map[string]any); isMap {
		return baseMap[prop.Name]
	}
	return nil
}

// evaluateIndexExpr gets a value from a slice, array, string, or map using
// the given index.
//
// Takes node (*IndexExpression) which is the index expression to evaluate.
// Takes scope (map[string]any) which provides variable bindings.
//
// Returns any which is the value at the index, or nil if the index is out of
// bounds, the key is not found, or the types do not match.
func evaluateIndexExpr(node *IndexExpression, scope map[string]any) any {
	base := EvaluateExpression(node.Base, scope)
	index := EvaluateExpression(node.Index, scope)

	baseVal := reflect.ValueOf(base)
	if !baseVal.IsValid() {
		return nil
	}

	switch baseVal.Kind() {
	case reflect.Slice, reflect.Array, reflect.String:
		idxInt := int(toFloat(index))
		if idxInt < 0 || idxInt >= baseVal.Len() {
			return nil
		}
		return baseVal.Index(idxInt).Interface()
	case reflect.Map:
		indexVal := reflect.ValueOf(index)
		if !indexVal.IsValid() {
			return nil
		}
		keyType := baseVal.Type().Key()
		if !indexVal.Type().AssignableTo(keyType) {
			return nil
		}
		mapVal := baseVal.MapIndex(indexVal)
		if !mapVal.IsValid() {
			return nil
		}
		return mapVal.Interface()
	default:
		return nil
	}
}

// evaluateCallExpr calls a function expression with the given arguments.
//
// Takes node (*CallExpression) which contains the function and its arguments.
// Takes scope (map[string]any) which provides variable bindings for evaluation.
//
// Returns any which is the return value from the function, or nil if the
// callee is not a function or returns no values.
func evaluateCallExpr(node *CallExpression, scope map[string]any) any {
	callee := EvaluateExpression(node.Callee, scope)
	fnVal := reflect.ValueOf(callee)

	if fnVal.Kind() != reflect.Func {
		return nil
	}

	arguments := prepareCallArguments(fnVal, node.Args, scope)
	results := fnVal.Call(arguments)

	if len(results) > 0 {
		if len(results) == 2 && !results[1].IsNil() {
			return results[1].Interface()
		}
		return results[0].Interface()
	}
	return nil
}

// prepareCallArguments evaluates argument expressions and converts them to the
// types expected by the target function.
//
// Takes fnVal (reflect.Value) which is the function to prepare arguments for.
// Takes argExprs ([]Expression) which contains the argument expressions to
// evaluate.
// Takes scope (map[string]any) which provides variables for expression
// evaluation.
//
// Returns []reflect.Value which contains the prepared arguments ready for use
// with reflect.Value.Call.
func prepareCallArguments(fnVal reflect.Value, argExprs []Expression, scope map[string]any) []reflect.Value {
	fnType := fnVal.Type()
	numIn := fnType.NumIn()
	arguments := make([]reflect.Value, numIn)

	for i := range numIn {
		arguments[i] = reflect.Zero(fnType.In(i))
	}

	for i, argExpr := range argExprs {
		if i >= len(arguments) {
			break
		}
		argVal := EvaluateExpression(argExpr, scope)
		paramType := fnType.In(i)
		providedVal := reflect.ValueOf(argVal)

		if !providedVal.IsValid() {
			arguments[i] = reflect.Zero(paramType)
		} else if providedVal.Type().ConvertibleTo(paramType) {
			arguments[i] = providedVal.Convert(paramType)
		} else {
			arguments[i] = convertArgument(providedVal, paramType)
		}
	}
	return arguments
}

// convertArgument changes a value to match the expected parameter type.
//
// Takes providedVal (reflect.Value) which is the value to change.
// Takes paramType (reflect.Type) which is the target type.
//
// Returns reflect.Value which is the changed value, or the original value
// if no change is needed.
func convertArgument(providedVal reflect.Value, paramType reflect.Type) reflect.Value {
	switch paramType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if providedVal.Kind() == reflect.Float64 {
			return reflect.ValueOf(int64(providedVal.Float())).Convert(paramType)
		}
		return providedVal
	default:
		return providedVal
	}
}

// evaluateDateTimeLiteral parses a date-time literal node into a time value.
//
// Takes node (*DateTimeLiteral) which holds the RFC3339 formatted string.
//
// Returns any which is a time.Time value on success, or an error if parsing
// fails.
func evaluateDateTimeLiteral(node *DateTimeLiteral) any {
	t, err := time.Parse(time.RFC3339, node.Value)
	if err != nil {
		return fmt.Errorf("parsing date-time literal %q: %w", node.Value, err)
	}
	return t
}

// evaluateDateLiteral parses a date literal node and returns the time value.
//
// Takes node (*DateLiteral) which contains the date string to parse.
//
// Returns any which is a time.Time value on success, or an error if parsing
// fails.
func evaluateDateLiteral(node *DateLiteral) any {
	t, err := time.ParseInLocation("2006-01-02", node.Value, time.UTC)
	if err != nil {
		return fmt.Errorf("parsing date literal %q: %w", node.Value, err)
	}
	return t
}

// evaluateTimeLiteral parses a time literal node into a time value.
//
// Takes node (*TimeLiteral) which holds the time string to parse.
//
// Returns any which is a time.Time value or an error if parsing fails.
func evaluateTimeLiteral(node *TimeLiteral) any {
	t, err := time.ParseInLocation("15:04:05", node.Value, time.UTC)
	if err != nil {
		return fmt.Errorf("parsing time literal %q: %w", node.Value, err)
	}
	return t
}

// evaluateDurationLiteral parses a duration literal node and returns its value.
//
// Takes node (*DurationLiteral) which holds the duration string to parse.
//
// Returns any which is the parsed time.Duration, or an error if parsing fails.
func evaluateDurationLiteral(node *DurationLiteral) any {
	d, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("parsing duration literal %q: %w", node.Value, err)
	}
	return d
}

// evaluateArrayLiteral processes an array literal and returns its elements.
//
// Takes node (*ArrayLiteral) which contains the array elements to process.
// Takes scope (map[string]any) which provides variable bindings for evaluation.
//
// Returns any which is a slice of evaluated element values.
func evaluateArrayLiteral(node *ArrayLiteral, scope map[string]any) any {
	elements := make([]any, len(node.Elements))
	for i, elemExpr := range node.Elements {
		elements[i] = EvaluateExpression(elemExpr, scope)
	}
	return elements
}

// evaluateObjectLiteral builds a map from an object literal node.
//
// Takes node (*ObjectLiteral) which contains the key-value pairs to evaluate.
// Takes scope (map[string]any) which provides variable bindings for evaluation.
//
// Returns any which is the resulting map with evaluated values.
func evaluateObjectLiteral(node *ObjectLiteral, scope map[string]any) any {
	result := make(map[string]any, len(node.Pairs))
	for key, valueExpr := range node.Pairs {
		result[key] = EvaluateExpression(valueExpr, scope)
	}
	return result
}

// evaluateTernaryExpr checks a condition and returns one of two values.
//
// Takes node (*TernaryExpression) which holds the condition and both
// branch values.
// Takes scope (map[string]any) which provides variable values for lookup.
//
// Returns any which is the result from the true or false branch.
func evaluateTernaryExpr(node *TernaryExpression, scope map[string]any) any {
	conditionResult := EvaluateExpression(node.Condition, scope)
	if toBool(conditionResult) {
		return EvaluateExpression(node.Consequent, scope)
	}
	return EvaluateExpression(node.Alternate, scope)
}

// evaluateUnary applies a unary operator to a value and returns the result.
//
// Takes op (UnaryOp) which specifies the unary operation to perform.
// Takes value (any) which is the value to apply the operation to.
//
// Returns any which is the result of the operation, or nil if the operator is
// not known.
func evaluateUnary(op UnaryOp, value any) any {
	if d, ok := value.(maths.Decimal); ok {
		switch op {
		case OpNeg:
			return d.Negate()
		case OpNot:
			return !toBool(d)
		case OpTruthy:
			return toBool(d)
		}
	}

	switch op {
	case OpNot:
		return !toBool(value)
	case OpNeg:
		return -toFloat(value)
	case OpTruthy:
		return toBool(value)
	default:
		return nil
	}
}

// evaluateBinary applies a binary operation to two operands.
//
// It first tries handlers for time, decimal, and big integer types. If none
// of these handle the operation, it falls back to standard evaluation.
//
// Takes op (BinaryOp) which specifies the operation to perform.
// Takes left (any) which is the left operand.
// Takes right (any) which is the right operand.
//
// Returns any which is the result of the operation.
func evaluateBinary(op BinaryOp, left any, right any) any {
	if result, handled := tryEvaluateTimeOperation(op, left, right); handled {
		return result
	}
	if result, handled := tryEvaluateDecimalOperation(op, left, right); handled {
		return result
	}
	if result, handled := tryEvaluateBigIntOperation(op, left, right); handled {
		return result
	}
	return evaluateStandardBinary(op, left, right)
}

// tryEvaluateTimeOperation attempts to evaluate a binary operation on time
// and duration values.
//
// Takes op (BinaryOp) which specifies the operation to perform.
// Takes left (any) which is the left operand, expected to be time.Time or
// time.Duration.
// Takes right (any) which is the right operand, expected to be time.Time or
// time.Duration.
//
// Returns any which is the result of the operation if successful.
// Returns bool which indicates whether the operation was valid.
func tryEvaluateTimeOperation(op BinaryOp, left any, right any) (any, bool) {
	leftTime, isLeftTime := left.(time.Time)
	rightTime, isRightTime := right.(time.Time)
	leftDuration, isLeftDuration := left.(time.Duration)
	rightDuration, isRightDuration := right.(time.Duration)

	if isLeftTime && isRightDuration {
		return evaluateTimeWithDuration(op, leftTime, rightDuration)
	}
	if isLeftDuration && isRightTime {
		if op == OpPlus {
			return rightTime.Add(leftDuration), true
		}
		return nil, false
	}
	if isLeftDuration && isRightDuration {
		return evaluateDurationBinary(op, leftDuration, rightDuration)
	}
	if isLeftTime && isRightTime {
		return evaluateTimeBinary(op, leftTime, rightTime)
	}
	return nil, false
}

// evaluateTimeWithDuration adds or subtracts a duration from a time value.
//
// Takes op (BinaryOp) which specifies the operation to perform.
// Takes t (time.Time) which is the base time value.
// Takes d (time.Duration) which is the duration to add or subtract.
//
// Returns any which is the resulting time, or nil if the operation is not
// supported.
// Returns bool which is true if the operation was successful.
func evaluateTimeWithDuration(op BinaryOp, t time.Time, d time.Duration) (any, bool) {
	if op == OpPlus {
		return t.Add(d), true
	}
	if op == OpMinus {
		return t.Add(-d), true
	}
	return nil, false
}

// evaluateDurationBinary performs a binary operation on two duration values.
//
// Takes op (BinaryOp) which specifies the operation to perform.
// Takes left (time.Duration) which is the left operand.
// Takes right (time.Duration) which is the right operand.
//
// Returns any which is the result of the operation, or an error if the
// operator is not supported.
// Returns bool which shows whether the evaluation ran.
func evaluateDurationBinary(op BinaryOp, left time.Duration, right time.Duration) (any, bool) {
	switch op {
	case OpPlus:
		return left + right, true
	case OpMinus:
		return left - right, true
	case OpEq, OpLooseEq:
		return left == right, true
	case OpNe, OpLooseNe:
		return left != right, true
	case OpGt:
		return left > right, true
	case OpLt:
		return left < right, true
	case OpGe:
		return left >= right, true
	case OpLe:
		return left <= right, true
	default:
		return fmt.Errorf("operator %s not supported for time.Duration", op), true
	}
}

// evaluateTimeBinary performs a binary operation on two time values.
//
// Takes op (BinaryOp) which specifies the comparison or arithmetic operation.
// Takes left (time.Time) which is the first time value.
// Takes right (time.Time) which is the second time value.
//
// Returns any which holds the result or an error if the operation is not
// supported.
// Returns bool which shows whether the operation was handled.
func evaluateTimeBinary(op BinaryOp, left time.Time, right time.Time) (any, bool) {
	switch op {
	case OpMinus:
		return left.Sub(right), true
	case OpEq, OpLooseEq:
		return left.Equal(right), true
	case OpNe, OpLooseNe:
		return !left.Equal(right), true
	case OpGt:
		return left.After(right), true
	case OpLt:
		return left.Before(right), true
	case OpGe:
		return left.After(right) || left.Equal(right), true
	case OpLe:
		return left.Before(right) || left.Equal(right), true
	default:
		return fmt.Errorf("operator %s not supported for time.Time", op), true
	}
}

// tryEvaluateDecimalOperation tries a binary operation using decimal values.
//
// Takes op (BinaryOp) which specifies the operation to perform.
// Takes left (any) which is the left operand, converted to decimal if needed.
// Takes right (any) which is the right operand, converted to decimal if needed.
//
// Returns any which is the result of the operation.
// Returns bool which is false when neither operand is a decimal.
func tryEvaluateDecimalOperation(op BinaryOp, left any, right any) (any, bool) {
	leftDec, isLeftDec := left.(maths.Decimal)
	rightDec, isRightDec := right.(maths.Decimal)

	if !isLeftDec && !isRightDec {
		return nil, false
	}

	if !isLeftDec {
		leftDec = maths.NewDecimalFromFloat(toFloat(left))
	}
	if !isRightDec {
		rightDec = maths.NewDecimalFromFloat(toFloat(right))
	}

	return evaluateDecimalBinary(op, leftDec, rightDec), true
}

// evaluateDecimalBinary runs a binary operation on two decimal values.
//
// Takes op (BinaryOp) which specifies the operation to run.
// Takes left (maths.Decimal) which is the left-hand value.
// Takes right (maths.Decimal) which is the right-hand value.
//
// Returns any which is the result of the operation. This is a decimal for
// maths operations, or a boolean for comparisons.
func evaluateDecimalBinary(op BinaryOp, left maths.Decimal, right maths.Decimal) any {
	switch op {
	case OpPlus:
		return left.Add(right)
	case OpMinus:
		return left.Subtract(right)
	case OpMul:
		return left.Multiply(right)
	case OpDiv:
		return left.Divide(right)
	case OpEq, OpLooseEq:
		equal, _ := left.Equals(right)
		return equal
	case OpNe, OpLooseNe:
		equal, _ := left.Equals(right)
		return !equal
	case OpGt:
		greaterThan, _ := left.GreaterThan(right)
		return greaterThan
	case OpLt:
		lessThan, _ := left.LessThan(right)
		return lessThan
	case OpGe:
		greaterThan, _ := left.GreaterThan(right)
		equal, _ := left.Equals(right)
		return greaterThan || equal
	case OpLe:
		lessThan, _ := left.LessThan(right)
		equal, _ := left.Equals(right)
		return lessThan || equal
	default:
		return maths.ZeroDecimalWithError(fmt.Errorf("operator %s not supported for decimals", op))
	}
}

// tryEvaluateBigIntOperation tries to run a binary operation when at least
// one operand is a BigInt.
//
// Takes op (BinaryOp) which specifies the binary operation to perform.
// Takes left (any) which is the left operand.
// Takes right (any) which is the right operand.
//
// Returns any which is the result of the operation, or nil if not applicable.
// Returns bool which indicates whether the operation was performed.
func tryEvaluateBigIntOperation(op BinaryOp, left any, right any) (any, bool) {
	_, isLeftBigInt := left.(maths.BigInt)
	_, isRightBigInt := right.(maths.BigInt)

	if !isLeftBigInt && !isRightBigInt {
		return nil, false
	}

	l := promoteToBigInt(left)
	r := promoteToBigInt(right)

	return evaluateBigIntBinary(op, l, r), true
}

// promoteToBigInt converts a value to a BigInt.
//
// If the value is already a BigInt, it returns the value unchanged. Otherwise,
// the value is converted to a float and then to a BigInt.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.BigInt which is the converted BigInt value.
func promoteToBigInt(value any) maths.BigInt {
	if bigIntVal, ok := value.(maths.BigInt); ok {
		return bigIntVal
	}
	return maths.NewBigIntFromInt(int64(toFloat(value)))
}

// evaluateBigIntBinary applies a binary operator to two BigInt values.
//
// Takes op (BinaryOp) which specifies the operation to perform.
// Takes left (maths.BigInt) which is the left operand.
// Takes right (maths.BigInt) which is the right operand.
//
// Returns any which is the result of the operation, either a BigInt for
// arithmetic operations or a bool for comparison operations.
func evaluateBigIntBinary(op BinaryOp, left maths.BigInt, right maths.BigInt) any {
	switch op {
	case OpPlus:
		return left.Add(right)
	case OpMinus:
		return left.Subtract(right)
	case OpMul:
		return left.Multiply(right)
	case OpDiv:
		return left.Divide(right)
	case OpMod:
		return left.Remainder(right)
	case OpEq, OpLooseEq:
		equal, _ := left.Equals(right)
		return equal
	case OpNe, OpLooseNe:
		equal, _ := left.Equals(right)
		return !equal
	case OpGt:
		cmp, _ := left.Cmp(right)
		return cmp > 0
	case OpLt:
		cmp, _ := left.Cmp(right)
		return cmp < 0
	case OpGe:
		cmp, _ := left.Cmp(right)
		return cmp >= 0
	case OpLe:
		cmp, _ := left.Cmp(right)
		return cmp <= 0
	default:
		return maths.ZeroBigIntWithError(fmt.Errorf("operator %s not supported for bigints", op))
	}
}

// evaluateStandardBinary works out the result of a binary operation on two
// values. It tries logical, equality, comparison, and arithmetic operations
// in turn until one handles the operation.
//
// Takes op (BinaryOp) which specifies the operation to perform.
// Takes left (any) which is the left operand.
// Takes right (any) which is the right operand.
//
// Returns any which is the result of the binary operation.
func evaluateStandardBinary(op BinaryOp, left any, right any) any {
	if result, handled := evaluateLogicalOp(op, left, right); handled {
		return result
	}
	if result, handled := evaluateEqualityOp(op, left, right); handled {
		return result
	}
	if result, handled := evaluateComparisonOp(op, left, right); handled {
		return result
	}
	return evaluateArithmeticOp(op, left, right)
}

// evaluateLogicalOp performs a logical binary operation on two values.
//
// Takes op (BinaryOp) which specifies the logical operation to perform.
// Takes left (any) which is the left operand.
// Takes right (any) which is the right operand.
//
// Returns any which is the result of the logical operation.
// Returns bool which is true when the operation succeeds, false for unknown
// operators.
func evaluateLogicalOp(op BinaryOp, left any, right any) (any, bool) {
	switch op {
	case OpCoalesce:
		if left == nil {
			return right, true
		}
		return left, true
	case OpAnd:
		return toBool(left) && toBool(right), true
	case OpOr:
		return toBool(left) || toBool(right), true
	default:
		return nil, false
	}
}

// evaluateEqualityOp compares two values using the given equality operator.
//
// Takes op (BinaryOp) which specifies the type of equality check to perform.
// Takes left (any) which is the first value to compare.
// Takes right (any) which is the second value to compare.
//
// Returns any which is the boolean result of the comparison, or nil if the
// operator is not known.
// Returns bool which is true if the operator was handled.
func evaluateEqualityOp(op BinaryOp, left any, right any) (any, bool) {
	switch op {
	case OpEq:
		return isStrictlyEqual(left, right), true
	case OpNe:
		return !isStrictlyEqual(left, right), true
	case OpLooseEq:
		return isLooselyEqual(left, right), true
	case OpLooseNe:
		return !isLooselyEqual(left, right), true
	default:
		return nil, false
	}
}

// evaluateComparisonOp compares two values using the given comparison operator.
//
// Takes op (BinaryOp) which specifies the comparison to perform.
// Takes left (any) which is the left-hand value.
// Takes right (any) which is the right-hand value.
//
// Returns any which is the boolean result of the comparison, or nil if the
// operator is not a comparison operator.
// Returns bool which is true when op is a valid comparison operator.
func evaluateComparisonOp(op BinaryOp, left any, right any) (any, bool) {
	switch op {
	case OpGt:
		return toFloat(left) > toFloat(right), true
	case OpLt:
		return toFloat(left) < toFloat(right), true
	case OpGe:
		return toFloat(left) >= toFloat(right), true
	case OpLe:
		return toFloat(left) <= toFloat(right), true
	default:
		return nil, false
	}
}

// evaluateArithmeticOp carries out the given arithmetic operation on two values.
//
// Takes op (BinaryOp) which specifies the operation to carry out.
// Takes left (any) which is the left operand.
// Takes right (any) which is the right operand.
//
// Returns any which is the result, or nil for unknown operations.
func evaluateArithmeticOp(op BinaryOp, left any, right any) any {
	switch op {
	case OpPlus:
		return evaluatePlus(left, right)
	case OpMinus:
		return toFloat(left) - toFloat(right)
	case OpMul:
		return toFloat(left) * toFloat(right)
	case OpDiv:
		return evaluateDivide(left, right)
	case OpMod:
		return evaluateModulo(left, right)
	default:
		return nil
	}
}

// evaluatePlus adds two values together, handling both strings and numbers.
//
// When either value is a string, joins both values as strings. Otherwise,
// adds the values as numbers.
//
// Takes left (any) which is the first value to add.
// Takes right (any) which is the second value to add.
//
// Returns any which is the joined string or the numeric sum.
func evaluatePlus(left any, right any) any {
	lString, lNum := tryStringOrFloat(left)
	rString, rNum := tryStringOrFloat(right)
	if lString != nil || rString != nil {
		return toString(left) + toString(right)
	}
	return lNum + rNum
}

// evaluateDivide divides two values after converting them to floats.
//
// When the right value is zero, returns 0.0 to avoid division by zero.
//
// Takes left (any) which is the dividend.
// Takes right (any) which is the divisor.
//
// Returns any which is the result as float64, or 0.0 if the divisor is zero.
func evaluateDivide(left any, right any) any {
	den := toFloat(right)
	if den == 0 {
		return 0.0
	}
	return toFloat(left) / den
}

// evaluateModulo computes the modulo of two values as integers.
//
// Takes left (any) which is the dividend value.
// Takes right (any) which is the divisor value.
//
// Returns any which is the remainder as a float64, or 0.0 if the divisor is
// zero.
func evaluateModulo(left any, right any) any {
	den := toFloat(right)
	denInt := int64(den)
	if denInt == 0 {
		return 0.0
	}
	return float64(int64(toFloat(left)) % denInt)
}

// toBool converts any value to a boolean using truthiness rules.
//
// When value is nil, returns false. For bool, returns the value directly.
// For string, returns false if empty, "0", or "false" (case-insensitive).
// For numeric types, returns false if zero. For time.Time, returns false
// if the time is the zero value. All other types return true.
//
// Takes value (any) which is the value to convert.
//
// Returns bool which is the result of the truthiness check.
func toBool(value any) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "0" && !strings.EqualFold(v, "false")
	case float64:
		return v != 0
	case rune:
		return v != 0
	case maths.BigInt:
		isZero, err := v.IsZero()
		return err == nil && !isZero
	case maths.Decimal:
		isZero, err := v.IsZero()
		return err == nil && !isZero
	case time.Time:
		return !v.IsZero()
	default:
		return true
	}
}

// toFloat converts a value of any type to a float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value, or 0 if value is nil or not a
// supported type.
func toFloat(value any) float64 {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case rune:
		return float64(v)
	case string:
		return stringToFloat(v)
	case bool:
		return boolToFloat(v)
	case maths.Decimal:
		return decimalToFloat(v)
	case maths.BigInt:
		return bigIntToFloat(v)
	default:
		return 0
	}
}

// stringToFloat parses a string as a float64, returning zero on failure.
//
// Takes s (string) which is the value to parse.
//
// Returns float64 which is the parsed value, or zero if parsing fails.
func stringToFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}

// boolToFloat converts a boolean value to a float64.
//
// Takes b (bool) which is the value to convert.
//
// Returns float64 which is 1.0 for true and 0.0 for false.
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// decimalToFloat converts a decimal value to a float64.
//
// Takes d (maths.Decimal) which is the value to convert.
//
// Returns float64 which is the converted value, or 0 if conversion fails.
func decimalToFloat(d maths.Decimal) float64 {
	f, err := d.Float64()
	if err != nil {
		return 0
	}
	return f
}

// bigIntToFloat converts a big integer to a float64 value.
//
// Takes bi (maths.BigInt) which is the value to convert.
//
// Returns float64 which is the converted value, or 0 if conversion fails.
func bigIntToFloat(bi maths.BigInt) float64 {
	i, err := bi.Int64()
	if err != nil {
		return 0
	}
	return float64(i)
}

// tryStringOrFloat checks if a value is a string or converts it to a float.
//
// Takes value (any) which is the value to check and convert.
//
// Returns *string which points to the string if value is a string, or nil.
// Returns float64 which is the converted number if value is not a string.
func tryStringOrFloat(value any) (*string, float64) {
	if s, ok := value.(string); ok {
		return &s, 0
	}
	return nil, toFloat(value)
}

// isLooselyEqual compares two values for equality with type coercion.
//
// When either value is nil, returns true only if both are nil. When both
// values are numeric-like, compares them as floats. Otherwise, uses
// reflect.DeepEqual for comparison.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns bool which indicates whether the values are loosely equal.
func isLooselyEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}

	if isNumericLike(a) && isNumericLike(b) {
		return toFloat(a) == toFloat(b)
	}

	if aStr, ok := a.(string); ok {
		if bStr, ok := b.(string); ok {
			return aStr == bStr
		}
	}

	return reflect.DeepEqual(a, b)
}

// isStrictlyEqual compares two values for strict equality with type checking.
//
// When either value is nil, returns true only if both are nil. For Decimal,
// BigInt, and Time types, uses their native equality methods. For all other
// types, requires matching types and uses reflect.DeepEqual.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns bool which is true if the values are strictly equal.
func isStrictlyEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	if matched, equal := tryStrictlyEqualSpecialTypes(a, b); matched {
		return equal
	}
	if matched, equal := tryStrictlyEqualPrimitives(a, b); matched {
		return equal
	}

	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}

	return reflect.DeepEqual(a, b)
}

// tryStrictlyEqualSpecialTypes attempts strict equality for Decimal, BigInt,
// and time.Time values.
//
// Takes a (any) which is the left operand.
// Takes b (any) which is the right operand.
//
// Returns bool which indicates whether both operands share the same special type.
// Returns bool which indicates whether the operands are equal.
func tryStrictlyEqualSpecialTypes(a, b any) (matched bool, equal bool) {
	if decA, ok := a.(maths.Decimal); ok {
		if decB, ok := b.(maths.Decimal); ok {
			eq, _ := decA.Equals(decB)
			return true, eq
		}
		return true, false
	}
	if bigA, ok := a.(maths.BigInt); ok {
		if bigB, ok := b.(maths.BigInt); ok {
			eq, _ := bigA.Equals(bigB)
			return true, eq
		}
		return true, false
	}
	if timeA, ok := a.(time.Time); ok {
		if timeB, ok := b.(time.Time); ok {
			return true, timeA.Equal(timeB)
		}
		return true, false
	}
	return false, false
}

// tryStrictlyEqualPrimitives attempts strict equality for Go primitive types
// (string, bool, int, int64, float64).
//
// Takes a (any) which is the left operand.
// Takes b (any) which is the right operand.
//
// Returns bool which indicates whether both operands share the same primitive type.
// Returns bool which indicates whether the operands are equal.
func tryStrictlyEqualPrimitives(a, b any) (matched bool, equal bool) {
	if aStr, ok := a.(string); ok {
		if bStr, ok := b.(string); ok {
			return true, aStr == bStr
		}
		return true, false
	}
	if aBool, ok := a.(bool); ok {
		if bBool, ok := b.(bool); ok {
			return true, aBool == bBool
		}
		return true, false
	}
	if aInt, ok := a.(int); ok {
		if bInt, ok := b.(int); ok {
			return true, aInt == bInt
		}
		return true, false
	}
	if aInt64, ok := a.(int64); ok {
		if bInt64, ok := b.(int64); ok {
			return true, aInt64 == bInt64
		}
		return true, false
	}
	if aFloat, ok := a.(float64); ok {
		if bFloat, ok := b.(float64); ok {
			return true, aFloat == bFloat
		}
		return true, false
	}
	return false, false
}

// isNumericLike reports whether the given value is numeric or can be parsed
// as a number.
//
// Takes v (any) which is the value to check.
//
// Returns bool which is true if v is a numeric type, a boolean, a Decimal, a
// BigInt, or a string that can be parsed as a float.
func isNumericLike(v any) bool {
	switch value := v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool:
		return true
	case string:
		_, err := strconv.ParseFloat(value, 64)
		return err == nil
	case maths.Decimal:
		return true
	case maths.BigInt:
		return true
	default:
		return false
	}
}
