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
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// searchBoxCharLimit is the maximum number of characters allowed in the
	// search box.
	searchBoxCharLimit = 128

	// searchBoxDefaultWidth is the default character width for search input fields.
	searchBoxDefaultWidth = 40

	// searchBoxInitialWidth is the default width of the search box in characters.
	searchBoxInitialWidth = 50

	// searchBoxMinInputWidth is the smallest character width for the search input.
	searchBoxMinInputWidth = 20
)

// SearchBox provides a text input field for search queries in a k9s-style
// interface. It implements io.Closer.
type SearchBox struct {
	// onClose is called when the search box closes with the search text and
	// whether the user confirmed or cancelled.
	onClose func(query string, confirmed bool)

	// input is the text input widget for entering search queries.
	input textinput.Model

	// width is the total width of the search box in characters.
	width int

	// active indicates whether the search box is open and accepting input.
	active bool
}

var (
	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(0, 1)

	searchLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)
)

// NewSearchBox creates a new search input widget.
//
// Returns *SearchBox which is the configured search input ready for use.
func NewSearchBox() *SearchBox {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = searchBoxCharLimit
	ti.SetWidth(searchBoxDefaultWidth)
	ti.Prompt = ""

	return &SearchBox{
		input:   ti,
		width:   searchBoxInitialWidth,
		onClose: nil,
		active:  false,
	}
}

// SetWidth sets the width of the search box.
//
// Takes w (int) which specifies the width in characters.
func (s *SearchBox) SetWidth(w int) {
	s.width = w
	s.input.SetWidth(max(searchBoxMinInputWidth, w-12))
}

// SetOnClose sets the callback for when search is closed.
//
// Takes callback (func(query string, confirmed bool)) which
// receives the final query text and whether the user confirmed
// or cancelled the search.
func (s *SearchBox) SetOnClose(callback func(query string, confirmed bool)) {
	s.onClose = callback
}

// Active returns whether the search box is active.
//
// Returns bool which is true when the search box is in active state.
func (s *SearchBox) Active() bool {
	return s.active
}

// Open activates the search box and prepares it for input.
//
// Returns tea.Cmd which starts the cursor blink animation.
func (s *SearchBox) Open() tea.Cmd {
	s.active = true
	s.input.Reset()
	s.input.Focus()
	return textinput.Blink
}

// Close deactivates the search box and removes input focus.
//
// Takes confirmed (bool) which indicates whether to submit the current query.
func (s *SearchBox) Close(confirmed bool) {
	s.active = false
	s.input.Blur()
	if s.onClose != nil {
		query := ""
		if confirmed {
			query = s.input.Value()
		}
		s.onClose(query, confirmed)
	}
}

// Query returns the current search query.
//
// Returns string which is the current query text entered by the user.
func (s *SearchBox) Query() string {
	return s.input.Value()
}

// SetQuery sets the search query text.
//
// Takes q (string) which specifies the query text to set.
func (s *SearchBox) SetQuery(q string) {
	s.input.SetValue(q)
}

// Update handles input messages when the search box is active.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns *SearchBox which is the updated search box.
// Returns tea.Cmd which is the command to run, or nil if none.
func (s *SearchBox) Update(message tea.Msg) (*SearchBox, tea.Cmd) {
	if !s.active {
		return s, nil
	}

	if keyMessage, ok := message.(tea.KeyPressMsg); ok {
		switch keyMessage.String() {
		case "enter":
			s.Close(true)
			return s, nil
		case "esc":
			s.Close(false)
			return s, nil
		}
	}

	var command tea.Cmd
	s.input, command = s.input.Update(message)
	return s, command
}

// View renders the search box as a styled string.
//
// Returns string which holds the rendered view, or an empty string if the
// search box is not active.
func (s *SearchBox) View() string {
	if !s.active {
		return ""
	}

	label := searchLabelStyle.Render("/ ")
	inputView := s.input.View()

	content := lipgloss.JoinHorizontal(lipgloss.Center, label, inputView)
	return searchBoxStyle.Width(s.width - 4).Render(content)
}
