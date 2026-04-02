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

package i18n_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testGetTranslation(store *Store, locale, key string, pool *StrBufPool) *Translation {
	entry, found := store.Get(locale, key)
	if !found {
		entry, found = store.Get(store.DefaultLocale(), key)
	}
	if !found {
		return NewTranslationFromString(key, key, pool)
	}
	return NewTranslationWithLocale(key, entry, pool, locale)
}

func TestTranslation_SimpleKey(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello, World!",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "greeting", pool)
	result := trans.String()

	assert.Equal(t, "Hello, World!", result)
}

func TestTranslation_Fallback(t *testing.T) {
	pool := NewStrBufPool(64)

	trans := NewTranslationFromString("missing", "Default Value", pool)
	result := trans.String()

	assert.Equal(t, "Default Value", result)
}

func TestTranslation_FallbackToKey(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "missing.key", pool)
	result := trans.String()

	assert.Equal(t, "missing.key", result)
}

func TestTranslation_StringVar(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello, ${name}!",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "greeting", pool)
	trans.StringVar("name", "Alice")
	result := trans.String()

	assert.Equal(t, "Hello, Alice!", result)
}

func TestTranslation_IntVar(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"count": "You have ${n} items.",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "count", pool)
	trans.IntVar("n", 42)
	result := trans.String()

	assert.Equal(t, "You have 42 items.", result)
}

func TestTranslation_FloatVar(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"price": "Total: ${amount}",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "price", pool)
	trans.FloatVar("amount", 19.99)
	result := trans.String()

	assert.Equal(t, "Total: 19.99", result)
}

func TestTranslation_Var(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"message": "Value: ${val}",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "message", pool)
	trans.Var("val", "anything")
	result := trans.String()

	assert.Equal(t, "Value: anything", result)
}

func TestTranslation_Count(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"items": "one item|${count} items",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "items", pool)
	trans.Count(1)
	assert.Equal(t, "one item", trans.String())

	trans = testGetTranslation(store, "en", "items", pool)
	trans.Count(5)
	assert.Equal(t, "5 items", trans.String())
}

func TestTranslation_Chaining(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"message": "${name} has ${count} items worth ${total}.",
	})
	pool := NewStrBufPool(64)

	result := testGetTranslation(store, "en", "message", pool).
		StringVar("name", "Alice").
		IntVar("count", 3).
		FloatVar("total", 29.97).
		String()

	assert.Equal(t, "Alice has 3 items worth 29.97.", result)
}

func TestTranslation_MultipleVars(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"full": "${a} ${b} ${c} ${d}",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "full", pool)
	trans.StringVar("a", "1")
	trans.StringVar("b", "2")
	trans.StringVar("c", "3")
	trans.StringVar("d", "4")
	result := trans.String()

	assert.Equal(t, "1 2 3 4", result)
}

func TestTranslation_OverflowVars(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"many": "${a} ${b} ${c} ${d} ${e} ${f}",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en", "many", pool)
	trans.StringVar("a", "1")
	trans.StringVar("b", "2")
	trans.StringVar("c", "3")
	trans.StringVar("d", "4")
	trans.StringVar("e", "5")
	trans.StringVar("f", "6")
	result := trans.String()

	assert.Equal(t, "1 2 3 4 5 6", result)
}

func TestTranslation_LocaleFallback(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
		"farewell": "Goodbye",
	})
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, mate",
	})
	pool := NewStrBufPool(64)

	trans := testGetTranslation(store, "en-GB", "greeting", pool)
	assert.Equal(t, "Hello, mate", trans.String())

	trans = testGetTranslation(store, "en-GB", "farewell", pool)
	assert.Equal(t, "Goodbye", trans.String())
}

func TestTranslation_LookupVar_NotFound(t *testing.T) {
	pool := NewStrBufPool(64)

	trans := NewTranslation("test", nil, pool)
	trans.StringVar("name", "Alice")

	lookupBuf := NewStrBuf(64)
	found := trans.LookupVar("missing", lookupBuf)
	assert.False(t, found)
}

func TestTranslation_LookupVar_Found(t *testing.T) {
	pool := NewStrBufPool(64)

	trans := NewTranslation("test", nil, pool)
	trans.StringVar("name", "Alice")

	lookupBuf := NewStrBuf(64)
	found := trans.LookupVar("name", lookupBuf)
	assert.True(t, found)
	assert.Equal(t, "Alice", lookupBuf.String())
}

func TestTranslation_LookupVar_InOverflow(t *testing.T) {
	pool := NewStrBufPool(64)

	trans := NewTranslation("test", nil, pool)
	trans.StringVar("a", "1")
	trans.StringVar("b", "2")
	trans.StringVar("c", "3")
	trans.StringVar("d", "4")
	trans.StringVar("e", "5")

	lookupBuf := NewStrBuf(64)
	found := trans.LookupVar("e", lookupBuf)
	assert.True(t, found)
	assert.Equal(t, "5", lookupBuf.String())
}

func TestTranslation_ImplementsStringer(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
	})
	pool := NewStrBufPool(64)

	var stringer interface{ String() string }
	trans := testGetTranslation(store, "en", "greeting", pool)
	stringer = trans

	assert.Equal(t, "Hello", stringer.String())
}

func BenchmarkTranslation_Simple(b *testing.B) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello, ${name}!",
	})
	pool := NewStrBufPool(256)
	entry, _ := store.Get("en", "greeting")
	b.ResetTimer()

	for b.Loop() {
		trans := NewTranslationWithLocale("greeting", entry, pool, "en")
		trans.StringVar("name", "Alice")
		_ = trans.String()
	}
}

func BenchmarkTranslation_MultipleVars(b *testing.B) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"message": "${a} ${b} ${c} ${d}",
	})
	pool := NewStrBufPool(256)
	entry, _ := store.Get("en", "message")
	b.ResetTimer()

	for b.Loop() {
		trans := NewTranslationWithLocale("message", entry, pool, "en")
		trans.StringVar("a", "one")
		trans.StringVar("b", "two")
		trans.StringVar("c", "three")
		trans.StringVar("d", "four")
		_ = trans.String()
	}
}

func BenchmarkTranslation_WithCount(b *testing.B) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"items": "one item|${count} items",
	})
	pool := NewStrBufPool(256)
	entry, _ := store.Get("en", "items")
	b.ResetTimer()

	i := 0
	for b.Loop() {
		trans := NewTranslationWithLocale("items", entry, pool, "en")
		trans.Count(i % 10)
		_ = trans.String()
		i++
	}
}

func BenchmarkTranslation_LookupVar(b *testing.B) {
	pool := NewStrBufPool(256)
	lookupBuf := NewStrBuf(64)

	trans := NewTranslation("test", nil, pool)
	trans.StringVar("name", "Alice")
	trans.IntVar("count", 42)
	trans.FloatVar("price", 19.99)
	b.ResetTimer()

	for b.Loop() {
		lookupBuf.Reset()
		_ = trans.LookupVar("count", lookupBuf)
	}
}
