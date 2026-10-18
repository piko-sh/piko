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

package generator_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

const (
	collectionItemsDeterminismRuns = 32
)

func TestExtractCollectionItemsFromModule_SharedCollectionInHashOrder(t *testing.T) {
	t.Parallel()

	newTemplate := func(slugs ...string) *annotator_dto.VirtualComponent {
		instances := make([]annotator_dto.VirtualPageInstance, 0, len(slugs))
		for _, slug := range slugs {
			instances = append(instances, annotator_dto.VirtualPageInstance{Slug: slug, InitialProps: map[string]any{}})
		}
		return &annotator_dto.VirtualComponent{
			Source:           &annotator_dto.ParsedComponent{HasCollection: true, CollectionName: "docs"},
			VirtualInstances: instances,
		}
	}
	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"hash_d": newTemplate("d1", "d2"),
			"hash_b": newTemplate("b1"),
			"hash_a": newTemplate("a1", "a2"),
			"hash_c": newTemplate("c1"),
			"hash_e": newTemplate("e1"),
			"hash_f": newTemplate("f1"),
		},
	}
	want := []string{"a1", "a2", "b1", "c1", "d1", "d2", "e1", "f1"}

	for range collectionItemsDeterminismRuns {
		items := extractCollectionItemsFromModule(vm)["docs"]

		slugs := make([]string, 0, len(items))
		for _, item := range items {
			slugs = append(slugs, item.Slug)
		}
		require.Equal(t, want, slugs, "items of a shared collection must follow component hash order on every run")
	}
}
