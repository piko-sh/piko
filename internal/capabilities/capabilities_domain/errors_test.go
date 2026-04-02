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

package capabilities_domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFatalError_WrapsWithSentinel(t *testing.T) {
	t.Parallel()

	original := errors.New("something went wrong")
	wrapped := NewFatalError(original)

	require.Error(t, wrapped)
	assert.True(t, errors.Is(wrapped, ErrFatal), "expected the wrapped error to satisfy errors.Is for ErrFatal")
}

func TestNewFatalError_PreservesOriginal(t *testing.T) {
	t.Parallel()

	original := errors.New("parse failure")
	wrapped := NewFatalError(original)

	require.Error(t, wrapped)
	assert.True(t, errors.Is(wrapped, original), "expected the original error to remain in the chain")
	assert.Contains(t, wrapped.Error(), "parse failure", "expected the original message to be preserved")
}

func TestIsFatalError_PlainError(t *testing.T) {
	t.Parallel()

	plain := errors.New("foo")
	assert.False(t, IsFatalError(plain), "a plain error should not be recognised as fatal")
}

func TestIsFatalError_DoubleWrapped(t *testing.T) {
	t.Parallel()

	inner := errors.New("bad input")
	fatal := NewFatalError(inner)
	outer := fmt.Errorf("outer: %w", fatal)

	assert.True(t, IsFatalError(outer), "a double-wrapped fatal error should still be recognised as fatal")
}

func TestIsFatalError_NilError(t *testing.T) {
	t.Parallel()

	assert.False(t, IsFatalError(nil), "nil should not be recognised as a fatal error")
}
