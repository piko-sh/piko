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
	"slices"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestNewBrowser(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	t.Run("creates browser with default options", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		browser, err := NewBrowser(opts)
		if err != nil {
			t.Fatalf("NewBrowser() error = %v", err)
		}
		defer browser.Close()

		if browser.BrowserCtx() == nil {
			t.Error("browser context should not be nil")
		}
	})

	t.Run("creates browser in headless mode", func(t *testing.T) {
		opts := BrowserOptions{Headless: true}
		browser, err := NewBrowser(opts)
		if err != nil {
			t.Fatalf("NewBrowser(headless=true) error = %v", err)
		}
		defer browser.Close()

		if !browser.headless {
			t.Error("browser should be in headless mode")
		}
	})
}

func TestBrowser_NewIncognitoPage(t *testing.T) {
	t.Parallel()
	browser := requireBrowser(t)

	t.Run("creates incognito page successfully", func(t *testing.T) {
		page, err := browser.NewIncognitoPage()
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}
		defer func() { _ = page.Close() }()

		if page.Ctx == nil {
			t.Error("page context should not be nil")
		}
	})

	t.Run("multiple pages are isolated", func(t *testing.T) {
		page1, err := browser.NewIncognitoPage()
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}
		defer func() { _ = page1.Close() }()

		page2, err := browser.NewIncognitoPage()
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}
		defer func() { _ = page2.Close() }()

		if page1.Ctx == page2.Ctx {
			t.Error("pages should have different contexts")
		}
	})
}

func TestIncognitoPage_Close(t *testing.T) {
	t.Parallel()
	browser := requireBrowser(t)

	t.Run("close does not error", func(t *testing.T) {
		page, err := browser.NewIncognitoPage()
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}

		err = page.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("double close does not panic", func(t *testing.T) {
		page, err := browser.NewIncognitoPage()
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}

		_ = page.Close()
		_ = page.Close()
	})
}

func TestIncognitoPage_CloseContext(t *testing.T) {
	t.Parallel()
	browser := requireBrowser(t)

	t.Run("CloseContext disposes browser context", func(t *testing.T) {
		page, err := browser.NewIncognitoPage()
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}

		page.Cancel()

		err = page.CloseContext()
		if err != nil {
			t.Errorf("CloseContext() error = %v", err)
		}
	})

	t.Run("CloseContext is idempotent", func(t *testing.T) {
		page, err := browser.NewIncognitoPage()
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}
		page.Cancel()

		_ = page.CloseContext()
		err = page.CloseContext()
		if err != nil {
			t.Errorf("second CloseContext() error = %v", err)
		}
	})
}

func TestPageHelper_Navigate(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPageNoNav(t, func(t *testing.T, page *PageHelper) {
		err := page.Navigate(server.URL)
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		var title string
		err = chromedp.Run(page.Ctx(), chromedp.Title(&title))
		if err != nil {
			t.Fatalf("getting title: %v", err)
		}
		if title != "Test" {
			t.Errorf("title = %q, want %q", title, "Test")
		}
	})
}

func TestPageHelper_ConsoleCapture(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.log('test log message')`, nil),
		)
		if err != nil {
			t.Fatalf("evaluating console.log: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		logs := page.ConsoleLogs()
		if !slices.Contains(logs, "test log message") {
			t.Errorf("expected 'test log message' in logs, got %v", logs)
		}
	})
}

func TestPageHelper_ConsoleLogsWithLevel(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.error('test error message')`, nil),
		)
		if err != nil {
			t.Fatalf("evaluating console.error: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		logs := page.ConsoleLogsWithLevel()
		found := false
		for _, log := range logs {
			if log.Level == "error" && log.Message == "test error message" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error level log with 'test error message', got %v", logs)
		}
	})
}

func TestPageHelper_ConsoleLogsByLevel(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.warn('test warn message')`, nil),
		)
		if err != nil {
			t.Fatalf("evaluating console.warn: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		warnLogs := page.ConsoleLogsByLevel("warn")
		if len(warnLogs) == 0 {
			t.Error("expected at least one warn level log")
		}
	})
}

func TestPageHelper_HasConsoleErrors(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		if page.HasConsoleErrors() {
			t.Error("should not have console errors initially")
		}

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.error('test error')`, nil),
		)
		if err != nil {
			t.Fatalf("evaluating console.error: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		if !page.HasConsoleErrors() {
			t.Error("should have console errors after emitting error")
		}
	})
}

func TestPageHelper_ConsoleErrors(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.error('test error message')`, nil),
		)
		if err != nil {
			t.Fatalf("evaluating console.error: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		errors := page.ConsoleErrors()
		if len(errors) == 0 {
			t.Error("expected at least one error log")
			return
		}

		if errors[0].Level != "error" {
			t.Errorf("expected error level, got %q", errors[0].Level)
		}
	})
}

func TestPageHelper_ClearConsoleLogs(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.log('before clear')`, nil),
		)
		if err != nil {
			t.Fatalf("evaluating console.log: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		logs := page.ConsoleLogs()
		if len(logs) == 0 {
			t.Fatal("expected at least one log before clearing")
		}

		page.ClearConsoleLogs()

		logs = page.ConsoleLogs()
		if len(logs) != 0 {
			t.Errorf("expected empty logs after clear, got %v", logs)
		}
	})
}

func TestWaitStable(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html><body>
<div id="content">Initial</div>
<script>
setTimeout(function() {
    document.getElementById('content').textContent = 'Updated';
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPageNoNav(t, func(t *testing.T, page *PageHelper) {

		err := chromedp.Run(page.Ctx(),
			chromedp.Navigate(server.URL),
			chromedp.WaitReady("body"),
		)
		if err != nil {
			t.Fatalf("navigation error: %v", err)
		}

		err = WaitStable(page.Ctx(), 200*time.Millisecond)
		if err != nil {
			t.Fatalf("WaitStable() error = %v", err)
		}

		var text string
		err = chromedp.Run(page.Ctx(),
			chromedp.Text("#content", &text, chromedp.ByID),
		)
		if err != nil {
			t.Fatalf("getting text: %v", err)
		}
		if text != "Updated" {
			t.Errorf("text = %q, want %q", text, "Updated")
		}
	})
}

func TestNewPageHelper(t *testing.T) {
	t.Parallel()
	browser := requireBrowser(t)

	incognito, err := browser.NewIncognitoPage()
	if err != nil {
		t.Fatalf("NewIncognitoPage() error = %v", err)
	}
	defer func() { _ = incognito.Close() }()

	page := NewPageHelper(incognito.Ctx)
	defer page.Close()

	if page.Ctx() == nil {
		t.Error("page context should not be nil")
	}

	logs := page.ConsoleLogs()
	if len(logs) != 0 {
		t.Errorf("expected empty console logs, got %d", len(logs))
	}
}

func TestPageHelper_Close(t *testing.T) {
	t.Parallel()
	browser := requireBrowser(t)

	incognito, err := browser.NewIncognitoPage()
	if err != nil {
		t.Fatalf("NewIncognitoPage() error = %v", err)
	}

	page := NewPageHelper(incognito.Ctx)

	done := make(chan struct{})
	go func() {
		page.Close()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(5 * time.Second):
		t.Error("Close() timed out")
	}

	_ = incognito.Close()
}
