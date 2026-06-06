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

package emitter_shared

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestBuildAsyncExecMethod_HasErrorOnlySignature(t *testing.T) {
	t.Parallel()

	query := &querier_dto.AnalysedQuery{
		Name:    "PurgeOldEvents",
		Command: querier_dto.QueryCommandAsyncExec,
	}
	mappings := &querier_dto.TypeMappingTable{}
	tracker := NewImportTracker()
	strategy := &fakeRuntimeBuilderStrategy{}

	decl := BuildAsyncExecMethod(query, mappings, tracker, strategy)
	require.NotNil(t, decl)

	require.NotNil(t, decl.Type.Results)
	require.Len(t, decl.Type.Results.List, 1, "async exec returns only error")
}

func TestBuildAsyncExecMethod_AttachesAsyncDocComment(t *testing.T) {
	t.Parallel()

	query := &querier_dto.AnalysedQuery{
		Name:    "PurgeOldEvents",
		Command: querier_dto.QueryCommandAsyncExec,
	}
	mappings := &querier_dto.TypeMappingTable{}
	tracker := NewImportTracker()
	strategy := &fakeRuntimeBuilderStrategy{}

	decl := BuildAsyncExecMethod(query, mappings, tracker, strategy)
	require.NotNil(t, decl)
	require.NotNil(t, decl.Doc, "async exec method must carry a doc comment")

	var builder strings.Builder
	for _, comment := range decl.Doc.List {
		builder.WriteString(comment.Text)
		builder.WriteByte('\n')
	}
	combined := builder.String()

	assert.Contains(t, combined, "PurgeOldEvents")
	assert.Contains(t, combined, "asynchronous mutation")
	assert.Contains(t, combined, "system.mutations")
	assert.Contains(t, combined, "acceptance step")
}

func TestBuildAsyncExecMethod_BodyMatchesExecMethod(t *testing.T) {
	t.Parallel()

	query := &querier_dto.AnalysedQuery{
		Name:    "PurgeOldEvents",
		Command: querier_dto.QueryCommandAsyncExec,
	}
	mappings := &querier_dto.TypeMappingTable{}
	asyncTracker := NewImportTracker()
	syncTracker := NewImportTracker()
	strategy := &fakeRuntimeBuilderStrategy{}

	asyncDecl := BuildAsyncExecMethod(query, mappings, asyncTracker, strategy)

	syncQuery := &querier_dto.AnalysedQuery{
		Name:    "PurgeOldEvents",
		Command: querier_dto.QueryCommandExec,
	}
	syncDecl := BuildExecMethod(syncQuery, mappings, syncTracker, strategy)

	require.NotNil(t, asyncDecl)
	require.NotNil(t, syncDecl)

	asyncDecl.Doc = nil

	asyncSource := renderDecl(t, asyncDecl)
	syncSource := renderDecl(t, syncDecl)

	assert.Equal(t, syncSource, asyncSource,
		"async-exec body must be identical to exec body; doc comment is the only difference")
}
