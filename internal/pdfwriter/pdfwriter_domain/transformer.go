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

package pdfwriter_domain

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

// PdfTransformerRegistry holds a set of named PDF transformers.
type PdfTransformerRegistry struct {
	// transformers maps transformer names to their implementations.
	transformers map[string]PdfTransformerPort
}

// NewPdfTransformerRegistry creates a new, empty transformer registry.
//
// Returns *PdfTransformerRegistry which is ready to have transformers added.
func NewPdfTransformerRegistry() *PdfTransformerRegistry {
	return &PdfTransformerRegistry{
		transformers: make(map[string]PdfTransformerPort),
	}
}

// Register adds a new transformer to the registry.
//
// Takes transformer (PdfTransformerPort) which is the transformer to add.
//
// Returns error when transformer is nil, has an empty name, or a transformer
// with the same name is already registered.
func (r *PdfTransformerRegistry) Register(transformer PdfTransformerPort) error {
	if transformer == nil {
		return errTransformerNil
	}
	name := transformer.Name()
	if name == "" {
		return errTransformerNameEmpty
	}
	if _, exists := r.transformers[name]; exists {
		return fmt.Errorf("transformer %q: %w", name, ErrTransformerAlreadyRegistered)
	}
	r.transformers[name] = transformer
	return nil
}

// Get retrieves a transformer by its registered name.
//
// Takes name (string) which is the registered transformer name to look up.
//
// Returns PdfTransformerPort which is the resolved transformer.
// Returns error when no transformer with the given name is registered.
func (r *PdfTransformerRegistry) Get(name string) (PdfTransformerPort, error) {
	transformer, ok := r.transformers[name]
	if !ok {
		return nil, fmt.Errorf("transformer %q: %w", name, ErrTransformerNotFound)
	}
	return transformer, nil
}

// Has checks if a transformer with the given name is registered.
//
// Takes name (string) which is the transformer name to look up.
//
// Returns bool which is true if the transformer exists, false otherwise.
func (r *PdfTransformerRegistry) Has(name string) bool {
	_, ok := r.transformers[name]
	return ok
}

// GetNames returns a sorted list of all registered transformer names.
//
// Returns []string which contains the names in alphabetical order.
func (r *PdfTransformerRegistry) GetNames() []string {
	names := make([]string, 0, len(r.transformers))
	for name := range r.transformers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// PdfTransformerChain represents an ordered sequence of transformers to be
// applied to PDF bytes.
type PdfTransformerChain struct {
	// config holds transformer options keyed by transformer name.
	config *pdfwriter_dto.TransformConfig

	// transformers holds the ordered list of transformers to apply.
	transformers []PdfTransformerPort
}

// NewPdfTransformerChain creates and sorts a new transformer chain based on
// a configuration. The chain is sorted by ascending priority and validated
// for constraint violations before being returned.
//
// When config is nil, returns an empty but valid chain.
//
// Takes registry (*PdfTransformerRegistry) which provides available
// transformers.
// Takes config (*pdfwriter_dto.TransformConfig) which specifies which
// transformers to enable.
//
// Returns *PdfTransformerChain which contains the sorted transformers ready
// for use.
// Returns error when the registry is nil, a transformer cannot be resolved,
// or the chain violates ordering constraints.
func NewPdfTransformerChain(registry *PdfTransformerRegistry, config *pdfwriter_dto.TransformConfig) (*PdfTransformerChain, error) {
	if registry == nil {
		return nil, errRegistryNil
	}
	if config == nil {
		return &PdfTransformerChain{}, nil
	}

	chain := &PdfTransformerChain{
		transformers: make([]PdfTransformerPort, 0, len(config.EnabledTransformers)),
		config:       config,
	}

	for _, name := range config.EnabledTransformers {
		transformer, err := registry.Get(name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve transformer '%s': %w", name, err)
		}
		chain.transformers = append(chain.transformers, transformer)
	}

	slices.SortFunc(chain.transformers, func(a, b PdfTransformerPort) int {
		return cmp.Compare(a.Priority(), b.Priority())
	})

	if err := ValidateChain(chain.transformers); err != nil {
		return nil, fmt.Errorf("chain validation failed: %w", err)
	}

	return chain, nil
}

// IsEmpty returns true if the chain contains no transformers.
//
// Returns bool which indicates whether the chain is empty.
func (c *PdfTransformerChain) IsEmpty() bool {
	return len(c.transformers) == 0
}

// Transform applies all transformers in ascending priority order to the
// provided PDF bytes.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pdf ([]byte) which is the input PDF document.
//
// Returns []byte which is the fully transformed PDF.
// Returns error when any transformer in the chain fails.
func (c *PdfTransformerChain) Transform(ctx context.Context, pdf []byte) ([]byte, error) {
	if c.IsEmpty() {
		return pdf, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	current := pdf
	for _, transformer := range c.transformers {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var options any
		if c.config != nil && c.config.TransformerOptions != nil {
			options = c.config.TransformerOptions[transformer.Name()]
		}

		start := time.Now()
		transformed, err := transformer.Transform(ctx, current, options)
		elapsed := float64(time.Since(start).Milliseconds())

		attrs := metric.WithAttributes(attribute.String("transformer", transformer.Name()))
		transformsTotal.Add(ctx, 1, attrs)
		transformDuration.Record(ctx, elapsed, attrs)

		if err != nil {
			transformErrorsTotal.Add(ctx, 1, attrs)
			return nil, fmt.Errorf("transformer '%s' failed: %w", transformer.Name(), err)
		}

		current = transformed
		l.Trace("Applied PDF transformer",
			logger_domain.String(logFieldTransformer, transformer.Name()),
			logger_domain.Int(logFieldPriority, transformer.Priority()))
	}
	return current, nil
}
