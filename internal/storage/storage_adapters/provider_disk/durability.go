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

package provider_disk

import (
	"context"
	"fmt"
	"os"
)

// Durability selects when the disk provider makes writes durable.
type Durability int

const (
	// DurabilityImmediate syncs each object and its directory before Put returns and is the
	// default.
	DurabilityImmediate Durability = iota

	// DurabilityDeferred postpones filesystem synchronisation until Flush.
	//
	// Writes remain atomic, but a power or kernel failure before Flush may lose them.
	// Unsupported platforms use DurabilityImmediate.
	DurabilityDeferred
)

// Flush makes every object written so far durable. Under DurabilityImmediate there is
// nothing left to do.
//
// Returns error when the filesystem cannot be synced.
func (d *DiskProvider) Flush(_ context.Context) error {
	if d.durability != DurabilityDeferred {
		return nil
	}
	root, err := d.sandbox.OpenFile(".", os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("opening the storage directory to sync it: %w", err)
	}
	defer func() { _ = root.Close() }()
	if err := syncFilesystem(root.Fd()); err != nil {
		return fmt.Errorf("syncing the storage filesystem: %w", err)
	}
	return nil
}
