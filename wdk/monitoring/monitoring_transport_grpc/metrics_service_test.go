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
	"testing"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestMetricsService_GetMetrics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		telemetry     TelemetryProvider
		name          string
		expectedCount int
		expectedNil   bool
	}{
		{
			name:          "nil telemetry provider returns empty",
			telemetry:     nil,
			expectedCount: 0,
			expectedNil:   true,
		},
		{
			name: "returns metrics from provider",
			telemetry: &monitoring_domain.MockTelemetryProvider{
				GetMetricsFunc: func() []monitoring_domain.MetricData {
					return []monitoring_domain.MetricData{
						{
							Name:        "test.metric",
							Description: "A test metric",
							Unit:        "count",
							Type:        "counter",
							DataPoints: []monitoring_domain.MetricDataPoint{
								{TimestampMs: 1000, Value: 42.0},
							},
						},
					}
				},
			},
			expectedCount: 1,
			expectedNil:   false,
		},
		{
			name: "returns multiple metrics",
			telemetry: &monitoring_domain.MockTelemetryProvider{
				GetMetricsFunc: func() []monitoring_domain.MetricData {
					return []monitoring_domain.MetricData{
						{Name: "metric1"},
						{Name: "metric2"},
						{Name: "metric3"},
					}
				},
			},
			expectedCount: 3,
			expectedNil:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewMetricsService(tc.telemetry, nil, nil, nil)

			response, err := service.GetMetrics(context.Background(), &pb.GetMetricsRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectedNil && response.Metrics != nil {
				t.Errorf("expected nil metrics, got %v", response.Metrics)
			}

			if !tc.expectedNil && len(response.Metrics) != tc.expectedCount {
				t.Errorf("expected %d metrics, got %d", tc.expectedCount, len(response.Metrics))
			}

			if response.TimestampMs <= 0 {
				t.Error("expected positive timestamp")
			}
		})
	}
}

func TestMetricsService_GetTraces(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		telemetry     TelemetryProvider
		request       *pb.GetTracesRequest
		name          string
		expectedCount int
	}{
		{
			name:          "nil telemetry provider returns empty",
			telemetry:     nil,
			request:       &pb.GetTracesRequest{},
			expectedCount: 0,
		},
		{
			name: "returns spans from provider",
			telemetry: &monitoring_domain.MockTelemetryProvider{
				GetSpansFunc: func(_ int, _ bool) []monitoring_domain.SpanData {
					return []monitoring_domain.SpanData{
						{TraceID: "trace-1", SpanID: "span-1", Name: "test-span"},
					}
				},
			},
			request:       &pb.GetTracesRequest{},
			expectedCount: 1,
		},
		{
			name: "returns spans by trace ID",
			telemetry: &monitoring_domain.MockTelemetryProvider{
				GetSpanByTraceIDFunc: func(traceID string) []monitoring_domain.SpanData {
					m := map[string][]monitoring_domain.SpanData{
						"specific-trace": {
							{TraceID: "specific-trace", SpanID: "span-1"},
							{TraceID: "specific-trace", SpanID: "span-2"},
						},
					}
					return m[traceID]
				},
			},
			request:       &pb.GetTracesRequest{TraceId: "specific-trace"},
			expectedCount: 2,
		},
		{
			name: "returns spans with events",
			telemetry: &monitoring_domain.MockTelemetryProvider{
				GetSpansFunc: func(_ int, _ bool) []monitoring_domain.SpanData {
					return []monitoring_domain.SpanData{
						{
							TraceID: "trace-events",
							SpanID:  "span-events",
							Name:    "span-with-events",
							Events: []monitoring_domain.SpanEvent{
								{
									Name:        "event1",
									TimestampMs: 5000,
									Attributes:  map[string]string{"key": "val"},
								},
								{
									Name:        "event2",
									TimestampMs: 6000,
								},
							},
						},
					}
				},
			},
			request:       &pb.GetTracesRequest{},
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewMetricsService(tc.telemetry, nil, nil, nil)

			response, err := service.GetTraces(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(response.Spans) != tc.expectedCount {
				t.Errorf("expected %d spans, got %d", tc.expectedCount, len(response.Spans))
			}

			if response.TimestampMs <= 0 {
				t.Error("expected positive timestamp")
			}
		})
	}
}

func TestMetricsService_GetTraces_SpanWithEventsFields(t *testing.T) {
	t.Parallel()

	telemetry := &monitoring_domain.MockTelemetryProvider{
		GetSpansFunc: func(_ int, _ bool) []monitoring_domain.SpanData {
			return []monitoring_domain.SpanData{
				{
					TraceID:       "t1",
					SpanID:        "s1",
					ParentSpanID:  "p1",
					Name:          "my-span",
					Kind:          "server",
					Status:        "OK",
					StatusMessage: "",
					ServiceName:   "my-service",
					StartTimeMs:   1000,
					EndTimeMs:     2000,
					DurationNs:    1000000000,
					Attributes:    map[string]string{"http.method": "GET"},
					Events: []monitoring_domain.SpanEvent{
						{
							Name:        "exception",
							TimestampMs: 1500,
							Attributes:  map[string]string{"exception.message": "timeout"},
						},
					},
				},
			}
		},
	}

	service := NewMetricsService(telemetry, nil, nil, nil)
	response, err := service.GetTraces(context.Background(), &pb.GetTracesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(response.Spans))
	}

	span := response.Spans[0]
	if span.TraceId != "t1" {
		t.Errorf("expected TraceId t1, got %s", span.TraceId)
	}
	if span.SpanId != "s1" {
		t.Errorf("expected SpanId s1, got %s", span.SpanId)
	}
	if span.ParentSpanId != "p1" {
		t.Errorf("expected ParentSpanId p1, got %s", span.ParentSpanId)
	}
	if span.Name != "my-span" {
		t.Errorf("expected Name my-span, got %s", span.Name)
	}
	if span.Kind != "server" {
		t.Errorf("expected Kind server, got %s", span.Kind)
	}
	if span.ServiceName != "my-service" {
		t.Errorf("expected ServiceName my-service, got %s", span.ServiceName)
	}
	if span.StartTimeMs != 1000 {
		t.Errorf("expected StartTimeMs 1000, got %d", span.StartTimeMs)
	}
	if span.EndTimeMs != 2000 {
		t.Errorf("expected EndTimeMs 2000, got %d", span.EndTimeMs)
	}
	if span.DurationNs != 1000000000 {
		t.Errorf("expected DurationNs 1000000000, got %d", span.DurationNs)
	}

	if len(span.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(span.Events))
	}
	if span.Events[0].Name != "exception" {
		t.Errorf("expected event name 'exception', got %s", span.Events[0].Name)
	}
	if span.Events[0].TimestampMs != 1500 {
		t.Errorf("expected event timestamp 1500, got %d", span.Events[0].TimestampMs)
	}
	if span.Events[0].Attributes["exception.message"] != "timeout" {
		t.Errorf("expected exception.message=timeout, got %s", span.Events[0].Attributes["exception.message"])
	}
}

func TestMetricsService_GetSystemStats(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		systemProvider   SystemStatsProvider
		name             string
		expectEmptyBuild bool
	}{
		{
			name:             "nil system provider returns empty response",
			systemProvider:   nil,
			expectEmptyBuild: true,
		},
		{
			name: "returns stats from provider",
			systemProvider: &monitoring_domain.MockSystemStatsProvider{
				GetStatsFunc: func() monitoring_domain.SystemStats {
					return monitoring_domain.SystemStats{
						TimestampMs:   1234567890,
						UptimeMs:      3600000,
						NumCPU:        8,
						GOMAXPROCS:    8,
						NumGoroutines: 50,
						Build: monitoring_domain.BuildInfo{
							GoVersion: "go1.23.0",
							Version:   "1.0.0",
						},
						Memory: monitoring_domain.MemoryInfo{
							Alloc:     1024 * 1024,
							HeapAlloc: 512 * 1024,
						},
					}
				},
			},
			expectEmptyBuild: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewMetricsService(nil, tc.systemProvider, nil, nil)

			response, err := service.GetSystemStats(context.Background(), &pb.GetSystemStatsRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectEmptyBuild {
				if response.Build != nil {
					t.Error("expected nil Build for empty response")
				}
			} else {
				if response.Build == nil {
					t.Error("expected non-nil Build")
				}
				if response.Build.GoVersion != "go1.23.0" {
					t.Errorf("expected GoVersion go1.23.0, got %s", response.Build.GoVersion)
				}
				if response.NumCpu != 8 {
					t.Errorf("expected NumCpu 8, got %d", response.NumCpu)
				}
			}
		})
	}
}

func TestMetricsService_GetSystemStats_CacheStats(t *testing.T) {
	t.Parallel()

	t.Run("nil cache stats provider omits cache field", func(t *testing.T) {
		t.Parallel()

		systemProvider := &monitoring_domain.MockSystemStatsProvider{
			GetStatsFunc: func() monitoring_domain.SystemStats {
				return monitoring_domain.SystemStats{
					NumCPU: 4,
				}
			},
		}
		service := NewMetricsService(nil, systemProvider, nil, nil)

		response, err := service.GetSystemStats(context.Background(), &pb.GetSystemStatsRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if response.Cache != nil {
			t.Errorf("expected nil Cache when provider is nil, got %+v", response.Cache)
		}
	})

	t.Run("cache stats provider populates cache field", func(t *testing.T) {
		t.Parallel()

		systemProvider := &monitoring_domain.MockSystemStatsProvider{
			GetStatsFunc: func() monitoring_domain.SystemStats {
				return monitoring_domain.SystemStats{
					NumCPU: 4,
				}
			},
		}
		cacheProvider := &mockRenderCacheStatsProvider{
			componentCacheSize: 42,
			svgCacheSize:       7,
		}
		service := NewMetricsService(nil, systemProvider, nil, cacheProvider)

		response, err := service.GetSystemStats(context.Background(), &pb.GetSystemStatsRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if response.Cache == nil {
			t.Fatal("expected non-nil Cache")
		}
		if response.Cache.ComponentCacheSize != 42 {
			t.Errorf("expected ComponentCacheSize 42, got %d", response.Cache.ComponentCacheSize)
		}
		if response.Cache.SvgCacheSize != 7 {
			t.Errorf("expected SvgCacheSize 7, got %d", response.Cache.SvgCacheSize)
		}
	})
}

func TestMetricsService_GetFileDescriptors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		fdProvider    ResourceProvider
		name          string
		expectedTotal int32
	}{
		{
			name:          "nil provider returns empty",
			fdProvider:    nil,
			expectedTotal: 0,
		},
		{
			name: "returns data from provider",
			fdProvider: &monitoring_domain.MockResourceProvider{
				GetResourcesFunc: func() monitoring_domain.ResourceData {
					return monitoring_domain.ResourceData{
						Total:       15,
						TimestampMs: 1234567890,
						Categories: []monitoring_domain.ResourceCategory{
							{
								Category: "file",
								Count:    10,
								Resources: []monitoring_domain.ResourceInfo{
									{FD: 3, Category: "file", Target: "/tmp/test"},
								},
							},
							{
								Category: "tcp",
								Count:    5,
							},
						},
					}
				},
			},
			expectedTotal: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewMetricsService(nil, nil, tc.fdProvider, nil)

			response, err := service.GetFileDescriptors(context.Background(), &pb.GetFileDescriptorsRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if response.Total != tc.expectedTotal {
				t.Errorf("expected total %d, got %d", tc.expectedTotal, response.Total)
			}

			if response.TimestampMs <= 0 {
				t.Error("expected positive timestamp")
			}
		})
	}
}
