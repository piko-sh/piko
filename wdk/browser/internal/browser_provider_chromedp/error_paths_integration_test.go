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
	"context"
	"testing"
	"time"
)

const (
	testHTMLErrorPaths = `<!DOCTYPE html>
<html>
<head><title>Error Paths Test</title></head>
<body>
<button id="btn">Button</button>
<input type="text" id="input" value="hello" data-role="textbox" class="form-input primary" />
<input type="checkbox" id="cb" checked />
<div id="visible" style="display:block">Visible</div>
<div id="hidden" style="display:none">Hidden</div>
<div id="disabled-wrapper"><input type="text" id="disabled-input" disabled /></div>
<form id="myform"><input name="field1" value="val1" /></form>
<div id="content" data-info="test-value">Some <b>HTML</b> content</div>
<iframe id="myframe" srcdoc="<html><body><div id='inner'>Frame</div></body></html>"></iframe>
</body>
</html>`
	noSuchSelector = "#nonexistent-element-xyz-12345"
)

func TestActionErrorPaths(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		testCases := []struct {
			action func() error
			name   string
		}{
			{name: "Click", action: func() error { return Click(ctx, noSuchSelector) }},
			{name: "DoubleClick", action: func() error { return DoubleClick(ctx, noSuchSelector) }},
			{name: "Hover", action: func() error { return Hover(ctx, noSuchSelector) }},
			{name: "RightClick", action: func() error { return RightClick(ctx, noSuchSelector) }},
			{name: "Fill", action: func() error { return Fill(ctx, noSuchSelector, "text") }},
			{name: "Clear", action: func() error { return Clear(ctx, noSuchSelector) }},
			{name: "Submit", action: func() error { return Submit(ctx, noSuchSelector) }},
			{name: "Check", action: func() error { return Check(ctx, noSuchSelector) }},
			{name: "Uncheck", action: func() error { return Uncheck(ctx, noSuchSelector) }},
			{name: "Focus", action: func() error { return Focus(ctx, noSuchSelector) }},
			{name: "Blur", action: func() error { return Blur(ctx, noSuchSelector) }},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.action()
				if err == nil {
					t.Errorf("%s() with non-existent selector should return error", tc.name)
				}
			})
		}
	})
}

func TestAssertionErrorPaths(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		testCases := []struct {
			action func() error
			name   string
		}{
			{name: "CheckVisible", action: func() error { return CheckVisible(ctx, noSuchSelector) }},

			{name: "CheckEnabled", action: func() error { return CheckEnabled(ctx, noSuchSelector) }},
			{name: "CheckDisabled", action: func() error { return CheckDisabled(ctx, noSuchSelector) }},
			{name: "CheckChecked", action: func() error { return CheckChecked(ctx, noSuchSelector) }},
			{name: "CheckUnchecked", action: func() error { return CheckUnchecked(ctx, noSuchSelector) }},
			{name: "CheckFocused", action: func() error { return CheckFocused(ctx, noSuchSelector) }},
			{name: "CheckNotFocused", action: func() error { return CheckNotFocused(ctx, noSuchSelector) }},
			{name: "CheckHTML", action: func() error { return CheckHTML(ctx, noSuchSelector, "") }},
			{name: "CheckAttribute", action: func() error { return CheckAttribute(ctx, noSuchSelector, "id", "") }},
			{name: "CheckAttributeContains", action: func() error {
				return CheckAttributeContains(ctx, noSuchSelector, "id", "x")
			}},
			{name: "CheckAttributeNotContains", action: func() error {
				return CheckAttributeNotContains(ctx, noSuchSelector, "id", "x")
			}},
			{name: "CheckClass", action: func() error { return CheckClass(ctx, noSuchSelector, "x") }},
			{name: "CheckStyle", action: func() error { return CheckStyle(ctx, noSuchSelector, "color", "red") }},
			{name: "CheckFormData", action: func() error {
				return CheckFormData(ctx, noSuchSelector, map[string]any{"k": "v"})
			}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.action()
				if err == nil {
					t.Errorf("%s() with non-existent selector should return error", tc.name)
				}
			})
		}
	})
}

func TestDOMErrorPaths(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("FindElement", func(t *testing.T) {
			_, err := FindElement(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("FindElements returns empty", func(t *testing.T) {
			elems, err := FindElements(page.Ctx(), noSuchSelector)
			if err != nil {
				t.Logf("FindElements error (acceptable): %v", err)
			}
			if len(elems) > 0 {
				t.Errorf("expected 0 elements, got %d", len(elems))
			}
		})

		t.Run("GetElementText", func(t *testing.T) {
			_, err := GetElementText(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("GetElementHTML", func(t *testing.T) {
			_, err := GetElementHTML(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("GetElementValue", func(t *testing.T) {
			_, err := GetElementValue(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("GetElementAttribute", func(t *testing.T) {
			_, err := GetElementAttribute(page.Ctx(), noSuchSelector, "id")
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("IsElementVisible returns false for non-existent", func(t *testing.T) {
			visible, err := IsElementVisible(page.Ctx(), noSuchSelector)
			if err != nil {
				t.Logf("IsElementVisible() error (acceptable): %v", err)
			}
			if visible {
				t.Error("expected non-existent element to not be visible")
			}
		})

		t.Run("IsElementChecked", func(t *testing.T) {
			_, err := IsElementChecked(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("IsElementEnabled", func(t *testing.T) {
			_, err := IsElementEnabled(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("EvalOnElement", func(t *testing.T) {
			_, err := EvalOnElement(page.Ctx(), noSuchSelector, "return true;")
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("ScrollIntoView", func(t *testing.T) {
			err := ScrollIntoView(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("GetAllAttributes", func(t *testing.T) {
			_, err := GetAllAttributes(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("SetElementAttribute", func(t *testing.T) {
			err := SetElementAttribute(page.Ctx(), noSuchSelector, "data-x", "v")
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("RemoveElementAttribute", func(t *testing.T) {
			err := RemoveElementAttribute(page.Ctx(), noSuchSelector, "data-x")
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("GetElementDimensions", func(t *testing.T) {
			_, err := GetElementDimensions(page.Ctx(), noSuchSelector)
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

func TestNavigationStop(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("stop loading", func(t *testing.T) {
			err := Stop(ctx)
			if err != nil {
				t.Errorf("Stop() error = %v", err)
			}
		})
	})
}

func TestGetURLAndTitle(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("GetURL returns current URL", func(t *testing.T) {
			url, err := GetURL(ctx)
			if err != nil {
				t.Fatalf("GetURL() error = %v", err)
			}
			if url == "" {
				t.Error("expected non-empty URL")
			}
		})

		t.Run("GetTitle returns current title", func(t *testing.T) {
			title, err := GetTitle(ctx)
			if err != nil {
				t.Fatalf("GetTitle() error = %v", err)
			}
			if title != "Error Paths Test" {
				t.Errorf("expected 'Error Paths Test', got %q", title)
			}
		})
	})
}

func TestCookieValueOperations(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("SetCookie then GetCookieValue", func(t *testing.T) {
			err := SetCookie(ctx, "testcookie", "testvalue", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}

			value, err := GetCookieValue(ctx, "testcookie")
			if err != nil {
				t.Fatalf("GetCookieValue() error = %v", err)
			}
			if value != "testvalue" {
				t.Errorf("expected 'testvalue', got %q", value)
			}
		})

		t.Run("GetCookieValue non-existent returns empty", func(t *testing.T) {
			value, err := GetCookieValue(ctx, "no-such-cookie-xyz")
			if err != nil {
				t.Fatalf("GetCookieValue() error = %v", err)
			}
			if value != "" {
				t.Errorf("expected empty string, got %q", value)
			}
		})

		t.Run("HasCookie", func(t *testing.T) {
			found, err := HasCookie(ctx, "testcookie")
			if err != nil {
				t.Fatalf("HasCookie() error = %v", err)
			}
			if !found {
				t.Error("expected cookie to be found")
			}
		})

		t.Run("DeleteCookie", func(t *testing.T) {
			err := DeleteCookie(ctx, "testcookie")
			if err != nil {
				t.Fatalf("DeleteCookie() error = %v", err)
			}

			found, err := HasCookie(ctx, "testcookie")
			if err != nil {
				t.Fatalf("HasCookie() error = %v", err)
			}
			if found {
				t.Error("expected cookie to be deleted")
			}
		})

		t.Run("ClearCookies", func(t *testing.T) {
			_ = SetCookie(ctx, "a", "1", nil)
			_ = SetCookie(ctx, "b", "2", nil)
			err := ClearCookies(ctx)
			if err != nil {
				t.Fatalf("ClearCookies() error = %v", err)
			}
		})
	})
}

func TestAssertionSuccessPaths(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(300 * time.Millisecond)

		t.Run("CheckVisible on visible element succeeds", func(t *testing.T) {
			err := CheckVisible(ctx, "#visible")
			if err != nil {
				t.Errorf("CheckVisible() error = %v", err)
			}
		})

		t.Run("CheckVisible on hidden element fails", func(t *testing.T) {
			err := CheckVisible(ctx, "#hidden")
			if err == nil {
				t.Error("expected error for hidden element")
			}
		})

		t.Run("CheckHidden on hidden element succeeds", func(t *testing.T) {
			err := CheckHidden(ctx, "#hidden")
			if err != nil {
				t.Errorf("CheckHidden() error = %v", err)
			}
		})

		t.Run("CheckHidden on visible element fails", func(t *testing.T) {
			err := CheckHidden(ctx, "#visible")
			if err == nil {
				t.Error("expected error for visible element")
			}
		})

		t.Run("CheckEnabled on enabled input succeeds", func(t *testing.T) {
			err := CheckEnabled(ctx, "#input")
			if err != nil {
				t.Errorf("CheckEnabled() error = %v", err)
			}
		})

		t.Run("CheckEnabled on disabled input fails", func(t *testing.T) {
			err := CheckEnabled(ctx, "#disabled-input")
			if err == nil {
				t.Error("expected error for disabled element")
			}
		})

		t.Run("CheckDisabled on disabled input succeeds", func(t *testing.T) {
			err := CheckDisabled(ctx, "#disabled-input")
			if err != nil {
				t.Errorf("CheckDisabled() error = %v", err)
			}
		})

		t.Run("CheckDisabled on enabled input fails", func(t *testing.T) {
			err := CheckDisabled(ctx, "#input")
			if err == nil {
				t.Error("expected error for enabled element")
			}
		})

		t.Run("CheckChecked on checked checkbox succeeds", func(t *testing.T) {
			err := CheckChecked(ctx, "#cb")
			if err != nil {
				t.Errorf("CheckChecked() error = %v", err)
			}
		})

		t.Run("CheckChecked on button fails", func(t *testing.T) {
			err := CheckChecked(ctx, "#btn")
			if err == nil {
				t.Error("expected error for non-checkbox")
			}
		})

		t.Run("CheckUnchecked on button fails", func(t *testing.T) {
			err := CheckUnchecked(ctx, "#btn")
			if err == nil {
				t.Error("expected error for checked element")
			}
		})

		t.Run("CheckFocused on unfocused element fails", func(t *testing.T) {
			err := CheckFocused(ctx, "#btn")
			if err == nil {
				t.Error("expected error for unfocused element")
			}
		})

		t.Run("CheckNotFocused on unfocused element succeeds", func(t *testing.T) {
			err := CheckNotFocused(ctx, "#btn")
			if err != nil {
				t.Errorf("CheckNotFocused() error = %v", err)
			}
		})

		t.Run("CheckFocused after focus succeeds", func(t *testing.T) {
			err := Focus(ctx, "#input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			err = CheckFocused(ctx, "#input")
			if err != nil {
				t.Errorf("CheckFocused() error = %v", err)
			}
		})

		t.Run("CheckHTML matches", func(t *testing.T) {

			err := CheckHTML(ctx, "#btn", `<button id="btn">Button</button>`)
			if err != nil {
				t.Errorf("CheckHTML() error = %v", err)
			}
		})

		t.Run("CheckHTML mismatch", func(t *testing.T) {
			err := CheckHTML(ctx, "#btn", "Wrong")
			if err == nil {
				t.Error("expected error for HTML mismatch")
			}
		})

		t.Run("CheckAttribute matches", func(t *testing.T) {
			err := CheckAttribute(ctx, "#input", "type", "text")
			if err != nil {
				t.Errorf("CheckAttribute() error = %v", err)
			}
		})

		t.Run("CheckAttribute mismatch", func(t *testing.T) {
			err := CheckAttribute(ctx, "#input", "type", "password")
			if err == nil {
				t.Error("expected error for attribute mismatch")
			}
		})

		t.Run("CheckAttribute null check", func(t *testing.T) {
			err := CheckAttribute(ctx, "#input", "data-nonexistent", NullAttributeValue)
			if err != nil {
				t.Errorf("CheckAttribute() null check error = %v", err)
			}
		})

		t.Run("CheckAttribute expects null but exists", func(t *testing.T) {
			err := CheckAttribute(ctx, "#input", "type", NullAttributeValue)
			if err == nil {
				t.Error("expected error when attribute exists but null expected")
			}
		})

		t.Run("CheckAttribute expects value but null", func(t *testing.T) {
			err := CheckAttribute(ctx, "#input", "data-nonexistent", "somevalue")
			if err == nil {
				t.Error("expected error when attribute is null but value expected")
			}
		})

		t.Run("CheckClass matches", func(t *testing.T) {
			err := CheckClass(ctx, "#input", "form-input")
			if err != nil {
				t.Errorf("CheckClass() error = %v", err)
			}
		})

		t.Run("CheckClass mismatch", func(t *testing.T) {
			err := CheckClass(ctx, "#input", "no-such-class")
			if err == nil {
				t.Error("expected error for class mismatch")
			}
		})

		t.Run("CheckStyle on element", func(t *testing.T) {
			err := CheckStyle(ctx, "#visible", "display", "block")
			if err != nil {
				t.Errorf("CheckStyle() error = %v", err)
			}
		})

		t.Run("CheckStyle mismatch", func(t *testing.T) {
			err := CheckStyle(ctx, "#visible", "display", "none")
			if err == nil {
				t.Error("expected error for style mismatch")
			}
		})

		t.Run("CheckFormData matches", func(t *testing.T) {
			err := CheckFormData(ctx, "#myform", map[string]any{
				"field1": "val1",
			})
			if err != nil {
				t.Errorf("CheckFormData() error = %v", err)
			}
		})

		t.Run("CheckFormData mismatch", func(t *testing.T) {
			err := CheckFormData(ctx, "#myform", map[string]any{
				"field1": "wrong",
			})
			if err == nil {
				t.Error("expected error for form data mismatch")
			}
		})
	})
}

func TestScreenshotOperations(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("ScreenshotElementWithPadding", func(t *testing.T) {
			data, err := ScreenshotElementWithPadding(ctx, "#content", 10.0)
			if err != nil {
				t.Fatalf("ScreenshotElementWithPadding() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty screenshot data")
			}
		})

		t.Run("ScreenshotElementWithPadding zero padding", func(t *testing.T) {
			data, err := ScreenshotElementWithPadding(ctx, "#content", 0.0)
			if err != nil {
				t.Fatalf("ScreenshotElementWithPadding() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty screenshot data")
			}
		})

		t.Run("ScreenshotElementWithPadding nonexistent", func(t *testing.T) {
			_, err := ScreenshotElementWithPadding(ctx, noSuchSelector, 10.0)
			if err == nil {
				t.Error("expected error for non-existent element")
			}
		})

		t.Run("ScreenshotElement", func(t *testing.T) {
			data, err := ScreenshotElement(ctx, "#content")
			if err != nil {
				t.Fatalf("ScreenshotElement() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty screenshot data")
			}
		})

		t.Run("ScreenshotElement nonexistent", func(t *testing.T) {
			_, err := ScreenshotElement(ctx, noSuchSelector)
			if err == nil {
				t.Error("expected error for non-existent element")
			}
		})

		t.Run("ScreenshotViewport", func(t *testing.T) {
			data, err := ScreenshotViewport(ctx)
			if err != nil {
				t.Fatalf("ScreenshotViewport() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty screenshot data")
			}
		})

		t.Run("ScreenshotFull", func(t *testing.T) {
			data, err := ScreenshotFull(ctx)
			if err != nil {
				t.Fatalf("ScreenshotFull() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty screenshot data")
			}
		})
	})
}

func TestFrameOperationsExtended(t *testing.T) {
	t.Parallel()
	const testHTMLWithFrame = `<!DOCTYPE html>
<html>
<head><title>Frame Test</title></head>
<body>
<iframe id="test-frame" srcdoc="<html><body><div id='inner-el'>Hello Frame</div><input id='frame-input' type='text' value='' /></body></html>"></iframe>
</body>
</html>`

	server := newTestServer(testHTMLWithFrame)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		time.Sleep(500 * time.Millisecond)

		t.Run("GetFrames", func(t *testing.T) {
			frames, err := GetFrames(ctx)
			if err != nil {
				t.Fatalf("GetFrames() error = %v", err)
			}
			if len(frames) == 0 {
				t.Error("expected at least one frame")
			}
		})

		t.Run("CountFrames", func(t *testing.T) {
			count, err := CountFrames(ctx)
			if err != nil {
				t.Fatalf("CountFrames() error = %v", err)
			}
			if count == 0 {
				t.Error("expected at least one frame")
			}
		})

		t.Run("GetFrameBySelector", func(t *testing.T) {
			frame, err := GetFrameBySelector(ctx, "#test-frame")
			if err != nil {
				t.Fatalf("GetFrameBySelector() error = %v", err)
			}
			if frame == nil {
				t.Error("expected non-nil frame info")
			}
		})

		t.Run("GetFrameBySelector nonexistent", func(t *testing.T) {
			_, err := GetFrameBySelector(ctx, noSuchSelector)
			if err == nil {
				t.Error("expected error for non-existent frame")
			}
		})

		t.Run("IsFrameLoaded", func(t *testing.T) {
			loaded, err := IsFrameLoaded(ctx, "#test-frame")
			if err != nil {
				t.Fatalf("IsFrameLoaded() error = %v", err)
			}
			if !loaded {
				t.Error("expected frame to be loaded")
			}
		})

		t.Run("GetFrameDocument", func(t *testing.T) {
			html, err := GetFrameDocument(ctx, "#test-frame")
			if err != nil {
				t.Fatalf("GetFrameDocument() error = %v", err)
			}
			if html == "" {
				t.Error("expected non-empty HTML")
			}
		})

		t.Run("GetTextInFrame", func(t *testing.T) {
			text, err := GetTextInFrame(ctx, "#test-frame", "#inner-el")
			if err != nil {
				t.Fatalf("GetTextInFrame() error = %v", err)
			}
			if text != "Hello Frame" {
				t.Errorf("expected 'Hello Frame', got %q", text)
			}
		})

		t.Run("FillInFrame", func(t *testing.T) {
			err := FillInFrame(ctx, "#test-frame", "#frame-input", "frame-value")
			if err != nil {
				t.Errorf("FillInFrame() error = %v", err)
			}
		})

		t.Run("ExecuteInFrameContext", func(t *testing.T) {
			err := ExecuteInFrameContext(ctx, "#test-frame", func(_ context.Context) error {
				return nil
			})
			if err != nil {
				t.Fatalf("ExecuteInFrameContext() error = %v", err)
			}
		})

		t.Run("ExecuteInFrameContext nonexistent frame", func(t *testing.T) {
			err := ExecuteInFrameContext(ctx, noSuchSelector, func(_ context.Context) error {
				return nil
			})
			if err == nil {
				t.Error("expected error for non-existent frame")
			}
		})
	})
}

func TestSelectionOperations(t *testing.T) {
	t.Parallel()
	const testHTMLSelection = `<!DOCTYPE html>
<html>
<head><title>Selection Test</title></head>
<body>
<input type="text" id="text-input" value="hello world" />
<div id="editable" contenteditable="true">editable content</div>
</body>
</html>`

	server := newTestServer(testHTMLSelection)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("SetCursorPosition", func(t *testing.T) {
			err := Focus(ctx, "#text-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			err = SetCursorPosition(ctx, "#text-input", 3)
			if err != nil {
				t.Errorf("SetCursorPosition() error = %v", err)
			}
		})

		t.Run("GetCursorPosition", func(t *testing.T) {
			err := Focus(ctx, "#text-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			position, err := GetCursorPosition(ctx, "#text-input")
			if err != nil {
				t.Errorf("GetCursorPosition() error = %v", err)
			}
			t.Logf("cursor position: %d", position)
		})

		t.Run("SetSelection", func(t *testing.T) {
			err := Focus(ctx, "#text-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			err = SetSelection(ctx, "#text-input", 0, 5)
			if err != nil {

				t.Logf("SetSelection() returned error (may be expected): %v", err)
			}
		})

		t.Run("GetSelection", func(t *testing.T) {
			err := Focus(ctx, "#text-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			_ = SetSelection(ctx, "#text-input", 0, 5)
			start, end, err := GetSelection(ctx, "#text-input")
			if err != nil {
				t.Errorf("GetSelection() error = %v", err)
			}
			t.Logf("selection: start=%d end=%d", start, end)
		})

		t.Run("SelectAll", func(t *testing.T) {
			err := Focus(ctx, "#text-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			err = SelectAll(ctx, "#text-input")
			if err != nil {
				t.Errorf("SelectAll() error = %v", err)
			}
		})

		t.Run("CollapseSelection to end", func(t *testing.T) {
			err := Focus(ctx, "#text-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			err = CollapseSelection(ctx, true)
			if err != nil {
				t.Errorf("CollapseSelection(true) error = %v", err)
			}
		})

		t.Run("CollapseSelection to start", func(t *testing.T) {
			err := Focus(ctx, "#text-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}
			err = CollapseSelection(ctx, false)
			if err != nil {
				t.Errorf("CollapseSelection(false) error = %v", err)
			}
		})
	})
}

func TestStorageOperationsExtended(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("local storage set and get", func(t *testing.T) {
			err := SetLocalStorageItem(ctx, "ls-key", "ls-value")
			if err != nil {
				t.Fatalf("SetLocalStorageItem() error = %v", err)
			}

			value, _, err := GetLocalStorageItem(ctx, "ls-key")
			if err != nil {
				t.Fatalf("GetLocalStorageItem() error = %v", err)
			}
			if value != "ls-value" {
				t.Errorf("expected 'ls-value', got %q", value)
			}
		})

		t.Run("local storage length", func(t *testing.T) {
			length, err := GetLocalStorageLength(ctx)
			if err != nil {
				t.Fatalf("GetLocalStorageLength() error = %v", err)
			}
			if length == 0 {
				t.Error("expected non-zero length")
			}
		})

		t.Run("local storage get all", func(t *testing.T) {
			all, err := GetAllLocalStorage(ctx)
			if err != nil {
				t.Fatalf("GetAllLocalStorage() error = %v", err)
			}
			if len(all) == 0 {
				t.Error("expected non-empty storage")
			}
		})

		t.Run("local storage remove", func(t *testing.T) {
			err := RemoveLocalStorageItem(ctx, "ls-key")
			if err != nil {
				t.Fatalf("RemoveLocalStorageItem() error = %v", err)
			}
		})

		t.Run("local storage clear", func(t *testing.T) {
			_ = SetLocalStorageItem(ctx, "a", "1")
			err := ClearLocalStorage(ctx)
			if err != nil {
				t.Fatalf("ClearLocalStorage() error = %v", err)
			}
		})

		t.Run("session storage set and get", func(t *testing.T) {
			err := SetSessionStorageItem(ctx, "ss-key", "ss-value")
			if err != nil {
				t.Fatalf("SetSessionStorageItem() error = %v", err)
			}

			value, _, err := GetSessionStorageItem(ctx, "ss-key")
			if err != nil {
				t.Fatalf("GetSessionStorageItem() error = %v", err)
			}
			if value != "ss-value" {
				t.Errorf("expected 'ss-value', got %q", value)
			}
		})

		t.Run("session storage length", func(t *testing.T) {
			length, err := GetSessionStorageLength(ctx)
			if err != nil {
				t.Fatalf("GetSessionStorageLength() error = %v", err)
			}
			if length == 0 {
				t.Error("expected non-zero length")
			}
		})

		t.Run("session storage get all", func(t *testing.T) {
			all, err := GetAllSessionStorage(ctx)
			if err != nil {
				t.Fatalf("GetAllSessionStorage() error = %v", err)
			}
			if len(all) == 0 {
				t.Error("expected non-empty storage")
			}
		})

		t.Run("session storage remove", func(t *testing.T) {
			err := RemoveSessionStorageItem(ctx, "ss-key")
			if err != nil {
				t.Fatalf("RemoveSessionStorageItem() error = %v", err)
			}
		})

		t.Run("session storage clear", func(t *testing.T) {
			_ = SetSessionStorageItem(ctx, "b", "2")
			err := ClearSessionStorage(ctx)
			if err != nil {
				t.Fatalf("ClearSessionStorage() error = %v", err)
			}
		})
	})
}

func TestKeyboardOperationsExtended(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLKeyboard)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := Focus(ctx, "#input")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		t.Run("Type text", func(t *testing.T) {
			err := Type(ctx, "hello")
			if err != nil {
				t.Errorf("Type() error = %v", err)
			}
		})

		t.Run("Press Enter", func(t *testing.T) {
			err := Press(ctx, "Enter")
			if err != nil {
				t.Errorf("Press(Enter) error = %v", err)
			}
		})

		t.Run("Press Tab", func(t *testing.T) {
			err := Press(ctx, "Tab")
			if err != nil {
				t.Errorf("Press(Tab) error = %v", err)
			}
		})

		t.Run("Press Escape", func(t *testing.T) {
			err := Press(ctx, "Escape")
			if err != nil {
				t.Errorf("Press(Escape) error = %v", err)
			}
		})

		t.Run("Press ArrowDown", func(t *testing.T) {
			err := Press(ctx, "ArrowDown")
			if err != nil {
				t.Errorf("Press(ArrowDown) error = %v", err)
			}
		})

		t.Run("Press character a", func(t *testing.T) {
			err := Press(ctx, "a")
			if err != nil {
				t.Errorf("Press(a) error = %v", err)
			}
		})

		t.Run("Press Ctrl+a modifier", func(t *testing.T) {
			err := Press(ctx, "Control+a")
			if err != nil {
				t.Errorf("Press(Control+a) error = %v", err)
			}
		})

		t.Run("Press Shift+a modifier", func(t *testing.T) {
			err := Press(ctx, "Shift+a")
			if err != nil {
				t.Errorf("Press(Shift+a) error = %v", err)
			}
		})

		t.Run("Press Backspace", func(t *testing.T) {
			err := Press(ctx, "Backspace")
			if err != nil {
				t.Errorf("Press(Backspace) error = %v", err)
			}
		})

		t.Run("Press Delete", func(t *testing.T) {
			err := Press(ctx, "Delete")
			if err != nil {
				t.Errorf("Press(Delete) error = %v", err)
			}
		})

		t.Run("Press Home", func(t *testing.T) {
			err := Press(ctx, "Home")
			if err != nil {
				t.Errorf("Press(Home) error = %v", err)
			}
		})

		t.Run("Press End", func(t *testing.T) {
			err := Press(ctx, "End")
			if err != nil {
				t.Errorf("Press(End) error = %v", err)
			}
		})

		t.Run("Press space", func(t *testing.T) {
			err := Press(ctx, " ")
			if err != nil {
				t.Errorf("Press(space) error = %v", err)
			}
		})
	})
}

func TestEmulationOperations(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("SetViewport", func(t *testing.T) {
			err := SetViewport(ctx, 800, 600)
			if err != nil {
				t.Errorf("SetViewport() error = %v", err)
			}
		})

		t.Run("EmulateDevice", func(t *testing.T) {
			err := EmulateDevice(ctx, DeviceParams{
				Width:  375,
				Height: 812,
				Scale:  2.0,
				Mobile: true,
			})
			if err != nil {
				t.Errorf("EmulateDevice() error = %v", err)
			}
		})

		t.Run("ResetEmulation", func(t *testing.T) {
			err := ResetEmulation(ctx)
			if err != nil {
				t.Errorf("ResetEmulation() error = %v", err)
			}
		})
	})
}

func TestCaptureDOMExtended(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("CaptureDOM success", func(t *testing.T) {
			html, err := CaptureDOM(ctx, "body", false)
			if err != nil {
				t.Fatalf("CaptureDOM() error = %v", err)
			}
			if html == "" {
				t.Error("expected non-empty DOM capture")
			}
		})
	})
}

func TestHeaderOperations(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLErrorPaths)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("SetExtraHTTPHeaders", func(t *testing.T) {
			err := SetExtraHTTPHeaders(ctx, map[string]string{
				"X-Test-Header": "test-value",
			})
			if err != nil {
				t.Errorf("SetExtraHTTPHeaders() error = %v", err)
			}
		})

		t.Run("SetUserAgent", func(t *testing.T) {
			err := SetUserAgent(ctx, "TestUserAgent/1.0")
			if err != nil {
				t.Errorf("SetUserAgent() error = %v", err)
			}
		})

		t.Run("GetResponseHeaders", func(t *testing.T) {
			headers, err := GetResponseHeaders(ctx)
			if err != nil {
				t.Fatalf("GetResponseHeaders() error = %v", err)
			}
			if len(headers) == 0 {
				t.Error("expected non-empty headers")
			}
		})
	})
}
