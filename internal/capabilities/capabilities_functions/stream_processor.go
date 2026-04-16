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
	"errors"
	"io"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// copyBufferSize defines the size of the buffer used for streaming data
	// through the processing pipe. 32KB is a common and effective size.
	copyBufferSize = 32 * 1024
)

// bufferPool reuses byte-slice buffers to reduce allocation pressure during
// stream processing.
var bufferPool = sync.Pool{
	New: func() any {
		return new(make([]byte, copyBufferSize))
	},
}

// writerFactory is a function type for creating streaming io.WriteCloser
// wrappers, such as gzip.Writer or brotli.Writer, around a destination writer.
type writerFactory func(dst io.Writer) (io.WriteCloser, error)

// processStream sets up a cancellable, streaming data transformation pipeline.
// It creates an io.Pipe and runs a goroutine to pull data from the input,
// transform it using a writer from the factory, and push it into the pipe.
//
// Takes inputData (io.Reader) which provides the source data to transform.
// Takes factory (writerFactory) which creates the writer for transformation.
//
// Returns io.Reader which is the reading end of the pipe for the consumer.
//
// The spawned goroutine runs until the copy completes, the context is
// cancelled, or an error occurs. Errors are propagated through the pipe.
func processStream(ctx context.Context, inputData io.Reader, factory writerFactory) io.Reader {
	ctx, l := logger_domain.From(ctx, log)
	pr, pw := io.Pipe()

	go func() {
		defer func() { _ = pw.Close() }()

		streamingWriter, err := factory(pw)
		if err != nil {
			l.Error("Failed to create streaming writer for pipe", logger_domain.Error(err))
			_ = pw.CloseWithError(err)
			return
		}

		bufferPointer, ok := bufferPool.Get().(*[]byte)
		if !ok {
			err := errors.New("invalid type in buffer pool")
			l.Error("Buffer pool corruption detected", logger_domain.Error(err))
			_ = pw.CloseWithError(err)
			return
		}
		defer bufferPool.Put(bufferPointer)

		_, copyErr := copyWithContext(ctx, streamingWriter, inputData, *bufferPointer)

		closeErr := streamingWriter.Close()

		if copyErr != nil {
			_ = pw.CloseWithError(copyErr)

			if !errors.Is(copyErr, context.Canceled) && !errors.Is(copyErr, context.DeadlineExceeded) {
				l.Error("Failed during streaming data copy", logger_domain.Error(copyErr))
			}
			return
		}

		if closeErr != nil {
			l.Error("Failed closing streaming writer", logger_domain.Error(closeErr))
			_ = pw.CloseWithError(closeErr)
			return
		}
	}()

	return pr
}

// copyWithContext performs an io.Copy-like operation that is
// cancellable via the context, reading from src, writing to dst,
// and checking for cancellation before each read.
//
// Takes ctx (context.Context) which controls cancellation of the copy
// operation.
// Takes destination (io.Writer) which is the destination to write data to.
// Takes source (io.Reader) which is the source to read data from.
//
// Returns int64 which is the total number of bytes written.
// Returns error when the context is cancelled, a read fails, or a write
// fails.
//
// loop requires nested control flow.
//
//nolint:revive // core streaming dispatch
func copyWithContext(ctx context.Context, destination io.Writer, source io.Reader, buffer []byte) (int64, error) {
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		nr, readErr := source.Read(buffer)
		if nr > 0 {
			n, err := writeAll(destination, buffer[:nr])
			written += int64(n)
			if err != nil {
				return written, err
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return written, nil
			}
			return written, readErr
		}
	}
}

// writeAll writes all bytes from a buffer to a destination.
//
// Takes destination (io.Writer) which is the destination to write to.
// Takes data ([]byte) which is the buffer to write.
//
// Returns int which is the number of bytes written.
// Returns error when the write fails or not all bytes are written.
func writeAll(destination io.Writer, data []byte) (int, error) {
	nw, err := destination.Write(data)
	if err != nil {
		return nw, err
	}
	if nw != len(data) {
		return nw, io.ErrShortWrite
	}
	return nw, nil
}
