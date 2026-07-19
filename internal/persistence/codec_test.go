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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestStringKeyCodec_RoundTrip(t *testing.T) {
	codec := StringKeyCodec{}

	testCases := []struct {
		name string
		key  string
	}{
		{name: "simple key", key: "artefact-123"},
		{name: "empty key", key: ""},
		{name: "uuid key", key: "550e8400-e29b-41d4-a716-446655440000"},
		{name: "key with special chars", key: "path/to/file.jpg"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := codec.EncodeKey(tc.key)
			require.NoError(t, err, "EncodeKey failed")

			decoded, err := codec.DecodeKey(encoded)
			require.NoError(t, err, "DecodeKey failed")

			assert.Equal(t, tc.key, decoded, "round-trip failed: got %q, want %q", decoded, tc.key)
		})
	}
}

func TestArtefactMetaCodec_RoundTrip(t *testing.T) {
	codec := ArtefactMetaCodec{}

	now := time.Now().Truncate(time.Millisecond)

	testCases := []struct {
		artefact *registry_dto.ArtefactMeta
		name     string
	}{
		{
			name: "minimal artefact",
			artefact: &registry_dto.ArtefactMeta{
				ID:         "artefact-1",
				SourcePath: "/images/test.jpg",
				Status:     registry_dto.VariantStatusReady,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
		{
			name: "artefact with variants",
			artefact: &registry_dto.ArtefactMeta{
				ID:         "artefact-2",
				SourcePath: "/images/photo.png",
				Status:     registry_dto.VariantStatusReady,
				CreatedAt:  now,
				UpdatedAt:  now,
				ActualVariants: []registry_dto.Variant{
					{
						VariantID:        "variant-1",
						StorageKey:       "storage/variant-1.webp",
						StorageBackendID: "default",
						MimeType:         "image/webp",
						Status:           registry_dto.VariantStatusReady,
						SizeBytes:        12345,
						CreatedAt:        now,
					},
				},
			},
		},
		{
			name: "artefact with profiles",
			artefact: &registry_dto.ArtefactMeta{
				ID:         "artefact-3",
				SourcePath: "/videos/clip.mp4",
				Status:     registry_dto.VariantStatusPending,
				CreatedAt:  now,
				UpdatedAt:  now,
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumbnail",
						Profile: registry_dto.DesiredProfile{
							Priority:       registry_dto.PriorityNeed,
							CapabilityName: "image.resize",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := codec.EncodeValue(tc.artefact)
			require.NoError(t, err, "EncodeValue failed")

			decoded, err := codec.DecodeValue(encoded)
			require.NoError(t, err, "DecodeValue failed")

			assert.Equal(t, tc.artefact.ID, decoded.ID, "ID mismatch: got %q, want %q", decoded.ID, tc.artefact.ID)
			assert.Equal(t, tc.artefact.SourcePath, decoded.SourcePath,
				"SourcePath mismatch: got %q, want %q", decoded.SourcePath, tc.artefact.SourcePath)
			assert.Equal(t, tc.artefact.Status, decoded.Status,
				"Status mismatch: got %v, want %v", decoded.Status, tc.artefact.Status)
			assert.Len(t, decoded.ActualVariants, len(tc.artefact.ActualVariants),
				"ActualVariants length mismatch: got %d, want %d",
				len(decoded.ActualVariants), len(tc.artefact.ActualVariants))
			assert.Len(t, decoded.DesiredProfiles, len(tc.artefact.DesiredProfiles),
				"DesiredProfiles length mismatch: got %d, want %d",
				len(decoded.DesiredProfiles), len(tc.artefact.DesiredProfiles))
		})
	}
}

func TestArtefactMetaCodec_NilArtefact(t *testing.T) {
	codec := ArtefactMetaCodec{}

	encoded, err := codec.EncodeValue(nil)
	require.NoError(t, err, "EncodeValue(nil) failed")

	decoded, err := codec.DecodeValue(encoded)
	require.NoError(t, err, "DecodeValue failed")

	assert.NotNil(t, decoded, "decoded should not be nil for null JSON")
}

func TestTaskCodec_RoundTrip(t *testing.T) {
	codec := TaskCodec{}

	now := time.Now().Truncate(time.Millisecond)

	testCases := []struct {
		task *orchestrator_domain.Task
		name string
	}{
		{
			name: "minimal task",
			task: &orchestrator_domain.Task{
				ID:         "task-1",
				WorkflowID: "workflow-1",
				Executor:   "image.process",
				Status:     orchestrator_domain.StatusPending,
				CreatedAt:  now,
				UpdatedAt:  now,
				ExecuteAt:  now,
			},
		},
		{
			name: "task with payload",
			task: &orchestrator_domain.Task{
				ID:         "task-2",
				WorkflowID: "workflow-2",
				Executor:   "video.transcode",
				Status:     orchestrator_domain.StatusProcessing,
				CreatedAt:  now,
				UpdatedAt:  now,
				ExecuteAt:  now,
				Payload: map[string]any{
					"source":  "/videos/input.mp4",
					"format":  "webm",
					"quality": 80,
				},
			},
		},
		{
			name: "task with config",
			task: &orchestrator_domain.Task{
				ID:               "task-3",
				WorkflowID:       "workflow-3",
				Executor:         "email.send",
				Status:           orchestrator_domain.StatusRetrying,
				DeduplicationKey: "email-user-123",
				Attempt:          2,
				LastError:        "connection timeout",
				CreatedAt:        now,
				UpdatedAt:        now,
				ExecuteAt:        now,
				Config: orchestrator_domain.TaskConfig{
					Priority:   orchestrator_domain.PriorityHigh,
					MaxRetries: 5,
					Timeout:    30 * time.Second,
				},
			},
		},
		{
			name: "scheduled task",
			task: &orchestrator_domain.Task{
				ID:                 "task-4",
				WorkflowID:         "workflow-4",
				Executor:           "report.generate",
				Status:             orchestrator_domain.StatusScheduled,
				CreatedAt:          now,
				UpdatedAt:          now,
				ExecuteAt:          now.Add(time.Hour),
				ScheduledExecuteAt: now.Add(time.Hour),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := codec.EncodeValue(tc.task)
			require.NoError(t, err, "EncodeValue failed")

			decoded, err := codec.DecodeValue(encoded)
			require.NoError(t, err, "DecodeValue failed")

			assert.Equal(t, tc.task.ID, decoded.ID, "ID mismatch: got %q, want %q", decoded.ID, tc.task.ID)
			assert.Equal(t, tc.task.WorkflowID, decoded.WorkflowID,
				"WorkflowID mismatch: got %q, want %q", decoded.WorkflowID, tc.task.WorkflowID)
			assert.Equal(t, tc.task.Executor, decoded.Executor,
				"Executor mismatch: got %q, want %q", decoded.Executor, tc.task.Executor)
			assert.Equal(t, tc.task.Status, decoded.Status,
				"Status mismatch: got %v, want %v", decoded.Status, tc.task.Status)
			assert.Equal(t, tc.task.Attempt, decoded.Attempt,
				"Attempt mismatch: got %d, want %d", decoded.Attempt, tc.task.Attempt)
			assert.Equal(t, tc.task.DeduplicationKey, decoded.DeduplicationKey,
				"DeduplicationKey mismatch: got %q, want %q",
				decoded.DeduplicationKey, tc.task.DeduplicationKey)
			assert.Equal(t, tc.task.Config.Priority, decoded.Config.Priority,
				"Config.Priority mismatch: got %v, want %v",
				decoded.Config.Priority, tc.task.Config.Priority)
		})
	}
}

func TestTaskCodec_NilTask(t *testing.T) {
	codec := TaskCodec{}

	encoded, err := codec.EncodeValue(nil)
	require.NoError(t, err, "EncodeValue(nil) failed")

	decoded, err := codec.DecodeValue(encoded)
	require.NoError(t, err, "DecodeValue failed")

	assert.NotNil(t, decoded, "decoded should not be nil for null JSON")
}

func TestTaskCodec_InvalidJSON(t *testing.T) {
	codec := TaskCodec{}

	_, err := codec.DecodeValue([]byte("not valid json"))
	assert.Error(t, err, "DecodeValue should fail for invalid JSON")
}

func TestArtefactMetaCodec_InvalidJSON(t *testing.T) {
	codec := ArtefactMetaCodec{}

	_, err := codec.DecodeValue([]byte("not valid json"))
	assert.Error(t, err, "DecodeValue should fail for invalid JSON")
}
