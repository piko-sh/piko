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

package seo_domain

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/wdk/safedisk"
)

func TestDetermineLastMod_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("uses explicit lastmod when provided", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		builder := newSitemapBuilder(
			config.SitemapConfig{Hostname: "https://example.com"},
			"en",
			nil,
			withSitemapSandbox(sandbox),
		)

		result := builder.determineLastMod(new(time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)), "/project/page.pk")

		assert.Equal(t, "2025-06-15", result)
	})

	t.Run("uses file modification time from sandbox when no explicit time", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		_ = sandbox.WriteFile("page.pk", []byte("content"), 0644)

		builder := newSitemapBuilder(
			config.SitemapConfig{Hostname: "https://example.com"},
			"en",
			nil,
			withSitemapSandbox(sandbox),
		)

		result := builder.determineLastMod(nil, "/project/page.pk")

		today := time.Now().Format("2006-01-02")
		assert.Equal(t, today, result)
	})

	t.Run("falls back to current time when Stat fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.StatErr = errors.New("file not found")

		builder := newSitemapBuilder(
			config.SitemapConfig{Hostname: "https://example.com"},
			"en",
			nil,
			withSitemapSandbox(sandbox),
		)

		result := builder.determineLastMod(nil, "/project/nonexistent.pk")

		today := time.Now().Format("2006-01-02")
		assert.Equal(t, today, result)
	})

	t.Run("falls back to current time when source path is empty", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		builder := newSitemapBuilder(
			config.SitemapConfig{Hostname: "https://example.com"},
			"en",
			nil,
			withSitemapSandbox(sandbox),
		)

		result := builder.determineLastMod(nil, "")

		today := time.Now().Format("2006-01-02")
		assert.Equal(t, today, result)
	})
}

func TestNewSitemapBuilder_WithSandboxOption(t *testing.T) {
	t.Parallel()

	t.Run("creates builder with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		builder := newSitemapBuilder(
			config.SitemapConfig{
				Hostname:          "https://example.com",
				MaxURLsPerSitemap: 1000,
			},
			"en",
			nil,
			withSitemapSandbox(sandbox),
		)

		assert.NotNil(t, builder)
		assert.Equal(t, sandbox, builder.sandbox)
	})

	t.Run("creates builder without sandbox option", func(t *testing.T) {
		t.Parallel()

		builder := newSitemapBuilder(
			config.SitemapConfig{
				Hostname:          "https://example.com",
				MaxURLsPerSitemap: 1000,
			},
			"en",
			nil,
		)

		assert.NotNil(t, builder)
		assert.Nil(t, builder.sandbox)
	})

	t.Run("applies default MaxURLsPerSitemap when zero", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		builder := newSitemapBuilder(
			config.SitemapConfig{
				Hostname:          "https://example.com",
				MaxURLsPerSitemap: 0,
			},
			"en",
			nil,
			withSitemapSandbox(sandbox),
		)

		assert.Equal(t, defaultMaxURLsPerSitemap, builder.config.MaxURLsPerSitemap)
	})
}
