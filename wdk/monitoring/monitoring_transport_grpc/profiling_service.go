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
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// captureChunkSize is the maximum size of each streaming chunk when
	// delivering captured profile data.
	captureChunkSize = 32 * 1024

	// maxCaptureBytes is the maximum total profile size delivered via gRPC.
	// Larger profiles should use the pprof HTTP server directly.
	maxCaptureBytes = 64 * 1024 * 1024

	// maxTCPPort is the highest valid TCP port number. Used to validate
	// caller-supplied profiling port values.
	maxTCPPort = 65535
)

// errCaptureLimitExceeded is returned when a profile capture exceeds
// maxCaptureBytes during writing.
var errCaptureLimitExceeded = errors.New("profile capture exceeded size limit")

// limitedBuffer wraps bytes.Buffer with a maximum size enforced during
// writing. Once the accumulated data exceeds the limit, all subsequent
// writes return errCaptureLimitExceeded immediately, preventing the full
// profile from being buffered in memory before the size check.
type limitedBuffer struct {
	// buffer holds the accumulated profile data.
	buffer bytes.Buffer

	// limit is the maximum number of bytes the buffer may hold.
	limit int

	// exceeded is set once a write would breach the limit, causing all
	// subsequent writes to fail immediately.
	exceeded bool
}

// newLimitedBuffer creates a limitedBuffer that rejects writes once the
// total written bytes exceed limit.
//
// Takes limit (int) which is the maximum byte count before writes are rejected.
//
// Returns *limitedBuffer ready for use as an io.Writer.
func newLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{limit: limit}
}

// Write appends data to the buffer, returning errCaptureLimitExceeded if
// the write would cause the buffer to exceed the configured limit.
//
// Takes data ([]byte) which is the bytes to append.
//
// Returns int which is the number of bytes written.
// Returns error when the limit has been or would be exceeded.
func (limitedWriter *limitedBuffer) Write(data []byte) (int, error) {
	if limitedWriter.exceeded {
		return 0, errCaptureLimitExceeded
	}

	if limitedWriter.buffer.Len()+len(data) > limitedWriter.limit {
		limitedWriter.exceeded = true

		return 0, errCaptureLimitExceeded
	}

	return limitedWriter.buffer.Write(data)
}

// Bytes returns the buffered data.
//
// Returns []byte which is the accumulated profile bytes.
func (limitedWriter *limitedBuffer) Bytes() []byte {
	return limitedWriter.buffer.Bytes()
}

// Len returns the number of bytes currently buffered.
//
// Returns int which is the current buffer length.
func (limitedWriter *limitedBuffer) Len() int {
	return limitedWriter.buffer.Len()
}

// ProfilingService implements the gRPC profiling service interface.
type ProfilingService struct {
	pb.UnimplementedProfilingServiceServer

	// controller manages the profiling lifecycle (enable, disable, capture).
	controller monitoring_domain.ProfilingController
}

// NewProfilingService creates a new ProfilingService.
//
// Takes controller (monitoring_domain.ProfilingController) which manages
// the profiling lifecycle.
//
// Returns *ProfilingService ready for gRPC registration.
func NewProfilingService(controller monitoring_domain.ProfilingController) *ProfilingService {
	return &ProfilingService{controller: controller}
}

// EnableProfiling starts the pprof server and configures runtime profiling rates.
//
// Takes request (*pb.EnableProfilingRequest) which specifies duration, port,
// and sampling rates.
//
// Returns *pb.EnableProfilingResponse which contains the active session details.
// Returns error when validation fails or the controller returns an error.
func (s *ProfilingService) EnableProfiling(ctx context.Context, request *pb.EnableProfilingRequest) (*pb.EnableProfilingResponse, error) {
	durationMs := request.GetDurationMs()
	if durationMs <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "duration_ms must be positive")
	}

	port := request.GetPort()
	if port != 0 && (port < 1 || port > maxTCPPort) {
		return nil, status.Errorf(codes.InvalidArgument, "port must be between 1 and %d", maxTCPPort)
	}

	opts := monitoring_domain.ProfilingEnableOpts{
		Duration:             time.Duration(durationMs) * time.Millisecond,
		Port:                 int(port),
		BlockProfileRate:     int(request.GetBlockProfileRate()),
		MutexProfileFraction: int(request.GetMutexProfileFraction()),
	}

	profilingStatus, err := s.controller.Enable(ctx, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "enabling profiling: %v", err)
	}

	return &pb.EnableProfilingResponse{
		AlreadyEnabled:       profilingStatus.AlreadyEnabled,
		ExpiresAtMs:          profilingStatus.ExpiresAt.UnixMilli(),
		Port:                 safeconv.IntToInt32(profilingStatus.Port),
		PprofBaseUrl:         profilingStatus.PprofBaseURL,
		BlockProfileRate:     safeconv.IntToInt32(profilingStatus.BlockProfileRate),
		MutexProfileFraction: safeconv.IntToInt32(profilingStatus.MutexProfileFraction),
		MemProfileRate:       safeconv.IntToInt32(profilingStatus.MemProfileRate),
	}, nil
}

// DisableProfiling stops the pprof server and resets runtime profiling rates.
//
// Returns *pb.DisableProfilingResponse which carries the prior enabled
// state of the profiling session.
// Returns error when the controller fails to disable profiling.
func (s *ProfilingService) DisableProfiling(ctx context.Context, _ *pb.DisableProfilingRequest) (*pb.DisableProfilingResponse, error) {
	wasEnabled, err := s.controller.Disable(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "disabling profiling: %v", err)
	}

	return &pb.DisableProfilingResponse{
		WasEnabled: wasEnabled,
	}, nil
}

// GetProfilingStatus returns the current profiling state.
//
// Returns *pb.GetProfilingStatusResponse which contains enabled state, rates,
// and available profiles.
// Returns error (always nil; included for interface compliance).
func (s *ProfilingService) GetProfilingStatus(ctx context.Context, _ *pb.GetProfilingStatusRequest) (*pb.GetProfilingStatusResponse, error) {
	profilingStatus := s.controller.Status(ctx)

	response := &pb.GetProfilingStatusResponse{
		Enabled:              profilingStatus.Enabled,
		Port:                 safeconv.IntToInt32(profilingStatus.Port),
		PprofBaseUrl:         profilingStatus.PprofBaseURL,
		BlockProfileRate:     safeconv.IntToInt32(profilingStatus.BlockProfileRate),
		MutexProfileFraction: safeconv.IntToInt32(profilingStatus.MutexProfileFraction),
		MemProfileRate:       safeconv.IntToInt32(profilingStatus.MemProfileRate),
		AvailableProfiles:    profilingStatus.AvailableProfiles,
	}

	if profilingStatus.Enabled {
		response.ExpiresAtMs = profilingStatus.ExpiresAt.UnixMilli()
		response.RemainingMs = max(time.Until(profilingStatus.ExpiresAt).Milliseconds(), 0)
	}

	return response, nil
}

// CaptureProfile captures a Go runtime profile and streams the data back
// in chunks.
//
// Takes request (*pb.CaptureProfileRequest) which specifies the profile type
// and optional duration.
// Takes stream (pb.ProfilingService_CaptureProfileServer) which receives the
// chunked profile data.
//
// Returns error when validation, capture, or streaming fails.
func (s *ProfilingService) CaptureProfile(request *pb.CaptureProfileRequest, stream pb.ProfilingService_CaptureProfileServer) error {
	profileType := request.GetProfileType()
	if profileType == "" {
		return status.Errorf(codes.InvalidArgument, "profile_type must not be empty")
	}

	durationSeconds := request.GetDurationSeconds()
	if durationSeconds < 0 {
		return status.Errorf(codes.InvalidArgument, "duration_seconds must not be negative")
	}

	captureBuffer := newLimitedBuffer(maxCaptureBytes)

	warning, err := s.controller.CaptureProfile(
		stream.Context(),
		profileType,
		int(durationSeconds),
		captureBuffer,
	)
	if err != nil {
		if errors.Is(err, errCaptureLimitExceeded) {
			return status.Errorf(codes.ResourceExhausted,
				"profile size exceeds maximum of %d bytes; use the pprof HTTP server directly",
				maxCaptureBytes)
		}

		return status.Errorf(codes.Internal, "capturing %s profile: %v", profileType, err)
	}

	return sendProfileChunks(stream, captureBuffer.Bytes(), warning)
}

// sendProfileChunks writes profile data to the stream in fixed-size chunks,
// attaching any server warning to the first chunk.
//
// Takes stream (pb.ProfilingService_CaptureProfileServer) which receives each chunk.
// Takes data ([]byte) which is the complete profile payload.
// Takes warning (string) which is attached to the first chunk if non-empty.
//
// Returns error when the stream context is cancelled or a send fails.
func sendProfileChunks(stream pb.ProfilingService_CaptureProfileServer, data []byte, warning string) error {
	if len(data) == 0 {
		if err := stream.Send(&pb.CaptureProfileChunk{
			IsLast:  true,
			Warning: warning,
		}); err != nil {
			return fmt.Errorf("sending empty profile response: %w", err)
		}
		return nil
	}

	for offset := 0; offset < len(data); offset += captureChunkSize {
		if err := stream.Context().Err(); err != nil {
			return err
		}

		end := min(offset+captureChunkSize, len(data))
		isLast := end >= len(data)

		chunk := &pb.CaptureProfileChunk{
			Data:   data[offset:end],
			IsLast: isLast,
		}
		if offset == 0 && warning != "" {
			chunk.Warning = warning
		}

		if err := stream.Send(chunk); err != nil {
			return fmt.Errorf("sending profile chunk: %w", err)
		}
	}

	return nil
}
