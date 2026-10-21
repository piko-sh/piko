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

package orchestrator_test_bench

import (
	"testing"
)

func BenchmarkLifecycle(b *testing.B) {
	forEachLayer(b, func(b *testing.B, opts rigOptions) {
		r := newRig(b, opts)

		b.ReportAllocs()
		for b.Loop() {
			base := r.settledCount()
			if err := r.dispatcher.Dispatch(r.ctx, newBenchTask()); err != nil {
				b.Fatalf("dispatch: %v", err)
			}
			if _, err := r.completions.await(1, base); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkServiceDispatchWait(b *testing.B) {
	forEachLayer(b, func(b *testing.B, opts rigOptions) {
		r := newRig(b, opts)

		b.ReportAllocs()
		for b.Loop() {
			base := r.settledCount()
			receipt, err := r.service.Dispatch(r.ctx, newBenchTask())
			if err != nil {
				b.Fatalf("dispatch: %v", err)
			}
			if err := receipt.Wait(r.ctx); err != nil {
				b.Fatalf("wait: %v", err)
			}
			if _, err := r.completions.await(1, base); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func forEachLayer(b *testing.B, fn func(*testing.B, rigOptions)) {
	for _, store := range storeKinds {
		for _, transport := range transportKinds {
			opts := rigOptions{store: store, transport: transport}
			for _, mode := range logModes {
				b.Run(opts.String()+"/log="+mode, func(b *testing.B) {
					applyLogMode(b, mode)
					fn(b, opts)
				})
			}
		}
	}
}
