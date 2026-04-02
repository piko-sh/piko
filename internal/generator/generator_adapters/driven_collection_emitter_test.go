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

package generator_adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/wdk/safedisk"
)

type mockCollectionEncoder struct {
	encodeErr    error
	encodeResult []byte
}

func (m *mockCollectionEncoder) EncodeCollection(_ []collection_dto.ContentItem) ([]byte, error) {
	return m.encodeResult, m.encodeErr
}

func (m *mockCollectionEncoder) DecodeCollectionItem(_ []byte, _ string) ([]byte, []byte, []byte, error) {
	return nil, nil, nil, nil
}

type collectionWriteRecord struct {
	path string
	data []byte
}

func newCollectionTrackingFSWriter(writeErr error, writeErrOnCall int) (*generator_domain.MockFSWriter, *[]collectionWriteRecord) {
	var writes []collectionWriteRecord
	callCount := 0
	return &generator_domain.MockFSWriter{
		WriteFileFunc: func(_ context.Context, filePath string, data []byte) error {
			callCount++
			if writeErrOnCall > 0 && callCount == writeErrOnCall {
				return writeErr
			}
			if writeErrOnCall == 0 && writeErr != nil {
				return writeErr
			}
			writes = append(writes, collectionWriteRecord{path: filePath, data: data})
			return nil
		},
	}, &writes
}

func TestNewDrivenCollectionEmitter(t *testing.T) {
	t.Parallel()

	encoder := &mockCollectionEncoder{}
	fsWriter := &generator_domain.MockFSWriter{}
	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer sandbox.Close()

	emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

	require.NotNil(t, emitter)
}

func TestDrivenCollectionEmitter_EmitCollection(t *testing.T) {
	t.Parallel()

	items := []collection_dto.ContentItem{
		{ID: "1", Slug: "hello-world"},
	}

	t.Run("success writes binary and Go wrapper", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("binary-data")}
		fsWriter, writes := newCollectionTrackingFSWriter(nil, 0)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		packagePath, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.NoError(t, err)
		assert.Equal(t, "mymod/dist/collections/docs", packagePath)
		require.Len(t, *writes, 2)

		assert.Equal(t, "dist/collections/docs/data.bin", (*writes)[0].path)
		assert.Equal(t, []byte("binary-data"), (*writes)[0].data)

		assert.Equal(t, "dist/collections/docs/generated.go", (*writes)[1].path)
		goCode := string((*writes)[1].data)
		assert.Contains(t, goCode, "package docs")
		assert.Contains(t, goCode, "//go:embed data.bin")
		assert.Contains(t, goCode, `RegisterStaticCollectionBlob(context.Background(), "docs"`)
	})

	t.Run("MkdirAll error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("data")}
		fsWriter := &generator_domain.MockFSWriter{}
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.MkdirAllErr = errors.New("cannot create directory")

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create collection directory")
	})

	t.Run("encode error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeErr: errors.New("encode failed")}
		fsWriter := &generator_domain.MockFSWriter{}
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode collection")
	})

	t.Run("binary write error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("data")}
		fsWriter, _ := newCollectionTrackingFSWriter(errors.New("write failed"), 1)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write binary data")
	})

	t.Run("Go wrapper write error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("data")}
		fsWriter, _ := newCollectionTrackingFSWriter(errors.New("write failed"), 2)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write Go wrapper")
	})
}
