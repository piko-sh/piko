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

package casefile

// Edition applies source edits to the original project snapshot.
type Edition struct {
	// ID identifies the recorded item.
	ID string

	// Write maps edited source paths to their replacement contents.
	Write map[string]string `json:",omitempty"`

	// Remove lists source paths removed from the baseline snapshot.
	Remove []string `json:",omitempty"`

	// Probes holds follow-up requests in execution order.
	Probes []Probe
}

// UsesCollections reports whether the case includes the external collection module.
//
// Returns bool which reports whether the saved sources include the collection module.
func (c *Case) UsesCollections() bool {
	return c.Target == "collection-routes" || c.Target == "content-editions"
}
