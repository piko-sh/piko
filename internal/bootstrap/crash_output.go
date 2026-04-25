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

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// crashOutputFileMode is the permission mode used for the crash output file.
// Restricted to owner-read/write because crash output may contain process
// memory addresses and stack frames the operator does not want world-readable.
const crashOutputFileMode = 0o600

// crashOutputPathLogField is the structured-log field name used when reporting
// the crash output file path. Centralised so the value matches across all
// callers.
const crashOutputPathLogField = "path"

// validTracebackLevels is the set of values accepted by SetTraceback. The
// runtime treats any other value as "single" silently, which makes typos
// hard to spot, so we validate explicitly to surface them.
var validTracebackLevels = map[string]struct{}{
	"none":   {},
	"single": {},
	"all":    {},
	"system": {},
	"crash":  {},
	"wer":    {},
}

// ErrInvalidCrashTracebackLevel is returned when the user supplies an
// unrecognised GOTRACEBACK level via WithCrashTraceback.
var ErrInvalidCrashTracebackLevel = errors.New("invalid crash traceback level")

// InstallCrashOutput configures the Go runtime to mirror unrecovered panic
// output and fatal-error tracebacks based on the container's crash settings.
// It must be called before any application goroutines are spawned because
// the runtime takes ownership of the file descriptor for its lifetime.
//
// The function is best-effort: any failure to open the crash file is logged
// at Warn level but does NOT propagate, so a misconfigured crash path cannot
// prevent the process from starting. SetTraceback errors do propagate
// because they indicate a programming error in option construction (an
// invalid level constant).
//
// Path validation is delegated to a safedisk sandbox rooted at the parent
// directory of the configured path. The sandbox catches typos, traversal,
// and unwritable parents at Open time. The actual file is then opened with
// os.OpenFile because runtime/debug.SetCrashOutput requires *os.File and
// safedisk's FileHandle interface intentionally hides the underlying type.
//
// Takes container (*Container) which provides the configured crash output
// path and traceback level.
//
// Returns func() which releases the crash file descriptor on shutdown, or
// nil when no file was opened.
// Returns error when the configured traceback level is invalid.
func InstallCrashOutput(ctx context.Context, container *Container) (closeFn func(), err error) {
	if container == nil {
		return nil, nil
	}

	ctx, l := logger_domain.From(ctx, log)

	if err := applyCrashTraceback(ctx, container.crashTracebackLevel); err != nil {
		return nil, err
	}

	path := container.crashOutputPath
	if path == "" {
		return nil, nil
	}

	cleanPath, validateErr := validateCrashOutputPath(path)
	if validateErr != nil {
		l.Warn("Crash output path failed sandbox validation; continuing without crash mirroring",
			logger_domain.String(crashOutputPathLogField, path),
			logger_domain.Error(validateErr),
		)
		return nil, nil
	}

	return openAndRegisterCrashFile(ctx, cleanPath)
}

// applyCrashTraceback validates and applies the configured GOTRACEBACK level
// when one is provided. An invalid level surfaces as an error so callers can
// treat it as a programming mistake in option construction.
//
// Takes level (string) which is the GOTRACEBACK level to apply; empty is a
// no-op.
//
// Returns error wrapping ErrInvalidCrashTracebackLevel when level is not a
// recognised GOTRACEBACK constant.
func applyCrashTraceback(ctx context.Context, level string) error {
	if level == "" {
		return nil
	}
	if _, ok := validTracebackLevels[level]; !ok {
		return fmt.Errorf("%w: %q (valid: none, single, all, system, crash, wer)",
			ErrInvalidCrashTracebackLevel, level)
	}
	debug.SetTraceback(level)
	_, l := logger_domain.From(ctx, log)
	l.Internal("Configured runtime traceback level",
		logger_domain.String("level", level),
	)
	return nil
}

// openAndRegisterCrashFile opens the validated path with os.OpenFile (the
// runtime API requires *os.File) and registers it with
// runtime/debug.SetCrashOutput.
//
// Takes cleanPath (string) which is the safedisk-validated absolute file
// path to open in append mode.
//
// Returns func() which detaches the FD from the runtime and closes it, or
// nil when the file could not be opened or registered (best-effort).
// Returns error which is currently always nil and exists for symmetry with
// callers that may surface a real error in the future.
func openAndRegisterCrashFile(ctx context.Context, cleanPath string) (func(), error) {
	_, l := logger_domain.From(ctx, log)

	file, err := os.OpenFile(cleanPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, crashOutputFileMode) //nolint:gosec // path is operator-supplied trusted config validated via safedisk above
	if err != nil {
		l.Warn("Failed to open crash output file; continuing without crash mirroring",
			logger_domain.String(crashOutputPathLogField, cleanPath),
			logger_domain.Error(err),
		)
		return nil, nil
	}

	if registerErr := debug.SetCrashOutput(file, debug.CrashOptions{}); registerErr != nil {
		_ = file.Close()
		l.Warn("Failed to register crash output file with runtime; continuing without crash mirroring",
			logger_domain.String(crashOutputPathLogField, cleanPath),
			logger_domain.Error(registerErr),
		)
		return nil, nil
	}

	l.Internal("Crash output mirroring enabled",
		logger_domain.String(crashOutputPathLogField, cleanPath),
	)

	return func() {
		_ = debug.SetCrashOutput(nil, debug.CrashOptions{})
		_ = file.Close()
	}, nil
}

// validateCrashOutputPath sandbox-validates the crash output path.
//
// The safedisk sandbox catches relative paths that escape the parent
// (".." traversal), parent directories that don't exist or aren't
// writable, and symlink-based escapes (subject to the sandbox
// implementation).
//
// Takes path (string) which is the operator-supplied crash output file
// path, possibly relative or unclean.
//
// Returns string which is the cleaned absolute path on success.
// Returns error which wraps the underlying safedisk failure when the
// path cannot be sandbox-validated.
func validateCrashOutputPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	parent := filepath.Dir(cleanPath)
	base := filepath.Base(cleanPath)

	sandbox, err := safedisk.NewSandbox(parent, safedisk.ModeReadWrite)
	if err != nil {
		return "", fmt.Errorf("creating crash output sandbox at %q: %w", parent, err)
	}
	defer func() { _ = sandbox.Close() }()

	probe, err := sandbox.OpenFile(base, os.O_WRONLY|os.O_CREATE|os.O_APPEND, crashOutputFileMode)
	if err != nil {
		return "", fmt.Errorf("validating crash output path %q via sandbox: %w", cleanPath, err)
	}
	if closeErr := probe.Close(); closeErr != nil {
		return "", fmt.Errorf("closing sandbox probe for %q: %w", cleanPath, closeErr)
	}

	return cleanPath, nil
}
