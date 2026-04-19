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

//go:build integration

package registry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func runFidelity(t *testing.T, config Config) {
	t.Run("every variant field round-trips", func(t *testing.T) {
		store := config.NewStore(t)

		artefact := buildArtefact("fid-1", "source/fid1")
		variant := buildVariant("full", "full/fid1")
		variant.SRIHash = "sha384-abc123"
		variant.ContentHash = "content-hash-xyz"
		variant.SizeBytes = 12345
		variant.MimeType = "image/webp"
		variant.MetadataTags.SetByName("encoding", "br")
		variant.Producer = registry_dto.ProducerBuild
		variant.Kind = registry_dto.KindDerived
		variant.Transform = registry_dto.VariantTransform{
			ParentVariantID:   "source",
			ParentContentHash: "content-hash-parent",
			CapabilityName:    "image-transform",
			CapabilityVersion: 3,
		}
		variant.Transform.Params.SetByName("width", "720")
		artefact.ActualVariants = append(artefact.ActualVariants, variant)
		upsert(t, store, artefact)

		got, err := store.GetArtefact(ctx(t), "fid-1")
		require.NoError(t, err)
		full := findVariant(got, "full")
		require.NotNil(t, full)

		assert.Equal(t, "content-hash-xyz", full.ContentHash, "ContentHash must survive")
		assert.Equal(t, int64(12345), full.SizeBytes, "SizeBytes must survive")
		assert.Equal(t, "image/webp", full.MimeType, "MimeType must survive")
		encoding, ok := full.MetadataTags.GetByName("encoding")
		assert.True(t, ok, "a metadata tag must survive")
		assert.Equal(t, "br", encoding)

		assert.Equal(t, registry_dto.ProducerBuild, full.Producer, "Producer must survive")
		assert.Equal(t, registry_dto.KindDerived, full.Kind, "Kind must survive")
		assert.Equal(t, "source", full.Transform.ParentVariantID, "the transform parent must survive")
		assert.Equal(t, "content-hash-parent", full.Transform.ParentContentHash, "the transform parent hash must survive")
		assert.Equal(t, "image-transform", full.Transform.CapabilityName, "the transform capability must survive")
		assert.Equal(t, uint32(3), full.Transform.CapabilityVersion, "the transform capability version must survive")
		width, ok := full.Transform.Params.GetByName("width")
		assert.True(t, ok, "a transform parameter must survive")
		assert.Equal(t, "720", width)

		if config.SupportsSRIHashPersistence {
			assert.Equal(t, "sha384-abc123", full.SRIHash,
				"SRIHash must survive: it drives the integrity attribute on script and link tags, "+
					"and a backend that drops it silently disables Subresource Integrity in production")
		}
	})
}
