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

package i18n_domain

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// NoPluralForm is the TemplateProblem.FormIndex of a problem in a template that has no
	// plural forms.
	NoPluralForm = -1

	// maxReportedTemplateProblems caps how many failing templates are named in a load
	// warning so a badly broken translation set cannot produce an oversized log entry.
	maxReportedTemplateProblems = 10
)

// TemplateProblem describes a translation template, or one plural form of it, that could
// not be parsed. The affected template or form renders as its literal text.
type TemplateProblem struct {
	// Locale is the locale of the translation, when known.
	Locale string

	// Key is the translation key, when known.
	Key string

	// Messages holds the parser's messages.
	Messages []string

	// FormIndex is the zero-based index of the plural form that failed, or NoPluralForm when
	// the template has no plural forms.
	FormIndex int
}

// String describes the problem for log output.
//
// Returns string which names the locale, key, plural form (when there is one) and the
// parser messages.
func (p TemplateProblem) String() string {
	location := p.Locale + ":" + p.Key
	if p.FormIndex != NoPluralForm {
		location += "[" + strconv.Itoa(p.FormIndex) + "]"
	}
	return location + ": " + strings.Join(p.Messages, "; ")
}

// ReportTemplateProblems logs one warning naming the translation templates that could not
// be parsed while loading a set of translations. Call it once per load, where the
// translations are loaded, so each problem is reported exactly once and never on the
// render path.
//
// Takes source (string) which names where the translations came from.
// Takes problems ([]TemplateProblem) which are the problems to report, sorted in place by
// locale, key and plural form; nothing is logged when it is empty.
func ReportTemplateProblems(ctx context.Context, source string, problems []TemplateProblem) {
	if len(problems) == 0 {
		return
	}
	slices.SortFunc(problems, func(a, b TemplateProblem) int {
		return cmp.Or(cmp.Compare(a.Locale, b.Locale), cmp.Compare(a.Key, b.Key), cmp.Compare(a.FormIndex, b.FormIndex))
	})
	_, l := logger_domain.From(ctx, log)
	l.Warn("Some i18n translation templates could not be parsed and render as literal text",
		logger_domain.String("source", source),
		logger_domain.Int("problem_count", len(problems)),
		logger_domain.Strings("problems", DescribeTemplateProblems(problems, maxReportedTemplateProblems)))
}

// DescribeTemplateProblems renders up to limit problems for a log entry, noting how many
// more were left out so a badly broken translation set cannot produce an oversized entry.
//
// Takes problems ([]TemplateProblem) which are the problems to describe.
// Takes limit (int) which caps how many problems are described individually.
//
// Returns []string which holds one description per reported problem, plus a count of the
// omitted ones when the limit is exceeded.
func DescribeTemplateProblems(problems []TemplateProblem, limit int) []string {
	reported := problems[:min(len(problems), max(limit, 0))]
	descriptions := make([]string, 0, len(reported)+1)
	for _, problem := range reported {
		descriptions = append(descriptions, problem.String())
	}
	if omitted := len(problems) - len(reported); omitted > 0 {
		descriptions = append(descriptions, strconv.Itoa(omitted)+" more")
	}
	return descriptions
}
