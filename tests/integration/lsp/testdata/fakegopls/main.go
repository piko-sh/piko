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

package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"go.lsp.dev/jsonrpc2"
)

const (
	modeWedge      = "wedge"
	modeCrash      = "crash"
	modeCrashOnce  = "crashonce"
	modeOldVersion = "oldversion"
	modeNullInit   = "nullinit"
	supportedVersion = "v0.21.0"
	tooOldVersion    = "v0.11.9"
	crashDelay = 750 * time.Millisecond
	crashOnceDelay = 1500 * time.Millisecond
	crashMarkerEnv = "FAKEGOPLS_CRASH_MARKER"
	syntheticDiagnosticMessage = "fakegopls synthetic diagnostic"
)

type stdioStream struct{}

func (stdioStream) Read(payload []byte) (int, error)  { return os.Stdin.Read(payload) }
func (stdioStream) Write(payload []byte) (int, error) { return os.Stdout.Write(payload) }
func (stdioStream) Close() error                      { return nil }

type fakeServer struct {
	protocol.Server
	mode    string
	crasher bool
	mu      sync.Mutex
	client  protocol.Client
}

func (s *fakeServer) setClient(client protocol.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client
}

func (s *fakeServer) currentClient() protocol.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client
}

func (s *fakeServer) Initialize(_ context.Context, _ *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	switch s.mode {
	case modeNullInit:
		return nil, nil
	case modeOldVersion:
		return &protocol.InitializeResult{ServerInfo: &protocol.ServerInfo{Name: "fakegopls", Version: tooOldVersion}}, nil
	default:
		return &protocol.InitializeResult{ServerInfo: &protocol.ServerInfo{Name: "fakegopls", Version: supportedVersion}}, nil
	}
}

func (s *fakeServer) Initialized(_ context.Context, _ *protocol.InitializedParams) error {
	s.signalReady()
	if s.mode == modeCrash {
		go func() {
			time.Sleep(crashDelay)
			os.Exit(1)
		}()
	}
	return nil
}

func (*fakeServer) Shutdown(_ context.Context) error { return nil }

func (*fakeServer) Exit(_ context.Context) error {
	os.Exit(0)
	return nil
}

func (s *fakeServer) DidOpen(_ context.Context, params *protocol.DidOpenTextDocumentParams) error {
	if s.mode == modeCrashOnce {
		s.publishSyntheticDiagnostic(params.TextDocument.URI)
		if s.crasher {
			go func() {
				time.Sleep(crashOnceDelay)
				os.Exit(1)
			}()
		}
	}
	return nil
}

func (*fakeServer) DidChange(_ context.Context, _ *protocol.DidChangeTextDocumentParams) error {
	return nil
}

func (*fakeServer) DidClose(_ context.Context, _ *protocol.DidCloseTextDocumentParams) error {
	return nil
}

func (s *fakeServer) Hover(ctx context.Context, _ *protocol.HoverParams) (*protocol.Hover, error) {
	if s.mode == modeWedge {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return nil, nil
}

func (s *fakeServer) Completion(ctx context.Context, _ *protocol.CompletionParams) (*protocol.CompletionList, error) {
	if s.mode == modeWedge {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return nil, nil
}

func (s *fakeServer) Definition(ctx context.Context, _ *protocol.DefinitionParams) ([]protocol.Location, error) {
	if s.mode == modeWedge {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return nil, nil
}

func (s *fakeServer) signalReady() {
	client := s.currentClient()
	if client == nil {
		return
	}
	_ = client.Progress(context.Background(), &protocol.ProgressParams{
		Token: *protocol.NewProgressToken("fakegopls/load"),
		Value: map[string]any{"kind": "end"},
	})
}

func (s *fakeServer) publishSyntheticDiagnostic(uri protocol.DocumentURI) {
	client := s.currentClient()
	if client == nil {
		return
	}
	_ = client.PublishDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
		URI: uri,
		Diagnostics: []protocol.Diagnostic{{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 1},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "compiler",
			Message:  syntheticDiagnosticMessage,
		}},
	})
}

func claimCrasher(markerPath string) bool {
	if markerPath == "" {
		return true
	}
	file, err := os.OpenFile(markerPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func main() {
	mode := os.Getenv("FAKEGOPLS_MODE")
	server := &fakeServer{mode: mode}
	if mode == modeCrashOnce {
		server.crasher = claimCrasher(os.Getenv(crashMarkerEnv))
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	_, conn, client := protocol.NewServer(context.Background(), server, jsonrpc2.NewStream(stdioStream{}), logger)
	server.setClient(client)
	<-conn.Done()
}
