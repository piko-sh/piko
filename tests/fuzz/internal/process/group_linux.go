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

package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// supervise configures child-process cleanup for this platform.
//
// Takes command (*exec.Cmd) which is the child process command.
func supervise(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,

		Pdeathsig: syscall.SIGKILL,
	}
	command.Cancel = func() error {
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}

// reap kills every process left in the child's process group after the child exits.
//
// Takes command (*exec.Cmd) which is the finished child process command.
//
// Returns error which reports a kill failure other than an already empty group, or nil.
func reap(command *exec.Cmd) error {
	return killGroup(command)
}

// killGroup sends SIGKILL to the child's process group, ignoring an empty group.
//
// Takes command (*exec.Cmd) which is the child process command.
//
// Returns error which reports a kill failure other than an already empty group, or nil.
func killGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}
	err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return fmt.Errorf("kill process group %d: %w", command.Process.Pid, err)
}
