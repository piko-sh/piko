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

package daemon_dto

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileUpload_WithContentType(t *testing.T) {
	t.Parallel()

	header := &multipart.FileHeader{
		Filename: "doc.pdf",
		Size:     1024,
		Header:   textproto.MIMEHeader{"Content-Type": {"application/pdf"}},
	}

	fu := NewFileUpload(header)

	assert.Equal(t, "doc.pdf", fu.Name)
	assert.Equal(t, int64(1024), fu.Size)
	assert.Equal(t, "application/pdf", fu.ContentType)
}

func TestNewFileUpload_NilHeaderMap(t *testing.T) {
	t.Parallel()

	header := &multipart.FileHeader{
		Filename: "image.png",
		Size:     512,
		Header:   nil,
	}

	fu := NewFileUpload(header)

	assert.Equal(t, "image.png", fu.Name)
	assert.Equal(t, int64(512), fu.Size)
	assert.Empty(t, fu.ContentType, "ContentType should be empty when Header is nil")
}

func TestFileUpload_Open_NilHeader(t *testing.T) {
	t.Parallel()

	fu := &FileUpload{}

	rc, err := fu.Open()
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestFileUpload_Open_WithContent(t *testing.T) {
	t.Parallel()

	content := []byte("hello world")
	header := createTestFileHeader(t, "test.txt", "text/plain", content)

	fu := NewFileUpload(header)

	rc, err := fu.Open()
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestFileUpload_ReadAll_NilHeader(t *testing.T) {
	t.Parallel()

	fu := &FileUpload{}

	data, err := fu.ReadAll()
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestFileUpload_ReadAll_WithContent(t *testing.T) {
	t.Parallel()

	content := []byte("file content here")
	header := createTestFileHeader(t, "data.bin", "application/octet-stream", content)

	fu := NewFileUpload(header)

	data, err := fu.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestFileUpload_Header(t *testing.T) {
	t.Parallel()

	header := &multipart.FileHeader{
		Filename: "test.txt",
		Size:     10,
	}

	fu := NewFileUpload(header)
	assert.Same(t, header, fu.Header())
}

func TestFileUpload_Header_Nil(t *testing.T) {
	t.Parallel()

	fu := &FileUpload{}
	assert.Nil(t, fu.Header())
}

func TestFileUpload_ReadAllWithLimit_RejectsOversize(t *testing.T) {
	t.Parallel()

	content := bytes.Repeat([]byte("A"), 1024)
	header := createTestFileHeader(t, "big.bin", "application/octet-stream", content)
	fu := NewFileUpload(header)

	data, err := fu.ReadAllWithLimit(int64(len(content) - 1))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFileUploadTooLarge)
	assert.Contains(t, err.Error(), "big.bin")
	assert.Nil(t, data)
}

func TestFileUpload_ReadAllWithLimit_AcceptsAtBoundary(t *testing.T) {
	t.Parallel()

	content := bytes.Repeat([]byte("B"), 256)
	header := createTestFileHeader(t, "ok.bin", "application/octet-stream", content)
	fu := NewFileUpload(header)

	data, err := fu.ReadAllWithLimit(int64(len(content)))
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestFileUpload_ReadAll_HonoursDefaultCap(t *testing.T) {
	original := CurrentMaxUploadFileBytes()
	t.Cleanup(func() {
		SetDefaultMaxUploadFileBytes(original)
	})

	content := bytes.Repeat([]byte("C"), 512)
	header := createTestFileHeader(t, "cap.bin", "application/octet-stream", content)
	fu := NewFileUpload(header)

	SetDefaultMaxUploadFileBytes(int64(len(content) - 1))

	data, err := fu.ReadAll()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrFileUploadTooLarge))
	assert.Nil(t, data)
}

func TestSetDefaultMaxUploadFileBytes_NonPositiveResets(t *testing.T) {
	original := CurrentMaxUploadFileBytes()
	t.Cleanup(func() {
		SetDefaultMaxUploadFileBytes(original)
	})

	SetDefaultMaxUploadFileBytes(123)
	assert.Equal(t, int64(123), CurrentMaxUploadFileBytes())

	SetDefaultMaxUploadFileBytes(0)
	assert.Equal(t, DefaultMaxUploadFileBytes, CurrentMaxUploadFileBytes())

	SetDefaultMaxUploadFileBytes(-1)
	assert.Equal(t, DefaultMaxUploadFileBytes, CurrentMaxUploadFileBytes())
}

func createTestFileHeader(t *testing.T, filename, contentType string, content []byte) *multipart.FileHeader {
	t.Helper()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	require.NoError(t, err)

	_, err = part.Write(content)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	reader := multipart.NewReader(&buffer, writer.Boundary())
	form, err := reader.ReadForm(1 << 20)
	require.NoError(t, err)

	files := form.File["file"]
	require.Len(t, files, 1)

	return files[0]
}
