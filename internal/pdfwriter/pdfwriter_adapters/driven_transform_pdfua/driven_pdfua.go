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

package driven_transform_pdfua

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the PDF/UA transformer.
	defaultPriority = 210

	// defaultLang is the default document language tag (BCP 47).
	defaultLang = "en"

	// catalogKey is the PDF dictionary key for the document catalog
	// reference in the trailer.
	catalogKey = "Root"

	// markInfoKey is the PDF dictionary key for the mark information
	// dictionary.
	markInfoKey = "MarkInfo"

	// structTreeRootKey is the PDF dictionary key for the structure tree
	// root.
	structTreeRootKey = "StructTreeRoot"

	// langKey is the PDF dictionary key for the document language.
	langKey = "Lang"

	// viewerPreferencesKey is the PDF dictionary key for viewer
	// preferences.
	viewerPreferencesKey = "ViewerPreferences"

	// infoKey is the PDF dictionary key for the document information
	// dictionary reference in the trailer.
	infoKey = "Info"
)

// PdfUATransformer adds PDF/UA (Universal Accessibility) metadata to a
// PDF document.
//
// It parses the PDF, locates the document catalog, and ensures that the
// required structural entries are present: MarkInfo, StructTreeRoot, Lang,
// and ViewerPreferences. All enhancements are additive, so existing entries
// are preserved.
type PdfUATransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*PdfUATransformer)(nil)

// New creates a new PDF/UA enhancement transformer with default name and
// priority.
//
// Returns *PdfUATransformer which is ready for use.
func New() *PdfUATransformer {
	return &PdfUATransformer{
		name:     "pdfua-enhance",
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *PdfUATransformer) Name() string { return t.name }

// Type returns TransformerCompliance.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a
// compliance transformer.
func (*PdfUATransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerCompliance
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *PdfUATransformer) Priority() int { return t.priority }

// Transform applies PDF/UA enhancements to the document catalog. Options
// must be PdfUAOptions or *PdfUAOptions.
//
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be PdfUAOptions or *PdfUAOptions.
//
// Returns []byte which is the enhanced PDF.
// Returns error when the PDF cannot be parsed or the enhancements cannot
// be applied.
func (*PdfUATransformer) Transform(_ context.Context, pdf []byte, options any) ([]byte, error) {
	if _, err := castOptions(options); err != nil {
		return nil, err
	}

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("pdfua: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("pdfua: creating writer: %w", err)
	}

	catalogObjNum, catalogDict, err := getCatalog(writer)
	if err != nil {
		return nil, fmt.Errorf("pdfua: %w", err)
	}

	modified := false
	modified = ensureMarkInfo(&catalogDict) || modified
	modified = ensureStructTreeRoot(writer, &catalogDict) || modified
	modified = ensureLang(&catalogDict) || modified
	modified = ensureViewerPreferences(&catalogDict) || modified
	modified = ensureInfoTitle(writer) || modified

	if modified {
		writer.SetObject(catalogObjNum, pdfparse.DictObj(catalogDict))
	}

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("pdfua: writing PDF: %w", err)
	}
	return output, nil
}

// castOptions extracts PdfUAOptions from the generic options.
//
// Takes options (any) which is the untyped options value to assert.
//
// Returns pdfwriter_dto.PdfUAOptions which holds the typed options.
// Returns error when the options type does not match.
func castOptions(options any) (pdfwriter_dto.PdfUAOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.PdfUAOptions:
		return v, nil
	case *pdfwriter_dto.PdfUAOptions:
		if v == nil {
			return pdfwriter_dto.PdfUAOptions{}, errors.New("pdfua: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.PdfUAOptions{}, fmt.Errorf("pdfua: expected PdfUAOptions, got %T", options)
	}
}

// getCatalog resolves the document catalog dictionary from the writer's
// trailer.
//
// Takes writer (*pdfparse.Writer) which provides access to the PDF objects.
//
// Returns int which is the catalog object number.
// Returns pdfparse.Dict which is the catalog dictionary.
// Returns error when the catalog cannot be located or is not a dictionary.
func getCatalog(writer *pdfparse.Writer) (int, pdfparse.Dict, error) {
	trailer := writer.Trailer()
	rootRef := trailer.GetRef(catalogKey)
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

// ensureMarkInfo adds /MarkInfo << /Marked true >> to the catalog if the
// key is absent.
//
// Takes catalogDict (*pdfparse.Dict) which is the catalog dictionary to
// modify.
//
// Returns bool which indicates whether the catalog was modified.
func ensureMarkInfo(catalogDict *pdfparse.Dict) bool {
	if catalogDict.Has(markInfoKey) {
		return false
	}

	markInfo := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Marked", Value: pdfparse.Bool(true)},
	}}
	catalogDict.Set(markInfoKey, pdfparse.DictObj(markInfo))
	return true
}

// ensureStructTreeRoot adds a minimal /StructTreeRoot to the catalog if
// the key is absent.
//
// The structure tree is created as a new indirect object with Type
// /StructTreeRoot, an empty /ParentTree number tree, and an empty /K array.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer for adding objects.
// Takes catalogDict (*pdfparse.Dict) which is the catalog dictionary to
// modify.
//
// Returns bool which indicates whether the catalog was modified.
func ensureStructTreeRoot(writer *pdfparse.Writer, catalogDict *pdfparse.Dict) bool {
	if catalogDict.Has(structTreeRootKey) {
		return false
	}

	structTree := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("StructTreeRoot")},
		{Key: "ParentTree", Value: pdfparse.DictObj(pdfparse.Dict{})},
		{Key: "K", Value: pdfparse.Arr()},
	}}
	objNum := writer.AddObject(pdfparse.DictObj(structTree))
	catalogDict.Set(structTreeRootKey, pdfparse.RefObj(objNum, 0))
	return true
}

// ensureLang adds /Lang (en) to the catalog if the key is absent.
//
// Takes catalogDict (*pdfparse.Dict) which is the catalog dictionary to
// modify.
//
// Returns bool which indicates whether the catalog was modified.
func ensureLang(catalogDict *pdfparse.Dict) bool {
	if catalogDict.Has(langKey) {
		return false
	}

	catalogDict.Set(langKey, pdfparse.Str(defaultLang))
	return true
}

// ensureViewerPreferences adds /ViewerPreferences << /DisplayDocTitle
// true >> to the catalog if the key is absent.
//
// Takes catalogDict (*pdfparse.Dict) which is the catalog dictionary to
// modify.
//
// Returns bool which indicates whether the catalog was modified.
func ensureViewerPreferences(catalogDict *pdfparse.Dict) bool {
	if catalogDict.Has(viewerPreferencesKey) {
		return false
	}

	viewerPrefs := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "DisplayDocTitle", Value: pdfparse.Bool(true)},
	}}
	catalogDict.Set(viewerPreferencesKey, pdfparse.DictObj(viewerPrefs))
	return true
}

// ensureInfoTitle ensures the document information dictionary has a
// /Title entry.
//
// If no /Info reference exists in the trailer, this is a no-op. If /Info
// exists but has no /Title, an empty string title is added.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer for reading and
// modifying objects.
//
// Returns bool which indicates whether any object was modified.
func ensureInfoTitle(writer *pdfparse.Writer) bool {
	trailer := writer.Trailer()
	infoRef := trailer.GetRef(infoKey)
	if infoRef.Number == 0 {
		return false
	}

	infoObj := writer.GetObject(infoRef.Number)
	infoDict, ok := infoObj.Value.(pdfparse.Dict)
	if !ok {
		return false
	}

	if infoDict.Has("Title") {
		return false
	}

	infoDict.Set("Title", pdfparse.Str(""))
	writer.SetObject(infoRef.Number, pdfparse.DictObj(infoDict))
	return true
}
