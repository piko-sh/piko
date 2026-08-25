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

package testutil

import (
	"os"
	"strconv"
)

const (
	// toolchainBuildProcs caps GOMAXPROCS for the Go toolchain subprocesses that test
	// harnesses spawn, so a full-suite run does not exhaust machine memory.
	toolchainBuildProcs = 2
)

// ToolchainEnv builds the environment for a Go toolchain subprocess spawned by a test.
//
// Takes extra (...string) which provides "KEY=VALUE" entries appended after the cap, so
// callers can add settings such as "GOWORK=off", or deliberately override the cap by
// setting GOMAXPROCS themselves, since a later entry wins over an earlier one.
//
// Returns []string which holds the current environment, the GOMAXPROCS cap, then extra.
func ToolchainEnv(extra ...string) []string {
	environment := os.Environ()
	environment = append(environment, "GOMAXPROCS="+strconv.Itoa(toolchainBuildProcs))
	return append(environment, extra...)
}
