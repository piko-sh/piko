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

package annotator_domain

// Reports literal names in client scripts and templates that can never resolve at
// runtime. These are pk.refs names that no p-ref declares, partial names that no partial
// has, and [modal=name] selectors that no modal element matches. Only literal names are
// checked, and every check counts the templates the runtime can add to a page later, so a
// warning means the name is missing from every template that could supply it.

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/jsscan"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxReferenceDiagnosticsPerComponent caps how many unresolved reference warnings one
	// component reports, so a broken component cannot flood the build output.
	maxReferenceDiagnosticsPerComponent = 50

	// maxClientScriptTokens caps how many tokens of one client script are checked.
	maxClientScriptTokens = 1 << 20

	// attributeModal is the attribute that names a modal element.
	attributeModal = "modal"

	// attributeModalSelector is the attribute that opens a modal by selector.
	attributeModalSelector = "p-modal:selector"

	// attributeModalSelectorProp is the component prop that passes a modal selector on.
	attributeModalSelectorProp = "modal_selector"

	// attributeRef is the name of the p-ref directive when it is bound dynamically.
	attributeRef = "p-ref"

	// modalSelectorPrefix starts the simple modal selector form, [modal=name].
	modalSelectorPrefix = "[modal="

	// modalSelectorSuffix ends the simple modal selector form.
	modalSelectorSuffix = "]"

	// partialSourceExtension is the file extension of a partial's source file.
	partialSourceExtension = ".pk"

	// pageContextName is the name of the pk context in page and partial scripts.
	pageContextName = "pk"

	// refsPropertyName is the property of a script context that holds its refs.
	refsPropertyName = "refs"
)

var (
	// partialNameCalls lists the runtime calls whose first argument is a partial name.
	partialNameCalls = [][]string{
		{"piko", "partial"},
		{"piko", "partials", "reload"},
		{"piko", "partials", "autoRefresh"},
	}

	// partialGroupCall is the runtime call whose first argument is an array of partial
	// names.
	partialGroupCall = []string{"piko", "partials", "reloadGroup"}
)

// templateNames holds the literal names that one or more templates declare, and whether
// any of them sets such a name at runtime, which makes every name possible.
type templateNames struct {
	// refs holds the p-ref names.
	refs map[string]struct{}

	// modals holds the modal attribute values.
	modals map[string]struct{}

	// partialNames holds the partial_name attribute values written on invocations.
	partialNames map[string]struct{}

	// hasDynamicRef is true when a p-ref is bound or written in raw HTML.
	hasDynamicRef bool

	// hasDynamicModal is true when a modal attribute is bound or written in raw HTML.
	hasDynamicModal bool

	// hasDynamicPartialName is true when a partial_name is bound or written in raw HTML.
	hasDynamicPartialName bool
}

// referenceChecker reports literal references that no template in the build can resolve.
// It indexes every template once, so each check is a map lookup.
type referenceChecker struct {
	// virtualModule holds every component in the build.
	virtualModule *annotator_dto.VirtualModule

	// componentNames holds the names each component's own template declares, keyed by hashed
	// name.
	componentNames map[string]*templateNames

	// projectNames holds the names every template declares.
	projectNames *templateNames

	// partialTemplateNames holds the names every partial's template declares. Partials can
	// be added to any page at runtime by partial reloads, modals and remote rendering.
	partialTemplateNames *templateNames

	// partialNames holds every name the partial reload calls can find.
	partialNames map[string]struct{}

	// partialNameByImportPath maps a partial's module import path to its partial name.
	partialNameByImportPath map[string]string

	// modalUsages holds the modal selectors that pages do not resolve, keyed by where the
	// selector is written.
	modalUsages map[modalUsageKey]*modalUsage

	// partialsSourceDir is the folder that partial names are relative to.
	partialsSourceDir string
}

// modalUsageKey identifies a modal selector by where it is written.
type modalUsageKey struct {
	// sourcePath is the file the selector is written in.
	sourcePath string

	// line is the selector's line.
	line int

	// column is the selector's column.
	column int
}

// modalUsage is a modal selector that one or more pages do not resolve.
type modalUsage struct {
	// selector is the selector as written.
	selector string

	// name is the modal name the selector looks for.
	name string

	// suggestion is the closest modal name, or empty when none is close.
	suggestion string

	// pages lists the pages that do not resolve the selector.
	pages []string

	// location is where the selector is written.
	location ast_domain.Location
}

// scriptLocator turns byte offsets in a client script into locations in its source file.
type scriptLocator struct {
	// lines places offsets within the script.
	lines *jsscan.LineIndex

	// start is where the script content begins in its source file.
	start ast_domain.Location
}

// newTemplateNames creates an empty set of template names.
//
// Returns *templateNames which is ready to collect names.
func newTemplateNames() *templateNames {
	return &templateNames{
		refs:                  make(map[string]struct{}),
		modals:                make(map[string]struct{}),
		partialNames:          make(map[string]struct{}),
		hasDynamicRef:         false,
		hasDynamicModal:       false,
		hasDynamicPartialName: false,
	}
}

// addTree collects the names a template declares.
//
// Takes tree (*ast_domain.TemplateAST) which is the template to read, or nil.
func (n *templateNames) addTree(tree *ast_domain.TemplateAST) {
	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		n.addNode(node)
		return true
	})
}

// addNode collects the names one node declares.
//
// Takes node (*ast_domain.TemplateNode) which is the node to read.
func (n *templateNames) addNode(node *ast_domain.TemplateNode) {
	if node.NodeType == ast_domain.NodeRawHTML {
		n.addRawHTML(node.InnerHTML)
		return
	}
	if node.DirRef != nil && node.DirRef.RawExpression != "" {
		n.refs[node.DirRef.RawExpression] = struct{}{}
	}
	for index := range node.Attributes {
		attribute := &node.Attributes[index]
		switch strings.ToLower(attribute.Name) {
		case attributeModal:
			n.modals[attribute.Value] = struct{}{}
		case attributePartialName:
			for name := range strings.FieldsSeq(attribute.Value) {
				n.partialNames[name] = struct{}{}
			}
		}
	}
	n.hasDynamicRef = n.hasDynamicRef || isBoundAttribute(node, attributeRef)
	n.hasDynamicModal = n.hasDynamicModal || isBoundAttribute(node, attributeModal)
	n.hasDynamicPartialName = n.hasDynamicPartialName || isBoundAttribute(node, attributePartialName)
}

// addRawHTML marks the names that raw HTML may set, since its markup is not parsed.
//
// Takes html (string) which is the raw HTML.
func (n *templateNames) addRawHTML(html string) {
	n.hasDynamicRef = n.hasDynamicRef || strings.Contains(html, attributeRef)
	n.hasDynamicModal = n.hasDynamicModal || strings.Contains(html, attributeModal+"=")
	n.hasDynamicPartialName = n.hasDynamicPartialName || strings.Contains(html, attributePartialName)
}

// merge adds the names of another set to this one.
//
// Takes other (*templateNames) which holds the names to add.
func (n *templateNames) merge(other *templateNames) {
	maps.Copy(n.refs, other.refs)
	maps.Copy(n.modals, other.modals)
	maps.Copy(n.partialNames, other.partialNames)
	n.hasDynamicRef = n.hasDynamicRef || other.hasDynamicRef
	n.hasDynamicModal = n.hasDynamicModal || other.hasDynamicModal
	n.hasDynamicPartialName = n.hasDynamicPartialName || other.hasDynamicPartialName
}

// newReferenceChecker indexes the names every component in the build declares.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which holds every component.
// Takes partialsSourceDir (string) which is the folder partial names are relative to.
//
// Returns *referenceChecker which is ready to check components.
func newReferenceChecker(virtualModule *annotator_dto.VirtualModule, partialsSourceDir string) *referenceChecker {
	checker := &referenceChecker{
		virtualModule:           virtualModule,
		componentNames:          make(map[string]*templateNames, len(virtualModule.ComponentsByHash)),
		projectNames:            newTemplateNames(),
		partialTemplateNames:    newTemplateNames(),
		partialNames:            make(map[string]struct{}),
		partialNameByImportPath: make(map[string]string),
		modalUsages:             make(map[modalUsageKey]*modalUsage),
		partialsSourceDir:       partialsSourceDir,
	}
	for hashedName, component := range virtualModule.ComponentsByHash {
		checker.indexComponent(hashedName, component)
	}
	maps.Copy(checker.partialNames, checker.projectNames.partialNames)
	return checker
}

// indexComponent records the names one component declares.
//
// Takes hashedName (string) which is the component's hashed name.
// Takes component (*annotator_dto.VirtualComponent) which is the component to index.
func (c *referenceChecker) indexComponent(hashedName string, component *annotator_dto.VirtualComponent) {
	names := newTemplateNames()
	if component.Source != nil {
		names.addTree(component.Source.Template)
	}
	c.componentNames[hashedName] = names
	c.projectNames.merge(names)
	if component.PartialName == "" {
		return
	}
	c.partialTemplateNames.merge(names)
	c.partialNames[component.PartialName] = struct{}{}
	if component.Source != nil && component.Source.ModuleImportPath != "" {
		c.partialNameByImportPath[component.Source.ModuleImportPath] = component.PartialName
	}
}

// checkComponent checks a component's client script and, for a page, the modal selectors
// on the page. Modal findings are held until modalDiagnostics is called, so a selector
// that several pages miss is reported once.
//
// Takes hashedName (string) which is the component's hashed name.
// Takes component (*annotator_dto.VirtualComponent) which is the component to check.
// Takes result (*annotator_dto.AnnotationResult) which holds the component's expanded
// template, or nil.
//
// Returns []*ast_domain.Diagnostic which holds the script warnings found.
func (c *referenceChecker) checkComponent(
	hashedName string,
	component *annotator_dto.VirtualComponent,
	result *annotator_dto.AnnotationResult,
) []*ast_domain.Diagnostic {
	if component.Source == nil || !hasPKScriptContext(component) {
		return nil
	}
	if isPageComponent(component) && result != nil {
		c.collectModalUsages(component, result.AnnotatedAST)
	}
	if component.Source.ClientScript == "" {
		return nil
	}
	tokens, _ := jsscan.Tokens(component.Source.ClientScript, maxClientScriptTokens)
	locator := newScriptLocator(component.Source.ClientScript, component.Source.ClientScriptLocation)
	diagnostics := c.checkScriptRefs(hashedName, component, tokens, locator)
	diagnostics = append(diagnostics, c.checkScriptPartialNames(component, tokens, locator)...)
	if len(diagnostics) > maxReferenceDiagnosticsPerComponent {
		diagnostics = diagnostics[:maxReferenceDiagnosticsPerComponent]
	}
	return diagnostics
}

// checkScriptRefs reports pk.refs names that no element the script can reach declares.
//
// A page script's refs resolve against the whole page, so the page's own template and
// every partial count. A partial script's refs resolve against the partial's root, which
// holds content slotted in by any caller, so every template counts.
//
// Takes hashedName (string) which is the component's hashed name.
// Takes component (*annotator_dto.VirtualComponent) which owns the script.
// Takes tokens ([]jsscan.Token) which are the script's tokens.
// Takes locator (*scriptLocator) which places findings in the source file.
//
// Returns []*ast_domain.Diagnostic which holds a warning per unresolved ref.
func (c *referenceChecker) checkScriptRefs(
	hashedName string,
	component *annotator_dto.VirtualComponent,
	tokens []jsscan.Token,
	locator *scriptLocator,
) []*ast_domain.Diagnostic {
	reads := jsscan.FindPropertyReads(tokens, pageContextName, refsPropertyName)
	if len(reads) == 0 {
		return nil
	}
	reachable, scopeText := c.reachableRefNames(hashedName, component)
	if reachable == nil {
		return nil
	}
	for _, read := range reads {
		if read.Assigned {
			reachable[read.Value] = struct{}{}
		}
	}
	var diagnostics []*ast_domain.Diagnostic
	var candidates []string
	for _, read := range reads {
		if _, found := reachable[read.Value]; found {
			continue
		}
		if candidates == nil {
			candidates = sortedNames(reachable)
		}
		message := fmt.Sprintf("pk.refs.%s matches no element with p-ref=%q %s, so it is always null",
			read.Value, read.Value, scopeText)
		message += didYouMean(linguistics_domain.ClosestName(read.Value, candidates))
		diagnostics = append(diagnostics, ast_domain.NewDiagnosticWithCode(
			ast_domain.Warning, message, read.Value, annotator_dto.CodeUnresolvedRef,
			locator.locate(read.Start), component.Source.SourcePath,
		))
	}
	return diagnostics
}

// reachableRefNames returns the p-ref names a component's script can reach.
//
// Takes hashedName (string) which is the component's hashed name.
// Takes component (*annotator_dto.VirtualComponent) which owns the script.
//
// Returns map[string]struct{} which holds a fresh copy of the names, or nil when a p-ref
// set at runtime makes every name reachable.
// Returns string which describes where the names were looked for.
func (c *referenceChecker) reachableRefNames(hashedName string, component *annotator_dto.VirtualComponent) (map[string]struct{}, string) {
	if !isPageComponent(component) {
		if c.projectNames.hasDynamicRef {
			return nil, ""
		}
		return maps.Clone(c.projectNames.refs), "in any template"
	}
	own := c.componentNames[hashedName]
	if own == nil || own.hasDynamicRef || c.partialTemplateNames.hasDynamicRef {
		return nil, ""
	}
	reachable := maps.Clone(own.refs)
	maps.Copy(reachable, c.partialTemplateNames.refs)
	return reachable, "on this page or in any partial"
}

// checkScriptPartialNames reports literal partial names, passed to the partial reload
// calls, that no partial in the build has.
//
// Takes component (*annotator_dto.VirtualComponent) which owns the script.
// Takes tokens ([]jsscan.Token) which are the script's tokens.
// Takes locator (*scriptLocator) which places findings in the source file.
//
// Returns []*ast_domain.Diagnostic which holds a warning per unknown partial name.
func (c *referenceChecker) checkScriptPartialNames(
	component *annotator_dto.VirtualComponent,
	tokens []jsscan.Token,
	locator *scriptLocator,
) []*ast_domain.Diagnostic {
	if c.projectNames.hasDynamicPartialName {
		return nil
	}
	var diagnostics []*ast_domain.Diagnostic
	report := func(call []string, literal jsscan.Literal) {
		if _, found := c.partialNames[literal.Value]; found {
			return
		}
		message := fmt.Sprintf("%s(%q) names no partial in the build, so the partial is never found",
			strings.Join(call, "."), literal.Value)
		message += c.partialNameHint(component, literal.Value)
		diagnostic := ast_domain.NewDiagnosticWithCode(
			ast_domain.Warning, message, literal.Value, annotator_dto.CodeUnresolvedPartialName,
			locator.locate(literal.Start), component.Source.SourcePath,
		)
		diagnostic.SourceLength = literal.Length
		diagnostics = append(diagnostics, diagnostic)
	}
	for _, call := range partialNameCalls {
		for _, literal := range jsscan.FindCallArguments(tokens, call...) {
			report(call, literal)
		}
	}
	for _, literal := range jsscan.FindCallArrayArguments(tokens, partialGroupCall...) {
		report(partialGroupCall, literal)
	}
	slices.SortStableFunc(diagnostics, compareDiagnosticLocations)
	return diagnostics
}

// partialNameHint suggests the partial name meant by an unknown one. It suggests the
// partial behind an import alias, the name without its folder or extension, or the
// closest name.
//
// Takes component (*annotator_dto.VirtualComponent) which owns the script.
// Takes name (string) which is the unknown partial name.
//
// Returns string which is the hint to add to the message, or empty.
func (c *referenceChecker) partialNameHint(component *annotator_dto.VirtualComponent, name string) string {
	for _, pikoImport := range component.Source.PikoImports {
		if pikoImport.Alias != name {
			continue
		}
		if partialName, found := c.partialNameByImportPath[pikoImport.Path]; found {
			return fmt.Sprintf(". %q is the import alias; the partial's name is %q", name, partialName)
		}
	}
	trimmed := strings.TrimSuffix(name, partialSourceExtension)
	if c.partialsSourceDir != "" {
		trimmed = strings.TrimPrefix(trimmed, path.Clean(c.partialsSourceDir)+"/")
	}
	if _, found := c.partialNames[trimmed]; found && trimmed != name {
		return didYouMean(trimmed)
	}
	return didYouMean(linguistics_domain.ClosestName(name, sortedNames(c.partialNames)))
}

// collectModalUsages records the simple modal selectors on a page that no modal element
// on the page, or in any partial, matches. Partials count because the runtime can add
// them to the page, as a modal's own content is.
//
// Takes page (*annotator_dto.VirtualComponent) which is the page.
// Takes tree (*ast_domain.TemplateAST) which is the page's expanded template, holding its
// layout and partials.
func (c *referenceChecker) collectModalUsages(page *annotator_dto.VirtualComponent, tree *ast_domain.TemplateAST) {
	if tree == nil || c.partialTemplateNames.hasDynamicModal {
		return
	}
	pageNames := newTemplateNames()
	pageNames.addTree(tree)
	if pageNames.hasDynamicModal {
		return
	}
	known := maps.Clone(pageNames.modals)
	maps.Copy(known, c.partialTemplateNames.modals)
	pageLabel := componentLabel(page)
	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		for index := range node.Attributes {
			c.recordModalSelector(node, &node.Attributes[index], known, page, pageLabel)
		}
		return true
	})
}

// recordModalSelector records one attribute when it holds a simple modal selector that no
// known modal name matches.
//
// Takes node (*ast_domain.TemplateNode) which holds the attribute.
// Takes attribute (*ast_domain.HTMLAttribute) which may hold a modal selector.
// Takes known (map[string]struct{}) which holds the modal names in reach.
// Takes page (*annotator_dto.VirtualComponent) which is the page being checked.
// Takes pageLabel (string) which names the page in messages.
func (c *referenceChecker) recordModalSelector(
	node *ast_domain.TemplateNode,
	attribute *ast_domain.HTMLAttribute,
	known map[string]struct{},
	page *annotator_dto.VirtualComponent,
	pageLabel string,
) {
	if !isModalSelectorAttribute(attribute.Name) {
		return
	}
	name, ok := parseModalSelector(attribute.Value)
	if !ok {
		return
	}
	if _, found := known[name]; found {
		return
	}
	key := modalUsageKey{
		sourcePath: nodeSourcePath(node, page.Source.SourcePath),
		line:       attribute.Location.Line,
		column:     attribute.Location.Column,
	}
	usage, exists := c.modalUsages[key]
	if !exists {
		usage = &modalUsage{
			selector:   attribute.Value,
			name:       name,
			suggestion: linguistics_domain.ClosestName(name, sortedNames(known)),
			pages:      nil,
			location:   attribute.Location,
		}
		c.modalUsages[key] = usage
	}
	if !slices.Contains(usage.pages, pageLabel) {
		usage.pages = append(usage.pages, pageLabel)
	}
}

// modalDiagnostics returns one warning for each modal selector that a page does not
// resolve, naming the pages.
//
// Returns []*ast_domain.Diagnostic which holds the warnings in source order.
func (c *referenceChecker) modalDiagnostics() []*ast_domain.Diagnostic {
	keys := slices.SortedFunc(maps.Keys(c.modalUsages), compareModalUsageKeys)
	diagnostics := make([]*ast_domain.Diagnostic, 0, len(keys))
	for _, key := range keys {
		usage := c.modalUsages[key]
		slices.Sort(usage.pages)
		pagesText := "page " + usage.pages[0]
		switch others := len(usage.pages) - 1; others {
		case 0:
		case 1:
			pagesText += " (and 1 other page)"
		default:
			pagesText += fmt.Sprintf(" (and %d other pages)", others)
		}
		message := fmt.Sprintf("Modal selector %q matches no element with modal=%q on %s or in any partial",
			usage.selector, usage.name, pagesText)
		message += didYouMean(usage.suggestion)
		diagnostics = append(diagnostics, ast_domain.NewDiagnosticWithCode(
			ast_domain.Warning, message, usage.selector, annotator_dto.CodeUnresolvedModalSelector,
			usage.location, key.sourcePath,
		))
	}
	return diagnostics
}

// newScriptLocator indexes the lines of a client script.
//
// Takes script (string) which is the client script.
// Takes start (ast_domain.Location) which is where its content begins in the source file.
//
// Returns *scriptLocator which places offsets in the source file.
func newScriptLocator(script string, start ast_domain.Location) *scriptLocator {
	return &scriptLocator{lines: jsscan.NewLineIndex(script), start: start}
}

// locate turns a byte offset in the script into a location in its source file.
//
// Takes offset (int) which is the byte offset in the script.
//
// Returns ast_domain.Location which is the location in the source file.
func (l *scriptLocator) locate(offset int) ast_domain.Location {
	line, column := l.lines.Position(offset)
	if line == 1 {
		column += max(l.start.Column, 1) - 1
	}
	return ast_domain.Location{Line: max(l.start.Line, 1) + line - 1, Column: column, Offset: 0}
}

// checkUnresolvedReferences reports literal references in the given components that no
// template in the build can resolve.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which holds every component.
// Takes components (map[string]*annotator_dto.VirtualComponent) which are the components
// to check, keyed by hashed name.
// Takes results (map[string]*annotator_dto.AnnotationResult) which hold the expanded
// templates, keyed by hashed name.
// Takes partialsSourceDir (string) which is the folder partial names are relative to.
//
// Returns []*ast_domain.Diagnostic which holds the warnings in a stable order.
func checkUnresolvedReferences(
	ctx context.Context,
	virtualModule *annotator_dto.VirtualModule,
	components map[string]*annotator_dto.VirtualComponent,
	results map[string]*annotator_dto.AnnotationResult,
	partialsSourceDir string,
) []*ast_domain.Diagnostic {
	if virtualModule == nil || len(components) == 0 {
		return nil
	}
	_, l := logger_domain.From(ctx, log)
	checker := newReferenceChecker(virtualModule, partialsSourceDir)
	var diagnostics []*ast_domain.Diagnostic
	for _, hashedName := range slices.Sorted(maps.Keys(components)) {
		diagnostics = append(diagnostics, checker.checkComponent(hashedName, components[hashedName], results[hashedName])...)
	}
	diagnostics = append(diagnostics, checker.modalDiagnostics()...)
	l.Internal("[ANNOTATOR] Checked literal references.",
		logger_domain.Int("components", len(components)),
		logger_domain.Int(attributeKeyDiagnosticCount, len(diagnostics)))
	return diagnostics
}

// hasPKScriptContext reports whether a component's client script runs with a pk context.
// Pages and partials do, and emails and PDFs do not.
//
// Takes component (*annotator_dto.VirtualComponent) which is the component to check.
//
// Returns bool which is true for pages and partials.
func hasPKScriptContext(component *annotator_dto.VirtualComponent) bool {
	return isPageComponent(component) || component.PartialName != ""
}

// isPageComponent reports whether a component is a page, including an error page.
//
// Takes component (*annotator_dto.VirtualComponent) which is the component to check.
//
// Returns bool which is true for pages.
func isPageComponent(component *annotator_dto.VirtualComponent) bool {
	return (component.IsPage || component.IsErrorPage) && !component.IsEmail && !component.IsPdf
}

// isBoundAttribute reports whether a node sets an attribute from an expression.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes name (string) which is the attribute name.
//
// Returns bool which is true when the attribute is bound.
func isBoundAttribute(node *ast_domain.TemplateNode, name string) bool {
	if _, bound := node.Binds[name]; bound {
		return true
	}
	return slices.ContainsFunc(node.DynamicAttributes, func(attribute ast_domain.DynamicAttribute) bool {
		return strings.EqualFold(attribute.Name, name)
	})
}

// isModalSelectorAttribute reports whether an attribute holds a modal selector.
//
// Takes name (string) which is the attribute name.
//
// Returns bool which is true for p-modal:selector and modal_selector.
func isModalSelectorAttribute(name string) bool {
	lowered := strings.ToLower(name)
	return lowered == attributeModalSelector || lowered == attributeModalSelectorProp
}

// parseModalSelector reads the modal name from the simple selector form [modal=name],
// [modal="name"] or [modal='name']. Any other selector is not read.
//
// Takes selector (string) which is the selector as written.
//
// Returns string which is the modal name.
// Returns bool which is false when the selector is not in the simple form.
func parseModalSelector(selector string) (string, bool) {
	inner, found := strings.CutPrefix(strings.TrimSpace(selector), modalSelectorPrefix)
	if !found {
		return "", false
	}
	inner, found = strings.CutSuffix(inner, modalSelectorSuffix)
	if !found {
		return "", false
	}
	for _, quote := range []string{`"`, `'`} {
		if len(inner) >= 2 && strings.HasPrefix(inner, quote) && strings.HasSuffix(inner, quote) {
			inner = inner[1 : len(inner)-1]
			break
		}
	}
	if inner == "" || strings.ContainsAny(inner, "[]\"'\\{}$,()=* \t\r\n") {
		return "", false
	}
	return inner, true
}

// nodeSourcePath returns the file a node was written in.
//
// Takes node (*ast_domain.TemplateNode) which is the node.
// Takes fallback (string) which is used when the node does not record its file.
//
// Returns string which is the node's source file.
func nodeSourcePath(node *ast_domain.TemplateNode, fallback string) string {
	if node.GoAnnotations != nil && node.GoAnnotations.OriginalSourcePath != nil && *node.GoAnnotations.OriginalSourcePath != "" {
		return *node.GoAnnotations.OriginalSourcePath
	}
	return fallback
}

// componentLabel names a component in messages by its module import path, or its source
// path when it has none.
//
// Takes component (*annotator_dto.VirtualComponent) which is the component to name.
//
// Returns string which is the label.
func componentLabel(component *annotator_dto.VirtualComponent) string {
	return cmp.Or(component.Source.ModuleImportPath, component.Source.SourcePath)
}

// didYouMean formats a suggestion for a message.
//
// Takes suggestion (string) which is the suggested name, or empty.
//
// Returns string which is the text to add to the message, or empty.
func didYouMean(suggestion string) string {
	if suggestion == "" {
		return ""
	}
	return fmt.Sprintf(". Did you mean '%s'?", suggestion)
}

// sortedNames returns the names of a set in order, so suggestions are stable.
//
// Takes names (map[string]struct{}) which is the set.
//
// Returns []string which holds the sorted names.
func sortedNames(names map[string]struct{}) []string {
	return slices.Sorted(maps.Keys(names))
}

// compareDiagnosticLocations orders diagnostics by line and then column.
//
// Takes a (*ast_domain.Diagnostic) which is the first diagnostic.
// Takes b (*ast_domain.Diagnostic) which is the second diagnostic.
//
// Returns int which is negative, zero or positive as a sorts before, with or after b.
func compareDiagnosticLocations(a, b *ast_domain.Diagnostic) int {
	return cmp.Or(cmp.Compare(a.Location.Line, b.Location.Line), cmp.Compare(a.Location.Column, b.Location.Column))
}

// compareModalUsageKeys orders modal selectors by file, line and column.
//
// Takes a (modalUsageKey) which is the first key.
// Takes b (modalUsageKey) which is the second key.
//
// Returns int which is negative, zero or positive as a sorts before, with or after b.
func compareModalUsageKeys(a, b modalUsageKey) int {
	return cmp.Or(strings.Compare(a.sourcePath, b.sourcePath), cmp.Compare(a.line, b.line), cmp.Compare(a.column, b.column))
}
