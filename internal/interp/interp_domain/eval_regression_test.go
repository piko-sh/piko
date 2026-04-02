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

package interp_domain

import (
	"context"
	"encoding/json"
	"fmt"
	"go/importer"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testArenaNode struct {
	Name     string
	NodeType int
}

type testArena struct {
	count int
}

func (a *testArena) GetNode() *testArenaNode {
	a.count++
	return &testArenaNode{Name: fmt.Sprintf("node%d", a.count)}
}

func (a *testArena) GetSlice(n int) []string {
	result := make([]string, n)
	for i := range n {
		result[i] = fmt.Sprintf("item%d", i)
	}
	return result
}

type testRenderData struct {
	Title string
}

type testRenderMeta struct {
	Hit bool
}

type testCapabilityID string

func (c testCapabilityID) String() string {
	return "cap:" + string(c)
}

type testItem struct {
	ID   testCapabilityID
	Name string
}

type testWriter struct {
	buf strings.Builder
}

func (w *testWriter) AppendEscapeString(s string) {
	w.buf.WriteString(s)
}

func (w *testWriter) Result() string {
	return w.buf.String()
}

func TestShiftWithUintAmount(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(8), "shift_left_uint_amount", `a := 1; b := uint(3); a << b`},
		{int64(4), "shift_right_uint_amount", `a := 16; b := uint(2); a >> b`},
		{int64(16), "shift_left_uint_var", `x := 1; var n uint = 4; x << n`},

		{int64(8), "shift_left_const", `a := 1; a << 3`},
		{int64(4), "shift_right_const", `a := 16; a >> 2`},
	})
}

func TestGoDispatchShiftWithUintAmount(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(8), "shift_left_uint_amount", `a := 1; b := uint(3); a << b`},
		{int64(4), "shift_right_uint_amount", `a := 16; b := uint(2); a >> b`},
		{int64(16), "shift_left_uint_var", `x := 1; var n uint = 4; x << n`},
		{int64(8), "shift_left_const", `a := 1; a << 3`},
		{int64(4), "shift_right_const", `a := 16; a >> 2`},
	})
}

func TestStringSliceCompoundAssign(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{"hello world", "string_slice_concat", `s := []string{"hello"}; s[0] += " world"; s[0]`},
		{"abc", "string_slice_concat_multi", `s := []string{"a", "b"}; s[0] += "bc"; s[0]`},
	})
}

func TestUintSliceCompoundAssign(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{uint64(15), "uint_slice_add", `s := []uint{10}; s[0] += 5; s[0]`},
		{uint64(6), "uint_slice_mul", `s := []uint{3}; s[0] *= 2; s[0]`},
	})
}

func TestIntSliceCompoundAssignRegression(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(15), "int_slice_add", `s := []int{10}; s[0] += 5; s[0]`},
		{float64(4.0), "float_slice_add", `s := []float64{1.5}; s[0] += 2.5; s[0]`},
	})
}

func TestUnexportedIntFieldAccess(t *testing.T) {
	t.Parallel()

	runEvalTable(t, nil, []evalTestCase{
		{int64(5), "exported_field_set_get", `type S struct { X int }; s := S{}; s.X = 5; s.X`},
		{int64(15), "exported_field_compound", `type S struct { X int }; s := S{X: 10}; s.X += 5; s.X`},
	})
}

func TestGoDispatchUnexportedIntFieldAccess(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(5), "exported_field_set_get", `type S struct { X int }; s := S{}; s.X = 5; s.X`},
		{int64(15), "exported_field_compound", `type S struct { X int }; s := S{X: 10}; s.X += 5; s.X`},
	})
}

func TestStructFieldOps(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "field_literal_init", `type S struct { X int }; s := S{X: 42}; s.X`},
		{int64(10), "field_assign", `type S struct { X int }; s := S{}; s.X = 10; s.X`},
		{int64(3), "field_add_assign", `type S struct { X int }; s := S{X: 1}; s.X += 2; s.X`},
		{int64(7), "field_arithmetic", `type S struct { A int; B int }; s := S{A: 3, B: 4}; s.A + s.B`},
		{int64(20), "pointer_field_set", `type S struct { X int }; s := &S{}; s.X = 20; s.X`},
		{"hello", "string_field", `type S struct { Name string }; s := S{Name: "hello"}; s.Name`},
		{true, "bool_field", `type S struct { Active bool }; s := S{Active: true}; s.Active`},
		{float64(3.14), "float_field", `type S struct { Value float64 }; s := S{Value: 3.14}; s.Value`},
	})
}

func TestGoDispatchStructFieldOps(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "field_literal_init", `type S struct { X int }; s := S{X: 42}; s.X`},
		{int64(10), "field_assign", `type S struct { X int }; s := S{}; s.X = 10; s.X`},
		{int64(3), "field_add_assign", `type S struct { X int }; s := S{X: 1}; s.X += 2; s.X`},
		{int64(20), "pointer_field_set", `type S struct { X int }; s := &S{}; s.X = 20; s.X`},
	})
}

func TestSortSliceWithTimeAfter(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"sort": {
			"Slice": reflect.ValueOf(sort.Slice),
		},
		"time": {
			"Now":   reflect.ValueOf(time.Now),
			"Date":  reflect.ValueOf(time.Date),
			"UTC":   reflect.ValueOf(time.UTC),
			"Time":  reflect.ValueOf((*time.Time)(nil)),
			"Hour":  reflect.ValueOf(time.Hour),
			"Since": reflect.ValueOf(time.Since),
		},
	})
	symbols.SynthesiseAll()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "sort.Slice using time.Time.After",
			source: `package main

import "sort"
import "time"

func run() bool {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	type Item struct {
		Name      string
		CreatedAt time.Time
	}

	items := []Item{
		{Name: "first", CreatedAt: t1},
		{Name: "second", CreatedAt: t2},
		{Name: "third", CreatedAt: t3},
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	// After sorting descending by time: second (June), third (March), first (Jan)
	return items[0].Name == "second" && items[1].Name == "third" && items[2].Name == "first"
}

func main() {}
`,
			entrypoint: "run",
			expect:     true,
		},
		{
			name: "sort.Slice using time.Time.Before",
			source: `package main

import "sort"
import "time"

func run() bool {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	type Item struct {
		Name      string
		CreatedAt time.Time
	}

	items := []Item{
		{Name: "first", CreatedAt: t1},
		{Name: "second", CreatedAt: t2},
		{Name: "third", CreatedAt: t3},
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	// After sorting ascending by time: first (Jan), third (March), second (June)
	return items[0].Name == "first" && items[1].Name == "third" && items[2].Name == "second"
}

func main() {}
`,
			entrypoint: "run",
			expect:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			service.UseSymbols(symbols)
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

type nativePageItem struct {
	Name             string
	Published        bool
	CreatedAt        time.Time
	VersionCreatedAt time.Time
}

type nativePageWithVersion struct {
	ID                        [16]byte
	WebsiteID                 [16]byte
	BlueprintID               [16]byte
	CreatedAt                 time.Time
	VersionID                 [16]byte
	VersionEnvironmentID      [16]byte
	VersionBlueprintVersionID [16]byte
	VersionAccountID          [16]byte
	VersionStatus             string
	VersionReason             json.RawMessage
	VersionData               json.RawMessage
	VersionPublished          bool
	VersionCreatedAt          time.Time
}

func TestSortSliceWithNativeStructTimeAfter(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	makeItems := func() []*nativePageItem {
		return []*nativePageItem{
			{Name: "first", Published: true, CreatedAt: t1, VersionCreatedAt: t1},
			{Name: "second", Published: false, CreatedAt: t2, VersionCreatedAt: t2},
			{Name: "third", Published: true, CreatedAt: t3, VersionCreatedAt: t3},
		}
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"sort": {
			"Slice": reflect.ValueOf(sort.Slice),
		},
		"time": {
			"Now":      reflect.ValueOf(time.Now),
			"Date":     reflect.ValueOf(time.Date),
			"UTC":      reflect.ValueOf(time.UTC),
			"Time":     reflect.ValueOf((*time.Time)(nil)),
			"Duration": reflect.ValueOf((*time.Duration)(nil)),
			"Hour":     reflect.ValueOf(time.Hour),
			"Since":    reflect.ValueOf(time.Since),
		},
		"mypkg": {
			"MakeItems": reflect.ValueOf(makeItems),
		},
	})
	symbols.SynthesiseAll()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "sort native struct slice by CreatedAt.After",
			source: `package main

import "sort"
import "mypkg"

func run() string {
	items := mypkg.MakeItems()
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items[0].Name + "," + items[1].Name + "," + items[2].Name
}

func main() {}
`,
			entrypoint: "run",
			expect:     "second,third,first",
		},
		{
			name: "sort native struct slice by VersionCreatedAt.Before",
			source: `package main

import "sort"
import "mypkg"

func run() string {
	items := mypkg.MakeItems()
	sort.Slice(items, func(i, j int) bool {
		return items[i].VersionCreatedAt.Before(items[j].VersionCreatedAt)
	})
	return items[0].Name + "," + items[1].Name + "," + items[2].Name
}

func main() {}
`,
			entrypoint: "run",
			expect:     "first,third,second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			service.UseSymbols(symbols)
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSortSliceWithRealTypesPackage(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	makeItems := func() []*nativePageItem {
		return []*nativePageItem{
			{Name: "first", Published: true, CreatedAt: t1, VersionCreatedAt: t1},
			{Name: "second", Published: false, CreatedAt: t2, VersionCreatedAt: t2},
			{Name: "third", Published: true, CreatedAt: t3, VersionCreatedAt: t3},
		}
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"sort": {
			"Slice": reflect.ValueOf(sort.Slice),
		},
		"time": {
			"Now":      reflect.ValueOf(time.Now),
			"Date":     reflect.ValueOf(time.Date),
			"UTC":      reflect.ValueOf(time.UTC),
			"Time":     reflect.ValueOf((*time.Time)(nil)),
			"Duration": reflect.ValueOf((*time.Duration)(nil)),
			"Hour":     reflect.ValueOf(time.Hour),
			"Since":    reflect.ValueOf(time.Since),
		},
		"mypkg": {
			"MakeItems": reflect.ValueOf(makeItems),
		},
	})

	imp := importer.Default()
	realTimePkg, err := imp.Import("time")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("time", realTimePkg)

	realSortPkg, err := imp.Import("sort")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("sort", realSortPkg)

	symbols.SynthesiseAll()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "sort native struct slice by CreatedAt.After with real types",
			source: `package main

import "sort"
import "mypkg"

func run() string {
	items := mypkg.MakeItems()
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items[0].Name + "," + items[1].Name + "," + items[2].Name
}

func main() {}
`,
			entrypoint: "run",
			expect:     "second,third,first",
		},
		{
			name: "sort native struct slice by VersionCreatedAt.Before with real types",
			source: `package main

import "sort"
import "mypkg"

func run() string {
	items := mypkg.MakeItems()
	sort.Slice(items, func(i, j int) bool {
		return items[i].VersionCreatedAt.Before(items[j].VersionCreatedAt)
	})
	return items[0].Name + "," + items[1].Name + "," + items[2].Name
}

func main() {}
`,
			entrypoint: "run",
			expect:     "first,third,second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			service.UseSymbols(symbols)
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestNativeStructFieldWithRealTypesPackage(t *testing.T) {
	t.Parallel()

	type nativeItem struct {
		Name   string
		Schema json.RawMessage
	}

	makeItem := func() *nativeItem {
		return &nativeItem{
			Name:   "test",
			Schema: json.RawMessage(`{"key": "value"}`),
		}
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"encoding/json": {
			"RawMessage": reflect.ValueOf((*json.RawMessage)(nil)),
			"Marshal":    reflect.ValueOf(json.Marshal),
			"Unmarshal":  reflect.ValueOf(json.Unmarshal),
		},
		"mypkg": {
			"MakeItem": reflect.ValueOf(makeItem),
		},
	})

	imp := importer.Default()
	realJSONPkg, err := imp.Import("encoding/json")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("encoding/json", realJSONPkg)
	symbols.SynthesiseAll()

	source := `package main

import "encoding/json"
import "mypkg"

func process(data json.RawMessage) string {
	return string(data)
}

func run() string {
	item := mypkg.MakeItem()
	return process(item.Schema)
}

func main() {}
`

	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, `{"key": "value"}`, result)
}

func TestSortSliceProductionLayout(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	makeItems := func() []*nativePageWithVersion {
		return []*nativePageWithVersion{
			{VersionStatus: "first", VersionPublished: true, CreatedAt: t1, VersionCreatedAt: t1},
			{VersionStatus: "second", VersionPublished: false, CreatedAt: t2, VersionCreatedAt: t2},
			{VersionStatus: "third", VersionPublished: true, CreatedAt: t3, VersionCreatedAt: t3},
		}
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"sort": {
			"Slice": reflect.ValueOf(sort.Slice),
		},
		"time": {
			"Now":      reflect.ValueOf(time.Now),
			"Date":     reflect.ValueOf(time.Date),
			"UTC":      reflect.ValueOf(time.UTC),
			"Time":     reflect.ValueOf((*time.Time)(nil)),
			"Duration": reflect.ValueOf((*time.Duration)(nil)),
			"Hour":     reflect.ValueOf(time.Hour),
			"Since":    reflect.ValueOf(time.Since),
		},
		"encoding/json": {
			"RawMessage": reflect.ValueOf((*json.RawMessage)(nil)),
			"Marshal":    reflect.ValueOf(json.Marshal),
			"Unmarshal":  reflect.ValueOf(json.Unmarshal),
		},
		"mypkg": {
			"MakeItems": reflect.ValueOf(makeItems),
		},
	})

	imp := importer.Default()
	realTimePkg, err := imp.Import("time")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("time", realTimePkg)

	realSortPkg, err := imp.Import("sort")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("sort", realSortPkg)

	realJSONPkg, err := imp.Import("encoding/json")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("encoding/json", realJSONPkg)

	symbols.SynthesiseAll()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "sort by VersionCreatedAt.After (field index 12)",
			source: `package main

import "sort"
import "mypkg"

func run() string {
	items := mypkg.MakeItems()
	sort.Slice(items, func(i, j int) bool {
		return items[i].VersionCreatedAt.After(items[j].VersionCreatedAt)
	})
	return items[0].VersionStatus + "," + items[1].VersionStatus + "," + items[2].VersionStatus
}

func main() {}
`,
			entrypoint: "run",
			expect:     "second,third,first",
		},
		{
			name: "sort by CreatedAt.Before (field index 3)",
			source: `package main

import "sort"
import "mypkg"

func run() string {
	items := mypkg.MakeItems()
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items[0].VersionStatus + "," + items[1].VersionStatus + "," + items[2].VersionStatus
}

func main() {}
`,
			entrypoint: "run",
			expect:     "first,third,second",
		},
		{
			name: "switch with multiple sort.Slice closures (production pattern)",
			source: `package main

import "sort"
import "mypkg"

func run() string {
	items := mypkg.MakeItems()
	sortValue := "updated"

	switch sortValue {
	case "created":
		sort.Slice(items, func(i, j int) bool {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			return items[i].VersionCreatedAt.After(items[j].VersionCreatedAt)
		})
	}
	return items[0].VersionStatus + "," + items[1].VersionStatus + "," + items[2].VersionStatus
}

func main() {}
`,
			entrypoint: "run",
			expect:     "second,third,first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			service.UseSymbols(symbols)
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestMethodCallSameNameAsPackageFunction(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"time": {
			"Time":  reflect.ValueOf((*time.Time)(nil)),
			"After": reflect.ValueOf(time.After),
			"Now":   reflect.ValueOf(time.Now),
			"Date":  reflect.ValueOf(time.Date),
		},
	})

	imp := importer.Default()
	realTimePkg, err := imp.Import("time")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("time", realTimePkg)
	symbols.SynthesiseAll()

	source := `
package main

import "time"

func run() bool {
	a := time.Date(2025, 1, 2, 0, 0, 0, 0, time.Now().Location())
	b := time.Date(2025, 1, 1, 0, 0, 0, 0, time.Now().Location())
	return a.After(b)
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, true, result)
}

func TestStructLiteralSliceFieldIndexRegression(t *testing.T) {
	t.Parallel()

	source := `
package main

type TableCell struct {
	Text         string
	Emphasis     bool
	IsBadge      bool
	BadgeVariant string
}

type TableRow struct {
	ID           string
	BlueprintID  string
	BlueprintURL string
	DetailURL    string
	Title        string
	TableCells   []TableCell
}

type Response struct {
	EnvironmentID string
	Filter        *string
	FilterValue   string
	SortValue     string
	StatusValue   string
	IsFiltering   bool
	HasResults    bool
	TotalPages    int
	CurrentIndex  int
	ItemsPerPage  int
	HeaderCells   []TableCell
	TableRows     []TableRow
	BasePath      string
}

func run() string {
	headerCells := []TableCell{{Text: "Title"}, {Text: "Blueprint"}, {Text: "Last updated"}, {Text: "Status"}}
	rows := []TableRow{
		{ID: "p1", BlueprintID: "b1", BlueprintURL: "/b1", DetailURL: "/p1", Title: "Page One", TableCells: []TableCell{{Text: "Page One", Emphasis: true}, {Text: "Blog"}, {Text: "1 Jan 2025"}, {Text: "draft", IsBadge: true, BadgeVariant: "warning"}}},
	}
	filterValue := ""
	sortValue := "title"
	statusValue := ""
	basePath := "/environments/env1/pages"

	resp := Response{
		EnvironmentID: "env1",
		Filter:        nil,
		FilterValue:   filterValue,
		SortValue:     sortValue,
		StatusValue:   statusValue,
		IsFiltering:   false,
		HasResults:    len(rows) > 0,
		TotalPages:    1,
		CurrentIndex:  0,
		ItemsPerPage:  10,
		HeaderCells:   headerCells,
		TableRows:     rows,
		BasePath:      basePath,
	}

	// Verify fields are assigned to the correct indices.
	if len(resp.HeaderCells) != 4 {
		return "wrong HeaderCells length"
	}
	if resp.HeaderCells[0].Text != "Title" {
		return "wrong HeaderCells[0].Text"
	}
	if len(resp.TableRows) != 1 {
		return "wrong TableRows length"
	}
	if resp.TableRows[0].ID != "p1" {
		return "wrong TableRows[0].ID"
	}
	if resp.TableRows[0].TableCells[0].Text != "Page One" {
		return "wrong TableRows[0].TableCells[0].Text"
	}
	if resp.BasePath != "/environments/env1/pages" {
		return "wrong BasePath"
	}
	return "ok"
}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestStructLiteralSliceFieldIndexWithLoopRegression(t *testing.T) {
	t.Parallel()

	source := `
package main

type TableCell struct {
	Text         string
	Emphasis     bool
	IsBadge      bool
	BadgeVariant string
}

type TableRow struct {
	ID           string
	BlueprintID  string
	BlueprintURL string
	DetailURL    string
	Title        string
	TableCells   []TableCell
}

type Response struct {
	EnvironmentID string
	Filter        *string
	FilterValue   string
	SortValue     string
	StatusValue   string
	IsFiltering   bool
	HasResults    bool
	TotalPages    int
	CurrentIndex  int
	ItemsPerPage  int
	HeaderCells   []TableCell
	TableRows     []TableRow
	BasePath      string
}

func buildResponse() Response {
	titles := []string{"Page One", "Page Two", "Page Three"}
	headerCells := []TableCell{{Text: "Title"}, {Text: "Blueprint"}, {Text: "Last updated"}, {Text: "Status"}}
	rows := make([]TableRow, 0, len(titles))
	for i, title := range titles {
		status := "draft"
		badgeVariant := "warning"
		if i == 0 {
			status = "published"
			badgeVariant = "success"
		}
		rows = append(rows, TableRow{
			ID:           title,
			BlueprintID:  "bp1",
			BlueprintURL: "/bp1",
			DetailURL:    "/p/" + title,
			Title:        title,
			TableCells: []TableCell{
				{Text: title, Emphasis: true},
				{Text: "Blog"},
				{Text: "1 Jan 2025"},
				{Text: status, IsBadge: true, BadgeVariant: badgeVariant},
			},
		})
	}
	filterValue := ""
	sortValue := "title"
	statusValue := ""
	basePath := "/environments/env1/pages"

	return Response{EnvironmentID: "env1", Filter: nil, FilterValue: filterValue, SortValue: sortValue, StatusValue: statusValue, IsFiltering: false, HasResults: len(rows) > 0, TotalPages: 1, CurrentIndex: 0, ItemsPerPage: 10, HeaderCells: headerCells, TableRows: rows, BasePath: basePath}
}

func run() string {
	resp := buildResponse()
	if len(resp.HeaderCells) != 4 {
		return "wrong HeaderCells length"
	}
	if resp.HeaderCells[0].Text != "Title" {
		return "wrong HeaderCells[0].Text"
	}
	if len(resp.TableRows) != 3 {
		return "wrong TableRows length"
	}
	if resp.TableRows[0].ID != "Page One" {
		return "wrong TableRows[0].ID"
	}
	if resp.TableRows[2].Title != "Page Three" {
		return "wrong TableRows[2].Title"
	}
	if resp.TableRows[0].TableCells[0].Text != "Page One" {
		return "wrong nested TableCells"
	}
	if resp.BasePath != "/environments/env1/pages" {
		return "wrong BasePath"
	}
	return "ok"
}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestStructLiteralSliceFieldIndexMultiReturn(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"fmt": {
			"Sprintf": reflect.ValueOf(fmt.Sprintf),
		},
		"sort": {
			"Slice": reflect.ValueOf(sort.Slice),
		},
		"strings": {
			"Contains": reflect.ValueOf(strings.Contains),
		},
	})
	imp := importer.Default()
	fmtPkg, err := imp.Import("fmt")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("fmt", fmtPkg)
	sortPkg, err := imp.Import("sort")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("sort", sortPkg)
	stringsPkg, err := imp.Import("strings")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("strings", stringsPkg)
	symbols.SynthesiseAll()

	source := `
package main

import (
	"fmt"
	"sort"
	"strings"
)

type TableCell struct {
	Text         string
	Emphasis     bool
	IsBadge      bool
	BadgeVariant string
}

type TableRow struct {
	ID           string
	BlueprintID  string
	BlueprintURL string
	DetailURL    string
	Title        string
	TableCells   []TableCell
}

type Response struct {
	EnvironmentID string
	Filter        *string
	FilterValue   string
	SortValue     string
	StatusValue   string
	IsFiltering   bool
	HasResults    bool
	TotalPages    int
	CurrentIndex  int
	ItemsPerPage  int
	HeaderCells   []TableCell
	TableRows     []TableRow
	BasePath      string
}

type Metadata struct {
	Status int
}

func render(envID string, filter *string, sortParam string, statusParam string) (Response, Metadata, error) {
	titles := []string{"Page One", "Page Two", "Page Three", "Page Four", "Page Five"}
	blueprintNames := map[string]string{
		"bp1": "Blog",
		"bp2": "Landing",
	}
	blueprintIDs := []string{"bp1", "bp2", "bp1", "bp1", "bp2"}

	// Sort titles to simulate sort.Slice usage (register pressure).
	sort.Slice(titles, func(i, j int) bool {
		return titles[i] < titles[j]
	})

	allPages := titles
	totalItems := len(allPages)
	itemsPerPage := 10
	currentIndex := 0
	totalPages := (totalItems + itemsPerPage - 1) / itemsPerPage

	startIdx := currentIndex * itemsPerPage
	endIdx := startIdx + itemsPerPage
	if endIdx > totalItems {
		endIdx = totalItems
	}
	pages := allPages
	if startIdx < totalItems {
		pages = allPages[startIdx:endIdx]
	} else {
		pages = nil
	}

	basePath := fmt.Sprintf("/environments/%s/pages", envID)
	headerCells := []TableCell{{Text: "Title"}, {Text: "Blueprint"}, {Text: "Last updated"}, {Text: "Status"}}
	rows := make([]TableRow, 0, len(pages))

	for i, title := range pages {
		status := "draft"
		badgeVariant := "warning"
		if i == 0 {
			status = "published"
			badgeVariant = "success"
		}
		bpID := blueprintIDs[i]
		bpName := blueprintNames[bpID]
		if bpName == "" {
			bpName = "Unknown"
		}
		detailURL := fmt.Sprintf("/environments/%s/pages/%s/", envID, title)
		blueprintURL := fmt.Sprintf("/environments/%s/blueprints/%s/pages", envID, bpID)

		rows = append(rows, TableRow{
			ID:           title,
			BlueprintID:  bpID,
			BlueprintURL: blueprintURL,
			DetailURL:    detailURL,
			Title:        title,
			TableCells: []TableCell{
				{Text: title, Emphasis: true},
				{Text: bpName},
				{Text: "1 Jan 2025"},
				{Text: status, IsBadge: true, BadgeVariant: badgeVariant},
			},
		})
	}

	filterValue := ""
	if filter != nil {
		filterValue = *filter
	}
	sortValue := sortParam
	statusValue := statusParam

	// Intermediate variables to add register pressure.
	searchTerm := ""
	if filter != nil {
		searchTerm = *filter
	}
	_ = searchTerm

	hasResults := len(rows) > 0
	isFiltering := filter != nil || statusValue != ""
	_ = strings.Contains(basePath, envID)

	return Response{EnvironmentID: envID, Filter: filter, FilterValue: filterValue, SortValue: sortValue, StatusValue: statusValue, IsFiltering: isFiltering, HasResults: hasResults, TotalPages: totalPages, CurrentIndex: currentIndex, ItemsPerPage: itemsPerPage, HeaderCells: headerCells, TableRows: rows, BasePath: basePath}, Metadata{}, nil
}

func run() string {
	resp, meta, err := render("env1", nil, "title", "")
	_ = meta
	if err != nil {
		return "unexpected error"
	}
	if resp.EnvironmentID != "env1" {
		return "wrong EnvironmentID"
	}
	if resp.Filter != nil {
		return "wrong Filter"
	}
	if len(resp.HeaderCells) != 4 {
		return fmt.Sprintf("wrong HeaderCells length: %d", len(resp.HeaderCells))
	}
	if resp.HeaderCells[0].Text != "Title" {
		return "wrong HeaderCells[0].Text"
	}
	if len(resp.TableRows) != 5 {
		return fmt.Sprintf("wrong TableRows length: %d", len(resp.TableRows))
	}
	if resp.TableRows[0].TableCells[0].Emphasis != true {
		return "wrong TableRows[0].TableCells[0].Emphasis"
	}
	if resp.BasePath != "/environments/env1/pages" {
		return "wrong BasePath"
	}
	if resp.TotalPages != 1 {
		return "wrong TotalPages"
	}
	return "ok"
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestStructFieldIndexCrossFunction(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"fmt": {
			"Sprintf": reflect.ValueOf(fmt.Sprintf),
		},
	})
	imp := importer.Default()
	fmtPkg, err := imp.Import("fmt")
	require.NoError(t, err)
	symbols.RegisterTypesPackage("fmt", fmtPkg)
	symbols.SynthesiseAll()

	source := `
package main

import "fmt"

type TableCell struct {
	Text         string
	Emphasis     bool
	IsBadge      bool
	BadgeVariant string
}

type TableRow struct {
	ID           string
	BlueprintID  string
	BlueprintURL string
	DetailURL    string
	Title        string
	TableCells   []TableCell
}

type Response struct {
	EnvironmentID string
	Filter        *string
	FilterValue   string
	SortValue     string
	StatusValue   string
	IsFiltering   bool
	HasResults    bool
	TotalPages    int
	CurrentIndex  int
	ItemsPerPage  int
	HeaderCells   []TableCell
	TableRows     []TableRow
	BasePath      string
}

type Metadata struct {
	Status int
}

func buildData(envID string) (Response, Metadata, error) {
	headerCells := []TableCell{{Text: "Title"}, {Text: "Blueprint"}, {Text: "Last updated"}, {Text: "Status"}}
	rows := make([]TableRow, 0, 3)
	for i := 0; i < 3; i++ {
		title := fmt.Sprintf("Page %d", i)
		rows = append(rows, TableRow{
			ID:           fmt.Sprintf("id%d", i),
			BlueprintID:  "bp1",
			BlueprintURL: "/bp1",
			DetailURL:    "/p/" + title,
			Title:        title,
			TableCells: []TableCell{
				{Text: title, Emphasis: true},
				{Text: "Blog"},
				{Text: "1 Jan 2025"},
				{Text: "draft", IsBadge: true, BadgeVariant: "warning"},
			},
		})
	}
	return Response{EnvironmentID: envID, Filter: nil, FilterValue: "", SortValue: "title", StatusValue: "", IsFiltering: false, HasResults: len(rows) > 0, TotalPages: 1, CurrentIndex: 0, ItemsPerPage: 10, HeaderCells: headerCells, TableRows: rows, BasePath: "/env/pages"}, Metadata{}, nil
}

func buildAST(envID string) string {
	pageData, _, _ := buildData(envID)

	// Read HeaderCells - like loopIter_3 := pageData.HeaderCells
	headerIter := pageData.HeaderCells
	result := ""
	for _, cell := range headerIter {
		result += cell.Text + ","
	}

	// Read TableRows - like loopIter_4 := pageData.TableRows
	rowIter := pageData.TableRows
	for _, row := range rowIter {
		result += row.Title + ":"
		for _, cell := range row.TableCells {
			result += cell.Text + ";"
		}
		result += "|"
	}

	return result
}

func run() string {
	got := buildAST("env1")
	if len(got) == 0 {
		return "empty result"
	}
	return "ok"
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestDoubleRecycleRedeclaredVar(t *testing.T) {
	t.Parallel()

	type Cell struct{ Text string }
	type Row struct {
		ID    string
		Cells []Cell
	}
	type Result struct {
		HeaderCells []Cell
		TableRows   []Row
	}

	type RecycleCell struct{ Text string }
	type RecycleRow struct {
		ID    string
		Cells []RecycleCell
	}
	type RecycleResult struct {
		HeaderCells []RecycleCell
		TableRows   []RecycleRow
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"mypkg": {
			"Cell":   reflect.ValueOf((*RecycleCell)(nil)),
			"Row":    reflect.ValueOf((*RecycleRow)(nil)),
			"Result": reflect.ValueOf((*RecycleResult)(nil)),

			"Load1": reflect.ValueOf(func() ([]string, error) {
				return []string{"a"}, nil
			}),
			"Load2": reflect.ValueOf(func() ([]string, error) {
				return []string{"b"}, nil
			}),
		},
	})
	symbols.SynthesiseAll()

	source := `
package main

import "mypkg"

func run() string {
	// data1 declared as general register (slice), err as general (error)
	data1, err := mypkg.Load1()
	if err != nil {
		return "err1"
	}

	// data2 declared, err re-declared - same register, tracked twice
	data2, err := mypkg.Load2()
	if err != nil {
		return "err2"
	}

	// err is now dead (last use was stmt above).
	// Its register is recycled TWICE from the two activeDeclaration entries.
	// data1 and data2 are still alive (used in the return below).

	// headerCells picks up err's register (first LIFO pop)
	headerCells := []mypkg.Cell{{Text: "Title"}, {Text: "Status"}}
	// rows picks up err's register AGAIN (second LIFO pop)
	rows := make([]mypkg.Row, 0, 2)

	// The for-loop writes to rows' register, which is the same as
	// headerCells' register - overwriting headerCells!
	for i := 0; i < 2; i++ {
		rows = append(rows, mypkg.Row{
			ID:    "row",
			Cells: []mypkg.Cell{{Text: "cell"}},
		})
	}

	result := mypkg.Result{
		HeaderCells: headerCells,
		TableRows:   rows,
	}

	// Keep data1 and data2 alive past headerCells/rows declarations
	_ = len(data1) + len(data2)

	if len(result.HeaderCells) != 2 {
		return "wrong header count"
	}
	if result.HeaderCells[0].Text != "Title" {
		return "wrong header text"
	}
	if len(result.TableRows) != 2 {
		return "wrong row count"
	}
	return "ok"
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestLargeNativeCallWithRecycling(t *testing.T) {
	t.Parallel()

	type renderResult struct {
		Title string
	}
	type renderMeta struct {
		Hit bool
	}

	renderFunc := func(name string) (renderResult, renderMeta, error) {
		return renderResult{Title: name}, renderMeta{}, nil
	}
	nodeCounter := 0
	getNodeFunc := func() string {
		nodeCounter++
		return fmt.Sprintf("node%d", nodeCounter)
	}
	getSliceFunc := func(n int) []string {
		result := make([]string, n)
		for i := range n {
			result[i] = fmt.Sprintf("item%d", i)
		}
		return result
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"partial1": {"Render": reflect.ValueOf(renderFunc)},
		"partial2": {"Render": reflect.ValueOf(renderFunc)},
		"partial3": {"Render": reflect.ValueOf(renderFunc)},
		"partial4": {"Render": reflect.ValueOf(renderFunc)},
		"partial5": {"Render": reflect.ValueOf(renderFunc)},
		"partial6": {"Render": reflect.ValueOf(renderFunc)},
		"partial7": {"Render": reflect.ValueOf(renderFunc)},
		"partial8": {"Render": reflect.ValueOf(renderFunc)},
		"factory": {
			"GetNode":  reflect.ValueOf(getNodeFunc),
			"GetSlice": reflect.ValueOf(getSliceFunc),
		},
	})
	symbols.SynthesiseAll()

	source := `
package main

import "partial1"
import "partial2"
import "partial3"
import "partial4"
import "partial5"
import "partial6"
import "partial7"
import "partial8"
import "factory"

func run() string {
	// Phase 1: Multi-return calls with blank discards (register recycling)
	data1, meta1, err1 := partial1.Render("p1")
	if err1 != nil { return "err1" }
	_ = data1; _ = meta1; _ = err1

	data2, meta2, err2 := partial2.Render("p2")
	if err2 != nil { return "err2" }
	_ = data2; _ = meta2; _ = err2

	data3, meta3, err3 := partial3.Render("p3")
	if err3 != nil { return "err3" }
	_ = data3; _ = meta3; _ = err3

	data4, meta4, err4 := partial4.Render("p4")
	if err4 != nil { return "err4" }
	_ = data4; _ = meta4; _ = err4

	data5, meta5, err5 := partial5.Render("p5")
	if err5 != nil { return "err5" }
	_ = data5; _ = meta5; _ = err5

	data6, meta6, err6 := partial6.Render("p6")
	if err6 != nil { return "err6" }
	_ = data6; _ = meta6; _ = err6

	data7, meta7, err7 := partial7.Render("p7")
	if err7 != nil { return "err7" }
	_ = data7; _ = meta7; _ = err7

	data8, meta8, err8 := partial8.Render("p8")
	if err8 != nil { return "err8" }
	_ = data8; _ = meta8; _ = err8

	// Phase 2: Many single-return native calls (0 args, 1 return)
	n1 := factory.GetNode()
	n2 := factory.GetNode()
	n3 := factory.GetNode()
	n4 := factory.GetNode()
	n5 := factory.GetNode()
	n6 := factory.GetNode()
	n7 := factory.GetNode()
	n8 := factory.GetNode()
	n9 := factory.GetNode()
	n10 := factory.GetNode()

	// Phase 3: Calls with args
	s1 := factory.GetSlice(3)
	s2 := factory.GetSlice(5)

	// Verify
	if n1 != "node1" { return "bad n1" }
	if n10 != "node10" { return "bad n10" }
	if len(s1) != 3 { return "bad s1" }
	if len(s2) != 5 { return "bad s2" }
	_ = n2; _ = n3; _ = n4; _ = n5; _ = n6; _ = n7; _ = n8; _ = n9
	return "ok"
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestLargeNativeMethodCallWithRecycling(t *testing.T) {
	t.Parallel()

	renderFunc := func(name string) (testRenderData, testRenderMeta, error) {
		return testRenderData{Title: name}, testRenderMeta{}, nil
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"partial1": {"Render": reflect.ValueOf(renderFunc)},
		"partial2": {"Render": reflect.ValueOf(renderFunc)},
		"partial3": {"Render": reflect.ValueOf(renderFunc)},
		"partial4": {"Render": reflect.ValueOf(renderFunc)},
		"partial5": {"Render": reflect.ValueOf(renderFunc)},
		"partial6": {"Render": reflect.ValueOf(renderFunc)},
		"partial7": {"Render": reflect.ValueOf(renderFunc)},
		"partial8": {"Render": reflect.ValueOf(renderFunc)},
		"factory": {
			"Arena":    reflect.ValueOf((*testArena)(nil)),
			"Node":     reflect.ValueOf((*testArenaNode)(nil)),
			"NewArena": reflect.ValueOf(func() *testArena { return &testArena{} }),
		},
	})
	symbols.SynthesiseAll()

	source := `
package main

import "partial1"
import "partial2"
import "partial3"
import "partial4"
import "partial5"
import "partial6"
import "partial7"
import "partial8"
import "factory"

func run() string {
	// Phase 1: Multi-return calls with blank discards
	data1, meta1, err1 := partial1.Render("p1")
	if err1 != nil { return "err1" }
	_ = data1; _ = meta1; _ = err1

	data2, meta2, err2 := partial2.Render("p2")
	if err2 != nil { return "err2" }
	_ = data2; _ = meta2; _ = err2

	data3, meta3, err3 := partial3.Render("p3")
	if err3 != nil { return "err3" }
	_ = data3; _ = meta3; _ = err3

	data4, meta4, err4 := partial4.Render("p4")
	if err4 != nil { return "err4" }
	_ = data4; _ = meta4; _ = err4

	data5, meta5, err5 := partial5.Render("p5")
	if err5 != nil { return "err5" }
	_ = data5; _ = meta5; _ = err5

	data6, meta6, err6 := partial6.Render("p6")
	if err6 != nil { return "err6" }
	_ = data6; _ = meta6; _ = err6

	data7, meta7, err7 := partial7.Render("p7")
	if err7 != nil { return "err7" }
	_ = data7; _ = meta7; _ = err7

	data8, meta8, err8 := partial8.Render("p8")
	if err8 != nil { return "err8" }
	_ = data8; _ = meta8; _ = err8

	// Phase 2: Method calls (arena.GetNode())
	arena := factory.NewArena()
	n1 := arena.GetNode()
	n1.NodeType = 1
	n2 := arena.GetNode()
	n2.NodeType = 2
	n3 := arena.GetNode()
	n3.NodeType = 3
	n4 := arena.GetNode()
	n4.NodeType = 4
	n5 := arena.GetNode()
	n5.NodeType = 5
	n6 := arena.GetNode()
	n6.NodeType = 6
	n7 := arena.GetNode()
	n7.NodeType = 7
	n8 := arena.GetNode()
	n8.NodeType = 8
	n9 := arena.GetNode()
	n9.NodeType = 9
	n10 := arena.GetNode()
	n10.NodeType = 10

	// Phase 3: Method calls with args
	s1 := arena.GetSlice(3)
	s2 := arena.GetSlice(5)

	// Verify
	if n1.Name != "node1" { return "bad n1: " + n1.Name }
	if n10.Name != "node10" { return "bad n10: " + n10.Name }
	if n10.NodeType != 10 { return "bad n10 type" }
	if len(s1) != 3 { return "bad s1" }
	if len(s2) != 5 { return "bad s2" }
	_ = n2; _ = n3; _ = n4; _ = n5; _ = n6; _ = n7; _ = n8; _ = n9
	return "ok"
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestVarDeclMapThenAddr(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"mypkg": {
			"Unmarshal": reflect.ValueOf(func(data []byte, v any) error {
				return json.Unmarshal(data, v)
			}),
		},
	})
	symbols.SynthesiseAll()

	source := `
package main

import "mypkg"

func run() string {
	var pageData map[string]any
	data := []byte(` + "`" + `{"name":"test","count":42}` + "`" + `)
	err := mypkg.Unmarshal(data, &pageData)
	if err != nil {
		return "unmarshal error: " + err.Error()
	}
	if pageData == nil {
		return "nil map"
	}
	if len(pageData) != 2 {
		return "wrong length"
	}
	return "ok"
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestNamedStringMethodCallAsArgument(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"mypkg": {
			"NewWriter":    reflect.ValueOf(func() *testWriter { return &testWriter{} }),
			"NewItem":      reflect.ValueOf(func(id, name string) testItem { return testItem{ID: testCapabilityID(id), Name: name} }),
			"Writer":       reflect.ValueOf((*testWriter)(nil)),
			"Item":         reflect.ValueOf((*testItem)(nil)),
			"CapabilityID": reflect.ValueOf((*testCapabilityID)(nil)),
		},
	})
	symbols.SynthesiseAll()

	source := `
package main

import "mypkg"

func run() string {
	item := mypkg.NewItem("abc", "test")
	writer := mypkg.NewWriter()
	writer.AppendEscapeString(item.ID.String())
	return writer.Result()
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "cap:abc", result)
}

func TestNamedStringMethodCallAsArgumentWithRecycling(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"mypkg": {
			"NewWriter": reflect.ValueOf(func() *testWriter { return &testWriter{} }),
			"NewItem": reflect.ValueOf(func(id, name string) testItem {
				return testItem{ID: testCapabilityID(id), Name: name}
			}),
			"Render": reflect.ValueOf(func() (string, string, error) {
				return "data", "meta", nil
			}),
			"Writer":       reflect.ValueOf((*testWriter)(nil)),
			"Item":         reflect.ValueOf((*testItem)(nil)),
			"CapabilityID": reflect.ValueOf((*testCapabilityID)(nil)),
		},
	})
	symbols.SynthesiseAll()

	source := `
package main

import "mypkg"

func run() string {
	// Multi-return calls with blank discards to populate the recycled register pool.
	data1, meta1, err1 := mypkg.Render()
	if err1 != nil { return "err1" }
	_ = data1; _ = meta1; _ = err1

	data2, meta2, err2 := mypkg.Render()
	if err2 != nil { return "err2" }
	_ = data2; _ = meta2; _ = err2

	// Now the recycled pool should have registers from dead vars.
	item := mypkg.NewItem("def", "test")
	writer := mypkg.NewWriter()
	writer.AppendEscapeString(item.ID.String())
	return writer.Result()
}
`
	service := NewService()
	service.UseSymbols(symbols)
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "cap:def", result)
}
