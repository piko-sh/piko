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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

func TestCreateCompressionCapability(t *testing.T) {
	t.Parallel()

	t.Run("should return a non-nil function", func(t *testing.T) {
		t.Parallel()
		config := compressionConfig{
			spanName:     "TestCompression",
			defaultLevel: 5,
			parseLevel: func(_ capabilities_domain.CapabilityParams, defaultLevel int) int {
				return defaultLevel
			},
			factory: func(level int) writerFactory {
				return func(dst io.Writer) (io.WriteCloser, error) {
					return &nopWriteCloser{Writer: dst}, nil
				}
			},
		}
		capabilityFunction := createCompressionCapability(config)
		require.NotNil(t, capabilityFunction)
	})

	t.Run("should pass data through with passthrough factory", func(t *testing.T) {
		t.Parallel()
		config := compressionConfig{
			spanName:     "TestPassthrough",
			defaultLevel: 0,
			parseLevel: func(_ capabilities_domain.CapabilityParams, defaultLevel int) int {
				return defaultLevel
			},
			factory: func(_ int) writerFactory {
				return func(dst io.Writer) (io.WriteCloser, error) {
					return &nopWriteCloser{Writer: dst}, nil
				}
			},
		}
		capabilityFunction := createCompressionCapability(config)

		stream, err := capabilityFunction(context.Background(), strings.NewReader("hello"), nil)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(output))
	})

	t.Run("should fail immediately on cancelled context", func(t *testing.T) {
		t.Parallel()
		config := compressionConfig{
			spanName:     "TestCancelledCtx",
			defaultLevel: 0,
			parseLevel: func(_ capabilities_domain.CapabilityParams, defaultLevel int) int {
				return defaultLevel
			},
			factory: func(_ int) writerFactory {
				return func(dst io.Writer) (io.WriteCloser, error) {
					return &nopWriteCloser{Writer: dst}, nil
				}
			},
		}
		capabilityFunction := createCompressionCapability(config)

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		result, err := capabilityFunction(ctx, strings.NewReader("data"), nil)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should use parsed level from params", func(t *testing.T) {
		t.Parallel()
		parsedLevel := -1
		config := compressionConfig{
			spanName:     "TestLevelParsing",
			defaultLevel: 5,
			parseLevel: func(params capabilities_domain.CapabilityParams, defaultLevel int) int {
				if l, ok := params["level"]; ok && l == "9" {
					parsedLevel = 9
					return 9
				}
				parsedLevel = defaultLevel
				return defaultLevel
			},
			factory: func(level int) writerFactory {
				return func(dst io.Writer) (io.WriteCloser, error) {
					return &nopWriteCloser{Writer: dst}, nil
				}
			},
		}
		capabilityFunction := createCompressionCapability(config)

		params := capabilities_domain.CapabilityParams{"level": "9"}
		stream, err := capabilityFunction(context.Background(), strings.NewReader("test"), params)
		require.NoError(t, err)
		require.NotNil(t, stream)

		_, err = io.ReadAll(stream)
		require.NoError(t, err)
		assert.Equal(t, 9, parsedLevel)
	})

	t.Run("should handle empty input", func(t *testing.T) {
		t.Parallel()
		config := compressionConfig{
			spanName:     "TestEmptyInput",
			defaultLevel: 0,
			parseLevel: func(_ capabilities_domain.CapabilityParams, defaultLevel int) int {
				return defaultLevel
			},
			factory: func(_ int) writerFactory {
				return func(dst io.Writer) (io.WriteCloser, error) {
					return &nopWriteCloser{Writer: dst}, nil
				}
			},
		}
		capabilityFunction := createCompressionCapability(config)

		stream, err := capabilityFunction(context.Background(), strings.NewReader(""), nil)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		assert.Empty(t, output)
	})
}
