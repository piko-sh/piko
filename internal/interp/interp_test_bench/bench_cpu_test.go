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

package interp_test_bench

import (
	"context"
	"testing"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

var fibCode10 = `func fib(x int) int {
	if x == 0 { return 0 }
	if x == 1 { return 1 }
	return fib(x-1) + fib(x-2)
}
fib(10)`

var fibCode35 = `func fib(x int) int {
	if x == 0 { return 0 }
	if x == 1 { return 1 }
	return fib(x-1) + fib(x-2)
}
fib(35)`

var closuresCode = `const calls = 90000
var b int
for i := 0; i < calls; i++ {
	func(x int) {
		b += x
	}(i)
}
b`

var iterationsCode = `const size = 400
s := make([]int, size)
for i := 0; i < size; i++ {
	s[i] = i
}
for _, x := range s {
	for j := range s {
		s[j] += x
	}
}
s[0]`

var stringsCode = `func containsRune(s string, target int) bool {
	for i := 0; i < len(s); i++ {
		if int(s[i]) == target {
			return true
		}
	}
	return false
}

const size = 100
var s string
for r := 0; r < size*2; r++ {
	if r%2 == 0 {
		s += string(rune(r))
	}
}
n := 0
for r := 0; r < size*2; r++ {
	if containsRune(s, r) {
		n++
	}
}
n`

func BenchmarkPiko_Fibonacci10(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), fibCode10)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPiko_Fibonacci35(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), fibCode35)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPiko_Closures(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), closuresCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPiko_Iterations(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), iterationsCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPiko_Strings(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), stringsCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var stringsStdlibCode = `import "strings"

const size = 100
var b strings.Builder
for r := 0; r < size*2; r++ {
	if r%2 == 0 {
		b.WriteString(string(rune(r)))
	}
}
s := b.String()
n := 0
for r := 0; r < size*2; r++ {
	if strings.Contains(s, string(rune(r))) {
		n++
	}
}
n`

func BenchmarkPiko_StringsStdlib(b *testing.B) {
	symbols := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	service := interp_domain.NewService()
	service.UseSymbols(symbols)
	compiledFunction, err := service.Compile(context.Background(), stringsStdlibCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var stringsScriggoCode = `import "strings"

const size = 100
var s string
for r := 0; r < size*2; r++ {
	if r%2 == 0 {
		s += string(rune(r))
	}
}
n := 0
for r := 0; r < size*2; r++ {
	if strings.ContainsRune(s, rune(r)) {
		n++
	}
}
n`

func BenchmarkPiko_StringsScriggo(b *testing.B) {
	symbols := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	service := interp_domain.NewService()
	service.UseSymbols(symbols)
	compiledFunction, err := service.Compile(context.Background(), stringsScriggoCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var realisticCode = `func itoa(n int) string {
	if n == 0 { return "0" }
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	s := ""
	for n > 0 {
		s = string(rune(48 + n % 10)) + s
		n = n / 10
	}
	if neg { s = "-" + s }
	return s
}

func discount(qty int) int {
	if qty >= 50 { return 20 }
	if qty >= 20 { return 10 }
	if qty >= 5 { return 5 }
	return 0
}

const numItems = 200

prices := make([]int, numItems)
quantities := make([]int, numItems)
for i := 0; i < numItems; i++ {
	prices[i] = 100 + (i * 37 % 500)
	quantities[i] = 1 + (i * 13 % 100)
}

totalRevenue := 0
totalDiscount := 0
tierCounts := make([]int, 4)
minPrice := prices[0]
maxPrice := prices[0]

for i := 0; i < numItems; i++ {
	subtotal := prices[i] * quantities[i]
	disc := discount(quantities[i])
	discountAmount := subtotal * disc / 100
	net := subtotal - discountAmount

	totalRevenue += net
	totalDiscount += discountAmount

	if prices[i] < minPrice { minPrice = prices[i] }
	if prices[i] > maxPrice { maxPrice = prices[i] }

	if disc == 0 { tierCounts[0] = tierCounts[0] + 1 }
	if disc == 5 { tierCounts[1] = tierCounts[1] + 1 }
	if disc == 10 { tierCounts[2] = tierCounts[2] + 1 }
	if disc == 20 { tierCounts[3] = tierCounts[3] + 1 }
}

avgPrice := 0
for i := 0; i < numItems; i++ {
	avgPrice += prices[i]
}
avgPrice = avgPrice / numItems

summary := "Items: " + itoa(numItems) + "\n"
summary += "Revenue: " + itoa(totalRevenue) + "\n"
summary += "Discount: " + itoa(totalDiscount) + "\n"
summary += "Min price: " + itoa(minPrice) + "\n"
summary += "Max price: " + itoa(maxPrice) + "\n"
summary += "Average price: " + itoa(avgPrice) + "\n"
summary += "No discount: " + itoa(tierCounts[0]) + "\n"
summary += "5%% tier: " + itoa(tierCounts[1]) + "\n"
summary += "10%% tier: " + itoa(tierCounts[2]) + "\n"
summary += "20%% tier: " + itoa(tierCounts[3]) + "\n"
summary`

func BenchmarkPiko_Realistic(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), realisticCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var mapOpsCode = `const size = 10000
m := make(map[int]int)
for i := 0; i < size; i++ {
	m[i] = i * 3
}
sum := 0
for i := 0; i < size; i++ {
	sum += m[i]
}
sum`

func BenchmarkPiko_MapOps(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), mapOpsCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var structFieldsCode = `type Point struct {
	X int
	Y int
	Z int
}
points := make([]Point, 1000)
for i := 0; i < 1000; i++ {
	points[i] = Point{X: i, Y: i*2, Z: i*3}
}
sum := 0
for i := 0; i < 1000; i++ {
	sum += points[i].X + points[i].Y + points[i].Z
}
sum`

func BenchmarkPiko_StructFields(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), structFieldsCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var methodCallsCode = `type Counter struct {
	V int
}
func (c *Counter) Add(n int) {
	c.V += n
}
func (c Counter) Get() int {
	return c.V
}
var c Counter
for i := 0; i < 100000; i++ {
	c.Add(i)
}
c.Get()`

func BenchmarkPiko_MethodCalls(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), methodCallsCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var sliceAppendCode = `var s []int
for i := 0; i < 50000; i++ {
	s = append(s, i)
}
sum := 0
for _, v := range s {
	sum += v
}
sum`

func BenchmarkPiko_SliceAppend(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), sliceAppendCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var higherOrderCode = `func apply(f func(int) int, x int) int {
	return f(x)
}
func double(x int) int {
	return x * 2
}
sum := 0
for i := 0; i < 50000; i++ {
	sum += apply(double, i)
}
sum`

func BenchmarkPiko_HigherOrder(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), higherOrderCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPiko_Fibonacci10_IncludingCompilation(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		service := interp_domain.NewService()
		_, _ = service.Eval(context.Background(), fibCode10)
	}
}

func BenchmarkPiko_Fibonacci35_IncludingCompilation(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		service := interp_domain.NewService()
		_, _ = service.Eval(context.Background(), fibCode35)
	}
}
