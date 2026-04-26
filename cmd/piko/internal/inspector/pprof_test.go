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

package inspector

import (
	"bytes"
	"errors"
	"regexp"
	"testing"

	"github.com/google/pprof/profile"
)

func makeSimpleProfile() *profile.Profile {
	alpha := &profile.Function{ID: 1, Name: "alpha", Filename: "/src/a.go"}
	beta := &profile.Function{ID: 2, Name: "beta", Filename: "/src/b.go"}

	locAlpha := &profile.Location{
		ID:   1,
		Line: []profile.Line{{Function: alpha, Line: 10}},
	}
	locBeta := &profile.Location{
		ID:   2,
		Line: []profile.Line{{Function: beta, Line: 20}},
	}

	return &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Location: []*profile.Location{locAlpha, locBeta},
				Value:    []int64{100},
			},
			{
				Location: []*profile.Location{locBeta},
				Value:    []int64{200},
			},
		},
		Function: []*profile.Function{alpha, beta},
		Location: []*profile.Location{locAlpha, locBeta},
	}
}

func TestProfileKeyLabel(t *testing.T) {
	t.Parallel()

	t.Run("function only when not by line", func(t *testing.T) {
		t.Parallel()
		key := profileKey{FunctionName: "main.run", FileName: "/src/main.go", Line: 42}
		got := key.label(false)
		if got != "main.run" {
			t.Errorf("got %q, want main.run", got)
		}
	})

	t.Run("function plus file:line when by line", func(t *testing.T) {
		t.Parallel()
		key := profileKey{FunctionName: "main.run", FileName: "/src/main.go", Line: 42}
		got := key.label(true)
		want := "main.run /src/main.go:42"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("function only when by line but file empty", func(t *testing.T) {
		t.Parallel()
		key := profileKey{FunctionName: "main.run"}
		got := key.label(true)
		if got != "main.run" {
			t.Errorf("got %q, want main.run", got)
		}
	})
}

func TestParseProfileSummaryMalformed(t *testing.T) {
	t.Parallel()

	_, err := ParseProfileSummary([]byte("not a real profile"), ProfileAggOpts{})
	if err == nil {
		t.Fatal("expected error for malformed bytes, got nil")
	}
}

func TestAggregateProfileNil(t *testing.T) {
	t.Parallel()

	_, err := AggregateProfile(nil, ProfileAggOpts{})
	if err == nil {
		t.Fatal("expected error for nil profile, got nil")
	}
}

func TestAggregateProfileSampleIndexOutOfRange(t *testing.T) {
	t.Parallel()

	prof := makeSimpleProfile()

	t.Run("negative index", func(t *testing.T) {
		t.Parallel()
		_, err := AggregateProfile(prof, ProfileAggOpts{SampleIndex: -1})
		if err == nil {
			t.Errorf("expected error for negative sample index")
		}
	})

	t.Run("past length", func(t *testing.T) {
		t.Parallel()
		_, err := AggregateProfile(prof, ProfileAggOpts{SampleIndex: 5})
		if err == nil {
			t.Errorf("expected error for sample index past length")
		}
	})
}

func TestAggregateProfileEmpty(t *testing.T) {
	t.Parallel()

	prof := &profile.Profile{
		SampleType: []*profile.ValueType{{Type: "cpu", Unit: "nanoseconds"}},
	}
	summary, err := AggregateProfile(prof, ProfileAggOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(summary.Entries))
	}
	if summary.Total != 0 {
		t.Errorf("Total = %d, want 0", summary.Total)
	}
	if summary.SampleType != "cpu" {
		t.Errorf("SampleType = %q", summary.SampleType)
	}
}

func TestAggregateProfileFunctionGrouping(t *testing.T) {
	t.Parallel()

	prof := makeSimpleProfile()
	summary, err := AggregateProfile(prof, ProfileAggOpts{TopN: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := map[string]ProfileEntry{}
	for _, entry := range summary.Entries {
		got[entry.Function] = entry
	}

	alpha, ok := got["alpha"]
	if !ok {
		t.Fatalf("alpha entry missing")
	}
	if alpha.Flat != 100 {
		t.Errorf("alpha.Flat = %d, want 100", alpha.Flat)
	}
	if alpha.Cum != 100 {
		t.Errorf("alpha.Cum = %d, want 100", alpha.Cum)
	}
	if alpha.Label != "alpha" {
		t.Errorf("alpha.Label = %q, want alpha (no by-line grouping)", alpha.Label)
	}

	beta, ok := got["beta"]
	if !ok {
		t.Fatalf("beta entry missing")
	}
	if beta.Flat != 200 {
		t.Errorf("beta.Flat = %d, want 200", beta.Flat)
	}
	if beta.Cum != 300 {
		t.Errorf("beta.Cum = %d, want 300 (sum of both samples)", beta.Cum)
	}

	if summary.Total != 300 {
		t.Errorf("Total = %d, want 300", summary.Total)
	}

	if summary.Entries[0].Function != "beta" {
		t.Errorf("first entry = %q, want beta", summary.Entries[0].Function)
	}
}

func TestAggregateProfileByLine(t *testing.T) {
	t.Parallel()

	prof := makeSimpleProfile()
	summary, err := AggregateProfile(prof, ProfileAggOpts{ByLine: true, TopN: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, entry := range summary.Entries {
		switch entry.Function {
		case "alpha":
			if entry.Label != "alpha /src/a.go:10" {
				t.Errorf("alpha label = %q", entry.Label)
			}
			if entry.File != "/src/a.go" {
				t.Errorf("alpha File = %q", entry.File)
			}
			if entry.Line != 10 {
				t.Errorf("alpha Line = %d", entry.Line)
			}
		case "beta":
			if entry.Label != "beta /src/b.go:20" {
				t.Errorf("beta label = %q", entry.Label)
			}
		}
	}
}

func TestAggregateProfileFocusRegex(t *testing.T) {
	t.Parallel()

	prof := makeSimpleProfile()

	t.Run("regex matches", func(t *testing.T) {
		t.Parallel()
		summary, err := AggregateProfile(prof, ProfileAggOpts{FocusRegex: regexp.MustCompile(`^alpha$`)})
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if summary.Total != 100 {
			t.Errorf("Total = %d, want 100 (only alpha sample)", summary.Total)
		}
	})

	t.Run("regex does not match", func(t *testing.T) {
		t.Parallel()
		summary, err := AggregateProfile(prof, ProfileAggOpts{FocusRegex: regexp.MustCompile(`^nope$`)})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if summary.Total != 0 {
			t.Errorf("Total = %d, want 0", summary.Total)
		}
	})
}

func TestAggregateProfileTopN(t *testing.T) {
	t.Parallel()

	prof := makeSimpleProfile()

	t.Run("topN of zero falls back to default", func(t *testing.T) {
		t.Parallel()
		summary, err := AggregateProfile(prof, ProfileAggOpts{TopN: 0})
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if len(summary.Entries) != 2 {
			t.Errorf("entries = %d, want 2", len(summary.Entries))
		}
	})

	t.Run("topN negative falls back to default", func(t *testing.T) {
		t.Parallel()
		summary, err := AggregateProfile(prof, ProfileAggOpts{TopN: -5})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(summary.Entries) != 2 {
			t.Errorf("entries = %d, want 2", len(summary.Entries))
		}
	})

	t.Run("topN of 1 keeps only top entry", func(t *testing.T) {
		t.Parallel()
		summary, err := AggregateProfile(prof, ProfileAggOpts{TopN: 1})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(summary.Entries) != 1 {
			t.Errorf("entries = %d, want 1", len(summary.Entries))
		}
		if summary.Entries[0].Function != "beta" {
			t.Errorf("top entry = %q, want beta", summary.Entries[0].Function)
		}
	})

	t.Run("topN larger than entry count returns all entries", func(t *testing.T) {
		t.Parallel()
		summary, err := AggregateProfile(prof, ProfileAggOpts{TopN: 1000})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(summary.Entries) != 2 {
			t.Errorf("entries = %d, want 2", len(summary.Entries))
		}
	})
}

func TestAggregateProfileZeroValueSamplesSkipped(t *testing.T) {
	t.Parallel()

	zeroFn := &profile.Function{ID: 1, Name: "zero"}
	loc := &profile.Location{ID: 1, Line: []profile.Line{{Function: zeroFn, Line: 1}}}
	prof := &profile.Profile{
		SampleType: []*profile.ValueType{{Type: "cpu", Unit: "nanoseconds"}},
		Sample: []*profile.Sample{
			{Location: []*profile.Location{loc}, Value: []int64{0}},
		},
		Function: []*profile.Function{zeroFn},
		Location: []*profile.Location{loc},
	}
	summary, err := AggregateProfile(prof, ProfileAggOpts{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(summary.Entries) != 0 {
		t.Errorf("zero-value sample should be skipped, got %d entries", len(summary.Entries))
	}
}

func TestParseProfileSummaryRoundtrip(t *testing.T) {
	t.Parallel()

	prof := makeSimpleProfile()
	var buffer bytes.Buffer
	if err := prof.Write(&buffer); err != nil {
		t.Fatalf("write: %v", err)
	}

	summary, err := ParseProfileSummary(buffer.Bytes(), ProfileAggOpts{TopN: 10})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if summary.Total != 300 {
		t.Errorf("Total = %d, want 300", summary.Total)
	}
}

func TestAggregateProfileTooManySamples(t *testing.T) {
	t.Parallel()

	dummyFn := &profile.Function{ID: 1, Name: "stub"}
	loc := &profile.Location{ID: 1, Line: []profile.Line{{Function: dummyFn, Line: 1}}}
	sample := &profile.Sample{Location: []*profile.Location{loc}, Value: []int64{1}}

	samples := make([]*profile.Sample, profileMaxSamples+1)
	for index := range samples {
		samples[index] = sample
	}
	prof := &profile.Profile{
		SampleType: []*profile.ValueType{{Type: "cpu", Unit: "nanoseconds"}},
		Sample:     samples,
		Function:   []*profile.Function{dummyFn},
		Location:   []*profile.Location{loc},
	}

	_, err := AggregateProfile(prof, ProfileAggOpts{})
	if err == nil {
		t.Fatal("expected error when sample count exceeds cap")
	}
	if !errors.Is(err, ErrProfileTooManySamples) {
		t.Errorf("err = %v, want ErrProfileTooManySamples", err)
	}
}
