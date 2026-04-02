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

package config_domain

import (
	"context"
	"fmt"
	"os"
)

// EnvResolver resolves placeholders like "env:VAR_NAME" by looking up the
// matching environment variable. It implements the Resolver interface.
type EnvResolver struct{}

var _ = Resolver(&EnvResolver{})

// GetPrefix returns the "env:" prefix.
//
// Returns string which is the prefix used to identify environment variables.
func (*EnvResolver) GetPrefix() string {
	return "env:"
}

// Resolve looks up the given value as an environment variable.
//
// Takes value (string) which is the name of the environment variable to find.
//
// Returns string which is the value of the environment variable.
// Returns error when the environment variable is not set.
func (*EnvResolver) Resolve(_ context.Context, value string) (string, error) {
	envValue, ok := os.LookupEnv(value)
	if !ok {
		return "", fmt.Errorf("environment variable %q not found", value)
	}
	return envValue, nil
}
