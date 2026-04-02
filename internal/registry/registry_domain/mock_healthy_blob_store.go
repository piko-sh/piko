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

package registry_domain

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// MockHealthyBlobStore composes MockBlobStore with health probe methods
// for blob stores that need to participate in health checking, where nil
// function fields return zero values and call counts are tracked
// atomically.
type MockHealthyBlobStore struct {
	// NameFunc is the function called by Name.
	NameFunc func() string

	// CheckFunc is the function called by Check.
	CheckFunc func(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status

	MockBlobStore

	// NameCallCount tracks how many times Name was
	// called.
	NameCallCount int64

	// CheckCallCount tracks how many times Check was
	// called.
	CheckCallCount int64
}

// Name returns the identifier of the component being checked.
//
// Returns string, or "" if NameFunc is nil.
func (m *MockHealthyBlobStore) Name() string {
	atomic.AddInt64(&m.NameCallCount, 1)
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return ""
}

// Check performs the health check.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes checkType (healthprobe_dto.CheckType) which
// specifies the type of health check to perform.
//
// Returns healthprobe_dto.Status, or zero value if CheckFunc is nil.
func (m *MockHealthyBlobStore) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	atomic.AddInt64(&m.CheckCallCount, 1)
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, checkType)
	}
	return healthprobe_dto.Status{}
}
