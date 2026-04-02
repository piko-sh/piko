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
	"encoding/json"
	"fmt"

	pikojson "piko.sh/piko/internal/json"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

var (
	_ wal_domain.KeyCodec[string] = StringKeyCodec{}

	_ wal_domain.ValueCodec[*registry_dto.ArtefactMeta] = ArtefactMetaCodec{}

	_ wal_domain.ValueCodec[*orchestrator_domain.Task] = TaskCodec{}
)

// StringKeyCodec implements wal_domain.KeyCodec for string keys.
// Uses simple byte conversion for minimal overhead.
type StringKeyCodec struct{}

// EncodeKey converts a string key to bytes.
//
// Takes key (string) which is the key to encode.
//
// Returns []byte which is the encoded key.
// Returns error which is always nil.
func (StringKeyCodec) EncodeKey(key string) ([]byte, error) {
	return []byte(key), nil
}

// DecodeKey converts bytes back to a string key.
//
// Takes data ([]byte) which is the encoded key.
//
// Returns string which is the decoded key.
// Returns error which is always nil.
func (StringKeyCodec) DecodeKey(data []byte) (string, error) {
	return string(data), nil
}

// ArtefactMetaCodec implements wal_domain.ValueCodec for
// *registry_dto.ArtefactMeta. It uses Sonic for fast JSON marshalling,
// consistent with the DTO's internal serialisation.
type ArtefactMetaCodec struct{}

// EncodeValue serialises an ArtefactMeta to JSON bytes.
//
// Takes v (*registry_dto.ArtefactMeta) which is the artefact to encode.
//
// Returns []byte which is the JSON-encoded artefact.
// Returns error when marshalling fails.
func (ArtefactMetaCodec) EncodeValue(v *registry_dto.ArtefactMeta) ([]byte, error) {
	return pikojson.Marshal(v)
}

// DecodeValue deserialises JSON bytes to an ArtefactMeta.
//
// Takes data ([]byte) which is the JSON-encoded artefact.
//
// Returns *registry_dto.ArtefactMeta which is the decoded artefact.
// Returns error when unmarshalling fails.
func (ArtefactMetaCodec) DecodeValue(data []byte) (*registry_dto.ArtefactMeta, error) {
	var v registry_dto.ArtefactMeta
	if err := pikojson.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("decoding artefact meta: %w", err)
	}
	return &v, nil
}

// TaskCodec implements wal_domain.ValueCodec for *orchestrator_domain.Task.
// Uses standard JSON marshalling.
type TaskCodec struct{}

// EncodeValue serialises a Task to JSON bytes.
//
// Takes v (*orchestrator_domain.Task) which is the task to encode.
//
// Returns []byte which is the JSON-encoded task.
// Returns error when marshalling fails.
func (TaskCodec) EncodeValue(v *orchestrator_domain.Task) ([]byte, error) {
	return json.Marshal(v)
}

// DecodeValue deserialises JSON bytes to a Task.
//
// Takes data ([]byte) which is the JSON-encoded task.
//
// Returns *orchestrator_domain.Task which is the decoded task.
// Returns error when unmarshalling fails.
func (TaskCodec) DecodeValue(data []byte) (*orchestrator_domain.Task, error) {
	var v orchestrator_domain.Task
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("decoding task: %w", err)
	}
	return &v, nil
}
