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

package pikotest_domain

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// httpStatusOK is the default HTTP success status code.
	httpStatusOK = 200

	// snapshotDirPermissions is the file mode for snapshot directories.
	snapshotDirPermissions = 0o750

	// snapshotFilePermissions is the file mode for snapshot golden files.
	snapshotFilePermissions = 0o600
)

// TestView wraps the result of a component render and provides methods to
// check state, metadata, and DOM structure.
type TestView struct {
	// tb is the test context for reporting errors and marking test helpers.
	tb testing.TB

	// state holds the current view state for test assertions.
	state any

	// renderer provides HTML rendering for test assertions; nil when not
	// available.
	renderer render_domain.RenderService

	// ast holds the parsed template structure for queries and rendering.
	ast *ast_domain.TemplateAST

	// request holds the HTTP request for CSRF token handling during rendering.
	request *http.Request

	// pageID is the page identifier used when rendering the template.
	pageID string

	// html stores the rendered HTML output as bytes.
	html []byte

	// metadata stores page settings for SEO, caching, and assets.
	metadata templater_dto.InternalMetadata

	// rendered indicates whether the HTML output has been generated.
	rendered bool
}

// State returns the raw state object from the component's Render function.
// Use a type assertion to access the fields of your component's response.
//
// Returns any which is the component's state, requiring type assertion.
func (v *TestView) State() any {
	return v.state
}

// AssertState provides a callback-based way to assert against the state.
//
// Takes callback (func(state any)) which receives the current
// state for assertions.
func (v *TestView) AssertState(callback func(state any)) {
	callback(v.state)
}

// Metadata returns the internal metadata (SEO, caching, assets) returned by
// the component.
//
// Returns templater_dto.InternalMetadata which contains the component's
// metadata.
func (v *TestView) Metadata() templater_dto.InternalMetadata {
	return v.metadata
}

// AST returns the template AST.
//
// Use this for advanced queries or custom assertions.
//
// Returns *ast_domain.TemplateAST which is the parsed template structure.
func (v *TestView) AST() *ast_domain.TemplateAST {
	return v.ast
}

// QueryAST searches the template AST for nodes matching the CSS selector.
// This is a direct way to assert against DOM structure without rendering
// HTML.
//
// Takes selector (string) which specifies the CSS selector to match nodes.
//
// Returns *ASTQueryResult which contains the matched nodes ready for
// assertions.
func (v *TestView) QueryAST(selector string) *ASTQueryResult {
	nodes, diagnostics := ast_domain.QueryAll(v.ast, selector, "test_view.go")

	if len(diagnostics) > 0 {
		v.tb.Helper()
		for _, diagnostic := range diagnostics {
			v.tb.Errorf("CSS selector error: %s", diagnostic.Message)
		}
		v.tb.FailNow()
	}

	return &ASTQueryResult{
		tb:    v.tb,
		nodes: nodes,
	}
}

// HTML returns the rendered HTML content of the test view.
//
// This triggers a full render if not already rendered. For most tests,
// prefer QueryAST which is faster and does not require rendering.
//
// Returns []byte which contains the rendered HTML output.
func (v *TestView) HTML() []byte {
	if !v.rendered {
		v.renderHTML()
	}
	return v.html
}

// HTMLString returns the rendered HTML as a string.
//
// Returns string which is the HTML content of the view.
func (v *TestView) HTMLString() string {
	return string(v.HTML())
}

// AssertTitle checks that the page title matches the expected value.
//
// Takes expected (string) which is the title to compare against.
func (v *TestView) AssertTitle(expected string) {
	v.tb.Helper()
	assert.Equal(v.tb, expected, v.metadata.Title, "Page title mismatch")
}

// AssertStatusCode checks that the HTTP status code matches the expected
// value.
//
// When a component does not explicitly set a status code, the metadata
// contains 0. Use AssertDefaultStatusCode to assert that no custom status was
// set, which the HTTP layer will interpret as 200 OK.
//
// Takes expected (int) which specifies the HTTP status code to compare
// against.
func (v *TestView) AssertStatusCode(expected int) {
	v.tb.Helper()
	actual := v.metadata.Status
	if actual == 0 && expected == httpStatusOK {
		return
	}
	assert.Equal(v.tb, expected, actual, "HTTP status code mismatch")
}

// AssertDefaultStatusCode checks that no custom status code was set.
// This means the HTTP layer will use the default 200 OK response.
func (v *TestView) AssertDefaultStatusCode() {
	v.tb.Helper()
	if v.metadata.Status != 0 {
		v.tb.Errorf("Expected default status code (0/200), but got: %d", v.metadata.Status)
	}
}

// AssertDescription checks that the meta description matches the expected
// value.
//
// Takes expected (string) which is the description to compare against.
func (v *TestView) AssertDescription(expected string) {
	v.tb.Helper()
	assert.Equal(v.tb, expected, v.metadata.Description, "Meta description mismatch")
}

// AssertHasMetaTag checks that a meta tag with the given name and content
// exists.
//
// Takes name (string) which specifies the meta tag name attribute.
// Takes content (string) which specifies the expected content value.
func (v *TestView) AssertHasMetaTag(name, content string) {
	v.tb.Helper()

	for _, tag := range v.metadata.MetaTags {
		if tag.Name == name && tag.Content == content {
			return
		}
	}

	v.tb.Errorf("Expected meta tag %s=%q not found", name, content)
}

// AssertHasOGTag checks that an Open Graph tag with the given property and
// content exists.
//
// Takes property (string) which specifies the OG tag property name to match.
// Takes content (string) which specifies the expected content value.
func (v *TestView) AssertHasOGTag(property, content string) {
	v.tb.Helper()

	for _, tag := range v.metadata.OGTags {
		if tag.Property == property && tag.Content == content {
			return
		}
	}

	v.tb.Errorf("Expected OG tag %s=%q not found", property, content)
}

// AssertClientRedirect checks that the component requested a client-side
// redirect to the given URL.
//
// Takes expectedURL (string) which specifies the URL to match against.
func (v *TestView) AssertClientRedirect(expectedURL string) {
	v.tb.Helper()
	assert.Equal(v.tb, expectedURL, v.metadata.ClientRedirect, "Client redirect URL mismatch")
}

// AssertServerRedirect checks that the component requested a server-side
// redirect to the given URL.
//
// Takes expectedURL (string) which is the URL the redirect should point to.
func (v *TestView) AssertServerRedirect(expectedURL string) {
	v.tb.Helper()
	assert.Equal(v.tb, expectedURL, v.metadata.ServerRedirect, "Server redirect URL mismatch")
}

// AssertLanguage checks that the page language matches the expected value.
//
// Takes expected (string) which is the language code to compare against.
func (v *TestView) AssertLanguage(expected string) {
	v.tb.Helper()
	assert.Equal(v.tb, expected, v.metadata.Language, "Page language mismatch")
}

// AssertCanonicalURL checks that the canonical URL matches the expected value.
//
// Takes expected (string) which is the canonical URL to compare against.
func (v *TestView) AssertCanonicalURL(expected string) {
	v.tb.Helper()
	assert.Equal(v.tb, expected, v.metadata.CanonicalURL, "Canonical URL mismatch")
}

// AssertJSScriptURLs checks that the page's client-side JavaScript URLs match
// the expected values.
//
// Takes expected ([]string) which is the anticipated JavaScript URLs to match.
func (v *TestView) AssertJSScriptURLs(expected []string) {
	v.tb.Helper()
	actual := make([]string, len(v.metadata.JSScriptMetas))
	for i, meta := range v.metadata.JSScriptMetas {
		actual[i] = meta.URL
	}
	assert.Equal(v.tb, expected, actual, "JS script URLs mismatch")
}

// AssertHasJSScript checks that the page has at least one JavaScript script
// URL.
func (v *TestView) AssertHasJSScript() {
	v.tb.Helper()
	if len(v.metadata.JSScriptMetas) == 0 {
		v.tb.Error("Expected page to have at least one JS script URL, but it was empty")
	}
}

// AssertNoJSScript checks that the page does not have any client-side
// JavaScript scripts.
func (v *TestView) AssertNoJSScript() {
	v.tb.Helper()
	if len(v.metadata.JSScriptMetas) > 0 {
		urls := make([]string, len(v.metadata.JSScriptMetas))
		for i, meta := range v.metadata.JSScriptMetas {
			urls[i] = meta.URL
		}
		v.tb.Errorf("Expected page to have no JS scripts, but found: %v", urls)
	}
}

// AssertJSScriptURLContains checks that at least one JavaScript script URL
// contains the given substring.
//
// Takes substring (string) which is the text to look for in a script URL.
func (v *TestView) AssertJSScriptURLContains(substring string) {
	v.tb.Helper()
	if len(v.metadata.JSScriptMetas) == 0 {
		v.tb.Error("JS script URLs are empty, cannot check for substring")
		return
	}
	for _, meta := range v.metadata.JSScriptMetas {
		if strings.Contains(meta.URL, substring) {
			return
		}
	}
	urls := make([]string, len(v.metadata.JSScriptMetas))
	for i, meta := range v.metadata.JSScriptMetas {
		urls[i] = meta.URL
	}
	v.tb.Errorf("Expected at least one JS script URL to contain %q, but got %v", substring, urls)
}

// MatchSnapshot compares the rendered HTML against a golden file.
//
// If the PIKO_UPDATE_SNAPSHOTS or UPDATE_SNAPSHOTS environment variable is
// set to "1", it updates the golden file instead of comparing. Snapshot files
// are stored in __snapshots__/<test-directory>/<name>.golden.html.
//
// Takes name (string) which identifies the snapshot file without extension.
func (v *TestView) MatchSnapshot(name string) {
	v.tb.Helper()

	html := v.HTML()

	testName := v.tb.Name()
	snapshotDir := filepath.Join("__snapshots__", filepath.Dir(testName))
	snapshotFile := filepath.Join(snapshotDir, name+".golden.html")

	updateSnapshots := os.Getenv("PIKO_UPDATE_SNAPSHOTS") == "1" ||
		os.Getenv("UPDATE_SNAPSHOTS") == "1"

	if updateSnapshots {
		err := os.MkdirAll(snapshotDir, snapshotDirPermissions)
		require.NoError(v.tb, err, "Failed to create snapshot directory")

		err = os.WriteFile(snapshotFile, html, snapshotFilePermissions)
		require.NoError(v.tb, err, "Failed to write snapshot file")

		v.tb.Logf("Updated snapshot: %s", snapshotFile)
		return
	}

	expected, err := os.ReadFile(snapshotFile)
	if err != nil {
		v.tb.Fatalf("Snapshot file not found: %s\nRun with PIKO_UPDATE_SNAPSHOTS=1 to create it", snapshotFile)
	}

	assert.Equal(v.tb, string(expected), string(html),
		"Snapshot mismatch for %s\nRun with PIKO_UPDATE_SNAPSHOTS=1 to update", name)
}

// WriteTo writes the rendered HTML to the given writer. Use it to debug
// or save test outputs.
//
// Takes w (io.Writer) which receives the rendered HTML output.
//
// Returns int64 which is the number of bytes written.
// Returns error when the write operation fails.
func (v *TestView) WriteTo(w io.Writer) (int64, error) {
	html := v.HTML()
	n, err := w.Write(html)
	return int64(n), err
}

// renderHTML builds the HTML output using the RenderService.
func (v *TestView) renderHTML() {
	v.tb.Helper()

	if v.renderer == nil {
		v.tb.Fatal("Cannot render HTML: RenderService not available")
	}

	var buffer bytes.Buffer
	websiteConfig := &config.WebsiteConfig{}

	err := v.renderer.RenderAST(context.Background(), &buffer, nil, v.request, render_domain.RenderASTOptions{
		PageID:     v.pageID,
		Template:   v.ast,
		Metadata:   &v.metadata,
		IsFragment: false,
		Styling:    "",
		SiteConfig: websiteConfig,
	})

	if err != nil {
		v.tb.Fatalf("Failed to render HTML: %v", err)
	}

	v.html = buffer.Bytes()
	v.rendered = true
}

// newTestView creates a new TestView from a component render result.
//
// Takes tb (testing.TB) which provides the testing interface for failures.
// Takes state (any) which holds the component state data.
// Takes metadata (*templater_dto.InternalMetadata) which holds template data.
// Takes ast (*ast_domain.TemplateAST) which represents the parsed template.
// Takes renderer (render_domain.RenderService) which handles template rendering.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes pageID (string) which identifies the page being tested.
//
// Returns *TestView which is the configured test view ready for assertions.
func newTestView(
	tb testing.TB,
	state any,
	metadata *templater_dto.InternalMetadata,
	ast *ast_domain.TemplateAST,
	renderer render_domain.RenderService,
	request *http.Request,
	pageID string,
) *TestView {
	return &TestView{
		tb:       tb,
		state:    state,
		renderer: renderer,
		ast:      ast,
		request:  request,
		metadata: *metadata,
		pageID:   pageID,
		html:     nil,
		rendered: false,
	}
}
