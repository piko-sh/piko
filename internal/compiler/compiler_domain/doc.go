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

// Package compiler_domain provides the core business logic for compiling
// Single File Components (.pkc) into JavaScript and HTML artefacts.
//
// This package handles the complete SFC compilation pipeline: parsing
// TypeScript/JavaScript, transforming the AST for reactive state management,
// building virtual DOM render methods, and generating optimised output for
// custom web components.
//
// # Compilation pipeline
//
// The compilation process follows these stages:
//
//  1. Parse SFC into template, script, and style sections
//  2. Parse JavaScript/TypeScript and extract metadata
//  3. Transform state properties into reactive bindings
//  4. Build VDOM render method from template AST
//  5. Inject event bindings and static CSS
//  6. Generate final JavaScript with custom element registration
//
// # Design rationale
//
// Pre-compiling .pkc files rather than interpreting them at runtime catches
// errors at build time and produces optimised JavaScript output. The pipeline
// converts between esbuild and tdewolff AST formats because esbuild handles
// TypeScript stripping and module resolution while tdewolff provides the
// fine-grained AST manipulation needed for reactive state transforms and
// VDOM generation. An orchestrator coordinates the stages so each step can
// be tested and traced independently.
package compiler_domain
