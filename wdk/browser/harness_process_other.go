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

package browser

import (
	"os/exec"
)

// configureServerProcessGroup leaves the server command unchanged on platforms without
// Unix process groups.
//
// Takes command (*exec.Cmd) which is the server command to configure.
func configureServerProcessGroup(_ *exec.Cmd) {}

// killServerProcessTree kills the server process.
//
// Takes command (*exec.Cmd) which is the started server command.
//
// Returns error when the process cannot be killed.
func killServerProcessTree(command *exec.Cmd) error {
	return command.Process.Kill()
}
