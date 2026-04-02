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

package config_resolver_aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"piko.sh/piko/wdk/json"
	pikoconfig "piko.sh/piko/wdk/config"
)

var _ = pikoconfig.Resolver(&Resolver{})

// Resolver fetches secrets from AWS Secrets Manager.
// It implements the config.Resolver interface.
//
// Circuit breaker protection is provided by the config Loader layer,
// not by this resolver directly.
//
// It supports two formats for secrets:
//   - Plain string: "aws-secret:my/secret/name"
//   - JSON key: "aws-secret:my/json/secret#key" fetches the value of "key"
//     from the JSON secret.
type Resolver struct {
	// client is the AWS Secrets Manager client used to fetch secret values.
	client *secretsmanager.Client
}

// NewResolver creates and initialises a new AWS Secrets Manager resolver.
//
// It uses the default AWS credential chain (environment variables, shared
// credentials file, IAM roles). Ensure the application has the
// `secretsmanager:GetSecretValue` IAM permission.
//
// Returns *Resolver which is the configured resolver ready for use.
// Returns error when the AWS configuration cannot be loaded.
func NewResolver(ctx context.Context) (*Resolver, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Resolver{
		client: secretsmanager.NewFromConfig(awsConfig),
	}, nil
}

// GetPrefix returns the prefix this resolver handles.
//
// Returns string which is the prefix "aws-secret:" used to identify secrets
// that should be resolved by this resolver.
func (*Resolver) GetPrefix() string {
	return "aws-secret:"
}

// Resolve fetches the secret value from AWS Secrets Manager.
//
// Takes value (string) which specifies the secret ID, optionally followed by
// "#key" to extract a specific field from a JSON secret.
//
// Returns string which is the resolved secret value.
// Returns error when the secret format is invalid, the secret is not found,
// the secret is binary rather than a string, or JSON parsing fails.
func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
	secretID, jsonKey, _ := strings.Cut(value, "#")
	if secretID == "" {
		return "", fmt.Errorf("invalid secret format: %q; must not be empty", value)
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}
	awsResult, err := r.client.GetSecretValue(ctx, input)
	if err != nil {
		if _, ok := errors.AsType[*types.ResourceNotFoundException](err); ok {
			return "", fmt.Errorf("secret %q not found in AWS Secrets Manager", secretID)
		}
		return "", fmt.Errorf("failed to get secret %q from AWS: %w", secretID, err)
	}
	if awsResult.SecretString == nil {
		return "", fmt.Errorf("secret %q is binary, not a string", secretID)
	}
	secretValue := *awsResult.SecretString

	if jsonKey != "" {
		var data map[string]any
		if err := json.UnmarshalString(secretValue, &data); err != nil {
			return "", fmt.Errorf("failed to unmarshal JSON secret %q: %w", secretID, err)
		}
		value, exists := data[jsonKey]
		if !exists {
			return "", fmt.Errorf("key %q not found in JSON secret %q", jsonKey, secretID)
		}
		secretValue = fmt.Sprintf("%v", value)
	}

	return secretValue, nil
}

// Register creates a new AWS Secrets Manager resolver and registers it in the
// global resolver registry. This is a convenience function equivalent to
// [NewResolver] followed by [config.RegisterResolver].
//
// Returns error when resolver creation or registration fails.
//
// Example:
//
//	func init() {
//	    if err := config_resolver_aws.Register(context.Background()); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func Register(ctx context.Context) error {
	resolver, err := NewResolver(ctx)
	if err != nil {
		return err
	}
	return pikoconfig.RegisterResolver(resolver)
}
