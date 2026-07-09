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
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSVGImageResolver_SVGDataURI(t *testing.T) {
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="200" height="150" viewBox="0 0 200 150"></svg>`
	encoded := base64.StdEncoding.EncodeToString([]byte(svgContent))
	source := "data:image/svg+xml;base64," + encoded

	dataAdapter := NewDataURISVGDataAdapter()
	resolver := NewSVGImageResolver(nil, dataAdapter)

	w, h, err := resolver.GetImageDimensions(context.Background(), source)
	require.NoError(t, err)
	assert.Equal(t, 200.0, w)
	assert.Equal(t, 150.0, h)
}

func TestSVGImageResolver_ViewBoxOnly(t *testing.T) {
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 200"></svg>`
	encoded := base64.StdEncoding.EncodeToString([]byte(svgContent))
	source := "data:image/svg+xml;base64," + encoded

	dataAdapter := NewDataURISVGDataAdapter()
	resolver := NewSVGImageResolver(nil, dataAdapter)

	w, h, err := resolver.GetImageDimensions(context.Background(), source)
	require.NoError(t, err)
	assert.Equal(t, 300.0, w)
	assert.Equal(t, 200.0, h)
}

func TestSVGImageResolver_NonSVG_FallsThrough(t *testing.T) {
	dataAdapter := NewDataURISVGDataAdapter()

	innerCalled := false
	inner := &mockImageResolver{
		fn: func(ctx context.Context, source string) (float64, float64, error) {
			innerCalled = true
			return 50, 75, nil
		},
	}

	resolver := NewSVGImageResolver(inner, dataAdapter)

	w, h, err := resolver.GetImageDimensions(context.Background(), "/images/photo.jpg")
	require.NoError(t, err)
	assert.True(t, innerCalled, "expected inner resolver to be called for non-SVG source")
	assert.Equal(t, 50.0, w)
	assert.Equal(t, 75.0, h)
}

func TestSVGImageResolver_NilInner_ReturnsDefault(t *testing.T) {
	dataAdapter := NewDataURISVGDataAdapter()
	resolver := NewSVGImageResolver(nil, dataAdapter)

	w, h, err := resolver.GetImageDimensions(context.Background(), "/images/photo.jpg")
	require.NoError(t, err)
	assert.Equal(t, 100.0, w)
	assert.Equal(t, 100.0, h)
}

type mockImageResolver struct {
	fn func(ctx context.Context, source string) (float64, float64, error)
}

func (m *mockImageResolver) GetImageDimensions(ctx context.Context, source string) (float64, float64, error) {
	return m.fn(ctx, source)
}
