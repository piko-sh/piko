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
package main_test

import (
	"context"
	"testing"

	"piko.sh/piko"

	products "testcase_10_attribute_assertions/pages/pages_products_30fa6c9e"
)

func TestAttributeAssertions_HasAttribute(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST,
		piko.WithPageID("pages/products"),
	)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".product-catalogue").HasAttribute("data-version", "1.0")
	view.QueryAST(".product-catalogue").HasAttribute("data-env", "production")
	view.QueryAST(".product-catalogue").HasAttribute("data-feature-flag", "new-layout")

	view.QueryAST(".catalogue-footer").HasAttribute("data-copyright", "2024")

	view.QueryAST(".product-grid").HasAttribute("role", "list")
	view.QueryAST(".product-grid").HasAttribute("aria-label", "Product listing")

	view.QueryAST(".product-card").HasAttribute("role", "listitem")
	view.QueryAST(".product-card").HasAttribute("tabindex", "0")
}

func TestAttributeAssertions_HasAttributeContaining(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".product-catalogue").HasAttributeContaining("data-version", "1")

	view.QueryAST(".product-grid").HasAttributeContaining("aria-label", "Product")
	view.QueryAST(".product-grid").HasAttributeContaining("aria-label", "listing")

	view.QueryAST(".product-catalogue").HasAttributeContaining("data-feature-flag", "new")
	view.QueryAST(".product-catalogue").HasAttributeContaining("data-feature-flag", "layout")
}

func TestAttributeAssertions_HasAttributePresent(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".product-catalogue").HasAttributePresent("data-version")
	view.QueryAST(".product-catalogue").HasAttributePresent("data-env")
	view.QueryAST(".product-catalogue").HasAttributePresent("data-feature-flag")

	view.QueryAST(".product-grid").HasAttributePresent("role")
	view.QueryAST(".product-grid").HasAttributePresent("aria-label")

	view.QueryAST("h1").HasAttributePresent("id")

	view.QueryAST(".catalogue-footer").HasAttributePresent("hidden")
}

func TestAttributeAssertions_HasClass(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("h1").HasClass("title")
	view.QueryAST("h1").HasClass("main-heading")

	view.QueryAST("article").HasClass("product-card")
	view.QueryAST("article").HasClass("item")

	view.QueryAST(".product-name").HasClass("heading-2")

	view.QueryAST(".in-stock").HasClass("stock-badge")
	view.QueryAST(".in-stock").HasClass("available")

	view.QueryAST(".out-of-stock").HasClass("stock-badge")
	view.QueryAST(".out-of-stock").HasClass("unavailable")
}

func TestAttributeAssertions_IdAttribute(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("h1").HasAttribute("id", "page-title")

	view.QueryAST("#page-title").Exists()
	view.QueryAST("#page-title").HasText("Products")
}

func TestAttributeAssertions_StaticDataStatus(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".in-stock").HasAttribute("data-status", "available")
	view.QueryAST(".out-of-stock").HasAttribute("data-status", "unavailable")
}

func TestAttributeAssertions_ChainedAttributeChecks(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".product-catalogue").
		Exists().
		HasAttributePresent("data-version").
		HasAttribute("data-env", "production").
		HasAttributeContaining("data-feature-flag", "layout")

	view.QueryAST("h1").
		HasAttribute("id", "page-title").
		HasClass("title").
		HasClass("main-heading").
		HasText("Products")
}

func TestAttributeAssertions_ARIAAttributes(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	grid := view.QueryAST(".product-grid")
	grid.HasAttribute("role", "list")
	grid.HasAttribute("aria-label", "Product listing")

	view.QueryAST(".product-card").HasAttribute("role", "listitem")

	view.QueryAST(".product-card").HasAttribute("tabindex", "0")
}
