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

// Package ast_domain defines the core Abstract Syntax Tree types and
// operations for Piko templates.
//
// It contains the data structures for representing parsed HTML templates
// (element nodes, text content, directives, expressions), the parser,
// expression evaluator, tree traversal utilities, CSS selector query
// engine, serialisation to Go source, and a pooled arena allocator for
// high-throughput rendering. Cache port interfaces for adapter packages
// are also defined here.
//
// # Usage
//
// Parse a template and traverse its nodes:
//
//	ast, err := ast_domain.ParseAndTransform(source, "page.pk")
//	if err != nil {
//	    return err
//	}
//
//	// Walk all nodes depth-first
//	ast.Walk(func(node *ast_domain.TemplateNode) bool {
//	    if node.TagName == "div" {
//	        // Process div elements
//	    }
//	    return true // continue walking
//	})
//
//	// Or use CSS selectors
//	results := ast.Query(".container > ul li")
//
// # Directive system
//
// Piko directives extend HTML with reactive behaviour:
//
//   - Control flow: p-if, p-else-if, p-else, p-for, p-show
//   - Data binding: p-bind, p-model, p-text, p-html
//   - Events: p-on, p-event
//   - Styling: p-class, p-style
//   - Identity: p-key, p-context, p-ref
//
// # Thread safety
//
// [TemplateAST] and [TemplateNode] instances are not safe for
// concurrent modification. Use DeepClone to create independent copies
// for concurrent processing. The parallel walk methods (ParallelWalk)
// are safe for read-only traversal. [RenderArena] instances must not
// be shared between goroutines; acquire one per request from the
// global pool.
package ast_domain
