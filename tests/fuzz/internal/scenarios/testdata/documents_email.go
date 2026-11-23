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

package email_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_adapters"
	"piko.sh/piko/internal/pml/pml_components"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/pml/pml_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_test/pages/mocks"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestGeneratedPikoScenario(t *testing.T) {
	ctx := t.Context()
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	seed, err := hex.DecodeString(token)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(seed), 2)

	title := "dispatch-" + token
	link := "https://example.test/message/" + token
	button := "open-" + token
	var source strings.Builder
	source.WriteString("<template><pml-container>")
	var markers []string
	for i := range int(seed[0]%4) + 1 {
		marker := fmt.Sprintf("line-%d-%s", i, token)
		markers = append(markers, marker)
		fmt.Fprintf(&source, `<pml-row><pml-col><pml-p padding="%dpx">%s</pml-p></pml-col></pml-row>`, seed[1]%8, marker)
	}
	fmt.Fprintf(&source, `<pml-row><pml-col><pml-button href="%s">%s</pml-button></pml-col></pml-row></pml-container></template>`, link, button)
	parts, err := sfcparser.Parse([]byte(source.String()))
	require.NoError(t, err)
	tree, err := ast_domain.Parse(ctx, parts.Template, "dispatch.pk", nil)
	require.NoError(t, err)
	require.Empty(t, tree.Diagnostics)
	registry, err := pml_components.RegisterBuiltIns(ctx)
	require.NoError(t, err)
	transformer := pml_domain.NewTransformer(registry, pml_adapters.NewMediaQueryCollector(), pml_adapters.NewMSOConditionalCollector())
	renderer := render_domain.NewRenderOrchestrator(transformer, nil, mocks.NewMockRegistry(t), mocks.NewMockCSRF())
	var buf bytes.Buffer
	req := httptest.NewRequest("GET", "http://example.test/message", nil)
	require.NoError(t, renderer.RenderEmail(ctx, &buf, req, render_domain.RenderEmailOptions{
		PageID: "dispatch", Template: tree,
		Metadata:         &templater_dto.InternalMetadata{Title: title, Language: "en"},
		PremailerOptions: buildPremailerOptions(nil),
	}))

	doc, err := html.Parse(&buf)
	require.NoError(t, err)
	var gotLines, hrefs, titles, labels []string
	tables := 0
	for n := range doc.Descendants() {
		if n.Type == html.ElementNode {
			require.False(t, strings.HasPrefix(n.Data, "pml-"), "untransformed email tag %s", n.Data)
			switch n.Data {
			case "table":
				tables++
			case "a":
				for _, a := range n.Attr {
					if a.Key == "href" {
						hrefs = append(hrefs, a.Val)
					}
				}
				for child := range n.Descendants() {
					if child.Type == html.TextNode && strings.TrimSpace(child.Data) != "" {
						labels = append(labels, strings.TrimSpace(child.Data))
					}
				}
			case "title":
				if n.FirstChild != nil {
					titles = append(titles, n.FirstChild.Data)
				}
			}
		}
		if n.Type == html.TextNode && strings.HasPrefix(strings.TrimSpace(n.Data), "line-") {
			gotLines = append(gotLines, strings.TrimSpace(n.Data))
		}
	}
	require.Positive(t, tables)
	require.Equal(t, markers, gotLines)
	require.Equal(t, []string{title}, titles)
	require.Equal(t, []string{link}, hrefs)
	require.Equal(t, []string{button}, labels)

	transformed, _, diagnostics := transformer.Transform(ctx, tree, pml_dto.DefaultConfig())
	for _, d := range diagnostics {
		require.NotEqual(t, pml_domain.SeverityError, d.Severity, "%s", d.Message)
	}
	require.NotNil(t, transformed)
	plain, err := renderer.RenderASTToPlainText(ctx, transformed)
	require.NoError(t, err)
	for _, marker := range append(markers, button) {
		require.Equal(t, 1, strings.Count(plain, marker), "plain-text marker %q", marker)
		_, rest, found := strings.Cut(plain, marker)
		require.True(t, found, "plain-text order at %q", marker)
		plain = rest
	}
}
