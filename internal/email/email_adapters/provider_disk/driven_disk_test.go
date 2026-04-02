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

package provider_disk

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestNewDiskProvider(t *testing.T) {
	t.Parallel()

	t.Run("empty outbox path returns error", func(t *testing.T) {
		t.Parallel()

		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: ""})

		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "outbox path cannot be empty")
	})

	t.Run("valid path creates provider", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})

		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.NoError(t, provider.Close(context.Background()))
	})
}

func TestGetProviderType(t *testing.T) {
	t.Parallel()

	t.Run("returns disk", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		assert.Equal(t, "disk", provider.GetProviderType())
	})
}

func TestSupportsBulkSending(t *testing.T) {
	t.Parallel()

	t.Run("returns false", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		assert.False(t, provider.SupportsBulkSending())
	})
}

func TestGetProviderMetadata(t *testing.T) {
	t.Parallel()

	t.Run("contains outbox_path key", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		metadata := provider.GetProviderMetadata()

		assert.Contains(t, metadata, "outbox_path")
		assert.Equal(t, outboxDirectory, metadata["outbox_path"])
	})
}

func TestName(t *testing.T) {
	t.Parallel()

	t.Run("returns EmailProvider (Disk)", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		assert.Equal(t, "EmailProvider (Disk)", provider.Name())
	})
}

func TestGenerateEmailFilename(t *testing.T) {
	t.Parallel()

	provider := &DiskProvider{}

	t.Run("sanitises @ to _at_", func(t *testing.T) {
		t.Parallel()

		filename := provider.generateEmailFilename("user@example.com")

		assert.Contains(t, filename, "_at_")
		assert.NotContains(t, filename, "@")
	})

	t.Run("sanitises dot to underscore", func(t *testing.T) {
		t.Parallel()

		filename := provider.generateEmailFilename("user@example.com")
		recipientPortion := filename[strings.LastIndex(filename, "_at_"):]
		recipientWithoutExtension := strings.TrimSuffix(recipientPortion, ".eml")

		assert.NotContains(t, recipientWithoutExtension, ".")
	})

	t.Run("sanitises slash to underscore", func(t *testing.T) {
		t.Parallel()

		filename := provider.generateEmailFilename("user/name@example.com")

		assert.NotContains(t, filename, "/")
	})

	t.Run("result ends with .eml", func(t *testing.T) {
		t.Parallel()

		filename := provider.generateEmailFilename("user@example.com")

		assert.True(t, strings.HasSuffix(filename, ".eml"))
	})
}

func TestValidateSendParams(t *testing.T) {
	t.Parallel()

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: 0,
		Burst:          0,
		Clock:          nil,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig)

	t.Run("no recipients returns ErrRecipientRequired", func(t *testing.T) {
		t.Parallel()

		provider := &DiskProvider{rateLimiter: rateLimiter}
		params := &email_dto.SendParams{
			To:       []string{},
			BodyHTML: "<p>Hello</p>",
		}

		err := provider.validateSendParams(context.Background(), params)

		require.ErrorIs(t, err, email_domain.ErrRecipientRequired)
	})

	t.Run("empty body returns ErrBodyRequired", func(t *testing.T) {
		t.Parallel()

		provider := &DiskProvider{rateLimiter: rateLimiter}
		params := &email_dto.SendParams{
			To:        []string{"user@example.com"},
			BodyHTML:  "",
			BodyPlain: "",
		}

		err := provider.validateSendParams(context.Background(), params)

		require.ErrorIs(t, err, email_domain.ErrBodyRequired)
	})

	t.Run("valid params returns nil", func(t *testing.T) {
		t.Parallel()

		provider := &DiskProvider{rateLimiter: rateLimiter}
		params := &email_dto.SendParams{
			To:       []string{"user@example.com"},
			BodyHTML: "<p>Hello</p>",
		}

		err := provider.validateSendParams(context.Background(), params)

		require.NoError(t, err)
	})
}

func TestSend(t *testing.T) {
	t.Parallel()

	t.Run("happy path writes eml file to outbox", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		params := &email_dto.SendParams{
			To:       []string{"recipient@example.com"},
			Subject:  "Test Subject",
			BodyHTML: "<p>Hello, world!</p>",
		}

		err = provider.Send(context.Background(), params)

		require.NoError(t, err)

		entries, readErr := os.ReadDir(outboxDirectory)
		require.NoError(t, readErr)
		require.Len(t, entries, 1)
		assert.True(t, strings.HasSuffix(entries[0].Name(), ".eml"))
	})

	t.Run("no recipients returns error", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		params := &email_dto.SendParams{
			To:       []string{},
			Subject:  "Test Subject",
			BodyHTML: "<p>Hello</p>",
		}

		err = provider.Send(context.Background(), params)

		require.Error(t, err)
		assert.ErrorIs(t, err, email_domain.ErrRecipientRequired)
	})
}

func TestSendBulk(t *testing.T) {
	t.Parallel()

	t.Run("empty list returns nil", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		err = provider.SendBulk(context.Background(), []*email_dto.SendParams{})

		require.NoError(t, err)
	})
}

func TestCheck(t *testing.T) {
	t.Parallel()

	t.Run("liveness with configured path returns healthy", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	})

	t.Run("liveness with empty path returns unhealthy", func(t *testing.T) {
		t.Parallel()

		provider := &DiskProvider{outboxPath: ""}

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	})

	t.Run("readiness with valid directory returns healthy", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)
		defer func() { _ = provider.Close(context.Background()) }()

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	})
}

func TestClose(t *testing.T) {
	t.Parallel()

	t.Run("close returns nil", func(t *testing.T) {
		t.Parallel()

		outboxDirectory := t.TempDir()
		provider, err := NewDiskProvider(context.Background(), DiskProviderArgs{OutboxPath: outboxDirectory})
		require.NoError(t, err)

		err = provider.Close(context.Background())

		require.NoError(t, err)
	})
}
