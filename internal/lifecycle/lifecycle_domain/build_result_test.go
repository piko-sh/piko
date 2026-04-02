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

package lifecycle_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildResult_HasFailures_True(t *testing.T) {
	t.Parallel()

	result := &BuildResult{
		TotalDispatched: 10,
		TotalCompleted:  8,
		TotalFailed:     2,
	}

	assert.True(t, result.HasFailures(), "expected HasFailures to return true when TotalFailed > 0")
}

func TestBuildResult_HasFailures_False(t *testing.T) {
	t.Parallel()

	result := &BuildResult{
		TotalDispatched: 5,
		TotalCompleted:  5,
		TotalFailed:     0,
	}

	assert.False(t, result.HasFailures(), "expected HasFailures to return false when TotalFailed == 0")
}

func TestBuildResult_HasFailures_ZeroStruct(t *testing.T) {
	t.Parallel()

	result := &BuildResult{}

	assert.False(t, result.HasFailures(), "expected HasFailures to return false for zero-value BuildResult")
}
