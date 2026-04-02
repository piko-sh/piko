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

//go:build integration

package layouter_test

import (
	"encoding/json"
	"fmt"
	"os"
)

type layouterTestSpec struct {
	Description       string                 `json:"description"`
	RequestURL        string                 `json:"requestURL"`
	Tolerance         float64                `json:"tolerance"`
	ViewportWidth     int                    `json:"viewportWidth"`
	ViewportHeight    int                    `json:"viewportHeight"`
	PageWidthPx       float64                `json:"pageWidthPx"`
	PageHeightPx      float64                `json:"pageHeightPx"`
	PageMarginPx      float64                `json:"pageMarginPx"`
	ExpectedPositions map[string]elementRect `json:"expectedPositions"`
	Skip              string                 `json:"skip"`
}

type elementRect struct {
	PageIndex int           `json:"pageIndex"`
	X         float64       `json:"x"`
	Y         float64       `json:"y"`
	Width     float64       `json:"width"`
	Height    float64       `json:"height"`
	TextRects []elementRect `json:"textRects,omitempty"`
}

func loadTestSpec(path string) (*layouterTestSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading test spec: %w", err)
	}

	var spec layouterTestSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing test spec: %w", err)
	}

	if spec.RequestURL == "" {
		spec.RequestURL = "/main"
	}
	if spec.Tolerance <= 0 {
		spec.Tolerance = 1.0
	}
	if spec.ViewportWidth <= 0 {
		spec.ViewportWidth = 800
	}
	if spec.ViewportHeight <= 0 {
		spec.ViewportHeight = 600
	}

	return &spec, nil
}
