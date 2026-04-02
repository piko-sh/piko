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

package browser_provider_chromedp

import (
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestCheckText(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("exact match", func(t *testing.T) {
			err := CheckText(ctx, "#target", "Content Text")
			if err != nil {
				t.Errorf("CheckText() error = %v", err)
			}
		})

		t.Run("mismatch returns error", func(t *testing.T) {
			err := CheckText(ctx, "#target", "Wrong Text")
			if err == nil {
				t.Error("expected error for text mismatch")
			}
		})
	})
}

func TestCheckTextContains(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("substring match", func(t *testing.T) {
			err := CheckTextContains(ctx, "#target", "Content")
			if err != nil {
				t.Errorf("CheckTextContains() error = %v", err)
			}
		})

		t.Run("no match returns error", func(t *testing.T) {
			err := CheckTextContains(ctx, "#target", "XYZ")
			if err == nil {
				t.Error("expected error for substring not found")
			}
		})
	})
}

func TestCheckValue(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("correct value", func(t *testing.T) {
			err := CheckValue(ctx, "#enabled-input", "enabled")
			if err != nil {
				t.Errorf("CheckValue() error = %v", err)
			}
		})

		t.Run("wrong value returns error", func(t *testing.T) {
			err := CheckValue(ctx, "#enabled-input", "wrong")
			if err == nil {
				t.Error("expected error for value mismatch")
			}
		})
	})
}

func TestCheckAttribute(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("exact match", func(t *testing.T) {
			err := CheckAttribute(ctx, "#target", "data-custom", "custom-value")
			if err != nil {
				t.Errorf("CheckAttribute() error = %v", err)
			}
		})

		t.Run("wrong value returns error", func(t *testing.T) {
			err := CheckAttribute(ctx, "#target", "data-custom", "wrong")
			if err == nil {
				t.Error("expected error for attribute mismatch")
			}
		})
	})
}

func TestCheckAttributeContains(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("substring match", func(t *testing.T) {
			err := CheckAttributeContains(ctx, "#target", "data-custom", "custom")
			if err != nil {
				t.Errorf("CheckAttributeContains() error = %v", err)
			}
		})

		t.Run("no match returns error", func(t *testing.T) {
			err := CheckAttributeContains(ctx, "#target", "data-custom", "xyz")
			if err == nil {
				t.Error("expected error for attribute substring not found")
			}
		})
	})
}

func TestCheckClass(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target" class="class1 class2 class3">Test</div>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("has class", func(t *testing.T) {
			err := CheckClass(ctx, "#target", "class2")
			if err != nil {
				t.Errorf("CheckClass() error = %v", err)
			}
		})

		t.Run("missing class returns error", func(t *testing.T) {
			err := CheckClass(ctx, "#target", "nonexistent")
			if err == nil {
				t.Error("expected error for missing class")
			}
		})
	})
}

func TestCheckStyle(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target" style="color: rgb(255, 0, 0); font-size: 16px;">Styled</div>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("style property match", func(t *testing.T) {
			err := CheckStyle(ctx, "#target", "color", "rgb(255, 0, 0)")
			if err != nil {
				t.Errorf("CheckStyle() error = %v", err)
			}
		})

		t.Run("wrong style returns error", func(t *testing.T) {
			err := CheckStyle(ctx, "#target", "color", "rgb(0, 0, 255)")
			if err == nil {
				t.Error("expected error for style mismatch")
			}
		})
	})
}

func TestCheckFocused(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<input type="text" id="input1" />
<input type="text" id="input2" />
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#input1", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing input: %v", err)
		}

		t.Run("focused element", func(t *testing.T) {
			err := CheckFocused(ctx, "#input1")
			if err != nil {
				t.Errorf("CheckFocused() error = %v", err)
			}
		})

		t.Run("non-focused returns error", func(t *testing.T) {
			err := CheckFocused(ctx, "#input2")
			if err == nil {
				t.Error("expected error for non-focused element")
			}
		})
	})
}

func TestCheckNotFocused(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<input type="text" id="input1" />
<input type="text" id="input2" />
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#input1", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing input: %v", err)
		}

		t.Run("non-focused element", func(t *testing.T) {
			err := CheckNotFocused(ctx, "#input2")
			if err != nil {
				t.Errorf("CheckNotFocused() error = %v", err)
			}
		})

		t.Run("focused returns error", func(t *testing.T) {
			err := CheckNotFocused(ctx, "#input1")
			if err == nil {
				t.Error("expected error for focused element")
			}
		})
	})
}

func TestCheckElementCount(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLMultipleElements)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("correct count", func(t *testing.T) {
			err := CheckElementCount(ctx, ".item", 5)
			if err != nil {
				t.Errorf("CheckElementCount() error = %v", err)
			}
		})

		t.Run("wrong count returns error", func(t *testing.T) {
			err := CheckElementCount(ctx, ".item", 10)
			if err == nil {
				t.Error("expected error for wrong count")
			}
		})

		t.Run("zero count for nonexistent", func(t *testing.T) {
			err := CheckElementCount(ctx, ".nonexistent", 0)
			if err != nil {
				t.Errorf("CheckElementCount() for 0 error = %v", err)
			}
		})
	})
}

func TestCheckHTML(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<span id="simple">Simple Content</span>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("exact match", func(t *testing.T) {

			err := CheckHTML(ctx, "#simple", `<span id="simple">Simple Content</span>`)
			if err != nil {
				t.Errorf("CheckHTML() error = %v", err)
			}
		})

		t.Run("no match returns error", func(t *testing.T) {
			err := CheckHTML(ctx, "#simple", "Different Content")
			if err == nil {
				t.Error("expected error for HTML mismatch")
			}
		})
	})
}

func TestCheckFormData(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<form id="testform">
<input type="text" name="username" value="testuser" />
<input type="email" name="email" value="test@example.com" />
<select name="role">
<option value="admin" selected>Admin</option>
<option value="user">User</option>
</select>
</form>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("all fields match", func(t *testing.T) {
			expected := map[string]any{
				"username": "testuser",
				"email":    "test@example.com",
				"role":     "admin",
			}
			err := CheckFormData(ctx, "#testform", expected)
			if err != nil {
				t.Errorf("CheckFormData() error = %v", err)
			}
		})

		t.Run("field mismatch returns error", func(t *testing.T) {
			expected := map[string]any{
				"username": "wronguser",
			}
			err := CheckFormData(ctx, "#testform", expected)
			if err == nil {
				t.Error("expected error for form data mismatch")
			}
		})
	})
}

func TestCheckExists(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("element exists", func(t *testing.T) {
			err := CheckExists(ctx, "#target")
			if err != nil {
				t.Errorf("CheckExists() error = %v", err)
			}
		})

		t.Run("nonexistent returns error", func(t *testing.T) {
			err := CheckExists(ctx, "#nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestCheckNotExists(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("element does not exist", func(t *testing.T) {
			err := CheckNotExists(ctx, "#nonexistent")
			if err != nil {
				t.Errorf("CheckNotExists() error = %v", err)
			}
		})

		t.Run("exists returns error", func(t *testing.T) {
			err := CheckNotExists(ctx, "#target")
			if err == nil {
				t.Error("expected error for existing element")
			}
		})
	})
}

func TestCheckVisible(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("visible element", func(t *testing.T) {
			err := CheckVisible(ctx, "#visible")
			if err != nil {
				t.Errorf("CheckVisible() error = %v", err)
			}
		})

		t.Run("hidden returns error", func(t *testing.T) {
			err := CheckVisible(ctx, "#hidden-display")
			if err == nil {
				t.Error("expected error for hidden element")
			}
		})
	})
}

func TestCheckHidden(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("hidden by display", func(t *testing.T) {
			err := CheckHidden(ctx, "#hidden-display")
			if err != nil {
				t.Errorf("CheckHidden() error = %v", err)
			}
		})

		t.Run("hidden by visibility", func(t *testing.T) {
			err := CheckHidden(ctx, "#hidden-visibility")
			if err != nil {
				t.Errorf("CheckHidden(visibility) error = %v", err)
			}
		})

		t.Run("visible returns error", func(t *testing.T) {
			err := CheckHidden(ctx, "#visible")
			if err == nil {
				t.Error("expected error for visible element")
			}
		})
	})
}

func TestCheckEnabled(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("enabled input", func(t *testing.T) {
			err := CheckEnabled(ctx, "#enabled-input")
			if err != nil {
				t.Errorf("CheckEnabled() error = %v", err)
			}
		})

		t.Run("disabled returns error", func(t *testing.T) {
			err := CheckEnabled(ctx, "#disabled-input")
			if err == nil {
				t.Error("expected error for disabled element")
			}
		})
	})
}

func TestCheckDisabled(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("disabled input", func(t *testing.T) {
			err := CheckDisabled(ctx, "#disabled-input")
			if err != nil {
				t.Errorf("CheckDisabled() error = %v", err)
			}
		})

		t.Run("enabled returns error", func(t *testing.T) {
			err := CheckDisabled(ctx, "#enabled-input")
			if err == nil {
				t.Error("expected error for enabled element")
			}
		})
	})
}

func TestCheckChecked(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCheckbox)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("checked checkbox", func(t *testing.T) {
			err := CheckChecked(ctx, "#checkbox2")
			if err != nil {
				t.Errorf("CheckChecked() error = %v", err)
			}
		})

		t.Run("unchecked returns error", func(t *testing.T) {
			err := CheckChecked(ctx, "#checkbox1")
			if err == nil {
				t.Error("expected error for unchecked checkbox")
			}
		})
	})
}

func TestCheckUnchecked(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCheckbox)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("unchecked checkbox", func(t *testing.T) {
			err := CheckUnchecked(ctx, "#checkbox1")
			if err != nil {
				t.Errorf("CheckUnchecked() error = %v", err)
			}
		})

		t.Run("checked returns error", func(t *testing.T) {
			err := CheckUnchecked(ctx, "#checkbox2")
			if err == nil {
				t.Error("expected error for checked checkbox")
			}
		})
	})
}

func TestCheckConsoleMessage(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.log('test log message')`, nil),
		)
		if err != nil {
			t.Fatalf("emitting console.log: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		t.Run("log message found", func(t *testing.T) {
			err := CheckConsoleMessage(ctx, "log", "test log")
			if err != nil {
				t.Errorf("CheckConsoleMessage() error = %v", err)
			}
		})

		t.Run("not found returns error", func(t *testing.T) {
			err := CheckConsoleMessage(ctx, "log", "xyz not found")
			if err == nil {
				t.Error("expected error for missing console message")
			}
		})
	})
}

func TestCheckNoConsoleMessage(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.log('specific message')`, nil),
		)
		if err != nil {
			t.Fatalf("emitting console.log: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		t.Run("message not present", func(t *testing.T) {
			err := CheckNoConsoleMessage(ctx, "log", "xyz not present")
			if err != nil {
				t.Errorf("CheckNoConsoleMessage() error = %v", err)
			}
		})

		t.Run("message present returns error", func(t *testing.T) {
			err := CheckNoConsoleMessage(ctx, "log", "specific message")
			if err == nil {
				t.Error("expected error when message is present")
			}
		})
	})
}

func TestCheckNoConsoleErrors(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	t.Run("no errors initially", func(t *testing.T) {
		withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
			ctx := newActionContext(page)
			err := CheckNoConsoleErrors(ctx)
			if err != nil {
				t.Errorf("CheckNoConsoleErrors() error = %v", err)
			}
		})
	})

	t.Run("errors present returns error", func(t *testing.T) {
		withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
			ctx := newActionContext(page)

			err := chromedp.Run(page.Ctx(),
				chromedp.Evaluate(`console.error('test error')`, nil),
			)
			if err != nil {
				t.Fatalf("emitting console.error: %v", err)
			}

			time.Sleep(200 * time.Millisecond)

			err = CheckNoConsoleErrors(ctx)
			if err == nil {
				t.Error("expected error when console errors are present")
			}
		})
	})
}

func TestCheckNoConsoleWarnings(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	t.Run("no warnings initially", func(t *testing.T) {
		withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
			ctx := newActionContext(page)
			err := CheckNoConsoleWarnings(ctx)
			if err != nil {
				t.Errorf("CheckNoConsoleWarnings() error = %v", err)
			}
		})
	})

	t.Run("warnings present returns error", func(t *testing.T) {
		withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
			ctx := newActionContext(page)

			err := chromedp.Run(page.Ctx(),
				chromedp.Evaluate(`console.warn('test warning')`, nil),
			)
			if err != nil {
				t.Fatalf("emitting console.warn: %v", err)
			}

			time.Sleep(200 * time.Millisecond)

			err = CheckNoConsoleWarnings(ctx)
			if err == nil {
				t.Error("expected error when console warnings are present")
			}
		})
	})
}

func TestCaptureDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures element HTML", func(t *testing.T) {
			html, err := CaptureDOM(ctx, "#target", false)
			if err != nil {
				t.Errorf("CaptureDOM() error = %v", err)
			}
			if html == "" {
				t.Error("expected non-empty HTML")
			}

			if !strings.Contains(html, "Content Text") {
				t.Errorf("HTML should contain 'Content Text', got %s", html)
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			_, err := CaptureDOM(ctx, "#nonexistent", false)
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestCaptureDOM_IncludeShadowRoots(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLSerialisableShadow)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("includes shadow root as declarative template", func(t *testing.T) {
			html, err := CaptureDOM(ctx, "#container", true)
			if err != nil {
				t.Fatalf("CaptureDOM() error = %v", err)
			}

			if !strings.Contains(html, `shadowrootmode="open"`) {
				t.Errorf("expected shadowrootmode=\"open\" in output, got: %s", html)
			}
			if !strings.Contains(html, "Shadow Content") {
				t.Errorf("expected shadow content, got: %s", html)
			}
			if !strings.Contains(html, "</template>") {
				t.Errorf("expected closing </template>, got: %s", html)
			}
		})

		t.Run("without flag excludes shadow root", func(t *testing.T) {
			html, err := CaptureDOM(ctx, "#container", false)
			if err != nil {
				t.Fatalf("CaptureDOM() error = %v", err)
			}

			if strings.Contains(html, "shadowrootmode") {
				t.Errorf("expected no shadow root serialisation, got: %s", html)
			}
			if strings.Contains(html, "Shadow Content") {
				t.Errorf("expected no shadow content, got: %s", html)
			}
		})
	})
}

func TestScreenshotElement(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures element screenshot", func(t *testing.T) {
			buffer, err := ScreenshotElement(ctx, "#target")
			if err != nil {
				t.Errorf("ScreenshotElement() error = %v", err)
			}
			if len(buffer) == 0 {
				t.Error("expected non-empty screenshot")
			}

			if len(buffer) >= 4 && string(buffer[:4]) != "\x89PNG" {
				t.Error("expected PNG format")
			}
		})
	})
}

func TestScreenshotViewport(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures viewport screenshot", func(t *testing.T) {
			buffer, err := ScreenshotViewport(ctx)
			if err != nil {
				t.Errorf("ScreenshotViewport() error = %v", err)
			}
			if len(buffer) == 0 {
				t.Error("expected non-empty screenshot")
			}
		})
	})
}

func TestScreenshotFull(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures full page screenshot", func(t *testing.T) {
			buffer, err := ScreenshotFull(ctx)
			if err != nil {
				t.Errorf("ScreenshotFull() error = %v", err)
			}
			if len(buffer) == 0 {
				t.Error("expected non-empty screenshot")
			}
		})
	})
}

func TestSetViewport(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("sets viewport dimensions", func(t *testing.T) {
			err := SetViewport(ctx, 1024, 768)
			if err != nil {
				t.Errorf("SetViewport() error = %v", err)
			}
		})

		t.Run("sets mobile viewport", func(t *testing.T) {
			err := SetViewport(ctx, 375, 667)
			if err != nil {
				t.Errorf("SetViewport(mobile) error = %v", err)
			}
		})
	})
}

func TestExecuteAssertion_AllActions(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head>
<title>Execute Assertion Test</title>
<style>
#styled { color: rgb(255, 0, 0); font-size: 16px; }
#hidden { display: none; }
#visible { display: block; }
</style>
</head>
<body>
<div id="target" data-custom="custom-value" class="class1 class2">Content Text</div>
<div id="styled">Styled Element</div>
<div id="hidden">Hidden</div>
<div id="visible">Visible</div>
<input type="text" id="input" value="input-value" />
<input type="text" id="focused-input" />
<input type="text" id="unfocused-input" />
<input type="text" id="enabled-input" />
<input type="text" id="disabled-input" disabled />
<input type="checkbox" id="checked-checkbox" checked />
<input type="checkbox" id="unchecked-checkbox" />
<ul>
<li class="item">Item 1</li>
<li class="item">Item 2</li>
<li class="item">Item 3</li>
</ul>
<span id="html-target">Simple Content</span>
<form id="testform">
<input type="text" name="field1" value="value1" />
<input type="text" name="field2" value="value2" />
</form>
</body>
</html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#focused-input", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing input: %v", err)
		}

		t.Run("checkText", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkText",
				Selector: "#target",
				Expected: "Content Text",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkText) error = %v", err)
			}
		})

		t.Run("checkValue", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkValue",
				Selector: "#input",
				Expected: "input-value",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkValue) error = %v", err)
			}
		})

		t.Run("checkAttribute", func(t *testing.T) {
			step := &BrowserStep{
				Action:    "checkAttribute",
				Selector:  "#target",
				Attribute: "data-custom",
				Expected:  "custom-value",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkAttribute) error = %v", err)
			}
		})

		t.Run("checkAttribute with contains", func(t *testing.T) {
			step := &BrowserStep{
				Action:    "checkAttribute",
				Selector:  "#target",
				Attribute: "data-custom",
				Contains:  "custom",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkAttribute contains) error = %v", err)
			}
		})

		t.Run("checkClass", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkClass",
				Selector: "#target",
				Expected: "class1",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkClass) error = %v", err)
			}
		})

		t.Run("checkStyle", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkStyle",
				Selector: "#styled",
				Name:     "color",
				Expected: "rgb(255, 0, 0)",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkStyle) error = %v", err)
			}
		})

		t.Run("checkFocused", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkFocused",
				Selector: "#focused-input",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkFocused) error = %v", err)
			}
		})

		t.Run("checkNotFocused", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkNotFocused",
				Selector: "#unfocused-input",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkNotFocused) error = %v", err)
			}
		})

		t.Run("checkVisible", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkVisible",
				Selector: "#visible",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkVisible) error = %v", err)
			}
		})

		t.Run("checkHidden", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkHidden",
				Selector: "#hidden",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkHidden) error = %v", err)
			}
		})

		t.Run("checkEnabled", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkEnabled",
				Selector: "#enabled-input",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkEnabled) error = %v", err)
			}
		})

		t.Run("checkDisabled", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkDisabled",
				Selector: "#disabled-input",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkDisabled) error = %v", err)
			}
		})

		t.Run("checkChecked", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkChecked",
				Selector: "#checked-checkbox",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkChecked) error = %v", err)
			}
		})

		t.Run("checkUnchecked", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkUnchecked",
				Selector: "#unchecked-checkbox",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkUnchecked) error = %v", err)
			}
		})

		t.Run("checkElementCount", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkElementCount",
				Selector: ".item",
				Expected: 3,
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkElementCount) error = %v", err)
			}
		})

		t.Run("checkHTML", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkHTML",
				Selector: "#html-target",
				Expected: `<span id="html-target">Simple Content</span>`,
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkHTML) error = %v", err)
			}
		})

		t.Run("checkFormData", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkFormData",
				Selector: "#testform",
				Expected: map[string]any{
					"field1": "value1",
					"field2": "value2",
				},
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkFormData) error = %v", err)
			}
		})

		t.Run("checkNoConsoleErrors", func(t *testing.T) {
			step := &BrowserStep{
				Action: "checkNoConsoleErrors",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkNoConsoleErrors) error = %v", err)
			}
		})

		t.Run("checkNoConsoleWarnings", func(t *testing.T) {
			step := &BrowserStep{
				Action: "checkNoConsoleWarnings",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkNoConsoleWarnings) error = %v", err)
			}
		})

		t.Run("unknown action returns error", func(t *testing.T) {
			step := &BrowserStep{
				Action: "unknownAssertion",
			}
			err := ExecuteAssertion(ctx, step)
			if err == nil {
				t.Error("expected error for unknown action")
			}
		})
	})
}

func TestExecuteAssertion_ConsoleMessages(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.log('test log message')`, nil),
		)
		if err != nil {
			t.Fatalf("emitting console.log: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		t.Run("checkConsoleMessage", func(t *testing.T) {
			step := &BrowserStep{
				Action:  "checkConsoleMessage",
				Level:   "log",
				Message: "test log",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkConsoleMessage) error = %v", err)
			}
		})

		t.Run("checkNoConsoleMessage", func(t *testing.T) {
			step := &BrowserStep{
				Action:  "checkNoConsoleMessage",
				Level:   "log",
				Message: "xyz not present",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkNoConsoleMessage) error = %v", err)
			}
		})
	})
}

func TestCheckAttribute_AllBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("exact value match", func(t *testing.T) {
			err := CheckAttribute(ctx, "#target", "data-custom", "custom-value")
			if err != nil {
				t.Errorf("CheckAttribute() error = %v", err)
			}
		})

		t.Run("value mismatch returns error", func(t *testing.T) {
			err := CheckAttribute(ctx, "#target", "data-custom", "wrong-value")
			if err == nil {
				t.Error("expected error for attribute value mismatch")
			}
		})

		t.Run("nonexistent attribute returns error", func(t *testing.T) {
			err := CheckAttribute(ctx, "#target", "data-nonexistent", "any-value")
			if err == nil {
				t.Error("expected error for nonexistent attribute")
			}
		})

		t.Run("check class attribute", func(t *testing.T) {
			err := CheckAttribute(ctx, "#target", "class", "class1 class2")
			if err != nil {
				t.Errorf("CheckAttribute(class) error = %v", err)
			}
		})
	})
}

func TestCheckFocused_AllBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFocus)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("unfocused element returns error", func(t *testing.T) {
			err := CheckFocused(ctx, "#input1")
			if err == nil {
				t.Error("expected error for unfocused element")
			}
		})

		t.Run("focused element succeeds", func(t *testing.T) {

			err := chromedp.Run(page.Ctx(), chromedp.Focus("#input1", chromedp.ByQuery))
			if err != nil {
				t.Fatalf("focusing element: %v", err)
			}

			err = CheckFocused(ctx, "#input1")
			if err != nil {
				t.Errorf("CheckFocused() error = %v", err)
			}
		})
	})
}

func TestCheckNotFocused_AllBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFocus)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("unfocused element succeeds", func(t *testing.T) {
			err := CheckNotFocused(ctx, "#input1")
			if err != nil {
				t.Errorf("CheckNotFocused() error = %v", err)
			}
		})

		t.Run("focused element returns error", func(t *testing.T) {

			err := chromedp.Run(page.Ctx(), chromedp.Focus("#input1", chromedp.ByQuery))
			if err != nil {
				t.Fatalf("focusing element: %v", err)
			}

			err = CheckNotFocused(ctx, "#input1")
			if err == nil {
				t.Error("expected error for focused element")
			}
		})
	})
}

func TestScreenshotViewport_AllBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures viewport screenshot", func(t *testing.T) {
			data, err := ScreenshotViewport(ctx)
			if err != nil {
				t.Errorf("ScreenshotViewport() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty screenshot data")
			}
		})
	})
}

func TestScreenshotFull_AllBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLScroll)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures full page screenshot", func(t *testing.T) {
			data, err := ScreenshotFull(ctx)
			if err != nil {
				t.Errorf("ScreenshotFull() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty screenshot data")
			}
		})
	})
}

func TestCheckTextNotContains(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("substring not present succeeds", func(t *testing.T) {
			err := CheckTextNotContains(ctx, "#target", "xyz-not-present")
			if err != nil {
				t.Errorf("CheckTextNotContains() error = %v", err)
			}
		})

		t.Run("substring present returns error", func(t *testing.T) {
			err := CheckTextNotContains(ctx, "#target", "Content")
			if err == nil {
				t.Error("expected error when substring is present")
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			err := CheckTextNotContains(ctx, "#nonexistent", "anything")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestCheckAttributeNotContains(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("substring not in attribute succeeds", func(t *testing.T) {
			err := CheckAttributeNotContains(ctx, "#target", "data-custom", "xyz-absent")
			if err != nil {
				t.Errorf("CheckAttributeNotContains() error = %v", err)
			}
		})

		t.Run("substring in attribute returns error", func(t *testing.T) {
			err := CheckAttributeNotContains(ctx, "#target", "data-custom", "custom")
			if err == nil {
				t.Error("expected error when substring is in attribute")
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			err := CheckAttributeNotContains(ctx, "#nonexistent", "data-custom", "anything")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestExecuteAssertion_CheckTextNotContains(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("via ExecuteAssertion", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkTextNotContains",
				Selector: "#target",
				Expected: "xyz-not-present",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkTextNotContains) error = %v", err)
			}
		})
	})
}

func TestExecuteAssertion_CheckAttributeContains_Dispatch(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("checkAttributeContains via dispatch", func(t *testing.T) {
			step := &BrowserStep{
				Action:    "checkAttributeContains",
				Selector:  "#target",
				Attribute: "data-custom",
				Expected:  "custom",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkAttributeContains) error = %v", err)
			}
		})

		t.Run("checkAttributeNotContains via dispatch", func(t *testing.T) {
			step := &BrowserStep{
				Action:    "checkAttributeNotContains",
				Selector:  "#target",
				Attribute: "data-custom",
				Expected:  "xyz-absent",
			}
			err := ExecuteAssertion(ctx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkAttributeNotContains) error = %v", err)
			}
		})
	})
}
