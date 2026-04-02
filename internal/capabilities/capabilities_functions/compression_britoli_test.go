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
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

func decompressBrotli(t *testing.T, data []byte) string {
	t.Helper()

	r := brotli.NewReader(bytes.NewReader(data))

	decompressed, err := io.ReadAll(r)
	require.NoError(t, err, "failed to read decompressed brotli data")

	return string(decompressed)
}

func TestBrotliCapability(t *testing.T) {
	const compressibleInput = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	testCases := []struct {
		validationFunc func(t *testing.T, input, output []byte)
		params         capabilities_domain.CapabilityParams
		name           string
		input          string
		expectedOutput string
		errorContains  string
		ctxTimeout     time.Duration
		useSlowReader  bool
		expectError    bool
	}{
		{
			name:           "should compress with default settings",
			input:          compressibleInput,
			params:         nil,
			expectedOutput: compressibleInput,
			validationFunc: func(t *testing.T, input, output []byte) {
				assert.Less(t, len(output), len(input), "compressed output should be smaller than input")
			},
		},
		{
			name:           "should handle empty input stream",
			input:          "",
			params:         nil,
			expectedOutput: "",
			validationFunc: func(t *testing.T, input, output []byte) {
				assert.NotEmpty(t, output, "brotli-ing an empty string should still produce some output bytes")
			},
		},
		{
			name:           "should handle non-compressible data",
			input:          "abcdefghijklmnopqrstuvwxyz1234567890",
			params:         nil,
			expectedOutput: "abcdefghijklmnopqrstuvwxyz1234567890",
			validationFunc: func(t *testing.T, input, output []byte) {
				assert.Greater(t, len(output), len(input), "compressed random data should be larger due to overhead")
			},
		},
		{
			name:           "should use specified best speed level",
			input:          compressibleInput,
			params:         capabilities_domain.CapabilityParams{paramBrotliLevel: "0"},
			expectedOutput: compressibleInput,
			validationFunc: func(t *testing.T, input, output []byte) {
				defaultOutput := runBrotliCapability(t, string(input), nil)
				assert.Greater(t, len(output), len(defaultOutput), "level 0 should compress less (be larger) than default")
			},
		},
		{
			name:           "should fall back to default for invalid level string",
			input:          compressibleInput,
			params:         capabilities_domain.CapabilityParams{paramBrotliLevel: "not-a-number"},
			expectedOutput: compressibleInput,
			validationFunc: func(t *testing.T, input, output []byte) {
				defaultOutput := runBrotliCapability(t, string(input), nil)
				assert.Equal(t, len(defaultOutput), len(output), "invalid level should produce same size as default")
			},
		},
		{
			name:           "should fall back to default for out-of-range level",
			input:          compressibleInput,
			params:         capabilities_domain.CapabilityParams{paramBrotliLevel: "99"},
			expectedOutput: compressibleInput,
			validationFunc: func(t *testing.T, input, output []byte) {
				defaultOutput := runBrotliCapability(t, string(input), nil)
				assert.Equal(t, len(defaultOutput), len(output), "out-of-range level should produce same size as default")
			},
		},
		{
			name:          "should abort on context cancellation during slow read",
			input:         compressibleInput,
			params:        nil,
			ctxTimeout:    50 * time.Millisecond,
			useSlowReader: true,
			expectError:   true,
			errorContains: "context deadline exceeded",
		},
	}

	brotliFunc := Brotli()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var cancel context.CancelFunc
			if tc.ctxTimeout > 0 {
				ctx, cancel = context.WithTimeoutCause(ctx, tc.ctxTimeout, fmt.Errorf("test: brotli compression timeout"))
				defer cancel()
			}

			var inputReader io.Reader = strings.NewReader(tc.input)
			if tc.useSlowReader {
				inputReader = &slowReader{
					r:         inputReader,
					delay:     20 * time.Millisecond,
					chunkSize: 5,
				}
			}

			outputStream, err := brotliFunc(ctx, inputReader, tc.params)

			require.NoError(t, err, "Brotli() function returned an immediate error")
			require.NotNil(t, outputStream, "Brotli() function returned a nil stream")

			outputBytes, readErr := io.ReadAll(outputStream)

			if tc.expectError {
				require.Error(t, readErr, "expected a stream read error but got none")
				assert.Contains(t, readErr.Error(), tc.errorContains, "error message did not contain expected text")
			} else {
				require.NoError(t, readErr, "expected no stream read error but got one")

				decompressed := decompressBrotli(t, outputBytes)
				assert.Equal(t, tc.expectedOutput, decompressed, "decompressed output did not match expected")

				if tc.validationFunc != nil {
					tc.validationFunc(t, []byte(tc.input), outputBytes)
				}
			}
		})
	}
}

func runBrotliCapability(t *testing.T, input string, params capabilities_domain.CapabilityParams) []byte {
	t.Helper()
	brotliFunc := Brotli()
	stream, err := brotliFunc(context.Background(), strings.NewReader(input), params)
	require.NoError(t, err)
	output, err := io.ReadAll(stream)
	require.NoError(t, err)
	return output
}

func TestParseBrotliLevel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		params       capabilities_domain.CapabilityParams
		name         string
		defaultLevel int
		expected     int
	}{
		{
			name:         "should return default when no level param",
			params:       capabilities_domain.CapabilityParams{},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.DefaultCompression,
		},
		{
			name:         "should return default when nil params",
			params:       nil,
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.DefaultCompression,
		},
		{
			name:         "should return default for non-numeric level",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: "abc"},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.DefaultCompression,
		},
		{
			name:         "should return default for empty string level",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: ""},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.DefaultCompression,
		},
		{
			name:         "should return default for negative level",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: "-1"},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.DefaultCompression,
		},
		{
			name:         "should return default for level above BestCompression",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: "12"},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.DefaultCompression,
		},
		{
			name:         "should parse valid BestSpeed level",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: "0"},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.BestSpeed,
		},
		{
			name:         "should parse valid BestCompression level",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: "11"},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.BestCompression,
		},
		{
			name:         "should parse valid middle level",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: "6"},
			defaultLevel: brotli.DefaultCompression,
			expected:     6,
		},
		{
			name:         "should use provided default level",
			params:       capabilities_domain.CapabilityParams{},
			defaultLevel: 4,
			expected:     4,
		},
		{
			name:         "should return default for float value",
			params:       capabilities_domain.CapabilityParams{paramBrotliLevel: "3.5"},
			defaultLevel: brotli.DefaultCompression,
			expected:     brotli.DefaultCompression,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseBrotliLevel(tc.params, tc.defaultLevel)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBrotliWriterFactory(t *testing.T) {
	t.Parallel()

	t.Run("should create a working brotli writer at default level", func(t *testing.T) {
		t.Parallel()
		factory := brotliWriterFactory(brotli.DefaultCompression)
		var buffer bytes.Buffer
		writer, err := factory(&buffer)
		require.NoError(t, err)
		require.NotNil(t, writer)

		_, err = writer.Write([]byte("test data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		decompressed := decompressBrotli(t, buffer.Bytes())
		assert.Equal(t, "test data", decompressed)
	})

	t.Run("should create writers at different levels", func(t *testing.T) {
		t.Parallel()
		levels := []int{brotli.BestSpeed, brotli.DefaultCompression, brotli.BestCompression}
		for _, level := range levels {
			factory := brotliWriterFactory(level)
			var buffer bytes.Buffer
			writer, err := factory(&buffer)
			require.NoError(t, err)
			require.NotNil(t, writer)

			_, err = writer.Write([]byte("test"))
			require.NoError(t, err)

			err = writer.Close()
			require.NoError(t, err)

			decompressed := decompressBrotli(t, buffer.Bytes())
			assert.Equal(t, "test", decompressed)
		}
	})
}

func TestBrotliPoolsGetPoolForLevel(t *testing.T) {
	t.Parallel()

	t.Run("should return same pool for same level", func(t *testing.T) {
		t.Parallel()
		pools := &brotliPools{
			pools: make(map[int]*sync.Pool),
		}
		pool1 := pools.getPoolForLevel(5)
		pool2 := pools.getPoolForLevel(5)
		assert.Same(t, pool1, pool2)
	})

	t.Run("should return different pools for different levels", func(t *testing.T) {
		t.Parallel()
		pools := &brotliPools{
			pools: make(map[int]*sync.Pool),
		}
		pool1 := pools.getPoolForLevel(0)
		pool2 := pools.getPoolForLevel(11)
		assert.NotSame(t, pool1, pool2)
	})

	t.Run("pool should produce valid brotli writers", func(t *testing.T) {
		t.Parallel()
		pools := &brotliPools{
			pools: make(map[int]*sync.Pool),
		}
		pool := pools.getPoolForLevel(brotli.DefaultCompression)
		item := pool.Get()
		require.NotNil(t, item)

		bw, ok := item.(*brotli.Writer)
		assert.True(t, ok)
		assert.NotNil(t, bw)
		pool.Put(bw)
	})
}

func TestPooledBrotliWriterClose(t *testing.T) {
	t.Parallel()

	t.Run("should close writer and return to pool", func(t *testing.T) {
		t.Parallel()
		pool := globalBrotliPools.getPoolForLevel(brotli.DefaultCompression)
		var buffer bytes.Buffer

		bw, ok := pool.Get().(*brotli.Writer)
		require.True(t, ok)
		bw.Reset(&buffer)

		pw := &pooledBrotliWriter{Writer: bw, pool: pool}
		_, err := pw.Write([]byte("test"))
		require.NoError(t, err)

		err = pw.Close()
		require.NoError(t, err)

		decompressed := decompressBrotli(t, buffer.Bytes())
		assert.Equal(t, "test", decompressed)
	})
}

func TestBrotliWriterFactory_PoolReturnsNonWriter(t *testing.T) {
	t.Parallel()

	testLevel := brotli.BestSpeed
	pool := globalBrotliPools.getPoolForLevel(testLevel)

	pool.Put("not a brotli writer")

	factory := brotliWriterFactory(testLevel)
	var buffer bytes.Buffer
	writer, err := factory(&buffer)
	require.NoError(t, err)
	require.NotNil(t, writer)

	_, err = writer.Write([]byte("test data"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	decompressed := decompressBrotli(t, buffer.Bytes())
	assert.Equal(t, "test data", decompressed)
}

func TestBrotliPoolsGetPoolForLevel_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	pools := &brotliPools{
		pools: make(map[int]*sync.Pool),
	}

	done := make(chan *sync.Pool, 10)
	for range 10 {
		go func() {
			done <- pools.getPoolForLevel(3)
		}()
	}

	var firstPool *sync.Pool
	for range 10 {
		pool := <-done
		if firstPool == nil {
			firstPool = pool
		}
		assert.Same(t, firstPool, pool, "all goroutines should get the same pool")
	}
}

func TestBrotliCapabilityWithCancelledContext(t *testing.T) {
	t.Parallel()

	brotliFunc := Brotli()
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	result, err := brotliFunc(ctx, strings.NewReader("data"), nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestBrotliCapabilityWithSpecificLevels(t *testing.T) {
	t.Parallel()

	brotliFunc := Brotli()
	input := strings.Repeat("a", 500)

	t.Run("should compress with BestSpeed", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{paramBrotliLevel: "0"}
		stream, err := brotliFunc(context.Background(), strings.NewReader(input), params)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		decompressed := decompressBrotli(t, output)
		assert.Equal(t, input, decompressed)
	})

	t.Run("should compress with BestCompression", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{paramBrotliLevel: "11"}
		stream, err := brotliFunc(context.Background(), strings.NewReader(input), params)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		decompressed := decompressBrotli(t, output)
		assert.Equal(t, input, decompressed)
	})
}
