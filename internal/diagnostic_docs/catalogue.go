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
	"regexp"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Severity is how seriously a diagnostic code is reported.
type Severity string

// Family groups the codes reported by one stage of the build or of rendering.
type Family string

// Example holds a diagnostic's source line, highlight position and width, and message.
type Example struct {
	// File is the path the diagnostic points at.
	File string `json:"file"`

	// Source is the text of the line the diagnostic points at.
	Source string `json:"source"`

	// Message is the diagnostic's message.
	Message string `json:"message"`

	// Line is the 1-based line number.
	Line int `json:"line"`

	// Column is the 1-based column where the highlight starts.
	Column int `json:"column"`

	// Width is the number of characters highlighted.
	Width int `json:"width"`
}

// Entry explains one diagnostic code.
type Entry struct {
	// Example shows the code as Piko reports it, or is nil.
	Example *Example `json:"example,omitempty"`

	// Code is the stable code, such as "T141".
	Code string `json:"code"`

	// Name is the code's constant name without its "Code" prefix, such as "ComparisonError".
	Name string `json:"name"`

	// Family is the stage that reports the code.
	Family Family `json:"family"`

	// Title is a short heading.
	Title string `json:"title"`

	// Summary is one sentence saying what the code means.
	Summary string `json:"summary"`

	// Detail says when the code is reported and why.
	Detail string `json:"detail"`

	// Fix says how to resolve it.
	Fix string `json:"fix"`

	// Severities lists the severities the code is reported with; the first is the usual one.
	Severities []Severity `json:"severities"`

	// Unused marks a code that is defined but not currently reported. Its number stays
	// assigned so it is never reused.
	Unused bool `json:"unused,omitempty"`
}

// Group is the entries of one family, in code order.
type Group struct {
	// Family names the group.
	Family Family

	// Slug is the group's anchor, such as "template-analysis".
	Slug string

	// Entries are the group's entries.
	Entries []Entry
}

const (
	// SeverityError stops the build, or marks a failed render.
	SeverityError Severity = "error"

	// SeverityWarning reports a likely mistake; the build continues.
	SeverityWarning Severity = "warning"

	// SeverityInfo reports something worth knowing; the build continues.
	SeverityInfo Severity = "info"

	// FamilyTemplateExpressions is the T0xx codes, reported while parsing a template's
	// expressions and directives.
	FamilyTemplateExpressions Family = "Template expressions"

	// FamilyTemplateAnalysis is the T1xx codes, reported while type-checking templates,
	// components, props, assets and styles.
	FamilyTemplateAnalysis Family = "Template analysis"

	// FamilyRendering is the R0xx codes, reported by generated code while a page renders.
	FamilyRendering Family = "Rendering"

	// maxSuggestionDistance bounds how different a suggestion may be from the input.
	maxSuggestionDistance = 3
)

var (
	// Families lists the families in display order.
	Families = []Family{FamilyTemplateExpressions, FamilyTemplateAnalysis, FamilyRendering}

	// codePattern matches anything shaped like a diagnostic code, in any case.
	codePattern = regexp.MustCompile(`^[a-z][0-9]{3}$`)

	// catalogue holds every entry in code order; built once from the per-family lists.
	catalogue = slices.Concat(templateExpressionCodes, templateAnalysisCodes, renderingCodes)
)

// URL returns the address of the entry's documentation page.
//
// Returns string which is the page address.
func (e *Entry) URL() string {
	return ast_domain.DiagnosticDocsURL(e.Code)
}

// All returns every entry in code order.
//
// Returns []Entry which is a copy of the catalogue.
func All() []Entry {
	return slices.Clone(catalogue)
}

// Lookup finds an entry by its code, in any case ("t141" finds "T141"), or by its name
// ("ComparisonError" or "CodeComparisonError").
//
// Takes id (string) which is the code or name to look up.
//
// Returns Entry which is the entry found.
// Returns bool which is false when no entry matches.
func Lookup(id string) (Entry, bool) {
	trimmed := strings.TrimSpace(id)
	code := strings.ToUpper(trimmed)
	name := strings.TrimPrefix(trimmed, "Code")
	for index := range catalogue {
		if catalogue[index].Code == code || strings.EqualFold(catalogue[index].Name, name) {
			return catalogue[index], true
		}
	}
	return Entry{}, false
}

// Suggest finds the closest diagnostic name, returning no suggestion for code-shaped IDs
// or distant matches.
//
// Takes id (string) which is the unknown code or name.
//
// Returns string which is the suggested code.
func Suggest(id string) string {
	input := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(id), "Code"))
	if input == "" || codePattern.MatchString(input) {
		return ""
	}
	threshold := min(len(input)/3+1, maxSuggestionDistance)
	best, bestDistance := "", threshold+1
	for index := range catalogue {
		distance := linguistics_domain.WagnerFischer(input, strings.ToLower(catalogue[index].Name), 1, 1, 1)
		if distance < bestDistance {
			best, bestDistance = catalogue[index].Code, distance
		}
	}
	return best
}

// ByFamily returns the entries grouped by family, in display order.
//
// Returns []Group which holds one group per family.
func ByFamily() []Group {
	groups := make([]Group, 0, len(Families))
	for _, family := range Families {
		group := Group{Family: family, Slug: FamilySlug(family), Entries: nil}
		for index := range catalogue {
			if catalogue[index].Family == family {
				group.Entries = append(group.Entries, catalogue[index])
			}
		}
		groups = append(groups, group)
	}
	return groups
}

// FamilySlug turns a family name into an anchor, such as "template-analysis".
//
// Takes family (Family) which is the family to name.
//
// Returns string which is the anchor.
func FamilySlug(family Family) string {
	return strings.ReplaceAll(strings.ToLower(string(family)), " ", "-")
}
