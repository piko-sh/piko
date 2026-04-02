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
)

func TestSVGImageResolver_SVGDataURI(t *testing.T) {
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="200" height="150" viewBox="0 0 200 150"></svg>`
	encoded := base64.StdEncoding.EncodeToString([]byte(svgContent))
	source := "data:image/svg+xml;base64," + encoded

	dataAdapter := NewDataURISVGDataAdapter()
	resolver := NewSVGImageResolver(nil, dataAdapter)

	w, h, err := resolver.GetImageDimensions(context.Background(), source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 200 {
		t.Errorf("width = %v, want 200", w)
	}
	if h != 150 {
		t.Errorf("height = %v, want 150", h)
	}
}

func TestSVGImageResolver_ViewBoxOnly(t *testing.T) {
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 200"></svg>`
	encoded := base64.StdEncoding.EncodeToString([]byte(svgContent))
	source := "data:image/svg+xml;base64," + encoded

	dataAdapter := NewDataURISVGDataAdapter()
	resolver := NewSVGImageResolver(nil, dataAdapter)

	w, h, err := resolver.GetImageDimensions(context.Background(), source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 300 {
		t.Errorf("width = %v, want 300", w)
	}
	if h != 200 {
		t.Errorf("height = %v, want 200", h)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !innerCalled {
		t.Error("expected inner resolver to be called for non-SVG source")
	}
	if w != 50 || h != 75 {
		t.Errorf("dimensions = %vx%v, want 50x75", w, h)
	}
}

func TestSVGImageResolver_NilInner_ReturnsDefault(t *testing.T) {
	dataAdapter := NewDataURISVGDataAdapter()
	resolver := NewSVGImageResolver(nil, dataAdapter)

	w, h, err := resolver.GetImageDimensions(context.Background(), "/images/photo.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %vx%v, want 100x100", w, h)
	}
}

type mockImageResolver struct {
	fn func(ctx context.Context, source string) (float64, float64, error)
}

func (m *mockImageResolver) GetImageDimensions(ctx context.Context, source string) (float64, float64, error) {
	return m.fn(ctx, source)
}
