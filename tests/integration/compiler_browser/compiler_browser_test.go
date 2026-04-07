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

//go:build integration

package compiler_browser_test

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"piko.sh/piko/internal/compiler/compiler_adapters"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/cssinliner"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/esbuild/compat"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/testutil/leakcheck"
	browserpkg "piko.sh/piko/wdk/browser"
	"piko.sh/piko/wdk/safedisk"
)

const assertionTimeout = 10 * time.Second

var (
	update = flag.Bool("update", false, "update golden files")
	pool   *browserpkg.BrowserPool
)

type TestSpec struct {
	Expected  any    `json:"expected,omitempty"`
	Detail    any    `json:"detail,omitempty"`
	Action    string `json:"action"`
	Selector  string `json:"selector"`
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	EventName string `json:"eventName,omitempty"`
}

func (s TestSpec) ExpectedString() string {
	if s.Expected == nil {
		return ""
	}
	if str, ok := s.Expected.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", s.Expected)
}

func TestMain(m *testing.M) {
	flag.Parse()

	opts := browserpkg.BrowserOptions{
		Headless: true,
	}

	done := make(chan error, 1)
	go func() {
		var err error
		poolSize := browserpkg.DefaultPoolSize()
		pool, err = browserpkg.NewBrowserPool(opts, poolSize, browserpkg.BrowserPoolConfig{
			MaxConcurrentPages: poolSize,
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("could not launch browser pool: %v\n", err)
			os.Exit(1)
		}
	case <-time.After(60 * time.Second):
		_, _ = fmt.Fprintf(os.Stderr, "timeout: browser pool failed to start within 60s\n")
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stderr, "browser pool: %d instances\n", pool.Size())

	code := m.Run()

	pool.Close()

	if code == 0 {
		if err := leakcheck.FindLeaks(

			goleak.IgnoreAnyFunction("github.com/chromedp/chromedp.NewContext.func1"),
		); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}

func TestCompiler_Functional(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compiler functional browser tests in short mode")
	}

	require.NotNil(t, pool, "browser pool not initialised. Ensure TestMain ran correctly.")

	testdataRoot := "testdata"
	testDirs, err := os.ReadDir(testdataRoot)
	require.NoError(t, err)

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}
		testName := entry.Name()
		testDir := filepath.Join(testdataRoot, testName)

		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			pkcPath := filepath.Join(testDir, "component.pkc")
			pkcContent, err := os.ReadFile(pkcPath)
			require.NoError(t, err)

			compilerOpts := buildCompilerOpts(t, testDir)
			compilerService := compiler_domain.NewCompilerOrchestrator(nil, nil, compilerOpts...)
			artefact, err := compilerService.CompileSFCBytes(context.Background(), pkcPath, pkcContent)
			require.NoError(t, err)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/harness.html":
					w.Header().Set("Content-Type", "text/html")
					_, _ = fmt.Fprint(w, createTestHarnessHTML(artefact))
				case "/_piko/dist/ppframework.core.es.js", "/_piko/dist/ppframework.components.es.js":
					w.Header().Set("Content-Type", "application/javascript")
					filename := strings.TrimPrefix(r.URL.Path, "/_piko/dist/")
					framework, err := daemon_frontend.EmbeddedFrontendTemplates.ReadFile("built/" + filename)
					if err != nil {
						t.Logf("ERROR: Failed to read embedded framework JS %s: %v", filename, err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, _ = fmt.Fprint(w, string(framework))
				case "/_piko/assets/pk-js/pk/actions.gen.js":
					w.Header().Set("Content-Type", "application/javascript")
					actionsJS, err := os.ReadFile(filepath.Join(testdataRoot, "actions.gen.js"))
					if err != nil {
						t.Logf("ERROR: Failed to read mock actions.gen.js: %v", err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, _ = fmt.Fprint(w, string(actionsJS))
				case "/" + artefact.BaseJSPath:
					w.Header().Set("Content-Type", "application/javascript")
					_, _ = fmt.Fprint(w, artefact.Files[artefact.BaseJSPath])
				default:
					requestedPath := strings.TrimPrefix(r.URL.Path, "/")
					testFilePath := filepath.Join(testDir, requestedPath)
					if content, err := os.ReadFile(testFilePath); err == nil {
						if strings.HasSuffix(requestedPath, ".js") {
							w.Header().Set("Content-Type", "application/javascript")
						} else if strings.HasSuffix(requestedPath, ".css") {
							w.Header().Set("Content-Type", "text/css")
						}
						_, _ = fmt.Fprint(w, string(content))
					} else {
						http.NotFound(w, r)
					}
				}
			}))
			defer server.Close()

			poolCtx, poolCancel := context.WithTimeoutCause(context.Background(), 90*time.Second, fmt.Errorf("test: integration test exceeded 90s timeout"))
			defer poolCancel()

			incognitoPage, err := pool.NewIncognitoPage(poolCtx)
			require.NoError(t, err)
			defer func() { _ = incognitoPage.Close() }()

			timer := time.AfterFunc(60*time.Second, func() {
				t.Errorf("test %s exceeded 60s - force cancelling page context", t.Name())
				incognitoPage.Cancel()
			})
			defer timer.Stop()

			pageHelper := browserpkg.NewPageHelper(incognitoPage.Ctx)
			defer pageHelper.Close()
			ctx := pageHelper.Ctx()

			srcSandbox, err := safedisk.NewNoOpSandbox(testDir, safedisk.ModeReadOnly)
			require.NoError(t, err)

			actionCtx := &browserpkg.ActionContext{
				Ctx:        ctx,
				SrcSandbox: srcSandbox,
				PageHelper: pageHelper,
				ServerURL:  server.URL,
			}

			err = browserpkg.SetViewport(actionCtx, 1280, 720)
			require.NoError(t, err)

			err = pageHelper.Navigate(server.URL + "/harness.html")
			require.NoError(t, err)

			err = browserpkg.WaitForSelector(actionCtx, artefact.TagName, 10*time.Second)
			require.NoError(t, err)
			t.Logf("Component <%s> is visible in the DOM.", artefact.TagName)

			time.Sleep(100 * time.Millisecond)

			var hasShadowRoot bool
			err = chromedp.Run(ctx, chromedp.Evaluate(
				fmt.Sprintf(`document.querySelector('%s').shadowRoot !== null`, artefact.TagName),
				&hasShadowRoot,
			))
			require.NoError(t, err)

			goldenRenderedPath := filepath.Join(testDir, "golden.rendered.html")
			if _, statErr := os.Stat(goldenRenderedPath); statErr == nil || *update {
				if hasShadowRoot {
					var actualRenderedHTML string
					err = chromedp.Run(ctx, chromedp.Evaluate(
						fmt.Sprintf(`document.querySelector('%s').shadowRoot.innerHTML`, artefact.TagName),
						&actualRenderedHTML,
					))
					require.NoError(t, err)

					if *update {
						t.Logf("Updating golden rendered HTML file for %s", testName)
						require.NoError(t, os.WriteFile(goldenRenderedPath, []byte(actualRenderedHTML), 0644))
					} else {
						expectedRenderedHTML, err := os.ReadFile(goldenRenderedPath)
						require.NoError(t, err, "failed to read golden.rendered.html")
						assert.Equal(t, string(expectedRenderedHTML), actualRenderedHTML, "Initial rendered HTML does not match golden.rendered.html")
					}
				} else {
					t.Logf("Component <%s> does not use shadow DOM, skipping rendered HTML validation", artefact.TagName)
				}
			}

			specPath := filepath.Join(testDir, "test_spec.json")
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				return
			}

			specContent, err := os.ReadFile(specPath)
			require.NoError(t, err)
			var specs []TestSpec
			require.NoError(t, json.Unmarshal(specContent, &specs))

			actionsRequiringShadowRoot := map[string]bool{
				"click": true, "checkText": true, "checkValue": true, "setValue": true,
				"checkComputedStyle": true, "checkAttribute": true, "checkFocus": true,
			}
			needsShadowRoot := false
			for _, spec := range specs {
				if actionsRequiringShadowRoot[spec.Action] {
					needsShadowRoot = true
					break
				}
			}
			if !hasShadowRoot && needsShadowRoot {
				t.Skipf("Component <%s> has test specs requiring shadow root but no shadow root available", artefact.TagName)
				return
			}

			for i, step := range specs {
				stepMessage := fmt.Sprintf("step %d: %s on '%s'", i+1, step.Action, step.Selector)

				buildSelector := func(selector string) string {
					return selector
				}

				switch step.Action {
				case "click":
					selector := buildSelector(step.Selector)
					err := browserpkg.Click(actionCtx, selector)
					require.NoError(t, err, stepMessage)

					time.Sleep(50 * time.Millisecond)
					t.Logf("Step %d: Clicked '%s'", i+1, step.Selector)

				case "fill":
					selector := buildSelector(step.Selector)
					err := browserpkg.Fill(actionCtx, selector, step.Value)
					require.NoError(t, err, stepMessage)
					t.Logf("Step %d: Filled '%s' with '%s'", i+1, step.Selector, step.Value)

				case "checkText":
					selector := buildSelector(step.Selector)
					require.Eventually(t, func() bool {
						text, err := browserpkg.GetElementText(ctx, selector)
						if err != nil {
							return false
						}
						return strings.TrimSpace(text) == step.ExpectedString()
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element '%s' to have text '%s'", step.Selector, step.ExpectedString())

					text, _ := browserpkg.GetElementText(ctx, selector)
					assert.Equal(t, step.ExpectedString(), strings.TrimSpace(text), stepMessage)
					t.Logf("Step %d: Verified text of '%s' is '%s'", i+1, step.Selector, step.Expected)

				case "checkElementCount":
					expectedCount, err := strconv.Atoi(step.ExpectedString())
					require.NoError(t, err, "step %d: 'expected' value for checkElementCount must be an integer", i+1)

					selector := buildSelector(step.Selector)
					require.Eventually(t, func() bool {
						nodes, err := browserpkg.FindElements(ctx, selector)
						if err != nil {
							return expectedCount == 0
						}
						return len(nodes) == expectedCount
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element count of '%s' to be %d", step.Selector, expectedCount)

					nodes, _ := browserpkg.FindElements(ctx, selector)
					assert.Len(t, nodes, expectedCount, stepMessage)
					t.Logf("Step %d: Verified element count of '%s' is %d", i+1, step.Selector, expectedCount)

				case "checkValue":
					selector := buildSelector(step.Selector)
					require.Eventually(t, func() bool {
						value, err := browserpkg.GetElementValue(ctx, selector)
						if err != nil {
							return false
						}
						return value == step.ExpectedString()
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element '%s' to have value '%s'", step.Selector, step.ExpectedString())

					value, _ := browserpkg.GetElementValue(ctx, selector)
					assert.Equal(t, step.ExpectedString(), value, stepMessage)
					t.Logf("Step %d: Verified value of '%s' is '%s'", i+1, step.Selector, step.Expected)

				case "setValue":
					selector := buildSelector(step.Selector)
					err := browserpkg.Fill(actionCtx, selector, step.Value)
					require.NoError(t, err, stepMessage)
					t.Logf("Step %d: Set value of '%s' to '%s'", i+1, step.Selector, step.Value)

				case "checkComputedStyle":
					selector := buildSelector(step.Selector)
					jsFunc := fmt.Sprintf(`function() { return window.getComputedStyle(this).getPropertyValue(%q); }`, step.Name)
					require.Eventually(t, func() bool {
						result, err := browserpkg.EvalOnElement(ctx, selector, jsFunc)
						if err != nil {
							return false
						}
						return fmt.Sprintf("%v", result) == step.ExpectedString()
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element '%s' computed style '%s' to be '%s'", step.Selector, step.Name, step.ExpectedString())

					result, _ := browserpkg.EvalOnElement(ctx, selector, jsFunc)
					assert.Equal(t, step.ExpectedString(), fmt.Sprintf("%v", result), stepMessage)
					t.Logf("Step %d: Verified computed style '%s' of '%s' is '%s'", i+1, step.Name, step.Selector, step.Expected)

				case "checkComputedStyleSlotted":

					jsFuncSlotted := fmt.Sprintf(`function() { return window.getComputedStyle(this).getPropertyValue(%q); }`, step.Name)
					require.Eventually(t, func() bool {
						result, err := browserpkg.EvalOnElement(ctx, step.Selector, jsFuncSlotted)
						if err != nil {
							return false
						}
						return fmt.Sprintf("%v", result) == step.ExpectedString()
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for slotted element '%s' computed style '%s' to be '%s'", step.Selector, step.Name, step.ExpectedString())

					result, _ := browserpkg.EvalOnElement(ctx, step.Selector, jsFuncSlotted)
					assert.Equal(t, step.ExpectedString(), fmt.Sprintf("%v", result), stepMessage)
					t.Logf("Step %d: Verified computed style '%s' of slotted '%s' is '%s'", i+1, step.Name, step.Selector, step.Expected)

				case "eval":
					var evalErr error
					if step.Selector == "window" || step.Selector == "document" {
						evalCode := strings.ReplaceAll(step.Value, "this", step.Selector)
						evalErr = chromedp.Run(ctx, chromedp.Evaluate(evalCode, nil))
					} else {
						selector := buildSelector(step.Selector)

						evalCode := step.Value
						trimmed := strings.TrimSpace(evalCode)
						if cut, found := strings.CutPrefix(trimmed, "() =>"); found {
							evalCode = cut
						}
						evalErr = browserpkg.Eval(actionCtx, selector, evalCode)
					}
					if evalErr != nil && !strings.Contains(evalErr.Error(), "apply") {
						require.NoError(t, evalErr, stepMessage)
					}
					t.Logf("Step %d: Evaluated script on '%s'", i+1, step.Selector)

				case "checkConsoleMessage":
					require.Eventually(t, func() bool {
						logs := pageHelper.ConsoleLogs()
						for _, log := range logs {
							if strings.Contains(log, step.ExpectedString()) {
								return true
							}
						}
						return false
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for console message containing '%s'", step.Expected)
					t.Logf("Step %d: Verified console message '%s' was logged", i+1, step.Expected)

				case "checkNoConsoleMessage":
					time.Sleep(100 * time.Millisecond)
					logs := pageHelper.ConsoleLogs()
					var foundMessage bool
					for _, log := range logs {
						if strings.Contains(log, step.ExpectedString()) {
							foundMessage = true
							break
						}
					}
					assert.False(t, foundMessage, stepMessage+": unexpected message found")
					t.Logf("Step %d: Verified console message '%s' was NOT logged", i+1, step.Expected)

				case "clearConsole":
					pageHelper.ClearConsoleLogs()
					_ = chromedp.Run(ctx, chromedp.Evaluate(`console.clear()`, nil))
					t.Logf("Step %d: Cleared test console log buffer", i+1)

				case "checkTextNot":
					selector := buildSelector(step.Selector)
					require.Eventually(t, func() bool {
						text, err := browserpkg.GetElementText(ctx, selector)
						if err != nil {
							return true
						}
						return strings.TrimSpace(text) != step.ExpectedString()
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element '%s' text to NOT be '%s'", step.Selector, step.ExpectedString())

					text, _ := browserpkg.GetElementText(ctx, selector)
					assert.NotEqual(t, step.ExpectedString(), strings.TrimSpace(text), stepMessage)
					t.Logf("Step %d: Verified text of '%s' is NOT '%s'", i+1, step.Selector, step.Expected)

				case "checkHTML":
					selector := buildSelector(step.Selector)
					require.Eventually(t, func() bool {
						html, err := browserpkg.GetElementHTML(ctx, selector)
						if err != nil {
							return false
						}
						return html == step.ExpectedString()
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element '%s' HTML to match", step.Selector)

					html, _ := browserpkg.GetElementHTML(ctx, selector)
					assert.Equal(t, step.ExpectedString(), html, stepMessage)
					t.Logf("Step %d: Verified HTML of '%s'", i+1, step.Selector)

				case "checkNotFocused":
					selector := buildSelector(step.Selector)
					result, err := browserpkg.EvalOnElement(ctx, selector, `function() { return this.getRootNode().activeElement === this; }`)
					require.NoError(t, err, stepMessage)
					isFocused, ok := result.(bool)
					require.True(t, ok, stepMessage+": expected bool result")
					assert.False(t, isFocused, stepMessage+": element was unexpectedly focused")
					t.Logf("Step %d: Verified element '%s' is NOT focused", i+1, step.Selector)

				case "checkFocused":
					selector := buildSelector(step.Selector)
					require.Eventually(t, func() bool {
						result, err := browserpkg.EvalOnElement(ctx, selector, `function() { return this.getRootNode().activeElement === this; }`)
						if err != nil {
							return false
						}
						isFocused, ok := result.(bool)
						return ok && isFocused
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element '%s' to be focused", step.Selector)
					t.Logf("Step %d: Verified element '%s' IS focused", i+1, step.Selector)

				case "listenForEvent":
					selector := buildSelector(step.Selector)
					err := browserpkg.ListenForEvent(ctx, selector, step.EventName)
					require.NoError(t, err, stepMessage)
					t.Logf("Step %d: Started listening for '%s' event on '%s'", i+1, step.EventName, step.Selector)

				case "checkEventReceived":
					require.Eventually(t, func() bool {
						detail, err := browserpkg.GetEventDetail(ctx, step.EventName)
						return err == nil && detail != nil
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for event '%s'", step.EventName)

					receivedDetail, err := browserpkg.GetEventDetail(ctx, step.EventName)
					require.NoError(t, err, stepMessage)

					expectedDetail, ok := step.Detail.(map[string]any)
					require.True(t, ok, "step.Detail must be a JSON object for checkEventReceived action")

					assert.Equal(t, expectedDetail, receivedDetail, stepMessage)
					t.Logf("Step %d: Verified event '%s' was received with correct detail", i+1, step.EventName)

				case "checkAttribute":
					selector := buildSelector(step.Selector)
					var expectedValue any
					if step.ExpectedString() == "null" {
						expectedValue = nil
					} else {
						expectedValue = step.ExpectedString()
					}

					require.Eventually(t, func() bool {
						attr, err := browserpkg.GetElementAttribute(ctx, selector, step.Name)
						if err != nil {
							return expectedValue == nil
						}
						if expectedValue == nil {
							return attr == nil
						}
						return attr != nil && *attr == expectedValue
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for element '%s' attribute '%s' to be '%v'", step.Selector, step.Name, expectedValue)

					attr, _ := browserpkg.GetElementAttribute(ctx, selector, step.Name)
					if expectedValue == nil {
						assert.Nil(t, attr, stepMessage)
					} else {
						require.NotNil(t, attr, stepMessage+": attribute was nil")
						assert.Equal(t, expectedValue, *attr, stepMessage)
					}
					t.Logf("Step %d: Verified attribute '%s' of '%s' is '%v'", i+1, step.Name, step.Selector, expectedValue)

				case "wait":
					timeout, _ := strconv.Atoi(step.Value)
					time.Sleep(time.Duration(timeout) * time.Millisecond)
					t.Logf("Step %d: Waited for %dms", i+1, timeout)

				case "checkFormData":
					expectedData, ok := step.Expected.(map[string]any)
					require.True(t, ok, stepMessage+": expected must be a JSON object for checkFormData")

					require.Eventually(t, func() bool {
						actual, err := browserpkg.GetFormData(ctx, step.Selector)
						if err != nil || len(actual) != len(expectedData) {
							return false
						}
						for k, v := range expectedData {
							if actual[k] != v {
								return false
							}
						}
						return true
					}, assertionTimeout, 100*time.Millisecond, "timed out waiting for form data to match on '%s'", step.Selector)

					actualData, _ := browserpkg.GetFormData(ctx, step.Selector)
					assert.Equal(t, expectedData, actualData, stepMessage)
					t.Logf("Step %d: Verified form data matches expected", i+1)

				default:
					t.Fatalf("Unknown test action: %s", step.Action)
				}
			}
		})
	}
}

type testFSReader struct{}

func (r *testFSReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func hasCSSFiles(testDir string) bool {
	entries, err := os.ReadDir(testDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".css") {
			return true
		}
	}
	return false
}

func buildCompilerOpts(t *testing.T, testDir string) []compiler_domain.OrchestratorOption {
	t.Helper()
	if !hasCSSFiles(testDir) {
		return nil
	}
	mockResolver := &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(_ context.Context, importPath string, containingDir string) (string, error) {
			if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
				return filepath.Join(containingDir, filepath.FromSlash(importPath)), nil
			}
			return "", fmt.Errorf("unsupported CSS import path in test: %s", importPath)
		},
	}
	processor := cssinliner.NewProcessor(cssinliner.ProcessorConfig{
		Resolver: mockResolver,
		Loader:   esbuildconfig.LoaderLocalCSS,
		Options: &esbuildconfig.Options{
			MinifyWhitespace:       true,
			MinifySyntax:           true,
			UnsupportedCSSFeatures: compat.Nesting,
		},
	})
	preProcessor := compiler_adapters.NewCSSPreProcessor(processor, &testFSReader{}, "", "")
	return []compiler_domain.OrchestratorOption{
		compiler_domain.WithOrchestratorCSSPreProcessor(preProcessor),
	}
}

func createTestHarnessHTML(artefact *compiler_dto.CompiledArtefact) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Compiler Functional Test</title>
			<script type="module" src="/%s"></script>
		</head>
		<body>
			<div id="test-harness">
				<%s></%s>
			</div>
			<!-- Spacer to make page scrollable for scroll event tests -->
			<div style="height: 2000px;"></div>
		</body>
		</html>
	`, artefact.BaseJSPath, artefact.TagName, artefact.TagName)
}
