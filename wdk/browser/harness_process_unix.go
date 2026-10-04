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

//go:build unix

package browser

import (
	"os/exec"
	"syscall"
)

// configureServerProcessGroup starts the server in its own process group so it and its
// children can be stopped together.
//
// Takes command (*exec.Cmd) which is the server command to configure.
func configureServerProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killServerProcessTree kills the server's process group, falling back to the server
// process alone when its group cannot be resolved.
//
// Takes command (*exec.Cmd) which is the started server command.
//
// Returns error when the kill signal cannot be delivered.
func killServerProcessTree(command *exec.Cmd) error {
	pgid, err := syscall.Getpgid(command.Process.Pid)
	if err != nil {
		return command.Process.Kill()
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
