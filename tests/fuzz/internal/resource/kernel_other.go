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

//go:build !linux

package resource

import (
	"context"
	"errors"
)

var (
	// errUnsupported reports a kernel query on a platform without cgroup v2.
	errUnsupported = errors.New("hard CPU and memory budgets require Linux cgroup v2")
)

// readKernel reports that kernel budgets are unavailable on this platform.
//
// Returns Sample which is always empty.
// Returns error which is always errUnsupported.
func readKernel(_ context.Context, _ kernelLimits, _ bool) (Sample, error) {
	return Sample{}, errUnsupported
}

// OwnCgroup reports that cgroups are unavailable on this platform.
//
// Returns string which is always empty.
// Returns error which is always errUnsupported.
func OwnCgroup() (string, error) { return "", errUnsupported }

// EnclosingLimit reports that cgroups are unavailable on this platform.
//
// Returns string which is always empty.
// Returns error which is always errUnsupported.
func EnclosingLimit() (string, error) { return "", errUnsupported }

// MemInfo reports that /proc/meminfo is unavailable on this platform.
//
// Returns uint64 which is always zero.
// Returns error which is always errUnsupported.
func MemInfo(_ string) (uint64, error) { return 0, errUnsupported }

// Affinity reports that CPU affinity is unavailable on this platform.
//
// Returns []int which is always nil.
// Returns error which is always errUnsupported.
func Affinity() ([]int, error) { return nil, errUnsupported }

// Lock reports that the campaign lock is unavailable on this platform.
//
// Returns func() which is always nil.
// Returns error which is always errUnsupported.
func Lock() (func(), error) { return nil, errUnsupported }

// DiskUsed reports that state-tree measurement is unavailable on this platform.
//
// Returns uint64 which is always zero.
// Returns error which is always errUnsupported.
func DiskUsed(_ context.Context, _ string) (uint64, error) { return 0, errUnsupported }
