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

package querier_domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "ErrMissingEnginePort has correct message",
			err:             ErrMissingEnginePort,
			expectedMessage: "querier service requires an engine port",
		},
		{
			name:            "ErrMissingEmitterPort has correct message",
			err:             ErrMissingEmitterPort,
			expectedMessage: "querier service requires a code emitter port",
		},
		{
			name:            "ErrMissingFileReaderPort has correct message",
			err:             ErrMissingFileReaderPort,
			expectedMessage: "querier service requires a file reader port",
		},
		{
			name:            "ErrMissingConfig has correct message",
			err:             ErrMissingConfig,
			expectedMessage: "querier service requires a database configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tt.err)
			assert.Equal(t, tt.expectedMessage, tt.err.Error())

			wrapped := fmt.Errorf("wrapped: %w", tt.err)
			assert.True(t, errors.Is(wrapped, tt.err))
		})
	}
}
