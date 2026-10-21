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

package capabilities_functions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/image/image_dto"
)

const (
	transformParamsDeterminismRuns = 32
)

func TestParseTransformParams_AliasedKeysResolveTheSameWay(t *testing.T) {
	t.Parallel()

	params := capabilities_domain.CapabilityParams{
		"background":          "blue",
		"bg":                  "red",
		"Width":               "100",
		"width":               "200",
		"aspectratio":         "4:3",
		"aspect_ratio":        "16:9",
		"withoutenlargement":  "true",
		"without_enlargement": "false",
	}

	for range transformParamsDeterminismRuns {
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}

		parseTransformParams(params, &spec)

		require.Equal(t, "red", spec.Background, "the same alias must win on every run")
		require.Equal(t, 200, spec.Width)
		require.Equal(t, "4:3", spec.AspectRatio)
		require.True(t, spec.WithoutEnlargement)
	}
}
