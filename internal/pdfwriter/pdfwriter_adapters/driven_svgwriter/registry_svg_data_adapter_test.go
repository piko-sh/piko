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

package driven_svgwriter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/render/render_domain"
)

type stubSVGAssetResolver struct {
	data map[string]*render_domain.ParsedSvgData
	err  error
}

func (s *stubSVGAssetResolver) GetAssetRawSVG(_ context.Context, assetID string) (*render_domain.ParsedSvgData, error) {
	if s.err != nil {
		return nil, s.err
	}
	d, ok := s.data[assetID]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func TestRegistrySVGDataAdapter_ResolvesAssetToMarkup(t *testing.T) {
	t.Parallel()

	registry := &stubSVGAssetResolver{data: map[string]*render_domain.ParsedSvgData{
		"me-piko2/lib/icons/mail.svg": {
			InnerHTML:  `<path d="M0 0h24v24H0z"/>`,
			Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}, {Name: "fill", Value: "currentColor"}},
		},
	}}
	adapter := NewRegistrySVGDataAdapter(registry, NewDataURISVGDataAdapter())

	markup, ok := adapter.GetSVGData(context.Background(), "me-piko2/lib/icons/mail.svg")
	require.True(t, ok, "expected the registry to resolve the asset")
	for _, want := range []string{`viewBox="0 0 24 24"`, `fill="currentColor"`, `<path d="M0 0h24v24H0z"/>`, "<svg", "</svg>", `xmlns=`} {
		assert.Contains(t, markup, want)
	}
}

func TestRegistrySVGDataAdapter_FallsBackForDataURI(t *testing.T) {
	t.Parallel()

	registry := &stubSVGAssetResolver{err: errors.New("registry should not be called for data URIs")}
	adapter := NewRegistrySVGDataAdapter(registry, NewDataURISVGDataAdapter())

	markup, ok := adapter.GetSVGData(context.Background(), "data:image/svg+xml,<svg><rect/></svg>")
	require.True(t, ok, "expected the data: URI to be handled by the fallback")
	assert.Contains(t, markup, "<rect/>", "expected decoded data-URI markup")
}

func TestRegistrySVGDataAdapter_UnresolvedReturnsFalse(t *testing.T) {
	t.Parallel()

	adapter := NewRegistrySVGDataAdapter(&stubSVGAssetResolver{data: map[string]*render_domain.ParsedSvgData{}}, NewDataURISVGDataAdapter())

	_, ok := adapter.GetSVGData(context.Background(), "me-piko2/lib/icons/missing.svg")
	assert.False(t, ok, "expected ok=false for an asset the registry cannot resolve")
	_, ok = adapter.GetSVGData(context.Background(), "")
	assert.False(t, ok, "expected ok=false for an empty source")
}

func TestReconstructSVGMarkupSkipsInvalidAttributeNames(t *testing.T) {
	t.Parallel()

	parsed := &render_domain.ParsedSvgData{
		InnerHTML: `<rect/>`,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: `onload="alert(1)" x`, Value: "evil"},
			{Name: "viewBox", Value: "0 0 24 24"},
			{Name: "xlink:href", Value: "#a"},
		},
	}

	markup := reconstructSVGMarkup(parsed)

	assert.NotContains(t, markup, "onload")
	assert.Contains(t, markup, `viewBox="0 0 24 24"`)
	assert.Contains(t, markup, `xlink:href="#a"`)
}

func TestIsValidXMLAttributeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		attribute string
		want      bool
	}{
		{name: "simple lower", attribute: "fill", want: true},
		{name: "namespaced", attribute: "xlink:href", want: true},
		{name: "with hyphen", attribute: "stroke-width", want: true},
		{name: "with digit and dot", attribute: "data-x.1", want: true},
		{name: "underscore start", attribute: "_private", want: true},
		{name: "empty", attribute: "", want: false},
		{name: "leading digit", attribute: "1abc", want: false},
		{name: "embedded space", attribute: "a b", want: false},
		{name: "quote injection", attribute: `a="x"`, want: false},
		{name: "angle bracket", attribute: "a>b", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, isValidXMLAttributeName(tt.attribute))
		})
	}
}

func TestReconstructSVGMarkupEnsuresXMLNS(t *testing.T) {
	t.Parallel()

	parsed := &render_domain.ParsedSvgData{
		InnerHTML:  `<rect/>`,
		Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 1 1"}},
	}

	markup := reconstructSVGMarkup(parsed)
	require.Contains(t, markup, `xmlns="http://www.w3.org/2000/svg"`)
}
