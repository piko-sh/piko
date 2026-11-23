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

package wasm_runner_test

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

func TestGeneratedPikoCase(t *testing.T) {
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	require.NotEmpty(t, token)
	title := "wasm-" + token
	sources := loadSources(t, "testdata/01_static_page_e2e/src")
	page, ok := sources["pages/home.pk"]
	require.True(t, ok)
	require.Contains(t, page, "Hello World")
	count := 2 + int(token[0])%4
	var expected []string
	var replacement strings.Builder
	fmt.Fprintf(&replacement, "<h1>%s</h1><ul id=\"wasm-rows\">", title)
	for i := range count {
		label := fmt.Sprintf("%s-%d", title, (i+int(token[len(token)-1]))%count)
		expected = append(expected, label)
		fmt.Fprintf(&replacement, "<li>%s</li>", label)
	}
	replacement.WriteString("</ul>")
	sources["pages/home.pk"] = strings.Replace(page, "<h1>Hello World</h1>", replacement.String(), 1)
	executor := NewWASMExecutor(wasmTestDir)
	spec := loadTestSpec(t, testCase{Path: "testdata/01_static_page_e2e"})
	response, err := executor.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{Sources: sources, ModuleName: spec.ModuleName, RequestURL: spec.RequestURL})
	require.NoError(t, err)
	if !response.Success {
		require.FailNow(t, fmt.Sprintf("PIKO_CHECK render: valid generated source rejected: %s", response.Error))
	}
	doc, err := html.Parse(strings.NewReader(response.HTML))
	require.NoError(t, err)
	var heading string
	var rows []string
	for node := range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		if node.Data == "h1" && node.FirstChild != nil {
			heading = node.FirstChild.Data
		}
		if node.Data != "ul" {
			continue
		}
		for _, attr := range node.Attr {
			if attr.Key != "id" || attr.Val != "wasm-rows" {
				continue
			}
			for child := range node.ChildNodes() {
				if child.Type == html.ElementNode && child.Data == "li" && child.FirstChild != nil {
					rows = append(rows, child.FirstChild.Data)
				}
			}
		}
	}
	if heading != title {
		require.FailNow(t, fmt.Sprintf("PIKO_CHECK heading: got %q, want %q", heading, title))
	}
	if !slices.Equal(rows, expected) {
		require.FailNow(t, fmt.Sprintf("PIKO_CHECK list-order: got %q, want %q", rows, expected))
	}
	dir := os.Getenv("PIKO_FUZZ_CASE_DIR")
	var manifest struct {
		Paths  []string
		Probes []struct {
			Path   string
			Output []struct {
				ID, Text, Value string
				HasValue        bool
			}
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, "case.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &manifest))
	shared := map[string]string{}
	for _, path := range manifest.Paths {
		if strings.HasPrefix(path, "pages/equivalence-") || strings.HasPrefix(path, "partials/equivalence/") {
			source, err := os.ReadFile(filepath.Join(dir, "src", path))
			require.NoError(t, err)
			shared[path] = string(source)
		}
	}
	if len(shared) == 0 {
		return
	}
	checked := 0
	for _, p := range manifest.Probes {
		if !strings.HasPrefix(p.Path, "/equivalence-") {
			continue
		}
		response, err := executor.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{Sources: shared, ModuleName: "example.org/pikofuzzcase", RequestURL: p.Path})
		require.NoError(t, err)
		require.True(t, response.Success, "PIKO_CHECK shared-render: %s, %s", p.Path, response.Error)
		doc, err := html.Parse(strings.NewReader(response.HTML))
		require.NoError(t, err)
		var got, want [][4]string
		for n := range doc.Descendants() {
			if n.Type != html.ElementNode || n.Data != "output" {
				continue
			}
			var row [4]string
			for _, a := range n.Attr {
				switch a.Key {
				case "data-fuzz-output":
					row[0] = a.Val
				case "data-value":
					row[2], row[3] = a.Val, "true"
				}
			}
			for text := range n.Descendants() {
				if text.Type == html.TextNode {
					row[1] += text.Data
				}
			}
			got = append(got, row)
		}
		for _, w := range p.Output {
			want = append(want, [4]string{w.ID, w.Text, w.Value, fmt.Sprint(w.HasValue)})
		}
		if !reflect.DeepEqual(got, want) {
			require.FailNow(t, fmt.Sprintf("PIKO_CHECK shared-partial-attribute: %s got %q want %q", p.Path, got, want))
		}
		checked++
	}
	require.Positive(t, checked, "PIKO_CHECK shared-subset: generated sources lacked modelled requests")

}
