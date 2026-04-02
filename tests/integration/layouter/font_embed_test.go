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
	"encoding/base64"
	"sync"

	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

var (
	fontDataURIOnce      sync.Once
	fontDataURIValue     string
	boldFontDataURIOnce  sync.Once
	boldFontDataURIValue string
)

func base64FontDataURI() string {
	fontDataURIOnce.Do(func() {
		fontDataURIValue = "data:font/ttf;base64," +
			base64.StdEncoding.EncodeToString(fonts.NotoSansRegularTTF)
	})
	return fontDataURIValue
}

func base64BoldFontDataURI() string {
	boldFontDataURIOnce.Do(func() {
		boldFontDataURIValue = "data:font/ttf;base64," +
			base64.StdEncoding.EncodeToString(fonts.NotoSansBoldTTF)
	})
	return boldFontDataURIValue
}

func newTestFontMetrics() (*layouter_adapters.GoTextFontMetrics, error) {
	return layouter_adapters.NewGoTextFontMetrics([]layouter_dto.FontEntry{
		{
			Family: fonts.NotoSansFamilyName,
			Weight: 400,
			Style:  int(layouter_domain.FontStyleNormal),
			Data:   fonts.NotoSansRegularTTF,
		},
		{
			Family: fonts.NotoSansFamilyName,
			Weight: 700,
			Style:  int(layouter_domain.FontStyleNormal),
			Data:   fonts.NotoSansBoldTTF,
		},
	})
}
