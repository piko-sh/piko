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

package browser_provider_chromedp

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"piko.sh/piko/wdk/safedisk"
)

func TestNewDownloadTracker(t *testing.T) {
	dt := NewDownloadTracker("/tmp/downloads")

	if dt == nil {
		t.Fatal("NewDownloadTracker() returned nil")
	}
	if dt.downloadDir != "/tmp/downloads" {
		t.Errorf("downloadDir = %q, expected %q", dt.downloadDir, "/tmp/downloads")
	}
	if dt.enabled {
		t.Error("new tracker should not be enabled")
	}
	if dt.stopped {
		t.Error("new tracker should not be stopped")
	}
	if len(dt.downloads) != 0 {
		t.Errorf("expected empty downloads map, got %d entries", len(dt.downloads))
	}
}

func TestDownloadTracker_GetDownload(t *testing.T) {
	dt := NewDownloadTracker("")

	dt.downloads["guid-1"] = &DownloadInfo{
		GUID:       "guid-1",
		URL:        "https://example.com/file.pdf",
		State:      "completed",
		TotalBytes: 1024,
	}

	t.Run("existing download", func(t *testing.T) {
		d := dt.GetDownload("guid-1")
		if d == nil {
			t.Fatal("expected non-nil download")
		}
		if d.GUID != "guid-1" {
			t.Errorf("GUID = %q, expected %q", d.GUID, "guid-1")
		}
		if d.URL != "https://example.com/file.pdf" {
			t.Errorf("URL = %q, expected %q", d.URL, "https://example.com/file.pdf")
		}
	})

	t.Run("non-existent download", func(t *testing.T) {
		d := dt.GetDownload("no-such-guid")
		if d != nil {
			t.Error("expected nil for non-existent download")
		}
	})
}

func TestDownloadTracker_GetAllDownloads(t *testing.T) {
	dt := NewDownloadTracker("")

	t.Run("empty tracker", func(t *testing.T) {
		downloads := dt.GetAllDownloads()
		if len(downloads) != 0 {
			t.Errorf("expected 0 downloads, got %d", len(downloads))
		}
	})

	t.Run("with downloads", func(t *testing.T) {
		dt.downloads["a"] = &DownloadInfo{GUID: "a"}
		dt.downloads["b"] = &DownloadInfo{GUID: "b"}

		downloads := dt.GetAllDownloads()
		if len(downloads) != 2 {
			t.Errorf("expected 2 downloads, got %d", len(downloads))
		}
	})
}

func TestDownloadTracker_ClearDownloads(t *testing.T) {
	dt := NewDownloadTracker("")
	dt.downloads["a"] = &DownloadInfo{GUID: "a"}
	dt.downloads["b"] = &DownloadInfo{GUID: "b"}

	dt.ClearDownloads()

	if len(dt.downloads) != 0 {
		t.Errorf("expected 0 downloads after clear, got %d", len(dt.downloads))
	}

	downloads := dt.GetAllDownloads()
	if len(downloads) != 0 {
		t.Errorf("GetAllDownloads expected 0 after clear, got %d", len(downloads))
	}
}

func TestDownloadTracker_CreateDownloadDir_Sandbox(t *testing.T) {
	t.Parallel()

	t.Run("creates directory via sandbox", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/downloads", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		dt := NewDownloadTracker("/downloads", WithDownloadSandbox(sandbox))
		err := dt.createDownloadDir()

		if err != nil {
			t.Fatalf("createDownloadDir() returned error: %v", err)
		}
		if sandbox.CallCounts["MkdirAll"] < 1 {
			t.Error("expected MkdirAll to be called at least once")
		}
	})

	t.Run("MkdirAll error propagates", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/downloads", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.MkdirAllErr = errors.New("disk full")

		dt := NewDownloadTracker("/downloads", WithDownloadSandbox(sandbox))
		err := dt.createDownloadDir()

		if err == nil {
			t.Fatal("expected error from createDownloadDir()")
		}
	})

	t.Run("empty download directory is noop", func(t *testing.T) {
		t.Parallel()
		dt := NewDownloadTracker("")
		err := dt.createDownloadDir()

		if err != nil {
			t.Fatalf("expected no error for empty directory, got: %v", err)
		}
	})
}

func TestSanitiseSuggestedFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		wantSafe  bool
		wantBase  string
		wantMatch func(string) bool
	}{
		{
			name:     "plain filename passes through",
			input:    "report.pdf",
			wantSafe: true,
			wantBase: "report.pdf",
		},
		{
			name:      "parent traversal is rewritten",
			input:     "../../etc/passwd",
			wantSafe:  false,
			wantMatch: func(s string) bool { return strings.HasPrefix(s, "download-") && !strings.Contains(s, "..") },
		},
		{
			name:      "absolute path is rewritten",
			input:     "/etc/passwd",
			wantSafe:  false,
			wantMatch: func(s string) bool { return strings.HasPrefix(s, "download-") },
		},
		{
			name:      "windows separator is rewritten",
			input:     "..\\..\\windows\\system32\\cmd.exe",
			wantSafe:  false,
			wantMatch: func(s string) bool { return strings.HasPrefix(s, "download-") },
		},
		{
			name:      "leading dot is rewritten",
			input:     ".bashrc",
			wantSafe:  false,
			wantMatch: func(s string) bool { return strings.HasPrefix(s, "download-") },
		},
		{
			name:      "empty input is rewritten",
			input:     "",
			wantSafe:  false,
			wantMatch: func(s string) bool { return strings.HasPrefix(s, "download-") },
		},
		{
			name:      "dot only is rewritten",
			input:     ".",
			wantSafe:  false,
			wantMatch: func(s string) bool { return strings.HasPrefix(s, "download-") },
		},
		{
			name:      "double dot is rewritten",
			input:     "..",
			wantSafe:  false,
			wantMatch: func(s string) bool { return strings.HasPrefix(s, "download-") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := sanitiseSuggestedFilename(tc.input)

			if strings.ContainsAny(got, "/\\") {
				t.Fatalf("sanitised filename %q still contains separators", got)
			}

			if tc.wantSafe {
				if got != tc.wantBase {
					t.Errorf("got %q, want %q", got, tc.wantBase)
				}
				return
			}

			if !tc.wantMatch(got) {
				t.Errorf("sanitised filename %q does not match expected shape", got)
			}
		})
	}
}

func TestSafeDownloadPath_RejectsEscapingAbsolutePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path, err := safeDownloadPath(dir, "/etc/passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	abs, _ := filepath.Abs(path)
	dirAbs, _ := filepath.Abs(dir)
	if !strings.HasPrefix(abs, dirAbs) {
		t.Fatalf("path %q escaped %q", abs, dirAbs)
	}
	if strings.Contains(filepath.Base(abs), "passwd") {
		t.Fatalf("rewritten name %q still contains attacker-controlled component", filepath.Base(abs))
	}
}

func TestSafeDownloadPath_AllowsPlainFilename(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path, err := safeDownloadPath(dir, "report.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(path) != "report.pdf" {
		t.Fatalf("expected base report.pdf, got %q", filepath.Base(path))
	}
}

func TestDownloadTracker_WaitForDownload_SanitisesSuggestedFilename(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dt := NewDownloadTracker(dir)

	dt.downloadCh <- &DownloadInfo{
		GUID:              "guid-traverse",
		URL:               "https://example.com/x",
		SuggestedFilename: "../../etc/passwd",
		State:             "completed",
	}

	info, err := dt.WaitForDownload(time.Second)
	if err != nil {
		t.Fatalf("WaitForDownload returned error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil download info")
	}

	dirAbs, _ := filepath.Abs(dir)
	pathAbs, _ := filepath.Abs(info.Path)
	if !strings.HasPrefix(pathAbs, dirAbs) {
		t.Fatalf("download path %q escapes download dir %q", pathAbs, dirAbs)
	}
	if strings.Contains(info.Path, "..") {
		t.Fatalf("download path %q still contains traversal", info.Path)
	}
}

func TestDownloadTracker_WaitForDownload_AbsoluteFilename(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dt := NewDownloadTracker(dir)

	dt.downloadCh <- &DownloadInfo{
		GUID:              "guid-abs",
		URL:               "https://example.com/x",
		SuggestedFilename: "/etc/passwd",
		State:             "completed",
	}

	info, err := dt.WaitForDownload(time.Second)
	if err != nil {
		t.Fatalf("WaitForDownload returned error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil download info")
	}

	dirAbs, _ := filepath.Abs(dir)
	pathAbs, _ := filepath.Abs(info.Path)
	if !strings.HasPrefix(pathAbs, dirAbs) {
		t.Fatalf("download path %q escapes download dir %q", pathAbs, dirAbs)
	}
}

func TestDownloadTracker_WaitForDownload_EmptyFilenameSynthesises(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dt := NewDownloadTracker(dir)

	dt.downloadCh <- &DownloadInfo{
		GUID:              "guid-empty",
		URL:               "https://example.com/x",
		SuggestedFilename: " ",
		State:             "completed",
	}

	info, err := dt.WaitForDownload(time.Second)
	if err != nil {
		t.Fatalf("WaitForDownload returned error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil download info")
	}

	if !strings.HasPrefix(filepath.Base(info.Path), "download-") {
		t.Fatalf("expected synthesised filename, got %q", filepath.Base(info.Path))
	}
}

func TestPathHasPrefix_CaseSensitiveDefault(t *testing.T) {
	previous := pathComparisonIsCaseInsensitive
	pathComparisonIsCaseInsensitive = false
	defer func() { pathComparisonIsCaseInsensitive = previous }()

	if !pathHasPrefix("/Foo/bar/", "/Foo/") {
		t.Error("expected exact prefix match")
	}
	if pathHasPrefix("/foo/bar/", "/Foo/") {
		t.Error("case-sensitive comparison must reject mismatched case")
	}
	if pathHasPrefix("/Foo/", "/Foo/bar/") {
		t.Error("path shorter than prefix must not match")
	}
}

func TestPathHasPrefix_CaseInsensitiveWindows(t *testing.T) {
	previous := pathComparisonIsCaseInsensitive
	pathComparisonIsCaseInsensitive = true
	defer func() { pathComparisonIsCaseInsensitive = previous }()

	cases := []struct {
		name   string
		path   string
		prefix string
		want   bool
	}{
		{name: "exact case matches", path: `C:\Downloads\file.pdf\`, prefix: `C:\Downloads\`, want: true},
		{name: "lowercase path matches uppercase prefix", path: `c:\downloads\file.pdf\`, prefix: `C:\Downloads\`, want: true},
		{name: "mixed case still matches", path: `C:\DOWNLOADS\FILE.PDF\`, prefix: `c:\downloads\`, want: true},
		{name: "different directory rejected", path: `C:\Other\file.pdf\`, prefix: `C:\Downloads\`, want: false},
		{name: "path shorter than prefix rejected", path: `C:\Down\`, prefix: `C:\Downloads\`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pathHasPrefix(tc.path, tc.prefix)
			if got != tc.want {
				t.Errorf("pathHasPrefix(%q, %q) = %v, want %v", tc.path, tc.prefix, got, tc.want)
			}
		})
	}
}

func TestSafeDownloadPath_WindowsCaseInsensitive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("safeDownloadPath case-insensitive Windows path semantics require GOOS=windows; case-insensitive prefix logic is exercised by TestPathHasPrefix_CaseInsensitiveWindows")
	}

	previous := pathComparisonIsCaseInsensitive
	pathComparisonIsCaseInsensitive = true
	defer func() { pathComparisonIsCaseInsensitive = previous }()

	dir := t.TempDir()
	path, err := safeDownloadPath(dir, "report.pdf")
	if err != nil {
		t.Fatalf("safeDownloadPath returned unexpected error with case-insensitive comparison: %v", err)
	}
	if filepath.Base(path) != "report.pdf" {
		t.Errorf("expected base report.pdf, got %q", filepath.Base(path))
	}
}
