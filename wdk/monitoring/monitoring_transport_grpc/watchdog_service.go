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
	"fmt"

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
// Returns error when the profile directory cannot be read.
func (s *WatchdogInspectorService) ListProfiles(ctx context.Context, _ *pb.ListProfilesRequest) (*pb.ListProfilesResponse, error) {
	profiles, err := s.inspector.ListProfiles(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "listing watchdog profiles: %v", err)
	}

	entries := make([]*pb.WatchdogProfileEntry, len(profiles))
	for index, profile := range profiles {
		entries[index] = &pb.WatchdogProfileEntry{
			Filename:    profile.Filename,
			Type:        profile.Type,
			TimestampMs: profile.Timestamp.UnixMilli(),
			SizeBytes:   profile.SizeBytes,
		}
	}

	return &pb.ListProfilesResponse{Profiles: entries}, nil
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
		return status.Errorf(codes.InvalidArgument, "filename must not be empty")
	}

	downloadBuffer := newLimitedBuffer(maxDownloadBytes)

	err := s.inspector.DownloadProfile(stream.Context(), filename, downloadBuffer)
	if err != nil {
		return status.Errorf(codes.NotFound, "downloading profile %q: %v", filename, err)
	}

	return sendDownloadChunks(stream, downloadBuffer.Bytes())
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
		return nil, status.Errorf(codes.Internal, "pruning watchdog profiles: %v", err)
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

	return &pb.GetWatchdogStatusResponse{
		Enabled:                watchdogStatus.Enabled,
		Stopped:                watchdogStatus.Stopped,
		ProfileDirectory:       watchdogStatus.ProfileDirectory,
		CheckIntervalMs:        watchdogStatus.CheckInterval.Milliseconds(),
		CooldownMs:             watchdogStatus.Cooldown.Milliseconds(),
		WarmUpDurationMs:       watchdogStatus.WarmUpDuration.Milliseconds(),
		StartedAtMs:            watchdogStatus.StartedAt.UnixMilli(),
		HeapThresholdBytes:     watchdogStatus.HeapThresholdBytes,
		HeapHighWater:          watchdogStatus.HeapHighWater,
		GoroutineThreshold:     safeconv.IntToInt32(watchdogStatus.GoroutineThreshold),
		GoroutineSafetyCeiling: safeconv.IntToInt32(watchdogStatus.GoroutineSafetyCeiling),
		MaxProfilesPerType:     safeconv.IntToInt32(watchdogStatus.MaxProfilesPerType),
	}, nil
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
