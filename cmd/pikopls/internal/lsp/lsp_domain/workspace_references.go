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
	"fmt"
	"maps"
	"slices"

	protocol "github.com/politepixels/golang-language-server"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// symbolTarget holds details of a symbol to search for in the codebase.
type symbolTarget struct {
	// sourcePath is the file path where the symbol is defined.
	sourcePath string

	// name is the identifier of the target symbol.
	name string

	// defLocation is the line and column where the symbol is defined.
	defLocation ast_domain.Location
}

// FindAllReferences searches for all references to a symbol across all open documents in
// the workspace.
//
// Takes uri (protocol.DocumentURI) which identifies the document containing the symbol.
// Takes position (protocol.Position) which specifies the position of the symbol.
//
// Returns []protocol.Location which contains all reference locations found, empty when no
// symbol with a real definition sits at the position.
// Returns error when the document cannot be analysed or the search is cancelled.
func (w *workspace) FindAllReferences(ctx context.Context, uri protocol.DocumentURI, position protocol.Position) ([]protocol.Location, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Finding all references across workspace", logger_domain.String(keyURI, uri.Filename()))

	target, err := w.identifyTargetSymbol(ctx, uri, position)
	if err != nil {
		return nil, fmt.Errorf("identifying the symbol to find references for: %w", err)
	}
	if target == nil {
		return []protocol.Location{}, nil
	}

	l.Debug("Target symbol identified",
		logger_domain.String("name", target.name),
		logger_domain.String("sourcePath", target.sourcePath),
		logger_domain.Int("defLine", target.defLocation.Line),
		logger_domain.Int("defCol", target.defLocation.Column))

	allLocations, err := w.searchAllDocuments(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("searching the workspace for references to %q: %w", target.name, err)
	}

	l.Debug("Workspace reference search complete",
		logger_domain.Int("totalReferences", len(allLocations)))

	return allLocations, nil
}

// identifyTargetSymbol finds the symbol at the given position and returns its target
// info.
//
// Takes uri (protocol.DocumentURI) which specifies the document location.
// Takes position (protocol.Position) which specifies the position within the document.
//
// Returns *symbolTarget which contains the symbol's definition location, source path, and
// name, or nil when the position holds no symbol with a real definition.
// Returns error when the document analysis fails.
func (w *workspace) identifyTargetSymbol(ctx context.Context, uri protocol.DocumentURI, position protocol.Position) (*symbolTarget, error) {
	document, err := w.RunAnalysisForURI(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("analysing document for reference search: %w", err)
	}
	if document == nil || document.AnnotationResult == nil || document.AnnotationResult.AnnotatedAST == nil {
		return nil, nil
	}

	targetExpr, _ := findExpressionAtPosition(ctx, document.AnnotationResult.AnnotatedAST, position, uri.Filename())
	if targetExpr == nil {
		return nil, nil
	}

	targetAnn := targetExpr.GetGoAnnotation()
	if targetAnn == nil || targetAnn.Symbol == nil {
		return nil, nil
	}

	defLocation := targetAnn.Symbol.ReferenceLocation
	if defLocation.IsSynthetic() {
		return nil, nil
	}

	var sourcePath string
	if targetAnn.OriginalSourcePath != nil {
		sourcePath = *targetAnn.OriginalSourcePath
	}

	return &symbolTarget{
		defLocation: defLocation,
		sourcePath:  sourcePath,
		name:        targetAnn.Symbol.Name,
	}, nil
}

// searchAllDocuments searches all open documents for references to the target symbol,
// stopping as soon as ctx is cancelled.
//
// Takes target (*symbolTarget) which specifies the symbol to search for.
//
// Returns []protocol.Location which contains all locations where the target symbol is
// found across documents.
// Returns error when ctx is cancelled before every document has been searched.
//
// Safe for concurrent use. Takes a read lock while copying the document list, then
// searches each document without holding the lock.
func (w *workspace) searchAllDocuments(ctx context.Context, target *symbolTarget) ([]protocol.Location, error) {
	w.mu.RLock()
	documentsToSearch := slices.Collect(maps.Values(w.documents))
	w.mu.RUnlock()

	var allLocations []protocol.Location
	for _, searchDoc := range documentsToSearch {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("reference search stopped: %w", context.Cause(ctx))
		}
		locations := searchDoc.findReferencesToSymbol(target.defLocation, target.sourcePath)
		allLocations = append(allLocations, locations...)
	}

	return allLocations, nil
}
