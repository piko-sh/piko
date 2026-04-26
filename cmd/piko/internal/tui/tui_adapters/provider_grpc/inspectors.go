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
	"math"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// captureDeadlineSlack is the extra time added to the user-supplied
	// capture duration before the gRPC context expires. The server
	// performs a sampling pass plus encode after the window closes, so
	// a hard deadline equal to duration would race the response.
	captureDeadlineSlack = 5 * time.Second

	// providersInitialCap is the initial capacity for the aggregated
	// providers slice. Sized for typical hexagons; grows as needed.
	providersInitialCap = 32

	// captureMaxBytes caps the concatenated profile bytes returned by
	// a CaptureProfile stream. Profiles larger than this abort with
	// ErrCaptureTooLarge so a hostile or runaway server cannot OOM
	// the TUI.
	captureMaxBytes = 256 * 1024 * 1024

	// captureChunkMaxBytes caps any single chunk returned by the
	// stream. Defensive against a malformed server.
	captureChunkMaxBytes = 32 * 1024 * 1024

	// captureMaxDurationSeconds caps the requested sampling window so
	// the int32 wire field cannot overflow.
	captureMaxDurationSeconds = 600

	// dlqLimitMax caps the requested DLQ entry count so the int32
	// wire field cannot overflow and so the TUI never asks for an
	// unbounded slice.
	dlqLimitMax = 10_000

	// listProvidersMaxConcurrency caps fan-out across resource types
	// when listing providers.
	listProvidersMaxConcurrency = 8

	// listProvidersPerCallTimeout bounds each per-resource-type
	// ListProviders RPC so a slow type cannot stall the whole list.
	listProvidersPerCallTimeout = 3 * time.Second
)

var (
	// ErrServiceUnavailable is the alias for tui_domain.ErrServiceUnavailable
	// kept here so adapter callers can errors.Is against it without
	// importing the domain package.
	ErrServiceUnavailable = tui_domain.ErrServiceUnavailable

	// ErrCaptureTooLarge is returned by Capture when the streamed
	// profile bytes exceed captureMaxBytes.
	ErrCaptureTooLarge = errors.New("profile capture exceeded byte budget")

	// ErrCaptureChunkTooLarge is returned by Capture when a single
	// stream chunk exceeds captureChunkMaxBytes.
	ErrCaptureChunkTooLarge = errors.New("profile capture chunk exceeded byte budget")

	// ErrCaptureDurationOutOfRange is returned by Capture when the
	// caller-requested duration cannot fit into the int32 wire field.
	ErrCaptureDurationOutOfRange = errors.New("profile capture duration out of range")
)

// Compile-time assertions that each adapter satisfies its port. Kept
// here so decorder's strict const,var,func ordering rule passes.
var (
	_ tui_domain.ProvidersInspector = (*ProvidersInspector)(nil)

	_ tui_domain.DLQInspector = (*DLQInspector)(nil)

	_ tui_domain.RateLimiterInspector = (*RateLimiterInspector)(nil)

	_ tui_domain.ProfilingInspector = (*ProfilingInspector)(nil)
)

// translateRPCError maps codes.Unimplemented to ErrServiceUnavailable.
//
// codes.Unimplemented is returned by gRPC when the requested service or
// method is not registered. Mapping to ErrServiceUnavailable lets callers
// detect "feature disabled" via errors.Is. The original gRPC error is
// preserved alongside the sentinel via a multi-target %w so callers can
// also inspect the underlying gRPC status. Other errors pass through
// unchanged.
//
// Takes err (error) which is the raw gRPC error.
//
// Returns error which wraps both ErrServiceUnavailable and the original
// gRPC error for unimplemented services, or the original err otherwise.
func translateRPCError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
		return fmt.Errorf("%w: %s: %w", ErrServiceUnavailable, st.Message(), err)
	}
	return err
}

// ProvidersInspector adapts the gRPC ProviderInfo service to the
// tui_domain.ProvidersInspector port.
type ProvidersInspector struct {
	// conn is the shared gRPC connection used for all RPCs.
	conn *Connection
}

// NewProvidersInspector constructs a ProvidersInspector backed by conn.
//
// Takes conn (*Connection) which is the shared gRPC connection.
//
// Returns *ProvidersInspector ready to expose via tui_domain.Providers.
func NewProvidersInspector(conn *Connection) *ProvidersInspector {
	return &ProvidersInspector{conn: conn}
}

// Name reports the provider identifier.
//
// Returns string which is the inspector identifier.
func (*ProvidersInspector) Name() string { return "grpc-providers" }

// Health checks the underlying gRPC connection.
//
// Takes ctx (context.Context) which controls the call lifetime.
//
// Returns error when the health probe fails.
func (p *ProvidersInspector) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking providers inspector health: %w", err)
	}
	return nil
}

// Close is a no-op; the connection is owned by the parent.
//
// Returns error which is always nil.
func (*ProvidersInspector) Close() error { return nil }

// ListProviders fans out across known resource types and aggregates
// the responses into a single flat list.
//
// Concurrency is bounded at listProvidersMaxConcurrency; per-call
// deadlines stop a slow type from stalling the whole list. Per-type
// errors are joined and returned alongside the partial list so callers
// can both render what succeeded and surface what did not.
//
// Takes ctx (context.Context) which controls the fan-out lifetime.
//
// Returns []tui_domain.ProviderEntry which is the aggregated list.
// Returns error which is the joined per-type errors, or nil.
func (p *ProvidersInspector) ListProviders(ctx context.Context) ([]tui_domain.ProviderEntry, error) {
	typesResp, err := p.conn.providerInfoClient.ListResourceTypes(ctx, &pb.ListResourceTypesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list resource types: %w", translateRPCError(err))
	}

	types := typesResp.GetResourceTypes()
	if len(types) == 0 {
		return nil, nil
	}

	results := make([]listProvidersResult, len(types))
	semaphore := make(chan struct{}, listProvidersMaxConcurrency)
	var wg sync.WaitGroup
	for i, rt := range types {
		semaphore <- struct{}{}
		wg.Go(func() {
			defer func() { <-semaphore }()
			results[i] = p.listOneType(ctx, rt)
		})
	}
	wg.Wait()

	out := make([]tui_domain.ProviderEntry, 0, providersInitialCap)
	var errs []error
	for _, r := range results {
		out = append(out, r.entries...)
		if r.err != nil {
			errs = append(errs, r.err)
		}
	}
	return out, errors.Join(errs...)
}

// listProvidersResult captures the per-resource-type fan-out result so
// the parent goroutine can join the partial entries and any error
// after wg.Wait() returns.
type listProvidersResult struct {
	// err is the per-type RPC error, or nil on success.
	err error

	// entries is the provider list returned for the resource type.
	entries []tui_domain.ProviderEntry
}

// DescribeProvider fetches the per-provider sections + sub-resources.
//
// Takes ctx (context.Context) which controls the call lifetime.
// Takes resourceType (string) which selects the provider resource type.
// Takes name (string) which identifies the provider within that type.
//
// Returns *tui_domain.ProviderDetail which is the populated detail.
// Returns error when the RPC fails.
func (p *ProvidersInspector) DescribeProvider(ctx context.Context, resourceType, name string) (*tui_domain.ProviderDetail, error) {
	response, err := p.conn.providerInfoClient.DescribeProvider(ctx, &pb.DescribeProviderRequest{
		ResourceType: resourceType,
		Name:         name,
	})
	if err != nil {
		return nil, fmt.Errorf("describe provider %s/%s: %w", resourceType, name, translateRPCError(err))
	}

	sections := make([]tui_domain.ProviderSection, 0, len(response.GetSections()))
	for _, s := range response.GetSections() {
		entries := make([]tui_domain.ProviderField, 0, len(s.GetEntries()))
		for _, e := range s.GetEntries() {
			entries = append(entries, tui_domain.ProviderField{Key: e.GetKey(), Value: e.GetValue()})
		}
		sections = append(sections, tui_domain.ProviderSection{Title: s.GetTitle(), Entries: entries})
	}

	subResp, subErr := p.conn.providerInfoClient.ListSubResources(ctx, &pb.ListSubResourcesRequest{
		ResourceType: resourceType,
		ProviderName: name,
	})
	subs := []tui_domain.ProviderSubResource{}
	if subErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Debug("Failed to list sub-resources for provider",
			logger.String("resource_type", resourceType),
			logger.String("name", name),
			logger.Error(subErr))
	} else {
		subType := subResp.GetSubResourceName()
		for _, r := range subResp.GetRows() {
			subs = append(subs, tui_domain.ProviderSubResource{
				Type:   subType,
				Name:   r.GetName(),
				Values: r.GetValues(),
			})
		}
	}

	return &tui_domain.ProviderDetail{
		ResourceType: resourceType,
		Name:         name,
		Sections:     sections,
		SubResources: subs,
	}, nil
}

// listOneType issues ListProviders for a single resource type with
// per-call timeout and panic recovery so a misbehaving server cannot
// take down the parent fan-out goroutine.
//
// Takes ctx (context.Context) which is the caller's context; a child
// context is derived per call so individual slow types cannot stall
// the whole list.
// Takes rt (string) which is the resource type to query.
//
// Returns listProvidersResult with either entries or err populated.
func (p *ProvidersInspector) listOneType(ctx context.Context, rt string) (result listProvidersResult) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = listProvidersResult{err: fmt.Errorf("list providers for %s: %v", rt, recovered)}
		}
	}()
	callCtx, cancel := context.WithTimeoutCause(ctx, listProvidersPerCallTimeout,
		fmt.Errorf("list providers for %s exceeded %s", rt, listProvidersPerCallTimeout))
	defer cancel()
	listResp, listErr := p.conn.providerInfoClient.ListProviders(callCtx, &pb.ListProvidersRequest{
		ResourceType: rt,
	})
	if listErr != nil {
		return listProvidersResult{err: fmt.Errorf("list providers for %s: %w", rt, translateRPCError(listErr))}
	}
	rows := listResp.GetRows()
	entries := make([]tui_domain.ProviderEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, tui_domain.ProviderEntry{
			ResourceType: rt,
			Name:         row.GetName(),
			IsDefault:    row.GetIsDefault(),
			Values:       row.GetValues(),
		})
	}
	return listProvidersResult{entries: entries}
}

// DLQInspector adapts the gRPC DispatcherInspector service to the
// tui_domain.DLQInspector port.
type DLQInspector struct {
	// conn is the shared gRPC connection used for all RPCs.
	conn *Connection
}

// NewDLQInspector constructs a DLQInspector backed by conn.
//
// Takes conn (*Connection) which is the shared gRPC connection.
//
// Returns *DLQInspector ready to expose via tui_domain.DLQInspector.
func NewDLQInspector(conn *Connection) *DLQInspector {
	return &DLQInspector{conn: conn}
}

// Name reports the provider identifier.
//
// Returns string which is the inspector identifier.
func (*DLQInspector) Name() string { return "grpc-dlq" }

// Health checks the underlying gRPC connection.
//
// Takes ctx (context.Context) which controls the call lifetime.
//
// Returns error when the health probe fails.
func (p *DLQInspector) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking DLQ inspector health: %w", err)
	}
	return nil
}

// Close is a no-op; the connection is owned by the parent.
//
// Returns error which is always nil.
func (*DLQInspector) Close() error { return nil }

// DispatcherSummaries lists every dispatcher's headline counters.
//
// Returns []tui_domain.DispatcherSummary which is the summary list.
// Returns error when the RPC fails.
func (p *DLQInspector) DispatcherSummaries(ctx context.Context) ([]tui_domain.DispatcherSummary, error) {
	response, err := p.conn.dispatcherClient.GetDispatcherSummary(ctx, &pb.GetDispatcherSummaryRequest{})
	if err != nil {
		return nil, fmt.Errorf("dispatcher summary: %w", translateRPCError(err))
	}
	out := make([]tui_domain.DispatcherSummary, 0, len(response.GetSummaries()))
	for _, s := range response.GetSummaries() {
		out = append(out, tui_domain.DispatcherSummary{
			Type:            s.GetType(),
			QueuedItems:     s.GetQueuedItems(),
			DeadLetterCount: s.GetDeadLetterCount(),
			TotalProcessed:  s.GetTotalProcessed(),
			TotalSuccessful: s.GetTotalSuccessful(),
			TotalFailed:     s.GetTotalFailed(),
			RetryQueueSize:  s.GetRetryQueueSize(),
			TotalRetries:    s.GetTotalRetries(),
			Uptime:          time.Duration(s.GetUptimeMs()) * time.Millisecond, //nolint:gosec // server-supplied counter
		})
	}
	return out, nil
}

// ListDLQEntries returns a bounded slice of dead-lettered items. The
// limit is clamped to dlqLimitMax both to defend against the int32
// wire field overflowing and to bound the returned slice when the
// server returns more rows than requested.
//
// Takes ctx (context.Context) which controls the call lifetime.
// Takes dispatcherType (string) which selects the dispatcher.
// Takes limit (int) which caps the requested row count.
//
// Returns []tui_domain.DLQEntry which is the bounded entry list.
// Returns error when the RPC fails.
func (p *DLQInspector) ListDLQEntries(ctx context.Context, dispatcherType string, limit int) ([]tui_domain.DLQEntry, error) {
	if limit <= 0 {
		limit = dlqLimitMax
	}
	if limit > dlqLimitMax {
		limit = dlqLimitMax
	}
	response, err := p.conn.dispatcherClient.ListDLQEntries(ctx, &pb.ListDLQEntriesRequest{
		DispatcherType: dispatcherType,
		Limit:          safeconv.IntToInt32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list DLQ entries for %s: %w", dispatcherType, translateRPCError(err))
	}
	rows := response.GetEntries()
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]tui_domain.DLQEntry, 0, len(rows))
	for _, e := range rows {
		out = append(out, tui_domain.DLQEntry{
			ID:            e.GetId(),
			Type:          e.GetType(),
			OriginalError: e.GetOriginalError(),
			TotalAttempts: e.GetTotalAttempts(),
			AddedAt:       msToTime(e.GetAddedAtMs()),
			LastAttempt:   msToTime(e.GetLastAttemptMs()),
		})
	}
	return out, nil
}

// RateLimiterInspector adapts the gRPC RateLimiterInspector service to
// the tui_domain.RateLimiterInspector port.
type RateLimiterInspector struct {
	// conn is the shared gRPC connection used for all RPCs.
	conn *Connection
}

// NewRateLimiterInspector constructs a RateLimiterInspector backed by
// conn.
//
// Takes conn (*Connection) which is the shared gRPC connection.
//
// Returns *RateLimiterInspector ready to expose via the port.
func NewRateLimiterInspector(conn *Connection) *RateLimiterInspector {
	return &RateLimiterInspector{conn: conn}
}

// Name reports the provider identifier.
//
// Returns string which is the inspector identifier.
func (*RateLimiterInspector) Name() string { return "grpc-ratelimiter" }

// Health checks the underlying gRPC connection.
//
// Takes ctx (context.Context) which controls the call lifetime.
//
// Returns error when the health probe fails.
func (p *RateLimiterInspector) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking rate-limiter inspector health: %w", err)
	}
	return nil
}

// Close is a no-op; the connection is owned by the parent.
//
// Returns error which is always nil.
func (*RateLimiterInspector) Close() error { return nil }

// GetStatus fetches the current rate-limiter status.
//
// Returns *tui_domain.RateLimiterStatus which is the populated status.
// Returns error when the RPC fails.
func (p *RateLimiterInspector) GetStatus(ctx context.Context) (*tui_domain.RateLimiterStatus, error) {
	response, err := p.conn.rateLimiterClient.GetRateLimiterStatus(ctx, &pb.GetRateLimiterStatusRequest{})
	if err != nil {
		return nil, fmt.Errorf("rate limiter status: %w", translateRPCError(err))
	}
	return &tui_domain.RateLimiterStatus{
		TokenBucketStore: response.GetTokenBucketStore(),
		CounterStore:     response.GetCounterStore(),
		FailPolicy:       response.GetFailPolicy(),
		KeyPrefix:        response.GetKeyPrefix(),
		TotalChecks:      response.GetTotalChecks(),
		TotalAllowed:     response.GetTotalAllowed(),
		TotalDenied:      response.GetTotalDenied(),
		TotalErrors:      response.GetTotalErrors(),
	}, nil
}

// ProfilingInspector adapts the gRPC ProfilingService to the
// tui_domain.ProfilingInspector port.
type ProfilingInspector struct {
	// conn is the shared gRPC connection used for all RPCs.
	conn *Connection
}

// NewProfilingInspector constructs a ProfilingInspector backed by conn.
//
// Takes conn (*Connection) which is the shared gRPC connection.
//
// Returns *ProfilingInspector ready to expose via the port.
func NewProfilingInspector(conn *Connection) *ProfilingInspector {
	return &ProfilingInspector{conn: conn}
}

// Name reports the provider identifier.
//
// Returns string which is the inspector identifier.
func (*ProfilingInspector) Name() string { return "grpc-profiling" }

// Health checks the underlying gRPC connection.
//
// Takes ctx (context.Context) which controls the call lifetime.
//
// Returns error when the health probe fails.
func (p *ProfilingInspector) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking profiling inspector health: %w", err)
	}
	return nil
}

// Close is a no-op; the connection is owned by the parent.
//
// Returns error which is always nil.
func (*ProfilingInspector) Close() error { return nil }

// Status returns the current profiling state.
//
// Returns *tui_domain.ProfilingStatus which is the populated state.
// Returns error when the RPC fails.
func (p *ProfilingInspector) Status(ctx context.Context) (*tui_domain.ProfilingStatus, error) {
	response, err := p.conn.profilingClient.GetProfilingStatus(ctx, &pb.GetProfilingStatusRequest{})
	if err != nil {
		return nil, fmt.Errorf("profiling status: %w", translateRPCError(err))
	}
	return &tui_domain.ProfilingStatus{
		Enabled:              response.GetEnabled(),
		AvailableProfiles:    response.GetAvailableProfiles(),
		ExpiresAt:            msToTime(response.GetExpiresAtMs()),
		Remaining:            time.Duration(response.GetRemainingMs()) * time.Millisecond, //nolint:gosec // server-supplied
		PprofBaseURL:         response.GetPprofBaseUrl(),
		BlockProfileRate:     response.GetBlockProfileRate(),
		MutexProfileFraction: response.GetMutexProfileFraction(),
		MemProfileRate:       response.GetMemProfileRate(),
		Port:                 response.GetPort(),
	}, nil
}

// Enable turns on the server's on-demand profiling for a fixed
// window. The window is intentionally short (the panel can re-enable
// as the user navigates around) so a stuck-on profile never silently
// consumes resources on the server.
//
// Returns error when the RPC fails.
func (p *ProfilingInspector) Enable(ctx context.Context) error {
	_, err := p.conn.profilingClient.EnableProfiling(ctx, &pb.EnableProfilingRequest{
		DurationMs: int64(captureDeadlineSlack / time.Millisecond),
	})
	if err != nil {
		return fmt.Errorf("enable profiling: %w", translateRPCError(err))
	}
	return nil
}

// Disable turns off all profiles.
//
// Returns error when the RPC fails.
func (p *ProfilingInspector) Disable(ctx context.Context) error {
	_, err := p.conn.profilingClient.DisableProfiling(ctx, &pb.DisableProfilingRequest{})
	if err != nil {
		return fmt.Errorf("disable profiling: %w", translateRPCError(err))
	}
	return nil
}

// Capture streams the chunks for a one-shot profile and returns the
// concatenated bytes.
//
// The context deadline always covers the full sampling window plus
// encoding overhead. Total bytes are capped at captureMaxBytes to
// defend against runaway servers; per-chunk bytes are capped at
// captureChunkMaxBytes for the same reason.
//
// Takes profile (string) which names the profile kind.
// Takes duration (time.Duration) which is the sampling window.
//
// Returns []byte which is the assembled pprof payload.
// Returns error when the RPC fails or a cap is exceeded.
func (p *ProfilingInspector) Capture(ctx context.Context, profile string, duration time.Duration) ([]byte, error) {
	durationSeconds, err := captureDurationToSeconds(duration)
	if err != nil {
		return nil, fmt.Errorf("capture %s profile: %w", profile, err)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, duration+captureDeadlineSlack,
		fmt.Errorf("%s profile capture exceeded budget", profile))
	defer cancel()

	stream, err := p.conn.profilingClient.CaptureProfile(ctx, &pb.CaptureProfileRequest{
		ProfileType:     profile,
		DurationSeconds: durationSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("capture %s profile: %w", profile, translateRPCError(err))
	}

	return drainCaptureStream(stream, profile)
}

// drainCaptureStream consumes the chunks streamed by CaptureProfile,
// concatenating them into a single buffer bounded by captureMaxBytes.
// The server may attach the last chunk to the same Recv() that
// returns its terminating error; that case is handled by appending
// the trailing chunk before propagating the error.
//
// Takes stream (pb.ProfilingService_CaptureProfileClient) which is
// the open server stream.
// Takes profile (string) which names the profile kind for error
// messages.
//
// Returns []byte which is the assembled pprof payload.
// Returns error when the stream fails or the cap is exceeded.
func drainCaptureStream(stream pb.ProfilingService_CaptureProfileClient, profile string) ([]byte, error) {
	buf := make([]byte, 0, captureChunkMaxBytes)
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			return buf, nil
		}
		if recvErr != nil {
			return handleCaptureRecvError(buf, chunk, profile, recvErr)
		}
		appended, appendErr := appendCaptureChunk(buf, chunk.GetData(), profile)
		if appendErr != nil {
			return nil, appendErr
		}
		buf = appended
		if chunk.GetIsLast() {
			return buf, nil
		}
	}
}

// handleCaptureRecvError handles the recvErr returned by Recv. When
// the server attached the last chunk to the same response that
// terminated the stream, appendCaptureChunk is given a final chance
// to fold it in before the error is propagated.
//
// Takes buf ([]byte) which is the accumulator so far.
// Takes chunk (*pb.CaptureProfileChunk) which may be nil; when
// present its IsLast flag indicates a server-side trailer.
// Takes profile (string) which names the profile for error messages.
// Takes recvErr (error) which is the error Recv returned.
//
// Returns ([]byte, error) the final buffer (when the trailer was
// folded in cleanly) or a wrapped error.
func handleCaptureRecvError(buf []byte, chunk *pb.CaptureProfileChunk, profile string, recvErr error) ([]byte, error) {
	if chunk == nil || !chunk.GetIsLast() {
		return nil, fmt.Errorf("read %s profile chunk: %w", profile, recvErr)
	}
	appended, appendErr := appendCaptureChunk(buf, chunk.GetData(), profile)
	if appendErr != nil {
		return nil, appendErr
	}
	return appended, nil
}

// captureDurationToSeconds validates a Capture duration argument and
// converts it to int32 seconds for the wire format. A duration of 0
// or larger than captureMaxDurationSeconds is rejected so the wire
// field cannot overflow.
//
// Takes duration (time.Duration) which is the caller-requested
// sampling window.
//
// Returns int32 with the seconds value clamped to [1, max].
// Returns error wrapping ErrCaptureDurationOutOfRange when the
// argument is out of range.
func captureDurationToSeconds(duration time.Duration) (int32, error) {
	if duration <= 0 {
		return 0, fmt.Errorf("%w: must be positive, got %s", ErrCaptureDurationOutOfRange, duration)
	}
	seconds := max(int64(duration/time.Second), 1)
	if seconds > captureMaxDurationSeconds {
		return 0, fmt.Errorf("%w: %s exceeds %ds cap", ErrCaptureDurationOutOfRange, duration, captureMaxDurationSeconds)
	}
	if seconds > math.MaxInt32 {
		return 0, fmt.Errorf("%w: %s does not fit in int32", ErrCaptureDurationOutOfRange, duration)
	}
	return safeconv.Int64ToInt32(seconds), nil
}

// appendCaptureChunk validates a single stream chunk against the
// per-chunk and total caps then appends it to buf. Returns a wrapped
// ErrCaptureTooLarge / ErrCaptureChunkTooLarge when either cap is
// exceeded so callers can surface the specific reason.
//
// Takes buf ([]byte) which is the accumulator so far.
// Takes data ([]byte) which is the chunk payload.
// Takes profile (string) which names the profile kind for context.
//
// Returns []byte which is the (possibly extended) accumulator.
// Returns error which is non-nil when a cap would be exceeded.
func appendCaptureChunk(buf, data []byte, profile string) ([]byte, error) {
	if len(data) > captureChunkMaxBytes {
		return buf, fmt.Errorf("%w: %s chunk is %d bytes (max %d)",
			ErrCaptureChunkTooLarge, profile, len(data), captureChunkMaxBytes)
	}
	if len(buf)+len(data) > captureMaxBytes {
		return buf, fmt.Errorf("%w: %s capture would exceed %d bytes",
			ErrCaptureTooLarge, profile, captureMaxBytes)
	}
	return append(buf, data...), nil
}

// msToTime converts an int64 millisecond timestamp into a time.Time;
// zero milliseconds map to the zero time so callers can detect "no
// value" without a sentinel.
//
// Takes ms (int64) which is the millisecond timestamp.
//
// Returns time.Time which is the converted time, or the zero value
// when ms is zero.
func msToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
