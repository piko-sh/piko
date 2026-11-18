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

package provider_otter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

const (
	hotPathKeyCount = 1024
)

type hotPathDocument struct {
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Embedding []float32 `json:"embedding"`
	Views     int       `json:"views"`
}

type hotPathDocumentCodec struct{}

func (hotPathDocumentCodec) EncodeValue(value hotPathDocument) ([]byte, error) {
	return json.Marshal(value)
}

func (hotPathDocumentCodec) DecodeValue(data []byte) (hotPathDocument, error) {
	var document hotPathDocument
	err := json.Unmarshal(data, &document)
	return document, err
}

type hotPathShape struct {
	schema *cache_dto.SearchSchema
	name   string
	tags   []string
}

var (
	hotPathShapes = []hotPathShape{
		{name: "plain", schema: nil, tags: nil},
		{name: "tags", schema: nil, tags: []string{"category:news", "author:7"}},
		{
			name:   "text",
			schema: cache_dto.NewSearchSchema(cache_dto.TextField("Title"), cache_dto.TextField("Body")),
			tags:   nil,
		},
		{
			name:   "vector",
			schema: cache_dto.NewSearchSchema(cache_dto.VectorField("Embedding", 8)),
			tags:   nil,
		},
	}
)

func newHotPathCache(b *testing.B, walEnabled bool, schema *cache_dto.SearchSchema) *OtterAdapter[string, hotPathDocument] {
	b.Helper()

	var providerSpecific any
	if walEnabled {
		walConfig := wal_domain.DefaultConfig(b.TempDir())
		walConfig.EnableCompression = false
		walConfig.SnapshotThreshold = 1 << 30
		providerSpecific = PersistenceConfig[string, hotPathDocument]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: hotPathDocumentCodec{},
		}
	}

	adapter, err := OtterProviderFactory(cache_dto.Options[string, hotPathDocument]{
		MaximumEntries:   hotPathKeyCount * 4,
		SearchSchema:     schema,
		ProviderSpecific: providerSpecific,
	})
	if err != nil {
		b.Fatalf("creating cache: %v", err)
	}
	b.Cleanup(func() { _ = adapter.Close(context.Background()) })

	cache, ok := adapter.(*OtterAdapter[string, hotPathDocument])
	if !ok {
		b.Fatalf("unexpected adapter type %T", adapter)
	}
	return cache
}

func hotPathKeys() []string {
	keys := make([]string, hotPathKeyCount)
	for index := range keys {
		keys[index] = fmt.Sprintf("document-%d", index)
	}
	return keys
}

func hotPathValues() []hotPathDocument {
	words := []string{"cache", "write", "ahead", "log", "otter", "bucket", "search", "vector"}
	values := make([]hotPathDocument, hotPathKeyCount)
	for index := range values {
		embedding := make([]float32, 8)
		for component := range embedding {
			embedding[component] = float32((index*31+component*17)%97) / 97
		}
		values[index] = hotPathDocument{
			Title:     fmt.Sprintf("%s %s %d", words[index%len(words)], words[(index+3)%len(words)], index),
			Body:      fmt.Sprintf("the %s of the %s keeps the %s in order", words[(index+1)%len(words)], words[(index+5)%len(words)], words[(index+2)%len(words)]),
			Embedding: embedding,
			Views:     index,
		}
	}
	return values
}

func BenchmarkHotPath_Set(b *testing.B) {
	keys, values := hotPathKeys(), hotPathValues()
	for _, walEnabled := range []bool{false, true} {
		for _, shape := range hotPathShapes {
			b.Run(fmt.Sprintf("wal=%t/%s", walEnabled, shape.name), func(b *testing.B) {
				cache := newHotPathCache(b, walEnabled, shape.schema)
				ctx := context.Background()
				b.ReportAllocs()
				index := 0
				for b.Loop() {
					if err := cache.Set(ctx, keys[index%hotPathKeyCount], values[index%hotPathKeyCount], shape.tags...); err != nil {
						b.Fatal(err)
					}
					index++
				}
			})
		}
	}
}

func BenchmarkHotPath_SetParallel(b *testing.B) {
	keys, values := hotPathKeys(), hotPathValues()
	for _, keyMode := range []string{"distinct", "same"} {
		for _, walEnabled := range []bool{false, true} {
			for _, shape := range hotPathShapes[:2] {
				b.Run(fmt.Sprintf("%s/wal=%t/%s", keyMode, walEnabled, shape.name), func(b *testing.B) {
					cache := newHotPathCache(b, walEnabled, shape.schema)
					ctx := context.Background()
					var worker atomic.Int64
					b.ReportAllocs()
					b.RunParallel(func(pb *testing.PB) {
						offset := int(worker.Add(1)) * 97
						index := 0
						for pb.Next() {
							slot := 0
							if keyMode == "distinct" {
								slot = (offset + index) % hotPathKeyCount
							}
							if err := cache.Set(ctx, keys[slot], values[(offset+index)%hotPathKeyCount], shape.tags...); err != nil {
								b.Error(err)
								return
							}
							index++
						}
					})
				})
			}
		}
	}
}

func BenchmarkHotPath_SetDeferred(b *testing.B) {
	keys, values := hotPathKeys(), hotPathValues()
	cache := newHotPathCache(b, true, nil)
	ctx := context.Background()
	b.ReportAllocs()
	index := 0
	for b.Loop() {
		wait, err := cache.SetDeferred(ctx, keys[index%hotPathKeyCount], values[index%hotPathKeyCount])
		if err != nil {
			b.Fatal(err)
		}
		if index%64 == 63 {
			if err := wait(); err != nil {
				b.Fatal(err)
			}
		}
		index++
	}
}

func BenchmarkHotPath_Invalidate(b *testing.B) {
	keys, values := hotPathKeys(), hotPathValues()
	for _, walEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("wal=%t", walEnabled), func(b *testing.B) {
			cache := newHotPathCache(b, walEnabled, nil)
			ctx := context.Background()
			b.ReportAllocs()
			index := 0
			for b.Loop() {
				slot := index % hotPathKeyCount
				if err := cache.Set(ctx, keys[slot], values[slot]); err != nil {
					b.Fatal(err)
				}
				if err := cache.Invalidate(ctx, keys[slot]); err != nil {
					b.Fatal(err)
				}
				index++
			}
		})
	}
}

func BenchmarkHotPath_BulkSet(b *testing.B) {
	keys, values := hotPathKeys(), hotPathValues()
	items := make(map[string]hotPathDocument, 64)
	for index := range 64 {
		items[keys[index]] = values[index]
	}
	for _, walEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("wal=%t", walEnabled), func(b *testing.B) {
			cache := newHotPathCache(b, walEnabled, nil)
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				if err := cache.BulkSet(ctx, items); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkHotPath_Get(b *testing.B) {
	keys, values := hotPathKeys(), hotPathValues()
	for _, walEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("wal=%t", walEnabled), func(b *testing.B) {
			cache := newHotPathCache(b, walEnabled, nil)
			ctx := context.Background()
			for index := range hotPathKeyCount {
				if err := cache.Set(ctx, keys[index], values[index]); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				index := 0
				for pb.Next() {
					if _, ok, _ := cache.GetIfPresent(ctx, keys[index%hotPathKeyCount]); !ok {
						b.Error("missing key")
						return
					}
					index++
				}
			})
		})
	}
}
