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

package telemetry_grpcfb

import (
	"encoding/binary"
	"errors"
	"math"
)

// fieldKind classifies one FlatBuffers field by how the verifier must walk it (scalar,
// string, byte vector, or vector of tables).
type fieldKind int

const (
	// MaxMessageSize bounds one streamed frame. Profile pprof blobs ride inline (in
	// ProfileMeta.blob), so this cap is what keeps a batch of profiles sendable; the marshal
	// path enforces the same ceiling on the sum of inline blobs in a batch.
	MaxMessageSize = 16 << 20

	// maxDepth caps how deeply nested table recursion may run, defeating a malformed frame
	// that tries to force unbounded recursion.
	maxDepth = 16

	// maxVectorLen caps a single table vector's element count before the per-buffer bounds
	// arithmetic, rejecting a hostile huge count up front.
	maxVectorLen = 1 << 20

	// maxTables caps how many tables one frame may contain, bounding the total work an
	// amplification frame can force the verifier to perform.
	maxTables = 1 << 20

	// maxStringLen caps a single string's byte length. Telemetry strings (URLs, stack JSON,
	// etc.) are far smaller; this rejects a hostile huge length up front.
	maxStringLen = 1 << 20

	// maxByteVectorLen caps a single byte vector's length.
	maxByteVectorLen = MaxMessageSize

	// sizeU16 is the on-wire byte width of a little-endian uint16 (vtable slots).
	sizeU16 = 2

	// sizeU32 is the on-wire byte width of a little-endian uint32 (offsets and lengths).
	sizeU32 = 4

	// sizeU64 is the on-wire byte width of a little-endian 64-bit scalar.
	sizeU64 = 8
)

const (
	// kBool marks a one-byte boolean scalar field.
	kBool fieldKind = iota

	// kInt32 marks a four-byte signed integer scalar field.
	kInt32

	// kInt64 marks an eight-byte signed integer scalar field.
	kInt64

	// kFloat64 marks an eight-byte floating-point scalar field.
	kFloat64

	// kString marks a length-prefixed string field.
	kString

	// kVectorTable marks a vector of table offsets the verifier recurses into.
	kVectorTable

	// kVectorByte marks a length-prefixed byte vector field.
	kVectorByte
)

var (
	// errOOB reports that an offset or read fell outside the buffer bounds.
	errOOB = errors.New("telemetry_grpcfb: offset out of bounds")

	// errTooLarge reports that the message exceeds the configured size limit.
	errTooLarge = errors.New("telemetry_grpcfb: message exceeds size limit")

	// errTooDeep reports that table nesting exceeded maxDepth.
	errTooDeep = errors.New("telemetry_grpcfb: nesting too deep")

	// errTooMany reports that the frame contained more tables than maxTables.
	errTooMany = errors.New("telemetry_grpcfb: too many tables")

	// errBadVector reports that a vector length fell outside its allowed bounds.
	errBadVector = errors.New("telemetry_grpcfb: vector length out of bounds")

	// errStringTooLong reports that a string's declared length exceeded maxStringLen.
	errStringTooLong = errors.New("telemetry_grpcfb: string length out of bounds")

	// errShort reports that the buffer was too short to hold a root offset.
	errShort = errors.New("telemetry_grpcfb: buffer too short")
)

// scalarWidth reports the inline byte width of a scalar field kind.
//
// Returns uint64 which is the byte width, or 0 when the kind is not a scalar.
func (k fieldKind) scalarWidth() uint64 {
	switch k {
	case kBool:
		return 1
	case kInt32:
		return sizeU32
	case kInt64, kFloat64:
		return sizeU64
	default:
		return 0
	}
}

// field describes one expected vtable field: its vtable byte-offset (4 + 2*index), its
// kind, and (for vectors of tables) the element table's field spec.
type field struct {
	// elem is the element table's field spec, set only for kVectorTable fields.
	elem []field

	// kind classifies how the verifier must walk the field.
	kind fieldKind

	// voffset is the field's vtable byte-offset (4 + 2*index).
	voffset uint16
}

// verifier holds the buffer under inspection and the running table count for one
// verifyMessage pass.
type verifier struct {
	// buf is the untrusted frame being structurally validated.
	buf []byte

	// tables counts the tables walked so far, bounded by maxTables.
	tables int
}

// table verifies a FlatBuffers table at absolute position pos.
//
// Takes pos (uint64) which is the table's absolute byte position in the buffer.
// Takes fields ([]field) which are the expected field specs for the table.
// Takes depth (int) which is the current recursion depth, bounded by maxDepth.
//
// Returns error which is non-nil when the table is structurally invalid.
func (v *verifier) table(pos uint64, fields []field, depth int) error {
	if depth >= maxDepth {
		return errTooDeep
	}
	v.tables++
	if v.tables > maxTables {
		return errTooMany
	}
	vtPos, vtLen, err := v.vtable(pos)
	if err != nil {
		return err
	}
	for _, f := range fields {
		fieldPos, present, err := v.fieldPos(pos, vtPos, vtLen, f)
		if err != nil {
			return err
		}
		if !present {
			continue
		}
		if err := v.field(fieldPos, f, depth); err != nil {
			return err
		}
	}
	return nil
}

// vtable locates and bounds-checks the vtable of the table at pos, returning the vtable's
// absolute position and length.
//
// Takes pos (uint64) which is the table's absolute byte position in the buffer.
//
// Returns vtPos (uint64) which is the vtable's absolute byte position.
// Returns vtLen (uint64) which is the vtable's byte length.
// Returns err (error) which is non-nil when the vtable is out of bounds.
func (v *verifier) vtable(pos uint64) (vtPos, vtLen uint64, err error) {
	buf := v.buf
	soff, ok := i32(buf, pos)
	if !ok {
		return 0, 0, errOOB
	}

	if pos > math.MaxInt64 {
		return 0, 0, errOOB
	}

	vtPosSigned := int64(pos) - int64(soff)
	if vtPosSigned < 0 || uint64(vtPosSigned) > uint64(len(buf)) {
		return 0, 0, errOOB
	}
	vtPos = uint64(vtPosSigned)
	vtLen16, ok := u16(buf, vtPos)
	if !ok || uint64(vtLen16) < sizeU32 || vtPos+uint64(vtLen16) > uint64(len(buf)) {
		return 0, 0, errOOB
	}
	return vtPos, uint64(vtLen16), nil
}

// fieldPos resolves the absolute position of field f within the table at pos, given the
// table's vtable. present is false when the field is absent or unset.
//
// Takes pos (uint64) which is the table's absolute byte position in the buffer.
// Takes vtPos (uint64) which is the vtable's absolute byte position.
// Takes vtLen (uint64) which is the vtable's byte length.
// Takes f (field) which is the field spec being resolved.
//
// Returns fieldPos (uint64) which is the field's absolute byte position.
// Returns present (bool) which is false when the field is absent or unset.
// Returns err (error) which is non-nil when the field offset is out of bounds.
func (v *verifier) fieldPos(pos, vtPos, vtLen uint64, f field) (fieldPos uint64, present bool, err error) {
	buf := v.buf
	if uint64(f.voffset) >= vtLen {
		return 0, false, nil
	}
	rel, ok := u16(buf, vtPos+uint64(f.voffset))
	if !ok {
		return 0, false, errOOB
	}
	if rel == 0 {
		return 0, false, nil
	}
	fieldPos = pos + uint64(rel)
	if fieldPos > uint64(len(buf)) {
		return 0, false, errOOB
	}
	return fieldPos, true, nil
}

// field bounds-checks one resolved field, dispatching on its kind and recursing into
// table vectors.
//
// Takes fieldPos (uint64) which is the field's absolute byte position in the buffer.
// Takes f (field) which is the field spec being verified.
// Takes depth (int) which is the current recursion depth, bounded by maxDepth.
//
// Returns error which is non-nil when the field is structurally invalid.
func (v *verifier) field(fieldPos uint64, f field, depth int) error {
	switch f.kind {
	case kBool, kInt32, kInt64, kFloat64:
		if fieldPos+f.kind.scalarWidth() > uint64(len(v.buf)) {
			return errOOB
		}
		return nil
	case kString:
		return v.verifyString(fieldPos)
	case kVectorByte:
		return v.verifyByteVector(fieldPos)
	case kVectorTable:
		return v.verifyTableVector(fieldPos, f, depth)
	default:
		return nil
	}
}

// verifyString bounds-checks the length-prefixed string referenced at fieldPos. An
// explicit length cap (matching the table-vector guard) rejects a hostile huge length up
// front, before the buffer-bounds arithmetic.
//
// Takes fieldPos (uint64) which is the field's absolute byte position in the buffer.
//
// Returns error which is errStringTooLong when the declared length exceeds maxStringLen,
// or errOOB when the string falls outside the buffer.
func (v *verifier) verifyString(fieldPos uint64) error {
	buf := v.buf
	rel, ok := u32(buf, fieldPos)
	if !ok {
		return errOOB
	}
	strPos := fieldPos + uint64(rel)
	slen, ok := u32(buf, strPos)
	if !ok {
		return errOOB
	}
	if uint64(slen) > maxStringLen {
		return errStringTooLong
	}
	if strPos+sizeU32+uint64(slen) > uint64(len(buf)) {
		return errOOB
	}
	return nil
}

// verifyByteVector bounds-checks the byte vector referenced at fieldPos. As with strings
// and table vectors, an explicit length cap rejects a hostile huge count before the
// per-buffer arithmetic, keeping all length-prefixed kinds symmetric.
//
// Takes fieldPos (uint64) which is the field's absolute byte position in the buffer.
//
// Returns error which is errBadVector when the declared length exceeds maxByteVectorLen,
// or errOOB when the byte vector falls outside the buffer.
func (v *verifier) verifyByteVector(fieldPos uint64) error {
	buf := v.buf
	rel, ok := u32(buf, fieldPos)
	if !ok {
		return errOOB
	}
	vecPos := fieldPos + uint64(rel)
	count, ok := u32(buf, vecPos)
	if !ok {
		return errOOB
	}
	if uint64(count) > maxByteVectorLen {
		return errBadVector
	}

	if vecPos+sizeU32+uint64(count) > uint64(len(buf)) {
		return errOOB
	}
	return nil
}

// verifyTableVector bounds-checks a vector of table offsets and recurses into each
// referenced element table.
//
// Takes fieldPos (uint64) which is the field's absolute byte position in the buffer.
// Takes f (field) which carries the element table's field spec in f.elem.
// Takes depth (int) which is the current recursion depth, bounded by maxDepth.
//
// Returns error which is non-nil when the vector or any element is invalid.
func (v *verifier) verifyTableVector(fieldPos uint64, f field, depth int) error {
	buf := v.buf
	rel, ok := u32(buf, fieldPos)
	if !ok {
		return errOOB
	}
	vecPos := fieldPos + uint64(rel)
	count, ok := u32(buf, vecPos)
	if !ok {
		return errOOB
	}
	if uint64(count) > maxVectorLen {
		return errBadVector
	}

	if vecPos+sizeU32+uint64(count)*sizeU32 > uint64(len(buf)) {
		return errBadVector
	}
	for i := range uint64(count) {
		elemPos := vecPos + sizeU32 + i*sizeU32
		erel, ok := u32(buf, elemPos)
		if !ok {
			return errOOB
		}
		if err := v.table(elemPos+uint64(erel), f.elem, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// u32 reads a little-endian uint32 at pos after a bounds check.
//
// Takes buf ([]byte) which is the buffer to read from.
// Takes pos (uint64) which is the absolute byte position to read at.
//
// Returns uint32 which is the decoded value, or 0 when out of bounds.
// Returns bool which is false when the read would fall outside the buffer.
func u32(buf []byte, pos uint64) (uint32, bool) {
	if pos+sizeU32 > uint64(len(buf)) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(buf[pos:]), true
}

// u16 reads a little-endian uint16 at pos after a bounds check.
//
// Takes buf ([]byte) which is the buffer to read from.
// Takes pos (uint64) which is the absolute byte position to read at.
//
// Returns uint16 which is the decoded value, or 0 when out of bounds.
// Returns bool which is false when the read would fall outside the buffer.
func u16(buf []byte, pos uint64) (uint16, bool) {
	if pos+sizeU16 > uint64(len(buf)) {
		return 0, false
	}
	return binary.LittleEndian.Uint16(buf[pos:]), true
}

// i32 reads a little-endian signed int32 (a FlatBuffers soffset) at pos.
//
// Takes buf ([]byte) which is the buffer to read from.
// Takes pos (uint64) which is the absolute byte position to read at.
//
// Returns int32 which is the decoded signed value, or 0 when out of bounds.
// Returns bool which is false when the read would fall outside the buffer.
func i32(buf []byte, pos uint64) (int32, bool) {
	v, ok := u32(buf, pos)
	if !ok {
		return 0, false
	}
	//nolint:gosec // G115: same-width int32/uint32 reinterpretation, not an overflow.
	return int32(v), true
}

// verifyMessage validates a root table against fields. It is the only entry point; after
// it returns nil, every read the decoder performs for those fields is safe.
//
// Takes buf ([]byte) which is the untrusted frame to validate.
// Takes fields ([]field) which are the expected field specs for the root table.
//
// Returns error which is non-nil when the frame is structurally invalid.
func verifyMessage(buf []byte, fields []field) error {
	if len(buf) < sizeU32 {
		return errShort
	}
	if len(buf) > MaxMessageSize {
		return errTooLarge
	}
	root, ok := u32(buf, 0)
	if !ok {
		return errOOB
	}
	v := &verifier{buf: buf}
	return v.table(uint64(root), fields, 0)
}
