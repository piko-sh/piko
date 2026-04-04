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

// Package cache_linguistics bridges the linguistics text analysis module
// with the cache search system.
module piko.sh/piko/wdk/cache/cache_linguistics

go 1.26.0

require (
	github.com/stretchr/testify v1.11.1
	piko.sh/piko v0.0.0-alpha.12
	piko.sh/piko/wdk/linguistics/linguistics_language_english v0.0.0-alpha.12
	piko.sh/piko/wdk/linguistics/linguistics_language_french v0.0.0-alpha.12
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kljensen/snowball v0.10.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90 // indirect
	golang.org/x/text v0.35.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	piko.sh/piko/wdk/linguistics/linguistics_phonetic_english v0.0.0-alpha.12 // indirect
	piko.sh/piko/wdk/linguistics/linguistics_phonetic_french v0.0.0-alpha.12 // indirect
	piko.sh/piko/wdk/linguistics/linguistics_stemmer_english v0.0.0-alpha.12 // indirect
	piko.sh/piko/wdk/linguistics/linguistics_stemmer_french v0.0.0-alpha.12 // indirect
	piko.sh/piko/wdk/linguistics/linguistics_stopwords_english v0.0.0-alpha.12 // indirect
	piko.sh/piko/wdk/linguistics/linguistics_stopwords_french v0.0.0-alpha.12 // indirect
)
