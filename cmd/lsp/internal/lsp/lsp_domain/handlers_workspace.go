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
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// minQueryLength is the minimum number of characters required for a workspace
// symbol search.
const minQueryLength = 2

// DidChangeWatchedFiles handles notifications when watched files change on
// disk.
//
// Takes params (*protocol.DidChangeWatchedFilesParams) which contains the list
// of file change events to process.
//
// Returns error when processing fails (currently always returns nil).
//
// Runs analysis for changed files using a bounded worker pool to prevent
// resource exhaustion. Uses the server context so analysis stops during
// shutdown.
func (s *Server) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidChangeWatchedFiles", logger_domain.Int("changes", len(params.Changes)))

	changedURIs, hasGoFileChange, hasStructuralChange := classifyFileChanges(l, params.Changes)

	if hasGoFileChange || hasStructuralChange {
		s.workspace.resetScopedAnalysisCache()
	}

	if hasGoFileChange {
		l.Debug("Go file changed, re-analysing all open documents")
		uris := s.docCache.GetAllURIs()
		s.goBackground(func(ctx context.Context) {
			s.runBoundedAnalysis(ctx, uris, "go file change")
		})
		return nil
	}

	if len(changedURIs) == 0 {
		return nil
	}

	s.goBackground(func(ctx context.Context) {
		s.runBoundedAnalysis(ctx, changedURIs, "watched file change")
	})

	return nil
}

// classifyFileChanges categorises watched file changes into Go file changes,
// structural changes (create/delete), and non-Go changed URIs needing
// re-analysis.
//
// Takes l (logger_domain.Logger) which is the logger for debug output.
// Takes changes ([]*protocol.FileEvent) which contains the file change events
// to classify.
//
// Returns changedURIs ([]protocol.DocumentURI) which holds non-Go changed URIs.
// Returns hasGoFileChange (bool) which is true when a Go source file changed.
// Returns hasStructuralChange (bool) which is true when files were created or
// deleted.
func classifyFileChanges(l logger_domain.Logger, changes []*protocol.FileEvent) (changedURIs []protocol.DocumentURI, hasGoFileChange, hasStructuralChange bool) {
	for _, change := range changes {
		l.Debug("File change", logger_domain.String(keyURI, change.URI.Filename()), logger_domain.Int("type", int(change.Type)))

		filename := change.URI.Filename()

		if isIgnoredWatchPath(filename) {
			l.Debug("Ignoring change in build output directory",
				logger_domain.String(keyURI, filename))
			continue
		}

		ext := strings.ToLower(filepath.Ext(filename))

		switch change.Type {
		case protocol.FileChangeTypeChanged:
			if ext == ".go" && !strings.HasSuffix(filename, "_test.go") {
				hasGoFileChange = true
			} else {
				changedURIs = append(changedURIs, change.URI)
			}
		case protocol.FileChangeTypeCreated, protocol.FileChangeTypeDeleted:
			hasStructuralChange = true
			if ext != ".go" {
				changedURIs = append(changedURIs, change.URI)
			}
		}
	}
	return changedURIs, hasGoFileChange, hasStructuralChange
}

// isIgnoredWatchPath returns true for paths inside build output directories
// that should not trigger re-analysis.
//
// Takes filename (string) which is the file path to check.
//
// Returns bool which is true for paths inside build output directories.
func isIgnoredWatchPath(filename string) bool {
	normalised := filepath.ToSlash(filename)

	return strings.Contains(normalised, "/"+config.PikoInternalPath+"/") ||
		strings.Contains(normalised, "/dist/")
}

// DidChangeWorkspaceFolders handles notifications when workspace folders are
// added or removed.
//
// Takes params (*protocol.DidChangeWorkspaceFoldersParams) which contains the
// added and removed workspace folders.
//
// Returns error when the folder changes cannot be processed.
func (*Server) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidChangeWorkspaceFolders", logger_domain.Int("added", len(params.Event.Added)), logger_domain.Int("removed", len(params.Event.Removed)))

	for _, added := range params.Event.Added {
		l.Debug("Workspace folder added", logger_domain.String(keyURI, string(added.URI)))
	}
	for _, removed := range params.Event.Removed {
		l.Debug("Workspace folder removed", logger_domain.String(keyURI, string(removed.URI)))
	}

	return nil
}

// DidChangeConfiguration handles notifications when the configuration changes.
//
// Returns error when the configuration update fails.
//
// Safe for concurrent use. Uses a mutex to protect configuration updates.
func (s *Server) DidChangeConfiguration(ctx context.Context, _ *protocol.DidChangeConfigurationParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidChangeConfiguration")

	s.mu.Lock()
	defer s.mu.Unlock()

	l.Debug("Configuration updated successfully")
	return nil
}

// ExecuteCommand handles requests to run a custom command.
//
// Takes params (*protocol.ExecuteCommandParams) which specifies the command
// name and its arguments.
//
// Returns any which is a status message describing the result.
// Returns error when the command is not recognised.
//
// Uses a bounded worker pool when refreshing diagnostics to prevent resource
// exhaustion. Uses the server context so analysis stops during shutdown.
func (s *Server) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (any, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("ExecuteCommand", logger_domain.String("command", params.Command), logger_domain.Int("arguments", len(params.Arguments)))

	switch params.Command {
	case "piko.refreshDiagnostics":
		uris := s.docCache.GetAllURIs()

		s.goBackground(func(ctx context.Context) {
			s.runBoundedAnalysis(ctx, uris, "diagnostics refresh")
		})
		return "Diagnostics refresh started", nil

	default:
		return nil, fmt.Errorf("unknown command: %s", params.Command)
	}
}

// Symbols handles workspace-wide symbol searches.
//
// Takes params (*protocol.WorkspaceSymbolParams) which contains the search
// query.
//
// Returns []protocol.SymbolInformation which contains matching symbols from
// Go files and PK documents.
// Returns error when the search fails.
func (s *Server) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Workspace Symbols", logger_domain.String("query", params.Query))

	if params.Query == "" || len(params.Query) < minQueryLength {
		return []protocol.SymbolInformation{}, nil
	}

	queryLower := strings.ToLower(params.Query)
	symbols := s.searchGoSymbols(queryLower)
	symbols = append(symbols, s.searchPKDocuments(queryLower)...)

	l.Debug("Workspace symbol search completed",
		logger_domain.Int("results", len(symbols)),
		logger_domain.String("query", params.Query))
	return symbols, nil
}

// searchGoSymbols searches through Go symbols from the TypeInspector.
//
// Takes queryLower (string) which is the lowercase search query to match
// against symbol names.
//
// Returns []protocol.SymbolInformation which contains matching symbols
// converted to LSP format, or nil if the TypeInspector is unavailable.
func (s *Server) searchGoSymbols(queryLower string) []protocol.SymbolInformation {
	if s.workspace.typeInspectorManager == nil {
		return nil
	}

	querier, ok := s.workspace.typeInspectorManager.GetQuerier()
	if !ok || querier == nil {
		return nil
	}

	var symbols []protocol.SymbolInformation
	for _, symbol := range querier.GetAllSymbols() {
		if info := convertGoSymbolToLSP(symbol, queryLower); info != nil {
			symbols = append(symbols, *info)
		}
	}
	return symbols
}

// searchPKDocuments searches through open PK documents for elements with
// matching IDs.
//
// Takes queryLower (string) which specifies the lowercase search query to
// match against element IDs.
//
// Returns []protocol.SymbolInformation which contains all matching symbols
// found across open documents.
//
// Safe for concurrent use. Takes a read lock on the workspace documents map
// before copying.
func (s *Server) searchPKDocuments(queryLower string) []protocol.SymbolInformation {
	s.workspace.mu.RLock()
	docs := make(map[protocol.DocumentURI]*document)
	maps.Copy(docs, s.workspace.documents)
	s.workspace.mu.RUnlock()

	symbols := make([]protocol.SymbolInformation, 0, len(docs))
	for uri, document := range docs {
		symbols = append(symbols, searchDocumentForIDs(uri, document, queryLower)...)
	}
	return symbols
}

// DocumentSymbol returns the document outline and structure.
//
// Takes params (*protocol.DocumentSymbolParams) which specifies the document
// to analyse.
//
// Returns []any which contains the document symbols that form the outline.
// Returns error when the analysis fails.
func (s *Server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]any, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("DocumentSymbol", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Error("DocumentSymbol: analysis failed", logger_domain.Error(err))
		return []any{}, nil
	}

	return document.GetDocumentSymbols()
}

// DidCreateFiles handles notifications when files are created.
//
// Takes params (*protocol.CreateFilesParams) which holds the list of files
// that were created.
//
// Returns error when the notification cannot be processed.
func (s *Server) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidCreateFiles", logger_domain.Int(keyFiles, len(params.Files)))

	for _, file := range params.Files {
		l.Debug("Created file", logger_domain.String(keyURI, string(file.URI)))
	}

	if len(params.Files) > 0 {
		s.workspace.resetScopedAnalysisCache()
	}

	return nil
}

// WillCreateFiles handles requests before files are created.
//
// Takes params (*protocol.CreateFilesParams) which contains the files about to
// be created.
//
// Returns *protocol.WorkspaceEdit which provides edits to apply before file
// creation.
// Returns error when the request cannot be processed.
func (*Server) WillCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("WillCreateFiles", logger_domain.Int(keyFiles, len(params.Files)))

	return &protocol.WorkspaceEdit{
		Changes: make(map[protocol.DocumentURI][]protocol.TextEdit),
	}, nil
}

// DidRenameFiles handles notifications when files are renamed.
//
// Takes params (*protocol.RenameFilesParams) which contains the old and new
// URIs for each renamed file.
//
// Returns error when the rename operation cannot be processed.
func (s *Server) DidRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidRenameFiles", logger_domain.Int(keyFiles, len(params.Files)))

	for _, file := range params.Files {
		l.Debug("Renamed file", logger_domain.String("oldURI", string(file.OldURI)), logger_domain.String("newURI", string(file.NewURI)))
		if content, found := s.docCache.Get(protocol.DocumentURI(file.OldURI)); found {
			s.docCache.Delete(protocol.DocumentURI(file.OldURI))
			s.docCache.Set(protocol.DocumentURI(file.NewURI), content)
		}
	}

	if len(params.Files) > 0 {
		s.workspace.resetScopedAnalysisCache()
	}

	return nil
}

// WillRenameFiles handles requests before files are renamed.
//
// Takes params (*protocol.RenameFilesParams) which contains the files to be
// renamed.
//
// Returns *protocol.WorkspaceEdit which contains edits to apply before the
// rename occurs.
// Returns error when the request cannot be processed.
func (*Server) WillRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("WillRenameFiles", logger_domain.Int(keyFiles, len(params.Files)))

	return &protocol.WorkspaceEdit{
		Changes: make(map[protocol.DocumentURI][]protocol.TextEdit),
	}, nil
}

// DidDeleteFiles handles notifications when files are deleted.
//
// Takes params (*protocol.DeleteFilesParams) which contains the list of
// deleted files.
//
// Returns error when handling fails.
func (s *Server) DidDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidDeleteFiles", logger_domain.Int(keyFiles, len(params.Files)))

	for _, file := range params.Files {
		l.Debug("Deleted file", logger_domain.String(keyURI, string(file.URI)))
		s.docCache.Delete(protocol.DocumentURI(file.URI))
	}

	if len(params.Files) > 0 {
		s.workspace.resetScopedAnalysisCache()
	}

	return nil
}

// WillDeleteFiles handles requests before files are deleted.
//
// Takes params (*protocol.DeleteFilesParams) which contains the files to be
// deleted.
//
// Returns *protocol.WorkspaceEdit which contains edits to apply before
// deletion.
// Returns error when the request cannot be processed.
func (*Server) WillDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("WillDeleteFiles", logger_domain.Int(keyFiles, len(params.Files)))

	return &protocol.WorkspaceEdit{
		Changes: make(map[protocol.DocumentURI][]protocol.TextEdit),
	}, nil
}

// CodeAction provides quick fixes and code actions for a document.
//
// Takes params (*protocol.CodeActionParams) which specifies the document and
// range for which code actions are requested.
//
// Returns []protocol.CodeAction which contains the available actions including
// format document, refresh diagnostics, and context-specific quick fixes.
// Returns error when the operation fails.
func (s *Server) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("CodeAction", logger_domain.String(keyURI, params.TextDocument.URI.Filename()), logger_domain.Field("range", params.Range))

	actions := []protocol.CodeAction{}

	formatAction := protocol.CodeAction{
		Title: "Format Document",
		Kind:  protocol.SourceOrganizeImports,
		Command: &protocol.Command{
			Title:     "Format Document",
			Command:   "piko.formatDocument",
			Arguments: []any{string(params.TextDocument.URI)},
		},
	}
	actions = append(actions, formatAction)

	refreshAction := protocol.CodeAction{
		Title: "Refresh Diagnostics",
		Kind:  protocol.Source,
		Command: &protocol.Command{
			Title:     "Refresh Diagnostics",
			Command:   "piko.refreshDiagnostics",
			Arguments: []any{},
		},
	}
	actions = append(actions, refreshAction)

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Debug("CodeAction: failed to get document", logger_domain.Error(err))
		document = nil
	}

	for i := range params.Context.Diagnostics {
		if document != nil {
			quickFixes := generateQuickFixes(ctx, params.Context.Diagnostics[i], document, s.workspace)
			actions = append(actions, quickFixes...)
		}
	}

	l.Debug("CodeAction generated", logger_domain.Int("actions", len(actions)))
	return actions, nil
}

// CodeLens provides inline commands and information within the document.
//
// Takes params (*protocol.CodeLensParams) which identifies the document to
// analyse.
//
// Returns []protocol.CodeLens which contains the code lenses for the document.
// Returns error when the document cannot be processed.
func (*Server) CodeLens(ctx context.Context, params *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("CodeLens", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	return []protocol.CodeLens{}, nil
}

// CodeLensRefresh asks the client to refresh all CodeLens items.
//
// Returns error when the refresh request fails.
func (*Server) CodeLensRefresh(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("CodeLensRefresh")
	return nil
}

// SemanticTokensRefresh requests a refresh of all semantic tokens.
//
// Returns error when the refresh request fails.
func (*Server) SemanticTokensRefresh(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("SemanticTokensRefresh")
	return nil
}

// ShowDocument asks the client to show a document.
//
// Takes params (*protocol.ShowDocumentParams) which specifies the document to
// display.
//
// Returns *protocol.ShowDocumentResult which indicates whether the document
// was shown.
// Returns error when the request fails.
func (*Server) ShowDocument(ctx context.Context, params *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("ShowDocument", logger_domain.String(keyURI, string(params.URI)))
	return &protocol.ShowDocumentResult{
		Success: false,
	}, nil
}

// convertGoSymbolToLSP converts a Go symbol to an LSP SymbolInformation if it
// matches the query.
//
// Takes symbol (inspector_dto.WorkspaceSymbol) which is the Go symbol to convert.
// Takes queryLower (string) which is the search query in lowercase.
//
// Returns *protocol.SymbolInformation which is the converted symbol, or nil if
// the symbol does not match the query or lacks the required location data.
func convertGoSymbolToLSP(symbol inspector_dto.WorkspaceSymbol, queryLower string) *protocol.SymbolInformation {
	if !strings.Contains(strings.ToLower(symbol.Name), queryLower) {
		return nil
	}

	if symbol.FilePath == "" || symbol.Line == 0 {
		return nil
	}

	return &protocol.SymbolInformation{
		Name:          symbol.Name,
		Kind:          mapGoSymbolKind(symbol.Kind),
		ContainerName: buildContainerName(symbol.PackageName, symbol.ContainerName),
		Location:      buildSymbolLocation(symbol.FilePath, symbol.Line, symbol.Column, symbol.Name),
	}
}

// mapGoSymbolKind converts a Go symbol kind string to an LSP SymbolKind.
//
// Takes kind (string) which is the Go symbol kind to convert.
//
// Returns protocol.SymbolKind which is the corresponding LSP symbol kind.
func mapGoSymbolKind(kind string) protocol.SymbolKind {
	switch kind {
	case "type":
		return protocol.SymbolKindClass
	case "function":
		return protocol.SymbolKindFunction
	case "method":
		return protocol.SymbolKindMethod
	case "field":
		return protocol.SymbolKindField
	default:
		return protocol.SymbolKindVariable
	}
}

// buildContainerName builds the full container name for a symbol.
//
// Takes packageName (string) which is the package name prefix.
// Takes containerName (string) which is the container name to add.
//
// Returns string which is the full container name in the format
// "pkg.container",
// or just the package name if containerName is empty.
func buildContainerName(packageName, containerName string) string {
	if containerName != "" {
		return packageName + "." + containerName
	}
	return packageName
}

// buildSymbolLocation creates an LSP Location for a symbol.
//
// Takes filePath (string) which is the path to the file containing the symbol.
// Takes line (int) which is the one-based line number of the symbol.
// Takes column (int) which is the one-based column number of the symbol.
// Takes name (string) which is the symbol name, used to calculate the range
// end position.
//
// Returns protocol.Location which holds the symbol's position as an LSP
// Location with zero-based line and character offsets.
func buildSymbolLocation(filePath string, line, column int, name string) protocol.Location {
	return protocol.Location{
		URI: protocol.DocumentURI("file://" + filePath),
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(line - 1),
				Character: safeconv.IntToUint32(column - 1),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(line - 1),
				Character: safeconv.IntToUint32(column - 1 + len(name)),
			},
		},
	}
}

// searchDocumentForIDs searches a single document for elements with IDs that
// match the query.
//
// Takes uri (protocol.DocumentURI) which identifies the document location.
// Takes document (*document) which is the document to search.
// Takes queryLower (string) which is the lowercase search query to match.
//
// Returns []protocol.SymbolInformation which contains matching ID symbols, or
// nil if the document has no annotations.
func searchDocumentForIDs(uri protocol.DocumentURI, document *document, queryLower string) []protocol.SymbolInformation {
	if document.AnnotationResult == nil || document.AnnotationResult.AnnotatedAST == nil {
		return nil
	}

	var symbols []protocol.SymbolInformation
	document.AnnotationResult.AnnotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if sym := findIDSymbolInNode(uri, node, queryLower); sym != nil {
			symbols = append(symbols, *sym)
		}
		return true
	})
	return symbols
}

// findIDSymbolInNode searches a node for an ID attribute that matches the
// query.
//
// Takes uri (protocol.DocumentURI) which is the document location.
// Takes node (*ast_domain.TemplateNode) which is the node to search.
// Takes queryLower (string) which is the lowercase search term.
//
// Returns *protocol.SymbolInformation which holds the matching ID symbol, or
// nil if no match is found.
func findIDSymbolInNode(uri protocol.DocumentURI, node *ast_domain.TemplateNode, queryLower string) *protocol.SymbolInformation {
	for i := range node.Attributes {
		attribute := &node.Attributes[i]
		if attribute.Name == "id" && strings.Contains(strings.ToLower(attribute.Value), queryLower) {
			return &protocol.SymbolInformation{
				Name: attribute.Value,
				Kind: protocol.SymbolKindClass,
				Location: protocol.Location{
					URI: uri,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      safeconv.IntToUint32(node.Location.Line - 1),
							Character: safeconv.IntToUint32(node.Location.Column - 1),
						},
						End: protocol.Position{
							Line:      safeconv.IntToUint32(node.Location.Line - 1),
							Character: safeconv.IntToUint32(node.Location.Column + len(node.TagName)),
						},
					},
				},
			}
		}
	}
	return nil
}
