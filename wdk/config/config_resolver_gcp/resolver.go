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

package config_resolver_gcp

import (
	"context"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"piko.sh/piko/wdk/config"
	"piko.sh/piko/wdk/json"
)

var _ = config.Resolver(&Resolver{})

// Resolver fetches secrets from Google Cloud Secret Manager.
// It implements the config.Resolver interface.
//
// Circuit breaker protection is provided by the config Loader layer,
// not by this resolver directly.
//
// The value must be the full resource name of the secret version, for example:
// "projects/my-project-id/secrets/my-secret/versions/latest"
//
// It supports two formats for secrets:
//   - Plain string: "gcp-secret:projects/.../versions/latest"
//   - JSON key: "gcp-secret:projects/.../versions/latest#key" extracts "key"
//     from a JSON secret.
type Resolver struct {
	// client accesses GCP Secret Manager to fetch secret versions.
	client *secretmanager.Client
}

// NewResolver creates and initialises a new GCP Secret Manager resolver.
//
// It uses Application Default Credentials (ADC) for authentication.
// Ensure the application has the secretmanager.secretAccessor IAM role.
//
// Returns *Resolver which is the configured resolver ready for use.
// Returns error when the GCP Secret Manager client cannot be created.
func NewResolver(ctx context.Context) (*Resolver, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Secret Manager client: %w", err)
	}

	return &Resolver{
		client: client,
	}, nil
}

// GetPrefix returns the prefix this resolver handles.
//
// Returns string which is the GCP secret prefix "gcp-secret:".
func (*Resolver) GetPrefix() string {
	return "gcp-secret:"
}

// Resolve fetches the secret value from GCP Secret Manager.
//
// Takes value (string) which specifies the secret resource name, optionally
// followed by "#key" to extract a specific field from a JSON secret.
//
// Returns string which is the secret value or extracted JSON field.
// Returns error when the secret format is invalid, the secret is not found,
// or JSON parsing fails when a key is specified.
func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
	secretName, jsonKey, _ := strings.Cut(value, "#")
	if secretName == "" {
		return "", fmt.Errorf("invalid secret format: %q; must not be empty", value)
	}

	request := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}
	gcpResult, err := r.client.AccessSecretVersion(ctx, request)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", fmt.Errorf("secret %q not found in GCP Secret Manager", secretName)
		}
		return "", fmt.Errorf("failed to get secret %q from GCP: %w", secretName, err)
	}
	secretValue := string(gcpResult.Payload.Data)

	if jsonKey != "" {
		var data map[string]any
		if err := json.UnmarshalString(secretValue, &data); err != nil {
			return "", fmt.Errorf("failed to unmarshal JSON secret %q: %w", secretName, err)
		}
		value, exists := data[jsonKey]
		if !exists {
			return "", fmt.Errorf("key %q not found in JSON secret %q", jsonKey, secretName)
		}
		secretValue = fmt.Sprintf("%v", value)
	}

	return secretValue, nil
}

// Register creates a new GCP Secret Manager resolver and registers it in the
// global resolver registry. This is a convenience function equivalent to
// [NewResolver] followed by [config.RegisterResolver].
//
// Returns error when resolver creation or registration fails.
//
// Example:
//
//	func init() {
//	    if err := config_resolver_gcp.Register(context.Background()); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func Register(ctx context.Context) error {
	resolver, err := NewResolver(ctx)
	if err != nil {
		return err
	}
	return config.RegisterResolver(resolver)
}
