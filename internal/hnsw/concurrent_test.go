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

package hnsw

import (
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/vectormaths"
)

func TestGraph_ConcurrentInsertAndSearch(t *testing.T) {
	const (
		dim     = 16
		writers = 4
		readers = 4
		perOp   = 50
	)

	g := New[int](dim, vectormaths.Cosine, WithRandomSeed(42))

	var wg sync.WaitGroup

	for w := range writers {
		wg.Go(func() {
			randomSource := rand.New(rand.NewPCG(uint64(w), uint64(w>>1|1)))
			for i := range perOp {
				vec := make([]float32, dim)
				for j := range dim {
					vec[j] = randomSource.Float32()
				}
				g.Insert(w*perOp+i, vec)
			}
		})
	}

	for range readers {
		wg.Go(func() {
			randomSource := rand.New(rand.NewPCG(99, 99>>1|1))
			for range perOp {
				q := make([]float32, dim)
				for j := range dim {
					q[j] = randomSource.Float32()
				}
				g.Search(q, 5, 0)
			}
		})
	}

	wg.Wait()

	assert.Equal(t, writers*perOp, g.Len())
}

func TestGraph_ConcurrentDeleteAndSearch(t *testing.T) {
	const (
		dim = 8
		n   = 100
	)

	g := New[int](dim, vectormaths.Cosine, WithRandomSeed(42))
	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))

	for i := range n {
		vec := make([]float32, dim)
		for j := range dim {
			vec[j] = randomSource.Float32()
		}
		g.Insert(i, vec)
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		for i := 0; i < n; i += 2 {
			g.Delete(i)
		}
	})

	for range 4 {
		wg.Go(func() {
			r := rand.New(rand.NewPCG(99, 99>>1|1))
			for range 20 {
				q := make([]float32, dim)
				for j := range dim {
					q[j] = r.Float32()
				}
				g.Search(q, 5, 0)
			}
		})
	}

	wg.Wait()
	assert.Equal(t, n/2, g.Len())
}
