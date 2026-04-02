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

//revive:disable:add-constant

package driven_disk

// Provides object pooling for byte slices and tag slices to reduce allocations
// during WAL encode/decode operations. Implements size-specific pools with
// automatic capacity selection based on required size.

import (
	"sync"
)

//nolint:revive // pool thresholds

// defaultTagCapacity is the default capacity for tag slices in the pool.
const defaultTagCapacity = 16

var (
	bufPool64 = sync.Pool{New: func() any { return new(make([]byte, 0, 64)) }}

	bufPool128 = sync.Pool{New: func() any { return new(make([]byte, 0, 128)) }}

	bufPool256 = sync.Pool{New: func() any { return new(make([]byte, 0, 256)) }}

	bufPool512 = sync.Pool{New: func() any { return new(make([]byte, 0, 512)) }}

	bufPool1K = sync.Pool{New: func() any { return new(make([]byte, 0, 1024)) }}

	bufPool2K = sync.Pool{New: func() any { return new(make([]byte, 0, 2048)) }}

	bufPool4K = sync.Pool{New: func() any { return new(make([]byte, 0, 4096)) }}

	bufPool8K = sync.Pool{New: func() any { return new(make([]byte, 0, 8192)) }}

	bufPool16K = sync.Pool{New: func() any { return new(make([]byte, 0, 16384)) }}

	bufPool32K = sync.Pool{New: func() any { return new(make([]byte, 0, 32768)) }}

	bufPool64K = sync.Pool{New: func() any { return new(make([]byte, 0, 65536)) }}

	bufPool128K = sync.Pool{New: func() any { return new(make([]byte, 0, 131072)) }}

	tagSlicePool = sync.Pool{
		New: func() any {
			return new(make([]string, 0, defaultTagCapacity))
		},
	}
)

// GetByteBuffer retrieves a pooled byte slice with at least the requested
// capacity. The capacity rounds up to the nearest bucket size (64, 128, 256,
// 512, 1K, 2K, 4K, 8K, 16K, 32K, 64K, 128K) and falls back to make() for
// capacities above 128K.
//
// Takes capacity (int) which specifies the minimum slice capacity needed.
//
// Returns *[]byte which is the pool pointer for use with PutByteBuffer,
// or nil for non-poolable capacities (above 128K) or zero/negative capacity.
// Returns []byte which is the slice ready for use, already sized to 0 length.
func GetByteBuffer(capacity int) (*[]byte, []byte) {
	if capacity <= 0 {
		return nil, nil
	}
	var ptr *[]byte
	var ok bool
	switch {
	case capacity <= 64:
		ptr, ok = bufPool64.Get().(*[]byte)
	case capacity <= 128:
		ptr, ok = bufPool128.Get().(*[]byte)
	case capacity <= 256:
		ptr, ok = bufPool256.Get().(*[]byte)
	case capacity <= 512:
		ptr, ok = bufPool512.Get().(*[]byte)
	case capacity <= 1024:
		ptr, ok = bufPool1K.Get().(*[]byte)
	case capacity <= 2048:
		ptr, ok = bufPool2K.Get().(*[]byte)
	case capacity <= 4096:
		ptr, ok = bufPool4K.Get().(*[]byte)
	case capacity <= 8192:
		ptr, ok = bufPool8K.Get().(*[]byte)
	case capacity <= 16384:
		ptr, ok = bufPool16K.Get().(*[]byte)
	case capacity <= 32768:
		ptr, ok = bufPool32K.Get().(*[]byte)
	case capacity <= 65536:
		ptr, ok = bufPool64K.Get().(*[]byte)
	case capacity <= 131072:
		ptr, ok = bufPool128K.Get().(*[]byte)
	default:
		return nil, make([]byte, 0, capacity)
	}
	if !ok {
		return nil, make([]byte, 0, capacity)
	}
	*ptr = (*ptr)[:0]
	return ptr, *ptr
}

// PutByteBuffer returns a byte slice to the pool using its pointer. The slice
// is synced back to the pointer in case append reallocated it.
//
// When ptr is nil, this is a no-op for non-poolable slices.
//
// Takes ptr (*[]byte) which is the pool pointer from GetByteBuffer.
// Takes b ([]byte) which is the current slice, which may have grown via append.
func PutByteBuffer(ptr *[]byte, b []byte) {
	if ptr == nil {
		return
	}
	*ptr = b
	c := cap(*ptr)
	switch c {
	case 64:
		bufPool64.Put(ptr)
	case 128:
		bufPool128.Put(ptr)
	case 256:
		bufPool256.Put(ptr)
	case 512:
		bufPool512.Put(ptr)
	case 1024:
		bufPool1K.Put(ptr)
	case 2048:
		bufPool2K.Put(ptr)
	case 4096:
		bufPool4K.Put(ptr)
	case 8192:
		bufPool8K.Put(ptr)
	case 16384:
		bufPool16K.Put(ptr)
	case 32768:
		bufPool32K.Put(ptr)
	case 65536:
		bufPool64K.Put(ptr)
	case 131072:
		bufPool128K.Put(ptr)
	}
}

// ResetBytePools resets all byte slice pools to their initial state,
// ensuring tests run in isolation from one another.
func ResetBytePools() {
	bufPool64 = sync.Pool{New: func() any { return new(make([]byte, 0, 64)) }}
	bufPool128 = sync.Pool{New: func() any { return new(make([]byte, 0, 128)) }}
	bufPool256 = sync.Pool{New: func() any { return new(make([]byte, 0, 256)) }}
	bufPool512 = sync.Pool{New: func() any { return new(make([]byte, 0, 512)) }}
	bufPool1K = sync.Pool{New: func() any { return new(make([]byte, 0, 1024)) }}
	bufPool2K = sync.Pool{New: func() any { return new(make([]byte, 0, 2048)) }}
	bufPool4K = sync.Pool{New: func() any { return new(make([]byte, 0, 4096)) }}
	bufPool8K = sync.Pool{New: func() any { return new(make([]byte, 0, 8192)) }}
	bufPool16K = sync.Pool{New: func() any { return new(make([]byte, 0, 16384)) }}
	bufPool32K = sync.Pool{New: func() any { return new(make([]byte, 0, 32768)) }}
	bufPool64K = sync.Pool{New: func() any { return new(make([]byte, 0, 65536)) }}
	bufPool128K = sync.Pool{New: func() any { return new(make([]byte, 0, 131072)) }}
}

// GetTagSlice retrieves a pooled string slice for tags.
//
// Returns *[]string which is the pool pointer for use with PutTagSlice.
// Returns []string which is the slice ready for use, already sized to 0 length.
func GetTagSlice() (*[]string, []string) {
	ptr, ok := tagSlicePool.Get().(*[]string)
	if !ok {
		s := make([]string, 0, defaultTagCapacity)
		return &s, s
	}
	*ptr = (*ptr)[:0]
	return ptr, *ptr
}

// PutTagSlice returns a tag slice to the pool.
// If ptr is nil, this is a no-op.
//
// Takes ptr (*[]string) which is the pool pointer from GetTagSlice.
// Takes s ([]string) which is the current slice (may have grown via append).
func PutTagSlice(ptr *[]string, s []string) {
	if ptr == nil {
		return
	}
	*ptr = s
	tagSlicePool.Put(ptr)
}
