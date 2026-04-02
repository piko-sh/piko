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

//go:build bench

package pml_test_bench

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_adapters"
	"piko.sh/piko/internal/pml/pml_components"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/pml/pml_dto"
)

func setupEngine() (pml_domain.Transformer, error) {
	registry, err := pml_components.RegisterBuiltIns(context.Background())
	if err != nil {
		return nil, err
	}

	mediaQueryCollector := pml_adapters.NewMediaQueryCollector()
	msoConditionalCollector := pml_adapters.NewMSOConditionalCollector()

	engine := pml_domain.NewTransformer(registry, mediaQueryCollector, msoConditionalCollector)
	return engine, nil
}

func loadTemplate(filename string) (string, error) {
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseAndTransform(b *testing.B, engine pml_domain.Transformer, templateContent string) {
	config := pml_dto.DefaultConfig()

	ast, err := ast_domain.Parse(context.Background(), templateContent, "benchmark.pk", nil)
	if err != nil {
		b.Fatalf("Failed to parse template: %v", err)
	}

	_, _, diagnostics := engine.Transform(context.Background(), ast, config)
	if len(diagnostics) > 0 {
		for _, diagnostic := range diagnostics {
			if diagnostic.Severity == pml_domain.SeverityError {
				b.Fatalf("Transformation error: %v", diagnostic)
			}
		}
	}
}

func BenchmarkTransform_SimpleEmail(b *testing.B) {
	engine, err := setupEngine()
	if err != nil {
		b.Fatalf("Failed to setup engine: %v", err)
	}

	templateContent, err := loadTemplate("simple_email.pk")
	if err != nil {
		b.Fatalf("Failed to load template: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		parseAndTransform(b, engine, templateContent)
	}
}

func BenchmarkTransform_MediumEmail(b *testing.B) {
	engine, err := setupEngine()
	if err != nil {
		b.Fatalf("Failed to setup engine: %v", err)
	}

	templateContent, err := loadTemplate("medium_email.pk")
	if err != nil {
		b.Fatalf("Failed to load template: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		parseAndTransform(b, engine, templateContent)
	}
}

func BenchmarkTransform_ComplexEmail(b *testing.B) {
	engine, err := setupEngine()
	if err != nil {
		b.Fatalf("Failed to setup engine: %v", err)
	}

	templateContent, err := loadTemplate("complex_email.pk")
	if err != nil {
		b.Fatalf("Failed to load template: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		parseAndTransform(b, engine, templateContent)
	}
}

func BenchmarkParse_SimpleEmail(b *testing.B) {
	templateContent, err := loadTemplate("simple_email.pk")
	if err != nil {
		b.Fatalf("Failed to load template: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := ast_domain.Parse(context.Background(), templateContent, "benchmark.pk", nil)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_ComplexEmail(b *testing.B) {
	templateContent, err := loadTemplate("complex_email.pk")
	if err != nil {
		b.Fatalf("Failed to load template: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := ast_domain.Parse(context.Background(), templateContent, "benchmark.pk", nil)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkTransformOnly_SimpleEmail(b *testing.B) {
	engine, err := setupEngine()
	if err != nil {
		b.Fatalf("Failed to setup engine: %v", err)
	}

	templateContent, err := loadTemplate("simple_email.pk")
	if err != nil {
		b.Fatalf("Failed to load template: %v", err)
	}

	ast, err := ast_domain.Parse(context.Background(), templateContent, "benchmark.pk", nil)
	if err != nil {
		b.Fatalf("Failed to parse template: %v", err)
	}

	config := pml_dto.DefaultConfig()

	b.ResetTimer()
	for b.Loop() {

		_, _, diagnostics := engine.Transform(context.Background(), ast, config)
		if len(diagnostics) > 0 {
			for _, diagnostic := range diagnostics {
				if diagnostic.Severity == pml_domain.SeverityError {
					b.Fatalf("Transformation error: %v", diagnostic)
				}
			}
		}
	}
}

func BenchmarkTransformOnly_ComplexEmail(b *testing.B) {
	engine, err := setupEngine()
	if err != nil {
		b.Fatalf("Failed to setup engine: %v", err)
	}

	templateContent, err := loadTemplate("complex_email.pk")
	if err != nil {
		b.Fatalf("Failed to load template: %v", err)
	}

	ast, err := ast_domain.Parse(context.Background(), templateContent, "benchmark.pk", nil)
	if err != nil {
		b.Fatalf("Failed to parse template: %v", err)
	}

	config := pml_dto.DefaultConfig()

	b.ResetTimer()
	for b.Loop() {

		_, _, diagnostics := engine.Transform(context.Background(), ast, config)
		if len(diagnostics) > 0 {
			for _, diagnostic := range diagnostics {
				if diagnostic.Severity == pml_domain.SeverityError {
					b.Fatalf("Transformation error: %v", diagnostic)
				}
			}
		}
	}
}

func BenchmarkEngineSetup(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_, err := setupEngine()
		if err != nil {
			b.Fatalf("Failed to setup engine: %v", err)
		}
	}
}
