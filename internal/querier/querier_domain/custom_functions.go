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

package querier_domain

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// mergeCustomFunctions adds user-defined custom function signatures into the catalogue's
// default schema.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state to modify.
// Takes engine (EngineTypeSystemPort) which provides type normalisation.
// Takes customFunctions ([]querier_dto.CustomFunctionConfig) which holds the user-defined
// function configurations.
//
// Returns []querier_dto.SourceError which holds warnings for any misconfigured custom
// function, such as an unrecognised nullable behaviour value.
func mergeCustomFunctions(
	catalogue *querier_dto.Catalogue,
	engine EngineTypeSystemPort,
	customFunctions []querier_dto.CustomFunctionConfig,
) []querier_dto.SourceError {
	schema := catalogue.Schemas[catalogue.DefaultSchema]
	if schema == nil {
		return nil
	}
	if schema.Functions == nil {
		schema.Functions = make(map[string][]*querier_dto.FunctionSignature)
	}

	var diagnostics []querier_dto.SourceError
	for _, config := range customFunctions {
		signature, conversionError := convertCustomFunction(config, engine)
		if conversionError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  fmt.Sprintf("custom function %q: %v", config.Name, conversionError),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeInternalNilGuard,
			})
		}
		if signature == nil {
			continue
		}

		functionName := strings.ToLower(config.Name)
		schema.Functions[functionName] = append(schema.Functions[functionName], signature)
	}

	return diagnostics
}

// convertCustomFunction converts a custom function config into a FunctionSignature,
// returning nil if invalid.
//
// Takes config (querier_dto.CustomFunctionConfig) which holds the function definition.
// Takes engine (EngineTypeSystemPort) which provides type normalisation.
//
// Returns *querier_dto.FunctionSignature which is the converted signature, or nil if the
// config is invalid.
// Returns error when the config is structurally usable but carries a misconfigured field,
// such as an unrecognised nullable behaviour value.
func convertCustomFunction(
	config querier_dto.CustomFunctionConfig,
	engine EngineTypeSystemPort,
) (*querier_dto.FunctionSignature, error) {
	if config.Name == "" || config.ReturnType == "" {
		return nil, nil
	}

	arguments := make([]querier_dto.FunctionArgument, len(config.Arguments))
	for argumentIndex, typeName := range config.Arguments {
		arguments[argumentIndex] = querier_dto.FunctionArgument{
			Name: fmt.Sprintf("arg%d", argumentIndex+1),
			Type: engine.NormaliseTypeName(typeName),
		}
	}

	minimumArguments := config.MinArguments
	if minimumArguments == 0 {
		minimumArguments = len(config.Arguments)
	}

	nullableBehaviour, nullableError := parseNullableBehaviour(config.Nullable)

	return &querier_dto.FunctionSignature{
		Name:              config.Name,
		Arguments:         arguments,
		ReturnType:        engine.NormaliseTypeName(config.ReturnType),
		IsAggregate:       config.IsAggregate,
		NullableBehaviour: nullableBehaviour,
		IsVariadic:        config.IsVariadic,
		MinArguments:      minimumArguments,
	}, nullableError
}

// parseNullableBehaviour converts a nullable string value into the corresponding
// FunctionNullableBehaviour enum. An empty value, "null_on_null", or
// "returns_null_on_null" selects the returns-null-on-null default.
//
// Takes nullable (string) which is the raw nullable configuration value.
//
// Returns querier_dto.FunctionNullableBehaviour which is the parsed enum value.
// Returns error when the value is non-empty and unrecognised, so the caller can surface a
// diagnostic rather than silently coercing the misconfiguration to the default.
func parseNullableBehaviour(nullable string) (querier_dto.FunctionNullableBehaviour, error) {
	switch strings.ToLower(nullable) {
	case "", "null_on_null", "returns_null_on_null":
		return querier_dto.FunctionNullableReturnsNullOnNull, nil
	case "never_null":
		return querier_dto.FunctionNullableNeverNull, nil
	case "called_on_null":
		return querier_dto.FunctionNullableCalledOnNull, nil
	default:
		return querier_dto.FunctionNullableReturnsNullOnNull, fmt.Errorf(
			"unrecognised nullable behaviour %q (expected null_on_null, never_null, or called_on_null)",
			nullable,
		)
	}
}
