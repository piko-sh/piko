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

package generate

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"piko.sh/safeconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// diagnosticPaddingChoices bounds the blank lines placed before generated sources.
	diagnosticPaddingChoices = 3
)

// addDiagnostics adds source and expected observations for paired diagnostic sources.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addDiagnostics(c *casefile.Case, p Project) {
	cases := []struct{ id, good, bad, code string }{
		{id: "member", good: "state.User.Name", bad: "state.User.", code: "T010"},
		{id: "leap-year", good: "d'2024-02-29'", bad: "d'2023-02-29'", code: "T004"},
		{id: "clock", good: "t'23:59:59'", bad: "t'25:59:59'", code: "T004"},
		{id: "rune", good: "r'a'", bad: "r'ab'", code: "T005"},
	}
	chosen := cases[(c.Index+c.Seed)%uint64(len(cases))]
	good, bad := "valid/"+chosen.id+".pk", "invalid/"+chosen.id+".pk"
	source := func(expression string) string {
		return "<template>\n" + strings.Repeat("\n", p.Count%diagnosticPaddingChoices) + fmt.Sprintf(`<section>É界🪶<output p-text=%q></output></section>`, expression) + "\n</template>"
	}
	c.Files[good], c.Files[bad] = source(chosen.good), source(chosen.bad)
	diagnostic := &casefile.DiagnosticPair{
		ID:               chosen.id,
		GoodPath:         good,
		BadPath:          bad,
		Code:             chosen.code,
		Line:             2 + p.Count%diagnosticPaddingChoices,
		Column:           0,
		ExpressionOffset: 0,
		Expression:       "",
		SectionPath:      "",
		Sections:         nil,
	}
	c.Expected.Diagnostic = diagnostic
	prefix := `<section>É界🪶<output p-text="`
	diagnostic.Expression = strings.TrimSuffix(chosen.bad[2:], "'")
	if chosen.id == "member" {
		diagnostic.ExpressionOffset = len(chosen.bad) - 1
		diagnostic.Expression = ""
	}
	diagnostic.Column = utf8.RuneCountInString(prefix) + 1 + diagnostic.ExpressionOffset
	diagnostic.SectionPath = "section-order/location-probe.pkc"
	diagnostic.Sections, c.Files[diagnostic.SectionPath] = sectionSource(c.Index, p.Count)
	c.Features = append(c.Features, "single-change-diagnostic", "diagnostic-repair", "ordered-source-spans")
}

// sectionSource emits source sections with independently recorded byte offsets.
//
// Takes index (uint64) which identifies the case's stream position.
// Takes count (int) which is the number of source sections to emit.
//
// Returns []casefile.SourceSection which holds expected section coordinates.
// Returns string which is the emitted source text.
func sectionSource(index uint64, count int) ([]casefile.SourceSection, string) {
	type block struct {
		values     map[string]string
		tag        string
		attributes string
		content    string
	}
	blocks := []block{
		{tag: "template", attributes: ` name="location-probe"`, content: `<main p-ref="tick" p-timeline:anchor="tick">見出し</main>`, values: map[string]string{"name": "location-probe"}},
		{tag: "style", attributes: "", content: ".a { color: navy; }", values: nil},
		{tag: "script", attributes: ` lang="ts"`, content: "const café = 1;", values: map[string]string{"lang": "ts"}},
		{tag: "i18n", attributes: ` lang="json"`, content: `{"en":{"caption":"見出し"}}`, values: map[string]string{"lang": "json"}},
		{tag: "piko:timeline", attributes: "", content: `<piko:at time="500ms"><piko:hide ref="tick"/></piko:at>`, values: nil},
		{tag: "style", attributes: " global", content: ".a { color: navy; }", values: map[string]string{"global": ""}},
	}
	var source strings.Builder
	sections := make([]casefile.SourceSection, 0, len(blocks))
	rotation := safeconv.Uint64ToInt(index % uint64(len(blocks)))
	for position := range blocks {
		selected := blocks[(position+rotation)%len(blocks)]
		source.WriteString(strings.Repeat("\n", 1+count%diagnosticPaddingChoices) + "<!-- É界🪶 --> ")
		section := casefile.SourceSection{Tag: selected.tag, Attributes: selected.values, Start: source.Len(), ContentStart: 0, ContentEnd: 0}
		fmt.Fprintf(&source, "<%s%s>", selected.tag, selected.attributes)
		section.ContentStart = source.Len()
		source.WriteString(selected.content)
		section.ContentEnd = source.Len()
		fmt.Fprintf(&source, "</%s>", selected.tag)
		sections = append(sections, section)
	}
	return sections, source.String()
}
