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

package storage_domain_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

type mockTransformer struct {
	transformFunction func(ctx context.Context, input io.Reader, options any) (io.Reader, error)
	reverseFunction   func(ctx context.Context, input io.Reader, options any) (io.Reader, error)
	name              string
	ttype             storage_dto.TransformerType
	priority          int
}

var _ storage_domain.StreamTransformerPort = (*mockTransformer)(nil)

func (m *mockTransformer) Name() string                      { return m.name }
func (m *mockTransformer) Type() storage_dto.TransformerType { return m.ttype }
func (m *mockTransformer) Priority() int                     { return m.priority }

func (m *mockTransformer) Transform(ctx context.Context, input io.Reader, options any) (io.Reader, error) {
	if m.transformFunction != nil {
		return m.transformFunction(ctx, input, options)
	}
	return input, nil
}

func (m *mockTransformer) Reverse(ctx context.Context, input io.Reader, options any) (io.Reader, error) {
	if m.reverseFunction != nil {
		return m.reverseFunction(ctx, input, options)
	}
	return input, nil
}

func uppercaseTransformer(name string, priority int) *mockTransformer {
	return &mockTransformer{
		name:     name,
		ttype:    storage_dto.TransformerCompression,
		priority: priority,
		transformFunction: func(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
			data, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(bytes.ToUpper(data)), nil
		},
		reverseFunction: func(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
			data, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(bytes.ToLower(data)), nil
		},
	}
}

func prefixTransformer(name string, priority int, prefix string) *mockTransformer {
	return &mockTransformer{
		name:     name,
		ttype:    storage_dto.TransformerCustom,
		priority: priority,
		transformFunction: func(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
			data, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(append([]byte(prefix), data...)), nil
		},
		reverseFunction: func(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
			data, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			trimmed := strings.TrimPrefix(string(data), prefix)
			return bytes.NewReader([]byte(trimmed)), nil
		},
	}
}

func TestTransformerRegistry_Register(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	transformer := &mockTransformer{
		name:     "zstd",
		ttype:    storage_dto.TransformerCompression,
		priority: 100,
	}

	err := registry.Register(transformer)
	require.NoError(t, err)

	got, err := registry.Get("zstd")
	require.NoError(t, err)
	assert.Equal(t, "zstd", got.Name())
	assert.Equal(t, storage_dto.TransformerCompression, got.Type())
	assert.Equal(t, 100, got.Priority())
}

func TestTransformerRegistry_Has(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	transformer := &mockTransformer{
		name:     "aes",
		ttype:    storage_dto.TransformerEncryption,
		priority: 200,
	}

	require.NoError(t, registry.Register(transformer))

	assert.True(t, registry.Has("aes"), "registered transformer should be found")
	assert.False(t, registry.Has("nonexistent"), "unregistered name should not be found")
}

func TestTransformerRegistry_GetNames(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	require.NoError(t, registry.Register(&mockTransformer{name: "zstd", ttype: storage_dto.TransformerCompression, priority: 100}))
	require.NoError(t, registry.Register(&mockTransformer{name: "aes", ttype: storage_dto.TransformerEncryption, priority: 200}))
	require.NoError(t, registry.Register(&mockTransformer{name: "gzip", ttype: storage_dto.TransformerCompression, priority: 150}))

	names := registry.GetNames()
	assert.Equal(t, []string{"aes", "gzip", "zstd"}, names, "names should be sorted alphabetically")
}

func TestTransformerRegistry_Get_NotFound(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	_, err := registry.Get("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTransformerRegistry_Register_Duplicate(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	transformer := &mockTransformer{
		name:     "zstd",
		ttype:    storage_dto.TransformerCompression,
		priority: 100,
	}

	require.NoError(t, registry.Register(transformer))

	duplicate := &mockTransformer{
		name:     "zstd",
		ttype:    storage_dto.TransformerCompression,
		priority: 110,
	}

	err := registry.Register(duplicate)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestTransformerRegistry_Register_NilTransformer(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	err := registry.Register(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestTransformerRegistry_Register_EmptyName(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	transformer := &mockTransformer{
		name:     "",
		ttype:    storage_dto.TransformerCustom,
		priority: 300,
	}

	err := registry.Register(transformer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestTransformerChain_NewWithNilConfig(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	chain, err := storage_domain.NewTransformerChain(registry, nil)
	require.NoError(t, err)
	assert.True(t, chain.IsEmpty(), "chain created with nil config should be empty")
}

func TestTransformerChain_NewWithNilRegistry(t *testing.T) {
	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"zstd"},
		TransformerOptions:  make(map[string]any),
	}

	_, err := storage_domain.NewTransformerChain(nil, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestTransformerChain_IsEmpty(t *testing.T) {
	testCases := []struct {
		config    *storage_dto.TransformConfig
		name      string
		wantEmpty bool
	}{
		{
			name:      "nil config yields empty chain",
			config:    nil,
			wantEmpty: true,
		},
		{
			name: "empty enabled list yields empty chain",
			config: &storage_dto.TransformConfig{
				EnabledTransformers: []string{},
				TransformerOptions:  make(map[string]any),
			},
			wantEmpty: true,
		},
		{
			name: "non-empty enabled list yields non-empty chain",
			config: &storage_dto.TransformConfig{
				EnabledTransformers: []string{"zstd"},
				TransformerOptions:  make(map[string]any),
			},
			wantEmpty: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := storage_domain.NewTransformerRegistry()
			require.NoError(t, registry.Register(&mockTransformer{
				name:     "zstd",
				ttype:    storage_dto.TransformerCompression,
				priority: 100,
			}))

			chain, err := storage_domain.NewTransformerChain(registry, tc.config)
			require.NoError(t, err)
			assert.Equal(t, tc.wantEmpty, chain.IsEmpty())
		})
	}
}

func TestTransformerChain_Transform(t *testing.T) {

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(prefixTransformer("prefixB", 200, "B-")))
	require.NoError(t, registry.Register(prefixTransformer("prefixA", 100, "A-")))

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"prefixA", "prefixB"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := storage_domain.NewTransformerChain(registry, config)
	require.NoError(t, err)

	ctx := context.Background()

	result, err := chain.Transform(ctx, bytes.NewReader([]byte("hello")))
	require.NoError(t, err)

	data, err := io.ReadAll(result)
	require.NoError(t, err)
	assert.Equal(t, "B-A-hello", string(data), "transformers should be applied in ascending priority order")
}

func TestTransformerChain_Reverse(t *testing.T) {

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(prefixTransformer("prefixA", 100, "A-")))
	require.NoError(t, registry.Register(prefixTransformer("prefixB", 200, "B-")))

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"prefixA", "prefixB"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := storage_domain.NewTransformerChain(registry, config)
	require.NoError(t, err)

	ctx := context.Background()

	result, err := chain.Reverse(ctx, bytes.NewReader([]byte("B-A-hello")))
	require.NoError(t, err)

	data, err := io.ReadAll(result)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data), "reverse should undo transformations in descending priority order")
}

func TestTransformerChain_TransformError(t *testing.T) {
	errBoom := errors.New("transformer exploded")

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(&mockTransformer{
		name:     "failing",
		ttype:    storage_dto.TransformerCustom,
		priority: 100,
		transformFunction: func(_ context.Context, _ io.Reader, _ any) (io.Reader, error) {
			return nil, errBoom
		},
	}))

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"failing"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := storage_domain.NewTransformerChain(registry, config)
	require.NoError(t, err)

	ctx := context.Background()

	_, err = chain.Transform(ctx, bytes.NewReader([]byte("data")))
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom, "underlying transformer error should be wrapped and propagated")
}

func TestTransformerChain_ReverseError(t *testing.T) {
	errBoom := errors.New("reverse exploded")

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(&mockTransformer{
		name:     "failing-reverse",
		ttype:    storage_dto.TransformerCustom,
		priority: 100,
		reverseFunction: func(_ context.Context, _ io.Reader, _ any) (io.Reader, error) {
			return nil, errBoom
		},
	}))

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"failing-reverse"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := storage_domain.NewTransformerChain(registry, config)
	require.NoError(t, err)

	ctx := context.Background()

	_, err = chain.Reverse(ctx, bytes.NewReader([]byte("data")))
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom, "underlying reverse error should be wrapped and propagated")
}

func TestTransformerChain_TransformEmptyChain(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	chain, err := storage_domain.NewTransformerChain(registry, nil)
	require.NoError(t, err)

	ctx := context.Background()
	input := bytes.NewReader([]byte("passthrough"))

	result, err := chain.Transform(ctx, input)
	require.NoError(t, err)

	data, err := io.ReadAll(result)
	require.NoError(t, err)
	assert.Equal(t, "passthrough", string(data), "empty chain should return input unchanged")
}

func TestTransformerChain_ReverseEmptyChain(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	chain, err := storage_domain.NewTransformerChain(registry, nil)
	require.NoError(t, err)

	ctx := context.Background()
	input := bytes.NewReader([]byte("passthrough"))

	result, err := chain.Reverse(ctx, input)
	require.NoError(t, err)

	data, err := io.ReadAll(result)
	require.NoError(t, err)
	assert.Equal(t, "passthrough", string(data), "empty chain should return input unchanged")
}

func TestTransformerChain_NewWithUnknownTransformer(t *testing.T) {
	registry := storage_domain.NewTransformerRegistry()

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"nonexistent"},
		TransformerOptions:  make(map[string]any),
	}

	_, err := storage_domain.NewTransformerChain(registry, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func newTestWrapperSetup(t *testing.T, transformers ...*mockTransformer) (*storage_domain.TransformerWrapper, *provider_mock.MockStorageProvider, *storage_domain.TransformerRegistry) {
	t.Helper()

	mockProvider := provider_mock.NewMockStorageProvider()
	registry := storage_domain.NewTransformerRegistry()

	for _, tr := range transformers {
		require.NoError(t, registry.Register(tr))
	}

	wrapper := storage_domain.NewTransformerWrapper(mockProvider, registry, nil, "test-provider")
	return wrapper, mockProvider, registry
}

func TestTransformerWrapper_Put_NoTransformers(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("hello world")

	params := &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "file.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}

	err := wrapper.Put(ctx, params)
	require.NoError(t, err)

	data, found := mockProvider.GetObjectData("repo", "file.txt")
	require.True(t, found, "object should exist in mock provider")
	assert.Equal(t, content, data, "data should be unchanged when no transformers are configured")

	calls := mockProvider.GetPutCalls()
	require.Len(t, calls, 1)
	assert.Nil(t, calls[0].Metadata, "metadata should be nil when no transformers are active")
}

func TestTransformerWrapper_Put_WithTransformers(t *testing.T) {
	upper := uppercaseTransformer("upper", 100)
	wrapper, mockProvider, _ := newTestWrapperSetup(t, upper)

	ctx := context.Background()
	content := []byte("hello world")

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"upper"},
		TransformerOptions:  make(map[string]any),
	}

	params := &storage_dto.PutParams{
		Repository:      "repo",
		Key:             "file.txt",
		Reader:          bytes.NewReader(content),
		Size:            int64(len(content)),
		ContentType:     "text/plain",
		TransformConfig: config,
	}

	err := wrapper.Put(ctx, params)
	require.NoError(t, err)

	data, found := mockProvider.GetObjectData("repo", "file.txt")
	require.True(t, found, "object should exist in mock provider")
	assert.Equal(t, "HELLO WORLD", string(data), "data should be uppercased by the transformer")

	calls := mockProvider.GetPutCalls()
	require.Len(t, calls, 1)
	assert.NotNil(t, calls[0].Metadata)
	metadataValue, ok := calls[0].Metadata["x-piko-transformers"]
	assert.True(t, ok, "transformer metadata key should be present")
	assert.Contains(t, metadataValue, "upper", "metadata should contain the transformer name")

	assert.Equal(t, int64(-1), calls[0].Size, "size should be -1 after transformation")
}

func TestTransformerWrapper_Get_NoTransformers(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("hello world")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "file.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	reader, err := wrapper.Get(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "file.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, data, "data should be returned unchanged when no transformers are present")
}

func TestTransformerWrapper_Get_WithReverseTransform(t *testing.T) {
	upper := uppercaseTransformer("upper", 100)
	wrapper, mockProvider, _ := newTestWrapperSetup(t, upper)

	ctx := context.Background()

	transformedContent := []byte("HELLO WORLD")
	metadata := map[string]string{
		"x-piko-transformers": `["upper"]`,
	}

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "file.txt",
		Reader:      bytes.NewReader(transformedContent),
		Size:        int64(len(transformedContent)),
		ContentType: "text/plain",
		Metadata:    metadata,
	})
	require.NoError(t, seedErr)

	reader, err := wrapper.Get(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "file.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data), "data should be reverse-transformed (lowercased) based on metadata")
}

func TestTransformerWrapper_Passthrough_Stat(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("stat me")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "stat.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	info, err := wrapper.Stat(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "stat.txt",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), info.Size)
	assert.Equal(t, "text/plain", info.ContentType)
}

func TestTransformerWrapper_Passthrough_Remove(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("remove me")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "remove.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	err := wrapper.Remove(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "remove.txt",
	})
	require.NoError(t, err)

	removeCalls := mockProvider.GetRemoveCalls()
	require.Len(t, removeCalls, 1)
	assert.Equal(t, "remove.txt", removeCalls[0].Key)
}

func TestTransformerWrapper_Passthrough_Rename(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("rename me")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "old-name.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	err := wrapper.Rename(ctx, "repo", "old-name.txt", "new-name.txt")
	require.NoError(t, err)

	data, found := mockProvider.GetObjectData("repo", "new-name.txt")
	require.True(t, found, "object should exist under the new key after rename")
	assert.Equal(t, content, data)
}

func TestTransformerWrapper_Passthrough_Exists(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("i exist")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "exists.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	exists, err := wrapper.Exists(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "exists.txt",
	})
	require.NoError(t, err)
	assert.True(t, exists, "object that was seeded should exist")

	notExists, err := wrapper.Exists(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "ghost.txt",
	})
	require.NoError(t, err)
	assert.False(t, notExists, "unseeded object should not exist")
}

func TestTransformerWrapper_Passthrough_PresignDownloadURL(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()

	expectedURL := "https://example.com/presigned-download"
	mockProvider.SetPresignedURL(expectedURL)

	url, err := wrapper.PresignDownloadURL(ctx, storage_dto.PresignDownloadParams{
		Repository: "repo",
		Key:        "download.txt",
	})
	require.NoError(t, err)
	assert.Equal(t, expectedURL, url, "presigned download URL should be delegated to the inner provider")
}

func TestTransformerWrapper_Passthrough_SupportsMultipart(t *testing.T) {
	wrapper, _, _ := newTestWrapperSetup(t)

	assert.True(t, wrapper.SupportsMultipart(), "should delegate to inner provider")
}

func TestTransformerWrapper_Passthrough_SupportsBatchOperations(t *testing.T) {
	wrapper, _, _ := newTestWrapperSetup(t)

	assert.True(t, wrapper.SupportsBatchOperations(), "should delegate to inner provider")
}

func TestTransformerWrapper_Passthrough_SupportsRetry(t *testing.T) {
	wrapper, _, _ := newTestWrapperSetup(t)

	assert.False(t, wrapper.SupportsRetry(), "should delegate to inner provider")
}

func TestTransformerWrapper_Passthrough_SupportsCircuitBreaking(t *testing.T) {
	wrapper, _, _ := newTestWrapperSetup(t)

	assert.False(t, wrapper.SupportsCircuitBreaking(), "should delegate to inner provider")
}

func TestTransformerWrapper_Passthrough_SupportsRateLimiting(t *testing.T) {
	wrapper, _, _ := newTestWrapperSetup(t)

	assert.False(t, wrapper.SupportsRateLimiting(), "should delegate to inner provider")
}

func TestTransformerWrapper_Passthrough_SupportsPresignedURLs(t *testing.T) {
	wrapper, _, _ := newTestWrapperSetup(t)

	assert.True(t, wrapper.SupportsPresignedURLs(), "should delegate to inner provider")
}

func TestTransformerWrapper_Put_WithDefaultConfig(t *testing.T) {
	upper := uppercaseTransformer("upper", 100)

	mockProvider := provider_mock.NewMockStorageProvider()
	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(upper))

	defaultConfig := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"upper"},
		TransformerOptions:  make(map[string]any),
	}

	wrapper := storage_domain.NewTransformerWrapper(mockProvider, registry, defaultConfig, "test-provider")

	ctx := context.Background()
	content := []byte("default config")

	params := &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "default.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}

	err := wrapper.Put(ctx, params)
	require.NoError(t, err)

	data, found := mockProvider.GetObjectData("repo", "default.txt")
	require.True(t, found)
	assert.Equal(t, "DEFAULT CONFIG", string(data), "default config should cause transformation when no per-operation config is set")
}

func TestTransformerWrapper_Put_OperationConfigOverridesDefault(t *testing.T) {
	upper := uppercaseTransformer("upper", 100)
	prefix := prefixTransformer("prefix", 200, "PREFIX-")

	mockProvider := provider_mock.NewMockStorageProvider()
	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(upper))
	require.NoError(t, registry.Register(prefix))

	defaultConfig := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"upper"},
		TransformerOptions:  make(map[string]any),
	}

	wrapper := storage_domain.NewTransformerWrapper(mockProvider, registry, defaultConfig, "test-provider")

	ctx := context.Background()
	content := []byte("override")

	opConfig := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"prefix"},
		TransformerOptions:  make(map[string]any),
	}

	params := &storage_dto.PutParams{
		Repository:      "repo",
		Key:             "override.txt",
		Reader:          bytes.NewReader(content),
		Size:            int64(len(content)),
		ContentType:     "text/plain",
		TransformConfig: opConfig,
	}

	err := wrapper.Put(ctx, params)
	require.NoError(t, err)

	data, found := mockProvider.GetObjectData("repo", "override.txt")
	require.True(t, found)

	assert.Equal(t, "PREFIX-override", string(data), "operation config should override default config")
}

func TestTransformerWrapper_Passthrough_Close(t *testing.T) {
	wrapper, _, _ := newTestWrapperSetup(t)

	ctx := context.Background()

	err := wrapper.Close(ctx)
	require.NoError(t, err, "close should delegate to the inner provider without error")
}

func TestTransformerWrapper_Passthrough_Copy(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("copy me")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "src.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	err := wrapper.Copy(ctx, "repo", "src.txt", "dst.txt")
	require.NoError(t, err)

	copyCalls := mockProvider.GetCopyCalls()
	require.Len(t, copyCalls, 1)
	assert.Equal(t, "src.txt", copyCalls[0].SourceKey)
	assert.Equal(t, "dst.txt", copyCalls[0].DestinationKey)
}

func TestTransformerWrapper_Passthrough_GetHash(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("hash me")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo",
		Key:         "hash.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	hash, err := wrapper.GetHash(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "hash.txt",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, hash, "hash should be returned from the inner provider")
}

func TestTransformerWrapper_Passthrough_PresignURL(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()

	expectedURL := "https://example.com/presigned-upload"
	mockProvider.SetPresignedURL(expectedURL)

	url, err := wrapper.PresignURL(ctx, storage_dto.PresignParams{
		Repository: "repo",
		Key:        "upload.txt",
	})
	require.NoError(t, err)
	assert.Equal(t, expectedURL, url, "presigned upload URL should be delegated to the inner provider")
}

func TestTransformerWrapper_RoundTrip(t *testing.T) {

	upper := uppercaseTransformer("upper", 100)
	wrapper, _, _ := newTestWrapperSetup(t, upper)

	ctx := context.Background()
	original := []byte("round trip data")

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"upper"},
		TransformerOptions:  make(map[string]any),
	}

	putErr := wrapper.Put(ctx, &storage_dto.PutParams{
		Repository:      "repo",
		Key:             "roundtrip.txt",
		Reader:          bytes.NewReader(original),
		Size:            int64(len(original)),
		ContentType:     "text/plain",
		TransformConfig: config,
	})
	require.NoError(t, putErr)

	reader, getErr := wrapper.Get(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "roundtrip.txt",
	})
	require.NoError(t, getErr)
	defer func() { _ = reader.Close() }()

	data, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	assert.Equal(t, string(original), string(data), "round-trip through Put and Get should return the original data")
}

func TestTransformerWrapper_RoundTrip_MultipleTransformers(t *testing.T) {

	prefixA := prefixTransformer("prefixA", 100, "A-")
	prefixB := prefixTransformer("prefixB", 200, "B-")
	wrapper, _, _ := newTestWrapperSetup(t, prefixA, prefixB)

	ctx := context.Background()
	original := []byte("multi")

	config := &storage_dto.TransformConfig{
		EnabledTransformers: []string{"prefixA", "prefixB"},
		TransformerOptions:  make(map[string]any),
	}

	putErr := wrapper.Put(ctx, &storage_dto.PutParams{
		Repository:      "repo",
		Key:             "multi.txt",
		Reader:          bytes.NewReader(original),
		Size:            int64(len(original)),
		ContentType:     "text/plain",
		TransformConfig: config,
	})
	require.NoError(t, putErr)

	reader, getErr := wrapper.Get(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "multi.txt",
	})
	require.NoError(t, getErr)
	defer func() { _ = reader.Close() }()

	data, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	assert.Equal(t, "multi", string(data), "round-trip with multiple transformers should restore original data")
}

func TestTransformerWrapper_Passthrough_RemoveMany(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()

	for _, key := range []string{"a.txt", "b.txt"} {
		seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
			Repository:  "repo",
			Key:         key,
			Reader:      bytes.NewReader([]byte("data")),
			Size:        4,
			ContentType: "text/plain",
		})
		require.NoError(t, seedErr)
	}

	result, err := wrapper.RemoveMany(ctx, storage_dto.RemoveManyParams{
		Repository: "repo",
		Keys:       []string{"a.txt", "b.txt"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalSuccessful)

	removeManyCalls := mockProvider.GetRemoveManyCalls()
	require.Len(t, removeManyCalls, 1)
	assert.Equal(t, []string{"a.txt", "b.txt"}, removeManyCalls[0].Keys)
}

func TestTransformerWrapper_Passthrough_CopyToAnotherRepository(t *testing.T) {
	wrapper, mockProvider, _ := newTestWrapperSetup(t)

	ctx := context.Background()
	content := []byte("cross-repo")

	seedErr := mockProvider.Put(ctx, &storage_dto.PutParams{
		Repository:  "src-repo",
		Key:         "src.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, seedErr)

	err := wrapper.CopyToAnotherRepository(ctx, "src-repo", "src.txt", "dst-repo", "dst.txt")
	require.NoError(t, err)

	copyCalls := mockProvider.GetCopyCalls()
	require.Len(t, copyCalls, 1)
	assert.Equal(t, "src-repo", copyCalls[0].SourceRepository)
	assert.Equal(t, "dst-repo", copyCalls[0].DestinationRepository)
}
