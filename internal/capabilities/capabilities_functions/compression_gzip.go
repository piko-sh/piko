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

	"github.com/klauspost/compress/gzip"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

const (
	// defaultGzipLevel is the default compression level used if not specified in
	// parameters.
	defaultGzipLevel = gzip.DefaultCompression

	// paramGzipLevel is the key used in the capability parameters map to specify
	// the compression level.
	paramGzipLevel = "level"
)

// gzipPools manages a collection of sync.Pools, one for each compression level.
// This is a performance optimisation to avoid frequent allocations of
// gzip.Writer objects, reducing GC pressure under high load.
type gzipPools struct {
	// pools maps compression levels to their writer pools.
	pools map[int]*sync.Pool

	// mu guards access to the pools map.
	mu sync.RWMutex
}

// getPoolForLevel retrieves or creates a sync.Pool for a specific compression
// level. It uses a double-checked lock pattern for safe, lazy initialisation
// of pools.
//
// Takes level (int) which specifies the gzip compression level.
//
// Returns *sync.Pool which provides pooled gzip writers for the given level.
//
// Safe for concurrent use. Uses a read-write mutex with double-checked locking
// to allow concurrent reads while serialising pool creation.
func (p *gzipPools) getPoolForLevel(level int) *sync.Pool {
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
			w, _ := gzip.NewWriterLevel(nil, level)
			return w
		},
	}
	p.pools[level] = pool
	return pool
}

// globalGzipPools is the singleton instance of our pool manager.
var globalGzipPools = &gzipPools{
	pools: make(map[int]*sync.Pool),
	mu:    sync.RWMutex{},
}

// pooledGzipWriter wraps a gzip.Writer and returns it to the pool on Close.
// It implements io.WriteCloser.
type pooledGzipWriter struct {
	*gzip.Writer

	// pool is a reference to the pool that owns this writer.
	pool *sync.Pool
}

// Close closes the underlying gzip.Writer to flush its buffers, and then
// returns the writer object to the pool it came from, making it available
// for reuse.
//
// Returns error when the underlying gzip.Writer fails to close.
func (w *pooledGzipWriter) Close() error {
	err := w.Writer.Close()

	w.pool.Put(w.Writer)

	if err != nil {
		return fmt.Errorf("closing gzip writer: %w", err)
	}
	return nil
}

// Gzip returns a capability function that performs streaming Gzip
// compression. It uses a generic stream processor and a correctly implemented
// sync.Pool to reuse gzip.Writer objects for improved performance and reduced
// memory allocations.
//
// Returns capabilities_domain.CapabilityFunc which handles the compression
// operation.
func Gzip() capabilities_domain.CapabilityFunc {
	return createCompressionCapability(compressionConfig{
		spanName:     "GzipCompression",
		defaultLevel: defaultGzipLevel,
		parseLevel:   parseGzipLevel,
		factory:      gzipWriterFactory,
	})
}

// parseGzipLevel gets the compression level from the given parameters.
//
// Takes params (CapabilityParams) which contains the settings to search for a
// gzip level value.
// Takes defaultLevel (int) which is the fallback level to use.
//
// Returns int which is the parsed compression level, or defaultLevel if the
// setting is missing, not a number, or outside the valid range (-2 to 9).
func parseGzipLevel(params capabilities_domain.CapabilityParams, defaultLevel int) int {
	levelString, ok := params[paramGzipLevel]
	if !ok {
		return defaultLevel
	}

	parsedLevel, err := strconv.Atoi(levelString)
	if err != nil {
		return defaultLevel
	}

	if parsedLevel < gzip.HuffmanOnly || parsedLevel > gzip.BestCompression {
		return defaultLevel
	}

	return parsedLevel
}

// gzipWriterFactory creates a writerFactory for a given compression level.
//
// Takes level (int) which sets the gzip compression level.
//
// Returns writerFactory which produces pooled gzip writers at the given level.
func gzipWriterFactory(level int) writerFactory {
	return func(destination io.Writer) (io.WriteCloser, error) {
		pool := globalGzipPools.getPoolForLevel(level)

		gw, ok := pool.Get().(*gzip.Writer)
		if !ok {
			gw, _ = gzip.NewWriterLevel(destination, level)
			return gw, nil
		}

		gw.Reset(destination)

		return &pooledGzipWriter{
			Writer: gw,
			pool:   pool,
		}, nil
	}
}
