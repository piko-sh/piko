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

package render_domain

import (
	"testing"

	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/ast/ast_domain"
)

func createTestPikoImgNode() *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  tagPikoImg,
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "/images/hero.jpg"},
			{Name: "alt", Value: "Hero image"},
			{Name: "sizes", Value: "(max-width: 768px) 100vw, 50vw"},
			{Name: "densities", Value: "1x, 2x, 3x"},
			{Name: "formats", Value: "webp, avif"},
			{Name: "widths", Value: "400, 800, 1200"},
			{Name: "class", Value: "hero-image"},
			{Name: "loading", Value: "lazy"},
		},
	}
}

func BenchmarkExtractPikoImgAttrs(b *testing.B) {
	node := createTestPikoImgNode()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		attrs := extractPikoImgAttrs(node)
		_ = attrs.src
	}
}

func BenchmarkExtractStaticPikoImgAttrs(b *testing.B) {
	node := createTestPikoImgNode()
	var result pikoImgAttrs
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result = pikoImgAttrs{}
		extractStaticPikoImgAttrs(node, &result)
		_ = result.src
	}
}

func BenchmarkExtractPikoImgAttrs_Pooled(b *testing.B) {
	node := createTestPikoImgNode()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		attrs := getPikoImgAttrs()
		extractPikoImgAttrsInto(node, attrs)
		_ = attrs.src
		putPikoImgAttrs(attrs)
	}
}

func BenchmarkIsPikoImgSpecialAttr_Map(b *testing.B) {
	attrs := []string{"profile", "src", "alt", "densities", "class", "sizes", "loading"}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, a := range attrs {
			_ = isPikoImgSpecialAttr(a)
		}
	}
}

func BenchmarkIsPikoImgSpecialAttr_Switch(b *testing.B) {
	attrs := []string{"profile", "src", "alt", "densities", "class", "sizes", "loading"}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, a := range attrs {
			_ = isPikoImgSpecialAttrSwitch(a)
		}
	}
}

func isPikoImgSpecialAttrSwitch(name string) bool {
	switch name {
	case "profile", "densities", "sizes", "formats", "widths", "variant", "cms-media":
		return true
	default:
		return false
	}
}

func BenchmarkAssignPikoImgAttr_Map(b *testing.B) {
	attrs := []struct{ name, value string }{
		{name: "src", value: "/images/test.jpg"},
		{name: "sizes", value: "(max-width: 768px) 100vw"},
		{name: "densities", value: "1x, 2x"},
		{name: "formats", value: "webp, avif"},
		{name: "widths", value: "400, 800"},
		{name: "variant", value: "thumb"},
		{name: "alt", value: "test"},
	}
	var result pikoImgAttrs
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result = pikoImgAttrs{}
		for _, a := range attrs {
			assignPikoImgAttr(a.name, a.value, &result)
		}
		_ = result.src
	}
}

func BenchmarkAssignPikoImgAttr_Switch(b *testing.B) {
	attrs := []struct{ name, value string }{
		{name: "src", value: "/images/test.jpg"},
		{name: "sizes", value: "(max-width: 768px) 100vw"},
		{name: "densities", value: "1x, 2x"},
		{name: "formats", value: "webp, avif"},
		{name: "widths", value: "400, 800"},
		{name: "variant", value: "thumb"},
		{name: "alt", value: "test"},
	}
	var result pikoImgAttrs
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result = pikoImgAttrs{}
		for _, a := range attrs {
			assignPikoImgAttrSwitch(a.name, a.value, &result)
		}
		_ = result.src
	}
}

func assignPikoImgAttrSwitch(name, value string, result *pikoImgAttrs) {
	switch name {
	case attributeSrc:
		result.src = value
	case "sizes":
		result.sizes = value
	case "densities":
		result.densities = value
	case "formats":
		result.formats = value
	case "widths":
		result.widths = value
	case "variant":
		result.variant = value
	case "cms-media":
		result.cmsMedia = true
	}
}

func BenchmarkParseCommaSeparated(b *testing.B) {
	value := "1x, 2x, 3x"
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result := parseCommaSeparated(value)
		_ = result
	}
}

func BenchmarkParseCommaSeparated_Longer(b *testing.B) {
	value := "webp, avif, jpeg, png, gif"
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result := parseCommaSeparated(value)
		_ = result
	}
}

func BenchmarkParseCommaSeparated_Empty(b *testing.B) {
	value := ""
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result := parseCommaSeparated(value)
		_ = result
	}
}

func BenchmarkToAssetProfile(b *testing.B) {
	attrs := pikoImgAttrs{
		src:       "/images/test.jpg",
		sizes:     "(max-width: 768px) 100vw, 50vw",
		densities: "1x, 2x, 3x",
		formats:   "webp, avif",
		widths:    "400, 800, 1200",
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		profile := attrs.toAssetProfile()
		_ = profile
	}
}

func BenchmarkAppendTransformedSrc(b *testing.B) {
	buffer := make([]byte, 0, 256)
	src := "github.com/example/assets/image.png"
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		buffer = buffer[:0]
		buffer = assetpath.AppendTransformed(buffer, src, assetpath.DefaultServePath)
		_ = buffer
	}
}

func BenchmarkAppendTransformedSrc_NoTransform(b *testing.B) {
	buffer := make([]byte, 0, 256)
	src := "https://cdn.example.com/image.png"
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		buffer = buffer[:0]
		buffer = assetpath.AppendTransformed(buffer, src, assetpath.DefaultServePath)
		_ = buffer
	}
}
