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

package generator_adapters

import (
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// ConvertManifestToSpecs converts an ActionManifest to a slice of ActionSpec.
//
// This bridges the annotator's auto-discovered actions to the existing
// code generation infrastructure.
//
// Takes manifest (*annotator_dto.ActionManifest) which contains discovered
// actions.
//
// Returns []annotator_dto.ActionSpec ready for the emitters.
func ConvertManifestToSpecs(manifest *annotator_dto.ActionManifest) []annotator_dto.ActionSpec {
	if manifest == nil || len(manifest.Actions) == 0 {
		return nil
	}

	specs := make([]annotator_dto.ActionSpec, 0, len(manifest.Actions))
	for i := range manifest.Actions {
		specs = append(specs, convertActionDefinitionToSpec(&manifest.Actions[i]))
	}
	return specs
}

// convertActionDefinitionToSpec converts a single ActionDefinition to ActionSpec.
//
// Takes action (*annotator_dto.ActionDefinition) which is the action definition
// to convert.
//
// Returns annotator_dto.ActionSpec which contains the converted action
// specification ready for code generation.
func convertActionDefinitionToSpec(action *annotator_dto.ActionDefinition) annotator_dto.ActionSpec {
	spec := annotator_dto.ActionSpec{
		Name:           action.Name,
		TSFunctionName: action.TSFunctionName,
		FilePath:       action.FilePath,
		PackagePath:    action.PackagePath,

		StructName:  action.StructName,
		PackageName: action.PackageName,

		CallParams: convertCallParamsToParamSpecs(action.CallParams),
		ReturnType: convertActionTypeInfoToTypeSpec(action.OutputType),
		HasError:   action.HasError,

		HasSSE:     action.Capabilities.HasSSE,
		Transports: deriveTransports(action.Capabilities),

		HTTPMethod:        action.HTTPMethod,
		HasMiddlewares:    action.Capabilities.HasMiddlewares,
		HasRateLimit:      action.Capabilities.HasRateLimit,
		RateLimit:         convertRateLimitConfig(action.Capabilities.RateLimit),
		HasResourceLimits: action.Capabilities.HasResourceLimits,
		ResourceLimits:    convertResourceLimitConfig(action.Capabilities.ResourceLimits),
		HasCacheConfig:    action.Capabilities.HasCacheConfig,
		CacheConfig:       convertCacheConfig(action.Capabilities.CacheConfig),

		Description: action.Description,
	}

	return spec
}

// convertCallParamsToParamSpecs converts a slice of ActionTypeInfo into
// ParamSpec entries for code generation.
//
// Takes params ([]annotator_dto.ActionTypeInfo) which are the call parameters.
//
// Returns []annotator_dto.ParamSpec which contains the converted parameter
// specifications.
func convertCallParamsToParamSpecs(params []annotator_dto.ActionTypeInfo) []annotator_dto.ParamSpec {
	if len(params) == 0 {
		return nil
	}

	specs := make([]annotator_dto.ParamSpec, 0, len(params))
	for i := range params {
		paramName := params[i].ParamName
		if paramName == "" {
			paramName = params[i].Name
		}

		isFileUpload := isPikoSpecialType(params[i].Name, params[i].PackagePath, "FileUpload")
		isRawBody := isPikoSpecialType(params[i].Name, params[i].PackagePath, "RawBody")

		tsType := params[i].TSType
		if isFileUpload {
			tsType = "File"
		}

		var structSpec *annotator_dto.TypeSpec
		if !isFileUpload && !isRawBody && params[i].PackagePath != "" {
			structSpec = convertActionTypeInfoToTypeSpec(&params[i])
		}

		specs = append(specs, annotator_dto.ParamSpec{
			Name:              paramName,
			GoType:            params[i].Name,
			TSType:            tsType,
			JSONName:          paramName,
			Optional:          params[i].IsPointer,
			Struct:            structSpec,
			IsFileUpload:      isFileUpload,
			IsFileUploadSlice: false,
			IsRawBody:         isRawBody,
		})
	}
	return specs
}

// isPikoSpecialType checks if a type is a special piko type (FileUpload,
// RawBody) by matching the type name and ensuring the package path is within
// piko.
//
// Takes typeName (string) which is the name of the type to check.
// Takes packagePath (string) which is the package path where the type is defined.
// Takes targetName (string) which is the special type name to match against.
//
// Returns bool which is true if the type matches the target name and the
// package path starts with "piko.sh/piko".
func isPikoSpecialType(typeName, packagePath, targetName string) bool {
	return typeName == targetName && strings.HasPrefix(packagePath, "piko.sh/piko")
}

// convertActionTypeInfoToTypeSpec converts ActionTypeInfo to TypeSpec.
//
// Takes info (*annotator_dto.ActionTypeInfo) which provides the type
// information to convert.
//
// Returns *annotator_dto.TypeSpec which is the converted type
// specification, or nil if info is nil.
func convertActionTypeInfoToTypeSpec(info *annotator_dto.ActionTypeInfo) *annotator_dto.TypeSpec {
	if info == nil {
		return nil
	}

	return &annotator_dto.TypeSpec{
		Name:        info.Name,
		PackagePath: info.PackagePath,
		PackageName: info.PackageName,
		Fields:      convertFieldsToFieldSpecs(info.Fields),
		Description: info.Description,
	}
}

// convertFieldsToFieldSpecs converts a slice of ActionFieldInfo to FieldSpec.
//
// Takes fields ([]annotator_dto.ActionFieldInfo) which contains the action
// field information to convert.
//
// Returns []annotator_dto.FieldSpec which contains the converted field
// specifications, or nil if the input slice is empty.
func convertFieldsToFieldSpecs(fields []annotator_dto.ActionFieldInfo) []annotator_dto.FieldSpec {
	if len(fields) == 0 {
		return nil
	}

	specs := make([]annotator_dto.FieldSpec, len(fields))
	for i, f := range fields {
		specs[i] = annotator_dto.FieldSpec{
			NestedType:  convertActionTypeInfoToTypeSpec(f.NestedType),
			Name:        f.Name,
			GoType:      f.GoType,
			TSType:      f.TSType,
			JSONName:    f.JSONName,
			Validation:  f.Validation,
			Optional:    f.Optional,
			Description: f.Description,
		}
	}
	return specs
}

// deriveTransports determines the supported transports based on capabilities.
//
// Takes caps (annotator_dto.ActionCapabilities) which specifies the action
// capabilities to check for transport support.
//
// Returns []annotator_dto.Transport which contains the list of supported
// transports, always including HTTP and optionally SSE.
func deriveTransports(caps annotator_dto.ActionCapabilities) []annotator_dto.Transport {
	transports := []annotator_dto.Transport{annotator_dto.TransportHTTP}

	if caps.HasSSE {
		transports = append(transports, annotator_dto.TransportSSE)
	}
	return transports
}

// convertRateLimitConfig converts RateLimitConfig to RateLimitSpec.
//
// Takes config (*annotator_dto.RateLimitConfig) which specifies the rate limit
// settings to convert.
//
// Returns *annotator_dto.RateLimitSpec which contains the converted rate
// limit specification, or nil if config is nil.
func convertRateLimitConfig(config *annotator_dto.RateLimitConfig) *annotator_dto.RateLimitSpec {
	if config == nil {
		return nil
	}
	return &annotator_dto.RateLimitSpec{
		RequestsPerMinute: config.RequestsPerMinute,
		BurstSize:         config.BurstSize,
		HasCustomKeyFunc:  config.HasCustomKeyFunc,
	}
}

// convertResourceLimitConfig converts ResourceLimitConfig to ResourceLimitSpec.
//
// Takes config (*annotator_dto.ResourceLimitConfig) which specifies the resource
// limits to convert.
//
// Returns *annotator_dto.ResourceLimitSpec which contains the converted
// resource limits, or nil when config is nil.
func convertResourceLimitConfig(config *annotator_dto.ResourceLimitConfig) *annotator_dto.ResourceLimitSpec {
	if config == nil {
		return nil
	}
	return &annotator_dto.ResourceLimitSpec{
		MaxRequestBodySize:   config.MaxRequestBodySize,
		MaxResponseSize:      config.MaxResponseSize,
		Timeout:              config.Timeout,
		SlowThreshold:        config.SlowThreshold,
		MaxConcurrent:        config.MaxConcurrent,
		MaxMemoryUsage:       config.MaxMemoryUsage,
		MaxSSEDuration:       config.MaxSSEDuration,
		SSEHeartbeatInterval: config.SSEHeartbeatInterval,
	}
}

// convertCacheConfig converts CacheConfig to CacheConfigSpec.
//
// Takes config (*annotator_dto.CacheConfig) which specifies the cache settings.
//
// Returns *annotator_dto.CacheConfigSpec which is the converted spec,
// or nil if config is nil.
func convertCacheConfig(config *annotator_dto.CacheConfig) *annotator_dto.CacheConfigSpec {
	if config == nil {
		return nil
	}
	return &annotator_dto.CacheConfigSpec{
		TTL:              config.TTL,
		VaryHeaders:      config.VaryHeaders,
		HasCustomKeyFunc: config.HasCustomKeyFunc,
	}
}
