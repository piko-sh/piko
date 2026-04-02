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

package pdf_test

import (
	"encoding/json"
	"fmt"
	"os"
)

type pdfTestSpec struct {
	Description   string             `json:"description"`
	PdfPath       string             `json:"pdfPath"`
	ComparisonURL string             `json:"comparisonURL"`
	RegularOnly   bool               `json:"regularOnly"`
	VariableFont  bool               `json:"variableFont"`
	Encryption    *pdfEncryptionSpec `json:"encryption,omitempty"`
}

type pdfEncryptionSpec struct {
	UserPassword  string `json:"userPassword"`
	OwnerPassword string `json:"ownerPassword"`
}

func loadTestSpec(path string) (*pdfTestSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading test spec: %w", err)
	}

	var spec pdfTestSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing test spec: %w", err)
	}

	if spec.PdfPath == "" {
		spec.PdfPath = "pdfs/main.pk"
	}
	if spec.ComparisonURL == "" {
		spec.ComparisonURL = "/"
	}

	return &spec, nil
}
