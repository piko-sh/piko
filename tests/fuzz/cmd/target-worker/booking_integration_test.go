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

//go:build integration

package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

func TestSeededReservationWorkers(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	var seen [4]bool
	workers := t.TempDir()
	for seed := range uint64(128) {
		savedCase := generate.New("binder-safe", seed, 0)
		family := savedCase.Data[0] % 4
		if seen[family] {
			continue
		}
		require.NoError(t, targets.RunWithWorkers(t.Context(), repository, t.TempDir(), workers, &savedCase), "family %d seed %d", family, seed)
		seen[family] = true
	}
	require.Equal(t, [4]bool{true, true, true, true}, seen, "seeded campaign omitted family")
}
