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

package daemon_domain

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// MockAssetPipeline is a test double for AssetPipelinePort where nil
// function fields return zero values and call counts are tracked atomically.
type MockAssetPipeline struct {
	// ProcessBuildResultFunc is the function called by
	// ProcessBuildResult.
	ProcessBuildResultFunc func(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error

	// ProcessBuildResultCallCount tracks how many times
	// ProcessBuildResult was called.
	ProcessBuildResultCallCount int64
}

var _ AssetPipelinePort = (*MockAssetPipeline)(nil)

// ProcessBuildResult handles the annotation result from a build operation.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signals.
// Takes result (*annotator_dto.ProjectAnnotationResult) which
// is the build output to process.
//
// Returns error, or nil if ProcessBuildResultFunc is nil.
func (m *MockAssetPipeline) ProcessBuildResult(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error {
	atomic.AddInt64(&m.ProcessBuildResultCallCount, 1)
	if m.ProcessBuildResultFunc != nil {
		return m.ProcessBuildResultFunc(ctx, result)
	}
	return nil
}
