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

//go:build unix

package metric_collector_grpcfb

import (
	"syscall"
)

// fdLimit returns the soft RLIMIT_NOFILE (max open files). Unix-only: the
// syscall.Rlimit/Getrlimit API does not exist on Windows.
//
// Returns uint64 which is the soft open-file limit, zero when unavailable.
// Returns bool which is false when the limit cannot be read.
func fdLimit() (uint64, bool) {
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err != nil {
		return 0, false
	}
	return rl.Cur, true
}
