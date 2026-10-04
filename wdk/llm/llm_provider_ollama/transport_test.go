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

package llm_provider_ollama

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wrappingRoundTripper struct {
	next http.RoundTripper
}

func (w wrappingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return w.next.RoundTrip(request)
}

type recordingBody struct {
	reader   io.Reader
	readErr  error
	closeErr error
	read     int
	closed   bool
}

func (b *recordingBody) Read(buffer []byte) (int, error) {
	if b.readErr != nil {
		return 0, b.readErr
	}
	n, err := b.reader.Read(buffer)
	b.read += n
	return n, err
}

func (b *recordingBody) Close() error {
	b.closed = true
	return b.closeErr
}

func TestCloneTransport(t *testing.T) {
	t.Parallel()

	base := &http.Transport{MaxIdleConns: 7}

	testCases := []struct {
		base             http.RoundTripper
		name             string
		wantMaxIdleConns int
	}{
		{name: "copies a concrete transport", base: base, wantMaxIdleConns: 7},
		{name: "falls back for a wrapped transport", base: wrappingRoundTripper{next: base}, wantMaxIdleConns: fallbackMaxIdleConnections},
		{name: "falls back for a missing transport", base: nil, wantMaxIdleConns: fallbackMaxIdleConnections},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			transport := cloneTransport(testCase.base)

			require.NotNil(t, transport)
			assert.NotSame(t, base, transport)
			assert.Equal(t, testCase.wantMaxIdleConns, transport.MaxIdleConns)
		})
	}
}

func TestNewFallbackTransport_HonoursProxyEnvironment(t *testing.T) {
	t.Parallel()

	transport := newFallbackTransport()

	assert.NotNil(t, transport.Proxy)
	assert.NotNil(t, transport.DialContext)
	assert.True(t, transport.ForceAttemptHTTP2)
	assert.Equal(t, fallbackTLSHandshakeTimeout, transport.TLSHandshakeTimeout)
}

func TestDrainAndClose(t *testing.T) {
	t.Parallel()

	errRead := errors.New("connection reset")
	errClose := errors.New("close failed")

	testCases := []struct {
		readErr      error
		closeErr     error
		name         string
		wantErrs     []error
		bodySize     int
		wantReadSize int
	}{
		{name: "drains a small body", bodySize: 10, wantReadSize: 10},
		{name: "stops draining at the limit", bodySize: responseDrainLimit * 2, wantReadSize: responseDrainLimit},
		{name: "reports a read failure", readErr: errRead, wantErrs: []error{errRead}},
		{name: "reports a close failure", closeErr: errClose, wantErrs: []error{errClose}},
		{name: "reports both failures", readErr: errRead, closeErr: errClose, wantErrs: []error{errRead, errClose}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			body := &recordingBody{
				reader:   strings.NewReader(strings.Repeat("x", testCase.bodySize)),
				readErr:  testCase.readErr,
				closeErr: testCase.closeErr,
			}

			err := drainAndClose(body)

			assert.True(t, body.closed)
			assert.Equal(t, testCase.wantReadSize, body.read)
			if len(testCase.wantErrs) == 0 {
				assert.NoError(t, err)
				return
			}
			for _, wantErr := range testCase.wantErrs {
				assert.ErrorIs(t, err, wantErr)
			}
		})
	}
}
