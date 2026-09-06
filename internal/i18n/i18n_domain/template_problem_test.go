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
	"bytes"
	"context"
	"log/slog"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/logger/logger_domain"
)

func TestTemplateProblem_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		want    string
		problem TemplateProblem
	}{
		{
			name:    "a whole template names the locale and key",
			problem: TemplateProblem{Locale: "en-GB", Key: "greeting", Messages: []string{"first", "second"}, FormIndex: NoPluralForm},
			want:    "en-GB:greeting: first; second",
		},
		{
			name:    "a plural form includes its index",
			problem: TemplateProblem{Locale: "fr-FR", Key: "items", Messages: []string{"bad"}, FormIndex: 2},
			want:    "fr-FR:items[2]: bad",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, tc.problem.String())
		})
	}
}

func TestDescribeTemplateProblems(t *testing.T) {
	t.Parallel()

	problems := make([]TemplateProblem, 0, 5)
	for index := range 5 {
		problems = append(problems, TemplateProblem{Locale: "en", Key: strconv.Itoa(index), Messages: []string{"bad"}, FormIndex: NoPluralForm})
	}

	testCases := []struct {
		name  string
		want  []string
		limit int
	}{
		{name: "everything fits under the limit", limit: 10, want: []string{"en:0: bad", "en:1: bad", "en:2: bad", "en:3: bad", "en:4: bad"}},
		{name: "problems beyond the limit are counted", limit: 2, want: []string{"en:0: bad", "en:1: bad", "3 more"}},
		{name: "a negative limit describes none", limit: -1, want: []string{"5 more"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, DescribeTemplateProblems(problems, tc.limit))
		})
	}
}

func TestReportTemplateProblems(t *testing.T) {
	t.Parallel()

	problemFor := func(locale, key string, formIndex int) TemplateProblem {
		problem := TemplateProblem{}
		problem.Locale = locale
		problem.Key = key
		problem.FormIndex = formIndex
		problem.Messages = []string{"Unterminated expression: expected '}'"}
		return problem
	}

	testCases := []struct {
		name         string
		wantContains []string
		problems     []TemplateProblem
	}{
		{name: "no problems logs nothing", problems: nil, wantContains: nil},
		{
			name: "problems are reported once, ordered by locale and key",
			problems: []TemplateProblem{
				problemFor("fr-FR", "b", NoPluralForm),
				problemFor("en-GB", "a", 1),
			},
			wantContains: []string{"problem_count=2", "en-GB:a[1]: Unterminated expression: expected '}',fr-FR:b: Unterminated"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, logOutput := captureLogs()

			ReportTemplateProblems(ctx, "i18n", tc.problems)

			if len(tc.wantContains) == 0 {
				assert.Empty(t, logOutput.String())
				return
			}
			assert.Equal(t, 1, strings.Count(logOutput.String(), "could not be parsed"))
			for _, want := range tc.wantContains {
				assert.Contains(t, logOutput.String(), want)
			}
		})
	}
}

func captureLogs() (context.Context, *bytes.Buffer) {
	logOutput := new(bytes.Buffer)
	ctx := logger_domain.WithLogger(context.Background(),
		logger_domain.New(slog.New(slog.NewTextHandler(logOutput, nil)), "i18n-test"))
	return ctx, logOutput
}
