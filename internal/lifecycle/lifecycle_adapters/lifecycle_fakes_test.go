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

package lifecycle_adapters

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type compileCall struct {
	ctx        context.Context
	packages   map[string]map[string]string
	modulePath string
}

type fakeInterpreter struct {
	compile    func(call compileCall) error
	registered map[string]bool
	calls      []compileCall
	mu         sync.Mutex
}

func (f *fakeInterpreter) CompileAndExecute(ctx context.Context, modulePath string, packages map[string]map[string]string) error {
	call := compileCall{ctx: ctx, modulePath: modulePath, packages: packages}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
	if f.compile == nil {
		return nil
	}
	return f.compile(call)
}

func (f *fakeInterpreter) HasRegisteredPackage(importPath string) bool {
	return f.registered[importPath]
}

func (f *fakeInterpreter) compileCalls() []compileCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]compileCall(nil), f.calls...)
}

type fakePool struct {
	interpreter templater_domain.InterpreterPort
	getErr      error
	loadErr     error
	loadCalls   atomic.Int32
}

func (p *fakePool) LoadModules(_ context.Context) error {
	p.loadCalls.Add(1)
	return p.loadErr
}

func (p *fakePool) Get() (templater_domain.InterpreterPort, error) {
	if p.getErr != nil {
		return nil, p.getErr
	}
	return p.interpreter, nil
}

type fakeProvider struct {
	pool    *fakePool
	created atomic.Int32
}

func (*fakeProvider) RegisterSymbols(templater_domain.SymbolExports) {}

func (p *fakeProvider) NewInterpreterPool() templater_domain.InterpreterPoolPort {
	p.created.Add(1)
	return p.pool
}

func registerTestBuilder(packagePath string) {
	templater_domain.RegisterASTFunc(packagePath, func(*templater_dto.RequestData, any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return nil, templater_dto.InternalMetadata{}, nil
	})
}

func registeringCompile(call compileCall) error {
	for relativePath := range call.packages {
		registerTestBuilder(call.modulePath + "/" + relativePath)
	}
	return nil
}

func newTestArtefact(projectRoot, moduleName, relativePath, content string) *generator_dto.GeneratedArtefact {
	sourcePath := filepath.Join(projectRoot, relativePath)
	packagePath := moduleName + "/" + strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
	hashedName := strings.ReplaceAll(packagePath, "/", "_")
	component := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: packagePath,
		HashedName:             hashedName,
		Source:                 &annotator_dto.ParsedComponent{SourcePath: sourcePath},
	}
	return &generator_dto.GeneratedArtefact{
		Content:   []byte(content),
		Component: component,
		Result: &annotator_dto.AnnotationResult{
			AnnotatedAST: &ast_domain.TemplateAST{SourcePath: &sourcePath},
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{PathToHashedName: map[string]string{sourcePath: hashedName}},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{hashedName: component},
			},
		},
	}
}

func newProjectResult(artefacts ...*generator_dto.GeneratedArtefact) *annotator_dto.ProjectAnnotationResult {
	components := make(map[string]*annotator_dto.VirtualComponent, len(artefacts))
	for _, artefact := range artefacts {
		components[artefact.Component.HashedName] = artefact.Component
	}
	return &annotator_dto.ProjectAnnotationResult{
		VirtualModule:           &annotator_dto.VirtualModule{ComponentsByHash: components},
		FinalGeneratedArtefacts: artefacts,
	}
}

func newTestOrchestrator(t *testing.T, moduleName string, pool templater_domain.InterpreterPoolPort, provider templater_domain.InterpreterProviderPort) (*InterpretedBuildOrchestrator, string) {
	t.Helper()
	projectRoot := t.TempDir()
	return NewInterpretedBuildOrchestrator(InterpretedBuildOrchestratorDeps{
		InterpreterPool:     pool,
		InterpreterProvider: provider,
		ModuleName:          moduleName,
		ProjectRoot:         projectRoot,
	}), projectRoot
}

func withPikoImports(artefact *generator_dto.GeneratedArtefact, importPaths ...string) *generator_dto.GeneratedArtefact {
	for _, importPath := range importPaths {
		artefact.Component.Source.PikoImports = append(artefact.Component.Source.PikoImports, annotator_dto.PikoImport{Path: importPath})
	}
	return artefact
}
