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
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"

	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// brotliWriterPool is a pool of brotli writers for reuse across compress operations. It
	// uses quality level 4 and window size 20.
	brotliWriterPool = sync.Pool{
		New: func() any {
			opts := brotli.WriterOptions{
				Quality: 4,
				LGWin:   20,
			}
			return brotli.NewWriterOptions(nil, opts)
		},
	}

	// gzipWriterPool is a pool of gzip writers to reduce memory allocation.
	gzipWriterPool = sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			return w
		},
	}

	// compressedResponseWriterPool provides reusable compressedResponseWriter structs.
	compressedResponseWriterPool = sync.Pool{
		New: func() any {
			return &compressedResponseWriter{}
		},
	}
)

// compressedResponseWriter wraps an http.ResponseWriter with compression. It implements
// io.Writer.
type compressedResponseWriter struct {
	http.ResponseWriter

	// compressor writes compressed data to the underlying response writer.
	compressor io.WriteCloser
}

// Write writes compressed data to the underlying response writer.
//
// Takes p ([]byte) which contains the data to compress and write.
//
// Returns int which is the number of bytes written.
// Returns error when the write to the underlying compressor fails.
func (w *compressedResponseWriter) Write(p []byte) (int, error) {
	return w.compressor.Write(p)
}

// Flush sends any buffered compressed data to the client. Required for HTTP/2 features
// such as 103 Early Hints that need to flush intermediate responses before the main body.
func (w *compressedResponseWriter) Flush() {
	if flusher, ok := w.compressor.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// determineCompression determines the encoding and capability based on the
// Accept-Encoding header.
//
// Takes acceptEncoding (string) which is the value of the Accept-Encoding header from the
// HTTP request.
//
// Returns string which is the encoding name (e.g. "br", "gzip") or empty if no supported
// encoding is found.
// Returns capabilities_dto.Capability which indicates the compression capability to use.
func determineCompression(acceptEncoding string) (string, capabilities_dto.Capability) {
	if strings.Contains(acceptEncoding, encodingBrotli) {
		return "br", capabilities_dto.CapabilityCompressBrotli
	}
	if strings.Contains(acceptEncoding, "gzip") {
		return "gzip", capabilities_dto.CapabilityCompressGzip
	}
	return "", ""
}

// getCompressedResponseWriter gets a compressedResponseWriter from the pool.
//
// Takes w (http.ResponseWriter) which is the underlying response writer.
// Takes compressor (io.WriteCloser) which compresses the response data.
//
// Returns *compressedResponseWriter which wraps the response writer with compression
// support.
func getCompressedResponseWriter(w http.ResponseWriter, compressor io.WriteCloser) *compressedResponseWriter {
	crw, ok := compressedResponseWriterPool.Get().(*compressedResponseWriter)
	if !ok {
		crw = &compressedResponseWriter{}
	}
	crw.ResponseWriter = w
	crw.compressor = compressor
	return crw
}

// releaseCompressedResponseWriter returns a compressedResponseWriter to the pool for
// reuse.
//
// Takes crw (*compressedResponseWriter) which is the writer to release.
func releaseCompressedResponseWriter(crw *compressedResponseWriter) {
	crw.ResponseWriter = nil
	crw.compressor = nil
	compressedResponseWriterPool.Put(crw)
}

// setupBrotliCompressor gets a brotli writer from the pool and sets the response headers
// for brotli compression.
//
// Takes w (http.ResponseWriter) which receives the compression headers.
//
// Returns io.WriteCloser which wraps the response in brotli compression, or nil when a
// pool writer cannot be obtained.
func setupBrotliCompressor(ctx context.Context, w http.ResponseWriter) io.WriteCloser {
	bw, ok := brotliWriterPool.Get().(*brotli.Writer)
	if !ok {
		_, l := logger_domain.From(ctx, log)
		l.Error("Failed to get brotli writer from pool")
		return nil
	}
	bw.Reset(w)
	w.Header()[headerContentEncoding] = headerValEncodingBrotli
	return bw
}

// setupGzipCompressor gets a gzip writer from a pool and sets the response headers for
// gzip encoding.
//
// Takes w (http.ResponseWriter) which receives the compression headers.
//
// Returns io.WriteCloser which wraps the response in gzip compression, or nil when a pool
// writer cannot be obtained.
func setupGzipCompressor(ctx context.Context, w http.ResponseWriter) io.WriteCloser {
	gw, ok := gzipWriterPool.Get().(*gzip.Writer)
	if !ok {
		_, l := logger_domain.From(ctx, log)
		l.Error("Failed to get gzip writer from pool")
		return nil
	}
	gw.Reset(w)
	w.Header()[headerContentEncoding] = headerValEncodingGzip
	return gw
}

// releaseCompressor returns a compressor to its pool for reuse.
//
// Takes compressor (io.WriteCloser) which is the compressor to return.
// Takes isBrotli (bool) which indicates whether this is a Brotli or gzip compressor.
func releaseCompressor(compressor io.WriteCloser, isBrotli bool) {
	if compressor == nil {
		return
	}
	_ = compressor.Close()
	if isBrotli {
		(compressor.(*brotli.Writer)).Reset(nil)
		brotliWriterPool.Put(compressor)
	} else {
		(compressor.(*gzip.Writer)).Reset(nil)
		gzipWriterPool.Put(compressor)
	}
}

// handleNonCacheableStream serves a response with streaming compression.
//
// When the client accepts Brotli or Gzip encoding, the response is compressed on the fly.
// When no compression is requested, the response is served uncompressed.
//
// Takes w (http.ResponseWriter) which receives the compressed response.
// Takes r (*http.Request) which provides the incoming request and headers.
// Takes next (http.Handler) which is the handler to serve the actual content.
func handleNonCacheableStream(w http.ResponseWriter, r *http.Request, next http.Handler) {
	acceptEncoding := r.Header.Get("Accept-Encoding")

	var compressor io.WriteCloser
	var isBrotli bool

	switch {
	case strings.Contains(acceptEncoding, encodingBrotli):
		compressor = setupBrotliCompressor(r.Context(), w)
		isBrotli = true
	case strings.Contains(acceptEncoding, encodingGzip):
		compressor = setupGzipCompressor(r.Context(), w)
	default:
		next.ServeHTTP(w, r)
		return
	}

	if compressor == nil {
		next.ServeHTTP(w, r)
		return
	}
	defer releaseCompressor(compressor, isBrotli)

	wrappedWriter := getCompressedResponseWriter(w, compressor)
	defer releaseCompressedResponseWriter(wrappedWriter)

	next.ServeHTTP(wrappedWriter, r)
}
