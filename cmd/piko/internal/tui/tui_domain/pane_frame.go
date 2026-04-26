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

import (
	"charm.land/lipgloss/v2"
)

// PaneFrameOpts configures the appearance of a pane frame rendered by
// RenderPaneFrame. Values left at their zero defaults render with sensible
// fallbacks; callers must always supply a Theme.
type PaneFrameOpts struct {
	// Theme supplies the styles used to render the frame.
	Theme *Theme

	// Title is the header text rendered inside the top border.
	Title string

	// Body is the content rendered between the borders.
	Body string

	// Indicator is an optional small marker shown alongside the title
	// (e.g. an icon for focus or status).
	Indicator string

	// Width is the outer width of the frame including borders.
	Width int

	// Height is the outer height of the frame including borders.
	Height int

	// Focused selects between the focused and inactive border styles.
	Focused bool
}

// RenderPaneFrame composes a pane's title bar, border, and body into the
// final string. Panels that defer chrome to the layout call this from their
// View; panels that own their own chrome (the migration path during the
// theme rollout) continue to use BasePanel.RenderFrame.
//
// Takes opts (PaneFrameOpts) which configures the frame.
//
// Returns string which is the framed pane sized to opts.Width by
// opts.Height.
func RenderPaneFrame(opts PaneFrameOpts) string {
	if opts.Width <= 0 || opts.Height <= 0 {
		return ""
	}

	style := paneFrameStyle(opts)

	titleRow := paneFrameTitle(opts)
	body := opts.Body

	inner := body
	if titleRow != "" {
		if body == "" {
			inner = titleRow
		} else {
			inner = lipgloss.JoinVertical(lipgloss.Left, titleRow, body)
		}
	}

	return style.
		Width(opts.Width).
		Height(opts.Height).
		Render(inner)
}

// paneFrameStyle picks the focused or inactive border style from the theme,
// falling back to the legacy globals when no theme is supplied so the frame
// remains visible in tests that do not configure a theme.
//
// Takes opts (PaneFrameOpts) which carries the theme and focus flag.
//
// Returns lipgloss.Style which is the border style ready to render the
// pane.
func paneFrameStyle(opts PaneFrameOpts) lipgloss.Style {
	if opts.Theme != nil {
		if opts.Focused {
			return opts.Theme.PanelFocused
		}
		return opts.Theme.Panel
	}
	if opts.Focused {
		return panelFocusedStyle
	}
	return panelStyle
}

// paneFrameTitle composes the title-and-indicator row drawn inside the
// frame. Returns the empty string when both are blank.
//
// Takes opts (PaneFrameOpts) which carries the title, indicator, and theme.
//
// Returns string which is the styled title row.
func paneFrameTitle(opts PaneFrameOpts) string {
	if opts.Title == "" && opts.Indicator == "" {
		return ""
	}

	var titleStyle lipgloss.Style
	if opts.Theme != nil {
		titleStyle = opts.Theme.PanelTitle
	} else {
		titleStyle = panelTitleStyle
	}

	if opts.Title == "" {
		return titleStyle.Render(opts.Indicator)
	}
	if opts.Indicator == "" {
		return titleStyle.Render(opts.Title)
	}
	return titleStyle.Render(opts.Title) + " " + titleStyle.Render(opts.Indicator)
}
