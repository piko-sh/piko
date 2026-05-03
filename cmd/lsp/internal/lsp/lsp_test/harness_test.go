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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"piko.sh/piko/wdk/json"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_adapters"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func newTestCacheService() cache_domain.Service {
	return cache_domain.NewService("")
}

type testCase struct {
	Name string
	Path string
}

type TopLevelTestSpec struct {
	Description string       `json:"description"`
	LSP         LSPSpec      `json:"lsp"`
	Generate    GenerateSpec `json:"generate"`
}

type GenerateSpec struct {
	PackageName       string           `json:"packageName,omitempty"`
	ErrorContains     string           `json:"errorContains,omitempty"`
	ExpectDiagnostics []DiagnosticSpec `json:"expectDiagnostics,omitempty"`
	IsPage            bool             `json:"isPage"`
	ShouldError       bool             `json:"shouldError,omitempty"`
}

type LSPSpec struct {
	Actions []LSPActionSpec `json:"actions"`
}

type LSPActionSpec struct {
	Action         string   `json:"action"`
	File           string   `json:"file,omitempty"`
	ExpectContains []string `json:"expectContains,omitempty"`
	Line           uint32   `json:"line,omitempty"`
	Character      uint32   `json:"character,omitempty"`
}

type DiagnosticSpec struct {
	MessageContains string `json:"messageContains"`
	Line            int    `json:"line,omitempty"`
}

type TestHarness struct {
	fsReader             annotator_domain.FSReaderPort
	resolver             resolver_domain.ResolverPort
	coordinatorService   coordinator_domain.CoordinatorService
	t                    testing.TB
	typeInspectorManager *inspector_domain.TypeBuilder
	pathsConfig          *config.PathsConfig
	tc                   testCase
	spec                 TopLevelTestSpec
	serverConfig         bootstrap.ServerConfig
}

func runTestCase(t *testing.T, tc testCase) {
	spec := loadTestSpec(t, tc)

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	harness := &TestHarness{
		t:    t,
		tc:   tc,
		spec: spec,
	}

	harness.setupServices(absSrcDir)

	if len(spec.LSP.Actions) > 0 {
		harness.runLSPTest()
	}
}

func (h *TestHarness) setupServices(absSrcDir string) {

	h.resolver = resolver_adapters.NewLocalModuleResolver(absSrcDir)
	err := h.resolver.DetectLocalModule(context.Background())
	require.NoError(h.t, err)

	h.serverConfig = bootstrap.ServerConfig{
		Paths: config.PathsConfig{
			BaseDir:             &absSrcDir,
			ComponentsSourceDir: new("components"),
			PagesSourceDir:      new("pages"),
			PartialsSourceDir:   new("partials"),
			EmailsSourceDir:     new("emails"),
			E2ESourceDir:        new("e2e"),
			AssetsSourceDir:     new("lib"),
			I18nSourceDir:       new("locales"),
			BaseServePath:       new(""),
			PartialServePath:    new("/_piko/partial"),
			ActionServePath:     new("/_piko/actions"),
			LibServePath:        new("/_piko/lib"),
			DistServePath:       new("/_piko/dist"),
			ArtefactServePath:   new("/_piko/assets"),
		},
	}

	h.pathsConfig = &h.serverConfig.Paths

	osReader := lsp_adapters.NewOsFSReader()
	h.fsReader = osReader

	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderLocalCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		h.resolver,
	)

	h.typeInspectorManager = inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: h.resolver.GetModuleName()},
		inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)),
	)

	annotatorComponentCache := annotator_adapters.NewComponentCache()
	annotatorService, err := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:      h.resolver,
		FSReader:      h.fsReader,
		TypeInspector: annotator_domain.NewTypeInspectorBuilderAdapter(h.typeInspectorManager),
		CSSProcessor:  cssProcessor,
		PathsConfig: annotator_domain.AnnotatorPathsConfig{
			PartialsSourceDir: *h.serverConfig.Paths.PartialsSourceDir,
			PagesSourceDir:    *h.serverConfig.Paths.PagesSourceDir,
			PartialServePath:  *h.serverConfig.Paths.PartialServePath,
		},
		Cache:               annotatorComponentCache,
		CompilationLogLevel: 0,
		CollectionService:   nil,
		EnableDebugLogFiles: true,
		DebugLogDir:         "tmp/logs",
	})
	require.NoError(h.t, err)

	cacheService := newTestCacheService()
	coordinatorCache, cacheErr := coordinator_adapters.NewBuildResultCache(context.Background(), cacheService)
	require.NoError(h.t, cacheErr)
	introspectionCache, cacheErr := coordinator_adapters.NewIntrospectionCache(context.Background(), cacheService)
	require.NoError(h.t, cacheErr)
	h.coordinatorService = coordinator_domain.NewService(
		context.Background(),
		annotatorService,
		coordinatorCache,
		introspectionCache,
		h.fsReader,
		h.resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
	)
	h.t.Cleanup(func() { h.coordinatorService.Shutdown(context.Background()) })

}

func (h *TestHarness) buildLSPServer() (*lsp_domain.Server, *lsp_domain.DocumentCache) {

	docCache := lsp_domain.NewDocumentCache()

	osReader := lsp_adapters.NewOsFSReader()
	lspReader, err := lsp_adapters.NewLspFSReader(docCache, osReader)
	require.NoError(h.t, err, "constructing LSP file reader")
	formatter := formatter_domain.NewFormatterService()

	pikoServer := lsp_domain.NewServer(lsp_domain.ServerDeps{
		Coordinator:          h.coordinatorService,
		Resolver:             h.resolver,
		TypeInspectorManager: h.typeInspectorManager,
		DocCache:             docCache,
		FSReader:             lspReader,
		PathsConfig:          h.pathsConfig,
		Formatter:            formatter,
		FormattingEnabled:    true,
	})

	return pikoServer, docCache
}

func (h *TestHarness) runLSPTest() {
	ctx := context.Background()

	clientReader, serverWriter, err := os.Pipe()
	require.NoError(h.t, err, "Failed to create client pipe")
	serverReader, clientWriter, err := os.Pipe()
	require.NoError(h.t, err, "Failed to create server pipe")

	clientStream := jsonrpc2.NewStream(struct {
		io.Reader
		io.WriteCloser
	}{Reader: clientReader, WriteCloser: clientWriter})

	serverStream := jsonrpc2.NewStream(struct {
		io.Reader
		io.WriteCloser
	}{Reader: serverReader, WriteCloser: serverWriter})

	mockClient := NewMockClient(h.t, clientStream)
	defer func() { _ = mockClient.Close() }()

	pikoServer, _ := h.buildLSPServer()

	go func() {
		defer func() {
			recover()
		}()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		_, conn, client := protocol.NewServer(ctx, pikoServer, serverStream, logger)
		pikoServer.SetClient(client)
		pikoServer.SetConn(conn)
		<-conn.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	rootURI := protocol.DocumentURI(uri.File(*h.serverConfig.Paths.BaseDir))

	initResult, err := mockClient.Initialize(ctx, rootURI)
	require.NoError(h.t, err, "Setup request should succeed")
	require.NotNil(h.t, initResult, "Setup result should not be nil")

	err = mockClient.Initialized(ctx)
	require.NoError(h.t, err, "Initialized notification should succeed")

	h.processLSPActions(ctx, mockClient)

	err = mockClient.Shutdown(ctx)
	require.NoError(h.t, err, "Shutdown should succeed")

	err = mockClient.Exit(ctx)
	require.NoError(h.t, err, "Exit should succeed")
}

func (h *TestHarness) processLSPActions(ctx context.Context, client *MockClient) {
	if len(h.spec.LSP.Actions) == 0 {
		return
	}

	for _, action := range h.spec.LSP.Actions {
		switch action.Action {
		case "didOpen":
			h.processDidOpen(ctx, client, action)
		case "hover":
			h.processHover(ctx, client, action)
		case "completion":
			h.processCompletion(ctx, client, action)
		case "definition":
			h.processDefinition(ctx, client, action)
		case "expectDiagnostics":
			h.processExpectDiagnostics(ctx, client, action)
		case "workspaceSymbol":
			h.processWorkspaceSymbol(ctx, client, action)
		case "rename":
			h.processRename(ctx, client, action)
		case "documentHighlight":
			h.processDocumentHighlight(ctx, client, action)
		case "foldingRange":
			h.processFoldingRange(ctx, client, action)
		case "codeAction":
			h.processCodeAction(ctx, client, action)
		default:
			h.t.Fatalf("Unknown LSP action: %s", action.Action)
		}
	}
}

func (h *TestHarness) processDidOpen(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))

	contentBytes, err := os.ReadFile(filePath)
	require.NoError(h.t, err, "Failed to read file: %s", action.File)

	err = client.DidOpen(ctx, fileURI, string(contentBytes))
	require.NoError(h.t, err, "DidOpen should succeed")

	analysisTimeout := 30 * time.Second
	if !client.WaitForAnalysisComplete(fileURI, analysisTimeout) {
		h.t.Fatalf("timed out after %v waiting for analysis to complete for %s", analysisTimeout, fileURI)
	}
}

func (h *TestHarness) processHover(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))
	position := protocol.Position{Line: action.Line, Character: action.Character}

	hover, err := client.Hover(ctx, fileURI, position)
	require.NoError(h.t, err, "Hover request should succeed")
	require.NotNil(h.t, hover, "Hover result should not be nil")

	hoverText := hover.Contents.Value

	for _, expected := range action.ExpectContains {
		assert.Contains(h.t, hoverText, expected, "Hover should contain: %s", expected)
	}
}

func (h *TestHarness) processCompletion(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))
	position := protocol.Position{Line: action.Line, Character: action.Character}

	completion, err := client.Completion(ctx, fileURI, position)
	require.NoError(h.t, err, "Completion request should succeed")
	require.NotNil(h.t, completion, "Completion result should not be nil")

	completionJSON, err := json.Marshal(completion)
	require.NoError(h.t, err, "Failed to marshal completion result")
	completionString := string(completionJSON)

	for _, expected := range action.ExpectContains {
		assert.Contains(h.t, completionString, expected, "Completion should contain: %s", expected)
	}
}

func (h *TestHarness) processDefinition(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))
	position := protocol.Position{Line: action.Line, Character: action.Character}

	locations, err := client.Definition(ctx, fileURI, position)
	require.NoError(h.t, err, "Definition request should succeed")
	require.NotEmpty(h.t, locations, "Definition should return at least one location")

	foundAllExpectations := true
	for _, expected := range action.ExpectContains {
		foundMatchInAnyLocation := false
		for _, loc := range locations {

			if strings.Contains(loc.URI.Filename(), expected) {
				foundMatchInAnyLocation = true
				break
			}

			if lineNumString, found := strings.CutPrefix(expected, "line:"); found {
				expectedLine, err := strconv.ParseUint(lineNumString, 10, 32)
				if err == nil {

					if uint32(expectedLine-1) == loc.Range.Start.Line {
						foundMatchInAnyLocation = true
						break
					}
				}
			}
		}
		if !foundMatchInAnyLocation {
			foundAllExpectations = false
			h.t.Errorf("Expected definition to contain '%s', but it was not found in any returned locations: %+v", expected, locations)
		}
	}
	assert.True(h.t, foundAllExpectations, "All expected definition criteria should be met")
}

func (h *TestHarness) processExpectDiagnostics(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))

	client.WaitForDiagnostics(fileURI, 5*time.Second)

	diagnostics := client.GetDiagnostics(fileURI)

	if len(action.ExpectContains) > 0 {
		for _, expected := range action.ExpectContains {
			found := false
			for _, diagnostic := range diagnostics {

				if strings.Contains(diagnostic.Message, expected) {
					found = true
					break
				}
			}
			assert.True(h.t, found, "Expected to find a diagnostic message containing: '%s'", expected)
		}
	}
}

func (h *TestHarness) processWorkspaceSymbol(ctx context.Context, client *MockClient, action LSPActionSpec) {
	query := ""
	if len(action.ExpectContains) > 0 {
		query = action.ExpectContains[0]
	}

	symbols, err := client.WorkspaceSymbol(ctx, query)
	require.NoError(h.t, err, "WorkspaceSymbol request should succeed")

	require.NotEmpty(h.t, symbols, "WorkspaceSymbol should return at least one symbol")
}

func (h *TestHarness) processRename(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))
	position := protocol.Position{Line: action.Line, Character: action.Character}

	newName := "renamedSymbol"
	if len(action.ExpectContains) > 0 {
		newName = action.ExpectContains[0]
	}

	edit, err := client.Rename(ctx, fileURI, position, newName)
	require.NoError(h.t, err, "Rename request should succeed")
	require.NotNil(h.t, edit, "Rename should return an edit")

	totalEdits := len(edit.DocumentChanges)
	for _, edits := range edit.Changes {
		totalEdits += len(edits)
	}
	require.NotZero(h.t, totalEdits, "Rename should return at least one edit (via Changes or DocumentChanges)")
}

func (h *TestHarness) processDocumentHighlight(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))
	position := protocol.Position{Line: action.Line, Character: action.Character}

	highlights, err := client.DocumentHighlight(ctx, fileURI, position)
	require.NoError(h.t, err, "DocumentHighlight request should succeed")

	require.NotEmpty(h.t, highlights, "DocumentHighlight should return at least one highlight")
}

func (h *TestHarness) processFoldingRange(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))

	foldingRanges, err := client.FoldingRange(ctx, fileURI)
	require.NoError(h.t, err, "FoldingRange request should succeed")

	require.NotEmpty(h.t, foldingRanges, "FoldingRange should return at least one range")
}

func (h *TestHarness) processCodeAction(ctx context.Context, client *MockClient, action LSPActionSpec) {
	filePath := filepath.Join(*h.serverConfig.Paths.BaseDir, action.File)
	fileURI := protocol.DocumentURI(uri.File(filePath))

	diagnostics := client.GetDiagnostics(fileURI)

	textRange := protocol.Range{
		Start: protocol.Position{Line: action.Line, Character: action.Character},
		End:   protocol.Position{Line: action.Line, Character: action.Character},
	}
	if len(diagnostics) > 0 {
		textRange = diagnostics[0].Range
	}

	codeActions, err := client.CodeAction(ctx, fileURI, textRange, diagnostics)
	require.NoError(h.t, err, "CodeAction request should succeed")

	require.NotEmpty(h.t, codeActions, "CodeAction should return at least one action")

	if len(action.ExpectContains) > 0 {
		for _, expected := range action.ExpectContains {
			found := false
			for _, ca := range codeActions {
				if strings.Contains(ca.Title, expected) {
					found = true
					break
				}
			}
			assert.True(h.t, found, "Expected to find a code action with title containing: '%s'", expected)
		}
	}
}

func loadTestSpec(t testing.TB, tc testCase) TopLevelTestSpec {
	specPath := filepath.Join(tc.Path, "testspec.json")
	data, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json for test case: %s", tc.Name)

	var spec TopLevelTestSpec
	err = json.Unmarshal(data, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for test case: %s", tc.Name)

	return spec
}
