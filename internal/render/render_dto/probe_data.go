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

package render_dto

import "sync"

// ProbeData holds data collected during the probe phase (CollectMetadata) that
// can be reused by the render phase (RenderAST) to avoid redundant cache
// lookups. This struct is pooled and designed to be expanded as more data is
// plumbed between probe and render in future.
type ProbeData struct {
	// ComponentMetadata maps component tag names to their metadata. Populated
	// by BulkGetComponentMetadata during the probe phase.
	ComponentMetadata map[string]*ComponentMetadata
}

// Reset clears all fields so the struct can be returned to the pool.
func (p *ProbeData) Reset() {
	clear(p.ComponentMetadata)
	p.ComponentMetadata = nil
}

var probeDataPool = sync.Pool{
	New: func() any {
		return &ProbeData{}
	},
}

// AcquireProbeData retrieves a ProbeData from the pool.
//
// Returns *ProbeData which is ready for use.
func AcquireProbeData() *ProbeData {
	if pd, ok := probeDataPool.Get().(*ProbeData); ok {
		return pd
	}
	return &ProbeData{}
}

// ReleaseProbeData returns a ProbeData to the pool after resetting it.
//
// Takes p (*ProbeData) which is the struct to recycle. Nil is safe.
func ReleaseProbeData(p *ProbeData) {
	if p == nil {
		return
	}
	p.Reset()
	probeDataPool.Put(p)
}
