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

// Package process executes local workers with bounded logs and cancellation.
package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"piko.sh/piko/tests/fuzz/internal/textlimit"
)

const (
	// logLimit bounds retained process-output bytes.
	logLimit = 256 << 10

	// waitDelay bounds how long output pipes may stay open after the child exits.
	waitDelay = 2 * time.Second
)

// outputLog retains bounded stream prefixes and suffixes while draining all output.
type outputLog struct {
	// head retains the beginning of process output.
	head []byte

	// tail retains the end of process output.
	tail []byte

	// total counts all process-output bytes.
	total int64

	// mu protects the mutable state.
	mu sync.Mutex
}

// tempKey identifies an invocation-private temporary directory in a context.
type tempKey struct{}

// Write accepts all output while retaining a bounded prefix and suffix.
//
// Takes data ([]byte) which holds the process-output bytes.
//
// Returns int which is the number of accepted bytes.
// Returns error which is always nil.
//
// Concurrency: Safe for concurrent use. The log mutex serialises writes and snapshots.
func (b *outputLog) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	accepted := len(data)
	b.total += int64(accepted)
	first := min(len(data), logLimit/2-len(b.head))
	b.head = append(b.head, data[:first]...)
	data = data[first:]
	if len(data) >= logLimit/2 {
		b.tail = append(b.tail[:0], data[len(data)-logLimit/2:]...)
	} else {
		excess := len(b.tail) + len(data) - logLimit/2
		if excess > 0 {
			b.tail = b.tail[:copy(b.tail, b.tail[excess:])]
		}
		b.tail = append(b.tail, data...)
	}
	return accepted, nil
}

// String formats the retained process-output prefix and suffix.
//
// When output was omitted, the prefix and suffix are trimmed to whole UTF-8 runes so the
// cut never splits a character.
//
// Returns string which is the bounded process log with omitted output indicated.
//
// Concurrency: Safe for concurrent use. The log mutex protects the copied snapshot.
func (b *outputLog) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	omitted := b.total - int64(len(b.head)+len(b.tail))
	if omitted > 0 {
		head := textlimit.CompletePrefix(b.head)
		tail := textlimit.CompleteSuffix(b.tail)
		omitted += int64(len(b.head) - len(head) + len(b.tail) - len(tail))
		return fmt.Sprintf("%s\n... %d log bytes omitted ...\n%s", head, omitted, tail)
	}
	return string(b.head) + string(b.tail)
}

// Run executes a subprocess in directory.
//
// Takes directory (string) which is the subprocess working directory.
// Takes environment ([]string) which adds or overrides inherited environment entries.
// Takes name (string) which is the subprocess executable.
// Takes arguments (...string) which supplies the subprocess arguments.
//
// Returns string which is the bounded subprocess output.
// Returns error which reports subprocess startup, execution or cancellation failure, or
// nil on success.
func Run(ctx context.Context, directory string, environment []string, name string, arguments ...string) (string, error) {
	return RunObserved(ctx, directory, environment, nil, name, arguments...)
}

// RunObserved copies the child's output to an optional observer while keeping the same
// bounded diagnostic log and process-group cleanup as Run.
//
// On Linux the child runs in its own process group, cancellation kills the whole group,
// and the group is killed again once the child exits, so descendants never outlive the
// call. Other platforms only stop the child itself.
//
// A descendant that keeps the inherited stdout or stderr open after the child exits makes
// the call wait waitDelay and then fail with exec.ErrWaitDelay, which surfaces a leaked
// background process rather than hiding it.
//
// Takes directory (string) which is the subprocess working directory.
// Takes environment ([]string) which adds or overrides inherited environment entries.
// Takes observer (io.Writer) which receives process output while the bounded log is
// retained.
// Takes name (string) which is the subprocess executable.
// Takes arguments (...string) which supplies the command arguments.
//
// Returns string which is the bounded subprocess output.
// Returns error which reports subprocess startup, execution or cancellation failure, or
// nil on success.
func RunObserved(ctx context.Context, directory string, environment []string, observer io.Writer, name string, arguments ...string) (string, error) {
	if ctx.Err() != nil {
		return "", context.Cause(ctx)
	}
	//nolint:gosec // callers launch only the Go toolchain, git, pdftotext or binaries the harness built; exec uses no shell.
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), environment...)
	if scratch, ok := ctx.Value(tempKey{}).(string); ok {
		command.Env = append(command.Env, "TMPDIR="+scratch, "GOTMPDIR="+scratch)
	}
	var log outputLog
	var output io.Writer = &log
	if observer != nil {
		output = io.MultiWriter(output, observer)
	}
	command.Stdout = output
	command.Stderr = output
	command.WaitDelay = waitDelay
	supervise(command)
	err := command.Run()
	reapErr := reap(command)
	if ctx.Err() != nil {
		return log.String(), context.Cause(ctx)
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return log.String(), fmt.Errorf("%s left output pipes open: %w", name, err)
	}
	if err != nil {
		return log.String(), err
	}
	if reapErr != nil {
		return log.String(), fmt.Errorf("%s process-group cleanup: %w", name, reapErr)
	}
	return log.String(), nil
}

// WithTempDir gives all phase descendants an owned temporary directory.
//
// Takes directory (string) which is the invocation-private temporary directory.
//
// Returns context.Context which carries the temporary directory for child processes.
func WithTempDir(ctx context.Context, directory string) context.Context {
	return context.WithValue(ctx, tempKey{}, directory)
}

// BuildWorker lazily links one invariant helper in the invocation's private directory.
//
// Callers run one phase at a time. Generated applications still compile against their own
// case-specific workspace and source files.
//
// Takes repository (string) which is the repository source directory.
// Takes workers (string) which is the invocation-private worker executable directory.
// Takes name (string) which selects the helper under tests/fuzz/cmd.
// Takes tag (string) which is the Go build tag.
//
// Returns string which is the cached private worker executable path.
// Returns error which reports private-worker setup or compilation failure, or nil on
// success.
func BuildWorker(ctx context.Context, repository, workers, name, tag string) (string, error) {
	return buildWorker(ctx, repository, workers, name, tag, "")
}

// BuildCoveredWorker caches an instrumented binary independently of the ordinary helper.
//
// Takes repository (string) which is the repository source directory.
// Takes workers (string) which is the invocation-private worker executable directory.
// Takes name (string) which selects the helper under tests/fuzz/cmd.
// Takes packages (string) which selects Go coverage instrumentation package patterns.
//
// Returns string which is the cached instrumented worker executable path.
// Returns error which reports private-worker setup or compilation failure, or nil on
// success.
func BuildCoveredWorker(ctx context.Context, repository, workers, name, packages string) (string, error) {
	return buildWorker(ctx, repository, workers, name, "", packages)
}

// buildWorker builds an invocation-private worker executable when absent.
//
// Takes repository (string) which is the repository source directory.
// Takes workers (string) which is the invocation-private worker executable directory.
// Takes name (string) which selects the helper under tests/fuzz/cmd.
// Takes tag (string) which is the Go build tag.
// Takes packages (string) which selects Go coverage instrumentation package patterns.
//
// Returns string which is the invocation-private worker executable path.
// Returns error which reports private-worker setup or compilation failure, or nil on
// success.
func buildWorker(ctx context.Context, repository, workers, name, tag, packages string) (string, error) {
	binary := filepath.Join(workers, name)
	if packages != "" {
		binary += "-covered"
	}
	if tag != "" {
		binary += "-" + tag
	}
	if _, err := os.Stat(binary); err == nil {
		return binary, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("worker-build/%s: inspect cached binary: %w", name, err)
	}
	staged := binary + ".tmp"
	arguments := []string{"build", "-buildvcs=false", "-o", staged}
	if packages != "" {
		arguments = append(arguments, "-cover", "-covermode=atomic", "-coverpkg="+packages)
	}
	if tag != "" {
		arguments = append(arguments, "-tags="+tag)
	}
	arguments = append(arguments, "./cmd/"+name)
	output, err := Run(ctx, filepath.Join(repository, "tests", "fuzz"), []string{"GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local"}, "go", arguments...)
	if err != nil {
		return "", errors.Join(fmt.Errorf("worker-build/%s: %w\n%s", name, err, output), removeStaged(staged))
	}
	if err := os.Rename(staged, binary); err != nil {
		return "", errors.Join(fmt.Errorf("worker-build/%s: install binary: %w", name, err), removeStaged(staged))
	}
	return binary, nil
}

// removeStaged deletes a partially built worker binary.
//
// Takes path (string) which is the staged binary path.
//
// Returns error which reports a removal failure other than absence, or nil.
func removeStaged(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove staged worker %s: %w", path, err)
	}
	return nil
}
