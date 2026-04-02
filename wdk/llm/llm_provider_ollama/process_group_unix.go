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

package llm_provider_ollama

import (
	"os"
	"os/exec"
	"syscall"
)

// configureManagedOllamaCommand sets up a dedicated process
// group for the managed Ollama subprocess on Unix.
//
// Takes command (*exec.Cmd) which is the subprocess to
// configure.
func configureManagedOllamaCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// interruptManagedOllamaCommand sends SIGINT to the managed
// process group.
//
// Takes command (*exec.Cmd) which is the subprocess to
// interrupt.
//
// Returns error if the signal cannot be delivered.
func interruptManagedOllamaCommand(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(command.Process.Pid)
	if err != nil {
		return command.Process.Signal(os.Interrupt)
	}
	return syscall.Kill(-pgid, syscall.SIGINT)
}

// killManagedOllamaCommand sends SIGKILL to the managed
// process group.
//
// Takes command (*exec.Cmd) which is the subprocess to kill.
//
// Returns error if the process cannot be killed.
func killManagedOllamaCommand(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(command.Process.Pid)
	if err != nil {
		return command.Process.Kill()
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
