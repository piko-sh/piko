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

package ast_domain

// Provides zero-allocation rendering for dynamic content by composing writer
// parts that map to quicktemplate's direct write methods. Supports lazy
// evaluation, type-specific rendering paths for strings, numbers, and booleans,
// and FNV hashing for key generation.

import (
	"fmt"
	"strconv"
	"sync"
)

// WriterPartType discriminates between part types in a DirectWriter. Each type
// maps to a zero-allocation render path using quicktemplate's direct write
// methods.
type WriterPartType uint8

const (
	// WriterPartString is a string value that is written without extra memory use.
	WriterPartString WriterPartType = iota

	// WriterPartInt represents a signed integer (int, int8, int16, int32, int64).
	// Renders via qw.N().DL() using strconv.AppendInt - zero allocation.
	WriterPartInt

	// WriterPartUint represents an unsigned integer (uint, uint8, uint16, uint32,
	// uint64, uintptr). Renders via qw.N().DUL() using strconv.AppendUint with
	// zero allocation.
	WriterPartUint

	// WriterPartFloat represents a floating-point number (float32, float64).
	// Renders via qw.N().F() using strconv.AppendFloat - zero allocation.
	WriterPartFloat

	// WriterPartBool represents a boolean value.
	// Renders via qw.N().S("true") or qw.N().S("false") - zero allocation.
	WriterPartBool

	// WriterPartAny marks a part containing an arbitrary value that is formatted
	// via Stringer or fmt.Sprint fallback. This is the slow path used when the
	// type is unknown at generator time, and may allocate during formatting.
	WriterPartAny

	// WriterPartEscapeString represents a string that needs HTML escaping at
	// render time.
	//
	// Renders via qw.E().S() using quicktemplate's htmlEscapeWriter with zero
	// allocation. Use for user-provided strings that may contain HTML special
	// characters.
	WriterPartEscapeString

	// WriterPartFNVString represents a dynamic string that should be
	// FNV-32 hashed, rendering as an 8-character hex string for p-key
	// values with problematic characters or unpredictable length.
	WriterPartFNVString

	// WriterPartFNVFloat represents a float value that should be FNV-32 hashed and
	// rendered as an 8-character hex string. This avoids float precision issues in
	// p-key values (e.g., 0.1 + 0.2 = 0.30000000000000004).
	WriterPartFNVFloat

	// WriterPartFNVAny represents an arbitrary value that is FNV-32 hashed and
	// rendered as an 8-character hex string. It is used for p-key values with
	// unknown types at generation time.
	WriterPartFNVAny

	// WriterPartBytes represents a raw byte slice that renders directly via
	// qw.N().SZ() without string conversion. Used for pre-encoded content like
	// base64-encoded action payloads where the bytes are already computed.
	WriterPartBytes

	// WriterPartEscapeBytes represents a byte slice that needs HTML escaping
	// at render time. Used for style values from *Bytes helpers where the
	// content may contain user input with HTML special characters.
	WriterPartEscapeBytes
)

const (
	// directWriterPartsCapacity is the size of the fixed array for common cases.
	// This avoids heap memory use for most calls.
	directWriterPartsCapacity = 8

	// decimalBase is the base for decimal number conversion.
	decimalBase = 10

	// float64BitSize is the bit size for float64 values passed to
	// strconv.AppendFloat.
	float64BitSize = 64

	// defaultByteBufCapacity is the default capacity for pooled byte buffers.
	// 512 bytes accommodates larger CSS class strings and style values without
	// reallocation.
	defaultByteBufCapacity = 512

	// defaultStringBufCapacity is the default capacity for pooled string
	// buffers (64 bytes), sufficient for most String() computations
	// whilst keeping memory overhead low.
	defaultStringBufCapacity = 64

	// estimatedFloatLen is the approximate rendered length for floats and unknown
	// types. Floats are variable length; this is a reasonable upper-bound estimate.
	estimatedFloatLen = 16

	// trueLiteralLen is the length of the string "true" in bytes.
	trueLiteralLen = 4

	// falseLiteralLen is the length of the string "false".
	falseLiteralLen = 5

	// fnvHexLen is the length of an FNV-32 hash when shown in hexadecimal.
	fnvHexLen = 8
)

// WriterPart represents one segment of a DirectWriter.
// Only one value field is used based on Type.
type WriterPart struct {
	// AnyValue holds a value of any type for generic output.
	AnyValue any

	// StringValue holds the text content for string-type writer parts.
	StringValue string

	// BytesValue holds raw bytes for WriterPartBytes, written via qw.N().SZ().
	BytesValue []byte

	// IntValue holds the integer value when this part represents a number.
	IntValue int64

	// UintValue holds the unsigned integer value when Type is WriterPartUint.
	UintValue uint64

	// FloatValue stores the floating-point number for float and FNV float parts.
	FloatValue float64

	// BoolValue holds the boolean value when the part type is WriterPartBool.
	BoolValue bool

	// Type specifies what kind of data this part holds, such as string or bytes.
	Type WriterPartType
}

// DirectWriter holds components for lazy, zero-allocation rendering
// directly to output. It implements fmt.Stringer and uses a fixed array
// to avoid slice allocation for common cases (up to 8 parts).
type DirectWriter struct {
	// cachedString holds the computed string form to avoid repeated work.
	cachedString string

	// Name is the identifier for this writer; empty means unnamed.
	Name string

	// overflow stores extra parts when the fixed-size parts array is full.
	overflow []WriterPart

	// borrowedBufs tracks byte buffers borrowed from a pool that must be returned
	// when Reset is called. Used by AppendPooledBytes to handle byte slices
	// without extra memory allocation.
	borrowedBufs []*[]byte

	// parts holds the fixed-size buffer for the first 8 parts, avoiding heap
	// allocation.
	parts [directWriterPartsCapacity]WriterPart

	// len is the number of parts held by the writer.
	len int

	// hasCachedString indicates whether cachedString holds a valid cached value.
	hasCachedString bool
}

var (
	// directWriterPool is a pool for DirectWriter objects.
	directWriterPool = sync.Pool{
		New: func() any {
			return &DirectWriter{}
		},
	}

	stringBufPool = sync.Pool{
		New: func() any {
			return new(make([]byte, 0, defaultStringBufCapacity))
		},
	}

	byteBufPool = sync.Pool{
		New: func() any {
			return new(make([]byte, 0, defaultByteBufCapacity))
		},
	}
)

// Reset clears the writer's state so it can be reused.
// Returns any borrowed byte buffers to the pool.
//
// The parts array is not zeroed because append() in AppendString, AppendInt,
// and similar methods fully sets all WriterPart fields. This saves about 5%
// CPU time per profile as WriterPart has 8 fields.
//
// The switch statement is unrolled for common buffer counts (0-2) to avoid
// loop overhead. String fields are only cleared when non-empty to avoid
// unnecessary writes. These changes improve performance by 5-18%.
func (dw *DirectWriter) Reset() {
	if dw.len == 0 && dw.Name == "" && len(dw.borrowedBufs) == 0 {
		return
	}

	dw.len = 0
	dw.overflow = dw.overflow[:0]

	switch len(dw.borrowedBufs) {
	case 0:
	case 1:
		PutByteBuf(dw.borrowedBufs[0])
	case 2:
		PutByteBuf(dw.borrowedBufs[0])
		PutByteBuf(dw.borrowedBufs[1])
	default:
		for _, bufferPointer := range dw.borrowedBufs {
			PutByteBuf(bufferPointer)
		}
	}
	dw.borrowedBufs = dw.borrowedBufs[:0]

	if dw.Name != "" {
		dw.Name = ""
	}
	if dw.hasCachedString {
		dw.cachedString = ""
		dw.hasCachedString = false
	}
}

// AppendString adds a string to the output buffer.
//
// Takes s (string) which is the string value to append.
func (dw *DirectWriter) AppendString(s string) {
	dw.append(WriterPart{
		Type:        WriterPartString,
		StringValue: s,
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
}

// AppendInt adds a signed integer to the output.
//
// Takes i (int64) which is the integer value to add.
func (dw *DirectWriter) AppendInt(i int64) {
	dw.append(WriterPart{
		Type:        WriterPartInt,
		StringValue: "",
		AnyValue:    nil,
		IntValue:    i,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
}

// AppendUint adds an unsigned integer part to the output.
//
// Takes u (uint64) which is the value to add.
func (dw *DirectWriter) AppendUint(u uint64) {
	dw.append(WriterPart{
		Type:        WriterPartUint,
		StringValue: "",
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   u,
		FloatValue:  0,
		BoolValue:   false,
	})
}

// AppendFloat adds a floating-point number to the output.
//
// Takes f (float64) which specifies the value to append.
func (dw *DirectWriter) AppendFloat(f float64) {
	dw.append(WriterPart{
		Type:        WriterPartFloat,
		StringValue: "",
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  f,
		BoolValue:   false,
	})
}

// AppendBool adds a boolean value to the output.
//
// Takes b (bool) which is the value to add.
func (dw *DirectWriter) AppendBool(b bool) {
	dw.append(WriterPart{
		Type:        WriterPartBool,
		StringValue: "",
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   b,
	})
}

// AppendAny adds an arbitrary value part to the writer.
//
// At render time, this checks for fmt.Stringer, then falls back to
// fmt.Sprint. Use this for non-primitive types that the generator cannot
// determine at compile time.
//
// Takes v (any) which is the value to append.
func (dw *DirectWriter) AppendAny(v any) {
	dw.append(WriterPart{
		Type:        WriterPartAny,
		StringValue: "",
		AnyValue:    v,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
}

// AppendPooledBytes adds a pooled byte slice to the output and
// tracks the buffer pointer for later release back to the pool
// when Reset is called.
//
// Takes bufferPointer (*[]byte) which is the pooled buffer
// pointer from GetByteBuf.
func (dw *DirectWriter) AppendPooledBytes(bufferPointer *[]byte) {
	if bufferPointer == nil {
		return
	}
	dw.append(WriterPart{
		Type:        WriterPartBytes,
		BytesValue:  *bufferPointer,
		StringValue: "",
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
	dw.borrowedBufs = append(dw.borrowedBufs, bufferPointer)
}

// AppendEscapePooledBytes appends a pooled byte slice that will
// be HTML-escaped at render time to prevent XSS.
//
// Takes bufferPointer (*[]byte) which is the pooled buffer
// pointer from GetByteBuf.
//
// Use this for style values from *Bytes helpers that may
// contain user input.
func (dw *DirectWriter) AppendEscapePooledBytes(bufferPointer *[]byte) {
	if bufferPointer == nil {
		return
	}
	dw.append(WriterPart{
		Type:        WriterPartEscapeBytes,
		BytesValue:  *bufferPointer,
		StringValue: "",
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
	dw.borrowedBufs = append(dw.borrowedBufs, bufferPointer)
}

// AppendEscapeString adds a string part that will be HTML-escaped at render
// time.
//
// Takes s (string) which is the user-provided content to escape.
//
// Use this for user-provided strings that may contain characters like
// <, >, &, ", '. The escaping uses zero allocation via quicktemplate's
// qw.E().S().
func (dw *DirectWriter) AppendEscapeString(s string) {
	dw.append(WriterPart{
		Type:        WriterPartEscapeString,
		StringValue: s,
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
}

// AppendFNVString adds a string part that will be FNV-32 hashed at render time.
//
// Takes s (string) which is the value to be hashed.
//
// Use this for p-key values where the string could contain problematic
// characters or be unpredictably long. The output is always an 8-character hex
// string.
func (dw *DirectWriter) AppendFNVString(s string) {
	dw.append(WriterPart{
		Type:        WriterPartFNVString,
		StringValue: s,
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
}

// AppendFNVFloat adds a float part that will be FNV-32 hashed at render time.
//
// Takes f (float64) which specifies the float value to hash.
//
// Use this for p-key values where float precision could cause issues.
// The output is always an 8-character hex string.
func (dw *DirectWriter) AppendFNVFloat(f float64) {
	dw.append(WriterPart{
		Type:        WriterPartFNVFloat,
		StringValue: "",
		AnyValue:    nil,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  f,
		BoolValue:   false,
	})
}

// AppendFNVAny adds an arbitrary value that will be FNV-32 hashed at render
// time.
//
// Takes v (any) which is the value to hash.
//
// Use this for p-key values with unknown types at generation time. The output
// is always an 8-character hex string.
func (dw *DirectWriter) AppendFNVAny(v any) {
	dw.append(WriterPart{
		Type:        WriterPartFNVAny,
		StringValue: "",
		AnyValue:    v,
		IntValue:    0,
		UintValue:   0,
		FloatValue:  0,
		BoolValue:   false,
	})
}

// SetName sets the attribute name for this DirectWriter.
//
// Takes name (string) which specifies the attribute name to set.
//
// Returns *DirectWriter which is self for method chaining.
func (dw *DirectWriter) SetName(name string) *DirectWriter {
	dw.Name = name
	return dw
}

// String returns the joined content of all parts as a single string.
//
// Returns string which is the combined content of all parts.
//
// The result is cached after the first call. Use this method only when a
// string is needed in Go code. For rendering, use WriteTo or write directly
// via the quicktemplate writer.
func (dw *DirectWriter) String() string {
	if dw == nil || dw.len == 0 {
		return ""
	}
	if dw.hasCachedString {
		return dw.cachedString
	}

	bufferPointer, ok := stringBufPool.Get().(*[]byte)
	if !ok {
		bufferPointer = new(make([]byte, 0, defaultStringBufCapacity))
	}
	buffer := *bufferPointer
	buffer = dw.WriteTo(buffer)

	dw.cachedString = string(buffer)
	dw.hasCachedString = true

	*bufferPointer = buffer[:0]
	stringBufPool.Put(bufferPointer)
	return dw.cachedString
}

// StringRaw returns the combined string content without HTML escaping.
// Used by premailer and CSS matching where the raw CSS value is needed
// rather than the HTML-escaped version.
//
// Returns string which is the joined content of all parts without escaping.
func (dw *DirectWriter) StringRaw() string {
	if dw == nil || dw.len == 0 {
		return ""
	}

	bufferPointer, ok := stringBufPool.Get().(*[]byte)
	if !ok {
		bufferPointer = new(make([]byte, 0, defaultStringBufCapacity))
	}
	buffer := *bufferPointer
	buffer = dw.WriteToRaw(buffer)

	result := string(buffer)
	*bufferPointer = buffer[:0]
	stringBufPool.Put(bufferPointer)
	return result
}

// SingleStringValue returns the string value if the DirectWriter contains
// exactly one WriterPartString part. This is a zero-allocation fast path
// for common cases like `:src="props.ImageUrl"` where the value is a simple
// string.
//
// Returns (string, true) if the writer has exactly one string part.
// Returns ("", false) otherwise, and the caller should use String() instead.
func (dw *DirectWriter) SingleStringValue() (string, bool) {
	if dw == nil || dw.len != 1 {
		return "", false
	}
	part := &dw.parts[0]
	if part.Type == WriterPartString {
		return part.StringValue, true
	}
	return "", false
}

// SingleBytesValue returns the byte slice if the DirectWriter contains
// exactly one WriterPartBytes or WriterPartEscapeBytes part. This is a
// zero-allocation fast path for style/class attributes using *Bytes helpers.
//
// Returns ([]byte, true) if the writer has exactly one bytes part.
// Returns (nil, false) otherwise.
func (dw *DirectWriter) SingleBytesValue() ([]byte, bool) {
	if dw == nil || dw.len != 1 {
		return nil, false
	}
	part := &dw.parts[0]
	if part.Type == WriterPartBytes || part.Type == WriterPartEscapeBytes {
		return part.BytesValue, true
	}
	return nil, false
}

// RenderedLen returns the total byte length of the rendered output without
// allocating memory. Use this for size estimation instead of
// len(dw.String()).
//
// Returns int which is the total byte length when rendered.
func (dw *DirectWriter) RenderedLen() int {
	if dw == nil || dw.len == 0 {
		return 0
	}
	total := 0
	for i := range dw.len {
		part := dw.Part(i)
		if part == nil {
			continue
		}
		total += partRenderedLen(part)
	}
	return total
}

// Len returns the number of parts held by the writer.
//
// Returns int which is the count of parts.
func (dw *DirectWriter) Len() int {
	return dw.len
}

// Part returns the part at the given index.
//
// Takes i (int) which specifies the zero-based index of the part to retrieve.
//
// Returns *WriterPart which is the part at the specified index, or nil if the
// index is out of bounds.
func (dw *DirectWriter) Part(i int) *WriterPart {
	if i < 0 || i >= dw.len {
		return nil
	}
	if i < directWriterPartsCapacity {
		return &dw.parts[i]
	}
	return &dw.overflow[i-directWriterPartsCapacity]
}

// WriteTo writes all parts directly to a byte slice, returning the extended
// slice.
//
// This is the zero-allocation render path for String, Int, Uint, Float, and
// Bool types. WriterPartAny may allocate via fmt.Stringer or fmt.Sprint.
// WriterPartEscapeString performs HTML escaping inline. WriterPartFNV* types
// produce 8-character hex hashes for safe p-key values.
//
// Takes buffer ([]byte) which is the destination slice to append rendered parts
// to.
//
// Returns []byte which is the extended slice containing all rendered parts.
func (dw *DirectWriter) WriteTo(buffer []byte) []byte {
	for i := range dw.len {
		part := dw.Part(i)
		if part == nil {
			continue
		}
		switch part.Type {
		case WriterPartString:
			buffer = append(buffer, part.StringValue...)
		case WriterPartEscapeString:
			buffer = appendHTMLEscape(buffer, part.StringValue)
		case WriterPartInt:
			buffer = strconv.AppendInt(buffer, part.IntValue, decimalBase)
		case WriterPartUint:
			buffer = strconv.AppendUint(buffer, part.UintValue, decimalBase)
		case WriterPartFloat:
			buffer = strconv.AppendFloat(buffer, part.FloatValue, 'f', -1, float64BitSize)
		case WriterPartBool:
			buffer = strconv.AppendBool(buffer, part.BoolValue)
		case WriterPartAny:
			buffer = appendAnyValue(buffer, part.AnyValue)
		case WriterPartFNVString:
			buffer = AppendFNVString(buffer, part.StringValue)
		case WriterPartFNVFloat:
			buffer = AppendFNVFloat(buffer, part.FloatValue)
		case WriterPartFNVAny:
			buffer = AppendFNVAny(buffer, part.AnyValue)
		case WriterPartBytes:
			buffer = append(buffer, part.BytesValue...)
		case WriterPartEscapeBytes:
			buffer = appendHTMLEscapeBytes(buffer, part.BytesValue)
		}
	}
	return buffer
}

// WriteToRaw writes all parts to a byte slice WITHOUT HTML escaping. Used by
// StringRaw() for premailer and CSS matching where the raw value is needed
// rather than the HTML-escaped version.
//
// Takes buffer ([]byte) which is the destination slice to append rendered parts
// to.
//
// Returns []byte which is the extended slice containing all rendered parts.
func (dw *DirectWriter) WriteToRaw(buffer []byte) []byte {
	for i := range dw.len {
		part := dw.Part(i)
		if part == nil {
			continue
		}
		switch part.Type {
		case WriterPartString, WriterPartEscapeString:
			buffer = append(buffer, part.StringValue...)
		case WriterPartInt:
			buffer = strconv.AppendInt(buffer, part.IntValue, decimalBase)
		case WriterPartUint:
			buffer = strconv.AppendUint(buffer, part.UintValue, decimalBase)
		case WriterPartFloat:
			buffer = strconv.AppendFloat(buffer, part.FloatValue, 'f', -1, float64BitSize)
		case WriterPartBool:
			buffer = strconv.AppendBool(buffer, part.BoolValue)
		case WriterPartAny:
			buffer = appendAnyValue(buffer, part.AnyValue)
		case WriterPartFNVString:
			buffer = AppendFNVString(buffer, part.StringValue)
		case WriterPartFNVFloat:
			buffer = AppendFNVFloat(buffer, part.FloatValue)
		case WriterPartFNVAny:
			buffer = AppendFNVAny(buffer, part.AnyValue)
		case WriterPartBytes, WriterPartEscapeBytes:
			buffer = append(buffer, part.BytesValue...)
		}
	}
	return buffer
}

// Clone creates a deep copy of the DirectWriter.
//
// The cloned writer is obtained from the pool and populated with copies of all
// parts. The clone is independent of the original - modifying one does not
// affect the other.
//
// For WriterPartBytes and WriterPartEscapeBytes, a new buffer is allocated from
// the pool and the data is copied. This prevents use-after-free when the
// original is reset and its buffers are returned to the pool.
//
// Returns *DirectWriter which is a new writer with copied parts, or nil if the
// receiver is nil.
func (dw *DirectWriter) Clone() *DirectWriter {
	if dw == nil {
		return nil
	}

	clone := GetDirectWriter()
	clone.Name = dw.Name

	for i := range dw.len {
		part := dw.Part(i)
		if part == nil {
			continue
		}

		switch part.Type {
		case WriterPartBytes:
			bufferPointer := GetByteBuf()
			*bufferPointer = append(*bufferPointer, part.BytesValue...)
			clone.AppendPooledBytes(bufferPointer)
		case WriterPartEscapeBytes:
			bufferPointer := GetByteBuf()
			*bufferPointer = append(*bufferPointer, part.BytesValue...)
			clone.AppendEscapePooledBytes(bufferPointer)
		default:
			clone.append(*part)
		}
	}

	return clone
}

// resetForArena clears the DirectWriter without returning byte buffers to
// the global pool. This is used when the arena manages buffer lifecycle,
// so we just clear our references without calling PutByteBuf.
//
// The parts array is not zeroed because append in AppendString, AppendInt,
// and similar methods fully sets all WriterPart fields.
func (dw *DirectWriter) resetForArena() {
	if dw.len == 0 && dw.Name == "" && len(dw.borrowedBufs) == 0 {
		return
	}

	dw.len = 0
	dw.overflow = dw.overflow[:0]

	dw.borrowedBufs = dw.borrowedBufs[:0]

	if dw.Name != "" {
		dw.Name = ""
	}
	if dw.hasCachedString {
		dw.cachedString = ""
		dw.hasCachedString = false
	}
}

// append adds a writer part to the buffer.
//
// Takes p (WriterPart) which is the part to add.
func (dw *DirectWriter) append(p WriterPart) {
	if dw.len < directWriterPartsCapacity {
		dw.parts[dw.len] = p
	} else {
		dw.overflow = append(dw.overflow, p)
	}
	dw.len++
}

// GetByteBuf retrieves a byte buffer from the pool for encoding operations.
// The caller must return the buffer using PutByteBuf when finished.
//
// Returns *[]byte which is a pointer to a reusable byte slice.
func GetByteBuf() *[]byte {
	bufferPointer, ok := byteBufPool.Get().(*[]byte)
	if !ok {
		return new(make([]byte, 0, defaultByteBufCapacity))
	}
	*bufferPointer = (*bufferPointer)[:0]
	return bufferPointer
}

// PutByteBuf returns a byte buffer to the pool for reuse.
//
// Takes bufferPointer (*[]byte) which is the buffer to return to the pool.
func PutByteBuf(bufferPointer *[]byte) {
	if bufferPointer == nil {
		return
	}
	*bufferPointer = (*bufferPointer)[:0]
	byteBufPool.Put(bufferPointer)
}

// ResetByteBufPool resets the byte buffer pool to its initial state.
//
// Use this in tests with t.Cleanup(ResetByteBufPool) to ensure test isolation.
func ResetByteBufPool() {
	byteBufPool = sync.Pool{
		New: func() any {
			return new(make([]byte, 0, defaultByteBufCapacity))
		},
	}
}

// GetDirectWriter retrieves a DirectWriter from the pool.
//
// Returns *DirectWriter which is reset and ready for use.
func GetDirectWriter() *DirectWriter {
	dw, ok := directWriterPool.Get().(*DirectWriter)
	if !ok {
		dw = &DirectWriter{}
	}
	dw.Reset()
	return dw
}

// PutDirectWriter returns a DirectWriter to the pool for reuse.
//
// It calls Reset to release any borrowed byte buffers before pooling. This
// stops BytesValue slice headers from pointing to memory that may be reused.
//
// Takes dw (*DirectWriter) which is the writer to return to the pool.
func PutDirectWriter(dw *DirectWriter) {
	if dw == nil {
		return
	}
	dw.Reset()
	directWriterPool.Put(dw)
}

// ResetDirectWriterPool clears the direct writer pool to ensure test isolation.
// Call this function via t.Cleanup(ResetDirectWriterPool) in tests.
func ResetDirectWriterPool() {
	directWriterPool = sync.Pool{
		New: func() any {
			return &DirectWriter{}
		},
	}
}

// partRenderedLen returns the rendered length of a single part.
//
// Takes part (*WriterPart) which specifies the writer part to measure.
//
// Returns int which is the length in bytes when rendered.
func partRenderedLen(part *WriterPart) int {
	switch part.Type {
	case WriterPartString, WriterPartEscapeString:
		return len(part.StringValue)
	case WriterPartBytes, WriterPartEscapeBytes:
		return len(part.BytesValue)
	case WriterPartInt:
		return intLen(part.IntValue)
	case WriterPartUint:
		return uintLen(part.UintValue)
	case WriterPartFloat, WriterPartAny:
		return estimatedFloatLen
	case WriterPartBool:
		if part.BoolValue {
			return trueLiteralLen
		}
		return falseLiteralLen
	case WriterPartFNVString, WriterPartFNVFloat, WriterPartFNVAny:
		return fnvHexLen
	}
	return 0
}

// intLen returns the number of characters needed to show an int64 as a string
// without allocating memory.
//
// Takes v (int64) which is the integer value to measure.
//
// Returns int which is the character count for the string form of v.
func intLen(v int64) int {
	if v == 0 {
		return 1
	}
	n := 0
	if v < 0 {
		n = 1
		v = -v
	}
	for v > 0 {
		n++
		v /= decimalBase
	}
	return n
}

// uintLen returns the number of decimal digits needed to represent a value.
//
// Takes v (uint64) which is the value to measure.
//
// Returns int which is the digit count, with a minimum of 1 for zero.
func uintLen(v uint64) int {
	if v == 0 {
		return 1
	}
	n := 0
	for v > 0 {
		n++
		v /= decimalBase
	}
	return n
}

// appendHTMLEscape appends the string s to buffer with HTML special characters
// escaped. The escaped characters are: < > & " '.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes s (string) which is the string to escape.
//
// Returns []byte which is the buffer with the escaped string added.
func appendHTMLEscape(buffer []byte, s string) []byte {
	for i := range len(s) {
		switch s[i] {
		case '<':
			buffer = append(buffer, "&lt;"...)
		case '>':
			buffer = append(buffer, "&gt;"...)
		case '&':
			buffer = append(buffer, "&amp;"...)
		case '"':
			buffer = append(buffer, "&quot;"...)
		case '\'':
			buffer = append(buffer, "&#39;"...)
		default:
			buffer = append(buffer, s[i])
		}
	}
	return buffer
}

// appendHTMLEscapeBytes appends b to buffer with HTML special
// characters escaped, handling byte slice input for < > & " '.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes b ([]byte) which is the byte slice to escape.
//
// Returns []byte which is the buffer with the escaped bytes added.
func appendHTMLEscapeBytes(buffer []byte, b []byte) []byte {
	for _, c := range b {
		switch c {
		case '<':
			buffer = append(buffer, "&lt;"...)
		case '>':
			buffer = append(buffer, "&gt;"...)
		case '&':
			buffer = append(buffer, "&amp;"...)
		case '"':
			buffer = append(buffer, "&quot;"...)
		case '\'':
			buffer = append(buffer, "&#39;"...)
		default:
			buffer = append(buffer, c)
		}
	}
	return buffer
}

// appendAnyValue converts a value to a string and appends it to the buffer.
//
// Uses fmt.Stringer if the value has it, otherwise uses fmt.Sprint. This path
// allocates memory but handles cases the generator cannot improve.
//
// When v is nil, returns the buffer unchanged.
//
// Takes buffer ([]byte) which is the destination buffer for the string value.
// Takes v (any) which is the value to convert and append.
//
// Returns []byte which is the buffer with the appended string value.
func appendAnyValue(buffer []byte, v any) []byte {
	if v == nil {
		return buffer
	}

	if stringer, ok := v.(fmt.Stringer); ok {
		return append(buffer, stringer.String()...)
	}

	return append(buffer, fmt.Sprint(v)...)
}

// cloneDirectWriterSlice creates a deep copy of a slice of DirectWriters.
//
// Takes writers ([]*DirectWriter) which is the slice to copy.
//
// Returns []*DirectWriter which is a new slice with copied writers, or nil if
// the input is nil or empty.
func cloneDirectWriterSlice(writers []*DirectWriter) []*DirectWriter {
	if len(writers) == 0 {
		return nil
	}

	clone := make([]*DirectWriter, len(writers))
	for i, w := range writers {
		clone[i] = w.Clone()
	}
	return clone
}
