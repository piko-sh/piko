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

// Package generator_adapters implements driven adapters for the generator
// domain.
//
// It implements the ports defined in generator_domain, covering file
// I/O, manifest serialisation, code emission, action code generation,
// and build artefact production. All
// file operations use sandboxed access to prevent path traversal attacks
// and atomic writes to prevent corruption from concurrent access.
package generator_adapters
