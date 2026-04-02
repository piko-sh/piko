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
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestGetFormData(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLForm)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("returns form field values", func(t *testing.T) {
			data, err := GetFormData(page.Ctx(), "#test-form")
			if err != nil {
				t.Fatalf("GetFormData() error = %v", err)
			}

			want := map[string]any{
				"username": "alice",
				"email":    "alice@example.com",
				"token":    "abc123",
			}

			if len(data) != len(want) {
				t.Fatalf("GetFormData() returned %d fields, want %d", len(data), len(want))
			}
			for k, v := range want {
				if data[k] != v {
					t.Errorf("field %q = %v, want %v", k, data[k], v)
				}
			}
		})

		t.Run("returns empty map for empty form", func(t *testing.T) {
			data, err := GetFormData(page.Ctx(), "#empty-form")
			if err != nil {
				t.Fatalf("GetFormData() error = %v", err)
			}
			if len(data) != 0 {
				t.Errorf("GetFormData(empty) returned %d fields, want 0", len(data))
			}
		})

		t.Run("finds ancestor form from child element", func(t *testing.T) {
			data, err := GetFormData(page.Ctx(), "#test-form input[name='username']")
			if err != nil {
				t.Fatalf("GetFormData(child) error = %v", err)
			}
			if data["username"] != "alice" {
				t.Errorf("username = %v, want alice", data["username"])
			}
		})

		t.Run("returns error for missing element", func(t *testing.T) {
			_, err := GetFormData(page.Ctx(), "#nonexistent")
			if err == nil {
				t.Error("expected error for missing element")
			}
		})
	})
}

func TestGetFormData_DynamicValues(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLForm)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		err := chromedp.Run(page.Ctx(), chromedp.Evaluate(
			`document.querySelector('input[name="username"]').value = "bob"`, nil))
		if err != nil {
			t.Fatalf("setting value: %v", err)
		}

		data, err := GetFormData(page.Ctx(), "#test-form")
		if err != nil {
			t.Fatalf("GetFormData() error = %v", err)
		}
		if data["username"] != "bob" {
			t.Errorf("username = %v, want bob", data["username"])
		}
	})
}

func TestListenForEvent_and_GetEventDetail(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEventListener)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("captures custom event detail", func(t *testing.T) {
			ctx := page.Ctx()

			err := ListenForEvent(ctx, "#target", "greeting")
			if err != nil {
				t.Fatalf("ListenForEvent() error = %v", err)
			}

			detail, err := GetEventDetail(ctx, "greeting")
			if err != nil {
				t.Fatalf("GetEventDetail() error = %v", err)
			}
			if detail != nil {
				t.Errorf("detail before fire = %v, want nil", detail)
			}

			err = chromedp.Run(ctx, chromedp.Click("#fire"))
			if err != nil {
				t.Fatalf("clicking fire: %v", err)
			}

			var receivedDetail any
			deadline := time.After(5 * time.Second)
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for {
				receivedDetail, err = GetEventDetail(ctx, "greeting")
				if err != nil {
					t.Fatalf("GetEventDetail() error = %v", err)
				}
				if receivedDetail != nil {
					break
				}
				select {
				case <-deadline:
					t.Fatal("timed out waiting for event detail")
				case <-ticker.C:
				}
			}

			detailMap, ok := receivedDetail.(map[string]any)
			if !ok {
				t.Fatalf("detail type = %T, want map[string]any", receivedDetail)
			}
			if detailMap["message"] != "hello" {
				t.Errorf("detail.message = %v, want hello", detailMap["message"])
			}
			if detailMap["count"] != float64(42) {
				t.Errorf("detail.count = %v, want 42", detailMap["count"])
			}
		})

		t.Run("returns error for missing element", func(t *testing.T) {
			err := ListenForEvent(page.Ctx(), "#nonexistent", "test")
			if err == nil {
				t.Error("expected error for missing element")
			}
		})
	})
}

func TestListenForEvent_ShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowEventListener)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := page.Ctx()

		err := ListenForEvent(ctx, "#host", "shadow-click")
		if err != nil {
			t.Fatalf("ListenForEvent() error = %v", err)
		}

		err = chromedp.Run(ctx, chromedp.Click("#host >>> #inner-btn", chromedp.ByQuery))
		if err != nil {

			err = chromedp.Run(ctx, chromedp.Evaluate(`
				document.getElementById('host').shadowRoot.getElementById('inner-btn').click()
			`, nil))
			if err != nil {
				t.Fatalf("clicking inner-btn: %v", err)
			}
		}

		var detail any
		deadline := time.After(5 * time.Second)
		for {
			detail, err = GetEventDetail(ctx, "shadow-click")
			if err != nil {
				t.Fatalf("GetEventDetail() error = %v", err)
			}
			if detail != nil {
				break
			}
			select {
			case <-deadline:
				t.Fatal("timed out waiting for shadow-click event")
			case <-time.After(50 * time.Millisecond):
			}
		}

		detailMap, ok := detail.(map[string]any)
		if !ok {
			t.Fatalf("detail type = %T, want map[string]any", detail)
		}
		if detailMap["from"] != "shadow" {
			t.Errorf("detail.from = %v, want shadow", detailMap["from"])
		}
	})
}

func TestGetEventDetail_UnlistenedEvent(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEventListener)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		detail, err := GetEventDetail(page.Ctx(), "never-listened")
		if err != nil {
			t.Fatalf("GetEventDetail() error = %v", err)
		}
		if detail != nil {
			t.Errorf("detail = %v, want nil for unlistened event", detail)
		}
	})
}

func TestListenForEvent_ContextCancellation(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEventListener)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx, cancel := context.WithCancelCause(page.Ctx())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		err := ListenForEvent(ctx, "#target", "test")
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})
}
