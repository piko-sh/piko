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

package daemon_adapters

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
)

const (
	// compressionThresholdBytes is the smallest body worth compressing.
	compressionThresholdBytes = 1024

	// maxResponseWriterUnwrapDepth bounds how far baseResponseWriter follows a chain.
	maxResponseWriterUnwrapDepth = 16
)

var (
	// compressibleActionMediaTypes lists the response media types worth compressing.
	compressibleActionMediaTypes = map[string]struct{}{
		"application/json":       {},
		"application/javascript": {},
		"application/xml":        {},
		"text/plain":             {},
		"text/html":              {},
		"text/css":               {},
		"text/xml":               {},
		"image/svg+xml":          {},
	}
)

// actionCompressWriter defers the decision to compress until the handler has declared its
// Content-Type.
type actionCompressWriter struct {
	// ResponseWriter is the underlying writer the response is written to.
	http.ResponseWriter

	// request supplies the request context, which the pooled compressor helpers use to
	// report a pool failure. It is read only for that.
	request *http.Request

	// compressor wraps the underlying writer once the decision to compress is taken.
	compressor io.WriteCloser

	// encoding is the negotiated Content-Encoding the client accepts.
	encoding string

	// pending holds the start of the body while the writer decides whether it is large
	// enough to be worth compressing.
	pending []byte

	// status is the status the handler declared, held back until the decision is settled.
	status int

	// once guards release so a deferred call after an explicit one is a no-op.
	once sync.Once

	// buffering reports that the body is being held back pending the size decision.
	buffering bool

	// headerWritten reports that the status line has gone out, after which no header change
	// can reach the client.
	headerWritten bool

	// decided records that the compress-or-not decision has been taken.
	decided bool

	// isBrotli selects which pool the compressor is returned to.
	isBrotli bool
}

// actionCompressMiddleware negotiates gzip or brotli for action responses.
//
// Takes enabled (bool) which turns the middleware into a pass-through when false.
//
// Returns func(http.Handler) http.Handler which is the middleware.
func actionCompressMiddleware(enabled bool) func(http.Handler) http.Handler {
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if !slices.Contains(writer.Header().Values(headerVary), headerAcceptEncoding) {
				writer.Header().Add(headerVary, headerAcceptEncoding)
			}

			encoding, _ := determineCompression(request.Header.Get(headerAcceptEncoding))
			if encoding == "" {
				next.ServeHTTP(writer, request)

				return
			}

			compressing := &actionCompressWriter{ResponseWriter: writer, encoding: encoding, request: request}
			defer compressing.release()

			next.ServeHTTP(compressing, request)
		})
	}
}

// WriteHeader takes the compression decision and writes the status.
//
// Takes status (int) which is the response status code.
func (w *actionCompressWriter) WriteHeader(status int) {
	if w.headerWritten {
		return
	}

	w.status = status

	if status < http.StatusOK {
		w.ResponseWriter.WriteHeader(status)

		return
	}

	if w.shouldConsiderCompressing() {
		w.buffering = true

		return
	}

	w.decide()
	w.emitHeader()
}

// Write takes the compression decision and writes the body through the compressor when
// one was selected.
//
// Takes data ([]byte) which is the body to write.
//
// Returns int which is the number of bytes accepted.
// Returns error which is any write failure.
func (w *actionCompressWriter) Write(data []byte) (int, error) {
	if !w.headerWritten && !w.buffering {
		w.WriteHeader(http.StatusOK)
	}

	if w.buffering {
		w.pending = append(w.pending, data...)
		if len(w.pending) < compressionThresholdBytes {
			return len(data), nil
		}

		return len(data), w.commitBuffered()
	}

	if w.compressor != nil {
		return w.compressor.Write(data)
	}

	return w.ResponseWriter.Write(data)
}

// Flush pushes any buffered output to the client, flushing the compressor first so a
// partially written frame is not stranded inside it.
func (w *actionCompressWriter) Flush() {
	if !w.headerWritten && !w.buffering {
		w.WriteHeader(http.StatusOK)
	}
	if w.buffering {
		_ = w.commitBuffered()
	}
	w.emitHeader()

	if flusher, ok := w.compressor.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}

	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Hijack surrenders the connection, returning any compressor to its pool first so nothing
// later writes a trailer onto a socket this writer no longer owns.
//
// Returns net.Conn which is the underlying connection.
// Returns *bufio.ReadWriter which buffers that connection.
// Returns error when the underlying writer does not support hijacking.
func (w *actionCompressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.buffering = false
	w.pending = nil
	w.decided = true
	w.headerWritten = true
	w.release()

	return http.NewResponseController(w.ResponseWriter).Hijack()
}

// Unwrap exposes the underlying writer so http.ResponseController can reach the
// connection, which the streaming write deadline depends on.
//
// Returns http.ResponseWriter which is the writer this one wraps.
func (w *actionCompressWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// shouldConsiderCompressing reports whether the declared response could be compressed, so
// its first bytes are worth buffering to see whether it is large enough to be worth it.
//
// Returns bool which is true when the response is a compressible media type that has not
// already been encoded.
func (w *actionCompressWriter) shouldConsiderCompressing() bool {
	if w.status == http.StatusNoContent || w.status == http.StatusNotModified {
		return false
	}
	if w.Header().Get(headerContentEncoding) != "" {
		return false
	}

	return isCompressibleActionMediaType(w.Header().Get(headerContentType))
}

// commitBuffered takes the compression decision now that enough of the body is known,
// then emits the headers and everything buffered so far.
//
// Returns error which is any failure writing the buffered body.
func (w *actionCompressWriter) commitBuffered() error {
	w.buffering = false
	w.decide()
	w.emitHeader()

	buffered := w.pending
	w.pending = nil

	if len(buffered) == 0 {
		return nil
	}

	var err error
	if w.compressor != nil {
		_, err = w.compressor.Write(buffered)
	} else {
		_, err = w.ResponseWriter.Write(buffered)
	}

	return err
}

// emitHeader sends the status line once the compression decision is settled.
func (w *actionCompressWriter) emitHeader() {
	if w.headerWritten {
		return
	}
	w.headerWritten = true

	w.ResponseWriter.WriteHeader(w.status)
}

// decide selects a compressor on the first write, or records that the response is served
// through unchanged.
//
// Takes status (int) which is the response status; an informational response is forwarded
// without forcing the decision, since its headers are not the final ones.
func (w *actionCompressWriter) decide() {
	if w.decided {
		return
	}
	w.decided = true

	if !w.shouldConsiderCompressing() {
		return
	}
	if len(w.pending) < compressionThresholdBytes {
		return
	}

	w.Header().Del(headerContentLength)

	if w.encoding == encodingBrotli {
		w.compressor = setupBrotliCompressor(w.request.Context(), w.ResponseWriter)
		w.isBrotli = true

		return
	}

	w.compressor = setupGzipCompressor(w.request.Context(), w.ResponseWriter)
}

// release returns the compressor to its pool, flushing its trailer. It is a no-op when
// the response was served uncompressed.
func (w *actionCompressWriter) release() {
	w.once.Do(func() {
		if w.buffering {
			w.buffering = false
			w.decide()
			w.emitHeader()

			if len(w.pending) > 0 {
				_, _ = w.ResponseWriter.Write(w.pending)
				w.pending = nil
			}
		}

		w.emitHeader()

		if w.compressor != nil {
			releaseCompressor(w.compressor, w.isBrotli)
			w.compressor = nil
		}
	})
}

// isCompressibleActionMediaType reports whether a Content-Type is worth compressing.
//
// Takes contentType (string) which is the raw Content-Type header value.
//
// Returns bool which is true when the media type should be compressed.
func isCompressibleActionMediaType(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if mediaType == "" {
		return false
	}

	_, ok := compressibleActionMediaTypes[mediaType]

	return ok
}

// baseResponseWriter unwraps middleware writers to reach the one the server owns.
//
// Takes writer (http.ResponseWriter) which may be wrapped.
//
// Returns http.ResponseWriter which is the innermost writer we can reach.
func baseResponseWriter(writer http.ResponseWriter) http.ResponseWriter {
	for range maxResponseWriterUnwrapDepth {
		unwrapper, ok := writer.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return writer
		}

		inner := unwrapper.Unwrap()
		if inner == nil || inner == writer {
			return writer
		}
		writer = inner
	}

	return writer
}
