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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestAssetValidationRejectsStaleConditionalResponses(t *testing.T) {
	want := &casefile.ResponseAsset{SHA256: "body", Revalidate: true}
	correct := AssetValidation{Tag: `"current"`, MatchedStatus: http.StatusNotModified, UnmatchedStatus: http.StatusOK, UnmatchedSHA256: "body"}
	require.NoError(t, checkValidation("/icon", want, &correct))
	for _, alter := range []func(*AssetValidation){
		func(validation *AssetValidation) { validation.Tag = "" },
		func(validation *AssetValidation) { validation.MatchedStatus = http.StatusOK },
		func(validation *AssetValidation) { validation.MatchedBodyBytes = 1 },
		func(validation *AssetValidation) { validation.UnmatchedStatus = http.StatusNotModified },
		func(validation *AssetValidation) { validation.UnmatchedSHA256 = "old body" },
	} {
		validation := correct
		alter(&validation)
		assert.Error(t, checkValidation("/icon", want, &validation), "accepted %+v", validation)
	}
}
