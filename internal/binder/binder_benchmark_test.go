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

package binder

import (
	"context"
	"fmt"
	"net/url"
	"testing"
)

type BenchmarkSimpleForm struct {
	FirstName    string
	LastName     string
	Email        string
	Age          int
	IsSubscribed bool
	Score        float64
}

type BenchmarkComplexForm struct {
	User struct {
		Name  string
		Email string
		ID    int
	}
	Items []struct {
		SKU   string
		Name  string
		Price float64
		Qty   int
	}
}

func BenchmarkBindSimpleForm(b *testing.B) {
	binder := NewASTBinder()
	src := url.Values{
		"FirstName":    {"John"},
		"LastName":     {"Doe"},
		"Email":        {"john.doe@example.com"},
		"Age":          {"42"},
		"IsSubscribed": {"true"},
		"Score":        {"98.6"},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var form BenchmarkSimpleForm

		_ = binder.Bind(context.Background(), &form, src)
	}
}

func BenchmarkBindComplexForm(b *testing.B) {
	binder := NewASTBinder()
	src := url.Values{
		"User.ID":        {"12345"},
		"User.Name":      {"Jane Doe"},
		"User.Email":     {"jane.doe@example.com"},
		"Items[0].SKU":   {"abc-123"},
		"Items[0].Name":  {"Thingamajig"},
		"Items[0].Price": {"19.99"},
		"Items[0].Qty":   {"2"},
		"Items[1].SKU":   {"xyz-789"},
		"Items[1].Name":  {"Doohickey"},
		"Items[1].Price": {"25.49"},
		"Items[1].Qty":   {"1"},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var form BenchmarkComplexForm
		_ = binder.Bind(context.Background(), &form, src)
	}
}

func BenchmarkBindComplexForm_Parallel(b *testing.B) {
	binder := GetBinder()
	src := url.Values{
		"User.ID":        {"12345"},
		"User.Name":      {"Jane Doe"},
		"User.Email":     {"jane.doe@example.com"},
		"Items[0].SKU":   {"abc-123"},
		"Items[0].Name":  {"Thingamajig"},
		"Items[0].Price": {"19.99"},
		"Items[0].Qty":   {"2"},
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var form BenchmarkComplexForm
			_ = binder.Bind(context.Background(), &form, src)
		}
	})
}

func BenchmarkBind_FirstVsSubsequent(b *testing.B) {
	src := url.Values{
		"User.ID":      {"999"},
		"User.Name":    {"Cache Test"},
		"Items[5].SKU": {"cache-test-sku"},
	}

	b.Run("FirstBind_PopulatesCaches", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {

			binder := NewASTBinder()
			var form BenchmarkComplexForm
			_ = binder.Bind(context.Background(), &form, src)
		}
	})

	b.Run("SubsequentBind_HitsCaches", func(b *testing.B) {

		binder := NewASTBinder()

		var form BenchmarkComplexForm
		_ = binder.Bind(context.Background(), &form, src)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var form BenchmarkComplexForm
			_ = binder.Bind(context.Background(), &form, src)
		}
	})
}

func BenchmarkBind_DeeplyNestedAndSparse(b *testing.B) {
	binder := NewASTBinder()

	src := make(url.Values)
	for i := 0; i < 20; i += 2 {
		keyPrefix := fmt.Sprintf("Items[%d]", i)
		src.Set(keyPrefix+".SKU", fmt.Sprintf("SKU-%d", i))
		src.Set(keyPrefix+".Name", fmt.Sprintf("Item %d", i))
		src.Set(keyPrefix+".Price", "10.0")
		src.Set(keyPrefix+".Qty", "1")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var form BenchmarkComplexForm
		_ = binder.Bind(context.Background(), &form, src)
	}
}
