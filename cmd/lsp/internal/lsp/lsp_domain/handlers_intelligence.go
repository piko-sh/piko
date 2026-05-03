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
	"piko.sh/piko/internal/logger/logger_domain"
)

// getDocumentForURIHandler retrieves the document for an LSP handler that
// operates on a URI without a cursor position.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is
// detached so analysis completes independently.
// Takes handlerName (string) which identifies the LSP operation for logging.
// Takes uri (protocol.DocumentURI) which specifies the document to analyse.
//
// Returns *document which is the analysed document, or nil if analysis fails.
// When analysis fails, the error is logged and nil is returned.
func (s *Server) getDocumentForURIHandler(ctx context.Context, handlerName string, uri protocol.DocumentURI) *document {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug(handlerName, logger_domain.String(keyURI, uri.Filename()))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), uri)
	if err != nil {
		l.Error(handlerName+": analysis failed", logger_domain.Error(err))
		return nil
	}
	return document
}

// getDocumentForHandler retrieves the document for an LSP handler operation.
// It logs the operation, runs analysis, and returns the document.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is
// detached so analysis completes independently.
// Takes handlerName (string) which identifies the LSP operation for logging.
// Takes uri (protocol.DocumentURI) which specifies the document to analyse.
// Takes position (protocol.Position) which provides the cursor
// position for logging.
//
// Returns *document which is the analysed document, or nil if analysis fails.
// When analysis fails, the error is logged and nil is returned.
func (s *Server) getDocumentForHandler(ctx context.Context, handlerName string, uri protocol.DocumentURI, position protocol.Position) *document {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug(handlerName, logger_domain.String(keyURI, uri.Filename()), logger_domain.Field(keyPosition, position))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), uri)
	if err != nil {
		l.Error(handlerName+": analysis failed", logger_domain.Error(err))
		return nil
	}
	return document
}

// CompletionResolve provides additional details for a selected completion
// item.
//
// Takes params (*protocol.CompletionItem) which specifies the item to resolve.
//
// Returns *protocol.CompletionItem which contains the enriched completion item.
// Returns error when the resolution fails.
func (*Server) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("CompletionResolve", logger_domain.String("label", params.Label))

	if params.Detail == "" {
		params.Detail = "Symbol from current scope"
	}

	return params, nil
}

// TypeDefinition handles requests to find the type definition of a symbol.
//
// Takes params (*protocol.TypeDefinitionParams) which specifies the document
// and position to find the type definition for.
//
// Returns []protocol.Location which contains the locations of the type
// definitions found, or an empty slice if none exist.
// Returns error when the type definition lookup fails.
func (s *Server) TypeDefinition(ctx context.Context, params *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	document := s.getDocumentForHandler(ctx, "TypeDefinition", params.TextDocument.URI, params.Position)
	if document == nil {
		return []protocol.Location{}, nil
	}
	return document.GetTypeDefinition(ctx, params.Position)
}

// Implementation finds all implementations of an interface or abstract method
// at the given position.
//
// Takes params (*protocol.ImplementationParams) which specifies the text
// document position to find implementations for.
//
// Returns []protocol.Location which contains the locations of all found
// implementations.
// Returns error when the lookup fails.
func (s *Server) Implementation(ctx context.Context, params *protocol.ImplementationParams) ([]protocol.Location, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Implementation", logger_domain.String(keyURI, params.TextDocument.URI.Filename()), logger_domain.Field(keyPosition, params.Position))

	if s.workspace == nil {
		return []protocol.Location{}, nil
	}

	document, exists := s.workspace.GetDocument(params.TextDocument.URI)
	if !exists {
		l.Debug("Implementation: Document not found")
		return []protocol.Location{}, nil
	}

	return document.GetImplementations(ctx, params.Position)
}

// Declaration finds the declaration of a symbol at the given position.
//
// In Go, declaration and definition are the same, so this delegates to the
// Definition handler.
//
// Takes params (*protocol.DeclarationParams) which specifies the text document
// position to find the declaration for.
//
// Returns []protocol.Location which contains the declaration locations found.
// Returns error when the lookup fails.
func (s *Server) Declaration(ctx context.Context, params *protocol.DeclarationParams) ([]protocol.Location, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Declaration", logger_domain.String(keyURI, params.TextDocument.URI.Filename()), logger_domain.Field(keyPosition, params.Position))

	defParams := &protocol.DefinitionParams{
		TextDocumentPositionParams: params.TextDocumentPositionParams,
	}
	return s.Definition(ctx, defParams)
}

// References handles requests to find all references to a symbol across the
// workspace.
//
// Takes params (*protocol.ReferenceParams) which specifies the text document
// and position to find references for.
//
// Returns []protocol.Location which contains all locations where the symbol
// is referenced.
// Returns error when the workspace search fails.
func (s *Server) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("References", logger_domain.String(keyURI, params.TextDocument.URI.Filename()), logger_domain.Field(keyPosition, params.Position))

	locations, err := s.workspace.FindAllReferences(ctx, params.TextDocument.URI, params.Position)
	if err != nil {
		l.Error("References: workspace search failed", logger_domain.Error(err))
		return []protocol.Location{}, nil
	}

	l.Debug("References found",
		logger_domain.Int("count", len(locations)),
		logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	return locations, nil
}

// DocumentHighlight finds all places where a symbol appears in a document.
//
// Takes params (*protocol.DocumentHighlightParams) which specifies the document
// and position to find highlights for.
//
// Returns []protocol.DocumentHighlight which contains the ranges of all
// matching symbol uses.
// Returns error when the highlights cannot be found.
func (s *Server) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	document := s.getDocumentForHandler(ctx, "DocumentHighlight", params.TextDocument.URI, params.Position)
	if document == nil {
		return []protocol.DocumentHighlight{}, nil
	}
	return document.GetDocumentHighlights(ctx, params.Position)
}

// SignatureHelp provides function signature information at a given position.
//
// Takes params (*protocol.SignatureHelpParams) which specifies the document
// position for which to provide signature help.
//
// Returns *protocol.SignatureHelp which contains the signature information at
// the requested position, or an empty result if no document is found.
// Returns error when the signature help request fails.
func (s *Server) SignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("SignatureHelp request received", logger_domain.String("uri", params.TextDocument.URI.Filename()))

	if result := s.tryFastPathSignatureHelp(ctx, params); result != nil {
		l.Debug("SignatureHelp: fast-path succeeded",
			logger_domain.String("uri", params.TextDocument.URI.Filename()),
			logger_domain.Int("signatures", len(result.Signatures)))
		return result, nil
	}

	document := s.getDocumentForHandler(ctx, "SignatureHelp", params.TextDocument.URI, params.Position)
	if document == nil {
		return &protocol.SignatureHelp{Signatures: []protocol.SignatureInformation{}}, nil
	}
	return document.GetSignatureHelp(params.Position)
}

// SemanticTokensFull provides semantic tokens for a whole document.
//
// Takes params (*protocol.SemanticTokensParams) which identifies the document
// to tokenise.
//
// Returns *protocol.SemanticTokens which contains the encoded token data for
// syntax highlighting.
// Returns error when token generation fails.
func (*Server) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("SemanticTokensFull", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	return &protocol.SemanticTokens{
		Data: []uint32{},
	}, nil
}

// SemanticTokensFullDelta provides incremental updates to semantic tokens.
//
// Takes params (*protocol.SemanticTokensDeltaParams) which identifies the
// document and the previous result to compute changes against.
//
// Returns any which contains the semantic tokens delta with edit operations.
// Returns error when the delta cannot be computed.
func (*Server) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (any, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("SemanticTokensFullDelta", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	return &protocol.SemanticTokensDelta{
		Edits: []protocol.SemanticTokensEdit{},
	}, nil
}

// SemanticTokensRange provides semantic tokens for a specific range in a
// document.
//
// Takes params (*protocol.SemanticTokensRangeParams) which specifies the
// document and range to analyse.
//
// Returns *protocol.SemanticTokens which contains the semantic tokens for the
// requested range.
// Returns error when token analysis fails.
func (*Server) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("SemanticTokensRange", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	return &protocol.SemanticTokens{
		Data: []uint32{},
	}, nil
}

// CodeLensResolve fills in additional details for a CodeLens item.
//
// Takes params (*protocol.CodeLens) which specifies the CodeLens to resolve.
//
// Returns *protocol.CodeLens which contains the resolved CodeLens with details.
// Returns error when the resolution fails.
func (*Server) CodeLensResolve(ctx context.Context, params *protocol.CodeLens) (*protocol.CodeLens, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("CodeLensResolve")
	return params, nil
}

// DocumentLink finds clickable links within a document.
//
// Takes params (*protocol.DocumentLinkParams) which specifies the document to
// scan for links.
//
// Returns []protocol.DocumentLink which contains the links found.
// Returns error when the analysis fails.
func (s *Server) DocumentLink(ctx context.Context, params *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("DocumentLink", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Error("DocumentLink: analysis failed", logger_domain.Error(err))
		return []protocol.DocumentLink{}, nil
	}

	return document.GetDocumentLinks(ctx)
}

// DocumentLinkResolve fills in extra details for a document link.
//
// Takes params (*protocol.DocumentLink) which specifies the link to resolve.
//
// Returns *protocol.DocumentLink which contains the resolved link details.
// Returns error when resolution fails.
func (*Server) DocumentLinkResolve(ctx context.Context, params *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DocumentLinkResolve")
	return params, nil
}

// DocumentColor returns colour information for colour literals in a document.
//
// Takes params (*protocol.DocumentColorParams) which specifies the document to
// check for colour literals.
//
// Returns []protocol.ColorInformation which contains the colours found.
// Returns error when the document cannot be read.
func (s *Server) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	document := s.getDocumentForURIHandler(ctx, "DocumentColor", params.TextDocument.URI)
	if document == nil {
		return []protocol.ColorInformation{}, nil
	}
	return document.GetDocumentColors()
}

// ColorPresentation provides colour format conversion options.
//
// Takes params (*protocol.ColorPresentationParams) which specifies the colour
// and text document location.
//
// Returns []protocol.ColorPresentation which contains the available colour
// format options.
// Returns error when document analysis fails.
func (s *Server) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("ColorPresentation", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Error("ColorPresentation: analysis failed", logger_domain.Error(err))
		return []protocol.ColorPresentation{}, nil
	}

	return document.GetColorPresentations(params.Color)
}

// Rename handles requests to rename a symbol across the workspace.
//
// Takes params (*protocol.RenameParams) which specifies the symbol position
// and new name.
//
// Returns *protocol.WorkspaceEdit which contains the text edits for all files.
// Returns error when the rename fails.
func (s *Server) Rename(ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Rename", logger_domain.String(keyURI, params.TextDocument.URI.Filename()), logger_domain.Field(keyPosition, params.Position), logger_domain.String("newName", params.NewName))

	locations, err := s.workspace.FindAllReferences(context.WithoutCancel(ctx), params.TextDocument.URI, params.Position)
	if err != nil {
		l.Error("Rename: failed to find references", logger_domain.Error(err))
		return &protocol.WorkspaceEdit{
			Changes: make(map[protocol.DocumentURI][]protocol.TextEdit),
		}, nil
	}

	if len(locations) == 0 {
		l.Debug("Rename: no references found")
		return &protocol.WorkspaceEdit{
			Changes: make(map[protocol.DocumentURI][]protocol.TextEdit),
		}, nil
	}

	editsByFile := make(map[protocol.DocumentURI][]protocol.TextEdit)
	for _, location := range locations {
		edit := protocol.TextEdit{
			Range:   location.Range,
			NewText: params.NewName,
		}
		editsByFile[location.URI] = append(editsByFile[location.URI], edit)
	}

	l.Debug("Rename: generated workspace edit",
		logger_domain.Int("totalEdits", len(locations)),
		logger_domain.Int("filesAffected", len(editsByFile)))

	return &protocol.WorkspaceEdit{
		Changes: editsByFile,
	}, nil
}

// PrepareRename checks if a symbol can be renamed and returns its range.
//
// Takes params (*protocol.PrepareRenameParams) which specifies the document
// and position of the symbol to check.
//
// Returns *protocol.Range which contains the range of the symbol if it can be
// renamed, or nil if renaming is not supported at that position.
// Returns error when the rename preparation fails.
func (s *Server) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (*protocol.Range, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("PrepareRename", logger_domain.String(keyURI, params.TextDocument.URI.Filename()), logger_domain.Field(keyPosition, params.Position))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Error("PrepareRename: analysis failed", logger_domain.Error(err))
		return nil, nil
	}

	return document.PrepareRename(ctx, params.Position)
}

// FoldingRanges provides code folding regions for a document.
//
// Takes params (*protocol.FoldingRangeParams) which identifies the document to
// analyse.
//
// Returns []protocol.FoldingRange which contains the folding regions found.
// Returns error when the document cannot be analysed.
func (s *Server) FoldingRanges(ctx context.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	document := s.getDocumentForURIHandler(ctx, "FoldingRanges", params.TextDocument.URI)
	if document == nil {
		return []protocol.FoldingRange{}, nil
	}
	return document.GetFoldingRanges()
}

// IncomingCalls provides callers of a function in the call hierarchy.
//
// Takes params (*protocol.CallHierarchyIncomingCallsParams) which specifies
// the call hierarchy item to find callers for.
//
// Returns []protocol.CallHierarchyIncomingCall which contains the incoming
// calls to the specified item.
// Returns error when the operation fails.
func (*Server) IncomingCalls(ctx context.Context, params *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("IncomingCalls", logger_domain.String("name", params.Item.Name))
	return []protocol.CallHierarchyIncomingCall{}, nil
}

// OutgoingCalls returns the functions called by a given call hierarchy item.
//
// Takes params (*protocol.CallHierarchyOutgoingCallsParams) which specifies
// the call hierarchy item to get outgoing calls for.
//
// Returns []protocol.CallHierarchyOutgoingCall which contains the functions
// called by the given item.
// Returns error when the outgoing calls cannot be found.
func (*Server) OutgoingCalls(ctx context.Context, params *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("OutgoingCalls", logger_domain.String("name", params.Item.Name))
	return []protocol.CallHierarchyOutgoingCall{}, nil
}

// PrepareCallHierarchy prepares a call hierarchy item at a given position.
//
// Takes params (*protocol.CallHierarchyPrepareParams) which specifies the
// text document and position to prepare the call hierarchy for.
//
// Returns []protocol.CallHierarchyItem which contains the call hierarchy items
// at the given position.
// Returns error when the preparation fails.
func (*Server) PrepareCallHierarchy(ctx context.Context, params *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("PrepareCallHierarchy", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
	return []protocol.CallHierarchyItem{}, nil
}

// LinkedEditingRange provides ranges that should be edited together.
//
// Takes params (*protocol.LinkedEditingRangeParams) which specifies the
// document and position to find linked ranges for.
//
// Returns *protocol.LinkedEditingRanges which contains the ranges that should
// be edited simultaneously.
// Returns error when the linked ranges cannot be determined.
func (s *Server) LinkedEditingRange(ctx context.Context, params *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	document := s.getDocumentForHandler(ctx, "LinkedEditingRange", params.TextDocument.URI, params.Position)
	if document == nil {
		return &protocol.LinkedEditingRanges{Ranges: []protocol.Range{}}, nil
	}
	return document.GetLinkedEditingRanges(params.Position)
}

// Moniker returns unique identifiers for symbols to support cross-repository
// navigation.
//
// Takes params (*protocol.MonikerParams) which specifies the text document
// position to get monikers for.
//
// Returns []protocol.Moniker which contains the unique identifiers for the
// symbol at the given position.
// Returns error when the moniker lookup fails.
func (s *Server) Moniker(ctx context.Context, params *protocol.MonikerParams) ([]protocol.Moniker, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Moniker", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	if s.workspace == nil {
		return []protocol.Moniker{}, nil
	}

	document, exists := s.workspace.GetDocument(params.TextDocument.URI)
	if !exists {
		l.Debug("Moniker: Document not found")
		return []protocol.Moniker{}, nil
	}

	return document.GetMonikers(ctx, params.Position)
}
