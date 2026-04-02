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

// Package crypto_domain defines the encryption port interfaces and service
// logic for the Piko framework.
//
// It coordinates envelope encryption, key rotation, streaming encryption
// for large files, and automatic re-encryption on key rotation.
//
// The service supports two batch encryption modes. Envelope encryption
// (the default) uses a single KMS call to generate a data key and then
// encrypts all items locally, keeping the data key briefly in secure
// memory. Direct KMS mode calls the KMS for each item individually so
// that no keys enter application memory.
//
// For large files, the streaming encrypt/decrypt operations provide
// constant memory usage regardless of file size.
//
// All terminal operations honour context cancellation and deadlines.
// All methods on the crypto service are safe for concurrent use.
package crypto_domain
