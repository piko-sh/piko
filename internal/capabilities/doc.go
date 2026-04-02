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

// Package capabilities provides a registry for named capabilities that can be
// invoked to transform content.
//
// Capabilities are pluggable functions for tasks such as compression,
// minification, image processing, video transcoding, and component
// compilation. Factory functions here create a service with all built-in
// capabilities registered. Some capabilities require optional provider
// dependencies passed via functional options.
//
// Sub-packages contain the domain interfaces, data transfer objects, and
// built-in capability implementations.
//
// # Usage
//
// Create a service with built-in capabilities:
//
//	service, err := capabilities.NewServiceWithBuiltins(
//	    capabilities.WithCompiler(compiler),
//	    capabilities.WithImageProvider(imageService),
//	)
//	if err != nil {
//	    return err
//	}
//
//	// Execute a capability
//	output, err := service.Execute(ctx, "compress-gzip", inputReader, nil)
//
// # Thread safety
//
// The Service implementation is safe for concurrent use. Capabilities can be
// registered and executed from multiple goroutines.
package capabilities
