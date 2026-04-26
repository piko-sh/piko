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
	"testing"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestConvertMetrics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		response      *pb.GetMetricsResponse
		name          string
		wantLen       int
		wantNameCount int
	}{
		{
			name:          "nil metrics slice returns empty map",
			response:      &pb.GetMetricsResponse{},
			wantLen:       0,
			wantNameCount: 0,
		},
		{
			name: "single metric with data points",
			response: &pb.GetMetricsResponse{
				Metrics: []*pb.Metric{
					{
						Name:        "http_requests",
						Unit:        "requests",
						Description: "total HTTP requests",
						DataPoints: []*pb.MetricDataPoint{
							{
								TimestampMs: 1700000000000,
								Value:       42.5,
								Attributes:  map[string]string{"method": "GET"},
							},
							{
								TimestampMs: 1700000001000,
								Value:       43.0,
								Attributes:  nil,
							},
						},
					},
				},
			},
			wantLen:       1,
			wantNameCount: 1,
		},
		{
			name: "multiple metrics preserves insertion order in names",
			response: &pb.GetMetricsResponse{
				Metrics: []*pb.Metric{
					{Name: "beta_metric"},
					{Name: "alpha_metric"},
				},
			},
			wantLen:       2,
			wantNameCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			metrics, names := convertMetrics(tc.response)

			if got := len(metrics); got != tc.wantLen {
				t.Errorf("metric map length: got %d, want %d", got, tc.wantLen)
			}
			if got := len(names); got != tc.wantNameCount {
				t.Errorf("names length: got %d, want %d", got, tc.wantNameCount)
			}
		})
	}

	t.Run("data point fields are mapped correctly", func(t *testing.T) {
		t.Parallel()

		response := &pb.GetMetricsResponse{
			Metrics: []*pb.Metric{
				{
					Name:        "cpu_usage",
					Unit:        "percent",
					Description: "CPU utilisation",
					DataPoints: []*pb.MetricDataPoint{
						{
							TimestampMs: 1700000000000,
							Value:       85.5,
							Attributes:  map[string]string{"host": "node-1"},
						},
					},
				},
			},
		}

		metrics, names := convertMetrics(response)

		if names[0] != "cpu_usage" {
			t.Errorf("name: got %q, want %q", names[0], "cpu_usage")
		}

		series := metrics["cpu_usage"]
		if series.Unit != "percent" {
			t.Errorf("unit: got %q, want %q", series.Unit, "percent")
		}
		if series.Description != "CPU utilisation" {
			t.Errorf("description: got %q, want %q", series.Description, "CPU utilisation")
		}
		if len(series.Values) != 1 {
			t.Fatalf("values length: got %d, want 1", len(series.Values))
		}

		dp := series.Values[0]
		wantTS := time.UnixMilli(1700000000000)
		if !dp.Timestamp.Equal(wantTS) {
			t.Errorf("timestamp: got %v, want %v", dp.Timestamp, wantTS)
		}
		if dp.Value != 85.5 {
			t.Errorf("value: got %f, want 85.5", dp.Value)
		}
		if dp.Labels["host"] != "node-1" {
			t.Errorf("labels[host]: got %q, want %q", dp.Labels["host"], "node-1")
		}
	})
}

func TestConvertSpanStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  tui_domain.SpanStatus
	}{
		{name: "OK", input: "OK", want: tui_domain.SpanStatusOK},
		{name: "ERROR", input: "ERROR", want: tui_domain.SpanStatusError},
		{name: "empty string defaults to unset", input: "", want: tui_domain.SpanStatusUnset},
		{name: "unknown value defaults to unset", input: "PENDING", want: tui_domain.SpanStatusUnset},
		{name: "lowercase ok is unset", input: "ok", want: tui_domain.SpanStatusUnset},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := convertSpanStatus(tc.input)
			if got != tc.want {
				t.Errorf("convertSpanStatus(%q): got %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestConvertSpans(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns empty slices", func(t *testing.T) {
		t.Parallel()

		spans, errors := convertSpans(nil)
		if len(spans) != 0 {
			t.Errorf("spans: got %d, want 0", len(spans))
		}
		if len(errors) != 0 {
			t.Errorf("errors: got %d, want 0", len(errors))
		}
	})

	t.Run("error spans are separated", func(t *testing.T) {
		t.Parallel()

		input := []*pb.Span{
			{
				TraceId:       "trace-1",
				SpanId:        "span-1",
				ParentSpanId:  "",
				Name:          "GET /health",
				ServiceName:   "api",
				Status:        "OK",
				StatusMessage: "success",
				StartTimeMs:   1700000000000,
				DurationNs:    5000000,
				Attributes:    map[string]string{"path": "/health"},
			},
			{
				TraceId:       "trace-2",
				SpanId:        "span-2",
				ParentSpanId:  "span-1",
				Name:          "DB query",
				ServiceName:   "db",
				Status:        "ERROR",
				StatusMessage: "timeout",
				StartTimeMs:   1700000001000,
				DurationNs:    30000000000,
			},
			{
				TraceId: "trace-3",
				SpanId:  "span-3",
				Name:    "background",
				Status:  "UNSET",
			},
		}

		spans, errors := convertSpans(input)

		if len(spans) != 3 {
			t.Fatalf("spans: got %d, want 3", len(spans))
		}
		if len(errors) != 1 {
			t.Fatalf("errors: got %d, want 1", len(errors))
		}

		if errors[0].SpanID != "span-2" {
			t.Errorf("error span ID: got %q, want %q", errors[0].SpanID, "span-2")
		}

		s := spans[0]
		if s.TraceID != "trace-1" {
			t.Errorf("TraceID: got %q, want %q", s.TraceID, "trace-1")
		}
		if s.ParentID != "" {
			t.Errorf("ParentID: got %q, want empty", s.ParentID)
		}
		if s.Service != "api" {
			t.Errorf("Service: got %q, want %q", s.Service, "api")
		}
		if s.Status != tui_domain.SpanStatusOK {
			t.Errorf("Status: got %v, want SpanStatusOK", s.Status)
		}
		wantDuration := time.Duration(5000000)
		if s.Duration != wantDuration {
			t.Errorf("Duration: got %v, want %v", s.Duration, wantDuration)
		}
		if s.Attributes["path"] != "/health" {
			t.Errorf("Attributes[path]: got %q, want %q", s.Attributes["path"], "/health")
		}
		if s.Children != nil {
			t.Errorf("Children: got %v, want nil", s.Children)
		}
	})
}

func TestConvertTask(t *testing.T) {
	t.Parallel()

	t.Run("long ID is truncated to 8 characters", func(t *testing.T) {
		t.Parallel()

		task := &pb.TaskListItem{
			Id:         "abcdef1234567890",
			WorkflowId: "wf-001",
			Executor:   "image-resize",
			Status:     "COMPLETE",
			Priority:   2,
			Attempt:    3,
			CreatedAt:  1700000000,
			UpdatedAt:  1700000100,
		}

		got := convertTask(task)

		if got.Kind != kindOrchestratorTask {
			t.Errorf("Kind: got %q, want %q", got.Kind, kindOrchestratorTask)
		}
		if got.ID != "abcdef1234567890" {
			t.Errorf("ID should be full: got %q", got.ID)
		}
		wantName := "image-resize (abcdef12)"
		if got.Name != wantName {
			t.Errorf("Name: got %q, want %q", got.Name, wantName)
		}
		if got.Status != tui_domain.ResourceStatusHealthy {
			t.Errorf("Status: got %v, want Healthy", got.Status)
		}
		if got.Metadata[metadataKeyPriority] != "High" {
			t.Errorf("priority metadata: got %q, want %q", got.Metadata[metadataKeyPriority], "High")
		}
		if got.Metadata[metadataKeyAttempt] != "3" {
			t.Errorf("attempt metadata: got %q, want %q", got.Metadata[metadataKeyAttempt], "3")
		}
	})

	t.Run("short ID is not truncated", func(t *testing.T) {
		t.Parallel()

		task := &pb.TaskListItem{
			Id:       "abcd",
			Executor: "e",
			Status:   "PENDING",
		}

		got := convertTask(task)
		wantName := "e (abcd)"
		if got.Name != wantName {
			t.Errorf("Name: got %q, want %q", got.Name, wantName)
		}
	})
}

func TestConvertWorkflow(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		workflow       *pb.WorkflowSummary
		name           string
		wantStatusText string
		wantProgress   string
		wantStatus     tui_domain.ResourceStatus
	}{
		{
			name: "failed tasks set unhealthy status",
			workflow: &pb.WorkflowSummary{
				WorkflowId:    "wf-123456789012",
				TaskCount:     10,
				CompleteCount: 5,
				FailedCount:   2,
				ActiveCount:   3,
			},
			wantStatus:     tui_domain.ResourceStatusUnhealthy,
			wantStatusText: "FAILED",
			wantProgress:   "50%",
		},
		{
			name: "active tasks with no failures set pending",
			workflow: &pb.WorkflowSummary{
				WorkflowId:    "wf-active",
				TaskCount:     10,
				CompleteCount: 3,
				FailedCount:   0,
				ActiveCount:   7,
			},
			wantStatus:     tui_domain.ResourceStatusPending,
			wantStatusText: "ACTIVE",
			wantProgress:   "30%",
		},
		{
			name: "all complete sets healthy",
			workflow: &pb.WorkflowSummary{
				WorkflowId:    "wf-done",
				TaskCount:     5,
				CompleteCount: 5,
				FailedCount:   0,
				ActiveCount:   0,
			},
			wantStatus:     tui_domain.ResourceStatusHealthy,
			wantStatusText: "COMPLETE",
			wantProgress:   "100%",
		},
		{
			name: "zero tasks sets unknown with zero progress",
			workflow: &pb.WorkflowSummary{
				WorkflowId:    "wf-empty",
				TaskCount:     0,
				CompleteCount: 0,
				FailedCount:   0,
				ActiveCount:   0,
			},
			wantStatus:     tui_domain.ResourceStatusHealthy,
			wantStatusText: "COMPLETE",
			wantProgress:   "0%",
		},
		{
			name: "no complete no active no failed defaults to unknown",
			workflow: &pb.WorkflowSummary{
				WorkflowId:    "wf-mystery",
				TaskCount:     5,
				CompleteCount: 0,
				FailedCount:   0,
				ActiveCount:   0,
			},
			wantStatus:     tui_domain.ResourceStatusUnknown,
			wantStatusText: "UNKNOWN",
			wantProgress:   "0%",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := convertWorkflow(tc.workflow)

			if got.Status != tc.wantStatus {
				t.Errorf("Status: got %v, want %v", got.Status, tc.wantStatus)
			}
			if got.StatusText != tc.wantStatusText {
				t.Errorf("StatusText: got %q, want %q", got.StatusText, tc.wantStatusText)
			}
			if got.Metadata[metadataKeyProgress] != tc.wantProgress {
				t.Errorf("progress: got %q, want %q", got.Metadata[metadataKeyProgress], tc.wantProgress)
			}
			if got.Kind != kindOrchestratorWorkflow {
				t.Errorf("Kind: got %q, want %q", got.Kind, kindOrchestratorWorkflow)
			}
		})
	}

	t.Run("long workflow ID is truncated with ellipsis", func(t *testing.T) {
		t.Parallel()

		wf := &pb.WorkflowSummary{
			WorkflowId: "abcdefghijklmnop",
			TaskCount:  1,
		}

		got := convertWorkflow(wf)
		wantName := "abcdefgh..."
		if got.Name != wantName {
			t.Errorf("Name: got %q, want %q", got.Name, wantName)
		}

		if got.ID != "abcdefghijklmnop" {
			t.Errorf("ID: got %q, want full ID", got.ID)
		}
	})
}

func TestConvertArtefact(t *testing.T) {
	t.Parallel()

	t.Run("uses source path as name when available", func(t *testing.T) {
		t.Parallel()

		artefact := &pb.ArtefactListItem{
			Id:           "art-001",
			SourcePath:   "/app/main.go",
			Status:       "READY",
			VariantCount: 3,
			TotalSize:    2048,
		}

		got := convertArtefact(artefact)

		if got.Name != "/app/main.go" {
			t.Errorf("Name: got %q, want %q", got.Name, "/app/main.go")
		}
		if got.Status != tui_domain.ResourceStatusHealthy {
			t.Errorf("Status: got %v, want Healthy", got.Status)
		}
		if got.Metadata[metadataKeyVariantCount] != "3" {
			t.Errorf("variant_count: got %q, want %q", got.Metadata[metadataKeyVariantCount], "3")
		}
	})

	t.Run("falls back to ID as name when source path is empty", func(t *testing.T) {
		t.Parallel()

		artefact := &pb.ArtefactListItem{
			Id:     "art-fallback",
			Status: "STALE",
		}

		got := convertArtefact(artefact)

		if got.Name != "art-fallback" {
			t.Errorf("Name: got %q, want %q", got.Name, "art-fallback")
		}
		if got.Status != tui_domain.ResourceStatusDegraded {
			t.Errorf("Status: got %v, want Degraded", got.Status)
		}
	})
}

func TestMapTaskStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  tui_domain.ResourceStatus
	}{
		{name: "COMPLETE", input: "COMPLETE", want: tui_domain.ResourceStatusHealthy},
		{name: "FAILED", input: "FAILED", want: tui_domain.ResourceStatusUnhealthy},
		{name: "PROCESSING", input: "PROCESSING", want: tui_domain.ResourceStatusDegraded},
		{name: "PENDING", input: "PENDING", want: tui_domain.ResourceStatusPending},
		{name: "SCHEDULED", input: "SCHEDULED", want: tui_domain.ResourceStatusPending},
		{name: "RETRYING", input: "RETRYING", want: tui_domain.ResourceStatusPending},
		{name: "empty string", input: "", want: tui_domain.ResourceStatusUnknown},
		{name: "unknown value", input: "CANCELLED", want: tui_domain.ResourceStatusUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := mapTaskStatus(tc.input)
			if got != tc.want {
				t.Errorf("mapTaskStatus(%q): got %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestMapArtefactStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  tui_domain.ResourceStatus
	}{
		{name: "READY", input: "READY", want: tui_domain.ResourceStatusHealthy},
		{name: "STALE", input: "STALE", want: tui_domain.ResourceStatusDegraded},
		{name: "PENDING", input: "PENDING", want: tui_domain.ResourceStatusPending},
		{name: "empty string", input: "", want: tui_domain.ResourceStatusUnknown},
		{name: "unknown value", input: "DELETED", want: tui_domain.ResourceStatusUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := mapArtefactStatus(tc.input)
			if got != tc.want {
				t.Errorf("mapArtefactStatus(%q): got %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestMapPriority(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		want  string
		input int32
	}{
		{name: "low", input: 0, want: "Low"},
		{name: "normal", input: 1, want: "Normal"},
		{name: "high", input: 2, want: "High"},
		{name: "custom P5", input: 5, want: "P5"},
		{name: "negative", input: -1, want: "P-1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := mapPriority(tc.input)
			if got != tc.want {
				t.Errorf("mapPriority(%d): got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestConvertFDsData(t *testing.T) {
	t.Parallel()

	t.Run("empty response returns empty categories", func(t *testing.T) {
		t.Parallel()

		response := &pb.GetFileDescriptorsResponse{
			Total:       0,
			TimestampMs: 1700000000000,
		}

		got := convertFDsData(response)

		if len(got.Categories) != 0 {
			t.Errorf("categories length: got %d, want 0", len(got.Categories))
		}
		if got.Total != 0 {
			t.Errorf("total: got %d, want 0", got.Total)
		}
		if got.Timestamp != 1700000000000 {
			t.Errorf("timestamp: got %d, want 1700000000000", got.Timestamp)
		}
	})

	t.Run("categories and FDs are mapped correctly", func(t *testing.T) {
		t.Parallel()

		response := &pb.GetFileDescriptorsResponse{
			Categories: []*pb.FileDescriptorCategory{
				{
					Category: "socket",
					Count:    2,
					Fds: []*pb.FileDescriptorInfo{
						{
							Fd:          3,
							Category:    "socket",
							Target:      "127.0.0.1:8080",
							FirstSeenMs: 1700000000000,
							AgeMs:       5000,
						},
						{
							Fd:       4,
							Category: "socket",
							Target:   "127.0.0.1:9090",
						},
					},
				},
			},
			Total:       2,
			TimestampMs: 1700000000000,
		}

		got := convertFDsData(response)

		if len(got.Categories) != 1 {
			t.Fatalf("categories length: got %d, want 1", len(got.Categories))
		}

		cat := got.Categories[0]
		if cat.Category != "socket" {
			t.Errorf("category name: got %q, want %q", cat.Category, "socket")
		}
		if cat.Count != 2 {
			t.Errorf("category count: got %d, want 2", cat.Count)
		}
		if len(cat.FDs) != 2 {
			t.Fatalf("FDs length: got %d, want 2", len(cat.FDs))
		}

		fd := cat.FDs[0]
		if fd.FD != 3 {
			t.Errorf("FD number: got %d, want 3", fd.FD)
		}
		if fd.Target != "127.0.0.1:8080" {
			t.Errorf("target: got %q, want %q", fd.Target, "127.0.0.1:8080")
		}
		if fd.AgeMs != 5000 {
			t.Errorf("age: got %d, want 5000", fd.AgeMs)
		}
	})
}

func TestParseProtoHealthStatus(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns unknown status", func(t *testing.T) {
		t.Parallel()

		got := parseProtoHealthStatus(nil)

		if got.Name != "unknown" {
			t.Errorf("Name: got %q, want %q", got.Name, "unknown")
		}
		if got.State != tui_domain.HealthStateUnknown {
			t.Errorf("State: got %v, want Unknown", got.State)
		}
	})

	t.Run("state string mapping", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			state string
			want  tui_domain.HealthState
		}{
			{name: "HEALTHY", state: "HEALTHY", want: tui_domain.HealthStateHealthy},
			{name: "ok", state: "ok", want: tui_domain.HealthStateHealthy},
			{name: "DEGRADED", state: "DEGRADED", want: tui_domain.HealthStateDegraded},
			{name: "UNHEALTHY", state: "UNHEALTHY", want: tui_domain.HealthStateUnhealthy},
			{name: "empty string", state: "", want: tui_domain.HealthStateUnknown},
			{name: "unexpected value", state: "PENDING", want: tui_domain.HealthStateUnknown},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got := parseProtoHealthStatus(&pb.HealthStatus{
					State: tc.state,
				})

				if got.State != tc.want {
					t.Errorf("State for %q: got %v, want %v", tc.state, got.State, tc.want)
				}
			})
		}
	})

	t.Run("duration is parsed correctly", func(t *testing.T) {
		t.Parallel()

		got := parseProtoHealthStatus(&pb.HealthStatus{
			Name:     "db",
			State:    "HEALTHY",
			Duration: "250ms",
		})

		want := 250 * time.Millisecond
		if got.Duration != want {
			t.Errorf("Duration: got %v, want %v", got.Duration, want)
		}
	})

	t.Run("invalid duration is zero", func(t *testing.T) {
		t.Parallel()

		got := parseProtoHealthStatus(&pb.HealthStatus{
			Duration: "not-a-duration",
		})

		if got.Duration != 0 {
			t.Errorf("Duration: got %v, want 0", got.Duration)
		}
	})

	t.Run("empty duration is zero", func(t *testing.T) {
		t.Parallel()

		got := parseProtoHealthStatus(&pb.HealthStatus{
			Duration: "",
		})

		if got.Duration != 0 {
			t.Errorf("Duration: got %v, want 0", got.Duration)
		}
	})

	t.Run("dependencies are converted recursively", func(t *testing.T) {
		t.Parallel()

		got := parseProtoHealthStatus(&pb.HealthStatus{
			Name:  "app",
			State: "DEGRADED",
			Dependencies: []*pb.HealthStatus{
				{
					Name:  "database",
					State: "UNHEALTHY",
					Dependencies: []*pb.HealthStatus{
						{
							Name:  "primary",
							State: "UNHEALTHY",
						},
					},
				},
				{
					Name:  "cache",
					State: "HEALTHY",
				},
			},
		})

		if len(got.Dependencies) != 2 {
			t.Fatalf("top-level deps: got %d, want 2", len(got.Dependencies))
		}

		dbDep := got.Dependencies[0]
		if dbDep.Name != "database" {
			t.Errorf("dependency name: got %q, want %q", dbDep.Name, "database")
		}
		if dbDep.State != tui_domain.HealthStateUnhealthy {
			t.Errorf("dependency state: got %v, want Unhealthy", dbDep.State)
		}
		if len(dbDep.Dependencies) != 1 {
			t.Fatalf("nested deps: got %d, want 1", len(dbDep.Dependencies))
		}
		if dbDep.Dependencies[0].Name != "primary" {
			t.Errorf("nested dependency name: got %q, want %q", dbDep.Dependencies[0].Name, "primary")
		}
	})

	t.Run("fields are mapped from proto", func(t *testing.T) {
		t.Parallel()

		got := parseProtoHealthStatus(&pb.HealthStatus{
			Name:        "cache",
			State:       "HEALTHY",
			Message:     "all good",
			TimestampMs: 1700000000000,
		})

		if got.Name != "cache" {
			t.Errorf("Name: got %q, want %q", got.Name, "cache")
		}
		if got.Message != "all good" {
			t.Errorf("Message: got %q, want %q", got.Message, "all good")
		}
		wantTS := time.UnixMilli(1700000000000)
		if !got.Timestamp.Equal(wantTS) {
			t.Errorf("Timestamp: got %v, want %v", got.Timestamp, wantTS)
		}
	})
}

func TestConvertSystemStats(t *testing.T) {
	t.Parallel()

	t.Run("nil response returns nil", func(t *testing.T) {
		t.Parallel()

		got := convertSystemStats(nil)
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("full response maps all fields", func(t *testing.T) {
		t.Parallel()

		response := &pb.GetSystemStatsResponse{
			TimestampMs:   1700000000000,
			UptimeMs:      3600000,
			NumCpu:        8,
			Gomaxprocs:    4,
			NumGoroutines: 50,
			NumCgoCalls:   100,
			CpuMillicores: 1500.5,
			Memory: &pb.MemoryInfo{
				Alloc:       1024,
				TotalAlloc:  2048,
				Sys:         4096,
				HeapAlloc:   512,
				HeapSys:     2048,
				HeapIdle:    1024,
				HeapInuse:   512,
				HeapObjects: 100,
			},
			Gc: &pb.GCInfo{
				NumGc:         42,
				LastGcNs:      1700000000000000,
				PauseTotalNs:  500000,
				LastPauseNs:   1000,
				NextGc:        8192,
				GcCpuFraction: 0.02,
				RecentPauses:  []uint64{100, 200, 300},
			},
			Build: &pb.BuildInfo{
				GoVersion: "go1.22.0",
				Version:   "1.0.0",
				Commit:    "abc123",
				BuildTime: "2024-01-01T00:00:00Z",
				Os:        "linux",
				Arch:      "amd64",
			},
			Runtime: &pb.RuntimeInfo{
				Gogc:       "100",
				Gomemlimit: "1GiB",
			},
			Process: &pb.ProcessInfo{
				Pid:         12345,
				ThreadCount: 10,
				FdCount:     42,
				Rss:         1048576,
			},
			Cache: &pb.CacheInfo{
				ComponentCacheSize: 150,
				SvgCacheSize:       25,
			},
		}

		got := convertSystemStats(response)

		if got == nil {
			t.Fatal("got nil, want non-nil")
		}

		wantTS := time.UnixMilli(1700000000000)
		if !got.Timestamp.Equal(wantTS) {
			t.Errorf("Timestamp: got %v, want %v", got.Timestamp, wantTS)
		}
		wantUptime := 3600000 * time.Millisecond
		if got.Uptime != wantUptime {
			t.Errorf("Uptime: got %v, want %v", got.Uptime, wantUptime)
		}
		if got.NumCPU != 8 {
			t.Errorf("NumCPU: got %d, want 8", got.NumCPU)
		}
		if got.GOMAXPROCS != 4 {
			t.Errorf("GOMAXPROCS: got %d, want 4", got.GOMAXPROCS)
		}
		if got.NumGoroutines != 50 {
			t.Errorf("NumGoroutines: got %d, want 50", got.NumGoroutines)
		}
		if got.NumCGOCalls != 100 {
			t.Errorf("NumCGOCalls: got %d, want 100", got.NumCGOCalls)
		}
		if got.CPUMillicores != 1500.5 {
			t.Errorf("CPUMillicores: got %f, want 1500.5", got.CPUMillicores)
		}
		if got.Cache.ComponentCacheSize != 150 {
			t.Errorf("Cache.ComponentCacheSize: got %d, want 150", got.Cache.ComponentCacheSize)
		}
		if got.Cache.SVGCacheSize != 25 {
			t.Errorf("Cache.SVGCacheSize: got %d, want 25", got.Cache.SVGCacheSize)
		}
	})
}

func TestConvertMemoryStats(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns zero values", func(t *testing.T) {
		t.Parallel()

		got := convertMemoryStats(nil)

		if got.Alloc != 0 || got.Sys != 0 || got.HeapAlloc != 0 {
			t.Errorf("expected zero values for nil input, got %+v", got)
		}
	})

	t.Run("fields are mapped", func(t *testing.T) {
		t.Parallel()

		mem := &pb.MemoryInfo{
			Alloc:        1000,
			TotalAlloc:   2000,
			Sys:          3000,
			HeapAlloc:    400,
			HeapSys:      500,
			HeapIdle:     600,
			HeapInuse:    700,
			HeapObjects:  80,
			HeapReleased: 90,
			StackSys:     100,
			Mallocs:      200,
			Frees:        150,
			LiveObjects:  50,
		}

		got := convertMemoryStats(mem)

		if got.Alloc != 1000 {
			t.Errorf("Alloc: got %d, want 1000", got.Alloc)
		}
		if got.HeapObjects != 80 {
			t.Errorf("HeapObjects: got %d, want 80", got.HeapObjects)
		}
		if got.LiveObjects != 50 {
			t.Errorf("LiveObjects: got %d, want 50", got.LiveObjects)
		}
	})
}

func TestConvertGCStats(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns zero values", func(t *testing.T) {
		t.Parallel()

		got := convertGCStats(nil)
		if got.NumGC != 0 || got.GCCPUFraction != 0 {
			t.Errorf("expected zero values for nil input, got %+v", got)
		}
	})

	t.Run("fields are mapped", func(t *testing.T) {
		t.Parallel()

		gc := &pb.GCInfo{
			NumGc:         10,
			LastGcNs:      999,
			PauseTotalNs:  500,
			LastPauseNs:   100,
			NextGc:        4096,
			GcCpuFraction: 0.05,
			RecentPauses:  []uint64{10, 20, 30},
		}

		got := convertGCStats(gc)

		if got.NumGC != 10 {
			t.Errorf("NumGC: got %d, want 10", got.NumGC)
		}
		if got.GCCPUFraction != 0.05 {
			t.Errorf("GCCPUFraction: got %f, want 0.05", got.GCCPUFraction)
		}
		if len(got.RecentPauses) != 3 {
			t.Errorf("RecentPauses length: got %d, want 3", len(got.RecentPauses))
		}
	})
}

func TestConvertBuildInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns zero values", func(t *testing.T) {
		t.Parallel()

		got := convertBuildInfo(nil)
		if got.GoVersion != "" || got.Version != "" {
			t.Errorf("expected zero values for nil input, got %+v", got)
		}
	})

	t.Run("fields are mapped", func(t *testing.T) {
		t.Parallel()

		build := &pb.BuildInfo{
			GoVersion: "go1.22.0",
			Version:   "2.0.0",
			Commit:    "deadbeef",
			BuildTime: "2024-06-15T10:00:00Z",
			Os:        "darwin",
			Arch:      "arm64",
		}

		got := convertBuildInfo(build)

		if got.GoVersion != "go1.22.0" {
			t.Errorf("GoVersion: got %q, want %q", got.GoVersion, "go1.22.0")
		}
		if got.OS != "darwin" {
			t.Errorf("OS: got %q, want %q", got.OS, "darwin")
		}
		if got.Arch != "arm64" {
			t.Errorf("Arch: got %q, want %q", got.Arch, "arm64")
		}
	})
}

func TestConvertRuntimeConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns zero values", func(t *testing.T) {
		t.Parallel()

		got := convertRuntimeConfig(nil)
		if got.GOGC != "" || got.GOMEMLIMIT != "" {
			t.Errorf("expected zero values for nil input, got %+v", got)
		}
	})

	t.Run("fields are mapped", func(t *testing.T) {
		t.Parallel()

		runtime := &pb.RuntimeInfo{
			Gogc:       "200",
			Gomemlimit: "2GiB",
		}

		got := convertRuntimeConfig(runtime)

		if got.GOGC != "200" {
			t.Errorf("GOGC: got %q, want %q", got.GOGC, "200")
		}
		if got.GOMEMLIMIT != "2GiB" {
			t.Errorf("GOMEMLIMIT: got %q, want %q", got.GOMEMLIMIT, "2GiB")
		}
	})
}

func TestConvertCacheStats(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns zero values", func(t *testing.T) {
		t.Parallel()

		got := convertCacheStats(nil)
		if got.ComponentCacheSize != 0 || got.SVGCacheSize != 0 {
			t.Errorf("expected zero values for nil input, got %+v", got)
		}
	})

	t.Run("populated input maps all fields", func(t *testing.T) {
		t.Parallel()

		cache := &pb.CacheInfo{
			ComponentCacheSize: 42,
			SvgCacheSize:       7,
		}

		got := convertCacheStats(cache)

		if got.ComponentCacheSize != 42 {
			t.Errorf("ComponentCacheSize: got %d, want 42", got.ComponentCacheSize)
		}
		if got.SVGCacheSize != 7 {
			t.Errorf("SVGCacheSize: got %d, want 7", got.SVGCacheSize)
		}
	})
}

func TestConvertProcessInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns zero values", func(t *testing.T) {
		t.Parallel()

		got := convertProcessInfo(nil)
		if got.PID != 0 || got.ThreadCount != 0 || got.FDCount != 0 {
			t.Errorf("expected zero values for nil input, got %+v", got)
		}
	})

	t.Run("int32/int64 to int conversion", func(t *testing.T) {
		t.Parallel()

		process := &pb.ProcessInfo{
			Pid:         54321,
			ThreadCount: 8,
			FdCount:     128,
			Rss:         2097152,
		}

		got := convertProcessInfo(process)

		if got.PID != 54321 {
			t.Errorf("PID: got %d, want 54321", got.PID)
		}
		if got.ThreadCount != 8 {
			t.Errorf("ThreadCount: got %d, want 8", got.ThreadCount)
		}
		if got.FDCount != 128 {
			t.Errorf("FDCount: got %d, want 128", got.FDCount)
		}
		if got.RSS != 2097152 {
			t.Errorf("RSS: got %d, want 2097152", got.RSS)
		}
	})
}
