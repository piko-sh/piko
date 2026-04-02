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

// Package driven_transform_pades adds PAdES (PDF Advanced Electronic
// Signatures) digital signatures as a post-processing transformer.
//
// It creates a CMS/PKCS#7 signature conforming to PAdES profiles, reserves
// space for the SignedData structure, computes the hash over the byte
// ranges excluding the signature contents, signs with the provided private
// key, and embeds the DER-encoded signature into the PDF. Conformance
// levels from B-B (basic) through B-LTA (long-term archival) are
// supported.
package driven_transform_pades
