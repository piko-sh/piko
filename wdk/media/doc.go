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

// Package media provides a provider-agnostic framework for image and
// video processing. It uses hexagonal architecture (ports and adapters)
// to support pluggable processing backends, allowing you to swap
// implementations without changing application code.
//
// Media services are optional and require explicit provider
// configuration.
//
// # Without providers
//
// If no providers are configured, media services are nil and
// components like piko:img and piko:video gracefully degrade to
// basic HTML output without responsive images or video transcoding.
//
// # Image transformations
//
// Use the fluent builder API to transform images at runtime:
//
//	builder, err := media.NewTransformBuilderFromDefault(reader)
//	if err != nil {
//	    return err
//	}
//	result, err := builder.
//	    Size(800, 600).
//	    Format("webp").
//	    Quality(80).
//	    Do(ctx)
//
// Predefined variants allow reusable transformation specifications:
//
//	config := media.Image().
//	    Provider("vips", vipsProvider).
//	    WithVariant("thumb",
//	        media.Variant().Size(200, 200).Cover().Build(),
//	    ).
//	    Build()
//
// # Available providers
//
// Image providers are available in the image_provider_* sub-packages.
// Video providers are available in the video_provider_* sub-packages.
//
// # Thread safety
//
// All exported functions and builder types are safe for concurrent
// use. The underlying providers manage their own concurrency.
package media
