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

package templater_adapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/templater/templater_dto"
)

type mockEmailTemplateService struct {
	renderFunc func(ctx context.Context, request *http.Request, templatePath string, props any, premailerOptions *premailer.Options, isPreviewMode bool) (*templater_dto.RenderedEmailContent, error)
}

func (m *mockEmailTemplateService) Render(ctx context.Context, request *http.Request, templatePath string, props any, premailerOptions *premailer.Options, isPreviewMode bool) (*templater_dto.RenderedEmailContent, error) {
	if m.renderFunc != nil {
		return m.renderFunc(ctx, request, templatePath, props, premailerOptions, isPreviewMode)
	}
	return nil, nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	adapter := New(&mockEmailTemplateService{})

	require.NotNil(t, adapter)
}

func TestAdapter_Render(t *testing.T) {
	t.Parallel()

	t.Run("delegates to implementation and maps fields", func(t *testing.T) {
		t.Parallel()

		mock := &mockEmailTemplateService{
			renderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options, _ bool) (*templater_dto.RenderedEmailContent, error) {
				return &templater_dto.RenderedEmailContent{
					HTML:      "<h1>Hello</h1>",
					PlainText: "Hello",
					CSS:       "h1 { color: red; }",
				}, nil
			},
		}

		adapter := New(mock)
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		rendered, err := adapter.Render(context.Background(), request, "welcome.pk", nil, nil)

		require.NoError(t, err)
		assert.Equal(t, "<h1>Hello</h1>", rendered.HTML)
		assert.Equal(t, "Hello", rendered.PlainText)
		assert.Nil(t, rendered.AttachmentRequests)
	})

	t.Run("propagates error from implementation", func(t *testing.T) {
		t.Parallel()

		renderError := errors.New("template not found")
		mock := &mockEmailTemplateService{
			renderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options, _ bool) (*templater_dto.RenderedEmailContent, error) {
				return nil, renderError
			},
		}

		adapter := New(mock)
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		rendered, err := adapter.Render(context.Background(), request, "missing.pk", nil, nil)

		require.Error(t, err)
		assert.Nil(t, rendered)
		assert.ErrorIs(t, err, renderError)
		assert.Contains(t, err.Error(), "rendering email template")
		assert.Contains(t, err.Error(), "missing.pk")
	})

	t.Run("maps attachment requests", func(t *testing.T) {
		t.Parallel()

		attachments := []*email_dto.EmailAssetRequest{
			{SourcePath: "assets/logo.png", CID: "logo123"},
			{SourcePath: "assets/banner.jpg", CID: "banner456"},
		}

		mock := &mockEmailTemplateService{
			renderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options, _ bool) (*templater_dto.RenderedEmailContent, error) {
				return &templater_dto.RenderedEmailContent{
					HTML:               "<img src='cid:logo123'>",
					PlainText:          "Logo image",
					AttachmentRequests: attachments,
				}, nil
			},
		}

		adapter := New(mock)
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		rendered, err := adapter.Render(context.Background(), request, "newsletter.pk", nil, nil)

		require.NoError(t, err)
		require.Len(t, rendered.AttachmentRequests, 2)
		assert.Equal(t, "assets/logo.png", rendered.AttachmentRequests[0].SourcePath)
		assert.Equal(t, "logo123", rendered.AttachmentRequests[0].CID)
		assert.Equal(t, "assets/banner.jpg", rendered.AttachmentRequests[1].SourcePath)
		assert.Equal(t, "banner456", rendered.AttachmentRequests[1].CID)
	})
}
