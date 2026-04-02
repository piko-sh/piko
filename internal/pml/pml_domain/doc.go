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

// Package pml_domain defines the core interfaces and domain models for
// the PikoML transformation engine.
//
// PikoML (Piko Markup Language) transforms custom <pml-*> components
// into email-safe HTML with responsive CSS. This package defines the
// contracts ([Transformer], [ComponentRegistry], [Component],
// [MediaQueryCollector], [MSOConditionalCollector]) between the
// transformation engine, component implementations, and validation
// logic, plus supporting types like [TransformationContext] and
// [StyleManager].
//
// # Transformation pipeline
//
// The engine performs a two-pass transformation:
//
//  1. Autowrap Pass: Ensures structural validity using post-order
//     traversal, wrapping loose content components into implicit
//     <pml-row><pml-col> layout structures
//  2. Transform Pass: Recursively converts PikoML components to
//     email-safe HTML
//
// Components receive a [TransformationContext] with access to
// configuration, computed styles (via [StyleManager]), parent
// context, and collectors for responsive CSS and MSO conditionals.
//
// # Email rendering
//
// For email contexts, use TransformForEmail which initialises an
// [EmailAssetRegistry] to collect CID embedding requests from
// <pml-img> tags.
package pml_domain
