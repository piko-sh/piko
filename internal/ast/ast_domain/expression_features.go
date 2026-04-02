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

// Defines feature flags for controlling which expression features are enabled in different parsing contexts.
// Provides bitmask constants for identifiers, operators, literals, and advanced features with predefined sets for paths, i18n, and compilation.

import "fmt"

// ExpressionFeature is a bitmask that shows which expression features are
// allowed in different parts of the system. It implements fmt.Stringer.
type ExpressionFeature uint32

const (
	// FeatureIdentifier allows simple identifiers (for example, "user").
	FeatureIdentifier ExpressionFeature = 1 << iota

	// FeatureMemberExpr allows member access (e.g. "user.name").
	FeatureMemberExpr

	// FeatureIndexExpr allows index access (e.g., "items[0]").
	FeatureIndexExpr

	// FeatureLiteralIndex requires indices to be literals (int or string).
	// When set, dynamic index expressions like "items[i]" are forbidden.
	FeatureLiteralIndex

	// FeatureBinaryExpression allows binary operators (e.g., "a + b", "x == y").
	FeatureBinaryExpression

	// FeatureUnaryExpression allows unary operators (e.g., "!x", "-y").
	FeatureUnaryExpression

	// FeatureCallExpression allows function calls (e.g., "format(x)").
	FeatureCallExpression

	// FeatureTernaryExpression allows ternary expressions (e.g., "a ? b : c").
	FeatureTernaryExpression

	// FeatureTemplateLiteral allows template literals (e.g., "`hello ${x}`").
	FeatureTemplateLiteral

	// FeatureForInExpr allows for-in expressions (e.g., "item in items").
	FeatureForInExpr

	// FeatureObjectLiteral allows object literals (e.g., "{a: 1}").
	FeatureObjectLiteral

	// FeatureArrayLiteral allows array literals (e.g., "[1, 2, 3]").
	FeatureArrayLiteral

	// FeatureLiterals allows all literal types (string, int, float, etc.).
	FeatureLiterals

	// FeatureLinkedMessage allows the @ operator for i18n linked messages.
	FeatureLinkedMessage

	// FeatureOptionalChaining allows ?. and ?.[ operators.
	FeatureOptionalChaining

	// FeatureNullishCoalescing allows the ?? operator.
	FeatureNullishCoalescing
)

const (
	// FeaturesNone is the default value that turns off all expression features.
	FeaturesNone ExpressionFeature = 0

	// FeaturesPath allows only path expressions: Identifier, MemberExpression,
	// IndexExpression with literals. Used by Binder for form field paths.
	FeaturesPath = FeatureIdentifier | FeatureMemberExpr | FeatureIndexExpr | FeatureLiteralIndex | FeatureLiterals

	// FeaturesI18n enables path expressions and the linked message operator.
	// Used by the i18n system for translation key references.
	FeaturesI18n = FeaturesPath | FeatureLinkedMessage

	// FeaturesCompiler is the feature set for the template compiler. It allows all
	// features except linked messages.
	FeaturesCompiler = FeatureIdentifier | FeatureMemberExpr | FeatureIndexExpr |
		FeatureBinaryExpression | FeatureUnaryExpression | FeatureCallExpression | FeatureTernaryExpression |
		FeatureTemplateLiteral | FeatureForInExpr | FeatureObjectLiteral |
		FeatureArrayLiteral | FeatureLiterals | FeatureOptionalChaining | FeatureNullishCoalescing

	// FeaturesAll enables all expression features.
	FeaturesAll = FeaturesCompiler | FeatureLinkedMessage
)

// Has checks if the feature set includes the given feature.
//
// Takes feature (ExpressionFeature) which is the feature to check for.
//
// Returns bool which is true if the feature is present in the set.
func (f ExpressionFeature) Has(feature ExpressionFeature) bool {
	return f&feature == feature
}

// String returns a readable name for the feature for use in error messages.
//
// Returns string which is the display name shown in diagnostics.
func (f ExpressionFeature) String() string {
	switch f {
	case FeatureIdentifier:
		return "identifiers"
	case FeatureMemberExpr:
		return "member access (.)"
	case FeatureIndexExpr:
		return "index access ([])"
	case FeatureLiteralIndex:
		return "literal indices"
	case FeatureBinaryExpression:
		return "binary operators"
	case FeatureUnaryExpression:
		return "unary operators"
	case FeatureCallExpression:
		return "function calls"
	case FeatureTernaryExpression:
		return "ternary expressions (?:)"
	case FeatureTemplateLiteral:
		return "template literals"
	case FeatureForInExpr:
		return "for-in expressions"
	case FeatureObjectLiteral:
		return "object literals"
	case FeatureArrayLiteral:
		return "array literals"
	case FeatureLiterals:
		return "literals"
	case FeatureLinkedMessage:
		return "linked messages (@)"
	case FeatureOptionalChaining:
		return "optional chaining (?.)"
	case FeatureNullishCoalescing:
		return "nullish coalescing (??)"
	default:
		return "expression features"
	}
}

// ValidateExpressionFeatures checks that an expression tree only uses allowed
// features.
//
// Takes expression (Expression) which is the expression tree to check.
// Takes allowed (ExpressionFeature) which is a bitmask of allowed features.
// Takes sourcePath (string) which is the source file path for error messages.
//
// Returns []*Diagnostic which holds Error-level results for each feature that
// is not allowed, or nil if all features are allowed.
func ValidateExpressionFeatures(expression Expression, allowed ExpressionFeature, sourcePath string) []*Diagnostic {
	if expression == nil {
		return nil
	}

	var diagnostics []*Diagnostic

	VisitExpression(expression, func(e Expression) bool {
		if diagnostic := validateSingleNode(e, allowed, sourcePath); diagnostic != nil {
			diagnostics = append(diagnostics, diagnostic)
		}
		return true
	})

	return diagnostics
}

// ValidatePathExpression checks that an expression is a valid path expression.
// This is a wrapper for common validation.
//
// Takes expression (Expression) which is the expression to check.
// Takes sourcePath (string) which identifies the source for diagnostics.
//
// Returns []*Diagnostic which contains any validation errors found.
func ValidatePathExpression(expression Expression, sourcePath string) []*Diagnostic {
	return ValidateExpressionFeatures(expression, FeaturesPath, sourcePath)
}

// IsPathExpression reports whether the given expression is valid for use as a
// path expression. This is a helper that returns a boolean instead of
// diagnostics.
//
// Takes expression (Expression) which is the expression to check.
//
// Returns bool which is true if the expression is valid for use as a path.
func IsPathExpression(expression Expression) bool {
	return len(ValidateExpressionFeatures(expression, FeaturesPath, "")) == 0
}

// validateSingleNode checks whether a single expression node is allowed.
//
// Takes expression (Expression) which is the expression node to check.
// Takes allowed (ExpressionFeature) which specifies the features that are
// allowed.
// Takes sourcePath (string) which is the file path for error messages.
//
// Returns *Diagnostic which describes the problem, or nil if the node is
// allowed.
func validateSingleNode(expression Expression, allowed ExpressionFeature, sourcePath string) *Diagnostic {
	if index, ok := expression.(*IndexExpression); ok {
		if allowed.Has(FeatureLiteralIndex) && !isLiteralIndex(index.Index) {
			message := "Dynamic index access not allowed in this context"
			d := NewDiagnosticForExpression(Error, message, expression, expression.GetRelativeLocation(), sourcePath)
			d.Code = CodeDisallowedFeature
			return d
		}
	}

	required, description := requiredFeatureForExpr(expression, allowed)
	if required == 0 {
		return nil
	}

	if allowed.Has(required) {
		return nil
	}

	message := fmt.Sprintf("%s not allowed in this context", description)
	d := NewDiagnosticForExpression(Error, message, expression, expression.GetRelativeLocation(), sourcePath)
	d.Code = CodeDisallowedFeature
	return d
}

// requiredFeatureForExpr returns the feature required for an expression node.
//
// Takes expression (Expression) which is the expression node to check.
// Takes allowed (ExpressionFeature) which specifies already allowed features.
//
// Returns ExpressionFeature which is the required feature, or 0 if the node
// is always allowed.
// Returns string which describes the feature, or empty if always allowed.
//
//nolint:revive // expression dispatch
func requiredFeatureForExpr(expression Expression, allowed ExpressionFeature) (ExpressionFeature, string) {
	switch n := expression.(type) {
	case *Identifier:
		return FeatureIdentifier, "Identifiers"

	case *MemberExpression:
		if n.Optional && !allowed.Has(FeatureOptionalChaining) {
			return FeatureOptionalChaining, "Optional chaining (?.)"
		}
		return FeatureMemberExpr, "Member access"

	case *IndexExpression:
		if n.Optional && !allowed.Has(FeatureOptionalChaining) {
			return FeatureOptionalChaining, "Optional chaining (?.[)"
		}
		return FeatureIndexExpr, "Index access"

	case *BinaryExpression:
		if n.Operator == "??" && !allowed.Has(FeatureNullishCoalescing) {
			return FeatureNullishCoalescing, "Nullish coalescing (??)"
		}
		return FeatureBinaryExpression, "Binary operators"

	case *UnaryExpression:
		return FeatureUnaryExpression, "Unary operators"

	case *CallExpression:
		return FeatureCallExpression, "Function calls"

	case *TernaryExpression:
		return FeatureTernaryExpression, "Ternary expressions"

	case *TemplateLiteral:
		return FeatureTemplateLiteral, "Template literals"

	case *ForInExpression:
		return FeatureForInExpr, "For-in expressions"

	case *ObjectLiteral:
		return FeatureObjectLiteral, "Object literals"

	case *ArrayLiteral:
		return FeatureArrayLiteral, "Array literals"

	case *LinkedMessageExpression:
		return FeatureLinkedMessage, "Linked messages (@)"

	case *StringLiteral, *IntegerLiteral, *FloatLiteral, *BooleanLiteral,
		*NilLiteral, *DecimalLiteral, *BigIntLiteral, *DateTimeLiteral,
		*DateLiteral, *TimeLiteral, *DurationLiteral, *RuneLiteral:
		return FeatureLiterals, "Literals"

	default:
		return 0, ""
	}
}

// isLiteralIndex checks whether an index expression uses a literal value.
//
// Takes index (Expression) which is the index expression to check.
//
// Returns bool which is true if the index is an integer or string literal.
func isLiteralIndex(index Expression) bool {
	switch index.(type) {
	case *IntegerLiteral, *StringLiteral:
		return true
	default:
		return false
	}
}
