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

func TestMinifyJavascriptCapability(t *testing.T) {
	const validJS = `
		// This is a single-line comment.
		(function() {
			'use strict';
			
			const longVariableName = 10;
			const anotherVar       = 20;

			/* This is a multi-line comment
			   that should be removed. */

			const result = longVariableName + anotherVar;
			console.log("The result is: ", result); // This console.log should be removed by the minifier.

			return result;
		})();
	`
	const expectedMinifiedJS = `(function(){"use strict";const t=10,n=20,e=t+n;return console.log("The result is: ",e),e})()`

	testCases := []struct {
		setupCtx           func() (context.Context, context.CancelFunc)
		name               string
		inputJS            string
		expectedOutput     string
		errorContains      string
		expectInitialError bool
		expectReadError    bool
	}{
		{
			name:           "should minify valid javascript",
			inputJS:        validJS,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: expectedMinifiedJS,
		},
		{
			name:           "should handle empty input stream",
			inputJS:        "",
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: "",
		},
		{
			name:           "should handle already minified javascript",
			inputJS:        expectedMinifiedJS,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: expectedMinifiedJS,
		},
		{
			name:            "should return a read error for invalid javascript",
			inputJS:         `function myFunction() { var x = 10;`,
			setupCtx:        func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectReadError: true,
			errorContains:   "unexpected EOF",
		},
		{
			name:            "should return a read error for syntax error",
			inputJS:         `const a =;`,
			setupCtx:        func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectReadError: true,
			errorContains:   "unexpected ;",
		},
		{
			name:    "should fail immediately if context is already cancelled",
			inputJS: validJS,
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(fmt.Errorf("test: simulating cancelled context"))
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			expectInitialError: true,
			errorContains:      "context canceled",
		},
		{
			name:    "should fail immediately if context times out before execution",
			inputJS: validJS,
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeoutCause(context.Background(), -1*time.Second, fmt.Errorf("test: simulating expired deadline"))
			},
			expectInitialError: true,
			errorContains:      "context deadline exceeded",
		},
	}

	minifyFunc := MinifyJavascript()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.setupCtx()
			defer cancel()

			inputReader := strings.NewReader(tc.inputJS)
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
				assert.LessOrEqual(t, len(outputBytes), len(tc.inputJS), "minified output should be smaller than or equal to the input")
			}
		})
	}
}
