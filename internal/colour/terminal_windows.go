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

//go:build windows

package colour

import "golang.org/x/sys/windows"

// enableVirtualTerminalProcessingFlag holds the Windows console mode flag for
// virtual terminal sequence processing.
const enableVirtualTerminalProcessingFlag = 0x4

// isTerminal reports whether the given file descriptor refers
// to a console.
//
// Takes fd (uintptr) which is the file descriptor to check.
//
// Returns bool which is true when fd refers to a console.
func isTerminal(fd uintptr) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(fd), &mode) == nil
}

// enableVirtualTerminalProcessing enables ANSI escape sequence processing on
// the Windows console.
func enableVirtualTerminalProcessing() {
	handle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	if windows.GetConsoleMode(handle, &mode) != nil {
		return
	}
	_ = windows.SetConsoleMode(handle, mode|enableVirtualTerminalProcessingFlag)
}
