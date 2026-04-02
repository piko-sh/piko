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

func TestRender_NilEntry(t *testing.T) {
	buffer := NewStrBuf(64)
	result := Render(nil, nil, nil, "en", buffer)
	assert.Equal(t, "", result)
}

func TestRender_SimpleLiteral(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Hello, World!")
	entry := &Entry{
		Template: "Hello, World!",
		Parts:    parts,
	}

	result := Render(entry, nil, nil, "en", buffer)
	assert.Equal(t, "Hello, World!", result)
}

func TestRender_SinglePlaceholder(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Hello, ${name}!")
	entry := &Entry{
		Template: "Hello, ${name}!",
		Parts:    parts,
	}
	vars := map[string]any{"name": "Alice"}

	result := Render(entry, vars, nil, "en", buffer)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestRender_MultiplePlaceholders(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("${greeting}, ${name}! You have ${count} messages.")
	entry := &Entry{
		Template: "${greeting}, ${name}! You have ${count} messages.",
		Parts:    parts,
	}
	vars := map[string]any{
		"greeting": "Hello",
		"name":     "Bob",
		"count":    42,
	}

	result := Render(entry, vars, nil, "en", buffer)
	assert.Equal(t, "Hello, Bob! You have 42 messages.", result)
}

func TestRender_MissingPlaceholder(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Hello, ${name}!")
	entry := &Entry{
		Template: "Hello, ${name}!",
		Parts:    parts,
	}

	result := Render(entry, nil, nil, "en", buffer)
	assert.Equal(t, "Hello, ${name}!", result)
}

func TestRender_CountPlaceholder(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("You have ${count} items.")
	entry := &Entry{
		Template: "You have ${count} items.",
		Parts:    parts,
	}
	result := Render(entry, nil, new(5), "en", buffer)
	assert.Equal(t, "You have 5 items.", result)
}

func TestRender_CountPlaceholderZero(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("You have ${count} items.")
	entry := &Entry{
		Template: "You have ${count} items.",
		Parts:    parts,
	}
	result := Render(entry, nil, new(0), "en", buffer)
	assert.Equal(t, "You have 0 items.", result)
}

func TestRender_PluralForms(t *testing.T) {
	buffer := NewStrBuf(64)
	entry := &Entry{
		Template:    "one item|${count} items",
		PluralForms: SplitPluralForms("one item|${count} items"),
		HasPlurals:  true,
	}

	count := 1
	result := Render(entry, nil, &count, "en", buffer)
	assert.Equal(t, "one item", result)

	count = 5
	result = Render(entry, nil, &count, "en", buffer)
	assert.Equal(t, "5 items", result)
}

func TestRender_PluralFormsWithVars(t *testing.T) {
	buffer := NewStrBuf(64)
	entry := &Entry{
		Template:    "${name} has one item|${name} has ${count} items",
		PluralForms: SplitPluralForms("${name} has one item|${name} has ${count} items"),
		HasPlurals:  true,
	}
	vars := map[string]any{"name": "Alice"}

	count := 1
	result := Render(entry, vars, &count, "en", buffer)
	assert.Equal(t, "Alice has one item", result)

	count = 3
	result = Render(entry, vars, &count, "en", buffer)
	assert.Equal(t, "Alice has 3 items", result)
}

func TestRender_IntVariable(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Total: ${total}")
	entry := &Entry{
		Template: "Total: ${total}",
		Parts:    parts,
	}
	vars := map[string]any{"total": 100}

	result := Render(entry, vars, nil, "en", buffer)
	assert.Equal(t, "Total: 100", result)
}

func TestRender_FloatVariable(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Price: ${price}")
	entry := &Entry{
		Template: "Price: ${price}",
		Parts:    parts,
	}
	vars := map[string]any{"price": 19.99}

	result := Render(entry, vars, nil, "en", buffer)
	assert.Equal(t, "Price: 19.99", result)
}

func TestRenderSimple_Basic(t *testing.T) {
	buffer := NewStrBuf(64)
	vars := map[string]any{"name": "World"}

	result := renderSimple("Hello, ${name}!", vars, buffer)
	assert.Equal(t, "Hello, World!", result)
}

func TestRenderSimple_NoVars(t *testing.T) {
	buffer := NewStrBuf(64)

	result := renderSimple("Hello, World!", nil, buffer)
	assert.Equal(t, "Hello, World!", result)
}

func TestRenderSimple_MissingVar(t *testing.T) {
	buffer := NewStrBuf(64)
	vars := map[string]any{}

	result := renderSimple("Hello, ${name}!", vars, buffer)
	assert.Equal(t, "Hello, ${name}!", result)
}

func TestRenderWithVars_Translation(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello, ${name}!",
	})

	pool := NewStrBufPool(64)
	entry, _ := store.Get("en", "greeting")
	trans := NewTranslationWithLocale("greeting", entry, pool, "en")
	trans.StringVar("name", "Alice")

	result := trans.String()
	assert.Equal(t, "Hello, Alice!", result)
}

func TestRenderWithVars_TranslationWithCount(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"items": "one item|${count} items",
	})

	pool := NewStrBufPool(64)
	entry, _ := store.Get("en", "items")

	trans := NewTranslationWithLocale("items", entry, pool, "en")
	trans.Count(1)
	result := trans.String()
	assert.Equal(t, "one item", result)

	trans = NewTranslationWithLocale("items", entry, pool, "en")
	trans.Count(5)
	result = trans.String()
	assert.Equal(t, "5 items", result)
}

func TestRender_LinkedMessage(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"app_name":        "MyApp",
		"welcome_message": "Welcome to @app_name!",
	})

	buffer := NewStrBuf(64)
	entry, _ := store.Get("en", "welcome_message")

	ctx := &renderContext{
		scope:    nil,
		resolver: store,
		locale:   "en",
		count:    nil,
		buffer:   buffer,
		depth:    0,
	}

	result := renderTemplate(entry.Parts, ctx)
	assert.Equal(t, "Welcome to MyApp!", result)
}

func TestRender_LinkedMessageWithVars(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello, ${name}",
		"message":  "@greeting! Welcome back.",
	})

	buffer := NewStrBuf(64)
	entry, _ := store.Get("en", "message")

	ctx := &renderContext{
		scope:    map[string]any{"name": "Alice"},
		resolver: store,
		locale:   "en",
		count:    nil,
		buffer:   buffer,
		depth:    0,
	}

	result := renderTemplate(entry.Parts, ctx)
	assert.Equal(t, "Hello, Alice! Welcome back.", result)
}

func TestRender_LinkedMessageNotFound(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"message": "Check out @missing.key here.",
	})

	buffer := NewStrBuf(64)
	entry, _ := store.Get("en", "message")

	ctx := &renderContext{
		scope:    nil,
		resolver: store,
		locale:   "en",
		count:    nil,
		buffer:   buffer,
		depth:    0,
	}

	result := renderTemplate(entry.Parts, ctx)
	assert.Equal(t, "Check out @missing.key here.", result)
}

func TestRender_EscapedDollar(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Price: \\$100")
	entry := &Entry{
		Template: "Price: \\$100",
		Parts:    parts,
	}

	result := Render(entry, nil, nil, "en", buffer)
	assert.Equal(t, "Price: $100", result)
}

func TestRender_EscapedAt(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Email: user\\@example.com")
	entry := &Entry{
		Template: "Email: user\\@example.com",
		Parts:    parts,
	}

	result := Render(entry, nil, nil, "en", buffer)
	assert.Equal(t, "Email: user@example.com", result)
}

func TestRender_MemberAccess(t *testing.T) {
	buffer := NewStrBuf(64)
	parts, _ := ParseTemplate("Hello, ${user.name}!")
	entry := &Entry{
		Template: "Hello, ${user.name}!",
		Parts:    parts,
	}
	vars := map[string]any{
		"user": map[string]any{
			"name": "Alice",
		},
	}

	result := Render(entry, vars, nil, "en", buffer)
	assert.Equal(t, "Hello, Alice!", result)
}

func BenchmarkRender_Simple(b *testing.B) {
	buffer := NewStrBuf(256)
	parts, _ := ParseTemplate("Hello, ${name}!")
	entry := &Entry{
		Template: "Hello, ${name}!",
		Parts:    parts,
	}
	vars := map[string]any{"name": "Alice"}
	b.ResetTimer()

	for b.Loop() {
		_ = Render(entry, vars, nil, "en", buffer)
	}
}

func BenchmarkRender_Complex(b *testing.B) {
	buffer := NewStrBuf(256)
	parts, _ := ParseTemplate("${greeting}, ${name}! You have ${count} new messages from ${sender}.")
	entry := &Entry{
		Template: "${greeting}, ${name}! You have ${count} new messages from ${sender}.",
		Parts:    parts,
	}
	vars := map[string]any{
		"greeting": "Hello",
		"name":     "Bob",
		"count":    42,
		"sender":   "Alice",
	}
	b.ResetTimer()

	for b.Loop() {
		_ = Render(entry, vars, nil, "en", buffer)
	}
}

func BenchmarkRender_Plurals(b *testing.B) {
	buffer := NewStrBuf(256)
	entry := &Entry{
		Template:    "one item|${count} items",
		PluralForms: SplitPluralForms("one item|${count} items"),
		HasPlurals:  true,
	}
	count := 5
	b.ResetTimer()

	for b.Loop() {
		_ = Render(entry, nil, &count, "en", buffer)
	}
}

func BenchmarkRenderWithVars_Translation(b *testing.B) {
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
