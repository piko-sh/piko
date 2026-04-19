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

package driver_markdown

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/wdk/safedisk"
)

type stubAssetRegistrar struct {
	serveURLFor   func(sandboxRelPath string) string
	registerError error
	mu    sync.Mutex
	calls []stubRegistrarCall
}

type stubRegistrarCall struct {
	sandboxRelPath string
	collectionName string
}

func (s *stubAssetRegistrar) RegisterCollectionAsset(
	_ context.Context,
	_ safedisk.Sandbox,
	sandboxRelPath string,
	collectionName string,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, stubRegistrarCall{sandboxRelPath, collectionName})
	if s.registerError != nil {
		return "", s.registerError
	}
	if s.serveURLFor != nil {
		return s.serveURLFor(sandboxRelPath), nil
	}
	return "/_piko/assets/" + sandboxRelPath, nil
}

func newSandboxForRewriter(t *testing.T, files map[string][]byte) safedisk.Sandbox {
	t.Helper()
	directory := t.TempDir()
	for relativePath, data := range files {
		full := filepath.Join(directory, relativePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, data, 0o644))
	}
	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })
	return sandbox
}

func TestRewriteRelativeURLs_ImgDoubleQuoted(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"diagrams/one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	raw := `<p><img src="../diagrams/one.svg" alt="x"/></p>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"tutorials", "docs", "tutorials/foo.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<p><img src="/_piko/assets/diagrams/one.svg" alt="x"/></p>`, result)
	require.Len(t, registrar.calls, 1)
	assert.Equal(t, "diagrams/one.svg", registrar.calls[0].sandboxRelPath)
	assert.Equal(t, "docs", registrar.calls[0].collectionName)
}

func TestRewriteRelativeURLs_ImgSingleQuoted(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"diagrams/one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	raw := `<img src='../diagrams/one.svg' alt='x'/>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"tutorials", "docs", "tutorials/foo.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<img src='/_piko/assets/diagrams/one.svg' alt='x'/>`, result)
}

func TestRewriteRelativeURLs_ImgUnquotedValue(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	raw := `<img src=one.svg>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"", "docs", "foo.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<img src=/_piko/assets/one.svg>`, result)
}

func TestRewriteRelativeURLs_MultipleImages(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"diagrams/one.svg": []byte("<svg/>"),
		"diagrams/two.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	raw := `<div><img src="../diagrams/one.svg"/><span/><img src="../diagrams/two.svg"/></div>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"tutorials", "docs", "tutorials/foo.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<div><img src="/_piko/assets/diagrams/one.svg"/><span/><img src="/_piko/assets/diagrams/two.svg"/></div>`, result)
	require.Len(t, registrar.calls, 2)
}

func TestRewriteRelativeURLs_ImgUppercaseTagAndAttribute(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	raw := `<IMG SRC="one.svg"/>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"", "docs", "foo.md", raw)
	require.True(t, changed)
	assert.Contains(t, result, `/_piko/assets/one.svg`)
}

func TestRewriteRelativeURLs_ImgSkipsAbsoluteURLs(t *testing.T) {
	registrar := &stubAssetRegistrar{}
	raw := `<img src="https://example.com/x.png"/><img src="/absolute/y.png"/><img src="data:image/png;base64,AAA"/>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil), registrar, nil,
		"", "docs", "foo.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
	assert.Empty(t, registrar.calls)
}

func TestRewriteRelativeURLs_ImgSkipsPathEscape(t *testing.T) {
	registrar := &stubAssetRegistrar{}
	raw := `<img src="../../../etc/passwd"/>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil), registrar, nil,
		"tutorials", "docs", "tutorials/foo.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
	assert.Empty(t, registrar.calls)
}

func TestRewriteRelativeURLs_IgnoresNonRewriteableTags(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	raw := `<video src="one.svg"/><source src="one.svg"/>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"", "docs", "foo.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
	assert.Empty(t, registrar.calls)
}

func TestRewriteRelativeURLs_RegistrarErrorPreservesSrc(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{registerError: errors.New("simulated failure")}
	raw := `<img src="one.svg"/>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"", "docs", "foo.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_CancelledContext(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	raw := `<img src="one.svg"/>`
	result, changed := rewriteRelativeURLs(ctx, sandbox, registrar, nil,
		"", "docs", "foo.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_NilRegistrarAndAnalyser(t *testing.T) {
	raw := `<img src="one.svg"/><a href="x.md">link</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		collection_domain.AssetRegistrar(nil), nil, "", "docs", "foo.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_NilRegistrarOnlyAnchorsRewrite(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<img src="one.svg"/><a href="concepts.md">link</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<img src="one.svg"/><a href="/docs/get-started/concepts">link</a>`, result)
}

func TestRewriteRelativeURLs_EmptyRawHTML(t *testing.T) {
	registrar := &stubAssetRegistrar{}
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil), registrar, nil,
		"", "docs", "foo.md", "")
	assert.False(t, changed)
	assert.Equal(t, "", result)
}

func TestRewriteRelativeURLs_MaxReplacements(t *testing.T) {
	files := map[string][]byte{}
	var builder strings.Builder
	total := maxRawHTMLReplacements + 10
	files["img.svg"] = []byte("<svg/>")
	for range total {
		builder.WriteString(`<img src="img.svg"/>`)
	}
	registrar := &stubAssetRegistrar{}
	sandbox := newSandboxForRewriter(t, files)
	_, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, nil,
		"", "docs", "foo.md", builder.String())
	assert.True(t, changed)
	registrar.mu.Lock()
	count := len(registrar.calls)
	registrar.mu.Unlock()
	assert.LessOrEqual(t, count, maxRawHTMLReplacements)
}

func TestRewriteRelativeURLs_AnchorRelativeMd(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<p>see <a href="../tutorials/01-your-first-page.md">first page</a></p>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<p>see <a href="/docs/tutorials/01-your-first-page">first page</a></p>`, result)
}

func TestRewriteRelativeURLs_AnchorPreservesFragment(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="concepts.md#installation">link</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<a href="/docs/get-started/concepts#installation">link</a>`, result)
}

func TestRewriteRelativeURLs_AnchorPreservesQuery(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="concepts.md?from=install">link</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<a href="/docs/get-started/concepts?from=install">link</a>`, result)
}

func TestRewriteRelativeURLs_AnchorSkipsAbsolute(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="https://github.com/x/y.md">x</a><a href="/docs/foo">y</a><a href="//cdn/page.md">z</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_AnchorSkipsMailtoTel(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="mailto:foo@example.com">m</a><a href="tel:+44">t</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_AnchorSkipsFragmentOnly(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="#section">jump</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_AnchorSkipsNonMarkdown(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="../assets/file.pdf">pdf</a><a href="script.js">js</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_AnchorSkipsPathEscape(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="../../etc/passwd.md">x</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	assert.False(t, changed)
	assert.Equal(t, raw, result)
}

func TestRewriteRelativeURLs_AnchorUppercaseTagAndAttribute(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<A HREF="concepts.md">x</A>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	require.True(t, changed)
	assert.Contains(t, result, `/docs/get-started/concepts`)
}

func TestRewriteRelativeURLs_AnchorIndexFile(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<a href="../tutorials/index.md">overview</a>`
	result, changed := rewriteRelativeURLs(context.Background(), newSandboxForRewriter(t, nil),
		nil, analyser, "get-started", "docs", "get-started/install.md", raw)
	require.True(t, changed)
	assert.Equal(t, `<a href="/docs/tutorials/">overview</a>`, result)
}

func TestRewriteRelativeURLs_MixedImgAndAnchorSinglePass(t *testing.T) {
	sandbox := newSandboxForRewriter(t, map[string][]byte{
		"diagrams/one.svg": []byte("<svg/>"),
	})
	registrar := &stubAssetRegistrar{}
	analyser := newPathAnalyser([]string{"en"}, "en")
	raw := `<p><img src="../diagrams/one.svg"/> see <a href="../tutorials/foo.md">foo</a></p>`
	result, changed := rewriteRelativeURLs(context.Background(), sandbox, registrar, analyser,
		"get-started", "docs", "get-started/install.md", raw)
	require.True(t, changed)
	assert.Equal(t,
		`<p><img src="/_piko/assets/diagrams/one.svg"/> see <a href="/docs/tutorials/foo">foo</a></p>`,
		result)
}

func TestSplitQuotes_DoubleQuoted(t *testing.T) {
	trimmed, opener, closer := splitQuotes([]byte(`"value"`))
	assert.Equal(t, []byte("value"), trimmed)
	assert.Equal(t, byte('"'), opener)
	assert.Equal(t, byte('"'), closer)
}

func TestSplitQuotes_SingleQuoted(t *testing.T) {
	trimmed, opener, closer := splitQuotes([]byte(`'value'`))
	assert.Equal(t, []byte("value"), trimmed)
	assert.Equal(t, byte('\''), opener)
	assert.Equal(t, byte('\''), closer)
}

func TestSplitQuotes_Unquoted(t *testing.T) {
	trimmed, opener, closer := splitQuotes([]byte(`value`))
	assert.Equal(t, []byte("value"), trimmed)
	assert.Equal(t, byte(0), opener)
	assert.Equal(t, byte(0), closer)
}

func TestSplitQuotes_OnlyOpenQuote(t *testing.T) {
	trimmed, opener, closer := splitQuotes([]byte(`"value`))
	assert.Equal(t, []byte("value"), trimmed)
	assert.Equal(t, byte('"'), opener)
	assert.Equal(t, byte(0), closer)
}

func TestSplitQuotes_Empty(t *testing.T) {
	trimmed, opener, closer := splitQuotes(nil)
	assert.Empty(t, trimmed)
	assert.Equal(t, byte(0), opener)
	assert.Equal(t, byte(0), closer)
}

func TestApplyReplacements_InOrder(t *testing.T) {
	source := []byte(`<img src="old1"/><img src="old2"/>`)
	reps := []srcReplacement{
		{start: 9, end: 15, replacement: `"new1"`},
		{start: 26, end: 32, replacement: `"new2"`},
	}
	got := applyReplacements(string(source), source, reps)
	assert.Equal(t, `<img src="new1"/><img src="new2"/>`, got)
}

func TestApplyReplacements_OutOfOrder(t *testing.T) {
	source := []byte(`AAAABBBBCCCC`)
	reps := []srcReplacement{
		{start: 8, end: 12, replacement: "ccc"},
		{start: 0, end: 4, replacement: "aaa"},
		{start: 4, end: 8, replacement: "bbb"},
	}
	got := applyReplacements(string(source), source, reps)
	assert.Equal(t, "aaabbbccc", got)
}

func TestApplyReplacements_EmptyList(t *testing.T) {
	source := []byte(`unchanged`)
	got := applyReplacements(string(source), source, nil)
	assert.Equal(t, "unchanged", got)
}

func TestApplyReplacements_IgnoresInvalidSpans(t *testing.T) {
	source := []byte(`hello world`)
	reps := []srcReplacement{
		{start: -1, end: 3, replacement: "X"},
		{start: 6, end: 100, replacement: "Y"},
	}
	got := applyReplacements(string(source), source, reps)
	assert.Equal(t, "hello world", got)
}

func TestSplitURLSuffix(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantPath   string
		wantSuffix string
	}{
		{"no suffix", "foo.md", "foo.md", ""},
		{"fragment", "foo.md#section", "foo.md", "#section"},
		{"query", "foo.md?x=1", "foo.md", "?x=1"},
		{"query then fragment", "foo.md?x=1#a", "foo.md", "?x=1#a"},
		{"fragment then query", "foo.md#a?x=1", "foo.md", "#a?x=1"},
		{"fragment only", "#section", "", "#section"},
		{"query only", "?x=1", "", "?x=1"},
		{"empty", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPath, gotSuffix := splitURLSuffix(c.input)
			assert.Equal(t, c.wantPath, gotPath)
			assert.Equal(t, c.wantSuffix, gotSuffix)
		})
	}
}

func TestResolveAnchorHref(t *testing.T) {
	analyser := newPathAnalyser([]string{"en"}, "en")
	cases := []struct {
		name        string
		href        string
		mdDirectory string
		want        string
		wantOK      bool
	}{
		{"sibling md", "concepts.md", "get-started", "/docs/get-started/concepts", true},
		{"parent md", "../tutorials/foo.md", "get-started", "/docs/tutorials/foo", true},
		{"with fragment", "concepts.md#install", "get-started", "/docs/get-started/concepts#install", true},
		{"with query", "concepts.md?ref=x", "get-started", "/docs/get-started/concepts?ref=x", true},
		{"index file", "../tutorials/index.md", "get-started", "/docs/tutorials/", true},
		{"deeply nested", "a/b/c.md", "", "/docs/a/b/c", true},
		{"root file", "intro.md", "", "/docs/intro", true},
		{"https url", "https://example.com/x.md", "get-started", "", false},
		{"http url", "http://example.com/x.md", "get-started", "", false},
		{"protocol relative", "//cdn/x.md", "get-started", "", false},
		{"absolute path", "/docs/foo", "get-started", "", false},
		{"mailto", "mailto:foo@example.com", "get-started", "", false},
		{"tel", "tel:+1", "get-started", "", false},
		{"data uri", "data:text/plain,hello", "get-started", "", false},
		{"fragment only", "#section", "get-started", "", false},
		{"query only", "?x=1", "get-started", "", false},
		{"non-md path", "foo.txt", "get-started", "", false},
		{"non-md image", "../diagrams/x.svg", "get-started", "", false},
		{"empty", "", "get-started", "", false},
		{"escapes sandbox", "../../etc/passwd.md", "get-started", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := resolveAnchorHref(c.href, c.mdDirectory, "docs", analyser)
			assert.Equal(t, c.wantOK, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestResolveAnchorHref_NilAnalyser(t *testing.T) {
	got, ok := resolveAnchorHref("foo.md", "", "docs", nil)
	assert.False(t, ok)
	assert.Empty(t, got)
}

func TestResolveAnchorHref_NonDefaultLocale(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr"}, "en")

	got, ok := resolveAnchorHref("concepts.fr.md", "get-started", "docs", analyser)
	require.True(t, ok)
	assert.Equal(t, "/fr/docs/get-started/concepts", got)
}
