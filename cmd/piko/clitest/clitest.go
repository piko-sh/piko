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

package clitest

import (
	"io"

	"piko.sh/piko/cmd/piko/internal/cli"
)

// RunCommandWithIO dispatches a CLI subcommand with explicit IO writers.
// This is a thin wrapper around the internal implementation, exposed solely
// for integration testing.
//
// Takes subcommand (string) which identifies the command to run.
// Takes arguments ([]string) which contains the remaining arguments after the
// subcommand.
// Takes stdout (io.Writer) which receives standard output.
// Takes stderr (io.Writer) which receives error output.
//
// Returns int which is the exit code: 0 for success, 1 for errors.
func RunCommandWithIO(subcommand string, arguments []string, stdout, stderr io.Writer) int {
	return cli.RunCommandWithIO(subcommand, arguments, stdout, stderr)
}
