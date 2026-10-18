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

package orchestrator_test_bench

import (
	"context"
	"testing"

	"piko.sh/piko/internal/logger/logger_domain"
)

func BenchmarkHotPathLog(b *testing.B) {
	const taskID = "0192f2a4-7a8e-7c3d-9b1e-3f6a2d4c8e10"
	for _, mode := range logModes {
		b.Run("log="+mode, func(b *testing.B) {
			applyLogMode(b, mode)
			logger := logger_domain.GetLogger("piko/internal/orchestrator/orchestrator_test_bench")
			ctx := context.Background()

			b.Run("op=from", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, _ = logger_domain.From(ctx, logger)
				}
			})
			b.Run("op=trace", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					logger.Trace("Task completed successfully",
						logger_domain.String("taskID", taskID),
						logger_domain.Int("attempt", 1))
				}
			})
			b.Run("op=span", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, span, _ := logger.Span(ctx, "TaskProcessingCore.handleTaskSuccess",
						logger_domain.String("taskID", taskID),
						logger_domain.String("workflowID", taskID))
					span.End()
				}
			})
		})
	}
}
