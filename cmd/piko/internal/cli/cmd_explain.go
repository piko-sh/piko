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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"piko.sh/piko/internal/colour"
	"piko.sh/piko/internal/diagnostic_docs"
)

const (
	// explainWrapWidth is the column explanations are wrapped at.
	explainWrapWidth = 88

	// explainIndent indents the body of each section.
	explainIndent = "  "

	// seagullDiagnosticsURL is where the SQL generator's Q-codes are explained.
	seagullDiagnosticsURL = "https://seagull.sh/diagnostics/"

	// diagnosticsIndexURL lists every Piko diagnostic code.
	diagnosticsIndexURL = "https://piko.sh/diagnostics/"
)

var (
	// sqlCodePattern matches the codes reported by the SQL generator.
	sqlCodePattern = regexp.MustCompile(`^[Qq][0-9]{3}$`)

	// noColourVariables turn colour off when any of them is set.
	noColourVariables = []string{"NO_COLOUR", "NO_COLOR"}

	// forceColourVariables turn colour on when any of them is set.
	forceColourVariables = []string{"FORCE_COLOUR", "FORCE_COLOR"}
)

// RunExplain runs the explain subcommand, writing to os.Stdout and os.Stderr. Examples
// are coloured only when os.Stdout is a terminal, so piped output stays plain.
//
// Takes arguments ([]string) which contains the command-line arguments to parse.
//
// Returns int which is the exit code, zero for success or non-zero on error.
func RunExplain(arguments []string) int {
	colour.SetEnabled(explainColour(os.Stdout))
	return RunExplainWithIO(arguments, os.Stdout, os.Stderr)
}

// explainColour reports whether output written to file should be coloured. NO_COLOUR and
// NO_COLOR turn colour off, FORCE_COLOUR and FORCE_COLOR turn it on, and otherwise it is
// on only for a terminal whose TERM is not "dumb".
//
// Takes file (*os.File) which is where the output goes.
//
// Returns bool which is true when the output should be coloured.
func explainColour(file *os.File) bool {
	if anyVariableSet(noColourVariables) {
		return false
	}
	if anyVariableSet(forceColourVariables) {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// anyVariableSet reports whether any of the named environment variables is set, to any
// value.
//
// Takes names ([]string) which are the variables to check.
//
// Returns bool which is true when at least one is set.
func anyVariableSet(names []string) bool {
	for _, name := range names {
		if _, set := os.LookupEnv(name); set {
			return true
		}
	}
	return false
}

// RunExplainWithIO runs the explain subcommand with explicit output writers.
//
// With codes, it explains each; without, it lists every code. --json prints the same data
// as JSON.
//
// Takes arguments ([]string) which contains the command-line arguments to parse.
// Takes stdout (io.Writer) which receives the explanations.
// Takes stderr (io.Writer) which receives usage and error messages.
//
// Returns int which is 0 on success, 1 for an unknown code, or 2 for a usage error.
func RunExplainWithIO(arguments []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("explain", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	asJSON := flagSet.Bool("json", false, "Print the explanation, or the list of codes, as JSON")
	flagSet.Usage = func() { explainUsage(stderr) }
	if err := flagSet.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if flagSet.NArg() == 0 {
		if *asJSON {
			return writeCatalogueJSON(stdout, stderr)
		}
		writeCodeList(stdout)
		return 0
	}

	exitCode := 0
	for index, id := range flagSet.Args() {
		if index > 0 && !*asJSON {
			_, _ = fmt.Fprintln(stdout)
		}
		if code := explainOne(id, *asJSON, stdout, stderr); code != 0 {
			exitCode = code
		}
	}
	return exitCode
}

// explainOne explains a single code, a SQL code, or reports an unknown one.
//
// Takes id (string) which is the code or name to explain.
// Takes asJSON (bool) which selects JSON output.
// Takes stdout (io.Writer) which receives the explanation.
// Takes stderr (io.Writer) which receives the error for an unknown code.
//
// Returns int which is 0 when the code was explained and 1 when it is unknown.
func explainOne(id string, asJSON bool, stdout, stderr io.Writer) int {
	entry, found := diagnostic_docs.Lookup(id)
	if found {
		if asJSON {
			return writeEntryJSON(entry, stdout, stderr)
		}
		writeExplanation(entry, stdout)
		return 0
	}

	trimmed := strings.ToUpper(strings.TrimSpace(id))
	if sqlCodePattern.MatchString(trimmed) {
		_, _ = fmt.Fprintf(stdout,
			"%s is a SQL diagnostic, reported by Piko's SQL generator, Seagull.\n"+
				"It is explained at %s%s\n"+
				"Piko projects write the directives it mentions with the piko. prefix rather than sgl.\n",
			trimmed, seagullDiagnosticsURL, trimmed)
		return 0
	}

	_, _ = fmt.Fprintf(stderr, "piko explain: %q is not a Piko diagnostic code\n", id)
	if suggestion := diagnostic_docs.Suggest(id); suggestion != "" {
		_, _ = fmt.Fprintf(stderr, "Did you mean %s?\n", suggestion)
	}
	_, _ = fmt.Fprintln(stderr, "Run 'piko explain' to list every code.")
	return 1
}

// writeExplanation prints one entry's explanation.
//
// Takes entry (diagnostic_docs.Entry) which is the code to explain.
// Takes w (io.Writer) which receives the output.
func writeExplanation(entry diagnostic_docs.Entry, w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s  %s  (%s)\n\n", entry.Code, entry.Title, strings.ToLower(entry.SeverityLabel()))
	if entry.Unused {
		_, _ = fmt.Fprintln(w, "Not currently reported. The number stays assigned so it is never reused.")
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, wrapText(entry.Summary, "", explainWrapWidth))
	writeSection(w, "When it is reported", entry.Detail)
	writeSection(w, "How to fix it", entry.Fix)
	if example := entry.RenderExample(); example != "" {
		_, _ = fmt.Fprintf(w, "\nExample\n%s\n", example)
	}
	_, _ = fmt.Fprintf(w, "\nMore: %s\n", entry.URL())
}

// writeSection prints a titled, wrapped, indented paragraph.
//
// Takes w (io.Writer) which receives the output.
// Takes title (string) which heads the section.
// Takes body (string) which is the paragraph.
func writeSection(w io.Writer, title, body string) {
	_, _ = fmt.Fprintf(w, "\n%s\n%s\n", title, wrapText(body, explainIndent, explainWrapWidth))
}

// writeCodeList prints every code, grouped by family.
//
// Takes w (io.Writer) which receives the output.
func writeCodeList(w io.Writer) {
	for index, group := range diagnostic_docs.ByFamily() {
		if index > 0 {
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, group.Family)
		for entryIndex := range group.Entries {
			entry := &group.Entries[entryIndex]
			title := entry.Title
			if entry.Unused {
				title += " (not currently reported)"
			}
			_, _ = fmt.Fprintf(w, "  %s  %-24s %s\n", entry.Code, strings.ToLower(entry.SeverityLabel()), title)
		}
	}
	_, _ = fmt.Fprintf(w, "\nRun 'piko explain <code>' for details, or see %s\n", diagnosticsIndexURL)
}

// writeEntryJSON prints one entry as JSON.
//
// Takes entry (diagnostic_docs.Entry) which is the code to print.
// Takes stdout (io.Writer) which receives the JSON.
// Takes stderr (io.Writer) which receives an encoding error.
//
// Returns int which is 0 on success and 1 when encoding fails.
func writeEntryJSON(entry diagnostic_docs.Entry, stdout, stderr io.Writer) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entry); err != nil {
		_, _ = fmt.Fprintf(stderr, "piko explain: %v\n", err)
		return 1
	}
	return 0
}

// writeCatalogueJSON prints every code as JSON, in the shape of
// docs/reference/diagnostics.json.
//
// Takes stdout (io.Writer) which receives the JSON.
// Takes stderr (io.Writer) which receives an encoding error.
//
// Returns int which is 0 on success and 1 when encoding fails.
func writeCatalogueJSON(stdout, stderr io.Writer) int {
	encoded, err := diagnostic_docs.RenderJSON()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "piko explain: %v\n", err)
		return 1
	}
	_, _ = stdout.Write(encoded)
	return 0
}

// wrapText wraps text at width, prefixing each line with indent.
//
// Takes text (string) which is the paragraph to wrap.
// Takes indent (string) which prefixes every line.
// Takes width (int) which is the maximum line length, including the indent.
//
// Returns string which is the wrapped paragraph.
func wrapText(text, indent string, width int) string {
	var lines []string
	line := indent
	for word := range strings.FieldsSeq(text) {
		if line != indent && len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = indent
		}
		if line != indent {
			line += " "
		}
		line += word
	}
	return strings.Join(append(lines, line), "\n")
}

// explainUsage prints the explain subcommand's usage.
//
// Takes w (io.Writer) which receives the usage text.
func explainUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage: piko explain [--json] [code...]

Explain Piko's diagnostic codes, such as the T141 in "error [T141]".

With no code, lists every code. A code can be given in any case (t141), or by
its name (ComparisonError). Q-codes come from the SQL generator and are
explained at https://seagull.sh/diagnostics/.

Flags:
  --json   Print the explanation, or the list of codes, as JSON

Examples:
  piko explain T141
  piko explain
  piko explain --json R003
`)
}
