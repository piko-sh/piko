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

//go:build netbsd

package provider_disk

import (
	"cmp"
	"fmt"

	"golang.org/x/sys/unix"

	"piko.sh/piko/wdk/safeconv"
)

// getDiskSpace returns available and total disk space in megabytes for the given path.
//
// NetBSD has no statfs; statvfs reports block counts in units of the fragment size
// (Frsize), falling back to the block size when the fragment size is not set.
//
// Takes rootPath (string) which specifies the filesystem path to check.
//
// Returns availableMB (uint64) which is the available space in megabytes.
// Returns totalMB (uint64) which is the total space in megabytes.
// Returns error when the filesystem cannot be queried.
func getDiskSpace(rootPath string) (availableMB, totalMB uint64, err error) {
	var stat unix.Statvfs_t
	if err := unix.Statvfs(rootPath, &stat); err != nil {
		return 0, 0, fmt.Errorf("querying disk space for %q: %w", rootPath, err)
	}
	blockSize := cmp.Or(safeconv.ToUint64(stat.Frsize), safeconv.ToUint64(stat.Bsize))
	availableMB = (stat.Bavail * blockSize) / bytesPerMegabyte
	totalMB = (stat.Blocks * blockSize) / bytesPerMegabyte
	return availableMB, totalMB, nil
}
