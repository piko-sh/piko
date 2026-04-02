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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestPropagateDataAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		catalogue  *querier_dto.Catalogue
		assertions func(t *testing.T, catalogue *querier_dto.Catalogue)
	}{
		{
			name: "no functions results in no changes",
			catalogue: &querier_dto.Catalogue{
				Schemas: map[string]*querier_dto.Schema{
					"public": {
						Name:      "public",
						Functions: make(map[string][]*querier_dto.FunctionSignature),
					},
				},
			},
			assertions: func(t *testing.T, catalogue *querier_dto.Catalogue) {
				t.Helper()

				assert.Empty(t, catalogue.Schemas["public"].Functions)
			},
		},
		{
			name: "read-only calling read-only stays read-only",
			catalogue: &querier_dto.Catalogue{
				Schemas: map[string]*querier_dto.Schema{
					"public": {
						Name: "public",
						Functions: map[string][]*querier_dto.FunctionSignature{
							"caller": {
								{
									Name:            "caller",
									DataAccess:      querier_dto.DataAccessReadOnly,
									CalledFunctions: []string{"callee"},
								},
							},
							"callee": {
								{
									Name:       "callee",
									DataAccess: querier_dto.DataAccessReadOnly,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, catalogue *querier_dto.Catalogue) {
				t.Helper()

				caller := catalogue.Schemas["public"].Functions["caller"][0]
				callee := catalogue.Schemas["public"].Functions["callee"][0]
				assert.Equal(t, querier_dto.DataAccessReadOnly, caller.DataAccess)
				assert.Equal(t, querier_dto.DataAccessReadOnly, callee.DataAccess)
			},
		},
		{
			name: "read-only calling data-modifying becomes data-modifying",
			catalogue: &querier_dto.Catalogue{
				Schemas: map[string]*querier_dto.Schema{
					"public": {
						Name: "public",
						Functions: map[string][]*querier_dto.FunctionSignature{
							"caller": {
								{
									Name:            "caller",
									DataAccess:      querier_dto.DataAccessReadOnly,
									CalledFunctions: []string{"writer"},
								},
							},
							"writer": {
								{
									Name:       "writer",
									DataAccess: querier_dto.DataAccessModifiesData,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, catalogue *querier_dto.Catalogue) {
				t.Helper()

				caller := catalogue.Schemas["public"].Functions["caller"][0]
				assert.Equal(t, querier_dto.DataAccessModifiesData, caller.DataAccess)
			},
		},
		{
			name: "transitive propagation through call chain",
			catalogue: &querier_dto.Catalogue{
				Schemas: map[string]*querier_dto.Schema{
					"public": {
						Name: "public",
						Functions: map[string][]*querier_dto.FunctionSignature{
							"a": {
								{
									Name:            "a",
									DataAccess:      querier_dto.DataAccessReadOnly,
									CalledFunctions: []string{"b"},
								},
							},
							"b": {
								{
									Name:            "b",
									DataAccess:      querier_dto.DataAccessReadOnly,
									CalledFunctions: []string{"c"},
								},
							},
							"c": {
								{
									Name:       "c",
									DataAccess: querier_dto.DataAccessModifiesData,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, catalogue *querier_dto.Catalogue) {
				t.Helper()

				a := catalogue.Schemas["public"].Functions["a"][0]
				b := catalogue.Schemas["public"].Functions["b"][0]
				c := catalogue.Schemas["public"].Functions["c"][0]
				assert.Equal(t, querier_dto.DataAccessModifiesData, a.DataAccess)
				assert.Equal(t, querier_dto.DataAccessModifiesData, b.DataAccess)
				assert.Equal(t, querier_dto.DataAccessModifiesData, c.DataAccess)
			},
		},
		{
			name: "already data-modifying stays unchanged",
			catalogue: &querier_dto.Catalogue{
				Schemas: map[string]*querier_dto.Schema{
					"public": {
						Name: "public",
						Functions: map[string][]*querier_dto.FunctionSignature{
							"writer": {
								{
									Name:            "writer",
									DataAccess:      querier_dto.DataAccessModifiesData,
									CalledFunctions: []string{"reader"},
								},
							},
							"reader": {
								{
									Name:       "reader",
									DataAccess: querier_dto.DataAccessReadOnly,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, catalogue *querier_dto.Catalogue) {
				t.Helper()

				writer := catalogue.Schemas["public"].Functions["writer"][0]
				assert.Equal(t, querier_dto.DataAccessModifiesData, writer.DataAccess)
			},
		},
		{
			name: "fixed-point convergence with circular calls",
			catalogue: &querier_dto.Catalogue{
				Schemas: map[string]*querier_dto.Schema{
					"public": {
						Name: "public",
						Functions: map[string][]*querier_dto.FunctionSignature{
							"alpha": {
								{
									Name:            "alpha",
									DataAccess:      querier_dto.DataAccessReadOnly,
									CalledFunctions: []string{"beta"},
								},
							},
							"beta": {
								{
									Name:            "beta",
									DataAccess:      querier_dto.DataAccessReadOnly,
									CalledFunctions: []string{"alpha"},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, catalogue *querier_dto.Catalogue) {
				t.Helper()

				alpha := catalogue.Schemas["public"].Functions["alpha"][0]
				beta := catalogue.Schemas["public"].Functions["beta"][0]
				assert.Equal(t, querier_dto.DataAccessReadOnly, alpha.DataAccess)
				assert.Equal(t, querier_dto.DataAccessReadOnly, beta.DataAccess)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			propagateDataAccess(tt.catalogue)
			tt.assertions(t, tt.catalogue)
		})
	}
}
