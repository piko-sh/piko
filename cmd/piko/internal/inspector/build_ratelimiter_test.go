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

package inspector

import (
	"testing"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestBuildRateLimiterDetailSectionsNil(t *testing.T) {
	t.Parallel()

	sections := BuildRateLimiterDetailSections(nil)
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2 (nil response should still emit two empty sections)", len(sections))
	}
	if sections[0].Heading != "Rate Limiter" {
		t.Errorf("sections[0].Heading = %q, want Rate Limiter", sections[0].Heading)
	}
	if sections[1].Heading != "Counters" {
		t.Errorf("sections[1].Heading = %q, want Counters", sections[1].Heading)
	}
}

func TestBuildRateLimiterDetailSectionsAllowRate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		response *pb.GetRateLimiterStatusResponse
		name     string
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
		{
			name: "all denied",
			response: &pb.GetRateLimiterStatusResponse{
				TotalAllowed: 0,
				TotalDenied:  50,
			},
			wantRate: "0.0% allowed",
		},
		{
			name: "all allowed",
			response: &pb.GetRateLimiterStatusResponse{
				TotalAllowed: 50,
				TotalDenied:  0,
			},
			wantRate: "100.0% allowed",
		},
		{
			name: "fractional rate",
			response: &pb.GetRateLimiterStatusResponse{
				TotalAllowed: 333,
				TotalDenied:  667,
			},
			wantRate: "33.3% allowed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sections := BuildRateLimiterDetailSections(tc.response)
			if len(sections) != 2 {
				t.Fatalf("got %d sections, want 2", len(sections))
			}

			counters := sections[1]
			var got string
			for _, row := range counters.Rows {
				if row.Label == "Allow Rate" {
					got = row.Value
				}
			}
			if got != tc.wantRate {
				t.Errorf("Allow Rate = %q, want %q", got, tc.wantRate)
			}
		})
	}
}

func TestBuildRateLimiterDetailSectionsConfiguration(t *testing.T) {
	t.Parallel()

	response := &pb.GetRateLimiterStatusResponse{
		TokenBucketStore: "memory",
		CounterStore:     "redis",
		FailPolicy:       "deny",
		KeyPrefix:        "tenant-1:",
		TotalChecks:      100,
		TotalAllowed:     90,
		TotalDenied:      10,
		TotalErrors:      0,
	}

	sections := BuildRateLimiterDetailSections(response)

	rateLimiter := sections[0]
	wantConfigOrder := []struct {
		label string
		value string
	}{
		{label: "Token Bucket Store", value: "memory"},
		{label: "Counter Store", value: "redis"},
		{label: "Fail Policy", value: "deny"},
		{label: "Key Prefix", value: "tenant-1:"},
	}
	if len(rateLimiter.Rows) != len(wantConfigOrder) {
		t.Fatalf("config rows = %d, want %d", len(rateLimiter.Rows), len(wantConfigOrder))
	}
	for index, expected := range wantConfigOrder {
		if rateLimiter.Rows[index].Label != expected.label {
			t.Errorf("config row %d label = %q, want %q", index, rateLimiter.Rows[index].Label, expected.label)
		}
		if rateLimiter.Rows[index].Value != expected.value {
			t.Errorf("config row %d value = %q, want %q", index, rateLimiter.Rows[index].Value, expected.value)
		}
	}

	counters := sections[1]
	wantCounters := map[string]string{
		"Total Checks":  "100",
		"Total Allowed": "90",
		"Total Denied":  "10",
		"Total Errors":  "0",
		"Allow Rate":    "90.0% allowed",
	}
	got := map[string]string{}
	for _, row := range counters.Rows {
		got[row.Label] = row.Value
	}
	for label, value := range wantCounters {
		if got[label] != value {
			t.Errorf("counter %q = %q, want %q", label, got[label], value)
		}
	}
}
