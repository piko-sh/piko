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

package compiler_adapters

import (
	"context"
	"fmt"
	"sync"

	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// memoryInputReader is an in-memory implementation of the InputReaderPort.
// It stores SFC content in a thread-safe map, primarily used for testing
// or scenarios where sources are dynamically generated.
type memoryInputReader struct {
	// dataStore maps source identifiers to their file content as byte slices.
	dataStore map[string][]byte

	// lock guards concurrent access to dataStore.
	lock sync.RWMutex
}

// ReadSFC retrieves an SFC's content from the in-memory store by its identifier.
//
// Takes sourceIdentifier (string) which specifies the key to look up in the
// store.
//
// Returns []byte which contains the SFC content for the given identifier.
// Returns error when no content exists for the given sourceIdentifier.
//
// Safe for concurrent use; uses a read lock when accessing the data store.
func (reader *memoryInputReader) ReadSFC(ctx context.Context, sourceIdentifier string) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "memoryInputReader.ReadSFC",
		logger_domain.String("sourceIdentifier", sourceIdentifier),
	)
	defer span.End()

	var content []byte
	var found bool

	err := l.RunInSpan(ctx, "ReadFromMemory", func(_ context.Context, _ logger_domain.Logger) error {
		reader.lock.RLock()
		defer reader.lock.RUnlock()

		content, found = reader.dataStore[sourceIdentifier]
		if !found {
			return fmt.Errorf("no in-memory data found for key: %s", sourceIdentifier)
		}
		return nil
	})

	if err != nil {
		l.ReportError(span, err, "No in-memory data found for key",
			logger_domain.String("key", sourceIdentifier))
		memoryReadErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("reading SFC from memory for %q: %w", sourceIdentifier, err)
	}

	contentSize := len(content)
	memoryReadSize.Record(ctx, int64(contentSize))
	memoryReadCount.Add(ctx, 1)

	l.Trace("Read SFC from memory", logger_domain.Int("contentSize", contentSize))

	return content, nil
}

// NewMemoryInputReader creates a new in-memory input reader with an empty
// data store.
//
// Returns compiler_domain.InputReaderPort which is ready for use with no
// initial data.
func NewMemoryInputReader() compiler_domain.InputReaderPort {
	return &memoryInputReader{
		dataStore: make(map[string][]byte),
		lock:      sync.RWMutex{},
	}
}
