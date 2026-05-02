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

//go:build crosslang && (linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package harness

import (
	"os/exec"
	"syscall"
)

// applyProcessAttributes configures the command with a fresh process group so we can
// SIGKILL the whole group on context cancellation. Matches the pattern used at
// internal/interp/interp_test/snippets/parity_test.go.
func applyProcessAttributes(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.WaitDelay = waitDelayAfterCancel
}

// platformPeakRSSKB extracts the peak resident-set-size from the syscall rusage payload.
// Linux reports KiB directly; Darwin reports bytes.
func platformPeakRSSKB(usage any) int64 {
	rusage, ok := usage.(*syscall.Rusage)
	if !ok || rusage == nil {
		return 0
	}
	maximum := int64(rusage.Maxrss)
	if platformOSIsDarwin() {
		return maximum / dividerForDarwinRSS
	}
	return maximum
}
