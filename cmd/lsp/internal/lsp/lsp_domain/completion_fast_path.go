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

// tryFastPathCompletion attempts to provide completions using the existing
// document's cached analysis data, without waiting for a new analysis cycle.
// This enables immediate feedback for member access completions (e.g.,
// "state.", "props.", "item.") while the full analysis runs in the background.
//
// The key insight is that when a document becomes dirty, its previous
// AnalysisMap and TypeInspector remain valid for type resolution. We can
// reuse this existing data instead of waiting for a rebuild.
//
// Takes params (*protocol.CompletionParams) which specifies the completion
// request parameters.
//
// Returns *protocol.CompletionList which contains the completions if the
// fast-path succeeded, or nil if the fast-path is not available and the
// caller should fall back to the normal path.
func (s *Server) tryFastPathCompletion(ctx context.Context, params *protocol.CompletionParams) *protocol.CompletionList {
	_, l := logger_domain.From(ctx, log)

	uri := params.TextDocument.URI
	position := params.Position

	content, ok := s.workspace.docCache.Get(uri)
	if !ok {
		l.Trace("Fast-path completion: no content in cache",
			logger_domain.String(keyURI, uri.Filename()))
		return nil
	}

	triggerCtx := analyseCompletionContextFromContent(content, position)

	if triggerCtx.TriggerKind == triggerDirective {
		l.Debug("Fast-path completion: directive completions",
			logger_domain.String(keyURI, uri.Filename()),
			logger_domain.String("prefix", triggerCtx.Prefix))
		return getStaticDirectiveCompletions(triggerCtx.Prefix)
	}

	document := s.getDocumentForFastPath(ctx, uri)
	if document == nil {
		return nil
	}

	return s.handleFastPathTrigger(ctx, document, uri, position, triggerCtx)
}

// getDocumentForFastPath retrieves a document that is ready for fast-path
// completion.
//
// Takes uri (protocol.DocumentURI) which identifies the document.
//
// Returns *document which is the document if it exists and has the needed
// analysis data, or nil otherwise.
func (s *Server) getDocumentForFastPath(ctx context.Context, uri protocol.DocumentURI) *document {
	_, l := logger_domain.From(ctx, log)

	document, exists := s.workspace.GetDocument(uri)
	if !exists {
		l.Trace("Fast-path completion: no document found",
			logger_domain.String(keyURI, uri.Filename()))
		return nil
	}

	if !document.hasCompletionPrerequisites() {
		l.Trace("Fast-path completion: document lacks analysis data",
			logger_domain.String(keyURI, uri.Filename()))
		return nil
	}

	return document
}

// handleFastPathTrigger dispatches to the appropriate completion handler based
// on the trigger kind.
//
// Takes document (*document) which is the document with cached analysis data.
// Takes uri (protocol.DocumentURI) which identifies the document for logging.
// Takes position (protocol.Position) which is the cursor position.
// Takes triggerCtx (completionContext) which describes the completion trigger.
//
// Returns *protocol.CompletionList which contains completions, or nil if the
// trigger kind is not supported by the fast path.
func (s *Server) handleFastPathTrigger(
	ctx context.Context,
	document *document,
	uri protocol.DocumentURI,
	position protocol.Position,
	triggerCtx completionContext,
) *protocol.CompletionList {
	switch triggerCtx.TriggerKind {
	case triggerMemberAccess:
		return s.handleMemberAccessFastPath(ctx, document, uri, position, triggerCtx)
	case triggerDirectiveValue:
		return s.handleDirectiveValueFastPath(ctx, document, uri, position, triggerCtx)
	default:
		return nil
	}
}

// handleMemberAccessFastPath handles member access completions
// (e.g., "state.").
//
// Takes document (*document) which contains the parsed document state.
// Takes uri (protocol.DocumentURI) which identifies the document location.
// Takes position (protocol.Position) which specifies the cursor position.
// Takes triggerCtx (completionContext) which provides the completion context
// including base expression and prefix.
//
// Returns *protocol.CompletionList which contains the completion items, or nil
// when member resolution fails.
func (*Server) handleMemberAccessFastPath(
	ctx context.Context,
	document *document,
	uri protocol.DocumentURI,
	position protocol.Position,
	triggerCtx completionContext,
) *protocol.CompletionList {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Fast-path completion: attempting member access",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.String("baseExpr", triggerCtx.BaseExpression),
		logger_domain.String("prefix", triggerCtx.Prefix))

	result, err := document.getMemberCompletions(ctx, position, triggerCtx.BaseExpression, triggerCtx.Prefix)
	if err != nil || result == nil || len(result.Items) == 0 {
		l.Debug("Fast-path completion: member resolution failed",
			logger_domain.String("baseExpr", triggerCtx.BaseExpression))
		return nil
	}

	l.Debug("Fast-path completion: success",
		logger_domain.String("baseExpr", triggerCtx.BaseExpression),
		logger_domain.Int("item_count", len(result.Items)))

	result.IsIncomplete = true
	return result
}

// handleDirectiveValueFastPath handles directive value completions
// (e.g., p-if="").
//
// Takes document (*document) which contains the parsed document state.
// Takes uri (protocol.DocumentURI) which identifies the document being edited.
// Takes position (protocol.Position) which specifies the cursor position.
// Takes triggerCtx (completionContext) which provides the completion trigger
// context including the prefix.
//
// Returns *protocol.CompletionList which contains matching scope symbols, or
// nil when no completions are found.
func (*Server) handleDirectiveValueFastPath(
	ctx context.Context,
	document *document,
	uri protocol.DocumentURI,
	position protocol.Position,
	triggerCtx completionContext,
) *protocol.CompletionList {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Fast-path completion: directive value",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.String("prefix", triggerCtx.Prefix))

	result, err := document.getScopeCompletionsWithPrefix(position, triggerCtx.Prefix)
	if err != nil || result == nil || len(result.Items) == 0 {
		l.Debug("Fast-path completion: no scope symbols found")
		return nil
	}

	l.Debug("Fast-path completion: directive value success",
		logger_domain.Int("item_count", len(result.Items)))

	result.IsIncomplete = true
	return result
}

// analyseCompletionContextFromContent finds the completion context from raw
// content bytes without needing a full document. This is used for fast-path
// completion to detect member access triggers from the current text.
//
// Takes content ([]byte) which is the raw document content.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns completionContext which describes the detected completion type.
func analyseCompletionContextFromContent(content []byte, position protocol.Position) completionContext {
	ctx := completionContext{
		TriggerKind: triggerScope,
	}

	line, found := getLineAtPosition(content, position.Line)
	if !found {
		return ctx
	}
	if int(position.Character) > len(line) {
		return ctx
	}

	textBeforeCursor := line[:position.Character]

	if tryMemberAccessContext(&ctx, textBeforeCursor) {
		return ctx
	}

	if tryDirectiveContext(&ctx, textBeforeCursor) {
		return ctx
	}

	if tryDirectiveValueContext(&ctx, textBeforeCursor) {
		return ctx
	}

	return ctx
}
