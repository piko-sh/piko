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

package capabilities_functions

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinifySVGCapability(t *testing.T) {
	const validSVG = `
		<?xml version="1.0" encoding="UTF-8"?>
		<!-- Created with a Fictional Vector Editor -->
		<svg width="100px" height="100px" version="1.1" xmlns="http://www.w3.org/2000/svg">
			<title>An Icon</title>
			<desc>A description to be removed.</desc>

			<!-- This is a comment -->
			<g id="useless-group">
				<rect
					style="fill:#ff0000; stroke:#000000; stroke-width:2px;"
					x="10"
					y="10"
					width="80"
					height="80" />
			</g>
		</svg>
	`
	const expectedMinifiedSVG = `<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg"><title>An Icon</title><desc>A description to be removed.</desc><g id="useless-group"><rect style="fill:#ff0000; stroke:#000000; stroke-width:2px;" x="10" y="10" width="80" height="80"/></g></svg>`

	testCases := []struct {
		setupCtx           func() (context.Context, context.CancelFunc)
		name               string
		inputSVG           string
		expectedOutput     string
		errorContains      string
		expectInitialError bool
		expectReadError    bool
	}{
		{
			name:           "should minify valid svg",
			inputSVG:       validSVG,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: expectedMinifiedSVG,
		},
		{
			name:           "should handle empty input stream",
			inputSVG:       "",
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: "",
		},
		{
			name:           "should handle already minified svg",
			inputSVG:       expectedMinifiedSVG,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: expectedMinifiedSVG,
		},
		{
			name:           "should handle unclosed tags gracefully",
			inputSVG:       `<svg><rect x="10" y="10"></svg>`,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: `<svg><rect x="10" y="10"/>`,
		},
		{
			name:           "should handle mismatched tags gracefully",
			inputSVG:       `<svg><rect></g></svg>`,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: `<svg><rect/></svg>`,
		},
		{
			name:           "should handle fundamentally broken xml gracefully",
			inputSVG:       `<svg`,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: `<svg`,
		},
		{
			name:     "should fail immediately if context is already cancelled",
			inputSVG: validSVG,
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(fmt.Errorf("test: simulating cancelled context"))
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			expectInitialError: true,
			errorContains:      "context canceled",
		},
		{
			name:     "should fail immediately if context times out before execution",
			inputSVG: validSVG,
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeoutCause(context.Background(), -1*time.Second, fmt.Errorf("test: simulating expired deadline"))
			},
			expectInitialError: true,
			errorContains:      "context deadline exceeded",
		},
	}

	minifyFunc := MinifySVG()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.setupCtx()
			defer cancel()

			inputReader := strings.NewReader(tc.inputSVG)
			outputStream, initialErr := minifyFunc(ctx, inputReader, nil)

			if tc.expectInitialError {
				require.Error(t, initialErr, "expected an initial error but got none")
				assert.Contains(t, initialErr.Error(), tc.errorContains, "initial error message did not contain expected text")
				assert.Nil(t, outputStream, "output stream should be nil on initial error")
				return
			}

			require.NoError(t, initialErr, "minify function returned an unexpected initial error")
			require.NotNil(t, outputStream, "minify function returned a nil stream")

			outputBytes, readErr := io.ReadAll(outputStream)

			if tc.expectReadError {
				require.Error(t, readErr, "expected a stream read error but got none")
				assert.Contains(t, readErr.Error(), tc.errorContains, "read error message did not contain expected text")
			} else {
				require.NoError(t, readErr, "expected no stream read error but got one")
				assert.Equal(t, tc.expectedOutput, string(outputBytes), "minified output did not match expected")
			}
		})
	}
}
