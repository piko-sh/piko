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
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	// SymbolCursorActive is the cursor shown when an item is selected.
	SymbolCursorActive = "▸ "

	// SymbolCursorInactive is the symbol shown when an item is not selected.
	SymbolCursorInactive = "  "

	// SymbolExpanded is shown when an item is open and its children are visible.
	SymbolExpanded = "▽"

	// SymbolCollapsed indicates an item is collapsed and hides its children.
	SymbolCollapsed = "▷"

	// SymbolStatusFilled is a filled circle that shows an active or known state.
	SymbolStatusFilled = "●"

	// SymbolStatusEmpty is an empty circle symbol for unknown or pending states.
	SymbolStatusEmpty = "○"

	// IndentCursor is the number of characters reserved for the cursor marker.
	IndentCursor = 2

	// IndentExpand is the number of characters reserved for the expand marker.
	IndentExpand = 2

	// IndentChild is the indent level for child or nested items.
	IndentChild = 4

	// IndentMetadata is the indentation for metadata detail rows.
	IndentMetadata = 6
)

// CursorConfig holds configuration for cursor rendering with customisable
// indentation.
type CursorConfig struct {
	// ActiveIndent is the number of spaces before the cursor when the item is
	// selected.
	ActiveIndent int

	// InactiveIndent is the number of spaces when the item is not selected. This
	// should typically be ActiveIndent + len(SymbolCursorActive) to maintain
	// alignment.
	InactiveIndent int
}

// DefaultCursorConfig returns the standard cursor settings for top-level items.
// Active items have no prefix spacing, while inactive items have two spaces.
//
// Returns CursorConfig which contains the default active and inactive indent
// settings.
func DefaultCursorConfig() CursorConfig {
	return CursorConfig{
		ActiveIndent:   0,
		InactiveIndent: IndentCursor,
	}
}

// ChildCursorConfig returns cursor settings for child or nested items. The
// active cursor uses a two-space prefix with an arrow, whilst the inactive
// cursor uses a four-space indent.
//
// Returns CursorConfig which contains the active and inactive indent settings.
func ChildCursorConfig() CursorConfig {
	return CursorConfig{
		ActiveIndent:   IndentCursor,
		InactiveIndent: IndentChild,
	}
}

// MetadataCursorConfig returns cursor settings for metadata and detail rows.
// Active rows use a four-space prefix with an arrow, inactive rows use six
// spaces.
//
// Returns CursorConfig which contains the indent settings for active and
// inactive metadata rows.
func MetadataCursorConfig() CursorConfig {
	return CursorConfig{
		ActiveIndent:   IndentChild,
		InactiveIndent: IndentMetadata,
	}
}

// RenderCursorStyled renders a cursor indicator with configurable
// indentation, showing a styled arrow when selected or blank space
// to maintain alignment otherwise.
//
// Takes selected (bool) which indicates whether the item is
// currently selected.
// Takes focused (bool) which indicates whether the panel has
// keyboard focus.
// Takes config (CursorConfig) which provides the indent settings
// for active and inactive states.
//
// Returns string which is the rendered cursor indicator with the
// correct indentation and styling.
func RenderCursorStyled(selected, focused bool, config CursorConfig) string {
	if !selected {
		return strings.Repeat(" ", config.InactiveIndent)
	}

	indent := strings.Repeat(" ", config.ActiveIndent)
	if focused {
		return indent + lipgloss.NewStyle().Foreground(colourPrimary).Render(SymbolCursorActive)
	}
	return indent + SymbolCursorActive
}
