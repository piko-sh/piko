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
)

const (
	testHTMLFrames = `<!DOCTYPE html>
<html>
<head><title>Frame Test</title></head>
<body>
<div id="content">Main Page Content</div>
<iframe id="frame1" name="test-frame" srcdoc="<html><body><div id='inner'>Frame Content</div><button id='frame-btn'>Click</button></body></html>"></iframe>
<iframe id="frame2" srcdoc="<html><body><span>Second Frame</span></body></html>"></iframe>
</body>
</html>`
	testHTMLFramesWithInput = `<!DOCTYPE html>
<html>
<head><title>Frame Input Test</title></head>
<body>
<iframe id="input-frame" srcdoc="<html><body><input id='text-input' type='text' value=''></body></html>"></iframe>
</body>
</html>`
)

func TestCountFrames(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("count iframes", func(t *testing.T) {
			count, err := CountFrames(ctx)
			if err != nil {
				t.Fatalf("CountFrames() error = %v", err)
			}
			if count != 2 {
				t.Errorf("CountFrames() = %d, want 2", count)
			}
		})
	})
}

func TestGetFrameBySelector(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("get frame info", func(t *testing.T) {
			info, err := GetFrameBySelector(ctx, "#frame1")
			if err != nil {
				t.Fatalf("GetFrameBySelector() error = %v", err)
			}
			if info == nil {
				t.Fatal("GetFrameBySelector() returned nil")
			}
			if info.Name != "test-frame" {
				t.Errorf("info.Name = %q, want %q", info.Name, "test-frame")
			}
		})

		t.Run("non-existent frame returns error", func(t *testing.T) {
			_, err := GetFrameBySelector(ctx, "#nonexistent")
			if err == nil {
				t.Error("expected error for non-existent frame")
			}
		})
	})
}

func TestIsFrameLoaded(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("check frame loaded", func(t *testing.T) {
			loaded, err := IsFrameLoaded(ctx, "#frame1")
			if err != nil {
				t.Fatalf("IsFrameLoaded() error = %v", err)
			}
			if !loaded {
				t.Log("Frame not yet loaded (may be expected)")
			}
		})

		t.Run("non-existent frame returns false", func(t *testing.T) {
			loaded, err := IsFrameLoaded(ctx, "#nonexistent")
			if err != nil {
				t.Fatalf("IsFrameLoaded() error = %v", err)
			}
			if loaded {
				t.Error("IsFrameLoaded() should return false for non-existent frame")
			}
		})
	})
}

func TestGetTextInFrame(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("get text from frame element", func(t *testing.T) {
			text, err := GetTextInFrame(ctx, "#frame1", "#inner")
			if err != nil {
				t.Fatalf("GetTextInFrame() error = %v", err)
			}
			if text != "Frame Content" {
				t.Errorf("GetTextInFrame() = %q, want %q", text, "Frame Content")
			}
		})
	})
}

func TestGetFrameDocument(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("get frame document HTML", func(t *testing.T) {
			html, err := GetFrameDocument(ctx, "#frame1")
			if err != nil {
				t.Fatalf("GetFrameDocument() error = %v", err)
			}
			if html == "" {
				t.Error("GetFrameDocument() returned empty string")
			}

			if len(html) < 10 {
				t.Errorf("GetFrameDocument() returned unexpectedly short HTML: %q", html)
			}
		})
	})
}

func TestGetFrames(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("get all frames info", func(t *testing.T) {
			frames, err := GetFrames(ctx)
			if err != nil {
				t.Fatalf("GetFrames() error = %v", err)
			}

			if len(frames) == 0 {
				t.Error("GetFrames() returned no frames")
			}
		})
	})
}

func TestEvalInFrame(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("evaluate JS in frame", func(t *testing.T) {
			result, err := EvalInFrame(ctx, "#frame1", "document.getElementById('inner').textContent")
			if err != nil {
				t.Fatalf("EvalInFrame() error = %v", err)
			}
			if result != "Frame Content" {
				t.Errorf("EvalInFrame() = %q, want %q", result, "Frame Content")
			}
		})

		t.Run("non-existent frame returns error", func(t *testing.T) {
			_, err := EvalInFrame(ctx, "#nonexistent", "1+1")
			if err == nil {
				t.Error("expected error for non-existent frame")
			}
		})
	})
}

func TestClickInFrame(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("click element in frame", func(t *testing.T) {
			err := ClickInFrame(ctx, "#frame1", "#frame-btn")
			if err != nil {
				t.Fatalf("ClickInFrame() error = %v", err)
			}
		})

		t.Run("click non-existent element returns error", func(t *testing.T) {
			err := ClickInFrame(ctx, "#frame1", "#nonexistent")
			if err == nil {
				t.Error("expected error for non-existent element")
			}
		})
	})
}

func TestFillInFrame(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFramesWithInput)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("fill input in frame", func(t *testing.T) {
			err := FillInFrame(ctx, "#input-frame", "#text-input", "test value")
			if err != nil {
				t.Fatalf("FillInFrame() error = %v", err)
			}
		})
	})
}

func TestWaitForElementInFrame(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("wait for existing element in frame", func(t *testing.T) {
			err := WaitForElementInFrame(ctx, "#frame1", "#inner")
			if err != nil {
				t.Fatalf("WaitForElementInFrame() error = %v", err)
			}
		})

		t.Run("wait for non-existent element returns error", func(t *testing.T) {
			err := WaitForElementInFrame(ctx, "#frame1", "#nonexistent-element-that-will-never-appear")
			if err == nil {
				t.Error("expected error for non-existent element")
			}
		})
	})
}

func TestExecuteInFrameContext(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLFrames)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("execute action in frame context", func(t *testing.T) {
			err := ExecuteInFrameContext(ctx, "#frame1", func(frameCtx context.Context) error {

				return nil
			})
			if err != nil {
				t.Fatalf("ExecuteInFrameContext() error = %v", err)
			}
		})
	})
}
