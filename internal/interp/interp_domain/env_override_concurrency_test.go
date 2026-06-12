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

package interp_domain

import (
	"context"
	"os"
	"reflect"
	"sync"
	"testing"
)

func TestEnvOverridesConcurrentClonesDoNotRace(t *testing.T) {
	t.Parallel()
	golden := NewService(WithEnv(map[string]string{"PIKO_RACE_PROBE": "1"}))
	golden.UseSymbols(NewSymbolRegistry(SymbolExports{
		"os": {"Getenv": reflect.ValueOf(os.Getenv)},
	}))

	const workers = 8
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			clone := golden.Clone()
			_, _ = clone.Eval(context.Background(), "1 + 1")
		}()
	}
	wg.Wait()
}
