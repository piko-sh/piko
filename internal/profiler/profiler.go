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

package profiler

import (
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// SetRuntimeRates configures the Go runtime block, mutex, and memory
// profiling rates.
//
// Takes config (Config) which provides BlockProfileRate, MutexProfileFraction,
// and MemProfileRate.
func SetRuntimeRates(config Config) {
	runtime.SetBlockProfileRate(config.BlockProfileRate)
	runtime.SetMutexProfileFraction(config.MutexProfileFraction)
	if config.MemProfileRate > 0 {
		runtime.MemProfileRate = config.MemProfileRate
	}
}

// GoroutineCount returns the current number of goroutines.
//
// Returns int which is the goroutine count at the time of the call.
func GoroutineCount() int {
	return runtime.NumGoroutine()
}

// CheckBuildFlags detects whether the binary was built with optimisations
// disabled.
//
// It checks two signals:
//
//  1. debug.ReadBuildInfo() for explicit -gcflags containing -N or -l.
//  2. Whether a debugger (e.g. Delve) is attached as a parent process,
//     since debuggers typically build with -gcflags="all=-N -l".
//
// Profiling such a binary produces results that do not reflect production
// behaviour.
//
// Returns string which contains a warning message if optimisations appear
// disabled, or an empty string if the build is suitable for profiling.
func CheckBuildFlags() string {
	if hasBuildInfoFlags() {
		return "Binary built with optimisations disabled (-gcflags contains -l/-N). " +
			"Profiling results will not reflect production behaviour. " +
			"Rebuild without these flags for accurate profiles."
	}

	if isDebuggerAttached() {
		return "Debugger detected (Delve). The binary was likely built with " +
			"optimisations disabled (-gcflags=\"all=-N -l\"). Profiling results " +
			"will not reflect production behaviour."
	}

	return ""
}

// hasBuildInfoFlags checks debug.ReadBuildInfo for -gcflags containing
// -N or -l, which indicate disabled optimisations.
//
// Returns bool which is true if the build flags suggest optimisations
// are disabled.
func hasBuildInfoFlags() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, s := range info.Settings {
		if s.Key == "-gcflags" && (strings.Contains(s.Value, "-l") || strings.Contains(s.Value, "-N")) {
			return true
		}
	}
	return false
}

// isDebuggerAttached checks whether the parent process is a known debugger
// such as Delve.
//
// Returns bool which is true if a debugger appears to be attached.
func isDebuggerAttached() bool {
	ppid := os.Getppid()
	exe, err := os.Readlink("/proc/" + strconv.Itoa(ppid) + "/exe")
	if err != nil {
		return false
	}
	base := exe
	if i := strings.LastIndex(exe, "/"); i >= 0 {
		base = exe[i+1:]
	}
	return base == "dlv" || base == "dlv-dap" || strings.HasPrefix(base, "dlv")
}
