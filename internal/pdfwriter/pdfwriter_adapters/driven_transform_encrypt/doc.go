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

// Package driven_transform_encrypt implements PDF encryption as a
// post-processing transformer.
//
// It parses existing PDF bytes, encrypts all string and stream objects
// using AES-256-CBC (PDF 2.0, V=5 R=6), and adds an /Encrypt dictionary
// to the trailer. The implementation follows ISO 32000-2 section 7.6 for
// encryption key generation, password hashing, and object encryption.
package driven_transform_encrypt
