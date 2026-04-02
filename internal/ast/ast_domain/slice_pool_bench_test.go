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

package ast_domain

import (
	"sync"
	"testing"
)

type testNode struct {
	value int
}

var preAllocNodes = []*testNode{{value: 1}, {value: 2}}

var directPool = sync.Pool{
	New: func() any { return make([]*testNode, 0, 8) },
}

func BenchmarkPool_Direct(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		s := directPool.Get().([]*testNode)[:0]
		s = append(s, preAllocNodes[0])
		s = append(s, preAllocNodes[1])
		clear(s)
		directPool.Put(s)
	}
}

var ptrPool = sync.Pool{
	New: func() any {
		return new(make([]*testNode, 0, 8))
	},
}

func BenchmarkPool_Pointer(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		ptr, ok := ptrPool.Get().(*[]*testNode)
		if !ok {
			b.Fatal("Pool returned wrong type")
		}
		s := (*ptr)[:0]
		s = append(s, preAllocNodes[0])
		s = append(s, preAllocNodes[1])
		clear(s)
		*ptr = s
		ptrPool.Put(ptr)
	}
}

func BenchmarkPoolParallel_Direct(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := directPool.Get().([]*testNode)[:0]
			s = append(s, preAllocNodes[0])
			s = append(s, preAllocNodes[1])
			clear(s)
			directPool.Put(s)
		}
	})
}

func BenchmarkPoolParallel_Pointer(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ptr, ok := ptrPool.Get().(*[]*testNode)
			if !ok {
				b.Fatal("Pool returned wrong type")
			}
			s := (*ptr)[:0]
			s = append(s, preAllocNodes[0])
			s = append(s, preAllocNodes[1])
			clear(s)
			*ptr = s
			ptrPool.Put(ptr)
		}
	})
}
