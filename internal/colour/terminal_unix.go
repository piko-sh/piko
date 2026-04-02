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

//go:build !windows && !js

package colour

import (
	"golang.org/x/sys/unix"
	"piko.sh/piko/wdk/safeconv"
)

// isTerminal reports whether the given file descriptor refers
// to a terminal.
//
// Takes fd (uintptr) which is the file descriptor to check.
//
// Returns bool which is true when fd refers to a terminal.
func isTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(safeconv.Uint64ToInt(uint64(fd)), ioctlGetTermios)
	return err == nil
}

// enableVirtualTerminalProcessing is a no-op on Unix targets.
func enableVirtualTerminalProcessing() {}
