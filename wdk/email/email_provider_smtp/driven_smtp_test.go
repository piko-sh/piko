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

package email_provider_smtp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestNewSMTPProvider(t *testing.T) {
	t.Parallel()

	t.Run("returns error when host is empty", func(t *testing.T) {
		t.Parallel()

		provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
			Host: "",
			Port: 587,
		})

		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "invalid SMTP host or port")
	})

	t.Run("returns error when port is zero", func(t *testing.T) {
		t.Parallel()

		provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
			Host: "smtp.example.com",
			Port: 0,
		})

		require.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("returns error when port is negative", func(t *testing.T) {
		t.Parallel()

		provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
			Host: "smtp.example.com",
			Port: -1,
		})

		require.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("returns error when port exceeds maximum", func(t *testing.T) {
		t.Parallel()

		provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
			Host: "smtp.example.com",
			Port: 65536,
		})

		require.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("succeeds with valid host and port", func(t *testing.T) {
		t.Parallel()

		provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
			Host:      "smtp.example.com",
			Port:      587,
			Username:  "user@example.com",
			Password:  "secret",
			FromEmail: "noreply@example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, "smtp.example.com", provider.host)
		assert.Equal(t, 587, provider.port)
		assert.Equal(t, "user@example.com", provider.username)
		assert.Equal(t, "secret", provider.password)
		assert.Equal(t, "noreply@example.com", provider.fromEmail)
	})

	t.Run("succeeds with maximum valid port", func(t *testing.T) {
		t.Parallel()

		provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
			Host: "smtp.example.com",
			Port: 65535,
		})

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("succeeds with port one", func(t *testing.T) {
		t.Parallel()

		provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
			Host: "smtp.example.com",
			Port: 1,
		})

		require.NoError(t, err)
		require.NotNil(t, provider)
	})
}

func TestSMTPProvider_GetProviderType(t *testing.T) {
	t.Parallel()

	provider := &SMTPProvider{}
	assert.Equal(t, "smtp", provider.GetProviderType())
}

func TestSMTPProvider_GetProviderMetadata(t *testing.T) {
	t.Parallel()

	provider := &SMTPProvider{
		host:      "smtp.example.com",
		port:      587,
		fromEmail: "sender@example.com",
	}

	metadata := provider.GetProviderMetadata()

	assert.Equal(t, "smtp.example.com", metadata["host"])
	assert.Equal(t, 587, metadata["port"])
	assert.Equal(t, "sender@example.com", metadata["from_email"])
}

func TestSMTPProvider_SupportsBulkSending(t *testing.T) {
	t.Parallel()

	provider := &SMTPProvider{}
	assert.False(t, provider.SupportsBulkSending())
}

func TestSMTPProvider_Name(t *testing.T) {
	t.Parallel()

	provider := &SMTPProvider{}
	assert.Equal(t, "EmailProvider (SMTP)", provider.Name())
}

func TestSMTPProvider_Check(t *testing.T) {
	t.Parallel()

	t.Run("returns unhealthy when host is empty", func(t *testing.T) {
		t.Parallel()

		provider := &SMTPProvider{
			host: "",
			port: 587,
		}

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Equal(t, "SMTP host or port not configured", status.Message)
	})

	t.Run("returns unhealthy when port is zero", func(t *testing.T) {
		t.Parallel()

		provider := &SMTPProvider{
			host: "smtp.example.com",
			port: 0,
		}

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	})

	t.Run("liveness check returns healthy with valid config", func(t *testing.T) {
		t.Parallel()

		provider := &SMTPProvider{
			host: "smtp.example.com",
			port: 587,
		}

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Contains(t, status.Message, "smtp.example.com:587")
	})

	t.Run("readiness check returns degraded without client", func(t *testing.T) {
		t.Parallel()

		provider := &SMTPProvider{
			host: "smtp.example.com",
			port: 587,
		}

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateDegraded, status.State)
		assert.Contains(t, status.Message, "not connected")
	})
}

func TestSMTPProvider_ValidateMessageParams(t *testing.T) {
	t.Parallel()

	provider := &SMTPProvider{}

	t.Run("returns error when no recipients", func(t *testing.T) {
		t.Parallel()

		err := provider.validateMessageParams(&email_dto.SendParams{
			To:        []string{},
			BodyPlain: "hello",
		})

		require.ErrorIs(t, err, email_domain.ErrRecipientRequired)
	})

	t.Run("returns error when both bodies empty", func(t *testing.T) {
		t.Parallel()

		err := provider.validateMessageParams(&email_dto.SendParams{
			To:        []string{"test@example.com"},
			BodyHTML:  "",
			BodyPlain: "",
		})

		require.ErrorIs(t, err, email_domain.ErrBodyRequired)
	})

	t.Run("succeeds with plain text body", func(t *testing.T) {
		t.Parallel()

		err := provider.validateMessageParams(&email_dto.SendParams{
			To:        []string{"test@example.com"},
			BodyPlain: "hello",
		})

		require.NoError(t, err)
	})

	t.Run("succeeds with HTML body", func(t *testing.T) {
		t.Parallel()

		err := provider.validateMessageParams(&email_dto.SendParams{
			To:       []string{"test@example.com"},
			BodyHTML: "<p>hello</p>",
		})

		require.NoError(t, err)
	})

	t.Run("succeeds with both bodies", func(t *testing.T) {
		t.Parallel()

		err := provider.validateMessageParams(&email_dto.SendParams{
			To:        []string{"test@example.com"},
			BodyHTML:  "<p>hello</p>",
			BodyPlain: "hello",
		})

		require.NoError(t, err)
	})
}

func TestSMTPProvider_Close_NilClient(t *testing.T) {
	t.Parallel()

	provider := &SMTPProvider{
		host: "smtp.example.com",
		port: 587,
	}

	err := provider.Close(context.Background())
	assert.NoError(t, err)
}

func TestSMTPProvider_SendBulk_EmptySlice(t *testing.T) {
	t.Parallel()

	provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
		Host: "smtp.example.com",
		Port: 587,
	})
	require.NoError(t, err)

	err = provider.SendBulk(context.Background(), []*email_dto.SendParams{})
	assert.NoError(t, err)
}

func TestSMTPProvider_SendBulk_NilSlice(t *testing.T) {
	t.Parallel()

	provider, err := NewSMTPProvider(context.Background(), SMTPProviderArgs{
		Host: "smtp.example.com",
		Port: 587,
	})
	require.NoError(t, err)

	err = provider.SendBulk(context.Background(), nil)
	assert.NoError(t, err)
}
