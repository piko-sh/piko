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

package orchestrator_domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

func TestNewFatalError_WrapsWithSentinel(t *testing.T) {
	t.Parallel()

	original := errors.New("something broke")
	result := NewFatalError(original)

	require.Error(t, result)
	assert.True(t, errors.Is(result, ErrFatal),
		"expected the returned error to match orchestrator ErrFatal sentinel")
}

func TestNewFatalError_PreservesOriginal(t *testing.T) {
	t.Parallel()

	original := errors.New("underlying cause")
	result := NewFatalError(original)

	require.Error(t, result)
	assert.True(t, errors.Is(result, original),
		"expected the original error to be preserved in the chain")
}

func TestIsFatalError_PlainError(t *testing.T) {
	t.Parallel()

	plain := errors.New("foo")
	assert.False(t, IsFatalError(plain),
		"a plain error should not be recognised as fatal")
}

func TestIsFatalError_DoubleWrapped(t *testing.T) {
	t.Parallel()

	inner := errors.New("inner failure")
	fatal := NewFatalError(inner)
	outer := fmt.Errorf("outer: %w", fatal)

	assert.True(t, IsFatalError(outer),
		"IsFatalError should detect ErrFatal even when double-wrapped")
}

func TestIsFatalError_NilError(t *testing.T) {
	t.Parallel()

	assert.False(t, IsFatalError(nil),
		"IsFatalError should return false for nil")
}

func TestIsFatalError_OrchestratorAndCapabilityAreSeparate(t *testing.T) {
	t.Parallel()

	t.Run("capability fatal is not orchestrator fatal", func(t *testing.T) {
		t.Parallel()

		inner := errors.New("cap boom")
		capFatal := capabilities_domain.NewFatalError(inner)

		assert.False(t, IsFatalError(capFatal),
			"an error wrapped with capabilities_domain.ErrFatal must not trigger orchestrator's IsFatalError")
	})

	t.Run("orchestrator fatal is not capability fatal", func(t *testing.T) {
		t.Parallel()

		inner := errors.New("orch boom")
		orchFatal := NewFatalError(inner)

		assert.False(t, capabilities_domain.IsFatalError(orchFatal),
			"an error wrapped with orchestrator ErrFatal must not trigger capabilities_domain.IsFatalError")
	})

	t.Run("sentinels are distinct values", func(t *testing.T) {
		t.Parallel()

		assert.False(t, errors.Is(ErrFatal, capabilities_domain.ErrFatal),
			"orchestrator ErrFatal and capabilities_domain ErrFatal must be distinct sentinels")
	})
}
