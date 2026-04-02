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

package colour

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func withColourEnabled(t *testing.T, value bool) {
	t.Helper()
	previous := Enabled()
	SetEnabled(value)
	t.Cleanup(func() { SetEnabled(previous) })
}

func TestNew_SingleAttribute(t *testing.T) {
	t.Parallel()
	style := New(FgRed)
	assert.Equal(t, []byte("\x1b[31m"), style.sequence)
}

func TestNew_MultipleAttributes(t *testing.T) {
	t.Parallel()
	style := New(FgRed, Bold)
	assert.Equal(t, []byte("\x1b[31;1m"), style.sequence)
}

func TestNew_ThreeAttributes(t *testing.T) {
	t.Parallel()
	style := New(FgHiBlack, Bold, Italic)
	assert.Equal(t, []byte("\x1b[90;1;3m"), style.sequence)
}

func TestWriteStart_Enabled(t *testing.T) {
	withColourEnabled(t, true)

	var builder strings.Builder
	style := New(FgGreen)
	style.WriteStart(&builder)
	assert.Equal(t, "\x1b[32m", builder.String())
}

func TestWriteStart_Disabled(t *testing.T) {
	withColourEnabled(t, false)

	var builder strings.Builder
	style := New(FgGreen)
	style.WriteStart(&builder)
	assert.Empty(t, builder.String())
}

func TestWriteReset_Enabled(t *testing.T) {
	withColourEnabled(t, true)

	var builder strings.Builder
	style := New(FgGreen)
	style.WriteReset(&builder)
	assert.Equal(t, "\x1b[0m", builder.String())
}

func TestWriteReset_Disabled(t *testing.T) {
	withColourEnabled(t, false)

	var builder strings.Builder
	style := New(FgGreen)
	style.WriteReset(&builder)
	assert.Empty(t, builder.String())
}

func TestWriteStartReset_RoundTrip(t *testing.T) {
	withColourEnabled(t, true)

	var builder strings.Builder
	style := New(FgCyan, Bold)
	style.WriteStart(&builder)
	builder.WriteString("hello")
	style.WriteReset(&builder)
	assert.Equal(t, "\x1b[36;1mhello\x1b[0m", builder.String())
}

func TestSprint_Enabled(t *testing.T) {
	withColourEnabled(t, true)

	style := New(FgRed)
	result := style.Sprint("error")
	assert.Equal(t, "\x1b[31merror\x1b[0m", result)
}

func TestSprint_Disabled(t *testing.T) {
	withColourEnabled(t, false)

	style := New(FgRed)
	result := style.Sprint("error")
	assert.Equal(t, "error", result)
}

func TestSprint_MultipleArgs(t *testing.T) {
	withColourEnabled(t, true)

	style := New(FgYellow)
	result := style.Sprint("hello", " ", "world")
	assert.Equal(t, "\x1b[33mhello world\x1b[0m", result)
}

func TestSprintf_Enabled(t *testing.T) {
	withColourEnabled(t, true)

	style := New(FgBlue, Bold)
	result := style.Sprintf("found %d issues", 42)
	assert.Equal(t, "\x1b[34;1mfound 42 issues\x1b[0m", result)
}

func TestSprintf_Disabled(t *testing.T) {
	withColourEnabled(t, false)

	style := New(FgBlue, Bold)
	result := style.Sprintf("found %d issues", 42)
	assert.Equal(t, "found 42 issues", result)
}

func TestSprintFunc_Enabled(t *testing.T) {
	withColourEnabled(t, true)

	colourCyan := New(FgCyan, Bold).SprintFunc()
	result := colourCyan("Piko")
	assert.Equal(t, "\x1b[36;1mPiko\x1b[0m", result)
}

func TestSprintFunc_Disabled(t *testing.T) {
	withColourEnabled(t, false)

	colourCyan := New(FgCyan, Bold).SprintFunc()
	result := colourCyan("Piko")
	assert.Equal(t, "Piko", result)
}

func TestSetEnabled_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		for range 1000 {
			SetEnabled(true)
		}
	}()

	go func() {
		defer waitGroup.Done()
		for range 1000 {
			_ = Enabled()
		}
	}()

	waitGroup.Wait()
}

func TestAllUsedAttributes(t *testing.T) {
	t.Parallel()

	withColourEnabled(t, true)

	attributes := []struct {
		attribute Attribute
		expected  string
	}{
		{Reset, "0"},
		{Bold, "1"},
		{Faint, "2"},
		{Italic, "3"},
		{FgBlack, "30"},
		{FgRed, "31"},
		{FgGreen, "32"},
		{FgYellow, "33"},
		{FgBlue, "34"},
		{FgMagenta, "35"},
		{FgCyan, "36"},
		{FgWhite, "37"},
		{FgHiBlack, "90"},
		{FgHiRed, "91"},
		{FgHiGreen, "92"},
		{FgHiYellow, "93"},
		{FgHiBlue, "94"},
		{FgHiMagenta, "95"},
		{FgHiWhite, "97"},
	}

	for _, testCase := range attributes {
		style := New(testCase.attribute)
		expected := "\x1b[" + testCase.expected + "m"
		assert.Equal(t, expected, string(style.sequence), "attribute %d", testCase.attribute)
	}
}

func BenchmarkWriteStartReset(b *testing.B) {
	SetEnabled(true)
	defer SetEnabled(false)

	style := New(FgRed, Bold)
	var builder strings.Builder
	builder.Grow(256)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.Reset()
		style.WriteStart(&builder)
		builder.WriteString("error message")
		style.WriteReset(&builder)
	}
}

func BenchmarkSprint(b *testing.B) {
	SetEnabled(true)
	defer SetEnabled(false)

	style := New(FgRed, Bold)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = style.Sprint("error message")
	}
}

func BenchmarkSprintDisabled(b *testing.B) {
	SetEnabled(false)

	style := New(FgRed, Bold)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = style.Sprint("error message")
	}
}
