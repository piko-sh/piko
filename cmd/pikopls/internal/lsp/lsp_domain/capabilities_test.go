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
	"reflect"
	"sync"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	unadvertisedServerMethods = map[string]string{
		"Initialize":                "lifecycle request, always available",
		"Initialized":               "lifecycle notification",
		"Shutdown":                  "lifecycle request",
		"Exit":                      "lifecycle notification",
		"SetTrace":                  "always-available notification",
		"LogTrace":                  "always-available notification",
		"WorkDoneProgressCancel":    "always-available notification",
		"Request":                   "dispatches inlay hints and type hierarchy, advertised by ExtendInitialiseResult",
		"SetClient":                 "composition-root wiring, not an LSP method",
		"SetConn":                   "composition-root wiring, not an LSP method",
		"DidOpenTextDocument":       "internal helper behind DidOpen",
		"DidChangeTextDocument":     "internal helper behind DidChange",
		"DidSaveTextDocument":       "internal helper behind DidSave",
		"DidCloseTextDocument":      "internal helper behind DidClose",
		"DidChangeConfiguration":    "always-available notification",
		"DidChangeWatchedFiles":     "registered dynamically in Initialized",
		"DidChangeWorkspaceFolders": "workspace folders are not advertised; the handler only logs",
		"CodeLens":                  "empty stub, code lens is not advertised",
		"CodeLensResolve":           "empty stub, code lens is not advertised",
		"CodeLensRefresh":           "client-direction method required by protocol.Server",
		"CompletionResolve":         "completion advertises resolveProvider false",
		"DocumentLinkResolve":       "document links advertise resolveProvider false",
		"PrepareCallHierarchy":      "empty stub, call hierarchy is not advertised",
		"IncomingCalls":             "empty stub, call hierarchy is not advertised",
		"OutgoingCalls":             "empty stub, call hierarchy is not advertised",
		"SemanticTokensFull":        "rejected with method-not-found, semantic tokens are not advertised",
		"SemanticTokensFullDelta":   "rejected with method-not-found, semantic tokens are not advertised",
		"SemanticTokensRange":       "rejected with method-not-found, semantic tokens are not advertised",
		"SemanticTokensRefresh":     "client-direction method required by protocol.Server",
		"ShowDocument":              "client-direction method required by protocol.Server",
		"WillCreateFiles":           "file operations are not advertised",
		"DidCreateFiles":            "file operations are not advertised",
		"WillRenameFiles":           "file operations are not advertised",
		"DidRenameFiles":            "file operations are not advertised",
		"WillDeleteFiles":           "file operations are not advertised",
		"DidDeleteFiles":            "file operations are not advertised",
	}
)

type registrationRecordingClient struct {
	mockClient
	err            error
	registrations  []*protocol.RegistrationParams
	mu             sync.Mutex
	waitForContext bool
}

func (c *registrationRecordingClient) RegisterCapability(ctx context.Context, params *protocol.RegistrationParams) error {
	c.mu.Lock()
	c.registrations = append(c.registrations, params)
	c.mu.Unlock()
	if c.waitForContext {
		<-ctx.Done()
		return ctx.Err()
	}
	return c.err
}

func (c *registrationRecordingClient) recorded() []*protocol.RegistrationParams {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*protocol.RegistrationParams(nil), c.registrations...)
}

func boolCapability(value any) bool {
	enabled, ok := value.(bool)
	return ok && enabled
}

func textDocumentSyncOptions(t *testing.T, capabilities protocol.ServerCapabilities) protocol.TextDocumentSyncOptions {
	t.Helper()
	options, ok := capabilities.TextDocumentSync.(protocol.TextDocumentSyncOptions)
	require.True(t, ok, "TextDocumentSync should be TextDocumentSyncOptions")
	return options
}

func advertisedServerMethods(t *testing.T) map[string]func(protocol.ServerCapabilities) bool {
	t.Helper()
	return map[string]func(protocol.ServerCapabilities) bool{
		"Hover":              func(c protocol.ServerCapabilities) bool { return boolCapability(c.HoverProvider) },
		"Definition":         func(c protocol.ServerCapabilities) bool { return boolCapability(c.DefinitionProvider) },
		"Declaration":        func(c protocol.ServerCapabilities) bool { return boolCapability(c.DeclarationProvider) },
		"TypeDefinition":     func(c protocol.ServerCapabilities) bool { return boolCapability(c.TypeDefinitionProvider) },
		"Implementation":     func(c protocol.ServerCapabilities) bool { return boolCapability(c.ImplementationProvider) },
		"References":         func(c protocol.ServerCapabilities) bool { return boolCapability(c.ReferencesProvider) },
		"DocumentHighlight":  func(c protocol.ServerCapabilities) bool { return boolCapability(c.DocumentHighlightProvider) },
		"DocumentSymbol":     func(c protocol.ServerCapabilities) bool { return boolCapability(c.DocumentSymbolProvider) },
		"Symbols":            func(c protocol.ServerCapabilities) bool { return boolCapability(c.WorkspaceSymbolProvider) },
		"DocumentLink":       func(c protocol.ServerCapabilities) bool { return c.DocumentLinkProvider != nil },
		"DocumentColor":      func(c protocol.ServerCapabilities) bool { return boolCapability(c.ColorProvider) },
		"ColorPresentation":  func(c protocol.ServerCapabilities) bool { return boolCapability(c.ColorProvider) },
		"Formatting":         func(c protocol.ServerCapabilities) bool { return boolCapability(c.DocumentFormattingProvider) },
		"RangeFormatting":    func(c protocol.ServerCapabilities) bool { return boolCapability(c.DocumentRangeFormattingProvider) },
		"OnTypeFormatting":   func(c protocol.ServerCapabilities) bool { return c.DocumentOnTypeFormattingProvider != nil },
		"FoldingRanges":      func(c protocol.ServerCapabilities) bool { return boolCapability(c.FoldingRangeProvider) },
		"LinkedEditingRange": func(c protocol.ServerCapabilities) bool { return boolCapability(c.LinkedEditingRangeProvider) },
		"Completion":         func(c protocol.ServerCapabilities) bool { return c.CompletionProvider != nil },
		"SignatureHelp":      func(c protocol.ServerCapabilities) bool { return c.SignatureHelpProvider != nil },
		"CodeAction":         func(c protocol.ServerCapabilities) bool { return boolCapability(c.CodeActionProvider) },
		"Moniker":            func(c protocol.ServerCapabilities) bool { return boolCapability(c.MonikerProvider) },
		"Rename":             func(c protocol.ServerCapabilities) bool { return c.RenameProvider != nil },
		"PrepareRename": func(c protocol.ServerCapabilities) bool {
			options, ok := c.RenameProvider.(*protocol.RenameOptions)
			return ok && options.PrepareProvider
		},
		"ExecuteCommand": func(c protocol.ServerCapabilities) bool {
			return c.ExecuteCommandProvider != nil && len(c.ExecuteCommandProvider.Commands) > 0
		},
		"DidOpen":  func(c protocol.ServerCapabilities) bool { return textDocumentSyncOptions(t, c).OpenClose },
		"DidClose": func(c protocol.ServerCapabilities) bool { return textDocumentSyncOptions(t, c).OpenClose },
		"DidChange": func(c protocol.ServerCapabilities) bool {
			return textDocumentSyncOptions(t, c).Change != protocol.TextDocumentSyncKindNone
		},
		"DidSave":           func(c protocol.ServerCapabilities) bool { return textDocumentSyncOptions(t, c).Save != nil },
		"WillSave":          func(c protocol.ServerCapabilities) bool { return textDocumentSyncOptions(t, c).WillSave },
		"WillSaveWaitUntil": func(c protocol.ServerCapabilities) bool { return textDocumentSyncOptions(t, c).WillSaveWaitUntil },
	}
}

func TestServerMethods_AreAdvertisedOrAllowListed(t *testing.T) {
	advertised := advertisedServerMethods(t)
	capabilities := buildServerCapabilities(true)
	serverType := reflect.TypeFor[*Server]()

	methodNames := make(map[string]bool, serverType.NumMethod())
	for method := range serverType.Methods() {
		name := method.Name
		methodNames[name] = true

		check, isAdvertised := advertised[name]
		_, isAllowListed := unadvertisedServerMethods[name]
		switch {
		case isAdvertised && isAllowListed:
			assert.Failf(t, "method in both lists", "%s is both advertised and allow-listed; keep it in one list", name)
		case isAdvertised:
			assert.Truef(t, check(capabilities), "%s is implemented but its capability is not advertised", name)
		case isAllowListed:
		default:
			assert.Failf(t, "unclassified method", "%s is an exported Server method that is neither advertised nor on the allow-list", name)
		}
	}

	for name := range advertised {
		assert.Truef(t, methodNames[name], "advertised method %s no longer exists on Server", name)
	}
	for name := range unadvertisedServerMethods {
		assert.Truef(t, methodNames[name], "allow-listed method %s no longer exists on Server", name)
	}
}

func TestBuildServerCapabilities_WillSaveWaitUntilFollowsFormatting(t *testing.T) {
	assert.True(t, textDocumentSyncOptions(t, buildServerCapabilities(true)).WillSaveWaitUntil)
	assert.False(t, textDocumentSyncOptions(t, buildServerCapabilities(false)).WillSaveWaitUntil)
	assert.Nil(t, buildServerCapabilities(true).SemanticTokensProvider, "semantic tokens must not be advertised")
}

func TestBuildServerCapabilities_FormattingEnabled(t *testing.T) {
	capabilities := buildServerCapabilities(true)

	assert.True(t, boolCapability(capabilities.DocumentFormattingProvider))
	assert.True(t, boolCapability(capabilities.DocumentRangeFormattingProvider))
	assert.NotNil(t, capabilities.DocumentOnTypeFormattingProvider)
}

func TestBuildServerCapabilities_FormattingDisabled(t *testing.T) {
	capabilities := buildServerCapabilities(false)

	assert.False(t, boolCapability(capabilities.DocumentFormattingProvider))
	assert.False(t, boolCapability(capabilities.DocumentRangeFormattingProvider))
	assert.Nil(t, capabilities.DocumentOnTypeFormattingProvider)
}

func TestBuildServerCapabilities_CoreCapabilities(t *testing.T) {
	capabilities := buildServerCapabilities(true)

	assert.True(t, boolCapability(capabilities.HoverProvider))
	assert.True(t, boolCapability(capabilities.DefinitionProvider))
	assert.True(t, boolCapability(capabilities.DocumentSymbolProvider))
	assert.True(t, boolCapability(capabilities.FoldingRangeProvider))
	assert.True(t, boolCapability(capabilities.CodeActionProvider))
	assert.NotNil(t, capabilities.CompletionProvider)
	assert.NotNil(t, capabilities.SignatureHelpProvider)
	assert.NotNil(t, capabilities.RenameProvider)
}

func TestBuildOnTypeFormattingOptions(t *testing.T) {
	options := buildOnTypeFormattingOptions(true)
	require.NotNil(t, options)
	assert.Equal(t, "\n", options.FirstTriggerCharacter)
	assert.NotEmpty(t, options.MoreTriggerCharacter)

	assert.Nil(t, buildOnTypeFormattingOptions(false), "on-type formatting is off when formatting is disabled")
}

func TestBuildServerCapabilities_CompletionTriggerCharacters(t *testing.T) {
	capabilities := buildServerCapabilities(true)

	require.NotNil(t, capabilities.CompletionProvider)
	triggers := capabilities.CompletionProvider.TriggerCharacters
	assert.Contains(t, triggers, ".", "member access")
	assert.Contains(t, triggers, "-", "directive completions")
	assert.Contains(t, triggers, "\"", "directive value completions")
}

func TestBuildServerCapabilities_SignatureHelpTriggerCharacters(t *testing.T) {
	capabilities := buildServerCapabilities(true)

	require.NotNil(t, capabilities.SignatureHelpProvider)
	assert.Contains(t, capabilities.SignatureHelpProvider.TriggerCharacters, "(")
}

func TestClientSupportsWatchedFilesRegistration(t *testing.T) {
	testCases := []struct {
		parameters *protocol.InitializeParams
		name       string
		expected   bool
	}{
		{name: "no initialisation parameters", parameters: nil},
		{name: "no workspace capabilities", parameters: &protocol.InitializeParams{}},
		{
			name: "no watched files capability",
			parameters: &protocol.InitializeParams{
				Capabilities: protocol.ClientCapabilities{Workspace: &protocol.WorkspaceClientCapabilities{}},
			},
		},
		{name: "dynamic registration disabled", parameters: watchedFilesParameters(false)},
		{name: "dynamic registration enabled", parameters: watchedFilesParameters(true), expected: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, clientSupportsWatchedFilesRegistration(testCase.parameters))
		})
	}
}

func TestBuildWatchedFilesRegistration(t *testing.T) {
	registration := buildWatchedFilesRegistration()

	require.Len(t, registration.Registrations, 1)
	entry := registration.Registrations[0]
	assert.Equal(t, watchedFilesRegistrationID, entry.ID)
	assert.Equal(t, methodDidChangeWatchedFiles, entry.Method)

	options, ok := entry.RegisterOptions.(protocol.DidChangeWatchedFilesRegistrationOptions)
	require.True(t, ok)
	require.Len(t, options.Watchers, len(watchedFileGlobs))
	for index, watcher := range options.Watchers {
		assert.Equal(t, watchedFileGlobs[index], watcher.GlobPattern)
		assert.Equal(t, watchedFileKinds, watcher.Kind)
	}
}

func TestRegisterWatchedFiles(t *testing.T) {
	t.Run("sends the registration to the client", func(t *testing.T) {
		client := &registrationRecordingClient{}

		registerWatchedFiles(t.Context(), client)

		assert.Len(t, client.recorded(), 1)
	})

	t.Run("tolerates a client that rejects the registration", func(t *testing.T) {
		client := &registrationRecordingClient{err: errors.New("registration rejected")}

		registerWatchedFiles(t.Context(), client)

		assert.Len(t, client.recorded(), 1)
	})

	t.Run("stops quietly when the server is shutting down", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		client := &registrationRecordingClient{waitForContext: true}

		registerWatchedFiles(ctx, client)

		assert.Len(t, client.recorded(), 1)
	})
}

func watchedFilesParameters(dynamicRegistration bool) *protocol.InitializeParams {
	return &protocol.InitializeParams{
		Capabilities: protocol.ClientCapabilities{
			Workspace: &protocol.WorkspaceClientCapabilities{
				DidChangeWatchedFiles: &protocol.DidChangeWatchedFilesWorkspaceClientCapabilities{DynamicRegistration: dynamicRegistration},
			},
		},
	}
}
