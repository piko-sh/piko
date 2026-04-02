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

package security_adapters

import (
	"errors"
	"net/http"
)

type mockSecureCookieWriter struct {
	setCookies []*http.Cookie
	isHTTPS    bool
}

func (m *mockSecureCookieWriter) SetCookie(w http.ResponseWriter, cookie *http.Cookie) {

	m.setCookies = append(m.setCookies, new(*cookie))
	http.SetCookie(w, cookie)
}

func (m *mockSecureCookieWriter) IsHTTPS() bool {
	return m.isHTTPS
}

type mockClientIPExtractor struct {
	extractFunc   func(r *http.Request) string
	isTrustedFunc func(ip string) bool
}

func (m *mockClientIPExtractor) ExtractClientIP(r *http.Request) string {
	if m.extractFunc != nil {
		return m.extractFunc(r)
	}
	return r.RemoteAddr
}

func (m *mockClientIPExtractor) IsTrustedProxy(ip string) bool {
	if m.isTrustedFunc != nil {
		return m.isTrustedFunc(ip)
	}
	return false
}

type deterministicRandReader struct {
	data   []byte
	offset int
}

func newDeterministicRandReader(data []byte) *deterministicRandReader {
	return &deterministicRandReader{data: data}
}

func (r *deterministicRandReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.data[r.offset%len(r.data)]
		r.offset++
	}
	return len(p), nil
}

type failingRandReader struct{}

func (failingRandReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated rand failure")
}
