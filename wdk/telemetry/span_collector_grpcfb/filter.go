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
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanFilter decides whether a finished span is forwarded off-box.
type SpanFilter func(sdktrace.ReadOnlySpan) bool

// WithSpanFilter installs a predicate consulted for every finished span before any
// translation work happens; only spans it keeps reach the wire.
//
// Takes fn (SpanFilter) which is the predicate deciding whether a span is forwarded.
//
// Returns Option which configures the collector with the supplied filter.
func WithSpanFilter(fn SpanFilter) Option {
	return func(c *Collector) {
		if fn != nil {
			c.filter = fn
		}
	}
}

// KeepScopePrefixes keeps a span when its instrumentation scope name starts with any of
// the supplied prefixes, and drops it otherwise.
//
// Takes prefixes (...string) which are the instrumentation scope name prefixes to keep.
//
// Returns SpanFilter which keeps spans whose scope matches one of the prefixes.
func KeepScopePrefixes(prefixes ...string) SpanFilter {
	kept := slices.Clone(prefixes)
	if len(kept) == 0 {
		return func(s sdktrace.ReadOnlySpan) bool { return s != nil }
	}
	return func(s sdktrace.ReadOnlySpan) bool {
		if s == nil {
			return false
		}
		name := s.InstrumentationScope().Name
		for _, prefix := range kept {
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
		return false
	}
}

// DropSpanNames drops a span whose operation name matches any of the supplied names
// exactly, and keeps everything else.
//
// Takes names (...string) which are the exact span operation names to drop.
//
// Returns SpanFilter which drops spans whose name matches one of the names.
func DropSpanNames(names ...string) SpanFilter {
	if len(names) == 0 {
		return func(s sdktrace.ReadOnlySpan) bool { return s != nil }
	}

	dropped := make(map[string]struct{}, len(names))
	for _, name := range names {
		dropped[name] = struct{}{}
	}

	return func(s sdktrace.ReadOnlySpan) bool {
		if s == nil {
			return false
		}
		_, drop := dropped[s.Name()]

		return !drop
	}
}

// MinDuration keeps a span only when it ran for at least d.
//
// Takes d (time.Duration) which is the shortest span duration still forwarded.
//
// Returns SpanFilter which keeps spans at or above the threshold.
func MinDuration(d time.Duration) SpanFilter {
	if d <= 0 {
		return func(s sdktrace.ReadOnlySpan) bool { return s != nil }
	}
	return func(s sdktrace.ReadOnlySpan) bool {
		if s == nil {
			return false
		}
		return s.EndTime().Sub(s.StartTime()) >= d
	}
}

// KeepErrors keeps a span whose status code is Error, regardless of anything else.
//
// Returns SpanFilter which keeps spans recorded with an error status.
func KeepErrors() SpanFilter {
	return func(s sdktrace.ReadOnlySpan) bool {
		return s != nil && s.Status().Code == codes.Error
	}
}

// AllOf keeps a span only when every supplied filter keeps it.
//
// Takes filters (...SpanFilter) which must all agree to keep a span.
//
// Returns SpanFilter which is the conjunction of the supplied filters.
func AllOf(filters ...SpanFilter) SpanFilter {
	live := compactFilters(filters)
	if len(live) == 0 {
		return func(sdktrace.ReadOnlySpan) bool { return true }
	}
	return func(s sdktrace.ReadOnlySpan) bool {
		for _, f := range live {
			if !f(s) {
				return false
			}
		}
		return true
	}
}

// AnyOf keeps a span when at least one supplied filter keeps it.
//
// Takes filters (...SpanFilter) which are consulted until one keeps the span.
//
// Returns SpanFilter which is the disjunction of the supplied filters.
func AnyOf(filters ...SpanFilter) SpanFilter {
	live := compactFilters(filters)
	if len(live) == 0 {
		return func(sdktrace.ReadOnlySpan) bool { return true }
	}
	return func(s sdktrace.ReadOnlySpan) bool {
		for _, f := range live {
			if f(s) {
				return true
			}
		}
		return false
	}
}

// compactFilters copies the non-nil filters so a caller mutating its slice afterwards
// cannot change the behaviour of a filter already installed on a collector.
//
// Takes filters ([]SpanFilter) which is the caller's slice, possibly holding nils.
//
// Returns []SpanFilter which holds only the non-nil filters, in order.
func compactFilters(filters []SpanFilter) []SpanFilter {
	return slices.DeleteFunc(slices.Clone(filters), func(filter SpanFilter) bool {
		return filter == nil
	})
}
