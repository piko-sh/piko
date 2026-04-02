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
)

func TestFindElement(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		nodes, err := FindElement(page.Ctx(), "#target")
		if err != nil {
			t.Fatalf("FindElement() error = %v", err)
		}
		if len(nodes) == 0 {
			t.Error("expected to find element")
		}
	})
}

func TestFindElement_NotFound(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		_, err := FindElement(page.Ctx(), "#nonexistent")

		if err == nil {
			t.Error("expected error for nonexistent element")
		}
	})
}

func TestFindElement_ShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOM)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		nodes, err := FindElement(page.Ctx(), "#host >>> #inner")
		if err != nil {
			t.Fatalf("FindElement(shadow) error = %v", err)
		}
		if len(nodes) == 0 {
			t.Error("expected to find shadow DOM element")
		}
	})
}

func TestFindElements(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLMultipleElements)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		nodes, err := FindElements(page.Ctx(), ".item")
		if err != nil {
			t.Fatalf("FindElements() error = %v", err)
		}
		if len(nodes) != 5 {
			t.Errorf("expected 5 elements, got %d", len(nodes))
		}
	})
}

func TestElementExists(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		exists := ElementExists(page.Ctx(), "#target")
		if !exists {
			t.Error("expected element to exist")
		}

		notExists := ElementExists(page.Ctx(), "#nonexistent")
		if notExists {
			t.Error("expected element not to exist")
		}
	})
}

func TestGetElementText(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		text, err := GetElementText(page.Ctx(), "#target")
		if err != nil {
			t.Fatalf("GetElementText() error = %v", err)
		}
		text = strings.TrimSpace(text)
		if text != "Content Text" {
			t.Errorf("text = %q, want %q", text, "Content Text")
		}
	})
}

func TestGetElementAttribute(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("existing attribute", func(t *testing.T) {
			attr, err := GetElementAttribute(page.Ctx(), "#target", "data-custom")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr == nil {
				t.Fatal("expected attribute to exist")
			}
			if *attr != "custom-value" {
				t.Errorf("attribute = %q, want %q", *attr, "custom-value")
			}
		})

		t.Run("nonexistent attribute", func(t *testing.T) {
			attr, err := GetElementAttribute(page.Ctx(), "#target", "nonexistent")
			if err != nil {
				t.Fatalf("GetElementAttribute() error = %v", err)
			}
			if attr != nil {
				t.Errorf("expected nil for nonexistent attribute, got %q", *attr)
			}
		})
	})
}

func TestGetElementHTML(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		html, err := GetElementHTML(page.Ctx(), "#target")
		if err != nil {
			t.Fatalf("GetElementHTML() error = %v", err)
		}
		if !strings.Contains(html, "Content Text") {
			t.Errorf("expected HTML to contain 'Content Text', got %q", html)
		}
	})
}

func TestGetElementValue(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		value, err := GetElementValue(page.Ctx(), "#enabled-input")
		if err != nil {
			t.Fatalf("GetElementValue() error = %v", err)
		}
		if value != "enabled" {
			t.Errorf("value = %q, want %q", value, "enabled")
		}
	})
}

func TestIsElementVisible(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("visible element", func(t *testing.T) {
			visible, err := IsElementVisible(page.Ctx(), "#visible")
			if err != nil {
				t.Fatalf("IsElementVisible() error = %v", err)
			}
			if !visible {
				t.Error("expected element to be visible")
			}
		})

		t.Run("hidden by display", func(t *testing.T) {
			visible, err := IsElementVisible(page.Ctx(), "#hidden-display")
			if err != nil {
				t.Fatalf("IsElementVisible() error = %v", err)
			}
			if visible {
				t.Error("expected element to be hidden")
			}
		})

		t.Run("hidden by visibility", func(t *testing.T) {
			visible, err := IsElementVisible(page.Ctx(), "#hidden-visibility")
			if err != nil {
				t.Fatalf("IsElementVisible() error = %v", err)
			}
			if visible {
				t.Error("expected element to be hidden by visibility")
			}
		})
	})
}

func TestIsElementEnabled(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("enabled input", func(t *testing.T) {
			enabled, err := IsElementEnabled(page.Ctx(), "#enabled-input")
			if err != nil {
				t.Fatalf("IsElementEnabled() error = %v", err)
			}
			if !enabled {
				t.Error("expected input to be enabled")
			}
		})

		t.Run("disabled input", func(t *testing.T) {
			enabled, err := IsElementEnabled(page.Ctx(), "#disabled-input")
			if err != nil {
				t.Fatalf("IsElementEnabled() error = %v", err)
			}
			if enabled {
				t.Error("expected input to be disabled")
			}
		})
	})
}

func TestIsElementChecked(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCheckbox)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("unchecked checkbox", func(t *testing.T) {
			checked, err := IsElementChecked(page.Ctx(), "#checkbox1")
			if err != nil {
				t.Fatalf("IsElementChecked() error = %v", err)
			}
			if checked {
				t.Error("expected checkbox1 to be unchecked")
			}
		})

		t.Run("checked checkbox", func(t *testing.T) {
			checked, err := IsElementChecked(page.Ctx(), "#checkbox2")
			if err != nil {
				t.Fatalf("IsElementChecked() error = %v", err)
			}
			if !checked {
				t.Error("expected checkbox2 to be checked")
			}
		})
	})
}

func TestScrollIntoView(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLScroll)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		err := ScrollIntoView(page.Ctx(), "#target")
		if err != nil {
			t.Fatalf("ScrollIntoView() error = %v", err)
		}

	})
}

func TestGetAllAttributes(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		attrs, err := GetAllAttributes(page.Ctx(), "#target")
		if err != nil {
			t.Fatalf("GetAllAttributes() error = %v", err)
		}

		if attrs["id"] != "target" {
			t.Errorf("id = %q, want %q", attrs["id"], "target")
		}
		if attrs["data-custom"] != "custom-value" {
			t.Errorf("data-custom = %q, want %q", attrs["data-custom"], "custom-value")
		}
		if attrs["title"] != "Element Title" {
			t.Errorf("title = %q, want %q", attrs["title"], "Element Title")
		}
	})
}

func TestSetElementAttribute(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		err := SetElementAttribute(page.Ctx(), "#target", "data-new", "new-value")
		if err != nil {
			t.Fatalf("SetElementAttribute() error = %v", err)
		}

		attr, err := GetElementAttribute(page.Ctx(), "#target", "data-new")
		if err != nil {
			t.Fatalf("GetElementAttribute() error = %v", err)
		}
		if attr == nil || *attr != "new-value" {
			t.Error("expected attribute to be set")
		}
	})
}

func TestRemoveElementAttribute(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		attr, _ := GetElementAttribute(page.Ctx(), "#target", "data-custom")
		if attr == nil {
			t.Fatal("expected attribute to exist before removal")
		}

		err := RemoveElementAttribute(page.Ctx(), "#target", "data-custom")
		if err != nil {
			t.Fatalf("RemoveElementAttribute() error = %v", err)
		}

		attr, err = GetElementAttribute(page.Ctx(), "#target", "data-custom")
		if err != nil {
			t.Fatalf("GetElementAttribute() error = %v", err)
		}
		if attr != nil {
			t.Errorf("expected attribute to be removed, got %q", *attr)
		}
	})
}

func TestGetElementDimensions(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target" style="width: 200px; height: 100px; position: absolute; top: 50px; left: 50px;">
Box
</div>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		dims, err := GetElementDimensions(page.Ctx(), "#target")
		if err != nil {
			t.Fatalf("GetElementDimensions() error = %v", err)
		}

		if dims == nil {
			t.Fatal("expected dimensions, got nil")
		}

		if dims.Width != 200 {
			t.Errorf("width = %v, want 200", dims.Width)
		}
		if dims.Height != 100 {
			t.Errorf("height = %v, want 100", dims.Height)
		}
	})
}

func TestEvalOnElement(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		result, err := EvalOnElement(page.Ctx(), "#target", "function() { return this.tagName; }")
		if err != nil {
			t.Fatalf("EvalOnElement() error = %v", err)
		}

		tagName, ok := result.(string)
		if !ok {
			t.Fatalf("expected string result, got %T", result)
		}
		if tagName != "DIV" {
			t.Errorf("tagName = %q, want %q", tagName, "DIV")
		}
	})
}
