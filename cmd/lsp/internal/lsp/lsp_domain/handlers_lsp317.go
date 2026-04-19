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
	"encoding/json"
	"fmt"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/logger/logger_domain"
)

// makeDocumentHandler returns a handler func that unmarshals raw params into P,
// looks up the document identified by docURI, and invokes invoke on it, returning
// emptyResult when no workspace is active or the document is not found.
//
// Takes server (*Server) which holds the workspace to query.
// Takes name (string) which is the handler name used in log and error messages.
// Takes emptyResult (any) which is returned when no document is available.
// Takes logField (func(P) (string, string)) which returns the log key and value.
// Takes docURI (func(P) protocol.DocumentURI) which extracts the lookup URI.
// Takes invoke (func(*document, context.Context, P) (any, error)) which calls
// the relevant document method.
//
// Returns func(context.Context, any) (any, error) ready for handler dispatch.
func makeDocumentHandler[P any](
	server *Server,
	name string,
	emptyResult any,
	logField func(P) (string, string),
	docURI func(P) protocol.DocumentURI,
	invoke func(*document, context.Context, P) (any, error),
) func(context.Context, any) (any, error) {
	return func(ctx context.Context, params any) (any, error) {
		_, l := logger_domain.From(ctx, log)

		var p P
		if err := remarshalParams(params, &p); err != nil {
			l.Error(name+": failed to parse params", logger_domain.Error(err))
			return emptyResult, nil
		}

		key, value := logField(p)
		l.Debug(name, logger_domain.String(key, value))

		if server.workspace == nil {
			return emptyResult, nil
		}

		document, exists := server.workspace.GetDocument(docURI(p))
		if !exists {
			l.Debug(name + ": Document not found")
			return emptyResult, nil
		}

		return invoke(document, ctx, p)
	}
}

// handleInlayHint handles textDocument/inlayHint requests.
// Returns inlay hints (type annotations, parameter names) for the given range.
//
// Takes params (any) which should be InlayHintParams.
//
// Returns any which contains the hints to display.
// Returns error when the request fails.
//
//nolint:dupl // type-specialised factory call
func (s *Server) handleInlayHint(ctx context.Context, params any) (any, error) {
	return makeDocumentHandler(
		s,
		"handleInlayHint",
		[]InlayHint{},
		func(p InlayHintParams) (string, string) {
			return keyURI, p.TextDocument.URI.Filename()
		},
		func(p InlayHintParams) protocol.DocumentURI {
			return p.TextDocument.URI
		},
		func(d *document, ctx context.Context, p InlayHintParams) (any, error) {
			return d.GetInlayHints(ctx, p.Range)
		},
	)(ctx, params)
}

// handlePrepareTypeHierarchy handles textDocument/prepareTypeHierarchy requests.
// Returns the type hierarchy item at the given position.
//
// Takes params (any) which should be TypeHierarchyPrepareParams.
//
// Returns any which contains the hierarchy items at the position.
// Returns error when the request fails.
//
//nolint:dupl // type-specialised factory call
func (s *Server) handlePrepareTypeHierarchy(ctx context.Context, params any) (any, error) {
	return makeDocumentHandler(
		s,
		"handlePrepareTypeHierarchy",
		[]TypeHierarchyItem{},
		func(p TypeHierarchyPrepareParams) (string, string) {
			return keyURI, p.TextDocument.URI.Filename()
		},
		func(p TypeHierarchyPrepareParams) protocol.DocumentURI {
			return p.TextDocument.URI
		},
		func(d *document, ctx context.Context, p TypeHierarchyPrepareParams) (any, error) {
			return d.PrepareTypeHierarchy(ctx, p.Position)
		},
	)(ctx, params)
}

// handleTypeHierarchySupertypes handles typeHierarchy/supertypes requests.
// Returns the supertypes (embedded types) of the given type.
//
// Takes params (any) which should be TypeHierarchySupertypesParams.
//
// Returns any which contains the supertypes as []TypeHierarchyItem.
// Returns error when the request fails.
//
//nolint:dupl // type-specialised factory call
func (s *Server) handleTypeHierarchySupertypes(ctx context.Context, params any) (any, error) {
	return makeDocumentHandler(
		s,
		"handleTypeHierarchySupertypes",
		[]TypeHierarchyItem{},
		func(p TypeHierarchySupertypesParams) (string, string) {
			return "typeName", p.Item.Name
		},
		func(p TypeHierarchySupertypesParams) protocol.DocumentURI {
			return p.Item.URI
		},
		func(d *document, ctx context.Context, p TypeHierarchySupertypesParams) (any, error) {
			return d.GetSupertypes(ctx, p.Item)
		},
	)(ctx, params)
}

// handleTypeHierarchySubtypes handles typeHierarchy/subtypes requests.
// Returns the subtypes (embedding types) of the given type.
//
// Takes params (any) which should be TypeHierarchySubtypesParams.
//
// Returns any which contains the subtypes as []TypeHierarchyItem.
// Returns error when the request fails.
//
//nolint:dupl // type-specialised factory call
func (s *Server) handleTypeHierarchySubtypes(ctx context.Context, params any) (any, error) {
	return makeDocumentHandler(
		s,
		"handleTypeHierarchySubtypes",
		[]TypeHierarchyItem{},
		func(p TypeHierarchySubtypesParams) (string, string) {
			return "typeName", p.Item.Name
		},
		func(p TypeHierarchySubtypesParams) protocol.DocumentURI {
			return p.Item.URI
		},
		func(d *document, ctx context.Context, p TypeHierarchySubtypesParams) (any, error) {
			return d.GetSubtypes(ctx, p.Item)
		},
	)(ctx, params)
}

// remarshalParams converts params to the expected struct type using JSON
// re-marshalling.
//
// Takes params (any) which is the source value, typically map[string]interface{}.
// Takes target (any) which is a pointer to the destination struct.
//
// Returns error when marshalling or unmarshalling fails.
func remarshalParams(params any, target any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshalling params: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshalling params: %w", err)
	}
	return nil
}
