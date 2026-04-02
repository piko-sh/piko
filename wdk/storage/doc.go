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

// Package storage provides a provider-agnostic framework for
// object storage operations.
//
// It abstracts away differences between storage backends behind a
// unified API, allowing applications to switch providers without
// changing business logic. Operations are composed using a fluent
// builder pattern:
//
//	err := storage.NewUploadBuilder(service, reader).
//	    Key("documents/report.pdf").
//	    ContentType("application/pdf").
//	    Do(ctx)
//
//	request := storage.NewRequestBuilder(service, "default", "report.pdf")
//	data, err := request.Get(ctx)
//	defer data.Close()
//
// Provider and transformer implementations live in sub-packages.
// Stream transformers (compression, encryption) run automatically
// during uploads and downloads when registered with the service.
//
// All [Service] methods are safe for concurrent use. Builders
// should not be shared between goroutines.
package storage
