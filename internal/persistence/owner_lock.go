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

package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// ownerLockFileName is the file in the write-ahead log directory whose exclusive lock
	// marks the process that owns the directory.
	ownerLockFileName = "owner.lock"

	// ownerDirPermissions is the mode the persistence directory is created with.
	ownerDirPermissions = 0o750

	// ownerLockFilePermissions is the mode the lock file is created with, matching the
	// write-ahead log files beside it.
	ownerLockFilePermissions = 0o600
)

var (
	// errOwnerLockHeld is returned when another process holds the owner lock.
	errOwnerLockHeld = errors.New("another process owns the persistence directory")
)

// ownerLock is an exclusive lock on a persistence directory, held by the one process that
// writes to it. The operating system releases it when that process exits, however it
// exits, so a lock is never left behind by a crash.
type ownerLock struct {
	// file is the open lock file, which holds the lock while open.
	file *os.File
}

// acquireOwnerLock takes the owner lock on a persistence directory without waiting, and
// records this process's ID in the lock file for the next process to report.
//
// Takes dir (string) which is the persistence directory.
//
// Returns *ownerLock which holds the lock until released.
// Returns int which is the owning process's ID when another process holds the lock, or
// zero when it is unknown.
// Returns error which wraps errOwnerLockHeld when another process holds the lock.
func acquireOwnerLock(dir string) (*ownerLock, int, error) {
	if err := os.MkdirAll(dir, ownerDirPermissions); err != nil { //nolint:gosec // dir is the operator-configured persistence directory
		return nil, 0, fmt.Errorf("creating persistence directory: %w", err)
	}
	path := filepath.Join(dir, ownerLockFileName)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, ownerLockFilePermissions) //nolint:gosec // fixed file name in the configured persistence directory
	if err != nil {
		return nil, 0, fmt.Errorf("opening owner lock: %w", err)
	}
	if err := lockFile(file); err != nil {
		_ = file.Close()
		if errors.Is(err, errOwnerLockHeld) {
			return nil, readOwnerPID(path), err
		}
		return nil, 0, fmt.Errorf("locking %s: %w", path, err)
	}
	if err := file.Truncate(0); err == nil {
		_, _ = file.WriteAt([]byte(fmt.Sprintf("%d\n%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))), 0)
	}
	return &ownerLock{file: file}, 0, nil
}

// release gives up the owner lock.
//
// Returns error when the lock file cannot be closed.
func (l *ownerLock) release() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = unlockFile(l.file)
	err := l.file.Close()
	l.file = nil
	return err
}

// readOwnerPID reads the owning process's ID from a lock file.
//
// Takes path (string) which is the lock file.
//
// Returns int which is the process ID, or zero when the file cannot be read.
func readOwnerPID(path string) int {
	data, err := os.ReadFile(path) //nolint:gosec // path is the owner lock file acquireOwnerLock opened
	if err != nil {
		return 0
	}
	first, _, _ := strings.Cut(string(data), "\n")
	pid, err := strconv.Atoi(strings.TrimSpace(first))
	if err != nil {
		return 0
	}
	return pid
}
