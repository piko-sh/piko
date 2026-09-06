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

package pdfwriter_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorePanicAsError(t *testing.T) {
	t.Parallel()

	t.Run("no panic leaves the error untouched", func(t *testing.T) {
		t.Parallel()

		existing := errors.New("existing")
		err := existing
		StorePanicAsError(context.Background(), "layout", nil, &err)
		assert.Same(t, existing, err)
	})

	t.Run("recovered value becomes a stack-free error", func(t *testing.T) {
		t.Parallel()

		var err error
		StorePanicAsError(context.Background(), "layout", "boom", &err)
		require.Error(t, err)
		assert.Equal(t, "pdfwriter: panic during layout: boom", err.Error())
		assert.NotContains(t, err.Error(), "goroutine")
	})

	t.Run("works when deferred around a panic", func(t *testing.T) {
		t.Parallel()

		run := func() (err error) {
			defer func() { StorePanicAsError(context.Background(), "test stage", recover(), &err) }()
			panic("stage failed")
		}

		err := run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic during test stage")
	})
}
