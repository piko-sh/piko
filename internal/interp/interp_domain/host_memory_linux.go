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

//go:build linux

package interp_domain

import (
	"os"
	"strconv"
	"strings"
)

const (
	// hostMemInfoPath is the kernel-exported file that exposes total physical RAM on Linux.
	// The MemTotal line is reported in kibibytes.
	hostMemInfoPath = "/proc/meminfo"

	// hostMemInfoKeyPrefix is the line prefix in /proc/meminfo carrying the total
	// physical-memory value.
	hostMemInfoKeyPrefix = "MemTotal:"

	// hostMemInfoUnitKiB names the suffix /proc/meminfo writes after the total memory value.
	hostMemInfoUnitKiB = "kB"

	// bytesPerKiB is the multiplier to convert /proc/meminfo's kibibyte figures into bytes.
	bytesPerKiB uint64 = 1024

	// hostMemInfoExpectedFieldCount is the minimum number of whitespace- separated tokens a
	// well-formed MemTotal: line must have ("MemTotal:", "<number>", "kB").
	hostMemInfoExpectedFieldCount = 3
)

// detectHostMemoryBytes parses /proc/meminfo and returns the host's total physical memory
// in bytes. A return of zero indicates that the value could not be determined (file
// missing, format unexpected, numeric overflow); the caller should fall back to a static
// default.
//
// Returns total host RAM in bytes, or 0 on any detection failure.
func detectHostMemoryBytes() uint64 {
	data, err := os.ReadFile(hostMemInfoPath)
	if err != nil {
		return 0
	}
	for line := range strings.Lines(string(data)) {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, hostMemInfoKeyPrefix) {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < hostMemInfoExpectedFieldCount || fields[2] != hostMemInfoUnitKiB {
			return 0
		}
		kib, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			return 0
		}
		if kib > ^uint64(0)/bytesPerKiB {
			return 0
		}
		return kib * bytesPerKiB
	}
	return 0
}
