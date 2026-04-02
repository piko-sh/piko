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

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

func decompressGzip(t *testing.T, data []byte) string {
	t.Helper()

	r, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err, "failed to create gzip reader for decompression")
	defer func() { _ = r.Close() }()

	decompressed, err := io.ReadAll(r)
	require.NoError(t, err, "failed to read decompressed data")

	return string(decompressed)
}

func TestGzipCapability(t *testing.T) {
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
				assert.NotEmpty(t, output, "gzipping an empty string should still produce header/footer bytes")
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
			name:           "should use specified no compression level",
			input:          compressibleInput,
			params:         capabilities_domain.CapabilityParams{paramGzipLevel: "0"},
			expectedOutput: compressibleInput,
			validationFunc: func(t *testing.T, input, output []byte) {
				assert.Greater(t, len(output), len(input), "level 0 should be larger than input due to overhead")
			},
		},
		{
			name:           "should fall back to default for invalid level string",
			input:          compressibleInput,
			params:         capabilities_domain.CapabilityParams{paramGzipLevel: "not-a-number"},
			expectedOutput: compressibleInput,
			validationFunc: func(t *testing.T, input, output []byte) {
				defaultOutput := runGzipCapability(t, string(input), nil)
				assert.Equal(t, len(defaultOutput), len(output), "invalid level should produce same size as default")
			},
		},
		{
			name:           "should fall back to default for out-of-range level",
			input:          compressibleInput,
			params:         capabilities_domain.CapabilityParams{paramGzipLevel: "99"},
			expectedOutput: compressibleInput,
			validationFunc: func(t *testing.T, input, output []byte) {
				defaultOutput := runGzipCapability(t, string(input), nil)
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

	gzipFunc := Gzip()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var cancel context.CancelFunc
			if tc.ctxTimeout > 0 {
				ctx, cancel = context.WithTimeoutCause(ctx, tc.ctxTimeout, fmt.Errorf("test: gzip compression timeout"))
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

			outputStream, err := gzipFunc(ctx, inputReader, tc.params)

			require.NoError(t, err, "Gzip() function returned an immediate error")
			require.NotNil(t, outputStream, "Gzip() function returned a nil stream")

			outputBytes, readErr := io.ReadAll(outputStream)

			if tc.expectError {
				require.Error(t, readErr, "expected a stream read error but got none")
				assert.Contains(t, readErr.Error(), tc.errorContains, "error message did not contain expected text")
			} else {
				require.NoError(t, readErr, "expected no stream read error but got one")

				decompressed := decompressGzip(t, outputBytes)
				assert.Equal(t, tc.expectedOutput, decompressed, "decompressed output did not match expected")

				if tc.validationFunc != nil {
					tc.validationFunc(t, []byte(tc.input), outputBytes)
				}
			}
		})
	}
}

func runGzipCapability(t *testing.T, input string, params capabilities_domain.CapabilityParams) []byte {
	t.Helper()
	gzipFunc := Gzip()
	stream, err := gzipFunc(context.Background(), strings.NewReader(input), params)
	require.NoError(t, err)
	output, err := io.ReadAll(stream)
	require.NoError(t, err)
	return output
}

func TestParseGzipLevel(t *testing.T) {
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
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.DefaultCompression,
		},
		{
			name:         "should return default when nil params",
			params:       nil,
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.DefaultCompression,
		},
		{
			name:         "should return default for non-numeric level",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "abc"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.DefaultCompression,
		},
		{
			name:         "should return default for empty string level",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: ""},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.DefaultCompression,
		},
		{
			name:         "should return default for level below HuffmanOnly",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "-3"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.DefaultCompression,
		},
		{
			name:         "should return default for level above BestCompression",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "10"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.DefaultCompression,
		},
		{
			name:         "should parse valid HuffmanOnly level",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "-2"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.HuffmanOnly,
		},
		{
			name:         "should parse valid no compression level",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "0"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.NoCompression,
		},
		{
			name:         "should parse valid BestSpeed level",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "1"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.BestSpeed,
		},
		{
			name:         "should parse valid BestCompression level",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "9"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.BestCompression,
		},
		{
			name:         "should parse valid middle level",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "5"},
			defaultLevel: gzip.DefaultCompression,
			expected:     5,
		},
		{
			name:         "should use provided default level",
			params:       capabilities_domain.CapabilityParams{},
			defaultLevel: 7,
			expected:     7,
		},
		{
			name:         "should return default for float value",
			params:       capabilities_domain.CapabilityParams{paramGzipLevel: "3.5"},
			defaultLevel: gzip.DefaultCompression,
			expected:     gzip.DefaultCompression,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseGzipLevel(tc.params, tc.defaultLevel)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGzipWriterFactory(t *testing.T) {
	t.Parallel()

	t.Run("should create a working gzip writer at default level", func(t *testing.T) {
		t.Parallel()
		factory := gzipWriterFactory(gzip.DefaultCompression)
		var buffer bytes.Buffer
		writer, err := factory(&buffer)
		require.NoError(t, err)
		require.NotNil(t, writer)

		_, err = writer.Write([]byte("test data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		decompressed := decompressGzip(t, buffer.Bytes())
		assert.Equal(t, "test data", decompressed)
	})

	t.Run("should create writers at different levels", func(t *testing.T) {
		t.Parallel()
		levels := []int{gzip.NoCompression, gzip.BestSpeed, gzip.DefaultCompression, gzip.BestCompression}
		for _, level := range levels {
			factory := gzipWriterFactory(level)
			var buffer bytes.Buffer
			writer, err := factory(&buffer)
			require.NoError(t, err)
			require.NotNil(t, writer)

			_, err = writer.Write([]byte("test"))
			require.NoError(t, err)

			err = writer.Close()
			require.NoError(t, err)

			decompressed := decompressGzip(t, buffer.Bytes())
			assert.Equal(t, "test", decompressed)
		}
	})
}

func TestGzipPoolsGetPoolForLevel(t *testing.T) {
	t.Parallel()

	t.Run("should return same pool for same level", func(t *testing.T) {
		t.Parallel()
		pools := &gzipPools{
			pools: make(map[int]*sync.Pool),
		}
		pool1 := pools.getPoolForLevel(5)
		pool2 := pools.getPoolForLevel(5)
		assert.Same(t, pool1, pool2)
	})

	t.Run("should return different pools for different levels", func(t *testing.T) {
		t.Parallel()
		pools := &gzipPools{
			pools: make(map[int]*sync.Pool),
		}
		pool1 := pools.getPoolForLevel(1)
		pool2 := pools.getPoolForLevel(9)
		assert.NotSame(t, pool1, pool2)
	})

	t.Run("pool should produce valid gzip writers", func(t *testing.T) {
		t.Parallel()
		pools := &gzipPools{
			pools: make(map[int]*sync.Pool),
		}
		pool := pools.getPoolForLevel(gzip.DefaultCompression)
		item := pool.Get()
		require.NotNil(t, item)

		gw, ok := item.(*gzip.Writer)
		assert.True(t, ok)
		assert.NotNil(t, gw)
		pool.Put(gw)
	})
}

func TestPooledGzipWriterClose(t *testing.T) {
	t.Parallel()

	t.Run("should close writer and return to pool", func(t *testing.T) {
		t.Parallel()
		pool := globalGzipPools.getPoolForLevel(gzip.DefaultCompression)
		var buffer bytes.Buffer

		gw, ok := pool.Get().(*gzip.Writer)
		require.True(t, ok)
		gw.Reset(&buffer)

		pw := &pooledGzipWriter{Writer: gw, pool: pool}
		_, err := pw.Write([]byte("test"))
		require.NoError(t, err)

		err = pw.Close()
		require.NoError(t, err)

		decompressed := decompressGzip(t, buffer.Bytes())
		assert.Equal(t, "test", decompressed)
	})
}

func TestGzipWriterFactory_PoolReturnsNonWriter(t *testing.T) {
	t.Parallel()

	testLevel := gzip.HuffmanOnly
	pool := globalGzipPools.getPoolForLevel(testLevel)

	pool.Put("not a gzip writer")

	factory := gzipWriterFactory(testLevel)
	var buffer bytes.Buffer
	writer, err := factory(&buffer)
	require.NoError(t, err)
	require.NotNil(t, writer)

	_, err = writer.Write([]byte("test data"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	decompressed := decompressGzip(t, buffer.Bytes())
	assert.Equal(t, "test data", decompressed)
}

func TestGzipPoolsGetPoolForLevel_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	pools := &gzipPools{
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

func TestGzipCapabilityWithCancelledContext(t *testing.T) {
	t.Parallel()

	gzipFunc := Gzip()
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	result, err := gzipFunc(ctx, strings.NewReader("data"), nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGzipCapabilityWithSpecificLevels(t *testing.T) {
	t.Parallel()

	gzipFunc := Gzip()
	input := strings.Repeat("a", 500)

	t.Run("should compress with BestSpeed", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{paramGzipLevel: "1"}
		stream, err := gzipFunc(context.Background(), strings.NewReader(input), params)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		decompressed := decompressGzip(t, output)
		assert.Equal(t, input, decompressed)
	})

	t.Run("should compress with BestCompression", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{paramGzipLevel: "9"}
		stream, err := gzipFunc(context.Background(), strings.NewReader(input), params)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		decompressed := decompressGzip(t, output)
		assert.Equal(t, input, decompressed)
	})
}
