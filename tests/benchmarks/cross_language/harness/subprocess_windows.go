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

//go:build crosslang && windows

package harness

import "os/exec"

// applyProcessAttributes is a no-op on Windows: process groups work
// differently and we rely on cmd.Process.Kill() via context cancellation.
func applyProcessAttributes(command *exec.Cmd) {
	command.WaitDelay = waitDelayAfterCancel
}

// platformPeakRSSKB has no portable implementation on Windows. Best-effort
// mode: return -1 to signal "unknown". The methodology README documents
// this caveat.
func platformPeakRSSKB(usage any) int64 {
	_ = usage
	return -1
}
