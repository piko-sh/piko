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

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

// DataURISVGDataAdapter implements SVGDataPort by extracting SVG markup
// from inline data URIs (data:image/svg+xml;base64,...). Non-SVG sources
// or non-data-URI sources return false.
type DataURISVGDataAdapter struct{}

var _ pdfwriter_domain.SVGDataPort = (*DataURISVGDataAdapter)(nil)

// NewDataURISVGDataAdapter creates an adapter that extracts SVG data
// from data URIs.
//
// Returns *DataURISVGDataAdapter which implements SVGDataPort.
func NewDataURISVGDataAdapter() *DataURISVGDataAdapter {
	return &DataURISVGDataAdapter{}
}

// GetSVGData extracts the raw SVG XML from a data URI.
//
// Takes source (string) which is the data URI to extract SVG data from.
//
// Returns string which is the decoded SVG XML markup.
// Returns bool which is true if the source is an SVG data URI, false otherwise.
func (*DataURISVGDataAdapter) GetSVGData(_ context.Context, source string) (string, bool) {
	if !strings.HasPrefix(source, "data:image/svg+xml") {
		return "", false
	}

	commaIdx := strings.Index(source, ",")
	if commaIdx < 0 {
		return "", false
	}

	header := source[len("data:"):commaIdx]
	encoded := source[commaIdx+1:]

	if strings.Contains(header, "base64") {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", false
		}
		return string(data), true
	}

	return encoded, true
}
