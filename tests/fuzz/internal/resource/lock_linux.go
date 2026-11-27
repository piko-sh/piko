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

//go:build linux

package resource

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	// lockFileName names the per-user campaign lock.
	lockFileName = "piko-fuzz.lock"

	// lockOwnerLimit bounds the owner PID read from a held lock.
	lockOwnerLimit = 32

	// lockDirectoryMode is the permission of the fallback lock directory.
	lockDirectoryMode = 0o700

	// lockFileMode is the permission of the lock file.
	lockFileMode = 0o600

	// groupOrOtherWrite selects write permission for group or others.
	groupOrOtherWrite = 0o022
)

var (
	// errUnsafeLockDirectory reports a lock directory another user could tamper with.
	errUnsafeLockDirectory = errors.New("per-user lock directory ownership or permissions are unsafe")
)

// Lock acquires the per-user campaign lock without waiting.
//
// Returns func() which releases the lock.
// Returns error which reports contention, an unsafe lock directory or a failed lock
// operation.
func Lock() (func(), error) {
	directory, err := openLockDirectory()
	if err != nil {
		return nil, err
	}
	defer func() { _ = directory.Close() }()
	path := filepath.Join(directory.Name(), lockFileName)
	file, err := directory.OpenFile(lockFileName, os.O_RDWR|os.O_CREATE|unix.O_NOFOLLOW, lockFileMode)
	if err != nil {
		return nil, fmt.Errorf("opening lock %s: %w", path, err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		owner, _ := io.ReadAll(io.LimitReader(file, lockOwnerLimit))
		closeErr := file.Close()
		detail := ""
		if pid := strings.TrimSpace(string(owner)); pid != "" && strings.Trim(pid, "0123456789") == "" {
			detail = " (PID " + pid + ")"
		}
		return nil, errors.Join(fmt.Errorf("another Piko fuzz run holds %s%s: %w", path, detail, err), closeErr)
	}
	if err := writeOwner(file); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return func() { _ = file.Close() }, nil
}

// openLockDirectory opens the current user's lock directory.
//
// It uses XDG_RUNTIME_DIR when set, or a private directory under the system temporary
// directory. The directory must be owned by the user and deny group and other writes.
//
// Returns *os.Root which the caller must close.
// Returns error which reports an unusable or unsafe directory.
func openLockDirectory() (*os.Root, error) {
	var directory *os.Root
	var err error
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		directory, err = os.OpenRoot(runtime)
	} else {
		directory, err = openPrivateTempDirectory()
	}
	if err != nil {
		return nil, fmt.Errorf("opening lock directory: %w", err)
	}
	info, err := directory.Stat(".")
	if err != nil {
		return nil, errors.Join(fmt.Errorf("checking lock directory: %w", err), directory.Close())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() || info.Mode().Perm()&groupOrOtherWrite != 0 {
		return nil, errors.Join(fmt.Errorf("%w: %s", errUnsafeLockDirectory, directory.Name()), directory.Close())
	}
	return directory, nil
}

// openPrivateTempDirectory opens or creates the user's temporary lock directory.
//
// Returns *os.Root which the caller must close.
// Returns error which reports a failure to create or open the directory.
func openPrivateTempDirectory() (*os.Root, error) {
	temporary, err := os.OpenRoot(os.TempDir())
	if err != nil {
		return nil, err
	}
	defer func() { _ = temporary.Close() }()
	name := "piko-fuzz-" + strconv.Itoa(os.Getuid())
	if err := temporary.Mkdir(name, lockDirectoryMode); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}
	return temporary.OpenRoot(name)
}

// writeOwner records this process's PID in the held lock file.
//
// Takes file (*os.File) which is the locked file.
//
// Returns error which reports a failed write.
func writeOwner(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("truncating lock: %w", err)
	}
	if _, err := file.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0); err != nil {
		return fmt.Errorf("recording lock owner: %w", err)
	}
	return nil
}
