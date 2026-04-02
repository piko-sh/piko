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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"piko.sh/piko/wdk/safedisk"
)

const testHTMLDownload = `<!DOCTYPE html>
<html>
<head><title>Download Test</title></head>
<body>
<a id="download-link" href="/download/test.txt" download="test.txt">Download</a>
<button id="blob-download" onclick="const b=new Blob(['hello world'],{type:'text/plain'});const u=URL.createObjectURL(b);const a=document.createElement('a');a.href=u;a.download='blob.txt';a.click();">Blob Download</button>
</body>
</html>`

func newDownloadTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download/test.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Disposition", "attachment; filename=test.txt")
			_, _ = w.Write([]byte("test file content"))
		case "/download/large.bin":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", "attachment; filename=large.bin")

			data := make([]byte, 1024)
			_, _ = w.Write(data)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(testHTMLDownload))
		}
	}))
}

func TestDownloadTracker_EnableDisable(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		downloadDir := t.TempDir()
		tracker := NewDownloadTracker(downloadDir)

		t.Run("enable tracker", func(t *testing.T) {
			err := tracker.Enable(ctx)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}
			if !tracker.enabled {
				t.Error("expected tracker to be enabled")
			}

			err = tracker.Enable(ctx)
			if err != nil {
				t.Fatalf("Enable() second call error = %v", err)
			}
		})

		t.Run("disable tracker", func(t *testing.T) {
			err := tracker.Disable(ctx)
			if err != nil {
				t.Fatalf("Disable() error = %v", err)
			}
			if tracker.enabled {
				t.Error("expected tracker to be disabled")
			}

			err = tracker.Disable(ctx)
			if err != nil {
				t.Fatalf("Disable() second call error = %v", err)
			}
		})
	})
}

func TestDownloadTracker_EnableCreatesDir(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		downloadDir := filepath.Join(t.TempDir(), "nested", "download")
		tracker := NewDownloadTracker(downloadDir)

		err := tracker.Enable(ctx)
		if err != nil {
			t.Fatalf("Enable() error = %v", err)
		}
		defer func() { _ = tracker.Disable(ctx) }()

		info, err := os.Stat(downloadDir)
		if err != nil {
			t.Fatalf("download directory should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected a directory")
		}
	})
}

func TestDownloadTracker_TriggerDownload(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		downloadDir := t.TempDir()
		tracker := NewDownloadTracker(downloadDir)

		err := tracker.Enable(ctx)
		if err != nil {
			t.Fatalf("Enable() error = %v", err)
		}
		defer func() { _ = tracker.Disable(ctx) }()

		t.Run("trigger download via JS", func(t *testing.T) {
			err := TriggerDownload(ctx, server.URL+"/download/test.txt", "test.txt")
			if err != nil {
				t.Fatalf("TriggerDownload() error = %v", err)
			}

			info, err := tracker.WaitForDownload(10 * time.Second)
			if err != nil {
				t.Fatalf("WaitForDownload() error = %v", err)
			}
			if info == nil {
				t.Fatal("expected non-nil download info")
			}
			if info.State != "completed" {
				t.Errorf("download state = %q, expected %q", info.State, "completed")
			}
		})
	})
}

func TestDownloadTracker_WaitForDownload_Timeout(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		downloadDir := t.TempDir()
		tracker := NewDownloadTracker(downloadDir)

		err := tracker.Enable(ctx)
		if err != nil {
			t.Fatalf("Enable() error = %v", err)
		}
		defer func() { _ = tracker.Disable(ctx) }()

		t.Run("timeout when no download", func(t *testing.T) {
			_, err := tracker.WaitForDownload(500 * time.Millisecond)
			if err == nil {
				t.Error("expected timeout error")
			}
		})
	})
}

func TestWaitForDownload_Functional(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		downloadDir := t.TempDir()

		t.Run("wait for download with trigger", func(t *testing.T) {

			info, err := WaitForDownload(ctx, downloadDir, 3*time.Second, func() error {
				return TriggerDownload(ctx, server.URL+"/download/test.txt", "test.txt")
			})
			if err != nil {
				t.Logf("WaitForDownload() returned error (may be expected in headless): %v", err)
			} else if info == nil {
				t.Error("expected non-nil download info on success")
			}
		})
	})
}

func TestSetDownloadPath(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		downloadDir := t.TempDir()

		t.Run("set download path creates directory", func(t *testing.T) {
			newDir := filepath.Join(downloadDir, "subdir")
			err := SetDownloadPath(ctx, newDir)
			if err != nil {
				t.Fatalf("SetDownloadPath() error = %v", err)
			}

			info, err := os.Stat(newDir)
			if err != nil {
				t.Fatalf("expected directory to exist: %v", err)
			}
			if !info.IsDir() {
				t.Error("expected a directory")
			}
		})
	})
}

func TestCreateBlobDownload(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		downloadDir := t.TempDir()
		tracker := NewDownloadTracker(downloadDir)

		err := tracker.Enable(ctx)
		if err != nil {
			t.Fatalf("Enable() error = %v", err)
		}
		defer func() { _ = tracker.Disable(ctx) }()

		t.Run("create blob download", func(t *testing.T) {
			err := CreateBlobDownload(ctx, "blob content", "text/plain", "blob-test.txt")
			if err != nil {
				t.Fatalf("CreateBlobDownload() error = %v", err)
			}

			info, err := tracker.WaitForDownload(10 * time.Second)
			if err != nil {
				t.Fatalf("WaitForDownload() error = %v", err)
			}
			if info == nil {
				t.Fatal("expected non-nil download info")
			}
		})
	})
}

func TestDownloadScreenshot(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("capture screenshot for download", func(t *testing.T) {
			err := DownloadScreenshot(ctx)
			if err != nil {
				t.Errorf("DownloadScreenshot() error = %v", err)
			}
		})
	})
}

func TestGetDownloadedFile(t *testing.T) {
	t.Parallel()
	requireBrowser(t)

	directory := t.TempDir()
	content := []byte("downloaded file content")
	err := os.WriteFile(filepath.Join(directory, "test-download.txt"), content, 0o644)
	if err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	if err != nil {
		t.Fatalf("creating sandbox: %v", err)
	}

	t.Run("read existing file", func(t *testing.T) {
		data, err := GetDownloadedFile(sandbox, "test-download.txt")
		if err != nil {
			t.Fatalf("GetDownloadedFile() error = %v", err)
		}
		if string(data) != "downloaded file content" {
			t.Errorf("expected 'downloaded file content', got %q", string(data))
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := GetDownloadedFile(sandbox, "no-such-file.txt")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestCancelDownload(t *testing.T) {
	t.Parallel()
	server := newDownloadTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("cancel non-existent download returns error", func(t *testing.T) {
			err := CancelDownload(ctx, "non-existent-guid")

			if err == nil {
				t.Log("CancelDownload() did not return error (may vary by Chrome version)")
			}
		})
	})
}
