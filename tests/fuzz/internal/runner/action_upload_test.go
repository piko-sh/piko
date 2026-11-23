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

package runner

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestStoredUploadReadbackIsBounded(t *testing.T) {
	work := t.TempDir()
	path := filepath.Join(work, storedUploadName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	const limit = 4096
	payload := bytes.Repeat([]byte{0, 255}, limit/2)
	require.NoError(t, os.WriteFile(path, payload, 0600))
	got, err := readStoredUpload(work)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256(payload)), got)
	require.NoError(t, os.WriteFile(path, append(payload, 0), 0600))
	_, err = readStoredUpload(work)
	assert.Error(t, err, "oversized persisted object passed readback")
	_, err = observeUpload(t.Context(), nil, "", "", "", casefile.ActionValue{Label: strings.Repeat("x", limit)}, true)
	assert.Error(t, err, "oversized upload reached the transport")
}

func TestStoreFailurePhasesSurviveInputChanges(t *testing.T) {
	for _, label := range []string{"left", "東京"} {
		input := casefile.ActionValue{Label: label, Quantity: len(label)}
		contract := &casefile.ActionContract{Input: input, UploadPublic: true}
		upload := expectedUpload(input, true)
		upload.RangeDigest = upload.Digest
		err := checkUploadedObject(contract, &upload)
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "action/storage-range:"), "%v", err)
		upload = expectedUpload(input, true)
		upload.PublicDigest = ""
		err = checkUploadedObject(contract, &upload)
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "action/public-download-body:"), "%v", err)
	}
}
