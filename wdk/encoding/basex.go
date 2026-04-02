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

package encoding

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strings"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// asciiLookupSize is the size of the ASCII reverse lookup table.
	asciiLookupSize = 256

	// invalidASCII marks a byte as invalid in the reverse lookup table.
	invalidASCII = -1

	// asciiMaxValue is the highest code point value in the ASCII character set.
	asciiMaxValue = 128

	// fastPathSize8 is the byte size for uint64-based fast path.
	fastPathSize8 = 8

	// fastPathSize16 is the byte size for uint128-based fast path.
	fastPathSize16 = 16

	// maxBase36Len8 is the maximum Base36 encoded length for 8 bytes.
	maxBase36Len8 = 13

	// maxBase36Len16 is the maximum Base36 encoded length for 16 bytes.
	maxBase36Len16 = 25

	// StdBase64Alphabet is the standard Base64 character set as defined in RFC
	// 4648.
	StdBase64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	// URLBase64Alphabet is the URL-safe Base64 alphabet as defined in RFC 4648.
	// It uses '-' and '_' instead of '+' and '/' to avoid problems in URLs.
	URLBase64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	// StdHexAlphabetLower is the standard lowercase hexadecimal alphabet.
	StdHexAlphabetLower = "0123456789abcdef"

	// StdHexAlphabetUpper is the standard uppercase hexadecimal alphabet.
	StdHexAlphabetUpper = "0123456789ABCDEF"

	// StdBase32Alphabet is the standard Base32 alphabet from RFC 4648.
	StdBase32Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

	// HexBase32Alphabet is the Extended Hex Base32 alphabet as defined in
	// RFC 4648. This alphabet keeps the sort order of the original data when
	// the encoded text is sorted.
	HexBase32Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUV"

	// maxUint64Digits is the maximum number of digits needed to represent a uint64
	// in base 10. This sets the buffer size for encoding integers.
	maxUint64Digits = 20

	// minFastPathBase is the minimum base for which the fast path
	// buffer is sized, assuming base-36 encoding that produces at
	// most 13 characters for 8 bytes.
	//
	// Smaller bases produce longer output and would overflow the
	// buffer.
	minFastPathBase = 36

	// bitsPerByte is the number of bits in a byte.
	bitsPerByte = 8

	// byteMask is the bitmask for getting the lowest 8 bits of an integer.
	byteMask = 0xFF

	// errFmtNonBaseChar is the format string for errors when input contains
	// characters outside the encoding's alphabet.
	errFmtNonBaseChar = "non-base character '%c' found in input"
)

var (
	// ErrAlphabetEmpty is returned when an empty alphabet string is provided to
	// NewEncoding.
	ErrAlphabetEmpty = errors.New("alphabet cannot be empty")

	// ErrAlphabetAmbiguous is returned when an alphabet contains duplicate
	// characters.
	ErrAlphabetAmbiguous = errors.New("ambiguous alphabet")
)

// fastPathEncoder defines methods for fast encoding and decoding of bytes.
// It allows base64 and hex encoders to be used in the same way.
type fastPathEncoder interface {
	// EncodeToString encodes the source bytes to a string representation.
	//
	// Takes source ([]byte) which is the data to encode.
	//
	// Returns string which is the encoded representation.
	EncodeToString(source []byte) string

	// DecodeString decodes the given string and returns the raw bytes.
	//
	// Takes s (string) which is the encoded string to decode.
	//
	// Returns []byte which contains the decoded data.
	// Returns error when the string cannot be decoded.
	DecodeString(s string) ([]byte, error)
}

// Encoding represents a generic base-X encoding scheme defined by a custom
// alphabet. It can handle arbitrary bases but will automatically delegate to
// optimised standard library implementations (like base64, hex) if a
// standard alphabet is provided.
type Encoding struct {
	// fastPath holds an optimised encoder for standard alphabets such as base64,
	// base32, and hex; nil when using the generic baseX algorithm.
	fastPath fastPathEncoder

	// reverseMap maps each rune in the alphabet to its numeric value for decoding.
	reverseMap map[rune]int

	// zeroString is the string for zero, stored to avoid memory allocation.
	zeroString string

	// alphabet holds the characters used for encoding digits.
	alphabet []rune

	// alphabetBytes stores each alphabet character as a byte for fast encoding.
	alphabetBytes []byte

	// reverseByteMap maps ASCII byte values to digit indices for O(1) decoding.
	reverseByteMap [asciiLookupSize]int

	// base is the number base for encoding; must be between 2 and 62.
	base int

	// zeroRune is the character that stands for zero in the encoding alphabet.
	zeroRune rune

	// zeroByte is the first byte of the alphabet, used to represent leading zeros.
	zeroByte byte

	// isASCII indicates whether all alphabet characters are ASCII (below 128).
	isASCII bool

	// hasReverseBytes indicates whether reverseByteMap has been filled in.
	hasReverseBytes bool
}

// NewEncoding creates an Encoding for the given alphabet, using optimised
// standard library implementations when the alphabet matches a well-known
// encoding such as Base64, Base32, or Hex.
//
// Takes alphabet (string) which specifies the characters for the encoding.
//
// Returns *Encoding which is ready for encoding and decoding operations.
// Returns error when the alphabet is empty or contains duplicate characters.
func NewEncoding(alphabet string) (*Encoding, error) {
	if alphabet == "" {
		return nil, ErrAlphabetEmpty
	}

	runes := []rune(alphabet)
	reverseMap := make(map[rune]int, len(runes))

	isASCII := true
	for i, character := range runes {
		if _, exists := reverseMap[character]; exists {
			return nil, fmt.Errorf("character %q repeated: %w", character, ErrAlphabetAmbiguous)
		}
		reverseMap[character] = i
		if character >= asciiMaxValue {
			isASCII = false
		}
	}

	e := &Encoding{
		reverseMap: reverseMap,
		alphabet:   runes,
		zeroString: string(runes[0]),
		base:       len(runes),
		zeroRune:   runes[0],
		isASCII:    isASCII,
	}

	if isASCII {
		e.initialiseASCIILookup(runes)
	}

	switch alphabet {
	case StdBase64Alphabet:
		e.fastPath = &base64Adapter{Encoding: base64.StdEncoding}
	case URLBase64Alphabet:
		e.fastPath = &base64Adapter{Encoding: base64.URLEncoding}
	case StdHexAlphabetLower, StdHexAlphabetUpper:
		e.fastPath = hexAdapter{}
	case StdBase32Alphabet:
		e.fastPath = &base32Adapter{Encoding: base32.StdEncoding}
	case HexBase32Alphabet:
		e.fastPath = &base32Adapter{Encoding: base32.HexEncoding}
	}

	return e, nil
}

// EncodeBytes converts a slice of bytes into a string using the encoding.
// It will automatically use the optimised standard library if a
// standard alphabet (Base64, Hex) was detected at creation time.
//
// Takes data ([]byte) which is the raw bytes to encode.
//
// Returns string which is the encoded representation of the input bytes.
func (enc *Encoding) EncodeBytes(data []byte) string {
	if enc.fastPath != nil {
		return enc.fastPath.EncodeToString(data)
	}

	return enc.encodeBytesGeneric(data)
}

// DecodeBytes converts a string created by EncodeBytes back into a slice of
// bytes. It will automatically use the optimised standard library if a
// standard alphabet (Base64, Hex) was detected at creation time.
//
// Takes input (string) which is the encoded string to decode.
//
// Returns []byte which is the decoded byte slice.
// Returns error when the input string is not valid for this encoding.
func (enc *Encoding) DecodeBytes(input string) ([]byte, error) {
	if enc.fastPath != nil {
		return enc.fastPath.DecodeString(input)
	}

	return enc.decodeBytesGeneric(input)
}

// EncodeUint64 converts an uint64 value into a string using the encoding.
// This is optimised for converting database IDs into short, clean identifiers.
//
// Takes value (uint64) which is the number to encode.
//
// Returns string which is the encoded representation.
func (enc *Encoding) EncodeUint64(value uint64) string {
	if enc.isASCII {
		return enc.encodeUint64ASCII(value)
	}

	return enc.encodeUint64Runes(value)
}

// DecodeUint64 converts a string created by EncodeUint64 back into an uint64
// value.
//
// Takes input (string) which is the encoded string to decode.
//
// Returns uint64 which is the decoded numeric value.
// Returns error when the input is empty, contains invalid runes for the
// alphabet, or the result would overflow uint64.
func (enc *Encoding) DecodeUint64(input string) (uint64, error) {
	if input == "" {
		return 0, errors.New("cannot decode empty string as uint64")
	}

	if enc.hasReverseBytes {
		return enc.decodeUint64ASCII(input)
	}

	return enc.decodeUint64Runes(input)
}

// initialiseASCIILookup sets up the byte lookup tables for ASCII alphabets.
//
// Takes runes ([]rune) which contains the alphabet characters to map.
func (enc *Encoding) initialiseASCIILookup(runes []rune) {
	enc.alphabetBytes = make([]byte, len(runes))
	for i := range enc.reverseByteMap {
		enc.reverseByteMap[i] = invalidASCII
	}
	for i, r := range runes {
		b := byte(r)
		enc.alphabetBytes[i] = b
		enc.reverseByteMap[b] = i
	}
	enc.zeroByte = enc.alphabetBytes[0]
	enc.hasReverseBytes = true
}

// encodeUint64ASCII encodes an uint64 using the byte-based alphabet.
//
// Takes value (uint64) which is the number to encode.
//
// Returns string which is the encoded representation, with zero allocation
// for zero value.
func (enc *Encoding) encodeUint64ASCII(value uint64) string {
	if value == 0 {
		return enc.zeroString
	}

	var buffer [maxUint64Digits]byte
	position := maxUint64Digits
	base := safeconv.IntToUint64(enc.base)

	for value > 0 {
		position--
		remainder := value % base
		value /= base
		buffer[position] = enc.alphabetBytes[remainder]
	}

	return string(buffer[position:])
}

// encodeUint64Runes encodes an uint64 using the rune-based alphabet for
// non-ASCII encodings.
//
// Takes value (uint64) which is the number to encode.
//
// Returns string which is the encoded result.
func (enc *Encoding) encodeUint64Runes(value uint64) string {
	if value == 0 {
		return enc.zeroString
	}

	base := safeconv.IntToUint64(enc.base)
	digits := make([]rune, 0, maxUint64Digits)

	for value > 0 {
		remainder := value % base
		value /= base
		digits = append(digits, enc.alphabet[remainder])
	}

	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	return string(digits)
}

// decodeUint64ASCII decodes using byte-based lookup for ASCII alphabets.
//
// Takes input (string) which is the encoded string to decode.
//
// Returns uint64 which is the decoded numeric value.
// Returns error when the input contains invalid characters or overflows.
func (enc *Encoding) decodeUint64ASCII(input string) (uint64, error) {
	var result uint64
	base := safeconv.IntToUint64(enc.base)

	for i := range len(input) {
		digitVal := enc.reverseByteMap[input[i]]
		if digitVal == invalidASCII {
			return 0, fmt.Errorf("invalid character '%c' for this alphabet", input[i])
		}

		safeValue := safeconv.IntToUint64(digitVal)
		if result > (math.MaxUint64-safeValue)/base {
			return 0, fmt.Errorf("overflow: input %q exceeds uint64 capacity", input)
		}
		result = result*base + safeValue
	}

	return result, nil
}

// decodeUint64Runes decodes using rune-based lookup for non-ASCII alphabets.
//
// Takes input (string) which is the encoded string to decode.
//
// Returns uint64 which is the decoded numeric value.
// Returns error when the input contains invalid runes or overflows.
func (enc *Encoding) decodeUint64Runes(input string) (uint64, error) {
	var result uint64
	base := safeconv.IntToUint64(enc.base)

	for _, r := range input {
		value, ok := enc.reverseMap[r]
		if !ok {
			return 0, fmt.Errorf("invalid rune '%c' for this alphabet", r)
		}

		safeValue := safeconv.IntToUint64(value)
		if result > (math.MaxUint64-safeValue)/base {
			return 0, fmt.Errorf("overflow: input %q exceeds uint64 capacity", input)
		}
		result = result*base + safeValue
	}

	return result, nil
}

// encodeBytesGeneric implements the generic arithmetic-based encoding for
// custom alphabets.
//
// Takes data ([]byte) which is the byte slice to encode.
//
// Returns string which is the encoded result using the alphabet.
func (enc *Encoding) encodeBytesGeneric(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	if isAllZeroBytes(data) {
		return strings.Repeat(enc.zeroString, len(data))
	}

	leadingZeroCount := countLeadingZeroBytes(data)

	if enc.canUseFastPath8(data) {
		return enc.encode8BytesFast(data, leadingZeroCount)
	}

	if enc.canUseFastPath16(data) {
		return enc.encode16BytesFast(data, leadingZeroCount)
	}

	digits := enc.convertBytesToDigits(data)

	if enc.isASCII {
		return enc.encodeBytesASCII(digits, leadingZeroCount)
	}

	return enc.encodeBytesRunes(digits, leadingZeroCount)
}

// encodeBytesASCII encodes digits to a string using the byte-based alphabet.
// This method avoids memory allocation where possible.
//
// Takes digits ([]int) which contains the numeric values to encode.
// Takes leadingZeroCount (int) which specifies zeros to add at the start.
//
// Returns string which is the encoded form of the digits.
func (enc *Encoding) encodeBytesASCII(digits []int, leadingZeroCount int) string {
	outputLen := leadingZeroCount + len(digits)
	buffer := make([]byte, outputLen)

	for i := range leadingZeroCount {
		buffer[i] = enc.zeroByte
	}

	position := leadingZeroCount
	for i := len(digits) - 1; i >= 0; i-- {
		buffer[position] = enc.alphabetBytes[digits[i]]
		position++
	}

	return mem.String(buffer)
}

// encodeBytesRunes encodes digits to a string using rune-based alphabet
// (for non-ASCII characters).
//
// Takes digits ([]int) which contains the numeric values to encode.
// Takes leadingZeroCount (int) which specifies zero padding at the start.
//
// Returns string which is the encoded result using alphabet runes.
func (enc *Encoding) encodeBytesRunes(digits []int, leadingZeroCount int) string {
	buffer := bytes.Buffer{}
	buffer.Grow(leadingZeroCount + len(digits)*4)

	for range leadingZeroCount {
		_, _ = buffer.WriteRune(enc.zeroRune)
	}

	for i := len(digits) - 1; i >= 0; i-- {
		_, _ = buffer.WriteRune(enc.alphabet[digits[i]])
	}

	return buffer.String()
}

// convertBytesToDigits converts raw bytes to base-N digits using a change of
// base algorithm.
//
// Takes data ([]byte) which contains the raw bytes to convert.
//
// Returns []int which contains the digits in little-endian order.
func (enc *Encoding) convertBytesToDigits(data []byte) []int {
	estimatedCap := len(data)*2 + 1
	digits := make([]int, 1, estimatedCap)

	for _, b := range data {
		carry := int(b)
		for i := range len(digits) {
			carry += digits[i] << bitsPerByte
			digits[i] = carry % enc.base
			carry /= enc.base
		}
		for carry > 0 {
			digits = append(digits, carry%enc.base)
			carry /= enc.base
		}
	}
	return digits
}

// decodeBytesGeneric implements the generic arithmetic-based decoding for
// custom alphabets.
//
// Takes input (string) which is the encoded string to decode.
//
// Returns []byte which contains the decoded bytes.
// Returns error when the input contains invalid characters.
func (enc *Encoding) decodeBytesGeneric(input string) ([]byte, error) {
	if input == "" {
		return []byte{}, nil
	}

	if enc.isASCII {
		return enc.decodeBytesASCII(input)
	}

	return enc.decodeBytesRunes(input)
}

// decodeBytesASCII decodes a string using byte-based lookups, which avoids
// []rune allocation.
//
// Takes input (string) which is the encoded string to decode.
//
// Returns []byte which contains the decoded bytes.
// Returns error when the input contains invalid characters.
func (enc *Encoding) decodeBytesASCII(input string) ([]byte, error) {
	if isAllZeroBytesString(input, enc.zeroByte) {
		return make([]byte, len(input)), nil
	}

	leadingZeroCount := countLeadingZeroBytesString(input, enc.zeroByte)

	bytesResult, err := enc.convertDigitsToBytesASCII(input)
	if err != nil {
		return nil, err
	}

	reverseBytes(bytesResult)

	totalLen := leadingZeroCount + len(bytesResult)
	decoded := make([]byte, totalLen)
	copy(decoded[leadingZeroCount:], bytesResult)

	return decoded, nil
}

// decodeBytesRunes decodes a string using rune-based lookups for non-ASCII
// alphabets.
//
// Takes input (string) which is the encoded string to decode.
//
// Returns []byte which contains the decoded bytes in big-endian order.
// Returns error when the input contains invalid characters.
func (enc *Encoding) decodeBytesRunes(input string) ([]byte, error) {
	runes := []rune(input)

	if isAllZeroRunes(runes, enc.zeroRune) {
		return make([]byte, len(runes)), nil
	}

	leadingZeroCount := countLeadingZeroRunes(runes, enc.zeroRune)

	bytesResult, err := enc.convertDigitsToBytesRunes(runes)
	if err != nil {
		return nil, err
	}

	reverseBytes(bytesResult)

	totalLen := leadingZeroCount + len(bytesResult)
	decoded := make([]byte, totalLen)
	copy(decoded[leadingZeroCount:], bytesResult)

	return decoded, nil
}

// convertDigitsToBytesASCII converts a base-N encoded string back to bytes
// using array lookups.
//
// Takes input (string) which is the base-N encoded string to convert.
//
// Returns []byte which contains the decoded bytes.
// Returns error when the input contains a character not in the alphabet.
func (enc *Encoding) convertDigitsToBytesASCII(input string) ([]byte, error) {
	estimatedCap := len(input) + 1
	bytesResult := make([]byte, 1, estimatedCap)

	for i := range len(input) {
		b := input[i]
		value := enc.reverseByteMap[b]
		if value == invalidASCII {
			return nil, fmt.Errorf(errFmtNonBaseChar, b)
		}

		carry := value
		for j := range len(bytesResult) {
			carry += int(bytesResult[j]) * enc.base
			bytesResult[j] = byte(carry & byteMask)
			carry >>= bitsPerByte
		}
		for carry > 0 {
			bytesResult = append(bytesResult, byte(carry&byteMask))
			carry >>= bitsPerByte
		}
	}
	return bytesResult, nil
}

// convertDigitsToBytesRunes converts base-N encoded runes back to bytes.
//
// Takes runes ([]rune) which contains the base-N encoded characters to decode.
//
// Returns []byte which contains the decoded byte values.
// Returns error when a rune is not a valid character in the encoding alphabet.
func (enc *Encoding) convertDigitsToBytesRunes(runes []rune) ([]byte, error) {
	estimatedCap := len(runes) + 1
	bytesResult := make([]byte, 1, estimatedCap)

	for _, r := range runes {
		value, ok := enc.reverseMap[r]
		if !ok {
			return nil, fmt.Errorf(errFmtNonBaseChar, r)
		}

		carry := value
		for i := range len(bytesResult) {
			carry += int(bytesResult[i]) * enc.base
			bytesResult[i] = byte(carry & byteMask)
			carry >>= bitsPerByte
		}
		for carry > 0 {
			bytesResult = append(bytesResult, byte(carry&byteMask))
			carry >>= bitsPerByte
		}
	}
	return bytesResult, nil
}

// encode8BytesFast encodes exactly 8 bytes using uint64 arithmetic.
// This is a fast path that avoids the O(n^2) generic algorithm.
//
// Takes data ([]byte) which must be exactly 8 bytes.
// Takes leadingZeroCount (int) which is the number of leading zero bytes.
//
// Returns string which is the encoded representation.
func (enc *Encoding) encode8BytesFast(data []byte, leadingZeroCount int) string {
	value := binary.BigEndian.Uint64(data)

	if value == 0 {
		return strings.Repeat(enc.zeroString, fastPathSize8)
	}

	var buffer [maxBase36Len8 + fastPathSize8]byte
	position := len(buffer)
	base := safeconv.IntToUint64(enc.base)

	for value > 0 {
		position--
		rem := value % base
		value /= base
		buffer[position] = enc.alphabetBytes[rem]
	}

	for range leadingZeroCount {
		position--
		buffer[position] = enc.zeroByte //nolint:gosec // position bounded by buffer size
	}

	return string(buffer[position:])
}

// encode16BytesFast encodes exactly 16 bytes using uint128 arithmetic.
// This is a fast path that avoids the O(n^2) generic algorithm.
//
// Takes data ([]byte) which must be exactly 16 bytes.
// Takes leadingZeroCount (int) which is the number of leading zero bytes.
//
// Returns string which is the encoded representation.
func (enc *Encoding) encode16BytesFast(data []byte, leadingZeroCount int) string {
	hi := binary.BigEndian.Uint64(data[:8])
	lo := binary.BigEndian.Uint64(data[8:])

	if hi == 0 && lo == 0 {
		return strings.Repeat(enc.zeroString, fastPathSize16)
	}

	var buffer [maxBase36Len16 + fastPathSize16]byte
	position := len(buffer)
	base := safeconv.IntToUint64(enc.base)

	for hi > 0 || lo > 0 {
		var rem uint64
		hi, lo, rem = divMod128(hi, lo, base)
		position--
		buffer[position] = enc.alphabetBytes[rem]
	}

	for range leadingZeroCount {
		position--
		buffer[position] = enc.zeroByte //nolint:gosec // position bounded by buffer size
	}

	return string(buffer[position:])
}

// canUseFastPath8 checks whether the 8-byte fast path can be used for
// encoding. This requires an ASCII alphabet, base >= 36, and the input to be
// exactly 8 bytes.
//
// Takes data ([]byte) which is the input data.
//
// Returns bool which is true if the fast path can be used.
func (enc *Encoding) canUseFastPath8(data []byte) bool {
	if !enc.isASCII || !enc.hasReverseBytes {
		return false
	}
	if enc.base < minFastPathBase {
		return false
	}
	return len(data) == fastPathSize8
}

// canUseFastPath16 checks whether the 16-byte fast path can be used for
// encoding. This requires an ASCII alphabet, base >= 36, and the input to be
// exactly 16 bytes.
//
// Takes data ([]byte) which is the input data.
//
// Returns bool which is true if the fast path can be used.
func (enc *Encoding) canUseFastPath16(data []byte) bool {
	if !enc.isASCII || !enc.hasReverseBytes {
		return false
	}
	if enc.base < minFastPathBase {
		return false
	}
	return len(data) == fastPathSize16
}

// base64Adapter wraps a standard library base64 encoding to satisfy the
// fastPathEncoder interface.
type base64Adapter struct {
	*base64.Encoding
}

// base32Adapter wraps a standard library base32 encoding to implement
// fastPathEncoder.
type base32Adapter struct {
	*base32.Encoding
}

// hexAdapter implements fastPathEncoder for hexadecimal encoding.
type hexAdapter struct{}

// EncodeToString encodes bytes to a hexadecimal string.
//
// Takes source ([]byte) which is the binary data to encode.
//
// Returns string which is the hexadecimal representation of the input.
func (hexAdapter) EncodeToString(source []byte) string {
	return hex.EncodeToString(source)
}

// DecodeString decodes a hexadecimal string to bytes.
//
// Takes s (string) which is the hexadecimal encoded input.
//
// Returns []byte which contains the decoded binary data.
// Returns error when the input contains invalid hexadecimal characters.
func (hexAdapter) DecodeString(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// isAllZeroBytes checks whether all bytes in a slice are zero.
//
// Takes data ([]byte) which is the byte slice to check.
//
// Returns bool which is true if all bytes are zero or the slice is empty,
// false otherwise.
func isAllZeroBytes(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// countLeadingZeroBytes counts the zero bytes at the start of a byte slice.
//
// Takes data ([]byte) which is the byte slice to check.
//
// Returns int which is the count of zero bytes at the start of the slice.
func countLeadingZeroBytes(data []byte) int {
	count := 0
	for _, b := range data {
		if b != 0 {
			break
		}
		count++
	}
	return count
}

// isAllZeroRunes checks if all runes match the zero rune.
//
// Takes runes ([]rune) which is the slice of runes to check.
// Takes zeroRune (rune) which is the rune to compare against.
//
// Returns bool which is true if all runes match the zero rune.
func isAllZeroRunes(runes []rune, zeroRune rune) bool {
	for _, r := range runes {
		if r != zeroRune {
			return false
		}
	}
	return true
}

// countLeadingZeroRunes counts how many runes at the start of a slice match
// the given zero rune.
//
// Takes runes ([]rune) which is the slice of runes to check.
// Takes zeroRune (rune) which is the rune to match against.
//
// Returns int which is the count of matching runes from the start.
func countLeadingZeroRunes(runes []rune, zeroRune rune) int {
	count := 0
	for _, r := range runes {
		if r != zeroRune {
			break
		}
		count++
	}
	return count
}

// isAllZeroBytesString checks whether every byte in a string matches the given
// byte value.
//
// Takes s (string) which is the string to check.
// Takes zeroByte (byte) which is the byte value to compare against.
//
// Returns bool which is true if all bytes in s equal zeroByte.
func isAllZeroBytesString(s string, zeroByte byte) bool {
	for i := range len(s) {
		if s[i] != zeroByte {
			return false
		}
	}
	return true
}

// countLeadingZeroBytesString counts the number of leading bytes that match
// the zero byte.
//
// Takes s (string) which is the string to examine.
// Takes zeroByte (byte) which is the byte value to count from the start.
//
// Returns int which is the count of leading bytes matching zeroByte.
func countLeadingZeroBytesString(s string, zeroByte byte) int {
	count := 0
	for i := range len(s) {
		if s[i] != zeroByte {
			break
		}
		count++
	}
	return count
}

// reverseBytes reverses a byte slice in place.
//
// Takes data ([]byte) which is the slice to reverse.
func reverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

// divMod128 performs division of a 128-bit number by a single-digit divisor.
//
// Takes hi (uint64) which is the upper 64 bits of the dividend.
// Takes lo (uint64) which is the lower 64 bits of the dividend.
// Takes divisor (uint64) which is the divisor (must be non-zero).
//
// Returns qHi (uint64) which is the upper 64 bits of the quotient.
// Returns qLo (uint64) which is the lower 64 bits of the quotient.
// Returns rem (uint64) which is the remainder.
func divMod128(hi, lo, divisor uint64) (qHi, qLo, rem uint64) {
	qHi = hi / divisor
	r := hi % divisor
	qLo, rem = bits.Div64(r, lo, divisor)
	return qHi, qLo, rem
}
