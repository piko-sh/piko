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

package provider_stdout

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func newTestProvider(t *testing.T) *stdoutProvider {
	t.Helper()

	provider, err := New(context.Background())
	require.NoError(t, err)

	typed, ok := provider.(*stdoutProvider)
	require.True(t, ok)

	return typed
}

func TestNew(t *testing.T) {
	t.Parallel()

	provider, err := New(context.Background())

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestFormatEmailMetadata(t *testing.T) {
	t.Parallel()

	t.Run("all fields populated", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			From:    new("sender@example.com"),
			To:      []string{"alice@example.com", "bob@example.com"},
			Cc:      []string{"cc@example.com"},
			Bcc:     []string{"bcc@example.com"},
			Subject: "Test Subject",
		}

		provider.formatEmailMetadata(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "From:    sender@example.com")
		assert.Contains(t, output, "To:      alice@example.com, bob@example.com")
		assert.Contains(t, output, "Cc:      cc@example.com")
		assert.Contains(t, output, "Bcc:     bcc@example.com")
		assert.Contains(t, output, "Subject: Test Subject")
	})

	t.Run("nil from shows not specified", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			From:    nil,
			To:      []string{"recipient@example.com"},
			Subject: "Test",
		}

		provider.formatEmailMetadata(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "[not specified]")
	})

	t.Run("no cc or bcc omits those lines", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			From:    new("sender@example.com"),
			To:      []string{"recipient@example.com"},
			Subject: "Test",
		}

		provider.formatEmailMetadata(&builder, params)
		output := builder.String()

		assert.NotContains(t, output, "Cc:")
		assert.NotContains(t, output, "Bcc:")
	})
}

func TestFormatEmailBody(t *testing.T) {
	t.Parallel()

	t.Run("plain text only", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			BodyPlain: "Hello, world!",
		}

		provider.formatEmailBody(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "BODY (PLAIN TEXT)")
		assert.Contains(t, output, "Hello, world!")
		assert.NotContains(t, output, "BODY (HTML)")
	})

	t.Run("html only", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			BodyHTML: "<h1>Hello</h1>",
		}

		provider.formatEmailBody(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "BODY (HTML)")
		assert.Contains(t, output, "<h1>Hello</h1>")
		assert.NotContains(t, output, "BODY (PLAIN TEXT)")
	})

	t.Run("both plain and html", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			BodyPlain: "Hello",
			BodyHTML:  "<p>Hello</p>",
		}

		provider.formatEmailBody(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "BODY (PLAIN TEXT)")
		assert.Contains(t, output, "BODY (HTML)")
	})

	t.Run("neither plain nor html", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{}

		provider.formatEmailBody(&builder, params)
		output := builder.String()

		assert.Empty(t, output)
	})
}

func TestFormatAttachments(t *testing.T) {
	t.Parallel()

	t.Run("zero attachments produces empty output", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{}

		provider.formatAttachments(&builder, params)
		output := builder.String()

		assert.Empty(t, output)
	})

	t.Run("attachment with content id", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			Attachments: []email_dto.Attachment{
				{
					Filename:  "logo.png",
					MIMEType:  "image/png",
					ContentID: "logo123",
					Content:   []byte("fake-image-data"),
				},
			},
		}

		provider.formatAttachments(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "cid=logo123")
		assert.Contains(t, output, "logo.png")
	})

	t.Run("empty mime type defaults to octet stream", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			Attachments: []email_dto.Attachment{
				{
					Filename: "data.bin",
					MIMEType: "",
					Content:  []byte("binary-data"),
				},
			},
		}

		provider.formatAttachments(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "application/octet-stream")
	})

	t.Run("multiple attachments shows correct count", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		var builder strings.Builder

		params := &email_dto.SendParams{
			Attachments: []email_dto.Attachment{
				{Filename: "file1.txt", MIMEType: "text/plain", Content: []byte("one")},
				{Filename: "file2.txt", MIMEType: "text/plain", Content: []byte("two")},
				{Filename: "file3.txt", MIMEType: "text/plain", Content: []byte("three")},
			},
		}

		provider.formatAttachments(&builder, params)
		output := builder.String()

		assert.Contains(t, output, "ATTACHMENTS (3)")
		assert.Contains(t, output, "file1.txt")
		assert.Contains(t, output, "file2.txt")
		assert.Contains(t, output, "file3.txt")
	})
}

func TestProviderMetadata(t *testing.T) {
	t.Parallel()

	t.Run("get provider type returns stdout", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)

		assert.Equal(t, "stdout", provider.GetProviderType())
	})

	t.Run("supports bulk sending returns true", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)

		assert.True(t, provider.SupportsBulkSending())
	})

	t.Run("get provider metadata contains expected keys", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)
		metadata := provider.GetProviderMetadata()

		assert.Contains(t, metadata, "version")
		assert.Contains(t, metadata, "environment")
		assert.Contains(t, metadata, "description")
	})

	t.Run("name returns expected value", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)

		assert.Equal(t, "EmailProvider (Stdout)", provider.Name())
	})

	t.Run("close returns nil", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)

		err := provider.Close(context.Background())

		assert.NoError(t, err)
	})

	t.Run("check returns healthy for liveness", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	})

	t.Run("check returns healthy for readiness", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)

		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	})
}

func TestSendBulk(t *testing.T) {
	t.Parallel()

	t.Run("empty list returns nil", func(t *testing.T) {
		t.Parallel()

		provider := newTestProvider(t)

		err := provider.SendBulk(context.Background(), []*email_dto.SendParams{})

		assert.NoError(t, err)
	})
}
