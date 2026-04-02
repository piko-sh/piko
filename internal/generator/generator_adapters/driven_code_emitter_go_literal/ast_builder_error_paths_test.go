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

package driven_code_emitter_go_literal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestBuildCircularDependencyDiagnostic_NoCycle(t *testing.T) {
	t.Parallel()

	builder := &astBuilder{}

	invocationsMap := map[string]*annotator_dto.PartialInvocation{
		"A": {
			PartialAlias:  "PartialA",
			InvocationKey: "A",
			Location:      ast_domain.Location{Line: 10},
		},
		"B": {
			PartialAlias:  "PartialB",
			InvocationKey: "B",
			Location:      ast_domain.Location{Line: 20},
		},
	}

	inDegree := map[string]int{
		"A": 0,
		"B": 0,
	}

	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
	}

	diagnostics := builder.buildCircularDependencyDiagnostic(invocationsMap, inDegree, virtualModule)

	assert.Nil(t, diagnostics, "Should return nil when no circular dependency exists")
}

func TestBuildCircularDependencyDiagnostic_WithCycle(t *testing.T) {
	t.Parallel()

	builder := &astBuilder{}

	invocationsMap := map[string]*annotator_dto.PartialInvocation{
		"A": {
			PartialAlias:  "PartialA",
			InvocationKey: "A",
			Location:      ast_domain.Location{Line: 10},
		},
	}

	inDegree := map[string]int{
		"A": 1,
	}

	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"c_a": {
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/app/partials/a.pk",
				},
			},
		},
	}

	diagnostics := builder.buildCircularDependencyDiagnostic(invocationsMap, inDegree, virtualModule)

	assert.NotNil(t, diagnostics, "Should return diagnostics for cycle")
	assert.Len(t, diagnostics, 1, "Should return one diagnostic")
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "circular dependency")
	assert.Equal(t, "PartialA", diagnostics[0].Expression)
}

func TestFindNodeInCycle_NoCycle(t *testing.T) {
	t.Parallel()

	invocationsMap := map[string]*annotator_dto.PartialInvocation{
		"A": {PartialAlias: "PartialA"},
		"B": {PartialAlias: "PartialB"},
	}

	inDegree := map[string]int{
		"A": 0,
		"B": 0,
	}

	result := findNodeInCycle(invocationsMap, inDegree)

	assert.Nil(t, result, "Should return nil when no node is in a cycle")
}

func TestFindNodeInCycle_WithCycle(t *testing.T) {
	t.Parallel()

	invocationsMap := map[string]*annotator_dto.PartialInvocation{
		"A": {PartialAlias: "PartialA"},
		"B": {PartialAlias: "PartialB"},
	}

	inDegree := map[string]int{
		"A": 0,
		"B": 1,
	}

	result := findNodeInCycle(invocationsMap, inDegree)

	if assert.NotNil(t, result, "Should find node in cycle") {
		assert.Equal(t, "PartialB", result.PartialAlias)
	}
}

func TestGetSourcePathForInvocation_ComponentNotFound(t *testing.T) {
	t.Parallel()

	invocation := &annotator_dto.PartialInvocation{
		InvokerHashedName: "c_missing",
	}

	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"c_home": {
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/app/home.pk",
				},
			},
		},
	}

	path := getSourcePathForInvocation(invocation, virtualModule)

	assert.Equal(t, "", path, "Should return empty string for missing component")
}

func TestGetSourcePathForInvocation_ComponentFound(t *testing.T) {
	t.Parallel()

	invocation := &annotator_dto.PartialInvocation{
		InvokerHashedName: "c_home",
	}

	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"c_home": {
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/app/home.pk",
				},
			},
		},
	}

	path := getSourcePathForInvocation(invocation, virtualModule)

	assert.Equal(t, "/app/home.pk", path)
}
