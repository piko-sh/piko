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

package encoder_gob

import (
	"reflect"
	"testing"

	"piko.sh/piko/internal/cache/cache_domain"
)

type testPerson struct {
	Name string
	Age  int
}

type testAddress struct {
	Street string
	City   string
}

type testPersonWithAddress struct {
	Address testAddress
	Person  testPerson
}

func TestGobEncoder_RoundTrip_String(t *testing.T) {
	enc := New[string]()
	original := "hello world"

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Marshal produced empty output")
	}

	var result string
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %q, want %q", result, original)
	}
}

func TestGobEncoder_RoundTrip_Int(t *testing.T) {
	enc := New[int]()
	original := 42

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result int
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %d, want %d", result, original)
	}
}

func TestGobEncoder_RoundTrip_Struct(t *testing.T) {
	enc := New[testPerson]()
	original := testPerson{Name: "Alice", Age: 30}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result testPerson
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %+v, want %+v", result, original)
	}
}

func TestGobEncoder_RoundTrip_NestedStruct(t *testing.T) {
	enc := New[testPersonWithAddress]()
	original := testPersonWithAddress{
		Person:  testPerson{Name: "Bob", Age: 25},
		Address: testAddress{Street: "123 Main St", City: "London"},
	}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result testPersonWithAddress
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %+v, want %+v", result, original)
	}
}

func TestGobEncoder_RoundTrip_Slice(t *testing.T) {
	enc := New[[]testPerson]()
	original := []testPerson{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result []testPerson
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result) != len(original) {
		t.Fatalf("got %d items, want %d", len(result), len(original))
	}
	for i := range original {
		if result[i] != original[i] {
			t.Errorf("index %d: got %+v, want %+v", i, result[i], original[i])
		}
	}
}

func TestGobEncoder_RoundTrip_Map(t *testing.T) {
	enc := New[map[string]int]()
	original := map[string]int{"one": 1, "two": 2, "three": 3}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]int
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result) != len(original) {
		t.Fatalf("got %d entries, want %d", len(result), len(original))
	}
	for k, v := range original {
		if result[k] != v {
			t.Errorf("key %q: got %d, want %d", k, result[k], v)
		}
	}
}

func TestGobEncoder_RoundTrip_ZeroValue(t *testing.T) {
	enc := New[testPerson]()
	original := testPerson{}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result testPerson
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %+v, want %+v", result, original)
	}
}

func TestGobEncoder_Unmarshal_CorruptData(t *testing.T) {
	enc := New[testPerson]()

	var result testPerson
	err := enc.Unmarshal([]byte{0xDE, 0xAD, 0xBE, 0xEF}, &result)
	if err == nil {
		t.Error("expected error for corrupt data, got nil")
	}
}

func TestGobEncoder_Unmarshal_EmptyData(t *testing.T) {
	enc := New[testPerson]()

	var result testPerson
	err := enc.Unmarshal([]byte{}, &result)
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestGobEncoder_AnyEncoder_MarshalAny(t *testing.T) {
	enc := New[testPerson]()
	anyEnc, ok := enc.(cache_domain.AnyEncoder)
	if !ok {
		t.Fatal("encoder does not implement AnyEncoder")
	}

	original := testPerson{Name: "Charlie", Age: 35}
	data, err := anyEnc.MarshalAny(original)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("MarshalAny produced empty output")
	}
}

func TestGobEncoder_AnyEncoder_MarshalAny_WrongType(t *testing.T) {
	enc := New[testPerson]()
	anyEnc, ok := enc.(cache_domain.AnyEncoder)
	if !ok {
		t.Fatal("expected encoder to implement AnyEncoder")
	}

	_, err := anyEnc.MarshalAny("not a person")
	if err == nil {
		t.Error("expected error for wrong type, got nil")
	}
}

func TestGobEncoder_AnyEncoder_UnmarshalAny(t *testing.T) {
	enc := New[testPerson]()
	anyEnc, ok := enc.(cache_domain.AnyEncoder)
	if !ok {
		t.Fatal("expected encoder to implement AnyEncoder")
	}

	original := testPerson{Name: "Diana", Age: 28}
	data, err := anyEnc.MarshalAny(original)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}

	result, err := anyEnc.UnmarshalAny(data)
	if err != nil {
		t.Fatalf("UnmarshalAny failed: %v", err)
	}

	person, ok := result.(testPerson)
	if !ok {
		t.Fatalf("expected testPerson, got %T", result)
	}
	if person != original {
		t.Errorf("got %+v, want %+v", person, original)
	}
}

func TestGobEncoder_AnyEncoder_HandlesType(t *testing.T) {
	enc := New[testPerson]()
	anyEnc, ok := enc.(cache_domain.AnyEncoder)
	if !ok {
		t.Fatal("expected encoder to implement AnyEncoder")
	}

	expected := reflect.TypeFor[testPerson]()
	if anyEnc.HandlesType() != expected {
		t.Errorf("got %v, want %v", anyEnc.HandlesType(), expected)
	}
}
