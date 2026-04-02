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

package registry_test_test

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/registry/registry_schema"
)

type fixtureArtefact struct {
	ID         string
	SourcePath string
	CreatedAt  int64
	UpdatedAt  int64
	Variants   []fixtureVariant
	Profiles   []fixtureProfile
}

type fixtureVariant struct {
	VariantID        string
	StorageKey       string
	StorageBackendID string
	MimeType         string
	SizeBytes        int64
	Status           string
	CreatedAt        int64
	Tags             map[string]string
}

type fixtureProfile struct {
	Name           string
	CapabilityName string
	Priority       string
	DependsOn      []string
	Tags           map[string]string
}

type fixtureBlobRef struct {
	StorageKey       string
	StorageBackendID string
	RefCount         int
	ContentHash      string
	SizeBytes        int64
	MimeType         string
	CreatedAt        int64
	LastReferencedAt int64
}

func loadBaseScenarioFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	artefacts := []fixtureArtefact{
		{
			ID:         "lib/main.css",
			SourcePath: "source/lib/main.css",
			CreatedAt:  1723982400,
			UpdatedAt:  1723982400,
			Variants: []fixtureVariant{
				{
					VariantID:        "source",
					StorageKey:       "source/hash1.css",
					StorageBackendID: "test_disk",
					MimeType:         "text/css",
					SizeBytes:        1024,
					Status:           "READY",
					CreatedAt:        1723982400,
				},
				{
					VariantID:        "minified",
					StorageKey:       "minified/hash2.css",
					StorageBackendID: "test_disk",
					MimeType:         "text/css",
					SizeBytes:        512,
					Status:           "READY",
					CreatedAt:        1723982400,
					Tags:             map[string]string{"type": "css"},
				},
			},
			Profiles: []fixtureProfile{
				{
					Name:           "minified",
					CapabilityName: "minify-css",
					Priority:       "NEED",
					DependsOn:      []string{"source"},
				},
			},
		},
		{
			ID:         "components/header.pkc",
			SourcePath: "source/components/header.pkc",
			CreatedAt:  1723982400,
			UpdatedAt:  1723982400,
			Variants: []fixtureVariant{
				{
					VariantID:        "source",
					StorageKey:       "source/hash3.pkc",
					StorageBackendID: "test_disk",
					MimeType:         "text/plain",
					SizeBytes:        2048,
					Status:           "READY",
					CreatedAt:        1723982400,
					Tags:             map[string]string{"type": "component"},
				},
			},
		},
		{
			ID:         "assets/logo.svg",
			SourcePath: "source/assets/logo.svg",
			CreatedAt:  1723982400,
			UpdatedAt:  1723982400,
		},
	}

	blobRefs := []fixtureBlobRef{
		{StorageKey: "source/hash1.css", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "hash1", SizeBytes: 1024, MimeType: "text/css", CreatedAt: 1723982400, LastReferencedAt: 1723982400},
		{StorageKey: "minified/hash2.css", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "hash2", SizeBytes: 512, MimeType: "text/css", CreatedAt: 1723982400, LastReferencedAt: 1723982400},
		{StorageKey: "source/hash3.pkc", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "hash3", SizeBytes: 2048, MimeType: "text/plain", CreatedAt: 1723982400, LastReferencedAt: 1723982400},
	}

	insertFixtureData(t, db, artefacts, blobRefs)
}

func loadComplexDependenciesFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	artefacts := []fixtureArtefact{

		{
			ID:         "scripts/app.js",
			SourcePath: "source/app.js",
			CreatedAt:  1724061600,
			UpdatedAt:  1724061600,
			Variants: []fixtureVariant{
				{VariantID: "source", StorageKey: "source/js_hash1", StorageBackendID: "test_disk", MimeType: "application/javascript", SizeBytes: 15360, Status: "READY", CreatedAt: 1724061600},
				{VariantID: "compiled", StorageKey: "compiled/js_hash2", StorageBackendID: "test_disk", MimeType: "application/javascript", SizeBytes: 12288, Status: "READY", CreatedAt: 1724061600, Tags: map[string]string{"type": "javascript", "module": "true"}},
				{VariantID: "minified", StorageKey: "minified/js_hash3", StorageBackendID: "test_disk", MimeType: "application/javascript", SizeBytes: 4096, Status: "READY", CreatedAt: 1724061600, Tags: map[string]string{"compression": "none"}},
				{VariantID: "gzipped", StorageKey: "gzipped/js_hash4", StorageBackendID: "test_disk", MimeType: "application/javascript", SizeBytes: 1024, Status: "READY", CreatedAt: 1724061600, Tags: map[string]string{"compression": "gzip"}},
				{VariantID: "brotli", StorageKey: "brotli/js_hash5", StorageBackendID: "test_disk", MimeType: "application/javascript", SizeBytes: 850, Status: "READY", CreatedAt: 1724061600, Tags: map[string]string{"compression": "br"}},
			},
			Profiles: []fixtureProfile{
				{Name: "compiled", CapabilityName: "compile-js", Priority: "NEED", DependsOn: []string{"source"}, Tags: map[string]string{"type": "javascript", "module": "true"}},
				{Name: "minified", CapabilityName: "minify-js", Priority: "WANT", DependsOn: []string{"compiled"}, Tags: map[string]string{"compression": "none"}},
				{Name: "gzipped", CapabilityName: "compress-gzip", Priority: "WANT", DependsOn: []string{"minified"}, Tags: map[string]string{"compression": "gzip"}},
				{Name: "brotli", CapabilityName: "compress-brotli", Priority: "WANT", DependsOn: []string{"minified"}, Tags: map[string]string{"compression": "br"}},
			},
		},

		{
			ID:         "components/user-profile.pkc",
			SourcePath: "source/user-profile.pkc",
			CreatedAt:  1724061600,
			UpdatedAt:  1724061600,
			Variants: []fixtureVariant{
				{VariantID: "source", StorageKey: "source/pkc_hash1", StorageBackendID: "test_disk", MimeType: "text/plain", SizeBytes: 5120, Status: "READY", CreatedAt: 1724061600},
				{VariantID: "compiled_js", StorageKey: "compiled/pkc_hash2", StorageBackendID: "test_disk", MimeType: "application/javascript", SizeBytes: 4096, Status: "READY", CreatedAt: 1724061600, Tags: map[string]string{"type": "component-js", "tagName": "user-profile"}},
				{VariantID: "minified_js", StorageKey: "minified/pkc_hash3_old", StorageBackendID: "test_disk", MimeType: "application/javascript", SizeBytes: 1024, Status: "STALE", CreatedAt: 1724061540, Tags: map[string]string{"type": "minified-js"}},
			},
			Profiles: []fixtureProfile{
				{Name: "compiled_js", CapabilityName: "compile-component", Priority: "NEED", DependsOn: []string{"source"}, Tags: map[string]string{"type": "component-js", "tagName": "user-profile"}},
				{Name: "minified_js", CapabilityName: "minify-js", Priority: "WANT", DependsOn: []string{"compiled_js"}, Tags: map[string]string{"type": "minified-js"}},
			},
		},

		{
			ID:         "assets/icon-sheet.svg",
			SourcePath: "source/icon-sheet.svg",
			CreatedAt:  1724061600,
			UpdatedAt:  1724061600,
			Variants: []fixtureVariant{
				{VariantID: "source", StorageKey: "source/svg_hash1", StorageBackendID: "test_disk", MimeType: "image/svg+xml", SizeBytes: 25600, Status: "READY", CreatedAt: 1724061600},
				{VariantID: "minified", StorageKey: "minified/svg_hash2", StorageBackendID: "test_disk", MimeType: "image/svg+xml", SizeBytes: 18432, Status: "READY", CreatedAt: 1724061600},
				{VariantID: "png_128x128", StorageKey: "raster/svg_hash3.png", StorageBackendID: "test_disk", MimeType: "image/png", SizeBytes: 5120, Status: "READY", CreatedAt: 1724061600, Tags: map[string]string{"type": "raster-image", "size": "128x128"}},
				{VariantID: "png_256x256", StorageKey: "raster/svg_hash4.png", StorageBackendID: "test_disk", MimeType: "image/png", SizeBytes: 9216, Status: "READY", CreatedAt: 1724061600, Tags: map[string]string{"type": "raster-image", "size": "256x256"}},
			},
			Profiles: []fixtureProfile{
				{Name: "minified", CapabilityName: "minify-svg", Priority: "NEED", DependsOn: []string{"source"}},
				{Name: "png_128x128", CapabilityName: "svg-to-png", Priority: "WANT", DependsOn: []string{"minified"}},
				{Name: "png_256x256", CapabilityName: "svg-to-png", Priority: "WANT", DependsOn: []string{"minified"}},
			},
		},
	}

	blobRefs := []fixtureBlobRef{

		{StorageKey: "source/js_hash1", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "js_hash1", SizeBytes: 15360, MimeType: "application/javascript", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "compiled/js_hash2", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "js_hash2", SizeBytes: 12288, MimeType: "application/javascript", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "minified/js_hash3", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "js_hash3", SizeBytes: 4096, MimeType: "application/javascript", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "gzipped/js_hash4", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "js_hash4", SizeBytes: 1024, MimeType: "application/javascript", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "brotli/js_hash5", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "js_hash5", SizeBytes: 850, MimeType: "application/javascript", CreatedAt: 1724061600, LastReferencedAt: 1724061600},

		{StorageKey: "source/pkc_hash1", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "pkc_hash1", SizeBytes: 5120, MimeType: "text/plain", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "compiled/pkc_hash2", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "pkc_hash2", SizeBytes: 4096, MimeType: "application/javascript", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "minified/pkc_hash3_old", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "pkc_hash3_old", SizeBytes: 1024, MimeType: "application/javascript", CreatedAt: 1724061540, LastReferencedAt: 1724061540},

		{StorageKey: "source/svg_hash1", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "svg_hash1", SizeBytes: 25600, MimeType: "image/svg+xml", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "minified/svg_hash2", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "svg_hash2", SizeBytes: 18432, MimeType: "image/svg+xml", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "raster/svg_hash3.png", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "svg_hash3", SizeBytes: 5120, MimeType: "image/png", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
		{StorageKey: "raster/svg_hash4.png", StorageBackendID: "test_disk", RefCount: 1, ContentHash: "svg_hash4", SizeBytes: 9216, MimeType: "image/png", CreatedAt: 1724061600, LastReferencedAt: 1724061600},
	}

	insertFixtureData(t, db, artefacts, blobRefs)
}

func insertFixtureData(t *testing.T, db *sql.DB, artefacts []fixtureArtefact, blobRefs []fixtureBlobRef) {
	t.Helper()

	for _, br := range blobRefs {
		_, err := db.Exec(`
			INSERT INTO blob_reference (storage_key, storage_backend_id, ref_count, content_hash, size_bytes, mime_type, created_at, last_referenced_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			br.StorageKey, br.StorageBackendID, br.RefCount, br.ContentHash, br.SizeBytes, br.MimeType, br.CreatedAt, br.LastReferencedAt)
		require.NoError(t, err, "failed to insert blob reference: %s", br.StorageKey)
	}

	for _, art := range artefacts {

		meta := buildArtefactMetaFromFixture(art)

		dataFbs := registry_schema.BuildArtefactMeta(meta)

		_, err := db.Exec(`
			INSERT INTO artefact (id, source_path, created_at, updated_at, data_fbs)
			VALUES (?, ?, ?, ?, ?)`,
			art.ID, art.SourcePath, art.CreatedAt, art.UpdatedAt, dataFbs)
		require.NoError(t, err, "failed to insert artefact: %s", art.ID)

		for _, v := range art.Variants {
			_, err := db.Exec(`
				INSERT INTO variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				art.ID, v.VariantID, v.StorageKey, v.StorageBackendID, v.MimeType, v.SizeBytes, v.Status, v.CreatedAt)
			require.NoError(t, err, "failed to insert variant: %s/%s", art.ID, v.VariantID)

			for key, value := range v.Tags {
				_, err := db.Exec(`
					INSERT INTO variant_tag (artefact_id, variant_id, tag_key, tag_value)
					VALUES (?, ?, ?, ?)`,
					art.ID, v.VariantID, key, value)
				require.NoError(t, err, "failed to insert variant tag: %s/%s/%s", art.ID, v.VariantID, key)
			}
		}

		for _, p := range art.Profiles {
			dependsOnJSON := "[]"
			if len(p.DependsOn) > 0 {
				dependsOnJSON = `["` + joinStrings(p.DependsOn, `","`) + `"]`
			}

			tagsJSON := "{}"
			if len(p.Tags) > 0 {
				tagsJSON = buildTagsJSON(p.Tags)
			}

			_, err := db.Exec(`
				INSERT INTO desired_profile (artefact_id, name, capability_name, priority, depends_on_json, tags_json)
				VALUES (?, ?, ?, ?, ?, ?)`,
				art.ID, p.Name, p.CapabilityName, p.Priority, dependsOnJSON, tagsJSON)
			require.NoError(t, err, "failed to insert desired profile: %s/%s", art.ID, p.Name)
		}
	}
}

func buildArtefactMetaFromFixture(art fixtureArtefact) *registry_dto.ArtefactMeta {
	meta := &registry_dto.ArtefactMeta{
		ID:              art.ID,
		SourcePath:      art.SourcePath,
		CreatedAt:       time.Unix(art.CreatedAt, 0),
		UpdatedAt:       time.Unix(art.UpdatedAt, 0),
		ActualVariants:  make([]registry_dto.Variant, 0, len(art.Variants)),
		DesiredProfiles: make([]registry_dto.NamedProfile, 0, len(art.Profiles)),
	}

	for _, v := range art.Variants {
		variant := registry_dto.Variant{
			VariantID:        v.VariantID,
			StorageKey:       v.StorageKey,
			StorageBackendID: v.StorageBackendID,
			MimeType:         v.MimeType,
			SizeBytes:        v.SizeBytes,
			Status:           registry_dto.VariantStatus(v.Status),
			CreatedAt:        time.Unix(v.CreatedAt, 0),
		}
		for k, value := range v.Tags {
			variant.MetadataTags.SetByName(k, value)
		}
		meta.ActualVariants = append(meta.ActualVariants, variant)
	}

	for _, p := range art.Profiles {
		profile := registry_dto.NamedProfile{
			Name: p.Name,
			Profile: registry_dto.DesiredProfile{
				CapabilityName: p.CapabilityName,
				Priority:       registry_dto.ProfilePriority(p.Priority),
			},
		}
		for _, dependency := range p.DependsOn {
			profile.Profile.DependsOn.Add(dependency)
		}
		for k, value := range p.Tags {
			profile.Profile.ResultingTags.SetByName(k, value)
		}
		meta.DesiredProfiles = append(meta.DesiredProfiles, profile)
	}

	return meta
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString(strs[0])
	for i := 1; i < len(strs); i++ {
		result.WriteString(sep + strs[i])
	}
	return result.String()
}

func buildTagsJSON(tags map[string]string) string {
	if len(tags) == 0 {
		return "{}"
	}
	var result strings.Builder
	result.WriteString("{")
	first := true
	for k, v := range tags {
		if !first {
			result.WriteString(", ")
		}
		result.WriteString(`"` + k + `": "` + v + `"`)
		first = false
	}
	result.WriteString("}")
	return result.String()
}
