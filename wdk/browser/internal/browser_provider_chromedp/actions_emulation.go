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
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
)

const (
	// ViewportWidthHD is the width in pixels of a Full HD (1920x1080) display.
	ViewportWidthHD = 1920

	// ViewportHeightHD is the height of a Full HD (1920x1080) display.
	ViewportHeightHD = 1080

	// ViewportWidth4K is the width in pixels of a 4K UHD display (3840x2160).
	ViewportWidth4K = 3840

	// ViewportHeight4K is the height of a 4K UHD (3840x2160) display.
	ViewportHeight4K = 2160
)

// CommonViewportSizes provides standard viewport sizes for responsive testing.
var CommonViewportSizes = map[string]struct {
	Width  int64
	Height int64
}{
	"mobile-s":   {Width: 320, Height: 568},
	"mobile-m":   {Width: 375, Height: 667},
	"mobile-l":   {Width: 414, Height: 896},
	"tablet":     {Width: 768, Height: 1024},
	"laptop":     {Width: 1024, Height: 768},
	"laptop-l":   {Width: 1440, Height: 900},
	"desktop":    {Width: 1920, Height: 1080},
	"desktop-4k": {Width: 3840, Height: 2160},
}

// EmulateIPhone14 emulates an iPhone 14.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when the emulation fails.
func EmulateIPhone14(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPhone14))
}

// EmulateIPhone14Pro emulates an iPhone 14 Pro.
//
// Takes ctx (*ActionContext) which provides the browser context to emulate.
//
// Returns error when the emulation fails to apply.
func EmulateIPhone14Pro(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPhone14Pro))
}

// EmulateIPhone14ProMax emulates an iPhone 14 Pro Max.
//
// Takes ctx (*ActionContext) which provides the browser context to emulate.
//
// Returns error when the emulation fails to apply.
func EmulateIPhone14ProMax(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPhone14ProMax))
}

// EmulateIPhone13 emulates an iPhone 13.
//
// Takes ctx (*ActionContext) which provides the browser context for emulation.
//
// Returns error when the emulation fails to apply.
func EmulateIPhone13(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPhone13))
}

// EmulateIPhone12 emulates an iPhone 12.
//
// Takes ctx (*ActionContext) which provides the browser context to configure.
//
// Returns error when the emulation fails to apply.
func EmulateIPhone12(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPhone12))
}

// EmulateIPhoneSE emulates an iPhone SE.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when the emulation fails.
func EmulateIPhoneSE(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPhoneSE))
}

// EmulateIPad emulates an iPad.
//
// Takes ctx (*ActionContext) which provides the browser context for emulation.
//
// Returns error when the emulation fails to apply.
func EmulateIPad(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPad))
}

// EmulateIPadMini emulates an iPad Mini.
//
// Takes ctx (*ActionContext) which provides the browser context to emulate.
//
// Returns error when the emulation fails to apply.
func EmulateIPadMini(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPadMini))
}

// EmulateIPadPro emulates an iPad Pro 11".
//
// Takes ctx (*ActionContext) which provides the browser context to emulate.
//
// Returns error when the emulation fails to apply.
func EmulateIPadPro(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPadPro11))
}

// EmulateGalaxyS9 emulates a Samsung Galaxy S9.
//
// Takes ctx (*ActionContext) which provides the browser context for emulation.
//
// Returns error when the emulation fails to apply.
func EmulateGalaxyS9(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.GalaxyS9))
}

// EmulateGalaxyS8 emulates a Samsung Galaxy S8.
//
// Takes ctx (*ActionContext) which provides the browser context to emulate.
//
// Returns error when the emulation fails to apply.
func EmulateGalaxyS8(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.GalaxyS8))
}

// EmulateMobile emulates a generic mobile device (iPhone 13).
// This sets a common mobile viewport with touch support.
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when the device emulation fails.
func EmulateMobile(ctx *ActionContext) error {
	return EmulateIPhone13(ctx)
}

// EmulateTablet emulates a generic tablet device (iPad).
// This sets a common tablet viewport with touch support.
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when the tablet emulation fails.
func EmulateTablet(ctx *ActionContext) error {
	return EmulateIPad(ctx)
}

// EmulateDesktop sets a standard desktop viewport without mobile emulation.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes width (int64) which specifies the viewport width in pixels.
// Takes height (int64) which specifies the viewport height in pixels.
//
// Returns error when the viewport cannot be set.
func EmulateDesktop(ctx *ActionContext, width, height int64) error {
	return chromedp.Run(ctx.Ctx, chromedp.EmulateViewport(width, height))
}

// EmulateDesktopHD sets a 1920x1080 desktop viewport (Full HD).
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when the viewport cannot be set.
func EmulateDesktopHD(ctx *ActionContext) error {
	return EmulateDesktop(ctx, ViewportWidthHD, ViewportHeightHD)
}

// EmulateDesktop4K sets a 3840x2160 desktop viewport (4K UHD).
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when the viewport cannot be set.
func EmulateDesktop4K(ctx *ActionContext) error {
	return EmulateDesktop(ctx, ViewportWidth4K, ViewportHeight4K)
}

// EmulateIPhone14Landscape emulates an iPhone 14 in landscape mode.
//
// Takes ctx (*ActionContext) which provides the browser context for emulation.
//
// Returns error when the emulation fails to apply.
func EmulateIPhone14Landscape(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPhone14landscape))
}

// EmulateIPadLandscape emulates an iPad in landscape mode.
//
// Takes ctx (*ActionContext) which provides the browser context for emulation.
//
// Returns error when the emulation command fails.
func EmulateIPadLandscape(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Emulate(device.IPadlandscape))
}

// EmulateViewportByName sets the viewport to a common size by name.
//
// Valid names: "mobile-s", "mobile-m", "mobile-l", "tablet", "laptop",
// "laptop-l", "desktop", "desktop-4k".
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes sizeName (string) which specifies the viewport size name to use.
//
// Returns error when the size name is not recognised.
func EmulateViewportByName(ctx *ActionContext, sizeName string) error {
	size, ok := CommonViewportSizes[sizeName]
	if !ok {
		return fmt.Errorf("unknown viewport size: %s", sizeName)
	}
	return EmulateDesktop(ctx, size.Width, size.Height)
}
