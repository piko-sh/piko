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

package pdfwriter_domain_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

type mockTransformer struct {
	transformFunction func(ctx context.Context, pdf []byte, options any) ([]byte, error)
	name              string
	ttype             pdfwriter_dto.TransformerType
	priority          int
}

var _ pdfwriter_domain.PdfTransformerPort = (*mockTransformer)(nil)

func (m *mockTransformer) Name() string                        { return m.name }
func (m *mockTransformer) Type() pdfwriter_dto.TransformerType { return m.ttype }
func (m *mockTransformer) Priority() int                       { return m.priority }

func (m *mockTransformer) Transform(ctx context.Context, pdf []byte, options any) ([]byte, error) {
	if m.transformFunction != nil {
		return m.transformFunction(ctx, pdf, options)
	}
	return pdf, nil
}

func uppercaseTransformer(name string, priority int) *mockTransformer {
	return &mockTransformer{
		name:     name,
		ttype:    pdfwriter_dto.TransformerContent,
		priority: priority,
		transformFunction: func(_ context.Context, pdf []byte, _ any) ([]byte, error) {
			return bytes.ToUpper(pdf), nil
		},
	}
}

func prefixTransformer(name string, priority int, prefix string) *mockTransformer {
	return &mockTransformer{
		name:     name,
		ttype:    pdfwriter_dto.TransformerContent,
		priority: priority,
		transformFunction: func(_ context.Context, pdf []byte, _ any) ([]byte, error) {
			return append([]byte(prefix), pdf...), nil
		},
	}
}

func TestPdfTransformerRegistry_Register(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()

	transformer := &mockTransformer{
		name:     "watermark",
		ttype:    pdfwriter_dto.TransformerContent,
		priority: 150,
	}

	err := registry.Register(transformer)
	require.NoError(t, err)
	assert.True(t, registry.Has("watermark"))
}

func TestPdfTransformerRegistry_Register_Nil(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	err := registry.Register(nil)
	assert.Error(t, err)
}

func TestPdfTransformerRegistry_Register_EmptyName(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	err := registry.Register(&mockTransformer{name: ""})
	assert.Error(t, err)
}

func TestPdfTransformerRegistry_Register_Duplicate(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()

	err := registry.Register(&mockTransformer{name: "watermark", priority: 150})
	require.NoError(t, err)

	err = registry.Register(&mockTransformer{name: "watermark", priority: 150})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestPdfTransformerRegistry_Get(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	original := &mockTransformer{name: "flatten", priority: 120}
	require.NoError(t, registry.Register(original))

	got, err := registry.Get("flatten")
	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestPdfTransformerRegistry_Get_NotFound(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	_, err := registry.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPdfTransformerRegistry_Has(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	require.NoError(t, registry.Register(&mockTransformer{name: "linearise", priority: 350}))

	assert.True(t, registry.Has("linearise"))
	assert.False(t, registry.Has("nonexistent"))
}

func TestPdfTransformerRegistry_GetNames(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	require.NoError(t, registry.Register(&mockTransformer{name: "watermark", priority: 150}))
	require.NoError(t, registry.Register(&mockTransformer{name: "flatten", priority: 120}))
	require.NoError(t, registry.Register(&mockTransformer{name: "aes-256", priority: 400}))

	names := registry.GetNames()
	assert.Equal(t, []string{"aes-256", "flatten", "watermark"}, names)
}

func TestPdfTransformerChain_NilConfig(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	chain, err := pdfwriter_domain.NewPdfTransformerChain(registry, nil)
	require.NoError(t, err)
	assert.True(t, chain.IsEmpty())
}

func TestPdfTransformerChain_NilRegistry(t *testing.T) {
	config := &pdfwriter_dto.TransformConfig{}
	_, err := pdfwriter_domain.NewPdfTransformerChain(nil, config)
	assert.Error(t, err)
}

func TestPdfTransformerChain_EmptyTransformers(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	config := &pdfwriter_dto.TransformConfig{
		EnabledTransformers: []string{},
	}
	chain, err := pdfwriter_domain.NewPdfTransformerChain(registry, config)
	require.NoError(t, err)
	assert.True(t, chain.IsEmpty())
}

func TestPdfTransformerChain_UnknownTransformer(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	config := &pdfwriter_dto.TransformConfig{
		EnabledTransformers: []string{"nonexistent"},
	}
	_, err := pdfwriter_domain.NewPdfTransformerChain(registry, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestPdfTransformerChain_PriorityOrder(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()

	require.NoError(t, registry.Register(prefixTransformer("second", 200, "[2]")))
	require.NoError(t, registry.Register(prefixTransformer("first", 100, "[1]")))

	config := &pdfwriter_dto.TransformConfig{
		EnabledTransformers: []string{"second", "first"},
	}

	chain, err := pdfwriter_domain.NewPdfTransformerChain(registry, config)
	require.NoError(t, err)
	assert.False(t, chain.IsEmpty())

	result, err := chain.Transform(context.Background(), []byte("data"))
	require.NoError(t, err)

	assert.Equal(t, "[2][1]data", string(result))
}

func TestPdfTransformerChain_Transform_Passthrough(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	require.NoError(t, registry.Register(&mockTransformer{name: "noop", priority: 100}))

	config := &pdfwriter_dto.TransformConfig{
		EnabledTransformers: []string{"noop"},
	}

	chain, err := pdfwriter_domain.NewPdfTransformerChain(registry, config)
	require.NoError(t, err)

	input := []byte("hello world")
	result, err := chain.Transform(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestPdfTransformerChain_Transform_MultiStep(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()
	require.NoError(t, registry.Register(uppercaseTransformer("upper", 100)))
	require.NoError(t, registry.Register(prefixTransformer("prefix", 200, ">>>")))

	config := &pdfwriter_dto.TransformConfig{
		EnabledTransformers: []string{"upper", "prefix"},
	}

	chain, err := pdfwriter_domain.NewPdfTransformerChain(registry, config)
	require.NoError(t, err)

	result, err := chain.Transform(context.Background(), []byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, ">>>HELLO", string(result))
}

func TestPdfTransformerChain_Transform_Error(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()

	failing := &mockTransformer{
		name:     "failing",
		ttype:    pdfwriter_dto.TransformerContent,
		priority: 100,
		transformFunction: func(_ context.Context, _ []byte, _ any) ([]byte, error) {
			return nil, errors.New("transform failed")
		},
	}
	require.NoError(t, registry.Register(failing))

	config := &pdfwriter_dto.TransformConfig{
		EnabledTransformers: []string{"failing"},
	}

	chain, err := pdfwriter_domain.NewPdfTransformerChain(registry, config)
	require.NoError(t, err)

	_, err = chain.Transform(context.Background(), []byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failing")
}

func TestPdfTransformerChain_Transform_Empty(t *testing.T) {
	chain := &pdfwriter_domain.PdfTransformerChain{}
	input := []byte("unchanged")
	result, err := chain.Transform(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestPdfTransformerChain_Transform_Options(t *testing.T) {
	registry := pdfwriter_domain.NewPdfTransformerRegistry()

	optReader := &mockTransformer{
		name:     "opt-reader",
		ttype:    pdfwriter_dto.TransformerContent,
		priority: 100,
		transformFunction: func(_ context.Context, pdf []byte, options any) ([]byte, error) {
			prefix, ok := options.(string)
			if !ok {
				return pdf, nil
			}
			return append([]byte(prefix), pdf...), nil
		},
	}
	require.NoError(t, registry.Register(optReader))

	config := &pdfwriter_dto.TransformConfig{
		EnabledTransformers: []string{"opt-reader"},
		TransformerOptions: map[string]any{
			"opt-reader": "PREFIX:",
		},
	}

	chain, err := pdfwriter_domain.NewPdfTransformerChain(registry, config)
	require.NoError(t, err)

	result, err := chain.Transform(context.Background(), []byte("data"))
	require.NoError(t, err)
	assert.Equal(t, "PREFIX:data", string(result))
}
