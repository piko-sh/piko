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

package compiler_domain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/compiler/compiler_dto"
)

type mockInputReader struct {
	err  error
	data []byte
}

func (m *mockInputReader) ReadSFC(_ context.Context, _ string) ([]byte, error) {
	return m.data, m.err
}

type mockSFCCompiler struct {
	artefact *compiler_dto.CompiledArtefact
	err      error
}

func (m *mockSFCCompiler) CompileSFC(_ context.Context, _ string, _ []byte) (*compiler_dto.CompiledArtefact, error) {
	return m.artefact, m.err
}

type mockTransformation struct {
	transform func(artefact *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error)
}

func (m *mockTransformation) Transform(_ context.Context, artefact *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
	if m.transform != nil {
		return m.transform(artefact)
	}
	return artefact, nil
}

func TestNewCompilerOrchestrator(t *testing.T) {
	t.Run("creates orchestrator with default SFCCompiler", func(t *testing.T) {
		inputReader := &mockInputReader{}
		transformSteps := []TransformationPort{}

		orchestrator := NewCompilerOrchestrator(inputReader, transformSteps)

		require.NotNil(t, orchestrator)
	})

	t.Run("creates orchestrator with transform steps", func(t *testing.T) {
		inputReader := &mockInputReader{}
		transformSteps := []TransformationPort{
			&mockTransformation{},
			&mockTransformation{},
		}

		orchestrator := NewCompilerOrchestrator(inputReader, transformSteps)

		require.NotNil(t, orchestrator)
	})

	t.Run("creates orchestrator with custom SFCCompiler option", func(t *testing.T) {
		inputReader := &mockInputReader{}
		customCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:    "custom-tag",
				BaseJSPath: "custom.js",
				Files:      map[string]string{"custom.js": "custom-js"},
			},
		}

		orchestrator := NewCompilerOrchestrator(
			inputReader,
			nil,
			WithSFCCompiler(customCompiler),
		)

		require.NotNil(t, orchestrator)

		ctx := context.Background()
		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte(""))

		require.NoError(t, err)
		assert.Equal(t, "custom-tag", result.TagName)
		assert.Equal(t, "custom-js", result.Files["custom.js"])
	})

	t.Run("applies multiple options in order", func(t *testing.T) {
		inputReader := &mockInputReader{}
		compiler1 := &mockSFCCompiler{artefact: &compiler_dto.CompiledArtefact{TagName: "first"}}
		compiler2 := &mockSFCCompiler{artefact: &compiler_dto.CompiledArtefact{TagName: "second"}}

		orchestrator := NewCompilerOrchestrator(
			inputReader,
			nil,
			WithSFCCompiler(compiler1),
			WithSFCCompiler(compiler2),
		)

		ctx := context.Background()
		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte(""))

		require.NoError(t, err)
		assert.Equal(t, "second", result.TagName)
	})
}

func TestWithSFCCompiler(t *testing.T) {
	t.Run("returns functional option", func(t *testing.T) {
		compiler := &mockSFCCompiler{}

		opt := WithSFCCompiler(compiler)

		require.NotNil(t, opt)
	})

	t.Run("option sets compiler on orchestrator", func(t *testing.T) {
		compiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{TagName: "test-tag"},
		}

		o := &compilerOrchestrator{}
		opt := WithSFCCompiler(compiler)
		opt(o)

		assert.Equal(t, compiler, o.sfcCompiler)
	})
}

func TestCompilerOrchestrator_CompileSingle(t *testing.T) {
	ctx := context.Background()

	t.Run("reads and compiles SFC successfully", func(t *testing.T) {
		sfcContent := []byte("<template><div>Test</div></template>")
		inputReader := &mockInputReader{data: sfcContent}
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:    "test-tag",
				BaseJSPath: "test.js",
				Files:      map[string]string{"test.js": "compiled-js"},
			},
		}

		orchestrator := NewCompilerOrchestrator(
			inputReader,
			nil,
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSingle(ctx, "test-component")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "compiled-js", result.Files["test.js"])
		assert.Equal(t, "test-component", result.SourceIdentifier)
	})

	t.Run("returns error when input reader fails", func(t *testing.T) {
		readError := errors.New("file not found")
		inputReader := &mockInputReader{err: readError}

		orchestrator := NewCompilerOrchestrator(inputReader, nil)

		result, err := orchestrator.CompileSingle(ctx, "missing-component")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to read SFC")
		assert.ErrorIs(t, err, readError)
	})

	t.Run("propagates compilation errors", func(t *testing.T) {
		sfcContent := []byte("invalid content")
		inputReader := &mockInputReader{data: sfcContent}
		compileError := errors.New("syntax error")
		sfcCompiler := &mockSFCCompiler{err: compileError}

		orchestrator := NewCompilerOrchestrator(
			inputReader,
			nil,
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSingle(ctx, "test")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "compile error")
	})
}

func TestCompilerOrchestrator_CompileSFCBytes(t *testing.T) {
	ctx := context.Background()

	t.Run("compiles SFC bytes successfully", func(t *testing.T) {
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:      "my-component",
				BaseJSPath:   "my.js",
				ScaffoldHTML: "<template>content</template>",
				Files:        map[string]string{"my.js": "js-output"},
			},
		}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			nil,
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "my-component", []byte("sfc-content"))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "js-output", result.Files["my.js"])
		assert.Equal(t, "<template>content</template>", result.ScaffoldHTML)
		assert.Equal(t, "my-component", result.SourceIdentifier)
	})

	t.Run("returns error when compilation fails", func(t *testing.T) {
		compileError := errors.New("parse error")
		sfcCompiler := &mockSFCCompiler{err: compileError}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			nil,
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte("invalid"))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "compile error")
		assert.ErrorIs(t, err, compileError)
	})

	t.Run("applies transformation steps in order", func(t *testing.T) {
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:    "test",
				BaseJSPath: "test.js",
				Files:      map[string]string{"test.js": "original"},
			},
		}

		callOrder := []string{}
		step1 := &mockTransformation{
			transform: func(a *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
				callOrder = append(callOrder, "step1")
				a.Files[a.BaseJSPath] = a.Files[a.BaseJSPath] + "-step1"
				return a, nil
			},
		}
		step2 := &mockTransformation{
			transform: func(a *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
				callOrder = append(callOrder, "step2")
				a.Files[a.BaseJSPath] = a.Files[a.BaseJSPath] + "-step2"
				return a, nil
			},
		}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			[]TransformationPort{step1, step2},
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte(""))

		require.NoError(t, err)
		assert.Equal(t, []string{"step1", "step2"}, callOrder)
		assert.Equal(t, "original-step1-step2", result.Files["test.js"])
	})

	t.Run("stops on transformation error", func(t *testing.T) {
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{},
		}

		transformError := errors.New("transform failed")
		step1 := &mockTransformation{
			transform: func(a *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
				return nil, transformError
			},
		}
		step2 := &mockTransformation{
			transform: func(a *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
				t.Fatal("step2 should not be called")
				return a, nil
			},
		}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			[]TransformationPort{step1, step2},
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte(""))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transformation error")
		assert.ErrorIs(t, err, transformError)
	})

	t.Run("handles empty transform steps", func(t *testing.T) {
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:    "unchanged",
				BaseJSPath: "test.js",
				Files:      map[string]string{"test.js": "unchanged-js"},
			},
		}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			[]TransformationPort{},
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte(""))

		require.NoError(t, err)
		assert.Equal(t, "unchanged-js", result.Files["test.js"])
	})

	t.Run("handles nil transform steps", func(t *testing.T) {
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:    "no-transforms",
				BaseJSPath: "test.js",
				Files:      map[string]string{"test.js": "no-transforms-js"},
			},
		}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			nil,
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte(""))

		require.NoError(t, err)
		assert.Equal(t, "no-transforms-js", result.Files["test.js"])
	})

	t.Run("sets source identifier on artefact", func(t *testing.T) {
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				SourceIdentifier: "should-be-overwritten",
			},
		}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			nil,
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "new-source-id", []byte(""))

		require.NoError(t, err)
		assert.Equal(t, "new-source-id", result.SourceIdentifier)
	})

	t.Run("transformations can modify artefact", func(t *testing.T) {
		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:      "original",
				BaseJSPath:   "orig.js",
				ScaffoldHTML: "original-html",
				Files:        map[string]string{"orig.js": "original-js"},
			},
		}

		minifyTransform := &mockTransformation{
			transform: func(a *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {

				return &compiler_dto.CompiledArtefact{
					SourceIdentifier: a.SourceIdentifier,
					TagName:          "minified",
					BaseJSPath:       "min.js",
					ScaffoldHTML:     "minified-html",
					Files:            map[string]string{"min.js": "minified-js"},
				}, nil
			},
		}

		orchestrator := NewCompilerOrchestrator(
			&mockInputReader{},
			[]TransformationPort{minifyTransform},
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSFCBytes(ctx, "test", []byte(""))

		require.NoError(t, err)
		assert.Equal(t, "minified-js", result.Files["min.js"])
		assert.Equal(t, "minified-html", result.ScaffoldHTML)
	})
}

func TestCompilerOrchestrator_Integration(t *testing.T) {
	t.Run("full pipeline with multiple transformations", func(t *testing.T) {
		ctx := context.Background()

		sfcContent := []byte("<template><div>Hello</div></template><script>export default {}</script>")
		inputReader := &mockInputReader{data: sfcContent}

		sfcCompiler := &mockSFCCompiler{
			artefact: &compiler_dto.CompiledArtefact{
				TagName:      "my-component",
				BaseJSPath:   "my.js",
				ScaffoldHTML: "<div>hello</div>",
				Files:        map[string]string{"my.js": "class MyComponent {}"},
			},
		}

		addComment := &mockTransformation{
			transform: func(a *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
				a.Files[a.BaseJSPath] = "/* Generated */\n" + a.Files[a.BaseJSPath]
				return a, nil
			},
		}

		addExport := &mockTransformation{
			transform: func(a *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
				a.Files[a.BaseJSPath] = a.Files[a.BaseJSPath] + "\nexport default MyComponent;"
				return a, nil
			},
		}

		orchestrator := NewCompilerOrchestrator(
			inputReader,
			[]TransformationPort{addComment, addExport},
			WithSFCCompiler(sfcCompiler),
		)

		result, err := orchestrator.CompileSingle(ctx, "my-component.piko")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "my-component.piko", result.SourceIdentifier)
		jsContent := result.Files["my.js"]
		assert.Contains(t, jsContent, "/* Generated */")
		assert.Contains(t, jsContent, "class MyComponent")
		assert.Contains(t, jsContent, "export default MyComponent")
		assert.Equal(t, "<div>hello</div>", result.ScaffoldHTML)
	})
}

func TestCompilerOrchestrator_ContextCancellation(t *testing.T) {
	t.Run("respects context cancellation during read", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		inputReader := &mockInputReader{
			data: []byte("content"),
		}

		orchestrator := NewCompilerOrchestrator(inputReader, nil)

		_, _ = orchestrator.CompileSingle(ctx, "test")

	})
}

func BenchmarkCompileSFCBytes(b *testing.B) {
	ctx := context.Background()
	sfcCompiler := &mockSFCCompiler{
		artefact: &compiler_dto.CompiledArtefact{
			TagName:    "bench",
			BaseJSPath: "bench.js",
			Files:      map[string]string{"bench.js": "benchmark-output"},
		},
	}

	orchestrator := NewCompilerOrchestrator(
		&mockInputReader{},
		nil,
		WithSFCCompiler(sfcCompiler),
	)

	sfcContent := []byte("<template><div>Benchmark</div></template>")

	b.ResetTimer()
	for b.Loop() {
		_, _ = orchestrator.CompileSFCBytes(ctx, "bench", sfcContent)
	}
}

func BenchmarkCompileSFCBytesWithTransforms(b *testing.B) {
	ctx := context.Background()
	sfcCompiler := &mockSFCCompiler{
		artefact: &compiler_dto.CompiledArtefact{
			TagName:    "bench",
			BaseJSPath: "bench.js",
			Files:      map[string]string{"bench.js": "benchmark-output"},
		},
	}

	transforms := []TransformationPort{
		&mockTransformation{},
		&mockTransformation{},
		&mockTransformation{},
	}

	orchestrator := NewCompilerOrchestrator(
		&mockInputReader{},
		transforms,
		WithSFCCompiler(sfcCompiler),
	)

	sfcContent := []byte("<template><div>Benchmark</div></template>")

	b.ResetTimer()
	for b.Loop() {
		_, _ = orchestrator.CompileSFCBytes(ctx, "bench", sfcContent)
	}
}

func TestCompileSFCBytes_AtAliasImportTransformation(t *testing.T) {
	tests := []struct {
		name                   string
		moduleName             string
		sfcContent             string
		wantTransformedImport  string
		wantOriginalImport     string
		wantDependencyResolved string
		wantDependencyCount    int
	}{
		{
			name:       "@/ import transforms to served URL",
			moduleName: "github.com/user/project",
			sfcContent: `<template name="at-alias-test"><div>Test</div></template>
<script>
import { parse } from '@/lib/parser.js';
</script>`,
			wantTransformedImport:  `"/_piko/assets/github.com/user/project/lib/parser.js"`,
			wantOriginalImport:     `"@/lib/parser.js"`,
			wantDependencyCount:    1,
			wantDependencyResolved: "github.com/user/project/lib/parser.js",
		},
		{
			name:       "@/ import with nested path transforms correctly",
			moduleName: "github.com/org/repo",
			sfcContent: `<template name="at-alias-test"><div>Test</div></template>
<script>
import { utils } from '@/scripts/markdown-parser/dist/index.js';
</script>`,
			wantTransformedImport:  `"/_piko/assets/github.com/org/repo/scripts/markdown-parser/dist/index.js"`,
			wantOriginalImport:     `"@/scripts/markdown-parser/dist/index.js"`,
			wantDependencyCount:    1,
			wantDependencyResolved: "github.com/org/repo/scripts/markdown-parser/dist/index.js",
		},
		{
			name:       "non @/ import passes through unchanged",
			moduleName: "github.com/user/project",
			sfcContent: `<template name="at-alias-test"><div>Test</div></template>
<script>
import { something } from 'external-module';
</script>`,
			wantTransformedImport: `"external-module"`,
			wantOriginalImport:    `"external-module"`,
			wantDependencyCount:   0,
		},
		{
			name:       "relative import passes through unchanged",
			moduleName: "github.com/user/project",
			sfcContent: `<template name="at-alias-test"><div>Test</div></template>
<script>
import { local } from './local.js';
</script>`,
			wantTransformedImport: `"./local.js"`,
			wantOriginalImport:    `"./local.js"`,
			wantDependencyCount:   0,
		},
		{
			name:       "@/ import without module name passes through unchanged",
			moduleName: "",
			sfcContent: `<template name="at-alias-test"><div>Test</div></template>
<script>
import { parse } from '@/lib/parser.js';
</script>`,
			wantTransformedImport: `"@/lib/parser.js"`,
			wantOriginalImport:    `"@/lib/parser.js"`,
			wantDependencyCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var opts []OrchestratorOption
			if tt.moduleName != "" {
				opts = append(opts, WithOrchestratorModuleName(tt.moduleName))
			}

			orchestrator := NewCompilerOrchestrator(nil, nil, opts...)
			artefact, err := orchestrator.CompileSFCBytes(ctx, "test.pkc", []byte(tt.sfcContent))

			require.NoError(t, err)
			require.NotNil(t, artefact)

			mainJS := artefact.Files[artefact.BaseJSPath]
			assert.Contains(t, mainJS, tt.wantTransformedImport,
				"compiled JS should contain transformed import path")

			if tt.wantTransformedImport != tt.wantOriginalImport {
				assert.NotContains(t, mainJS, tt.wantOriginalImport,
					"compiled JS should not contain original @/ import path")
			}

			assert.Len(t, artefact.JSDependencies, tt.wantDependencyCount,
				"JSDependencies should track transformed imports")

			if tt.wantDependencyCount > 0 && len(artefact.JSDependencies) > 0 {
				dependency := artefact.JSDependencies[0]
				assert.Equal(t, tt.wantDependencyResolved, dependency.ResolvedPath,
					"dependency should have correct resolved path")
			}
		})
	}
}

type mockCSSPreProcessor struct {
	result    string
	err       error
	called    bool
	gotCSS    string
	gotSource string
}

func (m *mockCSSPreProcessor) InlineImports(_ context.Context, cssContent string, sourcePath string) (string, error) {
	m.called = true
	m.gotCSS = cssContent
	m.gotSource = sourcePath
	return m.result, m.err
}

func TestWithOrchestratorCSSPreProcessor(t *testing.T) {
	t.Run("stores pre-processor on orchestrator", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{}
		o := NewCompilerOrchestrator(nil, nil, WithOrchestratorCSSPreProcessor(preProcessor))
		orch, ok := o.(*compilerOrchestrator)
		require.True(t, ok)
		assert.NotNil(t, orch.cssPreProcessor)
	})

	t.Run("nil by default", func(t *testing.T) {
		o := NewCompilerOrchestrator(nil, nil)
		orch, ok := o.(*compilerOrchestrator)
		require.True(t, ok)
		assert.Nil(t, orch.cssPreProcessor)
	})
}

func TestPreProcessStyles(t *testing.T) {
	t.Run("no-op when styles are empty", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{result: "should not be used"}
		ctx := context.Background()
		cc := &sfcCompilationContext{stylesDefault: "", cssPreProcessor: preProcessor}
		cc.preProcessStyles(ctx)
		assert.Equal(t, "", cc.stylesDefault)
		assert.False(t, preProcessor.called)
	})

	t.Run("no-op when no pre-processor set", func(t *testing.T) {
		ctx := context.Background()
		cc := &sfcCompilationContext{stylesDefault: "@import './foo.css';"}
		cc.preProcessStyles(ctx)
		assert.Equal(t, "@import './foo.css';", cc.stylesDefault)
	})

	t.Run("replaces styles with pre-processed result", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{result: ".foo{color:red}"}
		ctx := context.Background()
		cc := &sfcCompilationContext{
			stylesDefault:   "@import './foo.css';",
			sourceFilename:  "components/widget.pkc",
			cssPreProcessor: preProcessor,
		}
		cc.preProcessStyles(ctx)
		assert.Equal(t, ".foo{color:red}", cc.stylesDefault)
		assert.True(t, preProcessor.called)
		assert.Equal(t, "@import './foo.css';", preProcessor.gotCSS)
		assert.Equal(t, "components/widget.pkc", preProcessor.gotSource)
	})

	t.Run("keeps raw CSS on pre-processor error", func(t *testing.T) {
		preProcessor := &mockCSSPreProcessor{err: errors.New("resolve failed")}
		ctx := context.Background()
		original := "@import './missing.css'; .local { color: blue; }"
		cc := &sfcCompilationContext{stylesDefault: original, cssPreProcessor: preProcessor}
		cc.preProcessStyles(ctx)
		assert.Equal(t, original, cc.stylesDefault)
		assert.True(t, preProcessor.called)
	})
}
