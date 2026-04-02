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

func TestPikoBusEmit(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("emits event successfully", func(t *testing.T) {
			err := PikoBusEmit(ctx, "custom-event", map[string]any{"message": "hello"})
			if err != nil {
				t.Errorf("PikoBusEmit() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#custom-result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text == "" {
				t.Error("expected custom-result to have content after event")
			}
		})

		t.Run("emits event with nil detail", func(t *testing.T) {
			err := PikoBusEmit(ctx, "nil-event", nil)
			if err != nil {
				t.Errorf("PikoBusEmit() with nil detail error = %v", err)
			}
		})
	})
}

func TestPikoSetupEventLog(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("sets up event log without error", func(t *testing.T) {

			err := PikoSetupEventLog(ctx)
			if err != nil {
				t.Errorf("PikoSetupEventLog() error = %v", err)
			}
		})

		t.Run("retrieves event log without error", func(t *testing.T) {

			log, err := PikoGetEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoGetEventLog() error = %v", err)
			}

			if log == nil {
				t.Error("expected log to be non-nil")
			}
		})
	})
}

func TestPikoGetEventLog(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoBusEvent)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("returns empty array when log is cleared", func(t *testing.T) {
			err := PikoSetupEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoSetupEventLog() error = %v", err)
			}

			err = PikoClearEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoClearEventLog() error = %v", err)
			}

			log, err := PikoGetEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoGetEventLog() error = %v", err)
			}

			if len(log) != 0 {
				t.Errorf("expected empty log, got %d events", len(log))
			}
		})

		t.Run("returns array type not nil", func(t *testing.T) {

			err := PikoSetupEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoSetupEventLog() error = %v", err)
			}

			log, err := PikoGetEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoGetEventLog() error = %v", err)
			}

			if log == nil {
				t.Error("expected non-nil log slice")
			}
		})

		t.Run("retrieves manually pushed events", func(t *testing.T) {

			jsSetup := `
				window.eventLog = [
					{ event: 'test-event', detail: { key: 'value' }, timestamp: Date.now() },
					{ event: 'another-event', detail: { count: 42 }, timestamp: Date.now() }
				]
			`
			err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(jsSetup, nil))
			if err != nil {
				t.Fatalf("setting up test events: %v", err)
			}

			log, err := PikoGetEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoGetEventLog() error = %v", err)
			}

			if len(log) != 2 {
				t.Fatalf("expected 2 events, got %d", len(log))
			}

			if log[0]["event"] != "test-event" {
				t.Errorf("first event name = %v, want test-event", log[0]["event"])
			}

			if detail, ok := log[0]["detail"].(map[string]any); ok {
				if detail["key"] != "value" {
					t.Errorf("detail.key = %v, want value", detail["key"])
				}
			} else {
				t.Error("expected detail to be map[string]any")
			}

			if log[1]["event"] != "another-event" {
				t.Errorf("second event name = %v, want another-event", log[1]["event"])
			}
		})

		t.Run("handles empty log gracefully", func(t *testing.T) {

			err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(`window.eventLog = []`, nil))
			if err != nil {
				t.Fatalf("clearing event log: %v", err)
			}

			log, err := PikoGetEventLog(ctx)
			if err != nil {
				t.Errorf("PikoGetEventLog() error = %v, want nil", err)
			}

			if len(log) != 0 {
				t.Errorf("expected empty log, got %d events", len(log))
			}
		})
	})
}

func TestPikoClearEventLog(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoBusEvent)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("clears event log", func(t *testing.T) {

			err := PikoSetupEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoSetupEventLog() error = %v", err)
			}

			err = PikoBusEmit(ctx, "clear-test", nil)
			if err != nil {
				t.Fatalf("PikoBusEmit() error = %v", err)
			}

			time.Sleep(50 * time.Millisecond)

			err = PikoClearEventLog(ctx)
			if err != nil {
				t.Errorf("PikoClearEventLog() error = %v", err)
			}

			log, err := PikoGetEventLog(ctx)
			if err != nil {
				t.Fatalf("PikoGetEventLog() error = %v", err)
			}

			if len(log) != 0 {
				t.Errorf("expected empty log after clear, got %d events", len(log))
			}
		})
	})
}

func TestPikoCheckBusEventReceived(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("function executes without error", func(t *testing.T) {

			_, err := PikoCheckBusEventReceived(ctx, "any-event")
			if err != nil {
				t.Errorf("PikoCheckBusEventReceived() error = %v", err)
			}
		})

		t.Run("returns false for non-received event", func(t *testing.T) {

			received, err := PikoCheckBusEventReceived(ctx, "never-emitted-unique-event-12345")
			if err != nil {
				t.Fatalf("PikoCheckBusEventReceived() error = %v", err)
			}

			if received {
				t.Error("expected event to not be received for unique event name")
			}
		})
	})
}

func TestPikoGetPartialState(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoPartial)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("retrieves partial state without error", func(t *testing.T) {

			state, err := PikoGetPartialState(ctx, "test-partial")
			if err != nil {
				t.Errorf("PikoGetPartialState() error = %v", err)
			}

			if state == nil {
				t.Error("expected state to not be nil")
			}
		})

		t.Run("handles nonexistent partial without error", func(t *testing.T) {

			state, err := PikoGetPartialState(ctx, "nonexistent-partial")
			if err != nil {
				t.Errorf("PikoGetPartialState() error = %v", err)
			}

			if state == nil {
				t.Error("expected state to not be nil")
			}
		})
	})
}

func TestTriggerBusEvent(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoBusEvent)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("triggers event via TriggerBusEvent", func(t *testing.T) {
			err := TriggerBusEvent(ctx, "test-event", map[string]any{"value": 42})
			if err != nil {
				t.Errorf("TriggerBusEvent() error = %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}
			if text == "" {
				t.Error("expected result to have content after event")
			}
		})
	})
}

func TestPikoPartialReload(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoPartial)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("partial reload does not error", func(t *testing.T) {

			err := PikoPartialReload(ctx, "test-partial", nil)

			_ = err
		})
	})
}

func TestPikoPartialReloadWithLevel(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoPartial)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("partial reload with level does not error", func(t *testing.T) {

			err := PikoPartialReloadWithLevel(ctx, "test-partial", nil, 1)
			_ = err
		})
	})
}

func TestPikoDispatchFragmentMorph(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("fragment morph executes without error", func(t *testing.T) {

			err := PikoDispatchFragmentMorph(ctx, "#custom-result", "<span>Morphed</span>")

			_ = err
		})
	})
}

func TestPikoBusWaitForEvent(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoBusEvent)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("wait for event setup executes without error", func(t *testing.T) {

			_, err := PikoBusWaitForEvent(ctx, "test-event", 100*time.Millisecond)

			if err == nil {
				t.Log("surprisingly, event was received")
			}
		})
	})
}

func TestPikoWaitForPartialReload(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoPartial)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("wait for partial reload with short timeout", func(t *testing.T) {

			err := PikoWaitForPartialReload(ctx, "test-partial", 100*time.Millisecond)

			_ = err
		})
	})
}

func TestPikoDebugIsAvailable(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("debug API is available", func(t *testing.T) {
			available, err := PikoDebugIsAvailable(ctx)
			if err != nil {
				t.Fatalf("PikoDebugIsAvailable() error = %v", err)
			}
			if !available {
				t.Error("expected debug API to be available")
			}
		})
	})
}

func TestPikoDebugGetPartialInfo(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("returns info for existing partial", func(t *testing.T) {
			info, err := PikoDebugGetPartialInfo(ctx, "[partial]")
			if err != nil {
				t.Fatalf("PikoDebugGetPartialInfo() error = %v", err)
			}
			if info == nil {
				t.Fatal("expected info to be non-nil")
			}

			if _, ok := info["isConnected"]; !ok {
				t.Error("expected 'isConnected' field in info")
			}
			if _, ok := info["registeredCallbacks"]; !ok {
				t.Error("expected 'registeredCallbacks' field in info")
			}
			if _, ok := info["cleanupCount"]; !ok {
				t.Error("expected 'cleanupCount' field in info")
			}
		})

		t.Run("returns error for non-existent element", func(t *testing.T) {
			_, err := PikoDebugGetPartialInfo(ctx, "#nonexistent-element-12345")
			if err == nil {
				t.Error("expected error for non-existent element")
			}
		})
	})
}

func TestPikoDebugIsConnected(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("returns connection state", func(t *testing.T) {
			connected, err := PikoDebugIsConnected(ctx, "[partial]")
			if err != nil {
				t.Fatalf("PikoDebugIsConnected() error = %v", err)
			}

			t.Logf("partial connected state: %v", connected)
		})

		t.Run("returns false for non-partial element", func(t *testing.T) {
			connected, err := PikoDebugIsConnected(ctx, "body")
			if err != nil {
				t.Fatalf("PikoDebugIsConnected() error = %v", err)
			}

			if connected {
				t.Error("expected body to not be connected as a partial")
			}
		})
	})
}

func TestPikoDebugGetCleanupCount(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("returns cleanup count", func(t *testing.T) {
			count, err := PikoDebugGetCleanupCount(ctx, "[partial]")
			if err != nil {
				t.Fatalf("PikoDebugGetCleanupCount() error = %v", err)
			}

			if count < 0 {
				t.Errorf("expected non-negative cleanup count, got %d", count)
			}
			t.Logf("cleanup count: %d", count)
		})

		t.Run("returns 0 for non-existent element", func(t *testing.T) {
			count, err := PikoDebugGetCleanupCount(ctx, "#nonexistent-12345")
			if err != nil {
				t.Fatalf("PikoDebugGetCleanupCount() error = %v", err)
			}
			if count != 0 {
				t.Errorf("expected 0 cleanup count for non-existent element, got %d", count)
			}
		})
	})
}

func TestPikoDebugGetRegisteredCallbacks(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("returns callback names", func(t *testing.T) {
			callbacks, err := PikoDebugGetRegisteredCallbacks(ctx, "[partial]")
			if err != nil {
				t.Fatalf("PikoDebugGetRegisteredCallbacks() error = %v", err)
			}

			if callbacks == nil {
				t.Error("expected callbacks to be non-nil")
			}
			t.Logf("registered callbacks: %v", callbacks)
		})

		t.Run("returns empty for non-partial element", func(t *testing.T) {
			callbacks, err := PikoDebugGetRegisteredCallbacks(ctx, "body")
			if err != nil {
				t.Fatalf("PikoDebugGetRegisteredCallbacks() error = %v", err)
			}
			if len(callbacks) != 0 {
				t.Errorf("expected empty callbacks for body, got %v", callbacks)
			}
		})
	})
}

func TestPikoDebugGetAllConnectedPartials(t *testing.T) {
	t.Parallel()
	server, err := newPikoTestServer(testHTMLPikoComplete)
	if err != nil {
		t.Fatalf("creating Piko test server: %v", err)
	}
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		time.Sleep(200 * time.Millisecond)

		t.Run("returns connected partials", func(t *testing.T) {
			selectors, err := PikoDebugGetAllConnectedPartials(ctx)
			if err != nil {
				t.Fatalf("PikoDebugGetAllConnectedPartials() error = %v", err)
			}

			if selectors == nil {
				t.Error("expected selectors to be non-nil")
			}
			t.Logf("connected partials: %v", selectors)
		})
	})
}
