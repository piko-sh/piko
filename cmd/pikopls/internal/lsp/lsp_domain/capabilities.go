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
	"errors"
	"fmt"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// commandRefreshDiagnostics is the workspace/executeCommand identifier that re-analyses
	// every open document.
	commandRefreshDiagnostics = "piko.refreshDiagnostics"

	// methodDidChangeWatchedFiles is the LSP method dynamically registered so the client
	// watches project files on the server's behalf.
	methodDidChangeWatchedFiles = "workspace/didChangeWatchedFiles"

	// watchedFilesRegistrationID identifies the dynamic file-watcher registration.
	watchedFilesRegistrationID = "piko-watched-files"

	// watchedFilesRegistrationTimeout bounds how long the server waits for the client to
	// acknowledge the dynamic file-watcher registration.
	watchedFilesRegistrationTimeout = 10 * time.Second

	// watchedFileKinds asks for create, change and delete events. WatchKind is a float in
	// the protocol library, so the flags are combined by addition.
	watchedFileKinds = protocol.WatchKindCreate + protocol.WatchKindChange + protocol.WatchKindDelete
)

var (
	// watchedFileGlobs are the glob patterns the client is asked to watch when it supports
	// dynamic registration of workspace/didChangeWatchedFiles.
	watchedFileGlobs = []string{"**/*.{pk,pkc,go}", "**/go.mod", "**/go.work"}

	// errWatchedFilesRegistrationTimeout is the cause attached to the bounded registration
	// context so a client that never answers is distinguishable from a shutdown.
	errWatchedFilesRegistrationTimeout = errors.New("client did not acknowledge file watcher registration in time")
)

// buildServerCapabilities creates the server capabilities for the LSP response.
//
// Takes formattingEnabled (bool) which controls whether formatting features, including
// willSaveWaitUntil, are shown to the client.
//
// Returns protocol.ServerCapabilities which lists the features this server supports,
// including text sync, hover, completion, and optionally formatting.
func buildServerCapabilities(formattingEnabled bool) protocol.ServerCapabilities {
	return protocol.ServerCapabilities{
		TextDocumentSync: protocol.TextDocumentSyncOptions{
			OpenClose:         true,
			Change:            protocol.TextDocumentSyncKindFull,
			Save:              &protocol.SaveOptions{IncludeText: true},
			WillSave:          true,
			WillSaveWaitUntil: formattingEnabled,
		},
		HoverProvider:                    true,
		DefinitionProvider:               true,
		DocumentSymbolProvider:           true,
		DocumentFormattingProvider:       formattingEnabled,
		DocumentRangeFormattingProvider:  formattingEnabled,
		DocumentOnTypeFormattingProvider: buildOnTypeFormattingOptions(formattingEnabled),
		FoldingRangeProvider:             true,
		LinkedEditingRangeProvider:       true,
		CompletionProvider:               &protocol.CompletionOptions{TriggerCharacters: []string{".", "-", "\""}, ResolveProvider: false},
		SignatureHelpProvider:            &protocol.SignatureHelpOptions{TriggerCharacters: []string{"(", ","}, RetriggerCharacters: []string{","}},
		CodeActionProvider:               true,
		RenameProvider:                   &protocol.RenameOptions{PrepareProvider: true},
		MonikerProvider:                  true,
		ImplementationProvider:           true,
		ReferencesProvider:               true,
		DocumentHighlightProvider:        true,
		DocumentLinkProvider:             &protocol.DocumentLinkOptions{ResolveProvider: false},
		ColorProvider:                    true,
		TypeDefinitionProvider:           true,
		DeclarationProvider:              true,
		WorkspaceSymbolProvider:          true,
		ExecuteCommandProvider:           &protocol.ExecuteCommandOptions{Commands: []string{commandRefreshDiagnostics}},
	}
}

// buildOnTypeFormattingOptions creates the on-type formatting options.
//
// Takes enabled (bool) which controls whether on-type formatting is available.
//
// Returns *protocol.DocumentOnTypeFormattingOptions which lists the characters that start
// formatting as the user types, or nil if disabled.
func buildOnTypeFormattingOptions(enabled bool) *protocol.DocumentOnTypeFormattingOptions {
	if !enabled {
		return nil
	}
	return &protocol.DocumentOnTypeFormattingOptions{
		FirstTriggerCharacter: "\n",
		MoreTriggerCharacter:  []string{">", "}"},
	}
}

// clientSupportsWatchedFilesRegistration reports whether the client declared dynamic
// registration support for workspace/didChangeWatchedFiles.
//
// Takes params (*protocol.InitializeParams) which holds the client capabilities, or nil
// before initialisation.
//
// Returns bool which is true when the client accepts a dynamic file-watcher registration.
func clientSupportsWatchedFilesRegistration(params *protocol.InitializeParams) bool {
	if params == nil {
		return false
	}
	workspace := params.Capabilities.Workspace
	if workspace == nil || workspace.DidChangeWatchedFiles == nil {
		return false
	}
	return workspace.DidChangeWatchedFiles.DynamicRegistration
}

// buildWatchedFilesRegistration builds the dynamic registration asking the client to
// watch Piko components, Go sources and module files.
//
// Returns *protocol.RegistrationParams which holds the single file-watcher registration.
func buildWatchedFilesRegistration() *protocol.RegistrationParams {
	watchers := make([]protocol.FileSystemWatcher, 0, len(watchedFileGlobs))
	for _, glob := range watchedFileGlobs {
		watchers = append(watchers, protocol.FileSystemWatcher{
			GlobPattern: glob,
			Kind:        watchedFileKinds,
		})
	}
	return &protocol.RegistrationParams{
		Registrations: []protocol.Registration{{
			ID:              watchedFilesRegistrationID,
			Method:          methodDidChangeWatchedFiles,
			RegisterOptions: protocol.DidChangeWatchedFilesRegistrationOptions{Watchers: watchers},
		}},
	}
}

// registerWatchedFiles asks the client to watch project files on the server's behalf.
//
// Takes client (protocol.Client) which receives the registration request.
func registerWatchedFiles(ctx context.Context, client protocol.Client) {
	ctx, l := logger_domain.From(ctx, log)

	registrationCtx, cancel := context.WithTimeoutCause(ctx, watchedFilesRegistrationTimeout, errWatchedFilesRegistrationTimeout)
	defer cancel()

	err := client.RegisterCapability(registrationCtx, buildWatchedFilesRegistration())
	if err == nil {
		l.Debug("Registered workspace file watchers", logger_domain.Int("globs", len(watchedFileGlobs)))
		return
	}
	if ctx.Err() != nil {
		l.Debug("Workspace file watcher registration stopped by shutdown", logger_domain.Error(err))
		return
	}
	if cause := context.Cause(registrationCtx); cause != nil && !errors.Is(err, cause) {
		err = fmt.Errorf("%w: %w", err, cause)
	}
	l.Warn("Registering workspace file watchers failed; external file changes will not refresh diagnostics",
		logger_domain.Error(err))
}
