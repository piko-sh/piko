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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestFlatBufferRoundTrip_Simple(t *testing.T) {
	store := i18n_domain.NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, World!",
		"farewell": "Goodbye!",
	})

	tempDir := t.TempDir()
	relPath := "i18n.bin"
	absPath := filepath.Join(tempDir, relPath)

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(t, err)

	_, err = os.Stat(absPath)
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, relPath)
	loadedStore, err := provider.load()
	require.NoError(t, err)

	entry, found := loadedStore.Get("en-GB", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello, World!", entry.Template)

	entry, found = loadedStore.Get("en-GB", "farewell")
	require.True(t, found)
	assert.Equal(t, "Goodbye!", entry.Template)
}

func TestFlatBufferRoundTrip_MultipleLocales(t *testing.T) {
	store := i18n_domain.NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello",
	})
	store.AddTranslations("fr-FR", map[string]string{
		"greeting": "Bonjour",
	})
	store.AddTranslations("de-DE", map[string]string{
		"greeting": "Hallo",
	})

	tempDir := t.TempDir()
	relPath := "i18n.bin"

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, relPath)
	loadedStore, err := provider.load()
	require.NoError(t, err)

	locales := loadedStore.Locales()
	assert.Len(t, locales, 3)

	entry, found := loadedStore.Get("en-GB", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello", entry.Template)

	entry, found = loadedStore.Get("fr-FR", "greeting")
	require.True(t, found)
	assert.Equal(t, "Bonjour", entry.Template)

	entry, found = loadedStore.Get("de-DE", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hallo", entry.Template)
}

func TestFlatBufferRoundTrip_Placeholders(t *testing.T) {
	store := i18n_domain.NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, ${name}!",
		"message":  "${sender} says: ${text}",
	})

	tempDir := t.TempDir()
	relPath := "i18n.bin"

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, relPath)
	loadedStore, err := provider.load()
	require.NoError(t, err)

	entry, found := loadedStore.Get("en-GB", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello, ${name}!", entry.Template)
	require.Len(t, entry.Parts, 3)
	assert.Equal(t, i18n_domain.PartLiteral, entry.Parts[0].Kind)
	assert.Equal(t, "Hello, ", entry.Parts[0].Literal)
	assert.Equal(t, i18n_domain.PartExpression, entry.Parts[1].Kind)
	assert.Equal(t, "name", entry.Parts[1].ExprSource)
	assert.Equal(t, i18n_domain.PartLiteral, entry.Parts[2].Kind)
	assert.Equal(t, "!", entry.Parts[2].Literal)
}

func TestFlatBufferRoundTrip_Plurals(t *testing.T) {
	store := i18n_domain.NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"items": "one item|${count} items",
	})

	tempDir := t.TempDir()
	relPath := "i18n.bin"

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, relPath)
	loadedStore, err := provider.load()
	require.NoError(t, err)

	entry, found := loadedStore.Get("en-GB", "items")
	require.True(t, found)
	assert.True(t, entry.HasPlurals)
	require.Len(t, entry.PluralForms, 2)
	assert.Equal(t, "one item", entry.PluralForms[0])
	assert.Equal(t, "${count} items", entry.PluralForms[1])
}

func TestFlatBufferRoundTrip_ComplexTranslations(t *testing.T) {
	store := i18n_domain.NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting":      "Hello, ${name}!",
		"cart.items":    "You have one item|You have ${count} items",
		"order.summary": "${customer} ordered ${quantity} items for ${total}",
		"empty":         "",
	})
	store.AddTranslations("fr-FR", map[string]string{
		"greeting":   "Bonjour, ${name}!",
		"cart.items": "Vous avez un article|Vous avez ${count} articles",
	})

	tempDir := t.TempDir()
	relPath := "i18n.bin"

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, relPath)
	loadedStore, err := provider.load()
	require.NoError(t, err)

	entry, found := loadedStore.Get("en-GB", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello, ${name}!", entry.Template)

	entry, found = loadedStore.Get("en-GB", "cart.items")
	require.True(t, found)
	assert.True(t, entry.HasPlurals)

	entry, found = loadedStore.Get("en-GB", "order.summary")
	require.True(t, found)
	assert.Len(t, entry.Parts, 5)

	entry, found = loadedStore.Get("fr-FR", "greeting")
	require.True(t, found)
	assert.Equal(t, "Bonjour, ${name}!", entry.Template)
}

func TestFlatBufferRoundTrip_UsageWithTranslation(t *testing.T) {
	store := i18n_domain.NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting":   "Hello, ${name}!",
		"cart.items": "one item|${count} items",
	})

	tempDir := t.TempDir()
	relPath := "i18n.bin"

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(t, err)

	provider := newFlatBufferProvider(sandbox, relPath)
	loadedStore, err := provider.load()
	require.NoError(t, err)

	pool := i18n_domain.NewStrBufPool(256)

	entry, found := loadedStore.Get("en-GB", "greeting")
	require.True(t, found)
	trans := i18n_domain.NewTranslationWithLocale("greeting", entry, pool, "en-GB")
	trans.StringVar("name", "Alice")
	result := trans.String()
	assert.Equal(t, "Hello, Alice!", result)

	entry, found = loadedStore.Get("en-GB", "cart.items")
	require.True(t, found)
	trans = i18n_domain.NewTranslationWithLocale("cart.items", entry, pool, "en-GB")
	trans.Count(5)
	result = trans.String()
	assert.Equal(t, "5 items", result)
}

func TestFlatBufferProvider_FileNotFound(t *testing.T) {
	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	provider := newFlatBufferProvider(sandbox, "nonexistent/path/i18n.bin")
	_, err := provider.load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestFlatBufferProvider_EmptyPath(t *testing.T) {
	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	provider := newFlatBufferProvider(sandbox, "")
	_, err := provider.load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a valid file path")
}

func TestFlatBufferEmitter_AtomicWrite(t *testing.T) {
	store := i18n_domain.NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"test": "value",
	})

	tempDir := t.TempDir()
	relPath := filepath.Join("subdir", "i18n.bin")
	absPath := filepath.Join(tempDir, relPath)

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(t, err)

	_, err = os.Stat(absPath)
	require.NoError(t, err)

	tempPath := absPath + ".tmp"
	_, err = os.Stat(tempPath)
	assert.True(t, os.IsNotExist(err))
}

func BenchmarkFlatBufferLoad(b *testing.B) {
	store := i18n_domain.NewStore("en-GB")
	translations := make(map[string]string)
	for i := range 1000 {
		translations[fmt.Sprintf("key.%d", i)] = fmt.Sprintf("Translation %d with ${placeholder}", i)
	}
	store.AddTranslations("en-GB", translations)
	store.AddTranslations("fr-FR", translations)
	store.AddTranslations("de-DE", translations)

	tempDir := b.TempDir()
	relPath := "i18n.bin"

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		provider := newFlatBufferProvider(sandbox, relPath)
		_, err := provider.load()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlatBufferLookup(b *testing.B) {
	store := i18n_domain.NewStore("en-GB")
	translations := make(map[string]string)
	for i := range 1000 {
		translations[fmt.Sprintf("key.%d", i)] = fmt.Sprintf("Translation %d with ${placeholder}", i)
	}
	store.AddTranslations("en-GB", translations)

	tempDir := b.TempDir()
	relPath := "i18n.bin"

	sandbox, _ := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferEmitter(sandbox)
	err := emitter.Emit(context.Background(), store, "en-GB", relPath)
	require.NoError(b, err)

	provider := newFlatBufferProvider(sandbox, relPath)
	loadedStore, err := provider.load()
	require.NoError(b, err)

	b.ResetTimer()
	i := 0
	for b.Loop() {
		key := fmt.Sprintf("key.%d", i%1000)
		_, _ = loadedStore.Get("en-GB", key)
		i++
	}
}
