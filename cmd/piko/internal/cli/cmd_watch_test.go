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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestIsStreamDone(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err  error
		name string
		want bool
	}{
		{name: "io.EOF returns true", err: io.EOF, want: true},
		{name: "context.Canceled returns true", err: context.Canceled, want: true},
		{name: "wrapped io.EOF returns true", err: fmt.Errorf("stream closed: %w", io.EOF), want: true},
		{name: "other error returns false", err: errors.New("something went wrong"), want: false},
		{name: "nil returns false", err: nil, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isStreamDone(tc.err)
			if got != tc.want {
				t.Errorf("isStreamDone(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestClearScreen(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	clearScreen(&buffer)

	want := "\033[H\033[2J"
	got := buffer.String()
	if got != want {
		t.Errorf("clearScreen() wrote %q, want %q", got, want)
	}
}

func TestWatchHealth(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		health: &mockHealthClient{
			WatchHealthFunc: func(_ context.Context, _ *pb.WatchHealthRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.HealthUpdate], error) {
				return &mockStream[pb.HealthUpdate]{
					items: []*pb.HealthUpdate{
						{
							TimestampMs: 1700000000,
							Liveness:    &pb.HealthStatus{Name: "Liveness", State: "HEALTHY"},
							Readiness:   &pb.HealthStatus{Name: "Readiness", State: "DEGRADED"},
						},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		format  string
		wantAll []string
	}{
		{
			name:    "table output",
			format:  "table",
			wantAll: []string{"Health", "Liveness", "Readiness", "HEALTHY"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"timestamp_ms"`, `"liveness"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := watchHealth(context.Background(), conn, &buffer, p, time.Second)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestWatchTasks(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		orchestrator: &mockOrchestratorClient{
			WatchTasksFunc: func(_ context.Context, _ *pb.WatchTasksRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.TasksUpdate], error) {
				return &mockStream[pb.TasksUpdate]{
					items: []*pb.TasksUpdate{
						{
							TimestampMs: 1700000000,
							Summaries: []*pb.TaskSummary{
								{Status: "COMPLETED", Count: 42},
							},
							RecentTasks: []*pb.TaskListItem{
								{
									Id:         "task-001",
									WorkflowId: "wf-abc",
									Executor:   "worker-1",
									Status:     "COMPLETED",
									Attempt:    1,
									UpdatedAt:  1700000000,
								},
							},
						},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		format  string
		wantAll []string
	}{
		{
			name:    "table output",
			format:  "table",
			wantAll: []string{"Tasks", "Summary", "COMPLETED", "42", "task-001", "wf-abc", "worker-1"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"timestamp_ms"`, `"summaries"`, `"recent_tasks"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := watchTasks(context.Background(), conn, &buffer, p, time.Second)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestWatchArtefacts(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		registry: &mockRegistryClient{
			WatchArtefactsFunc: func(_ context.Context, _ *pb.WatchArtefactsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.ArtefactsUpdate], error) {
				return &mockStream[pb.ArtefactsUpdate]{
					items: []*pb.ArtefactsUpdate{
						{
							TimestampMs: 1700000000,
							Summaries: []*pb.ArtefactSummary{
								{Status: "READY", Count: 10},
							},
							RecentArtefacts: []*pb.ArtefactListItem{
								{
									Id:           "art-001",
									SourcePath:   "/images/logo.png",
									Status:       "READY",
									VariantCount: 3,
									TotalSize:    1024,
									UpdatedAt:    1700000000,
								},
							},
						},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		format  string
		wantAll []string
	}{
		{
			name:    "table output",
			format:  "table",
			wantAll: []string{"Artefacts", "Summary", "READY", "10", "art-001", "/images/logo.png"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"timestamp_ms"`, `"summaries"`, `"recent_artefacts"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := watchArtefacts(context.Background(), conn, &buffer, p, time.Second)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestWatchMetrics(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			WatchMetricsFunc: func(_ context.Context, _ *pb.WatchMetricsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.MetricsUpdate], error) {
				return &mockStream[pb.MetricsUpdate]{
					items: []*pb.MetricsUpdate{
						{
							TimestampMs: 1700000000,
							Metrics: []*pb.Metric{
								{
									Name: "http_requests_total",
									Type: "counter",
									Unit: "1",
									DataPoints: []*pb.MetricDataPoint{
										{TimestampMs: 1700000000, Value: 123.0},
									},
								},
							},
						},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		format  string
		wantAll []string
	}{
		{
			name:    "table output",
			format:  "table",
			wantAll: []string{"Metrics", "http_requests_total", "counter", "1"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"timestamp_ms"`, `"metrics"`, `"http_requests_total"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := watchMetrics(context.Background(), conn, &buffer, p, time.Second)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestWatchHealthError(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		health: &mockHealthClient{
			WatchHealthFunc: func(_ context.Context, _ *pb.WatchHealthRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.HealthUpdate], error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)
	err := watchHealth(context.Background(), conn, &buffer, p, time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "connection refused")
	}
}

func TestRunWatchMissingResource(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	cc, _, _ := newTestCC(conn)

	err := runWatch(context.Background(), cc, nil)
	if err == nil {
		t.Fatal("expected error for missing resource, got nil")
	}
	if !strings.Contains(err.Error(), "missing resource type") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing resource type")
	}
}

func TestRunWatchUnknownResource(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{}
	cc, _, _ := newTestCC(conn)

	err := runWatch(context.Background(), cc, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown resource, got nil")
	}
	if !strings.Contains(err.Error(), "unknown resource") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "unknown resource")
	}
}

func TestRunWatchInvalidFormat(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		health: &mockHealthClient{
			WatchHealthFunc: func(_ context.Context, _ *pb.WatchHealthRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.HealthUpdate], error) {
				return &mockStream[pb.HealthUpdate]{}, nil
			},
		},
	}
	cc, _, _ := newTestCC(conn)
	cc.Opts.Output = "text"

	err := runWatch(context.Background(), cc, []string{"health"})
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
	if !strings.Contains(err.Error(), "text") {
		t.Errorf("error = %q, want it to mention the invalid format %q", err.Error(), "text")
	}
}
