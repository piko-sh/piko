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

package lifecycle_adapters

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"

	"github.com/fsnotify/fsnotify"
)

const (
	// eventBufferSize is the buffer capacity for the file event channel.
	eventBufferSize = 100

	// debounceInterval is the minimum time between repeated file events.
	debounceInterval = 50 * time.Millisecond

	// fieldFile is the logging field key for file paths.
	fieldFile = "file"

	// fieldDir is the log field key for directory paths.
	fieldDir = "dir"

	// fieldPathName is the structured logging field key for file paths.
	fieldPathName = "path"
)

// errWatcherClosed is returned when an operation is attempted on a closed
// file watcher.
var errWatcherClosed = errors.New("watcher is closed")

// fsNotifyWatcher implements lifecycle_domain.FileSystemWatcher using the
// fsnotify library. It distinguishes between static recursive watches for core
// source directories and dynamic file watches for specific asset files that
// the build process has identified as dependencies.
type fsNotifyWatcher struct {
	// scanGroup prevents concurrent scans of the same directory root.
	scanGroup singleflight.Group

	// watcher is the underlying fsnotify watcher that monitors file system events.
	watcher *fsnotify.Watcher

	// staticDirs tracks directories added for static watching.
	staticDirs map[string]bool

	// dynamicFiles tracks files that are being watched for changes.
	dynamicFiles map[string]bool

	// debounced tracks when files were last processed to filter rapid
	// duplicate events.
	debounced map[string]time.Time

	// shutdownCh signals scan goroutines to stop; closed during shutdown.
	shutdownCh chan struct{}

	// sandboxFactory creates sandboxes for filesystem access within the watcher.
	sandboxFactory safedisk.Factory

	// scanWg tracks goroutines that scan directories.
	scanWg sync.WaitGroup

	// closeOnce guards single execution of close.
	closeOnce sync.Once

	// mu protects the isClosed field during Watch and Close operations.
	mu sync.Mutex

	// isClosed indicates whether the watcher has been closed.
	isClosed bool
}

var _ lifecycle_domain.FileSystemWatcher = (*fsNotifyWatcher)(nil)

// Watch sets up the initial, static watches on core source directories
// and starts the event loop. It recursively scans these directories
// and adds a watch to every subdirectory found.
//
// Takes ctx (context.Context) which controls the lifetime of the
// event loop.
// Takes staticRecursiveDirs ([]string) which lists directories to
// watch recursively.
//
// Returns <-chan lifecycle_dto.FileEvent which delivers file system
// events.
// Returns error when the watcher has been closed.
//
// Safe for concurrent use. Spawns goroutines to scan static
// directories and run the event loop.
func (f *fsNotifyWatcher) Watch(
	ctx context.Context,
	staticRecursiveDirs []string,
	_ []string,
) (<-chan lifecycle_dto.FileEvent, error) {
	f.mu.Lock()
	if f.isClosed {
		f.mu.Unlock()
		return nil, errWatcherClosed
	}
	f.mu.Unlock()

	ctx, l := logger_domain.From(ctx, log)
	out := make(chan lifecycle_dto.FileEvent, eventBufferSize)

	for _, directory := range staticRecursiveDirs {
		dirCopy := directory
		f.scanWg.Go(func() {
			select {
			case <-f.shutdownCh:
				return
			default:
			}

			l.Trace("Scanning static source directory for initial watch setup", logger_domain.String(fieldDir, dirCopy))
			if err := f.scanAndWatchDirectory(ctx, dirCopy); err != nil {
				l.Warn("Failed to scan static directory", logger_domain.String(fieldDir, dirCopy), logger_domain.Error(err))
			}
		})
	}

	go f.runEventLoop(ctx, out)
	return out, nil
}

// Close releases all resources held by the watcher.
//
// Returns error when the underlying file watcher fails to close.
//
// Safe for concurrent use. Concurrent callers block until the first
// call completes.
func (f *fsNotifyWatcher) Close() error {
	var closeErr error
	f.closeOnce.Do(func() {
		f.mu.Lock()
		f.isClosed = true
		close(f.shutdownCh)
		f.mu.Unlock()

		f.scanWg.Wait()

		f.mu.Lock()
		defer f.mu.Unlock()
		if f.watcher != nil {
			closeErr = f.watcher.Close()
		}
		f.staticDirs = make(map[string]bool)
		f.dynamicFiles = make(map[string]bool)
	})
	if closeErr != nil {
		return fmt.Errorf("closing watcher: %w", closeErr)
	}
	return nil
}

// UpdateWatchedFiles reconciles the watcher's state with the latest list of
// required asset files from the build system.
//
// Takes files ([]string) which specifies the current set of files to watch.
//
// Returns error when the watcher has been closed.
//
// Safe for concurrent use; protects internal state with a mutex.
func (f *fsNotifyWatcher) UpdateWatchedFiles(ctx context.Context, files []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isClosed {
		return errWatcherClosed
	}

	newFiles := make(map[string]bool, len(files))
	for _, file := range files {
		newFiles[file] = true
	}

	f.addNewDynamicWatches(ctx, newFiles)
	f.removeOldDynamicWatches(ctx, newFiles)

	f.dynamicFiles = newFiles
	return nil
}

// addNewDynamicWatches adds watches for files that are new dependencies.
//
// Takes newFiles (map[string]bool) which contains paths to check and watch.
//
// Must be called with mutex held.
func (f *fsNotifyWatcher) addNewDynamicWatches(ctx context.Context, newFiles map[string]bool) {
	_, l := logger_domain.From(ctx, log)
	for file := range newFiles {
		if f.dynamicFiles[file] {
			continue
		}
		if f.isPathCoveredByStaticWatch(file) {
			continue
		}
		if err := f.watcher.Add(file); err != nil {
			l.Warn("Failed to add dynamic watch on asset file", logger_domain.String(fieldFile, file), logger_domain.Error(err))
		} else {
			l.Trace("Now dynamically watching asset file", logger_domain.String(fieldFile, file))
		}
	}
}

// removeOldDynamicWatches removes watches for files that are no longer
// dependencies.
//
// Takes newFiles (map[string]bool) which contains the current set of
// dependency files.
//
// Must be called with mutex held.
func (f *fsNotifyWatcher) removeOldDynamicWatches(ctx context.Context, newFiles map[string]bool) {
	_, l := logger_domain.From(ctx, log)
	for file := range f.dynamicFiles {
		if newFiles[file] {
			continue
		}
		if f.isPathCoveredByStaticWatch(file) {
			continue
		}
		if err := f.watcher.Remove(file); err != nil {
			l.Trace("Could not remove dynamic watch (may have been deleted)", logger_domain.String(fieldFile, file), logger_domain.Error(err))
		} else {
			l.Trace("Stopped dynamically watching asset file", logger_domain.String(fieldFile, file))
		}
	}
}

// scanAndWatchDirectory walks a directory tree, adding every subdirectory to
// the underlying fsnotify watcher and recording it in the staticDirs map.
//
// Takes root (string) which specifies the directory path to scan and watch.
//
// Returns error when the watcher is closed or the directory walk fails.
//
// Safe for concurrent use. Uses singleflight to prevent concurrent scans of
// the same directory root.
func (f *fsNotifyWatcher) scanAndWatchDirectory(ctx context.Context, root string) error {
	f.mu.Lock()
	if f.isClosed {
		f.mu.Unlock()
		return errWatcherClosed
	}
	f.mu.Unlock()

	_, err, _ := f.scanGroup.Do(root, func() (any, error) {
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			select {
			case <-f.shutdownCh:
				return fs.SkipAll
			default:
			}
			return f.walkDirCallback(ctx, path, d, err)
		})
		if walkErr != nil && !errors.Is(walkErr, fs.SkipAll) {
			return nil, fmt.Errorf("walkDir failed for %s: %w", root, walkErr)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("scanning and watching directory %q: %w", root, err)
	}

	return nil
}

// walkDirCallback handles a single directory entry during a directory scan.
//
// Takes path (string) which is the file system path being visited.
// Takes d (fs.DirEntry) which holds details about the directory entry.
// Takes walkErr (error) which is any error found while reading the path.
//
// Returns error when adding a watch fails; returns nil to skip files and
// errors, or fs.SkipDir to skip excluded directories.
func (f *fsNotifyWatcher) walkDirCallback(ctx context.Context, path string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Internal("Skipping path due to walk error", logger_domain.String(fieldPathName, path), logger_domain.Error(walkErr))
		return nil
	}
	if !d.IsDir() {
		return nil
	}
	if f.shouldSkipDirectory(ctx, path, d) {
		return fs.SkipDir
	}
	if err := f.tryAddStaticWatch(ctx, path); err != nil {
		return fmt.Errorf("adding static watch for %q: %w", path, err)
	}
	return nil
}

// shouldSkipDirectory determines if a directory should be skipped during
// scanning.
//
// Takes path (string) which is the full path to the directory.
// Takes d (fs.DirEntry) which provides directory entry information.
//
// Returns bool which is true when the directory should be skipped.
func (*fsNotifyWatcher) shouldSkipDirectory(ctx context.Context, path string, d fs.DirEntry) bool {
	info, err := d.Info()
	if err != nil {
		return true
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	baseName := filepath.Base(path)
	if baseName == "node_modules" || baseName == ".git" {
		_, l := logger_domain.From(ctx, log)
		l.Trace("Skipping large or irrelevant directory", logger_domain.String(fieldDir, path))
		return true
	}
	return false
}

// tryAddStaticWatch attempts to add a static watch for a directory.
//
// Takes path (string) which specifies the directory to watch.
//
// Returns error when the watcher is closed (fs.SkipDir), nil otherwise.
//
// Safe for concurrent use; protects internal state with a mutex.
func (f *fsNotifyWatcher) tryAddStaticWatch(ctx context.Context, path string) error {
	f.mu.Lock()
	if f.isClosed {
		f.mu.Unlock()
		return fs.SkipDir
	}
	if f.staticDirs[path] {
		f.mu.Unlock()
		return nil
	}
	f.staticDirs[path] = true
	f.mu.Unlock()

	_, l := logger_domain.From(ctx, log)
	if addErr := f.watcher.Add(path); addErr != nil {
		l.Warn("Failed to add static watch", logger_domain.String(fieldDir, path), logger_domain.Error(addErr))
	} else {
		l.Trace("Now watching static directory", logger_domain.String(fieldDir, path))
	}
	return nil
}

// removeStaticWatch removes a watch on a directory that was deleted or
// renamed.
//
// Takes directory (string) which specifies the directory path to stop watching.
//
// Safe for concurrent use; protects access to the static directories map with
// a mutex.
func (f *fsNotifyWatcher) removeStaticWatch(ctx context.Context, directory string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.staticDirs[directory] {
		return
	}
	_, l := logger_domain.From(ctx, log)
	if err := f.watcher.Remove(directory); err != nil {
		l.Trace("Failed removing static watch (directory likely already gone)", logger_domain.String(fieldPathName, directory), logger_domain.Error(err))
	} else {
		l.Trace("Stopped watching static directory", logger_domain.String(fieldPathName, directory))
	}
	delete(f.staticDirs, directory)
}

// runEventLoop is the main loop that consumes events from fsnotify.
//
// Takes out (chan<- lifecycle_dto.FileEvent) which receives file events.
func (f *fsNotifyWatcher) runEventLoop(ctx context.Context, out chan<- lifecycle_dto.FileEvent) {
	defer close(out)
	defer goroutine.RecoverPanic(ctx, "lifecycle.fsnotifyEventLoop")
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("FSNotify event loop started.")
	for {
		select {
		case <-ctx.Done():
			l.Internal("FSNotify event loop context closed.")
			return
		case ev, ok := <-f.watcher.Events:
			if !ok {
				l.Internal("FSNotify events channel closed.")
				return
			}
			f.handleFsnotifyEvent(ctx, ev, out)
		case wErr, ok := <-f.watcher.Errors:
			if !ok {
				l.Internal("FSNotify errors channel closed.")
				return
			}
			l.Error("FSNotify error", logger_domain.String("error", wErr.Error()))
		}
	}
}

// handleFsnotifyEvent processes a single raw event from the file watcher.
//
// Takes ev (fsnotify.Event) which is the raw file system event to process.
// Takes out (chan<- lifecycle_dto.FileEvent) which receives the converted
// event if it passes filtering.
func (f *fsNotifyWatcher) handleFsnotifyEvent(
	ctx context.Context,
	ev fsnotify.Event,
	out chan<- lifecycle_dto.FileEvent,
) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Raw fsnotify event", logger_domain.String("event", ev.String()))

	if f.shouldIgnoreEvent(ctx, ev) {
		return
	}
	if f.shouldDebounceEvent(ctx, ev) {
		return
	}

	f.handleDirectoryCreation(ctx, ev)
	f.handleDirectoryRemoval(ctx, ev)
	f.forwardEvent(ctx, ev, out)
}

// shouldIgnoreEvent checks if the event should be ignored.
//
// Takes ev (fsnotify.Event) which is the file system event to check.
//
// Returns bool which is true when the event should be ignored, such as
// editor backup files ending with a tilde.
func (*fsNotifyWatcher) shouldIgnoreEvent(ctx context.Context, ev fsnotify.Event) bool {
	if strings.HasSuffix(ev.Name, "~") {
		_, l := logger_domain.From(ctx, log)
		l.Trace("Ignoring editor backup file", logger_domain.String(fieldFile, ev.Name))
		return true
	}
	return false
}

// shouldDebounceEvent checks if the event should be debounced.
//
// Takes ev (fsnotify.Event) which is the file system event to check.
//
// Returns bool which is true if the event was debounced and should not be
// processed.
//
// Safe for concurrent use. Protects debounce state with a mutex.
func (f *fsNotifyWatcher) shouldDebounceEvent(ctx context.Context, ev fsnotify.Event) bool {
	now := time.Now()
	f.mu.Lock()
	defer f.mu.Unlock()

	last, found := f.debounced[ev.Name]
	isCreateOrRemove := ev.Has(fsnotify.Create) || ev.Has(fsnotify.Remove)

	if found && now.Sub(last) < debounceInterval && !isCreateOrRemove {
		_, l := logger_domain.From(ctx, log)
		l.Trace("Debounced event", logger_domain.String(fieldPathName, ev.Name))
		return true
	}
	f.debounced[ev.Name] = now
	return false
}

// handleDirectoryCreation processes a create event for a new directory inside
// a statically watched tree.
//
// Takes ev (fsnotify.Event) which is the filesystem event to process.
//
// Spawns a goroutine to scan and watch the new directory and its children.
// Safe for concurrent use; guards internal state with a mutex.
func (f *fsNotifyWatcher) handleDirectoryCreation(ctx context.Context, ev fsnotify.Event) {
	if !ev.Has(fsnotify.Create) {
		return
	}

	parentDir := filepath.Dir(ev.Name)
	fileName := filepath.Base(ev.Name)
	var sandbox safedisk.Sandbox
	var err error
	if f.sandboxFactory != nil {
		sandbox, err = f.sandboxFactory.Create("fsnotify-stat", parentDir, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
	}
	if err != nil {
		return
	}
	defer func() { _ = sandbox.Close() }()

	info, err := sandbox.Stat(fileName)
	if err != nil || !info.IsDir() {
		return
	}

	f.mu.Lock()
	isCovered := f.isPathCoveredByStaticWatch(ev.Name)
	f.mu.Unlock()

	if !isCovered {
		return
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("Detected new directory in static tree; scanning recursively", logger_domain.String(fieldDir, ev.Name))
	dirToScan := ev.Name
	f.scanWg.Go(func() {
		select {
		case <-f.shutdownCh:
			return
		default:
		}

		if serr := f.scanAndWatchDirectory(ctx, dirToScan); serr != nil {
			l.Warn("Failed to scan newly created directory", logger_domain.String(fieldDir, dirToScan), logger_domain.Error(serr))
		}
	})
}

// handleDirectoryRemoval handles the case when a watched directory is
// removed or renamed.
//
// Takes ev (fsnotify.Event) which specifies the file system event to process.
func (f *fsNotifyWatcher) handleDirectoryRemoval(ctx context.Context, ev fsnotify.Event) {
	if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
		f.removeStaticWatch(ctx, ev.Name)
	}
}

// forwardEvent sends a file system event to the output channel.
//
// Takes ev (fsnotify.Event) which is the file system event to send.
// Takes out (chan<- lifecycle_dto.FileEvent) which receives the converted
// event.
func (*fsNotifyWatcher) forwardEvent(ctx context.Context, ev fsnotify.Event, out chan<- lifecycle_dto.FileEvent) {
	outEvt := lifecycle_dto.FileEvent{
		Path: ev.Name,
		Type: mapToFileEventType(ev.Op),
	}
	select {
	case out <- outEvt:
	case <-ctx.Done():
	}
}

// isPathCoveredByStaticWatch checks if a file's parent directory or any
// ancestor is already monitored by a static recursive watch, making a
// specific file watch redundant.
//
// Takes filePath (string) which is the path to check for coverage.
//
// Returns bool which is true if the path is covered by an existing watch.
//
// Must be called within a lock.
func (f *fsNotifyWatcher) isPathCoveredByStaticWatch(filePath string) bool {
	directory := filepath.Dir(filePath)
	for {
		if f.staticDirs[directory] {
			return true
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return false
}

// NewFSNotifyWatcher creates a new, uninitialised fsnotify-based watcher.
//
// Takes sandboxFactory (safedisk.Factory) which creates sandboxes for
// filesystem access.
//
// Returns lifecycle_domain.FileSystemWatcher which is ready to be configured
// and started.
// Returns error when the underlying fsnotify watcher cannot be created.
func NewFSNotifyWatcher(sandboxFactory safedisk.Factory) (lifecycle_domain.FileSystemWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("unable to create fsnotify watcher: %w", err)
	}
	return &fsNotifyWatcher{
		scanGroup:      singleflight.Group{},
		watcher:        w,
		staticDirs:     make(map[string]bool),
		dynamicFiles:   make(map[string]bool),
		debounced:      make(map[string]time.Time),
		shutdownCh:     make(chan struct{}),
		sandboxFactory: sandboxFactory,
		scanWg:         sync.WaitGroup{},
		closeOnce:      sync.Once{},
		mu:             sync.Mutex{},
		isClosed:       false,
	}, nil
}

// mapToFileEventType converts an fsnotify operation bitmask to the domain's
// event type.
//
// Takes op (fsnotify.Op) which is the filesystem operation bitmask to convert.
//
// Returns lifecycle_dto.FileEventType which represents the mapped domain event
// type, or FileEventTypeUnknown if the operation is not recognised.
func mapToFileEventType(op fsnotify.Op) lifecycle_dto.FileEventType {
	switch {
	case op&fsnotify.Create != 0:
		return lifecycle_dto.FileEventTypeCreate
	case op&fsnotify.Write != 0:
		return lifecycle_dto.FileEventTypeWrite
	case op&fsnotify.Remove != 0:
		return lifecycle_dto.FileEventTypeRemove
	case op&fsnotify.Rename != 0:
		return lifecycle_dto.FileEventTypeRename
	}
	return lifecycle_dto.FileEventTypeUnknown
}
