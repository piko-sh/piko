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

package builtin_detectors

// commonBigrams contains the most frequent English letter bigrams. Text
// composed of random characters will have a low hit rate against this set.
var commonBigrams = map[string]struct{}{
	"th": {}, "he": {}, "in": {}, "er": {}, "an": {},
	"re": {}, "on": {}, "at": {}, "en": {}, "nd": {},
	"ti": {}, "es": {}, "or": {}, "te": {}, "of": {},
	"ed": {}, "is": {}, "it": {}, "al": {}, "ar": {},
	"st": {}, "to": {}, "nt": {}, "ng": {}, "se": {},
	"ha": {}, "as": {}, "ou": {}, "io": {}, "le": {},
	"ve": {}, "co": {}, "me": {}, "de": {}, "hi": {},
	"ri": {}, "ro": {}, "ic": {}, "ne": {}, "ea": {},
	"ra": {}, "ce": {}, "li": {}, "ch": {}, "ll": {},
	"be": {}, "ma": {}, "si": {}, "om": {}, "ur": {},
	"ca": {}, "el": {}, "ta": {}, "la": {}, "ns": {},
	"ge": {}, "ad": {}, "ec": {}, "ai": {}, "il": {},
	"no": {}, "pe": {}, "di": {}, "ss": {}, "us": {},
	"sh": {}, "tr": {}, "ol": {}, "ot": {}, "et": {},
	"tu": {}, "ie": {}, "wh": {}, "em": {}, "ow": {},
	"ac": {}, "ag": {}, "am": {}, "ap": {}, "ee": {},
	"fo": {}, "id": {}, "ig": {}, "im": {}, "iv": {},
	"ly": {}, "mi": {}, "ni": {}, "op": {}, "ov": {},
	"pa": {}, "pl": {}, "po": {}, "pr": {}, "so": {},
	"su": {}, "ul": {}, "un": {}, "up": {}, "ut": {},
	"wa": {}, "wi": {},
}
