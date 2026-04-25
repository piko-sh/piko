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

package monitoring_domain

import (
	"io"
	"strconv"
	"strings"

	"piko.sh/piko/wdk/safedisk"
)

const (
	// cgroupFileReadLimit is the maximum number of bytes read from a cgroup
	// pseudo-file to prevent unexpected memory allocation from malicious
	// mounts.
	cgroupFileReadLimit = 64

	// cgroupBasePath is the root of the cgroup filesystem used for sandboxed
	// reads.
	cgroupBasePath = "/sys/fs/cgroup"
)

// readCgroupMemoryLimit reads the container memory limit from the cgroup
// filesystem. It tries cgroup v2 first, then falls back to cgroup v1.
//
// Returns uint64 which is the memory limit in bytes, or 0 when the limit
// cannot be determined (non-Linux, no cgroup, or unlimited).
func readCgroupMemoryLimit() uint64 {
	sandbox, err := safedisk.NewSandbox(cgroupBasePath, safedisk.ModeReadOnly)
	if err != nil {
		return 0
	}
	defer func() { _ = sandbox.Close() }()

	if limit := readCgroupUint64(sandbox, "memory.max"); limit > 0 {
		return limit
	}

	return readCgroupUint64(sandbox, "memory/memory.limit_in_bytes")
}

// readCgroupMemoryCurrent reads the container's current memory usage from the
// cgroup filesystem. It tries cgroup v2 first, then falls back to cgroup v1.
//
// Returns uint64 which is the current memory usage in bytes, or 0 when the
// value cannot be determined.
func readCgroupMemoryCurrent() uint64 {
	sandbox, err := safedisk.NewSandbox(cgroupBasePath, safedisk.ModeReadOnly)
	if err != nil {
		return 0
	}
	defer func() { _ = sandbox.Close() }()

	if current := readCgroupUint64(sandbox, "memory.current"); current > 0 {
		return current
	}

	return readCgroupUint64(sandbox, "memory/memory.usage_in_bytes")
}

// readCgroupUint64 reads a uint64 value from a cgroup pseudo-file within the
// provided sandbox. The read is capped at cgroupFileReadLimit bytes and
// yields 0 on any error or when the value is "max" (unlimited).
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed filesystem access
// rooted at the cgroup directory.
// Takes name (string) which is the relative path within the cgroup directory.
//
// Returns uint64 which is the parsed value, or 0 on error.
func readCgroupUint64(sandbox safedisk.Sandbox, name string) uint64 {
	file, err := sandbox.Open(name)
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, cgroupFileReadLimit))
	if err != nil {
		return 0
	}

	content := strings.TrimSpace(string(data))
	if content == "" || content == "max" {
		return 0
	}

	value, err := strconv.ParseUint(content, 10, 64)
	if err != nil {
		return 0
	}

	return value
}
