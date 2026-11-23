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
package fbs_test

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/fbs"
)

func TestPublicSchemaContract(t *testing.T) {
	b := []byte("table Item { value:int; }")
	want := sha256.Sum256(b)
	got := fbs.ComputeSchemaHash(b)
	payload := append([]byte(nil), want[:]...)
	payload = append(payload, 0, 1)
	require.True(t, got == fbs.SchemaHash(want) && fbs.ValidateHash(got, payload), "PIKO_CHECK schema-identity: valid prefix rejected")
	payload[3] ^= 1
	require.False(t, fbs.ValidateHash(got, payload) || fbs.ValidateHash(got, payload[:3]), "PIKO_CHECK changed-prefix: accepted corruption")
}
