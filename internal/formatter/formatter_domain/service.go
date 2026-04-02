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
	"context"
	"errors"
	"fmt"
	"go/format"
	"strings"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
)

// formatterServiceImpl implements the FormatterService interface.
type formatterServiceImpl struct {
	// options holds the default formatting settings used when none are given.
	options *FormatOptions
}

// Format formats a .pk file with default options.
//
// Takes source ([]byte) which contains the file content to format.
//
// Returns []byte which contains the formatted file content.
// Returns error when formatting fails.
func (s *formatterServiceImpl) Format(ctx context.Context, source []byte) ([]byte, error) {
	return s.FormatWithOptions(ctx, source, s.options)
}

// FormatWithOptions formats source with custom formatting options.
// It supports both .pk (Single File Component) and plain .html files.
//
// Each block type formatting is extracted into focused helpers.
//
// Takes source ([]byte) which contains the file content to format.
// Takes opts (*FormatOptions) which specifies custom formatting behaviour.
//
// Returns []byte which contains the formatted file content.
// Returns error when the source cannot be parsed.
func (s *formatterServiceImpl) FormatWithOptions(ctx context.Context, source []byte, opts *FormatOptions) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()
	s.recordFormatMetrics(ctx, source)
	defer s.recordformatDuration(ctx, startTime)

	l.Trace("Starting format operation", logger_domain.Int("sourceSize", len(source)))

	fileFormat := s.detectFileFormat(source, opts)

	if fileFormat == FormatHTML {
		return s.formatHTMLOnly(ctx, source, opts)
	}

	return s.formatPK(ctx, source, opts)
}

// FormatRange formats only a specific range within the source. Use it for
// LSP range formatting and format-on-type features.
//
// Takes source ([]byte) which contains the full document to format.
// Takes formatRange (Range) which specifies the line range to format.
// Takes opts (*FormatOptions) which provides formatting settings, or nil for
// defaults.
//
// Returns []byte which contains the source with the specified range formatted.
// Returns error when the range offsets are invalid or formatting fails.
func (s *formatterServiceImpl) FormatRange(ctx context.Context, source []byte, formatRange Range, opts *FormatOptions) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	if opts == nil {
		opts = s.options
	}

	l.Trace("Starting range format operation",
		logger_domain.Int("sourceSize", len(source)),
		logger_domain.Int("startLine", int(formatRange.StartLine)),
		logger_domain.Int("endLine", int(formatRange.EndLine)))

	formatted, err := s.FormatWithOptions(ctx, source, opts)
	if err != nil {
		return nil, fmt.Errorf("formatting document: %w", err)
	}

	startOffset, endOffset, err := getRangeOffsets(source, formatRange)
	if err != nil {
		return nil, fmt.Errorf("computing range offsets: %w", err)
	}

	formattedStartOffset, formattedEndOffset := getFormattedRangeOffsets(formatted, formatRange)

	result := replaceRangeInSource(source, formatted, rangeOffsets{
		sourceStart:    startOffset,
		sourceEnd:      endOffset,
		formattedStart: formattedStartOffset,
		formattedEnd:   formattedEndOffset,
	})

	l.Trace("Range format operation completed",
		logger_domain.Int("originalSize", len(source)),
		logger_domain.Int("resultSize", len(result)))

	return result, nil
}

// detectFileFormat determines whether the source is PK or plain HTML format.
// It uses the explicit format from opts if set, otherwise auto-detects based on
// the presence of a <template> tag.
//
// Takes source ([]byte) which is the content to analyse.
// Takes opts (*FormatOptions) which may contain an explicit format setting.
//
// Returns FileFormat which is the detected or specified file format.
func (*formatterServiceImpl) detectFileFormat(source []byte, opts *FormatOptions) FileFormat {
	if opts.FileFormat != FormatAuto {
		return opts.FileFormat
	}

	sourceString := string(source)
	if strings.Contains(sourceString, "<template>") {
		return FormatPK
	}

	if containsPKTemplateWithAttributes(sourceString) {
		return FormatPK
	}

	return FormatHTML
}

// formatPK formats a Piko Single File Component (.pk) file.
//
// A .pk file contains template, script, style, and i18n blocks. This method
// parses the file, formats each block, and reassembles them.
//
// Takes source ([]byte) which contains the raw .pk file content.
// Takes opts (*FormatOptions) which specifies formatting behaviour.
//
// Returns []byte which contains the formatted file content.
// Returns error when the source cannot be parsed as a valid SFC.
func (s *formatterServiceImpl) formatPK(ctx context.Context, source []byte, opts *FormatOptions) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	sfcResult, err := sfcparser.Parse(source)
	if err != nil {
		formatErrorCount.Add(ctx, 1)
		l.Error("Failed to parse SFC", logger_domain.Error(err))
		return nil, fmt.Errorf("parsing SFC: %w", err)
	}

	formattedTemplate := s.formatTemplateBlock(ctx, sfcResult.Template, opts)
	formattedScripts := s.formatScriptBlocks(ctx, sfcResult.Scripts)
	formattedStyles := s.formatStyleBlocks(ctx, sfcResult.Styles, opts)
	formattedI18nBlocks := s.formatI18nBlocks(ctx, sfcResult.I18nBlocks)

	result := reassembleSFC(sfcResult, formattedTemplate, formattedScripts, formattedStyles, formattedI18nBlocks)
	resultBytes := []byte(result)

	formatBytesOut.Add(ctx, int64(len(resultBytes)))
	l.Trace("Format operation completed (PK)",
		logger_domain.Int("inputSize", len(source)),
		logger_domain.Int("outputSize", len(resultBytes)))

	return resultBytes, nil
}

// formatHTMLOnly formats plain HTML content without SFC block structure.
// It parses the HTML with the AST parser and formats it.
//
// Takes source ([]byte) which contains the raw HTML content.
// Takes opts (*FormatOptions) which controls formatting behaviour.
//
// Returns []byte which contains the formatted HTML content.
// Returns error when the HTML cannot be parsed or contains errors.
func (*formatterServiceImpl) formatHTMLOnly(ctx context.Context, source []byte, opts *FormatOptions) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Formatting plain HTML", logger_domain.Int("sourceSize", len(source)))

	startLocation := &ast_domain.Location{
		Line:   1,
		Column: 1,
		Offset: 0,
	}

	var tree *ast_domain.TemplateAST
	var err error

	if opts.RawHTMLMode {
		tree, err = ast_domain.ParseWithOptions(ctx, string(source), "formatter", startLocation, &ast_domain.ParseOptions{
			RawMode: true,
		})
	} else {
		tree, err = ast_domain.Parse(ctx, string(source), "formatter", startLocation)
	}

	if err != nil {
		formatErrorCount.Add(ctx, 1)
		l.Error("Failed to parse HTML", logger_domain.Error(err))
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	if ast_domain.HasErrors(tree.Diagnostics) {
		formatErrorCount.Add(ctx, 1)
		return nil, errors.New("HTML contains parse errors")
	}

	printer := newPrettyPrinter(opts)

	for _, rootNode := range tree.RootNodes {
		if err := ast_domain.WalkWithVisitor(ctx, printer, rootNode); err != nil {
			formatErrorCount.Add(ctx, 1)
			return nil, fmt.Errorf("formatting HTML: %w", err)
		}
	}

	result := printer.String()
	resultBytes := []byte(result)

	formatBytesOut.Add(ctx, int64(len(resultBytes)))
	l.Trace("Format operation completed (HTML)",
		logger_domain.Int("inputSize", len(source)),
		logger_domain.Int("outputSize", len(resultBytes)))

	return resultBytes, nil
}

// recordFormatMetrics records initial metrics for the format operation.
//
// Takes source ([]byte) which is the input data to measure for byte counting.
func (*formatterServiceImpl) recordFormatMetrics(ctx context.Context, source []byte) {
	formatCount.Add(ctx, 1)
	formatBytesIn.Add(ctx, int64(len(source)))
}

// recordformatDuration records how long the format operation took.
//
// Takes startTime (time.Time) which marks when the format operation began.
func (*formatterServiceImpl) recordformatDuration(ctx context.Context, startTime time.Time) {
	duration := time.Since(startTime).Milliseconds()
	formatDuration.Record(ctx, float64(duration))
}

// formatTemplateBlock formats the template block, returning the original on
// error.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes template (string) which is the template content to format.
// Takes opts (*FormatOptions) which specifies the formatting behaviour.
//
// Returns string which is the formatted template, or the original if
// formatting fails.
func (s *formatterServiceImpl) formatTemplateBlock(ctx context.Context, template string, opts *FormatOptions) string {
	ctx, l := logger_domain.From(ctx, log)
	if template == "" {
		return ""
	}

	l.Trace("Formatting template block", logger_domain.Int("templateSize", len(template)))
	formatted, err := s.formatTemplate(ctx, template, opts)
	if err != nil {
		l.Error("Failed to format template", logger_domain.Error(err))
		return template
	}
	return formatted
}

// formatScriptBlocks formats all script blocks in a slice.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes scripts ([]sfcparser.Script) which contains the script blocks to
// format.
//
// Returns []sfcparser.Script which contains the formatted scripts. Go scripts
// are formatted using go/format; other scripts are kept unchanged.
func (s *formatterServiceImpl) formatScriptBlocks(ctx context.Context, scripts []sfcparser.Script) []sfcparser.Script {
	_, l := logger_domain.From(ctx, log)
	formattedScripts := make([]sfcparser.Script, 0, len(scripts))

	for i := range scripts {
		script := scripts[i]

		if script.IsGo() && script.Content != "" {
			l.Trace("Formatting Go script block", logger_domain.Int("scriptIndex", i))
			formatted, err := s.formatGoScript(script.Content)
			if err != nil {
				l.Warn("Failed to format Go script, keeping original",
					logger_domain.Int("scriptIndex", i),
					logger_domain.Error(err))
			} else {
				script.Content = formatted
			}
		}

		formattedScripts = append(formattedScripts, script)
	}

	return formattedScripts
}

// formatStyleBlocks formats all CSS style blocks using esbuild.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes styles ([]sfcparser.Style) which contains the style blocks to format.
// Takes opts (*FormatOptions) which holds the formatting settings.
//
// Returns []sfcparser.Style which contains the formatted style blocks.
func (s *formatterServiceImpl) formatStyleBlocks(ctx context.Context, styles []sfcparser.Style, opts *FormatOptions) []sfcparser.Style {
	_, l := logger_domain.From(ctx, log)
	formattedStyles := make([]sfcparser.Style, 0, len(styles))

	for i := range styles {
		style := styles[i]

		if style.Content != "" {
			l.Trace("Formatting style block", logger_domain.Int("styleIndex", i))
			formatted, err := s.formatCSS(style.Content, opts)
			if err != nil {
				l.Warn("Failed to format style block, keeping original",
					logger_domain.Int("styleIndex", i),
					logger_domain.Error(err))
			} else {
				style.Content = formatted
			}
		}

		formattedStyles = append(formattedStyles, style)
	}

	return formattedStyles
}

// formatI18nBlocks formats all i18n blocks (JSON format only).
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes i18nBlocks ([]sfcparser.I18nBlock) which contains the i18n blocks to
// format.
//
// Returns []sfcparser.I18nBlock which contains the formatted blocks, with
// original content preserved when formatting fails.
func (s *formatterServiceImpl) formatI18nBlocks(ctx context.Context, i18nBlocks []sfcparser.I18nBlock) []sfcparser.I18nBlock {
	_, l := logger_domain.From(ctx, log)
	formattedI18nBlocks := make([]sfcparser.I18nBlock, 0, len(i18nBlocks))

	for i := range i18nBlocks {
		i18n := i18nBlocks[i]
		lang := i18n.Attributes["lang"]

		if (lang == "json" || lang == "") && i18n.Content != "" {
			l.Trace("Formatting i18n block", logger_domain.Int("i18nIndex", i))
			formatted, err := s.formatI18nJSON(i18n.Content)
			if err != nil {
				l.Warn("Failed to format i18n block, keeping original",
					logger_domain.Int("i18nIndex", i),
					logger_domain.Error(err))
			} else {
				i18n.Content = formatted
			}
		}

		formattedI18nBlocks = append(formattedI18nBlocks, i18n)
	}

	return formattedI18nBlocks
}

// formatTemplate formats a template block using the AST-based PrettyPrinter.
//
// Takes templateContent (string) which is the raw template text to format.
// Takes opts (*FormatOptions) which controls the formatting behaviour.
//
// Returns string which is the formatted template output.
// Returns error when parsing fails or the template contains parse errors.
func (*formatterServiceImpl) formatTemplate(ctx context.Context, templateContent string, opts *FormatOptions) (string, error) {
	tree, err := ast_domain.Parse(ctx, templateContent, "formatter", &ast_domain.Location{
		Line:   1,
		Column: 1,
		Offset: 0,
	})
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	if ast_domain.HasErrors(tree.Diagnostics) {
		return "", errors.New("template contains parse errors")
	}

	printer := newPrettyPrinter(opts)

	for _, rootNode := range tree.RootNodes {
		if err := ast_domain.WalkWithVisitor(ctx, printer, rootNode); err != nil {
			return "", fmt.Errorf("formatting template: %w", err)
		}
	}

	return printer.String(), nil
}

// formatGoScript formats Go code using the standard go/format package.
//
// Takes scriptContent (string) which contains the Go source code to format.
//
// Returns string which contains the formatted Go source code.
// Returns error when the code cannot be parsed or formatted.
func (*formatterServiceImpl) formatGoScript(scriptContent string) (string, error) {
	formatted, err := format.Source([]byte(scriptContent))
	if err != nil {
		return "", fmt.Errorf("formatting Go script: %w", err)
	}

	return string(formatted), nil
}

// formatCSS formats a raw CSS string using the esbuild parser and printer.
//
// Takes cssContent (string) which is the raw CSS to format.
//
// Returns string which is the formatted CSS output.
// Returns error when the CSS cannot be parsed.
func (*formatterServiceImpl) formatCSS(cssContent string, _ *FormatOptions) (string, error) {
	if strings.TrimSpace(cssContent) == "" {
		return "", nil
	}

	esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
	source := es_logger.Source{
		KeyPath: es_logger.Path{
			Text:             "style-block",
			Namespace:        "",
			IgnoredSuffix:    "",
			ImportAttributes: es_logger.ImportAttributes{},
			Flags:            0,
		},
		Contents:       cssContent,
		IdentifierName: "",
		Index:          0,
	}

	tree := css_parser.Parse(esLog, source, css_parser.Options{})
	if esLog.HasErrors() {
		return "", fmt.Errorf("failed to parse CSS: %s", esLog.Done()[0].Data.Text)
	}

	printOptions := css_printer.Options{
		MinifyWhitespace:  false,
		LineLimit:         0,
		InputSourceIndex:  0,
		ASCIIOnly:         false,
		AddSourceMappings: false,
		LegalComments:     0,
		NeedsMetafile:     false,
	}

	symMap := es_ast.NewSymbolMap(1)
	symMap.SymbolsForSource[0] = tree.Symbols
	printed := css_printer.Print(tree, symMap, printOptions)

	return string(printed.CSS), nil
}

// formatI18nJSON formats JSON content within i18n blocks with consistent
// 2-space indentation.
//
// Takes content (string) which is the JSON text to format.
//
// Returns string which is the formatted JSON with 2-space indentation.
// Returns error when the JSON is malformed; the caller should preserve the
// original content.
func (*formatterServiceImpl) formatI18nJSON(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil
	}

	var data any
	if err := json.UnmarshalString(trimmed, &data); err != nil {
		return "", fmt.Errorf("parsing i18n JSON: %w", err)
	}

	formatted, err := json.ConfigStd.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting i18n JSON: %w", err)
	}

	return string(formatted), nil
}

// rangeOffsets holds byte positions for matching source and formatted ranges.
type rangeOffsets struct {
	// sourceStart is the byte offset where the range begins in the source.
	sourceStart int

	// sourceEnd is the byte offset where the target range ends in the source.
	sourceEnd int

	// formattedStart is the byte offset where the relevant section begins in the
	// formatted output.
	formattedStart int

	// formattedEnd is the exclusive end position in the formatted output.
	formattedEnd int
}

// NewFormatterService creates a new formatter service with default options.
//
// Returns FormatterService which is ready for use with default format options.
func NewFormatterService() FormatterService {
	return &formatterServiceImpl{
		options: DefaultFormatOptions(),
	}
}

// NewFormatterServiceWithOptions creates a new formatter service with custom
// options.
//
// Takes opts (*FormatOptions) which specifies the formatting settings. When
// nil, default options are used.
//
// Returns FormatterService which is the configured formatter ready for use.
func NewFormatterServiceWithOptions(opts *FormatOptions) FormatterService {
	if opts == nil {
		opts = DefaultFormatOptions()
	}
	return &formatterServiceImpl{
		options: opts,
	}
}

// containsPKTemplateWithAttributes checks whether the source contains a
// <template> tag with attributes that is NOT a declarative shadow DOM template.
// Shadow root templates use shadowrootmode and should be treated as plain HTML.
//
// Takes source (string) which is the template source text to scan.
//
// Returns bool which is true when the source contains at least one
// <template> tag with attributes other than shadowrootmode.
func containsPKTemplateWithAttributes(source string) bool {
	searchFrom := 0
	for {
		index := strings.Index(source[searchFrom:], "<template ")
		if index < 0 {
			return false
		}
		index += searchFrom

		closeIndex := strings.Index(source[index:], ">")
		if closeIndex < 0 {
			return false
		}

		tag := source[index : index+closeIndex+1]
		if !strings.Contains(tag, "shadowrootmode") {
			return true
		}

		searchFrom = index + closeIndex + 1
	}
}

// getRangeOffsets converts range positions to byte offsets in the source.
//
// Takes source ([]byte) which is the document content to index into.
// Takes formatRange (Range) which specifies the start and end positions.
//
// Returns startOffset (int) which is the byte offset of the range start.
// Returns endOffset (int) which is the byte offset of the range end.
// Returns err (error) when a position cannot be converted to an offset.
func getRangeOffsets(source []byte, formatRange Range) (startOffset, endOffset int, err error) {
	startOffset, err = positionToOffset(source, formatRange.StartLine, formatRange.StartCharacter)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start position: %w", err)
	}

	endOffset, err = positionToOffset(source, formatRange.EndLine, formatRange.EndCharacter)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end position: %w", err)
	}

	return startOffset, endOffset, nil
}

// getFormattedRangeOffsets finds the corresponding range in the formatted
// document using a simple approach of formatting entire lines.
//
// Takes formatted ([]byte) which is the formatted document content.
// Takes formatRange (Range) which specifies the line range to locate.
//
// Returns startOffset (int) which is the byte offset where the range begins.
// Returns endOffset (int) which is the byte offset where the range ends.
func getFormattedRangeOffsets(formatted []byte, formatRange Range) (startOffset, endOffset int) {
	formattedStartOffset, err := positionToOffset(formatted, formatRange.StartLine, 0)
	if err != nil {
		return 0, 0
	}

	formattedEndOffset, err := positionToOffset(formatted, formatRange.EndLine+1, 0)
	if err != nil {
		formattedEndOffset = len(formatted)
	}

	return formattedStartOffset, formattedEndOffset
}

// replaceRangeInSource creates a new byte slice with a given range replaced
// by the matching range from a formatted document.
//
// Takes source ([]byte) which is the original byte slice.
// Takes formatted ([]byte) which holds the replacement content.
// Takes offsets (rangeOffsets) which gives the start and end positions in both
// source and formatted slices.
//
// Returns []byte which is a new slice with the range replaced.
func replaceRangeInSource(source, formatted []byte, offsets rangeOffsets) []byte {
	result := make([]byte, 0, len(source))

	result = append(result, source[:offsets.sourceStart]...)

	if offsets.formattedStart < len(formatted) && offsets.formattedEnd <= len(formatted) {
		result = append(result, formatted[offsets.formattedStart:offsets.formattedEnd]...)
	}

	result = append(result, source[offsets.sourceEnd:]...)

	return result
}

// positionToOffset converts a line and character position to a byte offset.
//
// Takes source ([]byte) which contains the text to search.
// Takes line (uint32) which is the zero-based line number.
// Takes character (uint32) which is the zero-based character position.
//
// Returns int which is the byte offset for the given position.
// Returns error when the position is past the end of the source.
func positionToOffset(source []byte, line, character uint32) (int, error) {
	currentLine := uint32(0)
	currentChar := uint32(0)

	for i := range len(source) {
		if currentLine == line && currentChar == character {
			return i, nil
		}

		if source[i] == '\n' {
			currentLine++
			currentChar = 0
		} else {
			currentChar++
		}
	}

	if currentLine == line && currentChar == character {
		return len(source), nil
	}

	return 0, fmt.Errorf("position %d:%d is beyond end of source", line, character)
}
