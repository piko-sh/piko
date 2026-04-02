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

// Package pml_components implements all built-in PikoML components.
//
// PikoML (Piko Markup Language) is Piko's email templating system. Each
// component implements the [pml_domain.Component] interface, transforming
// PikoML tags (e.g., pml-row, pml-col, pml-button) into email-compatible
// HTML with proper table-based layouts and Outlook VML fallbacks.
//
// # Component registration
//
// Use [RegisterBuiltIns] to create a registry populated with all standard
// components:
//
//	registry, err := pml_components.RegisterBuiltIns()
//	if err != nil {
//	    return err
//	}
//	component, ok := registry.Get("pml-row")
//
// # Thread safety
//
// The [componentRegistry] implementation is safe for concurrent use. Component
// instances themselves are stateless and can be shared across goroutines.
package pml_components
