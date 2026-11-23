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

package persistence

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	d "piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/registry_blob_adapter"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestGeneratedPikoScenario(t *testing.T) {
	ctx := t.Context()
	disk, err := provider_disk.NewDiskProvider(provider_disk.Config{BaseDirectory: t.TempDir()})
	require.NoError(t, err)
	blob, err := registry_blob_adapter.NewBlobStoreAdapter(registry_blob_adapter.Config{Provider: disk})
	require.NoError(t, err)
	cfg := Config{RegistryCapacity: 40, OrchestratorCapacity: 8, Persistence: &PersistenceProviderConfig{Enabled: true, WALDir: t.TempDir(), SyncMode: wal_domain.SyncModeEveryWrite, SnapshotThreshold: 100}}
	open := func() (*Provider, d.RegistryService) {
		p := NewProvider(cfg)
		require.NoError(t, p.Connect(ctx))
		factory, err := p.RegistryDALFactory()
		require.NoError(t, err)
		store, err := factory.NewRegistryDAL()
		require.NoError(t, err)
		return p, d.NewRegistryService(store.(d.MetadataStore), map[string]d.BlobStore{"disk": blob}, nil, nil)
	}
	p, svc := open()
	defer func() { require.NoError(t, p.Close(ctx)) }()
	ds, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_DRAWS"))
	require.NoError(t, err)
	keys := []string{"a.css", "b.css", "c.css"}
	want := map[string]string{}
	steps := append([]byte{0, 5, 2, 1, 3, 4, 2}, ds...)
	steps = append(steps, 3)
	for i, b := range steps {
		k := keys[int(b/5)%len(keys)]
		switch b % 5 {
		case 0, 1:
			content := fmt.Sprintf("/* variant %d */", b/15%5)
			if b%5 == 1 {
				content = fmt.Sprintf("/* change %d */", i)
			}
			_, err := svc.UpsertArtefact(ctx, k, k, strings.NewReader(content), "disk", nil)
			require.NoError(t, err)
			want[k] = content
		case 2:
			require.NoError(t, p.Checkpoint(ctx))
		case 3:
			require.NoError(t, p.Close(ctx))
			p, svc = open()
		case 4:
			if _, ok := want[k]; ok {
				require.NoError(t, svc.DeleteArtefact(ctx, k))
				delete(want, k)
			}
		}
		blobs := map[string]string{}
		got := map[string]string{}
		for _, key := range keys {
			a, err := svc.GetArtefact(ctx, key)
			if errors.Is(err, d.ErrArtefactNotFound) {
				continue
			}
			require.NoError(t, err)
			for _, v := range a.ActualVariants {
				if v.VariantID != "source" {
					continue
				}
				require.Equal(t, "text/css; charset=utf-8", v.MimeType, "PIKO_CHECK asset-mime: %s", key)
				r, err := svc.GetVariantData(ctx, &v)
				require.NoError(t, err)
				bytes, err := io.ReadAll(io.LimitReader(r, 1024))
				require.NoError(t, err)
				require.NoError(t, r.Close())
				value := string(bytes)
				got[key] = value
				if other, ok := blobs[value]; ok {
					require.Equal(t, other, v.StorageKey, "PIKO_CHECK asset-dedup: %s", key)
				}
				blobs[value] = v.StorageKey
			}
		}
		require.Equal(t, want, got, "PIKO_CHECK asset-history: step %d", i)
	}
}
