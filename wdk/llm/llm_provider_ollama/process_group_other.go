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

//go:build !unix

package llm_provider_ollama

import "os/exec"

// configureManagedOllamaCommand is a no-op on non-Unix
// platforms.
//
// Takes command (*exec.Cmd) which is the subprocess to
// configure.
func configureManagedOllamaCommand(command *exec.Cmd) {}

// interruptManagedOllamaCommand sends an interrupt signal to
// the managed process.
//
// Takes command (*exec.Cmd) which is the subprocess to
// interrupt.
//
// Returns error if the signal cannot be delivered.
func interruptManagedOllamaCommand(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return nil
	}
	return command.Process.Signal(interruptSignal())
}

// killManagedOllamaCommand forcefully terminates the managed
// process.
//
// Takes command (*exec.Cmd) which is the subprocess to kill.
//
// Returns error if the process cannot be killed.
func killManagedOllamaCommand(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return nil
	}
	return command.Process.Kill()
}
