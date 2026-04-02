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

package wizardbase

import "charm.land/lipgloss/v2"

var (
	// TitleStyle is the bold coloured style used for step headings.
	TitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)

	// HelpStyle is the dim style used for help text at the bottom.
	HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// CursorStyle is the coloured style for the ">" cursor indicator.
	CursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	// SelectedStyle is the bold green style for highlighted/selected items.
	SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)

	// SuccessStyle is the green style used for "Done!" messages.
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)
