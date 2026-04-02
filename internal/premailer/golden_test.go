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

package premailer

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestGoldenFiles(t *testing.T) {
	inputDir := filepath.Join("testdata", "input")
	expectedDir := filepath.Join("testdata", "expected")

	entries, err := os.ReadDir(inputDir)
	require.NoError(t, err, "Failed to read input directory")

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		testName := strings.TrimSuffix(entry.Name(), ".html")
		t.Run(testName, func(t *testing.T) {
			inputPath := filepath.Join(inputDir, entry.Name())
			expectedPath := filepath.Join(expectedDir, entry.Name())

			inputHTML, err := os.ReadFile(inputPath)
			require.NoError(t, err, "Failed to read input file: %s", inputPath)

			tree, err := ast_domain.Parse(context.Background(), string(inputHTML), entry.Name(), nil)
			require.NoError(t, err, "Failed to parse input HTML")

			var opts []Option
			switch testName {
			case "09_remove_classes":
				opts = append(opts, WithRemoveClasses(true))
			case "10_keep_bang_important":
				opts = append(opts, WithKeepBangImportant(true))
			case "13_leftover_important":
				opts = append(opts, WithMakeLeftoverImportant(true))
			case "15_link_query_params":

				opts = append(opts, WithLinkQueryParams(map[string]string{
					"utm_source":   "newsletter",
					"utm_campaign": "test_campaign",
					"utm_medium":   "email",
				}))
			case "16_remove_ids", "18_smart_anchor_links":

				opts = append(opts, WithRemoveIDs(true))
			case "17_css_variables":

				opts = append(opts, WithTheme(map[string]string{
					"colour-primary":    "#6F47EB",
					"border-color":      "var(--g-colour-grey-200)",
					"g-colour-grey-200": "#CAD1D8",
					"border-width":      "2px",
					"spacing":           "1.5rem",
				}))
			}

			premailer := New(tree, opts...)
			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Failed to transform with premailer")

			actualOutput := renderTreeToHTML(transformedTree)

			if *updateGolden {
				err = os.MkdirAll(expectedDir, 0755)
				require.NoError(t, err, "Failed to create expected directory")

				err = os.WriteFile(expectedPath, []byte(actualOutput), 0644)
				require.NoError(t, err, "Failed to write golden file: %s", expectedPath)
				t.Logf("Updated golden file: %s", expectedPath)
				return
			}

			expectedOutput, err := os.ReadFile(expectedPath)
			if os.IsNotExist(err) {
				t.Fatalf("Golden file does not exist: %s\nRun with -update flag to create it.\nActual output:\n%s",
					expectedPath, actualOutput)
			}
			require.NoError(t, err, "Failed to read expected file: %s", expectedPath)

			actualNormalised := normaliseHTML(actualOutput)
			expectedNormalised := normaliseHTML(string(expectedOutput))

			if actualNormalised != expectedNormalised {
				t.Errorf("Output does not match golden file: %s\n\nExpected:\n%s\n\nActual:\n%s",
					expectedPath, expectedNormalised, actualNormalised)
			}
		})
	}
}

func renderTreeToHTML(tree *ast_domain.TemplateAST) string {
	var builder strings.Builder
	for _, node := range tree.RootNodes {
		renderNodeToHTML(&builder, node)
	}
	return builder.String()
}

func renderNodeToHTML(builder *strings.Builder, node *ast_domain.TemplateNode) {
	if node == nil {
		return
	}

	switch node.NodeType {
	case ast_domain.NodeElement:
		builder.WriteString("<")
		builder.WriteString(node.TagName)

		if len(node.Attributes) > 0 {
			attrs := make([]ast_domain.HTMLAttribute, len(node.Attributes))
			copy(attrs, node.Attributes)

			for i := range len(attrs) {
				for j := i + 1; j < len(attrs); j++ {
					if attrs[i].Name > attrs[j].Name {
						attrs[i], attrs[j] = attrs[j], attrs[i]
					}
				}
			}
			for _, attr := range attrs {
				builder.WriteString(" ")
				builder.WriteString(attr.Name)
				builder.WriteString(`="`)
				builder.WriteString(attr.Value)
				builder.WriteString(`"`)
			}
		}

		if isSelfClosing(node.TagName) {
			builder.WriteString(">")
			return
		}

		builder.WriteString(">")

		for _, child := range node.Children {
			renderNodeToHTML(builder, child)
		}

		builder.WriteString("</")
		builder.WriteString(node.TagName)
		builder.WriteString(">")

	case ast_domain.NodeText, ast_domain.NodeRawHTML:
		builder.WriteString(node.TextContent)

	case ast_domain.NodeComment:

		builder.WriteString("<!--")
		builder.WriteString(node.TextContent)
		builder.WriteString("-->")

	case ast_domain.NodeFragment:
		for _, child := range node.Children {
			renderNodeToHTML(builder, child)
		}
	}
}

func isSelfClosing(tagName string) bool {
	selfClosing := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}
	return selfClosing[strings.ToLower(tagName)]
}

func normaliseHTML(html string) string {

	html = strings.ReplaceAll(html, "\r\n", "\n")
	html = strings.ReplaceAll(html, "\r", "\n")

	lines := strings.Split(html, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}
