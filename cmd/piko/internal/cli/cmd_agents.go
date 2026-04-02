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

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/cmd/piko/internal/wizard/templates"
	"piko.sh/piko/cmd/piko/internal/wizardbase"
	"piko.sh/piko/wdk/safedisk"

	tea "charm.land/bubbletea/v2"
)

const (
	// agentsClaudeDir is the Claude config subdirectory name.
	agentsClaudeDir = ".claude"

	// agentsSkillsDir is the skills subdirectory name.
	agentsSkillsDir = "skills"

	// agentsPikoDir is the piko subdirectory name within skills.
	agentsPikoDir = "piko"

	// agentsAgentsMDFilename is the name of the project-level agents file.
	agentsAgentsMDFilename = "AGENTS.md"

	// agentsFilePerms is the file permission used when writing gitignore files.
	agentsFilePerms = 0o600

	// agentsErrorFmt is the format string for error output messages.
	agentsErrorFmt = "\nError: %v\n"

	// agentsTargetIdxAgentsMD is the index of the AGENTS.md target.
	agentsTargetIdxAgentsMD = 1

	// gitignoreEntries is the block appended to .gitignore when the user opts in.
	gitignoreEntries = `
# AI agent integration (regenerate with: piko agents install)
AGENTS.md
references/
`
)

const (
	// agentsStepSelect is the step where the user picks which AI tools to
	// configure.
	agentsStepSelect = iota

	// agentsStepInstalling is the step where files are being copied.
	agentsStepInstalling

	// agentsStepGitignore asks whether to add agent files to .gitignore.
	// Only shown when project-level AGENTS.md was installed.
	agentsStepGitignore

	// agentsStepDone is the final step showing a summary before exit.
	agentsStepDone
)

const (
	// agentsUninstallStepSelect is the step where the user picks which
	// installed agents to remove.
	agentsUninstallStepSelect = iota

	// agentsUninstallStepRemoving is the step where files are being deleted.
	agentsUninstallStepRemoving

	// agentsUninstallStepGitignore asks whether to remove agent entries from
	// .gitignore. Only shown when AGENTS.md was uninstalled and .gitignore
	// contains agent entries.
	agentsUninstallStepGitignore

	// agentsUninstallStepDone is the final step showing a summary before
	// exit.
	agentsUninstallStepDone
)

type (
	// agentsInstallDoneMessage signals that the install operation has completed.
	agentsInstallDoneMessage struct {
		// results holds summary lines from the install step.
		results []string
	}

	// agentsErrMessage wraps an error for use in the Bubble Tea message system.
	agentsErrMessage struct {
		// err is the underlying error that occurred.
		err error
	}
)

// agentTarget describes an AI tool that can be configured with Piko knowledge.
type agentTarget struct {
	// install copies the relevant files to the correct location.
	install func() (string, error)

	// name is the display name shown in the TUI.
	name string

	// destDesc is the destination path shown alongside the name.
	destDesc string

	// scope describes where files are installed and which tools benefit.
	scope string
}

// agentsModel holds the state for the interactive agents installer TUI.
type agentsModel struct {
	// err holds any error that occurred.
	err error

	// factory creates sandboxes for filesystem access.
	factory safedisk.Factory

	// targets holds the available AI tools to configure.
	targets []agentTarget

	// results holds summary lines after installation.
	results []string

	wizardbase.WizardBase

	// gitignoreCursor tracks the selected option in the gitignore step
	// (0 = Yes, 1 = No).
	gitignoreCursor int

	// gitignoreUpdated records whether .gitignore was modified.
	gitignoreUpdated bool
}

// Init returns the initial command to start the model.
//
// Returns tea.Cmd which starts the spinner tick.
func (m *agentsModel) Init() tea.Cmd {
	return m.Spinner.Tick
}

// Update handles incoming messages and updates the model state.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns tea.Model which is the updated model after processing.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (m *agentsModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyMessage(message)
	case agentsErrMessage:
		m.err = message.err
		return m, tea.Quit
	case agentsInstallDoneMessage:
		m.results = message.results
		if m.Selected[agentsTargetIdxAgentsMD] {
			m.Step = agentsStepGitignore
			m.Cursor = 0
			return m, nil
		}
		m.Step = agentsStepDone
		return m, tea.Quit
	}

	if m.Step == agentsStepInstalling {
		command := m.UpdateSpinner(message)
		return m, command
	}

	return m, nil
}

// View renders the current state of the user interface.
//
// Returns tea.View which contains the formatted terminal output.
func (m *agentsModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("\nError: %v\n", m.err))
	}

	var s strings.Builder

	switch m.Step {
	case agentsStepSelect:
		s.WriteString(wizardbase.TitleStyle.Render("Configure AI coding tools with Piko framework knowledge.") + "\n\n")
		s.WriteString("Which tools would you like to configure?\n\n")
		items := make([]wizardbase.CheckboxItem, len(m.targets))
		for i, t := range m.targets {
			items[i] = wizardbase.CheckboxItem{
				Label:    fmt.Sprintf("%-16s %s  (%s)", t.name, t.destDesc, t.scope),
				Selected: m.Selected[i],
			}
		}
		s.WriteString(wizardbase.RenderCheckboxList(items, m.Cursor))

	case agentsStepInstalling:
		s.WriteString(wizardbase.RenderSpinnerLine(m.Spinner.View(), "Installing agent files..."))

	case agentsStepGitignore:
		s.WriteString(wizardbase.TitleStyle.Render("Add agent files to .gitignore?") + "\n")
		s.WriteString(wizardbase.HelpStyle.Render("These files can be regenerated with 'piko agents install'.") + "\n\n")
		s.WriteString(wizardbase.RenderYesNo(m.gitignoreCursor))

	case agentsStepDone:
		s.WriteString(wizardbase.SuccessStyle.Render("Done!") + "\n\n")
		for _, r := range m.results {
			s.WriteString("  " + r + "\n")
		}
		if m.gitignoreUpdated {
			s.WriteString("  .gitignore     updated\n")
		}
		s.WriteString("\nRun 'piko agents install' after upgrading Piko to refresh these files.\n")
	}

	if m.Step == agentsStepSelect {
		s.WriteString("\n" + wizardbase.HelpStyle.Render("Use arrow keys to navigate, space to toggle, enter to confirm, ctrl+c to quit."))
	}
	if m.Step == agentsStepGitignore {
		s.WriteString("\n" + wizardbase.HelpStyle.Render("Use arrow keys to navigate, enter to confirm."))
	}

	return tea.NewView(s.String())
}

// handleKeyMessage processes keyboard input messages.
//
// Takes message (tea.KeyPressMsg) which holds the key event to process.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is the command to run, or nil if none.
func (m *agentsModel) handleKeyMessage(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c", "esc":
		command := m.HandleAbort()
		return m, command
	case "enter":
		if m.Step == agentsStepDone {
			return m, tea.Quit
		}
		return m.handleEnter()
	case "up", "k":
		if m.Step == agentsStepGitignore {
			if m.gitignoreCursor > 0 {
				m.gitignoreCursor--
			}
		} else {
			m.HandleNavigation(message, len(m.targets))
		}
	case "down", "j":
		if m.Step == agentsStepGitignore {
			if m.gitignoreCursor < 1 {
				m.gitignoreCursor++
			}
		} else {
			m.HandleNavigation(message, len(m.targets))
		}
	case "space":
		if m.Step == agentsStepSelect {
			m.HandleToggle()
		}
	}
	return m, nil
}

// handleEnter processes the enter key press at each step. In the select step,
// enter toggles a checkbox when the cursor is on a tool item, and only
// confirms when the cursor is on the "Continue" button.
//
// Returns tea.Model which is the updated model after processing.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (m *agentsModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.Step {
	case agentsStepSelect:
		if m.HandleToggle() {
			return m, nil
		}
		return m.handleSelectConfirm()
	case agentsStepGitignore:
		return m.handleGitignoreConfirm()
	}
	return m, nil
}

// runSelectedTargetOperations iterates over selected indices and calls operate
// for each. Returns the collected result strings, or the first error encountered.
//
// Takes count (int) which is the total number of targets.
// Takes selected ([]bool) which marks which indices are active.
// Takes operate (func(int) (string, error)) which performs the operation for a
// given index.
//
// Returns []string which holds one summary line per operated target.
// Returns error when any single operation fails.
func runSelectedTargetOperations(count int, selected []bool, operate func(int) (string, error)) ([]string, error) {
	var results []string
	for i := range count {
		if !selected[i] {
			continue
		}
		result, err := operate(i)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// handleSelectConfirm validates the tool selection and starts installation.
//
// Returns tea.Model which is the updated model.
// Returns tea.Cmd which runs the installation or nil on cancellation.
//
//nolint:dupl // similar to agentsUninstallModel.handleSelectConfirm
func (m *agentsModel) handleSelectConfirm() (tea.Model, tea.Cmd) {
	if !m.AnySelected() {
		m.Aborted = true
		return m, tea.Quit
	}

	m.Step = agentsStepInstalling

	return m, func() tea.Msg {
		results, err := runSelectedTargetOperations(len(m.targets), m.Selected, func(i int) (string, error) {
			return m.targets[i].install()
		})
		if err != nil {
			return agentsErrMessage{err}
		}
		return agentsInstallDoneMessage{results: results}
	}
}

// handleGitignoreConfirm processes the yes/no choice for updating .gitignore.
//
// Returns tea.Model which is the updated model.
// Returns tea.Cmd which is tea.Quit.
func (m *agentsModel) handleGitignoreConfirm() (tea.Model, tea.Cmd) {
	if m.gitignoreCursor == 0 {
		builder, err := m.factory.Create("agent-gitignore-add", ".", safedisk.ModeReadWrite)
		if err != nil {
			m.err = fmt.Errorf("failed to create sandbox: %w", err)
			return m, tea.Quit
		}
		defer builder.Close()

		if err := appendGitignoreEntries(builder); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.gitignoreUpdated = true
	}

	m.Step = agentsStepDone
	return m, tea.Quit
}

// agentUninstallTarget describes an installed agent that can be removed.
type agentUninstallTarget struct {
	// uninstall removes the relevant files and returns a summary line.
	uninstall func() (string, error)

	// name is the display name shown in the TUI.
	name string

	// location is the path shown alongside the name.
	location string
}

// agentsUninstallDoneMessage signals that the uninstall operation has completed.
type agentsUninstallDoneMessage struct {
	// results holds summary lines from the uninstall step.
	results []string
}

// agentsUninstallModel holds the state for the interactive agents uninstaller
// TUI.
type agentsUninstallModel struct {
	// err holds any error that occurred.
	err error

	// factory creates sandboxes for filesystem access.
	factory safedisk.Factory

	// targets holds the installed agents available for removal.
	targets []agentUninstallTarget

	// results holds summary lines after uninstallation.
	results []string

	wizardbase.WizardBase

	// gitignoreCursor tracks the selected option in the gitignore step
	// (0 = Yes, 1 = No).
	gitignoreCursor int

	// gitignoreUpdated records whether .gitignore was modified.
	gitignoreUpdated bool

	// hasGitignoreEntries indicates whether .gitignore contains agent
	// entries that can be removed.
	hasGitignoreEntries bool
}

// Init returns the initial command to start the model.
//
// Returns tea.Cmd which starts the spinner tick.
func (m *agentsUninstallModel) Init() tea.Cmd {
	return m.Spinner.Tick
}

// Update handles incoming messages and updates the model state.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns tea.Model which is the updated model after processing.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (m *agentsUninstallModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyMessage(message)
	case agentsErrMessage:
		m.err = message.err
		return m, tea.Quit
	case agentsUninstallDoneMessage:
		m.results = message.results
		if m.uninstalledAgentsMD() && m.hasGitignoreEntries {
			m.Step = agentsUninstallStepGitignore
			m.Cursor = 0
			return m, nil
		}
		m.Step = agentsUninstallStepDone
		return m, tea.Quit
	}

	if m.Step == agentsUninstallStepRemoving {
		command := m.UpdateSpinner(message)
		return m, command
	}

	return m, nil
}

// View renders the current state of the user interface.
//
// Returns tea.View which contains the formatted terminal output.
func (m *agentsUninstallModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("\nError: %v\n", m.err))
	}

	var s strings.Builder

	switch m.Step {
	case agentsUninstallStepSelect:
		if len(m.targets) == 0 {
			s.WriteString("No agent integrations are currently installed.\n")
			return tea.NewView(s.String())
		}

		s.WriteString(wizardbase.TitleStyle.Render("Remove AI coding tool integrations.") + "\n\n")
		s.WriteString("Which integrations would you like to remove?\n\n")
		items := make([]wizardbase.CheckboxItem, len(m.targets))
		for i, t := range m.targets {
			items[i] = wizardbase.CheckboxItem{
				Label:    fmt.Sprintf("%-16s %s", t.name, t.location),
				Selected: m.Selected[i],
			}
		}
		s.WriteString(wizardbase.RenderCheckboxList(items, m.Cursor))

	case agentsUninstallStepRemoving:
		s.WriteString(wizardbase.RenderSpinnerLine(m.Spinner.View(), "Removing agent files..."))

	case agentsUninstallStepGitignore:
		s.WriteString(wizardbase.TitleStyle.Render("Remove agent entries from .gitignore?") + "\n\n")
		s.WriteString(wizardbase.RenderYesNo(m.gitignoreCursor))

	case agentsUninstallStepDone:
		s.WriteString(wizardbase.SuccessStyle.Render("Done!") + "\n\n")
		for _, r := range m.results {
			s.WriteString("  " + r + "\n")
		}
		if m.gitignoreUpdated {
			s.WriteString("  .gitignore     updated\n")
		}
	}

	if m.Step == agentsUninstallStepSelect && len(m.targets) > 0 {
		s.WriteString("\n" + wizardbase.HelpStyle.Render("Use arrow keys to navigate, space to toggle, enter to confirm, ctrl+c to quit."))
	}
	if m.Step == agentsUninstallStepGitignore {
		s.WriteString("\n" + wizardbase.HelpStyle.Render("Use arrow keys to navigate, enter to confirm."))
	}

	return tea.NewView(s.String())
}

// uninstalledAgentsMD returns true if the AGENTS.md target was selected
// for removal.
//
// Returns bool which is true when AGENTS.md was selected.
func (m *agentsUninstallModel) uninstalledAgentsMD() bool {
	for i, t := range m.targets {
		if t.name == agentsAgentsMDFilename && m.Selected[i] {
			return true
		}
	}
	return false
}

// handleKeyMessage processes keyboard input messages.
//
// Takes message (tea.KeyPressMsg) which holds the key event to process.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is the command to run, or nil if none.
func (m *agentsUninstallModel) handleKeyMessage(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c", "esc":
		command := m.HandleAbort()
		return m, command
	case "enter":
		return m.handleUninstallEnterKey()
	case "up", "k":
		m.navigateUninstall(message, -1)
	case "down", "j":
		m.navigateUninstall(message, 1)
	case "space":
		if m.Step == agentsUninstallStepSelect {
			m.HandleToggle()
		}
	}
	return m, nil
}

// handleUninstallEnterKey handles the enter key for uninstall
// steps.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is the command to run, or nil if none.
func (m *agentsUninstallModel) handleUninstallEnterKey() (tea.Model, tea.Cmd) {
	if m.Step == agentsUninstallStepDone {
		return m, tea.Quit
	}
	if m.Step == agentsUninstallStepSelect && len(m.targets) == 0 {
		return m, tea.Quit
	}
	return m.handleEnter()
}

// navigateUninstall handles up/down navigation for the
// uninstall model.
//
// Takes message (tea.KeyPressMsg) which holds the key event.
// Takes directory (int) which is -1 for up or 1 for down.
func (m *agentsUninstallModel) navigateUninstall(message tea.KeyPressMsg, directory int) {
	if m.Step == agentsUninstallStepGitignore {
		m.gitignoreCursor = max(0, min(1, m.gitignoreCursor+directory))
	} else {
		m.HandleNavigation(message, len(m.targets))
	}
}

// handleEnter processes the enter key press at each step. In the select step,
// enter toggles a checkbox when the cursor is on an item, and only confirms
// when the cursor is on the "Continue" button.
//
// Returns tea.Model which is the updated model after processing.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (m *agentsUninstallModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.Step {
	case agentsUninstallStepSelect:
		if m.HandleToggle() {
			return m, nil
		}
		return m.handleSelectConfirm()
	case agentsUninstallStepGitignore:
		return m.handleGitignoreConfirm()
	}
	return m, nil
}

// handleSelectConfirm validates the selection and starts removal.
//
// Returns tea.Model which is the updated model.
// Returns tea.Cmd which runs the removal or nil on cancellation.
//
//nolint:dupl // similar to agentsModel.handleSelectConfirm
func (m *agentsUninstallModel) handleSelectConfirm() (tea.Model, tea.Cmd) {
	if !m.AnySelected() {
		m.Aborted = true
		return m, tea.Quit
	}

	m.Step = agentsUninstallStepRemoving

	return m, func() tea.Msg {
		results, err := runSelectedTargetOperations(len(m.targets), m.Selected, func(i int) (string, error) {
			return m.targets[i].uninstall()
		})
		if err != nil {
			return agentsErrMessage{err}
		}
		return agentsUninstallDoneMessage{results: results}
	}
}

// handleGitignoreConfirm processes the yes/no choice for cleaning .gitignore.
//
// Returns tea.Model which is the updated model.
// Returns tea.Cmd which is tea.Quit.
func (m *agentsUninstallModel) handleGitignoreConfirm() (tea.Model, tea.Cmd) {
	if m.gitignoreCursor == 0 {
		builder, err := m.factory.Create("agent-gitignore-remove", ".", safedisk.ModeReadWrite)
		if err != nil {
			m.err = fmt.Errorf("failed to create sandbox: %w", err)
			return m, tea.Quit
		}
		defer builder.Close()

		if err := removeGitignoreEntries(builder); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.gitignoreUpdated = true
	}

	m.Step = agentsUninstallStepDone
	return m, tea.Quit
}

// RunAgents dispatches the agents subcommand.
//
// Takes arguments ([]string) which holds the subcommand and any flags.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunAgents(arguments []string) int {
	if len(arguments) == 0 {
		agentsUsage()
		return 1
	}

	switch arguments[0] {
	case "install", "update":
		return runAgentsInstall(arguments[1:])
	case "uninstall":
		return runAgentsUninstall(arguments[1:])
	case "-h", "--help", "help":
		agentsUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown agents subcommand: %s\n\n", arguments[0])
		agentsUsage()
		return 1
	}
}

// newAgentsModel creates a fresh agentsModel with the available targets.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
//
// Returns *agentsModel which is ready for the select step.
func newAgentsModel(factory safedisk.Factory) *agentsModel {
	home, _ := os.UserHomeDir()
	claudeDest := filepath.Join("~", agentsClaudeDir, agentsSkillsDir, agentsPikoDir)
	claudeFullDest := filepath.Join(home, agentsClaudeDir, agentsSkillsDir, agentsPikoDir)

	targets := []agentTarget{
		{
			name:     "Claude Code",
			destDesc: claudeDest,
			scope:    "global, all Piko projects",
			install: func() (string, error) {
				if err := templates.CopyClaudeCodeSkill(claudeFullDest); err != nil {
					return "", err
				}
				return fmt.Sprintf("Claude Code      %s", claudeFullDest), nil
			},
		},
		{
			name:     agentsAgentsMDFilename,
			destDesc: "./" + agentsAgentsMDFilename,
			scope:    "Codex, Cursor, Copilot, Windsurf",
			install: func() (string, error) {
				if err := templates.CopyProjectAgents("."); err != nil {
					return "", err
				}
				cwd, _ := os.Getwd()
				return fmt.Sprintf("AGENTS.md        %s", cwd), nil
			},
		},
	}

	wb := wizardbase.NewWizardBase()
	wb.Step = agentsStepSelect
	wb.Selected = make([]bool, len(targets))
	wb.Cursor = len(targets)

	return &agentsModel{
		WizardBase: wb,
		factory:    factory,
		targets:    targets,
	}
}

// newAgentsUninstallModel creates a fresh agentsUninstallModel populated with
// targets that are currently installed.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
//
// Returns *agentsUninstallModel which is ready for the select step.
func newAgentsUninstallModel(factory safedisk.Factory) *agentsUninstallModel {
	home, _ := os.UserHomeDir()
	claudeDest := filepath.Join("~", agentsClaudeDir, agentsSkillsDir, agentsPikoDir)
	claudeFullDest := filepath.Join(home, agentsClaudeDir, agentsSkillsDir, agentsPikoDir)

	var targets []agentUninstallTarget

	if _, err := os.Stat(claudeFullDest); err == nil {
		targets = append(targets, agentUninstallTarget{
			name:     "Claude Code",
			location: claudeDest,
			uninstall: func() (string, error) {
				if err := os.RemoveAll(claudeFullDest); err != nil {
					return "", fmt.Errorf("failed to remove %s: %w", claudeFullDest, err)
				}
				return fmt.Sprintf("Claude Code      %s removed", claudeFullDest), nil
			},
		})
	}

	if _, err := os.Stat(agentsAgentsMDFilename); err == nil {
		cwd, _ := os.Getwd()
		targets = append(targets, agentUninstallTarget{
			name:     agentsAgentsMDFilename,
			location: "./" + agentsAgentsMDFilename,
			uninstall: func() (string, error) {
				if err := os.Remove(agentsAgentsMDFilename); err != nil && !os.IsNotExist(err) {
					return "", fmt.Errorf("failed to remove %s: %w", agentsAgentsMDFilename, err)
				}
				if err := os.RemoveAll("references"); err != nil {
					return "", fmt.Errorf("failed to remove references/: %w", err)
				}
				return fmt.Sprintf("%s        %s removed", agentsAgentsMDFilename, cwd), nil
			},
		})
	}

	hasGitignore := false
	if data, err := os.ReadFile(".gitignore"); err == nil {
		hasGitignore = strings.Contains(string(data), agentsAgentsMDFilename)
	}

	wb := wizardbase.NewWizardBase()
	wb.Step = agentsUninstallStepSelect
	wb.Selected = make([]bool, len(targets))
	wb.Cursor = len(targets)

	return &agentsUninstallModel{
		WizardBase:          wb,
		factory:             factory,
		targets:             targets,
		hasGitignoreEntries: hasGitignore,
	}
}

// removeGitignoreEntries removes agent file entries from .gitignore using the
// provided sandbox for filesystem access. If the exact block inserted by
// install is found it is removed; otherwise individual agent lines are
// stripped.
//
// Takes builder (safedisk.Sandbox) which provides sandboxed filesystem access.
//
// Returns error when the file cannot be read or written.
func removeGitignoreEntries(builder safedisk.Sandbox) error {
	const path = ".gitignore"

	existing, err := builder.ReadFile(path)
	if err != nil {
		if safedisk.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	content := string(existing)
	if !strings.Contains(content, agentsAgentsMDFilename) {
		return nil
	}

	updated := strings.Replace(content, gitignoreEntries, "", 1)

	if updated == content {
		var lines []string
		for line := range strings.SplitSeq(content, "\n") {
			trimmed := strings.TrimSpace(line)
			switch trimmed {
			case agentsAgentsMDFilename, "references/",
				"# AI agent integration (regenerate with: piko agents install)":
				continue
			}
			lines = append(lines, line)
		}
		updated = strings.Join(lines, "\n")
	}

	if updated == content {
		return nil
	}

	return builder.WriteFile(path, []byte(updated), agentsFilePerms)
}

// appendGitignoreEntries appends agent file entries to .gitignore using the
// provided sandbox for filesystem access. Creates the file if it does not
// exist.
//
// Takes builder (safedisk.Sandbox) which provides sandboxed filesystem access.
//
// Returns error when the file cannot be read or written.
func appendGitignoreEntries(builder safedisk.Sandbox) error {
	const path = ".gitignore"

	existing, err := builder.ReadFile(path)
	if err != nil && !safedisk.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	if strings.Contains(string(existing), agentsAgentsMDFilename) {
		return nil
	}

	f, err := builder.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, agentsFilePerms)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(gitignoreEntries); err != nil {
		return fmt.Errorf("failed to write to %s: %w", path, err)
	}

	return nil
}

// agentsUsage prints usage information for the agents subcommand.
func agentsUsage() {
	_, _ = fmt.Fprint(os.Stderr, `Usage: piko agents <command>

Commands:
  install     Configure AI coding tools with Piko framework knowledge
  update      Alias for install
  uninstall   Remove AI coding tool integrations

Run 'piko agents install' to get started.
`)
}

// runAgentsInstall shows an interactive TUI for selecting which AI coding
// tools to configure with Piko framework knowledge, then copies the relevant
// files to each tool's expected location.
//
// Takes arguments ([]string) which holds any command-line flags
// (currently unused).
//
// Returns int which is the exit code: 0 on success, 1 on error or
// cancellation.
func runAgentsInstall(arguments []string) int {
	_ = arguments

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating sandbox factory: %v\n", err)
		return 1
	}

	p := tea.NewProgram(newAgentsModel(factory))

	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running agents installer: %v\n", err)
		return 1
	}

	model, ok := m.(*agentsModel)
	if !ok {
		return 1
	}

	if model.Aborted {
		fmt.Println("\nAgent installation cancelled.")
		return 1
	}
	if model.err != nil {
		fmt.Fprintf(os.Stderr, agentsErrorFmt, model.err)
		return 1
	}

	return 0
}

// runAgentsUninstall shows an interactive TUI for selecting which installed
// agent integrations to remove.
//
// Takes arguments ([]string) which holds any command-line flags
// (currently unused).
//
// Returns int which is the exit code: 0 on success, 1 on error or
// cancellation.
func runAgentsUninstall(arguments []string) int {
	_ = arguments

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating sandbox factory: %v\n", err)
		return 1
	}

	model := newAgentsUninstallModel(factory)

	if len(model.targets) == 0 {
		fmt.Println("No agent integrations are currently installed.")
		return 0
	}

	p := tea.NewProgram(model)

	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running agents uninstaller: %v\n", err)
		return 1
	}

	result, ok := m.(*agentsUninstallModel)
	if !ok {
		return 1
	}

	if result.Aborted {
		fmt.Println("\nAgent uninstallation cancelled.")
		return 1
	}
	if result.err != nil {
		fmt.Fprintf(os.Stderr, agentsErrorFmt, result.err)
		return 1
	}

	return 0
}
