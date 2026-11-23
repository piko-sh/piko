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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func FuzzCacheHistory(f *testing.F) { fuzzHistory(f, "cache") }

func FuzzSearchIndex(f *testing.F) { fuzzHistory(f, "search") }

func TestCorruptHistoryObservations(t *testing.T) {
	want := map[string]string{"a": "amber", "b": "amber"}
	for _, got := range []map[string]string{{"a": "amber"}, {"a": "amber", "b": "stale"}, {"a": "amber", "b": "amber", "c": "ghost"}} {
		assert.Error(t, compareHistoryRows("cache", 1, want, got), "accepted cache rows %v", got)
	}
	for _, got := range [][]string{{"a"}, {"a", "a"}, {"a", "b", "stale"}} {
		assert.Error(t, matchSearchRoutes(want, "amber", 1, got), "accepted search routes %v", got)
	}
	assert.NoError(t, compareHistoryRows("cache", 1, want, map[string]string{"a": "amber", "b": "amber"}))
	assert.NoError(t, matchSearchRoutes(want, "amber", 1, []string{"b", "a"}))
}

func TestCacheHistoryReopens(t *testing.T) {
	for _, data := range [][]byte{nil, {255, 11, 4, 33, 7, 5, 12, 2, 166}} {
		require.NoError(t, run(t.Context(), &casefile.Case{Target: "cache", Data: data}))
	}
}

func TestStringCodecRoundTrip(t *testing.T) {
	codec := stringCodec{}

	key, err := codec.EncodeKey("år")
	require.NoError(t, err)
	decodedKey, err := codec.DecodeKey(key)
	require.NoError(t, err)
	assert.Equal(t, "år", decodedKey)

	value, err := codec.EncodeValue("七")
	require.NoError(t, err)
	decodedValue, err := codec.DecodeValue(value)
	require.NoError(t, err)
	assert.Equal(t, "七", decodedValue)
}

func fuzzHistory(f *testing.F, target string) {
	f.Helper()
	f.Add([]byte{})
	f.Add([]byte{255, 11, 4, 33, 7, 5, 12, 2, 166})
	f.Fuzz(func(t *testing.T, data []byte) {
		require.NoError(t, run(t.Context(), &casefile.Case{Target: target, Data: data[:min(256, len(data))]}))
	})
}
