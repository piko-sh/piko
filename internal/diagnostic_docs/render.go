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

package diagnostic_docs

import (
	"encoding/json"
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/colour"
)

const (
	// MarkdownFile is the generated reference page.
	MarkdownFile = "diagnostics.md"

	// JSONFile is the generated machine-readable catalogue, read by piko.sh.
	JSONFile = "diagnostics.json"

	// jsonVersion is the version of the JSON file's shape.
	jsonVersion = 1

	// introductionExampleCode is the code whose example opens the reference page.
	introductionExampleCode = "T141"

	// introductionExamplePlaceholder marks where the opening example goes.
	introductionExamplePlaceholder = "{{example}}"

	// markdownIntroduction opens the reference page.
	markdownIntroduction = `---
title: Diagnostics
description: Every diagnostic code Piko reports, what it means, and how to fix it.
nav:
  sidebar:
    section: "reference"
    subsection: "tooling"
    order: 20
---

<!-- Generated from internal/diagnostic_docs. Edit the catalogue there, then run: make generate-diagnostics-docs -->

# Diagnostics

When Piko builds a project or renders a page, it reports problems as diagnostics. Each one
has a severity and a stable code. Codes starting with T0 come from parsing template
expressions, T1 from analysing templates, and R0 from rendering. A code keeps its meaning
and its number is never reused, so editors, scripts and CI can rely on it.

Build output follows each coded diagnostic with a link to its explanation:

` + "```text" + `
{{example}}
` + "```" + `

Run ` + "`piko explain T141`" + ` to read an explanation in the terminal, or
` + "`piko explain`" + ` to list every code; see the [CLI reference](cli.md). Editors
that use the Piko language server link each code to its explanation; see
[LSP setup](../how-to/lsp-setup.md).

Severities:

- **Error** stops the build, or marks a render that failed.
- **Warning** reports a likely mistake. The build continues.
- **Info** reports something worth knowing. The build continues.

Some codes are reported with more than one severity, depending on the case; each code
below says when.

SQL diagnostics, whose codes start with Q, come from the SQL generator. They are listed in
the [querier reference](querier.md#diagnostics).
`
)

// familyDocument is one family in the JSON file.
type familyDocument struct {
	// Name is the family's display name.
	Name Family `json:"name"`

	// Slug is the family's anchor.
	Slug string `json:"slug"`

	// Prefix is the code prefix the family's codes share, such as "T1".
	Prefix string `json:"prefix"`

	// Introduction says when the family's codes are reported.
	Introduction string `json:"introduction"`
}

// jsonDocument is the shape of the JSON file.
type jsonDocument struct {
	// Families lists the families in display order.
	Families []familyDocument `json:"families"`

	// Codes lists every entry in code order.
	Codes []Entry `json:"codes"`

	// Version is the shape's version.
	Version int `json:"version"`
}

var (
	// severityLabels are the severities as written in the reference page.
	severityLabels = map[Severity]string{
		SeverityError:   "Error",
		SeverityWarning: "Warning",
		SeverityInfo:    "Info",
	}

	// astSeverities maps catalogue severities to the severities build output prints.
	astSeverities = map[Severity]ast_domain.Severity{
		SeverityError:   ast_domain.Error,
		SeverityWarning: ast_domain.Warning,
		SeverityInfo:    ast_domain.Info,
	}

	// familyPrefixes are the code prefixes of each family.
	familyPrefixes = map[Family]string{
		FamilyTemplateExpressions: "T0",
		FamilyTemplateAnalysis:    "T1",
		FamilyRendering:           "R0",
	}

	// familyIntroductions introduce each family's section of the reference page.
	familyIntroductions = map[Family]string{
		FamilyTemplateExpressions: "Reported while a template's expressions and directives are parsed, before any types are checked.",
		FamilyTemplateAnalysis:    "Reported while templates are type-checked against their Go code, and while components, props, assets and styles are linked.",
		FamilyRendering: "Reported by generated code while a page renders. They appear in the server log rather than in build output, and the page " +
			"still renders: a nil read or an out-of-range index gives a zero value instead of a panic.",
	}
)

// RenderExample returns the entry's example as build output prints it, or an empty string
// when the entry has none. Colour follows the colour package's setting.
//
// Returns string which is the formatted example.
func (e *Entry) RenderExample() string {
	example := e.Example
	if example == nil {
		return ""
	}
	runes := []rune(example.Source)
	start := min(max(example.Column-1, 0), len(runes))
	end := min(start+example.Width, len(runes))
	diagnostic := ast_domain.NewDiagnosticWithCode(
		astSeverities[e.Severities[0]],
		example.Message,
		string(runes[start:end]),
		e.Code,
		ast_domain.Location{Line: example.Line, Column: example.Column, Offset: 0},
		example.File,
	)
	source := strings.Repeat("\n", max(example.Line-1, 0)) + example.Source
	formatted := ast_domain.FormatDiagnostics(example.File, source, []*ast_domain.Diagnostic{diagnostic})
	if _, body, found := strings.Cut(formatted, "\n\n"); found {
		formatted = body
	}
	return strings.TrimRight(formatted, "\n")
}

// SeverityLabel returns the entry's severities as words, such as "Error or warning".
//
// Returns string which is the label.
func (e *Entry) SeverityLabel() string {
	words := make([]string, 0, len(e.Severities))
	for index, severity := range e.Severities {
		word := severityLabels[severity]
		if index > 0 {
			word = strings.ToLower(word)
		}
		words = append(words, word)
	}
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	default:
		return strings.Join(words[:len(words)-1], ", ") + " or " + words[len(words)-1]
	}
}

// RenderMarkdown returns the reference page, docs/reference/diagnostics.md.
//
// The examples are rendered without colour, whatever the colour package's setting; the
// setting is restored before returning.
//
// Returns []byte which is the page's content.
func RenderMarkdown() []byte {
	previous := colour.Enabled()
	colour.SetEnabled(false)
	defer colour.SetEnabled(previous)

	introductionExample, _ := Lookup(introductionExampleCode)

	var page strings.Builder
	page.WriteString(strings.Replace(markdownIntroduction, introductionExamplePlaceholder, markdownExample(&introductionExample), 1))
	page.WriteString("\n## All codes {#all-codes}\n")
	groups := ByFamily()
	for index := range groups {
		writeOverview(&page, &groups[index])
	}
	for index := range groups {
		writeFamily(&page, &groups[index])
	}
	return []byte(page.String())
}

// RenderJSON returns the machine-readable catalogue, docs/reference/diagnostics.json.
//
// Returns []byte which is the file's content.
// Returns error when the catalogue cannot be encoded.
func RenderJSON() ([]byte, error) {
	document := jsonDocument{Version: jsonVersion, Codes: All(), Families: make([]familyDocument, 0, len(Families))}
	for _, family := range Families {
		document.Families = append(document.Families, familyDocument{
			Name:         family,
			Slug:         FamilySlug(family),
			Prefix:       familyPrefixes[family],
			Introduction: familyIntroductions[family],
		})
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding diagnostics catalogue: %w", err)
	}
	return append(encoded, '\n'), nil
}

// writeOverview writes a family's table of codes, each linking to its section.
//
// Takes page (*strings.Builder) which receives the Markdown.
// Takes group (*Group) which is the family to write.
func writeOverview(page *strings.Builder, group *Group) {
	_, _ = fmt.Fprintf(page, "\n### %s {#%s-codes}\n\n", group.Family, group.Slug)
	page.WriteString("| Code | Severity | Meaning |\n|---|---|---|\n")
	for index := range group.Entries {
		entry := &group.Entries[index]
		_, _ = fmt.Fprintf(page, "| [%s](#%s) | %s | %s |\n", entry.Code, strings.ToLower(entry.Code), entry.SeverityLabel(), entry.Title)
	}
}

// writeFamily writes a family's section, with a subsection per code.
//
// Takes page (*strings.Builder) which receives the Markdown.
// Takes group (*Group) which is the family to write.
func writeFamily(page *strings.Builder, group *Group) {
	_, _ = fmt.Fprintf(page, "\n## %s {#%s}\n\n%s\n", group.Family, group.Slug, familyIntroductions[group.Family])
	for index := range group.Entries {
		entry := &group.Entries[index]
		_, _ = fmt.Fprintf(page, "\n### %s %s {#%s}\n\n", entry.Code, entry.Title, strings.ToLower(entry.Code))
		if entry.Unused {
			page.WriteString("Not currently reported. The number stays assigned so it is never reused.\n\n")
		}
		_, _ = fmt.Fprintf(page, "**%s.** %s\n\n", entry.SeverityLabel(), entry.Summary)
		_, _ = fmt.Fprintf(page, "**When it is reported.** %s\n\n", entry.Detail)
		_, _ = fmt.Fprintf(page, "**How to fix it.** %s\n", entry.Fix)
		if example := markdownExample(entry); example != "" {
			_, _ = fmt.Fprintf(page, "\n```text\n%s\n```\n", example)
		}
	}
}

// markdownExample returns the entry's example for the reference page, with trailing
// spaces removed so the generated file survives editors that strip them.
//
// Takes entry (*Entry) which is the entry whose example to render.
//
// Returns string which is the example, or an empty string when there is none.
func markdownExample(entry *Entry) string {
	lines := strings.Split(entry.RenderExample(), "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
