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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestConvertManifestToSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		manifest *annotator_dto.ActionManifest
		name     string
		wantLen  int
		wantNil  bool
	}{
		{
			name:     "nil manifest",
			manifest: nil,
			wantNil:  true,
		},
		{
			name:     "empty actions nil",
			manifest: &annotator_dto.ActionManifest{Actions: nil},
			wantNil:  true,
		},
		{
			name:     "empty actions slice",
			manifest: &annotator_dto.ActionManifest{Actions: []annotator_dto.ActionDefinition{}},
			wantNil:  true,
		},
		{
			name: "single action",
			manifest: &annotator_dto.ActionManifest{
				Actions: []annotator_dto.ActionDefinition{
					{
						Name:           "email.contact",
						TSFunctionName: "emailContact",
						FilePath:       "actions/email/contact.go",
						PackagePath:    "mymod/actions/email",
						StructName:     "ContactAction",
						PackageName:    "email",
						HTTPMethod:     "POST",
						HasError:       true,
						Description:    "Sends a contact email",
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "multiple actions",
			manifest: &annotator_dto.ActionManifest{
				Actions: []annotator_dto.ActionDefinition{
					{Name: "action.one", PackagePath: "mod/actions/one", PackageName: "one", StructName: "OneAction"},
					{Name: "action.two", PackagePath: "mod/actions/two", PackageName: "two", StructName: "TwoAction"},
					{Name: "action.three", PackagePath: "mod/actions/three", PackageName: "three", StructName: "ThreeAction"},
				},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ConvertManifestToSpecs(tt.manifest)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.Len(t, result, tt.wantLen)
		})
	}
}

func TestConvertManifestToSpecs_FieldMapping(t *testing.T) {
	t.Parallel()

	manifest := &annotator_dto.ActionManifest{
		Actions: []annotator_dto.ActionDefinition{
			{
				Name:           "email.contact",
				TSFunctionName: "emailContact",
				FilePath:       "actions/email/contact.go",
				PackagePath:    "mymod/actions/email",
				StructName:     "ContactAction",
				PackageName:    "email",
				HTTPMethod:     "POST",
				HasError:       true,
				Description:    "Sends a contact email",
				Capabilities: annotator_dto.ActionCapabilities{
					HasSSE:            true,
					HasMiddlewares:    true,
					HasRateLimit:      true,
					HasResourceLimits: true,
					HasCacheConfig:    true,
					RateLimit: &annotator_dto.RateLimitConfig{
						RequestsPerMinute: 60,
						BurstSize:         10,
					},
					ResourceLimits: &annotator_dto.ResourceLimitConfig{
						MaxRequestBodySize: 1024,
						Timeout:            5 * time.Second,
					},
					CacheConfig: &annotator_dto.CacheConfig{
						TTL:         10 * time.Minute,
						VaryHeaders: []string{"Accept"},
					},
				},
				CallParams: []annotator_dto.ActionTypeInfo{
					{
						Name:        "ContactInput",
						PackagePath: "mymod/actions/email",
						TSType:      "ContactInput",
						IsPointer:   false,
						Fields: []annotator_dto.ActionFieldInfo{
							{Name: "Email", GoType: "string", TSType: "string", JSONName: "email"},
						},
					},
				},
				OutputType: &annotator_dto.ActionTypeInfo{
					Name:        "ContactOutput",
					PackagePath: "mymod/actions/email",
					TSType:      "ContactOutput",
				},
			},
		},
	}

	specs := ConvertManifestToSpecs(manifest)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "email.contact", spec.Name)
	assert.Equal(t, "emailContact", spec.TSFunctionName)
	assert.Equal(t, "actions/email/contact.go", spec.FilePath)
	assert.Equal(t, "mymod/actions/email", spec.PackagePath)
	assert.Equal(t, "ContactAction", spec.StructName)
	assert.Equal(t, "email", spec.PackageName)
	assert.Equal(t, "POST", spec.HTTPMethod)
	assert.True(t, spec.HasError)
	assert.Equal(t, "Sends a contact email", spec.Description)

	assert.True(t, spec.HasSSE)
	assert.True(t, spec.HasMiddlewares)
	assert.True(t, spec.HasRateLimit)
	assert.True(t, spec.HasResourceLimits)
	assert.True(t, spec.HasCacheConfig)

	require.NotNil(t, spec.RateLimit)
	assert.Equal(t, 60, spec.RateLimit.RequestsPerMinute)
	assert.Equal(t, 10, spec.RateLimit.BurstSize)

	require.NotNil(t, spec.ResourceLimits)
	assert.Equal(t, int64(1024), spec.ResourceLimits.MaxRequestBodySize)
	assert.Equal(t, 5*time.Second, spec.ResourceLimits.Timeout)

	require.NotNil(t, spec.CacheConfig)
	assert.Equal(t, 10*time.Minute, spec.CacheConfig.TTL)
	assert.Equal(t, []string{"Accept"}, spec.CacheConfig.VaryHeaders)

	require.Len(t, spec.CallParams, 1)
	assert.Equal(t, "ContactInput", spec.CallParams[0].Name)
	assert.Equal(t, "ContactInput", spec.CallParams[0].GoType)

	require.NotNil(t, spec.ReturnType)
	assert.Equal(t, "ContactOutput", spec.ReturnType.Name)

	require.Contains(t, spec.Transports, annotator_dto.TransportHTTP)
	require.Contains(t, spec.Transports, annotator_dto.TransportSSE)
}

func TestConvertCallParamsToParamSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validate func(t *testing.T, params []annotator_dto.ParamSpec)
		name     string
		input    []annotator_dto.ActionTypeInfo
		wantLen  int
		wantNil  bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantNil: true,
		},
		{
			name: "basic type",
			input: []annotator_dto.ActionTypeInfo{
				{
					Name:   "ContactInput",
					TSType: "ContactInput",
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, params []annotator_dto.ParamSpec) {
				assert.Equal(t, "ContactInput", params[0].Name)
				assert.Equal(t, "ContactInput", params[0].GoType)
				assert.Equal(t, "ContactInput", params[0].TSType)
				assert.False(t, params[0].Optional)
			},
		},
		{
			name: "pointer type sets optional",
			input: []annotator_dto.ActionTypeInfo{
				{
					Name:      "OptionalInput",
					IsPointer: true,
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, params []annotator_dto.ParamSpec) {
				assert.True(t, params[0].Optional)
			},
		},
		{
			name: "type with fields populates struct",
			input: []annotator_dto.ActionTypeInfo{
				{
					Name:        "StructInput",
					PackagePath: "mymod/types",
					Fields: []annotator_dto.ActionFieldInfo{
						{Name: "Name", GoType: "string", TSType: "string", JSONName: "name"},
					},
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, params []annotator_dto.ParamSpec) {
				require.NotNil(t, params[0].Struct)
				assert.Equal(t, "StructInput", params[0].Struct.Name)
				assert.Equal(t, "mymod/types", params[0].Struct.PackagePath)
				require.Len(t, params[0].Struct.Fields, 1)
				assert.Equal(t, "Name", params[0].Struct.Fields[0].Name)
			},
		},
		{
			name: "empty struct with package path populates struct",
			input: []annotator_dto.ActionTypeInfo{
				{
					Name:        "LogoutInput",
					PackagePath: "mymod/actions/auth",
					Fields:      nil,
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, params []annotator_dto.ParamSpec) {
				require.NotNil(t, params[0].Struct, "Struct must not be nil for empty struct types with package path")
				assert.Equal(t, "LogoutInput", params[0].Struct.Name)
				assert.Equal(t, "mymod/actions/auth", params[0].Struct.PackagePath)
				assert.Nil(t, params[0].Struct.Fields)
			},
		},
		{
			name: "primitive type without package path has nil struct",
			input: []annotator_dto.ActionTypeInfo{
				{
					Name:   "string",
					TSType: "string",
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, params []annotator_dto.ParamSpec) {
				assert.Nil(t, params[0].Struct, "Struct must be nil for primitive types")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := convertCallParamsToParamSpecs(tt.input)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.Len(t, result, tt.wantLen)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConvertActionTypeInfoToTypeSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		info     *annotator_dto.ActionTypeInfo
		validate func(t *testing.T, spec *annotator_dto.TypeSpec)
		name     string
		wantNil  bool
	}{
		{
			name:    "nil info",
			info:    nil,
			wantNil: true,
		},
		{
			name: "populated info",
			info: &annotator_dto.ActionTypeInfo{
				Name:        "MyType",
				PackagePath: "mymod/types",
				Description: "A test type",
			},
			validate: func(t *testing.T, spec *annotator_dto.TypeSpec) {
				assert.Equal(t, "MyType", spec.Name)
				assert.Equal(t, "mymod/types", spec.PackagePath)
				assert.Equal(t, "A test type", spec.Description)
				assert.Nil(t, spec.Fields)
			},
		},
		{
			name: "with fields",
			info: &annotator_dto.ActionTypeInfo{
				Name: "WithFields",
				Fields: []annotator_dto.ActionFieldInfo{
					{Name: "ID", GoType: "int", TSType: "number", JSONName: "id"},
					{Name: "Name", GoType: "string", TSType: "string", JSONName: "name"},
				},
			},
			validate: func(t *testing.T, spec *annotator_dto.TypeSpec) {
				require.Len(t, spec.Fields, 2)
				assert.Equal(t, "ID", spec.Fields[0].Name)
				assert.Equal(t, "Name", spec.Fields[1].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := convertActionTypeInfoToTypeSpec(tt.info)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConvertFieldsToFieldSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fields  []annotator_dto.ActionFieldInfo
		wantNil bool
		wantLen int
	}{
		{
			name:    "nil fields",
			fields:  nil,
			wantNil: true,
		},
		{
			name:    "empty fields",
			fields:  []annotator_dto.ActionFieldInfo{},
			wantNil: true,
		},
		{
			name: "single field with all attributes",
			fields: []annotator_dto.ActionFieldInfo{
				{
					Name:        "Email",
					GoType:      "string",
					TSType:      "string",
					JSONName:    "email",
					Validation:  "required,email",
					Description: "User email address",
					Optional:    false,
				},
			},
			wantLen: 1,
		},
		{
			name: "multiple fields",
			fields: []annotator_dto.ActionFieldInfo{
				{Name: "ID", GoType: "int", TSType: "number", JSONName: "id"},
				{Name: "Name", GoType: "string", TSType: "string", JSONName: "name"},
				{Name: "Active", GoType: "bool", TSType: "boolean", JSONName: "active", Optional: true},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := convertFieldsToFieldSpecs(tt.fields)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.Len(t, result, tt.wantLen)

			for i, f := range tt.fields {
				assert.Equal(t, f.Name, result[i].Name)
				assert.Equal(t, f.GoType, result[i].GoType)
				assert.Equal(t, f.TSType, result[i].TSType)
				assert.Equal(t, f.JSONName, result[i].JSONName)
				assert.Equal(t, f.Validation, result[i].Validation)
				assert.Equal(t, f.Description, result[i].Description)
				assert.Equal(t, f.Optional, result[i].Optional)
			}
		})
	}
}

func TestDeriveTransports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		caps    annotator_dto.ActionCapabilities
		wantLen int
		wantSSE bool
	}{
		{
			name:    "HTTP only by default",
			caps:    annotator_dto.ActionCapabilities{},
			wantLen: 1,
		},
		{
			name:    "with SSE",
			caps:    annotator_dto.ActionCapabilities{HasSSE: true},
			wantLen: 2,
			wantSSE: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := deriveTransports(tt.caps)

			require.Len(t, result, tt.wantLen)
			assert.Contains(t, result, annotator_dto.TransportHTTP)

			if tt.wantSSE {
				assert.Contains(t, result, annotator_dto.TransportSSE)
			}
		})
	}
}

func TestConvertRateLimitConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, convertRateLimitConfig(nil))
	})

	t.Run("populated config", func(t *testing.T) {
		t.Parallel()

		config := &annotator_dto.RateLimitConfig{
			RequestsPerMinute: 120,
			BurstSize:         20,
			HasCustomKeyFunc:  true,
		}

		result := convertRateLimitConfig(config)

		require.NotNil(t, result)
		assert.Equal(t, 120, result.RequestsPerMinute)
		assert.Equal(t, 20, result.BurstSize)
		assert.True(t, result.HasCustomKeyFunc)
	})
}

func TestConvertResourceLimitConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, convertResourceLimitConfig(nil))
	})

	t.Run("populated config", func(t *testing.T) {
		t.Parallel()

		config := &annotator_dto.ResourceLimitConfig{
			MaxRequestBodySize:   1024 * 1024,
			MaxResponseSize:      2048 * 1024,
			Timeout:              30 * time.Second,
			SlowThreshold:        5 * time.Second,
			MaxConcurrent:        10,
			MaxMemoryUsage:       512 * 1024 * 1024,
			MaxSSEDuration:       24 * time.Hour,
			SSEHeartbeatInterval: 30 * time.Second,
		}

		result := convertResourceLimitConfig(config)

		require.NotNil(t, result)
		assert.Equal(t, int64(1024*1024), result.MaxRequestBodySize)
		assert.Equal(t, int64(2048*1024), result.MaxResponseSize)
		assert.Equal(t, 30*time.Second, result.Timeout)
		assert.Equal(t, 5*time.Second, result.SlowThreshold)
		assert.Equal(t, 10, result.MaxConcurrent)
		assert.Equal(t, int64(512*1024*1024), result.MaxMemoryUsage)
		assert.Equal(t, 24*time.Hour, result.MaxSSEDuration)
		assert.Equal(t, 30*time.Second, result.SSEHeartbeatInterval)
	})
}

func TestConvertCacheConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, convertCacheConfig(nil))
	})

	t.Run("populated config", func(t *testing.T) {
		t.Parallel()

		config := &annotator_dto.CacheConfig{
			TTL:              15 * time.Minute,
			VaryHeaders:      []string{"Accept", "Accept-Language"},
			HasCustomKeyFunc: true,
		}

		result := convertCacheConfig(config)

		require.NotNil(t, result)
		assert.Equal(t, 15*time.Minute, result.TTL)
		assert.Equal(t, []string{"Accept", "Accept-Language"}, result.VaryHeaders)
		assert.True(t, result.HasCustomKeyFunc)
	})
}
