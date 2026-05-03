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
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

type mockWatchdogEventStream struct {
	ctx     context.Context
	sendErr error
	sent    []*pb.WatchdogEventMessage
}

func (m *mockWatchdogEventStream) Send(message *pb.WatchdogEventMessage) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, message)
	return nil
}

func (m *mockWatchdogEventStream) Context() context.Context     { return m.ctx }
func (m *mockWatchdogEventStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockWatchdogEventStream) SendHeader(metadata.MD) error { return nil }
func (m *mockWatchdogEventStream) SetTrailer(metadata.MD)       {}
func (m *mockWatchdogEventStream) SendMsg(any) error            { return nil }
func (m *mockWatchdogEventStream) RecvMsg(any) error            { return nil }

type mockDownloadProfileStream struct {
	ctx     context.Context
	sendErr error
	sent    []*pb.DownloadProfileChunk
}

func (m *mockDownloadProfileStream) Send(chunk *pb.DownloadProfileChunk) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, chunk)
	return nil
}

func (m *mockDownloadProfileStream) Context() context.Context     { return m.ctx }
func (m *mockDownloadProfileStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockDownloadProfileStream) SendHeader(metadata.MD) error { return nil }
func (m *mockDownloadProfileStream) SetTrailer(metadata.MD)       {}
func (m *mockDownloadProfileStream) SendMsg(any) error            { return nil }
func (m *mockDownloadProfileStream) RecvMsg(any) error            { return nil }

func TestToGRPCError_NilReturnsNil(t *testing.T) {
	t.Parallel()

	require.NoError(t, toGRPCError(nil))
}

func TestToGRPCError_PreservesExistingStatus(t *testing.T) {
	t.Parallel()

	original := status.Error(codes.PermissionDenied, "denied")

	got := toGRPCError(original)

	require.Equal(t, original, got, "an existing gRPC status error must pass through untouched")
}

func TestToGRPCError_MapsContextCancelledToCanceled(t *testing.T) {
	t.Parallel()

	got := toGRPCError(fmt.Errorf("wrapping: %w", context.Canceled))

	gotStatus, ok := status.FromError(got)
	require.True(t, ok, "result must be a gRPC status error")
	assert.Equal(t, codes.Canceled, gotStatus.Code())
}

func TestToGRPCError_MapsDeadlineExceededToDeadlineExceeded(t *testing.T) {
	t.Parallel()

	got := toGRPCError(fmt.Errorf("wrapping: %w", context.DeadlineExceeded))

	gotStatus, ok := status.FromError(got)
	require.True(t, ok)
	assert.Equal(t, codes.DeadlineExceeded, gotStatus.Code())
}

func TestToGRPCError_MapsWatchdogStoppedToUnavailable(t *testing.T) {
	t.Parallel()

	got := toGRPCError(fmt.Errorf("wrapping: %w", monitoring_domain.ErrWatchdogStopped))

	gotStatus, ok := status.FromError(got)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, gotStatus.Code())
}

func TestToGRPCError_MapsSubscriberCapToResourceExhausted(t *testing.T) {
	t.Parallel()

	got := toGRPCError(fmt.Errorf("wrapping: %w", monitoring_domain.ErrEventSubscriberCapExceeded))

	gotStatus, ok := status.FromError(got)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, gotStatus.Code())
}

func TestToGRPCError_MapsFsErrNotExistToNotFound(t *testing.T) {
	t.Parallel()

	got := toGRPCError(fmt.Errorf("reading: %w", fs.ErrNotExist))

	gotStatus, ok := status.FromError(got)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, gotStatus.Code())
}

func TestToGRPCError_DefaultMapsToInternal(t *testing.T) {
	t.Parallel()

	got := toGRPCError(errors.New("boom"))

	gotStatus, ok := status.FromError(got)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, gotStatus.Code())
}

func TestWatchdogService_WatchEvents_ContextCancelledReturnsCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test cleanup"))

	channel := make(chan monitoring_domain.WatchdogEventInfo)
	cancelCalled := make(chan struct{}, 1)

	inspector := &mockWatchdogInspector{
		subscribeEventsFn: func(_ context.Context, _ time.Time) (<-chan monitoring_domain.WatchdogEventInfo, func()) {
			return channel, func() {
				select {
				case cancelCalled <- struct{}{}:
				default:
				}
			}
		},
	}

	service := NewWatchdogInspectorService(inspector)
	stream := &mockWatchdogEventStream{ctx: ctx}

	errCh := make(chan error, 1)
	go func() {
		errCh <- service.WatchEvents(&pb.WatchEventsRequest{}, stream)
	}()

	cancel(fmt.Errorf("test: simulating cancelled context"))
	close(channel)

	err := <-errCh
	if err == nil {
		t.Fatal("WatchEvents must surface the cancellation as an error")
	}
	gotStatus, ok := status.FromError(err)
	require.True(t, ok, "WatchEvents must return a gRPC status error, got %T: %v", err, err)
	assert.Equal(t, codes.Canceled, gotStatus.Code(),
		"context cancellation must map to codes.Canceled")
	select {
	case <-cancelCalled:
	default:
		t.Fatal("WatchEvents must call the SubscribeEvents cancel func on exit")
	}
}

func TestWatchdogService_WatchEvents_StreamSendErrorReturnsStatusError(t *testing.T) {
	t.Parallel()

	channel := make(chan monitoring_domain.WatchdogEventInfo, 1)
	channel <- monitoring_domain.WatchdogEventInfo{
		EventType: monitoring_domain.WatchdogEventType("info"),
		Message:   "hello",
		EmittedAt: time.Now(),
	}
	close(channel)

	inspector := &mockWatchdogInspector{
		subscribeEventsFn: func(_ context.Context, _ time.Time) (<-chan monitoring_domain.WatchdogEventInfo, func()) {
			return channel, func() {}
		},
	}

	service := NewWatchdogInspectorService(inspector)
	stream := &mockWatchdogEventStream{
		ctx:     context.Background(),
		sendErr: errors.New("transport closed"),
	}

	err := service.WatchEvents(&pb.WatchEventsRequest{}, stream)

	require.Error(t, err)
	gotStatus, ok := status.FromError(err)
	require.True(t, ok, "send-side errors must be wrapped as a gRPC status, got %T: %v", err, err)
	assert.Equal(t, codes.Internal, gotStatus.Code())
}

func TestWatchdogService_DownloadProfile_NotFoundReturnsNotFound(t *testing.T) {
	t.Parallel()

	inspector := &mockWatchdogInspector{
		downloadProfileFn: func(_ context.Context, filename string, _ io.Writer) error {
			return fmt.Errorf("opening %s: %w", filename, fs.ErrNotExist)
		},
	}

	service := NewWatchdogInspectorService(inspector)
	stream := &mockDownloadProfileStream{ctx: context.Background()}

	err := service.DownloadProfile(&pb.DownloadProfileRequest{Filename: "missing.pb.gz"}, stream)

	require.Error(t, err)
	gotStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, gotStatus.Code(),
		"a missing profile file must surface as codes.NotFound, not codes.Internal")
}

func TestWatchdogService_DownloadProfile_EmptyFilenameReturnsInvalidArgument(t *testing.T) {
	t.Parallel()

	inspector := &mockWatchdogInspector{}
	service := NewWatchdogInspectorService(inspector)
	stream := &mockDownloadProfileStream{ctx: context.Background()}

	err := service.DownloadProfile(&pb.DownloadProfileRequest{}, stream)

	require.Error(t, err)
	gotStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, gotStatus.Code())
}

func TestWatchdogService_ListProfiles_InternalErrorReturnsInternal(t *testing.T) {
	t.Parallel()

	inspector := &mockWatchdogInspector{
		listProfilesFn: func(_ context.Context) ([]monitoring_domain.WatchdogProfileInfo, error) {
			return nil, errors.New("disk gone")
		},
	}

	service := NewWatchdogInspectorService(inspector)

	_, err := service.ListProfiles(context.Background(), &pb.ListProfilesRequest{})

	require.Error(t, err)
	gotStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, gotStatus.Code())
}

func TestWatchdogService_DownloadSidecar_PropagatesNotFound(t *testing.T) {
	t.Parallel()

	inspector := &mockWatchdogInspector{
		downloadSidecarFn: func(_ context.Context, profileFilename string) ([]byte, bool, error) {
			return nil, false, fmt.Errorf("opening sidecar for %s: %w", profileFilename, fs.ErrNotExist)
		},
	}

	service := NewWatchdogInspectorService(inspector)

	_, err := service.DownloadSidecar(context.Background(), &pb.DownloadSidecarRequest{ProfileFilename: "ghost.pb.gz"})

	require.Error(t, err)
	gotStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, gotStatus.Code())
}

func TestWatchdogService_PruneProfiles_PropagatesContextDeadline(t *testing.T) {
	t.Parallel()

	inspector := &mockWatchdogInspector{
		pruneProfilesFn: func(_ context.Context, _ string) (int, error) {
			return 0, fmt.Errorf("pruning interrupted: %w", context.DeadlineExceeded)
		},
	}

	service := NewWatchdogInspectorService(inspector)

	_, err := service.PruneProfiles(context.Background(), &pb.PruneProfilesRequest{})

	require.Error(t, err)
	gotStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.DeadlineExceeded, gotStatus.Code())
}

func TestWatchdogService_GetStartupHistory_PropagatesInternal(t *testing.T) {
	t.Parallel()

	inspector := &mockWatchdogInspector{
		getStartupHistoryFn: func(_ context.Context) ([]monitoring_domain.WatchdogStartupHistoryEntry, error) {
			return nil, errors.New("corrupt history")
		},
	}

	service := NewWatchdogInspectorService(inspector)

	_, err := service.GetStartupHistory(context.Background(), &pb.GetStartupHistoryRequest{})

	require.Error(t, err)
	gotStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, gotStatus.Code())
}
