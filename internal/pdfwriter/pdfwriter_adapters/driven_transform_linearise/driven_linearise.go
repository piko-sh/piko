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

package driven_transform_linearise

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the linearise transformer.
	defaultPriority = 350

	// linearisedVersion is the linearisation parameter dictionary version
	// value, as defined by PDF spec Annex F.
	linearisedVersion = 1.0
)

// LineariseTransformer reorganises a PDF so that the first page's objects
// appear at the start of the file, enabling progressive rendering.
//
// It parses the PDF, identifies all objects the first page depends on,
// creates a new writer with objects numbered so that a linearisation
// parameter dictionary comes first, followed by the first page's
// dependencies, then the remaining objects. The linearisation parameter
// dictionary is written as the first object and contains /Linearized,
// /N (page count), and /O (first page object number) entries. Full hint
// tables are not generated; the primary benefit is first-page-first
// object ordering.
type LineariseTransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*LineariseTransformer)(nil)

// New creates a new linearise transformer with default name and priority.
//
// Returns *LineariseTransformer which is the initialised transformer.
func New() *LineariseTransformer {
	return &LineariseTransformer{
		name:     "linearise",
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *LineariseTransformer) Name() string { return t.name }

// Type returns TransformerDelivery.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a delivery
// transformer.
func (*LineariseTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerDelivery
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *LineariseTransformer) Priority() int { return t.priority }

// Transform linearises the PDF by reordering objects so the first page's
// dependencies appear first. Options must be LineariseOptions or
// *LineariseOptions.
//
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be LineariseOptions or *LineariseOptions.
//
// Returns []byte which is the linearised PDF.
// Returns error when the PDF cannot be parsed or linearisation fails.
func (*LineariseTransformer) Transform(_ context.Context, pdf []byte, options any) ([]byte, error) {
	if _, err := castOptions(options); err != nil {
		return nil, err
	}

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("linearise: parsing PDF: %w", err)
	}

	pageRefs, err := collectPageRefs(doc)
	if err != nil {
		return nil, fmt.Errorf("linearise: collecting pages: %w", err)
	}
	if len(pageRefs) == 0 {
		return nil, errors.New("linearise: document contains no pages")
	}

	firstPageObjNum := pageRefs[0]
	firstPageDeps := collectFirstPageDeps(doc, firstPageObjNum, buildPagesTreeSet(doc))

	writer, oldToNew, newFirstPageNum, err := buildLinearisedWriter(doc, firstPageDeps, len(pageRefs))
	if err != nil {
		return nil, fmt.Errorf("linearise: building writer: %w", err)
	}

	rewriteReferences(writer, oldToNew)
	rewriteTrailerRefs(writer, oldToNew)

	updateLinearisationDict(writer, newFirstPageNum)

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("linearise: writing PDF: %w", err)
	}
	return output, nil
}

// castOptions extracts LineariseOptions from the generic options.
//
// Takes options (any) which must be LineariseOptions or *LineariseOptions.
//
// Returns pdfwriter_dto.LineariseOptions which holds the extracted options.
// Returns error when the options type is invalid or nil.
func castOptions(options any) (pdfwriter_dto.LineariseOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.LineariseOptions:
		return v, nil
	case *pdfwriter_dto.LineariseOptions:
		if v == nil {
			return pdfwriter_dto.LineariseOptions{}, errors.New("linearise: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.LineariseOptions{}, fmt.Errorf("linearise: expected LineariseOptions, got %T", options)
	}
}

// collectPageRefs walks the page tree and returns object numbers for all
// leaf Page objects in document order.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
//
// Returns []int which holds the page object numbers.
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
// Takes objNum (int) which is the current tree node's object number.
//
// Returns []int which holds the collected page object numbers.
// Returns error when any node cannot be read.
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

// buildPagesTreeSet returns a set of object numbers that belong to the
// Pages tree.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
//
// Returns map[int]bool which holds the intermediate Pages tree node object numbers.
func buildPagesTreeSet(doc *pdfparse.Document) map[int]bool {
	set := make(map[int]bool)
	trailer := doc.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return set
	}

	catalog, err := doc.GetObject(rootRef.Number)
	if err != nil {
		return set
	}

	catalogDict, ok := catalog.Value.(pdfparse.Dict)
	if !ok {
		return set
	}

	pagesRef := catalogDict.GetRef("Pages")
	if pagesRef.Number == 0 {
		return set
	}

	collectPagesTreeNodes(doc, pagesRef.Number, set)
	return set
}

// collectPagesTreeNodes recursively adds intermediate Pages tree node
// object numbers to the set.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes objNum (int) which is the current node's object number.
// Takes set (map[int]bool) which accumulates the Pages tree node numbers.
func collectPagesTreeNodes(doc *pdfparse.Document, objNum int, set map[int]bool) {
	obj, err := doc.GetObject(objNum)
	if err != nil {
		return
	}

	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return
	}

	nodeType := dict.GetName("Type")
	if nodeType == "Pages" {
		set[objNum] = true
		kids := dict.GetArray("Kids")
		for _, kid := range kids {
			ref, ok := kid.Value.(pdfparse.Ref)
			if !ok {
				continue
			}
			collectPagesTreeNodes(doc, ref.Number, set)
		}
	}
}

// collectFirstPageDeps walks the first page's dictionary recursively and
// returns the set of object numbers that the first page depends on.
//
// This includes the page object itself, its content streams, resources, fonts,
// and images. The pagesTree set prevents recursion into the Pages tree which
// would pull in the entire document.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageObjNum (int) which is the first page's object number.
// Takes pagesTree (map[int]bool) which holds Pages tree nodes to exclude.
//
// Returns map[int]bool which holds the set of dependency object numbers.
func collectFirstPageDeps(doc *pdfparse.Document, pageObjNum int, pagesTree map[int]bool) map[int]bool {
	deps := make(map[int]bool)
	visited := make(map[int]bool)
	collectDepsRecursive(doc, pageObjNum, deps, visited, pagesTree)
	return deps
}

// collectDepsRecursive adds objNum to deps and recursively follows all
// references found in the object's value.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes objNum (int) which is the object number to process.
// Takes deps (map[int]bool) which accumulates dependency object numbers.
// Takes visited (map[int]bool) which tracks already-visited objects.
// Takes pagesTree (map[int]bool) which holds Pages tree nodes to skip.
func collectDepsRecursive(
	doc *pdfparse.Document,
	objNum int,
	deps map[int]bool,
	visited map[int]bool,
	pagesTree map[int]bool,
) {
	if visited[objNum] {
		return
	}
	visited[objNum] = true

	if pagesTree[objNum] {
		return
	}

	deps[objNum] = true

	obj, err := doc.GetObject(objNum)
	if err != nil {
		return
	}

	collectRefsFromObject(doc, obj, deps, visited, pagesTree)
}

// collectRefsFromObject extracts all indirect references from a PDF object
// and recursively collects their dependencies.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes obj (pdfparse.Object) which is the object to inspect.
// Takes deps (map[int]bool) which accumulates dependency object numbers.
// Takes visited (map[int]bool) which tracks already-visited objects.
// Takes pagesTree (map[int]bool) which holds Pages tree nodes to skip.
func collectRefsFromObject(
	doc *pdfparse.Document,
	obj pdfparse.Object,
	deps map[int]bool,
	visited map[int]bool,
	pagesTree map[int]bool,
) {
	switch obj.Type {
	case pdfparse.ObjectReference:
		collectRefFromReference(doc, obj, deps, visited, pagesTree)
	case pdfparse.ObjectArray:
		collectRefsFromArray(doc, obj, deps, visited, pagesTree)
	case pdfparse.ObjectDictionary, pdfparse.ObjectStream:
		collectRefsFromDictPairs(doc, obj, deps, visited, pagesTree)
	}
}

// collectRefFromReference follows a single indirect reference and
// recursively collects its dependencies.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes obj (pdfparse.Object) which is the reference object to follow.
// Takes deps (map[int]bool) which accumulates dependency object numbers.
// Takes visited (map[int]bool) which tracks already-visited objects.
// Takes pagesTree (map[int]bool) which holds Pages tree nodes to skip.
func collectRefFromReference(
	doc *pdfparse.Document,
	obj pdfparse.Object,
	deps map[int]bool,
	visited map[int]bool,
	pagesTree map[int]bool,
) {
	ref, ok := obj.Value.(pdfparse.Ref)
	if ok {
		collectDepsRecursive(doc, ref.Number, deps, visited, pagesTree)
	}
}

// collectRefsFromArray iterates over array elements and recursively
// collects their dependencies.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes obj (pdfparse.Object) which is the array object to inspect.
// Takes deps (map[int]bool) which accumulates dependency object numbers.
// Takes visited (map[int]bool) which tracks already-visited objects.
// Takes pagesTree (map[int]bool) which holds Pages tree nodes to skip.
func collectRefsFromArray(
	doc *pdfparse.Document,
	obj pdfparse.Object,
	deps map[int]bool,
	visited map[int]bool,
	pagesTree map[int]bool,
) {
	items, ok := obj.Value.([]pdfparse.Object)
	if !ok {
		return
	}
	for _, item := range items {
		collectRefsFromObject(doc, item, deps, visited, pagesTree)
	}
}

// collectRefsFromDictPairs walks a dictionary's pairs and recursively
// collects dependencies.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes obj (pdfparse.Object) which is the dictionary or stream object.
// Takes deps (map[int]bool) which accumulates dependency object numbers.
// Takes visited (map[int]bool) which tracks already-visited objects.
// Takes pagesTree (map[int]bool) which holds Pages tree nodes to skip.
func collectRefsFromDictPairs(
	doc *pdfparse.Document,
	obj pdfparse.Object,
	deps map[int]bool,
	visited map[int]bool,
	pagesTree map[int]bool,
) {
	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return
	}
	for _, pair := range dict.Pairs {
		if pair.Key == "Parent" {
			continue
		}
		collectRefsFromObject(doc, pair.Value, deps, visited, pagesTree)
	}
}

// buildLinearisedWriter creates a new Writer with objects renumbered so
// that the linearisation dictionary is object 1.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes firstPageDeps (map[int]bool) which holds the first page dependency set.
// Takes pageCount (int) which is the total number of pages.
//
// Returns *pdfparse.Writer which is the new writer with renumbered objects.
// Returns map[int]int which maps old object numbers to new object numbers.
// Returns int which is the new object number of the first page.
// Returns error when copying objects fails.
func buildLinearisedWriter(
	doc *pdfparse.Document,
	firstPageDeps map[int]bool,
	pageCount int,
) (*pdfparse.Writer, map[int]int, int, error) {
	firstPageNums, remainingNums := partitionObjects(doc, firstPageDeps)
	oldToNew := buildRenumberMap(firstPageNums, remainingNums)
	newFirstPageNum := findNewFirstPageNum(doc, firstPageDeps, oldToNew)

	writer := pdfparse.NewWriter()
	addLinearisationDict(writer, pageCount, newFirstPageNum)

	if err := copyObjects(doc, writer, oldToNew, firstPageNums, remainingNums); err != nil {
		return nil, nil, 0, err
	}

	writer.SetTrailer(doc.Trailer())
	return writer, oldToNew, newFirstPageNum, nil
}

// partitionObjects separates document object numbers into first-page
// dependencies and remaining objects, both sorted.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes firstPageDeps (map[int]bool) which holds the first page dependency set.
//
// Returns firstPage ([]int) which holds the first page dependency object numbers.
// Returns remaining ([]int) which holds all other object numbers.
func partitionObjects(doc *pdfparse.Document, firstPageDeps map[int]bool) (firstPage, remaining []int) {
	allObjNums := doc.ObjectNumbers()
	slices.Sort(allObjNums)

	for _, num := range allObjNums {
		if firstPageDeps[num] {
			firstPage = append(firstPage, num)
		} else {
			remaining = append(remaining, num)
		}
	}
	return firstPage, remaining
}

// buildRenumberMap creates an old-to-new object number mapping.
//
// Takes firstPageNums ([]int) which holds the first page dependency object numbers.
// Takes remainingNums ([]int) which holds all other object numbers.
//
// Returns map[int]int which maps old object numbers to new object numbers.
func buildRenumberMap(firstPageNums, remainingNums []int) map[int]int {
	totalLen := len(firstPageNums) + len(remainingNums)
	oldToNew := make(map[int]int, totalLen)
	nextNum := 2

	for _, oldNum := range firstPageNums {
		oldToNew[oldNum] = nextNum
		nextNum++
	}
	for _, oldNum := range remainingNums {
		oldToNew[oldNum] = nextNum
		nextNum++
	}
	return oldToNew
}

// findNewFirstPageNum looks up the remapped object number for the first
// page among first-page dependencies.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes firstPageDeps (map[int]bool) which holds the first page dependency set.
// Takes oldToNew (map[int]int) which maps old object numbers to new numbers.
//
// Returns int which is the new object number of the first page, or zero if not found.
func findNewFirstPageNum(doc *pdfparse.Document, firstPageDeps map[int]bool, oldToNew map[int]int) int {
	for oldNum, newNum := range oldToNew {
		if !firstPageDeps[oldNum] {
			continue
		}
		obj, err := doc.GetObject(oldNum)
		if err != nil {
			continue
		}
		dict, ok := obj.Value.(pdfparse.Dict)
		if !ok {
			continue
		}
		if dict.GetName("Type") == "Page" {
			return newNum
		}
	}
	return 0
}

// addLinearisationDict writes the linearisation parameter dictionary as
// object 1.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes pageCount (int) which is the total number of pages.
// Takes firstPageObjNum (int) which is the first page's object number.
func addLinearisationDict(writer *pdfparse.Writer, pageCount, firstPageObjNum int) {
	linDict := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Linearized", Value: pdfparse.Real(linearisedVersion)},
		{Key: "N", Value: pdfparse.Int(int64(pageCount))},
		{Key: "O", Value: pdfparse.Int(int64(firstPageObjNum))},
	}}
	writer.SetObject(1, pdfparse.DictObj(linDict))
}

// copyObjects copies all document objects into the writer using the
// renumbered object numbers.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes writer (*pdfparse.Writer) which is the target PDF writer.
// Takes oldToNew (map[int]int) which maps old object numbers to new numbers.
// Takes firstPageNums ([]int) which holds first page dependency object numbers.
// Takes remainingNums ([]int) which holds all other object numbers.
//
// Returns error when any object cannot be read.
func copyObjects(
	doc *pdfparse.Document,
	writer *pdfparse.Writer,
	oldToNew map[int]int,
	firstPageNums, remainingNums []int,
) error {
	allNums := make([]int, 0, len(firstPageNums)+len(remainingNums))
	allNums = append(allNums, firstPageNums...)
	allNums = append(allNums, remainingNums...)

	for _, oldNum := range allNums {
		obj, err := doc.GetObject(oldNum)
		if err != nil {
			return fmt.Errorf("reading object %d: %w", oldNum, err)
		}
		writer.SetObject(oldToNew[oldNum], obj)
	}
	return nil
}

// rewriteReferences walks all objects in the writer and remaps indirect
// references from old object numbers to new object numbers.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer to update.
// Takes oldToNew (map[int]int) which maps old object numbers to new numbers.
func rewriteReferences(writer *pdfparse.Writer, oldToNew map[int]int) {
	var objNums []int
	for i := 1; i < writer.NextObjectNumber(); i++ {
		obj := writer.GetObject(i)
		if !obj.IsNull() {
			objNums = append(objNums, i)
		}
	}

	for _, num := range objNums {
		obj := writer.GetObject(num)
		rewritten := remapObjectRefs(obj, oldToNew)
		writer.SetObject(num, rewritten)
	}
}

// remapObjectRefs recursively rewrites all indirect references in an
// object using the old-to-new mapping.
//
// Takes obj (pdfparse.Object) which is the object to rewrite.
// Takes oldToNew (map[int]int) which maps old object numbers to new numbers.
//
// Returns pdfparse.Object which is the rewritten object.
func remapObjectRefs(obj pdfparse.Object, oldToNew map[int]int) pdfparse.Object {
	switch obj.Type {
	case pdfparse.ObjectReference:
		ref, ok := obj.Value.(pdfparse.Ref)
		if ok {
			if newNum, exists := oldToNew[ref.Number]; exists {
				return pdfparse.RefObj(newNum, ref.Generation)
			}
		}
		return obj
	case pdfparse.ObjectArray:
		items, ok := obj.Value.([]pdfparse.Object)
		if !ok {
			return obj
		}
		newItems := make([]pdfparse.Object, len(items))
		for i, item := range items {
			newItems[i] = remapObjectRefs(item, oldToNew)
		}
		return pdfparse.Arr(newItems...)
	case pdfparse.ObjectDictionary:
		dict, ok := obj.Value.(pdfparse.Dict)
		if !ok {
			return obj
		}
		return pdfparse.DictObj(remapDictRefs(dict, oldToNew))
	case pdfparse.ObjectStream:
		dict, ok := obj.Value.(pdfparse.Dict)
		if !ok {
			return obj
		}
		newDict := remapDictRefs(dict, oldToNew)
		return pdfparse.StreamObj(newDict, obj.StreamData)
	default:
		return obj
	}
}

// remapDictRefs rewrites all indirect references in a dictionary's values.
//
// Takes dict (pdfparse.Dict) which is the dictionary to rewrite.
// Takes oldToNew (map[int]int) which maps old object numbers to new numbers.
//
// Returns pdfparse.Dict which is the dictionary with rewritten references.
func remapDictRefs(dict pdfparse.Dict, oldToNew map[int]int) pdfparse.Dict {
	newPairs := make([]pdfparse.DictPair, len(dict.Pairs))
	for i, pair := range dict.Pairs {
		newPairs[i] = pdfparse.DictPair{
			Key:   pair.Key,
			Value: remapObjectRefs(pair.Value, oldToNew),
		}
	}
	return pdfparse.Dict{Pairs: newPairs}
}

// rewriteTrailerRefs remaps indirect references in the trailer dictionary.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer to update.
// Takes oldToNew (map[int]int) which maps old object numbers to new numbers.
func rewriteTrailerRefs(writer *pdfparse.Writer, oldToNew map[int]int) {
	trailer := writer.Trailer()
	newTrailer := remapDictRefs(trailer, oldToNew)
	writer.SetTrailer(newTrailer)
}

// updateLinearisationDict updates the /O entry in the linearisation
// parameter dictionary (object 1) with the final first page object number.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer to update.
// Takes newFirstPageNum (int) which is the renumbered first page object number.
func updateLinearisationDict(writer *pdfparse.Writer, newFirstPageNum int) {
	obj := writer.GetObject(1)
	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return
	}
	dict.Set("O", pdfparse.Int(int64(newFirstPageNum)))
	writer.SetObject(1, pdfparse.DictObj(dict))
}
