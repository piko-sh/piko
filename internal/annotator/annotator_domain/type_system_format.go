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
	"fmt"
	"strings"

	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
)

// formattableTypeNames lists type names that FormatBuilder can format
// natively, where unlisted types trigger a warning.
//
// The empty string and "interface{}" are allowed because the annotator
// may not always resolve the concrete type.
var formattableTypeNames = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"Decimal": true, "maths.Decimal": true,
	"BigInt": true, "maths.BigInt": true,
	"Money": true, "maths.Money": true,
	"Time": true, "time.Time": true,
	"DateTime": true, "i18n_domain.DateTime": true,
	"Duration": true, "time.Duration": true,
	"string":      true,
	"bool":        true,
	"":            true,
	"interface{}": true,
	"any":         true,
}

// validateFormatFuncArgs checks that F() and LF() are called with exactly one
// argument and that the argument type is one FormatBuilder can format. An
// unsupported type emits a Warning (not Error) because the runtime falls back
// to fmt.Sprintf.
//
// Takes ctx (*AnalysisContext) which provides the diagnostic context.
// Takes callExpr (*ast_domain.CallExpression) which is the call to validate.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// resolved annotations for the arguments.
// Takes baseLocation (ast_domain.Location) which provides the source location
// for diagnostics.
func (*TypeResolver) validateFormatFuncArgs(ctx *AnalysisContext, callExpr *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation, baseLocation ast_domain.Location) {
	if len(callExpr.Args) != 1 {
		message := fmt.Sprintf("F/LF expects exactly one argument, got %d", len(callExpr.Args))
		ctx.addDiagnosticForExpression(ast_domain.Error, message, callExpr, baseLocation, nil, annotator_dto.CodeFormatDirectiveError)
		return
	}

	if len(argAnns) == 1 && argAnns[0] != nil && argAnns[0].ResolvedType != nil {
		typeName := goastutil.ASTToTypeString(argAnns[0].ResolvedType.TypeExpression, argAnns[0].ResolvedType.PackageAlias)
		if !isFormattableType(typeName) {
			message := fmt.Sprintf(
				"F/LF argument type '%s' may not be formattable; supported types are numeric (int, float, Decimal, BigInt, Money), temporal (time.Time, DateTime), and string",
				typeName,
			)
			ctx.addDiagnosticForExpression(ast_domain.Warning, message, callExpr, baseLocation, nil, annotator_dto.CodeFormatDirectiveError)
		}
	}
}

// isFormattableType checks whether a resolved type string is one that
// FormatBuilder handles natively. Pointer types are unwrapped first.
//
// Takes typeName (string) which is the resolved type name to check.
//
// Returns bool which is true if the type is natively formattable by
// FormatBuilder.
func isFormattableType(typeName string) bool {
	bare := strings.TrimLeft(typeName, "*")
	return formattableTypeNames[bare]
}

// getFormatFuncReturnType returns the *i18n_domain.FormatBuilder type
// information for F() and LF() formatting functions, enabling method
// chain resolution (e.g. F(x).Precision(2)) via the inspector.
//
// The stringability pipeline automatically adds .String() in string
// contexts since FormatBuilder implements fmt.Stringer.
//
// Returns *ast_domain.ResolvedTypeInfo which holds the *FormatBuilder type
// with its canonical package path for inspector method resolution.
func getFormatFuncReturnType(_ *TypeResolver, _ *AnalysisContext, _ *ast_domain.CallExpression, _ []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.StarExpr{
			X: &goast.SelectorExpr{
				X:   goast.NewIdent("i18n_domain"),
				Sel: goast.NewIdent("FormatBuilder"),
			},
		},
		PackageAlias:         "i18n_domain",
		CanonicalPackagePath: "piko.sh/piko/internal/i18n/i18n_domain",
	}
}
