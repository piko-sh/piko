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

package bootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogProfilingEndpoints(t *testing.T) {
	testCases := []struct {
		name                string
		rollingTraceEnabled bool
	}{
		{name: "without the rolling trace endpoint"},
		{name: "with the rolling trace endpoint", rollingTraceEnabled: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				logProfilingEndpoints(context.Background(), "http://127.0.0.1:6060/_piko/profiler", "127.0.0.1:6060", tc.rollingTraceEnabled)
			})
		})
	}
}
