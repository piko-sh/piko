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
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/internal/json"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		ts       int64
	}{
		{name: "zero returns dash", ts: 0, expected: "-"},
		{name: "valid timestamp", ts: 1700000000, expected: "2023-11-14"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatTimestamp(tc.ts)
			if !strings.Contains(got, tc.expected) {
				t.Errorf("formatTimestamp(%d) = %q, want it to contain %q", tc.ts, got, tc.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		ms       int64
	}{
		{name: "sub-second", ms: 500, expected: "500ms"},
		{name: "seconds", ms: 2500, expected: "2.5s"},
		{name: "minutes", ms: 90000, expected: "1m30s"},
		{name: "hours", ms: 3660000, expected: "1h1m"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatDuration(tc.ms)
			if got != tc.expected {
				t.Errorf("formatDuration(%d) = %q, want %q", tc.ms, got, tc.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		bytes    uint64
	}{
		{name: "bytes", bytes: 512, expected: "512 B"},
		{name: "kibibytes", bytes: 2048, expected: "2.0 KiB"},
		{name: "mebibytes", bytes: 5 * 1024 * 1024, expected: "5.0 MiB"},
		{name: "gibibytes", bytes: 3 * 1024 * 1024 * 1024, expected: "3.0 GiB"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatBytes(tc.bytes)
			if got != tc.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tc.bytes, got, tc.expected)
			}
		})
	}
}

func TestFormatNanos(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		ns       int64
	}{
		{name: "nanoseconds", ns: 500, expected: "500ns"},
		{name: "microseconds", ns: 1500, expected: "1.5us"},
		{name: "milliseconds", ns: 1500000, expected: "1.5ms"},
		{name: "seconds", ns: 1500000000, expected: "1.50s"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatNanos(tc.ns)
			if got != tc.expected {
				t.Errorf("formatNanos(%d) = %q, want %q", tc.ns, got, tc.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
		maxLen   int
	}{
		{name: "short string unchanged", input: "abc", maxLen: 10, expected: "abc"},
		{name: "exact boundary", input: "abcde", maxLen: 5, expected: "abcde"},
		{name: "truncated with ellipsis", input: "abcdefghij", maxLen: 7, expected: "abcd..."},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := truncate(tc.input, tc.maxLen)
			if got != tc.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.expected)
			}
		})
	}
}

func TestFilterErrorSpans(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		spans   []*pb.Span
		wantLen int
	}{
		{name: "empty input", spans: nil, wantLen: 0},
		{
			name: "filters errors only",
			spans: []*pb.Span{
				{Status: "error"},
				{Status: "ok"},
				{Status: "ERROR"},
				{Status: "healthy"},
			},
			wantLen: 2,
		},
		{
			name: "no errors returns empty",
			spans: []*pb.Span{
				{Status: "ok"},
			},
			wantLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := filterErrorSpans(tc.spans)
			if len(got) != tc.wantLen {
				t.Errorf("filterErrorSpans() returned %d spans, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestPrinter_IsJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		format   string
		expected bool
	}{
		{name: "json format", format: "json", expected: true},
		{name: "table format", format: "table", expected: false},
		{name: "empty format", format: "", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewPrinter(&bytes.Buffer{}, tc.format, false, false)
			if got := p.IsJSON(); got != tc.expected {
				t.Errorf("IsJSON() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestPrinter_PrintJSON(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "json", false, false)

	data := map[string]string{"key": "value"}
	if err := p.PrintJSON(data); err != nil {
		t.Fatalf("PrintJSON() error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key=value, got key=%s", result["key"])
	}
}

func TestPrinter_PrintTable(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", false, false)

	p.PrintTable(
		[]string{"NAME", "STATUS"},
		[][]string{
			{"alpha", "ok"},
			{"beta", "error"},
		},
	)

	output := buffer.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected output to contain header NAME")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("expected output to contain header STATUS")
	}
	if !strings.Contains(output, "alpha") {
		t.Error("expected output to contain row value alpha")
	}
	if !strings.Contains(output, "beta") {
		t.Error("expected output to contain row value beta")
	}

	if !strings.Contains(output, "----") {
		t.Error("expected output to contain separator dashes")
	}
}

func TestPrinter_ColourisedStatus_NoColour(t *testing.T) {
	t.Parallel()

	p := NewPrinter(&bytes.Buffer{}, "table", true, false)

	testCases := []struct {
		name   string
		status string
	}{
		{name: "healthy", status: "HEALTHY"},
		{name: "error", status: "error"},
		{name: "unknown", status: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := p.ColourisedStatus(tc.status)
			if got != tc.status {
				t.Errorf("ColourisedStatus(%q) with noColour=true = %q, want %q", tc.status, got, tc.status)
			}
		})
	}
}

func TestPrinter_IsWide(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		format   string
		expected bool
	}{
		{name: "wide format", format: "wide", expected: true},
		{name: "table format", format: "table", expected: false},
		{name: "json format", format: "json", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewPrinter(&bytes.Buffer{}, tc.format, false, false)
			if got := p.IsWide(); got != tc.expected {
				t.Errorf("IsWide() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestPrinter_PrintResource_Standard(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)

	columns := []Column{
		{Header: "NAME"},
		{Header: "STATUS"},
		{Header: "EXTRA", WideOnly: true},
	}
	rows := [][]string{
		{"alpha", "ok", "detail-a"},
		{"beta", "error", "detail-b"},
	}

	p.PrintResource(columns, rows)

	output := buffer.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected NAME header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("expected STATUS header")
	}
	if strings.Contains(output, "EXTRA") {
		t.Error("wide-only column EXTRA should not appear in standard mode")
	}
	if strings.Contains(output, "detail-a") {
		t.Error("wide-only data should not appear in standard mode")
	}
}

func TestPrinter_PrintResource_Wide(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "wide", true, false)

	columns := []Column{
		{Header: "NAME"},
		{Header: "STATUS"},
		{Header: "EXTRA", WideOnly: true},
	}
	rows := [][]string{
		{"alpha", "ok", "detail-a"},
	}

	p.PrintResource(columns, rows)

	output := buffer.String()
	if !strings.Contains(output, "EXTRA") {
		t.Error("wide-only column EXTRA should appear in wide mode")
	}
	if !strings.Contains(output, "detail-a") {
		t.Error("wide-only data should appear in wide mode")
	}
}

func TestPrinter_PrintResource_NoHeaders(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, true)

	columns := []Column{
		{Header: "NAME"},
		{Header: "STATUS"},
	}
	rows := [][]string{
		{"alpha", "ok"},
	}

	p.PrintResource(columns, rows)

	output := buffer.String()
	if strings.Contains(output, "NAME") {
		t.Error("headers should not appear when noHeaders is true")
	}
	if !strings.Contains(output, "alpha") {
		t.Error("row data should still appear")
	}
}

func TestPrinter_PrintDetail(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	p := NewPrinter(&buffer, "table", true, false)

	sections := []inspector.DetailSection{
		{
			Heading: "Component",
			Rows: []inspector.DetailRow{
				{Label: "State", Value: "HEALTHY"},
				{Label: "Message", Value: "All good"},
			},
			SubSections: []inspector.DetailSection{
				{
					Heading: "Database",
					Rows: []inspector.DetailRow{
						{Label: "State", Value: "HEALTHY"},
					},
				},
			},
		},
	}

	p.PrintDetail(sections)

	output := buffer.String()
	if !strings.Contains(output, "Component:") {
		t.Error("expected section title")
	}
	if !strings.Contains(output, "State:") {
		t.Error("expected field key")
	}
	if !strings.Contains(output, "HEALTHY") {
		t.Error("expected field value")
	}
	if !strings.Contains(output, "Database:") {
		t.Error("expected sub-section title")
	}
}

func TestVisibleColumnIndices(t *testing.T) {
	t.Parallel()

	columns := []Column{
		{Header: "A"},
		{Header: "B", WideOnly: true},
		{Header: "C"},
	}

	standard := visibleColumnIndices(columns, false)
	if len(standard) != 2 || standard[0] != 0 || standard[1] != 2 {
		t.Errorf("standard indices = %v, want [0, 2]", standard)
	}

	wide := visibleColumnIndices(columns, true)
	if len(wide) != 3 {
		t.Errorf("wide indices = %v, want [0, 1, 2]", wide)
	}
}

func TestMatchesFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		value    string
		filter   string
		expected bool
	}{
		{name: "empty filter matches all", value: "anything", filter: "", expected: true},
		{name: "exact match", value: "Liveness", filter: "Liveness", expected: true},
		{name: "case insensitive", value: "Liveness", filter: "liveness", expected: true},
		{name: "prefix match", value: "Liveness", filter: "Live", expected: true},
		{name: "no match", value: "Readiness", filter: "Liveness", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := matchesFilter(tc.value, tc.filter)
			if got != tc.expected {
				t.Errorf("matchesFilter(%q, %q) = %v, want %v", tc.value, tc.filter, got, tc.expected)
			}
		})
	}
}

func TestExtractFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		expected  string
		arguments []string
	}{
		{name: "empty arguments", arguments: []string{}, expected: ""},
		{name: "flag argument", arguments: []string{"--limit", "5"}, expected: ""},
		{name: "name argument", arguments: []string{"Liveness", "--limit", "5"}, expected: "Liveness"},
		{name: "name only", arguments: []string{"Liveness"}, expected: "Liveness"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractFilter(tc.arguments)
			if got != tc.expected {
				t.Errorf("extractFilter(%v) = %q, want %q", tc.arguments, got, tc.expected)
			}
		})
	}
}

func TestParseInterspersed(t *testing.T) {
	t.Parallel()

	t.Run("flags before positional", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		limit := fs.Int("limit", 20, "max items")
		positional, err := parseInterspersed(fs, []string{"--limit", "5", "Liveness"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *limit != 5 {
			t.Errorf("limit = %d, want 5", *limit)
		}
		if len(positional) != 1 || positional[0] != "Liveness" {
			t.Errorf("positional = %v, want [Liveness]", positional)
		}
	})

	t.Run("flags after positional", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		limit := fs.Int("limit", 20, "max items")
		positional, err := parseInterspersed(fs, []string{"Liveness", "--limit", "10"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *limit != 10 {
			t.Errorf("limit = %d, want 10", *limit)
		}
		if len(positional) != 1 || positional[0] != "Liveness" {
			t.Errorf("positional = %v, want [Liveness]", positional)
		}
	})

	t.Run("help flag returns ErrHelp", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(&bytes.Buffer{})
		_, err := parseInterspersed(fs, []string{"Liveness", "--help"})
		if !errors.Is(err, flag.ErrHelp) {
			t.Errorf("err = %v, want flag.ErrHelp", err)
		}
	})

	t.Run("no arguments", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		positional, err := parseInterspersed(fs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(positional) != 0 {
			t.Errorf("positional = %v, want empty", positional)
		}
	})

	t.Run("bool flag interspersed", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		showErrors := fs.Bool("errors", false, "show errors only")
		positional, err := parseInterspersed(fs, []string{"Liveness", "--errors"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !*showErrors {
			t.Error("errors flag should be true")
		}
		if len(positional) != 1 || positional[0] != "Liveness" {
			t.Errorf("positional = %v, want [Liveness]", positional)
		}
	})

	t.Run("equals syntax", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		limit := fs.Int("limit", 20, "max items")
		positional, err := parseInterspersed(fs, []string{"Liveness", "--limit=5"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *limit != 5 {
			t.Errorf("limit = %d, want 5", *limit)
		}
		if len(positional) != 1 || positional[0] != "Liveness" {
			t.Errorf("positional = %v, want [Liveness]", positional)
		}
	})
}

func TestGrpcError(t *testing.T) {
	t.Parallel()

	t.Run("unimplemented shows friendly message", func(t *testing.T) {
		t.Parallel()
		err := status.Error(codes.Unimplemented, "unknown service foo.bar.BazService")
		got := grpcError("fetching dispatcher summary", err)
		if !strings.Contains(got.Error(), "service not available") {
			t.Errorf("expected friendly message, got: %s", got)
		}
		if !strings.Contains(got.Error(), "fetching dispatcher summary") {
			t.Errorf("expected context in message, got: %s", got)
		}
	})

	t.Run("other errors pass through", func(t *testing.T) {
		t.Parallel()
		err := status.Error(codes.Unavailable, "connection refused")
		got := grpcError("fetching health", err)
		if strings.Contains(got.Error(), "service not available") {
			t.Errorf("non-Unimplemented error should not say 'service not available', got: %s", got)
		}
		if !strings.Contains(got.Error(), "connection refused") {
			t.Errorf("expected original error message, got: %s", got)
		}
	})

	t.Run("non-grpc errors pass through", func(t *testing.T) {
		t.Parallel()
		err := fmt.Errorf("some random error")
		got := grpcError("fetching tasks", err)
		if !strings.Contains(got.Error(), "some random error") {
			t.Errorf("expected original error, got: %s", got)
		}
	})
}

func TestPrinter_GetLimit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		globalLimit    int
		handlerDefault int
		expected       int
	}{
		{name: "no global limit uses handler default", globalLimit: 0, handlerDefault: 20, expected: 20},
		{name: "global limit overrides handler default", globalLimit: 5, handlerDefault: 20, expected: 5},
		{name: "different handler defaults", globalLimit: 0, handlerDefault: 50, expected: 50},
		{name: "global limit of 1", globalLimit: 1, handlerDefault: 20, expected: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewPrinter(&bytes.Buffer{}, "table", false, false)
			p.SetLimit(tc.globalLimit)
			got := p.GetLimit(tc.handlerDefault)
			if got != tc.expected {
				t.Errorf("GetLimit(%d) with global=%d = %d, want %d", tc.handlerDefault, tc.globalLimit, got, tc.expected)
			}
		})
	}
}

func TestValidateOutputFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		format  string
		command string
		allowed []string
		wantErr bool
	}{
		{name: "get table valid", format: "table", command: "get", allowed: []string{"table", "wide", "json"}, wantErr: false},
		{name: "get wide valid", format: "wide", command: "get", allowed: []string{"table", "wide", "json"}, wantErr: false},
		{name: "get json valid", format: "json", command: "get", allowed: []string{"table", "wide", "json"}, wantErr: false},
		{name: "get text invalid", format: "text", command: "get", allowed: []string{"table", "wide", "json"}, wantErr: true},
		{name: "get yaml invalid", format: "yaml", command: "get", allowed: []string{"table", "wide", "json"}, wantErr: true},
		{name: "describe text valid", format: "text", command: "describe", allowed: []string{"text", "json"}, wantErr: false},
		{name: "describe json valid", format: "json", command: "describe", allowed: []string{"text", "json"}, wantErr: false},
		{name: "describe table invalid", format: "table", command: "describe", allowed: []string{"text", "json"}, wantErr: true},
		{name: "describe wide invalid", format: "wide", command: "describe", allowed: []string{"text", "json"}, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateOutputFormat(tc.format, tc.command, tc.allowed)
			if tc.wantErr && err == nil {
				t.Errorf("validateOutputFormat(%q, %q, %v) = nil, want error", tc.format, tc.command, tc.allowed)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateOutputFormat(%q, %q, %v) = %v, want nil", tc.format, tc.command, tc.allowed, err)
			}
		})
	}
}

func TestValidateOutputFormat_ErrorMessage(t *testing.T) {
	t.Parallel()

	err := validateOutputFormat("text", "get", []string{"table", "wide", "json"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	message := err.Error()
	if !strings.Contains(message, "text") {
		t.Errorf("error should mention the invalid format, got: %s", message)
	}
	if !strings.Contains(message, "get") {
		t.Errorf("error should mention the command name, got: %s", message)
	}
	if !strings.Contains(message, "table, wide, json") {
		t.Errorf("error should list supported formats, got: %s", message)
	}
}

func TestArgsAfterFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		filter    string
		arguments []string
		wantLen   int
	}{
		{name: "no filter", arguments: []string{"--limit", "5"}, filter: "", wantLen: 2},
		{name: "with filter", arguments: []string{"Liveness", "--limit", "5"}, filter: "Liveness", wantLen: 2},
		{name: "filter only", arguments: []string{"Liveness"}, filter: "Liveness", wantLen: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := argsAfterFilter(tc.arguments, tc.filter)
			if len(got) != tc.wantLen {
				t.Errorf("argsAfterFilter(%v, %q) has %d elements, want %d", tc.arguments, tc.filter, len(got), tc.wantLen)
			}
		})
	}
}

func TestPrintHealthTree(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status  *pb.HealthStatus
		name    string
		wantAll []string
		depth   int
	}{
		{
			name:    "single node",
			status:  &pb.HealthStatus{Name: "Liveness", State: "HEALTHY"},
			depth:   0,
			wantAll: []string{"Liveness", "HEALTHY"},
		},
		{
			name:    "with message",
			status:  &pb.HealthStatus{Name: "DB", State: "UNHEALTHY", Message: "timeout"},
			depth:   0,
			wantAll: []string{"DB", "UNHEALTHY", "(timeout)"},
		},
		{
			name:    "with duration",
			status:  &pb.HealthStatus{Name: "Cache", State: "HEALTHY", Duration: "0.5ms"},
			depth:   0,
			wantAll: []string{"Cache", "[0.5ms]"},
		},
		{
			name: "nested deps",
			status: &pb.HealthStatus{
				Name:  "System",
				State: "HEALTHY",
				Dependencies: []*pb.HealthStatus{
					{Name: "DB", State: "HEALTHY"},
					{Name: "Cache", State: "DEGRADED"},
				},
			},
			depth:   0,
			wantAll: []string{"System", "DB", "Cache", "DEGRADED"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)
			printHealthTree(&buffer, p, tc.status, tc.depth)
			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestStatusStyle(t *testing.T) {
	t.Parallel()

	t.Run("noColour returns unstyled", func(t *testing.T) {
		t.Parallel()
		p := NewPrinter(&bytes.Buffer{}, "table", true, false)
		style := p.statusStyle("HEALTHY")
		rendered := style.Render("HEALTHY")
		if rendered != "HEALTHY" {
			t.Errorf("noColour style rendered %q, want %q", rendered, "HEALTHY")
		}
	})

	t.Run("known status gets colour style", func(t *testing.T) {
		t.Parallel()
		p := NewPrinter(&bytes.Buffer{}, "table", false, false)
		style := p.statusStyle("HEALTHY")

		var noColour lipgloss.NoColor
		fg := style.GetForeground()
		if fg == noColour {
			t.Error("expected foreground colour for HEALTHY status")
		}
	})

	t.Run("unknown status gets default colour style", func(t *testing.T) {
		t.Parallel()
		p := NewPrinter(&bytes.Buffer{}, "table", false, false)
		style := p.statusStyle("UNKNOWN_STATUS")
		var noColour lipgloss.NoColor
		fg := style.GetForeground()
		if fg == noColour {
			t.Error("expected default foreground colour for unknown status")
		}
	})
}
