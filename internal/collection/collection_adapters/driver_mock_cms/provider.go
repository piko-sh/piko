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

package driver_mock_cms

import (
	"context"
	"errors"
	"fmt"
	goast "go/ast"
	"go/token"
	"time"

	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// defaultCacheTTLSeconds is the default cache time to live in seconds for
// dynamic providers.
const defaultCacheTTLSeconds = 60

// MockCMSProvider simulates a headless CMS for testing and documentation.
//
// This is a reference implementation that shows how to build a dynamic
// collection provider. At build time it creates runtime fetcher function AST,
// and at runtime the RuntimeProvider fetches the actual data.
type MockCMSProvider struct {
	// name is the unique name for this provider.
	name string
}

// NewMockCMSProvider creates a new mock CMS provider.
//
// Takes name (string) which is the unique provider name (e.g., "mock-cms").
//
// Returns *MockCMSProvider which is a fully initialised provider.
func NewMockCMSProvider(name string) *MockCMSProvider {
	return &MockCMSProvider{
		name: name,
	}
}

// Name returns the unique identifier for this provider.
//
// Returns string which is the provider's unique name.
func (p *MockCMSProvider) Name() string {
	return p.name
}

// Check implements the healthprobe_domain.Probe interface.
// The mock CMS provider is always healthy as it has no external dependencies.
//
// Returns healthprobe_dto.Status which always reports healthy.
func (p *MockCMSProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Mock CMS provider operational",
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// Type returns the provider type (dynamic).
//
// Returns collection_domain.ProviderType which identifies this as a dynamic
// provider.
func (*MockCMSProvider) Type() collection_domain.ProviderType {
	return collection_domain.ProviderTypeDynamic
}

// DiscoverCollections returns an empty list; the mock has no content types.
//
// Returns []collection_dto.CollectionInfo which is always empty for this mock.
// Returns error which is always nil for this mock.
func (*MockCMSProvider) DiscoverCollections(
	ctx context.Context,
	_ collection_dto.ProviderConfig,
) ([]collection_dto.CollectionInfo, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Mock CMS provider discovery called")
	return []collection_dto.CollectionInfo{}, nil
}

// ValidateTargetType accepts any target type without validation.
//
// Returns error when the target type is incompatible with the CMS schema.
func (*MockCMSProvider) ValidateTargetType(_ goast.Expr) error {
	return nil
}

// FetchStaticContent is not supported for dynamic providers.
//
// Dynamic providers fetch data at runtime, not build time.
//
// Returns []collection_dto.ContentItem which is always nil for this provider.
// Returns error when called, as static fetching is not supported.
func (*MockCMSProvider) FetchStaticContent(
	_ context.Context,
	_ string,
) ([]collection_dto.ContentItem, error) {
	return nil, errors.New("mock CMS provider does not support static fetching (dynamic only)")
}

// GenerateRuntimeFetcher generates the Go AST for a runtime fetcher function.
//
// This is the core method for dynamic providers. It creates a complete
// function definition that will be injected into the component's generated
// code.
//
// The generated function will:
//  1. Declare a results variable of the target type
//  2. Call pikoruntime.FetchCollection to fetch data
//  3. Handle errors
//  4. Return the results
//
// Takes collectionName (string) which specifies the collection name
// (e.g. "blog").
// Takes targetType (goast.Expr) which provides the Go AST for the target type
// (e.g. AST for "Post").
// Takes options (collection_dto.FetchOptions) which configures fetch behaviour
// such as cache and locale settings.
//
// Returns *collection_dto.RuntimeFetcherCode which contains the complete
// function AST and metadata.
// Returns error when generation fails.
func (p *MockCMSProvider) GenerateRuntimeFetcher(
	ctx context.Context,
	collectionName string,
	targetType goast.Expr,
	options collection_dto.FetchOptions,
) (*collection_dto.RuntimeFetcherCode, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Generating runtime fetcher for mock CMS",
		logger_domain.String("collection", collectionName))

	fetcherFunc := p.buildFetcherFunctionAST(collectionName, targetType)

	var cacheStrategy string
	var cacheTTL time.Duration
	if options.Cache != nil {
		cacheStrategy = options.Cache.Strategy
		cacheTTL = options.Cache.GetTTLDuration()
	} else {
		cacheStrategy = "cache-first"
		cacheTTL = defaultCacheTTLSeconds * time.Second
	}

	return &collection_dto.RuntimeFetcherCode{
		FetcherFunc: fetcherFunc,
		RequiredImports: map[string]string{
			"context":                  "",
			"piko.sh/piko/wdk/runtime": "pikoruntime",
		},
		CacheStrategy: cacheStrategy,
		CacheTTL:      cacheTTL,
		RetryConfig:   collection_dto.DefaultRetryConfig(),
	}, nil
}

// buildFetcherFunctionAST builds the Go AST for a runtime fetcher function.
//
// Takes collectionName (string) which names the CMS collection to fetch.
// Takes targetType (goast.Expr) which sets the element type for the returned
// slice.
//
// Returns *goast.FuncDecl which is the AST for a function that fetches and
// returns a slice of the target type.
func (p *MockCMSProvider) buildFetcherFunctionAST(collectionName string, targetType goast.Expr) *goast.FuncDecl {
	sliceType := &goast.ArrayType{Elt: targetType}

	statements := []goast.Stmt{
		buildVarDeclStmt(sliceType),
		buildOptsStmt(),
		buildFetchCallStmt(p.name, collectionName),
		buildErrorCheckStmt(),
		&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("results"), goast.NewIdent("nil")}},
	}

	return &goast.FuncDecl{
		Name: goast.NewIdent("fetchCollectionFunc"),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{{
					Names: []*goast.Ident{goast.NewIdent("ctx")},
					Type:  &goast.SelectorExpr{X: goast.NewIdent("context"), Sel: goast.NewIdent("Context")},
				}},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{{Type: sliceType}, {Type: goast.NewIdent("error")}},
			},
		},
		Body: &goast.BlockStmt{List: statements},
	}
}

// buildVarDeclStmt builds a variable declaration statement of the form
// "var results []TargetType".
//
// Takes sliceType (goast.Expr) which specifies the slice type for the results
// variable.
//
// Returns goast.Stmt which is the built variable declaration statement.
func buildVarDeclStmt(sliceType goast.Expr) goast.Stmt {
	return &goast.DeclStmt{
		Decl: &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{goast.NewIdent("results")},
					Type:  sliceType,
				},
			},
		},
	}
}

// buildOptsStmt builds an assignment statement of the form
// "opts := pikoruntime.FetchOptions{}".
//
// Returns goast.Stmt which is the constructed assignment statement.
func buildOptsStmt() goast.Stmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent("opts")},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{
			&goast.CompositeLit{
				Type: &goast.SelectorExpr{
					X:   goast.NewIdent("pikoruntime"),
					Sel: goast.NewIdent("FetchOptions"),
				},
			},
		},
	}
}

// buildFetchCallStmt builds an assignment statement of the form "err :=
// pikoruntime.FetchCollection(ctx, providerName, collectionName, &opts,
// &results)".
//
// Takes providerName (string) which is the provider identifier.
// Takes collectionName (string) which is the collection to fetch.
//
// Returns goast.Stmt which is the constructed assignment statement.
func buildFetchCallStmt(providerName, collectionName string) goast.Stmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent("err")},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("pikoruntime"),
					Sel: goast.NewIdent("FetchCollection"),
				},
				Args: []goast.Expr{
					goast.NewIdent("ctx"),
					&goast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", providerName)},
					&goast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", collectionName)},
					&goast.UnaryExpr{Op: token.AND, X: goast.NewIdent("opts")},
					&goast.UnaryExpr{Op: token.AND, X: goast.NewIdent("results")},
				},
			},
		},
	}
}

// buildErrorCheckStmt builds an if statement that checks whether err is not
// nil and returns nil and err if so.
//
// Returns goast.Stmt which is the constructed error check statement.
func buildErrorCheckStmt() goast.Stmt {
	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: goast.NewIdent("err"), Op: token.NEQ, Y: goast.NewIdent("nil")},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("nil"), goast.NewIdent("err")}},
			},
		},
	}
}
