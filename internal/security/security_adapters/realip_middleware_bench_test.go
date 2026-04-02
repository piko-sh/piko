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

//go:build bench

package security_adapters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

func BenchmarkRealIPMiddleware_DirectConnection(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	middleware := NewRealIPMiddleware(extractor)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))

	request := httptest.NewRequest("GET", "/", nil)
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{}))
	request.RemoteAddr = "192.168.1.100:54321"
	recorder := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkRealIPMiddleware_TrustedProxy_XFF(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	middleware := NewRealIPMiddleware(extractor)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))

	request := httptest.NewRequest("GET", "/", nil)
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{}))
	request.RemoteAddr = "10.0.0.1:54321"
	request.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")
	recorder := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkRealIPMiddleware_TrustedProxy_XRealIP(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	middleware := NewRealIPMiddleware(extractor)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))

	request := httptest.NewRequest("GET", "/", nil)
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{}))
	request.RemoteAddr = "10.0.0.1:54321"
	request.Header.Set("X-Real-IP", "203.0.113.50")
	recorder := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkRealIPMiddleware_IncomingRequestID_Trusted(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"192.168.0.0/16"}, false)
	if err != nil {
		b.Fatal(err)
	}
	middleware := NewRealIPMiddleware(extractor)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))

	request := httptest.NewRequest("GET", "/", nil)
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{}))
	request.RemoteAddr = "192.168.1.100:54321"
	request.Header.Set("X-Request-Id", "existing-request-id-12345")
	recorder := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkRealIPMiddleware_IncomingRequestID_Untrusted(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	middleware := NewRealIPMiddleware(extractor)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))

	request := httptest.NewRequest("GET", "/", nil)
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{}))
	request.RemoteAddr = "192.168.1.100:54321"
	request.Header.Set("X-Request-Id", "injected-request-id")
	recorder := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		handler.ServeHTTP(recorder, request)
	}
}
