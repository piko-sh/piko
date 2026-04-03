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

package wizard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"piko.sh/piko/cmd/piko/internal/wizard/templates"
	"piko.sh/piko/cmd/piko/internal/wizardbase"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	// requiredGoVersion is the minimum Go version needed by scaffolded projects.
	requiredGoVersion = "go1.26"

	// projectNameCharLimit is the maximum number of characters allowed for the
	// project name input.
	projectNameCharLimit = 50

	// projectNameInputWidth is the width of the project name input field.
	projectNameInputWidth = 30

	// modulePathCharLimit is the maximum number of characters allowed for the Go
	// module path input.
	modulePathCharLimit = 100

	// modulePathInputWidth is the display width of the Go module path input field.
	modulePathInputWidth = 50
)

// Model holds the state for the interactive CLI wizard.
type Model struct {
	// Err holds any error that occurred during the wizard workflow.
	Err error

	// TidyWarning holds a non-fatal warning from the go mod tidy step.
	TidyWarning string

	// VersionWarning holds a non-fatal warning when the latest Piko version
	// could not be resolved from GitHub.
	VersionWarning string

	// Config holds the scaffold settings gathered from the wizard inputs.
	Config templates.ScaffoldData

	// Inputs holds the text input components for collecting user data
	// during the wizard steps.
	Inputs []textinput.Model

	// Choices holds the options shown to the user for the current wizard step.
	Choices []string

	wizardbase.WizardBase

	// Done indicates whether the wizard has finished.
	Done bool
}

type (
	// errMessage wraps an error for use in the Bubble Tea message system.
	errMessage struct {
		// err is the underlying error that caused this message.
		err error
	}

	// scaffoldDoneMessage signals that the scaffold operation has completed.
	scaffoldDoneMessage struct {
		// versionWarning is non-empty when the Piko version could not be
		// resolved from GitHub.
		versionWarning string
	}

	// tidyDoneMessage signals that the go mod tidy command has completed.
	tidyDoneMessage struct {
		// warning is non-empty when the tidy step failed non-fatally.
		warning string
	}
)

const (
	// StepProjectName is the first wizard step where the user enters the project
	// name.
	StepProjectName = iota

	// StepDestination is the wizard step where the user chooses the project
	// location.
	StepDestination

	// StepModulePath is the wizard step for entering the Go module path.
	StepModulePath

	// StepFeatures is the wizard step for selecting optional features.
	StepFeatures

	// StepScaffolding is the wizard step where project files are being created.
	StepScaffolding

	// StepTidying is the wizard step where go mod tidy runs after scaffolding.
	StepTidying

	// StepFinished indicates the wizard has completed all steps.
	StepFinished
)

// Init returns the initial command to start the model.
//
// Returns tea.Cmd which batches the text input blink and spinner tick commands.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.Spinner.Tick)
}

// Update handles incoming messages and updates the model state.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns tea.Model which is the updated model after processing.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyMessage(message)
	case errMessage:
		m.Err = message.err
		return m, tea.Quit
	case scaffoldDoneMessage:
		m.VersionWarning = message.versionWarning
		m.Step = StepTidying
		command := m.runGoModTidy()
		return m, command
	case tidyDoneMessage:
		m.Done = true
		m.TidyWarning = message.warning
		m.Step = StepFinished
		return m, tea.Quit
	}

	return m.updateInputOrSpinner(message)
}

// View renders the current state of the user interface.
//
// Returns tea.View which contains the formatted terminal output.
func (m *Model) View() tea.View {
	if m.Err != nil {
		return tea.NewView(fmt.Sprintf("\nError: %v\n", m.Err))
	}

	var s strings.Builder

	switch m.Step {
	case StepProjectName:
		s.WriteString(wizardbase.TitleStyle.Render("What is the name of your new Piko project?") + "\n")
		s.WriteString(m.Inputs[0].View() + "\n")

	case StepDestination:
		s.WriteString(wizardbase.TitleStyle.Render("Where should we create your project?") + "\n")
		s.WriteString(wizardbase.RenderChoiceList(m.Choices, m.Cursor))

	case StepModulePath:
		s.WriteString(wizardbase.TitleStyle.Render("What is your Go module path?") + "\n")
		s.WriteString(m.Inputs[1].View() + "\n")

	case StepFeatures:
		s.WriteString(wizardbase.TitleStyle.Render("Which optional features would you like to enable?") + "\n")
		items := make([]wizardbase.CheckboxItem, len(m.Choices))
		for i, choice := range m.Choices {
			items[i] = wizardbase.CheckboxItem{Label: choice, Selected: m.Selected[i]}
		}
		s.WriteString(wizardbase.RenderCheckboxList(items, m.Cursor))

	case StepScaffolding:
		s.WriteString(wizardbase.RenderSpinnerLine(m.Spinner.View(), "Scaffolding your project, please wait..."))

	case StepTidying:
		s.WriteString(wizardbase.RenderSpinnerLine(m.Spinner.View(), "Running go mod tidy..."))

	default:
	}

	help := "Use arrow keys to navigate, enter to confirm, ctrl+c to quit."
	if m.Step == StepFeatures {
		help = "Use arrow keys to navigate, space to toggle, enter to confirm, ctrl+c to quit."
	}
	s.WriteString("\n" + wizardbase.HelpStyle.Render(help))
	return tea.NewView(s.String())
}

// handleKeyMessage processes keyboard input messages.
//
// Takes message (tea.KeyPressMsg) which holds the key event to process.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is the command to run, or nil if none.
func (m *Model) handleKeyMessage(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c", "esc":
		command := m.HandleAbort()
		return m, command
	case "enter":
		if m.Step == StepFinished {
			return m, tea.Quit
		}
		return m.handleEnter()
	}
	if m.Step == StepDestination || m.Step == StepFeatures {
		m.handleChoiceNavigation(message)
	}
	return m.updateInputOrSpinner(message)
}

// handleChoiceNavigation moves the cursor up or down in choice and feature
// steps, and toggles selection with the space key in the features step.
//
// Takes message (tea.KeyPressMsg) which contains the key press to process.
func (m *Model) handleChoiceNavigation(message tea.KeyPressMsg) {
	maxCursor := len(m.Choices) - 1
	if m.Step == StepFeatures {
		maxCursor = len(m.Choices)
	}
	if m.HandleNavigation(message, maxCursor) {
		return
	}
	if message.String() == "space" && m.Step == StepFeatures {
		m.HandleToggle()
	}
}

// updateInputOrSpinner updates either the text input or spinner based on the
// current step.
//
// Takes message (tea.Msg) which contains the message to process.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is the command to run, if any.
func (m *Model) updateInputOrSpinner(message tea.Msg) (tea.Model, tea.Cmd) {
	var command tea.Cmd
	if m.Step < StepScaffolding && (m.Step == StepProjectName || m.Step == StepModulePath) {
		currentInputIndex := len(m.Inputs) - 1
		m.Inputs[currentInputIndex], command = m.Inputs[currentInputIndex].Update(message)
	} else if m.Step >= StepScaffolding {
		command = m.UpdateSpinner(message)
	}
	return m, command
}

// runGoModTidy returns a command that runs go mod tidy in the destination
// folder. Failures are non-blocking - the project is still created, and
// the user is advised to run the command manually.
//
// Returns tea.Cmd which runs the tidy operation and sends a completion message.
func (m *Model) runGoModTidy() tea.Cmd {
	return func() tea.Msg {
		command := exec.Command("go", "mod", "tidy")
		command.Dir = m.Config.DestinationPath
		if err := command.Run(); err != nil {
			return tidyDoneMessage{
				warning: fmt.Sprintf("'go mod tidy' failed: %v - you can run it manually later", err),
			}
		}
		return tidyDoneMessage{}
	}
}

// handleEnter processes the enter key press to move through wizard steps.
//
// Returns tea.Model which is the updated model after processing.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.Step {
	case StepProjectName:
		return m.enterProjectName()
	case StepDestination:
		return m.enterDestination()
	case StepModulePath:
		return m.enterModulePath()
	case StepFeatures:
		return m.enterFeatures()
	default:
		return m, nil
	}
}

// enterProjectName captures the project name and advances to the destination
// step.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is nil since no async work is needed.
func (m *Model) enterProjectName() (tea.Model, tea.Cmd) {
	m.Config.ProjectName = m.Inputs[0].Value()
	if m.Config.ProjectName == "" {
		m.Config.ProjectName = m.Inputs[0].Placeholder
	}
	m.Step = StepDestination
	m.Choices = []string{
		fmt.Sprintf("Create in a new folder: ./%s/", m.Config.ProjectName),
		"Create in the current folder: ./",
	}
	return m, nil
}

// enterDestination captures the destination path choice and advances to the
// module path step.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which starts the text input blink cursor.
func (m *Model) enterDestination() (tea.Model, tea.Cmd) {
	if m.Cursor == 0 {
		m.Config.DestinationPath = m.Config.ProjectName
	} else {
		m.Config.DestinationPath = "."
	}
	m.Step = StepModulePath

	ti := textinput.New()
	ti.Placeholder = m.Config.ProjectName
	ti.Focus()
	ti.CharLimit = modulePathCharLimit
	ti.SetWidth(modulePathInputWidth)
	m.Inputs = append(m.Inputs, ti)
	return m, textinput.Blink
}

// enterModulePath captures the Go module path and advances to the features
// step.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which is nil since no async work is needed.
func (m *Model) enterModulePath() (tea.Model, tea.Cmd) {
	m.Config.ModuleName = m.Inputs[1].Value()
	if m.Config.ModuleName == "" {
		m.Config.ModuleName = m.Inputs[1].Placeholder
	}

	m.Step = StepFeatures
	m.Choices = []string{
		"Struct validation (go-playground/validator)",
		"AI agent integration (AGENTS.md, Claude Code, Codex, Cursor, etc.)",
		"Sonic JSON provider (faster JSON encoding via bytedance/sonic)",
		"Experimental interpreted mode",
	}
	m.Selected = make([]bool, len(m.Choices))
	m.Selected[0] = true
	m.Selected[2] = true
	m.Cursor = len(m.Choices)
	return m, nil
}

// enterFeatures captures feature selections and starts the scaffolding step.
//
// Returns tea.Model which is the updated model state.
// Returns tea.Cmd which triggers the scaffold operation, or nil if the
// confirm button was toggled.
func (m *Model) enterFeatures() (tea.Model, tea.Cmd) {
	if m.HandleToggle() {
		return m, nil
	}

	const featureValidator = 0
	const featureAgents = 1
	const featureSonicJSON = 2
	const featureInterpreted = 3
	m.Config.EnableValidator = m.Selected[featureValidator]
	m.Config.EnableAgents = m.Selected[featureAgents]
	m.Config.EnableSonicJSON = m.Selected[featureSonicJSON]
	m.Config.EnableInterpreted = m.Selected[featureInterpreted]

	m.Step = StepScaffolding

	return m, func() tea.Msg {
		var versionWarning string

		version, err := resolveLatestVersion()
		if err != nil {
			m.Config.PikoVersion = fallbackVersion
			versionWarning = fmt.Sprintf(
				"could not resolve latest Piko version (%v) - using %s, update go.mod manually",
				err, fallbackVersion,
			)
		} else {
			m.Config.PikoVersion = version
		}

		if err := templates.CreateProject(m.Config); err != nil {
			return errMessage{err}
		}
		return scaffoldDoneMessage{versionWarning: versionWarning}
	}
}

// Run starts the interactive project creation wizard and handles the result.
//
// Returns int which is the exit code: 0 on success, 1 on error or cancellation.
func Run() int {
	p := tea.NewProgram(newInitialModel())

	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error running wizard: %v\n", err)
		return 1
	}

	model, ok := m.(*Model)
	if !ok {
		return 1
	}

	if model.Aborted {
		fmt.Println("\nProject creation cancelled.")
		return 1
	}
	if model.Err != nil {
		fmt.Printf("\nAn error occurred during project creation: %v\n", model.Err)
		return 1
	}

	if w := goVersionWarning(); w != "" {
		fmt.Printf("\nWarning: %s\n", w)
	}
	if model.VersionWarning != "" {
		fmt.Printf("\nWarning: %s\n", model.VersionWarning)
	}
	if model.TidyWarning != "" {
		fmt.Printf("\nWarning: %s\n", model.TidyWarning)
	}

	fmt.Printf("\nPiko project '%s' created successfully!\n", model.Config.ProjectName)

	if model.Config.EnableAgents {
		fmt.Println("\nProject-level AI agent integration added:")
		fmt.Println("  AGENTS.md + references/   (Codex, Cursor, Copilot, Windsurf)")
		fmt.Println("\n  For Claude Code global setup, run: piko agents install")
		fmt.Println("  Run 'piko agents install' after upgrading Piko to refresh these files.")
	}

	fmt.Println("\nNext Steps:")
	step := 1
	fmt.Printf("%d. cd %s\n", step, model.Config.ProjectName)
	step++
	if model.TidyWarning != "" {
		fmt.Printf("%d. Run 'go mod tidy' to fetch dependencies.\n", step)
		step++
	}
	fmt.Printf("%d. Run 'go run ./cmd/generator/main.go all' to build your assets for the first time.\n", step)
	step++
	fmt.Printf("%d. Run 'air' to start the development server with live reloading.\n", step)
	fmt.Println("\nHappy coding!")

	return 0
}

// InitialModel creates a new Model with default settings for the wizard.
//
// Returns Model which is ready to begin at the project name step.
func InitialModel() Model {
	return *newInitialModel()
}

// newInitialModel creates a new Model pointer with default settings for the
// wizard.
//
// Returns *Model which is ready to begin at the project name step.
func newInitialModel() *Model {
	ti := textinput.New()
	ti.Placeholder = "my-piko-app"
	ti.Focus()
	ti.CharLimit = projectNameCharLimit
	ti.SetWidth(projectNameInputWidth)

	wb := wizardbase.NewWizardBase()
	wb.Step = StepProjectName

	return &Model{
		WizardBase: wb,
		Inputs:     []textinput.Model{ti},
	}
}

// goVersionWarning returns a warning message if the current Go runtime is
// older than the version required by scaffolded projects. Returns an empty
// string when the version is sufficient.
//
// Returns string which is the warning message, or empty if the Go
// version is sufficient.
func goVersionWarning() string {
	version := runtime.Version()
	if version >= requiredGoVersion {
		return ""
	}
	return fmt.Sprintf(
		"your Go version (%s) is older than the required %s - you may need to upgrade before building",
		version, requiredGoVersion,
	)
}
