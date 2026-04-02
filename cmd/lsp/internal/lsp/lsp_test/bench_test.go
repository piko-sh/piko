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

package lsp_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
)

type benchmarkEnv struct {
	client  *MockClient
	server  *lsp_domain.Server
	fileURI protocol.DocumentURI
	cleanup func()
}

func setupBenchmarkEnv(b *testing.B, testCaseName string) *benchmarkEnv {
	b.Helper()

	testCasePath := filepath.Join("./testdata", testCaseName)
	spec := loadTestSpec(b, testCase{Name: testCaseName, Path: testCasePath})

	srcDir := filepath.Join(testCasePath, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	if err != nil {
		b.Fatalf("failed to get absolute path: %v", err)
	}

	harness := &TestHarness{
		t:    b,
		tc:   testCase{Name: testCaseName, Path: testCasePath},
		spec: spec,
	}
	harness.setupServices(absSrcDir)

	ctx := context.Background()

	clientReader, serverWriter, err := os.Pipe()
	if err != nil {
		b.Fatalf("failed to create client pipe: %v", err)
	}
	serverReader, clientWriter, err := os.Pipe()
	if err != nil {
		b.Fatalf("failed to create server pipe: %v", err)
	}

	clientStream := jsonrpc2.NewStream(struct {
		io.Reader
		io.WriteCloser
	}{Reader: clientReader, WriteCloser: clientWriter})

	serverStream := jsonrpc2.NewStream(struct {
		io.Reader
		io.WriteCloser
	}{Reader: serverReader, WriteCloser: serverWriter})

	mockClient := NewMockClient(b, clientStream)

	pikoServer, _ := harness.buildLSPServer()

	go func() {
		defer func() { recover() }()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		_, conn, client := protocol.NewServer(ctx, pikoServer, serverStream, logger)
		pikoServer.SetClient(client)
		pikoServer.SetConn(conn)
		<-conn.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	rootURI := protocol.DocumentURI(uri.File(absSrcDir))
	initResult, err := mockClient.Initialize(ctx, rootURI)
	if err != nil || initResult == nil {
		b.Fatalf("initialize failed: %v", err)
	}

	if err := mockClient.Initialized(ctx); err != nil {
		b.Fatalf("initialized failed: %v", err)
	}

	var firstAction LSPActionSpec
	for _, action := range spec.LSP.Actions {
		if action.Action == "didOpen" {
			firstAction = action
			break
		}
	}

	filePath := filepath.Join(absSrcDir, firstAction.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		b.Fatalf("failed to read file: %v", err)
	}

	if err := mockClient.DidOpen(ctx, fileURI, string(contentBytes)); err != nil {
		b.Fatalf("didOpen failed: %v", err)
	}

	if !mockClient.WaitForAnalysisComplete(fileURI, 30*time.Second) {
		b.Fatal("timed out waiting for analysis to complete")
	}

	return &benchmarkEnv{
		client:  mockClient,
		server:  pikoServer,
		fileURI: fileURI,
		cleanup: func() {
			_ = mockClient.Shutdown(ctx)
			_ = mockClient.Exit(ctx)
			_ = mockClient.Close()
		},
	}
}

func BenchmarkLSP(b *testing.B) {
	env := setupBenchmarkEnv(b, "06_comprehensive_features")
	b.Cleanup(env.cleanup)

	ctx := context.Background()

	b.Run("Hover", func(b *testing.B) {
		position := protocol.Position{Line: 23, Character: 15}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := env.client.Hover(ctx, env.fileURI, position)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Completion", func(b *testing.B) {
		position := protocol.Position{Line: 25, Character: 14}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := env.client.Completion(ctx, env.fileURI, position)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("DocumentHighlight", func(b *testing.B) {
		position := protocol.Position{Line: 22, Character: 15}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := env.client.DocumentHighlight(ctx, env.fileURI, position)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("FoldingRange", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := env.client.FoldingRange(ctx, env.fileURI)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WorkspaceSymbol", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := env.client.WorkspaceSymbol(ctx, "Render")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
