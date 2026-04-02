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
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestSetCursorPosition(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		t.Run("set cursor at beginning", func(t *testing.T) {
			err := SetCursorPosition(ctx, "#editor", 0)
			if err != nil {
				t.Errorf("SetCursorPosition() error = %v", err)
			}

			position, err := GetCursorPosition(ctx, "#editor")
			if err != nil {
				t.Fatalf("GetCursorPosition() error = %v", err)
			}
			if position != 0 {
				t.Errorf("cursor position = %d, want 0", position)
			}
		})

		t.Run("set cursor in middle", func(t *testing.T) {
			err := SetCursorPosition(ctx, "#editor", 5)
			if err != nil {
				t.Errorf("SetCursorPosition() error = %v", err)
			}

			position, err := GetCursorPosition(ctx, "#editor")
			if err != nil {
				t.Fatalf("GetCursorPosition() error = %v", err)
			}
			if position != 5 {
				t.Errorf("cursor position = %d, want 5", position)
			}
		})

		t.Run("set cursor at end", func(t *testing.T) {
			err := SetCursorPosition(ctx, "#editor", 11)
			if err != nil {
				t.Errorf("SetCursorPosition() error = %v", err)
			}
		})
	})
}

func TestGetCursorPosition(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		err = SetCursorPosition(ctx, "#editor", 5)
		if err != nil {
			t.Fatalf("SetCursorPosition() error = %v", err)
		}

		t.Run("get valid cursor position", func(t *testing.T) {
			position, err := GetCursorPosition(ctx, "#editor")
			if err != nil {
				t.Errorf("GetCursorPosition() error = %v", err)
			}
			if position != 5 {
				t.Errorf("position = %d, want 5", position)
			}
		})
	})
}

func TestSetSelection(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		t.Run("select text range", func(t *testing.T) {
			err := SetSelection(ctx, "#editor", 0, 5)
			if err != nil {
				t.Errorf("SetSelection() error = %v", err)
			}

			start, end, err := GetSelection(ctx, "#editor")
			if err != nil {
				t.Fatalf("GetSelection() error = %v", err)
			}
			if start != 0 || end != 5 {
				t.Errorf("selection = (%d, %d), want (0, 5)", start, end)
			}
		})

		t.Run("select word in middle", func(t *testing.T) {
			err := SetSelection(ctx, "#editor", 6, 11)
			if err != nil {
				t.Errorf("SetSelection() error = %v", err)
			}

			start, end, err := GetSelection(ctx, "#editor")
			if err != nil {
				t.Fatalf("GetSelection() error = %v", err)
			}
			if start != 6 || end != 11 {
				t.Errorf("selection = (%d, %d), want (6, 11)", start, end)
			}
		})
	})
}

func TestGetSelection(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		err = SetSelection(ctx, "#editor", 2, 8)
		if err != nil {
			t.Fatalf("SetSelection() error = %v", err)
		}

		t.Run("get selection range", func(t *testing.T) {
			start, end, err := GetSelection(ctx, "#editor")
			if err != nil {
				t.Errorf("GetSelection() error = %v", err)
			}
			if start != 2 || end != 8 {
				t.Errorf("selection = (%d, %d), want (2, 8)", start, end)
			}
		})
	})
}

func TestSelectAll(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		t.Run("select all content", func(t *testing.T) {
			err := SelectAll(ctx, "#editor")
			if err != nil {
				t.Errorf("SelectAll() error = %v", err)
			}

		})
	})
}

func TestCollapseSelection(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		t.Run("collapse to start", func(t *testing.T) {

			err := SetSelection(ctx, "#editor", 2, 8)
			if err != nil {
				t.Fatalf("SetSelection() error = %v", err)
			}

			err = CollapseSelection(ctx, false)
			if err != nil {
				t.Errorf("CollapseSelection(false) error = %v", err)
			}

			position, err := GetCursorPosition(ctx, "#editor")
			if err != nil {
				t.Fatalf("GetCursorPosition() error = %v", err)
			}
			if position != 2 {
				t.Errorf("cursor position = %d, want 2", position)
			}
		})

		t.Run("collapse to end", func(t *testing.T) {

			err := SetSelection(ctx, "#editor", 2, 8)
			if err != nil {
				t.Fatalf("SetSelection() error = %v", err)
			}

			err = CollapseSelection(ctx, true)
			if err != nil {
				t.Errorf("CollapseSelection(true) error = %v", err)
			}

			position, err := GetCursorPosition(ctx, "#editor")
			if err != nil {
				t.Fatalf("GetCursorPosition() error = %v", err)
			}
			if position != 8 {
				t.Errorf("cursor position = %d, want 8", position)
			}
		})
	})
}

func TestSelectionErrors(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("nonexistent element", func(t *testing.T) {
			_, err := GetCursorPosition(ctx, "#nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})

		t.Run("set cursor on nonexistent element", func(t *testing.T) {
			err := SetCursorPosition(ctx, "#nonexistent", 5)
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})

		t.Run("set selection on nonexistent element", func(t *testing.T) {
			err := SetSelection(ctx, "#nonexistent", 0, 5)
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})

		t.Run("select all on nonexistent element", func(t *testing.T) {
			err := SelectAll(ctx, "#nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent element")
			}
		})
	})
}

func TestPlaceCursorInElement(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLInlineElements)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		t.Run("place cursor in bold element", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#editor", "#bold")
			if err != nil {
				t.Fatalf("PlaceCursorInElement() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			result, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if !containsString(result, "inside:bold") {
				t.Errorf("expected cursor inside bold, got %q", result)
			}
		})

		t.Run("place cursor in italic element", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#editor", "#italic")
			if err != nil {
				t.Fatalf("PlaceCursorInElement() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			result, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if !containsString(result, "inside:italic") {
				t.Errorf("expected cursor inside italic, got %q", result)
			}
		})

		t.Run("nonexistent child returns error", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#editor", "#nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent child element")
			}
		})

		t.Run("nonexistent parent returns error", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#nonexistent", "#bold")
			if err == nil {
				t.Error("expected error for nonexistent parent element")
			}
		})
	})
}

func TestPlaceCursorInElementInShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOMInlineElements)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		err := Focus(ctx, "#host >>> #editor")
		if err != nil {
			t.Fatalf("focusing shadow DOM editor: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		t.Run("place cursor in bold element in shadow DOM", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#host >>> #editor", "#bold")
			if err != nil {
				t.Fatalf("PlaceCursorInElement() error = %v", err)
			}

			time.Sleep(150 * time.Millisecond)

			result, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if !containsString(result, "inside:bold") {
				t.Errorf("expected cursor inside bold in shadow DOM, got %q", result)
			}
		})

		t.Run("place cursor in italic element in shadow DOM", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#host >>> #editor", "#italic")
			if err != nil {
				t.Fatalf("PlaceCursorInElement() error = %v", err)
			}

			time.Sleep(150 * time.Millisecond)

			result, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if !containsString(result, "inside:italic") {
				t.Errorf("expected cursor inside italic in shadow DOM, got %q", result)
			}
		})

		t.Run("nonexistent child in shadow DOM returns error", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#host >>> #editor", "#nonexistent")
			if err == nil {
				t.Error("expected error for nonexistent child element in shadow DOM")
			}
		})

		t.Run("nonexistent host returns error", func(t *testing.T) {
			err := PlaceCursorInElement(ctx, "#invalid-host >>> #editor", "#bold")
			if err == nil {
				t.Error("expected error for invalid shadow DOM host")
			}
		})
	})
}
