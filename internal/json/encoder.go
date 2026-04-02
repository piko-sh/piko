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

package json

// Encoder writes JSON values to an output stream.
type Encoder interface {
	// Encode serialises v as JSON and writes it to the underlying stream.
	Encode(v any) error
}

// Decoder reads JSON values from an input stream.
type Decoder interface {
	// Decode reads the next JSON value from the underlying stream into v.
	Decode(v any) error
}
