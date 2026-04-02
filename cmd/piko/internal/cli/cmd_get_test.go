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
	"flag"
	"strings"
	"testing"

	"google.golang.org/grpc"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func errForHelp() error  { return flag.ErrHelp }
func errForOther() error { return errors.New("some error") }

func TestFormatReady(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		status   *pb.HealthStatus
		expected string
	}{
		{name: "no dependencies", status: &pb.HealthStatus{}, expected: "-"},
		{
			name: "all healthy",
			status: &pb.HealthStatus{
				Dependencies: []*pb.HealthStatus{
					{State: "HEALTHY"},
					{State: "HEALTHY"},
					{State: "HEALTHY"},
				},
			},
			expected: "3/3",
		},
		{
			name: "some unhealthy",
			status: &pb.HealthStatus{
				Dependencies: []*pb.HealthStatus{
					{State: "HEALTHY"},
					{State: "DEGRADED"},
					{State: "HEALTHY"},
					{State: "UNHEALTHY"},
				},
			},
			expected: "2/4",
		},
		{
			name: "case insensitive healthy",
			status: &pb.HealthStatus{
				Dependencies: []*pb.HealthStatus{
					{State: "healthy"},
					{State: "Healthy"},
				},
			},
			expected: "2/2",
		},
		{
			name: "none healthy",
			status: &pb.HealthStatus{
				Dependencies: []*pb.HealthStatus{
					{State: "UNHEALTHY"},
					{State: "DEGRADED"},
				},
			},
			expected: "0/2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatReady(tc.status)
			if got != tc.expected {
				t.Errorf("formatReady() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestBuildHealthRows(t *testing.T) {
	t.Parallel()

	response := &pb.GetHealthResponse{
		Liveness: &pb.HealthStatus{
			Name:    "Liveness",
			State:   "HEALTHY",
			Message: "All good",
			Dependencies: []*pb.HealthStatus{
				{State: "HEALTHY"},
				{State: "HEALTHY"},
			},
		},
		Readiness: &pb.HealthStatus{
			Name:    "Readiness",
			State:   "DEGRADED",
			Message: "Issues found",
			Dependencies: []*pb.HealthStatus{
				{State: "HEALTHY"},
				{State: "UNHEALTHY"},
			},
		},
	}

	testCases := []struct {
		name      string
		filter    string
		wantProbe string
		wantLen   int
	}{
		{name: "no filter returns both", filter: "", wantLen: 2},
		{name: "filter Liveness", filter: "Liveness", wantLen: 1, wantProbe: "Liveness"},
		{name: "filter Readiness", filter: "Readiness", wantLen: 1, wantProbe: "Readiness"},
		{name: "filter case insensitive", filter: "liveness", wantLen: 1, wantProbe: "Liveness"},
		{name: "filter prefix", filter: "Live", wantLen: 1, wantProbe: "Liveness"},
		{name: "no match", filter: "nonexistent", wantLen: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewPrinter(&bytes.Buffer{}, "table", true, false)
			rows := buildHealthRows(p, response, tc.filter)
			if len(rows) != tc.wantLen {
				t.Errorf("buildHealthRows() returned %d rows, want %d", len(rows), tc.wantLen)
			}
			if tc.wantProbe != "" && len(rows) > 0 && rows[0][0] != tc.wantProbe {
				t.Errorf("first row probe = %q, want %q", rows[0][0], tc.wantProbe)
			}
		})
	}
}

func TestHealthRow(t *testing.T) {
	t.Parallel()

	p := NewPrinter(&bytes.Buffer{}, "table", true, false)

	t.Run("nil status", func(t *testing.T) {
		t.Parallel()
		row := healthRow(p, "Liveness", nil)
		if row[0] != "Liveness" {
			t.Errorf("probe = %q, want Liveness", row[0])
		}
		if row[2] != "-" {
			t.Errorf("ready = %q, want -", row[2])
		}
	})

	t.Run("healthy with deps", func(t *testing.T) {
		t.Parallel()
		status := &pb.HealthStatus{
			State:   "HEALTHY",
			Message: "OK",
			Dependencies: []*pb.HealthStatus{
				{State: "HEALTHY"},
				{State: "HEALTHY"},
			},
		}
		row := healthRow(p, "Liveness", status)
		if row[0] != "Liveness" {
			t.Errorf("probe = %q, want Liveness", row[0])
		}
		if row[2] != "2/2" {
			t.Errorf("ready = %q, want 2/2", row[2])
		}
		if row[3] != "OK" {
			t.Errorf("message = %q, want OK", row[3])
		}
	})
}

func TestHelpOrError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err     error
		name    string
		wantNil bool
	}{
		{name: "ErrHelp returns nil", err: errForHelp(), wantNil: true},
		{name: "other error passes through", err: errForOther(), wantNil: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := helpOrError(tc.err)
			if tc.wantNil && got != nil {
				t.Errorf("helpOrError() = %v, want nil", got)
			}
			if !tc.wantNil && got == nil {
				t.Error("helpOrError() = nil, want non-nil error")
			}
		})
	}
}

func TestNewResourceFlagSet(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	fs := newResourceFlagSet("test", "piko get test [flags]", "Test command.", getFormatHelp, "table", &buffer)
	_ = fs.Int("limit", 10, "Maximum number of items")

	if err := fs.Parse([]string{"--help"}); err == nil {
		t.Error("expected ErrHelp from --help")
	}

	output := buffer.String()
	if output == "" {
		t.Error("expected help output, got empty string")
	}
}

func testHealthConn() monitoringConnection {
	return &mockConnection{
		health: &mockHealthClient{
			GetHealthFunc: func(_ context.Context, _ *pb.GetHealthRequest, _ ...grpc.CallOption) (*pb.GetHealthResponse, error) {
				return &pb.GetHealthResponse{
					Liveness:  &pb.HealthStatus{State: "HEALTHY", Message: "OK"},
					Readiness: &pb.HealthStatus{State: "DEGRADED", Message: "Issues"},
				}, nil
			},
		},
	}
}

func TestGetHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		output    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"Liveness", "Readiness", "HEALTHY", "DEGRADED"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"state": "HEALTHY"`, `"message": "OK"`},
		},
		{
			name:      "filter by name",
			output:    "table",
			arguments: []string{"Liveness"},
			wantAll:   []string{"Liveness"},
		},
	}

	conn := testHealthConn()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getHealth(context.Background(), conn, p, tc.arguments)
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

func TestGetHealthError(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		health: &mockHealthClient{
			GetHealthFunc: func(_ context.Context, _ *pb.GetHealthRequest, _ ...grpc.CallOption) (*pb.GetHealthResponse, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)
	err := getHealth(context.Background(), conn, p, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTasks(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		orchestrator: &mockOrchestratorClient{
			ListRecentTasksFunc: func(_ context.Context, _ *pb.ListRecentTasksRequest, _ ...grpc.CallOption) (*pb.ListRecentTasksResponse, error) {
				return &pb.ListRecentTasksResponse{
					Tasks: []*pb.TaskListItem{
						{Id: "task-1", WorkflowId: "wf-1", Executor: "exec-1", Status: "completed", Attempt: 1},
						{Id: "task-2", WorkflowId: "wf-2", Executor: "exec-2", Status: "running", Attempt: 2},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		output  string
		wantAll []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"task-1", "task-2", "wf-1", "exec-1", "completed"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"id": "task-1"`, `"workflow_id": "wf-1"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getTasks(context.Background(), conn, p, nil)
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

func TestGetWorkflows(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		orchestrator: &mockOrchestratorClient{
			ListWorkflowSummaryFunc: func(_ context.Context, _ *pb.ListWorkflowSummaryRequest, _ ...grpc.CallOption) (*pb.ListWorkflowSummaryResponse, error) {
				return &pb.ListWorkflowSummaryResponse{
					Summaries: []*pb.WorkflowSummary{
						{WorkflowId: "wf-abc", TaskCount: 5, CompleteCount: 3, FailedCount: 1, ActiveCount: 1},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		output  string
		wantAll []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"wf-abc", "5", "3", "1"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"workflow_id": "wf-abc"`, `"task_count": 5`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getWorkflows(context.Background(), conn, p, nil)
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

func TestGetArtefacts(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		registry: &mockRegistryClient{
			ListRecentArtefactsFunc: func(_ context.Context, _ *pb.ListRecentArtefactsRequest, _ ...grpc.CallOption) (*pb.ListRecentArtefactsResponse, error) {
				return &pb.ListRecentArtefactsResponse{
					Artefacts: []*pb.ArtefactListItem{
						{Id: "art-1", SourcePath: "/src/app.pk", Status: "ready", VariantCount: 3, TotalSize: 1024},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		output  string
		wantAll []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"art-1", "/src/app.pk", "ready", "3"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"id": "art-1"`, `"source_path": "/src/app.pk"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getArtefacts(context.Background(), conn, p, nil)
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

func TestGetVariants(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		registry: &mockRegistryClient{
			GetVariantSummaryFunc: func(_ context.Context, _ *pb.GetVariantSummaryRequest, _ ...grpc.CallOption) (*pb.GetVariantSummaryResponse, error) {
				return &pb.GetVariantSummaryResponse{
					Summaries: []*pb.VariantSummary{
						{Status: "ready", Count: 10},
						{Status: "pending", Count: 5},
					},
				}, nil
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)
	err := getVariants(context.Background(), conn, p, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	for _, want := range []string{"ready", "10", "pending", "5"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, output)
		}
	}
}

func TestGetMetrics(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetMetricsFunc: func(_ context.Context, _ *pb.GetMetricsRequest, _ ...grpc.CallOption) (*pb.GetMetricsResponse, error) {
				return &pb.GetMetricsResponse{
					Metrics: []*pb.Metric{
						{Name: "http.request.duration", Type: "histogram", Unit: "ms", DataPoints: []*pb.MetricDataPoint{{}}},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		output  string
		wantAll []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"http.request.duration", "histogram", "1"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"name": "http.request.duration"`, `"type": "histogram"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getMetrics(context.Background(), conn, p, nil)
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

func TestGetTraces(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetTracesFunc: func(_ context.Context, _ *pb.GetTracesRequest, _ ...grpc.CallOption) (*pb.GetTracesResponse, error) {
				return &pb.GetTracesResponse{
					Spans: []*pb.Span{
						{TraceId: "trace-abc", SpanId: "span-1", Name: "GET /api", ServiceName: "service-a", Status: "ok", DurationNs: 1_000_000},
						{TraceId: "trace-definition", SpanId: "span-2", Name: "POST /api", ServiceName: "service-b", Status: "error", DurationNs: 5_000_000},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		output    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"trace-abc", "GET /api", "service-a"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"trace_id": "trace-abc"`, `"name": "GET /api"`},
		},
		{
			name:      "errors only",
			output:    "table",
			arguments: []string{"-errors"},
			wantAll:   []string{"POST /api", "error"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getTraces(context.Background(), conn, p, tc.arguments)
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

func TestGetOpenResources(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetFileDescriptorsFunc: func(_ context.Context, _ *pb.GetFileDescriptorsRequest, _ ...grpc.CallOption) (*pb.GetFileDescriptorsResponse, error) {
				return &pb.GetFileDescriptorsResponse{
					Total: 42,
					Categories: []*pb.FileDescriptorCategory{
						{Category: "sockets", Count: 20},
						{Category: "files", Count: 22},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		output  string
		wantAll []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"sockets", "20", "files", "22", "TOTAL", "42"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"total": 42`, `"category": "sockets"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getOpenResources(context.Background(), conn, p, nil)
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

func TestGetRateLimiter(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		rateLimiter: &mockRateLimiterClient{
			GetRateLimiterStatusFunc: func(_ context.Context, _ *pb.GetRateLimiterStatusRequest, _ ...grpc.CallOption) (*pb.GetRateLimiterStatusResponse, error) {
				return &pb.GetRateLimiterStatusResponse{
					TokenBucketStore: "memory",
					CounterStore:     "memory",
					FailPolicy:       "deny",
					TotalChecks:      100,
					TotalAllowed:     95,
					TotalDenied:      5,
					KeyPrefix:        "rl:",
				}, nil
			},
		},
	}

	testCases := []struct {
		name    string
		output  string
		wantAll []string
	}{
		{
			name:    "table output",
			output:  "table",
			wantAll: []string{"memory", "deny", "100", "95", "5"},
		},
		{
			name:    "json output",
			output:  "json",
			wantAll: []string{`"token_bucket_store": "memory"`, `"total_checks": 100`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getRateLimiter(context.Background(), conn, p, nil)
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

func TestGetDLQSummary(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		dispatcher: &mockDispatcherClient{
			GetDispatcherSummaryFunc: func(_ context.Context, _ *pb.GetDispatcherSummaryRequest, _ ...grpc.CallOption) (*pb.GetDispatcherSummaryResponse, error) {
				return &pb.GetDispatcherSummaryResponse{
					Summaries: []*pb.DispatcherSummary{
						{Type: "email", QueuedItems: 5, DeadLetterCount: 2, TotalProcessed: 100},
					},
				}, nil
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)
	err := getDLQSummary(context.Background(), conn, p, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	for _, want := range []string{"email", "5", "2", "100"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, output)
		}
	}
}

func TestGetDLQEntries(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		dispatcher: &mockDispatcherClient{
			ListDLQEntriesFunc: func(_ context.Context, _ *pb.ListDLQEntriesRequest, _ ...grpc.CallOption) (*pb.ListDLQEntriesResponse, error) {
				return &pb.ListDLQEntriesResponse{
					Entries: []*pb.DLQEntry{
						{Id: "dlq-1", Type: "email", OriginalError: "timeout", TotalAttempts: 3},
					},
				}, nil
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)
	err := getDLQEntries(context.Background(), conn, p, "email", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	for _, want := range []string{"dlq-1", "email", "timeout", "3"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, output)
		}
	}
}

func TestGetProviderResourceTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		output  string
		types   []string
		wantAll []string
	}{
		{
			name:    "table with types",
			output:  "table",
			types:   []string{"cache", "email", "storage"},
			wantAll: []string{"cache", "email", "storage", "Usage:"},
		},
		{
			name:    "empty types",
			output:  "table",
			types:   nil,
			wantAll: []string{"No resource types registered"},
		},
		{
			name:    "json output",
			output:  "json",
			types:   []string{"cache"},
			wantAll: []string{`"resource_types"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conn := &mockConnection{
				providerInfo: &mockProviderInfoClient{
					ListResourceTypesFunc: func(_ context.Context, _ *pb.ListResourceTypesRequest, _ ...grpc.CallOption) (*pb.ListResourceTypesResponse, error) {
						return &pb.ListResourceTypesResponse{ResourceTypes: tc.types}, nil
					},
				},
			}

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.output, true, false)
			err := getProviderResourceTypes(context.Background(), conn, p)
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

func TestGetProviderList(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		providerInfo: &mockProviderInfoClient{
			ListProvidersFunc: func(_ context.Context, in *pb.ListProvidersRequest, _ ...grpc.CallOption) (*pb.ListProvidersResponse, error) {
				return &pb.ListProvidersResponse{
					Columns: []*pb.ProviderColumn{
						{Header: "NAME", Key: "name"},
						{Header: "TYPE", Key: "type"},
					},
					Rows: []*pb.ProviderRow{
						{Name: "otter", IsDefault: true, Values: map[string]string{"name": "otter", "type": "in-memory"}},
						{Name: "redis", IsDefault: false, Values: map[string]string{"name": "redis", "type": "distributed"}},
					},
				}, nil
			},
			ListSubResourcesFunc: func(_ context.Context, _ *pb.ListSubResourcesRequest, _ ...grpc.CallOption) (*pb.ListSubResourcesResponse, error) {
				return nil, errors.New("not supported")
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)
	err := getProviderList(context.Background(), conn, p, []string{"cache"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	for _, want := range []string{"otter", "redis", "in-memory", "distributed"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, output)
		}
	}
}

func TestGetProviderListSubResources(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		providerInfo: &mockProviderInfoClient{
			ListSubResourcesFunc: func(_ context.Context, _ *pb.ListSubResourcesRequest, _ ...grpc.CallOption) (*pb.ListSubResourcesResponse, error) {
				return &pb.ListSubResourcesResponse{
					Columns: []*pb.ProviderColumn{
						{Header: "NAMESPACE", Key: "ns"},
						{Header: "ENTRIES", Key: "entries"},
					},
					Rows: []*pb.ProviderRow{
						{Values: map[string]string{"ns": "default", "entries": "42"}},
					},
				}, nil
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)
	err := getProviderList(context.Background(), conn, p, []string{"cache", "otter"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	for _, want := range []string{"NAMESPACE", "default", "42"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, output)
		}
	}
}

func TestRunGetMissingResource(t *testing.T) {
	t.Parallel()

	cc, _, stderr := newTestCC(nil)
	err := runGet(context.Background(), cc, nil)
	if err == nil {
		t.Fatal("expected error for missing resource")
	}
	_ = stderr
	if !strings.Contains(err.Error(), "missing resource type") {
		t.Errorf("error = %q, want containing 'missing resource type'", err.Error())
	}
}

func TestRunGetUnknownResource(t *testing.T) {
	t.Parallel()

	cc, _, _ := newTestCC(nil)
	err := runGet(context.Background(), cc, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown resource")
	}
	if !strings.Contains(err.Error(), "unknown resource") {
		t.Errorf("error = %q, want containing 'unknown resource'", err.Error())
	}
}

func TestBuildProviderRows(t *testing.T) {
	t.Parallel()

	p := NewPrinter(&bytes.Buffer{}, "table", true, false)

	response := &pb.ListProvidersResponse{
		Columns: []*pb.ProviderColumn{
			{Header: "NAME", Key: "name"},
			{Header: "VERSION", Key: "version", WideOnly: true},
		},
		Rows: []*pb.ProviderRow{
			{Name: "otter", IsDefault: true, Values: map[string]string{"name": "otter", "version": "1.0"}},
			{Name: "redis", IsDefault: false, Values: map[string]string{"name": "redis", "version": "2.0"}},
		},
	}

	testCases := []struct {
		name      string
		filter    string
		wantFirst string
		wantCols  int
		wantRows  int
	}{
		{name: "no filter", filter: "", wantCols: 3, wantRows: 2},
		{name: "filter otter", filter: "otter", wantCols: 3, wantRows: 1, wantFirst: "otter"},
		{name: "no match", filter: "nonexistent", wantCols: 3, wantRows: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			columns, rows := buildProviderRows(p, response, tc.filter)
			if len(columns) != tc.wantCols {
				t.Errorf("columns = %d, want %d", len(columns), tc.wantCols)
			}
			if len(rows) != tc.wantRows {
				t.Errorf("rows = %d, want %d", len(rows), tc.wantRows)
			}
		})
	}
}

func TestBuildSubResourceRows(t *testing.T) {
	t.Parallel()

	response := &pb.ListSubResourcesResponse{
		Columns: []*pb.ProviderColumn{
			{Header: "NAMESPACE", Key: "ns"},
			{Header: "ENTRIES", Key: "entries"},
		},
		Rows: []*pb.ProviderRow{
			{Values: map[string]string{"ns": "default", "entries": "42"}},
			{Values: map[string]string{"ns": "users", "entries": "10"}},
		},
	}

	columns, rows := buildSubResourceRows(response)
	if len(columns) != 2 {
		t.Errorf("columns = %d, want 2", len(columns))
	}
	if len(rows) != 2 {
		t.Errorf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "default" {
		t.Errorf("first row ns = %q, want 'default'", rows[0][0])
	}
}

func TestGetDLQDispatcher(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		dispatcher: &mockDispatcherClient{
			GetDispatcherSummaryFunc: func(_ context.Context, _ *pb.GetDispatcherSummaryRequest, _ ...grpc.CallOption) (*pb.GetDispatcherSummaryResponse, error) {
				return &pb.GetDispatcherSummaryResponse{
					Summaries: []*pb.DispatcherSummary{
						{Type: "email", QueuedItems: 5},
					},
				}, nil
			},
			ListDLQEntriesFunc: func(_ context.Context, _ *pb.ListDLQEntriesRequest, _ ...grpc.CallOption) (*pb.ListDLQEntriesResponse, error) {
				return &pb.ListDLQEntriesResponse{
					Entries: []*pb.DLQEntry{
						{Id: "dlq-1", Type: "email"},
					},
				}, nil
			},
		},
	}

	t.Run("no filter routes to summary", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		p := NewPrinter(&buffer, "table", true, false)
		err := getDLQ(context.Background(), conn, p, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buffer.String(), "email") {
			t.Errorf("expected summary output, got: %s", buffer.String())
		}
	})

	t.Run("with filter routes to entries", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		p := NewPrinter(&buffer, "table", true, false)
		err := getDLQ(context.Background(), conn, p, []string{"email"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buffer.String(), "dlq-1") {
			t.Errorf("expected entry output, got: %s", buffer.String())
		}
	})
}

func TestGetProvidersDispatcher(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		providerInfo: &mockProviderInfoClient{
			ListResourceTypesFunc: func(_ context.Context, _ *pb.ListResourceTypesRequest, _ ...grpc.CallOption) (*pb.ListResourceTypesResponse, error) {
				return &pb.ListResourceTypesResponse{
					ResourceTypes: []string{"cache"},
				}, nil
			},
			ListProvidersFunc: func(_ context.Context, _ *pb.ListProvidersRequest, _ ...grpc.CallOption) (*pb.ListProvidersResponse, error) {
				return &pb.ListProvidersResponse{
					Columns: []*pb.ProviderColumn{{Header: "NAME", Key: "name"}},
					Rows:    []*pb.ProviderRow{{Name: "otter", Values: map[string]string{"name": "otter"}}},
				}, nil
			},
			ListSubResourcesFunc: func(_ context.Context, _ *pb.ListSubResourcesRequest, _ ...grpc.CallOption) (*pb.ListSubResourcesResponse, error) {
				return nil, errors.New("not supported")
			},
		},
	}

	t.Run("no arguments routes to resource types", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		p := NewPrinter(&buffer, "table", true, false)
		err := getProviders(context.Background(), conn, p, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buffer.String(), "cache") {
			t.Errorf("expected resource type output, got: %s", buffer.String())
		}
	})

	t.Run("with resource type routes to provider list", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		p := NewPrinter(&buffer, "table", true, false)
		err := getProviders(context.Background(), conn, p, []string{"cache"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buffer.String(), "otter") {
			t.Errorf("expected provider list output, got: %s", buffer.String())
		}
	})
}
