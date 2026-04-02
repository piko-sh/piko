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

package db_engine_duckdb_test

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

type recordingCodeEmitter struct {
	queries   []*querier_dto.AnalysedQuery
	catalogue *querier_dto.Catalogue
}

func (emitter *recordingCodeEmitter) EmitModels(
	_ string,
	catalogue *querier_dto.Catalogue,
	_ *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	emitter.catalogue = catalogue
	return nil, nil
}

func (emitter *recordingCodeEmitter) EmitQueries(
	_ string,
	queries []*querier_dto.AnalysedQuery,
	_ *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	emitter.queries = queries
	return nil, nil
}

func (*recordingCodeEmitter) EmitQuerier(_ string, _ querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{
		Name:    "querier.go",
		Content: []byte("// stub"),
	}, nil
}

func (*recordingCodeEmitter) EmitPrepared(_ string, _ []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{
		Name:    "prepared.go",
		Content: []byte("// stub"),
	}, nil
}

func (*recordingCodeEmitter) EmitOTel(_ string, _ []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{
		Name:    "otel.go",
		Content: []byte("// stub"),
	}, nil
}
