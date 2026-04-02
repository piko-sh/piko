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

// Manages per-file compilation logs with in-memory buffers and optional
// disk storage. Provides isolated logging for each component to help
// developers debug compilation issues in parallel builds.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

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

// CompilationLogStore holds the logs for each file processed during a build.
// It stores logs both in memory and on disk, and is safe for use by many
// goroutines at once in a parallel build.
type CompilationLogStore struct {
	// sandboxFactory creates sandboxes when no sandbox is directly injected.
	// When non-nil and sandbox is nil, this factory is used instead of
	// safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// sandbox is an optional filesystem sandbox for testing directory creation.
	// When nil, a real sandbox is created during construction.
	sandbox safedisk.Sandbox

	// buffers maps entry point file paths to their in-memory log buffers,
	// giving quick access to error details when a build fails.
	buffers map[string]*bytes.Buffer

	// logDir is the folder where log files for each component are saved.
	logDir string

	// closers tracks all open file writers (logrotate.Writer
	// instances) to ensure they can be properly closed at the end of
	// a build cycle.
	closers []io.Closer

	// minLogLevel is the minimum log level for buffer and file handlers.
	// It controls how much detail is logged, such as WARN in dev-i mode
	// or DEBUG in dev mode.
	minLogLevel slog.Level

	// mu protects concurrent access to the maps and slices from
	// parallel build workers.
	mu sync.RWMutex

	// enabled controls whether log files are written to disk; when false, logs
	// are kept in memory only.
	enabled bool
}

// CompilationLogStoreOption sets options for a CompilationLogStore when it is
// created.
type CompilationLogStoreOption func(*CompilationLogStore)

// NewCompilationLogStore creates a new compilation log store.
//
// When file logging is enabled, it tries to create the log directory.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes enabled (bool) which controls whether file logging is active.
// Takes logDir (string) which specifies the folder for log files.
// Takes minLogLevel (slog.Level) which sets the lowest log level to record.
// Takes opts (...CompilationLogStoreOption) which provides optional settings
// such as WithLogStoreSandbox for testing.
//
// Returns *CompilationLogStore which is the configured log store ready
// for use.
// Returns error when the log directory cannot be created.
func NewCompilationLogStore(ctx context.Context, enabled bool, logDir string, minLogLevel slog.Level, opts ...CompilationLogStoreOption) (*CompilationLogStore, error) {
	store := &CompilationLogStore{
		buffers:     make(map[string]*bytes.Buffer),
		closers:     make([]io.Closer, 0),
		enabled:     enabled,
		logDir:      logDir,
		minLogLevel: minLogLevel,
		mu:          sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(store)
	}

	if enabled && logDir != "" {
		if err := store.ensureLogDir(ctx, logDir); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// StartSession creates a new logger for a specific component path.
// The logger is separate from the main application logger and writes to both
// an in-memory buffer and, if enabled, a log file.
//
// Takes ctx (context.Context) which controls the lifetime of the background
// rotation goroutine for file-based session logs.
// Takes entryPointPath (string) which identifies the compilation entry point.
// Takes relativePath (string) which specifies the component's relative path.
//
// Returns logger_domain.Logger which is the session logger.
//
// Safe for concurrent use; protected by a mutex.
func (s *CompilationLogStore) StartSession(ctx context.Context, entryPointPath string, relativePath string) logger_domain.Logger {
	s.mu.Lock()
	defer s.mu.Unlock()

	buffer := new(bytes.Buffer)
	s.buffers[entryPointPath] = buffer
	bufferHandler := slog.NewJSONHandler(buffer, &slog.HandlerOptions{
		Level:       s.minLogLevel,
		AddSource:   true,
		ReplaceAttr: nil,
	})

	if !s.enabled {
		slogLogger := slog.New(bufferHandler)
		return logger_domain.New(slogLogger, "annotator-session-mem")
	}

	safeBaseName := strings.ReplaceAll(relativePath, string(filepath.Separator), "_")

	fileWriter, fileError := logrotate.New(ctx, logrotate.Config{
		Directory:  s.logDir,
		Filename:   safeBaseName + ".log",
		MaxSize:    maxLogFileSizeMB,
		MaxBackups: maxLogFileBackups,
		Compress:   true,
	})
	if fileError != nil {
		slogLogger := slog.New(bufferHandler)
		return logger_domain.New(slogLogger, "annotator-session-mem")
	}
	s.closers = append(s.closers, fileWriter)

	fileHandler := slog.NewJSONHandler(fileWriter, &slog.HandlerOptions{
		Level:       s.minLogLevel,
		AddSource:   true,
		ReplaceAttr: nil,
	})

	multiHandler := slog.NewMultiHandler(bufferHandler, fileHandler)
	slogLogger := slog.New(multiHandler)

	return logger_domain.New(slogLogger, "annotator-session-file")
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

// Clear removes all stored logs and closes any open file handles from the
// previous build. Call this before a new build starts to ensure a clean state.
//
// Safe for concurrent use.
func (s *CompilationLogStore) Clear(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, l := logger_domain.From(ctx, log)
	for _, closer := range s.closers {
		if err := closer.Close(); err != nil {
			l.Warn("closing log file handle during clear", logger_domain.Error(err))
		}
	}

	s.buffers = make(map[string]*bytes.Buffer)
	s.closers = []io.Closer{}
}

// Shutdown closes all open log file handles.
// Call this after a build finishes or fails to flush buffers and free
// resources.
//
// Safe for concurrent use.
func (s *CompilationLogStore) Shutdown(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, l := logger_domain.From(ctx, log)
	for _, closer := range s.closers {
		if err := closer.Close(); err != nil {
			l.Warn("closing log file handle during shutdown", logger_domain.Error(err))
		}
	}
	s.closers = []io.Closer{}
}

// ensureLogDir creates the log directory using the configured or temporary
// sandbox.
//
// Takes ctx (context.Context) which carries logging context.
// Takes logDir (string) which specifies the directory to create.
//
// Returns error when the directory cannot be created.
func (s *CompilationLogStore) ensureLogDir(ctx context.Context, logDir string) error {
	_, l := logger_domain.From(ctx, log)
	dirName := filepath.Base(logDir)

	if s.sandbox != nil {
		if err := s.sandbox.MkdirAll(dirName, logDirPermissions); err != nil {
			return fmt.Errorf("failed to create compiler debug log directory at '%s': %w", logDir, err)
		}
		return nil
	}

	parentDir := filepath.Dir(logDir)
	var sandbox safedisk.Sandbox
	var sandboxErr error
	if s.sandboxFactory != nil {
		sandbox, sandboxErr = s.sandboxFactory.Create("compilation-log", parentDir, safedisk.ModeReadWrite)
	} else {
		sandbox, sandboxErr = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadWrite)
	}
	if sandboxErr != nil {
		return fmt.Errorf("failed to create sandbox for log directory '%s': %w", logDir, sandboxErr)
	}
	if err := sandbox.MkdirAll(dirName, logDirPermissions); err != nil {
		if closeErr := sandbox.Close(); closeErr != nil {
			l.Warn("closing sandbox after mkdir failure", logger_domain.Error(closeErr))
		}
		return fmt.Errorf("failed to create compiler debug log directory at '%s': %w", logDir, err)
	}
	if closeErr := sandbox.Close(); closeErr != nil {
		l.Warn("closing temporary sandbox", logger_domain.Error(closeErr))
	}
	return nil
}

// WithLogStoreSandbox sets a sandbox for testing log folder creation.
// The caller must close the sandbox when done.
//
// Takes sandbox (safedisk.Sandbox) which provides file system access.
//
// Returns CompilationLogStoreOption which sets up the store to use the given
// sandbox.
func WithLogStoreSandbox(sandbox safedisk.Sandbox) CompilationLogStoreOption {
	return func(s *CompilationLogStore) {
		s.sandbox = sandbox
	}
}

// WithLogStoreSandboxFactory sets a factory for creating sandboxes when no
// sandbox is directly injected.
//
// Takes factory (safedisk.Factory) which creates sandboxes for log directory
// operations.
//
// Returns CompilationLogStoreOption which sets the factory on the store.
func WithLogStoreSandboxFactory(factory safedisk.Factory) CompilationLogStoreOption {
	return func(s *CompilationLogStore) {
		s.sandboxFactory = factory
	}
}
