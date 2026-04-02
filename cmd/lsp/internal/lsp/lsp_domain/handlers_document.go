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
	"bytes"
	"context"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxFormattingBytes is a large line number used to represent "end of
	// document" in formatting ranges. It covers the entire document when
	// replacing content.
	maxFormattingBytes = 1_000_000

	// logKeyOriginalSize is the log attribute key for document size before formatting.
	logKeyOriginalSize = "originalSize"

	// logKeyFormattedSize is the log key for the size of the formatted content.
	logKeyFormattedSize = "formattedSize"
)

// DidOpen handles notification that a text document was opened in the editor.
// This method satisfies the protocol.Server interface by delegating to
// DidOpenTextDocument.
//
// Takes params (*protocol.DidOpenTextDocumentParams) which contains the
// document URI and content.
//
// Returns error when the document cannot be processed.
func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	return s.DidOpenTextDocument(ctx, params)
}

// DidChange handles text document change notifications from the client.
// This method satisfies the protocol.Server interface by delegating to
// DidChangeTextDocument.
//
// Takes params (*protocol.DidChangeTextDocumentParams) which contains the
// document URI and the content changes to apply.
//
// Returns error when the document change cannot be processed.
func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	return s.DidChangeTextDocument(ctx, params)
}

// DidSave handles notification that a document was saved.
// This method satisfies the protocol.Server interface by delegating to
// DidSaveTextDocument.
//
// Takes params (*protocol.DidSaveTextDocumentParams) which contains the
// document URI and optional text content.
//
// Returns error when the save notification cannot be processed.
func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	return s.DidSaveTextDocument(ctx, params)
}

// DidClose handles notification that a text document was closed in the editor.
// This method satisfies the protocol.Server interface by delegating to
// DidCloseTextDocument.
//
// Takes params (*protocol.DidCloseTextDocumentParams) which identifies the
// closed document.
//
// Returns error when the underlying DidCloseTextDocument call fails.
func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	return s.DidCloseTextDocument(ctx, params)
}

// WillSave handles the pre-save notification for a document.
//
// Takes params (*protocol.WillSaveTextDocumentParams) which holds the document
// identifier and the reason for the save.
//
// Returns error when the pre-save handling fails.
func (*Server) WillSave(ctx context.Context, params *protocol.WillSaveTextDocumentParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("WillSave", logger_domain.String(keyURI, params.TextDocument.URI.Filename()), logger_domain.Int("reason", int(params.Reason)))

	return nil
}

// WillSaveWaitUntil handles the pre-save notification and returns edits to
// apply before saving.
//
// Takes params (*protocol.WillSaveTextDocumentParams) which identifies the
// document about to be saved.
//
// Returns []protocol.TextEdit which contains formatting edits to apply, or an
// empty slice if no changes are needed.
// Returns error when the operation fails.
func (s *Server) WillSaveWaitUntil(ctx context.Context, params *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("WillSaveWaitUntil", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	if !s.formattingEnabled {
		l.Debug("WillSaveWaitUntil skipped: formatting is disabled",
			logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return []protocol.TextEdit{}, nil
	}

	content, found := s.docCache.Get(params.TextDocument.URI)
	if !found {
		l.Warn("Document not found in cache for WillSaveWaitUntil", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return []protocol.TextEdit{}, nil
	}

	formatted, err := s.formatter.Format(ctx, content)
	if err != nil {
		l.Error("Formatting failed in WillSaveWaitUntil", logger_domain.Error(err), logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return []protocol.TextEdit{}, nil
	}

	if bytes.Equal(formatted, content) {
		l.Debug("Content unchanged after formatting", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return []protocol.TextEdit{}, nil
	}

	textEdit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: maxFormattingBytes, Character: 0},
		},
		NewText: string(formatted),
	}

	l.Debug("WillSaveWaitUntil formatting completed successfully",
		logger_domain.String(keyURI, params.TextDocument.URI.Filename()),
		logger_domain.Int(logKeyOriginalSize, len(content)),
		logger_domain.Int(logKeyFormattedSize, len(formatted)))

	return []protocol.TextEdit{textEdit}, nil
}

// Formatting handles requests to format an entire document.
//
// Takes params (*protocol.DocumentFormattingParams) which specifies the
// document to format and formatting options.
//
// Returns []protocol.TextEdit which contains the edits to apply, or nil if the
// document is not found or formatting fails.
// Returns error when the request cannot be processed.
func (s *Server) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Formatting request received", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	if !s.formattingEnabled {
		l.Debug("Formatting skipped: formatting is disabled",
			logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return []protocol.TextEdit{}, nil
	}

	content, found := s.docCache.Get(params.TextDocument.URI)
	if !found {
		l.Warn("Document not found in cache for formatting", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return nil, nil
	}

	formatted, err := s.formatter.Format(ctx, content)
	if err != nil {
		l.Error("Formatting failed", logger_domain.Error(err), logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return nil, nil
	}

	textEdit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: maxFormattingBytes, Character: 0},
		},
		NewText: string(formatted),
	}

	l.Debug("Formatting completed successfully",
		logger_domain.String(keyURI, params.TextDocument.URI.Filename()),
		logger_domain.Int(logKeyOriginalSize, len(content)),
		logger_domain.Int(logKeyFormattedSize, len(formatted)))

	return []protocol.TextEdit{textEdit}, nil
}

// RangeFormatting handles requests to format a specific range within a document.
//
// Takes params (*protocol.DocumentRangeFormattingParams) which specifies the
// document and range to format.
//
// Returns []protocol.TextEdit which contains the edits to apply the formatting.
// Returns error when the formatting fails.
func (s *Server) RangeFormatting(ctx context.Context, params *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("RangeFormatting request received",
		logger_domain.String(keyURI, params.TextDocument.URI.Filename()),
		logger_domain.Int("startLine", int(params.Range.Start.Line)),
		logger_domain.Int("endLine", int(params.Range.End.Line)))

	if !s.formattingEnabled {
		l.Debug("RangeFormatting skipped: formatting is disabled",
			logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return []protocol.TextEdit{}, nil
	}

	content, found := s.docCache.Get(params.TextDocument.URI)
	if !found {
		l.Warn("Document not found in cache for range formatting", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return nil, nil
	}

	formatRange := formatter_domain.Range{
		StartLine:      params.Range.Start.Line,
		StartCharacter: params.Range.Start.Character,
		EndLine:        params.Range.End.Line,
		EndCharacter:   params.Range.End.Character,
	}

	formatted, err := s.formatter.FormatRange(ctx, content, formatRange, nil)
	if err != nil {
		l.Error("Range formatting failed", logger_domain.Error(err), logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return nil, nil
	}

	textEdit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: maxFormattingBytes, Character: 0},
		},
		NewText: string(formatted),
	}

	l.Debug("Range formatting completed successfully",
		logger_domain.String(keyURI, params.TextDocument.URI.Filename()),
		logger_domain.Int(logKeyOriginalSize, len(content)),
		logger_domain.Int(logKeyFormattedSize, len(formatted)))

	return []protocol.TextEdit{textEdit}, nil
}

// OnTypeFormatting handles requests to format text as the user types.
//
// Takes params (*protocol.DocumentOnTypeFormattingParams) which contains the
// document, position, and trigger character for formatting.
//
// Returns []protocol.TextEdit which contains the edits to apply. Returns nil
// if the document is not in the cache or formatting fails.
// Returns error when something goes wrong that cannot be fixed. Most errors
// are handled inside by returning nil edits instead.
func (s *Server) OnTypeFormatting(ctx context.Context, params *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("OnTypeFormatting request received",
		logger_domain.String(keyURI, params.TextDocument.URI.Filename()),
		logger_domain.Field(keyPosition, params.Position),
		logger_domain.String("char", params.Ch))

	if !s.formattingEnabled {
		l.Debug("OnTypeFormatting skipped: formatting is disabled",
			logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return []protocol.TextEdit{}, nil
	}

	content, found := s.docCache.Get(params.TextDocument.URI)
	if !found {
		l.Warn("Document not found in cache for on-type formatting", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return nil, nil
	}

	formatRange := calculateFormatRange(params.Ch, params.Position.Line)

	formatted, err := s.formatter.FormatRange(ctx, content, formatRange, nil)
	if err != nil {
		l.Error("On-type formatting failed", logger_domain.Error(err), logger_domain.String(keyURI, params.TextDocument.URI.Filename()))
		return nil, nil
	}

	textEdit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: maxFormattingBytes, Character: 0},
		},
		NewText: string(formatted),
	}

	l.Debug("On-type formatting completed successfully",
		logger_domain.String(keyURI, params.TextDocument.URI.Filename()),
		logger_domain.Int(logKeyOriginalSize, len(content)),
		logger_domain.Int(logKeyFormattedSize, len(formatted)))

	return []protocol.TextEdit{textEdit}, nil
}

// DidOpenTextDocument handles the notification when a document is opened.
//
// Takes params (*protocol.DidOpenTextDocumentParams) which contains the
// document URI and text content.
//
// Returns error when the operation fails.
func (s *Server) DidOpenTextDocument(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidOpen", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	s.workspace.UpdateDocument(params.TextDocument.URI, []byte(params.TextDocument.Text))

	uri := params.TextDocument.URI
	s.goBackground(func(ctx context.Context) {
		_, l := logger_domain.From(ctx, log)
		if _, err := s.workspace.RunAnalysisForURI(ctx, uri); err != nil {
			l.Error("Failed to run analysis on document open", logger_domain.Error(err))
		}
	})

	return nil
}

// DidChangeTextDocument handles the notification when a document's content
// changes.
//
// Takes params (*protocol.DidChangeTextDocumentParams) which contains the
// document URI and the content changes to apply.
//
// Returns error when the document change cannot be processed.
func (s *Server) DidChangeTextDocument(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidChange", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	if len(params.ContentChanges) > 0 {
		s.workspace.UpdateDocument(params.TextDocument.URI, []byte(params.ContentChanges[0].Text))

		uri := params.TextDocument.URI
		s.goBackground(func(ctx context.Context) {
			_, l := logger_domain.From(ctx, log)
			if _, err := s.workspace.RunAnalysisForURI(ctx, uri); err != nil {
				l.Error("Failed to run analysis on document change", logger_domain.Error(err))
			}
		})
	}

	return nil
}

// DidSaveTextDocument handles the notification when a document is saved.
//
// Takes params (*protocol.DidSaveTextDocumentParams) which contains the saved
// document URI and optionally the document text.
//
// Returns error when the save notification cannot be processed.
func (s *Server) DidSaveTextDocument(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("DidSave", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	if params.Text != "" {
		s.workspace.UpdateDocument(params.TextDocument.URI, []byte(params.Text))
	}

	uri := params.TextDocument.URI
	s.goBackground(func(ctx context.Context) {
		_, l := logger_domain.From(ctx, log)
		if _, err := s.workspace.RunAnalysisForURI(ctx, uri); err != nil {
			l.Error("Failed to run analysis on document save", logger_domain.Error(err))
		}
	})

	return nil
}

// DidCloseTextDocument handles the notification when a text document is closed.
//
// Takes params (*protocol.DidCloseTextDocumentParams) which identifies the
// document that was closed.
//
// Returns error when the close notification cannot be processed.
func (s *Server) DidCloseTextDocument(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("DidClose", logger_domain.String(keyURI, params.TextDocument.URI.Filename()))

	s.workspace.RemoveDocument(ctx, params.TextDocument.URI)

	return nil
}

// calculateFormatRange works out the range of lines to format based on the
// trigger character. For newline and closing tag triggers, it includes the
// previous line for context.
//
// Takes triggerChar (string) which is the character that triggered formatting.
// Takes currentLine (uint32) which is the line number where formatting starts.
//
// Returns formatter_domain.Range which specifies the lines to format.
func calculateFormatRange(triggerChar string, currentLine uint32) formatter_domain.Range {
	switch triggerChar {
	case "\n", ">":
		startLine := currentLine
		if startLine > 0 {
			startLine--
		}
		return formatter_domain.Range{
			StartLine:      startLine,
			StartCharacter: 0,
			EndLine:        currentLine + 1,
			EndCharacter:   0,
		}
	default:
		return formatter_domain.Range{
			StartLine:      currentLine,
			StartCharacter: 0,
			EndLine:        currentLine + 1,
			EndCharacter:   0,
		}
	}
}
