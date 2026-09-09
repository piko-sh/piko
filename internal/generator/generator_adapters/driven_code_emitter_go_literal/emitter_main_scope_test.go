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
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func annotationResultForScope(t *testing.T, sourcePath, hashedName string) *annotator_dto.AnnotationResult {
	t.Helper()

	path := sourcePath

	return &annotator_dto.AnnotationResult{
		AnnotatedAST: &ast_domain.TemplateAST{SourcePath: &path},
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				hashedName: {HashedName: hashedName},
			},
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{sourcePath: hashedName},
			},
		},
	}
}

func TestResetStateSetsMainComponentScope(t *testing.T) {
	t.Parallel()

	em, ok := NewEmitter(context.Background()).(*emitter)
	require.True(t, ok)

	em.AnnotationResult = annotationResultForScope(t, "pages/main.pk", "pages_main_aaaa1111")
	em.resetState()

	require.Equal(t, "pages_main_aaaa1111", em.astBuilder.staticEmitter.mainComponentScope,
		"The static emitter must know the main component scope, because it decides the CSS "+
			"scope ID applied to slotted static content")
}

func TestResetStateDoesNotCarryScopeBetweenArtefacts(t *testing.T) {
	t.Parallel()

	em, ok := NewEmitter(context.Background()).(*emitter)
	require.True(t, ok)

	em.AnnotationResult = annotationResultForScope(t, "pages/first.pk", "pages_first_aaaa1111")
	em.resetState()
	require.Equal(t, "pages_first_aaaa1111", em.astBuilder.staticEmitter.mainComponentScope)

	em.AnnotationResult = annotationResultForScope(t, "pages/second.pk", "pages_second_bbbb2222")
	em.resetState()

	require.Equal(t, "pages_second_bbbb2222", em.astBuilder.staticEmitter.mainComponentScope,
		"A second artefact must not inherit the first artefact's scope. Pooling the ast "+
			"builder previously reused a static emitter whose scope was never reassigned, so "+
			"slotted content was scoped by whichever artefact happened to run before it")
}
