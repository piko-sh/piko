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

package email_provider_disk

import (
	"context"

	"piko.sh/piko/internal/email/email_adapters/provider_disk"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/wdk/email"
)

// DiskProviderArgs is an alias for the disk provider configuration arguments.
type DiskProviderArgs = provider_disk.DiskProviderArgs

// NewDiskProvider creates a disk-based email provider for local storage.
//
// Takes arguments (DiskProviderArgs) which specifies the disk storage settings.
// Takes opts (...email_domain.ProviderOption) which configures provider behaviour.
//
// Returns email.ProviderPort which is the configured provider ready for use.
// Returns error when the provider cannot be initialised.
func NewDiskProvider(ctx context.Context, arguments DiskProviderArgs, opts ...email_domain.ProviderOption) (email.ProviderPort, error) {
	return provider_disk.NewDiskProvider(ctx, arguments, opts...)
}
