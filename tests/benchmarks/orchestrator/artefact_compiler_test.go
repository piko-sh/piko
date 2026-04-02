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

// File: artefact_compiler_test.go (or wherever this setup function is located)

package orchestrator_test

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func setupCompilerBenchmark(b *testing.B) orchestrator_domain.TaskExecutor {
	b.Helper()

	fakeRegistry := NewFakeRegistry(nil, NewFakeBlobStore())
	fakeCapabilityService := &FakeCapabilityService{}
	executor := orchestrator_adapters.NewCompilerExecutor(fakeRegistry, fakeCapabilityService)

	artefactID := "source-artefact"
	desiredProfiles := []registry_dto.NamedProfile{
		{
			Name: "compiled",
			Profile: registry_dto.DesiredProfile{

				ResultingTags: registry_dto.TagsFromMap(map[string]string{
					"storageBackendId": "mem",
					"fileExtension":    ".js",
					"mimeType":         "application/javascript",
				}),
			},
		},
	}

	_, err := fakeRegistry.UpsertArtefact(
		context.Background(),
		artefactID,
		"source/the-key.txt",
		bytes.NewReader(nil),
		"mem",
		desiredProfiles,
	)
	if err != nil {
		b.Fatalf("Failed to set up source artefact for benchmark: %v", err)
	}

	return executor
}

var taskCounter atomic.Int64

func TestCompilerExecutor_BlobSizeTracking(t *testing.T) {

	fakeRegistry := NewFakeRegistry(nil, NewFakeBlobStore())
	fakeCapabilityService := &FakeCapabilityService{}
	executor := orchestrator_adapters.NewCompilerExecutor(fakeRegistry, fakeCapabilityService)

	sourceData := []byte("console.log('Hello, World!');")
	artefactID := "test-component.pkc"
	desiredProfiles := []registry_dto.NamedProfile{
		{
			Name: "compiled_js",
			Profile: registry_dto.DesiredProfile{
				ResultingTags: registry_dto.TagsFromMap(map[string]string{
					"storageBackendId": "mem",
					"fileExtension":    ".js",
					"mimeType":         "application/javascript",
				}),
			},
		},
	}

	ctx := context.Background()
	_, err := fakeRegistry.UpsertArtefact(
		ctx,
		artefactID,
		"source/test-component.pkc",
		bytes.NewReader(sourceData),
		"mem",
		desiredProfiles,
	)
	if err != nil {
		t.Fatalf("Failed to set up source artefact: %v", err)
	}

	payload := map[string]any{
		"artefactID":         artefactID,
		"sourceVariantID":    "source-variant",
		"desiredProfileName": "compiled_js",
		"capabilityToRun":    "compile",
		"taskID":             "test-task-123",
		"capabilityParams":   map[string]string{},
	}

	result, err := executor.Execute(ctx, payload)
	if err != nil {
		t.Fatalf("Executor.Execute failed: %v", err)
	}

	sizeBytes, ok := result["sizeBytes"].(int64)
	if !ok {
		t.Fatalf("Result missing sizeBytes field or wrong type: %v", result)
	}

	expectedSize := int64(len("COMPILED: ")) + int64(len(sourceData))

	if sizeBytes == 0 {
		t.Errorf("BUG REPRODUCED: Blob size is 0, expected %d bytes", expectedSize)
		t.Logf("This indicates the counter bug where finalSize reads from the wrong counter")
	} else if sizeBytes != expectedSize {
		t.Errorf("Blob size mismatch: got %d, want %d", sizeBytes, expectedSize)
	} else {
		t.Logf("SUCCESS: Blob size correctly tracked as %d bytes", sizeBytes)
	}

	artefact, err := fakeRegistry.GetArtefact(ctx, artefactID)
	if err != nil {
		t.Fatalf("Failed to get artefact after compilation: %v", err)
	}

	var compiledVariant *registry_dto.Variant
	for _, v := range artefact.ActualVariants {
		if v.VariantID == "compiled_js" {
			compiledVariant = &v
			break
		}
	}

	if compiledVariant == nil {
		t.Fatal("Compiled variant not found in artefact")
	}

	if compiledVariant.SizeBytes == 0 {
		t.Errorf("BUG REPRODUCED: Variant SizeBytes is 0, expected %d", expectedSize)
	} else if compiledVariant.SizeBytes != expectedSize {
		t.Errorf("Variant size mismatch: got %d, want %d", compiledVariant.SizeBytes, expectedSize)
	} else {
		t.Logf("SUCCESS: Variant correctly stored with size %d bytes", compiledVariant.SizeBytes)
	}
}

func TestCompilerExecutor_ChunkContentHashTracking(t *testing.T) {
	fakeRegistry := NewFakeRegistry(nil, NewFakeBlobStore())
	ctx := context.Background()

	chunk1Hash := "a1b2c3d4e5f6"
	chunk2Hash := "f6e5d4c3b2a1"

	variantWithChunks := registry_dto.Variant{
		VariantID:        "hls_playlist",
		StorageBackendID: "mem",
		StorageKey:       "video/playlist.m3u8",
		MimeType:         "application/x-mpegURL",
		SizeBytes:        1024,
		ContentHash:      "playlist_hash_123",
		Chunks: []registry_dto.VariantChunk{
			{
				ChunkID:          "chunk-0",
				StorageKey:       "video/segment-0.ts",
				StorageBackendID: "mem",
				SizeBytes:        512,
				ContentHash:      chunk1Hash,
				SequenceNumber:   0,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
			{
				ChunkID:          "chunk-1",
				StorageKey:       "video/segment-1.ts",
				StorageBackendID: "mem",
				SizeBytes:        512,
				ContentHash:      chunk2Hash,
				SequenceNumber:   1,
				MimeType:         "video/MP2T",
				CreatedAt:        time.Now(),
			},
		},
		CreatedAt: time.Now(),
		Status:    registry_dto.VariantStatusReady,
	}

	_, err := fakeRegistry.UpsertArtefact(
		ctx,
		"video.mp4",
		"source/video.mp4",
		bytes.NewReader([]byte("video data")),
		"mem",
		[]registry_dto.NamedProfile{},
	)
	require.NoError(t, err)

	artefact, err := fakeRegistry.AddVariant(ctx, "video.mp4", &variantWithChunks)
	require.NoError(t, err, "Failed to add variant with chunks")

	var storedVariant *registry_dto.Variant
	for _, v := range artefact.ActualVariants {
		if v.VariantID == "hls_playlist" {
			storedVariant = &v
			break
		}
	}

	require.NotNil(t, storedVariant, "Variant should be stored")
	require.Len(t, storedVariant.Chunks, 2, "Should have 2 chunks")

	assert.Equal(t, "chunk-0", storedVariant.Chunks[0].ChunkID)
	assert.Equal(t, chunk1Hash, storedVariant.Chunks[0].ContentHash,
		"Chunk 0 should have ContentHash preserved")
	assert.Equal(t, int64(512), storedVariant.Chunks[0].SizeBytes)

	assert.Equal(t, "chunk-1", storedVariant.Chunks[1].ChunkID)
	assert.Equal(t, chunk2Hash, storedVariant.Chunks[1].ContentHash,
		"Chunk 1 should have ContentHash preserved")
	assert.Equal(t, int64(512), storedVariant.Chunks[1].SizeBytes)

	t.Logf("SUCCESS: All chunks stored with correct ContentHash values")
}

func BenchmarkCompilerExecutor_Execute(b *testing.B) {
	executor := setupCompilerBenchmark(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {

			taskID := fmt.Sprintf("task-%d", taskCounter.Add(1))
			payload := map[string]any{
				"artefactID":         "source-artefact",
				"sourceVariantID":    "source-variant",
				"desiredProfileName": "compiled",
				"capabilityToRun":    "compile",
				"taskID":             taskID,
				"capabilityParams":   map[string]string{},
			}

			_, err := executor.Execute(ctx, payload)
			if err != nil {
				b.Fatalf("Executor.Execute failed: %v", err)
			}
		}
	})
}
