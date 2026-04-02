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

package i18n_domain

import (
	"fmt"
	"strconv"
	"sync"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/maths"
)

const (
	// base10 is the decimal base used when converting integers to strings.
	base10 = 10

	// bits64 is the bit size for 64-bit floats, used with strconv functions.
	bits64 = 64

	// zeroString is the fallback value written when a numeric type has an error.
	zeroString = "0"
)

// StrBuf is a buffer for building strings without memory allocation.
// It implements fmt.Stringer.
type StrBuf struct {
	// buffer holds the underlying byte slice for building strings.
	buffer []byte
}

// NewStrBuf creates a new StrBuf with the given initial capacity.
//
// Takes capacity (int) which specifies the initial buffer capacity in bytes.
//
// Returns *StrBuf which is the newly allocated string buffer.
func NewStrBuf(capacity int) *StrBuf {
	return &StrBuf{
		buffer: make([]byte, 0, capacity),
	}
}

// WriteString appends a string to the buffer.
//
// Takes s (string) which is the text to append.
func (b *StrBuf) WriteString(s string) {
	b.buffer = append(b.buffer, s...)
}

// WriteByte appends a single byte to the buffer.
//
// Takes c (byte) which is the byte value to append.
//
// Returns error which is always nil; present for io.ByteWriter compatibility.
func (b *StrBuf) WriteByte(c byte) error {
	b.buffer = append(b.buffer, c)
	return nil
}

// WriteInt appends an integer to the buffer.
//
// Takes n (int) which is the value to append.
func (b *StrBuf) WriteInt(n int) {
	b.buffer = strconv.AppendInt(b.buffer, int64(n), base10)
}

// WriteInt64 appends an int64 to the buffer.
//
// Takes n (int64) which is the value to append.
func (b *StrBuf) WriteInt64(n int64) {
	b.buffer = strconv.AppendInt(b.buffer, n, base10)
}

// WriteFloat appends a float64 to the buffer using default precision.
//
// Takes f (float64) which is the value to append.
func (b *StrBuf) WriteFloat(f float64) {
	b.buffer = strconv.AppendFloat(b.buffer, f, 'f', -1, bits64)
}

// WriteFloatPrec appends a float64 to the buffer with specified precision.
//
// Takes f (float64) which is the value to append.
// Takes prec (int) which specifies the number of decimal places.
func (b *StrBuf) WriteFloatPrec(f float64, prec int) {
	b.buffer = strconv.AppendFloat(b.buffer, f, 'f', prec, bits64)
}

// WriteFloatWithLocale appends a float64 to the buffer using locale-specific
// formatting. This uses the locale's decimal separator and thousand grouping
// conventions.
//
// Takes f (float64) which is the value to format and append.
// Takes locale (string) which specifies the locale for number formatting.
func (b *StrBuf) WriteFloatWithLocale(f float64, locale string) {
	if locale == "" {
		b.WriteFloat(f)
		return
	}
	s := strconv.FormatFloat(f, 'f', -1, bits64)
	b.WriteString(maths.FormatNumberString(s, maths.GetNumberLocale(locale)))
}

// WriteBool appends a boolean to the buffer.
//
// Takes v (bool) which is the value to append.
func (b *StrBuf) WriteBool(v bool) {
	b.buffer = strconv.AppendBool(b.buffer, v)
}

// WriteDecimal appends a Decimal to the buffer.
//
// Takes d (maths.Decimal) which is the decimal value to append.
func (b *StrBuf) WriteDecimal(d maths.Decimal) {
	if d.Err() != nil {
		b.WriteString(zeroString)
		return
	}
	b.WriteString(d.MustString())
}

// WriteDecimalWithLocale appends a Decimal to the buffer using locale-specific
// formatting with the locale's decimal separator and thousand grouping
// conventions. Full precision is preserved; no rounding occurs.
//
// Takes d (maths.Decimal) which is the decimal value to format and append.
// Takes locale (string) which specifies the locale for formatting conventions.
func (b *StrBuf) WriteDecimalWithLocale(d maths.Decimal, locale string) {
	if d.Err() != nil {
		b.WriteString(zeroString)
		return
	}
	if locale == "" {
		b.WriteString(d.MustString())
		return
	}
	b.WriteString(d.MustFormat(locale))
}

// WriteMoney appends a Money value to the buffer using default formatting.
//
// Takes m (maths.Money) which is the monetary value to format and append.
func (b *StrBuf) WriteMoney(m maths.Money) {
	if m.Err() != nil {
		b.WriteString(zeroString)
		return
	}
	b.WriteString(m.DefaultFormat())
}

// WriteMoneyWithLocale appends a formatted money value to the buffer using
// locale-specific settings. It formats currency symbols, decimal separators,
// and digit grouping based on the given locale.
//
// Takes m (maths.Money) which is the money value to format.
// Takes locale (string) which sets the locale for formatting; if empty, uses
// the default format.
func (b *StrBuf) WriteMoneyWithLocale(m maths.Money, locale string) {
	if m.Err() != nil {
		b.WriteString(zeroString)
		return
	}
	if locale == "" {
		b.WriteString(m.DefaultFormat())
		return
	}
	b.WriteString(m.MustFormat(locale))
}

// WriteBigInt appends a BigInt to the buffer.
//
// Takes bi (maths.BigInt) which is the value to append.
func (b *StrBuf) WriteBigInt(bi maths.BigInt) {
	if bi.Err() != nil {
		b.WriteString(zeroString)
		return
	}
	b.WriteString(bi.MustString())
}

// WriteAny appends any value to the buffer using type-specific formatting.
//
// Takes v (any) which is the value to append.
func (b *StrBuf) WriteAny(v any) {
	switch value := v.(type) {
	case string:
		b.WriteString(value)
	case int:
		b.WriteInt(value)
	case int64:
		b.WriteInt64(value)
	case float64:
		b.WriteFloat(value)
	case float32:
		b.WriteFloat(float64(value))
	case bool:
		b.WriteBool(value)
	case []byte:
		b.buffer = append(b.buffer, value...)
	case maths.Decimal:
		b.WriteDecimal(value)
	case maths.Money:
		b.WriteMoney(value)
	case maths.BigInt:
		b.WriteBigInt(value)
	default:
		b.WriteString(formatAny(value))
	}
}

// Reset clears the buffer so it can be reused without freeing memory.
func (b *StrBuf) Reset() {
	b.buffer = b.buffer[:0]
}

// Len returns the current length of the buffer.
//
// Returns int which is the number of bytes currently stored.
func (b *StrBuf) Len() int {
	return len(b.buffer)
}

// Cap returns the capacity of the buffer.
//
// Returns int which is the total space set aside for the buffer in bytes.
func (b *StrBuf) Cap() int {
	return cap(b.buffer)
}

// Bytes returns the buffer contents as a byte slice.
// The returned slice is only valid until the next write or reset.
//
// Returns []byte which contains the current buffer contents.
func (b *StrBuf) Bytes() []byte {
	return b.buffer
}

// String returns the buffer contents as a string.
// This allocates a new string.
//
// Returns string which contains the buffer's contents.
func (b *StrBuf) String() string {
	return string(b.buffer)
}

// UnsafeString returns the buffer contents as a string without allocation.
// In safe mode, this falls back to an allocating copy (identical to String()).
//
// Returns string which contains the buffer contents.
//
// WARNING: The returned string is only valid until the buffer is modified
// or reset. Use only when you know the buffer will not be modified before
// the string is used.
func (b *StrBuf) UnsafeString() string {
	return mem.String(b.buffer)
}

// StrBufPool provides a pool of reusable StrBuf instances.
type StrBufPool struct {
	// pool is the sync.Pool that stores reusable StrBuf instances.
	pool sync.Pool
}

// NewStrBufPool creates a new pool with buffers of the given initial size.
//
// Takes capacity (int) which sets the starting size of each buffer in the pool.
//
// Returns *StrBufPool which provides pooled string buffers for reuse.
func NewStrBufPool(capacity int) *StrBufPool {
	return &StrBufPool{
		pool: sync.Pool{
			New: func() any {
				return NewStrBuf(capacity)
			},
		},
	}
}

// Get retrieves a StrBuf from the pool.
//
// Returns *StrBuf which is a reset buffer ready for use.
func (p *StrBufPool) Get() *StrBuf {
	buffer, ok := p.pool.Get().(*StrBuf)
	if !ok {
		return NewStrBuf(defaultStrBufCapacity)
	}
	buffer.Reset()
	return buffer
}

// Put returns a buffer to the pool for reuse.
//
// Takes buffer (*StrBuf) which is the buffer to return.
func (p *StrBufPool) Put(buffer *StrBuf) {
	p.pool.Put(buffer)
}

// formatAny formats any value as a string.
// This is a fallback for types not handled elsewhere and causes allocation.
//
// Takes v (any) which is the value to format.
//
// Returns string which is the formatted text form of the value.
func formatAny(v any) string {
	return fmt.Sprint(v)
}
