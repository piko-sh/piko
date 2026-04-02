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

package encoder_json

import (
	"encoding/json"
	"reflect"
	"testing"

	"piko.sh/piko/internal/cache/cache_domain"
)

type testUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type testOrder struct {
	ID    string   `json:"id"`
	User  testUser `json:"user"`
	Items []string `json:"items"`
}

func TestJSONEncoder_RoundTrip_String(t *testing.T) {
	enc := New[string]()
	original := "hello world"

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !json.Valid(data) {
		t.Error("Marshal output is not valid JSON")
	}

	var result string
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %q, want %q", result, original)
	}
}

func TestJSONEncoder_RoundTrip_Int(t *testing.T) {
	enc := New[int]()
	original := 42

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !json.Valid(data) {
		t.Error("Marshal output is not valid JSON")
	}

	var result int
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %d, want %d", result, original)
	}
}

func TestJSONEncoder_RoundTrip_Float64(t *testing.T) {
	enc := New[float64]()
	original := 3.14159

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result float64
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %f, want %f", result, original)
	}
}

func TestJSONEncoder_RoundTrip_Bool(t *testing.T) {
	enc := New[bool]()

	for _, original := range []bool{true, false} {
		data, err := enc.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal(%v) failed: %v", original, err)
		}

		var result bool
		if err := enc.Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if result != original {
			t.Errorf("got %v, want %v", result, original)
		}
	}
}

func TestJSONEncoder_RoundTrip_Struct(t *testing.T) {
	enc := New[testUser]()
	original := testUser{Name: "Alice", Email: "alice@example.com", Age: 30}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !json.Valid(data) {
		t.Error("Marshal output is not valid JSON")
	}

	var result testUser
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %+v, want %+v", result, original)
	}
}

func TestJSONEncoder_RoundTrip_NestedStruct(t *testing.T) {
	enc := New[testOrder]()
	original := testOrder{
		ID:    "ord-001",
		User:  testUser{Name: "Bob", Email: "bob@example.com", Age: 25},
		Items: []string{"item-a", "item-b"},
	}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result testOrder
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result.ID != original.ID || result.User != original.User {
		t.Errorf("got %+v, want %+v", result, original)
	}
	if len(result.Items) != len(original.Items) {
		t.Fatalf("items length: got %d, want %d", len(result.Items), len(original.Items))
	}
	for i := range original.Items {
		if result.Items[i] != original.Items[i] {
			t.Errorf("items[%d]: got %q, want %q", i, result.Items[i], original.Items[i])
		}
	}
}

func TestJSONEncoder_RoundTrip_Slice(t *testing.T) {
	enc := New[[]int]()
	original := []int{1, 2, 3, 4, 5}

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result []int
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result) != len(original) {
		t.Fatalf("got %d items, want %d", len(result), len(original))
	}
	for i := range original {
		if result[i] != original[i] {
			t.Errorf("index %d: got %d, want %d", i, result[i], original[i])
		}
	}
}

func TestJSONEncoder_RoundTrip_Map(t *testing.T) {
	enc := New[map[string]int]()
	original := map[string]int{"a": 1, "b": 2, "c": 3}

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

func TestJSONEncoder_Unmarshal_InvalidJSON(t *testing.T) {
	enc := New[testUser]()

	var result testUser
	err := enc.Unmarshal([]byte("{invalid json"), &result)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestJSONEncoder_Unmarshal_EmptyData(t *testing.T) {
	enc := New[testUser]()

	var result testUser
	err := enc.Unmarshal([]byte{}, &result)
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestJSONEncoder_Unicode(t *testing.T) {
	enc := New[string]()
	original := "Hello \u4e16\u754c \U0001f600"

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result string
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %q, want %q", result, original)
	}
}

func TestJSONEncoder_HTMLNotEscaped(t *testing.T) {
	enc := New[string]()
	original := "<script>alert('xss')</script>"

	data, err := enc.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	dataString := string(data)
	if !json.Valid(data) {
		t.Error("output is not valid JSON")
	}

	if contains(dataString, `\u003c`) || contains(dataString, `\u003e`) {
		t.Error("HTML characters were escaped; expected EscapeHTML=false behaviour")
	}

	var result string
	if err := enc.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != original {
		t.Errorf("got %q, want %q", result, original)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestJSONEncoder_AnyEncoder_RoundTrip(t *testing.T) {
	enc := New[testUser]()
	anyEnc, ok := enc.(cache_domain.AnyEncoder)
	if !ok {
		t.Fatal("encoder does not implement AnyEncoder")
	}

	original := testUser{Name: "Eve", Email: "eve@example.com", Age: 22}
	data, err := anyEnc.MarshalAny(original)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}

	result, err := anyEnc.UnmarshalAny(data)
	if err != nil {
		t.Fatalf("UnmarshalAny failed: %v", err)
	}

	user, ok := result.(testUser)
	if !ok {
		t.Fatalf("expected testUser, got %T", result)
	}
	if user != original {
		t.Errorf("got %+v, want %+v", user, original)
	}
}

func TestJSONEncoder_AnyEncoder_MarshalAny_WrongType(t *testing.T) {
	enc := New[testUser]()
	anyEnc, ok := enc.(cache_domain.AnyEncoder)
	if !ok {
		t.Fatal("expected encoder to implement AnyEncoder")
	}

	_, err := anyEnc.MarshalAny(42)
	if err == nil {
		t.Error("expected error for wrong type, got nil")
	}
}

func TestJSONEncoder_AnyEncoder_HandlesType(t *testing.T) {
	enc := New[testUser]()
	anyEnc, ok := enc.(cache_domain.AnyEncoder)
	if !ok {
		t.Fatal("expected encoder to implement AnyEncoder")
	}

	expected := reflect.TypeFor[testUser]()
	if anyEnc.HandlesType() != expected {
		t.Errorf("got %v, want %v", anyEnc.HandlesType(), expected)
	}
}
