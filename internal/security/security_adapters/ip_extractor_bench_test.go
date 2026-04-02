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
	"net/http/httptest"
	"testing"
)

func BenchmarkExtractClientIP_DirectConnection(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "192.168.1.100:54321"

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = extractor.ExtractClientIP(request)
	}
}

func BenchmarkExtractClientIP_TrustedProxyXFF(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.1:54321"
	request.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = extractor.ExtractClientIP(request)
	}
}

func BenchmarkExtractClientIP_TrustedProxyCF(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, true)
	if err != nil {
		b.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.1:54321"
	request.Header.Set("CF-Connecting-IP", "203.0.113.50")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = extractor.ExtractClientIP(request)
	}
}

func BenchmarkExtractClientIP_TrustedProxyXRealIP(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.1:54321"
	request.Header.Set("X-Real-IP", "203.0.113.50")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = extractor.ExtractClientIP(request)
	}
}

func BenchmarkExtractClientIP_XFF_MultiHop(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8"}, false)
	if err != nil {
		b.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.1:54321"
	request.Header.Set("X-Forwarded-For", "spoofed, 203.0.113.50, 10.0.0.2, 10.0.0.3")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = extractor.ExtractClientIP(request)
	}
}

func BenchmarkIsTrustedProxy(b *testing.B) {
	extractor, err := NewTrustedProxyIPExtractor([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}, false)
	if err != nil {
		b.Fatal(err)
	}
	ipExtractor, ok := extractor.(*trustedProxyIPExtractor)
	if !ok {
		b.Fatal("expected *trustedProxyIPExtractor")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = ipExtractor.IsTrustedProxy("10.0.0.1")
	}
}
