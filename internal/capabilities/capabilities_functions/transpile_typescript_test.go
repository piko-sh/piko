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
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

func TestTranspileTypeScriptCapability(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setupCtx       func() (context.Context, context.CancelFunc)
		params         capabilities_domain.CapabilityParams
		name           string
		inputTS        string
		expectedOutput string
		errorContains  string
		expectError    bool
	}{
		{
			name: "should transpile valid typescript with type annotations",
			inputTS: `export function greet(name: string): string {
	return "Hello, " + name;
}`,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: "export function greet(name) {\n  return \"Hello, \" + name;\n}\n",
		},
		{
			name:           "should handle empty input",
			inputTS:        "",
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: "",
		},
		{
			name: "should strip interfaces and type aliases",
			inputTS: `interface User {
	name: string;
	age: number;
}
type ID = string | number;
export const id: ID = "abc";`,
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: `export const id = "abc";
`,
		},
		{
			name:     "should preserve ES module imports and exports",
			inputTS:  "import { foo } from \"./bar.js\";\nexport const result: number = foo(42);",
			setupCtx: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
		},
		{
			name:          "should return fatal error for invalid typescript",
			inputTS:       `export function broken( { return; }`,
			setupCtx:      func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectError:   true,
			errorContains: "transpiling typescript",
		},
		{
			name:          "should use sourcePath param in error context",
			inputTS:       `export function broken( { return; }`,
			setupCtx:      func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			params:        capabilities_domain.CapabilityParams{"sourcePath": "lib/my-file.ts"},
			expectError:   true,
			errorContains: "my-file.ts",
		},
		{
			name:    "should fail immediately if context is already cancelled",
			inputTS: `export const x: number = 1;`,
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(fmt.Errorf("test: simulating cancelled context"))
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			expectError:   true,
			errorContains: "context canceled",
		},
		{
			name:    "should fail immediately if context times out before execution",
			inputTS: `export const x: number = 1;`,
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeoutCause(context.Background(), -1*time.Second, fmt.Errorf("test: simulating expired deadline"))
			},
			expectError:   true,
			errorContains: "context deadline exceeded",
		},
	}

	transpileFunc := TranspileTypeScript()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := tc.setupCtx()
			defer cancel()

			inputReader := strings.NewReader(tc.inputTS)
			outputStream, err := transpileFunc(ctx, inputReader, tc.params)

			if tc.expectError {
				require.Error(t, err, "expected an error but got none")
				assert.Contains(t, err.Error(), tc.errorContains, "error message did not contain expected text")
				return
			}

			require.NoError(t, err, "transpile function returned an unexpected error")
			require.NotNil(t, outputStream, "transpile function returned a nil stream")

			outputBytes, readErr := io.ReadAll(outputStream)
			require.NoError(t, readErr, "reading output stream failed")

			if tc.expectedOutput != "" {
				assert.Equal(t, tc.expectedOutput, string(outputBytes), "transpiled output did not match expected")
			}
		})
	}
}

func TestTranspileTypeScript_FatalError(t *testing.T) {
	t.Parallel()

	transpileFunc := TranspileTypeScript()
	inputReader := strings.NewReader(`export function broken( { return; }`)

	_, err := transpileFunc(context.Background(), inputReader, nil)
	require.Error(t, err)
	assert.True(t, capabilities_domain.IsFatalError(err), "error should be a fatal error")
}
