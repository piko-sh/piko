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

//go:build integration

package e2e_browser_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"piko.sh/piko/internal/daemon/daemon_frontend"
)

const (
	runtimeDeployAction                = "runtimeDeploy"
	checkRuntimeDeployAction           = "checkRuntimeDeploy"
	runtimeDeployAdvertiseOtherVersion = "advertise-other-version"
	runtimeDeployFailRuntimeOnce       = "fail-runtime-once"
	runtimeDeployFailRuntimeAlways     = "fail-runtime-always"
	runtimeDeployReset                 = "reset"
	failRuntimeAlways                  = -1
	coreRuntimeFile                    = "ppframework.core.es.js"
	distPathPrefix                     = "/_piko/dist/"
	documentFetchDestination           = "document"
	pageLoadCountTimeout               = 5 * time.Second
	pageLoadCountPollInterval          = 50 * time.Millisecond
)

var (
	errUnknownRuntimeDeployMode  = errors.New("unknown runtime deploy mode")
	errRuntimeDeployProxyMissing = errors.New("runtime deploy proxy is not running")
	errPageLoadCountMismatch     = errors.New("page load count mismatch")
	runtimeFailingModes          = []string{runtimeDeployFailRuntimeOnce, runtimeDeployFailRuntimeAlways}
)

type runtimeDeployProxy struct {
	server           *httptest.Server
	transport        *http.Transport
	pageLoads        map[string]int
	failingLoadsLeft int
	mu               sync.Mutex
	currentLoadFails bool
	advertiseOther   bool
}

func (proxy *runtimeDeployProxy) serve(w http.ResponseWriter, r *http.Request, upstream http.Handler) {
	if proxy.recordRequest(r) {
		w.Header().Set("Cache-Control", "no-store")
		http.NotFound(w, r)
		return
	}
	upstream.ServeHTTP(w, r)
}

func (proxy *runtimeDeployProxy) recordRequest(r *http.Request) bool {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()

	if r.Header.Get("Sec-Fetch-Dest") == documentFetchDestination {
		proxy.pageLoads[r.URL.Path]++
		proxy.currentLoadFails = proxy.failingLoadsLeft != 0
		if proxy.failingLoadsLeft > 0 {
			proxy.failingLoadsLeft--
		}
	}
	return proxy.currentLoadFails && isVersionedCoreRuntime(r.URL.Path)
}

func (proxy *runtimeDeployProxy) rewriteRuntimeHeader(response *http.Response) error {
	proxy.mu.Lock()
	advertiseOther := proxy.advertiseOther
	proxy.mu.Unlock()

	current := response.Header.Get(daemon_frontend.RuntimeVersionHeader)
	if !advertiseOther || current == "" {
		return nil
	}
	response.Header.Set(daemon_frontend.RuntimeVersionHeader, otherRuntimeVersion(current))
	return nil
}

func (proxy *runtimeDeployProxy) apply(mode string) error {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()

	switch mode {
	case runtimeDeployAdvertiseOtherVersion:
		proxy.advertiseOther = true
	case runtimeDeployFailRuntimeOnce:
		proxy.failingLoadsLeft = 1
	case runtimeDeployFailRuntimeAlways:
		proxy.failingLoadsLeft = failRuntimeAlways
	case runtimeDeployReset:
		proxy.advertiseOther = false
		proxy.failingLoadsLeft = 0
		proxy.currentLoadFails = false
		clear(proxy.pageLoads)
	default:
		return fmt.Errorf("%w: %q", errUnknownRuntimeDeployMode, mode)
	}
	return nil
}

func (proxy *runtimeDeployProxy) pageLoadCount(path string) int {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	return proxy.pageLoads[path]
}

func (proxy *runtimeDeployProxy) waitForPageLoadCount(ctx context.Context, path string, want int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pageLoadCountPollInterval)
	defer ticker.Stop()

	for {
		got := proxy.pageLoadCount(path)
		if got == want {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w: %s was loaded %d times, want %d", errPageLoadCountMismatch, path, got, want)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for %s to load %d times: %w", path, want, context.Cause(ctx))
		case <-ticker.C:
		}
	}
}

func (proxy *runtimeDeployProxy) Close() {
	proxy.server.Close()
	proxy.transport.CloseIdleConnections()
}

func runRuntimeDeployStep(h *E2EBrowserHarness, step BrowserStep) error {
	if h.RuntimeDeployProxy == nil {
		return errRuntimeDeployProxyMissing
	}
	if err := h.RuntimeDeployProxy.apply(step.Value); err != nil {
		return err
	}
	if slices.Contains(runtimeFailingModes, step.Value) {
		return disableBrowserCache(h)
	}
	if step.Value == runtimeDeployReset {
		return enableBrowserCache(h)
	}
	return nil
}

func runCheckRuntimeDeployStep(h *E2EBrowserHarness, step BrowserStep) error {
	if h.RuntimeDeployProxy == nil {
		return errRuntimeDeployProxyMissing
	}
	want, err := strconv.Atoi(step.ExpectedString())
	if err != nil {
		return fmt.Errorf("reading the expected page load count %q: %w", step.ExpectedString(), err)
	}
	timeout := pageLoadCountTimeout
	if step.Timeout > 0 {
		timeout = time.Duration(step.Timeout) * time.Millisecond
	}
	return h.RuntimeDeployProxy.waitForPageLoadCount(h.PageHelper.Ctx(), step.Value, want, timeout)
}

func disableBrowserCache(h *E2EBrowserHarness) error {
	if err := chromedp.Run(h.PageHelper.Ctx(), network.Enable(), network.SetCacheDisabled(true)); err != nil {
		return fmt.Errorf("turning off the browser cache: %w", err)
	}
	return nil
}

func enableBrowserCache(h *E2EBrowserHarness) error {
	if err := chromedp.Run(h.PageHelper.Ctx(), network.Enable(), network.SetCacheDisabled(false)); err != nil {
		return fmt.Errorf("turning on the browser cache: %w", err)
	}
	return nil
}

func newRuntimeDeployProxy(upstreamURL string) (*runtimeDeployProxy, error) {
	target, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("parsing the test server URL: %w", err)
	}
	proxy := &runtimeDeployProxy{pageLoads: make(map[string]int), transport: &http.Transport{}}
	upstream := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.Out.Host = request.In.Host
		},
		Transport:      proxy.transport,
		ModifyResponse: proxy.rewriteRuntimeHeader,
	}
	proxy.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.serve(w, r, upstream)
	}))
	return proxy, nil
}

func usesRuntimeDeployProxy(spec *E2ETestSpec) bool {
	return slices.ContainsFunc(spec.BrowserSteps, func(step BrowserStep) bool {
		return step.Action == runtimeDeployAction || step.Action == checkRuntimeDeployAction
	})
}

func isVersionedCoreRuntime(path string) bool {
	assetPath, found := strings.CutPrefix(path, distPathPrefix)
	if !found {
		return false
	}
	_, fileName, versioned := daemon_frontend.SplitRuntimeVersion(assetPath)
	return versioned && fileName == coreRuntimeFile
}

func otherRuntimeVersion(current string) string {
	other := strings.Repeat("f", len(current))
	if other == current {
		return strings.Repeat("e", len(current))
	}
	return other
}

func newTestRuntimeDeployProxy(t *testing.T) *runtimeDeployProxy {
	t.Helper()
	return &runtimeDeployProxy{pageLoads: make(map[string]int), transport: &http.Transport{}}
}

func newRequestForDestination(t *testing.T, path, destination string) *http.Request {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	request.Header.Set("Sec-Fetch-Dest", destination)
	return request
}

func newPageLoadRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	return newRequestForDestination(t, path, documentFetchDestination)
}

func newSubresourceRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	return newRequestForDestination(t, path, "script")
}

func TestIsVersionedCoreRuntime(t *testing.T) {
	testCases := []struct {
		name string
		path string
		want bool
	}{
		{name: "versioned core runtime", path: "/_piko/dist/0123456789abcdef/ppframework.core.es.js", want: true},
		{name: "unversioned core runtime", path: "/_piko/dist/ppframework.core.es.js", want: false},
		{name: "versioned components bundle", path: "/_piko/dist/0123456789abcdef/ppframework.components.es.js", want: false},
		{name: "version that is not hex", path: "/_piko/dist/0123456789abcdeg/ppframework.core.es.js", want: false},
		{name: "core runtime outside the bundle prefix", path: "/assets/0123456789abcdef/ppframework.core.es.js", want: false},
		{name: "page", path: "/main", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, isVersionedCoreRuntime(testCase.path))
		})
	}
}

func TestOtherRuntimeVersion(t *testing.T) {
	testCases := []struct {
		name    string
		current string
		want    string
	}{
		{name: "ordinary version", current: "0123456789abcdef", want: "ffffffffffffffff"},
		{name: "version made of f", current: "ffffffffffffffff", want: "eeeeeeeeeeeeeeee"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := otherRuntimeVersion(testCase.current)
			assert.Equal(t, testCase.want, got)
			assert.NotEqual(t, testCase.current, got)
			assert.True(t, daemon_frontend.IsRuntimeVersion(got))
		})
	}
}

func TestRuntimeDeployProxyApply(t *testing.T) {
	testCases := []struct {
		name                 string
		modes                []string
		wantAdvertiseOther   bool
		wantFailingLoadsLeft int
		wantErr              error
	}{
		{name: "advertise another version", modes: []string{runtimeDeployAdvertiseOtherVersion}, wantAdvertiseOther: true},
		{name: "fail the runtime once", modes: []string{runtimeDeployFailRuntimeOnce}, wantFailingLoadsLeft: 1},
		{name: "fail the runtime always", modes: []string{runtimeDeployFailRuntimeAlways}, wantFailingLoadsLeft: failRuntimeAlways},
		{
			name:  "reset turns every change off",
			modes: []string{runtimeDeployAdvertiseOtherVersion, runtimeDeployFailRuntimeAlways, runtimeDeployReset},
		},
		{name: "unknown mode", modes: []string{"no-such-mode"}, wantErr: errUnknownRuntimeDeployMode},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			proxy := newTestRuntimeDeployProxy(t)
			var err error
			for _, mode := range testCase.modes {
				err = proxy.apply(mode)
			}
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.wantAdvertiseOther, proxy.advertiseOther)
			assert.Equal(t, testCase.wantFailingLoadsLeft, proxy.failingLoadsLeft)
		})
	}
}

func TestRuntimeDeployProxyResetClearsCountsAndTheFailingLoad(t *testing.T) {
	proxy := newTestRuntimeDeployProxy(t)
	require.NoError(t, proxy.apply(runtimeDeployFailRuntimeAlways))
	proxy.recordRequest(newPageLoadRequest(t, "/main"))

	require.NoError(t, proxy.apply(runtimeDeployReset))

	assert.Zero(t, proxy.pageLoadCount("/main"))
	assert.False(t, proxy.recordRequest(newSubresourceRequest(t, "/_piko/dist/0123456789abcdef/ppframework.core.es.js")))
}

func TestRuntimeDeployProxyRecordRequest(t *testing.T) {
	const runtimePath = "/_piko/dist/0123456789abcdef/ppframework.core.es.js"
	testCases := []struct {
		name          string
		mode          string
		requests      func(t *testing.T) []*http.Request
		wantFails     []bool
		wantPageLoads int
	}{
		{
			name: "no mode lets every request through",
			mode: runtimeDeployReset,
			requests: func(t *testing.T) []*http.Request {
				return []*http.Request{newPageLoadRequest(t, "/main"), newSubresourceRequest(t, runtimePath)}
			},
			wantFails:     []bool{false, false},
			wantPageLoads: 1,
		},
		{
			name: "failing once fails the runtime of the next load only",
			mode: runtimeDeployFailRuntimeOnce,
			requests: func(t *testing.T) []*http.Request {
				return []*http.Request{
					newPageLoadRequest(t, "/main"), newSubresourceRequest(t, runtimePath),
					newPageLoadRequest(t, "/main"), newSubresourceRequest(t, runtimePath),
				}
			},
			wantFails:     []bool{false, true, false, false},
			wantPageLoads: 2,
		},
		{
			name: "failing always fails the runtime of every load",
			mode: runtimeDeployFailRuntimeAlways,
			requests: func(t *testing.T) []*http.Request {
				return []*http.Request{
					newPageLoadRequest(t, "/main"), newSubresourceRequest(t, runtimePath),
					newPageLoadRequest(t, "/main"), newSubresourceRequest(t, runtimePath),
				}
			},
			wantFails:     []bool{false, true, false, true},
			wantPageLoads: 2,
		},
		{
			name: "a failing load lets other bundles through",
			mode: runtimeDeployFailRuntimeAlways,
			requests: func(t *testing.T) []*http.Request {
				return []*http.Request{
					newPageLoadRequest(t, "/main"),
					newSubresourceRequest(t, "/_piko/dist/0123456789abcdef/ppframework.components.es.js"),
					newSubresourceRequest(t, "/_piko/dist/ppframework.core.es.js"),
				}
			},
			wantFails:     []bool{false, false, false},
			wantPageLoads: 1,
		},
		{
			name: "fragment fetches are not page loads",
			mode: runtimeDeployReset,
			requests: func(t *testing.T) []*http.Request {
				return []*http.Request{newRequestForDestination(t, "/main?_f=1", "empty")}
			},
			wantFails:     []bool{false},
			wantPageLoads: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			proxy := newTestRuntimeDeployProxy(t)
			require.NoError(t, proxy.apply(testCase.mode))
			var fails []bool
			for _, request := range testCase.requests(t) {
				fails = append(fails, proxy.recordRequest(request))
			}
			assert.Equal(t, testCase.wantFails, fails)
			assert.Equal(t, testCase.wantPageLoads, proxy.pageLoadCount("/main"))
		})
	}
}

func TestRuntimeDeployProxyWaitsForAPageLoadThatArrivesLater(t *testing.T) {
	proxy := newTestRuntimeDeployProxy(t)
	recorded := make(chan struct{})
	go func() {
		defer close(recorded)
		proxy.recordRequest(newPageLoadRequest(t, "/main"))
	}()

	err := proxy.waitForPageLoadCount(t.Context(), "/main", 1, pageLoadCountTimeout)

	require.NoError(t, err)
	<-recorded
}

func TestRuntimeDeployProxyReportsAPageLoadCountThatNeverMatches(t *testing.T) {
	proxy := newTestRuntimeDeployProxy(t)
	proxy.recordRequest(newPageLoadRequest(t, "/main"))

	err := proxy.waitForPageLoadCount(t.Context(), "/main", 2, 2*pageLoadCountPollInterval)

	require.ErrorIs(t, err, errPageLoadCountMismatch)
}

func TestRuntimeDeployProxyRewritesTheRuntimeHeaderOnlyWhenAsked(t *testing.T) {
	proxy := newTestRuntimeDeployProxy(t)
	response := &http.Response{Header: http.Header{}}
	response.Header.Set(daemon_frontend.RuntimeVersionHeader, "0123456789abcdef")

	require.NoError(t, proxy.rewriteRuntimeHeader(response))
	assert.Equal(t, "0123456789abcdef", response.Header.Get(daemon_frontend.RuntimeVersionHeader))

	require.NoError(t, proxy.apply(runtimeDeployAdvertiseOtherVersion))
	require.NoError(t, proxy.rewriteRuntimeHeader(response))
	assert.Equal(t, "ffffffffffffffff", response.Header.Get(daemon_frontend.RuntimeVersionHeader))
}

func TestRuntimeDeployProxyCloseLeavesNoConnectionsOpen(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(daemon_frontend.RuntimeVersionHeader, "0123456789abcdef")
		_, _ = io.WriteString(w, "upstream")
	}))
	defer upstream.Close()
	ignoreUpstream := goleak.IgnoreCurrent()

	proxy, err := newRuntimeDeployProxy(upstream.URL)
	require.NoError(t, err)
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, proxy.server.URL+"/main", nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, "upstream", string(body))

	proxy.Close()

	goleak.VerifyNone(t, ignoreUpstream)
}
