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

package monitoring_transport_grpc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// downloadChunkSize is the maximum size of each streaming chunk when
	// delivering stored watchdog profile data.
	downloadChunkSize = 32 * 1024

	// maxDownloadBytes is the maximum total profile size delivered via gRPC.
	maxDownloadBytes = 64 * 1024 * 1024

	// maxSidecarBytes is the maximum sidecar JSON payload returned in a
	// single gRPC response.
	maxSidecarBytes = 1 * 1024 * 1024
)

// WatchdogInspectorService implements the gRPC watchdog inspector service
// interface, providing remote access to watchdog state and stored profiles.
type WatchdogInspectorService struct {
	pb.UnimplementedWatchdogInspectorServiceServer

	// inspector provides read-only access to watchdog state and stored
	// profiles.
	inspector monitoring_domain.WatchdogInspector
}

// NewWatchdogInspectorService creates a new WatchdogInspectorService.
//
// Takes inspector (monitoring_domain.WatchdogInspector) which provides
// read-only access to watchdog state and stored profiles.
//
// Returns *WatchdogInspectorService ready for gRPC registration.
func NewWatchdogInspectorService(inspector monitoring_domain.WatchdogInspector) *WatchdogInspectorService {
	return &WatchdogInspectorService{inspector: inspector}
}

// ListProfiles returns metadata for all stored watchdog profile files.
//
// Returns *pb.ListProfilesResponse which contains the profile entries.
// Returns error when the profile directory cannot be read; the error is
// converted to an appropriate gRPC status code via toGRPCError.
func (s *WatchdogInspectorService) ListProfiles(ctx context.Context, _ *pb.ListProfilesRequest) (*pb.ListProfilesResponse, error) {
	profiles, err := s.inspector.ListProfiles(ctx)
	if err != nil {
		return nil, toGRPCError(fmt.Errorf("listing watchdog profiles: %w", err))
	}

	entries := make([]*pb.WatchdogProfileEntry, len(profiles))
	for index, profile := range profiles {
		entries[index] = &pb.WatchdogProfileEntry{
			Filename:    profile.Filename,
			Type:        profile.Type,
			TimestampMs: profile.Timestamp.UnixMilli(),
			SizeBytes:   profile.SizeBytes,
			HasSidecar:  profile.HasSidecar,
		}
	}

	return &pb.ListProfilesResponse{Profiles: entries}, nil
}

// DownloadSidecar returns the JSON sidecar paired with a profile.
//
// Takes request (*pb.DownloadSidecarRequest) which specifies the profile
// filename whose sidecar to fetch.
//
// Returns *pb.DownloadSidecarResponse which contains the sidecar bytes
// and a presence flag.
// Returns error when the request is malformed or the read fails for
// reasons other than absence.
func (s *WatchdogInspectorService) DownloadSidecar(ctx context.Context, request *pb.DownloadSidecarRequest) (*pb.DownloadSidecarResponse, error) {
	filename := request.GetProfileFilename()
	if filename == "" {
		return nil, status.Error(codes.InvalidArgument, "profile_filename must not be empty")
	}

	data, present, err := s.inspector.DownloadSidecar(ctx, filename)
	if err != nil {
		return nil, toGRPCError(fmt.Errorf("downloading sidecar for %q: %w", filename, err))
	}
	if int64(len(data)) > maxSidecarBytes {
		return nil, status.Errorf(codes.ResourceExhausted, "sidecar for %q exceeds %d byte limit", filename, maxSidecarBytes)
	}

	return &pb.DownloadSidecarResponse{
		Data:    data,
		Present: present,
	}, nil
}

// DownloadProfile streams the raw bytes of a stored watchdog profile file
// back to the client in fixed-size chunks.
//
// Takes request (*pb.DownloadProfileRequest) which specifies the filename to
// download.
// Takes stream (pb.WatchdogInspectorService_DownloadProfileServer) which
// receives the chunked profile data.
//
// Returns error when the filename is empty, the profile cannot be read, or
// streaming fails.
func (s *WatchdogInspectorService) DownloadProfile(request *pb.DownloadProfileRequest, stream pb.WatchdogInspectorService_DownloadProfileServer) error {
	filename := request.GetFilename()
	if filename == "" {
		return status.Error(codes.InvalidArgument, "filename must not be empty")
	}

	downloadBuffer := newLimitedBuffer(maxDownloadBytes)

	err := s.inspector.DownloadProfile(stream.Context(), filename, downloadBuffer)
	if err != nil {
		return toGRPCError(fmt.Errorf("downloading profile %q: %w", filename, err))
	}

	if sendErr := sendDownloadChunks(stream, downloadBuffer.Bytes()); sendErr != nil {
		return toGRPCError(sendErr)
	}
	return nil
}

// PruneProfiles removes stored watchdog profile files and returns the count
// of files deleted.
//
// Takes request (*pb.PruneProfilesRequest) which specifies the optional
// profile type filter. When empty, all profiles are removed.
//
// Returns *pb.PruneProfilesResponse which contains the number of deleted files.
// Returns error when listing or removing files fails.
func (s *WatchdogInspectorService) PruneProfiles(ctx context.Context, request *pb.PruneProfilesRequest) (*pb.PruneProfilesResponse, error) {
	deletedCount, err := s.inspector.PruneProfiles(ctx, request.GetProfileType())
	if err != nil {
		return nil, toGRPCError(fmt.Errorf("pruning watchdog profiles: %w", err))
	}

	return &pb.PruneProfilesResponse{
		DeletedCount: safeconv.IntToInt32(deletedCount),
	}, nil
}

// GetWatchdogStatus returns the current watchdog state including
// configuration, thresholds, and runtime counters.
//
// Returns *pb.GetWatchdogStatusResponse which contains the current watchdog
// state.
// Returns error (always nil; included for interface compliance).
func (s *WatchdogInspectorService) GetWatchdogStatus(ctx context.Context, _ *pb.GetWatchdogStatusRequest) (*pb.GetWatchdogStatusResponse, error) {
	watchdogStatus := s.inspector.GetWatchdogStatus(ctx)

	contentionLastRunMs := int64(0)
	if !watchdogStatus.ContentionDiagnosticLastRun.IsZero() {
		contentionLastRunMs = watchdogStatus.ContentionDiagnosticLastRun.UnixMilli()
	}

	return &pb.GetWatchdogStatusResponse{
		Enabled:                        watchdogStatus.Enabled,
		Stopped:                        watchdogStatus.Stopped,
		ProfileDirectory:               watchdogStatus.ProfileDirectory,
		CheckIntervalMs:                watchdogStatus.CheckInterval.Milliseconds(),
		CooldownMs:                     watchdogStatus.Cooldown.Milliseconds(),
		WarmUpDurationMs:               watchdogStatus.WarmUpDuration.Milliseconds(),
		StartedAtMs:                    watchdogStatus.StartedAt.UnixMilli(),
		HeapThresholdBytes:             watchdogStatus.HeapThresholdBytes,
		HeapHighWater:                  watchdogStatus.HeapHighWater,
		GoroutineThreshold:             safeconv.IntToInt32(watchdogStatus.GoroutineThreshold),
		GoroutineSafetyCeiling:         safeconv.IntToInt32(watchdogStatus.GoroutineSafetyCeiling),
		MaxProfilesPerType:             safeconv.IntToInt32(watchdogStatus.MaxProfilesPerType),
		CaptureWindowMs:                watchdogStatus.CaptureWindow.Milliseconds(),
		MaxCapturesPerWindow:           safeconv.IntToInt32(watchdogStatus.MaxCapturesPerWindow),
		MaxWarningsPerWindow:           safeconv.IntToInt32(watchdogStatus.MaxWarningsPerWindow),
		FdPressureThresholdPercent:     watchdogStatus.FDPressureThresholdPercent,
		SchedulerLatencyP99ThresholdNs: watchdogStatus.SchedulerLatencyP99Threshold.Nanoseconds(),
		CrashLoopWindowMs:              watchdogStatus.CrashLoopWindow.Milliseconds(),
		CrashLoopThreshold:             safeconv.IntToInt32(watchdogStatus.CrashLoopThreshold),
		ContinuousProfilingEnabled:     watchdogStatus.ContinuousProfilingEnabled,
		ContinuousProfilingIntervalMs:  watchdogStatus.ContinuousProfilingInterval.Milliseconds(),
		ContinuousProfilingTypes:       watchdogStatus.ContinuousProfilingTypes,
		ContinuousProfilingRetention:   safeconv.IntToInt32(watchdogStatus.ContinuousProfilingRetention),
		ContentionDiagnosticWindowMs:   watchdogStatus.ContentionDiagnosticWindow.Milliseconds(),
		ContentionDiagnosticCooldownMs: watchdogStatus.ContentionDiagnosticCooldown.Milliseconds(),
		ContentionDiagnosticAutoFire:   watchdogStatus.ContentionDiagnosticAutoFire,
		ContentionDiagnosticLastRunMs:  contentionLastRunMs,
		GoroutineBaseline:              watchdogStatus.GoroutineBaseline,
		CaptureWindowUsed:              safeconv.IntToInt32(watchdogStatus.CaptureWindowUsed),
		WarningWindowUsed:              safeconv.IntToInt32(watchdogStatus.WarningWindowUsed),
	}, nil
}

// RunContentionDiagnostic enables block + mutex profiling for the configured
// window, captures both profiles, then disables. The call is synchronous so
// the gRPC response is returned only after the diagnostic completes.
//
// Returns *pb.RunContentionDiagnosticResponse with started=true on success.
// Returns error wrapped as a gRPC status when the diagnostic cannot run.
func (s *WatchdogInspectorService) RunContentionDiagnostic(ctx context.Context, _ *pb.RunContentionDiagnosticRequest) (*pb.RunContentionDiagnosticResponse, error) {
	if err := s.inspector.RunContentionDiagnostic(ctx); err != nil {
		return &pb.RunContentionDiagnosticResponse{
			Started: false,
			Error:   err.Error(),
		}, nil
	}
	return &pb.RunContentionDiagnosticResponse{Started: true}, nil
}

// GetStartupHistory returns the parsed startup-history ring.
//
// Returns *pb.GetStartupHistoryResponse which contains each entry's start
// and stop timestamps as unix milliseconds.
// Returns error when the history file is unreadable or corrupt.
func (s *WatchdogInspectorService) GetStartupHistory(ctx context.Context, _ *pb.GetStartupHistoryRequest) (*pb.GetStartupHistoryResponse, error) {
	entries, err := s.inspector.GetStartupHistory(ctx)
	if err != nil {
		return nil, toGRPCError(fmt.Errorf("reading startup history: %w", err))
	}

	pbEntries := make([]*pb.StartupHistoryEntry, len(entries))
	for index, entry := range entries {
		stoppedAtMs := int64(0)
		if !entry.StoppedAt.IsZero() {
			stoppedAtMs = entry.StoppedAt.UnixMilli()
		}
		pbEntries[index] = &pb.StartupHistoryEntry{
			StartedAtMs:     entry.StartedAt.UnixMilli(),
			StoppedAtMs:     stoppedAtMs,
			Pid:             safeconv.IntToInt32(entry.PID),
			Hostname:        entry.Hostname,
			Version:         entry.Version,
			GomemlimitBytes: entry.GomemlimitBytes,
			StopReason:      entry.Reason,
		}
	}

	return &pb.GetStartupHistoryResponse{Entries: pbEntries}, nil
}

// ListEvents returns recent watchdog events from the in-memory ring.
//
// Takes request (*pb.ListEventsRequest) which filters by limit, since,
// and event type.
//
// Returns *pb.ListEventsResponse which contains the matching events in
// chronological order.
// Returns error (always nil; included for interface compliance).
func (s *WatchdogInspectorService) ListEvents(ctx context.Context, request *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	since := time.Time{}
	if ms := request.GetSinceMs(); ms > 0 {
		since = time.UnixMilli(ms)
	}

	events := s.inspector.ListEvents(ctx, int(request.GetLimit()), since, request.GetEventType())

	pbEvents := make([]*pb.WatchdogEventMessage, len(events))
	for index, event := range events {
		pbEvents[index] = watchdogEventToProto(event)
	}

	return &pb.ListEventsResponse{Events: pbEvents}, nil
}

// WatchEvents streams newly emitted watchdog events to the client.
// Optionally back-fills from the ring buffer before live streaming begins.
//
// Takes request (*pb.WatchEventsRequest) which carries the optional
// since-millis back-fill watermark.
// Takes stream (pb.WatchdogInspectorService_WatchEventsServer) which
// receives each event message.
//
// Returns error when the subscription cannot be created or a send fails.
func (s *WatchdogInspectorService) WatchEvents(request *pb.WatchEventsRequest, stream pb.WatchdogInspectorService_WatchEventsServer) error {
	since := time.Time{}
	if ms := request.GetSinceMs(); ms > 0 {
		since = time.UnixMilli(ms)
	}

	ch, cancel := s.inspector.SubscribeEvents(stream.Context(), since)
	defer cancel()

	for event := range ch {
		if err := stream.Send(watchdogEventToProto(event)); err != nil {
			return toGRPCError(fmt.Errorf("sending watchdog event: %w", err))
		}
	}
	if ctxErr := stream.Context().Err(); ctxErr != nil {
		return toGRPCError(ctxErr)
	}
	return nil
}

// watchdogEventToProto maps a domain WatchdogEventInfo to its protobuf form.
//
// Takes event (monitoring_domain.WatchdogEventInfo) which is the domain
// event to project onto the wire format.
//
// Returns *pb.WatchdogEventMessage which captures every field of the
// inspector-facing event.
func watchdogEventToProto(event monitoring_domain.WatchdogEventInfo) *pb.WatchdogEventMessage {
	return &pb.WatchdogEventMessage{
		EventType:   string(event.EventType),
		Priority:    safeconv.IntToInt32(int(event.Priority)),
		Message:     event.Message,
		Fields:      event.Fields,
		EmittedAtMs: event.EmittedAt.UnixMilli(),
	}
}

// toGRPCError translates a domain or transport error into a gRPC status
// error with an appropriate code. Errors that are already gRPC status
// errors are returned unchanged so the original code is preserved.
//
// Mapping summary:
//   - context.Canceled            -> codes.Canceled
//   - context.DeadlineExceeded    -> codes.DeadlineExceeded
//   - monitoring_domain.ErrWatchdogStopped
//   - monitoring_domain.ErrEventSubscriberCapExceeded
//   - monitoring_domain.ErrProfilingControllerNil  -> codes.Unavailable
//   - fs.ErrNotExist              -> codes.NotFound
//   - default                     -> codes.Internal
//
// Takes err (error) which is the originating failure to translate.
//
// Returns error which is the gRPC status-coded error, or nil when err is nil.
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, monitoring_domain.ErrWatchdogStopped):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, monitoring_domain.ErrEventSubscriberCapExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, monitoring_domain.ErrProfilingControllerNil):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, fs.ErrNotExist):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// sendDownloadChunks writes profile data to the stream in fixed-size chunks.
//
// Takes stream (pb.WatchdogInspectorService_DownloadProfileServer) which
// receives each chunk.
// Takes data ([]byte) which is the complete profile payload.
//
// Returns error when the stream context is cancelled or a send fails.
func sendDownloadChunks(stream pb.WatchdogInspectorService_DownloadProfileServer, data []byte) error {
	if len(data) == 0 {
		if err := stream.Send(&pb.DownloadProfileChunk{
			IsLast: true,
		}); err != nil {
			return fmt.Errorf("sending empty download response: %w", err)
		}
		return nil
	}

	for offset := 0; offset < len(data); offset += downloadChunkSize {
		if err := stream.Context().Err(); err != nil {
			return err
		}

		end := min(offset+downloadChunkSize, len(data))
		isLast := end >= len(data)

		chunk := &pb.DownloadProfileChunk{
			Data:   data[offset:end],
			IsLast: isLast,
		}

		if err := stream.Send(chunk); err != nil {
			return fmt.Errorf("sending download chunk: %w", err)
		}
	}

	return nil
}
