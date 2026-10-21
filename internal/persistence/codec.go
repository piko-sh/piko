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
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

const (
	// taskCodecVersion2 identifies binary task records, distinguishing them from legacy JSON
	// records.
	taskCodecVersion2 byte = 0x02

	// taskVarintEstimate is the number of varints reserved when allocating a task encoding
	// buffer.
	taskVarintEstimate = 12

	// rerunVarintEstimate is the number of further varints room is reserved for when a task
	// carries a rerun request.
	rerunVarintEstimate = 4
)

const (
	// taskFlagFatal records the fatal-task flag.
	taskFlagFatal uint64 = 1 << iota

	// taskFlagBackground records the background-task flag.
	taskFlagBackground

	// taskFlagExecuteAt marks an encoded execution time.
	taskFlagExecuteAt

	// taskFlagCreatedAt marks an encoded creation time.
	taskFlagCreatedAt

	// taskFlagUpdatedAt marks an encoded update time.
	taskFlagUpdatedAt

	// taskFlagScheduledAt marks an encoded scheduled execution time.
	taskFlagScheduledAt

	// taskFlagPublishedAt marks an encoded publication time.
	taskFlagPublishedAt

	// taskFlagPayload marks an encoded task payload.
	taskFlagPayload

	// taskFlagResult marks an encoded task result.
	taskFlagResult

	// taskFlagRerun marks an encoded rerun request.
	taskFlagRerun

	// taskFlagRerunPayload marks an encoded rerun payload.
	taskFlagRerunPayload
)

var (
	// errTaskTruncated is returned when an encoded task ends before all of its fields.
	errTaskTruncated = errors.New("decoding task: encoding is truncated")

	_ wal_domain.KeyCodec[string] = StringKeyCodec{}

	_ wal_domain.ValueCodec[*registry_dto.ArtefactMeta] = ArtefactMetaCodec{}

	_ wal_domain.ValueCodec[*orchestrator_domain.Task] = TaskCodec{}
)

// StringKeyCodec implements wal_domain.KeyCodec for string keys. Uses simple byte
// conversion for minimal overhead.
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

// ArtefactMetaCodec implements wal_domain.ValueCodec for *registry_dto.ArtefactMeta. It
// uses Sonic for fast JSON marshalling, consistent with the DTO's internal serialisation.
type ArtefactMetaCodec struct{}

// EncodeValue serialises an ArtefactMeta to JSON bytes.
//
// Takes v (*registry_dto.ArtefactMeta) which is the artefact to encode.
//
// Returns []byte which is the JSON-encoded artefact.
// Returns error when marshalling fails.
func (ArtefactMetaCodec) EncodeValue(v *registry_dto.ArtefactMeta) ([]byte, error) {
	return json.Marshal(v)
}

// DecodeValue deserialises JSON bytes to an ArtefactMeta.
//
// Takes data ([]byte) which is the JSON-encoded artefact.
//
// Returns *registry_dto.ArtefactMeta which is the decoded artefact.
// Returns error when unmarshalling fails.
func (ArtefactMetaCodec) DecodeValue(data []byte) (*registry_dto.ArtefactMeta, error) {
	var v registry_dto.ArtefactMeta
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("decoding artefact meta: %w", err)
	}
	return &v, nil
}

// TaskCodec encodes tasks in binary with JSON payload and result maps.
//
// It also decodes legacy JSON task records.
type TaskCodec struct{}

// EncodeValue serialises a Task in the binary form.
//
// Takes v (*orchestrator_domain.Task) which is the task to encode; nil encodes an empty
// task.
//
// Returns []byte which is the encoded task.
// Returns error when the payload or result cannot be marshalled.
func (TaskCodec) EncodeValue(v *orchestrator_domain.Task) ([]byte, error) {
	if v == nil {
		v = &orchestrator_domain.Task{}
	}
	payload, result, rerunPayload, err := marshalTaskMaps(v)
	if err != nil {
		return nil, err
	}
	flags := taskFlags(v)

	size := 1 + binary.MaxVarintLen64*taskVarintEstimate + len(v.ID) + len(v.WorkflowID) + len(v.Executor) + len(v.Status) +
		len(v.LastError) + len(v.DeduplicationKey) + len(v.BuildTag) + len(v.OwnerInstance) +
		len(payload) + len(result) + len(rerunPayload)
	if v.Rerun != nil {
		size += len(v.Rerun.ID) + binary.MaxVarintLen64*rerunVarintEstimate
	}
	buffer := make([]byte, 0, size)
	buffer = append(buffer, taskCodecVersion2)
	buffer = binary.AppendUvarint(buffer, flags)
	for _, field := range [...]string{v.ID, v.WorkflowID, v.Executor, string(v.Status), v.LastError,
		v.DeduplicationKey, v.BuildTag, v.OwnerInstance} {
		buffer = appendString(buffer, field)
	}
	buffer = binary.AppendVarint(buffer, int64(v.Attempt))
	buffer = appendTaskConfig(buffer, v.Config)
	for _, at := range [...]time.Time{v.ExecuteAt, v.CreatedAt, v.UpdatedAt, v.ScheduledExecuteAt, v.LastPublishedAt} {
		if !at.IsZero() {
			buffer = binary.AppendVarint(buffer, at.UnixNano())
		}
	}
	if flags&taskFlagPayload != 0 {
		buffer = appendBytes(buffer, payload)
	}
	if flags&taskFlagResult != 0 {
		buffer = appendBytes(buffer, result)
	}
	if v.Rerun != nil {
		buffer = appendString(buffer, v.Rerun.ID)
		buffer = appendTaskConfig(buffer, v.Rerun.Config)
		if flags&taskFlagRerunPayload != 0 {
			buffer = appendBytes(buffer, rerunPayload)
		}
	}
	return buffer, nil
}

// marshalTaskMaps marshals the maps a task embeds in its encoding as JSON.
//
// Takes v (*orchestrator_domain.Task) which is the task being encoded.
//
// Returns payload ([]byte) which is the encoded payload, or nil when the task has none.
// Returns result ([]byte) which is the encoded result, or nil when the task has none.
// Returns rerunPayload ([]byte) which is the encoded rerun payload, or nil when the task
// has none.
// Returns error when a map cannot be marshalled.
func marshalTaskMaps(v *orchestrator_domain.Task) (payload []byte, result []byte, rerunPayload []byte, err error) {
	if v.Payload != nil {
		if payload, err = json.Marshal(v.Payload); err != nil {
			return nil, nil, nil, fmt.Errorf("encoding task %s payload: %w", v.ID, err)
		}
	}
	if v.Result != nil {
		if result, err = json.Marshal(v.Result); err != nil {
			return nil, nil, nil, fmt.Errorf("encoding task %s result: %w", v.ID, err)
		}
	}
	if v.Rerun != nil && v.Rerun.Payload != nil {
		if rerunPayload, err = json.Marshal(v.Rerun.Payload); err != nil {
			return nil, nil, nil, fmt.Errorf("encoding task %s rerun payload: %w", v.ID, err)
		}
	}
	return payload, result, rerunPayload, nil
}

// taskFlags returns the flag word recording which optional parts of a task its encoding
// holds.
//
// Takes v (*orchestrator_domain.Task) which is the task being encoded.
//
// Returns uint64 which is the flag word.
func taskFlags(v *orchestrator_domain.Task) uint64 {
	return boolFlag(v.Payload != nil, taskFlagPayload) | boolFlag(v.Result != nil, taskFlagResult) |
		boolFlag(v.Rerun != nil, taskFlagRerun) |
		boolFlag(v.Rerun != nil && v.Rerun.Payload != nil, taskFlagRerunPayload) |
		boolFlag(v.IsFatal, taskFlagFatal) | boolFlag(v.Background, taskFlagBackground) |
		timeFlag(v.ExecuteAt, taskFlagExecuteAt) | timeFlag(v.CreatedAt, taskFlagCreatedAt) |
		timeFlag(v.UpdatedAt, taskFlagUpdatedAt) | timeFlag(v.ScheduledExecuteAt, taskFlagScheduledAt) |
		timeFlag(v.LastPublishedAt, taskFlagPublishedAt)
}

// DecodeValue deserialises a Task written in the binary form, or as JSON by an earlier
// release.
//
// Takes data ([]byte) which is the encoded task.
//
// Returns *orchestrator_domain.Task which is the decoded task.
// Returns error when the encoding is malformed.
func (TaskCodec) DecodeValue(data []byte) (*orchestrator_domain.Task, error) {
	if len(data) == 0 || data[0] != taskCodecVersion2 {
		var v orchestrator_domain.Task
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("decoding task: %w", err)
		}
		return &v, nil
	}
	return decodeTaskV2(data[1:])
}

// decodeTaskV2 decodes the binary form after its version byte.
//
// Takes data ([]byte) which is the encoding after the version byte.
//
// Returns *orchestrator_domain.Task which is the decoded task.
// Returns error when the encoding is malformed.
func decodeTaskV2(data []byte) (*orchestrator_domain.Task, error) {
	r := taskReader{data: data, err: nil}
	flags := r.uvarint()
	task := &orchestrator_domain.Task{}
	task.ID = r.string()
	task.WorkflowID = r.string()
	task.Executor = r.string()
	task.Status = orchestrator_domain.TaskStatus(r.string())
	task.LastError = r.string()
	task.DeduplicationKey = r.string()
	task.BuildTag = r.string()
	task.OwnerInstance = r.string()
	task.Attempt = int(r.varint())
	task.Config = r.taskConfig()
	task.IsFatal = flags&taskFlagFatal != 0
	task.Background = flags&taskFlagBackground != 0
	for _, field := range [...]struct {
		at   *time.Time
		flag uint64
	}{
		{at: &task.ExecuteAt, flag: taskFlagExecuteAt},
		{at: &task.CreatedAt, flag: taskFlagCreatedAt},
		{at: &task.UpdatedAt, flag: taskFlagUpdatedAt},
		{at: &task.ScheduledExecuteAt, flag: taskFlagScheduledAt},
		{at: &task.LastPublishedAt, flag: taskFlagPublishedAt},
	} {
		if flags&field.flag != 0 {
			*field.at = time.Unix(0, r.varint()).UTC()
		}
	}
	if flags&taskFlagPayload != 0 {
		r.jsonMap(&task.Payload)
	}
	if flags&taskFlagResult != 0 {
		r.jsonMap(&task.Result)
	}
	if flags&taskFlagRerun != 0 {
		rerun := &orchestrator_domain.RerunRequest{ID: r.string(), Config: r.taskConfig(), Payload: nil}
		if flags&taskFlagRerunPayload != 0 {
			r.jsonMap(&rerun.Payload)
		}
		task.Rerun = rerun
	}
	if r.err != nil {
		return nil, fmt.Errorf("decoding task %q: %w", task.ID, r.err)
	}
	return task, nil
}

// taskReader reads the fields of a binary task, remembering the first error.
type taskReader struct {
	// err is the first decoding error, after which every read returns zero.
	err error

	// data is the unread remainder of the encoding.
	data []byte
}

// uvarint reads an unsigned varint.
//
// Returns uint64 which is the value, or zero after an error.
func (r *taskReader) uvarint() uint64 {
	if r.err != nil {
		return 0
	}
	value, n := binary.Uvarint(r.data)
	if n <= 0 {
		r.err = errTaskTruncated
		return 0
	}
	r.data = r.data[n:]
	return value
}

// varint reads a signed varint.
//
// Returns int64 which is the value, or zero after an error.
func (r *taskReader) varint() int64 {
	if r.err != nil {
		return 0
	}
	value, n := binary.Varint(r.data)
	if n <= 0 {
		r.err = errTaskTruncated
		return 0
	}
	r.data = r.data[n:]
	return value
}

// bytes reads a length-prefixed byte slice, which aliases the encoding.
//
// Returns []byte which is the slice, or nil after an error.
func (r *taskReader) bytes() []byte {
	length := r.uvarint()
	if r.err != nil {
		return nil
	}
	if length > uint64(len(r.data)) {
		r.err = errTaskTruncated
		return nil
	}
	value := r.data[:length]
	r.data = r.data[length:]
	return value
}

// string reads a length-prefixed string.
//
// Returns string which is the value, or empty after an error.
func (r *taskReader) string() string {
	return string(r.bytes())
}

// taskConfig reads a task's execution settings.
//
// Returns orchestrator_domain.TaskConfig which is the settings.
func (r *taskReader) taskConfig() orchestrator_domain.TaskConfig {
	return orchestrator_domain.TaskConfig{
		Timeout:    time.Duration(r.varint()),
		MaxRetries: int(r.varint()),
		Priority:   orchestrator_domain.TaskPriority(r.varint()),
	}
}

// jsonMap reads a length-prefixed JSON object into target.
//
// Takes target (*map[string]any) which receives the map.
func (r *taskReader) jsonMap(target *map[string]any) {
	encoded := r.bytes()
	if r.err != nil {
		return
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		r.err = err
	}
}

// appendTaskConfig appends a task's execution settings.
//
// Takes buffer ([]byte) which is the encoding so far.
// Takes config (orchestrator_domain.TaskConfig) which is the settings.
//
// Returns []byte which is the extended encoding.
func appendTaskConfig(buffer []byte, config orchestrator_domain.TaskConfig) []byte {
	buffer = binary.AppendVarint(buffer, int64(config.Timeout))
	buffer = binary.AppendVarint(buffer, int64(config.MaxRetries))
	return binary.AppendVarint(buffer, int64(config.Priority))
}

// appendString appends a length-prefixed string.
//
// Takes buffer ([]byte) which is the encoding so far.
// Takes value (string) which is the string.
//
// Returns []byte which is the extended encoding.
func appendString(buffer []byte, value string) []byte {
	buffer = binary.AppendUvarint(buffer, uint64(len(value)))
	return append(buffer, value...)
}

// appendBytes appends a length-prefixed byte slice.
//
// Takes buffer ([]byte) which is the encoding so far.
// Takes value ([]byte) which is the slice.
//
// Returns []byte which is the extended encoding.
func appendBytes(buffer, value []byte) []byte {
	buffer = binary.AppendUvarint(buffer, uint64(len(value)))
	return append(buffer, value...)
}

// boolFlag returns flag when set is true.
//
// Takes set (bool) which selects the flag.
// Takes flag (uint64) which is the flag bit.
//
// Returns uint64 which is flag or zero.
func boolFlag(set bool, flag uint64) uint64 {
	if set {
		return flag
	}
	return 0
}

// timeFlag returns flag when at is not the zero time.
//
// Takes at (time.Time) which is the time.
// Takes flag (uint64) which is the flag bit.
//
// Returns uint64 which is flag or zero.
func timeFlag(at time.Time, flag uint64) uint64 {
	return boolFlag(!at.IsZero(), flag)
}
