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

package worker_domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
)

func TestBuildSpec_ResolvesScheduleAndPriority(t *testing.T) {
	base := time.Date(2026, time.July, 8, 3, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(base)
	testService := &service{
		clk:         clk,
		idGenerator: uuidIDGenerator{clk: clk},
	}

	testCases := []struct {
		name         string
		req          worker_dto.EnqueueRequest
		wantAt       time.Time
		wantPriority int64
	}{
		{
			name: "immediate schedules at now with the default priority",
			req: worker_dto.EnqueueRequest{
				Kind: "k",
			},
			wantAt:       base,
			wantPriority: defaultEnqueuePriority,
		},
		{
			name: "relative delay schedules at now+delay",
			req: worker_dto.EnqueueRequest{
				Kind:  "k",
				Delay: 30 * time.Minute,
			},
			wantAt:       base.Add(30 * time.Minute),
			wantPriority: defaultEnqueuePriority,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			spec := testService.buildSpec(testCase.req)
			require.True(t, spec.ScheduledAt.Equal(testCase.wantAt), "ScheduledAt = %v, want %v", spec.ScheduledAt, testCase.wantAt)
			require.Equal(t, spec.Priority, testCase.wantPriority, "Priority")
		})
	}
}
