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

package span_collector_grpcfb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

type spanSpec struct {
	scope   string
	name    string
	dur     time.Duration
	errored bool
}

func endSpans(t *testing.T, col *Collector, specs ...spanSpec) {
	t.Helper()

	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(col))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	start := time.Unix(1_700_000_000, 0)
	for _, spec := range specs {
		_, span := tp.Tracer(spec.scope).Start(context.Background(), spec.name,
			trace.WithTimestamp(start))
		if spec.errored {
			span.SetStatus(codes.Error, "boom")
		}
		span.End(trace.WithTimestamp(start.Add(spec.dur)))
	}
}

func names(spans []telemetry_grpcfb.Span) []string {
	out := make([]string, 0, len(spans))
	for _, s := range spans {
		out = append(out, s.Operation)
	}
	return out
}

func TestWithSpanFilter_DropsWhatItRejectsAndCountsIt(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(func(s sdktrace.ReadOnlySpan) bool {
		return s.Name() == "keep"
	}))

	endSpans(t, col,
		spanSpec{scope: "sc", name: "keep", dur: time.Millisecond},
		spanSpec{scope: "sc", name: "drop", dur: time.Millisecond},
		spanSpec{scope: "sc", name: "drop", dur: time.Millisecond},
	)
	drain()

	assert.Equal(t, []string{"keep"}, names(snk.spans()))
	assert.Equal(t, int64(2), col.Filtered(), "a filtered collector says how much it shed")
}

func TestWithSpanFilter_NilIsIgnored(t *testing.T) {
	col := New(nil, WithSpanFilter(nil))
	assert.Nil(t, col.filter, "a nil filter must never replace a configured one")
}

func TestWithSpanFilter_LastCallWins(t *testing.T) {
	col := New(nil,
		WithSpanFilter(func(sdktrace.ReadOnlySpan) bool { return false }),
		WithSpanFilter(func(sdktrace.ReadOnlySpan) bool { return true }),
	)
	require.NotNil(t, col.filter)
	assert.True(t, col.filter(nil), "options replace rather than compose")
}

func TestFiltered_ZeroWithoutAFilter(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client)

	endSpans(t, col, spanSpec{scope: "sc", name: "op", dur: time.Millisecond})
	drain()

	assert.Len(t, snk.spans(), 1)
	assert.Zero(t, col.Filtered())
	assert.Zero(t, (*Collector)(nil).Filtered(), "a nil collector reports nothing filtered")
}

func TestKeepScopePrefixes_KeepsMatchingScopesOnly(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(KeepScopePrefixes(
		"piko/internal/daemon/daemon_adapters",
		"piko/db/",
	)))

	endSpans(t, col,
		spanSpec{scope: "piko/internal/daemon/daemon_adapters", name: "GET /widgets", dur: time.Millisecond},
		spanSpec{scope: "piko/db/sqlite", name: "SELECT", dur: time.Millisecond},
		spanSpec{scope: "piko/internal/orchestrator", name: "gc.sweep", dur: time.Millisecond},
	)
	drain()

	assert.ElementsMatch(t, []string{"GET /widgets", "SELECT"}, names(snk.spans()))
	assert.Equal(t, int64(1), col.Filtered())
}

func TestKeepScopePrefixes_EmptyKeepsEverything(t *testing.T) {

	keep := KeepScopePrefixes()
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(keep))

	endSpans(t, col, spanSpec{scope: "anything", name: "op", dur: time.Millisecond})
	drain()

	assert.Len(t, snk.spans(), 1)
	assert.Zero(t, col.Filtered())
}

func TestDropSpanNames(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(DropSpanNames("noisy", "alsoNoisy")))

	endSpans(t, col,
		spanSpec{scope: "sc", name: "noisy", dur: time.Millisecond},
		spanSpec{scope: "sc", name: "alsoNoisy", dur: time.Millisecond},
		spanSpec{scope: "sc", name: "wanted", dur: time.Millisecond},
	)
	drain()

	assert.Equal(t, []string{"wanted"}, names(snk.spans()))
}

func TestDropSpanNames_EmptyKeepsEverything(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(DropSpanNames()))

	endSpans(t, col, spanSpec{scope: "sc", name: "op", dur: time.Millisecond})
	drain()

	assert.Len(t, snk.spans(), 1)
}

func TestMinDuration(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(MinDuration(50*time.Millisecond)))

	endSpans(t, col,
		spanSpec{scope: "sc", name: "fast", dur: 2 * time.Millisecond},
		spanSpec{scope: "sc", name: "exactly", dur: 50 * time.Millisecond},
		spanSpec{scope: "sc", name: "slow", dur: time.Second},
	)
	drain()

	assert.ElementsMatch(t, []string{"exactly", "slow"}, names(snk.spans()))
	assert.Equal(t, int64(1), col.Filtered())
}

func TestMinDuration_NonPositiveKeepsEverything(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(MinDuration(0)))

	endSpans(t, col, spanSpec{scope: "sc", name: "instant"})
	drain()

	assert.Len(t, snk.spans(), 1)
}

func TestAnyOf_KeepErrorsSurvivesADurationThreshold(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(AnyOf(KeepErrors(), MinDuration(time.Second))))

	endSpans(t, col,
		spanSpec{scope: "sc", name: "fastFailure", dur: 2 * time.Millisecond, errored: true},
		spanSpec{scope: "sc", name: "fastSuccess", dur: 2 * time.Millisecond},
		spanSpec{scope: "sc", name: "slowSuccess", dur: 2 * time.Second},
	)
	drain()

	assert.ElementsMatch(t, []string{"fastFailure", "slowSuccess"}, names(snk.spans()))
}

func TestAllOf_RequiresEveryFilter(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithSpanFilter(AllOf(
		KeepScopePrefixes("piko/db/"),
		MinDuration(10*time.Millisecond),
	)))

	endSpans(t, col,
		spanSpec{scope: "piko/db/sqlite", name: "slowQuery", dur: time.Second},
		spanSpec{scope: "piko/db/sqlite", name: "fastQuery", dur: time.Millisecond},
		spanSpec{scope: "other", name: "slowOther", dur: time.Second},
	)
	drain()

	assert.Equal(t, []string{"slowQuery"}, names(snk.spans()))
	assert.Equal(t, int64(2), col.Filtered())
}

func TestCombinators_EmptyAndNilDegradeTowardsKeeping(t *testing.T) {
	always := func(sdktrace.ReadOnlySpan) bool { return true }

	assert.True(t, AllOf()(nil), "AllOf with no filters keeps everything")
	assert.True(t, AnyOf()(nil), "AnyOf with no filters keeps everything")
	assert.True(t, AllOf(nil, nil)(nil), "nil filters are skipped, not treated as drops")
	assert.True(t, AnyOf(nil, always)(nil))
}

func TestCombinators_CopyTheirInput(t *testing.T) {
	never := SpanFilter(func(sdktrace.ReadOnlySpan) bool { return false })
	filters := []SpanFilter{func(sdktrace.ReadOnlySpan) bool { return true }}

	combined := AllOf(filters...)
	filters[0] = never

	assert.True(t, combined(nil), "mutating the caller's slice must not change the filter")
}

func TestFilters_TreatANilSpanAsDropped(t *testing.T) {

	assert.False(t, KeepScopePrefixes("x")(nil))
	assert.False(t, DropSpanNames("x")(nil))
	assert.False(t, MinDuration(time.Second)(nil))
	assert.False(t, KeepErrors()(nil))
}

func TestOnEnd_PanickingFilterIsContainedAndCounted(t *testing.T) {
	client, sink, drain := newSinkClient(t)
	collector := New(client, WithSpanFilter(func(sdktrace.ReadOnlySpan) bool {
		panic("filter boom")
	}))

	require.NotPanics(t, func() {
		endSpans(t, collector,
			spanSpec{scope: "sc", name: "one", dur: time.Millisecond},
			spanSpec{scope: "sc", name: "two", dur: time.Millisecond},
			spanSpec{scope: "sc", name: "three", dur: time.Millisecond},
		)
	})
	drain()

	assert.Empty(t, sink.spans(), "no span survives a filter that cannot answer")
	assert.EqualValues(t, 3, collector.Recovered(), "every lost span is counted")
}

func TestPanics_NilCollectorReportsNothing(t *testing.T) {
	assert.Zero(t, (*Collector)(nil).Recovered())
}
