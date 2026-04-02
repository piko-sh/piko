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

	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/wdk/safedisk"
)

func TestParseProfileFlags_Defaults(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	flags, url, ok := parseProfileFlags([]string{"http://localhost:8080/"}, &stderr)
	if !ok {
		t.Fatalf("parseProfileFlags failed: %s", stderr.String())
	}

	if url != "http://localhost:8080/" {
		t.Errorf("url = %q, want %q", url, "http://localhost:8080/")
	}
	if flags.pprofPort != 6060 {
		t.Errorf("pprofPort = %d, want 6060", flags.pprofPort)
	}
	if flags.concurrency != 100 {
		t.Errorf("concurrency = %d, want 100", flags.concurrency)
	}
	if flags.duration != 30 {
		t.Errorf("duration = %d, want 30", flags.duration)
	}
	if flags.output != "pprof" {
		t.Errorf("output = %q, want %q", flags.output, "pprof")
	}
	if flags.topN != 60 {
		t.Errorf("topN = %d, want 60", flags.topN)
	}
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

	if !ok {
		t.Fatalf("parseProfileFlags failed: %s", stderr.String())
	}

	if url != "http://example.com/" {
		t.Errorf("url = %q, want %q", url, "http://example.com/")
	}
	if flags.pprofPort != 9090 {
		t.Errorf("pprofPort = %d, want 9090", flags.pprofPort)
	}
	if flags.concurrency != 50 {
		t.Errorf("concurrency = %d, want 50", flags.concurrency)
	}
	if flags.duration != 10 {
		t.Errorf("duration = %d, want 10", flags.duration)
	}
	if flags.output != "/tmp/profiles" {
		t.Errorf("output = %q, want %q", flags.output, "/tmp/profiles")
	}
	if flags.cookie != "session=abc" {
		t.Errorf("cookie = %q, want %q", flags.cookie, "session=abc")
	}
	if flags.topN != 20 {
		t.Errorf("topN = %d, want 20", flags.topN)
	}
	if flags.focus != "render" {
		t.Errorf("focus = %q, want %q", flags.focus, "render")
	}
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

	if !ok {
		t.Fatalf("parseProfileFlags failed: %s", stderr.String())
	}

	if url != "http://localhost:8080/" {
		t.Errorf("url = %q, want %q", url, "http://localhost:8080/")
	}
	if flags.concurrency != 200 {
		t.Errorf("concurrency = %d, want 200", flags.concurrency)
	}
	if !flags.tui {
		t.Error("tui should be true")
	}
	if flags.duration != 10 {
		t.Errorf("duration = %d, want 10", flags.duration)
	}
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
			if gotURL != tt.wantURL {
				t.Errorf("url = %q, want %q", gotURL, tt.wantURL)
			}
			if len(gotFlags) != len(tt.wantFlagArgs) {
				t.Fatalf("flagArgs len = %d, want %d", len(gotFlags), len(tt.wantFlagArgs))
			}
			for i, f := range gotFlags {
				if f != tt.wantFlagArgs[i] {
					t.Errorf("flagArgs[%d] = %q, want %q", i, f, tt.wantFlagArgs[i])
				}
			}
		})
	}
}

func TestParseProfileFlags_MissingURL(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	_, _, ok := parseProfileFlags([]string{}, &stderr)
	if ok {
		t.Error("expected parseProfileFlags to fail with no URL")
	}
	if !strings.Contains(stderr.String(), "URL to test is a required argument") {
		t.Errorf("stderr should mention missing URL, got: %s", stderr.String())
	}
}

func TestRunProfile_NoArgs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunProfileWithIO(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestRunProfile_Help(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	code := RunProfileWithIO([]string{"-h"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "piko profile") {
		t.Errorf("stderr should contain usage text, got: %s", stderr.String())
	}
}

func TestRunProfile_InvalidFocus(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunProfileWithIO([]string{"--focus", "[invalid", "http://localhost/"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid --focus pattern") {
		t.Errorf("stderr should mention invalid focus, got: %s", stderr.String())
	}
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

	if p50 := result.percentile(50); p50 != 6*time.Millisecond {
		t.Errorf("p50 = %v, want 6ms", p50)
	}
	if p100 := result.percentile(100); p100 != 10*time.Millisecond {
		t.Errorf("p100 = %v, want 10ms", p100)
	}
}

func TestLoadResult_RequestsPerSecond(t *testing.T) {
	t.Parallel()

	result := &loadResult{
		totalRequests: 1000,
		duration:      2 * time.Second,
	}

	rps := result.requestsPerSecond()
	if rps != 500 {
		t.Errorf("requestsPerSecond = %f, want 500", rps)
	}
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
	if mean != 4*time.Millisecond {
		t.Errorf("meanLatency = %v, want 4ms", mean)
	}
}

func TestLoadResult_EmptyLatencies(t *testing.T) {
	t.Parallel()

	result := &loadResult{}
	if result.meanLatency() != 0 {
		t.Error("meanLatency should be 0 for empty latencies")
	}
	if result.percentile(50) != 0 {
		t.Error("percentile should be 0 for empty latencies")
	}
}

func TestFetchProfilerStatus_ReturnsNilWhenEndpointMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	status, err := fetchProfilerStatus(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetchProfilerStatus returned error: %v", err)
	}
	if status != nil {
		t.Fatalf("status = %#v, want nil", status)
	}
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
	if err != nil {
		t.Fatalf("fetchProfilerStatus returned error: %v", err)
	}
	if status == nil {
		t.Fatal("status should not be nil")
	}
	if !status.RollingTrace.Enabled {
		t.Fatal("rolling trace should be enabled")
	}
	if status.RollingTrace.DownloadPath != profiler.RollingTracePath {
		t.Fatalf("download path = %q, want %q", status.RollingTrace.DownloadPath, profiler.RollingTracePath)
	}
}

func TestLoadResult_ZeroDuration(t *testing.T) {
	t.Parallel()

	result := &loadResult{}
	if result.requestsPerSecond() != 0 {
		t.Error("requestsPerSecond should be 0 when duration is 0")
	}
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

	if !strings.Contains(report, "LOAD TEST REPORT") {
		t.Error("report should contain header")
	}
	if !strings.Contains(report, "Complete requests:      10000") {
		t.Error("report should contain total requests")
	}
	if !strings.Contains(report, "Failed requests:        5") {
		t.Error("report should contain failed requests")
	}
	if !strings.Contains(report, "Requests per second:") {
		t.Error("report should contain request/s")
	}
	if !strings.Contains(report, "50%") {
		t.Error("report should contain percentiles")
	}
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
	if err := prof.Write(&buffer); err != nil {
		t.Fatalf("failed to write synthetic profile: %v", err)
	}
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

	if err != nil {
		t.Fatalf("generateProfileReport failed: %v", err)
	}

	report := buffer.String()

	if !strings.Contains(report, "allocs (alloc_space)") {
		t.Error("report should contain section title")
	}
	if !strings.Contains(report, "FuncA") {
		t.Error("report should contain FuncA")
	}
	if !strings.Contains(report, "FuncB") {
		t.Error("report should contain FuncB")
	}
	if !strings.Contains(report, "flat") {
		t.Error("report should contain header row")
	}
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

	if err != nil {
		t.Fatalf("generateProfileReport failed: %v", err)
	}

	report := buffer.String()

	if !strings.Contains(report, "/source/pkg/a.go:42") {
		t.Error("report should contain FuncA file:line")
	}
	if !strings.Contains(report, "/source/pkg/b.go:100") {
		t.Error("report should contain FuncB file:line")
	}
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

	if err != nil {
		t.Fatalf("generateProfileReport failed: %v", err)
	}

	report := buffer.String()

	if !strings.Contains(report, "FuncA") {
		t.Error("report should contain FuncA (matches focus)")
	}
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

	if err == nil {
		t.Error("expected error for out-of-range sample index")
	}
}

func TestGenerateProfileReport_InvalidData(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	err := generateProfileReport(&buffer, []byte("not a pprof file"), profileReportConfig{
		sectionTitle: "bad data",
		sampleIndex:  0,
		topN:         10,
	}, 0)

	if err == nil {
		t.Error("expected error for invalid pprof data")
	}
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

	if err != nil {
		t.Fatalf("generateProfileReport failed: %v", err)
	}

	report := buffer.String()

	if !strings.Contains(report, "flat/request") {
		t.Error("report should contain flat/request header when totalRequests > 0")
	}
	if !strings.Contains(report, "FuncA") {
		t.Error("report should contain FuncA")
	}
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
		if got != tt.expected {
			t.Errorf("profileFormatBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
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
		if got != tt.expected {
			t.Errorf("profileFormatDuration(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestProfileFormatCount(t *testing.T) {
	t.Parallel()

	if got := profileFormatCount(42); got != "42" {
		t.Errorf("profileFormatCount(42) = %q, want %q", got, "42")
	}
}

func TestBuildProfileSpecs(t *testing.T) {
	t.Parallel()

	flags := &profileFlags{topN: 40}
	specs := buildProfileSpecs(flags, nil)

	if len(specs) != 5 {
		t.Fatalf("expected 5 profile specs, got %d", len(specs))
	}

	expectedNames := []string{"cpu", "allocs", "heap", "mutex", "block"}
	for i, spec := range specs {
		if spec.name != expectedNames[i] {
			t.Errorf("spec[%d].name = %q, want %q", i, spec.name, expectedNames[i])
		}
	}

	if len(specs[1].reports) != 2 {
		t.Errorf("allocs spec should have 2 reports, got %d", len(specs[1].reports))
	}
}

func TestProfileUsage(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	profileUsage(&buffer)

	output := buffer.String()
	if !strings.Contains(output, "piko profile") {
		t.Error("usage should mention 'piko profile'")
	}
	if !strings.Contains(output, "--pprof-port") {
		t.Error("usage should mention --pprof-port flag")
	}
	if !strings.Contains(output, "--concurrency") {
		t.Error("usage should mention --concurrency flag")
	}
	if !strings.Contains(output, "--header") {
		t.Error("usage should mention --header flag")
	}
	if !strings.Contains(output, "--tui") {
		t.Error("usage should mention --tui flag")
	}
}

func TestHeaderFlag_Set(t *testing.T) {
	t.Parallel()

	var h headerFlag
	if err := h.Set("Authorization: Bearer token123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.Set("X-Custom: value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h.headers["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization = %q, want %q", h.headers["Authorization"], "Bearer token123")
	}
	if h.headers["X-Custom"] != "value" {
		t.Errorf("X-Custom = %q, want %q", h.headers["X-Custom"], "value")
	}
}

func TestHeaderFlag_SetInvalid(t *testing.T) {
	t.Parallel()

	var h headerFlag
	if err := h.Set("no-colon-here"); err == nil {
		t.Error("expected error for header without colon")
	}
	if err := h.Set(": no-name"); err == nil {
		t.Error("expected error for header with empty name")
	}
}

func TestHeaderFlag_String(t *testing.T) {
	t.Parallel()

	var h headerFlag
	if s := h.String(); s != "" {
		t.Errorf("empty headerFlag.String() = %q, want empty", s)
	}

	_ = h.Set("X-Test: hello")
	s := h.String()
	if !strings.Contains(s, "X-Test: hello") {
		t.Errorf("headerFlag.String() = %q, want to contain %q", s, "X-Test: hello")
	}
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

	if !ok {
		t.Fatalf("parseProfileFlags failed: %s", stderr.String())
	}

	if flags.headers.headers["Authorization"] != "Bearer abc" {
		t.Errorf("Authorization header = %q, want %q", flags.headers.headers["Authorization"], "Bearer abc")
	}
	if flags.headers.headers["X-Custom"] != "val" {
		t.Errorf("X-Custom header = %q, want %q", flags.headers.headers["X-Custom"], "val")
	}
	if flags.cookie != "session=xyz" {
		t.Errorf("cookie = %q, want %q", flags.cookie, "session=xyz")
	}
}

func TestParseProfileFlags_TUI(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	flags, _, ok := parseProfileFlags([]string{
		"--tui",
		"http://localhost:8080/",
	}, &stderr)

	if !ok {
		t.Fatalf("parseProfileFlags failed: %s", stderr.String())
	}

	if !flags.tui {
		t.Error("tui flag should be true")
	}
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

	if h["Cookie"] != "session=abc" {
		t.Errorf("Cookie = %q, want %q", h["Cookie"], "session=abc")
	}
	if h["Authorization"] != "Bearer token" {
		t.Errorf("Authorization = %q, want %q", h["Authorization"], "Bearer token")
	}
	if h["X-Custom"] != "val" {
		t.Errorf("X-Custom = %q, want %q", h["X-Custom"], "val")
	}
}

func TestMergedHeaders_NoCookie(t *testing.T) {
	t.Parallel()

	flags := &profileFlags{}
	h := flags.mergedHeaders()

	if _, hasCookie := h["Cookie"]; hasCookie {
		t.Error("should not have Cookie header when cookie is empty")
	}
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
	if err != nil {
		t.Fatalf("creating sandbox: %v", err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := writeProfileStats(sandbox, "cpu.pprof.stats", result, 200); err != nil {
		t.Fatalf("writeProfileStats failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(directory, "cpu.pprof.stats"))
	if err != nil {
		t.Fatalf("reading stats file: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "total_requests:     50000") {
		t.Error("stats should contain total_requests")
	}
	if !strings.Contains(content, "failed_requests:    12") {
		t.Error("stats should contain failed_requests")
	}
	if !strings.Contains(content, "concurrency:        200") {
		t.Error("stats should contain concurrency")
	}
	if !strings.Contains(content, "requests_per_sec:") {
		t.Error("stats should contain requests_per_sec")
	}
	if !strings.Contains(content, "bytes_received:     524288000") {
		t.Error("stats should contain bytes_received")
	}
}

func TestProfileTUIModel_Init(t *testing.T) {
	t.Parallel()

	metricsCh := make(chan metricsMessage, 1)
	goroutineCh := make(chan goroutineMessage, 1)
	phaseCh := make(chan phaseMessage, 1)
	doneCh := make(chan profileDoneMessage, 1)

	model := newProfileTUIModel("http://localhost/", 30, metricsCh, goroutineCh, phaseCh, doneCh)

	if model.targetURL != "http://localhost/" {
		t.Errorf("targetURL = %q, want %q", model.targetURL, "http://localhost/")
	}
	if len(model.phases) != 6 {
		t.Errorf("phases count = %d, want 6", len(model.phases))
	}
	for _, p := range model.phases {
		if model.phaseStatus[p] != phasePending {
			t.Errorf("phase %q should be pending", p)
		}
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
	if !ok {
		t.Fatal("unexpected model type")
	}
	if m.activePhase != "cpu" {
		t.Errorf("activePhase = %q, want %q", m.activePhase, "cpu")
	}
	if m.phaseStatus["cpu"] != phaseActive {
		t.Error("cpu phase should be active")
	}

	updated, _ = m.Update(phaseMessage{name: "cpu", status: phaseDone})
	m, ok = updated.(*profileTUIModel)
	if !ok {
		t.Fatal("unexpected model type")
	}
	if m.phaseStatus["cpu"] != phaseDone {
		t.Error("cpu phase should be done")
	}
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
	if !ok {
		t.Fatal("unexpected model type")
	}

	if m.currentRPS != 1200 {
		t.Errorf("currentRPS = %f, want 1200", m.currentRPS)
	}
	if m.totalRequests != 200 {
		t.Errorf("totalRequests = %d, want 200", m.totalRequests)
	}
	if m.rpsHistory.Len() != 2 {
		t.Errorf("rpsHistory.Len() = %d, want 2", m.rpsHistory.Len())
	}
	if m.p50Ms != 2.5 {
		t.Errorf("p50Ms = %f, want 2.5", m.p50Ms)
	}
	if m.p80Ms != 3.5 {
		t.Errorf("p80Ms = %f, want 3.5", m.p80Ms)
	}
	if m.p99Ms != 7.0 {
		t.Errorf("p99Ms = %f, want 7.0", m.p99Ms)
	}
	if m.p100Ms != 9.0 {
		t.Errorf("p100Ms = %f, want 9.0", m.p100Ms)
	}
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

	if p50 := latencyPercentileMs(sorted, 50); p50 != 6.0 {
		t.Errorf("p50 = %f, want 6.0", p50)
	}
	if p99 := latencyPercentileMs(sorted, 99); p99 != 10.0 {
		t.Errorf("p99 = %f, want 10.0", p99)
	}
	if p100 := latencyPercentileMs(sorted, 100); p100 != 10.0 {
		t.Errorf("p100 = %f, want 10.0", p100)
	}

	if p := latencyPercentileMs(nil, 50); p != 0 {
		t.Errorf("empty p50 = %f, want 0", p)
	}
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
	if err := prof.Write(&buffer); err != nil {
		t.Fatalf("failed to write synthetic heap profile: %v", err)
	}
	return buffer.Bytes()
}

func TestComputeDeltaProfile(t *testing.T) {
	t.Parallel()

	before := createSyntheticHeapProfile(t, 1000, 1024*1024)
	after := createSyntheticHeapProfile(t, 1500, 1536*1024)

	deltaData, err := computeDeltaProfile(before, after)
	if err != nil {
		t.Fatalf("computeDeltaProfile failed: %v", err)
	}

	prof, err := profile.ParseData(deltaData)
	if err != nil {
		t.Fatalf("parsing delta profile: %v", err)
	}

	var totalObjects, totalSpace int64
	for _, s := range prof.Sample {
		totalObjects += s.Value[0]
		totalSpace += s.Value[1]
	}

	if totalObjects != 500 {
		t.Errorf("delta alloc_objects = %d, want 500", totalObjects)
	}
	expectedSpace := int64(512 * 1024)
	if totalSpace != expectedSpace {
		t.Errorf("delta alloc_space = %d, want %d", totalSpace, expectedSpace)
	}
}

func TestComputeDeltaProfile_InvalidData(t *testing.T) {
	t.Parallel()

	valid := createSyntheticHeapProfile(t, 100, 1024)

	if _, err := computeDeltaProfile([]byte("bad"), valid); err == nil {
		t.Error("expected error for invalid before data")
	}
	if _, err := computeDeltaProfile(valid, []byte("bad")); err == nil {
		t.Error("expected error for invalid after data")
	}
}

func TestWriteAllocChurnSummary(t *testing.T) {
	t.Parallel()

	before := createSyntheticHeapProfile(t, 1000, 1024*1024)
	after := createSyntheticHeapProfile(t, 6000, 6*1024*1024)

	deltaData, err := computeDeltaProfile(before, after)
	if err != nil {
		t.Fatalf("computeDeltaProfile failed: %v", err)
	}

	var buffer bytes.Buffer
	if err := writeAllocChurnSummary(&buffer, deltaData, 1000); err != nil {
		t.Fatalf("writeAllocChurnSummary failed: %v", err)
	}

	output := buffer.String()

	if !strings.Contains(output, "ALLOCATION CHURN") {
		t.Error("summary should contain header")
	}
	if !strings.Contains(output, "5000") {
		t.Error("summary should contain delta alloc_objects (5000)")
	}
	if !strings.Contains(output, "Requests during load: 1000") {
		t.Error("summary should contain request count")
	}
	if !strings.Contains(output, "/request") {
		t.Error("summary should contain per-request stats")
	}
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
	if allocsSpec == nil {
		t.Fatal("allocs spec not found")
	}
	if !allocsSpec.delta {
		t.Error("allocs spec should have delta=true")
	}

	for _, spec := range specs {
		if spec.name == "heap" && spec.delta {
			t.Error("heap spec should not have delta=true")
		}
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
			if !strings.Contains(result, tt.wantSub) {
				t.Errorf("profileUnitFormatter(%q, %q)(%d) = %q, want substring %q",
					tt.typeName, tt.unit, tt.input, result, tt.wantSub)
			}
		})
	}
}

func TestPct_ZeroDenominator(t *testing.T) {
	t.Parallel()

	result := pct(100, 0)
	if result != 0 {
		t.Errorf("pct(100, 0) = %f, want 0", result)
	}
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
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNonZero && len(data) == 0 {
				t.Error("expected non-empty data")
			}
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
	if err == nil {
		t.Error("expected error from cancelled context")
	}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty data")
		}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty data from delta")
		}
		if callCount.Load() < 2 {
			t.Error("delta should call fetchProfile at least twice (before + after)")
		}
	})

	t.Run("non-duration non-delta", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(syntheticProfile)
		}))
		t.Cleanup(server.Close)

		spec := profileSpec{name: "heap", endpoint: "heap"}
		data, err := fetchProfileData(context.Background(), spec, server.URL, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty data")
		}
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
		if err == nil {
			t.Error("expected error from cancelled context during delta wait")
		}
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
			if got != tt.want {
				t.Errorf("fetchGoroutineCount() = %d, want %d", got, tt.want)
			}
		})
	}

	t.Run("0 on unreachable server", func(t *testing.T) {
		t.Parallel()
		got := fetchGoroutineCount(context.Background(), "http://127.0.0.1:0")
		if got != 0 {
			t.Errorf("fetchGoroutineCount() = %d, want 0", got)
		}
	})
}

func TestSnapshotGoroutines(t *testing.T) {
	t.Parallel()

	t.Run("writes file on success", func(t *testing.T) {
		t.Parallel()

		_, pprofBase := newMockPprofServer(t)
		directory := t.TempDir()
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		if err != nil {
			t.Fatalf("sandbox: %v", err)
		}
		defer func() { _ = sandbox.Close() }()

		var stdout, stderr bytes.Buffer
		snapshotGoroutines(context.Background(), &stdout, &stderr, pprofBase, "baseline", sandbox)

		content, err := os.ReadFile(filepath.Join(directory, "baseline.goroutines.txt"))
		if err != nil {
			t.Fatalf("expected goroutine snapshot file: %v", err)
		}
		if len(content) == 0 {
			t.Error("expected non-empty goroutine snapshot")
		}
		if !strings.Contains(stdout.String(), "Goroutine snapshot") {
			t.Error("expected stdout to mention goroutine snapshot")
		}
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
		if err != nil {
			t.Fatalf("sandbox: %v", err)
		}
		defer func() { _ = sandbox.Close() }()

		var stdout, stderr bytes.Buffer
		snapshotGoroutines(context.Background(), &stdout, &stderr, server.URL+"/debug/pprof", "test", sandbox)

		if !strings.Contains(stderr.String(), "Warning") {
			t.Error("expected warning in stderr")
		}
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

		if result.totalRequests < 20 {
			t.Errorf("totalRequests = %d, want >= 20", result.totalRequests)
		}
		if result.failedRequests != 0 {
			t.Errorf("failedRequests = %d, want 0", result.failedRequests)
		}
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

		if result.failedRequests < 10 {
			t.Errorf("failedRequests = %d, want >= 10", result.failedRequests)
		}
	})

	t.Run("respects maxRequests", func(t *testing.T) {
		t.Parallel()
		server := newMockLoadServer(t)

		result := runLoad(context.Background(), loadConfig{
			url:         server.URL,
			concurrency: 4,
			maxRequests: 50,
		})

		if result.totalRequests < 50 {
			t.Errorf("totalRequests = %d, want >= 50", result.totalRequests)
		}
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

	if result.totalRequests == 0 {
		t.Error("expected some requests to complete before cancellation")
	}
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

	if len(records) == 0 {
		t.Fatal("expected error records")
	}

	record := records[0]
	if record.Phase != "test-phase" {
		t.Errorf("Phase = %q, want %q", record.Phase, "test-phase")
	}
	if record.Kind != "status" {
		t.Errorf("Kind = %q, want %q", record.Kind, "status")
	}
	if record.StatusCode != http.StatusTeapot {
		t.Errorf("StatusCode = %d, want %d", record.StatusCode, http.StatusTeapot)
	}
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
	if !ok || got != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
	}
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
		if message.total != 100 {
			t.Errorf("total = %d, want 100", message.total)
		}
		if message.failed != 5 {
			t.Errorf("failed = %d, want 5", message.failed)
		}
		if message.bytesReceived != 50000 {
			t.Errorf("bytesReceived = %d, want 50000", message.bytesReceived)
		}
	case <-timer.C:
		t.Fatal("timed out waiting for metrics message")
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
		t.Fatal("emitLiveMetrics did not stop after context cancellation")
	}
}

func TestWriteErrorLog(t *testing.T) {
	t.Parallel()

	t.Run("writes JSONL records", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		if err != nil {
			t.Fatalf("sandbox: %v", err)
		}
		defer func() { _ = sandbox.Close() }()

		errorChannel := make(chan loadErrorRecord, 10)
		errorChannel <- loadErrorRecord{Time: "2026-01-01T00:00:00Z", Phase: "cpu", Kind: "status", StatusCode: 500}
		errorChannel <- loadErrorRecord{Time: "2026-01-01T00:00:01Z", Phase: "cpu", Kind: "transport", Error: "connection refused"}
		close(errorChannel)

		if err := writeErrorLog(errorChannel, sandbox); err != nil {
			t.Fatalf("writeErrorLog error: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(directory, "errors.jsonl"))
		if err != nil {
			t.Fatalf("read errors.jsonl: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		var record loadErrorRecord
		if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
			t.Fatalf("unmarshal first line: %v", err)
		}
		if record.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", record.StatusCode)
		}
	})

	t.Run("empty channel produces empty file", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		if err != nil {
			t.Fatalf("sandbox: %v", err)
		}
		defer func() { _ = sandbox.Close() }()

		errorChannel := make(chan loadErrorRecord)
		close(errorChannel)

		if err := writeErrorLog(errorChannel, sandbox); err != nil {
			t.Fatalf("writeErrorLog error: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(directory, "errors.jsonl"))
		if err != nil {
			t.Fatalf("read errors.jsonl: %v", err)
		}
		if len(data) != 0 {
			t.Errorf("expected empty file, got %d bytes", len(data))
		}
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
	if command == nil {
		t.Error("Init() returned nil command, expected batch")
	}
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
				if m.width != 120 {
					t.Errorf("width = %d, want 120", m.width)
				}
				if m.height != 40 {
					t.Errorf("height = %d, want 40", m.height)
				}
			},
		},
		{
			name:    "phaseMessage active updates activePhase",
			message: phaseMessage{name: "cpu", status: phaseActive},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				if m.activePhase != "cpu" {
					t.Errorf("activePhase = %q, want %q", m.activePhase, "cpu")
				}
				if m.phaseStatus["cpu"] != phaseActive {
					t.Error("expected cpu phase to be active")
				}
			},
		},
		{
			name:    "phaseMessage done",
			message: phaseMessage{name: "cpu", status: phaseDone},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				if m.phaseStatus["cpu"] != phaseDone {
					t.Error("expected cpu phase to be done")
				}
			},
		},
		{
			name:    "profileDoneMessage nil err",
			message: profileDoneMessage{err: nil},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				if !m.done {
					t.Error("expected done=true")
				}
				if m.resultErr != nil {
					t.Errorf("resultErr = %v, want nil", m.resultErr)
				}
			},
		},
		{
			name:    "profileDoneMessage with error",
			message: profileDoneMessage{err: fmt.Errorf("pipeline failed")},
			checkFunc: func(t *testing.T, m *profileTUIModel, _ tea.Cmd) {
				t.Helper()
				if !m.done {
					t.Error("expected done=true")
				}
				if m.resultErr == nil {
					t.Error("expected non-nil resultErr")
				}
			},
		},
		{
			name:    "unknown message returns nil command",
			message: "some unknown message",
			checkFunc: func(t *testing.T, _ *profileTUIModel, command tea.Cmd) {
				t.Helper()
				if command != nil {
					t.Error("expected nil command for unknown message")
				}
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

	if model.totalRequests != 500 {
		t.Errorf("totalRequests = %d, want 500", model.totalRequests)
	}
	if model.failedRequests != 10 {
		t.Errorf("failedRequests = %d, want 10", model.failedRequests)
	}
	if model.goroutineCount != 77 {
		t.Errorf("goroutineCount = %d, want 77", model.goroutineCount)
	}
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
			if !strings.Contains(view.Content, tt.wantIn) {
				t.Errorf("View() body does not contain %q", tt.wantIn)
			}
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
		if command == nil {
			t.Fatal("expected non-nil command")
		}

		message := command()
		pm, ok := message.(phaseMessage)
		if !ok {
			t.Fatalf("expected phaseMessage, got %T", message)
		}
		if pm.name != "cpu" {
			t.Errorf("name = %q, want %q", pm.name, "cpu")
		}
	})

	t.Run("closed channel returns nil", func(t *testing.T) {
		t.Parallel()
		phaseChannel := make(chan phaseMessage)
		close(phaseChannel)

		command := listenPhase(phaseChannel)
		message := command()
		if message != nil {
			t.Errorf("expected nil from closed channel, got %T", message)
		}
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
		if !ok {
			t.Fatalf("expected profileDoneMessage, got %T", message)
		}
		if dm.err == nil {
			t.Error("expected non-nil error")
		}
	})

	t.Run("closed channel returns empty message", func(t *testing.T) {
		t.Parallel()
		doneChannel := make(chan profileDoneMessage)
		close(doneChannel)

		command := listenDone(doneChannel)
		message := command()
		dm, ok := message.(profileDoneMessage)
		if !ok {
			t.Fatalf("expected profileDoneMessage, got %T", message)
		}
		if dm.err != nil {
			t.Errorf("expected nil error from closed channel, got %v", dm.err)
		}
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
		if message.count == 0 {
			t.Error("expected non-zero goroutine count")
		}
	case <-timer.C:
		t.Fatal("timed out waiting for goroutine count")
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
		t.Fatal("pollGoroutineCount did not stop after context cancellation")
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
			if message.name != "cpu" || message.status != phaseActive {
				t.Errorf("got phase{%q, %d}, want {cpu, active}", message.name, message.status)
			}
		default:
			t.Error("expected message on phaseCh")
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

		if lc.url != "http://localhost:8080" {
			t.Errorf("url = %q", lc.url)
		}
		if lc.concurrency != 50 {
			t.Errorf("concurrency = %d", lc.concurrency)
		}
		if lc.metricsInterval != 0 {
			t.Errorf("metricsInterval = %v, want 0", lc.metricsInterval)
		}
		if lc.metricsCh != nil {
			t.Error("metricsCh should be nil without TUI")
		}
		if lc.phase != "baseline" {
			t.Errorf("phase = %q, want baseline", lc.phase)
		}
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

		if lc.metricsInterval != 200*time.Millisecond {
			t.Errorf("metricsInterval = %v, want 200ms", lc.metricsInterval)
		}
		if lc.metricsCh == nil {
			t.Error("metricsCh should be set with TUI")
		}
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
	if err != nil {
		t.Fatalf("NewCLIFactory error: %v", err)
	}

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

	if err != nil {
		t.Fatalf("runPipeline error: %v\nstderr: %s", err, stderr.String())
	}

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
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
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
	if err != nil {
		t.Fatalf("NewCLIFactory error: %v", err)
	}

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

	if err != nil {
		t.Fatalf("runPipeline error: %v", err)
	}

	if !strings.Contains(stdout.String(), "Interrupted") {
		t.Error("expected 'Interrupted' in stdout")
	}
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
	if err != nil {
		t.Fatalf("sandbox: %v", err)
	}
	defer func() { _ = sandbox.Close() }()

	reportFile, err := sandbox.Create("test_report.txt")
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
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
	if captureErr != nil {
		t.Fatalf("capturePhase error: %v", captureErr)
	}

	pprofPath := filepath.Join(directory, "cpu.pprof")
	if _, err := os.Stat(pprofPath); os.IsNotExist(err) {
		t.Error("expected cpu.pprof to exist")
	}
}

var _ = io.Discard
var _ = http.StatusOK
var _ = (*httptest.Server)(nil)
var _ = json.Unmarshal
var _ = (*atomic.Value)(nil)
var _ = tea.Quit
var _ = context.Background
