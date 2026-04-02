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

package i18n_adapters

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewStoreFromTranslations_Empty(t *testing.T) {
	t.Parallel()

	translations := i18n_domain.Translations{}
	store := i18n_domain.NewStoreFromTranslations(translations, "en")
	assert.NotNil(t, store)
	assert.Empty(t, store.Locales())
}

func TestNewStoreFromTranslations_WithEntries(t *testing.T) {
	t.Parallel()

	translations := i18n_domain.Translations{
		"en": {"greeting": "Hello", "farewell": "Goodbye"},
		"fr": {"greeting": "Bonjour"},
	}

	store := i18n_domain.NewStoreFromTranslations(translations, "en")
	require.NotNil(t, store)

	locales := store.Locales()
	sort.Strings(locales)
	assert.Equal(t, []string{"en", "fr"}, locales)

	entry, found := store.Get("en", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello", entry.Template)

	entry, found = store.Get("fr", "greeting")
	require.True(t, found)
	assert.Equal(t, "Bonjour", entry.Template)
}

func TestNewEmptyService(t *testing.T) {
	t.Parallel()

	service := newEmptyService("en-GB")
	assert.NotNil(t, service)
	assert.Equal(t, "en-GB", service.DefaultLocale())
	assert.Nil(t, service.GetStore())
	assert.Nil(t, service.GetStrBufPool())
}

func TestNewEmptyService_CustomLocale(t *testing.T) {
	t.Parallel()

	service := newEmptyService("ja-JP")
	assert.Equal(t, "ja-JP", service.DefaultLocale())
}

func TestFsService_GetStore(t *testing.T) {
	t.Parallel()

	store := i18n_domain.NewStore("en")
	service := &fsService{
		store:         store,
		defaultLocale: "en",
	}

	assert.Equal(t, store, service.GetStore())
}

func TestFsService_GetStore_Nil(t *testing.T) {
	t.Parallel()

	service := &fsService{
		store:         nil,
		defaultLocale: "en",
	}

	assert.Nil(t, service.GetStore())
}

func TestFsService_GetStrBufPool(t *testing.T) {
	t.Parallel()

	pool := i18n_domain.NewStrBufPool(256)
	service := &fsService{
		strBufPool:    pool,
		defaultLocale: "en",
	}

	assert.Equal(t, pool, service.GetStrBufPool())
}

func TestFsService_GetStrBufPool_Nil(t *testing.T) {
	t.Parallel()

	service := &fsService{
		strBufPool:    nil,
		defaultLocale: "en",
	}

	assert.Nil(t, service.GetStrBufPool())
}

func TestFsService_DefaultLocale(t *testing.T) {
	t.Parallel()

	service := &fsService{
		defaultLocale: "fr-FR",
	}

	assert.Equal(t, "fr-FR", service.DefaultLocale())
}

func TestNewFSServiceFromDir_EmptyDirPath(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	service, err := newFSServiceFromDir(context.Background(), sandbox, "", "en")
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.Equal(t, "en", service.DefaultLocale())
}

func TestNewFSServiceFromDir_NonexistentDir(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	service, err := newFSServiceFromDir(context.Background(), sandbox, "nonexistent", "en")
	require.NoError(t, err)
	require.NotNil(t, service)
}

func TestNewFSServiceFromDir_ValidDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	i18nDir := filepath.Join(tempDir, "i18n")
	require.NoError(t, os.MkdirAll(i18nDir, 0o750))

	enJSON := `{"greeting": "Hello", "farewell": "Goodbye"}`
	require.NoError(t, os.WriteFile(filepath.Join(i18nDir, "en.json"), []byte(enJSON), 0o600))

	frJSON := `{"greeting": "Bonjour"}`
	require.NoError(t, os.WriteFile(filepath.Join(i18nDir, "fr.json"), []byte(frJSON), 0o600))

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	service, err := newFSServiceFromDir(context.Background(), sandbox, "i18n", "en")
	require.NoError(t, err)
	require.NotNil(t, service)

	store := service.GetStore()
	require.NotNil(t, store)

	entry, found := store.Get("en", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello", entry.Template)

	entry, found = store.Get("fr", "greeting")
	require.True(t, found)
	assert.Equal(t, "Bonjour", entry.Template)
}

func TestNewFSServiceFromDir_SkipsNonJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	i18nDir := filepath.Join(tempDir, "i18n")
	require.NoError(t, os.MkdirAll(i18nDir, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(i18nDir, "en.json"), []byte(`{"test": "value"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(i18nDir, "readme.txt"), []byte("not json"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(i18nDir, "subdir"), 0o750))

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	service, err := newFSServiceFromDir(context.Background(), sandbox, "i18n", "en")
	require.NoError(t, err)
	require.NotNil(t, service)

	store := service.GetStore()
	require.NotNil(t, store)
	assert.Len(t, store.Locales(), 1)
	assert.Contains(t, store.Locales(), "en")
}

func TestNewFlatBufferProvider(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	provider := newFlatBufferProvider(sandbox, "i18n.bin")
	require.NotNil(t, provider)
	assert.Equal(t, "i18n.bin", provider.filePath)
	assert.Nil(t, provider.data)
}

func TestFlatBufferProvider_RawData_BeforeLoad(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	provider := newFlatBufferProvider(sandbox, "i18n.bin")
	assert.Nil(t, provider.rawData())
}

func TestFlatBufferProvider_RawData_AfterLoad(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	store := i18n_domain.NewStore("en")
	store.AddTranslations("en", map[string]string{"test": "value"})
	emitter := NewFlatBufferEmitter(sandbox)
	require.NoError(t, emitter.Emit(context.Background(), store, "en", "i18n.bin"))

	provider := newFlatBufferProvider(sandbox, "i18n.bin")
	_, err := provider.load()
	require.NoError(t, err)
	assert.NotNil(t, provider.rawData())
	assert.NotEmpty(t, provider.rawData())
}

func TestNewJSONProvider(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	provider := newJSONProvider(sandbox, "i18n")
	require.NotNil(t, provider)
	assert.Equal(t, "i18n", provider.directory)
}

func TestJSONProvider_Load_EmptyDirectory(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	provider := newJSONProvider(sandbox, "")
	_, err := provider.load("en")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a valid directory path")
}

func TestJSONProvider_Load_NonexistentDir(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	provider := newJSONProvider(sandbox, "nonexistent")
	_, err := provider.load("en")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read i18n directory")
}

func TestJSONProvider_Load_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	i18nDir := filepath.Join(tempDir, "i18n")
	require.NoError(t, os.MkdirAll(i18nDir, 0o750))

	require.NoError(t, os.WriteFile(
		filepath.Join(i18nDir, "en.json"),
		[]byte(`{"hello": "world", "test": "value"}`),
		0o600,
	))

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	provider := newJSONProvider(sandbox, "i18n")
	store, err := provider.load("en")
	require.NoError(t, err)
	require.NotNil(t, store)

	entry, found := store.Get("en", "hello")
	require.True(t, found)
	assert.Equal(t, "world", entry.Template)
}

func TestJSONProvider_Load_InvalidJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	i18nDir := filepath.Join(tempDir, "i18n")
	require.NoError(t, os.MkdirAll(i18nDir, 0o750))

	require.NoError(t, os.WriteFile(
		filepath.Join(i18nDir, "en.json"),
		[]byte(`{invalid json}`),
		0o600,
	))

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	provider := newJSONProvider(sandbox, "i18n")
	_, err := provider.load("en")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load")
}

func TestPartKindToSchema(t *testing.T) {
	t.Parallel()

	t.Run("literal", func(t *testing.T) {
		t.Parallel()

		result := partKindToSchema(i18n_domain.PartLiteral)
		assert.NotNil(t, result)
	})

	t.Run("expression", func(t *testing.T) {
		t.Parallel()

		result := partKindToSchema(i18n_domain.PartExpression)
		assert.NotNil(t, result)
	})

	t.Run("linked message", func(t *testing.T) {
		t.Parallel()

		result := partKindToSchema(i18n_domain.PartLinkedMessage)
		assert.NotNil(t, result)
	})
}

func TestSchemaToPartKind_RoundTrip(t *testing.T) {
	t.Parallel()

	schemaLit := partKindToSchema(i18n_domain.PartLiteral)
	domainLit := schemaToPartKind(schemaLit)
	assert.Equal(t, i18n_domain.PartLiteral, domainLit)

	schemaExpr := partKindToSchema(i18n_domain.PartExpression)
	domainExpr := schemaToPartKind(schemaExpr)
	assert.Equal(t, i18n_domain.PartExpression, domainExpr)

	schemaLinked := partKindToSchema(i18n_domain.PartLinkedMessage)
	domainLinked := schemaToPartKind(schemaLinked)
	assert.Equal(t, i18n_domain.PartLinkedMessage, domainLinked)
}

func TestEmitFromTranslations(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	emitter := NewFlatBufferEmitter(sandbox)

	translations := i18n_domain.Translations{
		"en": {"hello": "Hello, World!", "bye": "Goodbye!"},
		"fr": {"hello": "Bonjour le monde!"},
	}

	err := emitter.EmitFromTranslations(context.Background(), translations, "en", "test_output.bin")
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, "test_output.bin")
	store, err := provider.load()
	require.NoError(t, err)

	entry, found := store.Get("en", "hello")
	require.True(t, found)
	assert.Equal(t, "Hello, World!", entry.Template)

	entry, found = store.Get("fr", "hello")
	require.True(t, found)
	assert.Equal(t, "Bonjour le monde!", entry.Template)
}

func TestNewFlatBufferEmitter(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	emitter := NewFlatBufferEmitter(sandbox)
	require.NotNil(t, emitter)
	assert.Equal(t, sandbox, emitter.sandbox)
}

func TestNewJSONEmitter(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	emitter := newJSONEmitter(sandbox)
	require.NotNil(t, emitter)
}

func TestJSONEmitter_Emit(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	store := i18n_domain.NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
		"farewell": "Goodbye",
	})
	store.AddTranslations("fr", map[string]string{
		"greeting": "Bonjour",
	})

	emitter := newJSONEmitter(sandbox)
	err := emitter.emit(store, "output")
	require.NoError(t, err)

	enData, err := os.ReadFile(filepath.Join(tempDir, "output", "en.json"))
	require.NoError(t, err)
	assert.Contains(t, string(enData), "Hello")

	frData, err := os.ReadFile(filepath.Join(tempDir, "output", "fr.json"))
	require.NoError(t, err)
	assert.Contains(t, string(frData), "Bonjour")
}

func TestJSONEmitter_EmitSingle(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	store := i18n_domain.NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
	})
	store.AddTranslations("fr", map[string]string{
		"greeting": "Bonjour",
	})

	emitter := newJSONEmitter(sandbox)
	err := emitter.emitSingle(store, "output/all.json")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tempDir, "output", "all.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "Hello")
	assert.Contains(t, string(data), "Bonjour")
}

func TestJSONEmitter_EmitSingle_EmptyStore(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	store := i18n_domain.NewStore("en")

	emitter := newJSONEmitter(sandbox)
	err := emitter.emitSingle(store, "output/empty.json")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tempDir, "output", "empty.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "{}")
}

func TestFlatBufferEmitter_EmptyStore(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	store := i18n_domain.NewStore("en")
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en", "empty.bin")
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, "empty.bin")
	loadedStore, err := provider.load()
	require.NoError(t, err)
	assert.Empty(t, loadedStore.Locales())
}

func TestFlatBufferRoundTrip_LinkedMessages(t *testing.T) {
	store := i18n_domain.NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
		"welcome":  "@:greeting, welcome!",
	})

	tempDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en", "linked.bin")
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, "linked.bin")
	loadedStore, err := provider.load()
	require.NoError(t, err)

	entry, found := loadedStore.Get("en", "welcome")
	require.True(t, found)
	assert.Equal(t, "@:greeting, welcome!", entry.Template)
}

func TestLoadJSONFile_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "en.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"key1": "value1", "key2": "value2"}`), 0o600))

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	translations, err := loadJSONFile(sandbox, "en.json")
	require.NoError(t, err)
	assert.Equal(t, "value1", translations["key1"])
	assert.Equal(t, "value2", translations["key2"])
}

func TestLoadJSONFile_FileNotFound(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	_, err := loadJSONFile(sandbox, "nonexistent.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading i18n JSON file")
}

func TestLoadJSONFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "bad.json"), []byte("{invalid}"), 0o600))

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	_, err := loadJSONFile(sandbox, "bad.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing and flattening")
}

func TestI18nAdapterConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 512, defaultStrBufPoolCapacity)
	assert.Equal(t, "i18n.bin", flatBufferFileName)
	assert.Equal(t, 4096, initialBuilderSize)
	assert.Equal(t, 4, uOffsetTSize)
	assert.Equal(t, "1.0.0", schemaVersion)
}
