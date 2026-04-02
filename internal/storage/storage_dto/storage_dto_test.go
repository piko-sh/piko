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

package storage_dto

import (
	"strings"
	"testing"
	"time"
)

func TestBatchResult_HasErrors(t *testing.T) {
	r := &BatchResult{TotalFailed: 0}
	if r.HasErrors() {
		t.Error("HasErrors() should be false when TotalFailed is 0")
	}

	r.TotalFailed = 1
	if !r.HasErrors() {
		t.Error("HasErrors() should be true when TotalFailed > 0")
	}
}

func TestBatchResult_IsPartialSuccess(t *testing.T) {
	tests := []struct {
		name       string
		successful int
		failed     int
		want       bool
	}{
		{name: "all success", successful: 5, failed: 0, want: false},
		{name: "all failed", successful: 0, failed: 5, want: false},
		{name: "partial", successful: 3, failed: 2, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &BatchResult{TotalSuccessful: tt.successful, TotalFailed: tt.failed}
			if got := r.IsPartialSuccess(); got != tt.want {
				t.Errorf("IsPartialSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBatchResult_FormatSummary(t *testing.T) {
	allSuccess := &BatchResult{
		TotalRequested:  5,
		TotalSuccessful: 5,
		ProcessingTime:  100 * time.Millisecond,
	}
	if !strings.Contains(allSuccess.formatSummary(), "All 5 operations succeeded") {
		t.Errorf("unexpected summary: %s", allSuccess.formatSummary())
	}

	partial := &BatchResult{
		TotalRequested:  5,
		TotalSuccessful: 3,
		TotalFailed:     2,
		ProcessingTime:  200 * time.Millisecond,
	}
	s := partial.formatSummary()
	if !strings.Contains(s, "Partial success") || !strings.Contains(s, "3/5") {
		t.Errorf("unexpected partial summary: %s", s)
	}

	allFailed := &BatchResult{
		TotalRequested: 5,
		TotalFailed:    5,
		ProcessingTime: 50 * time.Millisecond,
	}
	if !strings.Contains(allFailed.formatSummary(), "All 5 operations failed") {
		t.Errorf("unexpected all-failed summary: %s", allFailed.formatSummary())
	}
}

func TestDefaultMultipartConfig(t *testing.T) {
	c := DefaultMultipartConfig()
	if c.PartSize != 100*1024*1024 {
		t.Errorf("PartSize = %d, want %d", c.PartSize, 100*1024*1024)
	}
	if c.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", c.Concurrency)
	}
	if !c.EnableChecksum {
		t.Error("EnableChecksum should be true")
	}
	if c.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", c.MaxRetries)
	}
}
