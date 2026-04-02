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

package layouter_dto

// FontEntry describes a single font face to register with a font metrics
// implementation.
type FontEntry struct {
	// Family is the CSS font-family name.
	Family string

	// Data is the raw TTF or OTF font bytes.
	Data []byte

	// Weight is the CSS font-weight value (100-900).
	Weight int

	// Style is the font style variant (0 = normal, 1 = italic),
	// matching layouter_domain.FontStyle values.
	Style int

	// WeightMin defines the minimum weight axis value for variable fonts.
	WeightMin int

	// WeightMax defines the maximum weight axis value for variable fonts.
	WeightMax int

	// IsVariable indicates this is an OpenType variable font (has fvar table).
	IsVariable bool
}
