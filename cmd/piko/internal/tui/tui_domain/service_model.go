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
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
)

// mouseWheelStepCount is the number of cursor steps applied per wheel
// notch. Three matches the conventional terminal-emulator feel.
const mouseWheelStepCount = 3

// ThemeAware is implemented by panels and widgets that want to receive the
// active theme. Panels that do not implement this continue to read styles
// from the package-level legacy globals.
type ThemeAware interface {
	// SetTheme replaces the panel's active colour theme.
	SetTheme(theme *Theme)
}

// Model is the Bubbletea model for the TUI.
type Model struct {
	// lastRefresh records the time of the most recent data refresh.
	lastRefresh time.Time

	// stateStore persists per-panel cursor, scroll, and filter state.
	stateStore PanelStateStore

	// lastError holds the most recent fatal error surfaced in the status bar.
	lastError error

	// paneAssigner decides which panel renders in each layout pane.
	paneAssigner PaneAssigner

	// resourceSummary holds counts of resources grouped by kind and status.
	resourceSummary map[string]map[ResourceStatus]int

	// commandRegistry holds the available command-bar commands.
	commandRegistry *CommandRegistry

	// layoutPicker picks the responsive layout based on terminal size.
	layoutPicker *LayoutPicker

	// config is the immutable TUI configuration.
	config *tui_dto.Config

	// focusManager tracks which panel currently owns keyboard focus.
	focusManager *FocusManager

	// overlays is the modal overlay stack (help, confirmations, popups).
	overlays *OverlayManager

	// toasts is the queue of transient status notifications.
	toasts *ToastQueue

	// statusBar renders the bottom status row.
	statusBar *StatusBarRenderer

	// providerInfo tracks per-provider refresh state and errors.
	providerInfo map[string]*ProviderInfo

	// groupVisibility records per-group column visibility overrides.
	groupVisibility map[GroupID]GroupVisibilityState

	// commandBar is the inline command/filter input widget.
	commandBar *CommandBar

	// theme is the active colour theme for rendering.
	theme *Theme

	// breadcrumb tracks the visible context (panel chain, scope, watch).
	breadcrumb *Breadcrumb

	// resourcesByKind caches resources grouped by their kind.
	resourcesByKind map[string][]Resource

	// menuCursorByGroup remembers each group's last menu cursor position.
	menuCursorByGroup map[GroupID]int

	// mouseRouter dispatches mouse events to the panel under the cursor.
	mouseRouter *MouseRouter

	// groupTabBar renders the group selector at the top of the screen.
	groupTabBar *GroupTabBar

	// groupedView composes the menu / centre / detail body for a group.
	groupedView *GroupedView

	// activeItemByGroup remembers each group's last active item.
	activeItemByGroup map[GroupID]ItemID

	// activeGroupID identifies the currently-active group.
	activeGroupID GroupID

	// groups is the ordered registered group list.
	groups []PanelGroup

	// panels is the flat list of every registered panel.
	panels []Panel

	// groupFocus indicates which pane (menu / centre / detail) owns focus.
	groupFocus FocusTarget

	// layoutOriginY is the row where the body area begins.
	layoutOriginY int

	// activePanelIndex tracks the active panel within the registered
	// list. Used by panel-cycling shortcuts; the grouped UI overrides
	// this when an item is selected via menu navigation.
	activePanelIndex int

	// height is the current terminal height in rows.
	height int

	// width is the current terminal width in columns.
	width int

	// resourceDataMutex guards resourceSummary and resourcesByKind.
	resourceDataMutex sync.RWMutex

	// showHelp toggles the help overlay.
	showHelp bool

	// quitting indicates the model is winding down on user request.
	quitting bool
}

// NewModel creates a new Model with the given settings.
//
// Takes config (*tui_dto.Config) which sets the TUI options.
//
// Returns *Model which is ready for use.
func NewModel(config *tui_dto.Config) *Model {
	configThemeName := ""
	if config != nil {
		configThemeName = config.Theme
	}
	theme := ResolveTheme(GlobalThemeRegistry(), configThemeName)

	applyThemeToLegacyGlobals(&theme)

	registry := NewCommandRegistry()
	RegisterBuiltinCommands(registry)

	model := &Model{
		lastRefresh:       time.Time{},
		lastError:         nil,
		providerInfo:      make(map[string]*ProviderInfo),
		config:            config,
		theme:             &theme,
		stateStore:        NewInMemoryPanelStateStore(),
		layoutPicker:      NewLayoutPicker(),
		paneAssigner:      NewDefaultPaneAssigner(),
		focusManager:      NewFocusManager(),
		overlays:          NewOverlayManager(&theme),
		toasts:            NewToastQueue(),
		statusBar:         NewStatusBarRenderer(&theme),
		breadcrumb:        &Breadcrumb{},
		commandRegistry:   registry,
		mouseRouter:       NewMouseRouter(),
		layoutOriginY:     0,
		resourcesByKind:   make(map[string][]Resource),
		resourceSummary:   make(map[string]map[ResourceStatus]int),
		panels:            make([]Panel, 0),
		groups:            make([]PanelGroup, 0),
		groupTabBar:       NewGroupTabBar(&theme),
		groupedView:       NewGroupedView(&theme),
		activeItemByGroup: make(map[GroupID]ItemID),
		menuCursorByGroup: make(map[GroupID]int),
		groupVisibility:   make(map[GroupID]GroupVisibilityState),
		groupFocus:        FocusCentre,
		activePanelIndex:  0,
		height:            0,
		width:             0,
		resourceDataMutex: sync.RWMutex{},
		showHelp:          false,
		quitting:          false,
	}
	model.commandBar = NewCommandBar(registry, &theme)
	return model
}

// SetGroups replaces the model's group registry.
//
// Each group's DefaultItemID becomes that group's initial active item.
// The first visible group becomes the active group; if no group is
// visible the active group is left unset.
//
// Takes groups ([]PanelGroup) which is the new ordered group list.
func (m *Model) SetGroups(groups []PanelGroup) {
	m.groups = groups
	m.activeItemByGroup = make(map[GroupID]ItemID, len(groups))
	m.menuCursorByGroup = make(map[GroupID]int, len(groups))
	m.groupVisibility = make(map[GroupID]GroupVisibilityState, len(groups))

	for _, g := range groups {
		if g == nil {
			continue
		}
		m.activeItemByGroup[g.ID()] = g.DefaultItemID()
		m.menuCursorByGroup[g.ID()] = 0
	}

	m.activeGroupID = ""
	for _, g := range groups {
		if g != nil && g.Visible() {
			m.activeGroupID = g.ID()
			break
		}
	}
}

// Groups returns the registered group list.
//
// Returns []PanelGroup which is the model's group registry.
func (m *Model) Groups() []PanelGroup { return m.groups }

// ActiveGroupID returns the identifier of the currently-active group.
//
// Returns GroupID; empty when no groups are registered or visible.
func (m *Model) ActiveGroupID() GroupID { return m.activeGroupID }

// ActiveGroup returns the currently-active PanelGroup, or nil when no
// groups are registered or none are visible.
//
// Returns PanelGroup or nil.
func (m *Model) ActiveGroup() PanelGroup {
	for _, g := range m.groups {
		if g != nil && g.ID() == m.activeGroupID {
			return g
		}
	}
	return nil
}

// ActiveItem returns the currently-active MenuItem within the active
// group, or the zero value when no group is active or the group has
// no items. Callers should check the returned MenuItem.Panel for nil
// before using it.
//
// Returns MenuItem (zero value when none).
func (m *Model) ActiveItem() MenuItem {
	g := m.ActiveGroup()
	if g == nil {
		return MenuItem{}
	}
	itemID := m.activeItemByGroup[g.ID()]
	if itemID == "" {
		itemID = g.DefaultItemID()
	}
	for _, it := range g.Items() {
		if it.ID == itemID {
			return it
		}
	}
	return MenuItem{}
}

// LayoutPicker returns the model's responsive layout picker. Tests and
// future phases (command bar, status bar) read the active breakpoint from
// here.
//
// Returns *LayoutPicker which is owned by the model.
func (m *Model) LayoutPicker() *LayoutPicker {
	return m.layoutPicker
}

// SetPaneAssigner replaces the pane assigner. Useful for services that
// want to express domain-specific panel pairings without subclassing.
//
// Takes assigner (PaneAssigner) which becomes the new assigner.
func (m *Model) SetPaneAssigner(assigner PaneAssigner) {
	if assigner == nil {
		return
	}
	m.paneAssigner = assigner
}

// PanelStateStore returns the in-memory store used to persist per-panel
// cursor, scroll, and filter state across panel switches.
//
// Returns PanelStateStore which is owned by the model.
func (m *Model) PanelStateStore() PanelStateStore {
	return m.stateStore
}

// Theme returns the active theme used to render the TUI.
//
// Returns *Theme which is the resolved theme; never nil after NewModel.
func (m *Model) Theme() *Theme {
	return m.theme
}

// SetTheme replaces the active theme and propagates it to any panel that
// implements ThemeAware. Useful for live theme switching via the command
// bar.
//
// Takes theme (*Theme) which is the new theme to apply.
func (m *Model) SetTheme(theme *Theme) {
	if theme == nil {
		return
	}
	m.theme = theme
	applyThemeToLegacyGlobals(theme)
	if m.overlays != nil {
		m.overlays.SetTheme(theme)
	}
	if m.statusBar != nil {
		m.statusBar.SetTheme(theme)
	}
	if m.commandBar != nil {
		m.commandBar.SetTheme(theme)
	}
	for _, p := range m.panels {
		if aware, ok := p.(ThemeAware); ok {
			aware.SetTheme(theme)
		}
	}
}

// Overlays returns the model's overlay manager so callers can push help
// dialogues, confirmations, or detail popups.
//
// Returns *OverlayManager which is owned by the model.
func (m *Model) Overlays() *OverlayManager {
	return m.overlays
}

// Toasts returns the model's toast queue. The refresh orchestrator pushes
// provider errors here; the status bar reads them.
//
// Returns *ToastQueue which is owned by the model.
func (m *Model) Toasts() *ToastQueue {
	return m.toasts
}

// Breadcrumb returns the model's breadcrumb so panels and the service can
// update the visible context (panel chain, scope, watch indicator).
//
// Returns *Breadcrumb which is owned by the model.
func (m *Model) Breadcrumb() *Breadcrumb {
	return m.breadcrumb
}

// CommandRegistry returns the registry of command-bar commands. Panels and
// services may register their own commands at start-up.
//
// Returns *CommandRegistry which is owned by the model.
func (m *Model) CommandRegistry() *CommandRegistry {
	return m.commandRegistry
}

// CommandBar returns the command bar so callers can programmatically open
// or close it.
//
// Returns *CommandBar which is owned by the model.
func (m *Model) CommandBar() *CommandBar {
	return m.commandBar
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

	m.resourceSummary = summary
	m.resourcesByKind = resources
}

// AddPanel adds a panel to the model.
//
// Takes p (Panel) which is the panel to add. The first panel added receives
// focus automatically. Panels implementing ThemeAware receive the active
// theme during this call.
func (m *Model) AddPanel(p Panel) {
	m.panels = append(m.panels, p)
	if aware, ok := p.(ThemeAware); ok && m.theme != nil {
		aware.SetTheme(m.theme)
	}
	if len(m.panels) == 1 {
		p.SetFocused(true)
	}
	if m.focusManager != nil {
		m.focusManager.SetPanels(m.panels)
		if len(m.panels) == 1 {
			m.focusManager.SetActive(p.ID())
		}
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
	if cmds, handled := m.routeToCommandBar(message); handled {
		return m, tea.Batch(cmds...)
	}
	if cmds, handled := m.routeToOverlay(message); handled {
		return m, tea.Batch(cmds...)
	}

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
	switch {
	case m.quitting:
		content = "Goodbye!\n"
	case m.width == 0 || m.height == 0:
		content = "Initialising..."
	default:
		content = m.renderLayout()
		if m.overlays != nil && !m.overlays.Empty() {
			content = m.overlays.Render(content, m.width, m.height)
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// routeToCommandBar dispatches the message to the active command bar
// when one is open. Resize messages additionally propagate to the
// underlying layout so panels keep their sizes in sync.
//
// Takes message (tea.Msg) which is the incoming message.
//
// Returns []tea.Cmd which is the resulting command list.
// Returns bool which is true when the message was consumed.
func (m *Model) routeToCommandBar(message tea.Msg) ([]tea.Cmd, bool) {
	if m.commandBar == nil || !m.commandBar.Active() {
		return nil, false
	}
	cmd := m.commandBar.Update(message, m)
	cmds := []tea.Cmd{}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if _, isResize := message.(tea.WindowSizeMsg); isResize {
		cmds = append(cmds, m.dispatchMessage(message)...)
	}
	return cmds, true
}

// routeToOverlay dispatches the message to the top overlay on the
// stack. Resize messages additionally propagate to the underlying
// layout so panels keep their sizes in sync; other messages are
// consumed by the overlay.
//
// Takes message (tea.Msg) which is the incoming message.
//
// Returns []tea.Cmd which is the resulting command list.
// Returns bool which is true when the message was consumed.
func (m *Model) routeToOverlay(message tea.Msg) ([]tea.Cmd, bool) {
	if m.overlays == nil || m.overlays.Empty() {
		return nil, false
	}
	cmd, consumed := m.overlays.Update(message)
	if !consumed {
		return nil, false
	}
	cmds := []tea.Cmd{}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if _, isResize := message.(tea.WindowSizeMsg); isResize {
		cmds = append(cmds, m.dispatchMessage(message)...)
	}
	return cmds, true
}

// handleKeyMessage processes keyboard input and returns the matching command.
//
// Takes message (tea.KeyPressMsg) which contains the key event to process.
//
// Returns tea.Cmd which is the command to run, or nil if the key is not
// handled.
func (m *Model) handleKeyMessage(message tea.KeyPressMsg) tea.Cmd {
	if cmd, handled := m.handleGroupedKey(message); handled {
		return cmd
	}

	switch message.String() {
	case "q", "ctrl+c":
		return func() tea.Msg { return quitMessage{} }

	case "?":
		return func() tea.Msg { return toggleHelpMessage{} }

	case ":":
		return m.openCommandBarCmd(CommandModeCommand)

	case "/":
		return m.openCommandBarCmd(CommandModeFilter)

	case "r":
		return func() tea.Msg { return forceRefreshMessage{} }
	}

	return nil
}

// handleGroupedKey handles keys that have group-aware semantics. The
// keys consumed here are: F1-F4 (switch top group), 1-9 / 0 (jump to
// menu item by hotkey), [ / ] (toggle left / right column visibility),
// and tab / shift+tab (cycle focus between menu / centre / detail).
//
// Up/Down/j/k are consumed only when focus is on the menu (where they
// move the menu cursor with auto-commit). When focus is on the centre
// or detail, Up/Down pass through to the focused pane so the user can
// scroll the list inside it.
//
// Takes message (tea.KeyPressMsg) which is the key event.
//
// Returns tea.Cmd which is the resulting command, may be nil.
// Returns bool which is true when the key was consumed by the grouped
// layer so the caller does not propagate it to the flat keymap.
func (m *Model) handleGroupedKey(message tea.KeyPressMsg) (tea.Cmd, bool) {
	key := message.String()

	switch key {
	case "f1", "f2", "f3", "f4":
		m.activateGroupByFKey(key)
		return nil, true
	case "alt+left":
		m.cycleGroup(-1)
		return nil, true
	case "alt+right":
		m.cycleGroup(+1)
		return nil, true
	case "alt+up":
		m.moveMenuCursor(-1)
		m.commitMenuCursor()
		return nil, true
	case "alt+down":
		m.moveMenuCursor(+1)
		m.commitMenuCursor()
		return nil, true
	case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0",
		"shift+1", "shift+2", "shift+3", "shift+4", "shift+5",
		"shift+6", "shift+7", "shift+8", "shift+9", "shift+0":
		m.activateItemByHotkey(key)
		return nil, true
	case "j", "down", "k", "up":

		if m.groupFocus != FocusMenu {
			return nil, false
		}
		direction := +1
		if key == "k" || key == "up" {
			direction = -1
		}
		m.moveMenuCursor(direction)
		m.commitMenuCursor()
		return nil, true
	case "[":
		m.toggleMenuColumn()
		return nil, true
	case "]":
		m.toggleDetailColumn()
		return nil, true
	case "tab":
		m.cycleFocus(+1)
		return nil, true
	case "shift+tab":
		m.cycleFocus(-1)
		return nil, true
	}
	return nil, false
}

// cycleGroup advances the active group by direction positions among
// visible groups, wrapping at the ends. Used by Alt+Left/Right so
// users can step through groups regardless of which pane has focus.
//
// Takes direction (int) which is -1 (previous) or +1 (next).
func (m *Model) cycleGroup(direction int) {
	visible := make([]PanelGroup, 0, len(m.groups))
	for _, g := range m.groups {
		if g != nil && g.Visible() {
			visible = append(visible, g)
		}
	}
	if len(visible) == 0 {
		return
	}
	index := 0
	for i, g := range visible {
		if g.ID() == m.activeGroupID {
			index = i
			break
		}
	}
	index = (index + direction + len(visible)) % len(visible)
	m.activeGroupID = visible[index].ID()
}

// activateGroupByFKey switches to the group keyed by the supplied
// function-key string ("f1" -> group with Hotkey '1', etc.). No-op when
// no matching group exists.
//
// Takes key (string) which is "f1"/"f2"/"f3"/"f4".
func (m *Model) activateGroupByFKey(key string) {
	if len(key) < 2 {
		return
	}
	target := rune(key[1])
	for _, g := range m.groups {
		if g != nil && g.Visible() && g.Hotkey() == target {
			m.activeGroupID = g.ID()
			return
		}
	}
}

// activateItemByHotkey activates the menu item whose Hotkey matches
// key within the active group. No-op when no matching item exists or
// no group is active.
//
// Takes key (string) which is the pressed-rune string.
func (m *Model) activateItemByHotkey(key string) {
	if key == "" {
		return
	}
	g := m.ActiveGroup()
	if g == nil {
		return
	}
	for i, it := range g.Items() {
		if it.Hotkey == key {
			m.activeItemByGroup[g.ID()] = it.ID
			m.menuCursorByGroup[g.ID()] = i
			return
		}
	}
}

// moveMenuCursor moves the highlighted-but-not-active cursor in the
// active group's menu by delta positions, clamping to the item range.
//
// Takes delta (int) which is the cursor movement (-1 or +1).
func (m *Model) moveMenuCursor(delta int) {
	g := m.ActiveGroup()
	if g == nil {
		return
	}
	items := g.Items()
	if len(items) == 0 {
		return
	}
	cur := m.menuCursorByGroup[g.ID()]
	cur += delta
	cur = max(0, min(cur, len(items)-1))
	m.menuCursorByGroup[g.ID()] = cur
}

// commitMenuCursor activates the item at the current menu cursor.
func (m *Model) commitMenuCursor() {
	g := m.ActiveGroup()
	if g == nil {
		return
	}
	items := g.Items()
	cur := m.menuCursorByGroup[g.ID()]
	if cur < 0 || cur >= len(items) {
		return
	}
	if items[cur].Panel != nil {
		m.activeItemByGroup[g.ID()] = items[cur].ID
	}
}

// toggleMenuColumn flips the per-group left-column visibility override.
// The first toggle inverts the breakpoint default; subsequent toggles
// flip the explicit override.
func (m *Model) toggleMenuColumn() {
	g := m.ActiveGroup()
	if g == nil {
		return
	}
	v := m.groupVisibility[g.ID()]
	bp := PickBreakpoint(DefaultBreakpoints, m.width, m.height)
	current := columnVisible(v.LeftOverride, bp.ShowsLeftByDefault)
	v.LeftOverride = new(!current)
	m.groupVisibility[g.ID()] = v
}

// toggleDetailColumn flips the per-group right-column visibility
// override.
func (m *Model) toggleDetailColumn() {
	g := m.ActiveGroup()
	if g == nil {
		return
	}
	v := m.groupVisibility[g.ID()]
	bp := PickBreakpoint(DefaultBreakpoints, m.width, m.height)
	current := columnVisible(v.RightOverride, bp.ShowsRightByDefault)
	v.RightOverride = new(!current)
	m.groupVisibility[g.ID()] = v
}

// cycleFocus rotates focus between centre and detail (and menu, when
// the menu is visible). Direction +1 cycles forward, -1 backward.
//
// Takes direction (int) which is +1 or -1.
func (m *Model) cycleFocus(direction int) {
	g := m.ActiveGroup()
	if g == nil {
		return
	}
	v := m.groupVisibility[g.ID()]
	bp := PickBreakpoint(DefaultBreakpoints, m.width, m.height)
	rightVisible := columnVisible(v.RightOverride, bp.ShowsRightByDefault)
	leftVisible := columnVisible(v.LeftOverride, bp.ShowsLeftByDefault)

	stops := []FocusTarget{FocusCentre}
	if rightVisible {
		stops = append(stops, FocusDetail)
	}
	if leftVisible {
		stops = append(stops, FocusMenu)
	}

	index := 0
	for i, s := range stops {
		if s == m.groupFocus {
			index = i
			break
		}
	}
	index = (index + direction + len(stops)) % len(stops)
	m.groupFocus = stops[index]
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
	m.pushDataToPanel(m.panels[index])

	if m.focusManager != nil {
		m.focusManager.SetPanels(m.panels)
		m.focusManager.SetActive(m.panels[index].ID())
	}
}

// handleMouseClick routes a click event through the MouseRouter; clicks
// on a pane focus that pane.
//
// Takes mouse (tea.Mouse) which carries the click coordinates and button.
func (m *Model) handleMouseClick(mouse tea.Mouse) {
	if m.mouseRouter == nil {
		return
	}
	target, ok := m.mouseRouter.Hit(mouse.X, mouse.Y)
	if !ok {
		return
	}
	switch target.Kind {
	case MouseTargetPane, MouseTargetTab:
		if target.PanelID != "" {
			m.focusPanelByID(target.PanelID)
		}
	}
}

// handleMouseWheel translates a wheel event into a synthetic up/down key
// press dispatched to the panel under the pointer.
//
// Takes mouse (tea.Mouse) which carries the wheel direction and position.
//
// Returns []tea.Cmd which is the command(s) produced by the panel's
// Update method.
func (m *Model) handleMouseWheel(mouse tea.Mouse) []tea.Cmd {
	if m.mouseRouter == nil {
		return nil
	}
	target, ok := m.mouseRouter.Hit(mouse.X, mouse.Y)
	if !ok || target.Kind != MouseTargetPane || target.PanelID == "" {
		return nil
	}

	var key tea.KeyPressMsg
	switch mouse.Button {
	case tea.MouseWheelUp:
		key = tea.KeyPressMsg{Code: 'k'}
	case tea.MouseWheelDown:
		key = tea.KeyPressMsg{Code: 'j'}
	default:
		return nil
	}

	for i, p := range m.panels {
		if p.ID() != target.PanelID {
			continue
		}
		cmds := make([]tea.Cmd, 0, mouseWheelStepCount)
		for range mouseWheelStepCount {
			updated, cmd := p.Update(key)
			m.panels[i] = updated
			p = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return cmds
	}
	return nil
}

// openCommandBarCmd returns a tea.Cmd that opens the command bar in the
// supplied mode. The Cmd is wrapped in a closure so that the underlying
// textinput's blink command is initiated on the bubbletea dispatch loop.
//
// Takes mode (CommandBarMode) which selects the bar's prompt and routing.
//
// Returns tea.Cmd which opens the bar and returns the cursor-blink command.
func (m *Model) openCommandBarCmd(mode CommandBarMode) tea.Cmd {
	if m.commandBar == nil {
		return nil
	}
	return m.commandBar.Open(mode)
}

// toggleHelp opens the help overlay if it is not currently on top of the
// stack, or pops it if it is. Called by the toggleHelpMessage handler.
func (m *Model) toggleHelp() {
	if m.overlays == nil {
		m.showHelp = !m.showHelp
		return
	}

	if top := m.overlays.Top(); top != nil && top.ID() == "help" {
		m.overlays.Pop()
		return
	}

	var panelTitle string
	var panelKeys []KeyBinding
	if active := m.ActivePanel(); active != nil {
		panelTitle = active.Title()
		panelKeys = active.KeyMap()
	}

	var commands []Command
	if m.commandRegistry != nil {
		commands = m.commandRegistry.Commands()
	}
	overlay := NewHelpOverlay(m.theme, GlobalKeyBindings(), panelTitle, panelKeys, commands)
	m.overlays.Push(overlay)
}

// focusVisiblePanel moves focus to the next or previous visible panel as
// determined by the layout's pane assignments.
//
// Takes direction (int) which is +1 for next or -1 for previous. Other
// values are treated as no-ops.
func (m *Model) focusVisiblePanel(direction int) {
	if m.focusManager == nil || len(m.panels) == 0 {
		return
	}

	var nextID string
	switch {
	case direction > 0:
		nextID = m.focusManager.NextVisible()
	case direction < 0:
		nextID = m.focusManager.PrevVisible()
	default:
		return
	}
	if nextID == "" {
		return
	}
	m.focusPanelByID(nextID)
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
		m.pushDataToPanelLocked(panel)
	}
}

// pushDataToPanel sends the stored data to a single panel.
//
// Takes panel (Panel) which is the target panel to receive the cached data.
//
// Concurrency: acquires a read lock on resourceDataMutex.
func (m *Model) pushDataToPanel(panel Panel) {
	m.resourceDataMutex.RLock()
	defer m.resourceDataMutex.RUnlock()

	m.pushDataToPanelLocked(panel)
}

// pushDataToPanelLocked sends the stored data to a single panel. The caller
// must hold resourceDataMutex.
//
// Takes panel (Panel) which is the target panel to receive the cached data.
func (m *Model) pushDataToPanelLocked(panel Panel) {
	m.pushSummaryToPanel(panel)
	m.pushArtefactsToPanel(panel)
	m.pushTasksToPanel(panel)
	m.pushWorkflowsToPanel(panel)
	m.pushResourcesToPanel(panel)
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

// renderLayout renders the full terminal interface layout. The body is
// produced by the active responsive layout (single, two-column, or
// three-column) using assignments from the pane assigner.
//
// Returns string which contains the formatted terminal output.
func (m *Model) renderLayout() string {
	if m.showHelp {
		return m.renderHelp()
	}

	contentHeight := max(1, m.height-LayoutChromeHeight)
	header := m.renderHeader()
	headerLines := strings.Count(header, "\n") + 1
	m.layoutOriginY = headerLines
	if m.mouseRouter != nil {
		m.mouseRouter.Reset()
	}

	body := m.renderGroupedBody(m.width, contentHeight)
	statusBar := m.renderStatusBar()
	return header + "\n" + body + "\n" + statusBar
}

// renderGroupedBody composes the menu / centre / detail layout for the
// active group via the GroupedView renderer. Falls back to a placeholder
// when no groups are registered or visible.
//
// Takes width (int) and height (int) which are the layout area
// dimensions (chrome already subtracted).
//
// Returns string with the composed body.
func (m *Model) renderGroupedBody(width, height int) string {
	if m.groupedView == nil || len(m.groups) == 0 {
		return strings.Repeat(" ", width)
	}

	group := m.ActiveGroup()
	if group == nil {
		return PadRightANSI("no groups available", width)
	}

	item := m.ActiveItem()
	if item.Panel == nil {
		return PadRightANSI("group has no items", width)
	}

	m.syncItemFocus(group, item)

	bp := PickBreakpoint(DefaultBreakpoints, width, height)

	cursor := m.menuCursorByGroup[group.ID()]
	visibility := m.groupVisibility[group.ID()]

	return m.groupedView.Compose(GroupedViewArgs{
		Theme:      m.theme,
		Group:      group,
		Item:       item,
		MenuCursor: cursor,
		Focus:      m.groupFocus,
		Visibility: visibility,
		Breakpoint: bp,
		Width:      width,
		Height:     height,
	})
}

// syncItemFocus pushes the current focus state down to every panel in
// the active group. Inactive panels receive Focused(false) so their
// internal styling drops the focus indicator; the active panel
// receives Focused(true) when the centre owns focus.
//
// Detail focus is currently a no-op because every panel renders both
// centre and detail itself; if a future split introduces independent
// detail panels they should be threaded here.
//
// Takes group (PanelGroup) which is the active group.
// Takes active (MenuItem) which is the active item.
func (m *Model) syncItemFocus(group PanelGroup, active MenuItem) {
	for _, it := range group.Items() {
		if it.Panel == nil {
			continue
		}
		isActive := it.ID == active.ID
		it.Panel.SetFocused(isActive && m.groupFocus == FocusCentre)
	}
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

// renderTabBar renders the group tab bar at the top of the screen.
//
// Returns string which contains the formatted tab bar with styled hotkeys.
func (m *Model) renderTabBar() string {
	if m.groupTabBar == nil || len(m.groups) == 0 {
		return ""
	}
	width := m.width
	if width <= 0 {
		width = defaultRenderWidth
	}
	return m.groupTabBar.Render(m.groups, m.activeGroupID, width)
}

// renderStatusBar builds the status bar text. When the command bar is
// active it takes over the status row so the user can see what they are
// typing.
//
// Returns string which contains the styled status bar showing either the
// active command bar, an error message, or help text for keyboard
// shortcuts.
func (m *Model) renderStatusBar() string {
	if m.commandBar != nil && m.commandBar.Active() {
		width := m.width
		if width <= 0 {
			width = defaultRenderWidth
		}
		return m.commandBar.View(width)
	}
	if m.lastError != nil {
		return statusBarStyle.Render("Error: " + m.lastError.Error())
	}
	status := "F1-F4 group | 1-9 item | ↑/↓ navigate | [/] toggle panes | / search | ? help | q quit"
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
	if cmds, handled := m.dispatchSystemMessage(message); handled {
		return cmds
	}
	if cmds, handled := m.dispatchFocusMessage(message); handled {
		return cmds
	}
	return m.dispatchOverlayMessage(message)
}

// dispatchSystemMessage routes data / lifecycle / input messages to
// their handlers. Returns (commands, true) when handled.
//
// Takes message (tea.Msg) which is the incoming message.
//
// Returns []tea.Cmd which is the resulting command list.
// Returns bool which is true when the message was handled.
func (m *Model) dispatchSystemMessage(message tea.Msg) ([]tea.Cmd, bool) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		if command := m.handleKeyMessage(message); command != nil {
			return []tea.Cmd{command}, true
		}
		return nil, true
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		if m.layoutPicker != nil {
			contentHeight := max(1, m.height-LayoutChromeHeight)
			m.layoutPicker.Reflow(m.width, contentHeight)
		}
		return nil, true
	case tickMessage:
		return m.handleTickMessage(message), true
	case dataRefreshedMessage:
		m.handleDataRefreshedMessage(message)
		return nil, true
	case dataUpdatedMessage:
		return m.handleDataUpdatedMessage(message), true
	case providerStatusMessage:
		m.handleProviderStatusMessage(message)
		return nil, true
	case errorMessage:
		m.lastError = message.err
		return nil, true
	case quitMessage:
		m.quitting = true
		return nil, true
	}
	return nil, false
}

// dispatchFocusMessage routes panel-focus and mouse messages.
// Returns (commands, true) when handled.
//
// Takes message (tea.Msg) which is the incoming message.
//
// Returns []tea.Cmd which is the resulting command list.
// Returns bool which is true when the message was handled.
func (m *Model) dispatchFocusMessage(message tea.Msg) ([]tea.Cmd, bool) {
	switch message := message.(type) {
	case focusPanelMessage:
		m.focusPanelByID(message.panelID)
	case nextPanelMessage:
		m.focusNextPanel()
	case previousPanelMessage:
		m.focusPreviousPanel()
	case nextVisiblePanelMessage:
		m.focusVisiblePanel(+1)
	case previousVisiblePanelMessage:
		m.focusVisiblePanel(-1)
	case tea.MouseClickMsg:
		m.handleMouseClick(message.Mouse())
	case tea.MouseWheelMsg:
		return m.handleMouseWheel(message.Mouse()), true
	default:
		return nil, false
	}
	return nil, true
}

// dispatchOverlayMessage routes overlay-stack messages and the help
// toggle. Returns commands (always nil here, but the signature
// matches the others for symmetry).
//
// Takes message (tea.Msg) which is the incoming message.
//
// Returns []tea.Cmd which is always nil here, kept for symmetry.
func (m *Model) dispatchOverlayMessage(message tea.Msg) []tea.Cmd {
	switch message := message.(type) {
	case toggleHelpMessage:
		m.toggleHelp()
	case pushOverlayMessage:
		if m.overlays != nil && message.Overlay != nil {
			m.overlays.Push(message.Overlay)
		}
	case popOverlayMessage:
		if m.overlays != nil {
			if top := m.overlays.Top(); top != nil && top.ID() == message.ID {
				m.overlays.Pop()
			}
		}
	}
	return nil
}

// handleTickMessage handles tick messages for regular updates.
// Iterates all panels on the main goroutine to avoid data races and collects
// any commands they return.
//
// Takes message (tickMessage) which contains the tick time.
//
// Returns []tea.Cmd which contains the next tick command plus any commands
// returned by panels.
func (m *Model) handleTickMessage(message tickMessage) []tea.Cmd {
	m.lastRefresh = message.time

	cmds := []tea.Cmd{m.tickCmd()}

	tick := TickMessage{Time: message.time}
	for i, panel := range m.panels {
		updatedPanel, command := panel.Update(tick)
		m.panels[i] = updatedPanel
		if command != nil {
			cmds = append(cmds, command)
		}
	}

	return cmds
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

// updateActivePanel sends a message to the active group's centre and
// detail panels. Only the active item's children receive messages so
// off-screen panels do not pay the per-tick cost.
//
// Takes message (tea.Msg) which is the message to send.
//
// Returns tea.Cmd which is the batched commands from centre + detail.
func (m *Model) updateActivePanel(message tea.Msg) tea.Cmd {
	if m.showHelp {
		return nil
	}
	item := m.ActiveItem()
	if item.Panel == nil {
		return nil
	}
	updated, cmd := item.Panel.Update(message)

	_ = updated
	return cmd
}
