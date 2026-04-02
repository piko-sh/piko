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

package pdfwriter_domain

// Renders a layout box tree to a multi-page PDF using absolute coordinates.
// Embeds TrueType fonts as CIDFont Type2 when font entries are provided,
// falling back to Helvetica Type1 when no embedded fonts are available.
// The Y-axis is flipped: layout uses top-left origin, PDF uses bottom-left.

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

const (
	// baselineRatio is the fallback fraction of font size used as baseline offset.
	baselineRatio = 0.8

	// fontWeightNormal is CSS font-weight: normal (400).
	fontWeightNormal = 400

	// fontWeightBold is the CSS threshold for bold (700).
	fontWeightBold = 700

	// percentageDivisor converts a percentage value to a fraction.
	percentageDivisor = 100

	// percentageDivisorFloat converts a percentage to a fraction (float64).
	percentageDivisorFloat = 100.0

	// luminanceRed is the BT.709 luminance coefficient for red.
	luminanceRed = 0.2126

	// luminanceGreen is the BT.709 luminance coefficient for green.
	luminanceGreen = 0.7152

	// luminanceBlue is the BT.709 luminance coefficient for blue.
	luminanceBlue = 0.0722

	// syntheticItalicSkew is tan(12 deg) for oblique synthesis.
	syntheticItalicSkew = 0.2126

	// borderDoubleMinWidth is the minimum width for double border rendering.
	borderDoubleMinWidth = 3

	// borderDoubleDivisor splits the border width into thirds for double borders.
	borderDoubleDivisor = 3

	// borderSideCount is the number of border sides (top, right, bottom, left).
	borderSideCount = 4

	// darkenFactor is the default factor for darkening 3D border sides.
	darkenFactor = 0.5

	// lightenFactor is the default factor for lightening 3D border sides.
	lightenFactor = 0.5

	// shadowBlurSteps is the number of layered passes for blurred shadows.
	shadowBlurSteps = 8

	// textShadowBlurSteps is the number of passes for blurred text shadows.
	textShadowBlurSteps = 6

	// defaultFormFontSize is the default font size for form fields.
	defaultFormFontSize = 12

	// selectArrowInset is the right-side inset for the select dropdown arrow.
	selectArrowInset = 8

	// selectArrowStrokeGrey is the grey value for the select arrow stroke.
	selectArrowStrokeGrey = 0.3

	// selectArrowLineWidth is the stroke width for the select arrow.
	selectArrowLineWidth = 0.75

	// syntheticBoldStrokeRatio is the stroke width fraction for synthetic bold.
	syntheticBoldStrokeRatio = 0.025

	// textLineThroughFraction is the vertical fraction for line-through decoration.
	textLineThroughFraction = 0.45

	// textUnderlineOffset is the vertical offset fraction for underlines.
	textUnderlineOffset = 0.1

	// decorationDoubleGapRatio is the gap multiplier for double decoration lines.
	decorationDoubleGapRatio = 1.5

	// originDefaultFraction is the default origin fraction (50%).
	originDefaultFraction = 0.5

	// pdfDictCloseSuffix is the closing token for a PDF dictionary.
	pdfDictCloseSuffix = " >>"

	// percentSuffix is the percent character suffix.
	percentSuffix = "%"

	// formFieldTagSelect is the HTML tag name for select elements.
	formFieldTagSelect = "select"

	// formFieldAttrValue is the HTML attribute name "value".
	formFieldAttrValue = "value"
)

// resolveBaselineOffset returns the distance from the top of the text
// content box to the text baseline. Uses the layout-computed value
// when available, falling back to the hardcoded baselineRatio for
// boxes that did not pass through the full inline layout pipeline.
//
// Takes box (*layouter_domain.LayoutBox) which specifies the layout box to measure.
//
// Returns float64 which holds the baseline offset in points.
func resolveBaselineOffset(box *layouter_domain.LayoutBox) float64 {
	if box.BaselineOffset != 0 {
		return box.BaselineOffset
	}
	return box.Style.FontSize * baselineRatio
}

// pdfAnnotation represents a link annotation to be written into a page.
type pdfAnnotation struct {
	// uri holds the external URI target for the link.
	uri string

	// dest holds the internal named destination target for the link.
	dest string

	// x1 holds the left edge of the annotation rectangle in points.
	x1 float64

	// y1 holds the bottom edge of the annotation rectangle in points.
	y1 float64

	// x2 holds the right edge of the annotation rectangle in points.
	x2 float64

	// y2 holds the top edge of the annotation rectangle in points.
	y2 float64

	// pageIndex holds the zero-based index of the page containing this annotation.
	pageIndex int
}

// namedDestination represents a location in the PDF that can be linked to.
type namedDestination struct {
	// name holds the destination identifier used for internal linking.
	name string

	// y holds the vertical position on the page in points.
	y float64

	// pageIndex holds the zero-based index of the page containing this destination.
	pageIndex int
}

// PdfMetadata holds optional metadata fields for the PDF info dictionary.
type PdfMetadata struct {
	// Title holds the document title for the PDF info dictionary.
	Title string

	// Author holds the document author for the PDF info dictionary.
	Author string

	// Subject holds the document subject for the PDF info dictionary.
	Subject string

	// Keywords holds the document keywords for the PDF info dictionary.
	Keywords string

	// Creator holds the creating application name for the PDF info dictionary.
	Creator string
}

// pdfFontKey identifies a font by family, weight, and style.
type pdfFontKey struct {
	// family holds the font family name.
	family string

	// weight holds the CSS font weight value.
	weight int

	// style holds the CSS font style value.
	style int
}

// PdfPainter renders a layout box tree to PDF.
type PdfPainter struct {
	// svgWriter is an optional port for rendering SVGs as native PDF
	// vector commands. When nil, SVGs fall through to the raster image
	// path.
	svgWriter SVGWriterPort

	// imageData holds the port that provides image bytes for embedding.
	imageData ImageDataPort

	// svgData is an optional port that provides raw SVG markup for
	// sources. Required alongside svgWriter for vector rendering.
	svgData SVGDataPort

	// variableFonts tracks which font keys are variable font instances.
	variableFonts map[pdfFontKey]bool

	// viewerPrefs holds optional viewer preference settings for the PDF.
	viewerPrefs *ViewerPreferences

	// fontDataMap holds raw font data keyed by family, weight, and style.
	fontDataMap map[pdfFontKey][]byte

	// acroformBuilder holds the builder for interactive form fields.
	acroformBuilder *AcroFormBuilder

	// pdfaConfig holds the optional PDF/A conformance configuration.
	pdfaConfig *PdfAConfig

	// glyphWidthFunc computes variation-aware glyph advance width in font
	// design units, set by the caller for variable font support.
	glyphWidthFunc func(family string, weight int, style int, glyphID uint16) int

	// fontEmbedder holds the font embedder that writes CIDFont objects.
	fontEmbedder *FontEmbedder

	// extGStateManager holds the manager for extended graphics state resources.
	extGStateManager *ExtGStateManager

	// imageEmbedder holds the image embedder that writes XObject images.
	imageEmbedder *ImageEmbedder

	// shadingManager holds the manager for gradient shading resources.
	shadingManager *ShadingManager

	// metadata holds optional PDF info dictionary metadata.
	metadata *PdfMetadata

	// outlineBuilder holds the builder for the PDF document outline (bookmarks).
	outlineBuilder *OutlineBuilder

	// writer is set during Paint() to give sub-methods access to the
	// document writer for allocating form XObject numbers.
	writer *PdfDocumentWriter

	// structTree holds the structure tree for tagged PDF accessibility.
	structTree *StructTree

	// watermark holds the optional watermark configuration.
	watermark *WatermarkConfig

	// maskFormObjects collects transparency group form XObjects created
	// for mask-image during painting. Written to the PDF after page
	// content streams.
	maskFormObjects []maskFormObject

	// fontEntries holds the font entries available for embedding.
	fontEntries []layouter_dto.FontEntry

	// structStack holds the current stack of structure tree nodes during painting.
	structStack []*StructNode

	// namedDests holds collected named destinations for internal linking.
	namedDests []namedDestination

	// annotations holds collected link annotations across all pages.
	annotations []pdfAnnotation

	// pageLabels holds optional page label ranges for custom page numbering.
	pageLabels []PageLabelRange

	// pageWidth holds the page width in points.
	pageWidth float64

	// pageHeight holds the page height in points.
	pageHeight float64

	// pageYOffset holds the current vertical offset for coordinate translation.
	pageYOffset float64

	// basePageYOffset holds the base vertical offset for the current page.
	basePageYOffset float64
}

// pageObj holds the allocated object numbers for a single PDF page.
type pageObj struct {
	// pageNumber holds the PDF object number for the page dictionary.
	pageNumber int

	// contentNumber holds the PDF object number for the page content stream.
	contentNumber int
}

// maskFormObject holds the data for a deferred transparency group
// form XObject used as an SMask.
type maskFormObject struct {
	// bbox holds the bounding box array string for the form XObject.
	bbox string

	// shadingName holds the shading resource name, if any.
	shadingName string

	// content holds the raw content stream bytes for the form XObject.
	content []byte

	// objectNumber holds the allocated PDF object number.
	objectNumber int
}

// NewPdfPainter creates a new PDF painter with the given page dimensions
// and optional font entries for embedding.
//
// Takes pageWidth (float64) which is the page width in points.
// Takes pageHeight (float64) which is the page height in points.
// Takes fontEntries ([]layouter_dto.FontEntry) which are the fonts
// available for embedding. May be nil for Helvetica fallback.
// Takes imageData (ImageDataPort) which provides image bytes for
// embedding. May be nil to skip image rendering.
//
// Returns *PdfPainter.
func NewPdfPainter(pageWidth, pageHeight float64, fontEntries []layouter_dto.FontEntry, imageData ImageDataPort) *PdfPainter {
	fontDataMap := make(map[pdfFontKey][]byte, len(fontEntries))
	for _, entry := range fontEntries {
		if entry.IsVariable {
			for weight := entry.WeightMin; weight <= entry.WeightMax; weight += percentageDivisor {
				key := pdfFontKey{family: entry.Family, weight: weight, style: entry.Style}
				fontDataMap[key] = entry.Data
			}
			continue
		}
		key := pdfFontKey{family: entry.Family, weight: entry.Weight, style: entry.Style}
		fontDataMap[key] = entry.Data
	}

	return &PdfPainter{
		pageWidth:        pageWidth,
		pageHeight:       pageHeight,
		fontEntries:      fontEntries,
		fontEmbedder:     NewFontEmbedder(),
		fontDataMap:      fontDataMap,
		extGStateManager: NewExtGStateManager(),
		imageEmbedder:    NewImageEmbedder(),
		imageData:        imageData,
		shadingManager:   NewShadingManager(),
		outlineBuilder:   NewOutlineBuilder(),
		acroformBuilder:  NewAcroFormBuilder(),
	}
}

// PainterConfig holds optional configuration applied to a PdfPainter before
// painting. Use ConfigurePainter to apply this configuration in a single call.
type PainterConfig struct {
	// SVGWriter is an optional SVG-to-PDF vector writer. When set, SVGs
	// are rendered as native PDF drawing commands instead of raster images.
	SVGWriter SVGWriterPort

	// SVGData provides raw SVG markup for sources. Required alongside
	// SVGWriter for vector rendering.
	SVGData SVGDataPort

	// Metadata holds optional PDF metadata fields (title, author, etc.).
	Metadata *PdfMetadata

	// ViewerPrefs configures how PDF viewers display the document.
	ViewerPrefs *ViewerPreferences

	// Watermark holds optional watermark configuration.
	Watermark *WatermarkConfig

	// PdfAConfig holds optional PDF/A conformance configuration.
	PdfAConfig *PdfAConfig

	// GlyphWidthFunc computes variation-aware glyph advance widths in font
	// design units for variable font support.
	//
	// When nil, widths are read from hmtx via GlyphAdvanceWidth.
	GlyphWidthFunc func(family string, weight int, style int, glyphID uint16) int

	// PageLabels holds optional page label ranges for custom page numbering.
	PageLabels []PageLabelRange

	// Tagged enables PDF structure tagging for accessibility (PDF/UA).
	Tagged bool
}

// ConfigurePainter applies all settings from config to the painter. Call this
// after NewPdfPainter and before Paint.
//
// Takes painter (*PdfPainter) which is the painter to configure.
// Takes config (PainterConfig) which holds the settings to apply.
func ConfigurePainter(painter *PdfPainter, config PainterConfig) {
	if config.Metadata != nil {
		painter.setMetadata(config.Metadata)
	}
	if config.ViewerPrefs != nil {
		painter.setViewerPreferences(config.ViewerPrefs)
	}
	if len(config.PageLabels) > 0 {
		painter.setPageLabels(config.PageLabels)
	}
	if config.Watermark != nil {
		painter.setWatermarkConfig(*config.Watermark)
	}
	if config.Tagged {
		painter.enableTaggedPDF()
	}
	if config.PdfAConfig != nil {
		painter.setPdfA(config.PdfAConfig)
	}
	if config.SVGWriter != nil && config.SVGData != nil {
		painter.setSVGWriter(config.SVGWriter, config.SVGData)
	}
	if config.GlyphWidthFunc != nil {
		painter.setGlyphWidthFunc(config.GlyphWidthFunc)
	}
}

// MarkVariableFont records that a font key corresponds to a variable font instance.
//
// Takes family (string) which specifies the font family name.
// Takes weight (int) which specifies the CSS font weight value.
// Takes style (int) which specifies the CSS font style value.
func (painter *PdfPainter) MarkVariableFont(family string, weight int, style int) {
	if painter.variableFonts == nil {
		painter.variableFonts = make(map[pdfFontKey]bool)
	}
	painter.variableFonts[pdfFontKey{family: family, weight: weight, style: style}] = true
}

// Paint renders the layout result to the given writer as a PDF document.
//
// Takes result (*layouter_dto.LayoutResult) which holds the layout tree and page list.
// Takes output (io.Writer) which specifies the destination for the PDF bytes.
//
// Returns error when the root box is invalid or image embedding fails.
func (painter *PdfPainter) Paint(ctx context.Context, result *layouter_dto.LayoutResult, output io.Writer) error {
	rootBox, ok := result.RootBox.(*layouter_domain.LayoutBox)
	if !ok {
		return fmt.Errorf("pdfwriter: %w", ErrInvalidRootBox)
	}

	pageCount := len(result.Pages)
	if pageCount == 0 {
		pageCount = 1
	}

	writer := &PdfDocumentWriter{}
	painter.writer = writer
	writer.WriteHeader()

	catalogueNumber := writer.AllocateObject()
	pagesNumber := writer.AllocateObject()

	pageObjs := make([]pageObj, pageCount)
	for i := range pageObjs {
		pageObjs[i].pageNumber = writer.AllocateObject()
		pageObjs[i].contentNumber = writer.AllocateObject()
	}

	watermarkPrefix, watermarkFontResource := painter.prepareWatermark(writer)
	streams := painter.renderPageStreams(ctx, rootBox, pageCount, watermarkPrefix)

	kids := make([]string, pageCount)
	for i, po := range pageObjs {
		kids[i] = FormatReference(po.pageNumber)
	}
	writer.WriteObject(pagesNumber, fmt.Sprintf(
		"<< /Type /Pages /Kids [%s] /Count %d >>",
		strings.Join(kids, " "), pageCount))

	for i, s := range streams {
		writer.WriteStreamObject(pageObjs[i].contentNumber, "", []byte(s.String()))
	}

	fontResourceEntries := painter.writeFontResources(writer)
	resources, resourcesError := painter.buildResourcesDict(writer, fontResourceEntries, watermarkFontResource)
	if resourcesError != nil {
		return resourcesError
	}
	pageAnnotRefs := painter.writeAnnotations(writer, pageCount)

	pageObjNumbers := extractPageObjectNumbers(pageObjs)

	acroformNumber := painter.writeAcroformObjects(writer, pageObjNumbers, pageAnnotRefs, pageCount)

	painter.writePageObjects(writer, pageObjs, pagesNumber, resources, pageAnnotRefs, pageCount)

	painter.writeCatalogueAndTrailer(writer, catalogueNumber, pagesNumber, acroformNumber, pageObjNumbers)

	_, writeError := output.Write(writer.Bytes())
	return writeError
}

// writeCatalogueAndTrailer writes the catalogue object,
// info dictionary, and cross-reference trailer to the PDF
// document writer.
//
// Takes writer (*PdfDocumentWriter) which specifies the document writer.
// Takes catalogueNumber (int) which specifies the catalogue object number.
// Takes pagesNumber (int) which specifies the pages tree root object number.
// Takes acroformNumber (int) which specifies the AcroForm object number, or zero if none.
// Takes pageObjNumbers ([]int) which holds the per-page object numbers.
func (painter *PdfPainter) writeCatalogueAndTrailer(
	writer *PdfDocumentWriter, catalogueNumber int, pagesNumber int, acroformNumber int, pageObjNumbers []int,
) {
	outlineRootNumber := painter.outlineBuilder.WriteObjects(writer, pageObjNumbers)
	structTreeRootNumber := painter.writeStructTree(writer, pageObjNumbers)
	catalogueDict := painter.buildCatalogueDict(pagesNumber, outlineRootNumber, structTreeRootNumber, acroformNumber, pageObjNumbers, writer)
	writer.WriteObject(catalogueNumber, catalogueDict)

	infoNumber := writer.AllocateObject()
	writer.WriteObject(infoNumber, painter.buildInfoDictionary())
	writer.WriteTrailer(catalogueNumber, infoNumber)
}

// setMetadata sets optional metadata fields for the PDF info dictionary.
//
// Takes metadata (*PdfMetadata) which holds the metadata fields to apply.
func (painter *PdfPainter) setMetadata(metadata *PdfMetadata) {
	painter.metadata = metadata
}

// setViewerPreferences configures how PDF viewers display the document.
//
// Takes prefs (*ViewerPreferences) which specifies the viewer preference settings.
func (painter *PdfPainter) setViewerPreferences(prefs *ViewerPreferences) {
	painter.viewerPrefs = prefs
}

// setPageLabels configures page label ranges for the document.
//
// Takes ranges ([]PageLabelRange) which specifies the page label ranges to apply.
func (painter *PdfPainter) setPageLabels(ranges []PageLabelRange) {
	painter.pageLabels = ranges
}

// setWatermarkConfig configures a watermark with full control over styling.
//
// Takes config (WatermarkConfig) which specifies the watermark settings to apply.
func (painter *PdfPainter) setWatermarkConfig(config WatermarkConfig) {
	painter.watermark = &config
}

// setSVGWriter configures an optional SVG writer that renders SVGs as
// native PDF vector commands.
//
// Takes writer (SVGWriterPort) which specifies the SVG-to-PDF writer.
// Takes data (SVGDataPort) which provides raw SVG markup for sources.
func (painter *PdfPainter) setSVGWriter(writer SVGWriterPort, data SVGDataPort) {
	painter.svgWriter = writer
	painter.svgData = data
}

// enableTaggedPDF enables structure tagging.
func (painter *PdfPainter) enableTaggedPDF() {
	painter.structTree = NewStructTree()
}

// setPdfA configures PDF/A conformance output.
//
// Takes config (*PdfAConfig) which specifies the PDF/A conformance level and settings.
func (painter *PdfPainter) setPdfA(config *PdfAConfig) {
	painter.pdfaConfig = config
	if config != nil && config.Level == PdfA2A {
		if painter.structTree == nil {
			painter.enableTaggedPDF()
		}
	}
}

// setGlyphWidthFunc sets a function that computes variation-aware glyph
// advance widths in font design units.
//
// Takes fn (func) which specifies the glyph width computation function.
func (painter *PdfPainter) setGlyphWidthFunc(fn func(family string, weight int, style int, glyphID uint16) int) {
	painter.glyphWidthFunc = fn
}

// extractPageObjectNumbers returns the page object numbers from a slice
// of pageObj values.
//
// Takes objs ([]pageObj) which holds the page objects to extract numbers from.
//
// Returns []int which holds the page object numbers in order.
func extractPageObjectNumbers(objs []pageObj) []int {
	numbers := make([]int, len(objs))
	for i := range objs {
		numbers[i] = objs[i].pageNumber
	}
	return numbers
}

// writeAcroformObjects writes the AcroForm objects and
// merges any form widget references into the per-page
// annotation reference slice.
//
// Takes writer (*PdfDocumentWriter) which specifies the document writer.
// Takes pageObjNumbers ([]int) which holds the per-page object numbers.
// Takes pageAnnotRefs ([][]string) which holds the
// per-page annotation references to merge into.
// Takes pageCount (int) which specifies the total number of pages.
//
// Returns int which holds the AcroForm object number, or zero if no fields exist.
func (painter *PdfPainter) writeAcroformObjects(
	writer *PdfDocumentWriter,
	pageObjNumbers []int,
	pageAnnotRefs [][]string,
	pageCount int,
) int {
	if !painter.acroformBuilder.HasFields() {
		return 0
	}
	acroformNumber, formWidgetRefs := painter.acroformBuilder.WriteObjects(writer, pageObjNumbers)
	for pageIdx, refs := range formWidgetRefs {
		if pageIdx >= 0 && pageIdx < pageCount {
			pageAnnotRefs[pageIdx] = append(pageAnnotRefs[pageIdx], refs...)
		}
	}
	return acroformNumber
}

// prepareWatermark builds the watermark prefix stream and font resource.
//
// Takes writer (*PdfDocumentWriter) which specifies
// the document writer for font allocation.
//
// Returns string which holds the watermark content stream prefix.
// Returns string which holds the watermark font resource entry.
func (painter *PdfPainter) prepareWatermark(writer *PdfDocumentWriter) (prefix string, fontResource string) {
	if painter.watermark == nil || painter.watermark.Text == "" {
		return "", ""
	}
	painter.watermark.applyDefaults()
	wmGsName := painter.extGStateManager.RegisterOpacity(painter.watermark.Opacity)
	prefix = buildWatermarkStream(painter.watermark, "FW", wmGsName, painter.pageWidth, painter.pageHeight)
	wmFontNumber := writer.AllocateObject()
	writer.WriteObject(wmFontNumber,
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	fontResource = fmt.Sprintf(" /FW %s", FormatReference(wmFontNumber))
	return prefix, fontResource
}

// renderPageStreams paints all pages and returns their
// content streams.
//
// Takes rootBox (*layouter_domain.LayoutBox) which holds
// the root of the layout tree.
// Takes pageCount (int) which specifies the number of
// pages to render.
// Takes watermarkPrefix (string) which holds the watermark
// stream prefix prepended to each page.
//
// Returns []*ContentStream which holds the content streams
// for each page.
func (painter *PdfPainter) renderPageStreams(ctx context.Context, rootBox *layouter_domain.LayoutBox, pageCount int, watermarkPrefix string) []*ContentStream {
	streams := make([]*ContentStream, pageCount)
	for i := range streams {
		if ctx.Err() != nil {
			break
		}
		streams[i] = &ContentStream{}
		if watermarkPrefix != "" {
			streams[i].builder.WriteString(watermarkPrefix)
		}
		painter.basePageYOffset = float64(i) * painter.pageHeight
		painter.pageYOffset = painter.basePageYOffset
		painter.paintPageBoxes(ctx, streams[i], rootBox, i)
	}
	painter.pageYOffset = 0
	return streams
}

// writeFontResources writes font objects and returns the resource entries string.
//
// Takes writer (*PdfDocumentWriter) which specifies the document writer.
//
// Returns string which holds the font resource dictionary entries.
func (painter *PdfPainter) writeFontResources(writer *PdfDocumentWriter) string {
	if painter.fontEmbedder.HasFonts() {
		entries := painter.fontEmbedder.WriteObjects(writer)
		if entries != "" {
			return entries
		}
	}
	helveticaNumber := writer.AllocateObject()
	writer.WriteObject(helveticaNumber,
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	return fmt.Sprintf(" /F1 %s", FormatReference(helveticaNumber))
}

// buildResourcesDict assembles the PDF resources dictionary string.
//
// Takes writer (*PdfDocumentWriter) which specifies the document writer.
// Takes fontEntries (string) which holds the font resource entries.
// Takes watermarkFont (string) which holds the watermark font resource entry.
//
// Returns string which holds the assembled resources dictionary.
// Returns error when image embedding fails.
func (painter *PdfPainter) buildResourcesDict(writer *PdfDocumentWriter, fontEntries, watermarkFont string) (string, error) {
	resources := fmt.Sprintf("<< /Font <<%s%s >>", fontEntries, watermarkFont)

	if painter.extGStateManager.HasStates() {
		extGstateEntries := painter.extGStateManager.WriteObjects(writer)
		resources += fmt.Sprintf(" /ExtGState <<%s >>", extGstateEntries)
	}
	if painter.imageEmbedder.HasImages() {
		imageEntries, imageError := painter.imageEmbedder.WriteObjects(writer)
		if imageError != nil {
			return "", imageError
		}
		resources += fmt.Sprintf(" /XObject <<%s >>", imageEntries)
	}
	if painter.shadingManager.HasShadings() {
		shadingEntries := painter.shadingManager.WriteObjects(writer)
		resources += fmt.Sprintf(" /Shading <<%s >>", shadingEntries)
	}

	for _, mfo := range painter.maskFormObjects {
		formResources := ""
		if mfo.shadingName != "" {
			ref := painter.shadingManager.ShadingRef(mfo.shadingName)
			if ref != "" {
				formResources = fmt.Sprintf(" /Resources << /Shading << /%s %s >> >>", mfo.shadingName, ref)
			}
		}
		dict := fmt.Sprintf("/Type /XObject /Subtype /Form /BBox [%s] /Group << /Type /Group /S /Transparency /CS /DeviceGray >>%s",
			mfo.bbox, formResources)
		writer.WriteStreamObject(mfo.objectNumber, dict, mfo.content)
	}

	resources += pdfDictCloseSuffix
	return resources, nil
}

// writeAnnotations groups link annotations by page and writes annotation objects.
//
// Takes writer (*PdfDocumentWriter) which specifies the document writer.
// Takes pageCount (int) which specifies the total number of pages.
//
// Returns [][]string which holds per-page slices of annotation object references.
func (painter *PdfPainter) writeAnnotations(writer *PdfDocumentWriter, pageCount int) [][]string {
	pageAnnotRefs := make([][]string, pageCount)
	for _, annot := range painter.annotations {
		if annot.pageIndex < 0 || annot.pageIndex >= pageCount {
			continue
		}
		annotNumber := writer.AllocateObject()
		rect := fmt.Sprintf("[%s %s %s %s]",
			FormatNumber(annot.x1), FormatNumber(annot.y1),
			FormatNumber(annot.x2), FormatNumber(annot.y2))

		var annotDict string
		if annot.dest != "" {
			escapedDest := pdfEscapeString(annot.dest)
			annotDict = fmt.Sprintf(
				"<< /Type /Annot /Subtype /Link /Rect %s /Border [0 0 0] /A << /Type /Action /S /GoTo /D (%s) >> >>",
				rect, escapedDest)
		} else {
			escapedURI := pdfEscapeString(annot.uri)
			annotDict = fmt.Sprintf(
				"<< /Type /Annot /Subtype /Link /Rect %s /Border [0 0 0] /A << /Type /Action /S /URI /URI (%s) >> >>",
				rect, escapedURI)
		}
		writer.WriteObject(annotNumber, annotDict)
		pageAnnotRefs[annot.pageIndex] = append(pageAnnotRefs[annot.pageIndex], FormatReference(annotNumber))
	}
	return pageAnnotRefs
}

// writePageObjects writes the individual page objects to the PDF.
//
// Takes writer (*PdfDocumentWriter) which specifies the document writer.
// Takes pageObjs ([]pageObj) which holds the allocated page object numbers.
// Takes pagesNumber (int) which specifies the pages tree root object number.
// Takes resources (string) which holds the shared resources dictionary.
// Takes pageAnnotRefs ([][]string) which holds per-page annotation references.
func (painter *PdfPainter) writePageObjects(writer *PdfDocumentWriter, pageObjs []pageObj, pagesNumber int, resources string, pageAnnotRefs [][]string, _ int) {
	mediaBox := FormatArray(
		FormatNumber(0), FormatNumber(0),
		FormatNumber(painter.pageWidth), FormatNumber(painter.pageHeight))

	for i, po := range pageObjs {
		pageDict := fmt.Sprintf(
			"<< /Type /Page /Parent %s /MediaBox %s /Resources %s /Contents %s",
			FormatReference(pagesNumber), mediaBox, resources,
			FormatReference(pageObjs[i].contentNumber))
		if len(pageAnnotRefs[i]) > 0 {
			pageDict += fmt.Sprintf(" /Annots [%s]", strings.Join(pageAnnotRefs[i], " "))
		}
		if painter.structTree != nil && !painter.structTree.IsEmpty() {
			pageDict += fmt.Sprintf(" /StructParents %d", i)
		}
		pageDict += pdfDictCloseSuffix
		writer.WriteObject(po.pageNumber, pageDict)
	}
}

// writeStructTree writes the structure tree for tagged PDF if present.
//
// Takes writer (*PdfDocumentWriter) which specifies the document writer.
// Takes pageObjNumbers ([]int) which holds the per-page object numbers.
//
// Returns int which holds the structure tree root object number, or zero if none.
func (painter *PdfPainter) writeStructTree(writer *PdfDocumentWriter, pageObjNumbers []int) int {
	if painter.structTree != nil && !painter.structTree.IsEmpty() {
		return painter.structTree.WriteObjects(writer, pageObjNumbers)
	}
	return 0
}

// buildCatalogueDict assembles the PDF catalogue
// dictionary string.
//
// Takes pagesNumber (int) which specifies the pages tree
// root object number.
// Takes outlineRootNumber (int) which specifies the
// outline root object number, or zero if none.
// Takes structTreeRootNumber (int) which specifies the
// structure tree root object number, or zero if none.
// Takes acroformNumber (int) which specifies the AcroForm
// object number, or zero if none.
// Takes pageObjNumbers ([]int) which holds the per-page
// object numbers.
// Takes writer (*PdfDocumentWriter) which specifies the
// document writer.
//
// Returns string which holds the assembled catalogue
// dictionary.
func (painter *PdfPainter) buildCatalogueDict(pagesNumber, outlineRootNumber, structTreeRootNumber, acroformNumber int, pageObjNumbers []int, writer *PdfDocumentWriter) string {
	catalogueDict := fmt.Sprintf("<< /Type /Catalog /Pages %s", FormatReference(pagesNumber))
	if outlineRootNumber > 0 {
		catalogueDict += fmt.Sprintf(" /Outlines %s", FormatReference(outlineRootNumber))
	}
	if structTreeRootNumber > 0 {
		catalogueDict += fmt.Sprintf(" /StructTreeRoot %s /MarkInfo << /Marked true >>",
			FormatReference(structTreeRootNumber))
	}
	if acroformNumber > 0 {
		catalogueDict += fmt.Sprintf(" /AcroForm %s", FormatReference(acroformNumber))
	}
	catalogueDict += buildViewerPreferencesDict(painter.viewerPrefs, writer)
	catalogueDict += buildPageLabelsDict(painter.pageLabels, writer)
	catalogueDict += painter.buildNamedDestsDict(writer, pageObjNumbers)
	if painter.pdfaConfig != nil {
		catalogueDict += writePdfAObjects(writer, painter.pdfaConfig, painter.metadata, time.Now())
	}
	catalogueDict += pdfDictCloseSuffix
	return catalogueDict
}

// isVariableFont reports whether the given font key is a variable font instance.
//
// Takes key (pdfFontKey) which specifies the font to check.
//
// Returns bool which indicates whether the font is a variable font instance.
func (painter *PdfPainter) isVariableFont(key pdfFontKey) bool {
	return painter.variableFonts != nil && painter.variableFonts[key]
}

// glyphAdvanceWidth returns the glyph advance width in font design units.
//
// Takes key (pdfFontKey) which specifies the font to look up.
// Takes fontData ([]byte) which holds the raw font data for hmtx fallback.
// Takes glyphID (uint16) which specifies the glyph to measure.
//
// Returns int which holds the advance width in font design units.
func (painter *PdfPainter) glyphAdvanceWidth(key pdfFontKey, fontData []byte, glyphID uint16) int {
	if painter.isVariableFont(key) && painter.glyphWidthFunc != nil {
		return painter.glyphWidthFunc(key.family, key.weight, key.style, glyphID)
	}
	return GlyphAdvanceWidth(fontData, glyphID)
}

// structTreeParent returns the current parent node in the structure tree stack.
//
// Returns *StructNode which holds the current parent structure node.
func (painter *PdfPainter) structTreeParent() *StructNode {
	if len(painter.structStack) > 0 {
		return painter.structStack[len(painter.structStack)-1]
	}
	return painter.structTree.root
}

// resolvedFont holds the result of a font data lookup.
type resolvedFont struct {
	// data holds the raw font bytes for the resolved font.
	data []byte

	// key holds the font key that was matched.
	key pdfFontKey

	// found indicates whether a font was successfully resolved.
	found bool
}

// resolveFontData looks up font data for the given family, weight, and
// style combination.
//
// Takes family (string) which specifies the font family name.
// Takes weight (int) which specifies the CSS font weight.
// Takes style (int) which specifies the CSS font style.
//
// Returns resolvedFont which holds the matched font data and key.
func (painter *PdfPainter) resolveFontData(family string, weight int, style int) resolvedFont {
	if resolved, ok := painter.tryExactFontMatch(family, weight, style); ok {
		return resolved
	}
	if resolved, ok := painter.tryCrossFamilyFontMatch(weight, style); ok {
		return resolved
	}
	return painter.fallbackToFirstFont()
}

// tryExactFontMatch attempts to find font data within the requested family.
//
// Takes family (string) which specifies the font family name.
// Takes weight (int) which specifies the CSS font weight.
// Takes style (int) which specifies the CSS font style.
//
// Returns resolvedFont which holds the matched font data if found.
// Returns bool which indicates whether a match was found.
func (painter *PdfPainter) tryExactFontMatch(family string, weight int, style int) (resolvedFont, bool) {
	exact := pdfFontKey{family: family, weight: weight, style: style}
	if data, ok := painter.fontDataMap[exact]; ok {
		return resolvedFont{data: data, key: exact, found: true}, true
	}
	if style != 0 {
		key := pdfFontKey{family: family, weight: weight, style: 0}
		if data, ok := painter.fontDataMap[key]; ok {
			return resolvedFont{data: data, key: key, found: true}, true
		}
	}
	if weight != fontWeightNormal {
		key := pdfFontKey{family: family, weight: fontWeightNormal, style: style}
		if data, ok := painter.fontDataMap[key]; ok {
			return resolvedFont{data: data, key: key, found: true}, true
		}
		if style != 0 {
			key := pdfFontKey{family: family, weight: fontWeightNormal, style: 0}
			if data, ok := painter.fontDataMap[key]; ok {
				return resolvedFont{data: data, key: key, found: true}, true
			}
		}
	}
	return resolvedFont{}, false
}

// tryCrossFamilyFontMatch searches across all registered families.
//
// Takes weight (int) which specifies the CSS font weight.
// Takes style (int) which specifies the CSS font style.
//
// Returns resolvedFont which holds the matched font data if found.
// Returns bool which indicates whether a match was found.
func (painter *PdfPainter) tryCrossFamilyFontMatch(weight int, style int) (resolvedFont, bool) {
	for key, data := range painter.fontDataMap {
		if key.weight == weight && key.style == style {
			return resolvedFont{data: data, key: key, found: true}, true
		}
	}
	for key, data := range painter.fontDataMap {
		if key.weight == weight && key.style == 0 {
			return resolvedFont{data: data, key: key, found: true}, true
		}
	}
	return resolvedFont{}, false
}

// fallbackToFirstFont returns the first registered font entry, if any.
//
// Returns resolvedFont which holds the first font entry, or an empty result.
func (painter *PdfPainter) fallbackToFirstFont() resolvedFont {
	if len(painter.fontEntries) > 0 {
		entry := painter.fontEntries[0]
		key := pdfFontKey{family: entry.Family, weight: entry.Weight, style: entry.Style}
		return resolvedFont{data: entry.Data, key: key, found: true}
	}
	return resolvedFont{}
}

// fontInstanceKey generates a unique instance key for a resolved font.
//
// Takes key (pdfFontKey) which specifies the font to generate a key for.
//
// Returns string which holds the formatted instance key.
func fontInstanceKey(key pdfFontKey) string {
	return fmt.Sprintf("%s:%d:%d", key.family, key.weight, key.style)
}

// needsSyntheticBold reports whether synthetic bold strokes are needed.
//
// Takes requested (pdfFontKey) which specifies the requested font key.
// Takes resolved (pdfFontKey) which specifies the actually resolved font key.
//
// Returns bool which indicates whether the resolved font lacks bold weight.
func needsSyntheticBold(requested pdfFontKey, resolved pdfFontKey) bool {
	return requested.weight >= fontWeightBold && resolved.weight < fontWeightBold
}

// needsSyntheticItalic reports whether synthetic italic skew is needed.
//
// Takes requested (pdfFontKey) which specifies the requested font key.
// Takes resolved (pdfFontKey) which specifies the actually resolved font key.
//
// Returns bool which indicates whether the resolved font lacks italic style.
func needsSyntheticItalic(requested pdfFontKey, resolved pdfFontKey) bool {
	return requested.style == int(layouter_domain.FontStyleItalic) && resolved.style == 0
}

// paintPageBoxes recursively paints boxes belonging to the given page.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box to paint.
// Takes pageIndex (int) which specifies the zero-based page index to paint.
func (painter *PdfPainter) paintPageBoxes(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox, pageIndex int) {
	if ctx.Err() != nil {
		return
	}
	if box.Type == layouter_domain.BoxNone {
		return
	}
	if box.PageIndex == pageIndex {
		painter.paintBoxToStream(ctx, stream, box)
		return
	}
	for _, child := range box.Children {
		painter.paintPageBoxes(ctx, stream, child, pageIndex)
	}
}

// paintBoxToStream renders a single layout box and its children.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box to render.
func (painter *PdfPainter) paintBoxToStream(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox) {
	if box.Type == layouter_domain.BoxNone {
		return
	}

	savedOffset := painter.pageYOffset
	painter.pageYOffset = painter.basePageYOffset - box.PageYOffset
	defer func() { painter.pageYOffset = savedOffset }()

	states := painter.applyBoxStates(stream, box)
	painter.paintBoxContent(ctx, stream, box, states)
	painter.restoreBoxStates(stream, states)
}

// boxRenderStates tracks which graphical states were applied around a box.
type boxRenderStates struct {
	// hasOverflowClip indicates whether an overflow clip was applied.
	hasOverflowClip bool

	// hasOpacity indicates whether an opacity graphics state was applied.
	hasOpacity bool

	// hasBlendMode indicates whether a blend mode graphics state was applied.
	hasBlendMode bool

	// hasMask indicates whether a mask image was applied.
	hasMask bool

	// hasFilterOpacity indicates whether a CSS filter opacity was applied.
	hasFilterOpacity bool

	// hasClipPath indicates whether a CSS clip-path was applied.
	hasClipPath bool

	// hasTransform indicates whether a CSS transform was applied.
	hasTransform bool

	// hasStructTag indicates whether a structure tag was opened.
	hasStructTag bool
}

// applyBoxStates applies graphical states for the box.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box to apply states for.
//
// Returns boxRenderStates which tracks which states were applied for later restoration.
func (painter *PdfPainter) applyBoxStates(stream *ContentStream, box *layouter_domain.LayoutBox) boxRenderStates {
	var s boxRenderStates
	s.hasOverflowClip = box.Style.OverflowX == layouter_domain.OverflowHidden ||
		box.Style.OverflowY == layouter_domain.OverflowHidden

	if box.Style.Opacity < 1.0 && box.Style.Opacity > 0 {
		s.hasOpacity = true
		stream.SaveState()
		name := painter.extGStateManager.RegisterOpacity(box.Style.Opacity)
		stream.SetExtGState(name)
	}
	if box.Style.MixBlendMode != layouter_domain.BlendModeNormal {
		s.hasBlendMode = true
		stream.SaveState()
		name := painter.extGStateManager.RegisterBlendMode(box.Style.MixBlendMode.String())
		stream.SetExtGState(name)
	}
	if box.Style.MaskImage != "" && box.Style.MaskImage != "none" {
		s.hasMask = painter.applyMaskImage(stream, box)
	}

	s.hasFilterOpacity = painter.applyFilterOpacity(stream, box)
	s.hasClipPath = painter.applyClipPath(stream, box)
	s.hasTransform = painter.applyTransform(stream, box)
	s.hasStructTag = painter.applyStructTag(stream, box)
	return s
}

// applyFilterOpacity applies CSS filter: opacity() via ExtGState.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box with filter styles.
//
// Returns bool which indicates whether an opacity filter was applied.
func (painter *PdfPainter) applyFilterOpacity(stream *ContentStream, box *layouter_domain.LayoutBox) bool {
	for _, f := range box.Style.Filter {
		if f.Function == layouter_domain.FilterOpacity && f.Amount < 1.0 {
			stream.SaveState()
			name := painter.extGStateManager.RegisterOpacity(f.Amount)
			stream.SetExtGState(name)
			return true
		}
	}
	return false
}

// applyClipPath applies a CSS clip-path if present.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box with clip-path styles.
//
// Returns bool which indicates whether a clip path was applied.
func (painter *PdfPainter) applyClipPath(stream *ContentStream, box *layouter_domain.LayoutBox) bool {
	if box.Style.ClipPath == "" || box.Style.ClipPath == "none" {
		return false
	}
	clipShape := ParseClipPath(box.Style.ClipPath, box.BorderBoxWidth(), box.BorderBoxHeight())
	if clipShape.Type == ClipShapeNone {
		return false
	}
	stream.SaveState()
	pdfClipX := box.BorderBoxX()
	pdfClipY := painter.pageHeight + painter.pageYOffset - box.BorderBoxY() - box.BorderBoxHeight()
	EmitClipPath(stream, clipShape, pdfClipX, pdfClipY, box.BorderBoxWidth(), box.BorderBoxHeight())
	stream.ClipNonZero()
	return true
}

// applyTransform applies a CSS transform if present.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box with transform styles.
//
// Returns bool which indicates whether a transform was applied.
func (painter *PdfPainter) applyTransform(stream *ContentStream, box *layouter_domain.LayoutBox) bool {
	if !box.Style.HasTransform || box.Style.TransformValue == "" {
		return false
	}
	stream.SaveState()
	originX, originY := painter.resolveTransformOrigin(box)
	stream.ConcatMatrix(1, 0, 0, 1, originX, originY)
	m, _ := ParseCSSTransform(box.Style.TransformValue)
	stream.ConcatMatrix(m.a, -m.b, -m.c, m.d, m.e, -m.f)
	stream.ConcatMatrix(1, 0, 0, 1, -originX, -originY)
	return true
}

// applyStructTag wraps the element in a marked content sequence for tagged PDF.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box with source node metadata.
//
// Returns bool which indicates whether a structure tag was opened.
func (painter *PdfPainter) applyStructTag(stream *ContentStream, box *layouter_domain.LayoutBox) bool {
	if painter.structTree == nil || box.SourceNode == nil {
		return false
	}
	tag := MapHTMLToStructTag(box.SourceNode.TagName)
	if tag == "" {
		return false
	}
	parent := painter.structTreeParent()
	node := painter.structTree.AddChild(parent, tag)

	if tag == TagFigure {
		for i := range box.SourceNode.Attributes {
			if box.SourceNode.Attributes[i].Name == "alt" {
				node.altText = box.SourceNode.Attributes[i].Value
				break
			}
		}
	}

	mcid := painter.structTree.MarkContent(node, box.PageIndex)
	stream.BeginMarkedContent(string(tag), mcid)
	painter.structStack = append(painter.structStack, node)
	return true
}

// paintBoxContent paints the visible content and children of a box.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes box (*layouter_domain.LayoutBox) which holds the box to paint.
// Takes states (boxRenderStates) which tracks the applied graphical states.
func (painter *PdfPainter) paintBoxContent(ctx context.Context, stream *ContentStream, box *layouter_domain.LayoutBox, states boxRenderStates) {
	isVisible := box.Style.Visibility == layouter_domain.VisibilityVisible

	if isVisible {
		painter.paintOuterBoxShadows(stream, box)
		painter.paintBackground(ctx, stream, box)
		painter.paintInsetBoxShadows(stream, box)
		painter.paintBorders(stream, box)
		painter.paintBorderImage(ctx, stream, box)
		painter.paintImage(ctx, stream, box)
		painter.paintTextShadows(stream, box)
		painter.paintText(stream, box)
		painter.paintTextDecorations(stream, box)
		painter.paintOutline(stream, box)
		painter.paintColumnRules(stream, box)
		painter.collectLinkAnnotation(box)
		painter.collectNamedDestination(box)
		painter.collectOutlineEntry(box)
		painter.paintFormVisual(stream, box)
		painter.collectFormField(box)
	}

	if states.hasOverflowClip {
		stream.SaveState()
		painter.emitOverflowClip(stream, box)
	}

	skipChildren := box.SourceNode != nil && isEditableFormElement(box.SourceNode.TagName)
	if !skipChildren {
		for _, child := range box.Children {
			painter.paintBoxToStream(ctx, stream, child)
		}
	}

	if states.hasStructTag {
		stream.EndMarkedContent()
		painter.structStack = painter.structStack[:len(painter.structStack)-1]
	}
}

// restoreBoxStates undoes the graphical states applied by applyBoxStates.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes states (boxRenderStates) which tracks which states need restoring.
func (*PdfPainter) restoreBoxStates(stream *ContentStream, states boxRenderStates) {
	if states.hasTransform {
		stream.RestoreState()
	}
	if states.hasFilterOpacity {
		stream.RestoreState()
	}
	if states.hasClipPath {
		stream.RestoreState()
	}
	if states.hasMask {
		stream.RestoreState()
	}
	if states.hasBlendMode {
		stream.RestoreState()
	}
	if states.hasOpacity {
		stream.RestoreState()
	}
	if states.hasOverflowClip {
		stream.RestoreState()
	}
}

// setFillColour sets the fill colour on the content stream.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes colour (layouter_domain.Colour) which specifies the fill colour to set.
func (*PdfPainter) setFillColour(stream *ContentStream, colour layouter_domain.Colour) {
	switch colour.Space {
	case layouter_domain.ColourSpaceGrey:
		stream.SetFillColourGrey(colour.Red)
	case layouter_domain.ColourSpaceCMYK:
		stream.SetFillColourCMYK(colour.Cyan, colour.Magenta, colour.Yellow, colour.Key)
	default:
		stream.SetFillColourRGB(colour.Red, colour.Green, colour.Blue)
	}
}

// setStrokeColour sets the stroke colour on the content stream.
//
// Takes stream (*ContentStream) which holds the page content stream.
// Takes colour (layouter_domain.Colour) which specifies the stroke colour to set.
func (*PdfPainter) setStrokeColour(stream *ContentStream, colour layouter_domain.Colour) {
	switch colour.Space {
	case layouter_domain.ColourSpaceGrey:
		stream.SetStrokeColourGrey(colour.Red)
	case layouter_domain.ColourSpaceCMYK:
		stream.SetStrokeColourCMYK(colour.Cyan, colour.Magenta, colour.Yellow, colour.Key)
	default:
		stream.SetStrokeColourRGB(colour.Red, colour.Green, colour.Blue)
	}
}

// darkenColour returns a darker variant of the colour.
//
// Takes c (layouter_domain.Colour) which specifies the base colour.
// Takes factor (float64) which specifies the darkening factor between 0 and 1.
//
// Returns layouter_domain.Colour which holds the darkened colour.
func darkenColour(c layouter_domain.Colour, factor float64) layouter_domain.Colour {
	return layouter_domain.Colour{
		Red: c.Red * factor, Green: c.Green * factor, Blue: c.Blue * factor,
		Cyan: c.Cyan, Magenta: c.Magenta, Yellow: c.Yellow, Key: c.Key,
		Alpha: c.Alpha, Space: c.Space,
	}
}

// lightenColour returns a lighter variant of the colour.
//
// Takes c (layouter_domain.Colour) which specifies the base colour.
// Takes factor (float64) which specifies the lightening factor between 0 and 1.
//
// Returns layouter_domain.Colour which holds the lightened colour.
func lightenColour(c layouter_domain.Colour, factor float64) layouter_domain.Colour {
	return layouter_domain.Colour{
		Red: c.Red + (1-c.Red)*factor, Green: c.Green + (1-c.Green)*factor, Blue: c.Blue + (1-c.Blue)*factor,
		Cyan: c.Cyan, Magenta: c.Magenta, Yellow: c.Yellow, Key: c.Key,
		Alpha: c.Alpha, Space: c.Space,
	}
}

// hasAnyBorderRadius reports whether the box has any non-zero border radius.
//
// Takes box (*layouter_domain.LayoutBox) which specifies the box to check.
//
// Returns bool which indicates whether any corner has a non-zero radius.
func (*PdfPainter) hasAnyBorderRadius(box *layouter_domain.LayoutBox) bool {
	return box.Style.BorderTopLeftRadius > 0 ||
		box.Style.BorderTopRightRadius > 0 ||
		box.Style.BorderBottomRightRadius > 0 ||
		box.Style.BorderBottomLeftRadius > 0
}
