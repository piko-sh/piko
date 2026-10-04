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
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPublicAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		address string
		want    bool
	}{
		{name: "public IPv4", address: "93.184.216.34", want: true},
		{name: "public IPv4 DNS resolver", address: "8.8.8.8", want: true},
		{name: "public IPv6", address: "2606:4700:4700::1111", want: true},
		{name: "IPv4-mapped public IPv4", address: "::ffff:8.8.8.8", want: true},
		{name: "NAT64 embedding public IPv4", address: "64:ff9b::808:808", want: true},
		{name: "6to4 embedding public IPv4", address: "2002:808:808::1", want: true},
		{name: "IPv4 loopback", address: "127.0.0.1", want: false},
		{name: "IPv4 loopback range", address: "127.255.0.9", want: false},
		{name: "IPv6 loopback", address: "::1", want: false},
		{name: "IPv4 unspecified", address: "0.0.0.0", want: false},
		{name: "IPv6 unspecified", address: "::", want: false},
		{name: "this network", address: "0.1.2.3", want: false},
		{name: "private 10/8", address: "10.0.0.1", want: false},
		{name: "private 172.16/12", address: "172.16.5.4", want: false},
		{name: "private 192.168/16", address: "192.168.1.1", want: false},
		{name: "unique local IPv6", address: "fd00::1", want: false},
		{name: "link-local IPv4", address: "169.254.1.1", want: false},
		{name: "cloud metadata service", address: "169.254.169.254", want: false},
		{name: "link-local IPv6", address: "fe80::1", want: false},
		{name: "link-local IPv6 with zone", address: "fe80::1%eth0", want: false},
		{name: "IPv4 multicast", address: "224.0.0.1", want: false},
		{name: "IPv6 multicast", address: "ff02::1", want: false},
		{name: "carrier-grade NAT", address: "100.64.0.1", want: false},
		{name: "carrier-grade NAT upper bound", address: "100.127.255.254", want: false},
		{name: "benchmarking range", address: "198.18.0.1", want: false},
		{name: "documentation range", address: "203.0.113.7", want: false},
		{name: "reserved range", address: "240.0.0.1", want: false},
		{name: "limited broadcast", address: "255.255.255.255", want: false},
		{name: "IPv4-mapped loopback", address: "::ffff:127.0.0.1", want: false},
		{name: "IPv4-mapped metadata service", address: "::ffff:169.254.169.254", want: false},
		{name: "IPv4-mapped private", address: "::ffff:10.1.2.3", want: false},
		{name: "IPv4-compatible loopback", address: "::127.0.0.1", want: false},
		{name: "NAT64 embedding loopback", address: "64:ff9b::7f00:1", want: false},
		{name: "NAT64 embedding metadata service", address: "64:ff9b::a9fe:a9fe", want: false},
		{name: "6to4 embedding private", address: "2002:a00:1::1", want: false},
		{name: "local-use NAT64", address: "64:ff9b:1::1", want: false},
		{name: "Teredo", address: "2001::1", want: false},
		{name: "IPv6 documentation", address: "2001:db8::1", want: false},
		{name: "IPv6 site-local", address: "fec0::1", want: false},
		{name: "IPv6 discard-only", address: "100::1", want: false},
		{name: "IPv6 outside global unicast space", address: "4000::1", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			addr, err := netip.ParseAddr(testCase.address)
			require.NoError(t, err)
			assert.Equal(t, testCase.want, isPublicAddress(addr))
		})
	}
}

func TestIsPublicAddress_RefusesZeroAddress(t *testing.T) {
	t.Parallel()

	assert.False(t, isPublicAddress(netip.Addr{}))
}

func TestNewAddressControl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		network string
		address string
		wantErr bool
	}{
		{name: "public IPv4 over tcp", network: "tcp", address: "8.8.8.8:443", wantErr: false},
		{name: "public IPv4 over tcp4", network: "tcp4", address: "8.8.8.8:80", wantErr: false},
		{name: "public IPv6 over tcp6", network: "tcp6", address: "[2606:4700:4700::1111]:443", wantErr: false},
		{name: "loopback", network: "tcp4", address: "127.0.0.1:8080", wantErr: true},
		{name: "metadata service", network: "tcp4", address: "169.254.169.254:80", wantErr: true},
		{name: "IPv4-mapped private", network: "tcp6", address: "[::ffff:192.168.0.1]:80", wantErr: true},
		{name: "unspecified", network: "tcp", address: "0.0.0.0:80", wantErr: true},
		{name: "udp network", network: "udp", address: "8.8.8.8:53", wantErr: true},
		{name: "unparseable address", network: "tcp", address: "example.com:80", wantErr: true},
	}

	control := newAddressControl(isPublicAddress)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := control(context.Background(), testCase.network, testCase.address, nil)
			if !testCase.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrNonPublicDestination)
		})
	}
}

func TestNewRedirectPolicy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr      error
		name         string
		target       string
		viaCount     int
		maxRedirects int
	}{
		{name: "first redirect to public host", target: "https://example.com/a", viaCount: 1, maxRedirects: 5, wantErr: nil},
		{name: "last permitted redirect", target: "https://example.com/a", viaCount: 5, maxRedirects: 5, wantErr: nil},
		{name: "redirect budget exhausted", target: "https://example.com/a", viaCount: 6, maxRedirects: 5, wantErr: errTooManyRedirects},
		{name: "redirects disabled", target: "https://example.com/a", viaCount: 1, maxRedirects: 0, wantErr: errTooManyRedirects},
		{name: "redirect to file scheme", target: "file:///etc/passwd", viaCount: 1, maxRedirects: 5, wantErr: errUnsupportedRedirectScheme},
		{name: "redirect to gopher scheme", target: "gopher://example.com/", viaCount: 1, maxRedirects: 5, wantErr: errUnsupportedRedirectScheme},
		{name: "redirect to loopback literal", target: "http://127.0.0.1:8080/", viaCount: 1, maxRedirects: 5, wantErr: ErrNonPublicDestination},
		{name: "redirect to metadata literal", target: "http://169.254.169.254/latest/meta-data", viaCount: 1, maxRedirects: 5, wantErr: ErrNonPublicDestination},
		{name: "redirect to IPv6 loopback literal", target: "http://[::1]/", viaCount: 1, maxRedirects: 5, wantErr: ErrNonPublicDestination},
		{name: "redirect to public literal", target: "http://8.8.8.8/", viaCount: 1, maxRedirects: 5, wantErr: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			target, err := url.Parse(testCase.target)
			require.NoError(t, err)

			via := make([]*http.Request, testCase.viaCount)
			policy := newRedirectPolicy(testCase.maxRedirects, isPublicAddress)
			err = policy(&http.Request{URL: target}, via)

			if testCase.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, testCase.wantErr)
		})
	}
}

func TestNewPublicHTTPClient_AppliesOptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		options          []PublicHTTPClientOption
		wantTimeout      time.Duration
		wantMaxRedirects int
	}{
		{
			name:             "defaults",
			options:          nil,
			wantTimeout:      defaultPublicClientTimeout,
			wantMaxRedirects: defaultPublicClientMaxRedirects,
		},
		{
			name:             "custom values",
			options:          []PublicHTTPClientOption{WithPublicClientTimeout(time.Minute), WithPublicClientMaxRedirects(2)},
			wantTimeout:      time.Minute,
			wantMaxRedirects: 2,
		},
		{
			name:             "zero redirects refuses every redirect",
			options:          []PublicHTTPClientOption{WithPublicClientMaxRedirects(0)},
			wantTimeout:      defaultPublicClientTimeout,
			wantMaxRedirects: 0,
		},
		{
			name:             "invalid values keep defaults",
			options:          []PublicHTTPClientOption{WithPublicClientTimeout(-time.Second), WithPublicClientMaxRedirects(-1)},
			wantTimeout:      defaultPublicClientTimeout,
			wantMaxRedirects: defaultPublicClientMaxRedirects,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client := NewPublicHTTPClient(testCase.options...)
			t.Cleanup(client.CloseIdleConnections)

			assert.Equal(t, testCase.wantTimeout, client.Timeout)

			transport, ok := client.Transport.(*http.Transport)
			require.True(t, ok)
			assert.Nil(t, transport.Proxy)
			assert.Equal(t, testCase.wantTimeout, transport.ResponseHeaderTimeout)
			assert.Equal(t, int64(publicClientMaxResponseHeaderBytes), transport.MaxResponseHeaderBytes)

			target, err := url.Parse("https://example.com/")
			require.NoError(t, err)
			request := &http.Request{URL: target}

			allowed := make([]*http.Request, testCase.wantMaxRedirects)
			assert.NoError(t, client.CheckRedirect(request, allowed))

			exceeded := make([]*http.Request, testCase.wantMaxRedirects+1)
			assert.ErrorIs(t, client.CheckRedirect(request, exceeded), errTooManyRedirects)
		})
	}
}

func TestNewPublicHTTPClient_RefusesLoopbackServer(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := NewPublicHTTPClient(WithPublicClientTimeout(5 * time.Second))
	t.Cleanup(client.CloseIdleConnections)

	for _, target := range []string{server.URL, localhostURL(t, server.URL)} {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
		require.NoError(t, err)

		response, err := client.Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNonPublicDestination)
	}

	assert.Zero(t, hits.Load())
}

func TestNewGuardedHTTPClient_FollowsPermittedRedirects(t *testing.T) {
	t.Parallel()

	server := newRedirectChainServer(t)
	client := newLoopbackTestClient(t, 2, permitLoopbackIPv4)

	body, err := fetchBody(t, client, server.URL+"/hop/2")
	require.NoError(t, err)
	assert.Equal(t, "arrived", body)
}

func TestNewGuardedHTTPClient_StopsAfterRedirectLimit(t *testing.T) {
	t.Parallel()

	server := newRedirectChainServer(t)
	client := newLoopbackTestClient(t, 1, permitLoopbackIPv4)

	_, err := fetchBody(t, client, server.URL+"/hop/2")
	require.Error(t, err)
	assert.ErrorIs(t, err, errTooManyRedirects)
}

func TestNewGuardedHTTPClient_RechecksRedirectHops(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback is unavailable: %v", err)
	}

	var forbiddenHits atomic.Int32
	forbidden := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		forbiddenHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	require.NoError(t, forbidden.Listener.Close())
	forbidden.Listener = listener
	forbidden.Start()
	t.Cleanup(forbidden.Close)

	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, forbidden.URL+"/secret", http.StatusFound)
	}))
	t.Cleanup(entry.Close)

	client := newLoopbackTestClient(t, defaultPublicClientMaxRedirects, permitLoopbackIPv4)

	_, err = fetchBody(t, client, entry.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNonPublicDestination)
	assert.Zero(t, forbiddenHits.Load())
}

func newRedirectChainServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/hop/2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hop/1", http.StatusFound)
	})
	mux.HandleFunc("/hop/1", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "arrived")
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newLoopbackTestClient(t *testing.T, maxRedirects int, isPermitted func(netip.Addr) bool) *http.Client {
	t.Helper()

	client := newGuardedHTTPClient(&publicHTTPClientConfig{
		isPermitted:  isPermitted,
		timeout:      5 * time.Second,
		maxRedirects: maxRedirects,
	})
	t.Cleanup(client.CloseIdleConnections)
	return client
}

func fetchBody(t *testing.T, client *http.Client, target string) (string, error) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	require.NoError(t, err)

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	return string(body), err
}

func localhostURL(t *testing.T, serverURL string) string {
	t.Helper()

	parsed, err := url.Parse(serverURL)
	require.NoError(t, err)
	parsed.Host = net.JoinHostPort("localhost", parsed.Port())
	return parsed.String()
}

func permitLoopbackIPv4(addr netip.Addr) bool {
	return addr.Unmap() == netip.MustParseAddr("127.0.0.1")
}
