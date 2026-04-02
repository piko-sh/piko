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

package collection_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderConfig_GetCustomString(t *testing.T) {
	t.Parallel()

	config := &ProviderConfig{
		Custom: map[string]any{
			"apiURL":  "https://cms.example.com",
			"timeout": 30,
		},
	}

	assert.Equal(t, "https://cms.example.com", config.GetCustomString("apiURL", ""))
	assert.Equal(t, "fallback", config.GetCustomString("missing", "fallback"))
	assert.Equal(t, "fallback", config.GetCustomString("timeout", "fallback"))
}

func TestProviderConfig_GetCustomInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		custom       map[string]any
		key          string
		defaultValue int
		want         int
	}{
		{name: "int value", custom: map[string]any{"timeout": 30}, key: "timeout", defaultValue: 0, want: 30},
		{name: "int64 value", custom: map[string]any{"timeout": int64(60)}, key: "timeout", defaultValue: 0, want: 60},
		{name: "float64 value", custom: map[string]any{"timeout": float64(90)}, key: "timeout", defaultValue: 0, want: 90},
		{name: "missing key", custom: map[string]any{}, key: "timeout", defaultValue: 42, want: 42},
		{name: "non-numeric", custom: map[string]any{"timeout": "slow"}, key: "timeout", defaultValue: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &ProviderConfig{Custom: tt.custom}
			assert.Equal(t, tt.want, config.GetCustomInt(tt.key, tt.defaultValue))
		})
	}
}

func TestProviderConfig_GetCustomBool(t *testing.T) {
	t.Parallel()

	config := &ProviderConfig{
		Custom: map[string]any{
			"useHTTPS": true,
			"debug":    false,
			"name":     "test",
		},
	}

	assert.True(t, config.GetCustomBool("useHTTPS", false))
	assert.False(t, config.GetCustomBool("debug", true))
	assert.True(t, config.GetCustomBool("missing", true))
	assert.False(t, config.GetCustomBool("name", false))
}

func TestProviderConfig_HasCustom(t *testing.T) {
	t.Parallel()

	config := &ProviderConfig{
		Custom: map[string]any{"apiURL": "https://example.com"},
	}

	assert.True(t, config.HasCustom("apiURL"))
	assert.False(t, config.HasCustom("missing"))
}

func TestCollectionInfo_HasLocale(t *testing.T) {
	t.Parallel()

	info := &CollectionInfo{Locales: []string{"en", "fr", "de"}}

	assert.True(t, info.HasLocale("en"))
	assert.True(t, info.HasLocale("fr"))
	assert.False(t, info.HasLocale("ja"))
}

func TestCollectionInfo_IsMultiLocale(t *testing.T) {
	t.Parallel()

	t.Run("multiple locales", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{Locales: []string{"en", "fr"}}
		assert.True(t, info.IsMultiLocale())
	})

	t.Run("single locale", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{Locales: []string{"en"}}
		assert.False(t, info.IsMultiLocale())
	})

	t.Run("no locales", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{}
		assert.False(t, info.IsMultiLocale())
	})
}

func TestCollectionInfo_HasSchema(t *testing.T) {
	t.Parallel()

	t.Run("with schema", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{Schema: map[string]string{"title": "string"}}
		assert.True(t, info.HasSchema())
	})

	t.Run("empty schema", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{Schema: map[string]string{}}
		assert.False(t, info.HasSchema())
	})

	t.Run("nil schema", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{}
		assert.False(t, info.HasSchema())
	})
}

func TestCollectionInfo_GetFieldType(t *testing.T) {
	t.Parallel()

	t.Run("known field", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{Schema: map[string]string{
			"title": "string",
			"views": "int",
		}}
		assert.Equal(t, "string", info.GetFieldType("title"))
		assert.Equal(t, "int", info.GetFieldType("views"))
	})

	t.Run("unknown field", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{Schema: map[string]string{"title": "string"}}
		assert.Equal(t, "", info.GetFieldType("missing"))
	})

	t.Run("nil schema", func(t *testing.T) {
		t.Parallel()

		info := &CollectionInfo{}
		assert.Equal(t, "", info.GetFieldType("title"))
	})
}
