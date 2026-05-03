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

package pdfwriter_adapters_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestRegistryImageDataAdapter_RejectsOversizedImage(t *testing.T) {
	const oversize = (100 << 20) + 1

	mock := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ActualVariants: []registry_dto.Variant{{VariantID: "source"}},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(repeatingReader(oversize)), nil
		},
	}

	adapter := pdfwriter_adapters.NewRegistryImageDataAdapter(mock)

	_, _, err := adapter.GetImageData(context.Background(), "/_piko/assets/huge.png")
	require.Error(t, err)
	assert.ErrorIs(t, err, pdfwriter_adapters.ErrImageDataTooLarge)
}

func TestRegistryImageDataAdapter_AllowsExactlyMaxSize(t *testing.T) {
	const exactSize = 100 << 20

	mock := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ActualVariants: []registry_dto.Variant{{VariantID: "source"}},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			data := bytes.Repeat([]byte{0xFF}, exactSize)
			copy(data, []byte{0xFF, 0xD8, 0xFF, 0xE0})
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}

	adapter := pdfwriter_adapters.NewRegistryImageDataAdapter(mock)

	data, format, err := adapter.GetImageData(context.Background(), "/_piko/assets/borderline.jpg")
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
	assert.Len(t, data, exactSize)
}

func TestRegistryImageDataAdapter_PropagatesArtefactNotFound(t *testing.T) {
	mock := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
	}

	adapter := pdfwriter_adapters.NewRegistryImageDataAdapter(mock)
	_, _, err := adapter.GetImageData(context.Background(), "missing")
	require.Error(t, err)
	assert.NotErrorIs(t, err, pdfwriter_adapters.ErrImageDataTooLarge)
}

func TestRegistryImageDataAdapter_PropagatesReadError(t *testing.T) {
	wantErr := errors.New("network glitch")
	mock := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ActualVariants: []registry_dto.Variant{{VariantID: "source"}},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(&erroringReader{err: wantErr}), nil
		},
	}

	adapter := pdfwriter_adapters.NewRegistryImageDataAdapter(mock)
	_, _, err := adapter.GetImageData(context.Background(), "broken")
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

type erroringReader struct {
	err error
}

func (e *erroringReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

func repeatingReader(n int) io.Reader {
	header := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	body := bytes.Repeat([]byte{0x00}, n-len(header))
	return io.MultiReader(bytes.NewReader(header), bytes.NewReader(body))
}
