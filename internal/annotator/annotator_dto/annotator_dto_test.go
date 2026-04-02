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

package annotator_dto

import "testing"

func TestActionManifest_AddAndGet(t *testing.T) {
	m := NewActionManifest()
	m.AddAction(ActionDefinition{Name: "email.contact", StructName: "ContactAction"})
	m.AddAction(ActionDefinition{Name: "user.create", StructName: "CreateAction"})

	if len(m.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(m.Actions))
	}

	got := m.GetAction("email.contact")
	if got == nil || got.StructName != "ContactAction" {
		t.Errorf("GetAction(email.contact) = %v", got)
	}

	if m.GetAction("nonexistent") != nil {
		t.Error("GetAction(nonexistent) should return nil")
	}
}

func TestActionDefinition_Method(t *testing.T) {
	a := &ActionDefinition{}
	if got := a.Method(); got != "POST" {
		t.Errorf("Method() default = %q, want POST", got)
	}

	a.HTTPMethod = "GET"
	if got := a.Method(); got != "GET" {
		t.Errorf("Method() = %q, want GET", got)
	}
}

func TestActionSpec_Method(t *testing.T) {
	s := &ActionSpec{}
	if got := s.Method(); got != "POST" {
		t.Errorf("Method() default = %q, want POST", got)
	}

	s.HTTPMethod = "DELETE"
	if got := s.Method(); got != "DELETE" {
		t.Errorf("Method() = %q, want DELETE", got)
	}
}

func TestActionSpec_Getters(t *testing.T) {
	t.Parallel()

	params := []ParamSpec{
		{Name: "email", GoType: "string"},
		{Name: "name", GoType: "string"},
	}
	retType := &TypeSpec{Name: "UserResponse"}

	spec := &ActionSpec{
		Name:        "user.create",
		CallParams:  params,
		ReturnType:  retType,
		Description: "Creates a new user account.",
	}

	t.Run("GetName", func(t *testing.T) {
		t.Parallel()
		if got := spec.GetName(); got != "user.create" {
			t.Errorf("GetName() = %q, want user.create", got)
		}
	})

	t.Run("GetCallParams", func(t *testing.T) {
		t.Parallel()
		got := spec.GetCallParams()
		if len(got) != 2 {
			t.Fatalf("GetCallParams() length = %d, want 2", len(got))
		}
		if got[0].Name != "email" || got[1].Name != "name" {
			t.Errorf("GetCallParams() = %v", got)
		}
	})

	t.Run("GetReturnType", func(t *testing.T) {
		t.Parallel()
		got := spec.GetReturnType()
		if got == nil || got.Name != "UserResponse" {
			t.Errorf("GetReturnType() = %v, want UserResponse", got)
		}
	})

	t.Run("GetDescription", func(t *testing.T) {
		t.Parallel()
		if got := spec.GetDescription(); got != "Creates a new user account." {
			t.Errorf("GetDescription() = %q", got)
		}
	})
}
