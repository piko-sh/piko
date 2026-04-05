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

package driver_markdown

import (
	"context"
	"io"
	"io/fs"
	"net/http"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

var _ render_domain.RenderService = (*mockRenderService)(nil)

type mockRenderService struct {
	RenderASTToPlainTextFunc func(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error)
}

func (m *mockRenderService) RenderASTToPlainText(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error) {
	if m.RenderASTToPlainTextFunc != nil {
		return m.RenderASTToPlainTextFunc(ctx, templateAST)
	}
	return "", nil
}

func (*mockRenderService) BuildThemeCSS(_ context.Context, _ *config.WebsiteConfig) ([]byte, error) {
	panic("BuildThemeCSS not expected in driver_markdown tests")
}

func (*mockRenderService) CollectMetadata(_ context.Context, _ *http.Request, _ *templater_dto.InternalMetadata, _ *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
	return nil, nil, nil
}

func (*mockRenderService) RenderAST(_ context.Context, _ io.Writer, _ http.ResponseWriter, _ *http.Request, _ render_domain.RenderASTOptions) error {
	panic("RenderAST not expected in driver_markdown tests")
}

func (*mockRenderService) RenderEmail(_ context.Context, _ io.Writer, _ *http.Request, _ render_domain.RenderEmailOptions) error {
	panic("RenderEmail not expected in driver_markdown tests")
}

func (*mockRenderService) GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest {
	panic("GetLastEmailAssetRequests not expected in driver_markdown tests")
}

type mockDirEntry struct {
	infoFunc func() (fs.FileInfo, error)
	name     string
	isDir    bool
}

func (m *mockDirEntry) Name() string      { return m.name }
func (m *mockDirEntry) IsDir() bool       { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode { return 0 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) {
	if m.infoFunc != nil {
		return m.infoFunc()
	}
	return nil, nil
}
