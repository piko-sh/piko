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

package pdfwriter_domain

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

type panickingTemplateRunner struct{}

func (panickingTemplateRunner) RunPdfWithProps(
	context.Context, string, *http.Request, any,
) (*ast_domain.TemplateAST, string, error) {
	panic("template runner exploded")
}

type stubLayouter struct {
	err    error
	result *layouter_dto.LayoutResult
	config layouter_dto.LayoutConfig
}

func (s *stubLayouter) Layout(
	_ context.Context, _ *ast_domain.TemplateAST, _ string, config layouter_dto.LayoutConfig,
) (*layouter_dto.LayoutResult, error) {
	s.config = config
	return s.result, s.err
}

func TestNewPdfWriterService_AppliesLimitOptions(t *testing.T) {
	t.Parallel()

	limits := layouter_dto.LayoutLimits{MaxPages: 7}
	service, ok := NewPdfWriterService(nil, nil, nil, nil, nil,
		WithLayoutLimits(limits),
		WithMaxImagePixels(1234),
	).(*pdfWriterService)
	require.True(t, ok)

	assert.Equal(t, limits, service.layoutLimits)
	assert.Equal(t, 1234, service.maxImagePixels)
}

func TestPdfWriterService_Render_RecoversPanics(t *testing.T) {
	t.Parallel()

	service := NewPdfWriterService(panickingTemplateRunner{}, nil, nil, nil, nil)

	var result *pdfwriter_dto.PdfResult
	var err error
	assert.NotPanics(t, func() {
		result, err = service.Render(context.Background(), nil, "doc.pk", nil, pdfwriter_dto.PdfConfig{})
	})
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic during PDF render")
}

func TestPdfWriterService_Render_PassesLimitsAndPropagatesLimitErrors(t *testing.T) {
	t.Parallel()

	layouter := &stubLayouter{err: errors.Join(errors.New("layout failed"), layouter_dto.ErrTooManyPages)}
	service := NewPdfWriterService(
		&stubTemplateRunner{ast: &ast_domain.TemplateAST{}},
		layouter, nil, nil, nil,
		WithLayoutLimits(layouter_dto.LayoutLimits{MaxPages: 3}),
	)

	_, err := service.Render(context.Background(), nil, "doc.pk", nil, pdfwriter_dto.PdfConfig{})

	assert.ErrorIs(t, err, layouter_dto.ErrTooManyPages)
	assert.Equal(t, 3, layouter.config.Limits.MaxPages)
}

func TestPdfWriterService_Render_NilAST(t *testing.T) {
	t.Parallel()

	service := NewPdfWriterService(&stubTemplateRunner{}, &stubLayouter{}, nil, nil, nil)

	_, err := service.Render(context.Background(), nil, "doc.pk", nil, pdfwriter_dto.PdfConfig{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil AST")
}

func newStubLayoutResult() *layouter_dto.LayoutResult {
	root := newLayoutBox().WithContentRect(0, 0, 100, 100).WithBoxType(layouter_domain.BoxBlock).Build()
	return &layouter_dto.LayoutResult{RootBox: root, Pages: []layouter_dto.PageOutput{{Index: 0, Width: 595, Height: 842}}}
}

func TestPdfWriterService_Render_PaintsTheLayout(t *testing.T) {
	t.Parallel()

	service := NewPdfWriterService(
		&stubTemplateRunner{ast: &ast_domain.TemplateAST{}},
		&stubLayouter{result: newStubLayoutResult()}, nil, nil, nil,
		WithSVGRenderer(nil, nil),
		WithMaxImagePixels(500),
	)

	result, err := service.Render(context.Background(), nil, "doc.pk", nil, pdfwriter_dto.PdfConfig{Page: layouter_dto.PageA4})

	require.NoError(t, err)
	assert.Equal(t, 1, result.PageCount)
	assert.True(t, bytes.HasPrefix(result.Content, []byte("%PDF-")))
}

func TestBuilderDo_RendersWithServiceLimits(t *testing.T) {
	t.Parallel()

	layouter := &stubLayouter{result: newStubLayoutResult()}
	service := NewPdfWriterService(
		&stubTemplateRunner{ast: &ast_domain.TemplateAST{}},
		layouter, nil, nil, nil,
		WithLayoutLimits(layouter_dto.LayoutLimits{MaxPages: 4}),
	)

	result, err := service.NewRender().Template("doc.pk").WithMaxImagePixels(50).Do(context.Background())

	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result.Content, []byte("%PDF-")))
	assert.Equal(t, 4, layouter.config.Limits.MaxPages)
}
