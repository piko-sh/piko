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

package db_catalogue_postgres

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
)

func TestBuildTypeModifiers_TemporalPrecision(t *testing.T) {
	row := columnRow{
		datetimePrecision: sql.NullInt64{Int64: 3, Valid: true},
	}

	modifiers := buildTypeModifiers(row)

	if want := []int{3}; !slices.Equal(modifiers, want) {
		t.Fatalf("buildTypeModifiers = %v, want %v", modifiers, want)
	}
}

func TestBuildTypeModifiers_TextLength(t *testing.T) {
	row := columnRow{
		characterMaximumLength: sql.NullInt64{Int64: 255, Valid: true},
	}

	modifiers := buildTypeModifiers(row)

	if want := []int{255}; !slices.Equal(modifiers, want) {
		t.Fatalf("buildTypeModifiers = %v, want %v", modifiers, want)
	}
}

func TestBuildTypeModifiers_NumericPrecisionScale(t *testing.T) {
	row := columnRow{
		numericPrecision: sql.NullInt64{Int64: 10, Valid: true},
		numericScale:     sql.NullInt64{Int64: 2, Valid: true},
	}

	modifiers := buildTypeModifiers(row)

	if want := []int{10, 2}; !slices.Equal(modifiers, want) {
		t.Fatalf("buildTypeModifiers = %v, want %v", modifiers, want)
	}
}

func TestBuildCatalogue_NotConfigured(t *testing.T) {
	provider := NewPgIntrospectionProvider(nil, nil)

	catalogue, sourceErrors, buildError := provider.BuildCatalogue(context.Background())

	if catalogue != nil {
		t.Fatalf("BuildCatalogue catalogue = %v, want nil", catalogue)
	}
	if sourceErrors != nil {
		t.Fatalf("BuildCatalogue sourceErrors = %v, want nil", sourceErrors)
	}
	if !errors.Is(buildError, ErrProviderNotConfigured) {
		t.Fatalf("BuildCatalogue error = %v, want ErrProviderNotConfigured", buildError)
	}
}

func TestAssembleIndexResults_SkipsEmptyColumns(t *testing.T) {
	indexMap := map[string]*indexEntry{
		"empty_expression_index": {},
		"name_index":             {columns: []string{"name"}},
	}
	indexOrder := []string{"empty_expression_index", "name_index"}

	indexes := assembleIndexResults(indexMap, indexOrder)

	if len(indexes) != 1 {
		t.Fatalf("assembleIndexResults length = %d, want 1", len(indexes))
	}
	if indexes[0].Name != "name_index" {
		t.Fatalf("assembleIndexResults[0].Name = %q, want %q", indexes[0].Name, "name_index")
	}
}
