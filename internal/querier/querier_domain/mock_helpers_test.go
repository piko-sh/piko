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

package querier_domain

import (
	"context"
	"io/fs"
	"os"

	"piko.sh/piko/internal/querier/querier_dto"
)

type mockEngine struct {
	parseStatementsFn            func(sql string) ([]querier_dto.ParsedStatement, error)
	applyDDLFn                   func(statement querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error)
	analyseQueryFn               func(catalogue *querier_dto.Catalogue, statement querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error)
	normaliseTypeNameFn          func(name string, modifiers ...int) querier_dto.SQLType
	promoteTypeFn                func(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType
	canImplicitCastFn            func(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool
	supportedExpressionsFn       func() querier_dto.SQLExpressionFeature
	commentStyleFn               func() querier_dto.CommentStyle
	supportedDirectivePrefixesFn func() []querier_dto.DirectiveParameterPrefix
	defaultSchemaFn              func() string
	builtinFunctionsFn           func() *querier_dto.FunctionCatalogue
	builtinTypesFn               func() *querier_dto.TypeCatalogue
	tableValuedFunctionColumnsFn func(functionName string) []querier_dto.ScopedColumn
	parameterStyleFn             func() querier_dto.ParameterStyle
	supportsReturningFn          func() bool
	dialectFn                    func() string
}

func (m *mockEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
	if m.parseStatementsFn != nil {
		return m.parseStatementsFn(sql)
	}
	return nil, nil
}

func (m *mockEngine) ApplyDDL(statement querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error) {
	if m.applyDDLFn != nil {
		return m.applyDDLFn(statement)
	}
	return nil, nil
}

func (m *mockEngine) AnalyseQuery(catalogue *querier_dto.Catalogue, statement querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error) {
	if m.analyseQueryFn != nil {
		return m.analyseQueryFn(catalogue, statement)
	}
	return nil, nil
}

func (m *mockEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	if m.normaliseTypeNameFn != nil {
		return m.normaliseTypeNameFn(name, modifiers...)
	}
	return querier_dto.SQLType{
		EngineName: name,
		Category:   querier_dto.TypeCategoryUnknown,
	}
}

func (m *mockEngine) PromoteType(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType {
	if m.promoteTypeFn != nil {
		return m.promoteTypeFn(left, right)
	}
	return left
}

func (m *mockEngine) CanImplicitCast(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
	if m.canImplicitCastFn != nil {
		return m.canImplicitCastFn(from, to)
	}
	return false
}

func (m *mockEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	if m.supportedExpressionsFn != nil {
		return m.supportedExpressionsFn()
	}
	return querier_dto.SQLFeaturesAll
}

func (m *mockEngine) CommentStyle() querier_dto.CommentStyle {
	if m.commentStyleFn != nil {
		return m.commentStyleFn()
	}
	return querier_dto.CommentStyle{LinePrefix: "--"}
}

func (m *mockEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	if m.supportedDirectivePrefixesFn != nil {
		return m.supportedDirectivePrefixesFn()
	}
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '$', IsNamed: false},
	}
}

func (m *mockEngine) DefaultSchema() string {
	if m.defaultSchemaFn != nil {
		return m.defaultSchemaFn()
	}
	return "public"
}

func (m *mockEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	if m.builtinFunctionsFn != nil {
		return m.builtinFunctionsFn()
	}
	return &querier_dto.FunctionCatalogue{
		Functions: make(map[string][]*querier_dto.FunctionSignature),
	}
}

func (m *mockEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	if m.builtinTypesFn != nil {
		return m.builtinTypesFn()
	}
	return &querier_dto.TypeCatalogue{
		Types: make(map[string]querier_dto.SQLType),
	}
}

func (m *mockEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	if m.tableValuedFunctionColumnsFn != nil {
		return m.tableValuedFunctionColumnsFn(functionName)
	}
	return nil
}

func (m *mockEngine) ParameterStyle() querier_dto.ParameterStyle {
	if m.parameterStyleFn != nil {
		return m.parameterStyleFn()
	}
	return querier_dto.ParameterStyleDollar
}

func (m *mockEngine) SupportsReturning() bool {
	if m.supportsReturningFn != nil {
		return m.supportsReturningFn()
	}
	return true
}

func (m *mockEngine) Dialect() string {
	if m.dialectFn != nil {
		return m.dialectFn()
	}
	return "mock"
}

type mockFileReader struct {
	files       map[string][]byte
	dirs        map[string][]os.DirEntry
	readFileErr map[string]error
	readDirErr  map[string]error
}

func (m *mockFileReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	if m.readFileErr != nil {
		if err, ok := m.readFileErr[path]; ok {
			return nil, err
		}
	}
	if m.files != nil {
		if content, ok := m.files[path]; ok {
			return content, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *mockFileReader) ReadDir(_ context.Context, directory string) ([]os.DirEntry, error) {
	if m.readDirErr != nil {
		if err, ok := m.readDirErr[directory]; ok {
			return nil, err
		}
	}
	if m.dirs != nil {
		if entries, ok := m.dirs[directory]; ok {
			return entries, nil
		}
	}
	return nil, os.ErrNotExist
}

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string {
	return m.name
}

func (m *mockDirEntry) IsDir() bool {
	return m.isDir
}

func (m *mockDirEntry) Type() fs.FileMode {
	if m.isDir {
		return os.ModeDir
	}
	return 0
}

func (m *mockDirEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}

type mockCodeEmitter struct {
	emitModelsFn   func(packageName string, catalogue *querier_dto.Catalogue, mappings *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error)
	emitQueriesFn  func(packageName string, queries []*querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error)
	emitQuerierFn  func(packageName string, capabilities querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error)
	emitPreparedFn func(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error)
	emitOTelFn     func(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error)
}

func (m *mockCodeEmitter) EmitModels(packageName string, catalogue *querier_dto.Catalogue, mappings *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error) {
	if m.emitModelsFn != nil {
		return m.emitModelsFn(packageName, catalogue, mappings)
	}
	return nil, nil
}

func (m *mockCodeEmitter) EmitQueries(packageName string, queries []*querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error) {
	if m.emitQueriesFn != nil {
		return m.emitQueriesFn(packageName, queries, mappings)
	}
	return nil, nil
}

func (m *mockCodeEmitter) EmitQuerier(packageName string, capabilities querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
	if m.emitQuerierFn != nil {
		return m.emitQuerierFn(packageName, capabilities)
	}
	return querier_dto.GeneratedFile{}, nil
}

func (m *mockCodeEmitter) EmitPrepared(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	if m.emitPreparedFn != nil {
		return m.emitPreparedFn(packageName, queries)
	}
	return querier_dto.GeneratedFile{}, nil
}

func (m *mockCodeEmitter) EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	if m.emitOTelFn != nil {
		return m.emitOTelFn(packageName, queries)
	}
	return querier_dto.GeneratedFile{}, nil
}

type mockMigrationExecutor struct {
	ensureMigrationTableFn func(ctx context.Context) error
	acquireLockFn          func(ctx context.Context) error
	tryAcquireLockFn       func(ctx context.Context) error
	releaseLockFn          func(ctx context.Context) error
	appliedVersionsFn      func(ctx context.Context) ([]querier_dto.AppliedMigration, error)
	executeMigrationFn     func(ctx context.Context, migration querier_dto.MigrationRecord, direction querier_dto.MigrationDirection, useTransaction bool) error
}

func (m *mockMigrationExecutor) EnsureMigrationTable(ctx context.Context) error {
	if m.ensureMigrationTableFn != nil {
		return m.ensureMigrationTableFn(ctx)
	}
	return nil
}

func (m *mockMigrationExecutor) AcquireLock(ctx context.Context) error {
	if m.acquireLockFn != nil {
		return m.acquireLockFn(ctx)
	}
	return nil
}

func (m *mockMigrationExecutor) TryAcquireLock(ctx context.Context) error {
	if m.tryAcquireLockFn != nil {
		return m.tryAcquireLockFn(ctx)
	}
	return nil
}

func (m *mockMigrationExecutor) ReleaseLock(ctx context.Context) error {
	if m.releaseLockFn != nil {
		return m.releaseLockFn(ctx)
	}
	return nil
}

func (m *mockMigrationExecutor) AppliedVersions(ctx context.Context) ([]querier_dto.AppliedMigration, error) {
	if m.appliedVersionsFn != nil {
		return m.appliedVersionsFn(ctx)
	}
	return nil, nil
}

func (m *mockMigrationExecutor) ExecuteMigration(ctx context.Context, migration querier_dto.MigrationRecord, direction querier_dto.MigrationDirection, useTransaction bool) error {
	if m.executeMigrationFn != nil {
		return m.executeMigrationFn(ctx, migration, direction, useTransaction)
	}
	return nil
}

func boolPtr(b bool) *bool { return new(b) }

func intPtr(i int) *int { return new(i) }

func stringPtr(s string) *string { return new(s) }

func newTestCatalogue(schema string) *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		DefaultSchema: schema,
		Schemas: map[string]*querier_dto.Schema{
			schema: {
				Name:           schema,
				Tables:         make(map[string]*querier_dto.Table),
				Views:          make(map[string]*querier_dto.View),
				Enums:          make(map[string]*querier_dto.Enum),
				Functions:      make(map[string][]*querier_dto.FunctionSignature),
				CompositeTypes: make(map[string]*querier_dto.CompositeType),
				Sequences:      make(map[string]*querier_dto.Sequence),
			},
		},
		Extensions: make(map[string]struct{}),
	}
}

func newTestTable(name string, columns ...querier_dto.Column) *querier_dto.Table {
	return &querier_dto.Table{
		Name:    name,
		Columns: columns,
	}
}

type mockSeedExecutor struct {
	ensureSeedTableFn  func(ctx context.Context) error
	appliedSeedsFn     func(ctx context.Context) ([]querier_dto.AppliedSeed, error)
	executeSeedFn      func(ctx context.Context, seed querier_dto.SeedRecord) error
	clearSeedHistoryFn func(ctx context.Context) error
}

func (m *mockSeedExecutor) EnsureSeedTable(ctx context.Context) error {
	if m.ensureSeedTableFn != nil {
		return m.ensureSeedTableFn(ctx)
	}
	return nil
}

func (m *mockSeedExecutor) AppliedSeeds(ctx context.Context) ([]querier_dto.AppliedSeed, error) {
	if m.appliedSeedsFn != nil {
		return m.appliedSeedsFn(ctx)
	}
	return nil, nil
}

func (m *mockSeedExecutor) ExecuteSeed(ctx context.Context, seed querier_dto.SeedRecord) error {
	if m.executeSeedFn != nil {
		return m.executeSeedFn(ctx, seed)
	}
	return nil
}

func (m *mockSeedExecutor) ClearSeedHistory(ctx context.Context) error {
	if m.clearSeedHistoryFn != nil {
		return m.clearSeedHistoryFn(ctx)
	}
	return nil
}

var (
	_ EnginePort            = (*mockEngine)(nil)
	_ FileReaderPort        = (*mockFileReader)(nil)
	_ os.DirEntry           = (*mockDirEntry)(nil)
	_ CodeEmitterPort       = (*mockCodeEmitter)(nil)
	_ MigrationExecutorPort = (*mockMigrationExecutor)(nil)
	_ SeedExecutorPort      = (*mockSeedExecutor)(nil)
)
