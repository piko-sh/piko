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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/security/security_dto"
)

type hijackRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true

	return nil, nil, nil
}

func serveThroughCompression(acceptEncoding string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.Action", nil)
	if acceptEncoding != "" {
		request.Header.Set(headerAcceptEncoding, acceptEncoding)
	}
	recorder := httptest.NewRecorder()
	actionCompressMiddleware(true)(handler).ServeHTTP(recorder, request)

	return recorder
}

func TestActionCompress_LeavesAnEventStreamUncompressed(t *testing.T) {
	t.Parallel()

	recorder := serveThroughCompression("br, gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, mediaTypeEventStream)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
		flusher, ok := w.(http.Flusher)
		require.True(t, ok, "a streaming action must still find a Flusher through the wrapper")
		flusher.Flush()
	})

	assert.Empty(t, recorder.Header().Get(headerContentEncoding),
		"compressing a stream withholds every event until the buffer fills, which reads as a hang")
	assert.Equal(t, "data: hello\n\n", recorder.Body.String())
	assert.True(t, recorder.Flushed, "the flush must reach the underlying writer")
}

func TestActionCompress_PreservesFlusherAndHijacker(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.Action", nil)
	request.Header.Set(headerAcceptEncoding, "gzip")
	recorder := &hijackRecorder{ResponseRecorder: httptest.NewRecorder()}

	actionCompressMiddleware(true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, isFlusher := w.(http.Flusher)
		assert.True(t, isFlusher, "the wrapper must satisfy http.Flusher")

		hijacker, isHijacker := w.(http.Hijacker)
		require.True(t, isHijacker, "the wrapper must satisfy http.Hijacker")

		_, _, err := hijacker.Hijack()
		assert.NoError(t, err)
	})).ServeHTTP(recorder, request)

	assert.True(t, recorder.hijacked, "Hijack must reach the underlying writer")
}

func TestActionCompress_CompressesJSONWithBrotli(t *testing.T) {
	t.Parallel()

	payload := `{"data":"` + strings.Repeat("piko", 8192) + `"}`

	recorder := serveThroughCompression("br, gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(payload))
	})

	require.Equal(t, encodingBrotli, recorder.Header().Get(headerContentEncoding))
	assert.Equal(t, headerAcceptEncoding, recorder.Header().Get(headerVary))
	assert.Less(t, recorder.Body.Len(), len(payload)/4, "a repetitive JSON body should compress hard")

	decoded, err := io.ReadAll(brotli.NewReader(recorder.Body))
	require.NoError(t, err)
	assert.Equal(t, payload, string(decoded))
}

func TestActionCompress_FallsBackToGzip(t *testing.T) {
	t.Parallel()

	payload := `{"data":"` + strings.Repeat("piko", 4096) + `"}`

	recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(payload))
	})

	require.Equal(t, encodingGzip, recorder.Header().Get(headerContentEncoding))

	reader, err := gzip.NewReader(recorder.Body)
	require.NoError(t, err)
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, payload, string(decoded))
}

func TestActionCompress_SkipsAResponseThatIsAlreadyEncoded(t *testing.T) {
	t.Parallel()

	recorder := serveThroughCompression("br, gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		w.Header().Set(headerContentEncoding, encodingGzip)
		_, _ = w.Write([]byte("already-compressed-bytes"))
	})

	assert.Equal(t, []string{encodingGzip}, recorder.Header().Values(headerContentEncoding),
		"a handler that encoded its own body must not be encoded twice")
	assert.Equal(t, "already-compressed-bytes", recorder.Body.String())
}

func TestActionCompress_DropsContentLengthWhenCompressing(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("a", 4096)

	recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		w.Header().Set(headerContentLength, "4096")
		_, _ = w.Write([]byte(payload))
	})

	assert.Empty(t, recorder.Header().Get(headerContentLength),
		"a stale Content-Length would describe the uncompressed body")
}

func TestActionCompress_ServesThroughWhenTheClientAcceptsNothing(t *testing.T) {
	t.Parallel()

	recorder := serveThroughCompression("", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	assert.Empty(t, recorder.Header().Get(headerContentEncoding))
	assert.Equal(t, `{"ok":true}`, recorder.Body.String())
	assert.Equal(t, headerAcceptEncoding, recorder.Header().Get(headerVary),
		"Vary must be set regardless, or a shared cache may serve the wrong encoding")
}

func TestActionCompress_DisabledServesThrough(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.Action", nil)
	request.Header.Set(headerAcceptEncoding, "br, gzip")
	recorder := httptest.NewRecorder()

	actionCompressMiddleware(false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})).ServeHTTP(recorder, request)

	assert.Empty(t, recorder.Header().Get(headerContentEncoding))
	assert.Equal(t, `{"ok":true}`, recorder.Body.String())
}

func TestActionHandlerMount_CompressesTheBatchEndpoint(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.compressResponses = true
	registerMetadataAction(handler, "bulky", func(_ any) (any, error) {
		return strings.Repeat("piko", 8192), nil
	})

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")

	request := batchRequestFor(false, "bulky")
	request.Header.Set(headerAcceptEncoding, "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, encodingGzip, recorder.Header().Get(headerContentEncoding),
		"the batch endpoint returns the largest payloads and must be compressed too")

	reader, err := gzip.NewReader(recorder.Body)
	require.NoError(t, err)
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Contains(t, string(decoded), `"success":true`)
}

func TestActionHandlerMount_LeavesResponsesUncompressedWhenDisabled(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	registerMetadataAction(handler, "bulky", func(_ any) (any, error) {
		return strings.Repeat("piko", 8192), nil
	})

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")

	request := batchRequestFor(false, "bulky")
	request.Header.Set(headerAcceptEncoding, "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Header().Get(headerContentEncoding))
	assert.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestActionCompress_FlushBeforeAnyWriteLeavesTheBodyUnencoded(t *testing.T) {
	t.Parallel()

	recorder := serveThroughCompression("br, gzip", func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		flusher.Flush()

		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	assert.Empty(t, recorder.Header().Get(headerContentEncoding),
		"a flush sends the headers, so a later decision would label encoded bytes as identity")
	assert.Equal(t, `{"ok":true}`, recorder.Body.String())
}

func TestActionCompress_DoesNotDuplicateVary(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.Action", nil)
	request.Header.Set(headerAcceptEncoding, "gzip")
	recorder := httptest.NewRecorder()
	recorder.Header().Add(headerVary, headerAcceptEncoding)

	actionCompressMiddleware(true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})).ServeHTTP(recorder, request)

	assert.Equal(t, []string{headerAcceptEncoding}, recorder.Header().Values(headerVary))
}

func TestActionCompress_SkipsNonCompressibleMediaTypes(t *testing.T) {
	t.Parallel()

	for _, mediaType := range []string{"image/png", "application/octet-stream", "video/mp4"} {
		recorder := serveThroughCompression("br, gzip", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(headerContentType, mediaType)
			_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
		})

		assert.Empty(t, recorder.Header().Get(headerContentEncoding),
			"%s is already compact or already compressed", mediaType)
	}
}

func TestActionCompress_HandlesAContentTypeWithParameters(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("piko", 4096)

	recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, "APPLICATION/JSON; charset=utf-8")
		_, _ = w.Write([]byte(payload))
	})

	assert.Equal(t, encodingGzip, recorder.Header().Get(headerContentEncoding),
		"a charset parameter and casing must not defeat the media type match")
}

func TestActionCompress_LeavesAnEmptyContentTypeUncompressed(t *testing.T) {
	t.Parallel()

	recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
	})

	assert.Empty(t, recorder.Header().Get(headerContentEncoding),
		"sniffing cannot work once the bytes have been replaced by compressed output")
}

func TestActionCompress_LeavesEmptyStatusesUncompressed(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusNoContent, http.StatusNotModified} {
		recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(headerContentType, contentTypeJSON)
			w.WriteHeader(status)
		})

		assert.Empty(t, recorder.Header().Get(headerContentEncoding), "status %d has no body", status)
	}
}

func TestActionCompress_LeavesASmallBodyUncompressed(t *testing.T) {
	t.Parallel()

	payload := `{"ok":true}`

	recorder := serveThroughCompression("br, gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(payload))
	})

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Header().Get(headerContentEncoding),
		"framing a short reply makes it bigger and still costs a pooled encoder")
	assert.Equal(t, payload, recorder.Body.String())
}

func TestActionCompress_CompressesOnceThePayloadIsWorthIt(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("piko", compressionThresholdBytes)

	recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		_, _ = w.Write([]byte(payload))
	})

	require.Equal(t, encodingGzip, recorder.Header().Get(headerContentEncoding))

	reader, err := gzip.NewReader(recorder.Body)
	require.NoError(t, err)
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, payload, string(decoded))
}

func TestActionCompress_PreservesAnExplicitStatusOnASmallBody(t *testing.T) {
	t.Parallel()

	recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	})

	assert.Equal(t, http.StatusCreated, recorder.Code,
		"holding the header back must not lose the status the handler chose")
	assert.Equal(t, `{"created":true}`, recorder.Body.String())
}

func TestActionCompress_WritesASmallBodyWrittenInManyPieces(t *testing.T) {
	t.Parallel()

	recorder := serveThroughCompression("gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		for _, piece := range []string{`{"a":`, `1,`, `"b":2}`} {
			_, _ = w.Write([]byte(piece))
		}
	})

	assert.Equal(t, `{"a":1,"b":2}`, recorder.Body.String(),
		"a buffered body must be emitted in full when the response ends below the threshold")
}

type selfUnwrappingWriter struct {
	http.ResponseWriter
}

type nilUnwrappingWriter struct {
	http.ResponseWriter
}

type pairedUnwrappingWriter struct {
	http.ResponseWriter
	partner http.ResponseWriter
}

func (p *pairedUnwrappingWriter) Unwrap() http.ResponseWriter {
	return p.partner
}

func (s *selfUnwrappingWriter) Unwrap() http.ResponseWriter {
	return s
}

func (n *nilUnwrappingWriter) Unwrap() http.ResponseWriter {
	return nil
}

func TestBaseResponseWriter_ReachesTheWriterTheServerOwns(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	assert.Same(t, recorder, baseResponseWriter(recorder),
		"an unwrapped writer is already the base")

	compressing := &actionCompressWriter{ResponseWriter: recorder, encoding: encodingGzip}
	assert.Same(t, recorder, baseResponseWriter(compressing))
	assert.Same(t, recorder, compressing.Unwrap())

	nested := &actionCompressWriter{ResponseWriter: compressing, encoding: encodingBrotli}
	assert.Same(t, recorder, baseResponseWriter(nested),
		"unwrapping must continue through every layer")

	dishonest := &nilUnwrappingWriter{ResponseWriter: recorder}
	assert.Same(t, dishonest, baseResponseWriter(dishonest),
		"a writer that unwraps to nothing is the last one we can reach")
}

func TestBaseResponseWriter_DoesNotSpinOnAWriterThatUnwrapsToItself(t *testing.T) {
	t.Parallel()

	looping := &selfUnwrappingWriter{ResponseWriter: httptest.NewRecorder()}

	done := make(chan http.ResponseWriter, 1)
	go func() { done <- baseResponseWriter(looping) }()

	select {
	case reached := <-done:
		assert.Same(t, looping, reached)
	case <-time.After(time.Second):
		require.Fail(t, "baseResponseWriter never returned for a self-referential writer")
	}
}

func TestBaseResponseWriter_DoesNotSpinOnACycleOfWrappers(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	first := &pairedUnwrappingWriter{ResponseWriter: recorder}
	second := &pairedUnwrappingWriter{ResponseWriter: recorder, partner: first}
	first.partner = second

	done := make(chan http.ResponseWriter, 1)
	go func() { done <- baseResponseWriter(first) }()

	select {
	case reached := <-done:
		assert.NotNil(t, reached, "the walk must give up rather than never return")
	case <-time.After(time.Second):
		require.Fail(t, "baseResponseWriter never returned for a cycle of wrappers")
	}
}
