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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package provider_union presents a read-only base and a writable overlay as one storage
// provider.
//
// Reads consult the overlay first and fall through to the base, while writes always go to
// the overlay, which keeps embedded release assets servable without copying them.
// Optional capabilities that are not part of StorageProviderPort, such as key listing and
// batch removal, are forwarded to whichever layer implements them. A union with no
// overlay rejects writes with ErrReadOnly, and one with neither layer rejects reads with
// ErrNoLayer.
package provider_union
