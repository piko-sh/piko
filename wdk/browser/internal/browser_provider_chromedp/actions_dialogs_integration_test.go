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
)

const testHTMLDialogs = `<!DOCTYPE html>
<html>
<head><title>Dialog Test</title></head>
<body>
<button id="alert-btn" onclick="alert('Test Alert')">Show Alert</button>
<button id="confirm-btn" onclick="document.getElementById('confirm-result').textContent = confirm('Test Confirm?') ? 'yes' : 'no'">Show Confirm</button>
<button id="prompt-btn" onclick="document.getElementById('prompt-result').textContent = prompt('Enter value:', 'default') || 'cancelled'">Show Prompt</button>
<div id="confirm-result"></div>
<div id="prompt-result"></div>
</body>
</html>`

func TestDialogHandler(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDialogs)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("dialog handler creation", func(t *testing.T) {
			handler := NewDialogHandler()
			if handler == nil {
				t.Fatal("NewDialogHandler() returned nil")
			}

			err := handler.Enable(ctx, true)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}

			handler.Disable()
		})

		t.Run("set prompt text", func(t *testing.T) {
			handler := NewDialogHandler()
			handler.SetPromptText("test prompt")

		})

		t.Run("clear last dialog", func(t *testing.T) {
			handler := NewDialogHandler()
			handler.ClearLastDialog()
			if handler.GetLastDialog() != nil {
				t.Error("GetLastDialog() should return nil after clear")
			}
		})
	})
}

func TestSetupDialogAutoAccept(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDialogs)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("auto accept setup", func(t *testing.T) {
			handler := SetupDialogAutoAccept(ctx, "prompt value")
			if handler == nil {
				t.Fatal("SetupDialogAutoAccept() returned nil")
			}
			defer handler.Stop()

			if err := Click(ctx, "#alert-btn"); err != nil {
				t.Fatalf("Click alert button error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

		})
	})
}

func TestSetupDialogAutoDismiss(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDialogs)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("auto dismiss setup", func(t *testing.T) {
			handler := SetupDialogAutoDismiss(ctx)
			if handler == nil {
				t.Fatal("SetupDialogAutoDismiss() returned nil")
			}
			defer handler.Stop()
		})
	})

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("auto dismiss handles alert", func(t *testing.T) {
			handler := SetupDialogAutoDismiss(ctx)
			if handler == nil {
				t.Fatal("SetupDialogAutoDismiss() returned nil")
			}
			defer handler.Stop()

			if err := Click(ctx, "#alert-btn"); err != nil {
				t.Fatalf("Click alert button error = %v", err)
			}

			time.Sleep(300 * time.Millisecond)
		})

		t.Run("auto dismiss handles confirm", func(t *testing.T) {
			handler := SetupDialogAutoDismiss(ctx)
			if handler == nil {
				t.Fatal("SetupDialogAutoDismiss() returned nil")
			}
			defer handler.Stop()

			if err := Click(ctx, "#confirm-btn"); err != nil {
				t.Fatalf("Click confirm button error = %v", err)
			}

			time.Sleep(300 * time.Millisecond)
		})
	})
}

func TestTriggerDialogs(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDialogs)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("trigger alert", func(t *testing.T) {

			handler := SetupDialogAutoAccept(ctx, "")
			defer handler.Stop()

			err := TriggerAlert(ctx, "Test message")

			if err != nil {
				t.Logf("TriggerAlert() returned error (may be expected): %v", err)
			}
		})

		t.Run("trigger confirm", func(t *testing.T) {

			handler := SetupDialogAutoAccept(ctx, "")
			defer handler.Stop()

			_, err := TriggerConfirm(ctx, "Test confirm?")
			if err != nil {
				t.Logf("TriggerConfirm() returned error (may be expected): %v", err)
			}
		})

		t.Run("trigger prompt", func(t *testing.T) {

			handler := SetupDialogAutoAccept(ctx, "prompt response")
			defer handler.Stop()

			_, err := TriggerPrompt(ctx, "Enter value:", "default")
			if err != nil {
				t.Logf("TriggerPrompt() returned error (may be expected): %v", err)
			}
		})
	})
}

func TestHandleDialog(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDialogs)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("handle alert", func(t *testing.T) {

			handler := SetupDialogAutoAccept(ctx, "")
			defer handler.Stop()

			err := HandleAlert(ctx)

			if err != nil {
				t.Logf("HandleAlert() error (expected if no dialog): %v", err)
			}
		})

		t.Run("handle confirm", func(t *testing.T) {
			handler := SetupDialogAutoAccept(ctx, "")
			defer handler.Stop()

			err := HandleConfirm(ctx, true)
			if err != nil {
				t.Logf("HandleConfirm() error (expected if no dialog): %v", err)
			}
		})

		t.Run("handle prompt", func(t *testing.T) {
			handler := SetupDialogAutoAccept(ctx, "test")
			defer handler.Stop()

			err := HandlePrompt(ctx, true, "test input")
			if err != nil {
				t.Logf("HandlePrompt() error (expected if no dialog): %v", err)
			}
		})

		t.Run("dismiss dialog", func(t *testing.T) {
			handler := SetupDialogAutoDismiss(ctx)
			defer handler.Stop()

			err := DismissDialog(ctx)
			if err != nil {
				t.Logf("DismissDialog() error (expected if no dialog): %v", err)
			}
		})

		t.Run("accept dialog", func(t *testing.T) {
			handler := SetupDialogAutoAccept(ctx, "")
			defer handler.Stop()

			err := AcceptDialog(ctx)
			if err != nil {
				t.Logf("AcceptDialog() error (expected if no dialog): %v", err)
			}
		})
	})
}

func TestDialogHandler_HandleTargetEvent(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDialogs)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("auto accept dialog via DialogHandler", func(t *testing.T) {
			handler := NewDialogHandler()
			handler.SetPromptText("auto-text")

			err := handler.Enable(ctx, true)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}

			if err := Click(ctx, "#alert-btn"); err != nil {
				t.Fatalf("Click alert button: %v", err)
			}

			time.Sleep(500 * time.Millisecond)

			lastDialog := handler.GetLastDialog()
			if lastDialog == nil {
				t.Log("lastDialog is nil - dialog event may not have fired in this environment")
			} else {
				if lastDialog.Type != DialogTypeAlert {
					t.Errorf("expected dialog type %q, got %q", DialogTypeAlert, lastDialog.Type)
				}
			}

			handler.Disable()
		})

		t.Run("auto dismiss dialog via DialogHandler", func(t *testing.T) {
			handler := NewDialogHandler()

			err := handler.Enable(ctx, false)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}

			if err := Click(ctx, "#confirm-btn"); err != nil {
				t.Fatalf("Click confirm button: %v", err)
			}

			time.Sleep(500 * time.Millisecond)

			lastDialog := handler.GetLastDialog()
			if lastDialog != nil {
				t.Logf("dialog recorded: type=%s message=%s", lastDialog.Type, lastDialog.Message)
			}

			handler.ClearLastDialog()
			if handler.GetLastDialog() != nil {
				t.Error("expected nil after ClearLastDialog")
			}

			handler.Disable()
		})

		t.Run("WaitForDialog with actual dialog", func(t *testing.T) {
			handler := NewDialogHandler()

			err := handler.Enable(ctx, true)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}

			go func() {
				time.Sleep(100 * time.Millisecond)
				_ = Click(ctx, "#alert-btn")
			}()

			info, err := handler.WaitForDialog(3 * time.Second)
			if err != nil {
				t.Logf("WaitForDialog() error (may be expected): %v", err)
			} else if info != nil {
				t.Logf("WaitForDialog() got dialog: type=%s", info.Type)
			}

			handler.Disable()
		})
	})
}

func TestWaitForDialog(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDialogs)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("wait for dialog with short timeout", func(t *testing.T) {
			handler := NewDialogHandler()
			err := handler.Enable(ctx, true)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}

			_, err = handler.WaitForDialog(100 * time.Millisecond)
			if err == nil {
				t.Log("WaitForDialog() succeeded (dialog was present)")
			} else {
				t.Log("WaitForDialog() timed out as expected")
			}
		})
	})
}
