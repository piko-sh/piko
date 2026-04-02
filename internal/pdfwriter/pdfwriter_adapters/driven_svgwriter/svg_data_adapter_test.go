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
	"strings"
	"testing"
)

func TestDataURISVGDataAdapter_Base64(t *testing.T) {
	adapter := NewDataURISVGDataAdapter()
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><rect width="100" height="100"/></svg>`
	encoded := base64.StdEncoding.EncodeToString([]byte(svgContent))
	source := "data:image/svg+xml;base64," + encoded

	data, ok := adapter.GetSVGData(context.Background(), source)
	if !ok {
		t.Fatal("expected ok=true for SVG data URI")
	}
	if !strings.Contains(data, "<svg") {
		t.Errorf("expected SVG markup, got %q", data[:50])
	}
}

func TestDataURISVGDataAdapter_PlainText(t *testing.T) {
	adapter := NewDataURISVGDataAdapter()
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`
	source := "data:image/svg+xml," + svgContent

	data, ok := adapter.GetSVGData(context.Background(), source)
	if !ok {
		t.Fatal("expected ok=true for plain text SVG data URI")
	}
	if data != svgContent {
		t.Errorf("expected %q, got %q", svgContent, data)
	}
}

func TestDataURISVGDataAdapter_NonSVG(t *testing.T) {
	adapter := NewDataURISVGDataAdapter()

	_, ok := adapter.GetSVGData(context.Background(), "data:image/png;base64,abc")
	if ok {
		t.Error("expected ok=false for non-SVG data URI")
	}

	_, ok = adapter.GetSVGData(context.Background(), "/images/photo.jpg")
	if ok {
		t.Error("expected ok=false for non-data-URI source")
	}
}
