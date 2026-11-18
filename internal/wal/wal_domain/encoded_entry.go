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

package wal_domain

// EncodedEntry owns an encoded log record until it is queued or released.
type EncodedEntry struct {
	// release returns data to its pool, or is nil when data is not pooled.
	release func(handle *[]byte, data []byte)

	// handle identifies the pooled buffer behind data, or is nil when not pooled.
	handle *[]byte

	// data holds the encoded record as the log stores it.
	data []byte
}

// NewEncodedEntry wraps an encoded record for DeferredAppender.EnqueueEncoded.
//
// Takes data ([]byte) which is the encoded record.
// Takes handle (*[]byte) which identifies the pooled buffer, or is nil when not pooled.
// Takes release (func(*[]byte, []byte)) which returns data to its pool, or is nil when
// data is not pooled.
//
// Returns EncodedEntry which owns data.
func NewEncodedEntry(data []byte, handle *[]byte, release func(handle *[]byte, data []byte)) EncodedEntry {
	return EncodedEntry{release: release, handle: handle, data: data}
}

// Data returns the entry's encoded record.
//
// The returned slice aliases the entry and is valid until Release.
//
// Returns []byte which contains the encoded record.
func (e *EncodedEntry) Data() []byte {
	return e.data
}

// Release returns the encoding's buffer to its pool and clears the entry, so a second
// call does nothing. It must be called once for an entry that is never queued.
func (e *EncodedEntry) Release() {
	release, handle, data := e.release, e.handle, e.data
	*e = EncodedEntry{}
	if release != nil {
		release(handle, data)
	}
}
