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

package driven_code_emitter_go_literal

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// getAnnotationFromExpression extracts the Go generator annotation from an
// expression.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the annotation, or nil if
// the expression is nil or has no annotation.
func getAnnotationFromExpression(expression ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	if expression == nil {
		return nil
	}

	if ann := getAnnotationFromOperatorExpr(expression); ann != nil {
		return ann
	}

	if ann := getAnnotationFromLiteralExpr(expression); ann != nil {
		return ann
	}

	return getAnnotationFromComplexExpr(expression)
}

// getAnnotationFromOperatorExpr extracts Go generator annotations from an
// operator expression.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the Go generator
// annotations, or nil if the expression type is not supported.
func getAnnotationFromOperatorExpr(expression ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	switch n := expression.(type) {
	case *ast_domain.MemberExpression:
		return n.GoAnnotations
	case *ast_domain.IndexExpression:
		return n.GoAnnotations
	case *ast_domain.UnaryExpression:
		return n.GoAnnotations
	case *ast_domain.BinaryExpression:
		return n.GoAnnotations
	case *ast_domain.CallExpression:
		return n.GoAnnotations
	case *ast_domain.TernaryExpression:
		return n.GoAnnotations
	}
	return nil
}

// getAnnotationFromLiteralExpr returns the Go-specific annotations from a
// literal expression.
//
// Takes expression (ast_domain.Expression) which is the literal
// expression to check.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the annotations for
// the literal, or nil if the expression is not a known literal type.
func getAnnotationFromLiteralExpr(expression ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	switch n := expression.(type) {
	case *ast_domain.StringLiteral:
		return n.GoAnnotations
	case *ast_domain.IntegerLiteral:
		return n.GoAnnotations
	case *ast_domain.FloatLiteral:
		return n.GoAnnotations
	case *ast_domain.BooleanLiteral:
		return n.GoAnnotations
	case *ast_domain.NilLiteral:
		return n.GoAnnotations
	case *ast_domain.DecimalLiteral:
		return n.GoAnnotations
	case *ast_domain.BigIntLiteral:
		return n.GoAnnotations
	case *ast_domain.DateTimeLiteral:
		return n.GoAnnotations
	case *ast_domain.DateLiteral:
		return n.GoAnnotations
	case *ast_domain.TimeLiteral:
		return n.GoAnnotations
	case *ast_domain.DurationLiteral:
		return n.GoAnnotations
	case *ast_domain.RuneLiteral:
		return n.GoAnnotations
	}
	return nil
}

// getAnnotationFromComplexExpr extracts Go annotations from a complex
// expression.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the Go annotations if
// the expression is a supported type (identifier, template literal, object
// literal, or array literal), or nil if the type is not supported.
func getAnnotationFromComplexExpr(expression ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	switch n := expression.(type) {
	case *ast_domain.Identifier:
		return n.GoAnnotations
	case *ast_domain.TemplateLiteral:
		return n.GoAnnotations
	case *ast_domain.ObjectLiteral:
		return n.GoAnnotations
	case *ast_domain.ArrayLiteral:
		return n.GoAnnotations
	}
	return nil
}

// getEffectiveKeyExpression returns the key expression to use for a node,
// using the progressive enrichment strategy.
//
// It prefers the EffectiveKeyExpression (set by the annotator) over the
// structural node.Key (set by the keyAssigner).
//
// Takes node (*ast_domain.TemplateNode) which is the template node to get the
// key expression from.
//
// Returns ast_domain.Expression which is the key expression to use.
func getEffectiveKeyExpression(node *ast_domain.TemplateNode) ast_domain.Expression {
	if node.GoAnnotations != nil && node.GoAnnotations.EffectiveKeyExpression != nil {
		return node.GoAnnotations.EffectiveKeyExpression
	}

	return node.Key
}
