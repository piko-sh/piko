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
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// tuiHistorySize is the number of samples kept in sparkline history rings.
	tuiHistorySize = 300

	// tuiProgressBarWidth is the width of phase progress bars in characters.
	tuiProgressBarWidth = 20

	// tuiSparklineWidth is the width of sparkline charts in characters.
	tuiSparklineWidth = 30

	// tuiChannelBuffer is the buffer size for TUI message channels.
	tuiChannelBuffer = 16

	// tuiMetricsFmt is the format string for labelled TUI metric lines.
	tuiMetricsFmt = "  %s  %s\n"
)

// phaseStatus represents the state of a profiling phase.
type phaseStatus int

const (
	// phasePending indicates the phase has not yet started.
	phasePending phaseStatus = iota

	// phaseActive indicates the phase is currently running.
	phaseActive

	// phaseDone indicates the phase has completed.
	phaseDone
)

// phaseMessage notifies the TUI of a phase transition.
type phaseMessage struct {
	// name identifies the profiling phase.
	name string

	// status is the new state of the phase.
	status phaseStatus
}

// profileDoneMessage signals that the entire profiling pipeline has completed.
type profileDoneMessage struct {
	// err holds any error from the pipeline, or nil on success.
	err error
}

// profileTickMessage drives periodic TUI refresh.
type profileTickMessage time.Time

// goroutineMessage delivers a goroutine count sample to the TUI.
type goroutineMessage struct {
	// count is the number of goroutines at the time of sampling.
	count int
}

// profileTUIModel is the BubbleTea model for the live profiling dashboard.
type profileTUIModel struct {
	// Pointer-containing fields grouped first to minimise GC scan area.
	resultErr error

	// phaseStatus maps phase names to their current state.
	phaseStatus map[string]phaseStatus

	// rpsHistory stores recent requests-per-second samples.
	rpsHistory *tui_domain.HistoryRing

	// latencyHistory stores recent latency samples.
	latencyHistory *tui_domain.HistoryRing

	// goroutineHistory stores recent goroutine count samples.
	goroutineHistory *tui_domain.HistoryRing

	// metricsCh receives live metrics from the load generator.
	metricsCh <-chan metricsMessage

	// goroutineCh receives goroutine count samples.
	goroutineCh <-chan goroutineMessage

	// phaseCh receives phase transition notifications.
	phaseCh <-chan phaseMessage

	// doneCh receives the pipeline completion signal.
	doneCh <-chan profileDoneMessage

	// phaseStart records when the current phase began.
	phaseStart time.Time

	// targetURL is the URL being profiled.
	targetURL string

	// activePhase is the name of the currently running phase.
	activePhase string

	// phases lists all phase names in execution order.
	phases []string

	// currentRPS is the latest requests-per-second value.
	currentRPS float64

	// currentLatency is the latest mean latency in milliseconds.
	currentLatency float64

	// p50Ms is the 50th percentile latency in milliseconds.
	p50Ms float64

	// p80Ms is the 80th percentile latency in milliseconds.
	p80Ms float64

	// p99Ms is the 99th percentile latency in milliseconds.
	p99Ms float64

	// p100Ms is the 100th percentile latency in milliseconds.
	p100Ms float64

	// totalRequests is the cumulative completed request count.
	totalRequests int64

	// failedRequests is the cumulative failed request count.
	failedRequests int64

	// bytesReceived is the cumulative response bytes read.
	bytesReceived int64

	// goroutineCount is the latest goroutine count.
	goroutineCount int

	// phaseDurSecs is the duration of each phase in seconds.
	phaseDurSecs int

	// width is the terminal width in columns.
	width int

	// height is the terminal height in rows.
	height int

	// done is true when the pipeline has finished.
	done bool
}

// profileTUIStyles holds lipgloss styles for the profile TUI.
var profileTUIStyles = struct {
	title     lipgloss.Style
	phase     lipgloss.Style
	active    lipgloss.Style
	done      lipgloss.Style
	pending   lipgloss.Style
	label     lipgloss.Style
	value     lipgloss.Style
	border    lipgloss.Style
	footer    lipgloss.Style
	sparkline lipgloss.Style
}{
	title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
	phase:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")),
	active:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
	done:      lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	pending:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	label:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	value:     lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
	border:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	footer:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true),
	sparkline: lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
}

// Init starts the periodic tick command.
//
// Returns tea.Cmd which batches the tick, phase, and done listeners.
func (m *profileTUIModel) Init() tea.Cmd {
	return tea.Batch(
		tickProfileTUI(),
		listenPhase(m.phaseCh),
		listenDone(m.doneCh),
	)
}

// Update handles incoming messages.
//
// Takes incomingMessage (tea.Msg) which is the message to
// process.
//
// Returns tea.Model which is the updated model.
// Returns tea.Cmd which is the next command to run.
func (m *profileTUIModel) Update(incomingMessage tea.Msg) (tea.Model, tea.Cmd) {
	switch message := incomingMessage.(type) {
	case tea.KeyPressMsg:
		if message.String() == "q" || message.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height

	case profileTickMessage:

		for {
			select {
			case met := <-m.metricsCh:
				m.currentRPS = met.rps
				m.currentLatency = met.meanLatencyMs
				m.totalRequests = met.total
				m.failedRequests = met.failed
				m.bytesReceived = met.bytesReceived
				m.p50Ms = met.p50Ms
				m.p80Ms = met.p80Ms
				m.p99Ms = met.p99Ms
				m.p100Ms = met.p100Ms
				m.rpsHistory.Append(met.rps)
				m.latencyHistory.Append(met.meanLatencyMs)
			case g := <-m.goroutineCh:
				m.goroutineCount = g.count
				m.goroutineHistory.Append(float64(g.count))
			default:
				goto drained
			}
		}
	drained:
		return m, tickProfileTUI()

	case phaseMessage:
		m.phaseStatus[message.name] = message.status
		if message.status == phaseActive {
			m.activePhase = message.name
			m.phaseStart = time.Now()
		}
		return m, listenPhase(m.phaseCh)

	case profileDoneMessage:
		m.done = true
		m.resultErr = message.err
		return m, tea.Quit
	}

	return m, nil
}

// View renders the TUI dashboard.
//
// Returns tea.View which contains the formatted terminal output.
func (m *profileTUIModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Initialising...")
	}

	var b strings.Builder
	m.renderHeader(&b)
	b.WriteByte('\n')
	m.renderPhaseProgress(&b)
	b.WriteByte('\n')
	m.renderSparklines(&b)
	b.WriteByte('\n')
	m.renderCounters(&b)
	b.WriteByte('\n')
	m.renderLatencyPercentiles(&b)
	b.WriteByte('\n')
	m.renderPhases(&b)
	m.renderFooter(&b)

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// renderHeader writes the title bar with border lines and
// target URL.
//
// Takes b (*strings.Builder) which accumulates the output.
func (m *profileTUIModel) renderHeader(b *strings.Builder) {
	b.WriteString(profileTUIStyles.border.Render(strings.Repeat("─", m.width)))
	b.WriteByte('\n')
	b.WriteString(profileTUIStyles.title.Render("  piko profile"))
	b.WriteString(profileTUIStyles.label.Render(" - "))
	b.WriteString(profileTUIStyles.value.Render(m.targetURL))
	b.WriteByte('\n')
	b.WriteString(profileTUIStyles.border.Render(strings.Repeat("─", m.width)))
	b.WriteByte('\n')
}

// renderSparklines writes the RPS, latency, and goroutine
// sparkline charts.
//
// Takes b (*strings.Builder) which accumulates the output.
func (m *profileTUIModel) renderSparklines(b *strings.Builder) {
	sparkConfig := tui_domain.DefaultSparklineConfig()
	sparkConfig.Width = tuiSparklineWidth
	sparkConfig.ShowCurrent = true
	sparkConfig.Style = profileTUIStyles.sparkline

	_, _ = fmt.Fprintf(b, tuiMetricsFmt,
		profileTUIStyles.label.Render("Requests/sec"),
		tui_domain.Sparkline(m.rpsHistory.Values(), &sparkConfig),
	)

	latencyConfig := sparkConfig
	latencyConfig.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	_, _ = fmt.Fprintf(b, tuiMetricsFmt,
		profileTUIStyles.label.Render("Latency (ms) "),
		tui_domain.Sparkline(m.latencyHistory.Values(), &latencyConfig),
	)

	goroutineConfig := sparkConfig
	goroutineConfig.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	_, _ = fmt.Fprintf(b, tuiMetricsFmt,
		profileTUIStyles.label.Render("Goroutines   "),
		tui_domain.Sparkline(m.goroutineHistory.Values(), &goroutineConfig),
	)
}

// renderCounters writes the total requests, failed, and bytes
// received lines.
//
// Takes b (*strings.Builder) which accumulates the output.
func (m *profileTUIModel) renderCounters(b *strings.Builder) {
	_, _ = fmt.Fprintf(b, tuiMetricsFmt,
		profileTUIStyles.label.Render("Total Requests:"),
		profileTUIStyles.value.Render(profileFormatCount(m.totalRequests)),
	)
	_, _ = fmt.Fprintf(b, tuiMetricsFmt,
		profileTUIStyles.label.Render("Failed:        "),
		profileTUIStyles.value.Render(profileFormatCount(m.failedRequests)),
	)
	_, _ = fmt.Fprintf(b, tuiMetricsFmt,
		profileTUIStyles.label.Render("Bytes Received:"),
		profileTUIStyles.value.Render(profileFormatBytes(m.bytesReceived)),
	)
}

// renderLatencyPercentiles writes the p50/p80/p99/p100
// latency line.
//
// Takes b (*strings.Builder) which accumulates the output.
func (m *profileTUIModel) renderLatencyPercentiles(b *strings.Builder) {
	_, _ = fmt.Fprintf(b, "  %s  %s  %s  %s  %s\n",
		profileTUIStyles.label.Render("Latency:"),
		profileTUIStyles.value.Render(fmt.Sprintf("p50 %.2fms", m.p50Ms)),
		profileTUIStyles.value.Render(fmt.Sprintf("p80 %.2fms", m.p80Ms)),
		profileTUIStyles.value.Render(fmt.Sprintf("p99 %.2fms", m.p99Ms)),
		profileTUIStyles.value.Render(fmt.Sprintf("p100 %.2fms", m.p100Ms)),
	)
}

// renderFooter writes the bottom border and quit hint.
//
// Takes b (*strings.Builder) which accumulates the output.
func (m *profileTUIModel) renderFooter(b *strings.Builder) {
	b.WriteByte('\n')
	b.WriteString(profileTUIStyles.border.Render(strings.Repeat("─", m.width)))
	b.WriteByte('\n')
	b.WriteString(profileTUIStyles.footer.Render("  q to quit early"))
	b.WriteByte('\n')
}

// renderPhaseProgress writes the current phase progress bar
// to b.
//
// Takes b (*strings.Builder) which accumulates the output.
func (m *profileTUIModel) renderPhaseProgress(b *strings.Builder) {
	if m.activePhase != "" {
		elapsed := time.Since(m.phaseStart)
		total := time.Duration(m.phaseDurSecs) * time.Second
		pct := elapsed.Seconds() / total.Seconds()
		if pct > 1 {
			pct = 1
		}

		filled := min(int(pct*float64(tuiProgressBarWidth)), tuiProgressBarWidth)
		bar := strings.Repeat("█", filled) + strings.Repeat("░", tuiProgressBarWidth-filled)

		_, _ = fmt.Fprintf(b, "  Phase: %s [%s] %ds/%ds\n",
			profileTUIStyles.phase.Render(m.activePhase),
			profileTUIStyles.active.Render(bar),
			int(elapsed.Seconds()),
			m.phaseDurSecs,
		)
	} else if m.done {
		b.WriteString("  " + profileTUIStyles.done.Render("Profiling complete!") + "\n")
	} else {
		b.WriteString("  " + profileTUIStyles.label.Render("Waiting for pipeline to start...") + "\n")
	}
}

// renderPhases writes the phase status indicators to b.
//
// Takes b (*strings.Builder) which accumulates the output.
func (m *profileTUIModel) renderPhases(b *strings.Builder) {
	b.WriteString("  Phases: ")
	for i, p := range m.phases {
		if i > 0 {
			b.WriteString("  ")
		}
		switch m.phaseStatus[p] {
		case phaseDone:
			b.WriteString(profileTUIStyles.done.Render("✓ " + p))
		case phaseActive:
			b.WriteString(profileTUIStyles.active.Render("● " + p))
		default:
			b.WriteString(profileTUIStyles.pending.Render("○ " + p))
		}
	}
	b.WriteByte('\n')
}

// profileTUIParams groups the parameters for the TUI entry point.
type profileTUIParams struct {
	// factory creates sandboxes for filesystem access.
	factory safedisk.Factory

	// flags holds the parsed CLI flags.
	flags *profileFlags

	// focusRegex optionally filters function names.
	focusRegex *regexp.Regexp

	// stdout receives normal output.
	stdout io.Writer

	// stderr receives error output.
	stderr io.Writer

	// headers holds HTTP request headers.
	headers map[string]string

	// url is the target URL to profile.
	url string

	// pprofBase is the pprof endpoint base URL.
	pprofBase string

	// profilerRoot is the server root URL used for profiler capability requests.
	profilerRoot string
}

// profilePipelineParams groups the parameters for runProfilePipeline.
type profilePipelineParams struct {
	// factory creates sandboxes for filesystem access.
	factory safedisk.Factory

	// flags holds the parsed CLI flags.
	flags *profileFlags

	// focusRegex optionally filters function names.
	focusRegex *regexp.Regexp

	// headers holds HTTP request headers.
	headers map[string]string

	// metricsCh receives live metrics.
	metricsCh chan<- metricsMessage

	// phaseCh receives phase transitions.
	phaseCh chan<- phaseMessage

	// doneCh receives completion.
	doneCh chan<- profileDoneMessage

	// url is the target URL to profile.
	url string

	// pprofBase is the pprof endpoint base URL.
	pprofBase string

	// profilerRoot is the server root URL used for profiler capability requests.
	profilerRoot string
}

// newProfileTUIModel creates a fresh profile TUI model wired to the
// given channels.
//
// Takes targetURL (string) which is the URL being profiled.
// Takes phaseDurSecs (int) which is the duration of each phase.
// Takes metricsCh (<-chan metricsMessage) which delivers live metrics.
// Takes goroutineCh (<-chan goroutineMessage) which delivers goroutine counts.
// Takes phaseCh (<-chan phaseMessage) which delivers phase transitions.
// Takes doneCh (<-chan profileDoneMessage) which signals completion.
//
// Returns profileTUIModel which is ready for BubbleTea.
func newProfileTUIModel(
	targetURL string,
	phaseDurSecs int,
	metricsCh <-chan metricsMessage,
	goroutineCh <-chan goroutineMessage,
	phaseCh <-chan phaseMessage,
	doneCh <-chan profileDoneMessage,
) profileTUIModel {
	phases := []string{"baseline", "cpu", "allocs", "heap", "mutex", "block"}
	phaseMap := make(map[string]phaseStatus, len(phases))
	for _, p := range phases {
		phaseMap[p] = phasePending
	}

	return profileTUIModel{
		targetURL:        targetURL,
		phases:           phases,
		phaseStatus:      phaseMap,
		phaseDurSecs:     phaseDurSecs,
		rpsHistory:       tui_domain.NewHistoryRing(tuiHistorySize),
		latencyHistory:   tui_domain.NewHistoryRing(tuiHistorySize),
		goroutineHistory: tui_domain.NewHistoryRing(tuiHistorySize),
		metricsCh:        metricsCh,
		goroutineCh:      goroutineCh,
		phaseCh:          phaseCh,
		doneCh:           doneCh,
	}
}

// tickProfileTUI returns a command that fires after 200ms.
//
// Returns tea.Cmd which sends a profileTickMessage after the delay.
func tickProfileTUI() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return profileTickMessage(t)
	})
}

// listenPhase returns a command that waits for the next phase message.
//
// Takes phaseChannel (<-chan phaseMessage) which delivers phase transitions.
//
// Returns tea.Cmd which blocks until a phase message arrives.
func listenPhase(phaseChannel <-chan phaseMessage) tea.Cmd {
	return func() tea.Msg {
		message, ok := <-phaseChannel
		if !ok {
			return nil
		}
		return message
	}
}

// listenDone returns a command that waits for the pipeline to finish.
//
// Takes doneChannel (<-chan profileDoneMessage) which signals pipeline completion.
//
// Returns tea.Cmd which blocks until the done message arrives.
func listenDone(doneChannel <-chan profileDoneMessage) tea.Cmd {
	return func() tea.Msg {
		message, ok := <-doneChannel
		if !ok {
			return profileDoneMessage{}
		}
		return message
	}
}

// runProfileTUI runs the profiling pipeline with a live BubbleTea
// dashboard.
//
// Takes params (profileTUIParams) which groups all TUI settings.
//
// Returns int which is the exit code: 0 on success, 1 on error.
//
// Spawns a goroutine that runs the profiling pipeline and
// another that polls goroutine counts from the pprof endpoint. Both
// goroutines are cancelled when the TUI exits.
func runProfileTUI(ctx context.Context, params profileTUIParams) int {
	metricsCh := make(chan metricsMessage, tuiChannelBuffer)
	goroutineCh := make(chan goroutineMessage, tuiChannelBuffer)
	phaseCh := make(chan phaseMessage, tuiChannelBuffer)
	doneCh := make(chan profileDoneMessage, 1)

	pipelineCtx, pipelineCancel := context.WithCancel(ctx)
	go runProfilePipeline(ctx, profilePipelineParams{
		factory:      params.factory,
		flags:        params.flags,
		url:          params.url,
		focusRegex:   params.focusRegex,
		pprofBase:    params.pprofBase,
		profilerRoot: params.profilerRoot,
		headers:      params.headers,
		metricsCh:    metricsCh,
		phaseCh:      phaseCh,
		doneCh:       doneCh,
	})
	go pollGoroutineCount(pipelineCtx, params.pprofBase, goroutineCh)

	model := newProfileTUIModel(params.url, params.flags.duration, metricsCh, goroutineCh, phaseCh, doneCh)
	p := tea.NewProgram(new(model))

	if _, err := p.Run(); err != nil {
		pipelineCancel()
		_, _ = fmt.Fprintf(params.stderr, "Error: TUI failed: %v\n", err)
		return 1
	}
	pipelineCancel()

	reportPath := filepath.Join(params.flags.output, "live_performance_report.txt")
	_, _ = fmt.Fprintf(params.stdout, "\nProfiling complete! Report: %s\n", reportPath)
	_, _ = fmt.Fprintf(params.stdout, "Raw profiles saved to %s/\n", params.flags.output)

	return 0
}

// runProfilePipeline executes the profiling flow via the shared
// pipeline, sending phase and metrics events to the TUI.
//
// Takes params (profilePipelineParams) which groups all pipeline
// settings.
func runProfilePipeline(ctx context.Context, params profilePipelineParams) {
	defer func() {
		close(params.phaseCh)
		close(params.doneCh)
	}()

	err := runPipeline(ctx, pipelineConfig{
		factory:      params.factory,
		flags:        params.flags,
		url:          params.url,
		pprofBase:    params.pprofBase,
		profilerRoot: params.profilerRoot,
		headers:      params.headers,
		specs:        buildProfileSpecs(params.flags, params.focusRegex),
		stdout:       io.Discard,
		stderr:       io.Discard,
		metricsCh:    params.metricsCh,
		phaseCh:      params.phaseCh,
	})

	params.doneCh <- profileDoneMessage{err: err}
}

// pollGoroutineCount periodically fetches the goroutine count from the pprof
// endpoint and sends it to the channel until the context is cancelled.
//
// Takes ctx (context.Context) which controls the polling lifetime.
// Takes pprofBase (string) which is the pprof endpoint base URL.
// Takes goroutineChannel (chan<- goroutineMessage) which receives
// goroutine count samples.
func pollGoroutineCount(ctx context.Context, pprofBase string, goroutineChannel chan<- goroutineMessage) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count := fetchGoroutineCount(ctx, pprofBase)
			if count > 0 {
				select {
				case goroutineChannel <- goroutineMessage{count: count}:
				default:
				}
			}
		}
	}
}
