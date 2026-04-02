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

	"github.com/stretchr/testify/assert"
	"piko.sh/piko"

	categories "testcase_14_nested_loops/pages/pages_categories_40fbe79b"
)

func TestNestedLoops_OuterLoop(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST,
		piko.WithPageID("pages/categories"),
	)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".category").Count(3)
	view.QueryAST(".category-name").Count(3)
}

func TestNestedLoops_InnerLoop(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".product").Count(6)
	view.QueryAST(".product-name").Count(6)
	view.QueryAST(".product-price").Count(6)
}

func TestNestedLoops_ProductLists(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".product-list").Count(3)
}

func TestNestedLoops_CategoryContent(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	categoryNames := view.QueryAST(".category-name")
	categoryNames.Index(0).HasText("Electronics")
	categoryNames.Index(1).HasText("Books")
	categoryNames.Index(2).HasText("Clothing")
}

func TestNestedLoops_ProductContent(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	productNames := view.QueryAST(".product-name")
	productNames.Index(0).HasText("Laptop")
	productNames.Index(1).HasText("Phone")
	productNames.Index(2).HasText("Go Programming")
	productNames.Index(3).HasText("Clean Code")
	productNames.Index(4).HasText("Design Patterns")
	productNames.Index(5).HasText("T-Shirt")

	productPrices := view.QueryAST(".product-price")
	productPrices.Index(0).HasText("$999")
	productPrices.Index(5).HasText("$25")
}

func TestNestedLoops_Count(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	assert.Equal(t, 3, view.QueryAST(".category").Len())
	assert.Equal(t, 6, view.QueryAST(".product").Len())
	assert.Equal(t, 6, view.QueryAST(".product-name").Len())
}

func TestNestedLoops_Structure(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".catalogue").Exists()
	view.QueryAST(".catalogue h1").Exists()
	view.QueryAST(".catalogue h1").HasText("Product Catalogue")
	view.QueryAST(".categories").Exists()

	view.QueryAST("section.category").Count(3)

	view.QueryAST(".category h2").Count(3)
	view.QueryAST(".category ul").Count(3)

	view.QueryAST(".product-list li").Count(6)
}

func TestNestedLoops_DescendantSelectors(t *testing.T) {
	tester := piko.NewComponentTester(t, categories.BuildAST)

	request := piko.NewTestRequest("GET", "/categories").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".catalogue .categories .category").Count(3)
	view.QueryAST(".category .product-list .product").Count(6)
	view.QueryAST(".catalogue .product").Count(6)
}
