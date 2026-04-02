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

package formatter_domain

import (
	"strings"

	"piko.sh/piko/internal/sfcparser"
)

const (
	// templateIndent is the prefix used to indent each line in a template block.
	templateIndent = "  "

	// newline is the line separator used when reassembling SFC blocks.
	newline = "\n"

	// space is the space character used between HTML tag attributes.
	space = " "

	// quote is the quotation mark used to wrap attribute values.
	quote = `"`
)

// reassembleSFC takes a parsed SFC result with formatted components and
// reassembles them into a complete .pk file. High-level orchestration function
// that delegates block writing to helper methods.
//
// Takes formattedTemplate (string) which is the formatted template HTML.
// Takes formattedScripts ([]sfcparser.Script) which contains the
// formatted script blocks.
// Takes formattedStyles ([]sfcparser.Style) which contains the
// formatted style blocks.
// Takes formattedI18nBlocks ([]sfcparser.I18nBlock) which contains the
// formatted i18n blocks.
//
// Returns string which is the reassembled .pk file content.
func reassembleSFC(
	_ *sfcparser.ParseResult,
	formattedTemplate string,
	formattedScripts []sfcparser.Script,
	formattedStyles []sfcparser.Style,
	formattedI18nBlocks []sfcparser.I18nBlock,
) string {
	var builder strings.Builder

	writeTemplateBlock(&builder, formattedTemplate)
	writeScriptBlocks(&builder, formattedScripts)
	writeStyleBlocks(&builder, formattedStyles)
	writeI18nBlocks(&builder, formattedI18nBlocks)

	return builder.String()
}

// writeTemplateBlock writes the formatted template content wrapped in
// <template> tags.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes formattedTemplate (string) which contains the template content to wrap.
func writeTemplateBlock(builder *strings.Builder, formattedTemplate string) {
	if formattedTemplate == "" {
		return
	}

	builder.WriteString("<template>" + newline)

	for line := range strings.SplitSeq(strings.TrimSpace(formattedTemplate), newline) {
		if line != "" {
			builder.WriteString(templateIndent)
			builder.WriteString(line)
		}
		builder.WriteString(newline)
	}

	builder.WriteString("</template>" + newline)
}

// writeScriptBlocks writes all script blocks with their attributes and content
// to the output.
//
// Takes builder (*strings.Builder) which collects the output text.
// Takes formattedScripts ([]sfcparser.Script) which holds the script blocks to
// write.
func writeScriptBlocks(builder *strings.Builder, formattedScripts []sfcparser.Script) {
	for i := range formattedScripts {
		script := &formattedScripts[i]
		builder.WriteString(newline + "<script")
		writeAttributes(builder, script.Attributes)
		builder.WriteString(">" + newline)
		writeBlockContent(builder, script.Content)
		builder.WriteString("</script>" + newline)
	}
}

// writeStyleBlocks writes all style blocks with their attributes and content.
//
// Takes builder (*strings.Builder) which collects the output text.
// Takes formattedStyles ([]sfcparser.Style) which holds the style blocks to
// write.
func writeStyleBlocks(builder *strings.Builder, formattedStyles []sfcparser.Style) {
	for i := range formattedStyles {
		style := &formattedStyles[i]
		builder.WriteString(newline + "<style")
		writeAttributes(builder, style.Attributes)
		builder.WriteString(">" + newline)
		writeBlockContent(builder, style.Content)
		builder.WriteString("</style>" + newline)
	}
}

// writeI18nBlocks writes all i18n blocks with their attributes and content.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes formattedI18nBlocks ([]sfcparser.I18nBlock) which contains the i18n
// blocks to write.
func writeI18nBlocks(builder *strings.Builder, formattedI18nBlocks []sfcparser.I18nBlock) {
	for i := range formattedI18nBlocks {
		i18n := &formattedI18nBlocks[i]
		builder.WriteString(newline + "<i18n")
		writeAttributes(builder, i18n.Attributes)
		builder.WriteString(">" + newline)
		writeBlockContent(builder, i18n.Content)
		builder.WriteString("</i18n>" + newline)
	}
}

// writeAttributes writes HTML attributes to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted attributes.
// Takes attributes (map[string]string) which contains the key-value pairs to
// write.
func writeAttributes(builder *strings.Builder, attributes map[string]string) {
	for key, value := range attributes {
		if value == "" {
			builder.WriteString(space + key)
		} else {
			builder.WriteString(space + key + "=" + quote + value + quote)
		}
	}
}

// writeBlockContent writes block content to the builder, making sure it ends
// with a newline.
//
// Takes builder (*strings.Builder) which receives the formatted content.
// Takes content (string) which is the raw content to write.
func writeBlockContent(builder *strings.Builder, content string) {
	if content == "" {
		return
	}

	trimmed := strings.TrimSpace(content)
	builder.WriteString(trimmed)

	if !strings.HasSuffix(trimmed, newline) {
		builder.WriteString(newline)
	}
}
