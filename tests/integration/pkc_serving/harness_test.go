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

package pkc_serving_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	registry_querier_adapter "piko.sh/piko/internal/registry/registry_dal/querier_adapter"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/registry_blob_adapter"
	"piko.sh/piko/tests/integration/testutil"
	db_testutil "piko.sh/piko/tests/testutil"

	_ "github.com/mattn/go-sqlite3"
)

type pkcServingHarness struct {
	*testutil.BaseHarness
	registryService registry_domain.RegistryService
	metadataStore   registry_domain.MetadataStore
	blobStore       registry_domain.BlobStore
	blobStores      map[string]registry_domain.BlobStore
	dbConn          *sql.DB
	tempDir         string
	testPath        string
	server          *httptest.Server
	cleanup         func()
	storedArtefacts []string
}

func runTestCase(t *testing.T, tc testutil.TestCase) {
	base := testutil.NewBaseHarness(t, tc)

	err := base.LoadSpec()
	require.NoError(t, err, "Failed to load testspec.json")

	harness := setupPKCServingFixture(t, base, tc.Path)
	defer harness.cleanup()

	ctx := context.Background()

	if len(base.Spec.Assets) > 0 {
		for _, asset := range base.Spec.Assets {
			sourcePath := filepath.Join(tc.Path, asset.SourcePath)

			err := storeAsset(ctx, harness, asset.ExpectedArtefactID, sourcePath)
			require.NoError(t, err, "Failed to store asset: %s", asset.SourcePath)

			harness.storedArtefacts = append(harness.storedArtefacts, asset.ExpectedArtefactID)

			stored, err := harness.registryService.GetArtefact(ctx, asset.ExpectedArtefactID)
			require.NoError(t, err, "Artefact should be retrievable: %s", asset.ExpectedArtefactID)
			assert.Equal(t, asset.ExpectedArtefactID, stored.ID, "Stored artefact ID mismatch")
		}
	}

	if len(base.Spec.HTTPChecks) > 0 {
		for _, check := range base.Spec.HTTPChecks {
			func() {
				url := harness.server.URL + check.RequestPath
				response, err := http.Get(url)
				require.NoError(t, err, "HTTP request failed: %s", check.RequestPath)
				defer func() { _ = response.Body.Close() }()

				assert.Equal(t, check.ExpectedStatus, response.StatusCode,
					"Status code mismatch for %s", check.RequestPath)

				if check.ExpectedContentType != "" {
					assert.Equal(t, check.ExpectedContentType, response.Header.Get("Content-Type"),
						"Content-Type mismatch for %s", check.RequestPath)
				}
			}()
		}
	}
}

func setupPKCServingFixture(t *testing.T, base *testutil.BaseHarness, testPath string) *pkcServingHarness {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "pkc-serving-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "metadata.db")
	err = db_testutil.RunRegistryMigrations(dbPath)
	require.NoError(t, err)

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=10000", dbPath)
	dbConn, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	dbConn.SetMaxOpenConns(1)

	metaStore := registry_querier_adapter.NewDAL(dbConn)

	blobDir := filepath.Join(tempDir, "blobs")
	diskProvider, err := provider_disk.NewDiskProvider(provider_disk.Config{
		BaseDirectory: blobDir,
	})
	require.NoError(t, err)
	blobStore, err := registry_blob_adapter.NewBlobStoreAdapter(registry_blob_adapter.Config{
		Provider:   diskProvider,
		Repository: "",
	})
	require.NoError(t, err)
	blobStores := map[string]registry_domain.BlobStore{"disk": blobStore}

	service := registry_domain.NewRegistryService(metaStore, blobStores, nil, nil)

	router := chi.NewRouter()
	router.Get("/_piko/assets/*", func(w http.ResponseWriter, r *http.Request) {
		artefactID := chi.URLParam(r, "*")
		if artefactID == "" {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()

		artefact, err := service.GetArtefact(ctx, artefactID)
		if err != nil {
			if errors.Is(err, registry_domain.ErrArtefactNotFound) {

				artefact, err = service.FindArtefactByVariantStorageKey(ctx, artefactID)
				if err != nil {
					http.NotFound(w, r)
					return
				}
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		var sourceVariant *registry_dto.Variant
		for i := range artefact.ActualVariants {
			if artefact.ActualVariants[i].VariantID == "source" {
				sourceVariant = &artefact.ActualVariants[i]
				break
			}
		}
		if sourceVariant == nil {
			http.NotFound(w, r)
			return
		}

		reader, err := blobStore.Get(ctx, sourceVariant.StorageKey)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = reader.Close() }()

		contentType := sourceVariant.MimeType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)

		buffer := make([]byte, 32*1024)
		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				_, _ = w.Write(buffer[:n])
			}
			if err != nil {
				break
			}
		}
	})

	server := httptest.NewServer(router)

	cleanup := func() {
		server.Close()
		_ = dbConn.Close()
		_ = os.RemoveAll(tempDir)
	}

	return &pkcServingHarness{
		BaseHarness:     base,
		registryService: service,
		metadataStore:   metaStore,
		blobStore:       blobStore,
		blobStores:      blobStores,
		dbConn:          dbConn,
		tempDir:         tempDir,
		testPath:        testPath,
		server:          server,
		cleanup:         cleanup,
	}
}

func storeAsset(ctx context.Context, h *pkcServingHarness, artefactID, sourcePath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("reading source file: %w", err)
	}

	ext := filepath.Ext(sourcePath)
	mimeType := "application/octet-stream"
	switch ext {
	case ".svg":
		mimeType = "image/svg+xml"
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".html":
		mimeType = "text/html"
	case ".js":
		mimeType = "application/javascript"
	case ".css":
		mimeType = "text/css"
	case ".json":
		mimeType = "application/json"
	}

	desiredProfiles := []registry_dto.NamedProfile{
		{
			Name: "original",
			Profile: registry_dto.DesiredProfile{
				Priority:       registry_dto.PriorityNeed,
				CapabilityName: "identity",
				Params:         registry_dto.ProfileParams{},
				ResultingTags: registry_dto.TagsFromMap(map[string]string{
					"mime_type": mimeType,
				}),
			},
		},
	}

	_, err = h.registryService.UpsertArtefact(
		ctx,
		artefactID,
		sourcePath,
		bytes.NewReader(data),
		"disk",
		desiredProfiles,
	)
	if err != nil {
		return fmt.Errorf("upserting artefact: %w", err)
	}

	return nil
}
