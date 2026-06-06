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

// tupleFloat64Pair constructs a Tuple(Float64, Float64) SQLType for the statistical-test
// aggregates that report a (statistic, p-value) pair.
//
// Centralising the shape avoids re-declaring the struct fields at every registration site
// and keeps the engine name aligned with ClickHouse's canonical Tuple spelling.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the element SQL types.
//
// Returns querier_dto.SQLType which is the constructed tuple type.
func tupleFloat64Pair(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "statistic", SQLType: b.float64Type},
			{Name: "p_value", SQLType: b.float64Type},
		},
	}
}

// registerStatisticalAggregateFunctions covers hypothesis-test and rank-correlation
// aggregates.
//
// Each statistical test returns Tuple(Float64, Float64) capturing the test statistic and
// p-value, while the scalar aggregates (Cramer's V, Theil's U, contingency coefficient,
// rank correlation) return a single Float64. It delegates to topical helpers to keep each
// registration helper within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStatisticalAggregateFunctions(b *FunctionCatalogueBuilder) {
	registerStatisticalTwoSampleTests(b)
	registerStatisticalSingleSampleTests(b)
	registerStatisticalAssociationMeasures(b)
}

// registerStatisticalTwoSampleTests covers the two-sample hypothesis tests
// Kolmogorov-Smirnov, Mann-Whitney U, Welch's T and Student's T.
//
// Each registration variant matches a documented signature shape from the ClickHouse
// manual page on hypothesis tests.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStatisticalTwoSampleTests(b *FunctionCatalogueBuilder) {
	pairType := tupleFloat64Pair(b)
	b.RegisterAggregate("kolmogorovSmirnovTest", pairType, b.float64Type, b.float64Type)
	b.RegisterAggregate("kolmogorovSmirnovTest", pairType, b.float64Type, b.float64Type, b.textType)
	b.RegisterAggregate("kolmogorovSmirnovTest", pairType, b.float64Type, b.float64Type, b.textType, b.textType)
	b.RegisterAggregate("mannWhitneyUTest", pairType, b.float64Type, b.float64Type)
	b.RegisterAggregate("mannWhitneyUTest", pairType, b.float64Type, b.float64Type, b.textType)
	b.RegisterAggregate("mannWhitneyUTest", pairType, b.float64Type, b.float64Type, b.textType, b.uint64Type)
	b.RegisterAggregate("welchTTest", pairType, b.float64Type, b.float64Type)
	b.RegisterAggregate("welchTTest", pairType, b.float64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("studentTTest", pairType, b.float64Type, b.float64Type)
	b.RegisterAggregate("studentTTest", pairType, b.float64Type, b.float64Type, b.float64Type)
}

// registerStatisticalSingleSampleTests covers the single-sample hypothesis tests and the
// z-tests on means and proportions which take pre-computed sample statistics rather than
// per-row values.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStatisticalSingleSampleTests(b *FunctionCatalogueBuilder) {
	pairType := tupleFloat64Pair(b)
	b.RegisterAggregate("studentTTestOneSample", pairType, b.float64Type, b.float64Type)
	b.RegisterAggregate("meanZTest", pairType, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("meanZTest", pairType, b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("proportionsZTest", pairType, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type)
	b.RegisterAggregate("proportionsZTest", pairType, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.float64Type, b.textType)
	b.RegisterAggregate("analysisOfVariance", pairType, b.float64Type, b.uint64Type)
}

// registerStatisticalAssociationMeasures covers the categorical association measures
// (Cramer's V, Theil's U, contingency coefficient), the rank-correlation aggregate and
// the categorical information value used for binary classifier feature selection.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStatisticalAssociationMeasures(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("cramersV", b.float64Type, b.unknownType, b.unknownType)
	b.RegisterAggregate("cramersVBiasCorrected", b.float64Type, b.unknownType, b.unknownType)
	b.RegisterAggregate("theilsU", b.float64Type, b.unknownType, b.unknownType)
	b.RegisterAggregate("contingency", b.float64Type, b.unknownType, b.unknownType)
	b.RegisterVariadicAggregate("categoricalInformationValue", b.float64Type, 2, b.unknownType)
	b.RegisterAggregate("rankCorr", b.float64Type, b.float64Type, b.float64Type)
}
