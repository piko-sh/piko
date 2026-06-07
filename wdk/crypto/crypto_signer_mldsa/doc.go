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

// Package crypto_signer_mldsa provides a post-quantum detached signer over ML-DSA-65.
//
// The scheme is ML-DSA-65 (FIPS 204, NIST security category 3) backed by
// cloudflare/circl. It mirrors crypto_signer_ed25519's shape (GenerateKeyPair, NewSigner
// / NewVerifier from raw bytes, Sign, Verify) so a host can compose the two into a hybrid
// classical+post-quantum signature that stays secure while either algorithm is unbroken.
package crypto_signer_mldsa
