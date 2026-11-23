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
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestEditionReceiptRequiresEditsAndOldValidators(t *testing.T) {
	first, second, third := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	valid := []EditionReceipt{
		{ID: "original", ContentSHA256: first, AssetSHA256: first, ETag: "old", Reached: true, Verified: true},
		{ID: "replace", ContentSHA256: second, AssetSHA256: second, ETag: "new", ConditionalStatus: http.StatusOK, Reached: true, Verified: true},
		{ID: "remove", ContentSHA256: third, AssetSHA256: second, ETag: "new", ConditionalStatus: http.StatusNotModified, Reached: true, Verified: true},
		{ID: "restore", ContentSHA256: first, AssetSHA256: first, ETag: "old", ConditionalStatus: http.StatusOK, Reached: true, Verified: true},
	}
	require.NoError(t, checkEditions(valid))
	for _, change := range []func([]EditionReceipt){
		func(receipts []EditionReceipt) { receipts[1].AssetSHA256 = first },
		func(receipts []EditionReceipt) { receipts[2].ContentSHA256 = second },
		func(receipts []EditionReceipt) { receipts[3].ContentSHA256 = third },
		func(receipts []EditionReceipt) { receipts[2].ConditionalStatus = http.StatusOK },
		func(receipts []EditionReceipt) { receipts[1].ETag = "" },
		func(receipts []EditionReceipt) { receipts[1].Reached = false },
	} {
		receipts := slices.Clone(valid)
		change(receipts)
		assert.Error(t, checkEditions(receipts), "accepted incorrect edition %+v", receipts)
	}
}

func TestEditionCaseAppliesTheEditAndAddsTheOldValidator(t *testing.T) {
	media := casefile.Probe{Path: "/media.png", Status: http.StatusOK, Asset: &casefile.ResponseAsset{MediaType: "image/png", SHA256: "new"}}
	base := &casefile.Case{Files: map[string]string{"keep.txt": "keep", "gone.txt": "gone"}}
	edition := &casefile.Edition{ID: "replace", Write: map[string]string{"media.png": "png"}, Remove: []string{"gone.txt"}, Probes: []casefile.Probe{{Path: "/"}, media}}

	first, mediaPath, conditional, err := editionCase(base, edition, nil)
	require.NoError(t, err)
	assert.Equal(t, "/media.png", mediaPath)
	assert.Zero(t, conditional)
	assert.Len(t, first.Probes, 2)
	assert.Equal(t, map[string]string{"keep.txt": "keep", "media.png": "png"}, first.Files)
	assert.Contains(t, base.Files, "gone.txt")

	changed, _, conditional, err := editionCase(base, edition, &EditionReceipt{ETag: `"old"`, AssetSHA256: "old"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, conditional)
	validator := changed.Probes[len(changed.Probes)-1]
	assert.Equal(t, `"old"`, validator.IfNoneMatch)
	assert.Equal(t, "previous-edition-validator", validator.Stage)
	assert.NotNil(t, validator.Asset)

	unchanged, _, conditional, err := editionCase(base, edition, &EditionReceipt{ETag: `"new"`, AssetSHA256: "new"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotModified, conditional)
	assert.Nil(t, unchanged.Probes[len(unchanged.Probes)-1].Asset)

	_, _, _, err = editionCase(base, &casefile.Edition{ID: "bare", Probes: []casefile.Probe{{Path: "/"}}}, nil)
	assert.ErrorContains(t, err, "missing media")
}

func TestVerifyEditionRecordsTheValidatorAndDigest(t *testing.T) {
	asset := &casefile.ResponseAsset{MediaType: "image/png", SHA256: strings.Repeat("a", 64)}
	c := &casefile.Case{Expected: casefile.Expected{Status: http.StatusOK}, Probes: []casefile.Probe{{Path: "/media.png", Status: http.StatusOK, Asset: asset}}}
	observation := func(validatorStatus int, validatorBody string) Observation {
		return Observation{Status: http.StatusOK, ContentType: "text/html", MetadataTitle: "Piko Page", Probes: []ProbeObservation{
			{Path: "/media.png", Status: http.StatusOK, Asset: &casefile.ResponseAsset{MediaType: "image/png", SHA256: asset.SHA256}, Validation: &AssetValidation{Tag: `"v1"`}},
			{Path: "/media.png", Status: validatorStatus, RawHTML: validatorBody},
		}}
	}

	got := observation(http.StatusNotModified, "")
	got.Probes = got.Probes[:1]
	var first EditionReceipt
	require.NoError(t, verifyEdition(c, &got, "/media.png", false, 0, &first))
	assert.True(t, first.Verified)
	assert.Equal(t, `"v1"`, first.ETag)
	assert.Equal(t, asset.SHA256, first.AssetSHA256)
	assert.Len(t, first.ContentSHA256, sha256HexLength)

	validated := *c
	validated.Probes = append(slices.Clone(c.Probes), casefile.Probe{Path: "/media.png", Status: http.StatusNotModified})
	got = observation(http.StatusNotModified, "")
	var second EditionReceipt
	require.NoError(t, verifyEdition(&validated, &got, "/media.png", true, http.StatusNotModified, &second))
	assert.Equal(t, http.StatusNotModified, second.ConditionalStatus)
	assert.Equal(t, first.ContentSHA256, second.ContentSHA256)

	got = observation(http.StatusNotModified, "stale body")
	assert.ErrorContains(t, verifyEdition(&validated, &got, "/media.png", true, http.StatusNotModified, &EditionReceipt{}), "old validator returned 304")
	got = Observation{Status: http.StatusInternalServerError}
	assert.Error(t, verifyEdition(c, &got, "/media.png", false, 0, &EditionReceipt{}))
}

func TestRunEditionsStopsAtTheFirstUnverifiedStage(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	workers := fakeWorkers(t)
	installFakeGo(t)
	c := generate.New("content-editions", 7, 0)

	o, err := RunWithWorkers(t.Context(), repo, t.TempDir(), workers, &c, modeCompiled, false)

	require.ErrorContains(t, err, "content-edition/"+c.Expected.Editions[0].ID)
	require.Len(t, o.Editions, 1)
	assert.True(t, o.Editions[0].Reached)
	assert.False(t, o.Editions[0].Verified)

	c.Expected.Editions = c.Expected.Editions[:1]
	_, err = RunWithWorkers(t.Context(), repo, t.TempDir(), workers, &c, modeCompiled, false)
	assert.ErrorContains(t, err, "need four saved stages")
}
