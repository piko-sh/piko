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

//go:build bench

package logger_domain_test

import (
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"piko.sh/piko/internal/logger/logger_domain"
)

func makeTestSlogAttrs(count int) []slog.Attr {
	attrs := make([]slog.Attr, count)
	for i := range count {
		switch i % 4 {
		case 0:
			attrs[i] = slog.String("key"+string(rune('a'+i)), "value")
		case 1:
			attrs[i] = slog.Int("num"+string(rune('a'+i)), i*100)
		case 2:
			attrs[i] = slog.Bool("flag"+string(rune('a'+i)), i%2 == 0)
		case 3:
			attrs[i] = slog.Float64("ratio"+string(rune('a'+i)), float64(i)/10.0)
		}
	}
	return attrs
}

func BenchmarkOtelAttrsFromSlog_Current(b *testing.B) {
	scenarios := []struct {
		name           string
		attributeCount int
	}{
		{name: "0_attrs", attributeCount: 0},
		{name: "1_attr", attributeCount: 1},
		{name: "3_attrs", attributeCount: 3},
		{name: "5_attrs", attributeCount: 5},
		{name: "10_attrs", attributeCount: 10},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			attrs := makeTestSlogAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				result := logger_domain.OtelAttrsFromSlog(attrs)
				_ = result
			}
		})
	}
}

func BenchmarkOtelAttrsFromSlog_NoPool(b *testing.B) {
	scenarios := []struct {
		name           string
		attributeCount int
	}{
		{name: "0_attrs", attributeCount: 0},
		{name: "1_attr", attributeCount: 1},
		{name: "3_attrs", attributeCount: 3},
		{name: "5_attrs", attributeCount: 5},
		{name: "10_attrs", attributeCount: 10},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			attrs := makeTestSlogAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				result := otelAttrsFromSlogNoPool(attrs)
				_ = result
			}
		})
	}
}

func otelAttrsFromSlogNoPool(attrs []slog.Attr) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		logger_domain.AddAttrRecursive(&kvs, "", a)
	}
	return kvs
}

func BenchmarkOtelAttrsFromSlog_PoolDirect(b *testing.B) {
	scenarios := []struct {
		name           string
		attributeCount int
	}{
		{name: "0_attrs", attributeCount: 0},
		{name: "1_attr", attributeCount: 1},
		{name: "3_attrs", attributeCount: 3},
		{name: "5_attrs", attributeCount: 5},
		{name: "10_attrs", attributeCount: 10},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			attrs := makeTestSlogAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				result := otelAttrsFromSlogPoolDirect(attrs)

				_ = result

			}
		})
	}
}

func otelAttrsFromSlogPoolDirect(attrs []slog.Attr) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	kvsPtr, ok := logger_domain.OtelAttrPool.Get().(*[]attribute.KeyValue)
	if !ok {
		kvsPtr = new(make([]attribute.KeyValue, 0, len(attrs)))
	}
	kvs := (*kvsPtr)[:0]
	for _, a := range attrs {
		logger_domain.AddAttrRecursive(&kvs, "", a)
	}

	return kvs
}

func BenchmarkOtelAttrsFromSlog_Inline(b *testing.B) {
	scenarios := []struct {
		name           string
		attributeCount int
	}{
		{name: "0_attrs", attributeCount: 0},
		{name: "1_attr", attributeCount: 1},
		{name: "3_attrs", attributeCount: 3},
		{name: "5_attrs", attributeCount: 5},
		{name: "10_attrs", attributeCount: 10},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			attrs := makeTestSlogAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				if len(attrs) > 0 {
					kvsPtr, ok := logger_domain.OtelAttrPool.Get().(*[]attribute.KeyValue)
					if !ok {
						kvsPtr = new(make([]attribute.KeyValue, 0, len(attrs)))
					}
					kvs := (*kvsPtr)[:0]
					for _, a := range attrs {
						logger_domain.AddAttrRecursive(&kvs, "", a)
					}

					_ = kvs

					*kvsPtr = kvs[:0]
					logger_domain.OtelAttrPool.Put(kvsPtr)
				}
			}
		})
	}
}
