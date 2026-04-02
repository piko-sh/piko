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

package cache_dto

import (
	"context"
	"errors"
	"testing"
)

func TestFieldConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metric   string
		field    FieldSchema
		wantType FieldType
		weight   float64
		dim      int
		sortable bool
	}{
		{
			name:     "TextField",
			field:    TextField("title"),
			wantType: FieldTypeText,
			weight:   1.0,
		},
		{
			name:     "TagField",
			field:    TagField("status"),
			wantType: FieldTypeTag,
		},
		{
			name:     "NumericField",
			field:    NumericField("price"),
			wantType: FieldTypeNumeric,
		},
		{
			name:     "SortableNumericField",
			field:    SortableNumericField("score"),
			wantType: FieldTypeNumeric,
			sortable: true,
		},
		{
			name:     "SortableTextField",
			field:    SortableTextField("name"),
			wantType: FieldTypeText,
			sortable: true,
			weight:   1.0,
		},
		{
			name:     "GeoField",
			field:    GeoField("location"),
			wantType: FieldTypeGeo,
		},
		{
			name:     "VectorField",
			field:    VectorField("embedding", 128),
			wantType: FieldTypeVector,
			dim:      128,
			metric:   "cosine",
		},
		{
			name:     "VectorFieldWithMetric",
			field:    VectorFieldWithMetric("embedding", 256, "euclidean"),
			wantType: FieldTypeVector,
			dim:      256,
			metric:   "euclidean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.field.Type != tt.wantType {
				t.Errorf("Type = %d, want %d", tt.field.Type, tt.wantType)
			}
			if tt.field.Sortable != tt.sortable {
				t.Errorf("Sortable = %v, want %v", tt.field.Sortable, tt.sortable)
			}
			if tt.field.Weight != tt.weight {
				t.Errorf("Weight = %f, want %f", tt.field.Weight, tt.weight)
			}
			if tt.field.Dimension != tt.dim {
				t.Errorf("Dimension = %d, want %d", tt.field.Dimension, tt.dim)
			}
			if tt.field.DistanceMetric != tt.metric {
				t.Errorf("DistanceMetric = %q, want %q", tt.field.DistanceMetric, tt.metric)
			}
		})
	}
}

func TestNewSearchSchema(t *testing.T) {
	t.Parallel()

	schema := NewSearchSchema(TextField("title"), TagField("status"))

	if len(schema.Fields) != 2 {
		t.Fatalf("Fields length = %d, want 2", len(schema.Fields))
	}
	if schema.Language != "english" {
		t.Errorf("Language = %q, want english", schema.Language)
	}
	if schema.TextAnalyser != nil {
		t.Error("TextAnalyser should be nil")
	}
}

func TestNewSearchSchemaWithAnalyser(t *testing.T) {
	t.Parallel()

	analyser := func(text string) []string { return []string{text} }
	schema := NewSearchSchemaWithAnalyser(analyser, TextField("body"))

	if len(schema.Fields) != 1 {
		t.Fatalf("Fields length = %d, want 1", len(schema.Fields))
	}
	if schema.Language != "english" {
		t.Errorf("Language = %q, want english", schema.Language)
	}
	if schema.TextAnalyser == nil {
		t.Error("TextAnalyser should not be nil")
	}

	result := schema.TextAnalyser("hello")
	if len(result) != 1 || result[0] != "hello" {
		t.Errorf("TextAnalyser(hello) = %v, want [hello]", result)
	}
}

func TestFilterConstructors(t *testing.T) {
	t.Parallel()

	t.Run("Eq", func(t *testing.T) {
		t.Parallel()
		f := Eq("status", "active")
		if f.Field != "status" || f.Operation != FilterOpEq || f.Value != "active" {
			t.Errorf("Eq = %+v", f)
		}
	})

	t.Run("Ne", func(t *testing.T) {
		t.Parallel()
		f := Ne("status", "deleted")
		if f.Field != "status" || f.Operation != FilterOpNe || f.Value != "deleted" {
			t.Errorf("Ne = %+v", f)
		}
	})

	t.Run("Gt", func(t *testing.T) {
		t.Parallel()
		f := Gt("price", 100)
		if f.Field != "price" || f.Operation != FilterOpGt || f.Value != 100 {
			t.Errorf("Gt = %+v", f)
		}
	})

	t.Run("Ge", func(t *testing.T) {
		t.Parallel()
		f := Ge("price", 50)
		if f.Field != "price" || f.Operation != FilterOpGe || f.Value != 50 {
			t.Errorf("Ge = %+v", f)
		}
	})

	t.Run("Lt", func(t *testing.T) {
		t.Parallel()
		f := Lt("age", 18)
		if f.Field != "age" || f.Operation != FilterOpLt || f.Value != 18 {
			t.Errorf("Lt = %+v", f)
		}
	})

	t.Run("Le", func(t *testing.T) {
		t.Parallel()
		f := Le("score", 99)
		if f.Field != "score" || f.Operation != FilterOpLe || f.Value != 99 {
			t.Errorf("Le = %+v", f)
		}
	})

	t.Run("In", func(t *testing.T) {
		t.Parallel()
		f := In("colour", "red", "blue", "green")
		if f.Field != "colour" || f.Operation != FilterOpIn {
			t.Errorf("In = %+v", f)
		}
		if len(f.Values) != 3 {
			t.Errorf("In Values length = %d, want 3", len(f.Values))
		}
	})

	t.Run("Between", func(t *testing.T) {
		t.Parallel()
		f := Between("price", 10, 100)
		if f.Field != "price" || f.Operation != FilterOpBetween {
			t.Errorf("Between = %+v", f)
		}
		if len(f.Values) != 2 || f.Values[0] != 10 || f.Values[1] != 100 {
			t.Errorf("Between Values = %v, want [10, 100]", f.Values)
		}
	})

	t.Run("Prefix", func(t *testing.T) {
		t.Parallel()
		f := Prefix("tag", "prod")
		if f.Field != "tag" || f.Operation != FilterOpPrefix || f.Value != "prod" {
			t.Errorf("Prefix = %+v", f)
		}
	})
}

func TestLoaderFunc_Load(t *testing.T) {
	t.Parallel()

	lf := LoaderFunc[string, int](func(_ context.Context, key string) (int, error) {
		if key == "answer" {
			return 42, nil
		}
		return 0, ErrNotFound
	})

	value, err := lf.Load(context.Background(), "answer")
	if err != nil {
		t.Fatalf("Load(answer) error: %v", err)
	}
	if value != 42 {
		t.Errorf("Load(answer) = %d, want 42", value)
	}

	_, err = lf.Load(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load(missing) error = %v, want ErrNotFound", err)
	}
}

func TestLoaderFunc_Reload(t *testing.T) {
	t.Parallel()

	called := false
	lf := LoaderFunc[string, int](func(_ context.Context, key string) (int, error) {
		called = true
		return 99, nil
	})

	value, err := lf.Reload(context.Background(), "key", 0)
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}
	if value != 99 {
		t.Errorf("Reload = %d, want 99", value)
	}
	if !called {
		t.Error("Reload should call the underlying function")
	}
}

func TestBulkLoaderFunc_BulkLoad(t *testing.T) {
	t.Parallel()

	blf := BulkLoaderFunc[string, int](func(_ context.Context, keys []string) (map[string]int, error) {
		result := make(map[string]int, len(keys))
		for _, k := range keys {
			result[k] = len(k)
		}
		return result, nil
	})

	result, err := blf.BulkLoad(context.Background(), []string{"a", "bb", "ccc"})
	if err != nil {
		t.Fatalf("BulkLoad error: %v", err)
	}
	if result["a"] != 1 || result["bb"] != 2 || result["ccc"] != 3 {
		t.Errorf("BulkLoad = %v", result)
	}
}

func TestBulkLoaderFunc_BulkReload(t *testing.T) {
	t.Parallel()

	called := false
	blf := BulkLoaderFunc[string, int](func(_ context.Context, keys []string) (map[string]int, error) {
		called = true
		return map[string]int{"x": 10}, nil
	})

	result, err := blf.BulkReload(context.Background(), []string{"x"}, []int{5})
	if err != nil {
		t.Fatalf("BulkReload error: %v", err)
	}
	if result["x"] != 10 {
		t.Errorf("BulkReload[x] = %d, want 10", result["x"])
	}
	if !called {
		t.Error("BulkReload should call the underlying function")
	}
}

func TestDefaultTransformConfig(t *testing.T) {
	t.Parallel()

	config := DefaultTransformConfig()

	if config.EnabledTransformers == nil {
		t.Error("EnabledTransformers should not be nil")
	}
	if len(config.EnabledTransformers) != 0 {
		t.Errorf("EnabledTransformers length = %d, want 0", len(config.EnabledTransformers))
	}
	if config.TransformerOptions == nil {
		t.Error("TransformerOptions should not be nil")
	}
	if len(config.TransformerOptions) != 0 {
		t.Errorf("TransformerOptions length = %d, want 0", len(config.TransformerOptions))
	}
}
