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
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
	"piko.sh/piko/cmd/piko/internal/inspector"
)

// ErrServiceUnavailable is the sentinel error panels detect via
// errors.Is when the server does not expose the requested gRPC
// service. The adapter wraps the gRPC codes.Unimplemented error in
// this sentinel so callers can react with a "feature disabled on
// the server" hint instead of a raw RPC error.
var ErrServiceUnavailable = errors.New("service not available on this server")

// IsServiceUnavailable reports whether err signals that the server
// does not expose the requested gRPC service.
//
// Takes err (error) which is the inspector's most recent error.
//
// Returns bool which is true when err signals a missing service.
func IsServiceUnavailable(err error) bool {
	return errors.Is(err, ErrServiceUnavailable)
}

// ServiceUnavailableHint returns a human-friendly explanation rendered
// in the panel body when a feature is disabled on the server. The
// hint mentions which command-line flag controls it so the user can
// recover.
//
// Takes feature (string) which names the feature ("DLQ", "Profiling").
// Takes flagHint (string) which describes how to enable the feature
// on the server (e.g. "start piko with --enable-profiling").
//
// Returns string containing the rendered hint.
func ServiceUnavailableHint(feature, flagHint string) string {
	if flagHint == "" {
		return feature + " is not exposed by this server."
	}
	return feature + " is not exposed by this server. " + flagHint
}

// inspectorOverviewArgs bundles the inputs to inspectorOverviewBody so
// new fields can be added without churning every call site.
type inspectorOverviewArgs struct {
	// lastRefresh is when the panel last received a refresh; the row
	// is omitted when zero.
	lastRefresh time.Time

	// err is the most recent refresh error; the row is omitted when
	// nil.
	err error

	// title is the heading shown at the top of the detail body.
	title string

	// itemLabel labels the row count (e.g. "Providers", "Dispatchers").
	itemLabel string

	// itemCount is the cached total row count.
	itemCount int
}

// inspectorOverviewBody builds the standard "no row selected"
// inspector.DetailBody used by the inspector-style panels (Providers, DLQ, ...).
// It surfaces the row count, last-refresh timestamp, and any error
// under a single "Status" section.
//
// Takes args (inspectorOverviewArgs) which provides the title and
// counters; see the field comments for details.
//
// Returns inspector.DetailBody ready to pass to RenderDetailBody.
func inspectorOverviewBody(args inspectorOverviewArgs) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: args.itemLabel, Value: formatInt(args.itemCount)},
	}
	if !args.lastRefresh.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Last refresh", Value: args.lastRefresh.Format(time.RFC3339)})
	}
	if args.err != nil {
		rows = append(rows, inspector.DetailRow{Label: "Error", Value: args.err.Error()})
	}
	return inspector.DetailBody{
		Title:    args.title,
		Sections: []inspector.DetailSection{{Heading: "Status", Rows: rows}},
	}
}

// handleOverviewControlMessage handles the two messages every overview
// panel responds to in the same way: the 'r' key triggers a refresh,
// and DataUpdatedMessage / TickMessage triggers a refresh too.
// Keeping the boilerplate in one place lets each panel's Update body
// focus on the panel-specific payload type.
//
// Takes msg (tea.Msg) which is the bubbletea message to inspect.
// Takes refresh (func() tea.Cmd) which produces a fresh fetch command.
//
// Returns tea.Cmd which is the resulting command (may be nil).
// Returns bool which is true when the message was consumed.
func handleOverviewControlMessage(msg tea.Msg, refresh func() tea.Cmd) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		if m.String() == "r" {
			cmd := refresh()
			return cmd, true
		}
	case DataUpdatedMessage, TickMessage:
		cmd := refresh()
		return cmd, true
	}
	return nil, false
}

// forwardSearchUpdate routes msg to the AssetViewer's search input
// when one is active. Lets inspector-style panels delegate the
// "search active swallows keys" prologue to a shared helper instead
// of repeating the if/IsActive/Update dance verbatim in every
// Update method.
//
// Generic parameter T is the AssetViewer item type.
//
// Takes viewer (*AssetViewer[T]) which is the embedded list view.
// Takes msg (tea.Msg) which is the bubbletea message to route.
//
// Returns tea.Cmd which is the resulting command (may be nil).
// Returns bool which is true when the search consumed the message.
func forwardSearchUpdate[T any](viewer *AssetViewer[T], msg tea.Msg) (tea.Cmd, bool) {
	if viewer.Search() == nil || !viewer.Search().IsActive() {
		return nil, false
	}
	handled, cmd := viewer.Search().Update(msg)
	return cmd, handled
}
