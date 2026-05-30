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

//go:build !safe && !(js && wasm)

package interp_domain

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestLoadIntKindFromUnsafe_AllWidths(t *testing.T) {
	t.Parallel()

	require.Equal(t, int64(-42), loadIntKindFromUnsafe(unsafe.Pointer(new(int(-42))), reflect.Int))

	require.Equal(t, int64(-7), loadIntKindFromUnsafe(unsafe.Pointer(new(int8(-7))), reflect.Int8))

	require.Equal(t, int64(-32000), loadIntKindFromUnsafe(unsafe.Pointer(new(int16(-32000))), reflect.Int16))

	require.Equal(t, int64(-1234567), loadIntKindFromUnsafe(unsafe.Pointer(new(int32(-1234567))), reflect.Int32))

	int64Value := int64(-9876543210)
	require.Equal(t, int64(-9876543210), loadIntKindFromUnsafe(unsafe.Pointer(&int64Value), reflect.Int64))

	require.Equal(t, int64(0), loadIntKindFromUnsafe(unsafe.Pointer(&int64Value), reflect.String),
		"unknown kind should return zero")
}

func TestReadUintAt_AllWidths(t *testing.T) {
	t.Parallel()

	uintValue := uint(42)
	require.Equal(t, uint64(42), readUintAt(unsafe.Pointer(&uintValue), reflect.Uint))

	require.Equal(t, uint64(7), readUintAt(unsafe.Pointer(new(uint8(7))), reflect.Uint8))

	require.Equal(t, uint64(32000), readUintAt(unsafe.Pointer(new(uint16(32000))), reflect.Uint16))

	require.Equal(t, uint64(1234567), readUintAt(unsafe.Pointer(new(uint32(1234567))), reflect.Uint32))

	require.Equal(t, uint64(9876543210), readUintAt(unsafe.Pointer(new(uint64(9876543210))), reflect.Uint64))

	require.Equal(t, uint64(54321), readUintAt(unsafe.Pointer(new(uintptr(54321))), reflect.Uintptr))

	require.Equal(t, uint64(0), readUintAt(unsafe.Pointer(&uintValue), reflect.String),
		"unknown kind should return zero")
}

func TestStoreIntKindAtUnsafe_AllWidths(t *testing.T) {
	t.Parallel()

	var intValue int
	storeIntKindAtUnsafe(unsafe.Pointer(&intValue), reflect.Int, -42)
	require.Equal(t, -42, intValue)

	var int8Value int8
	storeIntKindAtUnsafe(unsafe.Pointer(&int8Value), reflect.Int8, -7)
	require.Equal(t, int8(-7), int8Value)

	var int16Value int16
	storeIntKindAtUnsafe(unsafe.Pointer(&int16Value), reflect.Int16, -32000)
	require.Equal(t, int16(-32000), int16Value)

	var int32Value int32
	storeIntKindAtUnsafe(unsafe.Pointer(&int32Value), reflect.Int32, -1234567)
	require.Equal(t, int32(-1234567), int32Value)

	var int64Value int64
	storeIntKindAtUnsafe(unsafe.Pointer(&int64Value), reflect.Int64, -9876543210)
	require.Equal(t, int64(-9876543210), int64Value)

	original := int64(123)
	int64Value = original
	storeIntKindAtUnsafe(unsafe.Pointer(&int64Value), reflect.String, 999)
	require.Equal(t, original, int64Value, "unknown kind should not modify the destination")
}

func TestStoreUintKindAtUnsafe_AllWidths(t *testing.T) {
	t.Parallel()

	var uintValue uint
	storeUintKindAtUnsafe(unsafe.Pointer(&uintValue), reflect.Uint, 42)
	require.Equal(t, uint(42), uintValue)

	var uint8Value uint8
	storeUintKindAtUnsafe(unsafe.Pointer(&uint8Value), reflect.Uint8, 7)
	require.Equal(t, uint8(7), uint8Value)

	var uint16Value uint16
	storeUintKindAtUnsafe(unsafe.Pointer(&uint16Value), reflect.Uint16, 32000)
	require.Equal(t, uint16(32000), uint16Value)

	var uint32Value uint32
	storeUintKindAtUnsafe(unsafe.Pointer(&uint32Value), reflect.Uint32, 1234567)
	require.Equal(t, uint32(1234567), uint32Value)

	var uint64Value uint64
	storeUintKindAtUnsafe(unsafe.Pointer(&uint64Value), reflect.Uint64, 9876543210)
	require.Equal(t, uint64(9876543210), uint64Value)

	var uintptrValue uintptr
	storeUintKindAtUnsafe(unsafe.Pointer(&uintptrValue), reflect.Uintptr, 54321)
	require.Equal(t, uintptr(54321), uintptrValue)

	original := uint64(123)
	uint64Value = original
	storeUintKindAtUnsafe(unsafe.Pointer(&uint64Value), reflect.String, 999)
	require.Equal(t, original, uint64Value, "unknown kind should not modify the destination")
}

func TestStoreIntKindAtUnsafe_NarrowingTruncates(t *testing.T) {
	t.Parallel()

	var int8Value int8
	storeIntKindAtUnsafe(unsafe.Pointer(&int8Value), reflect.Int8, 257)
	require.Equal(t, int8(1), int8Value,
		"257 narrowed to int8 should wrap modulo 256 to 1")

	var int16Value int16
	storeIntKindAtUnsafe(unsafe.Pointer(&int16Value), reflect.Int16, 65537)
	require.Equal(t, int16(1), int16Value)
}

func TestStoreUintKindAtUnsafe_NarrowingTruncates(t *testing.T) {
	t.Parallel()

	var uint8Value uint8
	storeUintKindAtUnsafe(unsafe.Pointer(&uint8Value), reflect.Uint8, 257)
	require.Equal(t, uint8(1), uint8Value)

	var uint16Value uint16
	storeUintKindAtUnsafe(unsafe.Pointer(&uint16Value), reflect.Uint16, 65537)
	require.Equal(t, uint16(1), uint16Value)
}
