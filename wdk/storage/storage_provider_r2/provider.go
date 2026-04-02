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

package storage_provider_r2

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/wdk/storage"
	"piko.sh/piko/wdk/storage/storage_provider_s3"
)

// Config holds the settings needed for the R2Provider.
type Config struct {
	// RepositoryMappings maps repository names to their storage bucket names.
	RepositoryMappings map[string]string

	// AccountID is the Cloudflare account identifier used for API requests.
	AccountID string

	// AccessKey is the AWS access key ID for static credentials.
	AccessKey string

	// SecretKey is the AWS secret access key for static credentials.
	SecretKey string
}

// NewR2Provider creates a new R2 storage adapter by wrapping the S3 provider.
//
// Takes config (*Config) which specifies the R2 connection settings.
// Takes opts (variadic ProviderOption) for additional configuration.
//
// Returns storage.ProviderPort which is a configured R2 storage adapter.
// Returns error when configuration is invalid or the provider fails to initialise.
func NewR2Provider(ctx context.Context, config *Config, opts ...storage.ProviderOption) (storage.ProviderPort, error) {
	if config == nil {
		return nil, errors.New("R2 config is required")
	}
	if config.AccountID == "" {
		return nil, errors.New("accountID is required for R2 provider")
	}

	s3Config := &storage_provider_s3.Config{
		RepositoryMappings: config.RepositoryMappings,
		AccessKey:          config.AccessKey,
		SecretKey:          config.SecretKey,
		EndpointURL:        fmt.Sprintf("https://%s.r2.cloudflarestorage.com", config.AccountID),
		Region:             "auto",
		DisableChecksum:    true,
		UsePathStyle:       false,
	}

	return storage_provider_s3.NewS3Provider(ctx, s3Config, opts...)
}
