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

package registry_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func transformWith(parentHash string, version uint32, params map[string]string) VariantTransform {
	profileParams := ProfileParams{}
	for key, value := range params {
		profileParams.SetByName(key, value)
	}
	return VariantTransform{
		ParentVariantID:   "source",
		ParentContentHash: parentHash,
		CapabilityName:    "image-transform",
		CapabilityVersion: version,
		Params:            profileParams,
	}
}

func TestFingerprintDeterministic(t *testing.T) {
	first := transformWith("hash-1", 1, map[string]string{"width": "360", "format": "webp", "quality": "80"})
	second := transformWith("hash-1", 1, map[string]string{"quality": "80", "format": "webp", "width": "360"})

	firstPrint, err := first.Fingerprint()
	require.NoError(t, err)
	secondPrint, err := second.Fingerprint()
	require.NoError(t, err)

	assert.Equal(t, firstPrint, secondPrint, "insertion order must not change the fingerprint")
	assert.NotEmpty(t, firstPrint)
}

func TestFingerprintChangesWithEachInput(t *testing.T) {
	base := transformWith("hash-1", 1, map[string]string{"width": "360"})
	basePrint, err := base.Fingerprint()
	require.NoError(t, err)

	changedParent := transformWith("hash-2", 1, map[string]string{"width": "360"})
	changedParentPrint, err := changedParent.Fingerprint()
	require.NoError(t, err)
	assert.NotEqual(t, basePrint, changedParentPrint, "a changed parent content hash must change the fingerprint")

	changedVersion := transformWith("hash-1", 2, map[string]string{"width": "360"})
	changedVersionPrint, err := changedVersion.Fingerprint()
	require.NoError(t, err)
	assert.NotEqual(t, basePrint, changedVersionPrint, "a changed capability version must change the fingerprint")

	changedParams := transformWith("hash-1", 1, map[string]string{"width": "720"})
	changedParamsPrint, err := changedParams.Fingerprint()
	require.NoError(t, err)
	assert.NotEqual(t, basePrint, changedParamsPrint, "changed parameters must change the fingerprint")
}

func TestFingerprintCapabilityDistinguishes(t *testing.T) {
	imageTransform := transformWith("hash-1", 1, map[string]string{"width": "360"})
	other := imageTransform
	other.CapabilityName = "minify-css"

	imagePrint, err := imageTransform.Fingerprint()
	require.NoError(t, err)
	otherPrint, err := other.Fingerprint()
	require.NoError(t, err)
	assert.NotEqual(t, imagePrint, otherPrint, "different capabilities must fingerprint differently")
}

func TestFingerprintRejectsIncompleteInputs(t *testing.T) {
	noParent := transformWith("", 1, nil)
	_, err := noParent.Fingerprint()
	require.ErrorIs(t, err, ErrFingerprintNoParent)

	noCapability := transformWith("hash-1", 1, nil)
	noCapability.CapabilityName = ""
	_, err = noCapability.Fingerprint()
	require.ErrorIs(t, err, ErrFingerprintNoCapability)

	noVersion := transformWith("hash-1", 0, nil)
	_, err = noVersion.Fingerprint()
	require.ErrorIs(t, err, ErrFingerprintNoVersion)
}

func TestTransformIsZero(t *testing.T) {
	assert.True(t, VariantTransform{}.IsZero(), "the zero transform must report empty")
	assert.False(t, transformWith("hash-1", 1, nil).IsZero(), "a populated transform must not report empty")
}
