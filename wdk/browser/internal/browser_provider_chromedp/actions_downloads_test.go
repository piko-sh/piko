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
	"testing"

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
