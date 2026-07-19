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

package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/google/pprof/profile"
	"piko.sh/piko/wdk/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/wdk/safedisk"
)

func TestParseProfileFlags_Defaults(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	flags, url, ok := parseProfileFlags([]string{"http://localhost:8080/"}, &stderr)
	require.True(t, ok, "parseProfileFlags failed: %s", stderr.String())

	assert.Equal(t, "http://localhost:8080/", url, "url = %q, want %q", url, "http://localhost:8080/")
	assert.Equal(t, 6060, flags.pprofPort, "pprofPort = %d, want 6060", flags.pprofPort)
	assert.Equal(t, 100, flags.concurrency, "concurrency = %d, want 100", flags.concurrency)
	assert.Equal(t, 30, flags.duration, "duration = %d, want 30", flags.duration)
	assert.Equal(t, "pprof", flags.output, "output = %q, want %q", flags.output, "pprof")
	assert.Equal(t, 60, flags.topN, "topN = %d, want 60", flags.topN)
}

func TestParseProfileFlags_CustomValues(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	flags, url, ok := parseProfileFlags([]string{
		"--pprof-port", "9090",
		"--concurrency", "50",
		"--duration", "10",
		"--output", "/tmp/profiles",
		"--cookie", "session=abc",
		"--top", "20",
		"--focus", "render",
		"http://example.com/",
	}, &stderr)

	require.True(t, ok, "parseProfileFlags failed: %s", stderr.String())

	assert.Equal(t, "http://example.com/", url, "url = %q, want %q", url, "http://example.com/")
	assert.Equal(t, 9090, flags.pprofPort, "pprofPort = %d, want 9090", flags.pprofPort)
	assert.Equal(t, 50, flags.concurrency, "concurrency = %d, want 50", flags.concurrency)
	assert.Equal(t, 10, flags.duration, "duration = %d, want 10", flags.duration)
	assert.Equal(t, "/tmp/profiles", flags.output, "output = %q, want %q", flags.output, "/tmp/profiles")
	assert.Equal(t, "session=abc", flags.cookie, "cookie = %q, want %q", flags.cookie, "session=abc")
	assert.Equal(t, 20, flags.topN, "topN = %d, want 20", flags.topN)
	assert.Equal(t, "render", flags.focus, "focus = %q, want %q", flags.focus, "render")
}

func TestParseProfileFlags_URLFirst(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	flags, url, ok := parseProfileFlags([]string{
		"http://localhost:8080/",
		"--concurrency", "200",
		"--tui",
		"--duration", "10",
	}, &stderr)

	require.True(t, ok, "parseProfileFlags failed: %s", stderr.String())

	assert.Equal(t, "http://localhost:8080/", url, "url = %q, want %q", url, "http://localhost:8080/")
	assert.Equal(t, 200, flags.concurrency, "concurrency = %d, want 200", flags.concurrency)
	assert.True(t, flags.tui, "tui should be true")
	assert.Equal(t, 10, flags.duration, "duration = %d, want 10", flags.duration)
}

func TestExtractProfileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		arguments    []string
		wantURL      string
		wantFlagArgs []string
	}{
		{
			name:         "url first",
			arguments:    []string{"http://localhost:8080/", "--concurrency", "200"},
			wantURL:      "http://localhost:8080/",
			wantFlagArgs: []string{"--concurrency", "200"},
		},
		{
			name:         "url last",
			arguments:    []string{"--concurrency", "200", "http://localhost:8080/"},
			wantURL:      "http://localhost:8080/",
			wantFlagArgs: []string{"--concurrency", "200"},
		},
		{
			name:         "url middle",
			arguments:    []string{"--concurrency", "200", "https://example.com/", "--tui"},
			wantURL:      "https://example.com/",
			wantFlagArgs: []string{"--concurrency", "200", "--tui"},
		},
		{
			name:         "no url",
			arguments:    []string{"--concurrency", "200"},
			wantURL:      "",
			wantFlagArgs: []string{"--concurrency", "200"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotURL, gotFlags := extractProfileURL(tt.arguments)
			assert.Equal(t, tt.wantURL, gotURL, "url = %q, want %q", gotURL, tt.wantURL)
			require.Len(t, gotFlags, len(tt.wantFlagArgs), "flagArgs len = %d, want %d", len(gotFlags), len(tt.wantFlagArgs))
			for i, f := range gotFlags {
				assert.Equal(t, tt.wantFlagArgs[i], f, "flagArgs[%d] = %q, want %q", i, f, tt.wantFlagArgs[i])
			}
		})
	}
}

func TestParseProfileFlags_MissingURL(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	_, _, ok := parseProfileFlags([]string{}, &stderr)
	assert.False(t, ok, "expected parseProfileFlags to fail with no URL")
	assert.Contains(t, stderr.String(), "URL to test is a required argument", "stderr should mention missing URL, got: %s", stderr.String())
}

func TestRunProfile_NoArgs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunProfileWithIO(nil, &stdout, &stderr)
	assert.Equal(t, 1, code, "exit code = %d, want 1", code)
}

func TestRunProfile_Help(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	code := RunProfileWithIO([]string{"-h"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "exit code = %d, want 1", code)
	assert.Contains(t, stderr.String(), "piko profile", "stderr should contain usage text, got: %s", stderr.String())
}

func TestRunProfile_InvalidFocus(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunProfileWithIO([]string{"--focus", "[invalid", "http://localhost/"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "exit code = %d, want 1", code)
	assert.Contains(t, stderr.String(), "invalid --focus pattern", "stderr should mention invalid focus, got: %s", stderr.String())
}

func TestLoadResult_Percentiles(t *testing.T) {
	t.Parallel()

	result := &loadResult{
		totalRequests: 100,
		duration:      time.Second,
		latencies: []time.Duration{
			1 * time.Millisecond,
			2 * time.Millisecond,
			3 * time.Millisecond,
			4 * time.Millisecond,
			5 * time.Millisecond,
			6 * time.Millisecond,
			7 * time.Millisecond,
			8 * time.Millisecond,
			9 * time.Millisecond,
			10 * time.Millisecond,
		},
	}

	p50 := result.percentile(50)
	assert.Equal(t, 6*time.Millisecond, p50, "p50 = %v, want 6ms", p50)
	p100 := result.percentile(100)
	assert.Equal(t, 10*time.Millisecond, p100, "p100 = %v, want 10ms", p100)
}

func TestLoadResult_RequestsPerSecond(t *testing.T) {
	t.Parallel()

	result := &loadResult{
		totalRequests: 1000,
		duration:      2 * time.Second,
	}

	rps := result.requestsPerSecond()
	assert.EqualValues(t, 500, rps, "requestsPerSecond = %f, want 500", rps)
}

func TestLoadResult_MeanLatency(t *testing.T) {
	t.Parallel()

	result := &loadResult{
		latencies: []time.Duration{
			2 * time.Millisecond,
			4 * time.Millisecond,
			6 * time.Millisecond,
		},
	}

	mean := result.meanLatency()
	assert.Equal(t, 4*time.Millisecond, mean, "meanLatency = %v, want 4ms", mean)
}

func TestLoadResult_EmptyLatencies(t *testing.T) {
	t.Parallel()

	result := &loadResult{}
	assert.EqualValues(t, 0, result.meanLatency(), "meanLatency should be 0 for empty latencies")
	assert.EqualValues(t, 0, result.percentile(50), "percentile should be 0 for empty latencies")
}

func TestFetchProfilerStatus_ReturnsNilWhenEndpointMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	status, err := fetchProfilerStatus(context.Background(), server.URL)
	require.NoError(t, err, "fetchProfilerStatus returned error: %v", err)
	require.Nil(t, status, "status = %#v, want nil", status)
}

func TestFetchProfilerStatus_DecodesRollingTraceCapability(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != profiler.ProfilerStatusPath {
			http.NotFound(w, r)
			return
		}
		data, _ := json.Marshal(profiler.ServerStatus{
			PprofBasePath: profiler.BasePath + "/debug/pprof",
			StatusPath:    profiler.ProfilerStatusPath,
			RollingTrace: profiler.RollingTraceStatus{
				Enabled:      true,
				MinAge:       "15s",
				MaxBytes:     128 * 1024,
				DownloadPath: profiler.RollingTracePath,
			},
		})
		_, _ = w.Write(data)
	}))
	defer server.Close()

	status, err := fetchProfilerStatus(context.Background(), server.URL)
	require.NoError(t, err, "fetchProfilerStatus returned error: %v", err)
	require.NotNil(t, status, "status should not be nil")
	require.True(t, status.RollingTrace.Enabled, "rolling trace should be enabled")
	require.Equal(t, profiler.RollingTracePath, status.RollingTrace.DownloadPath, "download path = %q, want %q", status.RollingTrace.DownloadPath, profiler.RollingTracePath)
}

func TestLoadResult_ZeroDuration(t *testing.T) {
	t.Parallel()

	result := &loadResult{}
	assert.EqualValues(t, 0, result.requestsPerSecond(), "requestsPerSecond should be 0 when duration is 0")
}

func TestWriteLoadTestReport(t *testing.T) {
	t.Parallel()

	result := &loadResult{
		totalRequests:  10000,
		failedRequests: 5,
		duration:       2 * time.Second,
		bytesReceived:  1024 * 1024 * 100,
		latencies: []time.Duration{
			1 * time.Millisecond,
			2 * time.Millisecond,
			3 * time.Millisecond,
			4 * time.Millisecond,
			5 * time.Millisecond,
		},
	}

	var buffer bytes.Buffer
	writeLoadTestReport(&buffer, result)

	report := buffer.String()

	assert.Contains(t, report, "LOAD TEST REPORT", "report should contain header")
	assert.Contains(t, report, "Complete requests:      10000", "report should contain total requests")
	assert.Contains(t, report, "Failed requests:        5", "report should contain failed requests")
	assert.Contains(t, report, "Requests per second:", "report should contain request/s")
	assert.Contains(t, report, "50%", "report should contain percentiles")
}

func createSyntheticProfile(t *testing.T) []byte {
	t.Helper()

	funcA := &profile.Function{
		ID:       1,
		Name:     "example.com/pkg.FuncA",
		Filename: "/source/pkg/a.go",
	}
	funcB := &profile.Function{
		ID:       2,
		Name:     "example.com/pkg.FuncB",
		Filename: "/source/pkg/b.go",
	}

	locA := &profile.Location{
		ID: 1,
		Line: []profile.Line{
			{Function: funcA, Line: 42},
		},
	}
	locB := &profile.Location{
		ID: 2,
		Line: []profile.Line{
			{Function: funcB, Line: 100},
		},
	}

	prof := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "alloc_objects", Unit: "count"},
			{Type: "alloc_space", Unit: "bytes"},
		},
		Sample: []*profile.Sample{
			{
				Location: []*profile.Location{locA, locB},
				Value:    []int64{500, 1024 * 1024},
			},
			{
				Location: []*profile.Location{locB},
				Value:    []int64{300, 512 * 1024},
			},
		},
		Location: []*profile.Location{locA, locB},
		Function: []*profile.Function{funcA, funcB},
	}

	var buffer bytes.Buffer
	err := prof.Write(&buffer)
	require.NoError(t, err, "failed to write synthetic profile: %v", err)
	return buffer.Bytes()
}

func TestGenerateProfileReport_ByFunction(t *testing.T) {
	t.Parallel()

	data := createSyntheticProfile(t)

	var buffer bytes.Buffer
	err := generateProfileReport(&buffer, data, profileReportConfig{
		sectionTitle: "allocs (alloc_space)",
		sampleIndex:  1,
		byLine:       false,
		topN:         10,
	}, 0)

	require.NoError(t, err, "generateProfileReport failed: %v", err)

	report := buffer.String()

	assert.Contains(t, report, "allocs (alloc_space)", "report should contain section title")
	assert.Contains(t, report, "FuncA", "report should contain FuncA")
	assert.Contains(t, report, "FuncB", "report should contain FuncB")
	assert.Contains(t, report, "flat", "report should contain header row")
}

func TestGenerateProfileReport_ByLine(t *testing.T) {
	t.Parallel()

	data := createSyntheticProfile(t)

	var buffer bytes.Buffer
	err := generateProfileReport(&buffer, data, profileReportConfig{
		sectionTitle: "allocs (alloc_objects by line)",
		sampleIndex:  0,
		byLine:       true,
		topN:         10,
	}, 0)

	require.NoError(t, err, "generateProfileReport failed: %v", err)

	report := buffer.String()

	assert.Contains(t, report, "/source/pkg/a.go:42", "report should contain FuncA file:line")
	assert.Contains(t, report, "/source/pkg/b.go:100", "report should contain FuncB file:line")
}

func TestGenerateProfileReport_FocusFilter(t *testing.T) {
	t.Parallel()

	data := createSyntheticProfile(t)

	focusRegex := regexp.MustCompile("FuncA")

	var buffer bytes.Buffer
	err := generateProfileReport(&buffer, data, profileReportConfig{
		sectionTitle: "focused",
		sampleIndex:  0,
		byLine:       false,
		focusRegex:   focusRegex,
		topN:         10,
	}, 0)

	require.NoError(t, err, "generateProfileReport failed: %v", err)

	report := buffer.String()

	assert.Contains(t, report, "FuncA", "report should contain FuncA (matches focus)")
}

func TestGenerateProfileReport_InvalidSampleIndex(t *testing.T) {
	t.Parallel()

	data := createSyntheticProfile(t)

	var buffer bytes.Buffer
	err := generateProfileReport(&buffer, data, profileReportConfig{
		sectionTitle: "bad index",
		sampleIndex:  99,
		topN:         10,
	}, 0)

	assert.Error(t, err, "expected error for out-of-range sample index")
}

func TestGenerateProfileReport_InvalidData(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	err := generateProfileReport(&buffer, []byte("not a pprof file"), profileReportConfig{
		sectionTitle: "bad data",
		sampleIndex:  0,
		topN:         10,
	}, 0)

	assert.Error(t, err, "expected error for invalid pprof data")
}

func TestGenerateProfileReport_PerRequestColumn(t *testing.T) {
	t.Parallel()

	data := createSyntheticProfile(t)

	var buffer bytes.Buffer
	err := generateProfileReport(&buffer, data, profileReportConfig{
		sectionTitle: "allocs with per-request",
		sampleIndex:  0,
		byLine:       false,
		topN:         10,
	}, 100)

	require.NoError(t, err, "generateProfileReport failed: %v", err)

	report := buffer.String()

	assert.Contains(t, report, "flat/request", "report should contain flat/request header when totalRequests > 0")
	assert.Contains(t, report, "FuncA", "report should contain FuncA")
}

func TestProfileFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expected string
		input    int64
	}{
		{expected: "0B", input: 0},
		{expected: "512B", input: 512},
		{expected: "1.00KB", input: 1024},
		{expected: "1.00MB", input: 1024 * 1024},
		{expected: "1.00GB", input: 1024 * 1024 * 1024},
		{expected: "1.50MB", input: 1536 * 1024},
	}

	for _, tt := range tests {
		got := profileFormatBytes(tt.input)
		assert.Equal(t, tt.expected, got, "profileFormatBytes(%d) = %q, want %q", tt.input, got, tt.expected)
	}
}

func TestProfileFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expected string
		input    int64
	}{
		{expected: "500ns", input: 500},
		{expected: "1.50us", input: 1500},
		{expected: "1.50ms", input: 1500000},
		{expected: "1.50s", input: 1500000000},
		{expected: "1.50m", input: 90000000000},
	}

	for _, tt := range tests {
		got := profileFormatDuration(tt.input)
		assert.Equal(t, tt.expected, got, "profileFormatDuration(%d) = %q, want %q", tt.input, got, tt.expected)
	}
}

func TestProfileFormatCount(t *testing.T) {
	t.Parallel()

	got := profileFormatCount(42)
	assert.Equal(t, "42", got, "profileFormatCount(42) = %q, want %q", got, "42")
}

func TestBuildProfileSpecs(t *testing.T) {
	t.Parallel()

	flags := &profileFlags{topN: 40}
	specs := buildProfileSpecs(flags, nil)

	require.Len(t, specs, 5, "expected 5 profile specs, got %d", len(specs))

	expectedNames := []string{"cpu", "allocs", "heap", "mutex", "block"}
	for i, spec := range specs {
		assert.Equal(t, expectedNames[i], spec.name, "spec[%d].name = %q, want %q", i, spec.name, expectedNames[i])
	}

	assert.Len(t, specs[1].reports, 2, "allocs spec should have 2 reports, got %d", len(specs[1].reports))
}

func TestProfileUsage(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	profileUsage(&buffer)

	output := buffer.String()
	assert.Contains(t, output, "piko profile", "usage should mention 'piko profile'")
	assert.Contains(t, output, "--pprof-port", "usage should mention --pprof-port flag")
	assert.Contains(t, output, "--concurrency", "usage should mention --concurrency flag")
	assert.Contains(t, output, "--header", "usage should mention --header flag")
	assert.Contains(t, output, "--tui", "usage should mention --tui flag")
}

func TestHeaderFlag_Set(t *testing.T) {
	t.Parallel()

	var h headerFlag
	err := h.Set("Authorization: Bearer token123")
	require.NoError(t, err, "unexpected error: %v", err)
	err = h.Set("X-Custom: value")
	require.NoError(t, err, "unexpected error: %v", err)

	assert.Equal(t, "Bearer token123", h.headers["Authorization"], "Authorization = %q, want %q", h.headers["Authorization"], "Bearer token123")
	assert.Equal(t, "value", h.headers["X-Custom"], "X-Custom = %q, want %q", h.headers["X-Custom"], "value")
}

func TestHeaderFlag_SetInvalid(t *testing.T) {
	t.Parallel()

	var h headerFlag
	err := h.Set("no-colon-here")
	assert.Error(t, err, "expected error for header without colon")
	err = h.Set(": no-name")
	assert.Error(t, err, "expected error for header with empty name")
}

func TestHeaderFlag_String(t *testing.T) {
	t.Parallel()

	var h headerFlag
	s := h.String()
	assert.Empty(t, s, "empty headerFlag.String() = %q, want empty", s)

	_ = h.Set("X-Test: hello")
	s = h.String()
	assert.Contains(t, s, "X-Test: hello", "headerFlag.String() = %q, want to contain %q", s, "X-Test: hello")
}

func TestParseProfileFlags_Headers(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	flags, _, ok := parseProfileFlags([]string{
		"--header", "Authorization: Bearer abc",
		"--header", "X-Custom: val",
		"--cookie", "session=xyz",
		"http://localhost:8080/",
	}, &stderr)

	require.True(t, ok, "parseProfileFlags failed: %s", stderr.String())

	assert.Equal(t, "Bearer abc", flags.headers.headers["Authorization"], "Authorization header = %q, want %q", flags.headers.headers["Authorization"], "Bearer abc")
	assert.Equal(t, "val", flags.headers.headers["X-Custom"], "X-Custom header = %q, want %q", flags.headers.headers["X-Custom"], "val")
	assert.Equal(t, "session=xyz", flags.cookie, "cookie = %q, want %q", flags.cookie, "session=xyz")
}

func TestParseProfileFlags_TUI(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	flags, _, ok := parseProfileFlags([]string{
		"--tui",
		"http://localhost:8080/",
	}, &stderr)

	require.True(t, ok, "parseProfileFlags failed: %s", stderr.String())

	assert.True(t, flags.tui, "tui flag should be true")
}

func TestMergedHeaders_CookieAndHeaders(t *testing.T) {
	t.Parallel()

	flags := &profileFlags{
		cookie: "session=abc",
		headers: headerFlag{
			headers: map[string]string{
				"Authorization": "Bearer token",
				"X-Custom":      "val",
			},
		},
	}

	h := flags.mergedHeaders()

	assert.Equal(t, "session=abc", h["Cookie"], "Cookie = %q, want %q", h["Cookie"], "session=abc")
	assert.Equal(t, "Bearer token", h["Authorization"], "Authorization = %q, want %q", h["Authorization"], "Bearer token")
	assert.Equal(t, "val", h["X-Custom"], "X-Custom = %q, want %q", h["X-Custom"], "val")
}

func TestMergedHeaders_NoCookie(t *testing.T) {
	t.Parallel()

	flags := &profileFlags{}
	h := flags.mergedHeaders()

	_, hasCookie := h["Cookie"]
	assert.False(t, hasCookie, "should not have Cookie header when cookie is empty")
}

func TestWriteProfileStats(t *testing.T) {
	t.Parallel()

	result := &loadResult{
		totalRequests:  50000,
		failedRequests: 12,
		duration:       30 * time.Second,
		bytesReceived:  1024 * 1024 * 500,
	}

	directory := t.TempDir()
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err, "creating sandbox: %v", err)
	defer func() { _ = sandbox.Close() }()

	err = writeProfileStats(sandbox, "cpu.pprof.stats", result, 200)
	require.NoError(t, err, "writeProfileStats failed: %v", err)

	data, err := os.ReadFile(filepath.Join(directory, "cpu.pprof.stats"))
	require.NoError(t, err, "reading stats file: %v", err)

	content := string(data)

	assert.Contains(t, content, "total_requests:     50000", "stats should contain total_requests")
	assert.Contains(t, content, "failed_requests:    12", "stats should contain failed_requests")
	assert.Contains(t, content, "concurrency:        200", "stats should contain concurrency")
	assert.Contains(t, content, "requests_per_sec:", "stats should contain requests_per_sec")
	assert.Contains(t, content, "bytes_received:     524288000", "stats should contain bytes_received")
}

func TestProfileTUIModel_Init(t *testing.T) {
	t.Parallel()

	metricsCh := make(chan metricsMessage, 1)
	goroutineCh := make(chan goroutineMessage, 1)
	phaseCh := make(chan phaseMessage, 1)
	doneCh := make(chan profileDoneMessage, 1)

	model := newProfileTUIModel("http://localhost/", 30, metricsCh, goroutineCh, phaseCh, doneCh)

	assert.Equal(t, "http://localhost/", model.targetURL, "targetURL = %q, want %q", model.targetURL, "http://localhost/")
	assert.Len(t, model.phases, 6, "phases count = %d, want 6", len(model.phases))
	for _, p := range model.phases {
		assert.Equal(t, phasePending, model.phaseStatus[p], "phase %q should be pending", p)
	}
}

func TestProfileTUIModel_PhaseUpdate(t *testing.T) {
	t.Parallel()

	metricsCh := make(chan metricsMessage, 1)
	goroutineCh := make(chan goroutineMessage, 1)
	phaseCh := make(chan phaseMessage, 1)
	doneCh := make(chan profileDoneMessage, 1)

	model := newProfileTUIModel("http://localhost/", 30, metricsCh, goroutineCh, phaseCh, doneCh)

	updated, _ := model.Update(phaseMessage{name: "cpu", status: phaseActive})
	m, ok := updated.(*profileTUIModel)
	require.True(t, ok, "unexpected model type")
	assert.Equal(t, "cpu", m.activePhase, "activePhase = %q, want %q", m.activePhase, "cpu")
	assert.Equal(t, phaseActive, m.phaseStatus["cpu"], "cpu phase should be active")

	updated, _ = m.Update(phaseMessage{name: "cpu", status: phaseDone})
	m, ok = updated.(*profileTUIModel)
	require.True(t, ok, "unexpected model type")
	assert.Equal(t, phaseDone, m.phaseStatus["cpu"], "cpu phase should be done")
}

func TestProfileTUIModel_MetricsTick(t *testing.T) {
	t.Parallel()

	metricsCh := make(chan metricsMessage, 4)
	goroutineCh := make(chan goroutineMessage, 1)
	phaseCh := make(chan phaseMessage, 1)
	doneCh := make(chan profileDoneMessage, 1)

	model := newProfileTUIModel("http://localhost/", 30, metricsCh, goroutineCh, phaseCh, doneCh)

	metricsCh <- metricsMessage{rps: 1000, meanLatencyMs: 5.0, total: 100, failed: 2, bytesReceived: 4096, p50Ms: 3.0, p80Ms: 4.0, p99Ms: 8.0, p100Ms: 10.0}
	metricsCh <- metricsMessage{rps: 1200, meanLatencyMs: 4.5, total: 200, failed: 3, bytesReceived: 8192, p50Ms: 2.5, p80Ms: 3.5, p99Ms: 7.0, p100Ms: 9.0}

	updated, _ := model.Update(profileTickMessage(time.Now()))
	m, ok := updated.(*profileTUIModel)
	require.True(t, ok, "unexpected model type")

	assert.EqualValues(t, 1200, m.currentRPS, "currentRPS = %f, want 1200", m.currentRPS)
	assert.EqualValues(t, 200, m.totalRequests, "totalRequests = %d, want 200", m.totalRequests)
	assert.Equal(t, 2, m.rpsHistory.Len(), "rpsHistory.Len() = %d, want 2", m.rpsHistory.Len())
	assert.Equal(t, 2.5, m.p50Ms, "p50Ms = %f, want 2.5", m.p50Ms)
	assert.Equal(t, 3.5, m.p80Ms, "p80Ms = %f, want 3.5", m.p80Ms)
	assert.Equal(t, 7.0, m.p99Ms, "p99Ms = %f, want 7.0", m.p99Ms)
	assert.Equal(t, 9.0, m.p100Ms, "p100Ms = %f, want 9.0", m.p100Ms)
}

func TestLatencyPercentileMs(t *testing.T) {
	t.Parallel()

	sorted := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
		6 * time.Millisecond,
		7 * time.Millisecond,
		8 * time.Millisecond,
		9 * time.Millisecond,
		10 * time.Millisecond,
	}

	p50 := latencyPercentileMs(sorted, 50)
	assert.Equal(t, 6.0, p50, "p50 = %f, want 6.0", p50)
	p99 := latencyPercentileMs(sorted, 99)
	assert.Equal(t, 10.0, p99, "p99 = %f, want 10.0", p99)
	p100 := latencyPercentileMs(sorted, 100)
	assert.Equal(t, 10.0, p100, "p100 = %f, want 10.0", p100)

	p := latencyPercentileMs(nil, 50)
	assert.EqualValues(t, 0, p, "empty p50 = %f, want 0", p)
}

func createSyntheticHeapProfile(t *testing.T, allocObjects, allocSpace int64) []byte {
	t.Helper()

	profileFunction := &profile.Function{
		ID:       1,
		Name:     "example.com/pkg.Alloc",
		Filename: "/source/pkg/alloc.go",
	}
	loc := &profile.Location{
		ID:   1,
		Line: []profile.Line{{Function: profileFunction, Line: 10}},
	}

	prof := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "alloc_objects", Unit: "count"},
			{Type: "alloc_space", Unit: "bytes"},
			{Type: "inuse_objects", Unit: "count"},
			{Type: "inuse_space", Unit: "bytes"},
		},
		Sample: []*profile.Sample{
			{
				Location: []*profile.Location{loc},
				Value:    []int64{allocObjects, allocSpace, 0, 0},
			},
		},
		Location: []*profile.Location{loc},
		Function: []*profile.Function{profileFunction},
	}

	var buffer bytes.Buffer
	err := prof.Write(&buffer)
	require.NoError(t, err, "failed to write synthetic heap profile: %v", err)
	return buffer.Bytes()
}

func TestComputeDeltaProfile(t *testing.T) {
	t.Parallel()

	before := createSyntheticHeapProfile(t, 1000, 1024*1024)
	after := createSyntheticHeapProfile(t, 1500, 1536*1024)

	deltaData, err := computeDeltaProfile(before, after)
	require.NoError(t, err, "computeDeltaProfile failed: %v", err)

	prof, err := profile.ParseData(deltaData)
	require.NoError(t, err, "parsing delta profile: %v", err)

	var totalObjects, totalSpace int64
	for _, s := range prof.Sample {
		totalObjects += s.Value[0]
		totalSpace += s.Value[1]
	}

	assert.EqualValues(t, 500, totalObjects, "delta alloc_objects = %d, want 500", totalObjects)
	expectedSpace := int64(512 * 1024)
	assert.Equal(t, expectedSpace, totalSpace, "delta alloc_space = %d, want %d", totalSpace, expectedSpace)
}

func TestComputeDeltaProfile_InvalidData(t *testing.T) {
	t.Parallel()

	valid := createSyntheticHeapProfile(t, 100, 1024)

	_, err := computeDeltaProfile([]byte("bad"), valid)
	assert.Error(t, err, "expected error for invalid before data")
	_, err = computeDeltaProfile(valid, []byte("bad"))
	assert.Error(t, err, "expected error for invalid after data")
}

func TestWriteAllocChurnSummary(t *testing.T) {
	t.Parallel()

	before := createSyntheticHeapProfile(t, 1000, 1024*1024)
	after := createSyntheticHeapProfile(t, 6000, 6*1024*1024)

	deltaData, err := computeDeltaProfile(before, after)
	require.NoError(t, err, "computeDeltaProfile failed: %v", err)

	var buffer bytes.Buffer
	err = writeAllocChurnSummary(&buffer, deltaData, 1000)
	require.NoError(t, err, "writeAllocChurnSummary failed: %v", err)

	output := buffer.String()

	assert.Contains(t, output, "ALLOCATION CHURN", "summary should contain header")
	assert.Contains(t, output, "5000", "summary should contain delta alloc_objects (5000)")
	assert.Contains(t, output, "Requests during load: 1000", "summary should contain request count")
	assert.Contains(t, output, "/request", "summary should contain per-request stats")
}

func TestBuildProfileSpecs_AllocsDelta(t *testing.T) {
	t.Parallel()

	flags := &profileFlags{topN: 40}
	specs := buildProfileSpecs(flags, nil)

	var allocsSpec *profileSpec
	for i := range specs {
		if specs[i].name == "allocs" {
			allocsSpec = &specs[i]
			break
		}
	}
	require.NotNil(t, allocsSpec, "allocs spec not found")
	assert.True(t, allocsSpec.delta, "allocs spec should have delta=true")

	for _, spec := range specs {
		assert.False(t, spec.name == "heap" && spec.delta, "heap spec should not have delta=true")
	}
}

func TestProfileUnitFormatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeName string
		unit     string
		wantSub  string
		input    int64
	}{
		{name: "bytes unit", typeName: "alloc_space", unit: "bytes", wantSub: "KB", input: 1024},
		{name: "nanoseconds unit", typeName: "cpu", unit: "nanoseconds", wantSub: "s", input: 1_000_000_000},
		{name: "type contains space", typeName: "alloc_space", unit: "count", wantSub: "KB", input: 2048},
		{name: "type contains bytes", typeName: "inuse_bytes", unit: "count", wantSub: "KB", input: 4096},
		{name: "type contains cpu", typeName: "cpu_time", unit: "count", wantSub: "ms", input: 5_000_000},
		{name: "type contains time", typeName: "wall_time", unit: "count", wantSub: "ms", input: 5_000_000},
		{name: "type contains delay", typeName: "io_delay", unit: "count", wantSub: "ms", input: 5_000_000},
		{name: "unknown type and unit", typeName: "samples", unit: "count", wantSub: "42", input: 42},
		{name: "zero bytes", typeName: "alloc_space", unit: "bytes", wantSub: "0B", input: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			formatter := profileUnitFormatter(tt.typeName, tt.unit)
			result := formatter(tt.input)
			assert.Contains(t, result, tt.wantSub, "profileUnitFormatter(%q, %q)(%d) = %q, want substring %q", tt.typeName, tt.unit, tt.input, result, tt.wantSub)
		})
	}
}

func TestPct_ZeroDenominator(t *testing.T) {
	t.Parallel()

	result := pct(100, 0)
	assert.EqualValues(t, 0, result, "pct(100, 0) = %f, want 0", result)
}

func newMockPprofServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	syntheticProfile := createSyntheticProfile(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/profile", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(syntheticProfile)
	})
	mux.HandleFunc("/debug/pprof/heap", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(syntheticProfile)
	})
	mux.HandleFunc("/debug/pprof/mutex", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(syntheticProfile)
	})
	mux.HandleFunc("/debug/pprof/block", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(syntheticProfile)
	})
	mux.HandleFunc("/debug/pprof/goroutine", func(w http.ResponseWriter, r *http.Request) {
		debug := r.URL.Query().Get("debug")
		if debug == "2" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, "goroutine 1 [running]:\nmain.main()\n\t/app/main.go:10\n")
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "goroutine profile: total 42\n")
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, server.URL + "/debug/pprof"
}

func TestFetchProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		handler     http.HandlerFunc
		name        string
		duration    int
		wantErr     bool
		wantNonZero bool
	}{
		{
			name: "success returns bytes",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("profile data"))
			},
			duration:    0,
			wantNonZero: true,
		},
		{
			name: "zero duration omits seconds param",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Has("seconds") {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_, _ = w.Write([]byte("ok"))
			},
			duration:    0,
			wantNonZero: true,
		},
		{
			name: "non-200 returns error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			t.Cleanup(server.Close)

			data, err := fetchProfile(context.Background(), server.URL, "test", tt.duration)
			if tt.wantErr {
				assert.Error(t, err, "expected error, got nil")
				return
			}
			require.NoError(t, err, "unexpected error: %v", err)
			assert.False(t, tt.wantNonZero && len(data) == 0, "expected non-empty data")
		})
	}
}

func TestFetchProfile_CancelledContext(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		_, _ = w.Write([]byte("late"))
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test cancellation"))

	_, err := fetchProfile(ctx, server.URL, "test", 0)
	assert.Error(t, err, "expected error from cancelled context")
}

func TestFetchProfileData(t *testing.T) {
	t.Parallel()

	syntheticProfile := createSyntheticProfile(t)

	t.Run("duration-based routing", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !r.URL.Query().Has("seconds") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write(syntheticProfile)
		}))
		t.Cleanup(server.Close)

		spec := profileSpec{name: "cpu", endpoint: "profile", durationBased: true}
		data, err := fetchProfileData(context.Background(), spec, server.URL, 1)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.NotEmpty(t, data, "expected non-empty data")
	})

	t.Run("delta routing", func(t *testing.T) {
		t.Parallel()
		var callCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount.Add(1)
			_, _ = w.Write(syntheticProfile)
		}))
		t.Cleanup(server.Close)

		spec := profileSpec{name: "allocs", endpoint: "heap", delta: true}
		data, err := fetchProfileData(context.Background(), spec, server.URL, 1)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.NotEmpty(t, data, "expected non-empty data from delta")
		assert.False(t, callCount.Load() < 2, "delta should call fetchProfile at least twice (before + after)")
	})

	t.Run("non-duration non-delta", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(syntheticProfile)
		}))
		t.Cleanup(server.Close)

		spec := profileSpec{name: "heap", endpoint: "heap"}
		data, err := fetchProfileData(context.Background(), spec, server.URL, 1)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.NotEmpty(t, data, "expected non-empty data")
	})

	t.Run("context cancel during delta wait", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(syntheticProfile)
		}))
		t.Cleanup(server.Close)

		ctx, cancel := context.WithCancelCause(context.Background())

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel(fmt.Errorf("test cancellation"))
		}()

		spec := profileSpec{name: "allocs", endpoint: "heap", delta: true}
		_, err := fetchProfileData(ctx, spec, server.URL, 30)
		assert.Error(t, err, "expected error from cancelled context during delta wait")
	})
}

func TestFetchGoroutineCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		handler http.HandlerFunc
		name    string
		want    int
	}{
		{
			name: "parses total 42",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, "goroutine profile: total 42\n")
			},
			want: 42,
		},
		{
			name: "parses large count",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, "goroutine profile: total 10000\n")
			},
			want: 10000,
		},
		{
			name: "0 on non-200",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: 0,
		},
		{
			name: "0 on invalid body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, "not a goroutine profile")
			},
			want: 0,
		},
		{
			name: "0 on empty body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			t.Cleanup(server.Close)

			got := fetchGoroutineCount(context.Background(), server.URL)
			assert.Equal(t, tt.want, got, "fetchGoroutineCount() = %d, want %d", got, tt.want)
		})
	}

	t.Run("0 on unreachable server", func(t *testing.T) {
		t.Parallel()
		got := fetchGoroutineCount(context.Background(), "http://127.0.0.1:0")
		assert.Equal(t, 0, got, "fetchGoroutineCount() = %d, want 0", got)
	})
}

func TestSnapshotGoroutines(t *testing.T) {
	t.Parallel()

	t.Run("writes file on success", func(t *testing.T) {
		t.Parallel()

		_, pprofBase := newMockPprofServer(t)
		directory := t.TempDir()
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		require.NoError(t, err, "sandbox: %v", err)
		defer func() { _ = sandbox.Close() }()

		var stdout, stderr bytes.Buffer
		snapshotGoroutines(context.Background(), &stdout, &stderr, pprofBase, "baseline", sandbox)

		content, err := os.ReadFile(filepath.Join(directory, "baseline.goroutines.txt"))
		require.NoError(t, err, "expected goroutine snapshot file: %v", err)
		assert.NotEmpty(t, content, "expected non-empty goroutine snapshot")
		assert.Contains(t, stdout.String(), "Goroutine snapshot", "expected stdout to mention goroutine snapshot")
	})

	t.Run("handles non-200 for debug=2", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/goroutine", func(w http.ResponseWriter, r *http.Request) {
			debug := r.URL.Query().Get("debug")
			if debug == "2" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = fmt.Fprint(w, "goroutine profile: total 5\n")
		})
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)

		directory := t.TempDir()
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		require.NoError(t, err, "sandbox: %v", err)
		defer func() { _ = sandbox.Close() }()

		var stdout, stderr bytes.Buffer
		snapshotGoroutines(context.Background(), &stdout, &stderr, server.URL+"/debug/pprof", "test", sandbox)

		assert.Contains(t, stderr.String(), "Warning", "expected warning in stderr")
	})
}

func newMockLoadServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestRunLoad(t *testing.T) {
	t.Parallel()

	t.Run("success counting", func(t *testing.T) {
		t.Parallel()
		server := newMockLoadServer(t)

		result := runLoad(context.Background(), loadConfig{
			url:         server.URL,
			concurrency: 2,
			maxRequests: 20,
		})

		assert.False(t, result.totalRequests < 20, "totalRequests = %d, want >= 20", result.totalRequests)
		assert.EqualValues(t, 0, result.failedRequests, "failedRequests = %d, want 0", result.failedRequests)
	})

	t.Run("non-2xx failure counting", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}))
		t.Cleanup(server.Close)

		result := runLoad(context.Background(), loadConfig{
			url:         server.URL,
			concurrency: 2,
			maxRequests: 10,
		})

		assert.False(t, result.failedRequests < 10, "failedRequests = %d, want >= 10", result.failedRequests)
	})

	t.Run("respects maxRequests", func(t *testing.T) {
		t.Parallel()
		server := newMockLoadServer(t)

		result := runLoad(context.Background(), loadConfig{
			url:         server.URL,
			concurrency: 4,
			maxRequests: 50,
		})

		assert.False(t, result.totalRequests < 50, "totalRequests = %d, want >= 50", result.totalRequests)
	})
}

func TestRunLoad_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := newMockLoadServer(t)

	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		200*time.Millisecond,
		fmt.Errorf("test timeout"),
	)
	defer cancel()

	result := runLoad(ctx, loadConfig{
		url:         server.URL,
		concurrency: 2,
		maxRequests: 0,
	})

	assert.NotEqual(t, 0, result.totalRequests, "expected some requests to complete before cancellation")
}

func TestRunLoad_ErrorRecording(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(server.Close)

	errorCh := make(chan loadErrorRecord, 100)
	_ = runLoad(context.Background(), loadConfig{
		url:         server.URL,
		concurrency: 1,
		maxRequests: 5,
		errorCh:     errorCh,
		phase:       "test-phase",
	})
	close(errorCh)

	var records []loadErrorRecord
	for record := range errorCh {
		records = append(records, record)
	}

	require.NotEmpty(t, records, "expected error records")

	record := records[0]
	assert.Equal(t, "test-phase", record.Phase, "Phase = %q, want %q", record.Phase, "test-phase")
	assert.Equal(t, "status", record.Kind, "Kind = %q, want %q", record.Kind, "status")
	assert.Equal(t, http.StatusTeapot, record.StatusCode, "StatusCode = %d, want %d", record.StatusCode, http.StatusTeapot)
}

func TestRunLoad_HeadersPassed(t *testing.T) {
	t.Parallel()

	var captured atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Store(r.Header.Get("Authorization"))
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	_ = runLoad(context.Background(), loadConfig{
		url:         server.URL,
		concurrency: 1,
		maxRequests: 1,
		headers:     map[string]string{"Authorization": "Bearer test-token"},
	})

	got, ok := captured.Load().(string)
	assert.False(t, !ok || got != "Bearer test-token", "Authorization header = %q, want %q", got, "Bearer test-token")
}

func TestEmitLiveMetrics(t *testing.T) {
	t.Parallel()

	metricsChannel := make(chan metricsMessage, 32)
	latencyCh := make(chan time.Duration, 128)

	var completed, failed, bytesCount atomic.Int64
	completed.Store(100)
	failed.Store(5)
	bytesCount.Store(50000)

	for range 10 {
		latencyCh <- 10 * time.Millisecond
	}

	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		500*time.Millisecond,
		fmt.Errorf("test timeout"),
	)
	defer cancel()

	go emitLiveMetrics(ctx, liveMetricsParams{
		metricsChannel: metricsChannel,
		interval:       10 * time.Millisecond,
		start:          time.Now(),
		completed:      &completed,
		failed:         &failed,
		bytes:          &bytesCount,
		latencyCh:      latencyCh,
	})

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case message := <-metricsChannel:
		assert.EqualValues(t, 100, message.total, "total = %d, want 100", message.total)
		assert.EqualValues(t, 5, message.failed, "failed = %d, want 5", message.failed)
		assert.EqualValues(t, 50000, message.bytesReceived, "bytesReceived = %d, want 50000", message.bytesReceived)
	case <-timer.C:
		require.Fail(t, "timed out waiting for metrics message")
	}
}

func TestEmitLiveMetrics_StopsOnCancel(t *testing.T) {
	t.Parallel()

	metricsChannel := make(chan metricsMessage, 32)
	var completed, failed, bytesCount atomic.Int64

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("immediate cancel"))

	done := make(chan struct{})
	go func() {
		emitLiveMetrics(ctx, liveMetricsParams{
			metricsChannel: metricsChannel,
			interval:       10 * time.Millisecond,
			start:          time.Now(),
			completed:      &completed,
			failed:         &failed,
			bytes:          &bytesCount,
		})
		close(done)
	}()

	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	select {
	case <-done:

	case <-timer.C:
		require.Fail(t, "emitLiveMetrics did not stop after context cancellation")
	}
}

func TestWriteErrorLog(t *testing.T) {
	t.Parallel()

	t.Run("writes JSONL records", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		require.NoError(t, err, "sandbox: %v", err)
		defer func() { _ = sandbox.Close() }()

		errorChannel := make(chan loadErrorRecord, 10)
		errorChannel <- loadErrorRecord{Time: "2026-01-01T00:00:00Z", Phase: "cpu", Kind: "status", StatusCode: 500}
		errorChannel <- loadErrorRecord{Time: "2026-01-01T00:00:01Z", Phase: "cpu", Kind: "transport", Error: "connection refused"}
		close(errorChannel)

		err = writeErrorLog(errorChannel, sandbox)
		require.NoError(t, err, "writeErrorLog error: %v", err)

		data, err := os.ReadFile(filepath.Join(directory, "errors.jsonl"))
		require.NoError(t, err, "read errors.jsonl: %v", err)

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		require.Len(t, lines, 2, "expected 2 lines, got %d", len(lines))

		var record loadErrorRecord
		err = json.Unmarshal([]byte(lines[0]), &record)
		require.NoError(t, err, "unmarshal first line: %v", err)
		assert.Equal(t, 500, record.StatusCode, "StatusCode = %d, want 500", record.StatusCode)
	})

	t.Run("empty channel produces empty file", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		require.NoError(t, err, "sandbox: %v", err)
		defer func() { _ = sandbox.Close() }()

		errorChannel := make(chan loadErrorRecord)
		close(errorChannel)

		err = writeErrorLog(errorChannel, sandbox)
		require.NoError(t, err, "writeErrorLog error: %v", err)

		data, err := os.ReadFile(filepath.Join(directory, "errors.jsonl"))
		require.NoError(t, err, "read errors.jsonl: %v", err)
		assert.Empty(t, data, "expected empty file, got %d bytes", len(data))
	})
}

func TestProfileTUIModel_Init_ReturnsBatchCmd(t *testing.T) {
	t.Parallel()

	phaseCh := make(chan phaseMessage, 1)
	doneCh := make(chan profileDoneMessage, 1)
	metricsCh := make(chan metricsMessage, 1)
	goroutineCh := make(chan goroutineMessage, 1)

	model := newProfileTUIModel("http://test", 30, metricsCh, goroutineCh, phaseCh, doneCh)
	command := model.Init()
	assert.NotNil(t, command, "Init() returned nil command, expected batch")
}

func TestProfileTUIModel_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message   tea.Msg
		checkFunc func(t *testing.T, m *profileTUIModel, command tea.Cmd)
		name      string
	}{
		{
			name:    "WindowSizeMessage updates dimensions",
			message: tea.WindowSizeMsg{Width: 120, Height: 40},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				assert.Equal(t, 120, m.width, "width = %d, want 120", m.width)
				assert.Equal(t, 40, m.height, "height = %d, want 40", m.height)
			},
		},
		{
			name:    "phaseMessage active updates activePhase",
			message: phaseMessage{name: "cpu", status: phaseActive},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				assert.Equal(t, "cpu", m.activePhase, "activePhase = %q, want %q", m.activePhase, "cpu")
				assert.Equal(t, phaseActive, m.phaseStatus["cpu"], "expected cpu phase to be active")
			},
		},
		{
			name:    "phaseMessage done",
			message: phaseMessage{name: "cpu", status: phaseDone},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				assert.Equal(t, phaseDone, m.phaseStatus["cpu"], "expected cpu phase to be done")
			},
		},
		{
			name:    "profileDoneMessage nil err",
			message: profileDoneMessage{err: nil},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				assert.True(t, m.done, "expected done=true")
				assert.NoError(t, m.resultErr, "resultErr = %v, want nil", m.resultErr)
			},
		},
		{
			name:    "profileDoneMessage with error",
			message: profileDoneMessage{err: fmt.Errorf("pipeline failed")},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				assert.True(t, m.done, "expected done=true")
				assert.Error(t, m.resultErr, "expected non-nil resultErr")
			},
		},
		{
			name:    "unknown message returns nil command",
			message: "some unknown message",
			checkFunc: func(t *testing.T, _ *profileTUIModel, command tea.Cmd) {
				t.Helper()
				assert.Nil(t, command, "expected nil command for unknown message")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			phaseCh := make(chan phaseMessage, 16)
			doneCh := make(chan profileDoneMessage, 1)
			metricsCh := make(chan metricsMessage, 16)
			goroutineCh := make(chan goroutineMessage, 16)

			model := newProfileTUIModel("http://test", 30, metricsCh, goroutineCh, phaseCh, doneCh)
			_, command := model.Update(tt.message)
			tt.checkFunc(t, &model, command)
		})
	}
}

func TestProfileTUIModel_Update_TickDrainsChannels(t *testing.T) {
	t.Parallel()

	metricsCh := make(chan metricsMessage, 16)
	goroutineCh := make(chan goroutineMessage, 16)
	phaseCh := make(chan phaseMessage, 16)
	doneCh := make(chan profileDoneMessage, 1)

	model := newProfileTUIModel("http://test", 30, metricsCh, goroutineCh, phaseCh, doneCh)

	metricsCh <- metricsMessage{total: 500, failed: 10, bytesReceived: 25000, rps: 100.0}
	goroutineCh <- goroutineMessage{count: 77}

	model.Update(profileTickMessage(time.Now()))

	assert.EqualValues(t, 500, model.totalRequests, "totalRequests = %d, want 500", model.totalRequests)
	assert.EqualValues(t, 10, model.failedRequests, "failedRequests = %d, want 10", model.failedRequests)
	assert.Equal(t, 77, model.goroutineCount, "goroutineCount = %d, want 77", model.goroutineCount)
}

func TestProfileTUIModel_View(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(m *profileTUIModel)
		wantIn string
	}{
		{
			name:   "width=0 returns Initialising",
			setup:  func(_ *profileTUIModel) {},
			wantIn: "Initialising",
		},
		{
			name: "non-zero renders URL",
			setup: func(m *profileTUIModel) {
				m.width = 80
				m.height = 24
			},
			wantIn: "http://test",
		},
		{
			name: "active phase renders phase name",
			setup: func(m *profileTUIModel) {
				m.width = 80
				m.height = 24
				m.activePhase = "cpu"
				m.phaseStart = time.Now()
			},
			wantIn: "cpu",
		},
		{
			name: "done renders profiling complete",
			setup: func(m *profileTUIModel) {
				m.width = 80
				m.height = 24
				m.done = true
			},
			wantIn: "Profiling complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			phaseCh := make(chan phaseMessage, 16)
			doneCh := make(chan profileDoneMessage, 1)
			metricsCh := make(chan metricsMessage, 16)
			goroutineCh := make(chan goroutineMessage, 16)

			model := newProfileTUIModel("http://test", 30, metricsCh, goroutineCh, phaseCh, doneCh)
			tt.setup(&model)

			view := model.View()
			assert.Contains(t, view.Content, tt.wantIn, "View() body does not contain %q", tt.wantIn)
		})
	}
}

func TestListenPhase(t *testing.T) {
	t.Parallel()

	t.Run("channel message returns phaseMessage", func(t *testing.T) {
		t.Parallel()
		phaseChannel := make(chan phaseMessage, 1)
		phaseChannel <- phaseMessage{name: "cpu", status: phaseActive}

		command := listenPhase(phaseChannel)
		require.NotNil(t, command, "expected non-nil command")

		message := command()
		pm, ok := message.(phaseMessage)
		require.True(t, ok, "expected phaseMessage, got %T", message)
		assert.Equal(t, "cpu", pm.name, "name = %q, want %q", pm.name, "cpu")
	})

	t.Run("closed channel returns nil", func(t *testing.T) {
		t.Parallel()
		phaseChannel := make(chan phaseMessage)
		close(phaseChannel)

		command := listenPhase(phaseChannel)
		message := command()
		assert.Nil(t, message, "expected nil from closed channel, got %T", message)
	})
}

func TestListenDone(t *testing.T) {
	t.Parallel()

	t.Run("channel message returns profileDoneMessage", func(t *testing.T) {
		t.Parallel()
		doneChannel := make(chan profileDoneMessage, 1)
		doneChannel <- profileDoneMessage{err: fmt.Errorf("test error")}

		command := listenDone(doneChannel)
		message := command()
		dm, ok := message.(profileDoneMessage)
		require.True(t, ok, "expected profileDoneMessage, got %T", message)
		assert.Error(t, dm.err, "expected non-nil error")
	})

	t.Run("closed channel returns empty message", func(t *testing.T) {
		t.Parallel()
		doneChannel := make(chan profileDoneMessage)
		close(doneChannel)

		command := listenDone(doneChannel)
		message := command()
		dm, ok := message.(profileDoneMessage)
		require.True(t, ok, "expected profileDoneMessage, got %T", message)
		assert.NoError(t, dm.err, "expected nil error from closed channel, got %v", dm.err)
	})
}

func TestPollGoroutineCount(t *testing.T) {
	t.Parallel()

	_, pprofBase := newMockPprofServer(t)

	goroutineChannel := make(chan goroutineMessage, 16)
	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		3*time.Second,
		fmt.Errorf("test timeout"),
	)
	defer cancel()

	go pollGoroutineCount(ctx, pprofBase, goroutineChannel)

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case message := <-goroutineChannel:
		assert.NotEqual(t, 0, message.count, "expected non-zero goroutine count")
	case <-timer.C:
		require.Fail(t, "timed out waiting for goroutine count")
	}
}

func TestPollGoroutineCount_StopsOnCancel(t *testing.T) {
	t.Parallel()

	_, pprofBase := newMockPprofServer(t)
	goroutineChannel := make(chan goroutineMessage, 16)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("immediate cancel"))

	done := make(chan struct{})
	go func() {
		pollGoroutineCount(ctx, pprofBase, goroutineChannel)
		close(done)
	}()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case <-done:

	case <-timer.C:
		require.Fail(t, "pollGoroutineCount did not stop after context cancellation")
	}
}

func TestEmitPhase(t *testing.T) {
	t.Parallel()

	t.Run("sends to channel when present", func(t *testing.T) {
		t.Parallel()
		phaseCh := make(chan phaseMessage, 1)
		pipeline := pipelineConfig{
			phaseCh: phaseCh,
		}
		pipeline.emitPhase("cpu", phaseActive)

		select {
		case message := <-phaseCh:
			assert.False(t, message.name != "cpu" || message.status != phaseActive, "got phase{%q, %d}, want {cpu, active}", message.name, message.status)
		default:
			assert.Fail(t, "expected message on phaseCh")
		}
	})

	t.Run("no-op when nil", func(t *testing.T) {
		t.Parallel()
		pipeline := pipelineConfig{}
		pipeline.emitPhase("cpu", phaseActive)
	})
}

func TestBuildLoadConfig(t *testing.T) {
	t.Parallel()

	t.Run("without TUI", func(t *testing.T) {
		t.Parallel()
		errorCh := make(chan loadErrorRecord, 1)
		pipeline := pipelineConfig{
			url:     "http://localhost:8080",
			flags:   &profileFlags{concurrency: 50},
			headers: map[string]string{"X-Test": "val"},
		}
		lc := pipeline.buildLoadConfig("baseline", errorCh)

		assert.Equal(t, "http://localhost:8080", lc.url, "url = %q", lc.url)
		assert.Equal(t, 50, lc.concurrency, "concurrency = %d", lc.concurrency)
		assert.EqualValues(t, 0, lc.metricsInterval, "metricsInterval = %v, want 0", lc.metricsInterval)
		assert.Nil(t, lc.metricsCh, "metricsCh should be nil without TUI")
		assert.Equal(t, "baseline", lc.phase, "phase = %q, want baseline", lc.phase)
	})

	t.Run("with TUI", func(t *testing.T) {
		t.Parallel()
		metricsCh := make(chan metricsMessage, 1)
		errorCh := make(chan loadErrorRecord, 1)
		pipeline := pipelineConfig{
			url:       "http://localhost:8080",
			flags:     &profileFlags{concurrency: 100},
			metricsCh: metricsCh,
		}
		lc := pipeline.buildLoadConfig("cpu", errorCh)

		assert.Equal(t, 200*time.Millisecond, lc.metricsInterval, "metricsInterval = %v, want 200ms", lc.metricsInterval)
		assert.NotNil(t, lc.metricsCh, "metricsCh should be set with TUI")
	})
}

func TestRunPipeline_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	_, pprofBase := newMockPprofServer(t)
	loadServer := newMockLoadServer(t)

	directory := t.TempDir()
	var stdout, stderr bytes.Buffer

	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err, "NewCLIFactory error: %v", err)

	flags := &profileFlags{
		concurrency: 2,
		duration:    1,
		output:      directory,
		topN:        5,
	}

	err = runPipeline(context.Background(), pipelineConfig{
		factory:   factory,
		flags:     flags,
		url:       loadServer.URL,
		pprofBase: pprofBase,
		headers:   nil,
		specs:     buildProfileSpecs(flags, nil),
		stdout:    &stdout,
		stderr:    &stderr,
	})

	require.NoError(t, err, "runPipeline error: %v\nstderr: %s", err, stderr.String())

	expectedFiles := []string{
		"live_performance_report.txt",
		"cpu.pprof",
		"allocs.pprof",
		"heap.pprof",
		"mutex.pprof",
		"block.pprof",
		"errors.jsonl",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(directory, f)
		_, err := os.Stat(path)
		assert.False(t, os.IsNotExist(err), "expected file %s to exist", f)
	}
}

func TestRunPipeline_InterruptSkipsPhases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	_, pprofBase := newMockPprofServer(t)
	loadServer := newMockLoadServer(t)
	directory := t.TempDir()

	interruptCtx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("interrupted"))

	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err, "NewCLIFactory error: %v", err)

	var stdout, stderr bytes.Buffer
	flags := &profileFlags{
		concurrency: 2,
		duration:    1,
		output:      directory,
		topN:        5,
	}

	err = runPipeline(context.Background(), pipelineConfig{
		factory:   factory,
		flags:     flags,
		url:       loadServer.URL,
		pprofBase: pprofBase,
		specs:     buildProfileSpecs(flags, nil),
		stdout:    &stdout,
		stderr:    &stderr,
		interrupt: interruptCtx,
	})

	require.NoError(t, err, "runPipeline error: %v", err)

	assert.Contains(t, stdout.String(), "Interrupted", "expected 'Interrupted' in stdout")
}

func TestCapturePhase_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	_, pprofBase := newMockPprofServer(t)
	loadServer := newMockLoadServer(t)
	directory := t.TempDir()

	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err, "sandbox: %v", err)
	defer func() { _ = sandbox.Close() }()

	reportFile, err := sandbox.Create("test_report.txt")
	require.NoError(t, err, "create report: %v", err)
	defer func() { _ = reportFile.Close() }()

	flags := &profileFlags{
		concurrency: 2,
		duration:    1,
		output:      directory,
		topN:        5,
	}

	errorCh := make(chan loadErrorRecord, 100)
	spec := profileSpec{
		name:          "cpu",
		endpoint:      "profile",
		durationBased: true,
		reports: []profileReportConfig{
			{sectionTitle: "cpu", sampleIndex: 1, topN: 5},
		},
	}

	pipeline := pipelineConfig{
		flags:     flags,
		url:       loadServer.URL,
		pprofBase: pprofBase,
		stdout:    io.Discard,
		stderr:    io.Discard,
	}

	captureErr := capturePhase(context.Background(), pipeline, reportFile, spec, sandbox, errorCh)
	require.NoError(t, captureErr, "capturePhase error: %v", captureErr)

	pprofPath := filepath.Join(directory, "cpu.pprof")
	_, err = os.Stat(pprofPath)
	assert.False(t, os.IsNotExist(err), "expected cpu.pprof to exist")
}

var (
	_ = io.Discard
	_ = http.StatusOK
	_ = (*httptest.Server)(nil)
	_ = json.Unmarshal
	_ = (*atomic.Value)(nil)
	_ = tea.Quit
	_ = context.Background
)
