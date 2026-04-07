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

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectWriter_WriteTo_String(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("hello ")
	dw.AppendString("world")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "hello world"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_WriteTo_Int(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("r.0:0:")
	dw.AppendInt(42)
	dw.AppendString(":1")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "r.0:0:42:1"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_WriteTo_Uint(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("key:")
	dw.AppendUint(18446744073709551615)
	dw.AppendString(":end")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "key:18446744073709551615:end"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_WriteTo_Float(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("value:")
	dw.AppendFloat(3.14159)

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "value:3.14159"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_WriteTo_Bool(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("enabled:")
	dw.AppendBool(true)
	dw.AppendString(",disabled:")
	dw.AppendBool(false)

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "enabled:true,disabled:false"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_WriteTo_Mixed(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("s:")
	dw.AppendInt(-42)
	dw.AppendString(",u:")
	dw.AppendUint(100)
	dw.AppendString(",f:")
	dw.AppendFloat(2.5)
	dw.AppendString(",b:")
	dw.AppendBool(true)

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "s:-42,u:100,f:2.5,b:true"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_Overflow(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	for i := range 12 {
		dw.AppendInt(int64(i))
		dw.AppendString(".")
	}

	wantLen := 24
	if got := dw.Len(); got != wantLen {
		t.Errorf("Len() = %d, want %d", got, wantLen)
	}

	var buffer []byte
	buffer = dw.WriteTo(buffer)
	want := "0.1.2.3.4.5.6.7.8.9.10.11."
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() with overflow = %q, want %q", got, want)
	}
}

func TestDirectWriter_Reset(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()

	dw.AppendString("test")
	dw.AppendInt(123)

	if dw.Len() != 2 {
		t.Fatalf("expected len 2 before reset, got %d", dw.Len())
	}

	dw.Reset()

	if dw.Len() != 0 {
		t.Errorf("expected len 0 after reset, got %d", dw.Len())
	}

	dw.AppendString("new")
	var buffer []byte
	buffer = dw.WriteTo(buffer)
	if got := string(buffer); got != "new" {
		t.Errorf("after reset and reuse, got %q, want %q", got, "new")
	}

	PutDirectWriter(dw)
}

func TestDirectWriter_Part(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("static")
	dw.AppendInt(42)
	dw.AppendUint(100)
	dw.AppendFloat(3.14)
	dw.AppendBool(true)

	tests := []struct {
		wantS    string
		index    int
		wantI    int64
		wantU    uint64
		wantF    float64
		wantType WriterPartType
		wantB    bool
	}{
		{index: 0, wantType: WriterPartString, wantS: "static"},
		{index: 1, wantType: WriterPartInt, wantI: 42},
		{index: 2, wantType: WriterPartUint, wantU: 100},
		{index: 3, wantType: WriterPartFloat, wantF: 3.14},
		{index: 4, wantType: WriterPartBool, wantB: true},
	}

	for _, tc := range tests {
		part := dw.Part(tc.index)
		require.NotNilf(t, part, "Part(%d) = nil, want non-nil", tc.index)
		if part.Type != tc.wantType {
			t.Errorf("Part(%d).Type = %v, want %v", tc.index, part.Type, tc.wantType)
		}
		if tc.wantS != "" && part.StringValue != tc.wantS {
			t.Errorf("Part(%d).StringValue = %q, want %q", tc.index, part.StringValue, tc.wantS)
		}
		if tc.wantI != 0 && part.IntValue != tc.wantI {
			t.Errorf("Part(%d).IntValue = %d, want %d", tc.index, part.IntValue, tc.wantI)
		}
		if tc.wantU != 0 && part.UintValue != tc.wantU {
			t.Errorf("Part(%d).UintValue = %d, want %d", tc.index, part.UintValue, tc.wantU)
		}
		if tc.wantF != 0 && part.FloatValue != tc.wantF {
			t.Errorf("Part(%d).FloatValue = %f, want %f", tc.index, part.FloatValue, tc.wantF)
		}
		if tc.wantB && !part.BoolValue {
			t.Errorf("Part(%d).BoolValue = %v, want %v", tc.index, part.BoolValue, tc.wantB)
		}
	}

	if part := dw.Part(-1); part != nil {
		t.Error("Part(-1) should return nil")
	}
	if part := dw.Part(100); part != nil {
		t.Error("Part(100) should return nil")
	}
}

type customID struct {
	prefix string
	value  int
}

func (c customID) String() string {
	return c.prefix + "-" + strconv.Itoa(c.value)
}

func TestDirectWriter_AppendAny_Stringer(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("key:")
	dw.AppendAny(customID{prefix: "user", value: 42})

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "key:user-42"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_AppendAny_FallbackSprint(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("val:")
	dw.AppendAny([]int{1, 2, 3})

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "val:[1 2 3]"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_AppendAny_Nil(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("before")
	dw.AppendAny(nil)
	dw.AppendString("after")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "beforeafter"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_PoolReuse(t *testing.T) {
	t.Parallel()

	dw1 := GetDirectWriter()
	dw1.AppendString("first")
	dw1.AppendInt(1)
	PutDirectWriter(dw1)

	dw2 := GetDirectWriter()

	if dw2.Len() != 0 {
		t.Errorf("Pooled writer should be reset, but Len() = %d", dw2.Len())
	}

	PutDirectWriter(dw2)
}

func TestDirectWriter_SingleStringValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		setup      func(dw *DirectWriter)
		wantString string
		wantOK     bool
	}{
		{
			name:       "single string part returns value",
			setup:      func(dw *DirectWriter) { dw.AppendString("hello") },
			wantString: "hello",
			wantOK:     true,
		},
		{
			name:       "empty writer returns false",
			setup:      func(dw *DirectWriter) {},
			wantString: "",
			wantOK:     false,
		},
		{
			name: "multiple parts returns false",
			setup: func(dw *DirectWriter) {
				dw.AppendString("hello")
				dw.AppendString("world")
			},
			wantString: "",
			wantOK:     false,
		},
		{
			name:       "single int part returns false",
			setup:      func(dw *DirectWriter) { dw.AppendInt(42) },
			wantString: "",
			wantOK:     false,
		},
		{
			name:       "single escape string part returns false",
			setup:      func(dw *DirectWriter) { dw.AppendEscapeString("hello") },
			wantString: "",
			wantOK:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dw := GetDirectWriter()
			defer PutDirectWriter(dw)

			tc.setup(dw)
			gotString, gotOK := dw.SingleStringValue()

			if gotOK != tc.wantOK {
				t.Errorf("SingleStringValue() ok = %v, want %v", gotOK, tc.wantOK)
			}
			if gotString != tc.wantString {
				t.Errorf("SingleStringValue() str = %q, want %q", gotString, tc.wantString)
			}
		})
	}
}

func TestDirectWriter_SingleStringValue_Nil(t *testing.T) {
	t.Parallel()

	var dw *DirectWriter
	str, ok := dw.SingleStringValue()
	if ok {
		t.Error("SingleStringValue() on nil should return false")
	}
	if str != "" {
		t.Errorf("SingleStringValue() on nil should return empty string, got %q", str)
	}
}

func TestPutDirectWriter_Nil(t *testing.T) {
	t.Parallel()

	PutDirectWriter(nil)
}

func TestDirectWriter_Clone_Nil(t *testing.T) {
	t.Parallel()

	var dw *DirectWriter
	clone := dw.Clone()
	if clone != nil {
		t.Error("Clone() on nil should return nil")
	}
}

func TestDirectWriter_Clone_Basic(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.Name = "test-writer"
	dw.AppendString("hello ")
	dw.AppendInt(42)
	dw.AppendString(" world")

	clone := dw.Clone()
	defer PutDirectWriter(clone)

	if clone.Name != "test-writer" {
		t.Errorf("Clone().Name = %q, want %q", clone.Name, "test-writer")
	}

	var origBuf, cloneBuf []byte
	origBuf = dw.WriteTo(origBuf)
	cloneBuf = clone.WriteTo(cloneBuf)

	if string(origBuf) != string(cloneBuf) {
		t.Errorf("Clone content mismatch: original = %q, clone = %q", string(origBuf), string(cloneBuf))
	}
}

func TestDirectWriter_Clone_BytesValue_DeepCopy(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()

	bufferPointer := GetByteBuf()
	*bufferPointer = append(*bufferPointer, "test-payload-data"...)
	dw.AppendPooledBytes(bufferPointer)

	clone := dw.Clone()

	var cloneBufBefore []byte
	cloneBufBefore = clone.WriteTo(cloneBufBefore)

	PutDirectWriter(dw)

	reusedBuf := GetByteBuf()
	*reusedBuf = append(*reusedBuf, "CORRUPTED-DATA-XXX"...)
	PutByteBuf(reusedBuf)

	var cloneBufAfter []byte
	cloneBufAfter = clone.WriteTo(cloneBufAfter)

	if string(cloneBufBefore) != string(cloneBufAfter) {
		t.Errorf("Clone BytesValue was corrupted after original reset: before = %q, after = %q",
			string(cloneBufBefore), string(cloneBufAfter))
	}

	if string(cloneBufAfter) != "test-payload-data" {
		t.Errorf("Clone BytesValue has wrong content: got %q, want %q",
			string(cloneBufAfter), "test-payload-data")
	}

	PutDirectWriter(clone)
}

func TestDirectWriter_Clone_EscapeBytesValue_DeepCopy(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()

	bufferPointer := GetByteBuf()
	*bufferPointer = append(*bufferPointer, "<script>alert('xss')</script>"...)
	dw.AppendEscapePooledBytes(bufferPointer)

	clone := dw.Clone()

	var cloneBufBefore []byte
	cloneBufBefore = clone.WriteTo(cloneBufBefore)

	PutDirectWriter(dw)

	reusedBuf := GetByteBuf()
	*reusedBuf = append(*reusedBuf, "CORRUPTED-ESCAPE-DATA"...)
	PutByteBuf(reusedBuf)

	var cloneBufAfter []byte
	cloneBufAfter = clone.WriteTo(cloneBufAfter)

	if string(cloneBufBefore) != string(cloneBufAfter) {
		t.Errorf("Clone EscapeBytesValue was corrupted after original reset: before = %q, after = %q",
			string(cloneBufBefore), string(cloneBufAfter))
	}

	PutDirectWriter(clone)
}

func TestDirectWriter_Clone_MixedParts(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()

	dw.AppendString("class=\"")

	bufferPointer := GetByteBuf()
	*bufferPointer = append(*bufferPointer, "btn-primary"...)
	dw.AppendPooledBytes(bufferPointer)

	dw.AppendString(" active")
	dw.AppendInt(42)
	dw.AppendString("\"")

	clone := dw.Clone()

	var origBuf, cloneBuf []byte
	origBuf = dw.WriteTo(origBuf)
	cloneBuf = clone.WriteTo(cloneBuf)

	if string(origBuf) != string(cloneBuf) {
		t.Errorf("Clone content mismatch: original = %q, clone = %q", string(origBuf), string(cloneBuf))
	}

	PutDirectWriter(dw)

	var cloneBufAfter []byte
	cloneBufAfter = clone.WriteTo(cloneBufAfter)

	if string(cloneBuf) != string(cloneBufAfter) {
		t.Errorf("Clone was corrupted after original reset: before = %q, after = %q",
			string(cloneBuf), string(cloneBufAfter))
	}

	PutDirectWriter(clone)
}

func BenchmarkDirectWriter_WriteTo_String(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("r.0:0:1:2:0:0.")
	dw.AppendString("static-key")
	dw.AppendString(":suffix")

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_Int(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("r.0:0:1:2:0:0.")
	dw.AppendInt(12345)
	dw.AppendString(":0.")
	dw.AppendInt(7)

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_Uint(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("key:")
	dw.AppendUint(18446744073709551615)
	dw.AppendString(":end")

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_Float(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("value:")
	dw.AppendFloat(3.14159265358979)

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_Bool(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("enabled:")
	dw.AppendBool(true)

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_Mixed(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("r.")
	dw.AppendInt(42)
	dw.AppendString(":")
	dw.AppendUint(100)
	dw.AppendString(":")
	dw.AppendFloat(3.14)
	dw.AppendString(":")
	dw.AppendBool(true)

	var buffer [128]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_GetPut(b *testing.B) {
	var i int64
	for b.Loop() {
		dw := GetDirectWriter()
		dw.AppendString("prefix.")
		dw.AppendInt(i)
		PutDirectWriter(dw)
		i++
	}
}

func BenchmarkStringConcat_Baseline(b *testing.B) {

	for b.Loop() {
		_ = "r.0:0:1:2:0:0." + strconv.FormatInt(12345, 10) + ":0." + strconv.FormatInt(7, 10)
	}
}

func BenchmarkStringConcat_Uint_Baseline(b *testing.B) {

	for b.Loop() {
		_ = "key:" + strconv.FormatUint(18446744073709551615, 10) + ":end"
	}
}

func BenchmarkStringConcat_Float_Baseline(b *testing.B) {

	for b.Loop() {
		_ = "value:" + strconv.FormatFloat(3.14159265358979, 'f', -1, 64)
	}
}

func TestDirectWriter_SetName(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	result := dw.SetName("title")
	if result != dw {
		t.Error("SetName should return self for chaining")
	}

	if dw.Name != "title" {
		t.Errorf("Name = %q, want %q", dw.Name, "title")
	}
}

func TestDirectWriter_Reset_ClearsName(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.SetName("href")
	dw.AppendString("https://example.com")

	dw.Reset()

	if dw.Name != "" {
		t.Errorf("Name after Reset() = %q, want empty string", dw.Name)
	}
}

func TestDirectWriter_AppendEscapeString(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendEscapeString("<script>alert('xss')</script>")

	if dw.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", dw.Len())
	}

	part := dw.Part(0)
	require.NotNil(t, part, "Part(0) returned nil")
	if part.Type != WriterPartEscapeString {
		t.Errorf("Part(0).Type = %v, want WriterPartEscapeString", part.Type)
	}
	if part.StringValue != "<script>alert('xss')</script>" {
		t.Errorf("Part(0).StringValue = %q, want raw unescaped string", part.StringValue)
	}
}

func TestDirectWriter_WriteTo_EscapeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "escapes less than",
			input: "1 < 2",
			want:  "1 &lt; 2",
		},
		{
			name:  "escapes greater than",
			input: "2 > 1",
			want:  "2 &gt; 1",
		},
		{
			name:  "escapes ampersand",
			input: "A & B",
			want:  "A &amp; B",
		},
		{
			name:  "escapes double quote",
			input: `say "hello"`,
			want:  "say &quot;hello&quot;",
		},
		{
			name:  "escapes single quote",
			input: "it's fine",
			want:  "it&#39;s fine",
		},
		{
			name:  "escapes XSS payload",
			input: "<script>alert('xss')</script>",
			want:  "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:  "mixed safe and unsafe",
			input: "Hello <b>World</b> & \"Friends\"",
			want:  "Hello &lt;b&gt;World&lt;/b&gt; &amp; &quot;Friends&quot;",
		},
		{
			name:  "no escaping needed",
			input: "plain text 123",
			want:  "plain text 123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dw := GetDirectWriter()
			defer PutDirectWriter(dw)

			dw.AppendEscapeString(tc.input)

			var buffer []byte
			buffer = dw.WriteTo(buffer)

			if got := string(buffer); got != tc.want {
				t.Errorf("WriteTo() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDirectWriter_WriteTo_MixedEscapeAndNonEscape(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("<div>")
	dw.AppendEscapeString("<user>")
	dw.AppendString("</div>")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "<div>&lt;user&gt;</div>"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_String_Basic(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("hello ")
	dw.AppendInt(42)

	got := dw.String()
	want := "hello 42"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestDirectWriter_String_Caching(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("cached")
	dw.AppendInt(123)

	first := dw.String()

	second := dw.String()

	if first != second {
		t.Errorf("String() not consistent: first=%q, second=%q", first, second)
	}

	if !dw.hasCachedString {
		t.Error("hasCachedString should be true after String() call")
	}

	if dw.cachedString != first {
		t.Error("cachedString should equal returned string")
	}
}

func TestDirectWriter_String_Empty(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	got := dw.String()
	if got != "" {
		t.Errorf("String() on empty writer = %q, want empty string", got)
	}
}

func TestDirectWriter_String_Nil(t *testing.T) {
	t.Parallel()

	var dw *DirectWriter = nil
	got := dw.String()
	if got != "" {
		t.Errorf("String() on nil writer = %q, want empty string", got)
	}
}

func TestDirectWriter_Reset_ClearsCachedString(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("original")
	_ = dw.String()

	if !dw.hasCachedString {
		t.Fatal("hasCachedString should be true before Reset")
	}

	dw.Reset()

	if dw.hasCachedString {
		t.Error("hasCachedString should be false after Reset")
	}
	if dw.cachedString != "" {
		t.Error("cachedString should be empty after Reset")
	}
}

func TestDirectWriter_String_WithEscaping(t *testing.T) {
	t.Parallel()

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("Title: ")
	dw.AppendEscapeString("<script>")

	got := dw.String()
	want := "Title: &lt;script&gt;"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func BenchmarkDirectWriter_WriteTo_EscapeString(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendEscapeString("<script>alert('xss')</script>")

	var buffer [128]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_EscapeString_NoEscaping(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendEscapeString("plain text without special characters")

	var buffer [128]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_String_Cached(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("prefix:")
	dw.AppendInt(12345)
	dw.AppendString(":suffix")
	_ = dw.String()

	b.ResetTimer()
	for b.Loop() {
		_ = dw.String()
	}
}

func BenchmarkDirectWriter_String_Uncached(b *testing.B) {
	for b.Loop() {
		dw := GetDirectWriter()
		dw.AppendString("prefix:")
		dw.AppendInt(12345)
		dw.AppendString(":suffix")
		_ = dw.String()
		PutDirectWriter(dw)
	}
}

func TestDirectWriter_AppendFNVString(t *testing.T) {
	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendFNVString("hello")

	if dw.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", dw.Len())
	}

	part := dw.Part(0)
	require.NotNil(t, part, "Part(0) returned nil")
	if part.Type != WriterPartFNVString {
		t.Errorf("Part(0).Type = %v, want WriterPartFNVString", part.Type)
	}
	if part.StringValue != "hello" {
		t.Errorf("Part(0).StringValue = %q, want %q", part.StringValue, "hello")
	}
}

func TestDirectWriter_WriteTo_FNVString(t *testing.T) {
	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("r.0:1:0.")
	dw.AppendFNVString("user-provided-value")
	dw.AppendString(":2:0")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	result := string(buffer)

	if len(result) != len("r.0:1:0.")+8+len(":2:0") {
		t.Errorf("WriteTo() = %q (len %d), expected len %d", result, len(result), len("r.0:1:0.")+8+len(":2:0"))
	}

	hashPart := result[len("r.0:1:0.") : len(result)-len(":2:0")]
	if len(hashPart) != 8 {
		t.Errorf("FNV hash part = %q (len %d), want 8 chars", hashPart, len(hashPart))
	}
	for _, c := range hashPart {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("FNV hash contains non-hex char %q", c)
		}
	}

	dw2 := GetDirectWriter()
	defer PutDirectWriter(dw2)
	dw2.AppendString("r.0:1:0.")
	dw2.AppendFNVString("user-provided-value")
	dw2.AppendString(":2:0")
	buf2 := dw2.WriteTo(nil)

	if string(buffer) != string(buf2) {
		t.Errorf("FNV hash not deterministic: %q vs %q", buffer, buf2)
	}
}

func TestDirectWriter_WriteTo_FNVFloat(t *testing.T) {
	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("r.0:1:0.")
	dw.AppendFNVFloat(3.14159)
	dw.AppendString(":2:0")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	result := string(buffer)

	hashPart := result[len("r.0:1:0.") : len(result)-len(":2:0")]
	if len(hashPart) != 8 {
		t.Errorf("FNV hash part = %q (len %d), want 8 chars", hashPart, len(hashPart))
	}

	for _, c := range hashPart {
		if c == '.' {
			t.Error("FNV hash of float should NOT contain decimal point")
		}
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("FNV hash contains non-hex char %q", c)
		}
	}
}

func TestDirectWriter_WriteTo_FNVFloat_Precision(t *testing.T) {
	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendFNVFloat(0.1 + 0.2)
	result1 := string(dw.WriteTo(nil))

	dw2 := GetDirectWriter()
	defer PutDirectWriter(dw2)

	dw2.AppendFNVFloat(0.1 + 0.2)
	result2 := string(dw2.WriteTo(nil))

	if result1 != result2 {
		t.Errorf("Same float calculation produced different hashes: %q vs %q", result1, result2)
	}
}

func TestDirectWriter_WriteTo_FNVAny(t *testing.T) {
	type customStruct struct {
		Name string
		ID   int
	}

	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("r.0:1:0.")
	dw.AppendFNVAny(customStruct{Name: "test", ID: 42})
	dw.AppendString(":2:0")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	result := string(buffer)

	hashPart := result[len("r.0:1:0.") : len(result)-len(":2:0")]
	if len(hashPart) != 8 {
		t.Errorf("FNV hash part = %q (len %d), want 8 chars", hashPart, len(hashPart))
	}
}

func TestDirectWriter_WriteTo_FNVAny_Nil(t *testing.T) {
	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendString("before")
	dw.AppendFNVAny(nil)
	dw.AppendString("after")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	want := "beforeafter"
	if got := string(buffer); got != want {
		t.Errorf("WriteTo() = %q, want %q", got, want)
	}
}

func TestDirectWriter_WriteTo_FNVString_HTMLSpecialChars(t *testing.T) {
	dw := GetDirectWriter()
	defer PutDirectWriter(dw)

	dw.AppendFNVString("<script>alert('xss')</script>")

	var buffer []byte
	buffer = dw.WriteTo(buffer)

	result := string(buffer)

	if len(result) != 8 {
		t.Errorf("FNV hash = %q (len %d), want 8 chars", result, len(result))
	}
	for _, c := range result {
		if c == '<' || c == '>' || c == '\'' || c == '"' || c == '&' {
			t.Errorf("FNV hash should not contain HTML special char %q", c)
		}
	}
}

func TestDirectWriter_PKeyWithFNV_DifferentInputsDifferentHashes(t *testing.T) {
	dw1 := GetDirectWriter()
	defer PutDirectWriter(dw1)
	dw1.AppendFNVString("value1")
	result1 := string(dw1.WriteTo(nil))

	dw2 := GetDirectWriter()
	defer PutDirectWriter(dw2)
	dw2.AppendFNVString("value2")
	result2 := string(dw2.WriteTo(nil))

	if result1 == result2 {
		t.Errorf("Different inputs produced same hash: %q", result1)
	}
}

func BenchmarkDirectWriter_WriteTo_FNVString(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("r.0:1:0.")
	dw.AppendFNVString("user-provided-dynamic-value")
	dw.AppendString(":2:0")

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_FNVFloat(b *testing.B) {
	dw := GetDirectWriter()
	dw.AppendString("r.0:1:0.")
	dw.AppendFNVFloat(3.14159265358979)
	dw.AppendString(":2:0")

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func BenchmarkDirectWriter_WriteTo_FNVAny(b *testing.B) {
	type data struct {
		Name string
		ID   int
	}
	value := data{Name: "test", ID: 42}

	dw := GetDirectWriter()
	dw.AppendString("r.0:1:0.")
	dw.AppendFNVAny(value)
	dw.AppendString(":2:0")

	var buffer [64]byte
	b.ResetTimer()
	for b.Loop() {
		_ = dw.WriteTo(buffer[:0])
	}
}

func TestDirectWriter_StringRaw(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		setup func(dw *DirectWriter)
		want  string
	}{
		{
			name:  "nil writer returns empty string",
			setup: nil,
			want:  "",
		},
		{
			name:  "empty writer returns empty string",
			setup: func(dw *DirectWriter) {},
			want:  "",
		},
		{
			name: "plain strings are returned as-is",
			setup: func(dw *DirectWriter) {
				dw.AppendString("hello ")
				dw.AppendString("world")
			},
			want: "hello world",
		},
		{
			name: "escape string parts are NOT escaped",
			setup: func(dw *DirectWriter) {
				dw.AppendString("<div>")
				dw.AppendEscapeString("<user>")
				dw.AppendString("</div>")
			},
			want: "<div><user></div>",
		},
		{
			name: "mixed types without escaping",
			setup: func(dw *DirectWriter) {
				dw.AppendString("count:")
				dw.AppendInt(42)
				dw.AppendString(",pi:")
				dw.AppendFloat(3.14)
				dw.AppendString(",ok:")
				dw.AppendBool(true)
			},
			want: "count:42,pi:3.14,ok:true",
		},
		{
			name: "escape bytes are NOT escaped",
			setup: func(dw *DirectWriter) {
				bufferPointer := GetByteBuf()
				*bufferPointer = append(*bufferPointer, "<b>bold</b>"...)
				dw.AppendEscapePooledBytes(bufferPointer)
			},
			want: "<b>bold</b>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.setup == nil {
				var dw *DirectWriter
				got := dw.StringRaw()
				if got != tc.want {
					t.Errorf("StringRaw() = %q, want %q", got, tc.want)
				}
				return
			}

			dw := GetDirectWriter()
			defer PutDirectWriter(dw)

			tc.setup(dw)
			got := dw.StringRaw()
			if got != tc.want {
				t.Errorf("StringRaw() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDirectWriter_SingleBytesValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		setup    func(dw *DirectWriter)
		wantData string
		wantOK   bool
	}{
		{
			name: "single WriterPartBytes returns value",
			setup: func(dw *DirectWriter) {
				bufferPointer := GetByteBuf()
				*bufferPointer = append(*bufferPointer, "payload"...)
				dw.AppendPooledBytes(bufferPointer)
			},
			wantData: "payload",
			wantOK:   true,
		},
		{
			name: "single WriterPartEscapeBytes returns value",
			setup: func(dw *DirectWriter) {
				bufferPointer := GetByteBuf()
				*bufferPointer = append(*bufferPointer, "<html>"...)
				dw.AppendEscapePooledBytes(bufferPointer)
			},
			wantData: "<html>",
			wantOK:   true,
		},
		{
			name:   "empty writer returns nil and false",
			setup:  func(dw *DirectWriter) {},
			wantOK: false,
		},
		{
			name: "multiple parts returns nil and false",
			setup: func(dw *DirectWriter) {
				bufferPointer1 := GetByteBuf()
				*bufferPointer1 = append(*bufferPointer1, "a"...)
				dw.AppendPooledBytes(bufferPointer1)
				bufferPointer2 := GetByteBuf()
				*bufferPointer2 = append(*bufferPointer2, "b"...)
				dw.AppendPooledBytes(bufferPointer2)
			},
			wantOK: false,
		},
		{
			name: "single string part returns nil and false",
			setup: func(dw *DirectWriter) {
				dw.AppendString("not bytes")
			},
			wantOK: false,
		},
		{
			name: "single int part returns nil and false",
			setup: func(dw *DirectWriter) {
				dw.AppendInt(42)
			},
			wantOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dw := GetDirectWriter()
			defer PutDirectWriter(dw)

			tc.setup(dw)
			gotBytes, gotOK := dw.SingleBytesValue()

			if gotOK != tc.wantOK {
				t.Errorf("SingleBytesValue() ok = %v, want %v", gotOK, tc.wantOK)
			}
			if tc.wantOK && string(gotBytes) != tc.wantData {
				t.Errorf("SingleBytesValue() bytes = %q, want %q", gotBytes, tc.wantData)
			}
			if !tc.wantOK && gotBytes != nil {
				t.Errorf("SingleBytesValue() bytes = %v, want nil", gotBytes)
			}
		})
	}

	t.Run("nil writer returns nil and false", func(t *testing.T) {
		t.Parallel()

		var dw *DirectWriter
		gotBytes, gotOK := dw.SingleBytesValue()
		if gotOK {
			t.Error("SingleBytesValue() on nil should return false")
		}
		if gotBytes != nil {
			t.Error("SingleBytesValue() on nil should return nil bytes")
		}
	})
}

func TestDirectWriter_RenderedLen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setup func(dw *DirectWriter)
		name  string
		want  int
	}{
		{
			name:  "empty writer returns 0",
			setup: func(dw *DirectWriter) {},
			want:  0,
		},
		{
			name: "single string part",
			setup: func(dw *DirectWriter) {
				dw.AppendString("hello")
			},
			want: 5,
		},
		{
			name: "string + int",
			setup: func(dw *DirectWriter) {
				dw.AppendString("n=")
				dw.AppendInt(42)
			},
			want: 4,
		},
		{
			name: "string + uint",
			setup: func(dw *DirectWriter) {
				dw.AppendString("u=")
				dw.AppendUint(100)
			},
			want: 5,
		},
		{
			name: "bool true",
			setup: func(dw *DirectWriter) {
				dw.AppendBool(true)
			},
			want: 4,
		},
		{
			name: "bool false",
			setup: func(dw *DirectWriter) {
				dw.AppendBool(false)
			},
			want: 5,
		},
		{
			name: "float uses estimated length",
			setup: func(dw *DirectWriter) {
				dw.AppendFloat(3.14)
			},
			want: 16,
		},
		{
			name: "escape string uses string length",
			setup: func(dw *DirectWriter) {
				dw.AppendEscapeString("<div>")
			},
			want: 5,
		},
		{
			name: "bytes part",
			setup: func(dw *DirectWriter) {
				bufferPointer := GetByteBuf()
				*bufferPointer = append(*bufferPointer, "abc"...)
				dw.AppendPooledBytes(bufferPointer)
			},
			want: 3,
		},
		{
			name: "FNV string returns 8",
			setup: func(dw *DirectWriter) {
				dw.AppendFNVString("test")
			},
			want: 8,
		},
		{
			name: "FNV float returns 8",
			setup: func(dw *DirectWriter) {
				dw.AppendFNVFloat(3.14)
			},
			want: 8,
		},
		{
			name: "negative int",
			setup: func(dw *DirectWriter) {
				dw.AppendInt(-123)
			},
			want: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dw := GetDirectWriter()
			defer PutDirectWriter(dw)

			tc.setup(dw)
			got := dw.RenderedLen()
			if got != tc.want {
				t.Errorf("RenderedLen() = %d, want %d", got, tc.want)
			}
		})
	}

	t.Run("nil writer returns 0", func(t *testing.T) {
		t.Parallel()

		var dw *DirectWriter
		if got := dw.RenderedLen(); got != 0 {
			t.Errorf("RenderedLen() on nil = %d, want 0", got)
		}
	})
}

func TestIntLen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value int64
		want  int
	}{
		{name: "zero", value: 0, want: 1},
		{name: "single digit", value: 7, want: 1},
		{name: "two digits", value: 42, want: 2},
		{name: "three digits", value: 123, want: 3},
		{name: "large number", value: 1234567890, want: 10},
		{name: "negative single digit", value: -5, want: 2},
		{name: "negative multi digit", value: -42, want: 3},
		{name: "negative large", value: -1234567890, want: 11},
		{name: "max int64", value: 9223372036854775807, want: 19},
		{name: "one", value: 1, want: 1},
		{name: "ten", value: 10, want: 2},
		{name: "hundred", value: 100, want: 3},
		{name: "power of ten", value: 1000000, want: 7},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := intLen(tc.value)
			if got != tc.want {
				t.Errorf("intLen(%d) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

func TestUintLen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value uint64
		want  int
	}{
		{name: "zero", value: 0, want: 1},
		{name: "single digit", value: 7, want: 1},
		{name: "two digits", value: 42, want: 2},
		{name: "three digits", value: 999, want: 3},
		{name: "large number", value: 1234567890, want: 10},
		{name: "max uint64", value: 18446744073709551615, want: 20},
		{name: "one", value: 1, want: 1},
		{name: "ten", value: 10, want: 2},
		{name: "power of ten", value: 1000000, want: 7},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := uintLen(tc.value)
			if got != tc.want {
				t.Errorf("uintLen(%d) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

func TestDirectWriter_WriteToRaw(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		setup func(dw *DirectWriter)
		want  string
	}{
		{
			name: "string parts are unescaped",
			setup: func(dw *DirectWriter) {
				dw.AppendString("hello world")
			},
			want: "hello world",
		},
		{
			name: "escape string parts are NOT escaped",
			setup: func(dw *DirectWriter) {
				dw.AppendEscapeString("<script>alert('xss')</script>")
			},
			want: "<script>alert('xss')</script>",
		},
		{
			name: "mixed escape and non-escape strings",
			setup: func(dw *DirectWriter) {
				dw.AppendString("<div>")
				dw.AppendEscapeString("<user & 'friends'>")
				dw.AppendString("</div>")
			},
			want: "<div><user & 'friends'></div>",
		},
		{
			name: "int part",
			setup: func(dw *DirectWriter) {
				dw.AppendInt(-42)
			},
			want: "-42",
		},
		{
			name: "uint part",
			setup: func(dw *DirectWriter) {
				dw.AppendUint(100)
			},
			want: "100",
		},
		{
			name: "float part",
			setup: func(dw *DirectWriter) {
				dw.AppendFloat(3.14)
			},
			want: "3.14",
		},
		{
			name: "bool part",
			setup: func(dw *DirectWriter) {
				dw.AppendBool(true)
			},
			want: "true",
		},
		{
			name: "escape bytes are NOT escaped",
			setup: func(dw *DirectWriter) {
				bufferPointer := GetByteBuf()
				*bufferPointer = append(*bufferPointer, "<b>bold</b>"...)
				dw.AppendEscapePooledBytes(bufferPointer)
			},
			want: "<b>bold</b>",
		},
		{
			name: "bytes part",
			setup: func(dw *DirectWriter) {
				bufferPointer := GetByteBuf()
				*bufferPointer = append(*bufferPointer, "raw bytes"...)
				dw.AppendPooledBytes(bufferPointer)
			},
			want: "raw bytes",
		},
		{
			name: "mixed types all rendered correctly",
			setup: func(dw *DirectWriter) {
				dw.AppendString("s:")
				dw.AppendInt(1)
				dw.AppendString(",u:")
				dw.AppendUint(2)
				dw.AppendString(",f:")
				dw.AppendFloat(3.5)
				dw.AppendString(",b:")
				dw.AppendBool(false)
				dw.AppendString(",e:")
				dw.AppendEscapeString("<em>")
			},
			want: "s:1,u:2,f:3.5,b:false,e:<em>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dw := GetDirectWriter()
			defer PutDirectWriter(dw)

			tc.setup(dw)
			var buffer []byte
			buffer = dw.WriteToRaw(buffer)
			if got := string(buffer); got != tc.want {
				t.Errorf("WriteToRaw() = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("compare WriteTo vs WriteToRaw for escape string", func(t *testing.T) {
		t.Parallel()

		dw := GetDirectWriter()
		defer PutDirectWriter(dw)

		dw.AppendEscapeString("<b>test</b>")

		escaped := string(dw.WriteTo(nil))
		raw := string(dw.WriteToRaw(nil))

		if escaped == raw {
			t.Error("WriteTo and WriteToRaw should differ for escape string parts")
		}
		if raw != "<b>test</b>" {
			t.Errorf("WriteToRaw() = %q, want %q", raw, "<b>test</b>")
		}
	})
}
