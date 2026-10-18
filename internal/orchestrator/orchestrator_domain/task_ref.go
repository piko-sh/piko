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
	"encoding/binary"
	"errors"

	"piko.sh/safeconv"
)

// TaskRefFlags carries the properties of a referenced task a transport needs without
// reading the store.
type TaskRefFlags uint8

const (
	// TaskRefBackground marks a task no build waits for.
	TaskRefBackground TaskRefFlags = 1 << iota
)

const (
	// taskRefVersion is the first byte of an encoded TaskRef.
	taskRefVersion byte = 1

	// MaxTaskRefSize is the encoded size of a UUID task reference with an attempt below 128.
	MaxTaskRefSize = 43

	// taskRefHeaderSize counts the fixed version, priority, and flag bytes.
	taskRefHeaderSize = 3

	// minTaskRefSize includes the header and the minimum attempt and length encodings.
	minTaskRefSize = taskRefHeaderSize + 2
)

var (
	// errTaskRefMalformed is returned when bytes do not decode to a TaskRef.
	errTaskRefMalformed = errors.New("orchestrator: malformed task reference")
)

// TaskRef identifies a task delivery by its ID, attempt, and worker priority.
//
// The store decides whether a delivered attempt can still be claimed.
type TaskRef struct {
	// ID identifies the task.
	ID string

	// Attempt is the attempt the delivery may claim.
	Attempt int

	// Priority selects the workers that run the task.
	Priority TaskPriority

	// Flags carries the task's properties.
	Flags TaskRefFlags
}

// RefOf returns the reference that delivers a task's current attempt.
//
// Takes task (*Task) which is the task.
//
// Returns TaskRef which references it.
func RefOf(task *Task) TaskRef {
	ref := TaskRef{ID: task.ID, Attempt: task.Attempt, Priority: task.Config.Priority, Flags: 0}
	if task.Background {
		ref.Flags |= TaskRefBackground
	}
	return ref
}

// AppendTaskRef appends a reference's encoding to dst.
//
// Takes dst ([]byte) which receives the encoding.
// Takes ref (TaskRef) which is the reference.
//
// Returns []byte which is dst extended with the encoding.
func AppendTaskRef(dst []byte, ref TaskRef) []byte {
	priority := safeconv.Uint64ToByteTruncate(safeconv.Int64ToUint64Reinterpret(int64(ref.Priority)))
	dst = append(dst, taskRefVersion, priority, byte(ref.Flags))
	dst = binary.AppendUvarint(dst, uint64(max(ref.Attempt, 0)))
	dst = binary.AppendUvarint(dst, uint64(len(ref.ID)))
	return append(dst, ref.ID...)
}

// DecodeTaskRef decodes a reference encoded by AppendTaskRef.
//
// Takes data ([]byte) which is the encoding.
//
// Returns TaskRef which is the reference; its ID is a copy, so data may be reused.
// Returns error when data is not a reference.
func DecodeTaskRef(data []byte) (TaskRef, error) {
	if len(data) < minTaskRefSize || data[0] != taskRefVersion {
		return TaskRef{}, errTaskRefMalformed
	}
	ref := TaskRef{ID: "", Attempt: 0, Priority: TaskPriority(data[1]), Flags: TaskRefFlags(data[2])}
	rest := data[taskRefHeaderSize:]
	attempt, n := binary.Uvarint(rest)
	if n <= 0 {
		return TaskRef{}, errTaskRefMalformed
	}
	rest = rest[n:]
	length, n := binary.Uvarint(rest)
	if n <= 0 || length != safeconv.IntToUint64(len(rest)-n) || length == 0 {
		return TaskRef{}, errTaskRefMalformed
	}
	ref.Attempt = safeconv.Uint64ToInt(attempt)
	ref.ID = string(rest[n:])
	return ref, nil
}
