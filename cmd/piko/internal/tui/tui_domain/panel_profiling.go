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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// profilingRefreshTimeout caps the status RPC fetch.
	profilingRefreshTimeout = 5 * time.Second

	// profilingDefaultCaptureSeconds is the default sampling window
	// presented to the user for one-shot captures.
	profilingDefaultCaptureSeconds = 10

	// profileFilePerm is the POSIX permission applied to captured
	// profile files written under the OS temp dir. Read/write owner
	// only, captures may contain sensitive runtime state.
	profileFilePerm = 0o600

	// percentScale converts a [0,1] ratio to a percentage in [0,100].
	percentScale = 100

	// captureSectionHeading is the common heading for the
	// capture-status section in the detail body.
	captureSectionHeading = "Capture"
)

// profilingRefreshMessage carries a status fetch result.
type profilingRefreshMessage struct {
	// status is the freshly fetched profiling status, or nil on error.
	status *ProfilingStatus

	// err is the fetch error, or nil on success.
	err error
}

// profilingActionMessage carries the result of an enable/disable
// action.
type profilingActionMessage struct {
	// err is the action error, or nil on success.
	err error

	// action names the action ("enable" or "disable").
	action string
}

// profilingCaptureMessage carries the result of a one-shot capture.
type profilingCaptureMessage struct {
	// summary is the parsed top-N for the captured profile; nil when
	// the capture or parse failed.
	summary *inspector.ProfileSummary

	// err holds any capture or parse error.
	err error

	// profile names the profile kind that was captured (cpu, heap, ...).
	profile string

	// path is the on-disk path the bytes were written to.
	path string

	// bytes is the captured byte count.
	bytes int
}

// ProfilingPanel surfaces the on-demand profiling controls. It is the
// TUI counterpart of `piko profiling enable|disable|status|capture`.
type ProfilingPanel struct {
	// lastRefresh records when the panel last received a status payload.
	lastRefresh time.Time

	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies the profiling inspector port.
	provider ProfilingInspector

	// err holds the last status refresh error, or nil after success.
	err error

	// captureErr holds the most recent capture / enable / disable
	// failure. Cleared on the next successful action.
	captureErr error

	// status holds the most recent status snapshot.
	status *ProfilingStatus

	// captureMessage is the last user-facing capture status (saved
	// path, "in flight...", failure label).
	captureMessage string

	// lastSummary is the parsed top-N for the most recent successful
	// capture; nil before any capture finishes.
	lastSummary *inspector.ProfileSummary

	BasePanel

	// stateMutex guards status / err / capture state for safe
	// concurrent reads.
	stateMutex sync.RWMutex

	// capturing reports whether a capture RPC is currently in flight.
	capturing bool
}

var _ Panel = (*ProfilingPanel)(nil)

// NewProfilingPanel constructs a ProfilingPanel.
//
// Takes provider (ProfilingInspector) which supplies the profiling RPC port.
// Takes c (clock.Clock) which yields the current time; nil falls back
// to the real system clock.
//
// Returns *ProfilingPanel ready to register with the model.
func NewProfilingPanel(provider ProfilingInspector, c clock.Clock) *ProfilingPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &ProfilingPanel{
		BasePanel:  NewBasePanel("profiling", titleProfiling),
		clock:      c,
		provider:   provider,
		stateMutex: sync.RWMutex{},
	}
	p.SetKeyMap([]KeyBinding{
		{Key: "e", Description: "Enable profiling"},
		{Key: "d", Description: "Disable profiling"},
		{Key: "c", Description: "Capture CPU profile"},
		{Key: "h", Description: "Capture heap profile"},
		{Key: "r", Description: "Refresh"},
	})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which is the initial refresh command.
func (p *ProfilingPanel) Init() tea.Cmd { return p.refresh() }

// Update handles messages.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel (always the receiver).
// Returns tea.Cmd which is the resulting command.
func (p *ProfilingPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(msg)
	case profilingRefreshMessage:
		p.handleStatus(msg)
		return p, nil
	case profilingActionMessage:
		p.handleAction(msg)
		cmd := p.refresh()
		return p, cmd
	case profilingCaptureMessage:
		p.handleCapture(msg)
		return p, func() tea.Msg {
			return popOverlayMessage{ID: profilingCaptureOverlayID}
		}
	case DataUpdatedMessage, TickMessage:
		cmd := p.refresh()
		return p, cmd
	}
	return p, nil
}

// View renders the panel.
//
// Takes width (int) which is the allocated panel width.
// Takes height (int) which is the allocated panel height.
//
// Returns string which is the framed panel body.
func (p *ProfilingPanel) View(width, height int) string {
	p.SetSize(width, height)
	status, err := p.snapshot()
	body := p.renderBody(status, err)
	return p.RenderFrame(body)
}

// DetailView renders the right-pane detail.
//
// Takes width (int) which is the detail pane width.
// Takes height (int) which is the detail pane height.
//
// Returns string which is the rendered detail body.
func (p *ProfilingPanel) DetailView(width, height int) string {
	status, err := p.snapshot()
	return RenderDetailBody(nil, p.detailBody(status, err), width, height)
}

// snapshot returns the latest cached status and error under a read lock.
//
// Returns *ProfilingStatus which is the cached status snapshot or nil.
// Returns error which is the cached refresh error or nil.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProfilingPanel) snapshot() (*ProfilingStatus, error) {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return p.status, p.err
}

// handleKey routes a key press to the matching action command.
//
// Takes msg (tea.KeyPressMsg) which is the key event.
//
// Returns Panel which is the receiver.
// Returns tea.Cmd which is the action command, or nil.
func (p *ProfilingPanel) handleKey(msg tea.KeyPressMsg) (Panel, tea.Cmd) {
	switch msg.String() {
	case "r":
		cmd := p.refresh()
		return p, cmd
	case "e":
		cmd := p.enableCmd()
		return p, cmd
	case "d":
		cmd := p.disableCmd()
		return p, cmd
	case "c":
		cmd := p.captureCmd("cpu", profilingDefaultCaptureSeconds*time.Second)
		return p, cmd
	case "h":
		cmd := p.captureCmd("heap", time.Second)
		return p, cmd
	}
	return p, nil
}

// handleStatus stores a fresh status payload under a write lock.
//
// Takes msg (profilingRefreshMessage) which carries status or error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProfilingPanel) handleStatus(msg profilingRefreshMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if msg.err != nil {
		p.err = msg.err
		return
	}
	p.err = nil
	p.status = msg.status
	p.lastRefresh = p.clock.Now()
}

// handleAction stores the result of an enable/disable action.
//
// Takes msg (profilingActionMessage) which carries action and error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProfilingPanel) handleAction(msg profilingActionMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if msg.err != nil {
		p.captureMessage = msg.action + " failed"
		p.captureErr = msg.err
	} else {
		p.captureMessage = msg.action + " ok"
		p.captureErr = nil
	}
}

// handleCapture stores the result of a one-shot capture under a write lock.
//
// Takes msg (profilingCaptureMessage) which carries the capture result.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProfilingPanel) handleCapture(msg profilingCaptureMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	p.capturing = false
	if msg.err != nil && msg.summary == nil {
		p.captureMessage = msg.profile + " capture failed"
		p.captureErr = msg.err
		return
	}
	p.captureErr = nil
	if msg.path != "" {
		p.captureMessage = fmt.Sprintf("%s saved to %s (%d bytes)", msg.profile, msg.path, msg.bytes)
	}
	if msg.summary != nil {
		p.lastSummary = msg.summary
	}
}

// renderBody renders the centre-pane body for the profiling panel.
//
// Takes status (*ProfilingStatus) which is the latest status snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns string which is the rendered body.
func (p *ProfilingPanel) renderBody(status *ProfilingStatus, err error) string {
	if IsServiceUnavailable(err) {
		return RenderDimText(ServiceUnavailableHint("On-demand profiling",
			"Restart piko with --enable-profiling to enable this panel."))
	}
	if err != nil {
		var b strings.Builder
		RenderErrorState(&b, err)
		return strings.TrimSuffix(b.String(), stringNewline)
	}
	if status == nil {
		return RenderDimText("Fetching profiling status...")
	}
	return RenderDetailBody(nil, p.detailBody(status, nil).WithoutHeader(), p.ContentWidth(), p.ContentHeight())
}

// detailBody builds the structured inspector.DetailBody for the profiling panel.
//
// Takes status (*ProfilingStatus) which is the latest status snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns inspector.DetailBody describing the panel state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProfilingPanel) detailBody(status *ProfilingStatus, err error) inspector.DetailBody {
	if err != nil {
		return inspector.DetailBody{
			Title:    titleProfiling,
			Sections: []inspector.DetailSection{{Heading: "Error", Rows: []inspector.DetailRow{{Label: "Reason", Value: err.Error()}}}},
		}
	}
	if status == nil {
		return inspector.DetailBody{Title: titleProfiling, Subtitle: "no data yet"}
	}

	sections := []inspector.DetailSection{{Heading: "Status", Rows: profilingStatusRows(status)}}
	sections = append(sections, p.captureSection())
	p.stateMutex.RLock()
	summary := p.lastSummary
	p.stateMutex.RUnlock()
	if summary != nil && len(summary.Entries) > 0 {
		sections = append(sections, profileTopSection(summary))
	}

	return inspector.DetailBody{
		Title:    titleProfiling,
		Subtitle: "press e to enable, d to disable, c for CPU, h for heap",
		Sections: sections,
	}
}

// profilingStatusRows builds the Status section rows from the latest
// status snapshot. Hoisted from detailBody to keep the parent under
// the cyclomatic-complexity threshold.
//
// Takes status (*ProfilingStatus) which is the snapshot.
//
// Returns []inspector.DetailRow ready to wrap in a inspector.DetailSection.
func profilingStatusRows(status *ProfilingStatus) []inspector.DetailRow {
	enabledLabel := "disabled"
	if status.Enabled {
		enabledLabel = "enabled"
	}
	rows := []inspector.DetailRow{
		{Label: "State", Value: enabledLabel},
		{Label: "Available", Value: strings.Join(status.AvailableProfiles, ", ")},
		{Label: "Pprof URL", Value: status.PprofBaseURL},
		{Label: "Port", Value: fmt.Sprintf(fmtDecimal, status.Port)},
		{Label: "Block rate", Value: fmt.Sprintf(fmtDecimal, status.BlockProfileRate)},
		{Label: "Mutex frac", Value: fmt.Sprintf(fmtDecimal, status.MutexProfileFraction)},
		{Label: "Mem rate", Value: fmt.Sprintf(fmtDecimal, status.MemProfileRate)},
	}
	if !status.ExpiresAt.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Expires", Value: status.ExpiresAt.Format(time.RFC3339)})
	}
	if status.Remaining > 0 {
		rows = append(rows, inspector.DetailRow{Label: "Remaining", Value: status.Remaining.Truncate(time.Second).String()})
	}
	return rows
}

// captureSection builds the "Capture" section based on the panel's
// in-flight / last / error state. Returns a section with at most one
// row even when no capture has run, so callers can append it
// unconditionally.
//
// Returns inspector.DetailSection describing the capture state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProfilingPanel) captureSection() inspector.DetailSection {
	p.stateMutex.RLock()
	captureMessage := p.captureMessage
	captureErr := p.captureErr
	capturing := p.capturing
	p.stateMutex.RUnlock()

	switch {
	case capturing:
		return inspector.DetailSection{Heading: captureSectionHeading, Rows: []inspector.DetailRow{{Label: "Status", Value: "in flight..."}}}
	case captureErr != nil:
		return inspector.DetailSection{Heading: captureSectionHeading, Rows: []inspector.DetailRow{
			{Label: "Last", Value: captureMessage},
			{Label: "Error", Value: captureErr.Error()},
		}}
	case captureMessage != "":
		return inspector.DetailSection{Heading: captureSectionHeading, Rows: []inspector.DetailRow{{Label: "Last", Value: captureMessage}}}
	default:
		return inspector.DetailSection{Heading: captureSectionHeading, Rows: []inspector.DetailRow{{Label: "Status", Value: "no capture yet"}}}
	}
}

// profileTopSection builds the "TOP FUNCTIONS" detail section from
// the parsed pprof summary. Each row shows the function name (left)
// and a flat-percentage + value (right).
//
// Takes summary (*inspector.ProfileSummary) which is the parsed top-N list.
//
// Returns inspector.DetailSection ready to append to the detail body.
func profileTopSection(summary *inspector.ProfileSummary) inspector.DetailSection {
	rows := make([]inspector.DetailRow, 0, len(summary.Entries))
	for _, e := range summary.Entries {
		pct := 0.0
		if summary.Total > 0 {
			pct = float64(e.Flat) / float64(summary.Total) * percentScale
		}
		value := fmt.Sprintf("%5.1f%%  %d %s", pct, e.Flat, summary.SampleUnit)
		rows = append(rows, inspector.DetailRow{Label: e.Function, Value: value})
	}
	heading := fmt.Sprintf("Top %s (flat)", summary.SampleType)
	return inspector.DetailSection{Heading: heading, Rows: rows}
}

// refresh returns a Cmd that fetches the latest profiling status.
//
// Returns tea.Cmd which delivers a profilingRefreshMessage.
func (p *ProfilingPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return profilingRefreshMessage{err: errNoProfilingInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), profilingRefreshTimeout,
			errors.New("profiling status exceeded timeout"))
		defer cancel()
		status, err := p.provider.Status(ctx)
		return profilingRefreshMessage{status: status, err: err}
	}
}

// enableCmd returns a Cmd that enables on-demand profiling.
//
// Returns tea.Cmd which delivers a profilingActionMessage.
func (p *ProfilingPanel) enableCmd() tea.Cmd {
	provider := p.provider
	return func() tea.Msg {
		if provider == nil {
			return profilingActionMessage{action: "enable", err: errNoProfilingInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), profilingRefreshTimeout,
			errors.New("enable profiling exceeded timeout"))
		defer cancel()
		err := provider.Enable(ctx)
		return profilingActionMessage{action: "enable", err: err}
	}
}

// disableCmd returns a Cmd that disables on-demand profiling.
//
// Returns tea.Cmd which delivers a profilingActionMessage.
func (p *ProfilingPanel) disableCmd() tea.Cmd {
	provider := p.provider
	return func() tea.Msg {
		if provider == nil {
			return profilingActionMessage{action: "disable", err: errNoProfilingInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), profilingRefreshTimeout,
			errors.New("disable profiling exceeded timeout"))
		defer cancel()
		err := provider.Disable(ctx)
		return profilingActionMessage{action: "disable", err: err}
	}
}

// captureCmd returns a Cmd that captures a profile of the named kind.
//
// Takes profile (string) which names the profile kind (cpu, heap, ...).
// Takes duration (time.Duration) which is the sampling window.
//
// Returns tea.Cmd that delivers a profilingCaptureMessage and pushes
// the capture-in-progress overlay.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProfilingPanel) captureCmd(profile string, duration time.Duration) tea.Cmd {
	provider := p.provider
	clk := p.clock
	p.stateMutex.Lock()
	p.capturing = true
	p.captureErr = nil
	p.captureMessage = profile + " capturing..."
	p.stateMutex.Unlock()

	captureFn := func() tea.Msg {
		if provider == nil {
			return profilingCaptureMessage{profile: profile, err: errNoProfilingInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), duration+profilingRefreshTimeout,
			fmt.Errorf("%s profile capture exceeded timeout", profile))
		defer cancel()
		data, err := provider.Capture(ctx, profile, duration)
		if err != nil {
			return profilingCaptureMessage{profile: profile, err: err}
		}
		path, writeErr := writeCaptureToTempFile(clk, profile, data)
		if writeErr != nil {
			return profilingCaptureMessage{profile: profile, err: writeErr}
		}
		summary, parseErr := inspector.ParseProfileSummary(data, inspector.ProfileAggOpts{
			SampleIndex: 0,
			TopN:        inspector.ProfileTopDefault,
		})
		if parseErr != nil {
			return profilingCaptureMessage{profile: profile, path: path, bytes: len(data), err: parseErr}
		}
		return profilingCaptureMessage{profile: profile, path: path, bytes: len(data), summary: summary}
	}

	overlayPush := func() tea.Msg {
		return pushOverlayMessage{Overlay: NewProfilingCaptureOverlay(profile, duration, clk)}
	}

	return tea.Batch(overlayPush, captureFn)
}

// writeCaptureToTempFile persists the capture bytes to a deterministic
// temp-dir filename so the user can analyse it externally. The
// profile kind is sanitised to base-name characters so a hostile or
// mistyped value cannot escape the temp directory via path traversal.
//
// Takes clk (clock.Clock) which supplies the timestamp baked into the
// filename so tests can assert against a known clock.
// Takes profile (string) which names the profile kind.
// Takes data ([]byte) which is the captured pprof payload.
//
// Returns string which is the absolute path to the written file.
// Returns error which is non-nil when the file cannot be written.
func writeCaptureToTempFile(clk clock.Clock, profile string, data []byte) (string, error) {
	dir := os.TempDir()
	name := fmt.Sprintf("piko-%s-%s.pprof", sanitiseProfileName(profile), clk.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, profileFilePerm); err != nil {
		return "", fmt.Errorf("write %s capture: %w", profile, err)
	}
	return path, nil
}

// sanitiseProfileName strips any character that is not a letter,
// digit, hyphen, or underscore so the value is safe to interpolate
// into a filename without enabling directory traversal.
//
// Takes profile (string) which is the raw profile kind.
//
// Returns string containing only filename-safe characters; "profile"
// when every input character was disallowed.
func sanitiseProfileName(profile string) string {
	var b strings.Builder
	b.Grow(len(profile))
	for _, r := range profile {
		if isFilenameSafeRune(r) {
			_, _ = b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "profile"
	}
	return b.String()
}

// isFilenameSafeRune reports whether r is a letter, digit, hyphen,
// or underscore. Used to gate which runes survive the profile-name
// sanitiser.
//
// Takes r (rune) which is the rune to classify.
//
// Returns bool which is true when the rune is filename-safe.
func isFilenameSafeRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_':
		return true
	}
	return false
}
