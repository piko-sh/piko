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
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
)

const (
	// layoutHeaderFooterHeight is the space reserved for header and status bar.
	// Title: 1 line, Tab bar: 1 line, Status bar: 1 line, newlines: 2 = 5 total.
	layoutHeaderFooterHeight = 5
)

// Model is the Bubbletea model for the TUI.
type Model struct {
	// lastRefresh is when the model data was last updated.
	lastRefresh time.Time

	// lastError holds the most recent error to display in the status bar.
	lastError error

	// providerInfo maps provider names to their status and refresh details.
	providerInfo map[string]*ProviderInfo

	// config holds settings for the TUI such as title and refresh interval.
	config *tui_dto.Config

	// resourcesByKind maps resource kinds to their resource lists.
	resourcesByKind map[string][]Resource

	// resourceSummary maps resource kinds to their status counts.
	resourceSummary map[string]map[ResourceStatus]int

	// panels holds the UI panels managed by this model.
	panels []Panel

	// activePanelIndex is the index of the panel that has focus; -1 means none.
	activePanelIndex int

	// height is the terminal height in rows; 0 means not yet set.
	height int

	// width is the terminal width in columns; 0 means not yet set.
	width int

	// resourceDataMutex guards access to resource data during updates and reads.
	resourceDataMutex sync.RWMutex

	// showHelp indicates whether the help overlay is visible.
	showHelp bool

	// quitting indicates whether the application is shutting down.
	quitting bool
}

// NewModel creates a new Model with the given settings.
//
// Takes config (*tui_dto.Config) which sets the TUI options.
//
// Returns *Model which is ready for use.
func NewModel(config *tui_dto.Config) *Model {
	return &Model{
		lastRefresh:       time.Time{},
		lastError:         nil,
		providerInfo:      make(map[string]*ProviderInfo),
		config:            config,
		resourcesByKind:   make(map[string][]Resource),
		resourceSummary:   make(map[string]map[ResourceStatus]int),
		panels:            make([]Panel, 0),
		activePanelIndex:  0,
		height:            0,
		width:             0,
		resourceDataMutex: sync.RWMutex{},
		showHelp:          false,
		quitting:          false,
	}
}

// UpdateResourceData updates the cached resource data from a provider.
// This is called by the refresh orchestrator after a successful refresh.
//
// Takes summary (map[string]map[ResourceStatus]int) which contains counts of
// resources grouped by kind and status.
// Takes resources (map[string][]Resource) which contains the actual resource
// objects grouped by kind.
//
// Safe for concurrent use. Uses a mutex to protect the cached data.
func (m *Model) UpdateResourceData(
	summary map[string]map[ResourceStatus]int,
	resources map[string][]Resource,
) {
	m.resourceDataMutex.Lock()
	defer m.resourceDataMutex.Unlock()

	for kind, statusCounts := range summary {
		if m.resourceSummary[kind] == nil {
			m.resourceSummary[kind] = make(map[ResourceStatus]int)
		}
		maps.Copy(m.resourceSummary[kind], statusCounts)
	}

	maps.Copy(m.resourcesByKind, resources)
}

// AddPanel adds a panel to the model.
//
// Takes p (Panel) which is the panel to add. The first panel added receives
// focus automatically.
func (m *Model) AddPanel(p Panel) {
	m.panels = append(m.panels, p)
	if len(m.panels) == 1 {
		p.SetFocused(true)
	}
}

// ActivePanel returns the panel that currently has focus.
//
// Returns Panel which is the active panel, or nil if no panel has focus.
func (m *Model) ActivePanel() Panel {
	if m.activePanelIndex >= 0 && m.activePanelIndex < len(m.panels) {
		return m.panels[m.activePanelIndex]
	}
	return nil
}

// Init implements tea.Model.
//
// Returns tea.Cmd which batches setup commands from all panels and starts the
// refresh ticker.
func (m *Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.panels)+1)

	for _, p := range m.panels {
		if command := p.Init(); command != nil {
			cmds = append(cmds, command)
		}
	}

	cmds = append(cmds, m.tickCmd())

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which holds commands to run after the update.
func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	cmds := m.dispatchMessage(message)
	if m.quitting {
		return m, tea.Quit
	}

	if command := m.updateActivePanel(message); command != nil {
		cmds = append(cmds, command)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
//
// Returns tea.View which contains the rendered view of the current model state.
func (m *Model) View() tea.View {
	var content string
	if m.quitting {
		content = "Goodbye!\n"
	} else if m.width == 0 || m.height == 0 {
		content = "Initialising..."
	} else {
		content = m.renderLayout()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// handleKeyMessage processes keyboard input and returns the matching command.
//
// Takes message (tea.KeyPressMsg) which contains the key event to process.
//
// Returns tea.Cmd which is the command to run, or nil if the key is not
// handled.
func (m *Model) handleKeyMessage(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "q", "ctrl+c":
		return func() tea.Msg { return quitMessage{} }

	case "?":
		return func() tea.Msg { return toggleHelpMessage{} }

	case "tab":
		return func() tea.Msg { return nextPanelMessage{} }

	case "shift+tab":
		return func() tea.Msg { return previousPanelMessage{} }

	case "r":
		return func() tea.Msg { return forceRefreshMessage{} }

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		index := int(message.String()[0] - '1')
		if index < len(m.panels) {
			return func() tea.Msg { return focusPanelMessage{panelID: m.panels[index].ID()} }
		}
	}

	return nil
}

// focusPanelByID sets focus to the panel with the given ID.
//
// Takes id (string) which specifies the panel to focus.
func (m *Model) focusPanelByID(id string) {
	for i, p := range m.panels {
		if p.ID() == id {
			m.setActivePanel(i)
			return
		}
	}
}

// focusNextPanel moves focus to the next panel in the list.
func (m *Model) focusNextPanel() {
	if len(m.panels) == 0 {
		return
	}
	m.setActivePanel((m.activePanelIndex + 1) % len(m.panels))
}

// focusPreviousPanel moves focus to the previous panel in the list.
func (m *Model) focusPreviousPanel() {
	if len(m.panels) == 0 {
		return
	}
	index := m.activePanelIndex - 1
	if index < 0 {
		index = len(m.panels) - 1
	}
	m.setActivePanel(index)
}

// setActivePanel changes which panel is active.
//
// Takes index (int) which specifies the index of the panel to make active.
func (m *Model) setActivePanel(index int) {
	if index < 0 || index >= len(m.panels) {
		return
	}

	if m.activePanelIndex >= 0 && m.activePanelIndex < len(m.panels) {
		m.panels[m.activePanelIndex].SetFocused(false)
	}

	m.activePanelIndex = index
	m.panels[index].SetFocused(true)
}

// tickCmd returns a command that sends a tick message after the refresh
// interval.
//
// Returns tea.Cmd which triggers a tickMessage after the set refresh interval
// passes.
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.config.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMessage{time: t}
	})
}

// pushDataToPanels sends the stored data to all panels that need it.
//
// Safe for concurrent use. Acquires a read lock on resourceDataMutex.
func (m *Model) pushDataToPanels() {
	m.resourceDataMutex.RLock()
	defer m.resourceDataMutex.RUnlock()

	for _, panel := range m.panels {
		m.pushSummaryToPanel(panel)
		m.pushArtefactsToPanel(panel)
		m.pushTasksToPanel(panel)
		m.pushWorkflowsToPanel(panel)
		m.pushResourcesToPanel(panel)
	}
}

// pushSummaryToPanel sends the resource summary to a panel if it supports it.
//
// Takes panel (Panel) which is the target panel to receive the summary.
func (m *Model) pushSummaryToPanel(panel Panel) {
	if p, ok := panel.(interface {
		SetSummary(map[string]map[ResourceStatus]int)
	}); ok {
		p.SetSummary(m.resourceSummary)
	}
}

// pushArtefactsToPanel sends artefacts to a panel if it supports them.
//
// Takes panel (Panel) which receives the artefacts if it has a SetArtefacts
// method.
func (m *Model) pushArtefactsToPanel(panel Panel) {
	p, ok := panel.(interface{ SetArtefacts([]Resource) })
	if !ok {
		return
	}
	if artefacts, exists := m.resourcesByKind["artefact"]; exists {
		p.SetArtefacts(artefacts)
	}
}

// pushTasksToPanel sends tasks to a panel if it supports receiving them.
//
// Takes panel (Panel) which receives the tasks if it supports SetTasks.
func (m *Model) pushTasksToPanel(panel Panel) {
	p, ok := panel.(interface{ SetTasks([]Resource) })
	if !ok {
		return
	}
	if tasks, exists := m.resourcesByKind["task"]; exists {
		p.SetTasks(tasks)
	}
}

// pushWorkflowsToPanel sends workflows to a panel if it supports them.
//
// Takes panel (Panel) which receives the workflows if it has a SetWorkflows
// method.
func (m *Model) pushWorkflowsToPanel(panel Panel) {
	p, ok := panel.(interface{ SetWorkflows([]Resource) })
	if !ok {
		return
	}
	if workflows, exists := m.resourcesByKind["workflow"]; exists {
		p.SetWorkflows(workflows)
	}
}

// pushResourcesToPanel sends resources to a panel for expanded view.
//
// Takes panel (Panel) which receives the resources if it supports selection.
func (m *Model) pushResourcesToPanel(panel Panel) {
	p, ok := panel.(interface {
		SelectedKind() string
		SetResources([]Resource)
	})
	if !ok {
		return
	}
	kind := p.SelectedKind()
	if kind == "" {
		return
	}
	if resources, exists := m.resourcesByKind[kind]; exists {
		p.SetResources(resources)
	}
}

// notifyPanelsOfDataUpdate sends DataUpdatedMessage to all panels.
//
// Returns tea.Cmd which delivers the update notification when executed.
func (m *Model) notifyPanelsOfDataUpdate() tea.Cmd {
	return func() tea.Msg {
		return DataUpdatedMessage{Time: m.config.GetClock().Now()}
	}
}

// broadcastTickToAllPanels sends a tick message to all panels for background
// updates.
//
// Takes t (time.Time) which specifies the tick timestamp to send.
//
// Returns tea.Cmd which updates all panels with the tick message when run.
func (m *Model) broadcastTickToAllPanels(t time.Time) tea.Cmd {
	return func() tea.Msg {
		tickMessage := TickMessage{Time: t}
		for i, panel := range m.panels {
			updatedPanel, _ := panel.Update(tickMessage)
			m.panels[i] = updatedPanel
		}
		return nil
	}
}

// renderLayout renders the full terminal interface layout.
//
// Returns string which contains the formatted terminal output.
func (m *Model) renderLayout() string {
	if m.showHelp {
		return m.renderHelp()
	}

	if activePanel := m.ActivePanel(); activePanel != nil {
		contentHeight := max(1, m.height-layoutHeaderFooterHeight)

		header := m.renderHeader()
		content := activePanel.View(m.width, contentHeight)
		statusBar := m.renderStatusBar()

		return header + "\n" + content + "\n" + statusBar
	}

	return "No panels available"
}

// renderHeader builds the header section with the title and tab bar.
//
// Returns string which is the formatted header containing the title and tabs.
func (m *Model) renderHeader() string {
	title := m.config.Title
	if title == "" {
		title = "Piko TUI"
	}

	titleRow := titleStyle.Render(title)
	tabBar := m.renderTabBar()

	return titleRow + "\n" + tabBar
}

// renderTabBar renders the panel navigation tabs.
//
// Returns string which contains the formatted tab bar with styled hotkeys.
func (m *Model) renderTabBar() string {
	if len(m.panels) == 0 {
		return ""
	}

	tabs := make([]string, 0, len(m.panels))
	for i, panel := range m.panels {
		hotkeyNum := fmt.Sprintf("%d", i+1)
		title := panel.Title()

		var tab string
		if i == m.activePanelIndex {
			bracket := navItemActiveStyle.Render("[")
			hotkey := navItemHotkeyStyle.Render(hotkeyNum)
			titlePart := navItemActiveStyle.Render(" " + title)
			closeBracket := navItemActiveStyle.Render("]")
			tab = bracket + hotkey + titlePart + closeBracket
		} else {
			hotkey := navItemHotkeyStyle.Render(hotkeyNum)
			tab = navItemStyle.Render(fmt.Sprintf(" %s %s ", hotkey, title))
		}
		tabs = append(tabs, tab)
	}

	separator := helpSeparatorStyle.Render("│")
	return strings.Join(tabs, separator)
}

// renderStatusBar builds the status bar text.
//
// Returns string which contains the styled status bar showing either an error
// message or help text for keyboard shortcuts.
func (m *Model) renderStatusBar() string {
	if m.lastError != nil {
		return statusBarStyle.Render("Error: " + m.lastError.Error())
	}
	status := "Tab panels | ↑/↓ navigate | Space details | / search | ? help | q quit"
	return statusBarStyle.Render(status)
}

// renderHelp builds the help panel content.
//
// Returns string which contains the help text with key bindings.
func (m *Model) renderHelp() string {
	var b strings.Builder
	b.WriteString("=== Piko TUI Help ===\n\n")
	b.WriteString("Global Keys:\n")
	b.WriteString("  ?              Toggle this help\n")
	b.WriteString("  q, Ctrl+C      Quit\n")
	b.WriteString("  Tab            Next panel\n")
	b.WriteString("  Shift+Tab      Previous panel\n")
	b.WriteString("  1-9            Jump to panel\n")
	b.WriteString("  r              Force refresh\n")
	b.WriteString(stringNewline)

	if activePanel := m.ActivePanel(); activePanel != nil {
		b.WriteString("Panel: ")
		b.WriteString(activePanel.Title())
		b.WriteString(stringNewline)
		for _, kb := range activePanel.KeyMap() {
			b.WriteString("  ")
			b.WriteString(kb.Key)
			b.WriteString("  ")
			b.WriteString(kb.Description)
			b.WriteString(stringNewline)
		}
	}

	b.WriteString("\nPress ? to close help")
	return b.String()
}

// dispatchMessage sends messages to the correct handlers based on message type.
//
// Takes message (tea.Msg) which is the message to dispatch.
//
// Returns []tea.Cmd which contains commands produced by the handlers.
func (m *Model) dispatchMessage(message tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	switch message := message.(type) {
	case tea.KeyPressMsg:
		if command := m.handleKeyMessage(message); command != nil {
			cmds = append(cmds, command)
		}
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
	case tickMessage:
		cmds = m.handleTickMessage(message)
	case dataRefreshedMessage:
		m.handleDataRefreshedMessage(message)
	case dataUpdatedMessage:
		cmds = m.handleDataUpdatedMessage(message)
	case providerStatusMessage:
		m.handleProviderStatusMessage(message)
	case errorMessage:
		m.lastError = message.err
	case focusPanelMessage:
		m.focusPanelByID(message.panelID)
	case nextPanelMessage:
		m.focusNextPanel()
	case previousPanelMessage:
		m.focusPreviousPanel()
	case toggleHelpMessage:
		m.showHelp = !m.showHelp
	case quitMessage:
		m.quitting = true
	}

	return cmds
}

// handleTickMessage handles tick messages for regular updates.
//
// Takes message (tickMessage) which contains the tick time.
//
// Returns []tea.Cmd which contains commands to schedule the next tick and
// send the update to all panels.
func (m *Model) handleTickMessage(message tickMessage) []tea.Cmd {
	m.lastRefresh = message.time
	return []tea.Cmd{m.tickCmd(), m.broadcastTickToAllPanels(message.time)}
}

// handleDataRefreshedMessage updates provider state after a data refresh.
//
// Takes message (dataRefreshedMessage) which contains the refresh details.
func (m *Model) handleDataRefreshedMessage(message dataRefreshedMessage) {
	if info, ok := m.providerInfo[message.providerName]; ok {
		info.LastRefresh = m.config.GetClock().Now()
		info.RefreshCount++
	}
}

// handleDataUpdatedMessage processes data update messages from the orchestrator.
//
// Takes message (dataUpdatedMessage) which holds the update time and data.
//
// Returns []tea.Cmd which holds commands to tell panels about the update.
func (m *Model) handleDataUpdatedMessage(message dataUpdatedMessage) []tea.Cmd {
	m.lastRefresh = message.time
	m.pushDataToPanels()
	return []tea.Cmd{m.notifyPanelsOfDataUpdate()}
}

// handleProviderStatusMessage updates provider state when a status change occurs.
//
// Takes message (providerStatusMessage) which contains the
// provider name, new status, and any error.
func (m *Model) handleProviderStatusMessage(message providerStatusMessage) {
	info, ok := m.providerInfo[message.name]
	if !ok {
		return
	}
	info.Status = message.status
	info.LastError = message.err
	if message.err != nil {
		info.ErrorCount++
	}
}

// updateActivePanel sends a message to the active panel and updates it.
//
// Takes message (tea.Msg) which is the message to send to the panel.
//
// Returns tea.Cmd which is any command produced by the panel update.
func (m *Model) updateActivePanel(message tea.Msg) tea.Cmd {
	activePanel := m.ActivePanel()
	if activePanel == nil || m.showHelp {
		return nil
	}

	updatedPanel, command := activePanel.Update(message)
	if index := m.activePanelIndex; index >= 0 && index < len(m.panels) {
		m.panels[index] = updatedPanel
	}
	return command
}
