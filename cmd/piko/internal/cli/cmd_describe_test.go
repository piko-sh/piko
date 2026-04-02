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
	"strings"
	"testing"

	"google.golang.org/grpc"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestBuildHealthDetailSections(t *testing.T) {
	t.Parallel()

	response := &pb.GetHealthResponse{
		Liveness: &pb.HealthStatus{
			Name:    "Liveness",
			State:   "HEALTHY",
			Message: "All good",
			Dependencies: []*pb.HealthStatus{
				{Name: "Database", State: "HEALTHY", Duration: "0.5ms"},
				{Name: "Cache", State: "HEALTHY", Duration: "0.3ms"},
			},
		},
		Readiness: &pb.HealthStatus{
			Name:    "Readiness",
			State:   "DEGRADED",
			Message: "Issues found",
			Dependencies: []*pb.HealthStatus{
				{Name: "Database", State: "HEALTHY"},
				{Name: "Queue", State: "UNHEALTHY", Message: "timeout"},
			},
		},
	}

	testCases := []struct {
		name         string
		filter       string
		wantTitle    string
		wantSections int
		wantSubCount int
	}{
		{name: "no filter returns both", filter: "", wantSections: 2},
		{name: "filter Liveness", filter: "Liveness", wantSections: 1, wantTitle: "Liveness", wantSubCount: 2},
		{name: "filter Readiness", filter: "Readiness", wantSections: 1, wantTitle: "Readiness", wantSubCount: 2},
		{name: "no match", filter: "nonexistent", wantSections: 0},
		{name: "case insensitive", filter: "liveness", wantSections: 1, wantTitle: "Liveness"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewPrinter(&bytes.Buffer{}, "table", true, false)
			sections := buildHealthDetailSections(p, response, tc.filter)
			if len(sections) != tc.wantSections {
				t.Errorf("got %d sections, want %d", len(sections), tc.wantSections)
			}
			if tc.wantTitle != "" && len(sections) > 0 && sections[0].Title != tc.wantTitle {
				t.Errorf("first section title = %q, want %q", sections[0].Title, tc.wantTitle)
			}
			if tc.wantSubCount > 0 && len(sections) > 0 && len(sections[0].SubSections) != tc.wantSubCount {
				t.Errorf("sub-sections = %d, want %d", len(sections[0].SubSections), tc.wantSubCount)
			}
		})
	}
}

func TestBuildHealthDetailSections_NilProbe(t *testing.T) {
	t.Parallel()

	response := &pb.GetHealthResponse{
		Liveness: &pb.HealthStatus{Name: "Liveness", State: "HEALTHY"},
	}

	p := NewPrinter(&bytes.Buffer{}, "table", true, false)
	sections := buildHealthDetailSections(p, response, "")
	if len(sections) != 1 {
		t.Errorf("got %d sections, want 1 (nil readiness should be skipped)", len(sections))
	}
}

func TestBuildTaskDetailSections(t *testing.T) {
	t.Parallel()

	tasks := []*pb.TaskListItem{
		{Id: "task-001", WorkflowId: "wf-001", Executor: "img", Status: "RUNNING", Priority: 5, Attempt: 2},
		{Id: "task-002", WorkflowId: "wf-002", Executor: "pdf", Status: "COMPLETE"},
	}

	testCases := []struct {
		name         string
		filter       string
		wantSections int
	}{
		{name: "no filter", filter: "", wantSections: 2},
		{name: "filter by ID", filter: "task-001", wantSections: 1},
		{name: "prefix filter", filter: "task", wantSections: 2},
		{name: "no match", filter: "xyz", wantSections: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewPrinter(&bytes.Buffer{}, "table", true, false)
			sections := buildTaskDetailSections(p, tasks, tc.filter)
			if len(sections) != tc.wantSections {
				t.Errorf("got %d sections, want %d", len(sections), tc.wantSections)
			}
		})
	}
}

func TestBuildTaskDetailSections_WithLastError(t *testing.T) {
	t.Parallel()

	errMessage := "connection timeout"
	tasks := []*pb.TaskListItem{
		{Id: "task-001", Status: "FAILED", LastError: &errMessage},
	}

	p := NewPrinter(&bytes.Buffer{}, "table", true, false)
	sections := buildTaskDetailSections(p, tasks, "")
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}

	hasLastError := false
	for _, f := range sections[0].Fields {
		if f.Key == "Last Error" && f.Value == errMessage {
			hasLastError = true
		}
	}
	if !hasLastError {
		t.Error("expected Last Error field in task detail")
	}
}

func TestBuildWorkflowDetailSections(t *testing.T) {
	t.Parallel()

	workflows := []*pb.WorkflowSummary{
		{WorkflowId: "wf-001", TaskCount: 10, CompleteCount: 8, FailedCount: 1, ActiveCount: 1},
		{WorkflowId: "wf-002", TaskCount: 5, CompleteCount: 5},
	}

	testCases := []struct {
		name         string
		filter       string
		wantSections int
	}{
		{name: "no filter", filter: "", wantSections: 2},
		{name: "filter by ID", filter: "wf-001", wantSections: 1},
		{name: "no match", filter: "xyz", wantSections: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sections := buildWorkflowDetailSections(workflows, tc.filter)
			if len(sections) != tc.wantSections {
				t.Errorf("got %d sections, want %d", len(sections), tc.wantSections)
			}
		})
	}
}

func TestBuildArtefactDetailSections(t *testing.T) {
	t.Parallel()

	artefacts := []*pb.ArtefactListItem{
		{Id: "art-001", SourcePath: "/images/logo.png", Status: "READY", VariantCount: 3, TotalSize: 1024},
		{Id: "art-002", SourcePath: "/css/main.css", Status: "PROCESSING"},
	}

	testCases := []struct {
		name         string
		filter       string
		wantSections int
	}{
		{name: "no filter", filter: "", wantSections: 2},
		{name: "filter by ID", filter: "art-001", wantSections: 1},
		{name: "filter by source path", filter: "/css", wantSections: 1},
		{name: "no match", filter: "xyz", wantSections: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewPrinter(&bytes.Buffer{}, "table", true, false)
			sections := buildArtefactDetailSections(p, artefacts, tc.filter)
			if len(sections) != tc.wantSections {
				t.Errorf("got %d sections, want %d", len(sections), tc.wantSections)
			}
		})
	}
}

func TestBuildDLQDetailSections(t *testing.T) {
	t.Parallel()

	summaries := []*pb.DispatcherSummary{
		{Type: "email", QueuedItems: 5, DeadLetterCount: 2, TotalProcessed: 100},
		{Type: "sms", QueuedItems: 1, TotalProcessed: 50},
	}

	testCases := []struct {
		name         string
		filter       string
		wantSections int
	}{
		{name: "no filter", filter: "", wantSections: 2},
		{name: "filter email", filter: "email", wantSections: 1},
		{name: "no match", filter: "xyz", wantSections: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sections := buildDLQDetailSections(summaries, tc.filter)
			if len(sections) != tc.wantSections {
				t.Errorf("got %d sections, want %d", len(sections), tc.wantSections)
			}
		})
	}
}

func TestBuildRateLimiterDetailSections(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetRateLimiterStatusResponse
		wantRate string
	}{
		{
			name: "with traffic",
			response: &pb.GetRateLimiterStatusResponse{
				TokenBucketStore: "memory",
				CounterStore:     "redis",
				FailPolicy:       "deny",
				TotalChecks:      100,
				TotalAllowed:     90,
				TotalDenied:      10,
			},
			wantRate: "90.0% allowed",
		},
		{
			name: "no traffic",
			response: &pb.GetRateLimiterStatusResponse{
				TokenBucketStore: "memory",
				FailPolicy:       "allow",
			},
			wantRate: "-",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sections := buildRateLimiterDetailSections(tc.response)
			if len(sections) != 2 {
				t.Fatalf("got %d sections, want 2", len(sections))
			}

			counters := sections[1]
			var gotRate string
			for _, f := range counters.Fields {
				if f.Key == "Allow Rate" {
					gotRate = f.Value
				}
			}
			if gotRate != tc.wantRate {
				t.Errorf("Allow Rate = %q, want %q", gotRate, tc.wantRate)
			}
		})
	}
}

func TestBuildResourceDetailSections(t *testing.T) {
	t.Parallel()

	response := &pb.GetFileDescriptorsResponse{
		Total: 10,
		Categories: []*pb.FileDescriptorCategory{
			{
				Category: "socket",
				Count:    3,
				Fds: []*pb.FileDescriptorInfo{
					{Fd: 5, Target: "127.0.0.1:8080", AgeMs: 5000},
				},
			},
			{Category: "pipe", Count: 2},
		},
		TimestampMs: 1700000000000,
	}

	t.Run("no filter includes summary", func(t *testing.T) {
		t.Parallel()
		sections := buildResourceDetailSections(response, "")
		if len(sections) != 3 {
			t.Errorf("got %d sections, want 3 (summary + 2 categories)", len(sections))
		}
		if sections[0].Title != "Summary" {
			t.Errorf("first section title = %q, want Summary", sections[0].Title)
		}
	})

	t.Run("filter skips summary", func(t *testing.T) {
		t.Parallel()
		sections := buildResourceDetailSections(response, "socket")
		if len(sections) != 1 {
			t.Errorf("got %d sections, want 1", len(sections))
		}
		if len(sections[0].SubSections) != 1 {
			t.Errorf("socket sub-sections = %d, want 1", len(sections[0].SubSections))
		}
	})
}

func TestFilterTasks(t *testing.T) {
	t.Parallel()

	tasks := []*pb.TaskListItem{
		{Id: "task-001", Executor: "img"},
		{Id: "task-002", Executor: "pdf"},
		{Id: "other-003", Executor: "doc"},
	}

	testCases := []struct {
		name      string
		filter    string
		wantCount int
	}{
		{name: "no filter", filter: "", wantCount: 3},
		{name: "exact match", filter: "task-001", wantCount: 1},
		{name: "prefix match", filter: "task", wantCount: 2},
		{name: "no match", filter: "xyz", wantCount: 0},
		{name: "case insensitive", filter: "TASK", wantCount: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := filterTasks(tasks, tc.filter)
			if len(got) != tc.wantCount {
				t.Errorf("filterTasks() returned %d, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestFilterWorkflows(t *testing.T) {
	t.Parallel()

	workflows := []*pb.WorkflowSummary{
		{WorkflowId: "wf-001", TaskCount: 5},
		{WorkflowId: "wf-002", TaskCount: 10},
		{WorkflowId: "batch-003", TaskCount: 3},
	}

	testCases := []struct {
		name      string
		filter    string
		wantCount int
	}{
		{name: "no filter", filter: "", wantCount: 3},
		{name: "exact match", filter: "wf-001", wantCount: 1},
		{name: "prefix match", filter: "wf", wantCount: 2},
		{name: "no match", filter: "xyz", wantCount: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := filterWorkflows(workflows, tc.filter)
			if len(got) != tc.wantCount {
				t.Errorf("filterWorkflows() returned %d, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestFilterArtefacts(t *testing.T) {
	t.Parallel()

	artefacts := []*pb.ArtefactListItem{
		{Id: "art-001", SourcePath: "/images/logo.png"},
		{Id: "art-002", SourcePath: "/css/main.css"},
		{Id: "other-003", SourcePath: "/js/app.js"},
	}

	testCases := []struct {
		name      string
		filter    string
		wantCount int
	}{
		{name: "no filter", filter: "", wantCount: 3},
		{name: "filter by ID", filter: "art-001", wantCount: 1},
		{name: "filter by source path", filter: "/css", wantCount: 1},
		{name: "prefix match on ID", filter: "art", wantCount: 2},
		{name: "no match", filter: "xyz", wantCount: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := filterArtefacts(artefacts, tc.filter)
			if len(got) != tc.wantCount {
				t.Errorf("filterArtefacts() returned %d, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestDescribeHealth(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		health: &mockHealthClient{
			GetHealthFunc: func(_ context.Context, _ *pb.GetHealthRequest, _ ...grpc.CallOption) (*pb.GetHealthResponse, error) {
				return &pb.GetHealthResponse{
					Liveness:  &pb.HealthStatus{State: "HEALTHY", Message: "OK"},
					Readiness: &pb.HealthStatus{State: "DEGRADED", Message: "Issues"},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "text output",
			format:  "text",
			wantAll: []string{"Liveness", "Readiness", "HEALTHY", "DEGRADED"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"state": "HEALTHY"`, `"message": "OK"`},
		},
		{
			name:      "filter by probe name",
			format:    "text",
			arguments: []string{"Liveness"},
			wantAll:   []string{"Liveness", "HEALTHY"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeHealth(context.Background(), conn, p, tc.arguments)
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

func TestDescribeHealthError(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		health: &mockHealthClient{
			GetHealthFunc: func(_ context.Context, _ *pb.GetHealthRequest, _ ...grpc.CallOption) (*pb.GetHealthResponse, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "text", true, false)
	err := describeHealth(context.Background(), conn, p, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDescribeTrace(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetTracesFunc: func(_ context.Context, _ *pb.GetTracesRequest, _ ...grpc.CallOption) (*pb.GetTracesResponse, error) {
				return &pb.GetTracesResponse{
					Spans: []*pb.Span{
						{TraceId: "trace-abc", SpanId: "span-1", Name: "GET /api", ServiceName: "service-a", Status: "ok", DurationNs: 1_000_000},
						{TraceId: "trace-abc", SpanId: "span-2", ParentSpanId: "span-1", Name: "db.query", ServiceName: "service-a", Status: "ok", DurationNs: 500_000},
						{TraceId: "trace-other", SpanId: "span-3", Name: "POST /api", ServiceName: "service-b", Status: "error", DurationNs: 2_000_000},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:      "text output filters by trace ID",
			format:    "text",
			arguments: []string{"trace-abc"},
			wantAll:   []string{"trace-abc", "GET /api", "db.query", "Spans: 2"},
		},
		{
			name:      "json output",
			format:    "json",
			arguments: []string{"trace-abc"},
			wantAll:   []string{`"trace_id": "trace-abc"`, `"name": "GET /api"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeTrace(context.Background(), conn, p, tc.arguments)
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

func TestDescribeTraceMissingID(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "text", true, false)
	err := describeTrace(context.Background(), nil, p, nil)
	if err == nil {
		t.Fatal("expected error for missing trace ID, got nil")
	}
	if !strings.Contains(err.Error(), "missing trace ID") {
		t.Errorf("error = %q, want containing 'missing trace ID'", err.Error())
	}
}

func TestDescribeTraceNotFound(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetTracesFunc: func(_ context.Context, _ *pb.GetTracesRequest, _ ...grpc.CallOption) (*pb.GetTracesResponse, error) {
				return &pb.GetTracesResponse{
					Spans: []*pb.Span{
						{TraceId: "trace-other", SpanId: "span-1", Name: "GET /api"},
					},
				}, nil
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "text", true, false)
	err := describeTrace(context.Background(), conn, p, []string{"trace-nonexistent"})
	if err == nil {
		t.Fatal("expected error for non-existent trace, got nil")
	}
	if !strings.Contains(err.Error(), "no spans found") {
		t.Errorf("error = %q, want containing 'no spans found'", err.Error())
	}
}

func TestDescribeTask(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		orchestrator: &mockOrchestratorClient{
			ListRecentTasksFunc: func(_ context.Context, _ *pb.ListRecentTasksRequest, _ ...grpc.CallOption) (*pb.ListRecentTasksResponse, error) {
				return &pb.ListRecentTasksResponse{
					Tasks: []*pb.TaskListItem{
						{Id: "task-001", WorkflowId: "wf-001", Executor: "img", Status: "RUNNING", Priority: 5, Attempt: 2},
						{Id: "task-002", WorkflowId: "wf-002", Executor: "pdf", Status: "COMPLETE"},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "text all tasks",
			format:  "text",
			wantAll: []string{"task-001", "task-002", "RUNNING", "COMPLETE"},
		},
		{
			name:      "text filtered",
			format:    "text",
			arguments: []string{"task-001"},
			wantAll:   []string{"task-001", "wf-001", "img", "RUNNING"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"id": "task-001"`, `"workflow_id": "wf-001"`},
		},
		{
			name:      "json filtered",
			format:    "json",
			arguments: []string{"task-001"},
			wantAll:   []string{`"id": "task-001"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeTask(context.Background(), conn, p, tc.arguments)
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

func TestDescribeTaskError(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		orchestrator: &mockOrchestratorClient{
			ListRecentTasksFunc: func(_ context.Context, _ *pb.ListRecentTasksRequest, _ ...grpc.CallOption) (*pb.ListRecentTasksResponse, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "text", true, false)
	err := describeTask(context.Background(), conn, p, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDescribeWorkflow(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		orchestrator: &mockOrchestratorClient{
			ListWorkflowSummaryFunc: func(_ context.Context, _ *pb.ListWorkflowSummaryRequest, _ ...grpc.CallOption) (*pb.ListWorkflowSummaryResponse, error) {
				return &pb.ListWorkflowSummaryResponse{
					Summaries: []*pb.WorkflowSummary{
						{WorkflowId: "wf-001", TaskCount: 10, CompleteCount: 8, FailedCount: 1, ActiveCount: 1},
						{WorkflowId: "wf-002", TaskCount: 5, CompleteCount: 5},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "text all workflows",
			format:  "text",
			wantAll: []string{"wf-001", "wf-002", "10", "5"},
		},
		{
			name:      "text filtered",
			format:    "text",
			arguments: []string{"wf-001"},
			wantAll:   []string{"wf-001", "10", "8"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"workflow_id": "wf-001"`, `"task_count": 10`},
		},
		{
			name:      "json filtered",
			format:    "json",
			arguments: []string{"wf-001"},
			wantAll:   []string{`"workflow_id": "wf-001"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeWorkflow(context.Background(), conn, p, tc.arguments)
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

func TestDescribeArtefact(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		registry: &mockRegistryClient{
			ListRecentArtefactsFunc: func(_ context.Context, _ *pb.ListRecentArtefactsRequest, _ ...grpc.CallOption) (*pb.ListRecentArtefactsResponse, error) {
				return &pb.ListRecentArtefactsResponse{
					Artefacts: []*pb.ArtefactListItem{
						{Id: "art-001", SourcePath: "/images/logo.png", Status: "READY", VariantCount: 3, TotalSize: 1024},
						{Id: "art-002", SourcePath: "/css/main.css", Status: "PROCESSING"},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "text all artefacts",
			format:  "text",
			wantAll: []string{"art-001", "art-002", "READY", "PROCESSING"},
		},
		{
			name:      "text filtered by ID",
			format:    "text",
			arguments: []string{"art-001"},
			wantAll:   []string{"art-001", "/images/logo.png", "READY"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"id": "art-001"`, `"source_path": "/images/logo.png"`},
		},
		{
			name:      "json filtered",
			format:    "json",
			arguments: []string{"art-001"},
			wantAll:   []string{`"id": "art-001"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeArtefact(context.Background(), conn, p, tc.arguments)
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

func TestDescribeDLQ(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		dispatcher: &mockDispatcherClient{
			GetDispatcherSummaryFunc: func(_ context.Context, _ *pb.GetDispatcherSummaryRequest, _ ...grpc.CallOption) (*pb.GetDispatcherSummaryResponse, error) {
				return &pb.GetDispatcherSummaryResponse{
					Summaries: []*pb.DispatcherSummary{
						{Type: "email", QueuedItems: 5, DeadLetterCount: 2, TotalProcessed: 100},
						{Type: "sms", QueuedItems: 1, TotalProcessed: 50},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "text all dispatchers",
			format:  "text",
			wantAll: []string{"email", "sms", "100", "50"},
		},
		{
			name:      "text filtered",
			format:    "text",
			arguments: []string{"email"},
			wantAll:   []string{"email", "5", "2", "100"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"type": "email"`, `"queued_items": 5`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeDLQ(context.Background(), conn, p, tc.arguments)
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

func TestDescribeOpenResources(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetFileDescriptorsFunc: func(_ context.Context, _ *pb.GetFileDescriptorsRequest, _ ...grpc.CallOption) (*pb.GetFileDescriptorsResponse, error) {
				return &pb.GetFileDescriptorsResponse{
					Total: 10,
					Categories: []*pb.FileDescriptorCategory{
						{
							Category: "socket",
							Count:    3,
							Fds: []*pb.FileDescriptorInfo{
								{Fd: 5, Target: "127.0.0.1:8080", AgeMs: 5000},
							},
						},
						{Category: "pipe", Count: 2},
					},
					TimestampMs: 1700000000000,
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:    "text all categories",
			format:  "text",
			wantAll: []string{"Summary", "socket", "pipe", "10"},
		},
		{
			name:      "text filtered",
			format:    "text",
			arguments: []string{"socket"},
			wantAll:   []string{"socket", "127.0.0.1:8080"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"total": 10`, `"category": "socket"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeOpenResources(context.Background(), conn, p, tc.arguments)
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

func TestDescribeRateLimiter(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		rateLimiter: &mockRateLimiterClient{
			GetRateLimiterStatusFunc: func(_ context.Context, _ *pb.GetRateLimiterStatusRequest, _ ...grpc.CallOption) (*pb.GetRateLimiterStatusResponse, error) {
				return &pb.GetRateLimiterStatusResponse{
					TokenBucketStore: "memory",
					CounterStore:     "redis",
					FailPolicy:       "deny",
					TotalChecks:      100,
					TotalAllowed:     90,
					TotalDenied:      10,
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
			name:    "text output",
			format:  "text",
			wantAll: []string{"Rate Limiter", "memory", "redis", "deny", "90", "10"},
		},
		{
			name:    "json output",
			format:  "json",
			wantAll: []string{`"token_bucket_store": "memory"`, `"total_checks": 100`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeRateLimiter(context.Background(), conn, p, nil)
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

func TestDescribeProvider(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		providerInfo: &mockProviderInfoClient{
			DescribeProviderFunc: func(_ context.Context, in *pb.DescribeProviderRequest, _ ...grpc.CallOption) (*pb.DescribeProviderResponse, error) {
				return &pb.DescribeProviderResponse{
					Sections: []*pb.ProviderInfoSection{
						{
							Title: "Overview",
							Entries: []*pb.ProviderInfoEntry{
								{Key: "Name", Value: "otter"},
								{Key: "Type", Value: "in-memory"},
							},
						},
					},
				}, nil
			},
			ListSubResourcesFunc: func(_ context.Context, _ *pb.ListSubResourcesRequest, _ ...grpc.CallOption) (*pb.ListSubResourcesResponse, error) {
				return &pb.ListSubResourcesResponse{
					SubResourceName: "namespaces",
					Columns: []*pb.ProviderColumn{
						{Header: "NAME", Key: "name"},
						{Header: "ENTRIES", Key: "entries"},
					},
					Rows: []*pb.ProviderRow{
						{Name: "default", Values: map[string]string{"name": "default", "entries": "42"}},
					},
				}, nil
			},
		},
	}

	testCases := []struct {
		name      string
		format    string
		arguments []string
		wantAll   []string
	}{
		{
			name:      "text output",
			format:    "text",
			arguments: []string{"cache", "otter"},
			wantAll:   []string{"Overview", "Name", "otter", "in-memory", "Namespaces", "default"},
		},
		{
			name:      "json output",
			format:    "json",
			arguments: []string{"cache", "otter"},
			wantAll:   []string{`"title": "Overview"`, `"key": "Name"`, `"value": "otter"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := describeProvider(context.Background(), conn, p, tc.arguments)
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

func TestDescribeProviderMissingArgs(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "text", true, false)
	err := describeProvider(context.Background(), nil, p, []string{"cache"})
	if err == nil {
		t.Fatal("expected error for missing provider name, got nil")
	}
	if !strings.Contains(err.Error(), "missing resource type and provider name") {
		t.Errorf("error = %q, want containing 'missing resource type and provider name'", err.Error())
	}
}

func TestDescribeProviders(t *testing.T) {
	t.Parallel()

	t.Run("single argument calls DescribeResourceType", func(t *testing.T) {
		t.Parallel()

		conn := &mockConnection{
			providerInfo: &mockProviderInfoClient{
				DescribeResourceTypeFunc: func(_ context.Context, _ *pb.DescribeResourceTypeRequest, _ ...grpc.CallOption) (*pb.DescribeProviderResponse, error) {
					return &pb.DescribeProviderResponse{
						Sections: []*pb.ProviderInfoSection{
							{
								Title: "Cache Overview",
								Entries: []*pb.ProviderInfoEntry{
									{Key: "Provider Count", Value: "2"},
									{Key: "Default", Value: "otter"},
								},
							},
						},
					}, nil
				},
			},
		}

		var buffer bytes.Buffer
		p := NewPrinter(&buffer, "text", true, false)
		err := describeProviders(context.Background(), conn, p, []string{"cache"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buffer.String()
		for _, want := range []string{"Cache Overview", "Provider Count", "2", "Default", "otter"} {
			if !strings.Contains(output, want) {
				t.Errorf("output missing %q\nfull output:\n%s", want, output)
			}
		}
	})

	t.Run("two arguments delegates to describeProvider", func(t *testing.T) {
		t.Parallel()

		conn := &mockConnection{
			providerInfo: &mockProviderInfoClient{
				DescribeProviderFunc: func(_ context.Context, _ *pb.DescribeProviderRequest, _ ...grpc.CallOption) (*pb.DescribeProviderResponse, error) {
					return &pb.DescribeProviderResponse{
						Sections: []*pb.ProviderInfoSection{
							{
								Title: "Overview",
								Entries: []*pb.ProviderInfoEntry{
									{Key: "Name", Value: "otter"},
								},
							},
						},
					}, nil
				},
				ListSubResourcesFunc: func(_ context.Context, _ *pb.ListSubResourcesRequest, _ ...grpc.CallOption) (*pb.ListSubResourcesResponse, error) {
					return nil, errors.New("not supported")
				},
			},
		}

		var buffer bytes.Buffer
		p := NewPrinter(&buffer, "text", true, false)
		err := describeProviders(context.Background(), conn, p, []string{"cache", "otter"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buffer.String()
		if !strings.Contains(output, "otter") {
			t.Errorf("output missing 'otter'\nfull output:\n%s", output)
		}
	})

	t.Run("json output", func(t *testing.T) {
		t.Parallel()

		conn := &mockConnection{
			providerInfo: &mockProviderInfoClient{
				DescribeResourceTypeFunc: func(_ context.Context, _ *pb.DescribeResourceTypeRequest, _ ...grpc.CallOption) (*pb.DescribeProviderResponse, error) {
					return &pb.DescribeProviderResponse{
						Sections: []*pb.ProviderInfoSection{
							{
								Title: "Cache Overview",
								Entries: []*pb.ProviderInfoEntry{
									{Key: "Provider Count", Value: "2"},
								},
							},
						},
					}, nil
				},
			},
		}

		var buffer bytes.Buffer
		p := NewPrinter(&buffer, "json", true, false)
		err := describeProviders(context.Background(), conn, p, []string{"cache"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buffer.String()
		if !strings.Contains(output, `"title": "Cache Overview"`) {
			t.Errorf("output missing json title\nfull output:\n%s", output)
		}
	})
}

func TestDescribeProvidersMissingArgs(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "text", true, false)
	err := describeProviders(context.Background(), nil, p, nil)
	if err == nil {
		t.Fatal("expected error for missing resource type, got nil")
	}
	if !strings.Contains(err.Error(), "missing resource type") {
		t.Errorf("error = %q, want containing 'missing resource type'", err.Error())
	}
}

func TestRunDescribeMissingResource(t *testing.T) {
	t.Parallel()

	cc, _, _ := newTestCC(nil)
	err := runDescribe(context.Background(), cc, nil)
	if err == nil {
		t.Fatal("expected error for missing resource")
	}
	if !strings.Contains(err.Error(), "missing resource type") {
		t.Errorf("error = %q, want containing 'missing resource type'", err.Error())
	}
}

func TestRunDescribeUnknownResource(t *testing.T) {
	t.Parallel()

	cc, _, _ := newTestCC(nil)
	err := runDescribe(context.Background(), cc, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown resource")
	}
	if !strings.Contains(err.Error(), "unknown resource") {
		t.Errorf("error = %q, want containing 'unknown resource'", err.Error())
	}
}

func TestBuildSpanDetailSections(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		span         *pb.Span
		name         string
		wantContains []string
		wantSections int
	}{
		{
			name: "span with attributes",
			span: &pb.Span{
				SpanId: "abc123def456",
				Name:   "test-span",
				Attributes: map[string]string{
					"http.method": "GET",
					"http.url":    "https://example.com",
				},
			},
			wantSections: 1,
			wantContains: []string{"abc123def456", "test-span", "http.method", "GET", "http.url"},
		},
		{
			name: "span with events",
			span: &pb.Span{
				SpanId: "span-1",
				Name:   "root",
				Events: []*pb.SpanEvent{
					{
						Name:        "exception",
						TimestampMs: 1700000000,
						Attributes:  map[string]string{"error.type": "timeout"},
					},
				},
			},
			wantSections: 1,
			wantContains: []string{"Event: exception", "error.type", "timeout"},
		},
		{
			name: "span with attributes and events",
			span: &pb.Span{
				SpanId:     "span-2",
				Name:       "multi",
				Attributes: map[string]string{"key": "val"},
				Events: []*pb.SpanEvent{
					{Name: "log", TimestampMs: 1700000000, Attributes: map[string]string{"message": "hello"}},
				},
			},
			wantSections: 2,
			wantContains: []string{"key", "val", "Event: log", "message", "hello"},
		},
		{
			name:         "empty span",
			span:         &pb.Span{SpanId: "empty", Name: "noop"},
			wantSections: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewPrinter(&bytes.Buffer{}, "table", true, false)
			sections := buildSpanDetailSections(p, tc.span)
			if len(sections) != tc.wantSections {
				t.Errorf("got %d sections, want %d", len(sections), tc.wantSections)
			}

			var combined strings.Builder
			for _, s := range sections {
				combined.WriteString(s.Title)
				combined.WriteString(" ")
				for _, f := range s.Fields {
					combined.WriteString(f.Key)
					combined.WriteString(" ")
					combined.WriteString(f.Value)
					combined.WriteString(" ")
				}
			}
			output := combined.String()
			for _, want := range tc.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}
