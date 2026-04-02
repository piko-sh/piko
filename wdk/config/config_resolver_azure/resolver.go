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

package config_resolver_azure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"piko.sh/piko/wdk/config"
	"piko.sh/piko/wdk/json"
)

var _ config.Resolver = (*Resolver)(nil)

// Resolver fetches secrets from Azure Key Vault.
// It implements the config.Resolver interface.
//
// Circuit breaker protection is provided by the config Loader layer,
// not by this resolver directly.
//
// AUTHENTICATION:
// The resolver uses the Azure SDK's DefaultAzureCredential, which provides
// an automatic authentication chain. It will try Managed Identity,
// CLI credentials, environment variables, etc.
//
// USAGE FORMAT:
// The value is expected in the format:
// "azure-kv:vault-name/secret-name[#json-key]"
//
// Examples:
//  1. Plain secret: "azure-kv:my-prod-vault/database-connection-string"
//  2. JSON key: "azure-kv:my-prod-vault/api-keys#primary" -> fetches
//     "primary" from a JSON secret.
type Resolver struct {
	// credential authenticates requests to Azure Key Vault.
	credential azcore.TokenCredential

	// clientCache stores clients per Key Vault URL.
	clientCache map[string]*azsecrets.Client

	// clientMutex guards access to the clientCache.
	clientMutex sync.Mutex
}

// NewResolver creates and initialises a new Azure Key Vault resolver.
//
// Returns *Resolver which is ready to resolve secrets from Azure Key
// Vault.
// Returns error when the Azure default credential cannot be created.
func NewResolver() (*Resolver, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure default credential: %w", err)
	}

	return &Resolver{
		credential:  cred,
		clientCache: make(map[string]*azsecrets.Client),
	}, nil
}

// GetPrefix returns the prefix this resolver handles.
//
// Returns string which is the "azure-kv:" prefix identifying Azure Key Vault
// secrets.
func (*Resolver) GetPrefix() string {
	return "azure-kv:"
}

// Resolve fetches the secret value from Azure Key Vault.
//
// Takes value (string) which specifies the secret reference in the format
// "vault-name/secret-name" or "vault-name/secret-name#json-key" to extract
// a specific key from a JSON secret.
//
// Returns string which is the secret value or extracted JSON field.
// Returns error when the format is invalid, the secret is not found, or
// JSON parsing fails.
func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
	secretRef, jsonKey, _ := strings.Cut(value, "#")
	parts := strings.SplitN(secretRef, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Azure Key Vault format: %q; expected 'vault-name/secret-name'", value)
	}
	vaultName, secretName := parts[0], parts[1]

	client, err := r.getClient(vaultName)
	if err != nil {
		return "", err
	}

	response, err := client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		if respErr, ok := errors.AsType[*azcore.ResponseError](err); ok && respErr.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("secret %q not found in Azure Key Vault %q", secretName, vaultName)
		}
		return "", fmt.Errorf("failed to get secret %q from Azure Key Vault: %w", secretRef, err)
	}
	if response.Value == nil {
		return "", fmt.Errorf("secret %q in vault %q has a nil value", secretName, vaultName)
	}
	secretValue := *response.Value

	if jsonKey != "" {
		var data map[string]any
		if err := json.UnmarshalString(secretValue, &data); err != nil {
			return "", fmt.Errorf("failed to unmarshal JSON secret %q from Azure: %w", secretRef, err)
		}
		value, exists := data[jsonKey]
		if !exists {
			return "", fmt.Errorf("key %q not found in JSON secret %q from Azure", jsonKey, secretRef)
		}
		secretValue = fmt.Sprintf("%v", value)
	}

	return secretValue, nil
}

// getClient lazily creates and caches an azsecrets.Client for a given vault
// name.
//
// Takes vaultName (string) which specifies the Azure Key Vault name.
//
// Returns *azsecrets.Client which is the cached or newly created client.
// Returns error when the Azure SDK fails to create the client.
//
// Safe for concurrent use. Uses a mutex to protect the client cache.
func (r *Resolver) getClient(vaultName string) (*azsecrets.Client, error) {
	r.clientMutex.Lock()
	defer r.clientMutex.Unlock()

	vaultURL := fmt.Sprintf("https://%s.vault.azure.net/", vaultName)
	if client, ok := r.clientCache[vaultURL]; ok {
		return client, nil
	}

	client, err := azsecrets.NewClient(vaultURL, r.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Key Vault client for %s: %w", vaultURL, err)
	}

	r.clientCache[vaultURL] = client
	return client, nil
}

// Register creates a new Azure Key Vault resolver and registers it in the
// global resolver registry. This is a convenience function equivalent to
// [NewResolver] followed by [config.RegisterResolver].
//
// Returns error when resolver creation or registration fails.
//
// Example:
//
//	func init() {
//	    if err := config_resolver_azure.Register(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func Register() error {
	resolver, err := NewResolver()
	if err != nil {
		return err
	}
	return config.RegisterResolver(resolver)
}
