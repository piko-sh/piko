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
	"io/fs"
	"sync"
	"testing"
	"time"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func createTestWorkspace() *workspace {
	docCache := NewDocumentCache()
	return &workspace{
		documents:    make(map[protocol.DocumentURI]*document),
		docCache:     docCache,
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
		client:       &mockClient{},
	}
}

func TestWorkspace_GetDocument_NotFound(t *testing.T) {
	ws := createTestWorkspace()

	document, exists := ws.GetDocument("file:///nonexistent.pk")

	if exists {
		t.Error("expected exists=false for nonexistent document")
	}
	if document != nil {
		t.Error("expected nil document for nonexistent URI")
	}
}

func TestWorkspace_GetDocument_Found(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	expectedDoc := &document{
		URI:     uri,
		Content: []byte("<template>test</template>"),
	}
	ws.documents[uri] = expectedDoc

	document, exists := ws.GetDocument(uri)

	if !exists {
		t.Error("expected exists=true for existing document")
	}
	if document != expectedDoc {
		t.Error("expected to get the same document pointer")
	}
}

func TestWorkspace_UpdateDocument_New(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	content := []byte("<template>new</template>")

	ws.UpdateDocument(uri, content)

	document, exists := ws.documents[uri]
	if !exists {
		t.Fatal("expected document to be created")
	}
	if !document.dirty {
		t.Error("expected new document to be marked dirty")
	}

	cachedContent, found := ws.docCache.Get(uri)
	if !found {
		t.Error("expected content in docCache")
	}
	if string(cachedContent) != string(content) {
		t.Errorf("expected cached content %q, got %q", content, cachedContent)
	}
}

func TestWorkspace_UpdateDocument_Existing(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	existingDoc := &document{
		URI:     uri,
		Content: []byte("<template>old</template>"),
		dirty:   false,
	}
	ws.documents[uri] = existingDoc

	newContent := []byte("<template>updated</template>")
	ws.UpdateDocument(uri, newContent)

	if !existingDoc.dirty {
		t.Error("expected existing document to be marked dirty after update")
	}

	cachedContent, _ := ws.docCache.Get(uri)
	if string(cachedContent) != string(newContent) {
		t.Errorf("expected cached content %q, got %q", newContent, cachedContent)
	}
}

func TestWorkspace_ConcurrentUpdateDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				content := fmt.Appendf(nil, "<template>content-%d-%d</template>", id, j)
				ws.UpdateDocument(uri, content)
			}
		}(i)
	}

	wg.Wait()

	document, exists := ws.GetDocument(uri)
	if !exists {
		t.Fatal("expected document to exist after concurrent updates")
	}
	if !document.dirty {
		t.Error("expected document to be dirty after updates")
	}
}

func TestWorkspace_ConcurrentGetDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ws.documents[uri] = &document{
		URI:     uri,
		Content: []byte("<template>test</template>"),
	}

	const goroutines = 100
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				_, _ = ws.GetDocument(uri)
			}
		}()
	}

	wg.Wait()
}

func TestWorkspace_ConcurrentUpdateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				content := fmt.Appendf(nil, "<template>content-%d-%d</template>", id, j)
				ws.UpdateDocument(uri, content)
			}
		}(i)
	}

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				_, _ = ws.GetDocument(uri)
			}
		}()
	}

	wg.Wait()
}

func TestWorkspace_SetupAnalysisContext_CancelsPrevious(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ctx := context.Background()

	_, doneChan1 := ws.setupAnalysisContext(ctx, uri)

	ws.mu.RLock()
	_, hasCancel := ws.cancelFuncs[uri]
	_, hasDone := ws.analysisDone[uri]
	ws.mu.RUnlock()

	if !hasCancel {
		t.Error("expected cancel function to be registered")
	}
	if !hasDone {
		t.Error("expected done channel to be registered")
	}

	_, doneChan2 := ws.setupAnalysisContext(ctx, uri)

	select {
	case <-doneChan1:

	default:
		t.Error("expected first done channel to be closed when setting up new analysis")
	}

	select {
	case <-doneChan2:
		t.Error("expected second done channel to still be open")
	default:

	}
}

func TestWorkspace_CleanupAnalysisContext_SignalsCompletion(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ctx := context.Background()

	_, doneChan := ws.setupAnalysisContext(ctx, uri)

	ws.cleanupAnalysisContext(ctx, uri, doneChan)

	select {
	case <-doneChan:

	default:
		t.Error("expected done channel to be closed after cleanup")
	}

	ws.mu.RLock()
	_, hasCancel := ws.cancelFuncs[uri]
	_, hasDone := ws.analysisDone[uri]
	ws.mu.RUnlock()

	if hasCancel {
		t.Error("expected cancel function to be removed after cleanup")
	}
	if hasDone {
		t.Error("expected done channel to be removed after cleanup")
	}
}

func TestWorkspace_ConcurrentSetupCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	ws := createTestWorkspace()
	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			uri := protocol.DocumentURI(fmt.Sprintf("file:///test%d.pk", id))
			ctx := context.Background()

			for range iterations {
				_, doneChan := ws.setupAnalysisContext(ctx, uri)

				ws.cleanupAnalysisContext(ctx, uri, doneChan)
			}
		}(i)
	}

	wg.Wait()
}

func TestWorkspace_ConcurrentSetupCleanup_SameURI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			ctx := context.Background()

			for range iterations {
				_, doneChan := ws.setupAnalysisContext(ctx, uri)
				ws.cleanupAnalysisContext(ctx, uri, doneChan)
			}
		}()
	}

	wg.Wait()
}

func TestWorkspace_GetCachedCleanDocument_Dirty(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ws.documents[uri] = &document{
		URI:   uri,
		dirty: true,
	}

	document := ws.getCachedCleanDocument(context.Background(), uri)

	if document != nil {
		t.Error("expected nil for dirty document")
	}
}

func TestWorkspace_GetCachedCleanDocument_Clean(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	expectedDoc := &document{
		URI:   uri,
		dirty: false,
	}
	ws.documents[uri] = expectedDoc

	document := ws.getCachedCleanDocument(context.Background(), uri)

	if document != expectedDoc {
		t.Error("expected to get the cached clean document")
	}
}

func TestWorkspace_GetCachedCleanDocument_NotFound(t *testing.T) {
	ws := createTestWorkspace()

	document := ws.getCachedCleanDocument(context.Background(), "file:///nonexistent.pk")

	if document != nil {
		t.Error("expected nil for nonexistent document")
	}
}

func TestNewWorkspace_InitialisesMaps(t *testing.T) {
	deps := workspaceDeps{
		Client:   &mockClient{},
		DocCache: NewDocumentCache(),
	}
	rootURI := protocol.DocumentURI("file:///project")

	ws := newWorkspace(deps, rootURI)

	if ws == nil {
		t.Fatal("expected non-nil workspace")
	}
	if ws.documents == nil {
		t.Error("expected documents map to be initialised")
	}
	if ws.actionProviders == nil {
		t.Error("expected actionProviders map to be initialised")
	}
	if ws.cancelFuncs == nil {
		t.Error("expected cancelFuncs map to be initialised")
	}
	if ws.analysisDone == nil {
		t.Error("expected analysisDone map to be initialised")
	}
	if ws.rootURI != rootURI {
		t.Errorf("rootURI = %q, want %q", ws.rootURI, rootURI)
	}
	if ws.client != deps.Client {
		t.Error("expected client to be set from deps")
	}
	if ws.docCache != deps.DocCache {
		t.Error("expected docCache to be set from deps")
	}
}

func TestSetConn_UpdatesConnection(t *testing.T) {
	ws := createTestWorkspace()

	if ws.conn != nil {
		t.Error("expected initial conn to be nil")
	}

	ws.setConn(nil)

	conn := ws.getConn()
	if conn != nil {
		t.Error("expected conn to remain nil after setting nil")
	}
}

func TestRemoveDocument_RemovesFromAllMaps(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{URI: testURI}
	ws.docCache.Set(testURI, []byte("content"))

	ws.RemoveDocument(context.Background(), testURI)

	if _, exists := ws.documents[testURI]; exists {
		t.Error("expected document to be removed from documents map")
	}
	if _, found := ws.docCache.Get(testURI); found {
		t.Error("expected document to be removed from docCache")
	}
}

func TestRemoveDocument_CancelsInFlightAnalysis(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{URI: testURI}

	cancelled := false
	ws.cancelFuncs[testURI] = func(error) { cancelled = true }
	doneChan := make(chan struct{})
	ws.analysisDone[testURI] = doneChan

	ws.RemoveDocument(context.Background(), testURI)

	if !cancelled {
		t.Error("expected cancel function to be called")
	}
	if _, exists := ws.cancelFuncs[testURI]; exists {
		t.Error("expected cancelFunc to be removed")
	}
	if _, exists := ws.analysisDone[testURI]; exists {
		t.Error("expected analysisDone to be removed")
	}
}

func TestRemoveDocument_PublishesClearDiagnostics(t *testing.T) {
	published := make(chan *protocol.PublishDiagnosticsParams, 1)
	client := &mockClient{
		PublishDiagnosticsFunc: func(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
			published <- params
			return nil
		},
	}

	ws := createTestWorkspace()
	ws.client = client
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{URI: testURI}
	ws.docCache.Set(testURI, []byte("content"))

	ws.RemoveDocument(context.Background(), testURI)

	select {
	case params := <-published:
		if params.URI != testURI {
			t.Errorf("published URI = %q, want %q", params.URI, testURI)
		}
		if len(params.Diagnostics) != 0 {
			t.Errorf("expected empty diagnostics, got %d", len(params.Diagnostics))
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for diagnostics to be published")
	}
}

func TestRemoveDocument_NilClient_DoesNotPanic(t *testing.T) {
	ws := createTestWorkspace()
	ws.client = nil
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{URI: testURI}

	ws.RemoveDocument(context.Background(), testURI)
}

func TestSearchAllDocuments_FindsReferencesAcrossDocuments(t *testing.T) {
	ws := createTestWorkspace()

	ws.documents["file:///a.pk"] = &document{URI: "file:///a.pk"}
	ws.documents["file:///b.pk"] = &document{URI: "file:///b.pk"}

	target := &symbolTarget{
		sourcePath:  "/project/main.go",
		name:        "Foo",
		defLocation: ast_domain.Location{Line: 10, Column: 5},
	}

	locations := ws.searchAllDocuments(target)

	if len(locations) != 0 {
		t.Errorf("expected 0 locations, got %d", len(locations))
	}
}

func TestCopyActionProviders_ReturnsShallowCopy(t *testing.T) {
	ws := createTestWorkspace()
	ws.actionProviders = make(map[string]annotator_domain.ActionInfoProvider)
	ws.actionProviders["test"] = nil

	copied := ws.copyActionProviders()

	if len(copied) != 1 {
		t.Errorf("expected 1 provider, got %d", len(copied))
	}

	copied["new"] = nil
	if len(ws.actionProviders) != 1 {
		t.Error("modifying copy should not affect original")
	}
}

func TestCopyActionProviders_EmptyMap(t *testing.T) {
	ws := createTestWorkspace()
	ws.actionProviders = make(map[string]annotator_domain.ActionInfoProvider)

	copied := ws.copyActionProviders()

	if copied == nil {
		t.Fatal("expected non-nil map")
	}
	if len(copied) != 0 {
		t.Errorf("expected empty map, got %d entries", len(copied))
	}
}

func TestHandleNilProjectResult_WithNonCancelError_ReturnsErrorDoc(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.docCache.Set(testURI, []byte("<template>content</template>"))

	moduleCtx := &ModuleContext{
		Resolver: &resolver_domain.MockResolver{},
	}

	document, err := ws.handleNilProjectResult(context.Background(), testURI, moduleCtx, errors.New("analysis failed"))

	if err != nil {
		t.Errorf("expected nil error from handleNilProjectResult, got %v", err)
	}
	if document == nil {
		t.Fatal("expected non-nil document")
	}
	if document.URI != testURI {
		t.Errorf("document.URI = %q, want %q", document.URI, testURI)
	}
	if string(document.Content) != "<template>content</template>" {
		t.Errorf("document.Content = %q, want cached content", document.Content)
	}
	if document.dirty {
		t.Error("expected document to be clean")
	}
}

func TestHandleNilProjectResult_WithCancelError_ReturnsErrorDoc(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.docCache.Set(testURI, []byte("content"))

	moduleCtx := &ModuleContext{
		Resolver: &resolver_domain.MockResolver{},
	}

	document, err := ws.handleNilProjectResult(context.Background(), testURI, moduleCtx, context.Canceled)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if document == nil {
		t.Fatal("expected non-nil document")
	}
}

func TestHandleNilProjectResult_NilError_ReturnsErrorDoc(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	moduleCtx := &ModuleContext{
		Resolver: &resolver_domain.MockResolver{},
	}

	document, err := ws.handleNilProjectResult(context.Background(), testURI, moduleCtx, nil)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if document == nil {
		t.Fatal("expected non-nil document")
	}
}

func TestHandleNilProjectResult_StoresDocumentInWorkspace(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	moduleCtx := &ModuleContext{
		Resolver: &resolver_domain.MockResolver{},
	}

	document, _ := ws.handleNilProjectResult(context.Background(), testURI, moduleCtx, errors.New("fail"))

	storedDoc, exists := ws.GetDocument(testURI)
	if !exists {
		t.Fatal("expected document to be stored in workspace")
	}
	if storedDoc != document {
		t.Error("expected stored document to be the same as returned")
	}
}

func TestLogAnnotationResultStatus_NilAnnotation_DoesNotPanic(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	projectResult := &annotator_dto.ProjectAnnotationResult{}

	ws.logAnnotationResultStatus(context.Background(), testURI, "/test/file.pk", projectResult, nil)
}

func TestLogAnnotationResultStatus_WithAnnotation_DoesNotPanic(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	projectResult := &annotator_dto.ProjectAnnotationResult{}
	annotationResult := &annotator_dto.AnnotationResult{
		AnnotatedAST: newTestAnnotatedAST(),
	}

	ws.logAnnotationResultStatus(context.Background(), testURI, "/test/file.pk", projectResult, annotationResult)
}

func TestLogAnnotationResultStatus_NilAnnotation_WithVirtualModule_DoesNotPanic(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	projectResult := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{
					"/test/file.pk": "file_abc123",
				},
			},
		},
	}

	ws.logAnnotationResultStatus(context.Background(), testURI, "/test/file.pk", projectResult, nil)
}

func TestExtractTypedAnalysisMap(t *testing.T) {
	testCases := []struct {
		result    *annotator_dto.AnnotationResult
		name      string
		uri       protocol.DocumentURI
		wantCount int
		wantNil   bool
	}{
		{
			name:    "nil annotation result returns nil",
			uri:     "file:///test.pk",
			result:  nil,
			wantNil: true,
		},
		{
			name: "nil analysis map returns nil",
			uri:  "file:///test.pk",
			result: &annotator_dto.AnnotationResult{
				AnalysisMap: nil,
			},
			wantNil: true,
		},
		{
			name: "wrong type assertion returns nil",
			uri:  "file:///test.pk",
			result: &annotator_dto.AnnotationResult{
				AnalysisMap: "not a map",
			},
			wantNil: true,
		},
		{
			name: "correct type returns map",
			uri:  "file:///test.pk",
			result: &annotator_dto.AnnotationResult{
				AnalysisMap: map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
					newTestNode("div", 1, 1): {},
				},
			},
			wantNil:   false,
			wantCount: 1,
		},
		{
			name: "empty map of correct type returns empty map",
			uri:  "file:///test.pk",
			result: &annotator_dto.AnnotationResult{
				AnalysisMap: map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{},
			},
			wantNil:   false,
			wantCount: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ws := createTestWorkspace()
			result := ws.extractTypedAnalysisMap(context.Background(), tc.uri, tc.result)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got map with %d entries", len(result))
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil map")
			}
			if len(result) != tc.wantCount {
				t.Errorf("map has %d entries, want %d", len(result), tc.wantCount)
			}
		})
	}
}

func TestGetTypeQuerier_NilManager_ReturnsNil(t *testing.T) {
	ws := createTestWorkspace()
	ws.typeInspectorManager = nil

	result := ws.getTypeQuerier(context.Background())
	if result != nil {
		t.Error("expected nil when typeInspectorManager is nil")
	}
}

func TestExtractAnnotationResultForURI(t *testing.T) {
	testCases := []struct {
		name    string
		project *annotator_dto.ProjectAnnotationResult
		absPath string
		wantNil bool
	}{
		{
			name: "nil VirtualModule returns nil",
			project: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: nil,
			},
			absPath: "/test.pk",
			wantNil: true,
		},
		{
			name: "nil Graph returns nil",
			project: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: nil,
				},
			},
			absPath: "/test.pk",
			wantNil: true,
		},
		{
			name: "path not in PathToHashedName returns nil",
			project: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							"/other.pk": "other_abc",
						},
					},
				},
			},
			absPath: "/test.pk",
			wantNil: true,
		},
		{
			name: "hashed name not in ComponentResults returns nil",
			project: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							"/test.pk": "test_abc",
						},
					},
				},
				ComponentResults: map[string]*annotator_dto.AnnotationResult{},
			},
			absPath: "/test.pk",
			wantNil: true,
		},
		{
			name: "valid path and result returns annotation result",
			project: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							"/test.pk": "test_abc",
						},
					},
				},
				ComponentResults: map[string]*annotator_dto.AnnotationResult{
					"test_abc": {
						AnnotatedAST: newTestAnnotatedAST(),
					},
				},
			},
			absPath: "/test.pk",
			wantNil: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ws := createTestWorkspace()
			result := ws.extractAnnotationResultForURI(tc.project, tc.absPath)

			if tc.wantNil {
				if result != nil {
					t.Error("expected nil result")
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestPublishDiagnostics_NilClient_DoesNotPanic(t *testing.T) {
	ws := createTestWorkspace()
	ws.client = nil

	document := &document{URI: "file:///test.pk"}

	ws.publishDiagnostics(context.Background(), "file:///test.pk", document)
}

func TestPublishDiagnostics_NilProjectResult_ClearsDiagnostics(t *testing.T) {
	published := make(chan *protocol.PublishDiagnosticsParams, 1)
	client := &mockClient{
		PublishDiagnosticsFunc: func(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
			published <- params
			return nil
		},
	}

	ws := createTestWorkspace()
	ws.client = client

	testURI := protocol.DocumentURI("file:///test.pk")
	document := &document{URI: testURI, ProjectResult: nil}

	ws.publishDiagnostics(context.Background(), testURI, document)

	select {
	case params := <-published:
		if params.URI != testURI {
			t.Errorf("URI = %q, want %q", params.URI, testURI)
		}
		if len(params.Diagnostics) != 0 {
			t.Errorf("expected empty diagnostics, got %d", len(params.Diagnostics))
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for diagnostics")
	}
}

func TestPublishDiagnostics_WithProjectResult_PublishesDiagnostics(t *testing.T) {
	published := make(chan *protocol.PublishDiagnosticsParams, 1)
	client := &mockClient{
		PublishDiagnosticsFunc: func(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
			published <- params
			return nil
		},
	}

	ws := createTestWorkspace()
	ws.client = client

	testURI := protocol.DocumentURI("file:///test.pk")
	document := &document{
		URI: testURI,
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			AllDiagnostics: []*ast_domain.Diagnostic{},
		},
	}

	ws.publishDiagnostics(context.Background(), testURI, document)

	select {
	case params := <-published:
		if params.URI != testURI {
			t.Errorf("URI = %q, want %q", params.URI, testURI)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for diagnostics")
	}
}

type testDirEntry struct {
	name  string
	isDir bool
}

func (e *testDirEntry) Name() string               { return e.name }
func (e *testDirEntry) IsDir() bool                { return e.isDir }
func (e *testDirEntry) Type() fs.FileMode          { return 0 }
func (e *testDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestIsValidEntryPointFile(t *testing.T) {
	testCases := []struct {
		entry    fs.DirEntry
		name     string
		expected bool
	}{
		{
			name:     "valid .pk file",
			entry:    &testDirEntry{name: "index.pk", isDir: false},
			expected: true,
		},
		{
			name:     "directory is rejected",
			entry:    &testDirEntry{name: "components", isDir: true},
			expected: false,
		},
		{
			name:     "non-.pk file is rejected",
			entry:    &testDirEntry{name: "main.go", isDir: false},
			expected: false,
		},
		{
			name:     "private file (underscore prefix) is rejected",
			entry:    &testDirEntry{name: "_partial.pk", isDir: false},
			expected: false,
		},
		{
			name:     ".pk directory is rejected",
			entry:    &testDirEntry{name: "test.pk", isDir: true},
			expected: false,
		},
		{
			name:     "file with .pk in middle but different extension is rejected",
			entry:    &testDirEntry{name: "test.pk.bak", isDir: false},
			expected: false,
		},
		{
			name:     "deeply nested valid .pk file",
			entry:    &testDirEntry{name: "about.pk", isDir: false},
			expected: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidEntryPointFile(tc.entry)
			if result != tc.expected {
				t.Errorf("isValidEntryPointFile(%q) = %v, want %v", tc.entry.Name(), result, tc.expected)
			}
		})
	}
}

func TestCreateDocumentFromResult(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.docCache.Set(testURI, []byte("<template>test</template>"))

	moduleCtx := &ModuleContext{
		Resolver: &resolver_domain.MockResolver{},
	}

	projectResult := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{},
			},
		},
		ComponentResults: map[string]*annotator_dto.AnnotationResult{},
	}

	document := ws.createDocumentFromResult(context.Background(), testURI, projectResult, moduleCtx)

	if document == nil {
		t.Fatal("expected non-nil document")
	}
	if document.URI != testURI {
		t.Errorf("URI = %q, want %q", document.URI, testURI)
	}
	if string(document.Content) != "<template>test</template>" {
		t.Errorf("Content = %q, want cached content", document.Content)
	}
	if document.dirty {
		t.Error("expected document to be clean")
	}
	if document.ProjectResult != projectResult {
		t.Error("expected ProjectResult to be set")
	}
	if document.Resolver != moduleCtx.Resolver {
		t.Error("expected Resolver to be set from moduleCtx")
	}
}

func TestWorkspace_ConcurrentSetConnGetConn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	ws := createTestWorkspace()
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				ws.setConn(nil)
			}
		}()
	}

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				_ = ws.getConn()
			}
		}()
	}

	wg.Wait()
}

func TestPublishErrorDiagnostic_NilClient(t *testing.T) {

	ws := &workspace{
		documents:    make(map[protocol.DocumentURI]*document),
		docCache:     NewDocumentCache(),
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
	}

	ws.publishErrorDiagnostic(context.Background(), "file:///test.pk", "some error")
}

func TestPublishErrorDiagnostic_WithClient(t *testing.T) {
	ws := createTestWorkspace()

	var receivedParams *protocol.PublishDiagnosticsParams
	client := &mockClient{
		PublishDiagnosticsFunc: func(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
			receivedParams = params
			return nil
		},
	}
	ws.setClient(client)

	ws.publishErrorDiagnostic(context.Background(), "file:///test.pk", "test error message")

	if receivedParams == nil {
		t.Fatal("expected PublishDiagnostics to be called")
	}
	if receivedParams.URI != "file:///test.pk" {
		t.Errorf("expected URI file:///test.pk, got %s", receivedParams.URI)
	}
	if len(receivedParams.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(receivedParams.Diagnostics))
	}
	if receivedParams.Diagnostics[0].Message != "test error message" {
		t.Errorf("expected message 'test error message', got %q", receivedParams.Diagnostics[0].Message)
	}
	if receivedParams.Diagnostics[0].Severity != protocol.DiagnosticSeverityError {
		t.Errorf("expected error severity, got %v", receivedParams.Diagnostics[0].Severity)
	}
}

func TestPublishErrorDiagnostic_ClientReturnsError(t *testing.T) {
	ws := createTestWorkspace()

	client := &mockClient{
		PublishDiagnosticsFunc: func(_ context.Context, _ *protocol.PublishDiagnosticsParams) error {
			return errors.New("publish failed")
		},
	}
	ws.setClient(client)

	ws.publishErrorDiagnostic(context.Background(), "file:///test.pk", "error message")
}

func TestWorkspace_Close_DrainsAllGoroutines(t *testing.T) {
	t.Parallel()

	ws := createTestWorkspace()

	const goroutines = 10
	finished := make(chan struct{}, goroutines)
	for range goroutines {
		ws.spawnTracked(t.Context(), "lsp.workspace.test", func() {
			time.Sleep(20 * time.Millisecond)
			finished <- struct{}{}
		})
	}

	if err := ws.Close(t.Context()); err != nil {
		t.Fatalf("Close returned an error: %v", err)
	}

	if got := len(finished); got != goroutines {
		t.Fatalf("expected %d goroutines to finish before Close returned, got %d", goroutines, got)
	}
}

func TestWorkspace_Close_RecoversFromPanic(t *testing.T) {
	t.Parallel()

	ws := createTestWorkspace()

	ws.spawnTracked(t.Context(), "lsp.workspace.test.panic", func() {
		panic("simulated workspace panic")
	})

	if err := ws.Close(t.Context()); err != nil {
		t.Fatalf("Close returned an error: %v", err)
	}
}

func TestWorkspace_Close_NilSafe(t *testing.T) {
	t.Parallel()

	var ws *workspace
	if err := ws.Close(t.Context()); err != nil {
		t.Fatalf("Close on nil workspace returned an error: %v", err)
	}
}

func TestWorkspace_PublishErrorDiagnosticAsync_Tracked(t *testing.T) {
	t.Parallel()

	receivedURIs := make(chan protocol.DocumentURI, 1)
	client := &mockClient{
		PublishDiagnosticsFunc: func(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
			receivedURIs <- params.URI
			return nil
		},
	}
	ws := createTestWorkspace()
	ws.setClient(client)

	uri := protocol.DocumentURI("file:///drain.pk")
	ws.publishErrorDiagnosticAsync(t.Context(), uri, "boom", "lsp.workspace.test")

	if err := ws.Close(t.Context()); err != nil {
		t.Fatalf("Close returned an error: %v", err)
	}

	select {
	case got := <-receivedURIs:
		if got != uri {
			t.Fatalf("expected %s, got %s", uri, got)
		}
	default:
		t.Fatal("publishErrorDiagnosticAsync did not run before Close drained")
	}
}
