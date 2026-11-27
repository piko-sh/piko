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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"golang.org/x/net/html"
	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// pageExtension is the file extension of a Piko page or partial source.
	pageExtension = ".pk"

	// componentExtension is the file extension of a Piko client component source.
	componentExtension = ".pkc"
)

var (
	// errNoPikoSources reports a case without any PK or PKC source.
	errNoPikoSources = errors.New("case has no Piko sources")
)

// observedNode keeps the AST fields compared after formatting.
type observedNode struct {
	// Tag is the node's element name.
	Tag string

	// Text is the node's own raw text.
	Text string

	// Attributes lists the node's static and dynamic attributes in sorted order.
	Attributes []string

	// Directives lists the node's directives in sorted order.
	Directives []string

	// Children holds child nodes in document order.
	Children []observedNode

	// Kind is the parsed AST node kind.
	Kind ast_domain.NodeType
}

// templateSnapshot keeps markup structure and significant text.
type templateSnapshot struct {
	// markers maps marker kinds to their ordered values.
	markers map[string][]string

	// content holds ordered opening, text and closing markup tokens.
	content []string

	// attributes lists element attributes keyed by element position.
	attributes []string

	// prose lists whitespace-normalised inline phrases.
	prose []string
}

// recordMarkers records directives in their element and declaration order.
//
// Takes node (*html.Node) which is the HTML or template node to inspect.
// Takes element (int) which identifies the owning element in document order.
func (s *templateSnapshot) recordMarkers(node *html.Node, element int) {
	var id, ref string
	var markers []html.Attribute
	var attributes []string
	for _, attribute := range node.Attr {
		if !strings.HasPrefix(attribute.Key, "p-") || strings.HasPrefix(attribute.Key, "p-bind:") {
			key, value := attribute.Key, attribute.Val
			if bound, ok := strings.CutPrefix(key, "p-bind:"); ok {
				key = ":" + bound
			}
			switch key {
			case "checked", "disabled", "hidden", "readonly", "required", "multiple", "selected", "open", "autofocus", "inert", "novalidate":
				value = ""
			}
			attributes = append(attributes, fmt.Sprintf("%s=%q", key, value))
		}
		name, _, _ := strings.Cut(attribute.Key, ":")
		switch name {
		case "id":
			id = attribute.Val
		case "p-ref":
			ref = attribute.Val
		case "p-timeline", "p-slot", "p-memo", "p-event", "p-on":
			markers = append(markers, attribute)
		}
	}
	slices.Sort(attributes)
	for _, attribute := range attributes {
		s.attributes = append(s.attributes, fmt.Sprintf("%d <%s> %s", element, node.Data, attribute))
	}
	slices.SortStableFunc(markers, func(left, right html.Attribute) int {
		first, _, _ := strings.Cut(left.Key, ".")
		second, _, _ := strings.Cut(right.Key, ".")
		return strings.Compare(first, second)
	})
	for _, marker := range markers {
		name, _, _ := strings.Cut(marker.Key, ":")
		kind := strings.TrimPrefix(name, "p-")
		s.markers[kind] = append(s.markers[kind], fmt.Sprintf("%d <%s id=%q p-ref=%q> %s=%q", element, node.Data, id, ref, marker.Key, marker.Val))
	}
}

// eachSource parses each saved PK or PKC source and runs its check.
//
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes checkSource (func(string, string, *sfcparser.ParseResult) error) which checks one
// parsed source by path, text and parse result.
//
// Returns error which reports source parsing or the per-source callback's failure, or nil
// on success.
func eachSource(savedCase *casefile.Case, checkSource func(string, string, *sfcparser.ParseResult) error) error {
	count := 0
	for _, name := range slices.Sorted(maps.Keys(savedCase.Files)) {
		switch filepath.Ext(name) {
		case pageExtension, componentExtension:
		default:
			continue
		}
		content := savedCase.Files[name]
		blocks, err := sfcparser.Parse([]byte(content))
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		if blocks == nil || strings.TrimSpace(blocks.Template) == "" {
			return fmt.Errorf("%s has no template", name)
		}
		if err := checkSource(name, content, blocks); err != nil {
			if assertion, ok := errors.AsType[*propertyError](err); ok {
				return &propertyError{ID: assertion.ID, Source: name, Err: fmt.Errorf("%s: %w", name, err)}
			}
			return fmt.Errorf("%s: %w", name, err)
		}
		count++
	}
	if count == 0 {
		return errNoPikoSources
	}
	return nil
}

// checkSyntax checks that every generated source parses successfully.
//
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkSyntax(ctx context.Context, _ *tape.Tape, savedCase *casefile.Case) error {
	return eachSource(savedCase, func(name, source string, blocks *sfcparser.ParseResult) error {
		tree, err := ast_domain.Parse(ctx, blocks.Template, name, nil)
		if err != nil {
			return fmt.Errorf("parsing template: %w", err)
		}
		if tree == nil || len(tree.RootNodes) == 0 || len(tree.Diagnostics) != 0 {
			return property("parsed-tree", "invalid template AST: %+v", tree)
		}
		if !strings.Contains(source, blocks.Template) {
			return property("source-span", "template is absent from source")
		}
		if filepath.Ext(name) == componentExtension {
			return checkComponentName(savedCase, name, blocks)
		}
		return nil
	})
}

// checkComponentName checks a PKC source's custom-element registration name.
//
// Takes savedCase (*casefile.Case) which holds the expected component tag.
// Takes name (string) which is the PKC source path.
// Takes blocks (*sfcparser.ParseResult) which is the parsed source.
//
// Returns error which reports an invalid or unexpected registration name, or nil.
func checkComponentName(savedCase *casefile.Case, name string, blocks *sfcparser.ParseResult) error {
	tag := blocks.TemplateAttributes["name"]
	if tag == "" {
		tag = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	}
	if !strings.Contains(tag, "-") {
		return property("name", "component registration name %q must contain a hyphen", tag)
	}
	if filepath.Base(name) == savedCase.Expected.Tag+componentExtension && tag != savedCase.Expected.Tag {
		return property("name", "generated registration name %q differs from expected %q", tag, savedCase.Expected.Tag)
	}
	return nil
}

// projection keeps the AST structure used for semantic comparisons.
//
// Takes nodes ([]*ast_domain.TemplateNode) which holds the ordered template nodes.
//
// Returns []observedNode which lists observed node values in their recorded order.
func projection(nodes []*ast_domain.TemplateNode) []observedNode {
	result := make([]observedNode, 0, len(nodes))
	for _, templateNode := range nodes {
		node := observedNode{
			Kind:       templateNode.NodeType,
			Tag:        templateNode.TagName,
			Text:       templateNode.OwnRawText(),
			Attributes: nil,
			Directives: nil,
			Children:   projection(templateNode.Children),
		}
		for index := range templateNode.Attributes {
			attribute := &templateNode.Attributes[index]
			node.Attributes = append(node.Attributes, "static/"+attribute.Name+"="+attribute.Value)
		}
		for index := range templateNode.DynamicAttributes {
			attribute := &templateNode.DynamicAttributes[index]
			node.Attributes = append(node.Attributes, "dynamic/"+attribute.Name+"="+attribute.RawExpression)
		}
		for kind := ast_domain.DirectiveIf; kind <= ast_domain.DirectiveTimeline; kind++ {
			positions := map[string]int{}
			directives := templateNode.GetDirectives(kind)
			for directiveIndex := range directives {
				directive := &directives[directiveIndex]
				key := directive.Type.String() + "/" + directive.Arg
				node.Directives = append(node.Directives, fmt.Sprintf("%s[%d].%s=%s", key, positions[key], strings.Join(directive.EventModifiers, "."), directive.RawExpression))
				positions[key]++
			}
		}
		slices.Sort(node.Attributes)
		slices.Sort(node.Directives)
		result = append(result, node)
	}
	return result
}

// checkAST checks parsed source against the generated structural expectations.
//
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkAST(ctx context.Context, _ *tape.Tape, savedCase *casefile.Case) error {
	return eachSource(savedCase, func(name, _ string, blocks *sfcparser.ParseResult) error {
		tree, err := ast_domain.Parse(ctx, blocks.Template, name, nil)
		if err != nil {
			return fmt.Errorf("parsing template: %w", err)
		}
		want := projection(tree.RootNodes)
		wire, err := ast_adapters.EncodeAST(tree)
		if err != nil {
			return fmt.Errorf("encoding AST: %w", err)
		}
		got, err := ast_adapters.DecodeAST(ctx, wire)
		if err != nil {
			return fmt.Errorf("decoding AST: %w", err)
		}
		if !reflect.DeepEqual(projection(got.RootNodes), want) {
			return property("ast-roundtrip", "AST semantic projection changed after serialisation: got %+v, want %+v", projection(got.RootNodes), want)
		}
		return nil
	})
}

// checkFormatter checks source conservation and formatter idempotence.
//
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormatter(ctx context.Context, _ *tape.Tape, savedCase *casefile.Case) error {
	formatter := formatter_domain.NewFormatterService()
	checkSource := func(_, source string, blocks *sfcparser.ParseResult) error {
		return checkFormattedSource(ctx, formatter, source, blocks)
	}

	primary, others := partitionPreformatted(savedCase.Files)
	if len(primary) > 0 {
		subset := *savedCase
		subset.Files = primary
		if err := eachSource(&subset, checkSource); err != nil {
			return err
		}
		if len(others) == 0 {
			return nil
		}
	}
	subset := *savedCase
	subset.Files = others
	return eachSource(&subset, checkSource)
}

// partitionPreformatted splits Piko sources by whether they hold preformatted markup.
//
// Takes files (map[string]string) which maps source paths to their text.
//
// Returns primary (map[string]string) which holds sources with pre or textarea markup.
// Returns others (map[string]string) which holds the remaining Piko sources.
func partitionPreformatted(files map[string]string) (primary, others map[string]string) {
	primary, others = map[string]string{}, map[string]string{}
	for name, source := range files {
		if filepath.Ext(name) != pageExtension && filepath.Ext(name) != componentExtension {
			continue
		}
		if strings.Contains(source, "<pre>") || strings.Contains(source, "<textarea>") {
			primary[name] = source
		} else {
			others[name] = source
		}
	}
	return primary, others
}

// checkFormattedSource formats one source twice and checks what the formatter kept.
//
// Takes formatter (formatter_domain.FormatterService) which is the formatter under test.
// Takes source (string) which holds the original source text.
// Takes blocks (*sfcparser.ParseResult) which is the parsed original source.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormattedSource(ctx context.Context, formatter formatter_domain.FormatterService, source string, blocks *sfcparser.ParseResult) error {
	once, err := formatter.Format(ctx, []byte(source))
	if err != nil {
		return fmt.Errorf("formatting source: %w", err)
	}
	twice, err := formatter.Format(ctx, once)
	if err != nil {
		return fmt.Errorf("reformatting source: %w", err)
	}
	after, err := sfcparser.Parse(once)
	if err != nil {
		return fmt.Errorf("parsing formatted source: %w", err)
	}
	if err := checkPreformattedText(blocks, after); err != nil {
		return err
	}
	if err := checkFormattedTemplate(blocks, after); err != nil {
		return err
	}
	if !bytes.Equal(once, twice) {
		return property("idempotence", "formatting is not idempotent:\nfirst: %q\nsecond: %q", once, twice)
	}
	if len(blocks.Scripts) != len(after.Scripts) || len(blocks.Styles) != len(after.Styles) || strings.TrimSpace(after.Template) == "" {
		return property("block-count", "formatter lost an SFC block")
	}
	if !maps.Equal(after.TemplateAttributes, blocks.TemplateAttributes) {
		return property("template-attributes", "template attributes changed from %v to %v", blocks.TemplateAttributes, after.TemplateAttributes)
	}
	if err := checkFormattedScripts(blocks, after); err != nil {
		return err
	}
	return checkFormattedStyles(blocks, after)
}

// checkPreformattedText checks that visible preformatted text survives formatting.
//
// Takes before (*sfcparser.ParseResult) which holds the original source.
// Takes after (*sfcparser.ParseResult) which holds the formatted source.
//
// Returns error which reports changed preformatted text or malformed markup, or nil.
func checkPreformattedText(before, after *sfcparser.ParseResult) error {
	if !strings.Contains(before.Template, "<pre") && !strings.Contains(before.Template, "<textarea") {
		return nil
	}
	original, err := preformattedText(before.Template)
	if err != nil {
		return err
	}
	current, err := preformattedText(after.Template)
	if err != nil {
		return err
	}
	if !slices.Equal(current, original) {
		return property("preformatted-text", "visible preformatted text changed from %q to %q", original, current)
	}
	return nil
}

// checkFormattedScripts checks that script attributes and content survive formatting.
//
// Takes before (*sfcparser.ParseResult) which holds the original source.
// Takes after (*sfcparser.ParseResult) which holds the formatted source with the same
// number of scripts.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormattedScripts(before, after *sfcparser.ParseResult) error {
	for index := range before.Scripts {
		script, current := &before.Scripts[index], &after.Scripts[index]
		if !maps.Equal(current.Attributes, script.Attributes) {
			return property("script-attributes", "script %d changed attributes", index)
		}
		original, formatted := strings.TrimSpace(script.Content), strings.TrimSpace(current.Content)
		if script.IsGo() {
			canonical, err := format.Source([]byte(original))
			if err != nil {
				return fmt.Errorf("formatting original Go script %d: %w", index, err)
			}
			output, err := format.Source([]byte(formatted))
			if err != nil {
				return property("go-script", "formatted Go script %d is invalid: %w", index, err)
			}
			original, formatted = string(canonical), string(output)
		}
		if original != formatted {
			return property("script-content", "script %d changed content", index)
		}
	}
	return nil
}

// checkFormattedStyles checks that style attributes and minified content survive
// formatting.
//
// Takes before (*sfcparser.ParseResult) which holds the original source.
// Takes after (*sfcparser.ParseResult) which holds the formatted source with the same
// number of styles.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormattedStyles(before, after *sfcparser.ParseResult) error {
	minifier := minify.New()
	minifier.AddFunc("text/css", css.Minify)
	for index := range before.Styles {
		style, current := &before.Styles[index], &after.Styles[index]
		if !maps.Equal(current.Attributes, style.Attributes) {
			return property("style-attributes", "style %d changed attributes", index)
		}
		original, err := minifier.String("text/css", style.Content)
		if err != nil {
			return fmt.Errorf("minifying original style %d: %w", index, err)
		}
		formatted, err := minifier.String("text/css", current.Content)
		if err != nil || original != formatted {
			return property("style-content", "style %d changed from %q to %q (error %v)", index, original, formatted, err)
		}
	}
	return nil
}

// checkFormattedTemplate compares semantic markup before and after formatting.
//
// Takes before (*sfcparser.ParseResult) which holds the original source or observation.
// Takes after (*sfcparser.ParseResult) which holds the formatted source or observation.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormattedTemplate(before, after *sfcparser.ParseResult) error {
	if err := checkFormattedTranslations(before, after); err != nil {
		return err
	}
	if err := checkFormattedTimelines(before, after); err != nil {
		return err
	}
	want, err := templateContent(before.Template)
	if err != nil {
		return err
	}
	got, err := templateContent(after.Template)
	if err != nil {
		return err
	}
	return compareTemplateSnapshots(&want, &got)
}

// checkFormattedTranslations checks that translation blocks survive formatting.
//
// Takes before (*sfcparser.ParseResult) which holds the original source.
// Takes after (*sfcparser.ParseResult) which holds the formatted source.
//
// Returns error which reports a model mismatch, or nil when the check passes.
func checkFormattedTranslations(before, after *sfcparser.ParseResult) error {
	if len(before.I18nBlocks) != len(after.I18nBlocks) {
		return property("i18n-blocks", "got %d translation blocks, want %d", len(after.I18nBlocks), len(before.I18nBlocks))
	}
	for index := range before.I18nBlocks {
		block, other := &before.I18nBlocks[index], &after.I18nBlocks[index]
		if !maps.Equal(block.Attributes, other.Attributes) {
			return property("i18n-attributes", "translation block %d changed attributes", index)
		}
		if strings.TrimSpace(block.Content) == strings.TrimSpace(other.Content) {
			continue
		}
		if language := block.Attributes["lang"]; language != "" && language != "json" || !sameTranslations(block.Content, other.Content) {
			return property("i18n-content", "translation block %d changed content", index)
		}
	}
	return nil
}

// checkFormattedTimelines checks that timeline blocks keep their playback actions.
//
// Takes before (*sfcparser.ParseResult) which holds the original source.
// Takes after (*sfcparser.ParseResult) which holds the formatted source.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormattedTimelines(before, after *sfcparser.ParseResult) error {
	if len(before.Timelines) != len(after.Timelines) {
		return property("timeline-block-count", "got %d timeline blocks, want %d", len(after.Timelines), len(before.Timelines))
	}
	for index := range before.Timelines {
		original, current := &before.Timelines[index], &after.Timelines[index]
		if !maps.Equal(original.Attributes, current.Attributes) {
			return property("timeline-attributes", "timeline %d attributes changed from %v to %v", index, original.Attributes, current.Attributes)
		}

		want, err := compiler_domain.ParseTimeline(original.Content)
		if err != nil {
			return fmt.Errorf("source timeline %d: %w", index, err)
		}
		got, err := compiler_domain.ParseTimeline(current.Content)
		if err != nil {
			return property("timeline-actions", "formatted timeline %d: %w", index, err)
		}
		if got != want {
			return property("timeline-actions", "timeline %d actions changed from %s to %s", index, want, got)
		}
	}
	return nil
}

// compareTemplateSnapshots compares markers, attributes, prose and structure.
//
// Takes want (*templateSnapshot) which is the original template's snapshot.
// Takes got (*templateSnapshot) which is the formatted template's snapshot.
//
// Returns error which reports the first differing aspect, or nil when they match.
func compareTemplateSnapshots(want, got *templateSnapshot) error {
	for _, kind := range []string{"timeline", "slot", "memo", "event", "on"} {
		if !slices.Equal(want.markers[kind], got.markers[kind]) {
			return property(kind+"-markers", "p-%s bindings changed from %q to %q", kind, want.markers[kind], got.markers[kind])
		}
	}
	if !slices.Equal(want.attributes, got.attributes) {
		return property("template-attributes", "element attributes changed from %q to %q", want.attributes, got.attributes)
	}
	if !slices.Equal(want.prose, got.prose) {
		return property("inline-space", "inline text changed from %q to %q", want.prose, got.prose)
	}
	if !slices.Equal(want.content, got.content) {
		return property("template-content", "element structure or literal text changed from %q to %q", want.content, got.content)
	}
	return nil
}

// sameTranslations compares the decoded JSON values of translation sections.
//
// Takes before (string) which holds the original source or observation.
// Takes after (string) which holds the formatted source or observation.
//
// Returns bool which reports whether both JSON sections decode and contain equal values.
func sameTranslations(before, after string) bool {
	decode := func(text string) (any, bool) {
		if !json.Valid([]byte(text)) {
			return nil, false
		}
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		var value any
		err := decoder.Decode(&value)
		return value, err == nil
	}
	original, originalValid := decode(before)
	formatted, formattedValid := decode(after)
	return originalValid && formattedValid && reflect.DeepEqual(original, formatted)
}

// templateContent extracts markup structure and significant text for comparison.
//
// Takes source (string) which holds the saved source text.
//
// Returns templateSnapshot which records element structure, significant text and element
// markers.
// Returns error which reports malformed HTML template input, or nil on success.
func templateContent(source string) (templateSnapshot, error) {
	root, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return templateSnapshot{}, fmt.Errorf("parsing template HTML: %w", err)
	}
	result := templateSnapshot{markers: make(map[string][]string), content: nil, attributes: nil, prose: nil}
	element := 0
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		switch node.Type {
		case html.TextNode:
			if text := strings.Join(strings.Fields(node.Data), " "); text != "" {
				result.content = append(result.content, "text "+text)
			}
		case html.ElementNode:
			result.content = append(result.content, "open "+node.Data)
			result.recordMarkers(node, element)
			if words, ok := inlineProse(node); ok {
				result.prose = append(result.prose, words)
			}
			element++
		default:
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if node.Type == html.ElementNode {
			result.content = append(result.content, "close "+node.Data)
		}
	}
	visit(root)
	return result, nil
}

// inlineProse collects a complete inline phrase before normalising whitespace.
//
// Takes node (*html.Node) which is the HTML or template node to inspect.
//
// Returns string which is the collected normalised inline text.
// Returns bool which reports whether supported inline content was found.
func inlineProse(node *html.Node) (string, bool) {
	switch node.Data {
	case "p", "span", "a", "strong", "em", "b", "i", "small", "label", "button", "h1", "h2", "h3", "div":
	default:
		return "", false
	}
	var text strings.Builder
	hasInline := false
	for descendant := range node.Descendants() {
		switch descendant.Type {
		case html.TextNode:
			if strings.Contains(descendant.Data, "{{") {
				return "", false
			}
			text.WriteString(descendant.Data)
		case html.ElementNode:
			switch descendant.Data {
			case "span", "a", "b", "strong", "i", "em", "small", "code", "abbr", "mark", "s", "sub", "sup", "u":
				hasInline = true
			default:
				return "", false
			}
		default:
		}
	}
	return strings.Join(strings.Fields(text.String()), " "), hasInline
}

// preformattedText extracts whitespace-sensitive HTML text without normalisation.
//
// Takes template (string) which holds the template markup.
//
// Returns []string which lists pre and textarea contents in document order.
// Returns error which reports malformed HTML template input, or nil on success.
func preformattedText(template string) ([]string, error) {
	root, err := html.Parse(strings.NewReader(template))
	if err != nil {
		return nil, fmt.Errorf("parsing template HTML: %w", err)
	}
	var result []string
	for node := range root.Descendants() {
		if node.Type != html.ElementNode || (node.Data != "pre" && node.Data != "textarea") {
			continue
		}
		var content strings.Builder
		for child := range node.Descendants() {
			if child.Type == html.TextNode {
				content.WriteString(child.Data)
			}
		}
		result = append(result, content.String())
	}
	return result, nil
}
