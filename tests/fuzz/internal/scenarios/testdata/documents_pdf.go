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

package pdfwriter_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/sfcparser"
)

func TestGeneratedPikoScenario(t *testing.T) {
	ctx := t.Context()
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	seed, err := hex.DecodeString(token)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(seed), 3)
	title := "statement-" + token
	var links []string
	var source strings.Builder
	source.WriteString("<template><div><section class=\"dimensions\">Box</section>")
	for i := range int(seed[0]%3) + 1 {
		link := fmt.Sprintf("https://example.test/%s/%d", token, i)
		links = append(links, link)
		fmt.Fprintf(&source, `<div><a href="%s">Item %d</a></div>`, link, i)
	}
	source.WriteString("</div></template>")
	parts, err := sfcparser.Parse([]byte(source.String()))
	require.NoError(t, err)
	tree, err := ast_domain.Parse(ctx, parts.Template, "statement.pk", nil)
	require.NoError(t, err)
	require.Empty(t, tree.Diagnostics)
	entries := []layouter_dto.FontEntry{{Family: fonts.NotoSansFamilyName, Weight: 400, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansRegularTTF}}
	metrics, err := layouter_adapters.NewGoTextFontMetrics(entries)
	require.NoError(t, err)
	layout := pdfwriter_adapters.NewLayouterAdapter(metrics, &layouter_adapters.MockImageResolver{})
	page := layouter_dto.PageA4
	page.Width, page.Height = 440+float64(seed[1]), 600+float64(seed[2])
	width, height := 48+int(seed[0]%48), 18+int(seed[1]%16)
	padding := int(seed[2] % 5)
	css := fmt.Sprintf(".dimensions { width: %dpt; height: %dpt; padding: %dpt; box-sizing: %s; }", width, height, padding, []string{"content-box", "border-box"}[seed[0]%2])
	boxes, err := layout.Layout(ctx, tree, css, layouter_dto.LayoutConfig{Page: page, DefaultFontSize: 12})
	require.NoError(t, err)
	root, ok := boxes.RootBox.(*layouter_domain.LayoutBox)
	require.True(t, ok)
	var dimensions []*layouter_domain.LayoutBox
	var scan func(*layouter_domain.LayoutBox)
	scan = func(box *layouter_domain.LayoutBox) {
		if box.SourceNode != nil && box.SourceNode.TagName == "section" {
			dimensions = append(dimensions, box)
		}
		for _, child := range box.Children {
			scan(child)
		}
	}
	scan(root)
	require.Len(t, dimensions, 1, "PIKO_CHECK css-box: section geometry")
	if seed[0]%2 == 1 {
		width -= 2 * padding
		height -= 2 * padding
	}
	require.InDelta(t, width, dimensions[0].ContentWidth, 0.001, "PIKO_CHECK css-width: box-sizing")
	require.InDelta(t, height, dimensions[0].ContentHeight, 0.001, "PIKO_CHECK css-height: box-sizing")
	painter := pdfwriter_domain.NewPdfPainter(page.Width, page.Height, entries, nil)
	pdfwriter_domain.ConfigurePainter(painter, pdfwriter_domain.PainterConfig{Metadata: &pdfwriter_domain.PdfMetadata{Title: title}})
	var out bytes.Buffer
	require.NoError(t, painter.Paint(ctx, boxes, &out))

	doc, err := pdfparse.Parse(out.Bytes())
	require.NoError(t, err)
	dict := func(obj pdfparse.Object) pdfparse.Dict {
		t.Helper()
		resolved, err := doc.Resolve(obj)
		require.NoError(t, err)
		require.Equal(t, pdfparse.ObjectDictionary, resolved.Type)
		return resolved.Value.(pdfparse.Dict)
	}
	info := dict(doc.Trailer().Get("Info"))
	require.Equal(t, title, info.Get("Title").Value)
	catalog := dict(doc.Trailer().Get("Root"))
	require.Equal(t, "Catalog", catalog.GetName("Type"))
	pages := dict(catalog.Get("Pages"))
	require.EqualValues(t, 1, pages.GetInt("Count"))
	kids := pages.GetArray("Kids")
	require.Len(t, kids, 1)
	first := dict(kids[0])
	require.Equal(t, "Page", first.GetName("Type"))
	mediaBox := first.GetArray("MediaBox")
	require.Len(t, mediaBox, 4)
	for i, want := range []float64{0, 0, page.Width, page.Height} {
		require.EqualValues(t, want, mediaBox[i].Value, "MediaBox[%d]", i)
	}
	contents, err := doc.Resolve(first.Get("Contents"))
	require.NoError(t, err)
	stream, err := doc.DecodeStream(contents)
	require.NoError(t, err)
	require.Contains(t, string(stream), "BT", "expected text in page contents")
	var uris []string
	for _, annotRef := range first.GetArray("Annots") {
		annot := dict(annotRef)
		require.Equal(t, "Link", annot.GetName("Subtype"))
		action := dict(annot.Get("A"))
		require.Equal(t, "URI", action.GetName("S"))
		uri := action.Get("URI")
		require.Equal(t, pdfparse.ObjectString, uri.Type)
		uris = append(uris, uri.Value.(string))
	}
	require.ElementsMatch(t, links, uris, "clickable destinations on the page")
}
