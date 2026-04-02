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
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func newTestResponse() *pb.GetSystemStatsResponse {
	return &pb.GetSystemStatsResponse{
		UptimeMs:             3_661_000,
		SystemUptimeMs:       86_400_000,
		CpuMillicores:        1250.5,
		NumCpu:               8,
		Gomaxprocs:           8,
		NumGoroutines:        42,
		NumCgoCalls:          17,
		CgroupPath:           "/sys/fs/cgroup/piko",
		MonitoringListenAddr: ":9090",
		TimestampMs:          1_700_000_000_000,
		Build: &pb.BuildInfo{
			Version:       "v1.2.3",
			Commit:        "abc123def456",
			GoVersion:     "go1.23.0",
			Os:            "linux",
			Arch:          "amd64",
			BuildTime:     "2026-01-15T10:00:00Z",
			ModulePath:    "piko.sh/piko",
			ModuleVersion: "v1.2.3",
			VcsModified:   false,
			VcsTime:       "2026-01-15T09:55:00Z",
		},
		Runtime: &pb.RuntimeInfo{
			Gogc:       "100",
			Gomemlimit: "1073741824",
			Compiler:   "gc",
		},
		Memory: &pb.MemoryInfo{
			Alloc:       10 * 1024 * 1024,
			TotalAlloc:  500 * 1024 * 1024,
			Sys:         50 * 1024 * 1024,
			HeapAlloc:   8 * 1024 * 1024,
			HeapSys:     32 * 1024 * 1024,
			HeapIdle:    20 * 1024 * 1024,
			HeapInuse:   12 * 1024 * 1024,
			HeapObjects: 50000,
			StackInuse:  2 * 1024 * 1024,
			StackSys:    4 * 1024 * 1024,
			Mallocs:     120000,
			Frees:       70000,
			LiveObjects: 50000,
		},
		Gc: &pb.GCInfo{
			NumGc:         150,
			NumForcedGc:   3,
			LastPauseNs:   250_000,
			PauseTotalNs:  5_000_000,
			GcCpuFraction: 0.0123,
			NextGc:        16 * 1024 * 1024,
			LastGcNs:      1_700_000_000_000_000_000,
			RecentPauses:  []uint64{200_000, 300_000},
		},
		Process: &pb.ProcessInfo{
			Pid:              12345,
			Ppid:             1,
			Uid:              1000,
			Gid:              1000,
			ThreadCount:      10,
			FdCount:          64,
			MaxOpenFilesSoft: 1024,
			MaxOpenFilesHard: 4096,
			Rss:              100 * 1024 * 1024,
			Hostname:         "piko-host-01",
			Executable:       "/usr/bin/piko",
			Cwd:              "/var/lib/piko",
		},
	}
}

func TestInfoSystem(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetSystemStatsResponse
		wantAll  []string
	}{
		{
			name:     "all system fields present",
			response: newTestResponse(),
			wantAll: []string{
				"Uptime",
				"CPU Millicores",
				"1250.5",
				"Num CPUs",
				"8",
				"Goroutines",
				"42",
				"CGO Calls",
				"17",
				"GOMAXPROCS",
				"Cgroup Path",
				"/sys/fs/cgroup/piko",
				"Monitoring Address",
				":9090",
			},
		},
		{
			name:     "zero values produce output",
			response: &pb.GetSystemStatsResponse{},
			wantAll: []string{
				"Uptime",
				"CPU Millicores",
				"Goroutines",
				"CGO Calls",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)

			infoSystem(tc.response, p)

			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestInfoBuild(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetSystemStatsResponse
		wantAll  []string
		wantNone bool
	}{
		{
			name:     "all build fields present",
			response: newTestResponse(),
			wantAll: []string{
				"Version",
				"v1.2.3",
				"Commit",
				"abc123def456",
				"Go Version",
				"go1.23.0",
				"OS",
				"linux",
				"Arch",
				"amd64",
				"Build Time",
				"2026-01-15T10:00:00Z",
				"Module",
				"piko.sh/piko",
				"VCS Modified",
				"false",
			},
		},
		{
			name:     "nil build produces no output",
			response: &pb.GetSystemStatsResponse{},
			wantNone: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)

			infoBuild(tc.response, p)

			output := buffer.String()

			if tc.wantNone {
				if output != "" {
					t.Errorf("expected no output for nil build, got:\n%s", output)
				}
				return
			}

			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestInfoRuntime(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetSystemStatsResponse
		wantAll  []string
		wantNone bool
	}{
		{
			name:     "all runtime fields present",
			response: newTestResponse(),
			wantAll: []string{
				"GOGC",
				"100",
				"GOMEMLIMIT",
				"1073741824",
				"Compiler",
				"gc",
			},
		},
		{
			name:     "nil runtime produces no output",
			response: &pb.GetSystemStatsResponse{},
			wantNone: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)

			infoRuntime(tc.response, p)

			output := buffer.String()

			if tc.wantNone {
				if output != "" {
					t.Errorf("expected no output for nil runtime, got:\n%s", output)
				}
				return
			}

			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestInfoMemory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetSystemStatsResponse
		wantAll  []string
		wantNone bool
	}{
		{
			name:     "all memory fields present",
			response: newTestResponse(),
			wantAll: []string{
				"Heap Alloc",
				"8.0 MiB",
				"Sys",
				"50.0 MiB",
				"Live Objects",
				"50000",
				"Heap Objects",
				"50000",
				"Heap In Use",
				"12.0 MiB",
				"Stack In Use",
				"2.0 MiB",
				"Total Alloc",
				"Mallocs",
				"120000",
				"Frees",
				"70000",
			},
		},
		{
			name:     "nil memory produces no output",
			response: &pb.GetSystemStatsResponse{},
			wantNone: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)

			infoMemory(tc.response, p)

			output := buffer.String()

			if tc.wantNone {
				if output != "" {
					t.Errorf("expected no output for nil memory, got:\n%s", output)
				}
				return
			}

			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestInfoGC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetSystemStatsResponse
		wantAll  []string
		wantNone bool
	}{
		{
			name:     "all GC fields present",
			response: newTestResponse(),
			wantAll: []string{
				"Cycles",
				"150",
				"Forced Cycles",
				"3",
				"CPU Fraction",
				"1.2300%",
				"Next GC",
				"16.0 MiB",
				"Recent Pauses",
			},
		},
		{
			name:     "nil GC produces no output",
			response: &pb.GetSystemStatsResponse{},
			wantNone: true,
		},
		{
			name: "GC without recent pauses omits pauses row",
			response: &pb.GetSystemStatsResponse{
				Gc: &pb.GCInfo{
					NumGc:         5,
					GcCpuFraction: 0.001,
					NextGc:        4 * 1024 * 1024,
				},
			},
			wantAll: []string{
				"Cycles",
				"5",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)

			infoGC(tc.response, p)

			output := buffer.String()

			if tc.wantNone {
				if output != "" {
					t.Errorf("expected no output for nil GC, got:\n%s", output)
				}
				return
			}

			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestInfoProcess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetSystemStatsResponse
		wantAll  []string
		wantNone bool
	}{
		{
			name:     "all process fields present",
			response: newTestResponse(),
			wantAll: []string{
				"PID",
				"12345",
				"PPID",
				"1",
				"Threads",
				"10",
				"File Descriptors",
				"64",
				"RSS",
				"Hostname",
				"piko-host-01",
				"Executable",
				"/usr/bin/piko",
				"CWD",
				"/var/lib/piko",
				"Max Open Files (Soft)",
				"1024",
				"Max Open Files (Hard)",
				"4096",
			},
		},
		{
			name:     "nil process produces no output",
			response: &pb.GetSystemStatsResponse{},
			wantNone: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)

			infoProcess(tc.response, p)

			output := buffer.String()

			if tc.wantNone {
				if output != "" {
					t.Errorf("expected no output for nil process, got:\n%s", output)
				}
				return
			}

			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestInfoOverview(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *pb.GetSystemStatsResponse
		wantAll  []string
	}{
		{
			name:     "all category headers present",
			response: newTestResponse(),
			wantAll: []string{
				"=== System ===",
				"=== Build ===",
				"=== Runtime ===",
				"=== Memory ===",
				"=== GC ===",
				"=== Process ===",
				"Available categories:",
			},
		},
		{
			name:     "nil sub-messages still show headers",
			response: &pb.GetSystemStatsResponse{},
			wantAll: []string{
				"=== System ===",
				"=== Build ===",
				"=== Runtime ===",
				"=== Memory ===",
				"=== GC ===",
				"=== Process ===",
			},
		},
		{
			name:     "overview contains key data values",
			response: newTestResponse(),
			wantAll: []string{
				"v1.2.3",
				"abc123de",
				"go1.23.0",
				"42",
				"12345",
				"piko-host-01",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, "table", true, false)

			infoOverview(tc.response, p, &buffer)

			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestRunInfo(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetSystemStatsFunc: func(_ context.Context, _ *pb.GetSystemStatsRequest, _ ...grpc.CallOption) (*pb.GetSystemStatsResponse, error) {
				return newTestResponse(), nil
			},
		},
	}

	testCases := []struct {
		name      string
		output    string
		arguments []string
		wantAll   []string
		wantErr   bool
	}{
		{
			name:   "overview no arguments",
			output: "table",
			wantAll: []string{
				"=== System ===",
				"=== Build ===",
				"=== Runtime ===",
				"=== Memory ===",
				"=== GC ===",
				"=== Process ===",
			},
		},
		{
			name:      "system category",
			output:    "table",
			arguments: []string{"system"},
			wantAll: []string{
				"Uptime",
				"CPU Millicores",
				"1250.5",
				"Goroutines",
				"42",
			},
		},
		{
			name:      "build category",
			output:    "table",
			arguments: []string{"build"},
			wantAll: []string{
				"Version",
				"v1.2.3",
				"Commit",
				"abc123def456",
			},
		},
		{
			name:   "json output",
			output: "json",
			wantAll: []string{
				`"uptime_ms"`,
			},
		},
		{
			name:      "unknown category",
			output:    "table",
			arguments: []string{"nonexistent"},
			wantErr:   true,
		},
		{
			name:    "unsupported output format",
			output:  "yaml",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cc, stdout, _ := newTestCC(conn)
			cc.Opts.Output = tc.output

			err := runInfo(context.Background(), cc, tc.arguments)

			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := stdout.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func TestRunInfoError(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		metrics: &mockMetricsClient{
			GetSystemStatsFunc: func(_ context.Context, _ *pb.GetSystemStatsRequest, _ ...grpc.CallOption) (*pb.GetSystemStatsResponse, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	cc, _, _ := newTestCC(conn)
	err := runInfo(context.Background(), cc, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
