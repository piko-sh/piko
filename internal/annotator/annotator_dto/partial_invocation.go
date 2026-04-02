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

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// PartialInvocation represents a single instance of a partial being used.
// It is created by the PartialExpander and finalised by the ComponentLinker.
type PartialInvocation struct {
	// InvocationKey is the unique identifier for this invocation. It is temporary
	// during expansion and becomes final after linking.
	InvocationKey string

	// PartialAlias is the component alias from the `is="..."` attribute.
	PartialAlias string

	// PartialHashedName is the hashed name of the component being
	// invoked (Piko-domain key).
	PartialHashedName string

	// PassedProps maps property names to their values. It holds the properties
	// passed from the invoker, including static attributes and dynamic bindings.
	PassedProps map[string]ast_domain.PropValue

	// RequestOverrides maps property names to values that replace request object
	// properties when rendering this partial.
	RequestOverrides map[string]ast_domain.PropValue

	// InvokerHashedName is the hashed name of the component that
	// called this partial.
	InvokerHashedName string

	// InvokerInvocationKey is the canonical invocation key of the parent partial
	// that is invoking this partial; empty for partials invoked directly from
	// pages. This differentiates nested partial invocations that have identical
	// expression strings but belong to different parent instances.
	InvokerInvocationKey string

	// DependsOn contains the invocation keys of other partial
	// invocations that this invocation depends on, populated during
	// linking by analysing SourceInvocationKey of identifiers in
	// PassedProps expressions.
	//
	// Used for topological sorting of partial invocations during
	// code generation.
	DependsOn []string

	// Location is the source code position where this invocation appears.
	Location ast_domain.Location
}
