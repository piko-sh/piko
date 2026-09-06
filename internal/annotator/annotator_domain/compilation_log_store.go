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

package annotator_domain

// Manages per-file compilation logs with in-memory buffers and optional disk storage.
// Provides isolated logging for each component to help developers debug compilation
// issues in parallel builds.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path/filepath"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logrotate"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// logDirPermissions is the Unix file mode for log directories (rwxr-xr-x).
	logDirPermissions = 0755

	// maxLogFileSizeMB is the largest size in megabytes for a single log file.
	maxLogFileSizeMB = 5

	// maxLogFileBackups is the number of old log files to keep.
	maxLogFileBackups = 1
)

// CompilationLogStore holds the logs for each file processed during a build. It stores
// logs both in memory and on disk, and is safe for use by many goroutines at once in a
// parallel build.
type CompilationLogStore struct {
	// sandboxFactory creates the sandbox used to prepare the log directory when no sandbox
	// is directly injected. When nil, a sandboxing factory restricted to the log directory
	// is created on demand.
	sandboxFactory safedisk.Factory

	// sandbox is an optional filesystem sandbox, rooted at the parent of the log directory,
	// for testing directory creation.
	sandbox safedisk.Sandbox

	// buffers maps entry point file paths to their in-memory log buffers, giving quick
	// access to error details when a build fails.
	buffers map[string]*sessionBuffer

	// logDir is the folder where log files for each component are saved.
	logDir string

	// closers tracks all open file writers (logrotate.Writer instances) to ensure they can
	// be properly closed at the end of a build cycle.
	closers []io.Closer

	// minLogLevel is the minimum log level for buffer and file handlers. It controls how
	// much detail is logged, such as WARN in dev-i mode or DEBUG in dev mode.
	minLogLevel slog.Level

	// mu protects concurrent access to the maps and slices from parallel build workers.
	mu sync.RWMutex

	// prepareOnce ensures the log directory is prepared at most once.
	prepareOnce sync.Once

	// fileFailureReported records whether a session log file failure has been logged as a
	// warning since the last Clear.
	fileFailureReported atomic.Bool

	// fileLoggingEnabled controls whether log files are written to disk; when false, logs
	// are kept in memory only.
	fileLoggingEnabled atomic.Bool
}

// CompilationLogStoreOption sets options for a CompilationLogStore when it is created.
type CompilationLogStoreOption func(*CompilationLogStore)

// sessionBuffer holds one session's in-memory log. It is safe to read while the session
// logger writes to it.
type sessionBuffer struct {
	// buffer holds the log output.
	buffer bytes.Buffer

	// mu serialises writes and reads of buffer.
	mu sync.Mutex
}

// NewCompilationLogStore creates a new compilation log store. It performs no I/O; call
// PrepareLogDirectory before the first build to check the log directory.
//
// Takes enabled (bool) which controls whether file logging is active. File logging stays
// off when logDir is empty.
// Takes logDir (string) which specifies the folder for log files.
// Takes minLogLevel (slog.Level) which sets the lowest log level to record.
// Takes opts (...CompilationLogStoreOption) which provides optional settings such as
// WithLogStoreSandboxFactory.
//
// Returns *CompilationLogStore which is the configured log store ready for use.
func NewCompilationLogStore(enabled bool, logDir string, minLogLevel slog.Level, opts ...CompilationLogStoreOption) *CompilationLogStore {
	store := &CompilationLogStore{}
	store.buffers = make(map[string]*sessionBuffer)
	store.closers = make([]io.Closer, 0)
	store.logDir = logDir
	store.minLogLevel = minLogLevel
	store.fileLoggingEnabled.Store(enabled && logDir != "")

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// PrepareLogDirectory creates the log directory through a real filesystem sandbox. It
// runs once; later calls return immediately.
//
// An uncreatable log directory (a read-only filesystem, the normal state of an embedded
// single-binary container) downgrades the store to in-memory-only logging with a single
// warning rather than failing, because the file log is a debug convenience and must never
// block a build.
func (s *CompilationLogStore) PrepareLogDirectory(ctx context.Context) {
	s.prepareOnce.Do(func() {
		if !s.fileLoggingEnabled.Load() {
			return
		}
		if err := s.ensureLogDir(ctx); err != nil {
			s.fileLoggingEnabled.Store(false)
			_, l := logger_domain.From(ctx, log)
			l.Warn("Compilation log directory unavailable; keeping logs in memory only",
				logger_domain.String("logDir", s.logDir), logger_domain.Error(err))
		}
	})
}

// StartSession creates a new logger for a specific component path. The logger is separate
// from the main application logger and writes to both an in-memory buffer and, if
// enabled, a log file.
//
// When the log file cannot be opened the session falls back to the in-memory buffer and
// the failure is logged; the first failure in a build cycle is logged as a warning and
// later ones at internal level so that a broken log directory cannot flood the log.
//
// Takes entryPointPath (string) which identifies the compilation entry point.
// Takes relativePath (string) which specifies the component's relative path.
//
// Returns logger_domain.Logger which is the session logger.
//
// Safe for concurrent use; the mutex is held only while registering the session.
func (s *CompilationLogStore) StartSession(ctx context.Context, entryPointPath string, relativePath string) logger_domain.Logger {
	buffer := new(sessionBuffer)
	bufferHandler := slog.NewJSONHandler(buffer, &slog.HandlerOptions{
		Level:       s.minLogLevel,
		AddSource:   true,
		ReplaceAttr: nil,
	})

	s.mu.Lock()
	s.buffers[entryPointPath] = buffer
	s.mu.Unlock()

	if !s.fileLoggingEnabled.Load() {
		return logger_domain.New(slog.New(bufferHandler), "annotator-session-mem")
	}

	fileWriter, err := logrotate.New(ctx, logrotate.Config{
		Directory:  s.logDir,
		Filename:   sessionLogFileName(relativePath),
		MaxSize:    maxLogFileSizeMB,
		MaxBackups: maxLogFileBackups,
		Compress:   true,
		Sandbox:    nil,
		Clock:      nil,
		MaxAge:     0,
		LocalTime:  false,
	})
	if err != nil {
		s.reportSessionFileFailure(ctx, relativePath, err)
		return logger_domain.New(slog.New(bufferHandler), "annotator-session-mem")
	}

	s.mu.Lock()
	s.closers = append(s.closers, fileWriter)
	s.mu.Unlock()

	fileHandler := slog.NewJSONHandler(fileWriter, &slog.HandlerOptions{
		Level:       s.minLogLevel,
		AddSource:   true,
		ReplaceAttr: nil,
	})

	multiHandler := slog.NewMultiHandler(bufferHandler, fileHandler)
	return logger_domain.New(slog.New(multiHandler), "annotator-session-file")
}

// GetLogs retrieves the complete in-memory log content for a specific file.
//
// Takes filePath (string) which specifies the path to look up.
//
// Returns string which contains the log content for the file.
// Returns bool which indicates whether the file was found.
//
// Safe for concurrent use.
func (s *CompilationLogStore) GetLogs(filePath string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	buffer, ok := s.buffers[filePath]
	if !ok {
		return "", false
	}
	return buffer.String(), true
}

// Clear removes all stored logs and closes any open file handles from the previous build.
// Call this before a new build starts to ensure a clean state.
//
// Safe for concurrent use; the file handles are closed after the mutex is released.
func (s *CompilationLogStore) Clear(ctx context.Context) {
	s.mu.Lock()
	closers := s.closers
	s.buffers = make(map[string]*sessionBuffer)
	s.closers = make([]io.Closer, 0)
	s.mu.Unlock()

	s.fileFailureReported.Store(false)

	if err := closeLogWriters(closers); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Closing compilation log files during clear failed", logger_domain.Error(err))
	}
}

// Shutdown closes all open log file handles. Call this after a build finishes or fails to
// flush buffers and free resources.
//
// Safe for concurrent use; the file handles are closed after the mutex is released.
func (s *CompilationLogStore) Shutdown(ctx context.Context) {
	s.mu.Lock()
	closers := s.closers
	s.closers = make([]io.Closer, 0)
	s.mu.Unlock()

	if err := closeLogWriters(closers); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Closing compilation log files during shutdown failed", logger_domain.Error(err))
	}
}

// reportSessionFileFailure logs that a session log file could not be opened. The first
// failure since the last Clear is logged as a warning; later failures are logged at
// internal level so that a broken log directory does not flood the log.
//
// Takes relativePath (string) which identifies the component whose log file failed.
// Takes err (error) which is the failure from opening the log file.
func (s *CompilationLogStore) reportSessionFileFailure(ctx context.Context, relativePath string, err error) {
	_, l := logger_domain.From(ctx, log)
	if s.fileFailureReported.CompareAndSwap(false, true) {
		l.Warn("Compilation log file unavailable; keeping session logs in memory only",
			logger_domain.String("component", relativePath),
			logger_domain.String("logDir", s.logDir),
			logger_domain.Error(err))
		return
	}
	l.Internal("Compilation log file unavailable; keeping session logs in memory only",
		logger_domain.String("component", relativePath),
		logger_domain.Error(err))
}

// ensureLogDir creates the log directory. An injected sandbox, rooted at the directory's
// parent, is used when present; otherwise a sandbox is created on the log directory
// itself, which creates the directory and any missing parents.
//
// Returns error when the directory cannot be created.
func (s *CompilationLogStore) ensureLogDir(ctx context.Context) error {
	if s.sandbox != nil {
		if err := s.sandbox.MkdirAll(filepath.Base(s.logDir), logDirPermissions); err != nil {
			return fmt.Errorf("creating compiler debug log directory %q: %w", s.logDir, err)
		}
		return nil
	}

	factory, err := s.logDirectoryFactory()
	if err != nil {
		return err
	}
	sandbox, err := factory.Create("compilation-log", s.logDir, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("creating compiler debug log directory %q: %w", s.logDir, err)
	}
	if closeErr := sandbox.Close(); closeErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Closing compilation log directory sandbox failed", logger_domain.Error(closeErr))
	}
	return nil
}

// logDirectoryFactory returns the injected sandbox factory for preparing the log
// directory, or otherwise a sandboxing factory restricted to the log directory.
//
// Returns safedisk.Factory which creates the log directory sandbox.
// Returns error when the restricted factory cannot be created.
func (s *CompilationLogStore) logDirectoryFactory() (safedisk.Factory, error) {
	if s.sandboxFactory != nil {
		return s.sandboxFactory, nil
	}
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          "",
		AllowedPaths: []string{s.logDir},
		Enabled:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating sandbox factory for compiler debug log directory %q: %w", s.logDir, err)
	}
	return factory, nil
}

// Write appends a log record to the buffer.
//
// Takes payload ([]byte) which is the encoded log record.
//
// Returns int which is the number of bytes written.
// Returns error which is always nil; bytes.Buffer grows as needed.
//
// Safe for concurrent use.
func (b *sessionBuffer) Write(payload []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(payload)
}

// String returns a copy of the log content written so far.
//
// Returns string which is the buffered log output.
//
// Safe for concurrent use.
func (b *sessionBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// WithLogStoreSandbox sets a sandbox for testing log folder creation. The caller must
// close the sandbox when done.
//
// Takes sandbox (safedisk.Sandbox) which provides file system access.
//
// Returns CompilationLogStoreOption which sets up the store to use the given sandbox.
func WithLogStoreSandbox(sandbox safedisk.Sandbox) CompilationLogStoreOption {
	return func(s *CompilationLogStore) {
		s.sandbox = sandbox
	}
}

// WithLogStoreSandboxFactory sets a factory for creating sandboxes when no sandbox is
// directly injected.
//
// Takes factory (safedisk.Factory) which creates sandboxes for log directory operations.
//
// Returns CompilationLogStoreOption which sets the factory on the store.
func WithLogStoreSandboxFactory(factory safedisk.Factory) CompilationLogStoreOption {
	return func(s *CompilationLogStore) {
		s.sandboxFactory = factory
	}
}

// sessionLogFileName builds the log file name for a component. The path is escaped so
// that distinct component paths always map to distinct names by percent-encoding
// separators and every character outside letters, digits, '-', '_', '.' and '~', which
// keeps a/b_c.pk and a_b/c.pk apart.
//
// Takes relativePath (string) which is the component's path relative to the project.
//
// Returns string which is the file name, ending in .log.
func sessionLogFileName(relativePath string) string {
	return url.QueryEscape(filepath.ToSlash(relativePath)) + ".log"
}

// closeLogWriters closes every writer, continuing past failures.
//
// Takes closers ([]io.Closer) which are the writers to close.
//
// Returns error which joins every close failure, or nil when all writers closed.
func closeLogWriters(closers []io.Closer) error {
	var errs []error
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
