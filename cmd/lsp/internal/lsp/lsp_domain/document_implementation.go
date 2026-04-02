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

package lsp_domain

import (
	"context"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// GetImplementations finds all types that implement the interface at the given
// position. This enables the "Go to Implementation" feature for interfaces.
//
// Takes position (protocol.Position) which specifies the cursor position to query.
//
// Returns []protocol.Location which contains the definition locations of all
// implementing types.
// Returns error when the implementation lookup fails.
func (d *document) GetImplementations(ctx context.Context, position protocol.Position) ([]protocol.Location, error) {
	_, l := logger_domain.From(ctx, log)

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		l.Debug("GetImplementations: No annotated AST available")
		return []protocol.Location{}, nil
	}

	if d.TypeInspector == nil {
		l.Debug("GetImplementations: No TypeInspector available")
		return []protocol.Location{}, nil
	}

	typeName, packagePath, ok := d.extractImplementationTypeInfo(ctx, position)
	if !ok {
		return []protocol.Location{}, nil
	}

	l.Debug("GetImplementations: Looking for implementations",
		logger_domain.String("typeName", typeName),
		logger_domain.String("packagePath", packagePath))

	index := d.TypeInspector.GetImplementationIndex()
	if index == nil {
		l.Debug("GetImplementations: Implementation index not available")
		return []protocol.Location{}, nil
	}

	implementors := index.FindImplementations(packagePath, typeName)
	if len(implementors) == 0 {
		l.Debug("GetImplementations: No implementations found")
		return []protocol.Location{}, nil
	}

	l.Debug("GetImplementations: Found implementations",
		logger_domain.Int("count", len(implementors)))

	return buildImplementorLocations(implementors), nil
}

// extractImplementationTypeInfo finds the type name and package path at the
// given position.
//
// Takes position (protocol.Position) which specifies the location to inspect.
//
// Returns typeName (string) which is the extracted type name.
// Returns packagePath (string) which is the canonical package path of the type.
// Returns ok (bool) which indicates whether the extraction was successful.
func (d *document) extractImplementationTypeInfo(ctx context.Context, position protocol.Position) (typeName, packagePath string, ok bool) {
	return extractTypeInfoAtPosition(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename(), "GetImplementations")
}

// extractTypeInfoAtPosition finds the type name and package path of the
// expression at the given position. This is a shared helper used by both
// implementation lookup and type hierarchy preparation.
//
// Takes ast (*ast_domain.TemplateAST) which is the annotated AST to search.
// Takes position (protocol.Position) which specifies the location to inspect.
// Takes filename (string) which identifies the source file.
// Takes debugLabel (string) which is a prefix for debug log messages.
//
// Returns typeName (string) which is the extracted type name.
// Returns packagePath (string) which is the canonical package path of the type.
// Returns ok (bool) which indicates whether the extraction was successful.
func extractTypeInfoAtPosition(ctx context.Context, ast *ast_domain.TemplateAST, position protocol.Position, filename, debugLabel string) (typeName, packagePath string, ok bool) {
	_, l := logger_domain.From(ctx, log)

	targetExpr, _ := findExpressionAtPosition(ctx, ast, position, filename)
	if targetExpr == nil {
		l.Debug(debugLabel + ": No expression found at position")
		return "", "", false
	}

	ann := targetExpr.GetGoAnnotation()
	if ann == nil || ann.ResolvedType == nil {
		l.Debug(debugLabel + ": Expression has no resolved type")
		return "", "", false
	}

	typeName, _, ok = inspector_domain.DeconstructTypeExpr(ann.ResolvedType.TypeExpression)
	if !ok || typeName == "" {
		l.Debug(debugLabel + ": Could not extract type name from expression")
		return "", "", false
	}

	return typeName, ann.ResolvedType.CanonicalPackagePath, true
}

// buildImplementorLocations converts implementor info to protocol locations.
//
// Takes implementors ([]inspector_domain.ImplementorInfo) which contains the
// type implementors to convert.
//
// Returns []protocol.Location which contains the converted locations, skipping
// any implementors without a definition file.
func buildImplementorLocations(implementors []inspector_domain.ImplementorInfo) []protocol.Location {
	locations := make([]protocol.Location, 0, len(implementors))
	for _, implementor := range implementors {
		if implementor.DefinitionFile == "" {
			continue
		}
		locations = append(locations, buildImplementorLocation(implementor))
	}
	return locations
}

// buildImplementorLocation creates a protocol.Location for an implementor.
//
// Takes implementor (ImplementorInfo) which provides the type definition details.
//
// Returns protocol.Location which contains the file URI and range for the
// implementor's definition.
func buildImplementorLocation(implementor inspector_domain.ImplementorInfo) protocol.Location {
	return protocol.Location{
		URI: uri.File(implementor.DefinitionFile),
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(implementor.DefinitionLine - 1),
				Character: safeconv.IntToUint32(implementor.DefinitionCol - 1),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(implementor.DefinitionLine - 1),
				Character: safeconv.IntToUint32(implementor.DefinitionCol - 1 + len(implementor.TypeName)),
			},
		},
	}
}
