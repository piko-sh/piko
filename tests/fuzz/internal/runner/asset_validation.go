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
	"context"
	"fmt"
	"net/http"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// AssetValidation retains both responses so a stale validator and a stale body are
// distinguishable.
type AssetValidation struct {
	// Tag is the entity validator sent with the matching request.
	Tag string

	// UnmatchedSHA256 is the hex SHA-256 of the body returned for a stale validator.
	UnmatchedSHA256 string

	// MatchedStatus is the status returned for the matching validator.
	MatchedStatus int

	// MatchedBodyBytes is the body size returned for the matching validator.
	MatchedBodyBytes int

	// UnmatchedStatus is the status returned for a stale validator.
	UnmatchedStatus int
}

// observeValidation records matching and nonmatching conditional asset requests.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes path (string) which is the asset request path.
// Takes tag (string) which is the expected entity validator.
//
// Returns *AssetValidation which records matching and nonmatching conditional asset
// responses.
// Returns error which reports a conditional-request or response-read failure, or nil on
// success.
func observeValidation(ctx context.Context, client *http.Client, base, path, tag string) (*AssetValidation, error) {
	validation := &AssetValidation{Tag: tag, UnmatchedSHA256: "", MatchedStatus: 0, MatchedBodyBytes: 0, UnmatchedStatus: 0}
	if tag == "" {
		return validation, nil
	}
	status, _, body, err := getConditional(ctx, client, base, path, tag)
	if err != nil {
		return nil, err
	}
	validation.MatchedStatus, validation.MatchedBodyBytes = status, len(body)

	other := `"piko-fuzz-stale-a"`
	if tag == other || tag == "W/"+other {
		other = `"piko-fuzz-stale-b"`
	}
	status, _, body, err = getConditional(ctx, client, base, path, other)
	if err != nil {
		return nil, err
	}
	validation.UnmatchedStatus = status
	validation.UnmatchedSHA256 = sha256Hex(body)
	return validation, nil
}

// checkValidation checks an asset's conditional response and byte identity.
//
// Takes path (string) which identifies the asset request in failure messages.
// Takes want (*casefile.ResponseAsset) which holds the independently expected result.
// Takes validation (*AssetValidation) which holds the observed conditional responses.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkValidation(path string, want *casefile.ResponseAsset, validation *AssetValidation) error {
	if validation == nil || validation.Tag == "" ||
		validation.MatchedStatus != http.StatusNotModified || validation.MatchedBodyBytes != 0 ||
		validation.UnmatchedStatus != http.StatusOK || validation.UnmatchedSHA256 != want.SHA256 {
		return fmt.Errorf("probe/asset-revalidation: %s got %+v, want ETag, empty 304 and fresh 200 with digest %s", path, validation, want.SHA256)
	}
	return nil
}
