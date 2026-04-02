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

	"github.com/chromedp/chromedp"
)

const testHTMLEmulation = `<!DOCTYPE html>
<html>
<head><title>Emulation Test</title></head>
<body>
<div id="viewport-info"></div>
<script>
document.getElementById('viewport-info').textContent =
    window.innerWidth + 'x' + window.innerHeight;
</script>
</body>
</html>`

func TestDeviceEmulation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLEmulation)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("emulates iPhone 14", func(t *testing.T) {
			err := EmulateIPhone14(ctx)
			if err != nil {
				t.Fatalf("EmulateIPhone14() error = %v", err)
			}

			var width, height int
			err = chromedp.Run(ctx.Ctx,
				chromedp.Evaluate(`window.innerWidth`, &width),
				chromedp.Evaluate(`window.innerHeight`, &height),
			)
			if err != nil {
				t.Fatalf("failed to get viewport size: %v", err)
			}

			if width == 0 || height == 0 {
				t.Error("viewport dimensions should not be zero after emulation")
			}
		})

		t.Run("emulates iPad", func(t *testing.T) {
			err := EmulateIPad(ctx)
			if err != nil {
				t.Fatalf("EmulateIPad() error = %v", err)
			}

			var width int
			err = chromedp.Run(ctx.Ctx, chromedp.Evaluate(`window.innerWidth`, &width))
			if err != nil {
				t.Fatalf("failed to get viewport width: %v", err)
			}
			if width == 0 {
				t.Error("viewport width should not be zero after emulation")
			}
		})

		t.Run("emulates Galaxy S9", func(t *testing.T) {
			err := EmulateGalaxyS9(ctx)
			if err != nil {
				t.Fatalf("EmulateGalaxyS9() error = %v", err)
			}

			var width int
			err = chromedp.Run(ctx.Ctx, chromedp.Evaluate(`window.innerWidth`, &width))
			if err != nil {
				t.Fatalf("failed to get viewport width: %v", err)
			}
			if width == 0 {
				t.Error("viewport width should not be zero after emulation")
			}
		})

		t.Run("emulates generic mobile", func(t *testing.T) {
			err := EmulateMobile(ctx)
			if err != nil {
				t.Fatalf("EmulateMobile() error = %v", err)
			}
		})

		t.Run("emulates generic tablet", func(t *testing.T) {
			err := EmulateTablet(ctx)
			if err != nil {
				t.Fatalf("EmulateTablet() error = %v", err)
			}
		})

		t.Run("emulates iPhone 14 Pro", func(t *testing.T) {
			err := EmulateIPhone14Pro(ctx)
			if err != nil {
				t.Fatalf("EmulateIPhone14Pro() error = %v", err)
			}
		})

		t.Run("emulates iPhone 14 Pro Max", func(t *testing.T) {
			err := EmulateIPhone14ProMax(ctx)
			if err != nil {
				t.Fatalf("EmulateIPhone14ProMax() error = %v", err)
			}
		})

		t.Run("emulates iPhone 12", func(t *testing.T) {
			err := EmulateIPhone12(ctx)
			if err != nil {
				t.Fatalf("EmulateIPhone12() error = %v", err)
			}
		})

		t.Run("emulates iPhone SE", func(t *testing.T) {
			err := EmulateIPhoneSE(ctx)
			if err != nil {
				t.Fatalf("EmulateIPhoneSE() error = %v", err)
			}
		})

		t.Run("emulates iPad Mini", func(t *testing.T) {
			err := EmulateIPadMini(ctx)
			if err != nil {
				t.Fatalf("EmulateIPadMini() error = %v", err)
			}
		})

		t.Run("emulates iPad Pro", func(t *testing.T) {
			err := EmulateIPadPro(ctx)
			if err != nil {
				t.Fatalf("EmulateIPadPro() error = %v", err)
			}
		})

		t.Run("emulates Galaxy S8", func(t *testing.T) {
			err := EmulateGalaxyS8(ctx)
			if err != nil {
				t.Fatalf("EmulateGalaxyS8() error = %v", err)
			}
		})

		t.Run("emulates iPhone 14 landscape", func(t *testing.T) {
			err := EmulateIPhone14Landscape(ctx)
			if err != nil {
				t.Fatalf("EmulateIPhone14Landscape() error = %v", err)
			}
		})

		t.Run("emulates iPad landscape", func(t *testing.T) {
			err := EmulateIPadLandscape(ctx)
			if err != nil {
				t.Fatalf("EmulateIPadLandscape() error = %v", err)
			}
		})
	})
}

func TestDesktopEmulation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLEmulation)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("emulates HD desktop", func(t *testing.T) {
			err := EmulateDesktopHD(ctx)
			if err != nil {
				t.Fatalf("EmulateDesktopHD() error = %v", err)
			}

			var width, height int
			err = chromedp.Run(ctx.Ctx,
				chromedp.Evaluate(`window.innerWidth`, &width),
				chromedp.Evaluate(`window.innerHeight`, &height),
			)
			if err != nil {
				t.Fatalf("failed to get viewport size: %v", err)
			}
			if width != ViewportWidthHD || height != ViewportHeightHD {
				t.Errorf("EmulateDesktopHD() viewport = %dx%d, want %dx%d",
					width, height, ViewportWidthHD, ViewportHeightHD)
			}
		})

		t.Run("emulates 4K desktop", func(t *testing.T) {
			err := EmulateDesktop4K(ctx)
			if err != nil {
				t.Fatalf("EmulateDesktop4K() error = %v", err)
			}

			var width, height int
			err = chromedp.Run(ctx.Ctx,
				chromedp.Evaluate(`window.innerWidth`, &width),
				chromedp.Evaluate(`window.innerHeight`, &height),
			)
			if err != nil {
				t.Fatalf("failed to get viewport size: %v", err)
			}
			if width != ViewportWidth4K || height != ViewportHeight4K {
				t.Errorf("EmulateDesktop4K() viewport = %dx%d, want %dx%d",
					width, height, ViewportWidth4K, ViewportHeight4K)
			}
		})

		t.Run("emulates custom desktop size", func(t *testing.T) {
			err := EmulateDesktop(ctx, 1280, 720)
			if err != nil {
				t.Fatalf("EmulateDesktop() error = %v", err)
			}

			var width, height int
			err = chromedp.Run(ctx.Ctx,
				chromedp.Evaluate(`window.innerWidth`, &width),
				chromedp.Evaluate(`window.innerHeight`, &height),
			)
			if err != nil {
				t.Fatalf("failed to get viewport size: %v", err)
			}
			if width != 1280 || height != 720 {
				t.Errorf("EmulateDesktop() viewport = %dx%d, want 1280x720", width, height)
			}
		})
	})
}

func TestEmulateViewportByName(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLEmulation)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		testCases := []struct {
			name          string
			sizeName      string
			expectedWidth int64
		}{
			{name: "mobile-s", sizeName: "mobile-s", expectedWidth: 320},
			{name: "mobile-m", sizeName: "mobile-m", expectedWidth: 375},
			{name: "tablet", sizeName: "tablet", expectedWidth: 768},
			{name: "laptop", sizeName: "laptop", expectedWidth: 1024},
			{name: "desktop", sizeName: "desktop", expectedWidth: 1920},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := EmulateViewportByName(ctx, tc.sizeName)
				if err != nil {
					t.Fatalf("EmulateViewportByName(%q) error = %v", tc.sizeName, err)
				}

				var width int
				err = chromedp.Run(ctx.Ctx, chromedp.Evaluate(`window.innerWidth`, &width))
				if err != nil {
					t.Fatalf("failed to get viewport width: %v", err)
				}
				if int64(width) != tc.expectedWidth {
					t.Errorf("EmulateViewportByName(%q) width = %d, want %d",
						tc.sizeName, width, tc.expectedWidth)
				}
			})
		}

		t.Run("returns error for unknown size", func(t *testing.T) {
			err := EmulateViewportByName(ctx, "unknown-size")
			if err == nil {
				t.Error("EmulateViewportByName() expected error for unknown size")
			}
		})
	})
}
