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
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Attribute is an SGR (Select Graphic Rendition) parameter for ANSI escape
// sequences. Combine multiple attributes with [New] to create a [Style].
type Attribute uint8

const (
	// Reset clears all attributes.
	Reset Attribute = 0

	// Bold increases the font weight.
	Bold Attribute = 1

	// Faint decreases the font weight (dim text).
	Faint Attribute = 2

	// Italic renders text in italic where supported.
	Italic Attribute = 3

	// FgBlack sets the foreground to black.
	FgBlack Attribute = 30

	// FgRed sets the foreground to red.
	FgRed Attribute = 31

	// FgGreen sets the foreground to green.
	FgGreen Attribute = 32

	// FgYellow sets the foreground to yellow.
	FgYellow Attribute = 33

	// FgBlue sets the foreground to blue.
	FgBlue Attribute = 34

	// FgMagenta sets the foreground to magenta.
	FgMagenta Attribute = 35

	// FgCyan sets the foreground to cyan.
	FgCyan Attribute = 36

	// FgWhite sets the foreground to white.
	FgWhite Attribute = 37

	// FgHiBlack sets the foreground to bright black (grey).
	FgHiBlack Attribute = 90

	// FgHiRed sets the foreground to bright red.
	FgHiRed Attribute = 91

	// FgHiGreen sets the foreground to bright green.
	FgHiGreen Attribute = 92

	// FgHiYellow sets the foreground to bright yellow.
	FgHiYellow Attribute = 93

	// FgHiBlue sets the foreground to bright blue.
	FgHiBlue Attribute = 94

	// FgHiMagenta sets the foreground to bright magenta.
	FgHiMagenta Attribute = 95

	// FgHiWhite sets the foreground to bright white.
	FgHiWhite Attribute = 97
)

const (
	// escape holds the ANSI escape sequence prefix.
	escape = "\x1b["

	// suffix holds the ANSI escape sequence terminator.
	suffix = "m"
)

// resetSequence holds the pre-computed ANSI escape bytes that clear all active attributes.
var resetSequence = []byte(escape + "0" + suffix)

// Style holds pre-computed ANSI escape sequences for zero-allocation colour
// output. Create once with [New], reuse for every write.
//
// Style is a value type. The backing byte slices are never mutated after
// construction, so copies share the same data safely.
type Style struct {
	// sequence holds the pre-computed ANSI escape byte sequence.
	sequence []byte
}

// New creates a Style with pre-computed ANSI byte sequences for the given
// attributes. The escape sequence is built once at construction time; all
// subsequent writes are zero-allocation.
//
// Takes attributes (...[Attribute]) which specifies the SGR parameters to
// combine.
//
// Returns Style which holds the pre-computed sequences.
func New(attributes ...Attribute) Style {
	parts := make([]string, len(attributes))
	for i, attribute := range attributes {
		parts[i] = strconv.FormatUint(uint64(attribute), 10)
	}
	return Style{
		sequence: []byte(escape + strings.Join(parts, ";") + suffix),
	}
}

// WriteStart writes the opening ANSI escape sequence to w. When colour is
// disabled this is a no-op.
//
// Zero allocations - writes pre-computed bytes directly.
//
// Takes w ([io.Writer]) which is the destination for the escape sequence.
func (s Style) WriteStart(w io.Writer) {
	if !Enabled() {
		return
	}
	_, _ = w.Write(s.sequence)
}

// WriteReset writes the reset escape sequence to w. When colour is disabled
// this is a no-op.
//
// Zero allocations - writes pre-computed bytes directly.
//
// Takes w ([io.Writer]) which is the destination for the reset sequence.
func (Style) WriteReset(w io.Writer) {
	if !Enabled() {
		return
	}
	_, _ = w.Write(resetSequence)
}

// Sprint wraps the string representation of args with colour codes and
// returns the result.
//
// Allocates one string for the builder output. When colour is disabled
// returns fmt.Sprint(args...) directly.
//
// Takes args (...any) which are the values to format.
//
// Returns string which is the coloured text.
func (s Style) Sprint(args ...any) string {
	content := fmt.Sprint(args...)
	if !Enabled() {
		return content
	}
	var builder strings.Builder
	builder.Grow(len(s.sequence) + len(content) + len(resetSequence))
	builder.Write(s.sequence)
	builder.WriteString(content)
	builder.Write(resetSequence)
	return builder.String()
}

// Sprintf wraps the formatted string with colour codes and returns the
// result.
//
// Allocates one string for the builder output. When colour is disabled
// returns fmt.Sprintf(format, args...) directly.
//
// Takes format (string) which is the format string.
// Takes args (...any) which are the format arguments.
//
// Returns string which is the coloured formatted text.
func (s Style) Sprintf(format string, args ...any) string {
	content := fmt.Sprintf(format, args...)
	if !Enabled() {
		return content
	}
	var builder strings.Builder
	builder.Grow(len(s.sequence) + len(content) + len(resetSequence))
	builder.Write(s.sequence)
	builder.WriteString(content)
	builder.Write(resetSequence)
	return builder.String()
}

// SprintFunc returns a function that colours its arguments using this style.
// The returned function allocates one string per call.
//
// Returns func(...any) string which applies this style to its arguments.
func (s Style) SprintFunc() func(args ...any) string {
	return s.Sprint
}
