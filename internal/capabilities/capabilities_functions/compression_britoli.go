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
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/andybalholm/brotli"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

const (
	// defaultBrotliLevel is the default compression level used if not specified in
	// parameters.
	defaultBrotliLevel = brotli.DefaultCompression

	// paramBrotliLevel is the key used in the capability parameters map to specify
	// the compression level.
	paramBrotliLevel = "level"
)

// brotliPools manages a collection of sync.Pools, one for each compression
// level. This avoids frequent allocations of brotli.Writer objects and their
// internal buffers, reducing GC pressure under high load.
type brotliPools struct {
	// pools maps compression levels to their writer pools.
	pools map[int]*sync.Pool

	// mu guards access to pools for safe concurrent reads and writes.
	mu sync.RWMutex
}

// getPoolForLevel retrieves or creates a sync.Pool for a given compression
// level. It uses a double-checked lock pattern for safe, lazy initialisation
// of pools.
//
// Takes level (int) which specifies the Brotli compression quality.
//
// Returns *sync.Pool which provides reusable Brotli writers for the given
// level.
//
// Safe for concurrent use. Uses a read-write mutex with double-checked
// locking to allow concurrent reads while serialising pool creation.
func (p *brotliPools) getPoolForLevel(level int) *sync.Pool {
	p.mu.RLock()
	pool, ok := p.pools[level]
	p.mu.RUnlock()

	if ok {
		return pool
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if pool, ok = p.pools[level]; ok {
		return pool
	}

	pool = &sync.Pool{
		New: func() any {
			return brotli.NewWriterOptions(nil, brotli.WriterOptions{
				Quality: level,
				LGWin:   0,
			})
		},
	}
	p.pools[level] = pool
	return pool
}

// globalBrotliPools is the single shared instance of the pool manager.
var globalBrotliPools = &brotliPools{
	pools: make(map[int]*sync.Pool),
	mu:    sync.RWMutex{},
}

// pooledBrotliWriter wraps a brotli.Writer and implements io.WriteCloser.
// It holds a reference to the sync.Pool it came from for proper reuse.
type pooledBrotliWriter struct {
	*brotli.Writer

	// pool is the sync.Pool to return the writer to after use.
	pool *sync.Pool
}

// Close closes the underlying brotli.Writer to flush its buffers, then
// returns the writer object to the pool for reuse.
//
// Returns error when the underlying writer fails to close.
func (w *pooledBrotliWriter) Close() error {
	err := w.Writer.Close()
	w.pool.Put(w.Writer)
	if err != nil {
		return fmt.Errorf("closing brotli writer: %w", err)
	}
	return nil
}

// Brotli returns a capability function that performs streaming Brotli
// compression. It uses a sync.Pool to reuse brotli.Writer objects for better
// performance.
//
// Returns capabilities_domain.CapabilityFunc which compresses request or
// response bodies using Brotli.
func Brotli() capabilities_domain.CapabilityFunc {
	return createCompressionCapability(compressionConfig{
		spanName:     "BrotliCompression",
		defaultLevel: defaultBrotliLevel,
		parseLevel:   parseBrotliLevel,
		factory:      brotliWriterFactory,
	})
}

// parseBrotliLevel extracts the compression level from params.
//
// Takes params (CapabilityParams) which contains the settings to parse.
// Takes defaultLevel (int) which is the fallback if the value is missing or
// invalid.
//
// Returns int which is the parsed level, or defaultLevel if the parameter is
// missing, not a number, or outside the valid brotli range.
func parseBrotliLevel(params capabilities_domain.CapabilityParams, defaultLevel int) int {
	levelString, ok := params[paramBrotliLevel]
	if !ok {
		return defaultLevel
	}

	parsedLevel, err := strconv.Atoi(levelString)
	if err != nil {
		return defaultLevel
	}

	if parsedLevel < brotli.BestSpeed || parsedLevel > brotli.BestCompression {
		return defaultLevel
	}

	return parsedLevel
}

// brotliWriterFactory creates a writerFactory for a given compression level.
//
// Takes level (int) which sets the brotli compression quality.
//
// Returns writerFactory which produces pooled brotli writers for the given
// level.
func brotliWriterFactory(level int) writerFactory {
	return func(destination io.Writer) (io.WriteCloser, error) {
		pool := globalBrotliPools.getPoolForLevel(level)
		bw, ok := pool.Get().(*brotli.Writer)
		if !ok {
			bw = brotli.NewWriterOptions(destination, brotli.WriterOptions{
				Quality: level,
				LGWin:   0,
			})
			return bw, nil
		}
		bw.Reset(destination)
		return &pooledBrotliWriter{
			Writer: bw,
			pool:   pool,
		}, nil
	}
}
