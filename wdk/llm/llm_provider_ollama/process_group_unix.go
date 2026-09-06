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
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// configureManagedOllamaCommand sets up a dedicated process group for the managed Ollama
// subprocess on Unix.
//
// Takes command (*exec.Cmd) which is the subprocess to configure.
func configureManagedOllamaCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// interruptManagedOllamaCommand sends SIGINT to the managed process group.
//
// Takes command (*exec.Cmd) which is the subprocess to interrupt.
//
// Returns error if the signal cannot be delivered, wrapping os.ErrProcessDone when the
// process group has already gone.
func interruptManagedOllamaCommand(command *exec.Cmd) error {
	return signalManagedOllamaGroup(command, syscall.SIGINT, os.Interrupt)
}

// killManagedOllamaCommand sends SIGKILL to the managed process group.
//
// Takes command (*exec.Cmd) which is the subprocess to kill.
//
// Returns error if the process cannot be killed, wrapping os.ErrProcessDone when the
// process group has already gone.
func killManagedOllamaCommand(command *exec.Cmd) error {
	return signalManagedOllamaGroup(command, syscall.SIGKILL, os.Kill)
}

// signalManagedOllamaGroup delivers a signal to the managed process group, falling back
// to the process alone when its group cannot be resolved.
//
// Takes command (*exec.Cmd) which is the subprocess to signal.
// Takes groupSignal (syscall.Signal) which is sent to the process group.
// Takes processSignal (os.Signal) which is sent to the process when the group is unknown.
//
// Returns error if the signal cannot be delivered, wrapping os.ErrProcessDone when the
// process has already gone.
func signalManagedOllamaGroup(command *exec.Cmd, groupSignal syscall.Signal, processSignal os.Signal) error {
	if command == nil || command.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(command.Process.Pid)
	if err != nil {
		return command.Process.Signal(processSignal)
	}
	if err := syscall.Kill(-pgid, groupSignal); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return errors.Join(os.ErrProcessDone, err)
		}
		return err
	}
	return nil
}
