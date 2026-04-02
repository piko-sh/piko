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

//go:build fuzz

package encoding_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"piko.sh/piko/wdk/encoding"
)

func FuzzEncodeBytesBase58RoundTrip(f *testing.F) {

	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add(make([]byte, 16))
	f.Add([]byte{0, 0, 0, 0, 0xDE, 0xAD})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	f.Add([]byte{0xDE, 0xAD, 0xBE, 0xEF})

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", input, r)
			}
		}()

		encoded := encoding.EncodeBytesBase58(input)
		decoded, err := encoding.DecodeBytesBase58(encoded)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if !bytes.Equal(decoded, input) {
			t.Errorf("round-trip failed:\ninput:   %x\nencoded: %s\ndecoded: %x",
				input, encoded, decoded)
		}
	})
}

func FuzzEncodeBytesBase36RoundTrip(f *testing.F) {

	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add(make([]byte, 16))
	f.Add([]byte{0, 0, 0, 0, 0xDE, 0xAD})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	f.Add([]byte{0xDE, 0xAD, 0xBE, 0xEF})

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on input %x: %v", input, r)
			}
		}()

		encoded := encoding.EncodeBytesBase36(input)
		decoded, err := encoding.DecodeBytesBase36(encoded)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if !bytes.Equal(decoded, input) {
			t.Errorf("round-trip failed:\ninput:   %x\nencoded: %s\ndecoded: %x",
				input, encoded, decoded)
		}
	})
}

func FuzzEncodeUint64Base58RoundTrip(f *testing.F) {

	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(57))
	f.Add(uint64(58))
	f.Add(uint64(0xFFFFFFFFFFFFFFFF))
	f.Add(uint64(1000000000000))
	f.Add(uint64(256))
	f.Add(uint64(65535))
	f.Add(uint64(0x100000000))

	f.Fuzz(func(t *testing.T, value uint64) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on value %d: %v", value, r)
			}
		}()

		encoded := encoding.EncodeUint64Base58(value)
		decoded, err := encoding.DecodeUint64Base58(encoded)
		if err != nil {
			t.Fatalf("decode error for value %d: %v", value, err)
		}

		if decoded != value {
			t.Errorf("round-trip failed:\nvalue:   %d\nencoded: %s\ndecoded: %d",
				value, encoded, decoded)
		}
	})
}

func FuzzEncodeUint64Base36RoundTrip(f *testing.F) {

	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(35))
	f.Add(uint64(36))
	f.Add(uint64(0xFFFFFFFFFFFFFFFF))
	f.Add(uint64(1000000000000))
	f.Add(uint64(256))
	f.Add(uint64(65535))
	f.Add(uint64(0x100000000))

	f.Fuzz(func(t *testing.T, value uint64) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on value %d: %v", value, r)
			}
		}()

		encoded := encoding.EncodeUint64Base36(value)
		decoded, err := encoding.DecodeUint64Base36(encoded)
		if err != nil {
			t.Fatalf("decode error for value %d: %v", value, err)
		}

		if decoded != value {
			t.Errorf("round-trip failed:\nvalue:   %d\nencoded: %s\ndecoded: %d",
				value, encoded, decoded)
		}
	})
}

func FuzzDecodeInvalidInputBase58(f *testing.F) {

	f.Add("0OIl")
	f.Add("")
	f.Add("!!@@##")
	f.Add(strings.Repeat("z", 1000))
	f.Add("9999999999999999999999999999")
	f.Add("    ")
	f.Add("\x00\x00\x00")
	f.Add("abc\ndef")
	f.Add("🚀")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DecodeBytesBase58 panicked on input %q: %v", input, r)
			}
		}()

		_, _ = encoding.DecodeBytesBase58(input)
	})
}

func FuzzDecodeInvalidInputBase36(f *testing.F) {

	f.Add("abc")
	f.Add("")
	f.Add("!!@@##")
	f.Add(strings.Repeat("Z", 1000))
	f.Add("9999999999999999999999999999")
	f.Add("    ")
	f.Add("\x00\x00\x00")
	f.Add("ABC\nDEF")
	f.Add("🚀")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DecodeBytesBase36 panicked on input %q: %v", input, r)
			}
		}()

		_, _ = encoding.DecodeBytesBase36(input)
	})
}

func FuzzDecodeUint64InvalidInput(f *testing.F) {

	f.Add("0OIl", true)
	f.Add("", true)
	f.Add("", false)
	f.Add("!!@@##", true)
	f.Add("!!@@##", false)
	f.Add(strings.Repeat("z", 100), true)
	f.Add(strings.Repeat("Z", 100), false)
	f.Add("JPwcyDCgEuq21e1WW", true)
	f.Add("abc", false)

	f.Fuzz(func(t *testing.T, input string, useBase58 bool) {
		defer func() {
			if r := recover(); r != nil {
				encoding := "Base36"
				if useBase58 {
					encoding = "Base58"
				}
				t.Errorf("DecodeUint64%s panicked on input %q: %v", encoding, input, r)
			}
		}()

		if useBase58 {
			_, _ = encoding.DecodeUint64Base58(input)
		} else {
			_, _ = encoding.DecodeUint64Base36(input)
		}
	})
}

func FuzzCustomAlphabet(f *testing.F) {

	f.Add("01", []byte{0, 1, 2, 3})
	f.Add("0123456789", []byte{255})
	f.Add("0123456789ABCDEF", []byte{0xDE, 0xAD, 0xBE, 0xEF})
	f.Add("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", []byte{1})
	f.Add("AB", []byte{})
	f.Add("XY", []byte{0, 0, 0})

	f.Fuzz(func(t *testing.T, alphabet string, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic with alphabet %q, data %x: %v", alphabet, data, r)
			}
		}()

		runeCount := len([]rune(alphabet))

		if runeCount < 2 {
			return
		}

		if runeCount > 256 {
			return
		}

		if len(data) > 64 {
			return
		}

		seen := make(map[rune]bool)
		hasDuplicates := false
		for _, r := range alphabet {
			if seen[r] {
				hasDuplicates = true
				break
			}
			seen[r] = true
		}

		enc, err := encoding.NewEncoding(alphabet)
		if hasDuplicates {
			if err == nil {
				t.Errorf("expected error for alphabet with duplicates: %q", alphabet)
			}
			return
		}

		if err != nil {

			t.Fatalf("unexpected error for alphabet %q: %v", alphabet, err)
		}

		encoded := enc.EncodeBytes(data)
		decoded, err := enc.DecodeBytes(encoded)
		if err != nil {
			t.Fatalf("decode error for alphabet %q, data %x: %v", alphabet, data, err)
		}

		if !bytes.Equal(decoded, data) {
			t.Errorf("round-trip failed for alphabet %q:\ndata:    %x\nencoded: %s\ndecoded: %x",
				alphabet, data, encoded, decoded)
		}
	})
}

func FuzzConcurrentEncodeBase58(f *testing.F) {

	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 4)
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 8)
	f.Add([]byte{0}, 16)

	f.Fuzz(func(t *testing.T, data []byte, goroutines int) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic during concurrent encode: %v", r)
			}
		}()

		if goroutines < 1 {
			goroutines = 1
		}
		if goroutines > 32 {
			goroutines = 32
		}

		var wg sync.WaitGroup
		errors := make(chan error, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						errors <- errPanic{r}
					}
				}()

				encoded := encoding.EncodeBytesBase58(data)
				decoded, err := encoding.DecodeBytesBase58(encoded)
				if err != nil {
					errors <- err
					return
				}

				if !bytes.Equal(decoded, data) {
					errors <- errMismatch{
						expected: data,
						actual:   decoded,
					}
				}
			})
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("concurrent error: %v", err)
		}
	})
}

type errPanic struct {
	value any
}

func (e errPanic) Error() string {
	return "panic: " + stringify(e.value)
}

type errMismatch struct {
	expected []byte
	actual   []byte
}

func (e errMismatch) Error() string {
	return "mismatch: expected " + stringify(e.expected) + ", got " + stringify(e.actual)
}

func stringify(v any) string {
	switch value := v.(type) {
	case []byte:
		return string(value)
	case string:
		return value
	default:
		return "unknown"
	}
}
