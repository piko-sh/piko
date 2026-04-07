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

package compiler_adapters

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/cssinliner"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type inMemoryFSReader struct {
	files map[string][]byte
}

func (r *inMemoryFSReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	if content, ok := r.files[path]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func newTestCSSPreProcessor(files map[string][]byte) compiler_domain.CSSPreProcessorPort {
	mockResolver := &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(_ context.Context, importPath string, containingDir string) (string, error) {
			if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
				return filepath.Join(containingDir, filepath.FromSlash(importPath)), nil
			}
			return "", fmt.Errorf("unsupported import path in test: %s", importPath)
		},
	}
	processor := cssinliner.NewProcessor(cssinliner.ProcessorConfig{
		Resolver: mockResolver,
		Loader:   esbuildconfig.LoaderLocalCSS,
		Options: &esbuildconfig.Options{
			MinifyWhitespace: true,
			MinifySyntax:     true,
		},
	})
	return NewCSSPreProcessor(processor, &inMemoryFSReader{files: files}, "", "")
}

func TestCSSPreProcessor_InlineImports(t *testing.T) {
	t.Run("passes through CSS without imports", func(t *testing.T) {
		pp := newTestCSSPreProcessor(nil)
		result, err := pp.InlineImports(context.Background(), ".foo { color: red; }", "/test/style.css")
		require.NoError(t, err)
		assert.Contains(t, result, "color")
		assert.Contains(t, result, "red")
	})

	t.Run("inlines a relative import", func(t *testing.T) {
		files := map[string][]byte{
			"/test/base.css": []byte(".base { font-size: 16px; }"),
		}
		pp := newTestCSSPreProcessor(files)
		result, err := pp.InlineImports(context.Background(), `@import "./base.css";`, "/test/component.css")
		require.NoError(t, err)
		assert.Contains(t, result, "font-size")
		assert.NotContains(t, result, "@import")
	})

	t.Run("inlines nested imports", func(t *testing.T) {
		files := map[string][]byte{
			"/test/theme.css": []byte(`@import "./base.css"; .theme { color: blue; }`),
			"/test/base.css":  []byte(`.base { margin: 0; }`),
		}
		pp := newTestCSSPreProcessor(files)
		result, err := pp.InlineImports(context.Background(), `@import "./theme.css";`, "/test/component.css")
		require.NoError(t, err)
		assert.Contains(t, result, "margin")
		assert.Contains(t, result, "color")
		assert.NotContains(t, result, "@import")
	})

	t.Run("returns error for missing import file", func(t *testing.T) {
		pp := newTestCSSPreProcessor(nil)
		_, err := pp.InlineImports(context.Background(), `@import "./missing.css";`, "/test/component.css")
		require.Error(t, err)
	})
}

func TestCSSPreProcessor_ResolveToFilesystemPath(t *testing.T) {
	t.Run("returns path unchanged when moduleName is empty", func(t *testing.T) {
		p := &cssPreProcessor{moduleName: "", baseDir: "/project"}
		assert.Equal(t, "/some/path.css", p.resolveToFilesystemPath("/some/path.css"))
	})

	t.Run("returns path unchanged when baseDir is empty", func(t *testing.T) {
		p := &cssPreProcessor{moduleName: "github.com/org/repo", baseDir: ""}
		assert.Equal(t, "github.com/org/repo/styles/foo.css", p.resolveToFilesystemPath("github.com/org/repo/styles/foo.css"))
	})

	t.Run("returns path unchanged when not module-qualified", func(t *testing.T) {
		p := &cssPreProcessor{moduleName: "github.com/org/repo", baseDir: "/project"}
		assert.Equal(t, "/absolute/path.css", p.resolveToFilesystemPath("/absolute/path.css"))
	})

	t.Run("converts module-qualified path to filesystem path", func(t *testing.T) {
		p := &cssPreProcessor{moduleName: "github.com/org/repo", baseDir: "/project"}
		result := p.resolveToFilesystemPath("github.com/org/repo/components/widget.pkc")
		expected := filepath.Join("/project", "components", "widget.pkc")
		assert.Equal(t, expected, result)
	})

	t.Run("handles nested module-qualified paths", func(t *testing.T) {
		p := &cssPreProcessor{moduleName: "github.com/org/repo", baseDir: "/home/user/project"}
		result := p.resolveToFilesystemPath("github.com/org/repo/deep/nested/style.css")
		expected := filepath.Join("/home/user/project", "deep", "nested", "style.css")
		assert.Equal(t, expected, result)
	})

	t.Run("does not match partial module name prefix", func(t *testing.T) {
		p := &cssPreProcessor{moduleName: "github.com/org/repo", baseDir: "/project"}
		input := "github.com/org/repo-other/styles/foo.css"
		assert.Equal(t, input, p.resolveToFilesystemPath(input))
	})
}
