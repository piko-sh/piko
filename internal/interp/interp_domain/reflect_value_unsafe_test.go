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

//go:build go1.26 && !safe

package interp_domain

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestUnsafeNewAtParity(t *testing.T) {
	type nestedStruct struct {
		Name  string
		Count int
	}
	cases := []struct {
		ty   reflect.Type
		name string
	}{
		{name: "int64", ty: reflect.TypeFor[int64]()},
		{name: "int32", ty: reflect.TypeFor[int32]()},
		{name: "uint64", ty: reflect.TypeFor[uint64]()},
		{name: "float64", ty: reflect.TypeFor[float64]()},
		{name: "bool", ty: reflect.TypeFor[bool]()},
		{name: "string", ty: reflect.TypeFor[string]()},
		{name: "pointer-to-int", ty: reflect.TypeFor[*int]()},
		{name: "slice-of-byte", ty: reflect.TypeFor[[]byte]()},
		{name: "struct", ty: reflect.TypeFor[nestedStruct]()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			backing := reflect.New(tc.ty).Elem()
			p := backing.Addr().UnsafePointer()

			want := reflect.NewAt(tc.ty, p).Elem()

			got := unsafeNewAt(reflectValueABIType(tc.ty), p, tc.ty.Kind())

			if want.Type() != got.Type() {
				t.Fatalf("type mismatch: want=%v got=%v", want.Type(), got.Type())
			}
			if want.Kind() != got.Kind() {
				t.Fatalf("kind mismatch: want=%v got=%v", want.Kind(), got.Kind())
			}
			if want.CanAddr() != got.CanAddr() {
				t.Fatalf("CanAddr mismatch: want=%v got=%v", want.CanAddr(), got.CanAddr())
			}
			if want.CanSet() != got.CanSet() {
				t.Fatalf("CanSet mismatch: want=%v got=%v", want.CanSet(), got.CanSet())
			}

			if want.Addr().UnsafePointer() != got.Addr().UnsafePointer() {
				t.Fatalf("Addr().UnsafePointer mismatch: want=%p got=%p",
					want.Addr().UnsafePointer(), got.Addr().UnsafePointer())
			}

			switch tc.ty.Kind() {
			case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
				got.SetInt(0x4242_4242)
				if want.Int() != 0x4242_4242 {
					t.Fatalf("SetInt via unsafe → safe read: want=%d got=%d", 0x4242_4242, want.Int())
				}
			case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
				got.SetUint(0x4242)
				if want.Uint() != 0x4242 {
					t.Fatalf("SetUint via unsafe → safe read: want=%d got=%d", 0x4242, want.Uint())
				}
			case reflect.Float64, reflect.Float32:
				got.SetFloat(3.14)
				if want.Float() != 3.14 {
					t.Fatalf("SetFloat via unsafe → safe read: want=%v got=%v", 3.14, want.Float())
				}
			case reflect.Bool:
				got.SetBool(true)
				if !want.Bool() {
					t.Fatalf("SetBool via unsafe → safe read: want=true got=%v", want.Bool())
				}
			case reflect.String:
				got.SetString("hello")
				if want.String() != "hello" {
					t.Fatalf("SetString via unsafe → safe read: want=hello got=%q", want.String())
				}
			default:
			}
		})
	}
}

func TestUnsafeReadOnlyValue(t *testing.T) {
	p := unsafe.Pointer(new(int64(42)))
	intType := reflect.TypeFor[int64]()

	got := unsafeReadOnlyValue(reflectValueABIType(intType), p, intType.Kind())
	if got.Type() != intType {
		t.Fatalf("type mismatch: want=%v got=%v", intType, got.Type())
	}
	if got.Int() != 42 {
		t.Fatalf("read mismatch: want=42 got=%d", got.Int())
	}
	if got.CanAddr() {
		t.Fatalf("CanAddr should be false for read-only Value, got true")
	}
	if got.CanSet() {
		t.Fatalf("CanSet should be false for read-only Value, got true")
	}
}

func TestReflectValueABITypeExtraction(t *testing.T) {
	a := reflect.TypeFor[int64]()
	b := reflect.TypeFor[int64]()
	if reflectValueABIType(a) != reflectValueABIType(b) {
		t.Fatalf("ABI type pointers should match for the same Go type")
	}
	if reflectValueABIType(a) == nil {
		t.Fatalf("ABI type pointer should never be nil for a valid reflect.Type")
	}
}
