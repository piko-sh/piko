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

package contextaware

import (
	"context"
	"io"
)

var _ io.Reader = (*reader)(nil)

// reader wraps an io.Reader and checks the context before each read.
// When the context is cancelled, Read returns the context error
// immediately, allowing io.Copy loops to exit promptly.
type reader struct {
	// ctx is the context checked before each read call.
	ctx context.Context

	// r is the underlying reader to delegate reads to.
	r io.Reader
}

// Read delegates to the wrapped reader after checking the context.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) which is the context error or the underlying read error.
func (c *reader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// NewReader returns an [io.Reader] that checks ctx before each Read
// call. When ctx is cancelled or its deadline expires, Read returns
// the context error instead of delegating to r.
//
// Takes r (io.Reader) which is the underlying reader to wrap.
//
// Returns io.Reader which is a context-aware wrapper around r.
func NewReader(ctx context.Context, r io.Reader) io.Reader {
	return &reader{ctx: ctx, r: r}
}
