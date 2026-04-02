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
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopWriteCloser struct {
	io.Writer
}

func (n *nopWriteCloser) Close() error { return nil }

type errorReader struct {
	Err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.Err
}

type errorWriter struct {
	io.Writer
	ErrOnWrite error
	ErrOnClose error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	if w.ErrOnWrite != nil {
		return 0, w.ErrOnWrite
	}
	return w.Writer.Write(p)
}

func (w *errorWriter) Close() error {
	return w.ErrOnClose
}

type slowReader struct {
	r         io.Reader
	delay     time.Duration
	chunkSize int
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	limit := min(r.chunkSize, len(p))
	n, err = r.r.Read(p[:limit])
	if n > 0 {
		time.Sleep(r.delay)
	}
	return n, err
}

func Test_processStream(t *testing.T) {
	passthroughFactory := func(dst io.Writer) (io.WriteCloser, error) {
		return &nopWriteCloser{Writer: dst}, nil
	}

	testCases := []struct {
		setupCtx       func() (context.Context, context.CancelFunc)
		input          io.Reader
		factory        writerFactory
		name           string
		expectedOutput string
		errorContains  string
		expectError    bool
	}{
		{
			name: "Successful passthrough stream with content",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input:          strings.NewReader("hello world"),
			factory:        passthroughFactory,
			expectedOutput: "hello world",
			expectError:    false,
		},
		{
			name: "Successful empty input stream",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input:          strings.NewReader(""),
			factory:        passthroughFactory,
			expectedOutput: "",
			expectError:    false,
		},
		{
			name: "Successful large input stream",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input:          strings.NewReader(strings.Repeat("a", 5*copyBufferSize)),
			factory:        passthroughFactory,
			expectedOutput: strings.Repeat("a", 5*copyBufferSize),
			expectError:    false,
		},
		{
			name: "Error on factory creation",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input: strings.NewReader("some data"),
			factory: func(dst io.Writer) (io.WriteCloser, error) {
				return nil, errors.New("factory init failed")
			},
			expectError:   true,
			errorContains: "factory init failed",
		},
		{
			name: "Error from source reader",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input:   &errorReader{Err: errors.New("disk read error")},
			factory: passthroughFactory,

			expectError:   true,
			errorContains: "disk read error",
		},
		{
			name: "Error from destination writer on Write",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input: strings.NewReader("this will fail to write"),
			factory: func(dst io.Writer) (io.WriteCloser, error) {
				return &errorWriter{Writer: io.Discard, ErrOnWrite: errors.New("permission denied")}, nil
			},
			expectError:   true,
			errorContains: "permission denied",
		},
		{
			name: "Error from destination writer on Close",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input: strings.NewReader("write succeeds but close fails"),
			factory: func(dst io.Writer) (io.WriteCloser, error) {
				return &errorWriter{Writer: dst, ErrOnClose: errors.New("failed to flush")}, nil
			},

			expectedOutput: "write succeeds but close fails",
			expectError:    true,
			errorContains:  "failed to flush",
		},
		{
			name: "Context cancelled before operation starts",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(fmt.Errorf("test: simulating cancelled context"))
				return ctx, func() { cancel(fmt.Errorf("test: cleanup")) }
			},
			input:         strings.NewReader("this data will never be read"),
			factory:       passthroughFactory,
			expectError:   true,
			errorContains: "context canceled",
		},
		{
			name: "Context times out during a slow read operation",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeoutCause(context.Background(), 50*time.Millisecond, fmt.Errorf("test: slow read timeout"))
			},
			input: &slowReader{
				r:         strings.NewReader(strings.Repeat("abcde", 100)),
				delay:     20 * time.Millisecond,
				chunkSize: 5,
			},
			factory:       passthroughFactory,
			expectError:   true,
			errorContains: "context deadline exceeded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.setupCtx()
			defer cancel()

			outputStream := processStream(ctx, tc.input, tc.factory)
			outputBytes, err := io.ReadAll(outputStream)

			if tc.expectError {
				require.Error(t, err, "expected an error but got none")
				assert.Contains(t, err.Error(), tc.errorContains, "error message did not contain expected text")
			} else {
				require.NoError(t, err, "expected no error but got one")
				assert.Equal(t, tc.expectedOutput, string(outputBytes), "output did not match expected")
			}
		})
	}
}

func TestBufferPoolSafety(t *testing.T) {
	item := bufferPool.Get()
	require.NotNil(t, item, "buffer pool returned a nil item")

	_, ok := item.(*[]byte)
	assert.True(t, ok, "item from buffer pool was not of expected type *[]byte")

	bufferPool.Put(item)
}

func TestWriteAll(t *testing.T) {
	t.Parallel()

	t.Run("should write all bytes successfully", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		data := []byte("hello world")
		n, err := writeAll(&buffer, data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, "hello world", buffer.String())
	})

	t.Run("should write empty data successfully", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		n, err := writeAll(&buffer, []byte{})
		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Empty(t, buffer.String())
	})

	t.Run("should return error when writer fails", func(t *testing.T) {
		t.Parallel()
		writeErr := errors.New("disk full")
		w := &errorWriter{Writer: io.Discard, ErrOnWrite: writeErr}
		n, err := writeAll(w, []byte("data"))
		require.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, writeErr, err)
	})

	t.Run("should return ErrShortWrite when not all bytes written", func(t *testing.T) {
		t.Parallel()
		w := &shortWriter{maxBytes: 3}
		n, err := writeAll(w, []byte("hello"))
		require.Error(t, err)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrShortWrite)
	})
}

type shortWriter struct {
	maxBytes int
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) > w.maxBytes {
		return w.maxBytes, nil
	}
	return len(p), nil
}

func TestCopyWithContext(t *testing.T) {
	t.Parallel()

	t.Run("should copy all data from reader to writer", func(t *testing.T) {
		t.Parallel()
		src := strings.NewReader("hello world")
		var dst bytes.Buffer
		buffer := make([]byte, copyBufferSize)
		n, err := copyWithContext(context.Background(), &dst, src, buffer)
		require.NoError(t, err)
		assert.Equal(t, int64(11), n)
		assert.Equal(t, "hello world", dst.String())
	})

	t.Run("should copy empty data", func(t *testing.T) {
		t.Parallel()
		src := strings.NewReader("")
		var dst bytes.Buffer
		buffer := make([]byte, copyBufferSize)
		n, err := copyWithContext(context.Background(), &dst, src, buffer)
		require.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Empty(t, dst.String())
	})

	t.Run("should return error when context is already cancelled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))
		src := strings.NewReader("data")
		var dst bytes.Buffer
		buffer := make([]byte, copyBufferSize)
		_, err := copyWithContext(ctx, &dst, src, buffer)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should return error from source reader", func(t *testing.T) {
		t.Parallel()
		readErr := errors.New("network error")
		src := &errorReader{Err: readErr}
		var dst bytes.Buffer
		buffer := make([]byte, copyBufferSize)
		_, err := copyWithContext(context.Background(), &dst, src, buffer)
		require.Error(t, err)
		assert.Equal(t, readErr, err)
	})

	t.Run("should return error from destination writer", func(t *testing.T) {
		t.Parallel()
		src := strings.NewReader("data")
		writeErr := errors.New("disk full")
		dst := &errorWriter{Writer: io.Discard, ErrOnWrite: writeErr}
		buffer := make([]byte, copyBufferSize)
		_, err := copyWithContext(context.Background(), dst, src, buffer)
		require.Error(t, err)
		assert.Equal(t, writeErr, err)
	})

	t.Run("should handle large data", func(t *testing.T) {
		t.Parallel()
		largeData := strings.Repeat("x", 3*copyBufferSize+17)
		src := strings.NewReader(largeData)
		var dst bytes.Buffer
		buffer := make([]byte, copyBufferSize)
		n, err := copyWithContext(context.Background(), &dst, src, buffer)
		require.NoError(t, err)
		assert.Equal(t, int64(len(largeData)), n)
		assert.Equal(t, largeData, dst.String())
	})

	t.Run("should return short write error from destination", func(t *testing.T) {
		t.Parallel()
		src := strings.NewReader("hello world")
		dst := &shortWriter{maxBytes: 3}
		buffer := make([]byte, copyBufferSize)
		_, err := copyWithContext(context.Background(), dst, src, buffer)
		require.Error(t, err)
		assert.ErrorIs(t, err, io.ErrShortWrite)
	})
}
