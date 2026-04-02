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

package driven_transform_pdfa

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the PDF/A transformer.
	defaultPriority = 200

	// defaultLevel is the PDF/A conformance level used when the option is
	// empty.
	defaultLevel = "1b"

	// outputIntentSubtype is the PDF output intent subtype for PDF/A.
	outputIntentSubtype = "GTS_PDFA1"

	// sRGBProfileName is the human-readable name for the sRGB colour
	// profile used in the output intent.
	sRGBProfileName = "sRGB IEC61966-2.1"

	// xmpPacketBegin is the XML processing instruction that opens an XMP
	// packet.
	xmpPacketBegin = "<?xpacket begin=\"\xEF\xBB\xBF\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>"

	// xmpPacketEnd is the XML processing instruction that closes an XMP
	// packet.
	xmpPacketEnd = "<?xpacket end=\"w\"?>"
)

// PdfATransformer converts a PDF document to conform to a specified PDF/A
// archival standard level. It adds XMP metadata declaring the conformance
// level, inserts an sRGB output intent, and removes features that are
// prohibited under the target level (additional actions, JavaScript, and
// transparency groups for PDF/A-1b).
//
// Supported levels are "1b" (PDF/A-1b), "2b" (PDF/A-2b), and "3b"
// (PDF/A-3b). An empty level defaults to "1b".
type PdfATransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*PdfATransformer)(nil)

// New creates a new PDF/A transformer with default name and priority.
//
// Returns *PdfATransformer which is ready for use.
func New() *PdfATransformer {
	return &PdfATransformer{
		name:     "pdfa-convert",
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *PdfATransformer) Name() string { return t.name }

// Type returns TransformerCompliance.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a
// compliance transformer.
func (*PdfATransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerCompliance
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *PdfATransformer) Priority() int { return t.priority }

// Transform applies PDF/A conformance modifications to the PDF. Options
// must be PdfAOptions or *PdfAOptions.
//
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be PdfAOptions or *PdfAOptions.
//
// Returns []byte which is the PDF/A-conformant PDF.
// Returns error when the PDF cannot be parsed or the conversion fails.
func (*PdfATransformer) Transform(_ context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}

	level := opts.Level
	if level == "" {
		level = defaultLevel
	}

	part, conformance, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("pdfa: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("pdfa: creating writer: %w", err)
	}

	catalogObjNum, catalogDict, err := getCatalog(writer)
	if err != nil {
		return nil, fmt.Errorf("pdfa: %w", err)
	}

	addXMPMetadata(writer, &catalogDict, part, conformance)
	addOutputIntent(writer, &catalogDict)
	removeProhibitedFromCatalog(&catalogDict)

	pageRefs, err := collectPageRefs(doc)
	if err != nil {
		return nil, fmt.Errorf("pdfa: collecting pages: %w", err)
	}

	removePagesProhibited(writer, pageRefs, level)

	writer.SetObject(catalogObjNum, pdfparse.DictObj(catalogDict))

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("pdfa: writing PDF: %w", err)
	}
	return output, nil
}

// castOptions extracts PdfAOptions from the generic options.
//
// Takes options (any) which is the untyped options value to assert.
//
// Returns pdfwriter_dto.PdfAOptions which holds the typed options.
// Returns error when the options type does not match.
func castOptions(options any) (pdfwriter_dto.PdfAOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.PdfAOptions:
		return v, nil
	case *pdfwriter_dto.PdfAOptions:
		if v == nil {
			return pdfwriter_dto.PdfAOptions{}, errors.New("pdfa: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.PdfAOptions{}, fmt.Errorf("pdfa: expected PdfAOptions, got %T", options)
	}
}

// parseLevel splits a level string like "1b" into a numeric part and a
// conformance letter.
//
// Takes level (string) which specifies the PDF/A level such as "1b", "2b",
// or "3b".
//
// Returns part (string) which is the numeric part of the level.
// Returns conformance (string) which is the uppercase conformance letter.
// Returns err (error) when the level is not recognised.
func parseLevel(level string) (part string, conformance string, err error) {
	switch level {
	case "1b":
		return "1", "B", nil
	case "2b":
		return "2", "B", nil
	case "3b":
		return "3", "B", nil
	default:
		return "", "", fmt.Errorf("pdfa: unsupported level %q", level)
	}
}

// getCatalog locates the document catalog object from the writer's trailer
// and returns its object number and dictionary.
//
// Takes writer (*pdfparse.Writer) which provides access to the PDF objects.
//
// Returns int which is the catalog object number.
// Returns pdfparse.Dict which is the catalog dictionary.
// Returns error when the catalog cannot be located or is not a dictionary.
func getCatalog(writer *pdfparse.Writer) (int, pdfparse.Dict, error) {
	trailer := writer.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return 0, pdfparse.Dict{}, errors.New("no /Root in trailer")
	}

	catalogObj := writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return 0, pdfparse.Dict{}, errors.New("catalog is not a dictionary")
	}

	return rootRef.Number, catalogDict, nil
}

// addXMPMetadata creates an XMP metadata stream declaring PDF/A
// conformance and sets the catalog's /Metadata entry to reference it.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer for adding objects.
// Takes catalogDict (*pdfparse.Dict) which is the catalog dictionary to
// update with the metadata reference.
// Takes part (string) which is the numeric PDF/A part.
// Takes conformance (string) which is the uppercase conformance letter.
func addXMPMetadata(writer *pdfparse.Writer, catalogDict *pdfparse.Dict, part, conformance string) {
	xmp := buildXMPPacket(part, conformance)

	metaDict := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Metadata")},
		{Key: "Subtype", Value: pdfparse.Name("XML")},
	}}

	metaObjNum := writer.AddObject(pdfparse.StreamObj(metaDict, []byte(xmp)))
	catalogDict.Set("Metadata", pdfparse.RefObj(metaObjNum, 0))
}

// buildXMPPacket constructs the XMP XML payload with the pdfaid namespace
// declaring the given part and conformance values.
//
// Takes part (string) which is the numeric PDF/A part.
// Takes conformance (string) which is the uppercase conformance letter.
//
// Returns string which holds the complete XMP packet XML.
func buildXMPPacket(part, conformance string) string {
	return xmpPacketBegin + "\n" +
		"<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n" +
		"  <rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n" +
		"    <rdf:Description rdf:about=\"\"\n" +
		"      xmlns:pdfaid=\"http://www.aiim.org/pdfa/ns/id/\">\n" +
		"      <pdfaid:part>" + part + "</pdfaid:part>\n" +
		"      <pdfaid:conformance>" + conformance + "</pdfaid:conformance>\n" +
		"    </rdf:Description>\n" +
		"  </rdf:RDF>\n" +
		"</x:xmpmeta>\n" +
		xmpPacketEnd
}

// addOutputIntent adds an sRGB output intent to the catalog's
// /OutputIntents array if one is not already present.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer for adding objects.
// Takes catalogDict (*pdfparse.Dict) which is the catalog dictionary to
// update with the output intent.
func addOutputIntent(writer *pdfparse.Writer, catalogDict *pdfparse.Dict) {
	intentDict := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("OutputIntent")},
		{Key: "S", Value: pdfparse.Name(outputIntentSubtype)},
		{Key: "OutputConditionIdentifier", Value: pdfparse.Str(sRGBProfileName)},
		{Key: "RegistryName", Value: pdfparse.Str("http://www.color.org")},
		{Key: "Info", Value: pdfparse.Str(sRGBProfileName)},
	}}

	intentObjNum := writer.AddObject(pdfparse.DictObj(intentDict))
	catalogDict.Set("OutputIntents", pdfparse.Arr(pdfparse.RefObj(intentObjNum, 0)))
}

// removeProhibitedFromCatalog removes entries from the catalog that are
// prohibited under all PDF/A levels: /AA (additional actions) and
// /JavaScript from the /Names dictionary.
//
// Takes catalogDict (*pdfparse.Dict) which is the catalog dictionary to
// sanitise.
func removeProhibitedFromCatalog(catalogDict *pdfparse.Dict) {
	catalogDict.Remove("AA")

	namesObj := catalogDict.Get("Names")
	if namesObj.Type == pdfparse.ObjectDictionary {
		namesDict, ok := namesObj.Value.(pdfparse.Dict)
		if ok && namesDict.Has("JavaScript") {
			namesDict.Remove("JavaScript")
			if len(namesDict.Pairs) == 0 {
				catalogDict.Remove("Names")
			} else {
				catalogDict.Set("Names", pdfparse.DictObj(namesDict))
			}
		}
	}
}

// removePagesProhibited removes prohibited entries from each page dictionary.
//
// For PDF/A-1b, this includes /AA (additional actions) and transparency groups
// (/Group with /S /Transparency). For levels 2b and 3b, only /AA is removed
// since transparency is permitted.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer for mutations.
// Takes pageRefs ([]int) which holds the page object numbers to process.
// Takes level (string) which specifies the target PDF/A conformance level.
func removePagesProhibited(writer *pdfparse.Writer, pageRefs []int, level string) {
	for _, pageObjNum := range pageRefs {
		pageObj := writer.GetObject(pageObjNum)
		pageDict, ok := pageObj.Value.(pdfparse.Dict)
		if !ok {
			continue
		}

		changed := pageDict.Remove("AA")

		if level == defaultLevel {
			if removeTransparencyGroup(&pageDict) {
				changed = true
			}
		}

		if changed {
			writer.SetObject(pageObjNum, pdfparse.DictObj(pageDict))
		}
	}
}

// removeTransparencyGroup removes a /Group entry from a page dictionary
// if it has /S /Transparency.
//
// Takes pageDict (*pdfparse.Dict) which is the page dictionary to inspect.
//
// Returns bool which indicates whether the entry was removed.
func removeTransparencyGroup(pageDict *pdfparse.Dict) bool {
	groupObj := pageDict.Get("Group")
	if groupObj.Type != pdfparse.ObjectDictionary {
		return false
	}

	groupDict, ok := groupObj.Value.(pdfparse.Dict)
	if !ok {
		return false
	}

	if groupDict.GetName("S") == "Transparency" {
		return pageDict.Remove("Group")
	}

	return false
}

// collectPageRefs walks the page tree and returns object numbers for all
// leaf Page objects in document order.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
//
// Returns []int which holds the object numbers of leaf Page objects.
// Returns error when the page tree cannot be traversed.
func collectPageRefs(doc *pdfparse.Document) ([]int, error) {
	trailer := doc.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return nil, errors.New("no /Root in trailer")
	}

	catalog, err := doc.GetObject(rootRef.Number)
	if err != nil {
		return nil, fmt.Errorf("reading catalog: %w", err)
	}

	catalogDict, ok := catalog.Value.(pdfparse.Dict)
	if !ok {
		return nil, errors.New("catalog is not a dictionary")
	}

	pagesRef := catalogDict.GetRef("Pages")
	if pagesRef.Number == 0 {
		return nil, errors.New("no /Pages in catalog")
	}

	return walkPageTree(doc, pagesRef.Number)
}

// walkPageTree recursively collects leaf Page object numbers from a Pages
// tree node.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes objNum (int) which is the object number of the current tree node.
//
// Returns []int which holds the collected leaf Page object numbers.
// Returns error when a node cannot be read or is not a dictionary.
func walkPageTree(doc *pdfparse.Document, objNum int) ([]int, error) {
	obj, err := doc.GetObject(objNum)
	if err != nil {
		return nil, err
	}

	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return nil, fmt.Errorf("object %d is not a dictionary", objNum)
	}

	nodeType := dict.GetName("Type")
	if nodeType == "Page" {
		return []int{objNum}, nil
	}

	kids := dict.GetArray("Kids")
	var pages []int
	for _, kid := range kids {
		ref, ok := kid.Value.(pdfparse.Ref)
		if !ok {
			continue
		}
		childPages, err := walkPageTree(doc, ref.Number)
		if err != nil {
			return nil, err
		}
		pages = append(pages, childPages...)
	}
	return pages, nil
}
