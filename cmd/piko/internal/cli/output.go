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

package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

const (
	// outputDash is the dash placeholder used in formatted output.
	outputDash = "-"

	// outputIndent is the two-space indent used in formatted output.
	outputIndent = "  "

	// outputSecondsPerMinute is the number of seconds in a minute.
	outputSecondsPerMinute = 60
)

var (
	// ansiPattern matches ANSI escape sequences for stripping when calculating
	// visible string width.
	ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

	// statusColours maps lowercase status strings to ANSI colour codes.
	// Green (2) = success, Yellow (3) = in-progress, Red (1) = failure,
	// Grey (8) = unknown/default.
	statusColours = map[string]color.Color{
		"healthy":   lipgloss.Color("2"),
		"ok":        lipgloss.Color("2"),
		"complete":  lipgloss.Color("2"),
		"completed": lipgloss.Color("2"),
		"connected": lipgloss.Color("2"),
		"enabled":   lipgloss.Color("2"),
		"degraded":  lipgloss.Color("3"),
		"pending":   lipgloss.Color("3"),
		"active":    lipgloss.Color("3"),
		"running":   lipgloss.Color("3"),
		"disabled":  lipgloss.Color("3"),
		"stopped":   lipgloss.Color("3"),
		"unhealthy": lipgloss.Color("1"),
		"error":     lipgloss.Color("1"),
		"failed":    lipgloss.Color("1"),
	}
)

// Column defines a table column with optional wide-only visibility.
type Column struct {
	// Header is the column header text.
	Header string

	// WideOnly indicates the column is only shown with -o wide.
	WideOnly bool
}

// DetailSection represents a labelled group of key-value fields for describe
// output.
type DetailSection struct {
	// Title is the section header.
	Title string

	// Fields are the key-value pairs in this section.
	Fields []DetailField

	// SubSections are nested sections.
	SubSections []DetailSection
}

// DetailField represents a single key-value pair in describe output.
type DetailField struct {
	// Key is the field label.
	Key string

	// Value is the field value.
	Value string

	// IsStatus indicates the value should be colourised as a status.
	IsStatus bool
}

// Printer writes structured data in table, wide, or JSON format.
type Printer struct {
	// w is the destination writer.
	w io.Writer

	// format is the output format ("table", "wide", or "json").
	format string

	// noColour disables coloured output when true.
	noColour bool

	// noHeaders omits table headers from output when true.
	noHeaders bool

	// limit is the global item limit. Zero means handlers use their own default.
	limit int
}

// NewPrinter creates a new output printer.
//
// Takes w (io.Writer) which is the destination for output.
// Takes format (string) which selects the output mode ("table", "wide", or
// "json").
// Takes noColour (bool) which disables colour when true.
// Takes noHeaders (bool) which omits table headers when true.
//
// Returns *Printer which writes formatted output.
func NewPrinter(w io.Writer, format string, noColour, noHeaders bool) *Printer {
	return &Printer{
		w:         w,
		format:    format,
		noColour:  noColour,
		noHeaders: noHeaders,
	}
}

// PrintJSON writes data as indented JSON.
//
// Takes data (any) which is the value to serialise.
//
// Returns error when JSON marshalling fails.
func (p *Printer) PrintJSON(data any) error {
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintTable writes data as a formatted table with ANSI-aware column alignment.
//
// Takes headers ([]string) which contains the column headers.
// Takes rows ([][]string) which contains the row data.
func (p *Printer) PrintTable(headers []string, rows [][]string) {
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = visibleWidth(h)
	}
	for _, row := range rows {
		for i := range min(len(row), len(colWidths)) {
			if w := visibleWidth(row[i]); w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	p.writeRow(headers, colWidths)

	separator := make([]string, len(headers))
	for i := range headers {
		separator[i] = strings.Repeat("-", colWidths[i])
	}
	p.writeRow(separator, colWidths)

	for _, row := range rows {
		p.writeRow(row, colWidths)
	}
}

// IsJSON returns true if the output format is JSON.
//
// Returns bool which indicates whether JSON output is enabled.
func (p *Printer) IsJSON() bool {
	return p.format == "json"
}

// IsWide returns true if the output format is wide.
//
// Returns bool which indicates whether wide table output is enabled.
func (p *Printer) IsWide() bool {
	return p.format == "wide"
}

// SetLimit sets the global item limit on the printer.
//
// Takes n (int) which is the limit value from global flags. Zero means
// handlers should use their own default.
func (p *Printer) SetLimit(n int) {
	p.limit = n
}

// GetLimit returns the global limit if set, otherwise the handler's default.
//
// Takes handlerDefault (int) which is the default limit for the calling
// handler.
//
// Returns int which is the effective limit to use.
func (p *Printer) GetLimit(handlerDefault int) int {
	if p.limit > 0 {
		return p.limit
	}
	return handlerDefault
}

// PrintResource writes rows as a column-filtered table, respecting wide mode
// and no-headers settings.
//
// Takes columns ([]Column) which defines all available columns including
// wide-only columns.
// Takes rows ([][]string) which contains one string per column per row.
func (p *Printer) PrintResource(columns []Column, rows [][]string) {
	indices := visibleColumnIndices(columns, p.IsWide())
	headers := extractColumnHeaders(columns, indices)
	filtered := filterRowColumns(rows, indices)

	if p.noHeaders {
		colWidths := computeColumnWidths(headers, filtered)
		for _, row := range filtered {
			p.writeRow(row, colWidths)
		}
		return
	}

	p.PrintTable(headers, filtered)
}

// PrintDetail writes sections in kubectl-style key-value format to the
// printer's configured writer.
//
// Takes sections ([]DetailSection) which contains the labelled field groups.
func (p *Printer) PrintDetail(sections []DetailSection) {
	for i, section := range sections {
		if i > 0 {
			_, _ = fmt.Fprintln(p.w)
		}
		p.printSection(section, 0)
	}
}

// ColourisedStatus returns the status string with colour applied.
//
// Takes status (string) which is the status to colourise.
//
// Returns string which is the styled status text.
func (p *Printer) ColourisedStatus(status string) string {
	return p.statusStyle(status).Render(status)
}

// writeRow writes a single padded row to the printer's writer.
//
// Takes cells ([]string) which contains the cell values, possibly with ANSI
// codes.
// Takes colWidths ([]int) which contains the visible width for each column.
func (p *Printer) writeRow(cells []string, colWidths []int) {
	for i, cell := range cells {
		if i > 0 {
			_, _ = fmt.Fprint(p.w, "  ")
		}
		pad := colWidths[i] - visibleWidth(cell)
		_, _ = fmt.Fprint(p.w, cell)
		if i < len(cells)-1 && pad > 0 {
			_, _ = fmt.Fprint(p.w, strings.Repeat(" ", pad))
		}
	}
	_, _ = fmt.Fprintln(p.w)
}

// printSection recursively renders a detail section with indentation.
//
// Takes section (DetailSection) which is the section to render.
// Takes depth (int) which controls the indentation level.
func (p *Printer) printSection(section DetailSection, depth int) {
	indent := strings.Repeat("  ", depth)
	if section.Title != "" {
		_, _ = fmt.Fprintf(p.w, "%s%s:\n", indent, section.Title)
	}

	p.printFields(section.Fields, indent)

	for _, sub := range section.SubSections {
		p.printSection(sub, depth+1)
	}
}

// printFields renders aligned key-value pairs at the given indentation.
//
// Takes fields ([]DetailField) which contains the key-value pairs.
// Takes indent (string) which is the prefix whitespace.
func (p *Printer) printFields(fields []DetailField, indent string) {
	maxLen := maxFieldKeyLength(fields)
	for _, f := range fields {
		value := f.Value
		if f.IsStatus {
			value = p.ColourisedStatus(value)
		}
		_, _ = fmt.Fprintf(p.w, "%s  %-*s  %s\n", indent, maxLen+1, f.Key+":", value)
	}
}

// statusStyle returns a lipgloss style for a health or status string.
//
// Takes status (string) which is the health or status value to style.
//
// Returns lipgloss.Style which is colour-coded based on the status value.
func (p *Printer) statusStyle(status string) lipgloss.Style {
	if p.noColour {
		return lipgloss.NewStyle()
	}

	colour, ok := statusColours[strings.ToLower(status)]
	if !ok {
		colour = lipgloss.Color("8")
	}
	return lipgloss.NewStyle().Foreground(colour)
}

// visibleWidth returns the visible character count of s after stripping ANSI
// escape sequences.
//
// Takes s (string) which is the string to measure.
//
// Returns int which is the number of visible characters.
func visibleWidth(s string) int {
	return utf8.RuneCountInString(ansiPattern.ReplaceAllString(s, ""))
}

// maxFieldKeyLength returns the length of the longest key in a field slice.
//
// Takes fields ([]DetailField) which contains the fields to measure.
//
// Returns int which is the maximum key length.
func maxFieldKeyLength(fields []DetailField) int {
	maxLen := 0
	for _, f := range fields {
		if len(f.Key) > maxLen {
			maxLen = len(f.Key)
		}
	}
	return maxLen
}

// visibleColumnIndices returns the indices of columns to display based on
// whether wide mode is active.
//
// Takes columns ([]Column) which defines all columns.
// Takes isWide (bool) which indicates whether wide-only columns are included.
//
// Returns []int which contains the indices of visible columns.
func visibleColumnIndices(columns []Column, isWide bool) []int {
	indices := make([]int, 0, len(columns))
	for i, col := range columns {
		if !col.WideOnly || isWide {
			indices = append(indices, i)
		}
	}
	return indices
}

// extractColumnHeaders returns header strings for the given column indices.
//
// Takes columns ([]Column) which defines all columns.
// Takes indices ([]int) which specifies which columns to include.
//
// Returns []string which contains the visible column headers.
func extractColumnHeaders(columns []Column, indices []int) []string {
	headers := make([]string, len(indices))
	for i, index := range indices {
		headers[i] = columns[index].Header
	}
	return headers
}

// filterRowColumns returns rows with only the visible column values.
//
// Takes rows ([][]string) which contains the full row data.
// Takes indices ([]int) which specifies which columns to include.
//
// Returns [][]string which contains the filtered rows.
func filterRowColumns(rows [][]string, indices []int) [][]string {
	filtered := make([][]string, len(rows))
	for i, row := range rows {
		newRow := make([]string, len(indices))
		for j, index := range indices {
			if index < len(row) {
				newRow[j] = row[index]
			}
		}
		filtered[i] = newRow
	}
	return filtered
}

// computeColumnWidths returns the maximum visible width for each column across
// headers and rows.
//
// Takes headers ([]string) which contains the column headers.
// Takes rows ([][]string) which contains the row data.
//
// Returns []int which contains the width for each column.
func computeColumnWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = visibleWidth(h)
	}
	for _, row := range rows {
		for i := range min(len(row), len(widths)) {
			if w := visibleWidth(row[i]); w > widths[i] {
				widths[i] = w
			}
		}
	}
	return widths
}

// matchesFilter reports whether name matches the filter string. It performs
// case-insensitive exact match first, then prefix match.
//
// Takes name (string) which is the value to test.
// Takes filter (string) which is the filter to match against. Empty matches
// everything.
//
// Returns bool which indicates whether name matches.
func matchesFilter(name, filter string) bool {
	if filter == "" {
		return true
	}
	lower := strings.ToLower(name)
	filterLower := strings.ToLower(filter)
	return lower == filterLower || strings.HasPrefix(lower, filterLower)
}

// extractFilter returns the first non-flag positional argument as the name
// filter, or an empty string if none is present.
//
// Takes arguments ([]string) which contains the command arguments.
//
// Returns string which is the filter value.
func extractFilter(arguments []string) string {
	if len(arguments) == 0 {
		return ""
	}
	if strings.HasPrefix(arguments[0], "-") {
		return ""
	}
	return arguments[0]
}

// argsAfterFilter returns arguments with the filter argument removed if it was
// consumed.
//
// Takes arguments ([]string) which contains the original arguments.
// Takes filter (string) which is the filter value that was extracted.
//
// Returns []string which contains the remaining arguments.
func argsAfterFilter(arguments []string, filter string) []string {
	if filter == "" || len(arguments) == 0 {
		return arguments
	}
	return arguments[1:]
}

// parseInterspersed parses flags from arguments where flags and positional
// arguments may be mixed together.
//
// Go's flag.FlagSet.Parse stops at the first non-flag argument, so a command
// like "Liveness --help" would never see --help. This function separates flags
// from positional arguments, parses the flags, and returns the
// positional arguments.
//
// Takes fs (*flag.FlagSet) which has flags already defined on it.
// Takes arguments ([]string) which contains interspersed flags and
// positional arguments.
//
// Returns []string which contains the positional (non-flag) arguments.
// Returns error when flag parsing fails (including flag.ErrHelp for --help).
func parseInterspersed(fs *flag.FlagSet, arguments []string) ([]string, error) {
	var flagArgs, positional []string

	for i := 0; i < len(arguments); i++ {
		argument := arguments[i] //nolint:gosec // loop bounded
		if !strings.HasPrefix(argument, "-") {
			positional = append(positional, argument)
			continue
		}

		flagArgs = append(flagArgs, argument)

		name := strings.TrimLeft(argument, "-") //nolint:revive // flag prefix dash
		if _, _, ok := strings.Cut(name, "="); ok {
			continue
		}

		f := fs.Lookup(name)
		if f != nil && !isBoolFlag(f) && i+1 < len(arguments) {
			i++
			flagArgs = append(flagArgs, arguments[i])
		}
	}

	err := fs.Parse(flagArgs)
	return positional, err
}

// isBoolFlag reports whether the given flag is a boolean flag.
//
// Takes f (*flag.Flag) which is the flag to check.
//
// Returns bool which is true if the flag implements IsBoolFlag and returns
// true.
func isBoolFlag(f *flag.Flag) bool {
	bf, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && bf.IsBoolFlag()
}

// grpcError converts a gRPC error into a user-friendly message.
//
// When the error has status code Unimplemented, it returns a message stating
// the service is not available. Otherwise it wraps the error with the provided
// context.
//
// Takes context (string) which describes what was being fetched.
// Takes err (error) which is the gRPC error to convert.
//
// Returns error with a friendly message for Unimplemented, or the original
// error wrapped with context.
func grpcError(context string, err error) error {
	if s, ok := grpcstatus.FromError(err); ok && s.Code() == codes.Unimplemented {
		return fmt.Errorf("%s: service not available on this server", context)
	}
	return fmt.Errorf("%s: %w", context, err)
}

// validateOutputFormat checks that format is one of the allowed values. It
// returns an error listing the supported formats if the format is invalid.
//
// Takes format (string) which is the output format to validate.
// Takes command (string) which is the command name for error messages.
// Takes allowed ([]string) which lists the valid format values.
//
// Returns error when format is not in the allowed list.
func validateOutputFormat(format, command string, allowed []string) error {
	if slices.Contains(allowed, format) {
		return nil
	}
	return fmt.Errorf("unsupported output format %q for %s command (supported: %s)",
		format, command, strings.Join(allowed, ", "))
}

// formatTimestamp converts a Unix timestamp in seconds to a human-readable
// string.
//
// Takes ts (int64) which is the Unix timestamp in seconds.
//
// Returns string which is the formatted date and time, or "-" if ts is zero.
func formatTimestamp(ts int64) string {
	if ts == 0 {
		return outputDash
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

// formatDuration formats a millisecond duration for display.
//
// Takes ms (int64) which is the duration in milliseconds.
//
// Returns string which is the human-readable duration in ms, seconds, minutes,
// or hours depending on the magnitude.
func formatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	totalSeconds := int64(d.Seconds())
	if d < time.Hour {
		m := totalSeconds / outputSecondsPerMinute
		s := totalSeconds % outputSecondsPerMinute
		return fmt.Sprintf("%dm%ds", m, s)
	}
	totalMinutes := totalSeconds / outputSecondsPerMinute
	h := totalMinutes / outputSecondsPerMinute
	m := totalMinutes % outputSecondsPerMinute
	return fmt.Sprintf("%dh%dm", h, m)
}

// formatBytes formats a byte count for display.
//
// Takes n (uint64) which is the number of bytes to format.
//
// Returns string which is the human-readable size with appropriate unit
// (B, KiB, MiB, or GiB).
func formatBytes(n uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)

	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GiB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MiB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KiB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// formatNanos formats nanoseconds as a human-readable duration.
//
// Takes nanoseconds (int64) which is the duration in nanoseconds to format.
//
// Returns string which is the formatted duration with appropriate units.
func formatNanos(nanoseconds int64) string {
	d := time.Duration(nanoseconds) * time.Nanosecond
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", nanoseconds)
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(nanoseconds)/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(nanoseconds)/1e6)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// printHealthTree recursively prints a health status tree with indentation.
// This is used by the watch command for streaming output.
//
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which provides colourised status formatting.
// Takes status (*pb.HealthStatus) which is the health status node to print.
// Takes depth (int) which controls the indentation level.
func printHealthTree(w io.Writer, p *Printer, status *pb.HealthStatus, depth int) {
	indent := strings.Repeat(outputIndent, depth)
	state := p.ColourisedStatus(status.GetState())

	line := fmt.Sprintf("%s%s: %s", indent, status.GetName(), state)
	if status.GetMessage() != "" {
		line += fmt.Sprintf(" (%s)", status.GetMessage())
	}
	if status.GetDuration() != "" {
		line += fmt.Sprintf(" [%s]", status.GetDuration())
	}
	_, _ = fmt.Fprintln(w, line)

	for _, dependency := range status.GetDependencies() {
		printHealthTree(w, p, dependency, depth+1)
	}
}
