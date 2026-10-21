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

package otter

import (
	"fmt"
	"testing"
	"time"
)

const (
	indexPopulation = 10_000
)

func indexIDs(count int) []string {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = fmt.Sprintf("0192f2a4-7a8e-7c3d-9b1e-%012d", i)
	}
	return ids
}

func BenchmarkTimeIndex(b *testing.B) {
	d := newBenchDAL(b)
	ids := indexIDs(indexPopulation + 1)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, id := range ids[:indexPopulation] {
		d.executeIndex.Add(id, base.Add(time.Duration(i)*time.Millisecond))
	}
	moving := ids[indexPopulation]
	due := base.Add(time.Minute)

	b.Run("op=add-remove", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			d.executeIndex.Add(moving, due)
			d.executeIndex.Remove(moving)
		}
	})
	b.Run("op=range", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = d.executeIndex.KeysThrough(due, maintenanceBatchSize)
		}
	})
}

func BenchmarkTagIndexChurn(b *testing.B) {
	d := newBenchDAL(b)
	ids := indexIDs(indexPopulation + 1)
	for _, id := range ids[:indexPopulation] {
		d.workflowIndex.AddSingle(id, id)
	}
	moving := ids[indexPopulation]

	b.ReportAllocs()
	for b.Loop() {
		d.workflowIndex.AddSingle(moving, moving)
		d.workflowIndex.RemoveSingle(moving, moving)
	}
}

func newBenchDAL(b *testing.B) *DAL {
	dal, err := NewOtterDAL(Config{Capacity: indexPopulation * 2})
	if err != nil {
		b.Fatalf("creating DAL: %v", err)
	}
	b.Cleanup(func() { _ = dal.Close() })
	d, ok := dal.(*DAL)
	if !ok {
		b.Fatalf("NewOtterDAL returned %T, want *DAL", dal)
	}
	return d
}
