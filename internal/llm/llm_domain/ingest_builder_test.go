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

package llm_domain

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestNewIngestBuilder(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "test-namespace")

	require.NotNil(t, builder)
	assert.Equal(t, service, builder.service)
	assert.Equal(t, "test-namespace", builder.namespace)
	assert.Nil(t, builder.loader)
	assert.Nil(t, builder.splitter)
}

func TestIngestBuilder_FromFS(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	fsys := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("Hello World")},
	}

	result := builder.FromFS(fsys, "*.txt")

	assert.Equal(t, builder, result)
	assert.NotNil(t, builder.loader)
}

func TestIngestBuilder_FromDirectory(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	result := builder.FromDirectory("/tmp", "*.txt")

	assert.Equal(t, builder, result)
	assert.NotNil(t, builder.loader)
}

func TestIngestBuilder_Loader(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	mockLoader := &mockLoaderPort{
		docs: []Document{{ID: "doc1", Content: "content"}},
	}

	result := builder.Loader(mockLoader)

	assert.Equal(t, builder, result)
	assert.Equal(t, mockLoader, builder.loader)
}

func TestIngestBuilder_Splitter(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	splitter, err := NewRecursiveCharacterSplitter(100, 10)
	require.NoError(t, err)

	result := builder.Splitter(splitter)

	assert.Equal(t, builder, result)
	assert.Equal(t, splitter, builder.splitter)
}

func TestIngestBuilder_Do_NoLoader(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	err := builder.Do(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loader is required")
}

func TestIngestBuilder_Do_LoaderError(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	builder.Loader(&mockLoaderPort{
		err: errors.New("load failed"),
	})

	err := builder.Do(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading documents")
}

func TestIngestBuilder_Do_WithoutSplitter(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{
			{ID: "doc1", Content: "Hello World"},
			{ID: "doc2", Content: "Goodbye World"},
		},
	})

	err := builder.Do(context.Background())

	require.NoError(t, err)
	assert.Len(t, mockVS.bulkStoreCalls, 1)
	assert.Len(t, mockVS.bulkStoreCalls[0], 2)
}

func TestIngestBuilder_Do_WithSplitter(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	splitter, err := NewRecursiveCharacterSplitter(10, 0)
	require.NoError(t, err)

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{
			{ID: "doc1", Content: "This is a longer text that should be split into chunks"},
		},
	})
	builder.Splitter(splitter)

	err = builder.Do(context.Background())

	require.NoError(t, err)
	require.NotEmpty(t, mockVS.bulkStoreCalls)

	totalDocuments := 0
	for _, batch := range mockVS.bulkStoreCalls {
		totalDocuments += len(batch)
	}
	assert.Greater(t, totalDocuments, 1)
}

func TestIngestBuilder_Do_MethodChaining(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	fsys := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("content a")},
	}

	splitter, err := NewRecursiveCharacterSplitter(100, 0)
	require.NoError(t, err)

	err = NewIngestBuilder(service, "ns").
		FromFS(fsys, "*.txt").
		Splitter(splitter).
		Do(context.Background())

	require.NoError(t, err)
}

func TestIngestBuilder_Do_VectorStoreError(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{
		bulkStoreErr: errors.New("storage failed"),
	}
	service.SetVectorStore(mockVS)

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{{ID: "doc1", Content: "content"}},
	})

	err := builder.Do(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adding documents to vector store")
}

func TestHasRecursivePattern(t *testing.T) {
	assert.False(t, hasRecursivePattern(nil))
	assert.False(t, hasRecursivePattern([]string{"*.md"}))
	assert.False(t, hasRecursivePattern([]string{"*.md", "*.txt"}))
	assert.True(t, hasRecursivePattern([]string{"**/*.md"}))
	assert.True(t, hasRecursivePattern([]string{"*.txt", "**/*.md"}))
}

func TestStripRecursivePrefix(t *testing.T) {
	assert.Equal(t, []string{"*.md"}, stripRecursivePrefix([]string{"**/*.md"}))
	assert.Equal(t, []string{"*.txt", "*.md"}, stripRecursivePrefix([]string{"*.txt", "**/*.md"}))
	assert.Equal(t, []string{"*.md"}, stripRecursivePrefix([]string{"*.md"}))
}

func TestIngestBuilder_FromFS_RecursivePattern(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	fsys := fstest.MapFS{
		"top.md":        &fstest.MapFile{Data: []byte("top")},
		"sub/nested.md": &fstest.MapFile{Data: []byte("nested")},
	}

	builder.FromFS(fsys, "**/*.md")
	assert.NotNil(t, builder.loader)

	_, ok = builder.loader.(*RecursiveFSLoader)
	assert.True(t, ok, "expected RecursiveFSLoader for ** pattern")
}

func TestIngestBuilder_FromFS_FlatPattern(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")

	fsys := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("hello")},
	}

	builder.FromFS(fsys, "*.txt")
	assert.NotNil(t, builder.loader)

	_, ok = builder.loader.(*FSLoader)
	assert.True(t, ok, "expected FSLoader for flat pattern")
}

func TestIngestBuilder_Transform(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{
			{ID: "doc1", Content: "---\ntitle: Test\n---\nActual content"},
		},
	})
	builder.Transform(StripFrontmatter())

	err := builder.Do(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, mockVS.bulkStoreCalls)

	storedContent := mockVS.bulkStoreCalls[0][0].Content
	assert.Equal(t, "Actual content", storedContent)
}

func TestIngestBuilder_Transform_Chaining(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	addMeta := func(document Document) Document {
		if document.Metadata == nil {
			document.Metadata = make(map[string]any)
		}
		document.Metadata["custom"] = "value"
		return document
	}
	upperContent := func(document Document) Document {
		document.Content = "TRANSFORMED"
		return document
	}

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{
			{ID: "doc1", Content: "---\ntitle: Test\n---\nOriginal"},
		},
	})

	builder.Transform(StripFrontmatter()).
		Transform(addMeta).
		Transform(upperContent)

	err := builder.Do(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, mockVS.bulkStoreCalls)

	stored := mockVS.bulkStoreCalls[0][0]
	assert.Equal(t, "TRANSFORMED", stored.Content)
	assert.Equal(t, "value", stored.Metadata["custom"])
}

func TestIngestBuilder_Transform_OrderMatters(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	var order []string
	first := func(document Document) Document {
		order = append(order, "first")
		return document
	}
	second := func(document Document) Document {
		order = append(order, "second")
		return document
	}

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{{ID: "doc1", Content: "content"}},
	})
	builder.Transform(first).Transform(second)

	err := builder.Do(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, order)
}

func TestIngestBuilder_PostSplitTransform(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{
			{
				ID:       "doc1",
				Content:  "# Title\n\nSome content\n\n## Section\n\nMore content",
				Metadata: map[string]any{"doc_title": "My Doc"},
			},
		},
	})
	mdSplitter, err := NewMarkdownSplitter(5000, 0)
	require.NoError(t, err)
	builder.Splitter(mdSplitter)

	builder.PostSplitTransform(func(document Document) Document {
		if title, ok := document.Metadata["doc_title"].(string); ok {
			document.Content = title + ": " + document.Content
		}
		return document
	})

	err = builder.Do(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, mockVS.bulkStoreCalls)

	for _, batch := range mockVS.bulkStoreCalls {
		for _, d := range batch {
			assert.True(t, len(d.Content) > 0)
			assert.Contains(t, d.Content, "My Doc:")
		}
	}
}

func TestIngestBuilder_PostSplitTransform_FiltersEmptyContent(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{
			{ID: "doc1", Content: "keep this"},
			{ID: "doc2", Content: "drop this"},
			{ID: "doc3", Content: "keep this too"},
		},
	})

	builder.PostSplitTransform(func(document Document) Document {
		if document.ID == "doc2" {
			document.Content = ""
		}
		return document
	})

	err := builder.Do(context.Background())
	require.NoError(t, err)

	totalDocuments := 0
	for _, batch := range mockVS.bulkStoreCalls {
		totalDocuments += len(batch)
	}
	assert.Equal(t, 2, totalDocuments, "doc2 should have been filtered out")
}

func TestIngestBuilder_PostSplitTransform_Chaining(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	var order []string

	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{{ID: "doc1", Content: "content"}},
	})

	builder.
		PostSplitTransform(func(document Document) Document {
			order = append(order, "first")
			return document
		}).
		PostSplitTransform(func(document Document) Document {
			order = append(order, "second")
			return document
		})

	err := builder.Do(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, order)
}

type mockLoaderPort struct {
	err  error
	docs []Document
}

func (m *mockLoaderPort) Load(_ context.Context) ([]Document, error) {
	return m.docs, m.err
}

type mockVectorStore struct {
	bulkStoreErr   error
	closeErr       error
	bulkStoreCalls [][]*llm_dto.VectorDocument
	closeCalled    bool
}

func (m *mockVectorStore) Store(_ context.Context, _ string, _ *llm_dto.VectorDocument) error {
	return nil
}

func (m *mockVectorStore) BulkStore(_ context.Context, _ string, docs []*llm_dto.VectorDocument) error {
	m.bulkStoreCalls = append(m.bulkStoreCalls, docs)
	return m.bulkStoreErr
}

func (m *mockVectorStore) Search(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
	return &llm_dto.VectorSearchResponse{}, nil
}

func (m *mockVectorStore) Get(_ context.Context, _, _ string) (*llm_dto.VectorDocument, error) {
	return nil, nil
}

func (m *mockVectorStore) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockVectorStore) DeleteByFilter(_ context.Context, _ string, _ map[string]any) (int, error) {
	return 0, nil
}

func (m *mockVectorStore) CreateNamespace(_ context.Context, _ string, _ *VectorNamespaceConfig) error {
	return nil
}

func (m *mockVectorStore) DeleteNamespace(_ context.Context, _ string) error {
	return nil
}

func (m *mockVectorStore) Close(_ context.Context) error {
	m.closeCalled = true
	return m.closeErr
}
