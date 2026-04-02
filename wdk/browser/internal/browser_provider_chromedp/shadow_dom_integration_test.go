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
	"os"
	"path/filepath"
	"testing"
	"time"

	"piko.sh/piko/wdk/safedisk"
)

func createTempTestFile(name, content string) (*os.File, error) {
	f, err := os.CreateTemp("", name)
	if err != nil {
		return nil, err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return nil, err
	}
	return f, nil
}

func removeTempFile(path string) error {
	return os.Remove(path)
}

func TestClickInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("click button in shadow DOM", func(t *testing.T) {

			err := Click(ctx, "#host >>> #shadow-btn")
			if err != nil {
				t.Fatalf("Click() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "clicked" {
				t.Errorf("expected 'clicked', got %q", text)
			}
		})

		t.Run("click nonexistent element in shadow DOM", func(t *testing.T) {
			err := Click(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestDoubleClickInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("double-click button in shadow DOM", func(t *testing.T) {
			err := DoubleClick(ctx, "#host >>> #shadow-dblclick-btn")
			if err != nil {
				t.Fatalf("DoubleClick() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "double-clicked" {
				t.Errorf("expected 'double-clicked', got %q", text)
			}
		})

		t.Run("double-click nonexistent element", func(t *testing.T) {
			err := DoubleClick(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestRightClickInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("right-click button in shadow DOM", func(t *testing.T) {
			err := RightClick(ctx, "#host >>> #shadow-rightclick-btn")
			if err != nil {
				t.Fatalf("RightClick() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "right-clicked" {
				t.Errorf("expected 'right-clicked', got %q", text)
			}
		})

		t.Run("right-click nonexistent element", func(t *testing.T) {
			err := RightClick(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestHoverInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("hover over element in shadow DOM", func(t *testing.T) {
			err := Hover(ctx, "#host >>> #shadow-hover-target")
			if err != nil {
				t.Fatalf("Hover() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "hovered" {
				t.Errorf("expected 'hovered', got %q", text)
			}
		})

		t.Run("hover nonexistent element", func(t *testing.T) {
			err := Hover(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestFillInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("fill input in shadow DOM", func(t *testing.T) {
			err := Fill(ctx, "#host >>> #shadow-input", "test value")
			if err != nil {
				t.Fatalf("Fill() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "input:test value" {
				t.Errorf("expected 'input:test value', got %q", text)
			}
		})

		t.Run("fill nonexistent element", func(t *testing.T) {
			err := Fill(ctx, "#host >>> #nonexistent", "value")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestClearInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("clear input in shadow DOM", func(t *testing.T) {

			err := Fill(ctx, "#host >>> #shadow-input", "initial value")
			if err != nil {
				t.Fatalf("Fill() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			err = Clear(ctx, "#host >>> #shadow-input")
			if err != nil {
				t.Fatalf("Clear() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "input:" {
				t.Errorf("expected 'input:', got %q", text)
			}
		})

		t.Run("clear nonexistent element", func(t *testing.T) {
			err := Clear(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestCheckInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("check checkbox in shadow DOM", func(t *testing.T) {
			err := Check(ctx, "#host >>> #shadow-checkbox")
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "checkbox:true" {
				t.Errorf("expected 'checkbox:true', got %q", text)
			}
		})

		t.Run("check already checked checkbox", func(t *testing.T) {

			err := Check(ctx, "#host >>> #shadow-checkbox-checked")
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

		})

		t.Run("check nonexistent element", func(t *testing.T) {
			err := Check(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestUncheckInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("uncheck checkbox in shadow DOM", func(t *testing.T) {
			err := Uncheck(ctx, "#host >>> #shadow-checkbox-checked")
			if err != nil {
				t.Fatalf("Uncheck() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "checkbox-checked:false" {
				t.Errorf("expected 'checkbox-checked:false', got %q", text)
			}
		})

		t.Run("uncheck already unchecked checkbox", func(t *testing.T) {
			err := Uncheck(ctx, "#host >>> #shadow-checkbox")
			if err != nil {
				t.Fatalf("Uncheck() error = %v", err)
			}

		})

		t.Run("uncheck nonexistent element", func(t *testing.T) {
			err := Uncheck(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestFocusInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("focus input in shadow DOM", func(t *testing.T) {
			err := Focus(ctx, "#host >>> #shadow-focus-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "focused" {
				t.Errorf("expected 'focused', got %q", text)
			}
		})

		t.Run("focus nonexistent element", func(t *testing.T) {
			err := Focus(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestBlurInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("blur input in shadow DOM", func(t *testing.T) {

			err := Focus(ctx, "#host >>> #shadow-focus-input")
			if err != nil {
				t.Fatalf("Focus() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			err = Blur(ctx, "#host >>> #shadow-focus-input")
			if err != nil {
				t.Fatalf("Blur() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "blurred" {
				t.Errorf("expected 'blurred', got %q", text)
			}
		})

		t.Run("blur nonexistent element", func(t *testing.T) {
			err := Blur(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestSubmitInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("submit form in shadow DOM succeeds", func(t *testing.T) {

			err := Submit(ctx, "#host >>> #shadow-form")
			if err != nil {
				t.Fatalf("Submit() error = %v", err)
			}

		})

		t.Run("submit nonexistent form", func(t *testing.T) {
			err := Submit(ctx, "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestShadowDOMWithInvalidHost(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("click with invalid host", func(t *testing.T) {
			err := Click(ctx, "#invalid-host >>> #shadow-btn")
			if err == nil {
				t.Error("expected error for invalid host")
			}
		})

		t.Run("fill with invalid host", func(t *testing.T) {
			err := Fill(ctx, "#invalid-host >>> #shadow-input", "test")
			if err == nil {
				t.Error("expected error for invalid host")
			}
		})

		t.Run("hover with invalid host", func(t *testing.T) {
			err := Hover(ctx, "#invalid-host >>> #shadow-hover-target")
			if err == nil {
				t.Error("expected error for invalid host")
			}
		})
	})
}

func TestGetElementTextInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("get text from shadow DOM", func(t *testing.T) {
			text, err := GetElementText(page.Ctx(), "#host >>> #inner")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "Shadow Content" {
				t.Errorf("expected 'Shadow Content', got %q", text)
			}
		})

		t.Run("get text from nonexistent element", func(t *testing.T) {
			text, err := GetElementText(page.Ctx(), "#host >>> #nonexistent")

			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "" {
				t.Errorf("expected empty string, got %q", text)
			}
		})
	})
}

func TestGetElementAttributeInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("get existing attribute", func(t *testing.T) {
			attr, err := GetElementAttribute(page.Ctx(), "#host >>> #inner", "id")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr == nil {
				t.Fatal("expected attribute to exist")
			}
			if *attr != "inner" {
				t.Errorf("expected 'inner', got %q", *attr)
			}
		})

		t.Run("get nonexistent attribute", func(t *testing.T) {
			attr, err := GetElementAttribute(page.Ctx(), "#host >>> #inner", "nonexistent")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr != nil {
				t.Errorf("expected nil for nonexistent attribute, got %q", *attr)
			}
		})

		t.Run("get attribute from nonexistent element", func(t *testing.T) {
			attr, err := GetElementAttribute(page.Ctx(), "#host >>> #nonexistent", "id")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr != nil {
				t.Errorf("expected nil, got %q", *attr)
			}
		})
	})
}

func TestGetElementHTMLInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("get HTML from shadow DOM", func(t *testing.T) {
			html, err := GetElementHTML(page.Ctx(), "#host >>> #inner")
			if err != nil {
				t.Fatalf("GetElementHTML() error = %v", err)
			}
			if html == "" {
				t.Error("expected non-empty HTML")
			}

			if !containsString(html, "Shadow Content") {
				t.Errorf("HTML should contain 'Shadow Content', got %q", html)
			}
		})

		t.Run("get HTML from nonexistent element", func(t *testing.T) {
			html, err := GetElementHTML(page.Ctx(), "#host >>> #nonexistent")
			if err != nil {
				t.Fatalf("GetElementHTML() error = %v", err)
			}
			if html != "" {
				t.Errorf("expected empty string, got %q", html)
			}
		})
	})
}

func TestGetElementValueInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("get value after fill", func(t *testing.T) {

			err := Fill(ctx, "#host >>> #shadow-input", "test value")
			if err != nil {
				t.Fatalf("Fill() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			value, err := GetElementValue(page.Ctx(), "#host >>> #shadow-input")
			if err != nil {
				t.Fatalf("GetElementValue() error = %v", err)
			}
			if value != "test value" {
				t.Errorf("expected 'test value', got %q", value)
			}
		})

		t.Run("get value from nonexistent element", func(t *testing.T) {
			value, err := GetElementValue(page.Ctx(), "#host >>> #nonexistent")
			if err != nil {
				t.Fatalf("GetElementValue() error = %v", err)
			}
			if value != "" {
				t.Errorf("expected empty string, got %q", value)
			}
		})
	})
}

func TestIsElementVisibleInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("visible element", func(t *testing.T) {
			visible, err := IsElementVisible(page.Ctx(), "#host >>> #inner")
			if err != nil {
				t.Fatalf("IsElementVisible() error = %v", err)
			}
			if !visible {
				t.Error("expected element to be visible")
			}
		})

		t.Run("nonexistent element", func(t *testing.T) {
			visible, err := IsElementVisible(page.Ctx(), "#host >>> #nonexistent")
			if err != nil {
				t.Fatalf("IsElementVisible() error = %v", err)
			}
			if visible {
				t.Error("expected nonexistent element to not be visible")
			}
		})
	})
}

func TestIsElementCheckedInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("unchecked checkbox", func(t *testing.T) {
			checked, err := IsElementChecked(page.Ctx(), "#host >>> #shadow-checkbox")
			if err != nil {
				t.Fatalf("IsElementChecked() error = %v", err)
			}
			if checked {
				t.Error("expected checkbox to be unchecked")
			}
		})

		t.Run("checked checkbox", func(t *testing.T) {
			checked, err := IsElementChecked(page.Ctx(), "#host >>> #shadow-checkbox-checked")
			if err != nil {
				t.Fatalf("IsElementChecked() error = %v", err)
			}
			if !checked {
				t.Error("expected checkbox to be checked")
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			_, err := IsElementChecked(page.Ctx(), "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestIsElementEnabledInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("enabled element", func(t *testing.T) {
			enabled, err := IsElementEnabled(page.Ctx(), "#host >>> #shadow-input")
			if err != nil {
				t.Fatalf("IsElementEnabled() error = %v", err)
			}
			if !enabled {
				t.Error("expected element to be enabled")
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			_, err := IsElementEnabled(page.Ctx(), "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestScrollIntoViewInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("scroll to shadow element", func(t *testing.T) {
			err := ScrollIntoView(page.Ctx(), "#host >>> #inner")
			if err != nil {
				t.Fatalf("ScrollIntoView() error = %v", err)
			}

		})

		t.Run("scroll to nonexistent element", func(t *testing.T) {
			err := ScrollIntoView(page.Ctx(), "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestGetAllAttributesInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("get all attributes", func(t *testing.T) {
			attrs, err := GetAllAttributes(page.Ctx(), "#host >>> #inner")
			if err != nil {
				t.Fatalf("GetAllAttributes() error = %v", err)
			}
			if attrs["id"] != "inner" {
				t.Errorf("expected id='inner', got %q", attrs["id"])
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			_, err := GetAllAttributes(page.Ctx(), "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestSetElementAttributeInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("set attribute", func(t *testing.T) {
			err := SetElementAttribute(page.Ctx(), "#host >>> #inner", "data-test", "test-value")
			if err != nil {
				t.Fatalf("SetElementAttribute() error = %v", err)
			}

			attr, err := GetElementAttribute(page.Ctx(), "#host >>> #inner", "data-test")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr == nil || *attr != "test-value" {
				t.Error("expected attribute to be set")
			}
		})

		t.Run("set attribute on nonexistent element", func(t *testing.T) {
			err := SetElementAttribute(page.Ctx(), "#host >>> #nonexistent", "data-test", "value")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestRemoveElementAttributeInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("remove attribute", func(t *testing.T) {

			err := SetElementAttribute(page.Ctx(), "#host >>> #inner", "data-remove", "to-remove")
			if err != nil {
				t.Fatalf("SetElementAttribute() error = %v", err)
			}

			attr, err := GetElementAttribute(page.Ctx(), "#host >>> #inner", "data-remove")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr == nil {
				t.Fatal("expected attribute to exist before removal")
			}

			err = RemoveElementAttribute(page.Ctx(), "#host >>> #inner", "data-remove")
			if err != nil {
				t.Fatalf("RemoveElementAttribute() error = %v", err)
			}

			attr, err = GetElementAttribute(page.Ctx(), "#host >>> #inner", "data-remove")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr != nil {
				t.Errorf("expected attribute to be removed, got %q", *attr)
			}
		})

		t.Run("remove attribute from nonexistent element", func(t *testing.T) {
			err := RemoveElementAttribute(page.Ctx(), "#host >>> #nonexistent", "data-test")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestGetElementDimensionsInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("get dimensions", func(t *testing.T) {
			dims, err := GetElementDimensions(page.Ctx(), "#host >>> #inner")
			if err != nil {
				t.Fatalf("GetElementDimensions() error = %v", err)
			}
			if dims == nil {
				t.Fatal("expected dimensions, got nil")
			}

			if dims.Width <= 0 || dims.Height <= 0 {
				t.Errorf("expected positive dimensions, got width=%v height=%v", dims.Width, dims.Height)
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			_, err := GetElementDimensions(page.Ctx(), "#host >>> #nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestEvalOnElementInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("eval on shadow element", func(t *testing.T) {
			result, err := EvalOnElement(page.Ctx(), "#host >>> #inner", "function() { return this.tagName; }")
			if err != nil {
				t.Fatalf("EvalOnElement() error = %v", err)
			}
			tagName, ok := result.(string)
			if !ok {
				t.Fatalf("expected string result, got %T", result)
			}
			if tagName != "SPAN" {
				t.Errorf("expected 'SPAN', got %q", tagName)
			}
		})

		t.Run("eval on nonexistent element returns nil", func(t *testing.T) {
			result, err := EvalOnElement(page.Ctx(), "#host >>> #nonexistent", "function() { return this.tagName; }")
			if err != nil {
				t.Fatalf("EvalOnElement() error = %v", err)
			}

			if result != nil {
				t.Errorf("expected nil for nonexistent element, got %v", result)
			}
		})
	})
}

func TestFindElementsInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("find elements", func(t *testing.T) {

			nodes, err := FindElements(page.Ctx(), "#host >>> button")
			if err != nil {
				t.Fatalf("FindElements() error = %v", err)
			}

			if len(nodes) < 4 {
				t.Errorf("expected at least 4 buttons, got %d", len(nodes))
			}
		})

		t.Run("find no elements", func(t *testing.T) {
			nodes, err := FindElements(page.Ctx(), "#host >>> .nonexistent-class")
			if err != nil {
				t.Fatalf("FindElements() error = %v", err)
			}
			if len(nodes) != 0 {
				t.Errorf("expected 0 elements, got %d", len(nodes))
			}
		})
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestClickContenteditableInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMContenteditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("click positions cursor in contenteditable", func(t *testing.T) {

			err := Click(ctx, "#host >>> #editor")
			if err != nil {
				t.Fatalf("Click() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}

			if text == "" {
				t.Error("expected cursor position to be reported after click")
			}
			if !containsString(text, "cursor:") {
				t.Errorf("expected cursor position report, got %q", text)
			}
		})

		t.Run("type after click inserts at cursor position", func(t *testing.T) {

			err := Click(ctx, "#host >>> #editor")
			if err != nil {
				t.Fatalf("Click() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			err = Type(ctx, "TEST")
			if err != nil {
				t.Fatalf("Type() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}

			if !containsString(text, "content:") {
				t.Errorf("expected content report after typing, got %q", text)
			}
			if !containsString(text, "TEST") {
				t.Errorf("expected 'TEST' to be inserted, got %q", text)
			}
		})
	})
}

func TestHasShadowRoot(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("element with shadow root", func(t *testing.T) {
			hasShadow, err := HasShadowRoot(page.Ctx(), "#host")
			if err != nil {
				t.Fatalf("HasShadowRoot() error = %v", err)
			}
			if !hasShadow {
				t.Error("expected element to have shadow root")
			}
		})

		t.Run("element without shadow root", func(t *testing.T) {
			hasShadow, err := HasShadowRoot(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("HasShadowRoot() error = %v", err)
			}
			if hasShadow {
				t.Error("expected element to not have shadow root")
			}
		})

		t.Run("nonexistent element", func(t *testing.T) {
			hasShadow, err := HasShadowRoot(page.Ctx(), "#nonexistent")
			if err != nil {
				t.Fatalf("HasShadowRoot() error = %v", err)
			}
			if hasShadow {
				t.Error("expected nonexistent element to not have shadow root")
			}
		})
	})
}

func TestGetShadowRootHTML(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMComprehensive)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("get shadow root HTML", func(t *testing.T) {
			html, err := GetShadowRootHTML(page.Ctx(), "#host")
			if err != nil {
				t.Fatalf("GetShadowRootHTML() error = %v", err)
			}
			if html == "" {
				t.Error("expected non-empty HTML")
			}

			if !containsString(html, "Shadow Content") {
				t.Errorf("HTML should contain 'Shadow Content', got %q", html)
			}
			if !containsString(html, "shadow-btn") {
				t.Errorf("HTML should contain 'shadow-btn', got %q", html)
			}
		})

		t.Run("element without shadow root returns error", func(t *testing.T) {
			_, err := GetShadowRootHTML(page.Ctx(), "#result")
			if err == nil {
				t.Error("expected error for element without shadow root")
			}
		})

		t.Run("nonexistent element returns error", func(t *testing.T) {
			_, err := GetShadowRootHTML(page.Ctx(), "#nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestSetFilesInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMFileInput)
	defer server.Close()

	tmpFile, err := createTempTestFile("test-file.txt", "test content")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = removeTempFile(tmpFile.Name())
	}()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		srcSandbox, err := safedisk.NewNoOpSandbox(filepath.Dir(tmpFile.Name()), safedisk.ModeReadOnly)
		if err != nil {
			t.Fatalf("creating source sandbox: %v", err)
		}
		ctx.SrcSandbox = srcSandbox

		t.Run("set files on file input in shadow DOM", func(t *testing.T) {
			err := SetFiles(ctx, "#host >>> #file-input", []string{filepath.Base(tmpFile.Name())})
			if err != nil {
				t.Fatalf("SetFiles() error = %v", err)
			}

			time.Sleep(200 * time.Millisecond)

			fileInfo, err := GetElementText(page.Ctx(), "#host >>> #file-info")
			if err != nil {
				t.Fatalf("GetElementText(#file-info) error: %v", err)
			}

			if fileInfo == "No files" || fileInfo == "" {
				t.Errorf("expected file info to show filename, got %q", fileInfo)
			}
		})

		t.Run("set files on nonexistent element in shadow DOM", func(t *testing.T) {
			err := SetFiles(ctx, "#host >>> #nonexistent", []string{filepath.Base(tmpFile.Name())})
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}
