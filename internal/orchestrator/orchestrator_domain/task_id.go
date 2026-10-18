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

package orchestrator_domain

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// taskIDRandomBatch is how many IDs' worth of random bytes are read from the system at a
	// time.
	taskIDRandomBatch = 256

	// taskIDSize is the number of bytes in a task ID's UUID.
	taskIDSize = len(uuid.UUID{})

	// uuidVersionByte is the index of the byte whose high nibble holds a UUID's version.
	uuidVersionByte = 6

	// uuidVersionMask keeps the low nibble of the version byte.
	uuidVersionMask = 0x0f

	// uuidVersion7 is the high nibble of the version byte of a version 7 UUID.
	uuidVersion7 = 0x70

	// uuidVariantByte is the index of the byte whose high bits hold a UUID's variant.
	uuidVariantByte = 8

	// uuidVariantMask keeps the bits of the variant byte below the variant.
	uuidVariantMask = 0x3f

	// uuidVariantRFC9562 is the variant bits of an RFC 9562 UUID.
	uuidVariantRFC9562 = 0x80
)

var (
	// taskIDSource buffers random bytes for task IDs, which are identifiers rather than
	// secrets.
	taskIDSource struct {
		// buffer holds random bytes not yet used.
		buffer [taskIDSize * taskIDRandomBatch]byte

		// next is the offset of the first unused byte in buffer.
		next int

		// mu guards buffer and next.
		mu sync.Mutex
	}
)

func init() {
	taskIDSource.next = len(taskIDSource.buffer)
}

// NewTaskID returns a version 7 UUID with a leading timestamp.
//
// Returns string which is the ID in its canonical form.
//
// Safe for concurrent use.
func NewTaskID() string {
	var id uuid.UUID
	taskIDSource.mu.Lock()
	if taskIDSource.next == len(taskIDSource.buffer) {
		if _, err := rand.Read(taskIDSource.buffer[:]); err != nil {
			taskIDSource.mu.Unlock()
			return uuid.NewString()
		}
		taskIDSource.next = 0
	}
	copy(id[:], taskIDSource.buffer[taskIDSource.next:taskIDSource.next+taskIDSize])
	taskIDSource.next += taskIDSize
	taskIDSource.mu.Unlock()

	var stamp [8]byte
	binary.BigEndian.PutUint64(stamp[:], uint64(time.Now().UnixMilli()))
	copy(id[0:6], stamp[2:8])
	id[uuidVersionByte] = (id[uuidVersionByte] & uuidVersionMask) | uuidVersion7
	id[uuidVariantByte] = (id[uuidVariantByte] & uuidVariantMask) | uuidVariantRFC9562
	return id.String()
}
