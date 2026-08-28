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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	throttleTestTimeout = 5 * time.Second
)

func serveWithinTimeout(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)

		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	}()

	select {
	case <-done:
	case <-time.After(throttleTestTimeout):
		t.Fatalf("the request to %s was still waiting for a concurrency slot", path)
	}

	return recorder
}

func TestStreamAwareThrottleReleasesAStreamsSlotOnFirstWrite(t *testing.T) {
	t.Parallel()

	throttle := streamAwareThrottle(1, throttleTestTimeout)

	streaming := make(chan struct{})
	finish := make(chan struct{})

	stream := throttle(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")

		_, _ = writer.Write([]byte(": open\n\n"))
		close(streaming)
		<-finish
	}))

	page := throttle(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	var streamDone sync.WaitGroup
	streamDone.Go(func() {
		stream.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
	})

	<-streaming

	recorder := serveWithinTimeout(t, page, "/page")
	assert.Equal(t, http.StatusOK, recorder.Code,
		"an open stream must not hold a concurrency slot")

	close(finish)
	streamDone.Wait()
}

func TestStreamAwareThrottleReleasesAStreamsSlotOnWriteHeader(t *testing.T) {
	t.Parallel()

	throttle := streamAwareThrottle(1, throttleTestTimeout)

	streaming := make(chan struct{})
	finish := make(chan struct{})

	stream := throttle(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.WriteHeader(http.StatusOK)
		close(streaming)
		<-finish
	}))

	page := throttle(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	var streamDone sync.WaitGroup
	streamDone.Go(func() {
		stream.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
	})

	<-streaming

	recorder := serveWithinTimeout(t, page, "/page")
	assert.Equal(t, http.StatusOK, recorder.Code)

	close(finish)
	streamDone.Wait()
}

func TestStreamAwareThrottleHoldsTheSlotForAnOrdinaryRequest(t *testing.T) {
	t.Parallel()

	throttle := streamAwareThrottle(1, throttleTestTimeout)

	entered := make(chan struct{}, 2)
	release := make(chan struct{})

	slow := throttle(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release

		writer.WriteHeader(http.StatusOK)
	}))

	var everyone sync.WaitGroup
	everyone.Go(func() {
		slow.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/slow", nil))
	})

	<-entered

	secondEntered := make(chan struct{})
	everyone.Go(func() {
		defer close(secondEntered)

		slow.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/slow", nil))
	})

	select {
	case <-entered:
		t.Fatal("the second request entered the handler while the only slot was held")
	case <-secondEntered:
		t.Fatal("the second request completed while the only slot was held")
	case <-time.After(200 * time.Millisecond):
	}

	close(release)

	select {
	case <-secondEntered:
	case <-time.After(throttleTestTimeout):
		t.Fatal("the second request never ran after the slot was freed")
	}

	everyone.Wait()
}

func TestStreamAwareThrottleDisabled(t *testing.T) {
	t.Parallel()

	handler := streamAwareThrottle(0, 0)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTeapot)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusTeapot, recorder.Code)
}

func TestStreamAwareThrottleUnwrapsForResponseController(t *testing.T) {
	t.Parallel()

	var wrapped http.ResponseWriter

	handler := streamAwareThrottle(1, throttleTestTimeout)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		wrapped = writer
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	unwrapper, ok := wrapped.(interface{ Unwrap() http.ResponseWriter })
	require.True(t, ok, "the wrapper must expose the writer it wraps")
	assert.Same(t, recorder, unwrapper.Unwrap())
}

func TestStreamAwareThrottleShedsABurst(t *testing.T) {
	t.Parallel()

	const burst = 40

	throttle := streamAwareThrottle(2, 50*time.Millisecond)

	release := make(chan struct{})
	slow := throttle(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		<-release

		writer.WriteHeader(http.StatusOK)
	}))

	statuses := make(chan int, burst)

	var everyone sync.WaitGroup
	for range burst {
		everyone.Go(func() {
			recorder := httptest.NewRecorder()
			slow.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/slow", nil))
			statuses <- recorder.Code

			if recorder.Code == http.StatusTooManyRequests {
				assert.NotEmpty(t, recorder.Header().Get("Retry-After"),
					"a shed request must say when to try again")
			}
		})
	}

	shed := 0

	for range burst - 2 {
		select {
		case status := <-statuses:
			if status == http.StatusTooManyRequests {
				shed++
			}
		case <-time.After(throttleTestTimeout):
			close(release)
			everyone.Wait()
			t.Fatal("the burst was absorbed into an unbounded queue instead of being shed")
		}
	}

	assert.Positive(t, shed, "a burst beyond the limit must be shed, not queued without bound")

	close(release)
	everyone.Wait()
}

func TestStreamAwareThrottleShedsAfterWaitingTooLong(t *testing.T) {
	t.Parallel()

	throttle := streamAwareThrottle(1, 50*time.Millisecond)

	entered := make(chan struct{}, 1)
	release := make(chan struct{})

	slow := throttle(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release

		writer.WriteHeader(http.StatusOK)
	}))

	var everyone sync.WaitGroup
	everyone.Go(func() {
		slow.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/slow", nil))
	})
	<-entered

	recorder := serveWithinTimeout(t, slow, "/slow")
	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assert.Equal(t, "1", recorder.Header().Get("Retry-After"))

	close(release)
	everyone.Wait()
}
