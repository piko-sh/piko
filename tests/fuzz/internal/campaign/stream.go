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

package campaign

const (
	// soakWindowRecipes is the number of recipes generated from one soak seed.
	soakWindowRecipes = 64
)

// StreamPosition rotates the seed after a complete 64-recipe window.
//
// The absolute checkpoint and suite determine both coordinates on every resume.
//
// Takes suite (string) which selects the campaign preset.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes offset (uint64) which is the absolute campaign stream position.
//
// Returns uint64 which is the deterministic seed for the recipe window.
// Returns uint64 which is the absolute campaign stream position.
func StreamPosition(suite string, seed, offset uint64) (windowSeed, position uint64) {
	if suite != "soak" {
		return seed, offset
	}
	return seed + offset/soakWindowRecipes, offset
}
