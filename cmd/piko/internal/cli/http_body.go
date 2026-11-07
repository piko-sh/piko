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

package cli

import (
	"fmt"
	"io"
)

const (
	// maxDrainBytes caps how much of an unread HTTP response body is discarded before
	// Close.
	maxDrainBytes = 64 * 1024
)

// readBoundedBody reads at most maxBytes from body through io.LimitReader, failing
// rather than returning a silently truncated payload when the body is larger.
//
// Takes body (io.Reader) which is the stream to read; the caller still owns Close.
// Takes maxBytes (int64) which is the largest accepted payload in bytes.
// Takes tooLargeErr (error) which is the sentinel wrapped when the cap is exceeded.
//
// Returns []byte with the body bytes.
// Returns error wrapping tooLargeErr when the body exceeds maxBytes, or the read error.
func readBoundedBody(body io.Reader, maxBytes int64, tooLargeErr error) ([]byte, error) {
	limited := io.LimitReader(body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: more than %d bytes received", tooLargeErr, maxBytes)
	}
	return data, nil
}

// drainAndClose discards up to maxDrainBytes of what is left in an HTTP response body
// and then closes it, so the underlying connection can be reused on every path,
// including error paths that never read the body.
//
// Takes body (io.ReadCloser) which is the response body to drain and close.
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, maxDrainBytes))
	_ = body.Close()
}
