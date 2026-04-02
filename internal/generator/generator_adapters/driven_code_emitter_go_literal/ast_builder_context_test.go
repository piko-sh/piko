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
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEmitChildren_ContextCancellation(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	em.resetState(context.Background())
	em.ctx = NewEmitterContext()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Children: []*ast_domain.TemplateNode{
			{
				TagName:  "span",
				NodeType: ast_domain.NodeElement,
				Location: ast_domain.Location{Line: 5},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					IsStructurallyStatic: true,
				},
			},
		},
	}

	tempVarIdent := cachedIdent("node")

	nodeEmitter := requireNodeEmitter(t, em)

	_, diagnostics := nodeEmitter.emitChildren(ctx, tempVarIdent, node, nil, "")

	require.NotEmpty(t, diagnostics, "Should return diagnostics")
	assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "cancelled", "Should mention cancellation")

}
