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
	"errors"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

func mockWeigher[K comparable, V any](key K, value V) uint32 {
	return 1
}

func TestValidateOptions_ValidConfigurations(t *testing.T) {
	tests := []struct {
		name    string
		options cache_dto.Options[string, string]
	}{
		{
			name:    "empty options",
			options: cache_dto.Options[string, string]{},
		},
		{
			name: "only maximum size",
			options: cache_dto.Options[string, string]{
				MaximumSize: 100,
			},
		},
		{
			name: "maximum weight with weigher",
			options: cache_dto.Options[string, string]{
				MaximumWeight: 1000,
				Weigher:       mockWeigher[string, string],
			},
		},
		{
			name: "initial capacity only",
			options: cache_dto.Options[string, string]{
				InitialCapacity: 50,
			},
		},
		{
			name: "maximum size with initial capacity",
			options: cache_dto.Options[string, string]{
				MaximumSize:     100,
				InitialCapacity: 50,
			},
		},
		{
			name: "zero maximum size (unlimited)",
			options: cache_dto.Options[string, string]{
				MaximumSize: 0,
			},
		},
		{
			name: "zero initial capacity",
			options: cache_dto.Options[string, string]{
				InitialCapacity: 0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOptions(tc.options)
			if err != nil {
				t.Errorf("expected valid configuration, got error: %v", err)
			}
		})
	}
}

func TestValidateOptions_MaximumSizeAndWeight(t *testing.T) {
	options := cache_dto.Options[string, string]{
		MaximumSize:   100,
		MaximumWeight: 1000,
		Weigher:       mockWeigher[string, string],
	}

	err := ValidateOptions(options)
	if err == nil {
		t.Fatal("expected error when both MaximumSize and MaximumWeight are set, got nil")
	}

	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}

	expectedMessage := "cannot set both MaximumSize and MaximumWeight"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestValidateOptions_MaximumSizeAndWeigher(t *testing.T) {
	options := cache_dto.Options[string, string]{
		MaximumSize: 100,
		Weigher:     mockWeigher[string, string],
	}

	err := ValidateOptions(options)
	if err == nil {
		t.Fatal("expected error when both MaximumSize and Weigher are set, got nil")
	}

	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}

	expectedMessage := "cannot set both MaximumSize and a Weigher"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestValidateOptions_MaximumWeightWithoutWeigher(t *testing.T) {
	options := cache_dto.Options[string, string]{
		MaximumWeight: 1000,
		Weigher:       nil,
	}

	err := ValidateOptions(options)
	if err == nil {
		t.Fatal("expected error when MaximumWeight is set without Weigher, got nil")
	}

	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}

	expectedMessage := "MaximumWeight requires a Weigher function"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestValidateOptions_WeigherWithoutMaximumWeight(t *testing.T) {
	options := cache_dto.Options[string, string]{
		MaximumWeight: 0,
		Weigher:       mockWeigher[string, string],
	}

	err := ValidateOptions(options)
	if err == nil {
		t.Fatal("expected error when Weigher is set without MaximumWeight, got nil")
	}

	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}

	expectedMessage := "Weigher requires MaximumWeight to be set"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestValidateOptions_NegativeMaximumSize(t *testing.T) {
	options := cache_dto.Options[string, string]{
		MaximumSize: -100,
	}

	err := ValidateOptions(options)
	if err == nil {
		t.Fatal("expected error for negative MaximumSize, got nil")
	}

	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}

	expectedMessage := "MaximumSize must be non-negative"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestValidateOptions_NegativeInitialCapacity(t *testing.T) {
	options := cache_dto.Options[string, string]{
		InitialCapacity: -50,
	}

	err := ValidateOptions(options)
	if err == nil {
		t.Fatal("expected error for negative InitialCapacity, got nil")
	}

	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}

	expectedMessage := "InitialCapacity must be non-negative"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestValidateOptions_MultipleViolations(t *testing.T) {
	options := cache_dto.Options[string, string]{
		MaximumSize:   100,
		MaximumWeight: 1000,
		Weigher:       mockWeigher[string, string],
	}

	err := ValidateOptions(options)
	if err == nil {
		t.Fatal("expected error for invalid configuration, got nil")
	}

	expectedMessage := "cannot set both MaximumSize and MaximumWeight"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestValidateOptions_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		options   cache_dto.Options[string, string]
		wantError bool
	}{
		{
			name: "maximum size exactly 1",
			options: cache_dto.Options[string, string]{
				MaximumSize: 1,
			},
			wantError: false,
		},
		{
			name: "maximum weight exactly 1 with weigher",
			options: cache_dto.Options[string, string]{
				MaximumWeight: 1,
				Weigher:       mockWeigher[string, string],
			},
			wantError: false,
		},
		{
			name: "initial capacity exactly 1",
			options: cache_dto.Options[string, string]{
				InitialCapacity: 1,
			},
			wantError: false,
		},
		{
			name: "maximum size exactly 0 (should be valid - means unlimited)",
			options: cache_dto.Options[string, string]{
				MaximumSize: 0,
			},
			wantError: false,
		},
		{
			name: "maximum size -1 (boundary)",
			options: cache_dto.Options[string, string]{
				MaximumSize: -1,
			},
			wantError: true,
		},
		{
			name: "initial capacity -1 (boundary)",
			options: cache_dto.Options[string, string]{
				InitialCapacity: -1,
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOptions(tc.options)

			if tc.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
