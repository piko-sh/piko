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

package email_provider_gmail

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestNewGmailProvider(t *testing.T) {
	t.Parallel()

	t.Run("returns error when username is empty", func(t *testing.T) {
		t.Parallel()

		provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
			Username: "",
			Password: "app-password",
		})

		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "username must not be empty")
	})

	t.Run("returns error when password is empty", func(t *testing.T) {
		t.Parallel()

		provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
			Username: "user@gmail.com",
			Password: "",
		})

		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "password must not be empty")
	})

	t.Run("succeeds with valid credentials", func(t *testing.T) {
		t.Parallel()

		provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
			Username:  "user@gmail.com",
			Password:  "app-password",
			FromEmail: "noreply@example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("defaults from email to username when empty", func(t *testing.T) {
		t.Parallel()

		provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
			Username: "user@gmail.com",
			Password: "app-password",
		})

		require.NoError(t, err)
		assert.Equal(t, "user@gmail.com", provider.fromEmail)
	})

	t.Run("uses explicit from email when provided", func(t *testing.T) {
		t.Parallel()

		provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
			Username:  "user@gmail.com",
			Password:  "app-password",
			FromEmail: "noreply@example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "noreply@example.com", provider.fromEmail)
	})
}

func TestGmailProvider_GetProviderType(t *testing.T) {
	t.Parallel()

	provider := &GmailProvider{}
	assert.Equal(t, "gmail", provider.GetProviderType())
}

func TestGmailProvider_GetProviderMetadata(t *testing.T) {
	t.Parallel()

	provider := &GmailProvider{
		fromEmail: "sender@example.com",
	}

	metadata := provider.GetProviderMetadata()

	assert.Equal(t, gmailHost, metadata["host"])
	assert.Equal(t, gmailPort, metadata["port"])
	assert.Equal(t, "sender@example.com", metadata["from_email"])
}

func TestGmailProvider_SupportsBulkSending(t *testing.T) {
	t.Parallel()

	provider := &GmailProvider{}
	assert.False(t, provider.SupportsBulkSending())
}

func TestGmailProvider_Name(t *testing.T) {
	t.Parallel()

	provider := &GmailProvider{}
	assert.Equal(t, "EmailProvider (Gmail)", provider.Name())
}

func TestGmailProvider_Close(t *testing.T) {
	t.Parallel()

	provider := &GmailProvider{}
	err := provider.Close(context.Background())
	assert.NoError(t, err)
}

func TestGmailProvider_Check(t *testing.T) {
	t.Parallel()

	t.Run("liveness check returns healthy", func(t *testing.T) {
		t.Parallel()

		provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
			Username: "user@gmail.com",
			Password: "app-password",
		})
		require.NoError(t, err)

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Equal(t, "EmailProvider (Gmail)", status.Name)
		assert.Equal(t, "Gmail provider operational", status.Message)
	})

	t.Run("readiness check returns healthy with valid credentials", func(t *testing.T) {
		t.Parallel()

		provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
			Username: "user@gmail.com",
			Password: "app-password",
		})
		require.NoError(t, err)

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Equal(t, "Gmail provider configured and ready", status.Message)
	})

	t.Run("readiness check returns unhealthy with empty username", func(t *testing.T) {
		t.Parallel()

		provider := &GmailProvider{
			username: "",
			password: "app-password",
		}

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Equal(t, "Gmail credentials not configured", status.Message)
	})

	t.Run("readiness check returns unhealthy with empty password", func(t *testing.T) {
		t.Parallel()

		provider := &GmailProvider{
			username: "user@gmail.com",
			password: "",
		}

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Equal(t, "Gmail credentials not configured", status.Message)
	})
}

func TestValidateSendParams(t *testing.T) {
	t.Parallel()

	t.Run("returns error when no recipients", func(t *testing.T) {
		t.Parallel()

		err := validateSendParams(&email_dto.SendParams{
			To:        []string{},
			BodyPlain: "hello",
		})

		require.ErrorIs(t, err, email_domain.ErrRecipientRequired)
	})

	t.Run("returns error when both bodies empty", func(t *testing.T) {
		t.Parallel()

		err := validateSendParams(&email_dto.SendParams{
			To:        []string{"test@example.com"},
			BodyHTML:  "",
			BodyPlain: "",
		})

		require.ErrorIs(t, err, email_domain.ErrBodyRequired)
	})

	t.Run("succeeds with plain text body", func(t *testing.T) {
		t.Parallel()

		err := validateSendParams(&email_dto.SendParams{
			To:        []string{"test@example.com"},
			BodyPlain: "hello",
		})

		require.NoError(t, err)
	})

	t.Run("succeeds with HTML body", func(t *testing.T) {
		t.Parallel()

		err := validateSendParams(&email_dto.SendParams{
			To:       []string{"test@example.com"},
			BodyHTML: "<p>hello</p>",
		})

		require.NoError(t, err)
	})

	t.Run("succeeds with both bodies", func(t *testing.T) {
		t.Parallel()

		err := validateSendParams(&email_dto.SendParams{
			To:        []string{"test@example.com"},
			BodyHTML:  "<p>hello</p>",
			BodyPlain: "hello",
		})

		require.NoError(t, err)
	})
}

func TestGmailProvider_SendBulk_EmptySlice(t *testing.T) {
	t.Parallel()

	provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
		Username: "user@gmail.com",
		Password: "app-password",
	})
	require.NoError(t, err)

	err = provider.SendBulk(context.Background(), []*email_dto.SendParams{})
	assert.NoError(t, err)
}

func TestGmailProvider_SendBulk_NilSlice(t *testing.T) {
	t.Parallel()

	provider, err := NewGmailProvider(context.Background(), GmailProviderArgs{
		Username: "user@gmail.com",
		Password: "app-password",
	})
	require.NoError(t, err)

	err = provider.SendBulk(context.Background(), nil)
	assert.NoError(t, err)
}
