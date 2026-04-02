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

package crypto_dto

// ProviderType identifies which encryption provider to use and implements
// fmt.Stringer.
type ProviderType string

const (
	// ProviderTypeLocalAESGCM uses local AES-256-GCM encryption. Best for
	// development, testing, and single-server deployments.
	ProviderTypeLocalAESGCM ProviderType = "local_aes_gcm"

	// ProviderTypeAWSKMS uses AWS Key Management Service for encryption.
	// Best for production AWS deployments and multi-region setups.
	ProviderTypeAWSKMS ProviderType = "aws_kms"

	// ProviderTypeGCPKMS uses Google Cloud Key Management Service for encryption.
	ProviderTypeGCPKMS ProviderType = "gcp_kms"

	// ProviderTypeAzureKeyVault uses Azure Key Vault for key management.
	// Best for production Azure deployments.
	ProviderTypeAzureKeyVault ProviderType = "azure_keyvault"

	// ProviderTypeHashiCorpVault uses HashiCorp Vault for secrets management.
	// Suited for on-premise or multi-cloud deployments.
	ProviderTypeHashiCorpVault ProviderType = "hashicorp_vault"
)

// String returns the string representation of the provider type.
//
// Returns string which is the provider type as a plain string value.
func (p ProviderType) String() string {
	return string(p)
}

// IsValid reports whether the provider type is a known value.
//
// Returns bool which is true if the provider type matches a known provider.
func (p ProviderType) IsValid() bool {
	switch p {
	case ProviderTypeLocalAESGCM,
		ProviderTypeAWSKMS,
		ProviderTypeGCPKMS,
		ProviderTypeAzureKeyVault,
		ProviderTypeHashiCorpVault:
		return true
	default:
		return false
	}
}
