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

package component_adapters

import (
	"sync"
	"testing"

	"piko.sh/piko/internal/component/component_dto"
)

func TestNewInMemoryRegistry(t *testing.T) {
	registry := NewInMemoryRegistry()

	if registry == nil {
		t.Fatal("NewInMemoryRegistry() returned nil")
	}

	if registry.Count() != 0 {
		t.Errorf("new registry Count() = %d, want 0", registry.Count())
	}
}

func TestInMemoryRegistry_Register(t *testing.T) {
	t.Run("valid registration", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		err := registry.Register(component_dto.ComponentDefinition{
			TagName:    "my-button",
			SourcePath: "components/my-button.pkc",
		})

		if err != nil {
			t.Errorf("Register() error = %v, want nil", err)
		}

		if !registry.IsRegistered("my-button") {
			t.Error("IsRegistered(\"my-button\") = false, want true")
		}
	})

	t.Run("case insensitive lookup", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		err := registry.Register(component_dto.ComponentDefinition{
			TagName: "My-Button",
		})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		testCases := []string{"My-Button", "my-button", "MY-BUTTON", "mY-bUtToN"}
		for _, tc := range testCases {
			if !registry.IsRegistered(tc) {
				t.Errorf("IsRegistered(%q) = false, want true", tc)
			}
		}
	})

	t.Run("idempotent re-registration with same source succeeds", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		_ = registry.Register(component_dto.ComponentDefinition{
			TagName:    "my-button",
			SourcePath: "components/my-button.pkc",
		})

		err := registry.Register(component_dto.ComponentDefinition{
			TagName:    "my-button",
			SourcePath: "components/my-button.pkc",
		})

		if err != nil {
			t.Errorf("Register() same source = %v, want nil (idempotent)", err)
		}

		if registry.Count() != 1 {
			t.Errorf("Count() = %d, want 1 after idempotent re-registration", registry.Count())
		}
	})

	t.Run("case insensitive idempotent re-registration succeeds", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		_ = registry.Register(component_dto.ComponentDefinition{
			TagName:    "my-button",
			SourcePath: "components/my-button.pkc",
		})

		err := registry.Register(component_dto.ComponentDefinition{
			TagName:    "My-Button",
			SourcePath: "components/my-button.pkc",
		})

		if err != nil {
			t.Errorf("Register() case-insensitive same source = %v, want nil", err)
		}
	})

	t.Run("conflicting source path fails", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		_ = registry.Register(component_dto.ComponentDefinition{
			TagName:    "my-button",
			SourcePath: "components/my-button.pkc",
		})

		err := registry.Register(component_dto.ComponentDefinition{
			TagName:    "my-button",
			SourcePath: "external/my-button.pkc",
		})

		if err == nil {
			t.Error("Register() different source = nil, want error")
		}
	})

	t.Run("validation error for invalid tag", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		err := registry.Register(component_dto.ComponentDefinition{
			TagName: "button",
		})

		if err == nil {
			t.Error("Register() invalid tag = nil, want error")
		}
	})

	t.Run("validation error for piko prefix", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		err := registry.Register(component_dto.ComponentDefinition{
			TagName: "piko:slot",
		})

		if err == nil {
			t.Error("Register() piko prefix = nil, want error")
		}
	})
}

func TestInMemoryRegistry_RegisterBatch(t *testing.T) {
	t.Run("empty batch succeeds", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		err := registry.RegisterBatch(nil)
		if err != nil {
			t.Errorf("RegisterBatch(nil) error = %v, want nil", err)
		}

		err = registry.RegisterBatch([]component_dto.ComponentDefinition{})
		if err != nil {
			t.Errorf("RegisterBatch([]) error = %v, want nil", err)
		}
	})

	t.Run("valid batch registration", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		definitions := []component_dto.ComponentDefinition{
			{TagName: "comp-a"},
			{TagName: "comp-b"},
			{TagName: "comp-c"},
		}

		err := registry.RegisterBatch(definitions)
		if err != nil {
			t.Errorf("RegisterBatch() error = %v, want nil", err)
		}

		if registry.Count() != 3 {
			t.Errorf("Count() = %d, want 3", registry.Count())
		}

		for _, definition := range definitions {
			if !registry.IsRegistered(definition.TagName) {
				t.Errorf("IsRegistered(%q) = false, want true", definition.TagName)
			}
		}
	})

	t.Run("batch with invalid tag fails atomically", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		definitions := []component_dto.ComponentDefinition{
			{TagName: "comp-a"},
			{TagName: "invalid"},
			{TagName: "comp-c"},
		}

		err := registry.RegisterBatch(definitions)
		if err == nil {
			t.Error("RegisterBatch() with invalid tag = nil, want error")
		}

		if registry.Count() != 0 {
			t.Errorf("Count() after failed batch = %d, want 0", registry.Count())
		}
	})

	t.Run("batch with duplicate in batch fails", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		definitions := []component_dto.ComponentDefinition{
			{TagName: "comp-a"},
			{TagName: "comp-a"},
		}

		err := registry.RegisterBatch(definitions)
		if err == nil {
			t.Error("RegisterBatch() with duplicate = nil, want error")
		}

		if registry.Count() != 0 {
			t.Errorf("Count() after failed batch = %d, want 0", registry.Count())
		}
	})

	t.Run("batch conflicts with existing registration from different source", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		_ = registry.Register(component_dto.ComponentDefinition{
			TagName:    "existing-comp",
			SourcePath: "local/existing-comp.pkc",
		})

		definitions := []component_dto.ComponentDefinition{
			{TagName: "new-comp", SourcePath: "ext/new-comp.pkc"},
			{TagName: "existing-comp", SourcePath: "ext/existing-comp.pkc"},
		}

		err := registry.RegisterBatch(definitions)
		if err == nil {
			t.Error("RegisterBatch() conflicting source = nil, want error")
		}

		if registry.Count() != 1 {
			t.Errorf("Count() after failed batch = %d, want 1", registry.Count())
		}

		if registry.IsRegistered("new-comp") {
			t.Error("new-comp should not be registered after failed batch")
		}
	})

	t.Run("batch with idempotent existing registration succeeds", func(t *testing.T) {
		registry := NewInMemoryRegistry()

		_ = registry.Register(component_dto.ComponentDefinition{
			TagName:    "existing-comp",
			SourcePath: "components/existing-comp.pkc",
		})

		definitions := []component_dto.ComponentDefinition{
			{TagName: "new-comp", SourcePath: "components/new-comp.pkc"},
			{TagName: "existing-comp", SourcePath: "components/existing-comp.pkc"},
		}

		err := registry.RegisterBatch(definitions)
		if err != nil {
			t.Errorf("RegisterBatch() idempotent = %v, want nil", err)
		}

		if registry.Count() != 2 {
			t.Errorf("Count() after idempotent batch = %d, want 2", registry.Count())
		}
	})
}

func TestInMemoryRegistry_Get(t *testing.T) {
	registry := NewInMemoryRegistry()

	original := component_dto.ComponentDefinition{
		TagName:    "my-button",
		SourcePath: "components/my-button.pkc",
		IsExternal: false,
	}
	_ = registry.Register(original)

	t.Run("get existing component", func(t *testing.T) {
		definition, found := registry.Get("my-button")
		if !found {
			t.Fatal("Get() found = false, want true")
		}

		if definition.TagName != original.TagName {
			t.Errorf("Get().TagName = %q, want %q", definition.TagName, original.TagName)
		}
		if definition.SourcePath != original.SourcePath {
			t.Errorf("Get().SourcePath = %q, want %q", definition.SourcePath, original.SourcePath)
		}
	})

	t.Run("get non-existing component", func(t *testing.T) {
		definition, found := registry.Get("non-existent")
		if found {
			t.Error("Get() non-existent found = true, want false")
		}
		if definition != nil {
			t.Error("Get() non-existent definition != nil")
		}
	})

	t.Run("get returns copy", func(t *testing.T) {
		definition, _ := registry.Get("my-button")
		definition.SourcePath = "modified"

		def2, _ := registry.Get("my-button")
		if def2.SourcePath == "modified" {
			t.Error("Get() should return a copy, not the original")
		}
	})
}

func TestInMemoryRegistry_All(t *testing.T) {
	registry := NewInMemoryRegistry()

	definitions := []component_dto.ComponentDefinition{
		{TagName: "comp-c"},
		{TagName: "comp-a"},
		{TagName: "comp-b"},
	}
	_ = registry.RegisterBatch(definitions)

	all := registry.All()

	if len(all) != 3 {
		t.Fatalf("All() len = %d, want 3", len(all))
	}

	expected := []string{"comp-a", "comp-b", "comp-c"}
	for i, name := range expected {
		if all[i].TagName != name {
			t.Errorf("All()[%d].TagName = %q, want %q", i, all[i].TagName, name)
		}
	}
}

func TestInMemoryRegistry_TagNames(t *testing.T) {
	registry := NewInMemoryRegistry()

	definitions := []component_dto.ComponentDefinition{
		{TagName: "comp-c"},
		{TagName: "comp-a"},
		{TagName: "comp-b"},
	}
	_ = registry.RegisterBatch(definitions)

	names := registry.TagNames()

	if len(names) != 3 {
		t.Fatalf("TagNames() len = %d, want 3", len(names))
	}

	expected := []string{"comp-a", "comp-b", "comp-c"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("TagNames()[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestInMemoryRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewInMemoryRegistry()

	var wg sync.WaitGroup
	numGoroutines := 100

	for index := range numGoroutines {
		wg.Go(func() {
			tagName := "comp-" + string(rune('a'+index%26)) + "-" + string(rune('0'+index%10))
			_ = registry.Register(component_dto.ComponentDefinition{
				TagName: tagName,
			})
		})
	}

	for range numGoroutines {
		wg.Go(func() {
			_ = registry.Count()
			_ = registry.All()
			_ = registry.TagNames()
			_ = registry.IsRegistered("comp-a-0")
		})
	}

	wg.Wait()

	if registry.Count() == 0 {
		t.Error("expected some registrations to succeed")
	}
}
