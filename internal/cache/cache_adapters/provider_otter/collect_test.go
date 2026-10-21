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

package provider_otter

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"weak"

	"piko.sh/piko/internal/cache/cache_dto"
)

func TestOtterAdapter_IsCollectedWhenUnreachable(t *testing.T) {
	for _, closeFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("closed=%t", closeFirst), func(t *testing.T) {
			ref := fillAndDrop(t, closeFirst)
			for range 5 {
				runtime.GC()
				if ref.Value() == nil {
					return
				}
			}
			t.Fatal("an unreachable cache adapter was never collected")
		})
	}
}

func fillAndDrop(t *testing.T, closeFirst bool) weak.Pointer[OtterAdapter[string, []byte]] {
	t.Helper()
	ctx := context.Background()
	provider, err := OtterProviderFactory(cache_dto.Options[string, []byte]{})
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := provider.(*OtterAdapter[string, []byte])
	if !ok {
		t.Fatalf("factory returned %T", provider)
	}
	for i := range 256 {
		if err := adapter.Set(ctx, fmt.Sprintf("key-%d", i), make([]byte, 1024), "tag"); err != nil {
			t.Fatal(err)
		}
	}
	if closeFirst {
		if err := adapter.Close(ctx); err != nil {
			t.Fatal(err)
		}
	}
	return weak.Make(adapter)
}
