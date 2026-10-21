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

package provider_disk

import (
	"golang.org/x/sys/unix"
)

const (
	// deferredDurabilitySupported reports that Flush can sync a whole filesystem.
	deferredDurabilitySupported = true
)

// syncFilesystem syncs the filesystem holding the open file fd, in one call.
//
// Takes fd (uintptr) which is an open file on the filesystem.
//
// Returns error when the sync fails.
func syncFilesystem(fd uintptr) error {
	return unix.Syncfs(int(fd))
}
