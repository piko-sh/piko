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

package daemon_adapters

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func TestSelectVariantByID_PrefersInstanceRelease(t *testing.T) {
	now := time.Now()
	variants := []registry_dto.Variant{
		{VariantID: "source", BuildRelease: "relA", StorageKey: "build/relA/source", CreatedAt: now.Add(-time.Hour)},
		{VariantID: "source", BuildRelease: "relB", StorageKey: "build/relB/source", CreatedAt: now},
	}

	got := selectVariantByID(variants, "source", "relA")
	require.NotNil(t, got)
	require.Equal(t, "relA", got.BuildRelease)
}

func TestSelectVariantByID_FallsBackToNewest(t *testing.T) {
	now := time.Now()
	variants := []registry_dto.Variant{
		{VariantID: "source", BuildRelease: "relA", CreatedAt: now.Add(-time.Hour)},
		{VariantID: "source", BuildRelease: "relB", CreatedAt: now},
	}

	require.Equal(t, "relB", selectVariantByID(variants, "source", "").BuildRelease)

	require.Equal(t, "relB", findVariantByID(variants, "source").BuildRelease)
}

func TestSelectVariantByID_NoMatchReturnsNil(t *testing.T) {
	require.Nil(t, selectVariantByID(nil, "missing", "relA"))
}
