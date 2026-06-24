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

package gopls_bridge

import (
	"context"
	"path/filepath"
	"sync"

	protocol "github.com/politepixels/golang-language-server"
	"piko.sh/piko/internal/logger/logger_domain"
)

// diagnosticsSink receives gopls diagnostics for a virtual document so the bridge can
// signal overlay readiness and forward them for windowing, remapping and merging.
type diagnosticsSink func(ctx context.Context, params *protocol.PublishDiagnosticsParams)

// clientHandler implements protocol.Client: the server-to-client half that gopls calls
// back into. piko-lsp is the client of its gopls child, so these callbacks are inert
// except for detecting workspace-load completion and forwarding diagnostics.
type clientHandler struct {
	// sink receives gopls diagnostics for forwarding; nil until installed.
	sink diagnosticsSink

	// ready is closed once gopls finishes its initial workspace load.
	ready chan struct{}

	// moduleRoot is the single module directory this child serves.
	moduleRoot string

	// readyOnce guards closing the ready channel exactly once.
	readyOnce sync.Once

	// mu guards access to the sink field.
	mu sync.Mutex
}

// newClientHandler creates a client handler for one gopls child.
//
// Takes moduleRoot (string) which is the module directory this child serves.
//
// Returns the configured client handler.
func newClientHandler(moduleRoot string) *clientHandler {
	return &clientHandler{
		ready:      make(chan struct{}),
		moduleRoot: moduleRoot,
	}
}

// Progress observes gopls work-done progress. The terminal "end" of the workspace setup
// token marks the child ready for overlays.
//
// Takes params (*protocol.ProgressParams) which carries the progress kind and token.
//
// Returns error which is always nil; progress notifications never fail.
func (h *clientHandler) Progress(ctx context.Context, params *protocol.ProgressParams) error {
	value, ok := params.Value.(map[string]any)
	if !ok {
		_, l := logger_domain.From(ctx, log)
		l.Trace("gopls progress value was not the expected object; readiness relies on the WaitReady timeout",
			logger_domain.String("moduleRoot", h.moduleRoot))
		return nil
	}
	if kind, isString := value["kind"].(string); isString && kind == "end" {
		h.markReady()
	}
	return nil
}

// WorkDoneProgressCreate accepts a progress token gopls intends to use.
//
// Returns error which is always nil; the token is accepted unconditionally.
func (*clientHandler) WorkDoneProgressCreate(_ context.Context, _ *protocol.WorkDoneProgressCreateParams) error {
	return nil
}

// LogMessage forwards gopls log output to the bridge logger at debug level.
//
// Takes params (*protocol.LogMessageParams) which carries the log message text.
//
// Returns error which is always nil; logging never fails.
func (*clientHandler) LogMessage(ctx context.Context, params *protocol.LogMessageParams) error {
	_, l := logger_domain.From(ctx, log)
	l.Debug("gopls log", logger_domain.String("message", params.Message))
	return nil
}

// PublishDiagnostics hands gopls diagnostics to the configured sink, if any.
//
// Takes params (*protocol.PublishDiagnosticsParams) which carries the diagnostics to
// forward.
//
// Returns error which is always nil; forwarding never fails.
//
// Concurrency: acquires mu to read the sink before invoking it outside the lock.
func (h *clientHandler) PublishDiagnostics(ctx context.Context, params *protocol.PublishDiagnosticsParams) error {
	h.mu.Lock()
	sink := h.sink
	h.mu.Unlock()
	if sink != nil {
		sink(ctx, params)
	}
	return nil
}

// ShowMessage logs a gopls user-facing message.
//
// Takes params (*protocol.ShowMessageParams) which carries the message text.
//
// Returns error which is always nil; logging never fails.
func (*clientHandler) ShowMessage(ctx context.Context, params *protocol.ShowMessageParams) error {
	_, l := logger_domain.From(ctx, log)
	l.Debug("gopls message", logger_domain.String("message", params.Message))
	return nil
}

// ShowMessageRequest declines gopls message-request prompts. A nil action with a nil
// error is the protocol's representation of declining to pick an item.
//
// Returns *protocol.MessageActionItem which is always nil to decline the prompt.
// Returns error which is always nil; declining never fails.
func (*clientHandler) ShowMessageRequest(_ context.Context, _ *protocol.ShowMessageRequestParams) (*protocol.MessageActionItem, error) {
	return nil, nil
}

// Telemetry ignores gopls telemetry events.
//
// Returns error which is always nil; telemetry is discarded.
func (*clientHandler) Telemetry(_ context.Context, _ any) error {
	return nil
}

// RegisterCapability accepts dynamic capability registrations from gopls.
//
// Returns error which is always nil; registrations are accepted unconditionally.
func (*clientHandler) RegisterCapability(_ context.Context, _ *protocol.RegistrationParams) error {
	return nil
}

// UnregisterCapability accepts dynamic capability unregistrations from gopls.
//
// Returns error which is always nil; unregistrations are accepted unconditionally.
func (*clientHandler) UnregisterCapability(_ context.Context, _ *protocol.UnregistrationParams) error {
	return nil
}

// ApplyEdit refuses workspace edits originated by gopls; piko-lsp owns edits applied to
// the user's .pk files.
//
// Returns *protocol.ApplyWorkspaceEditResponse which refuses the gopls edit.
// Returns error which is always nil; refusal never fails.
func (*clientHandler) ApplyEdit(_ context.Context, _ *protocol.ApplyWorkspaceEditParams) (*protocol.ApplyWorkspaceEditResponse, error) {
	return &protocol.ApplyWorkspaceEditResponse{Applied: false}, nil
}

// Configuration returns default (nil) configuration for every requested item so gopls
// proceeds with its built-in defaults.
//
// Takes params (*protocol.ConfigurationParams) which lists the configuration items
// requested.
//
// Returns []any which holds one nil entry per requested item.
// Returns error which is always nil; default configuration never fails.
func (*clientHandler) Configuration(_ context.Context, params *protocol.ConfigurationParams) ([]any, error) {
	return make([]any, len(params.Items)), nil
}

// WorkspaceFolders reports the single module root this child serves.
//
// Returns []protocol.WorkspaceFolder which holds the one folder for this module.
// Returns error which is always nil; reporting the folder never fails.
func (h *clientHandler) WorkspaceFolders(_ context.Context) ([]protocol.WorkspaceFolder, error) {
	return []protocol.WorkspaceFolder{{
		URI:  string(fileURI(h.moduleRoot)),
		Name: filepath.Base(h.moduleRoot),
	}}, nil
}

// setSink installs the diagnostics callback. Safe for concurrent use.
//
// Takes sink (diagnosticsSink) which receives forwarded gopls diagnostics.
func (h *clientHandler) setSink(sink diagnosticsSink) {
	h.mu.Lock()
	h.sink = sink
	h.mu.Unlock()
}

// markReady closes the ready channel exactly once, signalling that gopls has finished its
// initial workspace load.
func (h *clientHandler) markReady() {
	h.readyOnce.Do(func() { close(h.ready) })
}
