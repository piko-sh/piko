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

import "charm.land/lipgloss/v2"

const (
	// PriorityCriticalGlyph marks critical-priority events.
	PriorityCriticalGlyph = "●"

	// PriorityHighGlyph marks high-priority events.
	PriorityHighGlyph = "●"

	// PriorityNormalGlyph marks routine-priority events. Drawn as a
	// hollow ring so it visually differs from filled glyphs in the
	// no-colour theme.
	PriorityNormalGlyph = "○"

	// SectionMarker is rendered to the left of the active section in
	// the watchdog Overview's section nav.
	SectionMarker = "▸"
)

// PriorityStyle returns the theme-aware style for an event priority.
//
// Takes theme (*Theme) which supplies the colour palette. Pass nil to
// fall back to the legacy global styles.
// Takes priority (WatchdogEventPriority) which selects the styling.
//
// Returns lipgloss.Style which is the appropriate styled glyph.
func PriorityStyle(theme *Theme, priority WatchdogEventPriority) lipgloss.Style {
	if theme != nil {
		switch priority {
		case WatchdogPriorityCritical:
			return theme.StatusUnhealthy.Bold(true)
		case WatchdogPriorityHigh:
			return theme.StatusDegraded.Bold(true)
		default:
			return theme.StatusUnknown
		}
	}
	switch priority {
	case WatchdogPriorityCritical:
		return statusUnhealthyStyle.Bold(true)
	case WatchdogPriorityHigh:
		return statusDegradedStyle.Bold(true)
	default:
		return statusUnknownStyle
	}
}

// PriorityGlyph returns the glyph for an event priority.
//
// Takes priority (WatchdogEventPriority) which selects the glyph.
//
// Returns string which is the unstyled glyph.
func PriorityGlyph(priority WatchdogEventPriority) string {
	switch priority {
	case WatchdogPriorityCritical:
		return PriorityCriticalGlyph
	case WatchdogPriorityHigh:
		return PriorityHighGlyph
	default:
		return PriorityNormalGlyph
	}
}

// StyledPriorityGlyph returns a styled glyph for the given priority,
// suitable for direct concatenation into a watchdog event row.
//
// Takes theme (*Theme) which supplies the colour palette.
// Takes priority (WatchdogEventPriority) which selects the styling.
//
// Returns string which is the styled glyph.
func StyledPriorityGlyph(theme *Theme, priority WatchdogEventPriority) string {
	return PriorityStyle(theme, priority).Render(PriorityGlyph(priority))
}

// CategoryLabel returns the short human label for a watchdog event
// category, suitable for badges and filters.
//
// Takes category (WatchdogEventCategory) which selects the label.
//
// Returns string which is the label.
func CategoryLabel(category WatchdogEventCategory) string {
	switch category {
	case WatchdogEventCategoryHeap:
		return "Heap"
	case WatchdogEventCategoryGoroutine:
		return "Goroutines"
	case WatchdogEventCategoryGC:
		return "GC"
	case WatchdogEventCategoryProcess:
		return "Process"
	case WatchdogEventCategoryDiagnostic:
		return "Diagnostic"
	default:
		return "Other"
	}
}
