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

package tui_domain

const (
	// LayoutHeaderHeight is the lines reserved for the title row at the top
	// of the screen.
	LayoutHeaderHeight = 1

	// LayoutTabBarHeight is the lines reserved for the tab bar shown beneath
	// the title in single-pane mode.
	LayoutTabBarHeight = 1

	// LayoutStatusBarHeight is the lines reserved for the status bar at the
	// bottom of the screen.
	LayoutStatusBarHeight = 1

	// LayoutGapsHeight is the lines reserved for blank gaps separating the
	// chrome sections (one between header/tab and content, one between
	// content and status bar).
	LayoutGapsHeight = 2

	// LayoutChromeHeight is the total terminal lines consumed by the chrome
	// (header + tab bar + status bar + interleaving gaps). Subtract from the
	// terminal height to get the content area height.
	LayoutChromeHeight = LayoutHeaderHeight + LayoutTabBarHeight + LayoutStatusBarHeight + LayoutGapsHeight
)

const (
	// PanelBorderHeight is the lines consumed by the top and bottom of the
	// panel border.
	PanelBorderHeight = 2

	// PanelBorderWidth is the columns consumed by the left and right of the
	// panel border.
	PanelBorderWidth = 2

	// PanelPaddingHeight is the lines consumed by the title row plus
	// vertical padding inside the panel border.
	PanelPaddingHeight = 2

	// PanelPaddingWidth is the columns consumed by horizontal padding inside
	// the left and right of the panel border.
	PanelPaddingWidth = 2

	// PanelChromeHeight is the total lines consumed by panel border and
	// padding combined. Subtract from a panel's allocated height to get the
	// height available for content.
	PanelChromeHeight = PanelBorderHeight + PanelPaddingHeight

	// PanelChromeWidth is the total columns consumed by panel border and
	// padding combined. Subtract from a panel's allocated width to get the
	// width available for content.
	PanelChromeWidth = PanelBorderWidth + PanelPaddingWidth
)
