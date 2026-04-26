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
	"io"
	"time"
)

// WatchdogProvider supplies watchdog state to the TUI. Implementations
// wrap the gRPC inspector client; Refresh polls snapshot operations
// concurrently and caches them so panels read synchronously.
type WatchdogProvider interface {
	RefreshableProvider

	// GetStatus returns the most recently cached watchdog status.
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	//
	// Returns *WatchdogStatus which is the cached snapshot. Implementations
	// that have not yet refreshed return a zero-value status with
	// LastUpdated zero so callers can detect the stale state.
	// Returns error when the underlying transport fails.
	GetStatus(ctx context.Context) (*WatchdogStatus, error)

	// ListProfiles returns the cached list of stored profile artefacts in
	// reverse chronological order.
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	//
	// Returns []WatchdogProfile which is the cached profile inventory.
	// Returns error when the underlying transport fails.
	ListProfiles(ctx context.Context) ([]WatchdogProfile, error)

	// GetStartupHistory returns the cached startup-history entries in
	// chronological order (oldest first).
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	//
	// Returns []WatchdogStartupEntry which is the cached history.
	// Returns error when the underlying transport fails.
	GetStartupHistory(ctx context.Context) ([]WatchdogStartupEntry, error)

	// ListEvents queries historical events from the server's in-memory
	// ring. This is a one-shot RPC; the streaming feed is exposed through
	// SubscribeEvents.
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	// Takes query (WatchdogEventQuery) which describes the filter.
	//
	// Returns []WatchdogEvent which is the matching events in
	// chronological order.
	// Returns error when the underlying transport fails.
	ListEvents(ctx context.Context, query WatchdogEventQuery) ([]WatchdogEvent, error)

	// SubscribeEvents opens a streaming subscription to live events,
	// optionally back-filling from since.
	//
	// Takes ctx (context.Context) which controls the subscription
	// lifetime; cancelling it closes the channel.
	// Takes since (time.Time) which is the back-fill cutoff. Zero
	// disables back-fill.
	//
	// Returns <-chan WatchdogEvent which delivers events in emission
	// order.
	// Returns func() which cancels the subscription.
	// Returns error when the subscription cannot be opened.
	SubscribeEvents(ctx context.Context, since time.Time) (<-chan WatchdogEvent, func(), error)

	// DroppedEvents returns the running count of events that the
	// underlying stream had to drop because the consumer fell behind.
	//
	// Returns uint64 which is the cumulative drop count.
	DroppedEvents() uint64

	// PruneProfiles removes stored profiles. When profileType is empty
	// every profile is removed.
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	// Takes profileType (string) which selects the profile category.
	//
	// Returns int which is the number of profiles removed.
	// Returns error when the underlying transport fails.
	PruneProfiles(ctx context.Context, profileType string) (int, error)

	// DownloadProfile streams the raw compressed bytes of a profile to
	// the supplied writer.
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	// Takes filename (string) which identifies the profile.
	// Takes w (io.Writer) which receives the bytes.
	//
	// Returns error when the transfer fails.
	DownloadProfile(ctx context.Context, filename string, w io.Writer) error

	// DownloadSidecar fetches the JSON sidecar paired with a profile.
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	// Takes profileFilename (string) which is the profile filename.
	//
	// Returns []byte which is the sidecar JSON, or nil when no sidecar
	// exists.
	// Returns bool which is true when a sidecar was found.
	// Returns error when the underlying transport fails for reasons
	// other than the sidecar being absent.
	DownloadSidecar(ctx context.Context, profileFilename string) ([]byte, bool, error)

	// RunContentionDiagnostic triggers an on-demand contention diagnostic
	// on the server. The call blocks until the diagnostic completes.
	//
	// Takes ctx (context.Context) which controls the request lifetime.
	//
	// Returns error when the diagnostic cannot start (already running,
	// in cooldown, watchdog stopped) or fails partway through.
	RunContentionDiagnostic(ctx context.Context) error
}
