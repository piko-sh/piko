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

package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/fuzzkit/tape"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/targets"
	"piko.sh/piko/tests/fuzz/internal/textlimit"
)

type subjectError struct {
	subject string
}

func (e subjectError) Error() string { return "detail with spaces: " + e.subject }

func (e subjectError) FailureSubject() string { return e.subject }

func FuzzBinder(f *testing.F) { fuzzTarget(f, "binder") }

func FuzzAST(f *testing.F) { fuzzTarget(f, "ast") }

func FuzzSubsystem(f *testing.F) {

	names := []string{"actions", "annotator", "assets", "ast", "binder", "collections", "documents", "formatter", "i18n", "inspector", "orchestrator", "persistence", "registry", "resolver", "routing", "storage", "syntax", "utilities", "cache", "search"}
	for index, name := range names {
		f.Add(uint8(index), uint8(0), generate.New(name, 0, 0).Data)
	}
	f.Fuzz(func(t *testing.T, which, file uint8, data []byte) {
		name := names[int(which)%len(names)]
		savedCase := generate.Replay(name, 0, 0, data[:min(len(data), 1024)])
		if name == "formatter" {
			savedCase = oneSource(savedCase, file)
		}
		require.NoError(t, run(t.Context(), &savedCase))
	})
}

func TestEachDirectTargetRuns(t *testing.T) {
	for _, name := range targets.List() {
		if !targets.UsesPackageWorker(name) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			require.Contains(t, checks, name, "missing worker check")
			for seed := range uint64(12) {
				require.NoError(t, run(t.Context(), new(generate.New(name, seed, 0))), "seed %d", seed)
			}
		})
	}
}

func TestRunUnknownSubsystem(t *testing.T) {
	err := run(t.Context(), &casefile.Case{Target: "no-such-subsystem"})

	require.ErrorIs(t, err, errUnknownSubsystem)
	result := resultOf(err)
	assert.True(t, result.Infrastructure)
	assert.Empty(t, result.Property)
}

func TestResultOfMarksUnclassifiedProductFailure(t *testing.T) {
	result := resultOf(errors.New("parse pages/main.pk: unexpected token"))

	assert.Equal(t, unclassifiedProperty, result.Property)
	assert.False(t, result.Infrastructure)
}

func TestResultOfMarksWrappedInfrastructureFailure(t *testing.T) {
	err := fmt.Errorf("cache step: %w", infrastructure("creating cache directory: %w", errors.New("no space left on device")))

	result := resultOf(err)

	assert.True(t, result.Infrastructure)
	assert.Empty(t, result.Property)
	assert.Equal(t, "cache step: creating cache directory: no space left on device", result.Detail)
}

func TestTargetResultEncodingOmitsEmptyClassification(t *testing.T) {
	raw, err := json.Marshal(resultOf(infrastructure("unknown subsystem %q", "x")))
	require.NoError(t, err)

	assert.JSONEq(t, `{"Source":"","Detail":"unknown subsystem \"x\"","Infrastructure":true}`, string(raw))
}

func TestRunRecoversPanicWithoutStack(t *testing.T) {
	const target = "panicking-test-target"
	checks[target] = func(context.Context, *tape.Tape, *casefile.Case) error {
		panic("boom")
	}
	t.Cleanup(func() { delete(checks, target) })

	err := run(t.Context(), &casefile.Case{Target: target})

	require.Error(t, err)
	assert.Equal(t, "panic: boom", err.Error())
	assert.NotContains(t, err.Error(), "goroutine")
	result := resultOf(err)
	assert.Equal(t, panicProperty, result.Property)
	assert.Equal(t, "panic: boom", result.Detail)
}

func TestShimCheckUsesTypedSubject(t *testing.T) {
	subsystemCheck := shimCheck("captcha", func(context.Context, []byte) error {
		return fmt.Errorf("wrapped: %w", subjectError{subject: "captcha-score"})
	})

	err := subsystemCheck(t.Context(), nil, &casefile.Case{})

	failure, ok := errors.AsType[*propertyError](err)
	require.True(t, ok)
	assert.Equal(t, "captcha-score", failure.ID)
}

func TestShimCheckPassesSuccess(t *testing.T) {
	subsystemCheck := shimCheck("csrf", func(context.Context, []byte) error { return nil })

	assert.NoError(t, subsystemCheck(t.Context(), nil, &casefile.Case{}))
}

func TestContractSubject(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{name: "typed subject", err: subjectError{subject: "auth-cookie"}, want: "auth-cookie"},
		{name: "wrapped typed subject", err: fmt.Errorf("outer: %w", subjectError{subject: "auth-cookie"}), want: "auth-cookie"},
		{name: "empty typed subject", err: subjectError{subject: ""}, want: "auth"},
		{name: "single word prefix", err: errors.New("auth-header: missing value"), want: "auth-header"},
		{name: "prefix with spaces", err: errors.New("value was rejected: twice"), want: "auth"},
		{name: "no colon", err: errors.New("rejected"), want: "auth"},
		{name: "empty prefix", err: errors.New(": rejected"), want: "auth"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, contractSubject("auth", tc.err))
		})
	}
}

func TestResultOfBoundsDetail(t *testing.T) {
	err := &propertyError{ID: "detail", Source: "pages/main.pk", Err: errors.New(strings.Repeat("七", maxDetailBytes))}

	result := resultOf(err)

	assert.Equal(t, "detail", result.Property)
	assert.Equal(t, "pages/main.pk", result.Source)
	assert.True(t, utf8.ValidString(result.Detail))
	assert.True(t, strings.HasSuffix(result.Detail, textlimit.Marker))
	assert.LessOrEqual(t, len(result.Detail), maxDetailBytes+len(textlimit.Marker))
}

func fuzzTarget(f *testing.F, target string) {
	f.Helper()
	for _, seed := range []uint64{0, 1, 23, 55} {
		f.Add(generate.New(target, seed, 0).Data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		require.NoError(t, run(t.Context(), new(generate.Replay(target, 0, 0, data[:min(len(data), 1024)]))))
	})
}

func oneSource(savedCase casefile.Case, which uint8) casefile.Case {
	names := sourceNames(savedCase)
	name := names[int(which)%len(names)]
	savedCase.Files = map[string]string{name: savedCase.Files[name]}
	return savedCase
}

func sourceNames(savedCase casefile.Case) []string {
	names := []string{"pages/main.pk"}
	for _, name := range slices.Sorted(maps.Keys(savedCase.Files)) {
		if name != names[0] && (filepath.Ext(name) == ".pk" || filepath.Ext(name) == ".pkc") {
			names = append(names, name)
		}
	}
	return names
}
