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

package driver_handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"piko.sh/piko/internal/colour"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// newline is the line break character used when writing log output.
	newline = "\n"
)

var (
	builderPool = sync.Pool{
		New: func() any {
			return &strings.Builder{}
		},
	}

	fieldOrder = []string{
		logger_domain.KeyTime, logger_domain.KeyLevel, logger_domain.KeyPID, logger_domain.KeyHost,
		logger_domain.KeyContext, logger_domain.KeyMethod, logger_domain.KeyMessage,
	}

	fieldPaddings = map[string]int{
		logger_domain.KeyLevel:     6,
		logger_domain.KeyContext:   28,
		logger_domain.KeyReference: 25,
		logger_domain.KeyMethod:    39,
	}

	fieldColours = map[string]colour.Style{
		logger_domain.KeyTime:      colour.New(colour.FgBlue),
		logger_domain.KeyMessage:   colour.New(colour.FgHiWhite),
		logger_domain.KeySource:    colour.New(colour.FgWhite),
		logger_domain.KeyPID:       colour.New(colour.FgWhite),
		logger_domain.KeyHost:      colour.New(colour.FgHiBlue),
		logger_domain.KeyContext:   colour.New(colour.FgMagenta),
		logger_domain.KeyReference: colour.New(colour.FgHiGreen),
		logger_domain.KeyMethod:    colour.New(colour.FgHiMagenta),
		logger_domain.KeyTaskQueue: colour.New(colour.FgMagenta),
		logger_domain.KeyIPAddress: colour.New(colour.FgHiMagenta),
		logger_domain.KeyAccountID: colour.New(colour.FgHiBlue),
		logger_domain.KeyAttempt:   colour.New(colour.FgRed),
		logger_domain.KeyError:     colour.New(colour.FgRed),
	}

	levelColours = map[slog.Level]colour.Style{
		logger_domain.LevelTrace:    colour.New(colour.FgHiBlack),
		logger_domain.LevelInternal: colour.New(colour.FgHiBlack, colour.Italic),
		slog.LevelDebug:             colour.New(colour.FgGreen),
		slog.LevelInfo:              colour.New(colour.FgYellow),
		logger_domain.LevelNotice:   colour.New(colour.FgHiYellow),
		slog.LevelWarn:              colour.New(colour.FgHiYellow, colour.Bold),
		slog.LevelError:             colour.New(colour.FgRed),
	}

	sourceColour = colour.New(colour.FgHiBlack)

	keyColour = colour.New(colour.FgCyan)

	whiteColour = colour.New(colour.FgWhite)
)

// Options configures the behaviour of the prettyHandler.
type Options struct {
	// Level sets the minimum log level to output. Defaults to Info if nil.
	Level slog.Leveler

	// AddSource enables logging of the source file and line number.
	AddSource bool

	// NoColour disables coloured output.
	NoColour bool
}

// prettyHandler is a slog.Handler that formats log records for easy reading
// in the console. It uses colours, padding, and ordered fields to make logs
// clear and simple to scan.
type prettyHandler struct {
	// w is the destination writer for formatted log output.
	w io.Writer

	// opts holds the handler settings.
	opts Options

	// pid is the process identifier of the current process.
	pid string

	// hostname is the machine's host name, included in each log entry.
	hostname string

	// attrs holds attributes added via WithAttrs for inclusion in log output.
	attrs []slog.Attr

	// groups holds the nested group names for log attributes.
	groups []string
}

// Enabled reports whether the handler handles the given log level.
// This implements the slog.Handler interface.
//
// Takes level (slog.Level) which is the log level to check.
//
// Returns bool which is true if the level meets or exceeds the handler's
// minimum level.
func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// WithAttrs returns a new prettyHandler with additional attributes.
// This implements the slog.Handler interface.
//
// Takes attrs ([]slog.Attr) which specifies the attributes to add.
//
// Returns slog.Handler which is a new handler with the combined attributes.
func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	newHandler := *h
	newHandler.attrs = make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs)+4)
	copy(newHandler.attrs, h.attrs)
	newHandler.attrs = append(newHandler.attrs, attrs...)
	return &newHandler
}

// WithGroup returns a new prettyHandler with an additional group name.
// This implements the slog.Handler interface.
//
// Takes name (string) which specifies the group name to add.
//
// Returns slog.Handler which is the new handler with the group appended.
func (h *prettyHandler) WithGroup(name string) slog.Handler {
	newHandler := *h
	newHandler.groups = make([]string, len(h.groups), len(h.groups)+1)
	copy(newHandler.groups, h.groups)
	newHandler.groups = append(newHandler.groups, name)
	return &newHandler
}

// Handle formats and outputs a log record to the configured writer.
// This implements the slog.Handler interface.
//
// Takes r (slog.Record) which contains the log entry to format.
//
// Returns error when writing to the underlying writer fails.
//
//nolint:gocritic // slog.Handler requires value receiver
func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := h.gatherAllAttrs(&r)

	builder, ok := builderPool.Get().(*strings.Builder)
	if !ok {
		builder = &strings.Builder{}
	}
	defer builderPool.Put(builder)
	builder.Reset()

	h.writeMainFields(builder, &r, attrs)
	h.writeExtraAttrs(builder, attrs)
	h.writeSource(builder, &r)

	builder.WriteString(newline)

	_, err := h.w.Write([]byte(builder.String()))
	if err != nil {
		return fmt.Errorf("writing formatted log record: %w", err)
	}
	return nil
}

// gatherAllAttrs collects all attributes from the handler and record into a
// single map.
//
// Takes r (*slog.Record) which provides the log record attributes to collect.
//
// Returns map[string]slog.Value which contains all attributes keyed by name.
func (h *prettyHandler) gatherAllAttrs(r *slog.Record) map[string]slog.Value {
	attrs := make(map[string]slog.Value, r.NumAttrs()+len(h.attrs))

	h.collectAttrs(attrs, h.attrs, h.groups)

	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value
		return true
	})

	return attrs
}

// writeMainFields writes the main fields (time, level, message, and others)
// to the builder in the set order, with colour and separators.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes r (*slog.Record) which provides the log record data.
// Takes attrs (map[string]slog.Value) which contains extra attributes.
func (h *prettyHandler) writeMainFields(builder *strings.Builder, r *slog.Record, attrs map[string]slog.Value) {
	for i, key := range fieldOrder {
		if i > 0 {
			builder.WriteString(" | ")
		}

		value, found := h.getFieldValue(key, r, attrs)
		c := h.getFieldColour(key, r.Level)
		finalValue := h.formatFieldValue(value, found, key)

		c.WriteStart(builder)
		builder.WriteString(finalValue)
		c.WriteReset(builder)
	}
}

// getFieldValue gets the value for a given field key.
//
// Takes key (string) which specifies the field to get.
// Takes r (*slog.Record) which provides the log record to read values from.
// Takes attrs (map[string]slog.Value) which holds extra attributes.
//
// Returns string which contains the field value.
// Returns bool which shows whether the key was found.
func (h *prettyHandler) getFieldValue(key string, r *slog.Record, attrs map[string]slog.Value) (string, bool) {
	switch key {
	case logger_domain.KeyTime:
		return r.Time.Format("15:04:05"), true
	case logger_domain.KeyLevel:
		return logger_domain.LevelName(r.Level), true
	case logger_domain.KeyMessage:
		return r.Message, true
	case logger_domain.KeyPID:
		return h.pid, true
	case logger_domain.KeyHost:
		return h.hostname, true
	default:
		if attributeValue, ok := attrs[key]; ok {
			delete(attrs, key)
			return attributeValue.String(), true
		}
		return "", false
	}
}

// getFieldColour returns the colour to use for a given field.
//
// Takes key (string) which identifies the field to colour.
// Takes level (slog.Level) which sets the colour for level fields.
//
// Returns colour.Style which is the colour for the field.
func (*prettyHandler) getFieldColour(key string, level slog.Level) colour.Style {
	if key == logger_domain.KeyLevel {
		if levelColour, ok := levelColours[level]; ok {
			return levelColour
		}
		return whiteColour
	}

	if c, ok := fieldColours[key]; ok {
		return c
	}
	return whiteColour
}

// formatFieldValue applies padding to field values as needed.
//
// Takes value (string) which is the field value to format.
// Takes found (bool) which shows whether the field was present.
// Takes key (string) which is the field name for padding lookup.
//
// Returns string which is the formatted value with correct padding.
func (*prettyHandler) formatFieldValue(value string, found bool, key string) string {
	padding, hasPadding := fieldPaddings[key]

	if !found {
		return strPad("~", padding, " ", "RIGHT")
	}

	if hasPadding {
		return strPad(value, padding, " ", "RIGHT")
	}
	return value
}

// writeExtraAttrs writes any extra attributes not in the main field order.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes attrs (map[string]slog.Value) which contains the extra attributes to
// write.
func (h *prettyHandler) writeExtraAttrs(builder *strings.Builder, attrs map[string]slog.Value) {
	if len(attrs) == 0 {
		return
	}

	builder.WriteString(" |")
	extraKeys := h.getSortedKeys(attrs)

	for _, key := range extraKeys {
		if key == "stack_trace" {
			h.writeStackTrace(builder, attrs[key])
			continue
		}
		h.writeKeyValue(builder, key, attrs[key])
	}
}

// getSortedKeys returns sorted keys from the attributes map.
//
// Takes attrs (map[string]slog.Value) which contains the attributes to sort.
//
// Returns []string which contains the attribute keys in alphabetical order.
func (*prettyHandler) getSortedKeys(attrs map[string]slog.Value) []string {
	return slices.Sorted(maps.Keys(attrs))
}

// writeStackTrace formats and writes a stack trace to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes valueString (slog.Value) which holds the stack trace data to extract.
func (h *prettyHandler) writeStackTrace(builder *strings.Builder, valueString slog.Value) {
	frames := h.extractStackFrames(valueString)

	if len(frames) == 0 {
		return
	}

	builder.WriteString(newline)
	sourceColour.WriteStart(builder)
	builder.WriteString("  Stack trace:" + newline)
	for _, frame := range frames {
		if frame != "" {
			builder.WriteString(frame)
			builder.WriteString(newline)
		}
	}
	sourceColour.WriteReset(builder)
}

// extractStackFrames extracts stack trace frames from a value.
//
// Takes valueString (slog.Value) which holds the stack trace data. Supports
// StackTrace, []string, or string (split by newlines) formats.
//
// Returns []string which holds the extracted stack frames, or nil if the
// value type is not recognised.
func (*prettyHandler) extractStackFrames(valueString slog.Value) []string {
	anyVal := valueString.Any()

	switch v := anyVal.(type) {
	case logger_domain.StackTrace:
		return v.Frames()
	case []string:
		return v
	case string:
		return strings.Split(v, "\n")
	default:
		return nil
	}
}

// writeKeyValue writes a single key=value pair to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes key (string) which is the attribute name to write.
// Takes valueString (slog.Value) which is the attribute value to format.
func (*prettyHandler) writeKeyValue(builder *strings.Builder, key string, valueString slog.Value) {
	builder.WriteString(" ")
	keyColour.WriteStart(builder)
	builder.WriteString(key)
	keyColour.WriteReset(builder)
	builder.WriteString("=")
	builder.WriteString(valueString.String())
}

// writeSource adds the source file location to the output if AddSource is on.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes r (*slog.Record) which provides the program counter for source lookup.
func (h *prettyHandler) writeSource(builder *strings.Builder, r *slog.Record) {
	if !h.opts.AddSource || r.PC == 0 {
		return
	}

	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	if f.File == "" {
		return
	}

	src := h.formatSourceLocation(f)

	builder.WriteString(" | ")
	sourceColour.WriteStart(builder)
	builder.WriteString(src)
	sourceColour.WriteReset(builder)
}

// formatSourceLocation formats a source file location as "file:line".
//
// Takes f (runtime.Frame) which provides the source file and line number.
//
// Returns string which contains the formatted location as "basename:line".
func (*prettyHandler) formatSourceLocation(f runtime.Frame) string {
	srcBuilder, ok := builderPool.Get().(*strings.Builder)
	if !ok {
		srcBuilder = &strings.Builder{}
	}
	defer builderPool.Put(srcBuilder)

	srcBuilder.Reset()
	srcBuilder.WriteString(filepath.Base(f.File))
	srcBuilder.WriteString(":")
	srcBuilder.WriteString(strconv.Itoa(f.Line))
	return srcBuilder.String()
}

// collectAttrs gathers attributes into a flat map with dot-separated group keys.
//
// Takes dst (map[string]slog.Value) which receives the gathered attributes.
// Takes as ([]slog.Attr) which contains the attributes to process.
// Takes groups ([]string) which holds the current group path for nested keys.
func (h *prettyHandler) collectAttrs(dst map[string]slog.Value, as []slog.Attr, groups []string) {
	for _, a := range as {
		if a.Value.Kind() == slog.KindGroup {
			h.collectAttrs(dst, a.Value.Group(), append(groups, a.Key))
		} else {
			key := a.Key
			if len(groups) > 0 {
				key = strings.Join(append(groups, a.Key), ".")
			}
			dst[key] = a.Value
		}
	}
}

// NewPrettyHandler creates a new pretty handler with the given writer and
// options. If opts is nil, default options are used (Info level, colours
// enabled).
//
// Takes w (io.Writer) which receives the formatted log output.
// Takes opts (*Options) which sets handler behaviour such as log level and
// colour settings.
//
// Returns slog.Handler which is ready to use with slog.
func NewPrettyHandler(w io.Writer, opts *Options) slog.Handler {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Level == nil {
		levelVar := new(slog.LevelVar)
		levelVar.Set(slog.LevelInfo)
		opts.Level = levelVar
	}
	colour.SetEnabled(!opts.NoColour)

	pid := strconv.Itoa(os.Getpid())
	hostname, _ := os.Hostname()

	return &prettyHandler{
		w:        w,
		opts:     *opts,
		pid:      pid,
		hostname: hostname,
		attrs:    []slog.Attr{},
		groups:   []string{},
	}
}

// strPad pads a string to reach a given length using a pad string.
//
// Takes input (string) which is the string to pad.
// Takes padLength (int) which is the target length for the result.
// Takes padString (string) which is the string used for padding.
// Takes padType (string) which sets where to add padding: "LEFT", "RIGHT",
// or "BOTH".
//
// Returns string which is the padded result, or the original input if it
// already meets or exceeds the target length.
func strPad(input string, padLength int, padString string, padType string) string {
	var output string
	inputLength := len(input)
	if inputLength >= padLength {
		return input
	}

	repeat := math.Ceil(float64(padLength-inputLength) / float64(len(padString)))

	switch padType {
	case "RIGHT":
		output = input + strings.Repeat(padString, int(repeat))
		output = output[:padLength]
	case "LEFT":
		output = strings.Repeat(padString, int(repeat)) + input
		output = output[len(output)-padLength:]
	case "BOTH":
		length := float64(padLength-inputLength) / 2
		left := strings.Repeat(padString, int(math.Ceil(length/float64(len(padString)))))
		right := strings.Repeat(padString, int(math.Ceil(length/float64(len(padString)))))
		output = left[:int(math.Floor(length))] + input + right[:int(math.Ceil(length))]
	}

	return output
}
