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

package inspector

import (
	"testing"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestShortCommitHash(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "shorter than limit", in: "abc", want: "abc"},
		{name: "exact limit", in: "abcdefgh", want: "abcdefgh"},
		{name: "longer than limit", in: "abcdef0123456789", want: "abcdef01"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ShortCommitHash(tc.in)
			if got != tc.want {
				t.Errorf("ShortCommitHash(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatGCCPUFraction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want string
		in   float64
	}{
		{name: "zero", in: 0, want: "0.0000%"},
		{name: "tiny", in: 0.0001, want: "0.0100%"},
		{name: "one percent", in: 0.01, want: "1.0000%"},
		{name: "rounded", in: 0.012345, want: "1.2345%"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := FormatGCCPUFraction(tc.in)
			if got != tc.want {
				t.Errorf("FormatGCCPUFraction(%g) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatNanosAsDuration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want string
		in   int64
	}{
		{name: "negative returns em-dash", in: -1, want: EmDashGlyph},
		{name: "zero returns em-dash", in: 0, want: EmDashGlyph},
		{name: "one second", in: 1_000_000_000, want: "1s"},
		{name: "two minutes", in: 120_000_000_000, want: "2m0s"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := FormatNanosAsDuration(tc.in)
			if got != tc.want {
				t.Errorf("FormatNanosAsDuration(%d) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildBuildDetailRows(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		got := BuildBuildDetailRows(nil)
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("populated", func(t *testing.T) {
		t.Parallel()
		build := &pb.BuildInfo{
			Version:       "v1.2.3",
			Commit:        "abcdef0123456789",
			GoVersion:     "go1.26.0",
			Os:            "linux",
			Arch:          "amd64",
			BuildTime:     "2026-04-01T00:00:00Z",
			ModulePath:    "piko.sh/piko",
			ModuleVersion: "v0.0.0",
			VcsModified:   true,
			VcsTime:       "2026-04-01T00:00:00Z",
		}
		rows := BuildBuildDetailRows(build)
		wantOrder := []string{
			"Version",
			"Commit",
			"Go Version",
			"OS",
			"Arch",
			"Build Time",
			"Module",
			"Module Version",
			"VCS Modified",
			"VCS Time",
		}
		if len(rows) != len(wantOrder) {
			t.Fatalf("rows = %d, want %d", len(rows), len(wantOrder))
		}
		for index, label := range wantOrder {
			if rows[index].Label != label {
				t.Errorf("row %d label = %q, want %q", index, rows[index].Label, label)
			}
		}
	})
}

func TestBuildRuntimeDetailRows(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if got := BuildRuntimeDetailRows(nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("populated", func(t *testing.T) {
		t.Parallel()
		runtime := &pb.RuntimeInfo{
			Gogc:       "100",
			Gomemlimit: "8GiB",
			Compiler:   "gc",
		}
		rows := BuildRuntimeDetailRows(runtime)
		if len(rows) != 3 {
			t.Fatalf("rows = %d, want 3", len(rows))
		}
		if rows[0].Label != "GOGC" || rows[0].Value != "100" {
			t.Errorf("rows[0] = %+v", rows[0])
		}
		if rows[2].Label != "Compiler" || rows[2].Value != "gc" {
			t.Errorf("rows[2] = %+v", rows[2])
		}
	})
}

func TestBuildMemoryDetailRows(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if got := BuildMemoryDetailRows(nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("populated row count", func(t *testing.T) {
		t.Parallel()
		memory := &pb.MemoryInfo{Alloc: 1024, HeapAlloc: 2048, Mallocs: 100, Frees: 50}
		rows := BuildMemoryDetailRows(memory)

		if len(rows) != 22 {
			t.Errorf("rows = %d, want 22", len(rows))
		}
		if rows[0].Label != "Alloc" {
			t.Errorf("first row = %q, want Alloc", rows[0].Label)
		}
	})
}

func TestBuildGCDetailRows(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if got := BuildGCDetailRows(nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("populated without recent pauses", func(t *testing.T) {
		t.Parallel()
		gc := &pb.GCInfo{
			NumGc:         5,
			NumForcedGc:   1,
			LastPauseNs:   1_000_000,
			PauseTotalNs:  5_000_000,
			GcCpuFraction: 0.005,
			NextGc:        4096,
			LastGcNs:      1_700_000_000_000_000_000,
		}
		rows := BuildGCDetailRows(gc)

		if len(rows) != 7 {
			t.Errorf("rows = %d, want 7", len(rows))
		}
		var hasRecentPauses bool
		for _, row := range rows {
			if row.Label == "Recent Pauses" {
				hasRecentPauses = true
			}
		}
		if hasRecentPauses {
			t.Errorf("Recent Pauses row should be absent when GCInfo has none")
		}
	})

	t.Run("populated with recent pauses", func(t *testing.T) {
		t.Parallel()
		gc := &pb.GCInfo{
			NumGc:        5,
			RecentPauses: []uint64{1_000_000, 2_000_000, 3_000_000},
		}
		rows := BuildGCDetailRows(gc)
		var recentRow DetailRow
		var found bool
		for _, row := range rows {
			if row.Label == "Recent Pauses" {
				recentRow = row
				found = true
			}
		}
		if !found {
			t.Fatalf("Recent Pauses row missing")
		}
		if recentRow.Value == "" {
			t.Errorf("Recent Pauses value empty")
		}
	})
}

func TestBuildProcessDetailRows(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if got := BuildProcessDetailRows(nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("populated", func(t *testing.T) {
		t.Parallel()
		process := &pb.ProcessInfo{
			Pid:              12345,
			Ppid:             1,
			Uid:              1000,
			Gid:              1000,
			ThreadCount:      8,
			FdCount:          24,
			MaxOpenFilesSoft: 4096,
			MaxOpenFilesHard: 8192,
			Rss:              1024 * 1024,
			Hostname:         "host-x",
			Executable:       "/usr/bin/piko",
			Cwd:              "/var/lib/piko",
		}
		rows := BuildProcessDetailRows(process)

		if len(rows) != 16 {
			t.Errorf("rows = %d, want 16", len(rows))
		}
		rowsByLabel := indexRows(rows)
		if rowsByLabel["PID"].Value != "12345" {
			t.Errorf("PID = %q", rowsByLabel["PID"].Value)
		}
		if rowsByLabel["Hostname"].Value != "host-x" {
			t.Errorf("Hostname = %q", rowsByLabel["Hostname"].Value)
		}
		if rowsByLabel["CWD"].Value != "/var/lib/piko" {
			t.Errorf("CWD = %q", rowsByLabel["CWD"].Value)
		}
	})
}
