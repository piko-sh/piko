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

package stablepool

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

type benchObj struct {
	Link
	Name   string
	Data   []byte
	Result int64
	_      [16]byte
}

func benchInit(o *benchObj) { o.Name = "test" }
func benchClean(o *benchObj) {
	o.Name = ""
	o.Data = o.Data[:0]
}

func work(obj *benchObj, innerIters, appendCount int) {
	obj.Name = "cpu_test"

	var result int64
	for i := range innerIters {
		result += int64(i * i * i)
		result ^= int64(i << 3)
		if i%1000 == 0 {
			result = result*31 + int64(i)
		}
	}
	obj.Result = result

	if cap(obj.Data) < appendCount {
		obj.Data = make([]byte, 0, appendCount)
	}
	obj.Data = obj.Data[:0]
	for i := range appendCount {
		obj.Data = append(obj.Data, byte(result>>uint(i%8)))
	}
}

const (
	benchParallelism = 1000
)

type scenario struct {
	name        string
	innerIters  int
	appendCount int
}

var (
	scenarios = []scenario{
		{"pool_only", 0, 0},
		{"low", 500, 32},
		{"medium", 10_000, 100},
		{"high", 100_000, 256},
		{"extreme", 1_000_000, 256},
	}
)

const (
	benchCapacity = 16384
)

func BenchmarkStablepool(b *testing.B) {
	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			p, err := New(benchInit, benchClean, benchCapacity)
			if err != nil {
				b.Fatal(err)
			}
			b.SetParallelism(benchParallelism)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					obj := p.Get()
					if obj == nil {
						b.Fatal("pool drained")
					}
					work(obj, sc.innerIters, sc.appendCount)
					p.Put(obj)
				}
			})
		})
	}
}

func BenchmarkStablepoolGCAware(b *testing.B) {
	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			p, err := New(benchInit, benchClean, benchCapacity, WithMode[benchObj](ModeGCAware))
			if err != nil {
				b.Fatal(err)
			}
			b.SetParallelism(benchParallelism)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					obj := p.Get()
					if obj == nil {
						b.Fatal("pool drained (GCAware should never drain)")
					}
					work(obj, sc.innerIters, sc.appendCount)
					p.Put(obj)
				}
			})
		})
	}
}

func BenchmarkSyncPool(b *testing.B) {
	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			p := sync.Pool{New: func() any { return &benchObj{Name: "test"} }}
			b.SetParallelism(benchParallelism)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					obj, ok := p.Get().(*benchObj)
					if !ok {
						b.Fatalf("sync.Pool returned wrong type")
					}
					work(obj, sc.innerIters, sc.appendCount)
					obj.Name = ""
					obj.Data = obj.Data[:0]
					p.Put(obj)
				}
			})
		})
	}
}

func TestPostGCLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("post-GC latency comparison is too slow for -short")
	}
	const rounds = 200
	const opsPerRound = 16
	const capacity = 1024

	measure := func(name string, getFn func() any, putFn func(any)) {
		t.Helper()
		warm := make([]any, capacity)
		for i := range warm {
			warm[i] = getFn()
		}
		for _, v := range warm {
			putFn(v)
		}

		samples := make([]time.Duration, rounds)
		for round := range samples {

			runtime.GC()

			runtime.GC()
			start := time.Now()
			for range opsPerRound {
				obj := getFn()
				putFn(obj)
			}
			samples[round] = time.Since(start) / opsPerRound
		}

		var totalNs int64
		minD := samples[0]
		maxD := samples[0]
		for _, d := range samples {
			totalNs += int64(d)
			if d < minD {
				minD = d
			}
			if d > maxD {
				maxD = d
			}
		}
		mean := time.Duration(totalNs / int64(len(samples)))
		t.Logf("%-12s mean=%v min=%v max=%v", name, mean, minD, maxD)
	}

	sp, err := New(benchInit, benchClean, capacity)
	if err != nil {
		t.Fatal(err)
	}
	measure("stablepool", func() any { return sp.Get() }, func(v any) { sp.Put(v.(*benchObj)) })

	syncP := sync.Pool{New: func() any { return &benchObj{Name: "test"} }}
	measure("sync.Pool", syncP.Get, syncP.Put)
}
