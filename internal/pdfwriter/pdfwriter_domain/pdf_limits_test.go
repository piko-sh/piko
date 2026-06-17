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

package pdfwriter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveEmbeddedDataLimits(t *testing.T) {
	t.Run("nil uses defaults", func(t *testing.T) {
		limits := resolveEmbeddedDataLimits(nil)
		assert.Equal(t, defaultMaxEmbeddedFiles, limits.MaxFiles, "expected default MaxFiles")
		assert.Equal(t, defaultMaxEmbeddedFileBytes, limits.MaxFileBytes, "expected default MaxFileBytes")
		assert.Equal(t, defaultMaxEmbeddedTotalBytes, limits.MaxTotalBytes, "expected default MaxTotalBytes")
		assert.Equal(t, defaultMaxStructuredMetadataBytes, limits.MaxStructuredMetadataBytes, "expected default MaxStructuredMetadataBytes")
	})

	t.Run("zero fields fall back to defaults", func(t *testing.T) {
		limits := resolveEmbeddedDataLimits(&EmbeddedDataLimits{MaxFiles: 5})
		assert.EqualValues(t, 5, limits.MaxFiles, "expected MaxFiles override 5")
		assert.Equal(t, defaultMaxEmbeddedFileBytes, limits.MaxFileBytes, "expected MaxFileBytes default")
	})
}

func TestValidateEmbeddedData(t *testing.T) {
	tests := []struct {
		wantErr    error
		structured *StructuredMetadata
		limits     *EmbeddedDataLimits
		name       string
		files      []EmbeddedFile
	}{
		{
			name:  "within defaults",
			files: []EmbeddedFile{{Name: "a.json", Data: []byte("{}")}},
		},
		{
			name:    "too many files",
			files:   []EmbeddedFile{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			limits:  &EmbeddedDataLimits{MaxFiles: 2},
			wantErr: ErrTooManyEmbeddedFiles,
		},
		{
			name:    "file too large",
			files:   []EmbeddedFile{{Name: "big.json", Data: make([]byte, 11)}},
			limits:  &EmbeddedDataLimits{MaxFileBytes: 10},
			wantErr: ErrEmbeddedFileTooLarge,
		},
		{
			name:       "structured metadata too large",
			structured: &StructuredMetadata{SchemaOrgJSONLD: "0123456789X"},
			limits:     &EmbeddedDataLimits{MaxStructuredMetadataBytes: 10},
			wantErr:    ErrStructuredMetadataTooLarge,
		},
		{
			name:    "file name too long",
			files:   []EmbeddedFile{{Name: "0123456789X"}},
			limits:  &EmbeddedDataLimits{MaxNameBytes: 10},
			wantErr: ErrEmbeddedFileMetadataTooLong,
		},
		{
			name:    "file description too long",
			files:   []EmbeddedFile{{Name: "a", Description: "0123456789X"}},
			limits:  &EmbeddedDataLimits{MaxDescriptionBytes: 10},
			wantErr: ErrEmbeddedFileMetadataTooLong,
		},
		{
			name: "aggregate total too large",
			files: []EmbeddedFile{
				{Name: "a", Data: make([]byte, 6)},
				{Name: "b", Data: make([]byte, 6)},
			},
			limits:  &EmbeddedDataLimits{MaxFileBytes: 100, MaxTotalBytes: 10},
			wantErr: ErrEmbeddedTotalTooLarge,
		},
		{
			name: "aggregate total within limit",
			files: []EmbeddedFile{
				{Name: "a", Data: make([]byte, 5)},
				{Name: "b", Data: make([]byte, 5)},
			},
			limits: &EmbeddedDataLimits{MaxFileBytes: 100, MaxTotalBytes: 10},
		},
		{
			name:    "MIME type too long",
			files:   []EmbeddedFile{{Name: "a", MIMEType: "0123456789X"}},
			limits:  &EmbeddedDataLimits{MaxNameBytes: 10},
			wantErr: ErrEmbeddedMIMETypeTooLong,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateEmbeddedData(test.files, test.structured, test.limits)
			if test.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, test.wantErr)
		})
	}
}
