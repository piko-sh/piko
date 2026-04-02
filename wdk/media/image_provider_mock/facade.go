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

package image_provider_mock

import (
	"piko.sh/piko/internal/image/image_adapters/transformer_mock"
)

// Provider is a thread-safe, in-memory implementation of ImageTransformerPort.
// It is designed for unit and integration testing of the image service,
// allowing call inspection and simulation of both successful transformations
// and errors.
type Provider = transformer_mock.Provider

// TransformCall records the parameters passed to a single call of the Transform
// method, letting tests inspect what the service is asking the transformer to
// do.
type TransformCall = transformer_mock.TransformCall

// NewProvider creates a new mock image transformer for use in tests.
//
// Returns *Provider which is ready for use.
var NewProvider = transformer_mock.NewProvider
