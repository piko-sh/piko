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

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/safedisk"
)

func TestClick(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLButton)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Click(actx, "#btn")
		if err != nil {
			t.Fatalf("Click() error = %v", err)
		}

		var text string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#result", &text, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting result text: %v", err)
		}
		if text != "clicked" {
			t.Errorf("result text = %q, want %q", text, "clicked")
		}
	})
}

func TestDoubleClick(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDoubleClickButton)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := DoubleClick(actx, "#btn")
		if err != nil {
			t.Fatalf("DoubleClick() error = %v", err)
		}

		var text string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#result", &text, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting result text: %v", err)
		}
		if text != "double-clicked" {
			t.Errorf("result text = %q, want %q", text, "double-clicked")
		}
	})
}

func TestRightClick(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLRightClickButton)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := RightClick(actx, "#btn")
		if err != nil {
			t.Fatalf("RightClick() error = %v", err)
		}

		var text string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#result", &text, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting result text: %v", err)
		}
		if text != "right-clicked" {
			t.Errorf("result text = %q, want %q", text, "right-clicked")
		}
	})
}

func TestFill(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Fill(actx, "#input", "hello world")
		if err != nil {
			t.Fatalf("Fill() error = %v", err)
		}

		var value string
		err = chromedp.Run(page.Ctx(), chromedp.Value("#input", &value, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting input value: %v", err)
		}
		if value != "hello world" {
			t.Errorf("input value = %q, want %q", value, "hello world")
		}
	})
}

func TestClear(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html><body>
<input type="text" id="input" value="prefilled" />
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		var valueBefore string
		err := chromedp.Run(page.Ctx(), chromedp.Value("#input", &valueBefore, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting input value: %v", err)
		}
		if valueBefore != "prefilled" {
			t.Fatalf("expected prefilled value, got %q", valueBefore)
		}

		err = Clear(actx, "#input")
		if err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		var valueAfter string
		err = chromedp.Run(page.Ctx(), chromedp.Value("#input", &valueAfter, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting input value: %v", err)
		}
		if valueAfter != "" {
			t.Errorf("input value = %q, want empty", valueAfter)
		}
	})
}

func TestPress(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLKeyboard)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Focus(actx, "#input")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		err = Press(actx, "Enter")
		if err != nil {
			t.Fatalf("Press(Enter) error = %v", err)
		}

		var keylog string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#keylog", &keylog, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting keylog: %v", err)
		}
		if keylog != "Enter" {
			t.Errorf("keylog = %q, want %q", keylog, "Enter")
		}
	})
}

func TestPress_WithModifiers(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLKeyboard)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Focus(actx, "#input")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		err = Press(actx, "Control+b")
		if err != nil {
			t.Fatalf("Press(Control+b) error = %v", err)
		}

		var keylog string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#keylog", &keylog, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting keylog: %v", err)
		}

		if keylog != "Ctrl+B" && keylog != "Ctrl+b" {
			t.Errorf("keylog = %q, want %q or %q", keylog, "Ctrl+b", "Ctrl+B")
		}
	})
}

func TestType(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Focus(actx, "#input")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		err = Type(actx, "hello")
		if err != nil {
			t.Fatalf("Type() error = %v", err)
		}

		var value string
		err = chromedp.Run(page.Ctx(), chromedp.Value("#input", &value, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting input value: %v", err)
		}
		if value != "hello" {
			t.Errorf("input value = %q, want %q", value, "hello")
		}
	})
}

func TestCheck(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCheckbox)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Check(actx, "#checkbox1")
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}

		var checked bool
		err = chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`document.getElementById('checkbox1').checked`, &checked),
		)
		if err != nil {
			t.Fatalf("getting checked state: %v", err)
		}
		if !checked {
			t.Error("checkbox should be checked")
		}
	})
}

func TestUncheck(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCheckbox)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Uncheck(actx, "#checkbox2")
		if err != nil {
			t.Fatalf("Uncheck() error = %v", err)
		}

		var checked bool
		err = chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`document.getElementById('checkbox2').checked`, &checked),
		)
		if err != nil {
			t.Fatalf("getting checked state: %v", err)
		}
		if checked {
			t.Error("checkbox should be unchecked")
		}
	})
}

func TestFocus(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFocus)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Focus(actx, "#input1")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var focusLog string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#focus-log", &focusLog, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting focus log: %v", err)
		}
		if focusLog != "input1-focused" {
			t.Errorf("focus log = %q, want %q", focusLog, "input1-focused")
		}
	})
}

func TestBlur(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFocus)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Focus(actx, "#input1")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		err = Blur(actx, "#input1")
		if err != nil {
			t.Fatalf("Blur() error = %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var focusLog string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#focus-log", &focusLog, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting focus log: %v", err)
		}
		if focusLog != "input1-blurred" {
			t.Errorf("focus log = %q, want %q", focusLog, "input1-blurred")
		}
	})
}

func TestHover(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLHover)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Hover(actx, "#hover-target")
		if err != nil {
			t.Fatalf("Hover() error = %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var result string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#hover-result", &result, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting hover result: %v", err)
		}
		if result != "hovered" {
			t.Errorf("hover result = %q, want %q", result, "hovered")
		}
	})
}

func TestWaitForSelector(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html><body>
<div id="container"></div>
<script>
setTimeout(function() {
    document.getElementById('container').innerHTML = '<span id="delayed">Appeared</span>';
}, 200);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForSelector(actx, "#delayed", 5*time.Second)
		if err != nil {
			t.Fatalf("WaitForSelector() error = %v", err)
		}

		var text string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#delayed", &text, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting text: %v", err)
		}
		if text != "Appeared" {
			t.Errorf("text = %q, want %q", text, "Appeared")
		}
	})
}

func TestWaitForVisible(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html><body>
<div id="target" style="display: none;">Hidden</div>
<script>
setTimeout(function() {
    document.getElementById('target').style.display = 'block';
}, 200);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForVisible(actx, "#target", 5*time.Second)
		if err != nil {
			t.Fatalf("WaitForVisible() error = %v", err)
		}
	})
}

func TestWaitForNotVisible(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html><body>
<div id="target">Visible</div>
<script>
setTimeout(function() {
    document.getElementById('target').style.display = 'none';
}, 200);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForNotVisible(actx, "#target", 5*time.Second)
		if err != nil {
			t.Fatalf("WaitForNotVisible() error = %v", err)
		}
	})
}

func TestNavigate(t *testing.T) {
	t.Parallel()
	server := newTestServerWithRoutes(map[string]string{
		"/":     `<!DOCTYPE html><html><head><title>Home</title></head><body></body></html>`,
		"/page": `<!DOCTYPE html><html><head><title>Page</title></head><body></body></html>`,
	})
	defer server.Close()

	withTestPageNoNav(t, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)
		actx.ServerURL = server.URL

		err := Navigate(actx, "/page")
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		var title string
		err = chromedp.Run(page.Ctx(), chromedp.Title(&title))
		if err != nil {
			t.Fatalf("getting title: %v", err)
		}
		if title != "Page" {
			t.Errorf("title = %q, want %q", title, "Page")
		}
	})
}

func TestGetTitle(t *testing.T) {
	t.Parallel()
	server := newTestServer(`<!DOCTYPE html><html><head><title>Test Title</title></head><body></body></html>`)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		title, err := GetTitle(actx)
		if err != nil {
			t.Fatalf("GetTitle() error = %v", err)
		}
		if title != "Test Title" {
			t.Errorf("title = %q, want %q", title, "Test Title")
		}
	})
}

func TestGetURL(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		url, err := GetURL(actx)
		if err != nil {
			t.Fatalf("GetURL() error = %v", err)
		}

		wantURL := server.URL
		if url != wantURL && url != wantURL+"/" {
			t.Errorf("url = %q, want %q or %q/", url, wantURL, wantURL)
		}
	})
}

func TestDispatchEvent(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCustomEvent)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		detail := map[string]any{"key": "value", "count": 42}
		err := DispatchEvent(actx, "#target", "custom-event", detail)
		if err != nil {
			t.Fatalf("DispatchEvent() error = %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var eventData string
		err = chromedp.Run(page.Ctx(), chromedp.Text("#event-data", &eventData, chromedp.ByID))
		if err != nil {
			t.Fatalf("getting event data: %v", err)
		}

		if eventData == "" {
			t.Error("expected event data to be captured")
		}
	})
}

func TestScroll(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLScroll)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Scroll(actx, "window", "1000")
		if err != nil {
			t.Fatalf("Scroll() error = %v", err)
		}

		var scrollY float64
		err = chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`window.scrollY`, &scrollY),
		)
		if err != nil {
			t.Fatalf("getting scroll position: %v", err)
		}
		if scrollY < 500 {
			t.Errorf("expected page to scroll to ~1000, got %v", scrollY)
		}
	})
}

func TestWait(t *testing.T) {
	t.Parallel()
	start := time.Now()
	Wait(100)
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("Wait(100) elapsed = %v, want >= 100ms", elapsed)
	}
}

func TestWaitForText(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target">Initial</div>
<script>
setTimeout(function() {
    document.getElementById('target').textContent = 'Updated Content';
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("waits for text to appear", func(t *testing.T) {
			err := WaitForText(actx, "#target", "Updated Content", 5*time.Second)
			if err != nil {
				t.Errorf("WaitForText() error = %v", err)
			}
		})
	})
}

func TestWaitForEnabled(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<button id="btn" disabled>Button</button>
<script>
setTimeout(function() {
    document.getElementById('btn').disabled = false;
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("waits for element to become enabled", func(t *testing.T) {
			err := WaitForEnabled(actx, "#btn", 5*time.Second)
			if err != nil {
				t.Errorf("WaitForEnabled() error = %v", err)
			}
		})
	})
}

func TestWaitForDisabled(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<button id="btn">Button</button>
<script>
setTimeout(function() {
    document.getElementById('btn').disabled = true;
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("waits for element to become disabled", func(t *testing.T) {
			err := WaitForDisabled(actx, "#btn", 5*time.Second)
			if err != nil {
				t.Errorf("WaitForDisabled() error = %v", err)
			}
		})
	})
}

func TestWaitForNotPresent(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target">Temporary</div>
<script>
setTimeout(function() {
    document.getElementById('target').remove();
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("waits for element to be removed", func(t *testing.T) {
			err := WaitForNotPresent(actx, "#target", 5*time.Second)
			if err != nil {
				t.Errorf("WaitForNotPresent() error = %v", err)
			}
		})
	})
}

func TestEmulateDevice(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("emulates mobile device", func(t *testing.T) {
			params := DeviceParams{
				Width:  375,
				Height: 667,
				Scale:  2.0,
				Mobile: true,
			}
			err := EmulateDevice(actx, params)
			if err != nil {
				t.Errorf("EmulateDevice() error = %v", err)
			}
		})

		t.Run("default scale", func(t *testing.T) {
			params := DeviceParams{
				Width:  1024,
				Height: 768,
				Scale:  0,
			}
			err := EmulateDevice(actx, params)
			if err != nil {
				t.Errorf("EmulateDevice() error = %v", err)
			}
		})
	})
}

func TestResetEmulation(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		params := DeviceParams{
			Width:  375,
			Height: 667,
			Mobile: true,
		}
		err := EmulateDevice(actx, params)
		if err != nil {
			t.Fatalf("EmulateDevice() error = %v", err)
		}

		err = ResetEmulation(actx)
		if err != nil {
			t.Errorf("ResetEmulation() error = %v", err)
		}
	})
}

func TestEval(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target">Original</div>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("eval on window", func(t *testing.T) {
			err := Eval(actx, "window", "window.testVar = 'hello'")
			if err != nil {
				t.Errorf("Eval(window) error = %v", err)
			}
		})

		t.Run("eval on element", func(t *testing.T) {
			err := Eval(actx, "#target", "this.textContent = 'Modified'")
			if err != nil {
				t.Errorf("Eval(element) error = %v", err)
			}

			text, err := GetElementText(page.Ctx(), "#target")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "Modified" {
				t.Errorf("text = %q, want %q", text, "Modified")
			}
		})

		t.Run("eval with empty selector", func(t *testing.T) {
			err := Eval(actx, "", "document.body.style.backgroundColor = 'red'")
			if err != nil {
				t.Errorf("Eval(empty) error = %v", err)
			}
		})
	})
}

func TestStop(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html><body>
<div id="content">Loading...</div>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Stop(actx)
		if err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	})
}

func TestExecuteStep_BasicActions(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLButton)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("click action", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "click",
				Selector: "#btn",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(click) error = %v", err)
			}

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != "clicked" {
				t.Errorf("result = %q, want %q", text, "clicked")
			}
		})

		t.Run("wait action", func(t *testing.T) {
			step := &BrowserStep{
				Action: "wait",
				Value:  "50",
			}
			start := time.Now()
			err := ExecuteStep(actx, step)
			elapsed := time.Since(start)
			if err != nil {
				t.Errorf("ExecuteStep(wait) error = %v", err)
			}
			if elapsed < 50*time.Millisecond {
				t.Errorf("wait elapsed = %v, want >= 50ms", elapsed)
			}
		})
	})
}

func TestExecuteStep_FillAction(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "fill",
			Selector: "#input",
			Value:    "test input",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(fill) error = %v", err)
		}

		text, err := GetElementText(page.Ctx(), "#mirror")
		if err != nil {
			t.Fatalf("GetElementText() error = %v", err)
		}
		if text != "test input" {
			t.Errorf("mirror = %q, want %q", text, "test input")
		}
	})
}

func TestExecuteStep_WaitForSelector(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="container"></div>
<script>
setTimeout(function() {
    var div = document.createElement('div');
    div.id = 'dynamic';
    div.textContent = 'Dynamic';
    document.getElementById('container').appendChild(div);
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "waitForSelector",
			Selector: "#dynamic",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(waitForSelector) error = %v", err)
		}
	})
}

func TestExecuteStep_Comment(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action: "comment",
			Value:  "This is a comment and should do nothing",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(comment) error = %v", err)
		}
	})
}

func TestExecuteStep_MoreActions(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLButton)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("doubleClick", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "doubleClick",
				Selector: "#btn",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(doubleClick) error = %v", err)
			}
		})

		t.Run("hover", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "hover",
				Selector: "#btn",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(hover) error = %v", err)
			}
		})

		t.Run("rightClick", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "rightClick",
				Selector: "#btn",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(rightClick) error = %v", err)
			}
		})

		t.Run("focus", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "focus",
				Selector: "#btn",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(focus) error = %v", err)
			}
		})

		t.Run("blur", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "blur",
				Selector: "#btn",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(blur) error = %v", err)
			}
		})
	})
}

func TestExecuteStep_Navigate(t *testing.T) {
	t.Parallel()
	server1 := newTestServer(testHTMLEmpty)
	defer server1.Close()

	server2 := newTestServer(`<!DOCTYPE html><html><head><title>Page 2</title></head><body></body></html>`)
	defer server2.Close()

	withTestPage(t, server1.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action: "navigate",
			Value:  server2.URL,
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(navigate) error = %v", err)
		}

		title, err := GetTitle(actx)
		if err != nil {
			t.Fatalf("GetTitle() error = %v", err)
		}
		if title != "Page 2" {
			t.Errorf("title = %q, want %q", title, "Page 2")
		}
	})
}

func TestExecuteStep_CheckActions(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCheckbox)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("check", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "check",
				Selector: "#checkbox1",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(check) error = %v", err)
			}
		})

		t.Run("uncheck", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "uncheck",
				Selector: "#checkbox1",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(uncheck) error = %v", err)
			}
		})
	})
}

func TestExecuteStep_TypeAndPress(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "focus",
			Selector: "#input",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Fatalf("ExecuteStep(focus) error = %v", err)
		}

		t.Run("type", func(t *testing.T) {
			step := &BrowserStep{
				Action: "type",
				Value:  "hello",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(type) error = %v", err)
			}
		})

		t.Run("press", func(t *testing.T) {
			step := &BrowserStep{
				Action: "press",
				Value:  "Enter",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(press) error = %v", err)
			}
		})
	})
}

func TestExecuteStep_ClearAction(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<input type="text" id="input" value="prefilled" />
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "clear",
			Selector: "#input",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(clear) error = %v", err)
		}
	})
}

func TestExecuteStep_ScrollAction(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLScroll)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "scroll",
			Selector: "window",
			Value:    "500",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(scroll) error = %v", err)
		}
	})
}

func TestExecuteStep_WaitForVisible(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "waitForVisible",
			Selector: "#visible",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(waitForVisible) error = %v", err)
		}
	})
}

func TestExecuteStep_WaitForNotVisible(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "waitForNotVisible",
			Selector: "#hidden-display",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(waitForNotVisible) error = %v", err)
		}
	})
}

func TestExecuteStep_DispatchEvent(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLButton)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:    "dispatchEvent",
			Selector:  "#btn",
			EventName: "click",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(dispatchEvent) error = %v", err)
		}
	})
}

func TestExecuteAssertion_CheckText(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("checkText passes", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkText",
				Selector: "#target",
				Expected: "Content Text",
			}
			err := ExecuteAssertion(actx, step)
			if err != nil {
				t.Errorf("ExecuteAssertion(checkText) error = %v", err)
			}
		})

		t.Run("checkText fails", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "checkText",
				Selector: "#target",
				Expected: "Wrong Text",
			}
			err := ExecuteAssertion(actx, step)
			if err == nil {
				t.Error("expected error for checkText mismatch")
			}
		})
	})
}

func TestExecuteAssertion_CheckVisible(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "checkVisible",
			Selector: "#visible",
		}
		err := ExecuteAssertion(actx, step)
		if err != nil {
			t.Errorf("ExecuteAssertion(checkVisible) error = %v", err)
		}
	})
}

func TestExecuteAssertion_CheckHidden(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "checkHidden",
			Selector: "#hidden-display",
		}
		err := ExecuteAssertion(actx, step)
		if err != nil {
			t.Errorf("ExecuteAssertion(checkHidden) error = %v", err)
		}
	})
}

func TestExecuteAssertion_CheckAttribute(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:    "checkAttribute",
			Selector:  "#target",
			Attribute: "data-custom",
			Expected:  "custom-value",
		}
		err := ExecuteAssertion(actx, step)
		if err != nil {
			t.Errorf("ExecuteAssertion(checkAttribute) error = %v", err)
		}
	})
}

func TestExecuteAssertion_CheckElementCount(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLMultipleElements)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "checkElementCount",
			Selector: ".item",
			Expected: 5,
		}
		err := ExecuteAssertion(actx, step)
		if err != nil {
			t.Errorf("ExecuteAssertion(checkElementCount) error = %v", err)
		}
	})
}

func TestExecuteAssertion_CheckConsoleMessage(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := chromedp.Run(page.Ctx(),
			chromedp.Evaluate(`console.log('test message')`, nil),
		)
		if err != nil {
			t.Fatalf("emitting console.log: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		step := &BrowserStep{
			Action:   "checkConsoleMessage",
			Level:    "log",
			Contains: "test message",
		}
		err = ExecuteAssertion(actx, step)
		if err != nil {
			t.Errorf("ExecuteAssertion(checkConsoleMessage) error = %v", err)
		}
	})
}

func TestExecuteAssertion_CheckNoConsoleErrors(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action: "checkNoConsoleErrors",
		}
		err := ExecuteAssertion(actx, step)
		if err != nil {
			t.Errorf("ExecuteAssertion(checkNoConsoleErrors) error = %v", err)
		}
	})
}

func TestExecuteStep_Submit(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<form id="form">
<input type="text" name="name" value="test" />
<button type="submit">Submit</button>
</form>
<div id="result"></div>
<script>
document.getElementById('form').addEventListener('submit', function(e) {
    e.preventDefault();
    document.getElementById('result').textContent = 'submitted';
});
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "submit",
			Selector: "#form",
		}
		err := ExecuteStep(actx, step)

		if err != nil {
			t.Errorf("ExecuteStep(submit) error = %v", err)
		}
	})
}

func TestExecuteStep_WaitForText(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target">Initial</div>
<script>
setTimeout(function() {
    document.getElementById('target').textContent = 'Updated Text';
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "waitForText",
			Selector: "#target",
			Expected: "Updated Text",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(waitForText) error = %v", err)
		}
	})
}

func TestExecuteStep_WaitForEnabled(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<button id="btn" disabled>Button</button>
<script>
setTimeout(function() {
    document.getElementById('btn').disabled = false;
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "waitForEnabled",
			Selector: "#btn",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(waitForEnabled) error = %v", err)
		}
	})
}

func TestExecuteStep_WaitForDisabled(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<button id="btn">Button</button>
<script>
setTimeout(function() {
    document.getElementById('btn').disabled = true;
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "waitForDisabled",
			Selector: "#btn",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(waitForDisabled) error = %v", err)
		}
	})
}

func TestExecuteStep_WaitForNotPresent(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><body>
<div id="target">To Remove</div>
<script>
setTimeout(function() {
    document.getElementById('target').remove();
}, 100);
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "waitForNotPresent",
			Selector: "#target",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(waitForNotPresent) error = %v", err)
		}
	})
}

func TestExecuteStep_Eval(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action: "eval",
			Value:  "document.title = 'Modified'",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(eval) error = %v", err)
		}

		title, err := GetTitle(actx)
		if err != nil {
			t.Fatalf("GetTitle() error = %v", err)
		}
		if title != "Modified" {
			t.Errorf("title = %q, want %q", title, "Modified")
		}
	})
}

func TestExecuteStep_ScrollIntoView(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLScroll)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:   "scrollIntoView",
			Selector: "#target",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(scrollIntoView) error = %v", err)
		}
	})
}

func TestExecuteStep_SetAttribute(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:    "setAttribute",
			Selector:  "#target",
			Attribute: "data-new",
			Value:     "new-value",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(setAttribute) error = %v", err)
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

func TestExecuteStep_RemoveAttribute(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action:    "removeAttribute",
			Selector:  "#target",
			Attribute: "data-custom",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(removeAttribute) error = %v", err)
		}

		attr, err := GetElementAttribute(page.Ctx(), "#target", "data-custom")
		if err != nil {
			t.Fatalf("GetElementAttribute() error = %v", err)
		}
		if attr != nil {
			t.Errorf("expected attribute to be removed, got %q", *attr)
		}
	})
}

func TestExecuteStep_SetViewport(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action: "setViewport",
			Width:  800,
			Height: 600,
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(setViewport) error = %v", err)
		}
	})
}

func TestExecuteStep_SelectionActions(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLContentEditable)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := chromedp.Run(page.Ctx(), chromedp.Focus("#editor", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("focusing editor: %v", err)
		}

		t.Run("setCursor", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "setCursor",
				Selector: "#editor",
				Offset:   5,
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(setCursor) error = %v", err)
			}
		})

		t.Run("setSelection", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "setSelection",
				Selector: "#editor",
				Start:    0,
				End:      5,
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(setSelection) error = %v", err)
			}
		})

		t.Run("selectAll", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "selectAll",
				Selector: "#editor",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(selectAll) error = %v", err)
			}
		})

		t.Run("collapseSelection", func(t *testing.T) {
			step := &BrowserStep{
				Action: "collapseSelection",
				ToEnd:  true,
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(collapseSelection) error = %v", err)
			}
		})
	})
}

func TestExecuteStep_KeyActions(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLKeyboard)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Focus(actx, "#input")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		t.Run("keyDown", func(t *testing.T) {
			step := &BrowserStep{
				Action: "keyDown",
				Value:  "Shift",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(keyDown) error = %v", err)
			}
		})

		t.Run("keyUp", func(t *testing.T) {
			step := &BrowserStep{
				Action: "keyUp",
				Value:  "Shift",
			}
			err := ExecuteStep(actx, step)
			if err != nil {
				t.Errorf("ExecuteStep(keyUp) error = %v", err)
			}
		})
	})
}

func TestExecuteStep_Stop(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		step := &BrowserStep{
			Action: "stop",
		}
		err := ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(stop) error = %v", err)
		}
	})
}

func TestSetFiles(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFileInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		tmpfile, err := os.CreateTemp("", "test*.txt")
		if err != nil {
			t.Fatalf("creating temp file: %v", err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()

		_, err = tmpfile.WriteString("test content")
		if err != nil {
			t.Fatalf("writing temp file: %v", err)
		}
		_ = tmpfile.Close()

		srcSandbox, err := safedisk.NewNoOpSandbox(filepath.Dir(tmpfile.Name()), safedisk.ModeReadOnly)
		if err != nil {
			t.Fatalf("creating source sandbox: %v", err)
		}
		actx.SrcSandbox = srcSandbox

		t.Run("set file on input", func(t *testing.T) {
			err := SetFiles(actx, "#file-input", []string{filepath.Base(tmpfile.Name())})
			if err != nil {
				t.Errorf("SetFiles() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)
			text, err := GetElementText(page.Ctx(), "#file-name")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}

			if text == "" {
				t.Error("expected file name to be displayed")
			}
		})

	})
}

func TestExecuteStep_SetFiles(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFileInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		tmpfile, err := os.CreateTemp("", "test*.txt")
		if err != nil {
			t.Fatalf("creating temp file: %v", err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		srcSandbox, err := safedisk.NewNoOpSandbox(filepath.Dir(tmpfile.Name()), safedisk.ModeReadOnly)
		if err != nil {
			t.Fatalf("creating source sandbox: %v", err)
		}
		actx.SrcSandbox = srcSandbox

		step := &BrowserStep{
			Action:   "setFiles",
			Selector: "#file-input",
			Files:    []string{filepath.Base(tmpfile.Name())},
		}
		err = ExecuteStep(actx, step)
		if err != nil {
			t.Errorf("ExecuteStep(setFiles) error = %v", err)
		}
	})
}

func TestScroll_EmptySelector(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html>
<head>
<title>Scroll Test</title>
<style>.content { height: 5000px; }</style>
</head>
<body><div class="content">Content</div></body>
</html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Scroll(actx, "", "200")
		if err != nil {
			t.Errorf("Scroll(empty) error = %v", err)
		}
	})
}

func TestScrollInShadowDOM(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html>
<head>
<title>Shadow DOM Scroll Test</title>
</head>
<body>
<div id="host"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = '<div id="scroll-container" style="height: 200px; overflow-y: scroll;"><div style="height: 1000px;">Scrollable</div></div>';
</script>
</body>
</html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("scroll element in shadow DOM", func(t *testing.T) {
			err := Scroll(actx, "#host >>> #scroll-container", "50")
			if err != nil {
				t.Errorf("Scroll(shadow DOM) error = %v", err)
			}
		})
	})
}

func TestDispatchEvent_DocumentAndAsteriskSelectors(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html>
<head><title>Dispatch Event Test</title></head>
<body>
<div id="result"></div>
<script>
document.addEventListener('document-event', function(e) {
	document.getElementById('result').textContent = 'document:' + JSON.stringify(e.detail);
});
</script>
</body>
</html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("dispatch on document with asterisk selector", func(t *testing.T) {
			err := DispatchEvent(actx, "*", "document-event", map[string]any{"doc": true})
			if err != nil {
				t.Errorf("DispatchEvent(*) error = %v", err)
			}
		})

		t.Run("dispatch on document", func(t *testing.T) {
			err := DispatchEvent(actx, "document", "document-event", map[string]any{"doc": "test"})
			if err != nil {
				t.Errorf("DispatchEvent(document) error = %v", err)
			}
		})
	})
}

func TestWaitForVisible_Timeout(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForVisible(actx, "#hidden-display", 100*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error for hidden element")
		}
	})
}

func TestWaitForEnabled_Timeout(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForEnabled(actx, "#disabled-input", 100*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error for disabled element")
		}
	})
}

func TestWaitForDisabled_Timeout(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForDisabled(actx, "#enabled-input", 100*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error for enabled element")
		}
	})
}

func TestWaitForNotVisible_Timeout(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLVisibility)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForNotVisible(actx, "#visible", 100*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error for visible element")
		}
	})
}

func TestWaitForText_Timeout(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForText(actx, "#target", "NonexistentText", 100*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error for nonexistent text")
		}
	})
}

func TestWaitForNotPresent_Timeout(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForNotPresent(actx, "#target", 100*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error for existing element")
		}
	})
}

func TestWaitForSelector_Timeout(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := WaitForSelector(actx, "#nonexistent", 100*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error for nonexistent element")
		}
	})
}

func TestDispatchEventInShadowDOM(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html>
<head><title>Shadow DOM Dispatch Event Test</title></head>
<body>
<div id="host"></div>
<div id="result"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = '<div id="shadow-target">Shadow Target</div>';
shadow.getElementById('shadow-target').addEventListener('shadow-event', function(e) {
	document.getElementById('result').textContent = 'shadow:' + JSON.stringify(e.detail);
});
</script>
</body>
</html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("dispatch on shadow DOM element", func(t *testing.T) {
			err := DispatchEvent(actx, "#host >>> #shadow-target", "shadow-event", map[string]any{"shadow": "value"})
			if err != nil {
				t.Errorf("DispatchEvent(shadow DOM) error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)
			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text != `shadow:{"shadow":"value"}` {
				t.Errorf("result = %q, want shadow event result", text)
			}
		})
	})
}

func TestGoBack(t *testing.T) {
	t.Parallel()
	server := newTestServerWithRoutes(map[string]string{
		"/":      `<!DOCTYPE html><html><head><title>Home</title></head><body><a id="link" href="/page2">Go to Page 2</a></body></html>`,
		"/page2": `<!DOCTYPE html><html><head><title>Page 2</title></head><body></body></html>`,
	})
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)
		actx.ServerURL = server.URL

		err := Navigate(actx, "/page2")
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		err = GoBack(actx)

		_ = err
	})
}

func TestGoForward(t *testing.T) {
	t.Parallel()
	server := newTestServerWithRoutes(map[string]string{
		"/":      `<!DOCTYPE html><html><head><title>Home</title></head><body></body></html>`,
		"/page2": `<!DOCTYPE html><html><head><title>Page 2</title></head><body></body></html>`,
	})
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)
		actx.ServerURL = server.URL

		err := Navigate(actx, "/page2")
		if err != nil {
			t.Fatalf("Navigate() error = %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		err = GoForward(actx)

		_ = err
	})
}

func TestExecuteStep_GoBackGoForward(t *testing.T) {
	t.Parallel()
	server := newTestServerWithRoutes(map[string]string{
		"/":      `<!DOCTYPE html><html><head><title>Home</title></head><body></body></html>`,
		"/page2": `<!DOCTYPE html><html><head><title>Page 2</title></head><body></body></html>`,
	})
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)
		actx.ServerURL = server.URL

		err := ExecuteStep(actx, &BrowserStep{Action: "navigate", Value: "/page2"})
		if err != nil {
			t.Fatalf("ExecuteStep(navigate) error = %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		t.Run("goBack handler", func(t *testing.T) {

			err := ExecuteStep(actx, &BrowserStep{Action: "goBack"})
			_ = err
		})

		t.Run("goForward handler", func(t *testing.T) {

			err := ExecuteStep(actx, &BrowserStep{Action: "goForward"})
			_ = err
		})
	})
}

func TestExecuteStep_TriggerBusEventHandler(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoBusEvent)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("triggerBusEvent handler", func(t *testing.T) {
			err := ExecuteStep(actx, &BrowserStep{
				Action:      "triggerBusEvent",
				EventName:   "test-event",
				EventDetail: map[string]any{"value": 123},
			})
			if err != nil {
				t.Errorf("ExecuteStep(triggerBusEvent) error = %v", err)
			}
		})

		t.Run("pikoBusEmit handler alias", func(t *testing.T) {
			err := ExecuteStep(actx, &BrowserStep{
				Action:      "pikoBusEmit",
				EventName:   "test-event-2",
				EventDetail: map[string]any{"value": 456},
			})
			if err != nil {
				t.Errorf("ExecuteStep(pikoBusEmit) error = %v", err)
			}
		})
	})
}

func TestExecuteStep_TriggerPartialReloadHandler(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoPartial)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("triggerPartialReload handler", func(t *testing.T) {

			err := ExecuteStep(actx, &BrowserStep{
				Action:      "triggerPartialReload",
				PartialName: "test-partial",
				Data:        map[string]any{"key": "value"},
			})

			_ = err
		})

		t.Run("pikoPartialReload handler alias", func(t *testing.T) {
			err := ExecuteStep(actx, &BrowserStep{
				Action:       "pikoPartialReload",
				PartialName:  "test-partial",
				RefreshLevel: 1,
			})

			_ = err
		})
	})
}

func TestClearConsoleLogs(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLConsole)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {

		err := chromedp.Run(page.Ctx(), chromedp.Click("#log-btn", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("clicking log button: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		logsBefore := page.ConsoleLogs()
		t.Logf("logs before clear: %d", len(logsBefore))

		page.ClearConsoleLogs()

		err = chromedp.Run(page.Ctx(), chromedp.Click("#warn-btn", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("clicking warn button: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		logsAfter := page.ConsoleLogs()
		t.Logf("logs after clear and new click: %d", len(logsAfter))

	})
}

func TestEval_AllBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLAttributes)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("eval on window", func(t *testing.T) {
			err := Eval(actx, "window", "window.__testVar = 'window-value'")
			if err != nil {
				t.Errorf("Eval(window) error = %v", err)
			}
		})

		t.Run("eval on document", func(t *testing.T) {
			err := Eval(actx, "document", "document.__docVar = 'doc-value'")
			if err != nil {
				t.Errorf("Eval(document) error = %v", err)
			}
		})

		t.Run("eval with empty selector", func(t *testing.T) {
			err := Eval(actx, "", "window.__emptyVar = 'empty-value'")
			if err != nil {
				t.Errorf("Eval(empty) error = %v", err)
			}
		})

		t.Run("eval on element", func(t *testing.T) {
			err := Eval(actx, "#target", "this.setAttribute('data-eval', 'evaluated')")
			if err != nil {
				t.Errorf("Eval(#target) error = %v", err)
			}
		})
	})
}

func TestEval_ShadowDOM(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLShadowDOM)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("eval on shadow DOM element", func(t *testing.T) {
			err := Eval(actx, "#host >>> #inner", "this.textContent = 'Evaluated'")
			if err != nil {
				t.Errorf("Eval(shadow DOM) error = %v", err)
			}
		})
	})
}

func TestKeyDownKeyUp_AllBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLKeyboard)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		err := Focus(actx, "#input")
		if err != nil {
			t.Fatalf("Focus() error = %v", err)
		}

		t.Run("modifier key (Shift)", func(t *testing.T) {
			err := KeyDown(actx, "Shift")
			if err != nil {
				t.Errorf("KeyDown(Shift) error = %v", err)
			}
			err = KeyUp(actx, "Shift")
			if err != nil {
				t.Errorf("KeyUp(Shift) error = %v", err)
			}
		})

		t.Run("modifier key (Control)", func(t *testing.T) {
			err := KeyDown(actx, "Control")
			if err != nil {
				t.Errorf("KeyDown(Control) error = %v", err)
			}
			err = KeyUp(actx, "Control")
			if err != nil {
				t.Errorf("KeyUp(Control) error = %v", err)
			}
		})

		t.Run("modifier key (Alt)", func(t *testing.T) {
			err := KeyDown(actx, "Alt")
			if err != nil {
				t.Errorf("KeyDown(Alt) error = %v", err)
			}
			err = KeyUp(actx, "Alt")
			if err != nil {
				t.Errorf("KeyUp(Alt) error = %v", err)
			}
		})

		t.Run("modifier key (Meta)", func(t *testing.T) {
			err := KeyDown(actx, "Meta")
			if err != nil {
				t.Errorf("KeyDown(Meta) error = %v", err)
			}
			err = KeyUp(actx, "Meta")
			if err != nil {
				t.Errorf("KeyUp(Meta) error = %v", err)
			}
		})

		t.Run("special key (Enter)", func(t *testing.T) {
			err := KeyDown(actx, "Enter")
			if err != nil {
				t.Errorf("KeyDown(Enter) error = %v", err)
			}
			err = KeyUp(actx, "Enter")
			if err != nil {
				t.Errorf("KeyUp(Enter) error = %v", err)
			}
		})

		t.Run("special key (Tab)", func(t *testing.T) {
			err := KeyDown(actx, "Tab")
			if err != nil {
				t.Errorf("KeyDown(Tab) error = %v", err)
			}
			err = KeyUp(actx, "Tab")
			if err != nil {
				t.Errorf("KeyUp(Tab) error = %v", err)
			}
		})

		t.Run("single character key (a)", func(t *testing.T) {
			err := KeyDown(actx, "a")
			if err != nil {
				t.Errorf("KeyDown(a) error = %v", err)
			}
			err = KeyUp(actx, "a")
			if err != nil {
				t.Errorf("KeyUp(a) error = %v", err)
			}
		})

		t.Run("single character key (z)", func(t *testing.T) {
			err := KeyDown(actx, "z")
			if err != nil {
				t.Errorf("KeyDown(z) error = %v", err)
			}
			err = KeyUp(actx, "z")
			if err != nil {
				t.Errorf("KeyUp(z) error = %v", err)
			}
		})

		t.Run("numeric character key (1)", func(t *testing.T) {
			err := KeyDown(actx, "1")
			if err != nil {
				t.Errorf("KeyDown(1) error = %v", err)
			}
			err = KeyUp(actx, "1")
			if err != nil {
				t.Errorf("KeyUp(1) error = %v", err)
			}
		})

		t.Run("unknown key returns error", func(t *testing.T) {
			err := KeyDown(actx, "Unknown")

			if err == nil {
				t.Error("expected error for unknown key, got nil")
			}
		})
	})
}

func TestExecuteStep_WaitForPartialReloadHandler(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoPartial)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("waitForPartialReload handler", func(t *testing.T) {

			step := &BrowserStep{
				Action:      "waitForPartialReload",
				PartialName: "test-partial",
				Timeout:     100,
			}
			err := ExecuteStep(actx, step)

			_ = err
		})
	})
}

func TestHandlerErrorBranches(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("type without value returns error", func(t *testing.T) {
			step := &BrowserStep{
				Action: "type",
				Value:  "",
			}
			err := ExecuteStep(actx, step)
			if err == nil {
				t.Error("expected error for type without value")
			}
		})

		t.Run("press without key returns error", func(t *testing.T) {
			step := &BrowserStep{
				Action: "press",
			}
			err := ExecuteStep(actx, step)
			if err == nil {
				t.Error("expected error for press without key")
			}
		})

		t.Run("keyDown without key returns error", func(t *testing.T) {
			step := &BrowserStep{
				Action: "keyDown",
			}
			err := ExecuteStep(actx, step)
			if err == nil {
				t.Error("expected error for keyDown without key")
			}
		})

		t.Run("keyUp without key returns error", func(t *testing.T) {
			step := &BrowserStep{
				Action: "keyUp",
			}
			err := ExecuteStep(actx, step)
			if err == nil {
				t.Error("expected error for keyUp without key")
			}
		})

		t.Run("unknown action returns error", func(t *testing.T) {
			step := &BrowserStep{
				Action: "unknownAction",
			}
			err := ExecuteStep(actx, step)
			if err == nil {
				t.Error("expected error for unknown action")
			}
		})
	})
}

func TestGetStepTimeout(t *testing.T) {
	t.Parallel()

	server := newTestServer(testHTMLEmpty)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		t.Run("waitForSelector with custom timeout", func(t *testing.T) {
			step := &BrowserStep{
				Action:   "waitForSelector",
				Selector: "#nonexistent",
				Timeout:  50,
			}
			start := time.Now()
			err := ExecuteStep(actx, step)
			elapsed := time.Since(start)
			if err == nil {
				t.Error("expected timeout error")
			}

			if elapsed > 1*time.Second {
				t.Errorf("expected timeout around 50ms, got %v", elapsed)
			}
		})
	})
}

func TestClick_StaleDOMReplacement(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLStaleDOMReplacement)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		var originalText string
		if err := chromedp.Run(page.Ctx(), chromedp.Text("#btn", &originalText, chromedp.ByQuery)); err != nil {
			t.Fatalf("initial query to populate CDP node cache: %v", err)
		}

		replaceJS := `document.getElementById('container').innerHTML = '<button id="btn">New Button</button><div id="result"></div><input type="text" id="input" />';`
		if err := chromedp.Run(page.Ctx(), chromedp.Evaluate(replaceJS, nil)); err != nil {
			t.Fatalf("replacing DOM: %v", err)
		}

		bindJS := `document.getElementById('btn').addEventListener('click', function(){ document.getElementById('result').textContent = 'clicked-after-replace'; });`
		if err := chromedp.Run(page.Ctx(), chromedp.Evaluate(bindJS, nil)); err != nil {
			t.Fatalf("binding event handler: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		if err := Click(actx, "#btn"); err != nil {
			t.Fatalf("Click() after DOM replacement: %v", err)
		}

		var text string
		if err := chromedp.Run(page.Ctx(), chromedp.Text("#result", &text, chromedp.ByID)); err != nil {
			t.Fatalf("getting result text: %v", err)
		}
		if text != "clicked-after-replace" {
			t.Errorf("result text = %q, want %q", text, "clicked-after-replace")
		}
	})
}

func TestDoubleClick_StaleDOMReplacement(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html><body>
<div id="container">
    <button id="btn">Double Click Me</button>
    <div id="result"></div>
</div>
<script>
document.getElementById('btn').addEventListener('dblclick', function() {
    document.getElementById('result').textContent = 'double-clicked';
});
</script>
</body></html>`

	server := newTestServer(html)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		var originalText string
		if err := chromedp.Run(page.Ctx(), chromedp.Text("#btn", &originalText, chromedp.ByQuery)); err != nil {
			t.Fatalf("initial query to populate CDP node cache: %v", err)
		}

		replaceJS := `document.getElementById('container').innerHTML = '<button id="btn">New Double Click</button><div id="result"></div>';`
		if err := chromedp.Run(page.Ctx(), chromedp.Evaluate(replaceJS, nil)); err != nil {
			t.Fatalf("replacing DOM: %v", err)
		}

		bindJS := `document.getElementById('btn').addEventListener('dblclick', function(){ document.getElementById('result').textContent = 'dblclicked-after-replace'; });`
		if err := chromedp.Run(page.Ctx(), chromedp.Evaluate(bindJS, nil)); err != nil {
			t.Fatalf("binding event handler: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		if err := DoubleClick(actx, "#btn"); err != nil {
			t.Fatalf("DoubleClick() after DOM replacement: %v", err)
		}

		var text string
		if err := chromedp.Run(page.Ctx(), chromedp.Text("#result", &text, chromedp.ByID)); err != nil {
			t.Fatalf("getting result text: %v", err)
		}
		if text != "dblclicked-after-replace" {
			t.Errorf("result text = %q, want %q", text, "dblclicked-after-replace")
		}
	})
}

func TestFill_StaleDOMReplacement(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLStaleDOMReplacement)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		var originalValue string
		if err := chromedp.Run(page.Ctx(), chromedp.Value("#input", &originalValue, chromedp.ByQuery)); err != nil {
			t.Fatalf("initial query to populate CDP node cache: %v", err)
		}

		replaceJS := `document.getElementById('container').innerHTML = '<button id="btn">Click</button><div id="result"></div><input type="text" id="input" value="" />';`
		if err := chromedp.Run(page.Ctx(), chromedp.Evaluate(replaceJS, nil)); err != nil {
			t.Fatalf("replacing DOM: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		if err := Fill(actx, "#input", "after replace"); err != nil {
			t.Fatalf("Fill() after DOM replacement: %v", err)
		}

		var value string
		if err := chromedp.Run(page.Ctx(), chromedp.Value("#input", &value, chromedp.ByID)); err != nil {
			t.Fatalf("getting input value: %v", err)
		}
		if value != "after replace" {
			t.Errorf("input value = %q, want %q", value, "after replace")
		}
	})
}

func TestClear_StaleDOMReplacement(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLStaleDOMReplacement)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		var originalValue string
		if err := chromedp.Run(page.Ctx(), chromedp.Value("#input", &originalValue, chromedp.ByQuery)); err != nil {
			t.Fatalf("initial query to populate CDP node cache: %v", err)
		}

		replaceJS := `document.getElementById('container').innerHTML = '<button id="btn">Click</button><div id="result"></div><input type="text" id="input" value="should-be-cleared" />';`
		if err := chromedp.Run(page.Ctx(), chromedp.Evaluate(replaceJS, nil)); err != nil {
			t.Fatalf("replacing DOM: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		if err := Clear(actx, "#input"); err != nil {
			t.Fatalf("Clear() after DOM replacement: %v", err)
		}

		var value string
		if err := chromedp.Run(page.Ctx(), chromedp.Value("#input", &value, chromedp.ByID)); err != nil {
			t.Fatalf("getting input value: %v", err)
		}
		if value != "" {
			t.Errorf("input value = %q, want empty", value)
		}
	})
}
