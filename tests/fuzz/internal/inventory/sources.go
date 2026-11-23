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

// Package inventory measures authored input shapes separately from runtime results.
package inventory

import (
	"cmp"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// slotDimension is the composition dimension and element name of slotted content.
	slotDimension = "slot"

	// bindingFactParts is the number of parts in a binding fact: kind, tag and directive.
	bindingFactParts = 3
)

var (
	// bindingDimensions maps directive names to their composition dimensions.
	bindingDimensions = map[string]string{
		"p-if":       "conditional",
		"p-else-if":  "conditional",
		"p-else":     "conditional",
		"p-for":      "loop",
		"p-key":      "key",
		"p-on":       "event",
		"p-model":    "control",
		"p-timeline": "timeline",
	}

	// tagDimensions maps element names to their composition dimensions.
	tagDimensions = map[string]string{
		"piko:partial": "partial",
		"piko:slot":    slotDimension,
		"slot":         slotDimension,
		"form":         "form",
		"select":       "form",
		"input":        "form",
		"textarea":     "form",
		"audio":        "media",
		"video":        "media",
		"picture":      "media",
		"img":          "media",
		"piko:img":     "media",
		"piko:picture": "media",
	}
)

// Frequency counts a shape once per source file and once per distinct project.
type Frequency struct {
	// Files counts distinct source files.
	Files int

	// Projects counts distinct projects.
	Projects int
}

// Population counts source files, features and compositions.
type Population struct {
	// Extensions counts source files by extension.
	Extensions map[string]int

	// Features maps observed features to file and project frequencies.
	Features map[string]Frequency

	// Compositions counts source-feature combinations.
	Compositions map[string]Frequency

	// Diagnostics maps source paths to their diagnostic messages.
	Diagnostics map[string][]string `json:",omitempty"`

	// Projects counts source projects.
	Projects int

	// Files counts source files.
	Files int
}

// project accumulates source features for one authored project.
type project struct {
	// population holds this project's population totals.
	population *Population

	// seen tracks features already counted for this project.
	seen map[string]bool

	// dimensions tracks composition dimensions found in this project.
	dimensions map[string]bool

	// name identifies the source project in the report.
	name string
}

// file records one source file's features and compositions.
//
// Takes name (string) which is the source path within the project.
// Takes source (string) which holds the saved source text.
//
// Returns error which is the cancellation error, or nil while analysis completes.
func (p *project) file(ctx context.Context, name, source string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	facts := map[string]bool{}
	extension := path.Ext(name)
	p.population.Files++
	p.population.Extensions[extension]++
	diagnostic := func(err error) {
		if p.population.Diagnostics == nil {
			p.population.Diagnostics = map[string][]string{}
		}
		key := p.name + "/" + name
		p.population.Diagnostics[key] = append(p.population.Diagnostics[key], err.Error())
	}
	if extension == ".go" {
		imports(name, source, facts, diagnostic)
	} else {
		componentFacts(ctx, name, source, facts, diagnostic)
	}
	p.record(extension, facts)
	return ctx.Err()
}

// record adds one file's facts and compositions to the population totals.
//
// Takes extension (string) which is the source file extension.
// Takes facts (map[string]bool) which holds the file's detected source-feature flags.
func (p *project) record(extension string, facts map[string]bool) {
	dimensions := compositionDimensions(extension, facts)
	maps.Copy(p.dimensions, dimensions)
	for _, key := range compositionKeys(dimensions) {
		frequency := p.population.Compositions[key]
		frequency.Files++
		p.population.Compositions[key] = frequency
	}
	for fact := range facts {
		key := extension + "/" + fact
		frequency := p.population.Features[key]
		frequency.Files++
		if !p.seen[key] {
			frequency.Projects++
			p.seen[key] = true
		}
		p.population.Features[key] = frequency
	}
}

// finish commits this project's distinct feature and composition counts.
func (p *project) finish() {
	for _, key := range compositionKeys(p.dimensions) {
		frequency := p.population.Compositions[key]
		frequency.Projects++
		p.population.Compositions[key] = frequency
	}
}

// AddCase uses exactly the source walk used for repository fixtures.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns error which is the cancellation error, or nil while analysis completes.
func (p *Population) AddCase(ctx context.Context, c *casefile.Case) error {
	current := p.project(fmt.Sprintf("%s/%d/%d", c.Target, c.Seed, c.Index))
	for _, name := range slices.Sorted(maps.Keys(c.Files)) {
		if isSource(name) {
			if err := current.file(ctx, name, c.Files[name]); err != nil {
				return err
			}
		}
	}
	current.finish()
	return nil
}

// project starts feature accounting for one authored project.
//
// Takes name (string) which identifies the source project in the report.
//
// Returns *project which accumulates distinct project features in the shared population.
func (p *Population) project(name string) *project {
	if p.Features == nil {
		p.Features = map[string]Frequency{}
		p.Compositions = map[string]Frequency{}
		p.Extensions = map[string]int{}
	}
	p.Projects++
	return &project{name: name, population: p, seen: map[string]bool{}, dimensions: map[string]bool{}}
}

// compositionDimensions selects source-feature dimensions relevant to a file type.
//
// Takes extension (string) which is the source file extension.
// Takes facts (map[string]bool) which holds the detected source-feature flags.
//
// Returns map[string]bool which identifies source-feature dimensions present in the file.
func compositionDimensions(extension string, facts map[string]bool) map[string]bool {
	dimensions := map[string]bool{strings.TrimPrefix(extension, "."): true}
	for fact := range facts {
		parts := strings.Split(fact, "/")
		if len(parts) < 2 {
			continue
		}
		if dimension := factDimension(parts); dimension != "" {
			dimensions[dimension] = true
		}
	}
	return dimensions
}

// factDimension maps one split source-feature fact to its composition dimension.
//
// Takes parts ([]string) which holds the fact's slash-separated parts, at least two.
//
// Returns string which is the composition dimension, or empty when the fact has none.
func factDimension(parts []string) string {
	switch parts[0] {
	case "section":
		return "section-" + parts[1]
	case "slot-owner":
		return slotDimension
	case "import-alias":
		return "alias"
	case "dependency":
		return parts[1]
	case "source-namespace":
		if parts[1] != "html" {
			return parts[1]
		}
	case "binding":
		if len(parts) >= bindingFactParts {
			return bindingDimensions[parts[2]]
		}
	case "tag":
		return tagDimensions[parts[1]]
	}
	return ""
}

// compositionKeys lists the observed source-feature combinations.
//
// Takes dimensions (map[string]bool) which holds observed source-composition dimensions.
//
// Returns []string which lists deterministic feature pairs and triples from sorted
// dimension names.
func compositionKeys(dimensions map[string]bool) []string {
	names := slices.Sorted(maps.Keys(dimensions))
	var keys []string
	for leftIndex, left := range names {
		for rightIndex := leftIndex + 1; rightIndex < len(names); rightIndex++ {
			pair := left + " + " + names[rightIndex]
			keys = append(keys, pair)
			for _, third := range names[rightIndex+1:] {
				keys = append(keys, pair+" + "+third)
			}
		}
	}
	return keys
}

// imports records source import features and parse failures.
//
// Takes name (string) which is the source path within the project.
// Takes source (string) which holds the saved source text.
// Takes facts (map[string]bool) which receives the discovered source-feature flags.
// Takes diagnostic (func(error)) which records a source parse or analysis failure.
func imports(name, source string, facts map[string]bool, diagnostic func(error)) {
	file, err := parser.ParseFile(token.NewFileSet(), name, source, parser.ImportsOnly)
	if err != nil {
		diagnostic(err)
		return
	}
	for _, item := range file.Imports {
		destination, err := strconv.Unquote(item.Path.Value)
		if err != nil {
			diagnostic(err)
			continue
		}
		owner := path.Dir(name)
		facts["import/"+owner+" -> "+destination] = true
		if item.Name != nil {
			facts["import-alias/"+item.Name.Name] = true
		}
		if strings.HasSuffix(destination, ".pk") {
			facts["dependency/partial"] = true
		}
		if strings.Contains(destination, "/internal/") {
			facts["dependency/internal-package"] = true
		}
	}
}

// templateFacts records features present in the parsed template.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template AST.
// Takes facts (map[string]bool) which receives the discovered source-feature flags.
func templateFacts(tree *ast_domain.TemplateAST, facts map[string]bool) {
	spaces := map[*ast_domain.TemplateNode]string{}
	for node, parent := range tree.NodesWithParent() {
		space := nodeNamespace(spaces, node, parent)
		spaces[node] = space
		tag := strings.ToLower(node.TagName)
		if node.NodeType != ast_domain.NodeElement || tag == "" {
			continue
		}
		facts["tag/"+tag] = true
		facts["source-namespace/"+space+"/"+tag] = true
		if parent != nil && parent.TagName != "" {
			facts["parent/"+strings.ToLower(parent.TagName)+">"+tag] = true
		}
		attributeFacts(node, tag, facts)
		directiveFacts(node, tag, facts)
	}
}

// nodeNamespace selects the markup namespace a template node is authored in.
//
// Takes spaces (map[*ast_domain.TemplateNode]string) which holds the namespaces of
// already visited nodes.
// Takes node (*ast_domain.TemplateNode) which is the node to classify.
// Takes parent (*ast_domain.TemplateNode) which is the node's parent, or nil at the root.
//
// Returns string which is html, svg or mathml.
func nodeNamespace(spaces map[*ast_domain.TemplateNode]string, node, parent *ast_domain.TemplateNode) string {
	space := cmp.Or(spaces[parent], "html")
	if parent != nil && strings.EqualFold(parent.TagName, "foreignObject") {
		space = "html"
	}
	switch strings.ToLower(node.TagName) {
	case "svg":
		space = "svg"
	case "math":
		space = "mathml"
	}
	return space
}

// attributeFacts records an element's static attributes and slot ownership.
//
// Takes node (*ast_domain.TemplateNode) which is the element.
// Takes tag (string) which is the element's lower-case tag name.
// Takes facts (map[string]bool) which receives the discovered source-feature flags.
func attributeFacts(node *ast_domain.TemplateNode, tag string, facts map[string]bool) {
	for index := range node.Attributes {
		attribute := &node.Attributes[index]
		facts["attribute/"+tag+"/"+attribute.Name] = true
		if attribute.Name == slotDimension || (attribute.Name == "name" && (tag == slotDimension || tag == "piko:slot")) {
			facts["slot-owner/"+tag+"/"+attribute.Value] = true
		}
	}
}

// directiveFacts records an element's directives, timelines and dynamic attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the element.
// Takes tag (string) which is the element's lower-case tag name.
// Takes facts (map[string]bool) which receives the discovered source-feature flags.
func directiveFacts(node *ast_domain.TemplateNode, tag string, facts map[string]bool) {
	for name, kind := range ast_domain.DirectiveNameToType {
		directives := node.GetDirectives(kind)
		for index := range directives {
			directive := &directives[index]
			facts["binding/"+tag+"/"+name] = true
			if directive.Arg != "" {
				facts["argument/"+tag+"/"+name+"/"+directive.Arg] = true
			}
			if len(directive.EventModifiers) > 0 {
				facts["modifiers/"+name+"/"+strings.Join(directive.EventModifiers, ".")] = true
			}
			if kind == ast_domain.DirectiveSlot {
				facts["slot-owner/"+tag+"/"+directive.RawExpression] = true
			}
		}
	}
	for argument := range node.TimelineDirectives {
		facts["binding/"+tag+"/p-timeline"] = true
		facts["argument/"+tag+"/p-timeline/"+argument] = true
	}
	for index := range node.DynamicAttributes {
		attribute := &node.DynamicAttributes[index]
		facts["binding/"+tag+"/p-bind"] = true
		facts["argument/"+tag+"/p-bind/"+attribute.Name] = true
	}
}

// isSource reports whether a path is an inventoried source type.
//
// Takes name (string) which is the source file path.
//
// Returns bool which reports whether the extension belongs to an inventoried source type.
func isSource(name string) bool {
	extension := path.Ext(name)
	return extension == ".pk" || extension == ".pkc" || (extension == ".go" && !strings.HasSuffix(name, "_test.go"))
}

// componentFacts records the sections and template features of a component source.
//
// Takes name (string) which is the source path within the project.
// Takes source (string) which holds the saved source text.
// Takes facts (map[string]bool) which receives the discovered source-feature flags.
// Takes diagnostic (func(error)) which records a source parse or analysis failure.
func componentFacts(ctx context.Context, name, source string, facts map[string]bool, diagnostic func(error)) {
	blocks, err := sfcparser.Parse([]byte(source))
	if err != nil {
		diagnostic(err)
		return
	}
	for _, script := range blocks.Scripts {
		if script.IsGo() {
			facts["section/go"] = true
			imports(name, script.Content, facts, diagnostic)
		} else {
			facts["section/client-script"] = true
		}
	}
	if len(blocks.Styles) > 0 {
		facts["section/style"] = true
	}
	if len(blocks.I18nBlocks) > 0 {
		facts["section/i18n"] = true
	}
	if len(blocks.Timelines) > 0 {
		facts["section/timeline"] = true
	}
	tree, err := ast_domain.Parse(ctx, blocks.Template, name, nil)
	if err != nil {
		diagnostic(err)
		return
	}
	for _, treeDiagnostic := range tree.Diagnostics {
		diagnostic(treeDiagnostic)
	}
	templateFacts(tree, facts)
	ast_domain.PutTree(tree)
}
