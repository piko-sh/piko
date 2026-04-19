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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/ast/ast_schema"
	"piko.sh/piko/internal/collection/collection_schema"
	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/generator/generator_schema"
	"piko.sh/piko/internal/i18n/i18n_schema"
	"piko.sh/piko/internal/inspector/inspector_schema"
	"piko.sh/piko/internal/interp/interp_schema"
	"piko.sh/piko/internal/registry/registry_schema"
	"piko.sh/piko/internal/search/search_schema"
	"piko.sh/piko/internal/typegen/typegen_schema"
)

func TestSchemaHashesAreNonEmpty(t *testing.T) {
	t.Parallel()

	emptyHash := fbs.ComputeSchemaHash(nil)

	cases := []struct {
		name string
		hash fbs.SchemaHash
	}{
		{"ast", ast_schema.SchemaHash},
		{"collection", collection_schema.SchemaHash},
		{"generator", generator_schema.SchemaHash},
		{"i18n", i18n_schema.SchemaHash},
		{"inspector", inspector_schema.SchemaHash},
		{"interp", interp_schema.SchemaHash},
		{"registry", registry_schema.SchemaHash},
		{"search", search_schema.SchemaHash},
		{"typegen", typegen_schema.SchemaHash},
	}

	seen := make(map[fbs.SchemaHash]string, len(cases))
	for _, tc := range cases {
		assert.NotEqualf(t, emptyHash, tc.hash, "%s_schema.SchemaHash is the hash of empty content; the .fbs is not embedded (missing //go:embed)", tc.name)
		if other, dup := seen[tc.hash]; dup {
			assert.Failf(t, "schema hash collision", "%s_schema.SchemaHash collides with %s_schema; both schemas hash identically, so version gating cannot tell them apart", tc.name, other)
		}
		seen[tc.hash] = tc.name
	}
}
