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

package capabilities

import (
	"fmt"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/capabilities/capabilities_functions"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/video/video_domain"
)

// Service is the capability service interface for registering and executing
// capabilities.
type Service = capabilities_domain.CapabilityService

// capabilityFunc is a function type that provides a specific capability.
type capabilityFunc = capabilities_domain.CapabilityFunc

// dependencies holds optional dependencies for the service.
type dependencies struct {
	// compiler provides component compilation; nil disables this feature.
	compiler compiler_domain.CompilerService

	// imageTransformer handles image transformations; nil disables this
	// capability.
	imageTransformer image_domain.Service

	// videoTranscoder provides video processing; nil disables video capabilities.
	videoTranscoder video_domain.Service
}

// Option configures dependencies using the functional options pattern.
type Option func(*dependencies)

// builtinCapabilities maps built-in capability types to their handler
// functions.
var builtinCapabilities = map[capabilities_dto.Capability]capabilityFunc{
	capabilities_dto.CapabilityCompressGzip:   capabilities_functions.Gzip(),
	capabilities_dto.CapabilityCompressBrotli: capabilities_functions.Brotli(),
	capabilities_dto.CapabilityCopyJS:         capabilities_functions.CopyJS(),
	capabilities_dto.CapabilityMinifyCSS:      capabilities_functions.MinifyCSS(),
	capabilities_dto.CapabilityMinifyJS:       capabilities_functions.MinifyJavascript(),
	capabilities_dto.CapabilityMinifySVG:      capabilities_functions.MinifySVG(),
}

// WithCompiler is a functional option that sets the compiler service.
//
// Takes c (CompilerService) which provides compilation functionality.
//
// Returns Option which sets the compiler on the dependencies.
func WithCompiler(c compiler_domain.CompilerService) Option {
	return func(d *dependencies) {
		d.compiler = c
	}
}

// WithImageProvider is a functional option that injects the image service.
//
// Takes service (image_domain.Service) which provides image transformation.
//
// Returns Option which sets the image provider dependency.
func WithImageProvider(service image_domain.Service) Option {
	return func(d *dependencies) {
		d.imageTransformer = service
	}
}

// WithVideoProvider is a functional option that sets the video provider.
//
// Takes service (video_domain.Service) which provides video transcoding.
//
// Returns Option which sets the video provider dependency.
func WithVideoProvider(service video_domain.Service) Option {
	return func(d *dependencies) {
		d.videoTranscoder = service
	}
}

// NewServiceWithBuiltins creates a new CapabilityService, registers all
// built-in capabilities, and applies any provided optional dependencies.
//
// Takes opts (...Option) which configures optional dependencies such as
// compiler and transformer.
//
// Returns Service which is the configured capability service with all
// built-in capabilities registered.
// Returns error when a capability fails to register.
func NewServiceWithBuiltins(opts ...Option) (Service, error) {
	deps := applyOptions(opts)
	service := capabilities_domain.NewCapabilityService(calculateCapacity(deps))

	if err := registerBuiltins(service); err != nil {
		return nil, fmt.Errorf("registering built-in capabilities: %w", err)
	}

	if err := registerOptionalCapabilities(service, deps); err != nil {
		return nil, fmt.Errorf("registering optional capabilities: %w", err)
	}

	return service, nil
}

// applyOptions applies functional options to create dependencies.
//
// Takes opts ([]Option) which specifies the functional options to apply.
//
// Returns *dependencies which contains the configured dependency values.
func applyOptions(opts []Option) *dependencies {
	deps := &dependencies{}
	for _, opt := range opts {
		opt(deps)
	}
	return deps
}

// calculateCapacity returns the initial capacity based on dependencies.
//
// Takes deps (*dependencies) which provides the optional service components.
//
// Returns int which is the total capacity needed for all capabilities.
func calculateCapacity(deps *dependencies) int {
	capacity := len(builtinCapabilities)
	if deps.compiler != nil {
		capacity++
	}
	if deps.imageTransformer != nil {
		capacity++
	}
	if deps.videoTranscoder != nil {
		capacity += 2
	}
	return capacity
}

// registerBuiltins registers all core built-in capabilities.
//
// Takes service (Service) which provides the registry for capabilities.
//
// Returns error when a built-in capability fails to register.
func registerBuiltins(service Service) error {
	for name, capability := range builtinCapabilities {
		if err := service.Register(name.String(), capability); err != nil {
			return fmt.Errorf("failed to register built-in capability '%s': %w", name, err)
		}
	}
	return nil
}

// registerOptionalCapabilities registers capabilities based on injected
// dependencies.
//
// Takes service (Service) which receives the capability registrations.
// Takes deps (*dependencies) which provides optional capability
// implementations.
//
// Returns error when a capability registration fails.
func registerOptionalCapabilities(service Service, deps *dependencies) error {
	if deps.compiler != nil {
		if err := service.Register(
			capabilities_dto.CapabilityCompileComponent.String(),
			capabilities_functions.CompileComponent(deps.compiler),
		); err != nil {
			return fmt.Errorf("failed to register 'compile-component' capability: %w", err)
		}
	}

	if deps.imageTransformer != nil {
		if err := service.Register(
			capabilities_dto.CapabilityImageTransform.String(),
			capabilities_functions.ImageTransform(deps.imageTransformer),
		); err != nil {
			return fmt.Errorf("failed to register 'image-transform' capability: %w", err)
		}
	}

	if deps.videoTranscoder != nil {
		if err := registerVideoCapabilities(service, deps.videoTranscoder); err != nil {
			return fmt.Errorf("registering video capabilities: %w", err)
		}
	}

	return nil
}

// registerVideoCapabilities registers video-related capabilities.
//
// Takes service (Service) which provides the capability registration interface.
// Takes transcoder (video_domain.Service) which handles video processing.
//
// Returns error when capability registration fails.
func registerVideoCapabilities(service Service, transcoder video_domain.Service) error {
	if err := service.Register(
		capabilities_dto.CapabilityVideoTranscode.String(),
		capabilities_functions.VideoTranscode(transcoder),
	); err != nil {
		return fmt.Errorf("failed to register 'video-transcode' capability: %w", err)
	}

	if err := service.Register(
		capabilities_dto.CapabilityVideoThumbnail.String(),
		capabilities_functions.VideoThumbnail(transcoder),
	); err != nil {
		return fmt.Errorf("failed to register 'video-thumbnail' capability: %w", err)
	}

	return nil
}
