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

package capabilities_functions

import (
	"errors"
	"io"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

// fatalReader wraps an io.Reader and marks any non-EOF read errors as fatal
// capability errors. This is used by lazy capabilities (like minifiers) whose
// parse errors surface during Read rather than at creation time.
type fatalReader struct {
	// inner is the wrapped reader that provides the raw data.
	inner io.Reader
}

// Read delegates to the inner reader and wraps non-EOF errors as fatal.
//
// Takes p ([]byte) which is the buffer to read into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) which is nil, io.EOF, or a fatal-wrapped error.
func (r *fatalReader) Read(p []byte) (n int, err error) {
	n, err = r.inner.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		err = capabilities_domain.NewFatalError(err)
	}
	return n, err
}

// newFatalReader wraps a reader so that any non-EOF error from Read is tagged
// as a fatal capability error via capabilities_domain.NewFatalError.
//
// Takes r (io.Reader) which is the underlying reader to wrap.
//
// Returns *fatalReader which wraps read errors as fatal.
func newFatalReader(r io.Reader) *fatalReader {
	return &fatalReader{inner: r}
}
