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

package cache_domain

import (
	"fmt"
	"slices"
	"testing"
)

func isolateProviderFactoryRegistry(t *testing.T) {
	t.Helper()
	providerFactoryBlueprintsMutex.Lock()
	original := providerFactoryBlueprints
	providerFactoryBlueprints = make(map[string]ProviderFactoryBlueprint)
	providerFactoryBlueprintsMutex.Unlock()
	t.Cleanup(func() {
		providerFactoryBlueprintsMutex.Lock()
		providerFactoryBlueprints = original
		providerFactoryBlueprintsMutex.Unlock()
	})
}

func TestRegisterProviderFactory_Success(t *testing.T) {
	isolateProviderFactoryRegistry(t)

	name := "test-factory-" + t.Name()
	RegisterProviderFactory(name, func(service Service, namespace string, options any) (any, error) {
		return nil, nil
	})

	factory, ok := GetProviderFactory(name)
	if !ok {
		t.Fatal("expected factory to be registered")
	}
	if factory == nil {
		t.Error("expected non-nil factory")
	}
}

func TestRegisterProviderFactory_DuplicatePanics(t *testing.T) {
	isolateProviderFactoryRegistry(t)

	name := "test-factory-dup-" + t.Name()
	RegisterProviderFactory(name, func(service Service, namespace string, options any) (any, error) {
		return nil, nil
	})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
		message := fmt.Sprintf("%v", r)
		if !contains(message, "already registered") {
			t.Errorf("unexpected panic message: %s", message)
		}
	}()

	RegisterProviderFactory(name, func(service Service, namespace string, options any) (any, error) {
		return nil, nil
	})
}

func TestGetProviderFactory_Found(t *testing.T) {
	isolateProviderFactoryRegistry(t)

	name := "test-factory-found-" + t.Name()
	RegisterProviderFactory(name, func(service Service, namespace string, options any) (any, error) {
		return "result", nil
	})

	factory, ok := GetProviderFactory(name)
	if !ok {
		t.Fatal("expected factory to be found")
	}

	result, err := factory(nil, "ns", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}
}

func TestGetProviderFactory_NotFound(t *testing.T) {
	_, ok := GetProviderFactory("nonexistent-factory-" + t.Name())
	if ok {
		t.Error("expected factory not to be found")
	}
}

func TestListProviderFactories(t *testing.T) {
	isolateProviderFactoryRegistry(t)

	name := "test-factory-list-" + t.Name()
	RegisterProviderFactory(name, func(service Service, namespace string, options any) (any, error) {
		return nil, nil
	})

	names := listProviderFactories()
	found := slices.Contains(names, name)
	if !found {
		t.Errorf("expected %q in factory list, got %v", name, names)
	}
}
