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

package provider_grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/internal/goroutine"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

// downloadProfileMaxBytes caps the bytes a single DownloadProfile
// stream may deliver before being aborted. Without a cap, a hostile or
// runaway server could keep streaming indefinitely and exhaust the
// caller's writer (often a file or in-memory buffer).
const downloadProfileMaxBytes = 256 * 1024 * 1024

// downloadSidecarMaxBytes caps the size of a single sidecar payload.
// Sidecars are small JSON metadata; 16 MiB is well above typical sizes
// while still bounding the response.
const downloadSidecarMaxBytes = 16 * 1024 * 1024

// ErrWatchdogDownloadTooLarge is returned by DownloadProfile when the
// streamed bytes exceed downloadProfileMaxBytes.
var ErrWatchdogDownloadTooLarge = errors.New("watchdog profile download exceeded byte budget")

// ErrWatchdogSidecarTooLarge is returned by DownloadSidecar when the
// payload exceeds downloadSidecarMaxBytes.
var ErrWatchdogSidecarTooLarge = errors.New("watchdog sidecar payload exceeded byte budget")

var _ tui_domain.WatchdogProvider = (*WatchdogProvider)(nil)

// WatchdogProvider bridges the watchdog gRPC inspector service to the
// TUI's WatchdogProvider interface. Snapshot calls populate an in-memory
// cache on Refresh; streaming events are exposed through SubscribeEvents
// directly.
type WatchdogProvider struct {
	// conn is the shared gRPC connection used for all RPCs.
	conn *Connection

	// status is the cached watchdog status snapshot.
	status *tui_domain.WatchdogStatus

	// profiles is the cached profile inventory.
	profiles []tui_domain.WatchdogProfile

	// history is the cached startup history.
	history []tui_domain.WatchdogStartupEntry

	// dropped counts events dropped because the local channel was full.
	dropped atomic.Uint64

	// interval is the snapshot refresh cadence requested by the host.
	interval time.Duration

	// mu guards status, profiles, and history.
	mu sync.RWMutex
}

// NewWatchdogProvider creates a new gRPC-backed WatchdogProvider.
//
// Takes conn (*Connection) which is the shared gRPC connection.
// Takes interval (time.Duration) which is the snapshot refresh cadence.
//
// Returns *WatchdogProvider ready to register on the service.
func NewWatchdogProvider(conn *Connection, interval time.Duration) *WatchdogProvider {
	return &WatchdogProvider{
		conn:     conn,
		interval: interval,
	}
}

// Name implements tui_domain.Provider.
//
// Returns string which is the provider identifier.
func (*WatchdogProvider) Name() string { return "grpc-watchdog" }

// Health performs a basic gRPC health check against the connection.
//
// Returns error when the context is cancelled, the health probe fails,
// or no connection exists. The context is checked first so a cancelled
// context produces ctx.Err rather than masking the cancellation as a
// "no connection" error.
func (p *WatchdogProvider) Health(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.conn == nil {
		return tui_domain.ErrWatchdogNoConnection
	}
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking watchdog provider health via gRPC: %w", err)
	}
	return nil
}

// Close releases the provider's resources. The underlying connection is
// not closed here because it is shared with other providers.
//
// Returns error which is always nil.
func (*WatchdogProvider) Close() error { return nil }

// RefreshInterval implements tui_domain.RefreshableProvider.
//
// Returns time.Duration which is the configured refresh interval.
func (p *WatchdogProvider) RefreshInterval() time.Duration { return p.interval }

// Refresh populates the cache by issuing the three snapshot RPCs
// concurrently. Partial failures are tolerated: successful sub-results
// still update the cache, while the first error encountered is returned.
//
// Returns error which is the first non-nil per-RPC error, or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu and spawns
// goroutines via WaitGroup.
func (p *WatchdogProvider) Refresh(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.conn == nil {
		return tui_domain.ErrWatchdogNoConnection
	}

	var (
		wg                                 sync.WaitGroup
		statusErr, profilesErr, historyErr error
		status                             *tui_domain.WatchdogStatus
		profiles                           []tui_domain.WatchdogProfile
		history                            []tui_domain.WatchdogStartupEntry
	)

	wg.Go(func() {
		defer goroutine.RecoverPanic(ctx, "watchdog-grpc.fetchStatus")
		s, err := p.fetchStatus(ctx)
		status = s
		statusErr = err
	})
	wg.Go(func() {
		defer goroutine.RecoverPanic(ctx, "watchdog-grpc.fetchProfiles")
		s, err := p.fetchProfiles(ctx)
		profiles = s
		profilesErr = err
	})
	wg.Go(func() {
		defer goroutine.RecoverPanic(ctx, "watchdog-grpc.fetchHistory")
		s, err := p.fetchHistory(ctx)
		history = s
		historyErr = err
	})
	wg.Wait()

	p.mu.Lock()
	if statusErr == nil {
		p.status = status
	}
	if profilesErr == nil {
		p.profiles = profiles
	}
	if historyErr == nil {
		p.history = history
	}
	p.mu.Unlock()

	return errors.Join(statusErr, profilesErr, historyErr)
}

// GetStatus returns the cached snapshot.
//
// Returns *tui_domain.WatchdogStatus which is the cached status, or nil.
// Returns error which is always nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogProvider) GetStatus(_ context.Context) (*tui_domain.WatchdogStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status, nil
}

// ListProfiles returns the cached profile inventory.
//
// Returns []tui_domain.WatchdogProfile which is a copy of the inventory.
// Returns error which is always nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogProvider) ListProfiles(_ context.Context) ([]tui_domain.WatchdogProfile, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]tui_domain.WatchdogProfile, len(p.profiles))
	copy(out, p.profiles)
	return out, nil
}

// GetStartupHistory returns the cached history.
//
// Returns []tui_domain.WatchdogStartupEntry which is a copy of the history.
// Returns error which is always nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogProvider) GetStartupHistory(_ context.Context) ([]tui_domain.WatchdogStartupEntry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]tui_domain.WatchdogStartupEntry, len(p.history))
	copy(out, p.history)
	return out, nil
}

// ListEvents performs a one-shot ListEvents RPC and converts the result
// to TUI-side events.
//
// Takes query (tui_domain.WatchdogEventQuery) which carries the filter.
//
// Returns []tui_domain.WatchdogEvent which is the converted event list.
// Returns error when the RPC fails or no connection exists.
func (p *WatchdogProvider) ListEvents(ctx context.Context, query tui_domain.WatchdogEventQuery) ([]tui_domain.WatchdogEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.conn == nil {
		return nil, tui_domain.ErrWatchdogNoConnection
	}
	request := &pb.ListEventsRequest{
		Limit:     safeconv.IntToInt32(query.Limit),
		EventType: query.EventType,
	}
	if !query.Since.IsZero() {
		request.SinceMs = query.Since.UnixMilli()
	}
	response, err := p.conn.watchdogClient.ListEvents(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("watchdog ListEvents: %w", err)
	}
	out := make([]tui_domain.WatchdogEvent, 0, len(response.GetEvents()))
	for _, ev := range response.GetEvents() {
		out = append(out, convertWatchdogEvent(ev))
	}
	return out, nil
}

// SubscribeEvents opens a server-streaming subscription and forwards
// events on a buffered channel. Drops are accounted for via the
// atomic dropped counter.
//
// Takes since (time.Time) which is the back-fill cutoff. Zero disables
// back-fill.
//
// Returns <-chan tui_domain.WatchdogEvent which delivers events.
// Returns func() which cancels the subscription.
// Returns error when the RPC fails to open.
func (p *WatchdogProvider) SubscribeEvents(ctx context.Context, since time.Time) (<-chan tui_domain.WatchdogEvent, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if p.conn == nil {
		return nil, nil, tui_domain.ErrWatchdogNoConnection
	}
	request := &pb.WatchEventsRequest{}
	if !since.IsZero() {
		request.SinceMs = since.UnixMilli()
	}

	streamCtx, cancel := context.WithCancelCause(ctx)
	stream, err := p.conn.watchdogClient.WatchEvents(streamCtx, request)
	if err != nil {
		cancel(errors.New("watchdog WatchEvents stream open failed"))
		return nil, nil, fmt.Errorf("watchdog WatchEvents: %w", err)
	}

	out := make(chan tui_domain.WatchdogEvent, tui_domain.EventStreamBufferSize)

	var wg sync.WaitGroup
	wg.Go(func() {
		defer close(out)
		defer goroutine.RecoverPanic(streamCtx, "watchdog-grpc.WatchEvents")
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case <-streamCtx.Done():
				return
			case out <- convertWatchdogEvent(msg):
			default:
				p.dropped.Add(1)
			}
		}
	})

	cancelFn := func() {
		cancel(errors.New("watchdog event subscription cancelled"))
		wg.Wait()
	}
	return out, cancelFn, nil
}

// DroppedEvents returns the cumulative drop count for the upstream
// stream.
//
// Returns uint64 which is the cumulative drop count.
func (p *WatchdogProvider) DroppedEvents() uint64 { return p.dropped.Load() }

// PruneProfiles issues the PruneProfiles RPC.
//
// Takes profileType (string) which selects the profile kind to prune.
//
// Returns int which is the number of pruned profiles.
// Returns error when the RPC fails or no connection exists.
func (p *WatchdogProvider) PruneProfiles(ctx context.Context, profileType string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if p.conn == nil {
		return 0, tui_domain.ErrWatchdogNoConnection
	}
	response, err := p.conn.watchdogClient.PruneProfiles(ctx, &pb.PruneProfilesRequest{ProfileType: profileType})
	if err != nil {
		return 0, fmt.Errorf("watchdog PruneProfiles: %w", err)
	}
	return safeconv.Int32ToInt(response.GetDeletedCount()), nil
}

// DownloadProfile streams the profile bytes to the supplied writer.
// The total bytes received are capped at downloadProfileMaxBytes;
// exceeding this cap returns ErrWatchdogDownloadTooLarge so a hostile
// server cannot exhaust the caller's writer.
//
// Takes filename (string) which selects the profile to download.
// Takes w (io.Writer) which receives the streamed bytes.
//
// Returns error when the RPC fails, a write fails, or the byte cap
// is exceeded.
func (p *WatchdogProvider) DownloadProfile(ctx context.Context, filename string, w io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.conn == nil {
		return tui_domain.ErrWatchdogNoConnection
	}
	stream, err := p.conn.watchdogClient.DownloadProfile(ctx, &pb.DownloadProfileRequest{Filename: filename})
	if err != nil {
		return fmt.Errorf("watchdog DownloadProfile: %w", err)
	}
	var received int64
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("watchdog DownloadProfile receive: %w", err)
		}
		data := chunk.GetData()
		received += int64(len(data))
		if received > downloadProfileMaxBytes {
			return fmt.Errorf("%w: received %d bytes, cap %d", ErrWatchdogDownloadTooLarge, received, downloadProfileMaxBytes)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("watchdog DownloadProfile write: %w", err)
		}
	}
}

// DownloadSidecar fetches a profile's JSON sidecar. The payload size
// is capped at downloadSidecarMaxBytes so a hostile server cannot
// force the client to hold an arbitrarily large blob.
//
// Takes profileFilename (string) which is the profile filename.
//
// Returns []byte which is the sidecar payload.
// Returns bool which is true when a sidecar exists.
// Returns error when the RPC fails, no connection exists, or the
// payload exceeds the byte cap.
func (p *WatchdogProvider) DownloadSidecar(ctx context.Context, profileFilename string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if p.conn == nil {
		return nil, false, tui_domain.ErrWatchdogNoConnection
	}
	response, err := p.conn.watchdogClient.DownloadSidecar(ctx, &pb.DownloadSidecarRequest{ProfileFilename: profileFilename})
	if err != nil {
		return nil, false, fmt.Errorf("watchdog DownloadSidecar: %w", err)
	}
	data := response.GetData()
	if len(data) > downloadSidecarMaxBytes {
		return nil, false, fmt.Errorf("%w: %d bytes, cap %d", ErrWatchdogSidecarTooLarge, len(data), downloadSidecarMaxBytes)
	}
	return data, response.GetPresent(), nil
}

// RunContentionDiagnostic invokes the diagnostic RPC.
//
// Returns error when the RPC fails or the diagnostic does not start.
func (p *WatchdogProvider) RunContentionDiagnostic(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.conn == nil {
		return tui_domain.ErrWatchdogNoConnection
	}
	response, err := p.conn.watchdogClient.RunContentionDiagnostic(ctx, &pb.RunContentionDiagnosticRequest{})
	if err != nil {
		return fmt.Errorf("watchdog RunContentionDiagnostic: %w", err)
	}
	if !response.GetStarted() {
		if reason := response.GetError(); reason != "" {
			return errors.New(reason)
		}
		return errors.New("contention diagnostic did not start")
	}
	return nil
}

// fetchStatus issues the GetWatchdogStatus RPC and converts the
// response to the TUI-side WatchdogStatus type.
//
// Returns *tui_domain.WatchdogStatus which is the converted status.
// Returns error when the RPC fails.
func (p *WatchdogProvider) fetchStatus(ctx context.Context) (*tui_domain.WatchdogStatus, error) {
	response, err := p.conn.watchdogClient.GetWatchdogStatus(ctx, &pb.GetWatchdogStatusRequest{})
	if err != nil {
		return nil, fmt.Errorf("watchdog GetWatchdogStatus: %w", err)
	}
	return convertWatchdogStatus(response), nil
}

// fetchProfiles issues the ListProfiles RPC.
//
// Returns []tui_domain.WatchdogProfile which is the converted inventory.
// Returns error when the RPC fails.
func (p *WatchdogProvider) fetchProfiles(ctx context.Context) ([]tui_domain.WatchdogProfile, error) {
	response, err := p.conn.watchdogClient.ListProfiles(ctx, &pb.ListProfilesRequest{})
	if err != nil {
		return nil, fmt.Errorf("watchdog ListProfiles: %w", err)
	}
	out := make([]tui_domain.WatchdogProfile, 0, len(response.GetProfiles()))
	for _, entry := range response.GetProfiles() {
		out = append(out, convertWatchdogProfile(entry))
	}
	return out, nil
}

// fetchHistory issues the GetStartupHistory RPC.
//
// Returns []tui_domain.WatchdogStartupEntry which is the converted history.
// Returns error when the RPC fails.
func (p *WatchdogProvider) fetchHistory(ctx context.Context) ([]tui_domain.WatchdogStartupEntry, error) {
	response, err := p.conn.watchdogClient.GetStartupHistory(ctx, &pb.GetStartupHistoryRequest{})
	if err != nil {
		return nil, fmt.Errorf("watchdog GetStartupHistory: %w", err)
	}
	out := make([]tui_domain.WatchdogStartupEntry, 0, len(response.GetEntries()))
	for _, entry := range response.GetEntries() {
		out = append(out, convertStartupEntry(entry))
	}
	return out, nil
}
