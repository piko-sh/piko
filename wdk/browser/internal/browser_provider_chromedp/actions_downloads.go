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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// downloadDirPerm is the permission mode for created download directories.
	downloadDirPerm = 0750

	// downloadChannelBufferSize is the buffer size for the download notification
	// channel.
	downloadChannelBufferSize = 10

	// errFmtCreateDownloadDir is the error format for download directory
	// creation failures.
	errFmtCreateDownloadDir = "creating download directory: %w"
)

// DownloadInfo holds details about a file download, including its progress
// and state.
type DownloadInfo struct {
	// GUID is the unique identifier for this download.
	GUID string

	// URL is the download location for this resource.
	URL string

	// SuggestedFilename is the filename from the Content-Disposition header.
	SuggestedFilename string

	// Path is the full file path where the download is saved.
	Path string

	// State indicates the download status: "inProgress", "completed", or "canceled".
	State string

	// ReceivedBytes is the number of bytes downloaded so far.
	ReceivedBytes int64

	// TotalBytes is the total size of the download in bytes; -1 if unknown.
	TotalBytes int64
}

// DownloadTracker monitors file downloads and manages their progress.
type DownloadTracker struct {
	// downloads maps download GUIDs to their tracking information.
	downloads map[string]*DownloadInfo

	// downloadCh receives download info when a download completes.
	downloadCh chan *DownloadInfo

	// stopChan signals the event listener to stop processing.
	stopChan chan struct{}

	// sandbox provides sandboxed filesystem access for download directory operations.
	sandbox safedisk.Sandbox

	// sandboxFactory creates sandboxes for filesystem access. When nil,
	// safedisk.NewNoOpSandbox is used as a fallback.
	sandboxFactory safedisk.Factory

	// downloadDir is the directory path for saving downloaded files; empty
	// means the path is not modified.
	downloadDir string

	// mu          sync.RWMutex // mu guards access to the tracker state.
	mu sync.RWMutex

	// enabled indicates whether download tracking is active.
	enabled bool

	// stopped tracks whether stopChan has been closed.
	stopped bool
}

// DownloadTrackerOption configures a DownloadTracker.
type DownloadTrackerOption func(*DownloadTracker)

// NewDownloadTracker creates a new download tracker.
//
// Takes downloadDir (string) which specifies the directory for downloads.
// Takes opts (...DownloadTrackerOption) which provides optional
// configuration for the tracker.
//
// Returns *DownloadTracker which is ready for use but not yet enabled.
func NewDownloadTracker(downloadDir string, opts ...DownloadTrackerOption) *DownloadTracker {
	dt := &DownloadTracker{
		downloads:   make(map[string]*DownloadInfo),
		downloadCh:  make(chan *DownloadInfo, downloadChannelBufferSize),
		stopChan:    make(chan struct{}),
		downloadDir: downloadDir,
		mu:          sync.RWMutex{},
		enabled:     false,
		stopped:     false,
	}

	for _, opt := range opts {
		opt(dt)
	}

	if dt.sandbox == nil && downloadDir != "" {
		var sandbox safedisk.Sandbox
		var err error
		if dt.sandboxFactory != nil {
			sandbox, err = dt.sandboxFactory.Create("browser-download-dir", downloadDir, safedisk.ModeReadWrite)
		} else {
			sandbox, err = safedisk.NewNoOpSandbox(downloadDir, safedisk.ModeReadWrite)
		}
		if err == nil {
			dt.sandbox = sandbox
		}
	}

	return dt
}

// Enable enables download tracking and sets the download directory.
//
// Takes ctx (*ActionContext) which provides the browser context for setting
// download behaviour and event listeners.
//
// Returns error when creating the download directory fails or setting the
// download behaviour fails.
//
// Safe for concurrent use; protected by a mutex.
func (dt *DownloadTracker) Enable(ctx *ActionContext) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.enabled {
		return nil
	}

	if err := dt.createDownloadDir(); err != nil {
		return err
	}

	if err := dt.setDownloadBehavior(ctx); err != nil {
		return err
	}

	dt.setupEventListener(ctx)
	dt.enabled = true
	return nil
}

// Disable disables download tracking.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// operation.
//
// Returns error when the download behaviour cannot be reset.
//
// Safe for concurrent use. Signals any active listener goroutine to stop.
func (dt *DownloadTracker) Disable(ctx *ActionContext) error {
	dt.mu.Lock()

	if !dt.enabled {
		dt.mu.Unlock()
		return nil
	}

	if !dt.stopped {
		close(dt.stopChan)
		dt.stopped = true
	}
	dt.enabled = false
	dt.mu.Unlock()

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorDefault).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("disabling download behaviour: %w", err)
	}

	return nil
}

// WaitForDownload waits for a download to complete within the timeout.
//
// Takes timeout (time.Duration) which specifies the maximum time to wait
// for a download to complete.
//
// Returns *DownloadInfo which contains details about the completed download.
// Returns error when the timeout is reached before a download completes.
func (dt *DownloadTracker) WaitForDownload(timeout time.Duration) (*DownloadInfo, error) {
	select {
	case info := <-dt.downloadCh:
		if dt.downloadDir != "" && info.SuggestedFilename != "" {
			info.Path = filepath.Join(dt.downloadDir, info.SuggestedFilename)
		}
		return info, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout waiting for download")
	}
}

// GetDownload returns information about a specific download by GUID.
//
// Takes guid (string) which identifies the download to retrieve.
//
// Returns *DownloadInfo which contains the download details, or nil if not
// found.
//
// Safe for concurrent use.
func (dt *DownloadTracker) GetDownload(guid string) *DownloadInfo {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.downloads[guid]
}

// GetAllDownloads returns all tracked downloads.
//
// Returns []*DownloadInfo which contains a copy of all current downloads.
//
// Safe for concurrent use.
func (dt *DownloadTracker) GetAllDownloads() []*DownloadInfo {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	downloads := make([]*DownloadInfo, 0, len(dt.downloads))
	for _, d := range dt.downloads {
		downloads = append(downloads, d)
	}
	return downloads
}

// ClearDownloads clears the download history.
//
// Safe for concurrent use.
func (dt *DownloadTracker) ClearDownloads() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.downloads = make(map[string]*DownloadInfo)
}

// createDownloadDir creates the download directory if it does not exist.
//
// Returns error when the directory cannot be created.
func (dt *DownloadTracker) createDownloadDir() error {
	if dt.downloadDir == "" {
		return nil
	}
	if dt.sandbox != nil {
		if err := dt.sandbox.MkdirAll(".", downloadDirPerm); err != nil {
			return fmt.Errorf(errFmtCreateDownloadDir, err)
		}
		return nil
	}
	if err := os.MkdirAll(dt.downloadDir, downloadDirPerm); err != nil {
		return fmt.Errorf(errFmtCreateDownloadDir, err)
	}
	return nil
}

// setDownloadBehavior configures the browser's download behaviour.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// operation.
//
// Returns error when the download behaviour cannot be set.
func (dt *DownloadTracker) setDownloadBehavior(ctx *ActionContext) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(dt.downloadDir).
			WithEventsEnabled(true).
			Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting download behaviour: %w", err)
	}
	return nil
}

// setupEventListener sets up the download event listener.
//
// Takes ctx (*ActionContext) which provides the browser context for listening.
//
// Spawns a callback that runs on each download event until the stop channel
// is closed.
func (dt *DownloadTracker) setupEventListener(ctx *ActionContext) {
	chromedp.ListenTarget(ctx.Ctx, func(ev any) {
		select {
		case <-dt.stopChan:
			return
		default:
		}

		dt.mu.RLock()
		enabled := dt.enabled
		dt.mu.RUnlock()

		if !enabled {
			return
		}

		dt.handleDownloadEvent(ev)
	})
}

// handleDownloadEvent processes download-related browser events.
//
// Takes ev (any) which is the browser event to process.
func (dt *DownloadTracker) handleDownloadEvent(ev any) {
	switch e := ev.(type) {
	case *browser.EventDownloadWillBegin:
		dt.handleDownloadWillBegin(e)
	case *browser.EventDownloadProgress:
		dt.handleDownloadProgress(e)
	}
}

// handleDownloadWillBegin handles the start of a download.
//
// Takes e (*browser.EventDownloadWillBegin) which contains the download event
// details including GUID, URL, and suggested filename.
//
// Safe for concurrent use. Uses mutex to protect access to the downloads map.
func (dt *DownloadTracker) handleDownloadWillBegin(e *browser.EventDownloadWillBegin) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.downloads[e.GUID] = &DownloadInfo{
		GUID:              e.GUID,
		URL:               e.URL,
		SuggestedFilename: e.SuggestedFilename,
		Path:              "",
		State:             "inProgress",
		ReceivedBytes:     0,
		TotalBytes:        0,
	}
}

// handleDownloadProgress handles download progress updates.
//
// Takes e (*browser.EventDownloadProgress) which contains the progress data.
//
// Safe for concurrent use; protected by mutex.
func (dt *DownloadTracker) handleDownloadProgress(e *browser.EventDownloadProgress) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	info, ok := dt.downloads[e.GUID]
	if !ok {
		return
	}

	info.ReceivedBytes = int64(e.ReceivedBytes)
	info.TotalBytes = int64(e.TotalBytes)
	info.State = string(e.State)

	if e.State == browser.DownloadProgressStateCompleted && !dt.stopped {
		select {
		case dt.downloadCh <- info:
		default:
		}
	}
}

// WithDownloadSandbox injects a sandbox for filesystem operations.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed filesystem
// access for download operations.
//
// Returns DownloadTrackerOption which configures the tracker with the
// given sandbox.
func WithDownloadSandbox(sandbox safedisk.Sandbox) DownloadTrackerOption {
	return func(dt *DownloadTracker) {
		dt.sandbox = sandbox
	}
}

// WithDownloadSandboxFactory sets a factory for creating sandboxes in the
// download tracker instead of falling back to safedisk.NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
//
// Returns DownloadTrackerOption which configures the tracker with the
// given factory.
func WithDownloadSandboxFactory(factory safedisk.Factory) DownloadTrackerOption {
	return func(dt *DownloadTracker) {
		dt.sandboxFactory = factory
	}
}

// GetDownloadedFile reads the contents of a downloaded file using a sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides access to the download
// directory.
// Takes filename (string) which is the name of the downloaded file, not a
// full path.
//
// Returns []byte which contains the file contents.
// Returns error when the file cannot be read.
func GetDownloadedFile(sandbox safedisk.Sandbox, filename string) ([]byte, error) {
	data, err := sandbox.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading downloaded file: %w", err)
	}
	return data, nil
}

// WaitForDownload waits for a download after triggering an action.
//
// It sets up download handling, executes the trigger function, and waits for
// the download to complete.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes downloadDir (string) which specifies the directory for downloaded files.
// Takes timeout (time.Duration) which sets the maximum time to wait.
// Takes trigger (func(...)) which is the function that initiates the download.
//
// Returns *DownloadInfo which contains details about the completed download.
// Returns error when directory creation fails, download setup fails, the
// trigger function returns an error, or the download times out.
func WaitForDownload(ctx *ActionContext, downloadDir string, timeout time.Duration, trigger func() error) (*DownloadInfo, error) {
	var sandbox safedisk.Sandbox
	var sErr error
	if ctx.SandboxFactory != nil {
		sandbox, sErr = ctx.SandboxFactory.Create("browser-download-verify", downloadDir, safedisk.ModeReadWrite)
	} else {
		sandbox, sErr = safedisk.NewNoOpSandbox(downloadDir, safedisk.ModeReadWrite)
	}
	if sErr != nil {
		return nil, fmt.Errorf("creating download sandbox: %w", sErr)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.MkdirAll(".", downloadDirPerm); err != nil {
		return nil, fmt.Errorf(errFmtCreateDownloadDir, err)
	}

	if err := setDownloadBehaviorForDir(ctx, downloadDir); err != nil {
		return nil, err
	}

	downloadCh := make(chan *DownloadInfo, 1)
	setupDownloadListener(ctx, downloadDir, downloadCh)

	if err := trigger(); err != nil {
		return nil, fmt.Errorf("triggering download: %w", err)
	}

	return waitForDownloadCompletion(downloadCh, timeout)
}

// CancelDownload cancels an in-progress download.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes guid (string) which identifies the download to cancel.
//
// Returns error when the download cannot be cancelled.
func CancelDownload(ctx *ActionContext, guid string) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return browser.CancelDownload(guid).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("cancelling download: %w", err)
	}
	return nil
}

// SetDownloadPath sets the download directory for future downloads.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes path (string) which specifies the directory path for downloads.
//
// Returns error when the directory cannot be created or the browser fails to
// accept the download path.
func SetDownloadPath(ctx *ActionContext, path string) error {
	var sandbox safedisk.Sandbox
	var sErr error
	if ctx.SandboxFactory != nil {
		sandbox, sErr = ctx.SandboxFactory.Create("browser-download-path", path, safedisk.ModeReadWrite)
	} else {
		sandbox, sErr = safedisk.NewNoOpSandbox(path, safedisk.ModeReadWrite)
	}
	if sErr != nil {
		return fmt.Errorf("creating download sandbox: %w", sErr)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.MkdirAll(".", downloadDirPerm); err != nil {
		return fmt.Errorf(errFmtCreateDownloadDir, err)
	}

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(path).
			WithEventsEnabled(true).
			Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting download path: %w", err)
	}
	return nil
}

// TriggerDownload triggers a file download via JavaScript.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes url (string) which specifies the URL of the file to download.
// Takes filename (string) which sets the name for the downloaded file.
//
// Returns error when the JavaScript execution fails.
func TriggerDownload(ctx *ActionContext, url, filename string) error {
	js := scripts.MustExecute("trigger_download.js.tmpl", map[string]any{
		"URL":      url,
		"Filename": filename,
	})

	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
}

// CreateBlobDownload creates and triggers a download from blob data.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes content (string) which contains the blob data to download.
// Takes mimeType (string) which specifies the MIME type of the content.
// Takes filename (string) which sets the name for the downloaded file.
//
// Returns error when the browser script execution fails.
func CreateBlobDownload(ctx *ActionContext, content, mimeType, filename string) error {
	js := scripts.MustExecute("create_blob_download.js.tmpl", map[string]any{
		"Content":  content,
		"MimeType": mimeType,
		"Filename": filename,
	})

	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
}

// DownloadScreenshot saves a screenshot as a download.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// screenshot capture.
//
// Returns error when the screenshot cannot be captured.
func DownloadScreenshot(ctx *ActionContext) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		_, err := page.CaptureScreenshot().Do(ctx2)
		return err
	}))
	if err != nil {
		return fmt.Errorf("capturing screenshot for download: %w", err)
	}
	return nil
}

// setDownloadBehaviorForDir configures download behaviour for a specific
// directory.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes downloadDir (string) which specifies the target download directory.
//
// Returns error when setting the download behaviour fails.
func setDownloadBehaviorForDir(ctx *ActionContext, downloadDir string) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(downloadDir).
			WithEventsEnabled(true).
			Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting download behaviour: %w", err)
	}
	return nil
}

// setupDownloadListener sets up a listener for download events.
//
// Takes ctx (*ActionContext) which provides the browser context to listen on.
// Takes downloadDir (string) which specifies the directory for downloaded files.
// Takes downloadCh (chan *DownloadInfo) which receives download information when
// a download begins.
func setupDownloadListener(ctx *ActionContext, downloadDir string, downloadCh chan *DownloadInfo) {
	chromedp.ListenTarget(ctx.Ctx, func(ev any) {
		if e, ok := ev.(*browser.EventDownloadWillBegin); ok {
			info := &DownloadInfo{
				GUID:              e.GUID,
				URL:               e.URL,
				SuggestedFilename: e.SuggestedFilename,
				Path:              filepath.Join(downloadDir, e.SuggestedFilename),
				State:             "inProgress",
				ReceivedBytes:     0,
				TotalBytes:        0,
			}
			select {
			case downloadCh <- info:
			default:
			}
		}
	})
}

// waitForDownloadCompletion waits for a download to complete and verifies
// the file exists.
//
// Takes downloadCh (chan *DownloadInfo) which receives download information.
// Takes timeout (time.Duration) which specifies the maximum wait time.
//
// Returns *DownloadInfo which contains details of the completed download.
// Returns error when the download times out or the file does not exist.
func waitForDownloadCompletion(downloadCh chan *DownloadInfo, timeout time.Duration) (*DownloadInfo, error) {
	select {
	case info := <-downloadCh:
		return waitForFileExists(info, timeout)
	case <-time.After(timeout):
		return nil, errors.New("timeout waiting for download to start")
	}
}

// waitForFileExists polls until the downloaded file exists on disk.
//
// Takes info (*DownloadInfo) which contains the path to check.
// Takes timeout (time.Duration) which sets how long to wait before giving up.
//
// Returns *DownloadInfo which has its state set to completed when found.
// Returns error when the file does not appear within the timeout period.
func waitForFileExists(info *DownloadInfo, timeout time.Duration) (*DownloadInfo, error) {
	time.Sleep(500 * time.Millisecond)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(info.Path); err == nil {
			info.State = "completed"
			return info, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("download file not found after timeout: %s", info.Path)
}
