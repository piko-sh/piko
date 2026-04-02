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

//go:build fuzz

package binder

import (
	"context"
	"reflect"
	"testing"
	"time"

	"piko.sh/piko/wdk/maths"
)

func FuzzBind(f *testing.F) {

	f.Add("Name", "Alice")
	f.Add("Age", "30")
	f.Add("IsActive", "true")
	f.Add("Items[0].Name", "Apple")
	f.Add("User.Email", "test@example.com")
	f.Add("", "")
	f.Add("InvalidField", "value")
	f.Add("Items[999999]", "overflow")
	f.Add("Items[-1]", "negative")
	f.Add("User..Name", "double-dot")

	type TestStruct struct {
		Score *float64
		Name  string
		Items []struct {
			Name  string
			Price float64
		}
		User struct {
			Email string
			ID    int
		}
		Age      int
		IsActive bool
	}

	f.Fuzz(func(t *testing.T, key string, value string) {
		binder := NewASTBinder()
		var target TestStruct

		src := map[string][]string{
			key: {value},
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Bind panicked with key=%q value=%q: %v", key, value, r)
			}
		}()

		_ = binder.Bind(context.Background(), &target, src)
	})
}

func FuzzConvertToType(f *testing.F) {

	f.Add("123", "int")
	f.Add("true", "bool")
	f.Add("3.14", "float64")
	f.Add("hello", "string")
	f.Add("", "int")
	f.Add("999999999999999999999999", "int")
	f.Add("-999999999999999999999999", "int")
	f.Add("not-a-number", "int")
	f.Add("maybe", "bool")
	f.Add("1.7976931348623157e+309", "float64")
	f.Add("2025-10-09T10:00:00Z", "time")

	f.Fuzz(func(t *testing.T, value string, typeHint string) {
		binder := NewASTBinder()

		var targetType reflect.Type
		switch typeHint {
		case "int":
			targetType = reflect.TypeFor[int]()
		case "int8":
			targetType = reflect.TypeFor[int8]()
		case "int16":
			targetType = reflect.TypeFor[int16]()
		case "int32":
			targetType = reflect.TypeFor[int32]()
		case "int64":
			targetType = reflect.TypeFor[int64]()
		case "uint":
			targetType = reflect.TypeFor[uint]()
		case "uint8":
			targetType = reflect.TypeFor[uint8]()
		case "uint16":
			targetType = reflect.TypeFor[uint16]()
		case "uint32":
			targetType = reflect.TypeFor[uint32]()
		case "uint64":
			targetType = reflect.TypeFor[uint64]()
		case "float32":
			targetType = reflect.TypeFor[float32]()
		case "float64":
			targetType = reflect.TypeFor[float64]()
		case "bool":
			targetType = reflect.TypeFor[bool]()
		case "string":
			targetType = reflect.TypeFor[string]()
		case "time":
			targetType = reflect.TypeFor[time.Time]()
		case "decimal":
			targetType = reflect.TypeFor[maths.Decimal]()
		case "money":
			targetType = reflect.TypeFor[maths.Money]()
		default:

			return
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("convertToType panicked with value=%q type=%s: %v", value, typeHint, r)
			}
		}()

		fi := makeFieldInfo(typeHint, targetType)
		_, _ = binder.convertToType(value, fi)
	})
}

func FuzzSliceGrowth(f *testing.F) {

	f.Add(0)
	f.Add(1)
	f.Add(10)
	f.Add(100)
	f.Add(1000)
	f.Add(-1)
	f.Add(-100)

	f.Fuzz(func(t *testing.T, index int) {

		if index > 100000 || index < -1 {
			return
		}

		binder := NewASTBinder()

		binder.SetMaxSliceSize(100000)

		initialLen := 5
		slice := make([]string, initialLen)
		sliceVal := reflect.ValueOf(&slice).Elem()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("growSliceToFitIndex panicked with index=%d: %v", index, r)
			}
		}()

		if index < 0 {

			return
		}

		err := growSliceToFitIndex(sliceVal, index, int(binder.maxSliceSize.Load()))

		if err == nil && index >= initialLen {
			if len(slice) != index+1 {
				t.Errorf("Expected slice length %d, got %d", index+1, len(slice))
			}
		} else if err == nil && index < initialLen {

			if len(slice) != initialLen {
				t.Errorf("Expected slice length to remain %d, got %d", initialLen, len(slice))
			}
		}

	})
}

func FuzzMultipleFields(f *testing.F) {

	f.Add("Name", "Alice", "Age", "30")
	f.Add("Items[0]", "item1", "Items[1]", "item2")
	f.Add("User.Email", "test@example.com", "User.ID", "123")
	f.Add("", "", "", "")

	type ComplexStruct struct {
		Name  string
		Email string
		Items []string
		User  struct {
			Email string
			ID    int
		}
		Nested struct {
			Field1 string
			Field2 int
		}
		Age int
	}

	f.Fuzz(func(t *testing.T, key1, val1, key2, val2 string) {
		binder := NewASTBinder()
		var target ComplexStruct

		src := map[string][]string{}
		if key1 != "" {
			src[key1] = []string{val1}
		}
		if key2 != "" {
			src[key2] = []string{val2}
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Bind panicked with fields [%q=%q, %q=%q]: %v", key1, val1, key2, val2, r)
			}
		}()

		_ = binder.Bind(context.Background(), &target, src)
	})
}

func FuzzPathComplexity(f *testing.F) {

	f.Add("simple")
	f.Add("nested.field")
	f.Add("array[0]")
	f.Add("deep.nested.field.path")
	f.Add("array[0].field")
	f.Add("array[999].deep.field")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("field.")
	f.Add(".field")
	f.Add("field..nested")
	f.Add("array[]")
	f.Add("array[")
	f.Add("array]")
	f.Add("array[-1]")
	f.Add("array[a]")
	f.Add("field[0][1]")
	f.Add("🚀.field")
	f.Add("field.🎯")

	type PathTestStruct struct {
		Simple string
		Nested struct {
			Field string
			Path  struct {
				Deep string
			}
		}
		Array []struct {
			Field string
			Deep  struct {
				Field string
			}
		}
	}

	f.Fuzz(func(t *testing.T, path string) {
		binder := NewASTBinder()
		var target PathTestStruct

		src := map[string][]string{
			path: {"test-value"},
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Bind panicked with path=%q: %v", path, r)
			}
		}()

		_ = binder.Bind(context.Background(), &target, src)
	})
}

func FuzzPointerFields(f *testing.F) {

	f.Add("PtrString", "value")
	f.Add("PtrInt", "42")
	f.Add("PtrBool", "true")
	f.Add("PtrFloat", "3.14")
	f.Add("PtrStruct.Field", "nested")

	type PointerStruct struct {
		PtrString *string
		PtrInt    *int
		PtrBool   *bool
		PtrFloat  *float64
		PtrStruct *struct {
			Field string
		}
	}

	f.Fuzz(func(t *testing.T, key, value string) {
		binder := NewASTBinder()
		var target PointerStruct

		src := map[string][]string{
			key: {value},
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Bind panicked with key=%q value=%q: %v", key, value, r)
			}
		}()

		_ = binder.Bind(context.Background(), &target, src)
	})
}
