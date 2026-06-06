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

package db_engine_clickhouse

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

// registerNLPFunctions covers the natural-language processing helpers.
//
// These span charset and language detection, stemming, lemmatising, synonym lookup, and
// tonality scoring. Language detection in mixed mode returns a Map(String, Float32) of
// language codes to confidence scores.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered function signatures.
func registerNLPFunctions(b *FunctionCatalogueBuilder) {
	scoreType := querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryMap,
		EngineName:  engineNameMap,
		KeyType:     new(b.textType),
		ElementType: new(b.float32Type),
	}
	b.Register("detectCharset", b.textType, b.textType)
	b.Register("detectLanguage", b.textType, b.textType)
	b.Register("detectLanguageMixed", scoreType, b.textType)
	b.Register("detectLanguageUnknown", b.textType, b.textType)
	b.Register("stem", b.textType, b.textType, b.textType)
	b.Register("lemmatize", b.textType, b.textType, b.textType)
	b.Register("synonyms", arrayOf(b.textType), b.textType, b.textType)
	b.Register("detectTonality", b.float32Type, b.textType)
}
