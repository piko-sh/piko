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

	products "testcase_04_loop_iteration/pages/pages_products_30fa6c9e"
)

func TestLoopIteration_Title(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST,
		piko.WithPageID("pages/products"),
	)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("Product Catalogue")
}

func TestLoopIteration_TotalCount(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("p.total").Exists().ContainsText("3")
}

func TestLoopIteration_ProductCount(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("li.product-item").Count(3)
}

func TestLoopIteration_ProductNames(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	names := view.QueryAST(".product-name")
	names.Count(3)

	names.Index(0).HasText("Widget")
	names.Index(1).HasText("Gadget")
	names.Index(2).HasText("Gizmo")
}

func TestLoopIteration_ProductPrices(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	prices := view.QueryAST(".product-price")
	prices.Count(3)

	prices.Index(0).ContainsText("29.99")
	prices.Index(1).ContainsText("49.99")
	prices.Index(2).ContainsText("19.99")
}

func TestLoopIteration_InStockStatus(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".in-stock").Count(2)

	view.QueryAST(".out-of-stock").Count(1)
}

func TestLoopIteration_DataAttributes(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	items := view.QueryAST("li.product-item")
	items.Count(3)

	items.Index(0).HasAttributePresent("data-id")
	items.Index(1).HasAttributePresent("data-id")
	items.Index(2).HasAttributePresent("data-id")
}

func TestLoopIteration_DOMStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, products.BuildAST)

	request := piko.NewTestRequest("GET", "/products").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("div.catalogue").Exists()

	view.QueryAST("h1").Exists().HasText("Products")
	view.QueryAST("ul.product-list").Exists()
}
