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

// Predefined page size constants in points (1 point = 1/72 inch).

// PageA4 is the ISO A4 page size (210mm x 297mm).
var PageA4 = PageConfig{
	Width:  595.28,
	Height: 841.89,
}

// PageA3 is the ISO A3 page size (297mm x 420mm).
var PageA3 = PageConfig{
	Width:  841.89,
	Height: 1190.55,
}

// PageLetter is the US Letter page size (8.5in x 11in).
var PageLetter = PageConfig{
	Width:  612.0,
	Height: 792.0,
}

// PageLegal is the US Legal page size (8.5in x 14in).
var PageLegal = PageConfig{
	Width:  612.0,
	Height: 1008.0,
}
