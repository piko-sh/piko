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

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"time"

	"piko.sh/fuzzkit/tape"
	"piko.sh/json"
	task "piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/persistence"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/storage/storage_adapters/provider_memory"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/encoding"
	"piko.sh/piko/wdk/maths"
	"piko.sh/safeconv"
)

const (
	// attemptTwoByteVarintLimit is the smallest count that needs a three-byte varint.
	attemptTwoByteVarintLimit = 1 << 14

	// taskPriorityCount is the number of drawn task priorities.
	taskPriorityCount = 3

	// maxPersistedAttempt bounds the drawn persisted attempt count.
	maxPersistedAttempt = 300

	// maxPayloadCount bounds the drawn payload counter.
	maxPayloadCount = 64

	// maxCreatedSeconds bounds the drawn creation time in Unix seconds.
	maxCreatedSeconds = 100000

	// maxExtraArtefacts bounds the artefacts drawn beyond the swapped pair.
	maxExtraArtefacts = 5

	// storageCapacityBytes bounds the memory storage provider.
	storageCapacityBytes = 65536

	// maxStoredObjectBytes bounds the drawn stored object length beyond one byte.
	maxStoredObjectBytes = 48

	// decimalBase is the radix used to spell integers.
	decimalBase = 10
)

// checkTaskReference checks task registration and reference identity.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkTaskReference(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	attempts := []int{0, 1, math.MaxInt8, math.MaxInt8 + 1, math.MaxUint8, math.MaxUint8 + 1, attemptTwoByteVarintLimit}
	var original task.Task
	original.ID = label(choices)
	original.Attempt = pick(choices, attempts)
	original.Background = choices.Bool(1, 2)
	original.Config.Priority = task.TaskPriority(drawInt(choices, taskPriorityCount))
	want := task.TaskRef{ID: original.ID, Priority: original.Config.Priority, Attempt: original.Attempt, Flags: 0}
	if original.Background {
		want.Flags = task.TaskRefBackground
	}
	encoded := task.AppendTaskRef([]byte{'T'}, task.RefOf(&original))
	got, err := task.DecodeTaskRef(encoded[1:])
	if err != nil {
		return fmt.Errorf("decoding task reference: %w", err)
	}
	if encoded[0] != 'T' || got != want {
		return property("reference", "decoded task ref %+v, want %+v", got, want)
	}
	return nil
}

// checkPersistence checks persistence operations against independent state.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkPersistence(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	id := label(choices)
	var want task.Task
	want.ID = id
	want.Executor = "fuzz.Exec"
	want.WorkflowID = "job-" + id
	want.Attempt = drawInt(choices, maxPersistedAttempt)
	want.Status = task.StatusRetrying
	want.Background = choices.Bool(1, 2)
	want.Payload = map[string]any{"label": id, "count": float64(choices.Uint(maxPayloadCount))}
	want.CreatedAt = time.Unix(drawInt64(choices, maxCreatedSeconds), 0).UTC()
	codec := persistence.TaskCodec{}
	wire, err := codec.EncodeValue(&want)
	if err != nil {
		return fmt.Errorf("encoding task: %w", err)
	}
	legacy, err := json.Marshal(want)
	if err != nil {
		return fmt.Errorf("encoding legacy task JSON: %w", err)
	}
	for _, encoded := range [][]byte{wire, legacy} {
		got, err := codec.DecodeValue(encoded)
		if err != nil {
			return fmt.Errorf("decoding task: %w", err)
		}
		if !got.CreatedAt.Equal(want.CreatedAt) {
			return property("timestamp", "persisted creation time changed")
		}
		got.CreatedAt = want.CreatedAt
		if !reflect.DeepEqual(*got, want) {
			return property("task-state", "task state changed across persisted representation: got %+v want %+v", got, want)
		}
	}
	return nil
}

// checkRegistry checks registry operations and identity.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkRegistry(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	count := 2 + drawInt(choices, maxExtraArtefacts)
	metadata := make([]*registry_dto.ArtefactMeta, 0, count)
	for index := range count {
		id := fmt.Sprintf("a/%d/%d", index, choices.Uint(byteRange))
		var variant registry_dto.Variant
		variant.VariantID = id
		variant.ContentHash = label(choices)
		var artefact registry_dto.ArtefactMeta
		artefact.ID = id
		artefact.ActualVariants = []registry_dto.Variant{variant}
		metadata = append(metadata, &artefact)
	}
	want := registry_domain.SeedDigest(metadata)
	metadata[0], metadata[1] = metadata[1], metadata[0]
	if registry_domain.SeedDigest(metadata) != want {
		return property("order", "release digest depends on item order")
	}
	metadata[0].ActualVariants[0].ContentHash += "!"
	if registry_domain.SeedDigest(metadata) == want {
		return property("content-change", "release digest ignored changed content")
	}
	return nil
}

// checkStorage checks bounded storage histories against a model.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkStorage(ctx context.Context, choices *tape.Tape, _ *casefile.Case) (retErr error) {
	provider, err := provider_memory.New(provider_memory.Config{MaxBytes: storageCapacityBytes})
	if err != nil {
		return fmt.Errorf("creating memory storage: %w", err)
	}
	defer func() {
		if closeErr := provider.Close(ctx); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("closing memory storage: %w", closeErr)
		}
	}()
	data := make([]byte, 1+drawInt(choices, maxStoredObjectBytes))
	for index := range data {
		data[index] = drawByte(choices)
	}
	var object storage_dto.GetParams
	object.Key = "key"
	object.Repository = "case"
	var put storage_dto.PutParams
	put.Repository = object.Repository
	put.Key = object.Key
	put.ContentType = "application/octet-stream"
	put.Reader = bytes.NewReader(data)
	put.Size = int64(len(data))
	if err := provider.Put(ctx, &put); err != nil {
		return fmt.Errorf("storing object: %w", err)
	}
	start := choices.Uint(uint64(len(data)))
	end := start + choices.Uint(uint64(len(data))-start)
	ranged := object
	ranged.ByteRange = &storage_dto.ByteRange{Start: safeconv.Uint64ToInt64(start), End: safeconv.Uint64ToInt64(end)}
	reader, err := provider.Get(ctx, ranged)
	if err != nil {
		return fmt.Errorf("reading byte range: %w", err)
	}
	got, err := readLimited(reader, int64(len(data)))
	closeErr := reader.Close()
	if errors.Is(err, errReadLimitExceeded) {
		return property("range", "inclusive byte range [%d,%d]: %w", start, end, err)
	}
	if err != nil {
		return fmt.Errorf("reading byte range: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("closing byte range: %w", closeErr)
	}
	if !bytes.Equal(got, data[start:end+1]) {
		return property("range", "inclusive byte range [%d,%d] returned %x", start, end, got)
	}
	if err := provider.Remove(ctx, object); err != nil {
		return fmt.Errorf("removing object: %w", err)
	}
	exists, err := provider.Exists(ctx, object)
	if err != nil {
		return fmt.Errorf("checking removed object: %w", err)
	}
	if exists {
		return property("remove", "removed object remained visible")
	}
	return nil
}

// checkUtilities checks selected utility operations against independent results.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkUtilities(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	multiplicand := safeconv.Uint64ToInt64Reinterpret(choices.Uint(0))
	multiplier := safeconv.Uint64ToInt64Reinterpret(choices.Uint(0))
	product := big.NewInt(0).Mul(big.NewInt(multiplicand), big.NewInt(multiplier)).String()
	got, err := maths.NewBigIntFromInt(multiplicand).Multiply(maths.NewBigIntFromInt(multiplier)).String()
	if err != nil {
		return fmt.Errorf("formatting integer product: %w", err)
	}
	if got != product {
		return property("bigint", "integer product %s, want %s", got, product)
	}
	payload := []byte(strconv.FormatInt(multiplicand, decimalBase) + label(choices))
	wire := encoding.EncodeBytesBase58(payload)
	decoded, err := encoding.DecodeBytesBase58(wire)
	if err != nil {
		return fmt.Errorf("decoding base58 payload: %w", err)
	}
	if !bytes.Equal(payload, decoded) {
		return property("base58", "base58 changed payload")
	}
	return nil
}
