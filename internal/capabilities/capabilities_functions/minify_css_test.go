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
	"go.opentelemetry.io/otel/trace/noop"
	es_logger "piko.sh/piko/internal/esbuild/logger"
)

func TestMinifyCSSCapability(t *testing.T) {
	const validCSS = `
		/* This is a comment that should be removed. */
		body {
			font-family:    "Helvetica Neue", Arial, sans-serif;
			background-color: #ffffff; /* Use a long hex code */
			margin: 10px 20px;
		}

		a:hover {
			color: red;
		}
	`
	const expectedMinifiedCSS = `body{font-family:Helvetica Neue,Arial,sans-serif;background-color:#fff;margin:10px 20px}a:hover{color:red}`

	testCases := []struct {
		setupCtx           func() (context.Context, context.CancelFunc)
		name               string
		inputCSS           string
		expectedOutput     string
		errorContains      string
		expectInitialError bool
		expectReadError    bool
	}{
		{
			name:           "should minify valid css",
			inputCSS:       validCSS,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: expectedMinifiedCSS,
		},
		{
			name:           "should handle empty input stream",
			inputCSS:       "",
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: "",
		},
		{
			name:           "should handle already minified css",
			inputCSS:       expectedMinifiedCSS,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: expectedMinifiedCSS,
		},
		{
			name:           "should handle unclosed block gracefully",
			inputCSS:       `a{color:blue} body { color: red;`,
			setupCtx:       func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			expectedOutput: `a{color:#00f}body{color:red}`,
		},
		{
			name:     "should fail immediately if context is already cancelled",
			inputCSS: validCSS,
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
			inputCSS: validCSS,
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeoutCause(context.Background(), -1*time.Second, fmt.Errorf("test: simulating expired deadline"))
			},
			expectInitialError: true,
			errorContains:      "context deadline exceeded",
		},
	}

	minifyFunc := MinifyCSS()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.setupCtx()
			defer cancel()

			inputReader := strings.NewReader(tc.inputCSS)
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

func TestMinifyCSSCapability_NestingLowering(t *testing.T) {
	minifyFunc := MinifyCSS()
	ctx := context.Background()

	testCases := []struct {
		name             string
		inputCSS         string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "should lower ampersand nesting",
			inputCSS: `.form-section {
				&:not(:last-of-type) {
					margin-bottom: 50px;
				}
			}`,
			shouldContain: []string{
				".form-section:not(:last-of-type)",
				"margin-bottom:50px",
			},
			shouldNotContain: []string{
				"&:not",
				"&:",
			},
		},
		{
			name: "should lower nesting in media queries",
			inputCSS: `
				@media screen and (max-width: 768px) {
					.form-section {
						&:not(:last-of-type) {
							margin-bottom: 30px;
						}
					}
				}
			`,
			shouldContain: []string{
				"@media",
				".form-section:not(:last-of-type)",
				"margin-bottom:30px",
			},
			shouldNotContain: []string{
				"&:not",
			},
		},
		{
			name: "should lower nested class selectors",
			inputCSS: `
				.parent {
					.child {
						color: blue;
					}
				}
			`,
			shouldContain: []string{
				".parent .child",
				"color:",
			},
			shouldNotContain: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputReader := strings.NewReader(tc.inputCSS)
			outputStream, err := minifyFunc(ctx, inputReader, nil)
			require.NoError(t, err)
			require.NotNil(t, outputStream)

			outputBytes, err := io.ReadAll(outputStream)
			require.NoError(t, err)

			output := string(outputBytes)
			t.Logf("Output: %s", output)

			for _, should := range tc.shouldContain {
				assert.Contains(t, output, should, "Output should contain: %s", should)
			}

			for _, shouldNot := range tc.shouldNotContain {
				assert.NotContains(t, output, shouldNot, "Output should NOT contain: %s", shouldNot)
			}
		})
	}
}

func TestCreateCSSSource(t *testing.T) {
	t.Parallel()

	t.Run("should create source with content", func(t *testing.T) {
		t.Parallel()
		input := []byte("body { color: red; }")
		source := createCSSSource(input)
		assert.Equal(t, "body { color: red; }", source.Contents)
		assert.Equal(t, "asset.css", source.KeyPath.Text)
		assert.Empty(t, source.KeyPath.Namespace)
	})

	t.Run("should create source with empty content", func(t *testing.T) {
		t.Parallel()
		source := createCSSSource([]byte{})
		assert.Empty(t, source.Contents)
		assert.Equal(t, "asset.css", source.KeyPath.Text)
	})

	t.Run("should preserve content with special characters", func(t *testing.T) {
		t.Parallel()
		input := []byte("body { content: \"hello\\nworld\"; }")
		source := createCSSSource(input)
		assert.Equal(t, string(input), source.Contents)
	})
}

func TestParseAndMinifyCSS(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "should minify simple CSS",
			input:    "body { color: red; }",
			expected: "body{color:red}",
		},
		{
			name:     "should handle empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "should remove comments",
			input:    "/* comment */ body { color: blue; }",
			expected: "body{color:#00f}",
		},
		{
			name:     "should collapse whitespace",
			input:    "body  {   color:   red;    margin:   0;   }",
			expected: "body{color:red;margin:0}",
		},
		{
			name:     "should handle multiple selectors",
			input:    "h1 { font-size: 20px; } h2 { font-size: 16px; }",
			expected: "h1{font-size:20px}h2{font-size:16px}",
		},
		{
			name:     "should shorten hex colors",
			input:    "a { color: #ff0000; }",
			expected: "a{color:red}",
		},
		{
			name:     "should handle media queries",
			input:    "@media (min-width: 768px) { body { color: red; } }",
			expected: "@media(min-width:768px){body{color:red}}",
		},
		{
			name:     "should handle CSS with nested selectors (lowering)",
			input:    ".parent { .child { color: red; } }",
			expected: ".parent .child{color:red}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseAndMinifyCSS([]byte(tc.input))
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestPrintMinifiedCSS(t *testing.T) {
	t.Parallel()

	t.Run("should produce non-nil output for parsed tree", func(t *testing.T) {
		t.Parallel()
		input := []byte("body { margin: 0; }")
		source := createCSSSource(input)
		esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
		tree := parseCSSTree(esLog, source)
		result := printMinifiedCSS(tree)
		require.NotNil(t, result)
		assert.Equal(t, "body{margin:0}", string(result))
	})

	t.Run("should handle empty tree", func(t *testing.T) {
		t.Parallel()
		source := createCSSSource([]byte{})
		esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
		tree := parseCSSTree(esLog, source)
		result := printMinifiedCSS(tree)
		assert.Empty(t, result)
	})
}

func TestExecuteCSSMinification(t *testing.T) {
	t.Parallel()

	t.Run("should minify valid CSS", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("body { color: red; }")
		result, duration, err := executeCSSMinification(context.Background(), input)
		require.NoError(t, err)
		assert.Equal(t, "body{color:red}", string(result))
		assert.GreaterOrEqual(t, duration.Nanoseconds(), int64(0))
	})

	t.Run("should handle empty input", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("")
		result, _, err := executeCSSMinification(context.Background(), input)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("should return error from failing reader", func(t *testing.T) {
		t.Parallel()
		input := &errorReader{Err: io.ErrUnexpectedEOF}
		_, _, err := executeCSSMinification(context.Background(), input)
		require.Error(t, err)
	})
}

func TestRecordCSSMinificationSuccess(t *testing.T) {
	t.Parallel()

	t.Run("should not panic", func(t *testing.T) {
		t.Parallel()
		span := noop.Span{}
		assert.NotPanics(t, func() {
			recordCSSMinificationSuccess(context.Background(), span, []byte("minified"), 100*time.Millisecond)
		})
	})

	t.Run("should not panic with empty output", func(t *testing.T) {
		t.Parallel()
		span := noop.Span{}
		assert.NotPanics(t, func() {
			recordCSSMinificationSuccess(context.Background(), span, []byte{}, 0)
		})
	})
}

func TestMinifyCSSCapability_ErrorFromReader(t *testing.T) {
	t.Parallel()

	minifyFunc := MinifyCSS()

	t.Run("should return error when input reader fails", func(t *testing.T) {
		t.Parallel()
		input := &errorReader{Err: io.ErrUnexpectedEOF}
		_, err := minifyFunc(context.Background(), input, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executing CSS minification")
	})
}

func TestParseCSSTree(t *testing.T) {
	t.Parallel()

	t.Run("should parse valid CSS into tree", func(t *testing.T) {
		t.Parallel()
		input := []byte("body { color: red; }")
		source := createCSSSource(input)
		esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
		tree := parseCSSTree(esLog, source)
		assert.NotEmpty(t, tree.Rules)
	})

	t.Run("should handle empty CSS", func(t *testing.T) {
		t.Parallel()
		source := createCSSSource([]byte{})
		esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
		tree := parseCSSTree(esLog, source)
		assert.Empty(t, tree.Rules)
	})

	t.Run("should handle multiple rules", func(t *testing.T) {
		t.Parallel()
		input := []byte("h1 { color: red; } h2 { color: blue; } p { margin: 0; }")
		source := createCSSSource(input)
		esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
		tree := parseCSSTree(esLog, source)
		assert.NotEmpty(t, tree.Rules)
	})
}
