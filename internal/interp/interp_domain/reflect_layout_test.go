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

//go:build go1.26

package interp_domain

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestReflectValueLayout(t *testing.T) {
	const (
		wantSize   = 24
		wantTypOff = 0
		wantPtrOff = unsafe.Sizeof(uintptr(0))
	)
	if got := unsafe.Sizeof(reflect.Value{}); got != wantSize {
		t.Fatalf("reflect.Value size = %d, want %d "+
			"(Go reflect layout changed; reflect_value_unsafe.go assumptions are stale)",
			got, wantSize)
	}

	const sentinel = int64(0x0123_4567_89AB_CDEF)
	heapValue := sentinel
	v := reflect.ValueOf(&heapValue).Elem()

	var view struct {
		typ  unsafe.Pointer
		ptr  unsafe.Pointer
		flag uintptr
	} = *(*struct {
		typ  unsafe.Pointer
		ptr  unsafe.Pointer
		flag uintptr
	})(unsafe.Pointer(&v))

	if view.typ == nil {
		t.Fatalf("reflect.Value.typ_ = nil at offset %d, want non-nil "+
			"(layout changed: word 0 is no longer the *abi.Type)", wantTypOff)
	}
	if view.ptr == nil {
		t.Fatalf("reflect.Value.ptr = nil at offset %d, want non-nil "+
			"(layout changed: word 1 is no longer the data pointer)", wantPtrOff)
	}

	const flagKindMask = uintptr(0x1f)
	const flagAddr = uintptr(1 << 8)
	const flagIndir = uintptr(1 << 7)
	gotKind := reflect.Kind(view.flag & flagKindMask)
	if gotKind != reflect.Int64 {
		t.Fatalf("reflect.Value.flag low bits = %v (Kind), want Int64 "+
			"(flag layout changed; reflect_value_unsafe.go constants are stale)",
			gotKind)
	}

	if view.flag&flagAddr == 0 {
		t.Fatalf("flagAddr bit missing on .Elem() Value (flag=%#x); "+
			"flagAddr offset may have moved", view.flag)
	}
	if view.flag&flagIndir == 0 {
		t.Fatalf("flagIndir bit missing on .Elem() Value (flag=%#x); "+
			"flagIndir offset may have moved", view.flag)
	}

	v.SetInt(0x5555_5555_5555_5555)
	if heapValue != 0x5555_5555_5555_5555 {
		t.Fatalf("Value.SetInt() did not update backing storage (got %#x); "+
			"the ptr field at offset %d may not point at the int value",
			heapValue, wantPtrOff)
	}
}
