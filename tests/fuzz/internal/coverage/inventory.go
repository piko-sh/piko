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

// Package coverage inventories directives recognised by Piko in generated SFCs.
package coverage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// MaxCases bounds the source inventory's preview cases.
	MaxCases = 512

	// counterComponent is the generated client component whose controls drive browser
	// actions.
	counterComponent = "components/fuzz-counter.pkc"

	// serverExtension is the file extension of server-rendered templates.
	serverExtension = ".pk"

	// clientExtension is the file extension of client components.
	clientExtension = ".pkc"

	// attributeID is the element identity attribute.
	attributeID = "id"

	// attributeType is the input type attribute.
	attributeType = "type"

	// attributeOutput is the attribute naming a generated output.
	attributeOutput = "data-fuzz-output"

	// directiveBind is the dynamic attribute binding directive.
	directiveBind = "p-bind"

	// directiveIf is the conditional directive.
	directiveIf = "p-if"

	// directiveElseIf is the chained conditional directive.
	directiveElseIf = "p-else-if"

	// directiveElse is the fallback conditional directive.
	directiveElse = "p-else"

	// directiveFor is the loop directive.
	directiveFor = "p-for"

	// directiveKey is the identity key directive.
	directiveKey = "p-key"

	// directiveText is the text content directive.
	directiveText = "p-text"

	// directiveHTML is the raw markup directive.
	directiveHTML = "p-html"

	// directiveContext is the scoped context directive.
	directiveContext = "p-context"

	// directiveModel is the two-way binding directive.
	directiveModel = "p-model"

	// directiveTimeline is the animation timeline directive.
	directiveTimeline = "p-timeline"

	// trueState is the serialised true state of an advanced component field.
	trueState = "true"

	// falseState is the serialised false state of an advanced component field.
	falseState = "false"

	// singleQuote delimits a literal string expression.
	singleQuote = "'"

	// selectTag is the select element's tag name.
	selectTag = "select"

	// clickEvent is the click event name.
	clickEvent = "click"

	// numberField names the number input type and its advanced state field.
	numberField = "number"

	// hideMemoKey is the advanced state field changed by the hide-memo action.
	hideMemoKey = "hidden"
)

var (
	// ErrInventoryFull reports an Add beyond MaxCases.
	ErrInventoryFull = errors.New("coverage inventory is full")

	// ErrMixedProfiles reports cases from different generator profiles in one inventory.
	ErrMixedProfiles = errors.New("mixed generator profiles")

	// directiveNotes explains the coverage boundary of selected directives.
	directiveNotes = map[string]string{
		"p-scaffold":  "Compiler-internal scaffold directive.",
		directiveBind: "Includes :attribute dynamic binding.",
		directiveKey: "Server key checks require a named namespace and a local key in the expected response. " +
			"Client identity requires expected DOM node IDs across a reorder or removal.",
		"p-on":            "Browser eligibility requires a targeted event with an independently asserted change.",
		"p-slot":          "PK partials only. Native HTML slots in client components do not imply p-slot.",
		"p-memo":          "PKC keyed-row memoisation applies during client reconciliation.",
		directiveTimeline: "PKC animation timelines require the component animation feature.",
	}

	// clientActionOps lists the browser actions the inventory counts.
	clientActionOps = []casefile.ClientOp{
		casefile.SetInput, casefile.SetLabel, casefile.ReverseRows, casefile.RemoveRow, casefile.ModifiedClick,
		casefile.ParentClick, casefile.ToggleCheck, casefile.SelectOption, casefile.ResetControls, casefile.ChooseRadio,
		casefile.SelectMany, casefile.TypeNumber, casefile.RefEdit, casefile.HideMemo, casefile.ShowRefocus,
		casefile.ChildPick, casefile.RemountChild, casefile.SeekTimeline,
	}

	// clickOps maps click-handling control IDs to their browser actions.
	clickOps = map[string]casefile.ClientOp{
		"client-modified": casefile.ModifiedClick,
		"client-parent":   casefile.ParentClick,
		"client-reverse":  casefile.ReverseRows,
		"client-remove":   casefile.RemoveRow,
		"client-reset":    casefile.ResetControls,
	}

	// modelOps maps extended model-bound control IDs to their browser actions.
	modelOps = map[string]casefile.ClientOp{
		"extra-red":    casefile.ChooseRadio,
		"extra-blue":   casefile.ChooseRadio,
		"extra-many":   casefile.SelectMany,
		"extra-number": casefile.TypeNumber,
		"extra-editor": casefile.RefEdit,
	}

	// refOps maps extended referenced element IDs to their browser actions.
	refOps = map[string]casefile.ClientOp{
		"extra-editor":   casefile.RefEdit,
		"extra-timeline": casefile.SeekTimeline,
	}
)

// Counts counts distinct cases containing a source feature.
type Counts struct {
	// SourceCases counts distinct source cases.
	SourceCases int

	// OracleEligibleCases counts source cases with independent checks.
	OracleEligibleCases int
}

// ActionCounts counts cases with a scripted interaction, plus the subset whose actual
// generated controls and independent post-action checks match the recipe.
type ActionCounts struct {
	// RecipeCases counts scripted cases.
	RecipeCases int

	// OracleEligibleCases counts scripted cases with matching independent checks.
	OracleEligibleCases int
}

// ProgramCounts compares expected leaves with matching IDs in each source file.
type ProgramCounts struct {
	// Cases counts generated program cases.
	Cases int

	// Requests counts program requests.
	Requests int

	// ExpectedOutputs counts expected output leaves.
	ExpectedOutputs int

	// MatchedOutputs counts expected leaves matched to source IDs.
	MatchedOutputs int

	// SourceOutputIDs counts output IDs found in the source.
	SourceOutputIDs int
}

// Directive compares a directive's server and client source coverage.
type Directive struct {
	// Name is the registered directive name.
	Name string

	// Note explains the directive's coverage boundary.
	Note string `json:",omitempty"`

	// Server holds server source coverage.
	Server Counts

	// Client holds client source coverage.
	Client Counts
}

// Scope summarises source features against a directive registry.
type Scope struct {
	// SourceOnly lists generated directives without independent checks.
	SourceOnly []string

	// Missing lists directives without generated source coverage.
	Missing []string

	// RegistryNames counts registered directive names.
	RegistryNames int

	// Seen counts generated directive names.
	Seen int

	// OracleEligible counts independently checkable directive names.
	OracleEligible int

	// DistinctTemplateShapes counts distinct generated template structures.
	DistinctTemplateShapes int
}

// Report compares generated source coverage with the directive registry.
type Report struct {
	// Programs holds generated-program source coverage.
	Programs map[string]ProgramCounts

	// ClientActions holds source and oracle coverage by browser action.
	ClientActions map[casefile.ClientOp]ActionCounts

	// Basis describes what the inventory measures.
	Basis string

	// ShapeDefinition describes what a distinct template shape is.
	ShapeDefinition string

	// Registry identifies the directive registry compared against.
	Registry string

	// Generator identifies the case generator.
	Generator string

	// ProfileSHA256 identifies the generator profile.
	ProfileSHA256 string

	// Target is the generated target.
	Target string

	// BrowserPolicy is the browser-sampling policy.
	BrowserPolicy string

	// Directives holds selected directive observations.
	Directives []Directive

	// Client holds client directive coverage.
	Client Scope

	// Server holds server directive coverage.
	Server Scope

	// BrowserSelected counts browser selections.
	BrowserSelected int

	// Cases counts previewed cases.
	Cases int

	// Start is the first stream position.
	Start uint64

	// Seed initialises the deterministic preview stream.
	Seed uint64
}

// Inventory accumulates generated-source coverage and distinct template shapes.
type Inventory struct {
	// server tracks distinct server template shapes.
	server map[[sha256.Size]byte]bool

	// client tracks distinct client template shapes.
	client map[[sha256.Size]byte]bool

	// Report holds the accumulated counts.
	Report
}

// caseScan accumulates one case's observations before they are added to the inventory.
type caseScan struct {
	// fuzzCase is the scanned case.
	fuzzCase *casefile.Case

	// seenServer maps server directive names to whether an independent check covers them.
	seenServer map[string]bool

	// seenClient maps client directive names to whether an independent check covers them.
	seenClient map[string]bool

	// actionControls records the browser actions the generated controls support.
	actionControls map[casefile.ClientOp]bool

	// perProgram holds program coverage by source file.
	perProgram map[string]ProgramCounts

	// serverShape accumulates the server template shape.
	serverShape strings.Builder

	// clientShape accumulates the client template shape.
	clientShape strings.Builder

	// directives lists the registry rows whose directives are noted.
	directives []Directive

	// browser selects browser-dependent oracle eligibility.
	browser bool
}

// reachability tracks output IDs reachable from a measured document root.
type reachability struct {
	// taken tracks branches reached by the independent model.
	taken map[*ast_domain.TemplateNode]bool

	// leaf selects reachable output IDs.
	leaf map[*ast_domain.TemplateNode]bool

	// counts accumulates program source coverage.
	counts ProgramCounts
}

// fileScan accumulates one template file's observations within a case scan.
type fileScan struct {
	// scan is the enclosing case scan.
	scan *caseScan

	// seen maps the file's directive names to whether an independent check covers them.
	seen map[string]bool

	// file is the saved PK or PKC source path.
	file string

	// program holds the file's program reachability.
	program reachability
}

// Add records source and independently checked coverage for a generated case.
//
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes browser (bool) which selects browser-dependent oracle eligibility without
// executing a browser.
//
// Returns error which reports cancellation, excessive case counts, mixed profiles or
// source parsing failure, or nil on success.
func (i *Inventory) Add(ctx context.Context, fuzzCase *casefile.Case, browser bool) error {
	if i.Cases >= MaxCases {
		return fmt.Errorf("%w: limit is %d cases", ErrInventoryFull, MaxCases)
	}
	if i.Cases > 0 && (i.Generator != fuzzCase.Generator || i.ProfileSHA256 != fuzzCase.ProfileSHA256) {
		return ErrMixedProfiles
	}
	scan := &caseScan{
		fuzzCase:       fuzzCase,
		seenServer:     map[string]bool{},
		seenClient:     map[string]bool{},
		actionControls: map[casefile.ClientOp]bool{},
		perProgram:     map[string]ProgramCounts{},
		serverShape:    strings.Builder{},
		clientShape:    strings.Builder{},
		directives:     i.Directives,
		browser:        browser,
	}
	for _, file := range slices.Sorted(maps.Keys(fuzzCase.Files)) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := scan.addFile(ctx, file); err != nil {
			return err
		}
	}
	i.recordPrograms(scan.perProgram)
	i.recordActions(fuzzCase, scan.actionControls, browser)
	i.Cases++
	if browser {
		i.BrowserSelected++
	}
	i.Generator, i.ProfileSHA256 = fuzzCase.Generator, fuzzCase.ProfileSHA256
	if scan.serverShape.Len() > 0 {
		i.server[sha256.Sum256([]byte(scan.serverShape.String()))] = true
	}
	if scan.clientShape.Len() > 0 {
		i.client[sha256.Sum256([]byte(scan.clientShape.String()))] = true
	}
	i.recordDirectives(scan.seenServer, scan.seenClient)
	return nil
}

// Result returns the accumulated source inventory.
//
// The returned report shares no mutable state with the inventory, so later Add calls do
// not change it.
//
// Returns Report which holds source counts, eligible checks and distinct template shapes.
func (i *Inventory) Result() Report {
	report := i.Report
	report.Directives = slices.Clone(i.Directives)
	report.ClientActions = maps.Clone(i.ClientActions)
	report.Programs = maps.Clone(i.Programs)
	report.Server, report.Client = i.summarise(false), i.summarise(true)
	report.Server.DistinctTemplateShapes = len(i.server)
	report.Client.DistinctTemplateShapes = len(i.client)
	return report
}

// recordPrograms adds one case's per-file program coverage.
//
// Takes perProgram (map[string]ProgramCounts) which holds the case's program coverage.
func (i *Inventory) recordPrograms(perProgram map[string]ProgramCounts) {
	for file, delta := range perProgram {
		programCounts := i.Programs[file]
		programCounts.Cases++
		programCounts.Requests += delta.Requests
		programCounts.ExpectedOutputs += delta.ExpectedOutputs
		programCounts.MatchedOutputs += delta.MatchedOutputs
		programCounts.SourceOutputIDs += delta.SourceOutputIDs
		i.Programs[file] = programCounts
	}
}

// recordActions counts the case's scripted browser actions and their eligible checks.
//
// Takes fuzzCase (*casefile.Case) which holds the saved expected results.
// Takes actionControls (map[casefile.ClientOp]bool) which records supported actions.
// Takes browser (bool) which selects browser-dependent oracle eligibility.
func (i *Inventory) recordActions(fuzzCase *casefile.Case, actionControls map[casefile.ClientOp]bool, browser bool) {
	transition := fuzzCase.Expected.Client
	if transition == nil {
		return
	}
	for op, counts := range i.ClientActions {
		if slices.ContainsFunc(transition.Steps, func(step casefile.ClientStep) bool { return step.Action.Op == op }) {
			counts.RecipeCases++
			if browser && actionControls[op] && stepOracle(transition, op) {
				counts.OracleEligibleCases++
			}
		}
		i.ClientActions[op] = counts
	}
}

// recordDirectives counts the case's seen and independently checked directives.
//
// Takes seenServer (map[string]bool) which records server directive observations.
// Takes seenClient (map[string]bool) which records client directive observations.
func (i *Inventory) recordDirectives(seenServer, seenClient map[string]bool) {
	for index := range i.Directives {
		row := &i.Directives[index]
		if ok, seen := seenServer[row.Name]; seen {
			row.Server.SourceCases++
			if ok {
				row.Server.OracleEligibleCases++
			}
		}
		if ok, seen := seenClient[row.Name]; seen {
			row.Client.SourceCases++
			if ok {
				row.Client.OracleEligibleCases++
			}
		}
	}
}

// summarise totals the registry's server or client directive coverage.
//
// Takes client (bool) which selects client rather than server coverage.
//
// Returns Scope which holds the registry, seen, eligible, missing and source-only
// directive counts.
func (i *Inventory) summarise(client bool) Scope {
	out := Scope{
		SourceOnly:             []string{},
		Missing:                []string{},
		RegistryNames:          len(i.Directives),
		Seen:                   0,
		OracleEligible:         0,
		DistinctTemplateShapes: 0,
	}
	for index := range i.Directives {
		row := &i.Directives[index]
		counts := row.Server
		if client {
			counts = row.Client
		}
		if counts.SourceCases == 0 {
			out.Missing = append(out.Missing, row.Name)
			continue
		}
		out.Seen++
		if counts.OracleEligibleCases > 0 {
			out.OracleEligible++
		} else {
			out.SourceOnly = append(out.SourceOnly, row.Name)
		}
	}
	return out
}

// addFile parses one template file and records its directives and shape.
//
// Takes file (string) which is the saved source path.
//
// Returns error which reports a parse failure or template diagnostics.
func (c *caseScan) addFile(ctx context.Context, file string) error {
	seen, shape := c.seenServer, &c.serverShape
	switch path.Ext(file) {
	case clientExtension:
		seen, shape = c.seenClient, &c.clientShape
	case serverExtension:
	default:
		return nil
	}
	sfc, err := sfcparser.Parse([]byte(c.fuzzCase.Files[file]))
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	tree, err := ast_domain.Parse(ctx, sfc.Template, file, nil)
	if err != nil {
		if tree != nil {
			ast_domain.PutTree(tree)
		}
		return fmt.Errorf("%s: %w", file, err)
	}
	defer ast_domain.PutTree(tree)
	if len(tree.Diagnostics) > 0 {
		return fmt.Errorf("%s: template diagnostics: %s", file, fmt.Sprint(tree.Diagnostics))
	}
	program := programReachability(c.fuzzCase, file, tree)
	if program.counts.Requests > 0 {
		c.perProgram[file] = program.counts
	}
	fmt.Fprintf(shape, "file %q\n", file)
	scan := fileScan{scan: c, seen: seen, file: file, program: program}
	for node, parent := range tree.NodesWithParent() {
		if file == counterComponent {
			markControls(node, c.actionControls)
		}
		attributes := scan.nodeAttributes(node, parent)
		fmt.Fprintf(shape, "%d:%q/%d %q\n", node.NodeType, node.TagName, len(node.Children), attributes)
	}
	return nil
}

// eligible reports whether a directive belongs to an independently checked output.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes name (string) which is the directive name.
// Takes arg (string) which is the directive argument.
//
// Returns bool which reports whether the directive belongs to an independently checked
// output.
func (r *reachability) eligible(node *ast_domain.TemplateNode, name, arg string) bool {
	switch name {
	case directiveFor, directiveIf, directiveElseIf, directiveElse:
		return r.taken[node]
	case directiveText:
		_, matched := r.leaf[node]
		return matched
	case directiveBind:
		return arg == "data-value" && r.leaf[node]
	}
	return false
}

// nodeAttributes notes a node's directives and lists its sorted attribute and directive
// names for the template shape.
//
// Raw directives may also be distributed into specialised AST fields. GetDirectives
// handles events and bind arguments. The per-case map keeps counts independent of
// repeated syntactic occurrences.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
//
// Returns []string which lists the sorted attribute names and directive observations.
func (f *fileScan) nodeAttributes(node, parent *ast_domain.TemplateNode) []string {
	attributes := make([]string, 0, len(node.Attributes))
	for index := range node.Attributes {
		attributes = append(attributes, node.Attributes[index].Name)
	}
	note := func(name, arg string) {
		attributes = append(attributes, name+":"+arg)
		f.note(node, parent, name, arg)
	}
	for index := range f.scan.directives {
		name := f.scan.directives[index].Name
		directives := node.GetDirectives(ast_domain.DirectiveNameToType[name])
		for directiveIndex := range directives {
			note(name, directives[directiveIndex].Arg)
		}
	}
	for name := range node.TimelineDirectives {
		note(directiveTimeline, name)
	}
	for index := range node.DynamicAttributes {
		note(directiveBind, node.DynamicAttributes[index].Name)
	}
	slices.Sort(attributes)
	return attributes
}

// note records one directive observation and whether an independent check covers it.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
// Takes name (string) which is the directive name.
// Takes arg (string) which is the directive argument.
func (f *fileScan) note(node, parent *ast_domain.TemplateNode, name, arg string) {
	found := f.seen[name]
	if path.Ext(f.file) == serverExtension || f.scan.browser {
		found = found || oracle(f.scan.fuzzCase, f.file, node, parent, name, arg) || f.program.eligible(node, name, arg)
	}
	f.seen[name] = found
}

// New initialises a source inventory for the selected target and browser policy.
//
// Takes target (string) which selects the generated target.
// Takes policy (string) which selects the browser-sampling policy.
// Takes seed (uint64) which initialises the deterministic stream.
// Takes start (uint64) which is the first stream position.
//
// Returns *Inventory which starts source accounting for the selected target and browser
// policy.
func New(target, policy string, seed, start uint64) *Inventory {
	inventory := new(Inventory)
	inventory.server = map[[sha256.Size]byte]bool{}
	inventory.client = map[[sha256.Size]byte]bool{}
	inventory.Target = target
	inventory.BrowserPolicy = policy
	inventory.Seed = seed
	inventory.Start = start
	inventory.Registry = "internal/ast/ast_domain/directive.go: DirectiveNameToType (registered directive names, " +
		"including internal directives; applicability differs by file type)"
	inventory.Programs = map[string]ProgramCounts{}
	inventory.Basis = "Piko SFC and template AST parsed. OracleEligibleCases counts source occurrences tied to an " +
		"independent expected response or selected browser snapshot. No server or browser executions are counted. " +
		"This inventory does not measure Go statement coverage."
	inventory.ShapeDefinition = "Distinct ordered SFC template trees across each complete project, including file " +
		"paths, node kinds, tags, attribute and directive names/arguments, and child counts. Literal values and " +
		"script/style contents are excluded."
	for _, name := range slices.Sorted(maps.Keys(ast_domain.DirectiveNameToType)) {
		inventory.Directives = append(inventory.Directives, Directive{
			Name:   name,
			Note:   directiveNotes[name],
			Server: Counts{SourceCases: 0, OracleEligibleCases: 0},
			Client: Counts{SourceCases: 0, OracleEligibleCases: 0},
		})
	}
	inventory.ClientActions = map[casefile.ClientOp]ActionCounts{}
	for _, op := range clientActionOps {
		inventory.ClientActions[op] = ActionCounts{RecipeCases: 0, OracleEligibleCases: 0}
	}
	return inventory
}

// attr reads a named attribute from a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes name (string) which is the attribute name.
//
// Returns string which is the selected attribute value, or an empty string when absent.
func attr(node *ast_domain.TemplateNode, name string) string {
	if node != nil {
		for index := range node.Attributes {
			if node.Attributes[index].Name == name {
				return node.Attributes[index].Value
			}
		}
	}
	return ""
}

// oracle checks whether a source directive has an independent saved assertion.
//
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes file (string) which is the saved PK or PKC source path.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
// Takes name (string) which is the directive name.
// Takes arg (string) which is the directive argument.
//
// Returns bool which reports whether the generated directive has a matching independent
// check.
func oracle(fuzzCase *casefile.Case, file string, node, parent *ast_domain.TemplateNode, name, arg string) bool {
	switch file {
	case "pages/main.pk":
		return mainOracle(fuzzCase, node, parent, name)
	case "pages/scoped.pk":
		return scopedOracle(fuzzCase, node, name)
	case "pages/partial-graph.pk", "pages/twin.pk":
		return slotOracle(fuzzCase, file, node, name)
	case "partials/badge.pk":
		return badgeOracle(fuzzCase, node, name)
	case counterComponent:
		return counterOracle(fuzzCase.Expected.Client, node, parent, name, arg)
	}
	return false
}

// mainOracle checks the main page's conditional and loop directives.
//
// Takes fuzzCase (*casefile.Case) which holds the saved expected results.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
// Takes name (string) which is the directive name.
//
// Returns bool which reports whether the directive has a matching independent check.
func mainOracle(fuzzCase *casefile.Case, node, parent *ast_domain.TemplateNode, name string) bool {
	switch name {
	case directiveIf:
		return attr(node, attributeID) == "case-visible" && !fuzzCase.Expected.Show
	case directiveFor:
		return attr(parent, attributeID) == "case-list" && len(fuzzCase.Expected.Rows) > 0
	}
	return false
}

// scopedOracle checks the scoped page's markup, context and key directives.
//
// Takes fuzzCase (*casefile.Case) which holds the saved probes.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes name (string) which is the directive name.
//
// Returns bool which reports whether the directive has a matching independent check.
func scopedOracle(fuzzCase *casefile.Case, node *ast_domain.TemplateNode, name string) bool {
	for index := range fuzzCase.Probes {
		probe := &fuzzCase.Probes[index]
		if !probe.Program || probe.Status != http.StatusOK || !strings.HasPrefix(probe.Path, "/scoped?") {
			continue
		}
		for outputIndex := range probe.Output {
			if scopedOutputOracle(node, name, &probe.Output[outputIndex]) {
				return true
			}
		}
	}
	return false
}

// scopedOutputOracle checks one scoped program output against a directive.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes name (string) which is the directive name.
// Takes output (*casefile.ProgramOutput) which is the expected program output.
//
// Returns bool which reports whether the output independently checks the directive.
func scopedOutputOracle(node *ast_domain.TemplateNode, name string, output *casefile.ProgramOutput) bool {
	if name == directiveHTML && attr(node, attributeID) == "rich-fragment" && output.ID == "markup" && output.HasValue {
		return true
	}
	scope, key, ok := strings.Cut(output.ScopedKey, "/")
	if !ok {
		return false
	}
	switch name {
	case directiveContext:
		return node.DirContext != nil && node.DirContext.RawExpression == singleQuote+scope+singleQuote
	case directiveKey:
		return attr(node, "data-fuzz-check-key") == trueState && attr(node, attributeOutput) == output.ID &&
			node.DirKey != nil && node.DirKey.RawExpression == singleQuote+key+singleQuote
	}
	return false
}

// slotOracle checks the caption slot of the graph and twin pages.
//
// Takes fuzzCase (*casefile.Case) which holds the saved sources and probes.
// Takes file (string) which is the saved page path.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes name (string) which is the directive name.
//
// Returns bool which reports whether the directive has a matching independent check.
func slotOracle(fuzzCase *casefile.Case, file string, node *ast_domain.TemplateNode, name string) bool {
	if name != "p-slot" || node.DirSlot == nil || node.DirSlot.RawExpression != "caption" || attr(node, attributeOutput) != "caption" ||
		!strings.Contains(fuzzCase.Files["partials/graph/frame.pk"], `<piko:slot name="caption">`) {
		return false
	}
	route := "/" + strings.TrimSuffix(strings.TrimPrefix(file, "pages/"), serverExtension) + "?"
	return slices.ContainsFunc(fuzzCase.Probes, func(probe casefile.Probe) bool {
		return probe.Program && probe.Status == http.StatusOK && strings.HasPrefix(probe.Path, route) && len(probe.Output) > 1 && probe.Output[0].ID == "caption"
	})
}

// badgeOracle checks the badge partial's text directive.
//
// Takes fuzzCase (*casefile.Case) which holds the saved probes.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes name (string) which is the directive name.
//
// Returns bool which reports whether the directive has a matching independent check.
func badgeOracle(fuzzCase *casefile.Case, node *ast_domain.TemplateNode, name string) bool {
	if name != directiveText || node.TagName != "span" {
		return false
	}
	return slices.ContainsFunc(fuzzCase.Probes, func(probe casefile.Probe) bool {
		return probe.Path == "/left" && probe.Text != "" && probe.Status == http.StatusOK
	})
}

// counterOracle checks the generated client component's directives.
//
// Takes transition (*casefile.ClientTransition) which holds independently expected
// component states.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
// Takes name (string) which is the directive name.
// Takes arg (string) which is the directive argument.
//
// Returns bool which reports whether the directive has a matching independent check.
func counterOracle(transition *casefile.ClientTransition, node, parent *ast_domain.TemplateNode, name, arg string) bool {
	if transition == nil || len(transition.Before) == 0 || len(transition.After) == 0 {
		return false
	}
	if result, decided := extendedOracle(transition, node, parent, name, arg); decided {
		return result
	}
	if op := advancedDirectiveOp(node, parent, name, arg); op != "" {
		return stepOracle(transition, op)
	}
	return basicCounterOracle(transition, node, parent, name, arg)
}

// extendedOracle checks the extended component's branch, markup, attribute and context
// directives.
//
// Takes transition (*casefile.ClientTransition) which holds independently expected
// component states.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
// Takes name (string) which is the directive name.
// Takes arg (string) which is the directive argument.
//
// Returns result (bool) which reports whether the directive has a matching independent
// check.
// Returns decided (bool) which reports whether the directive was one of these.
func extendedOracle(transition *casefile.ClientTransition, node, parent *ast_domain.TemplateNode, name, arg string) (result bool, decided bool) {
	id := attr(node, attributeID)
	if attr(parent, attributeID) == "extra-branches" {
		switch name {
		case directiveIf, directiveElseIf:
			return advancedChange(transition, casefile.RefEdit, "branch", "pristine", "editing"), true
		case directiveElse:
			return advancedChange(transition, casefile.HideMemo, "branch", "editing", hideMemoKey), true
		}
	}
	switch {
	case id == "extra-markup" && name == directiveHTML:
		return advancedChange(transition, casefile.HideMemo, "markup", "1|strong|open|Open & ready", "1|em|closed|Closed"), true
	case id == "extra-output" && name == directiveBind && arg == "data-phase":
		return advancedChange(transition, casefile.HideMemo, "phase-attr", "open", "shut"), true
	case id == "extra-output" && name == directiveBind && arg == "data-note":
		for _, step := range transition.Steps {
			if step.Action.Op == casefile.RefEdit {
				return advancedChange(transition, casefile.RefEdit, "note-attr", "", step.Action.Value), true
			}
		}
	case attr(parent, attributeID) == "extra-context" && name == directiveContext:
		return advancedChange(transition, casefile.HideMemo, "context-order", "alpha,beta", "beta,alpha") &&
			advancedChange(transition, casefile.HideMemo, "context-identity", trueState, trueState), true
	}
	return false, false
}

// basicCounterOracle checks the counter component's event, model, key, text, conditional
// and loop directives.
//
// Takes transition (*casefile.ClientTransition) which holds independently expected
// component states.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
// Takes name (string) which is the directive name.
// Takes arg (string) which is the directive argument.
//
// Returns bool which reports whether the directive has a matching independent check.
func basicCounterOracle(transition *casefile.ClientTransition, node, parent *ast_domain.TemplateNode, name, arg string) bool {
	id := attr(node, attributeID)
	initial, final := &transition.Before[0], &transition.After[0]
	switch name {
	case "p-on":
		if arg != clickEvent {
			return false
		}
		if id == "increment" {
			return final.Count != initial.Count
		}
		op, ok := clickOps[id]
		return ok && stepOracle(transition, op)
	case directiveModel:
		return modelOracle(transition, node, id)
	case directiveKey:
		return attr(parent, attributeID) == "client-list" && stepOracle(transition, casefile.ReverseRows) && stepOracle(transition, casefile.RemoveRow)
	case directiveText:
		return id == "count" && final.Count != initial.Count
	case directiveIf:
		return id == "client-visible" && final.Visible != initial.Visible
	case directiveFor:
		return attr(parent, attributeID) == "client-list" && len(initial.Rows) > 0 && !slices.Equal(initial.Rows, final.Rows)
	}
	return false
}

// modelOracle checks the counter component's model-bound controls.
//
// Takes transition (*casefile.ClientTransition) which holds independently expected
// component states.
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes id (string) which is the node's identity attribute.
//
// Returns bool which reports whether the control has a matching independent check.
func modelOracle(transition *casefile.ClientTransition, node *ast_domain.TemplateNode, id string) bool {
	switch id {
	case "client-input":
		return stepOracle(transition, casefile.SetInput)
	case "client-check":
		return node.TagName == "input" && attr(node, attributeType) == "checkbox" && stepOracle(transition, casefile.ToggleCheck)
	case "client-select":
		return node.TagName == selectTag && stepOracle(transition, casefile.SelectOption)
	}
	return false
}

// programReachability matches expected output IDs to reachable source nodes.
//
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes file (string) which is the saved PK or PKC source path.
// Takes tree (*ast_domain.TemplateAST) which is the parsed template AST.
//
// Returns reachability which holds reachable source IDs and their independent checks.
func programReachability(fuzzCase *casefile.Case, file string, tree *ast_domain.TemplateAST) reachability {
	var result reachability
	if !strings.HasPrefix(file, "pages/") {
		return result
	}
	route := "/" + strings.TrimSuffix(strings.TrimPrefix(file, "pages/"), serverExtension)
	expected, frequency := expectedOutputs(fuzzCase, route, &result.counts)
	if result.counts.Requests == 0 {
		return result
	}
	result.taken = map[*ast_domain.TemplateNode]bool{}
	result.leaf = map[*ast_domain.TemplateNode]bool{}
	parents := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{}
	for node, parent := range tree.NodesWithParent() {
		parents[node] = parent
		id := attr(node, attributeOutput)
		if node.TagName != "output" || id == "" {
			continue
		}
		root := programRoot(node, parents)
		if root == nil {
			continue
		}
		result.counts.SourceOutputIDs++
		binding, matched := expected[id]
		if !matched {
			continue
		}
		result.counts.MatchedOutputs += frequency[id]
		delete(frequency, id)
		result.leaf[node] = binding
		for current := node; current != root; current = parents[current] {
			result.taken[current] = true
		}
		result.taken[root] = true
	}
	return result
}

// expectedOutputs collects the expected program outputs of a page route.
//
// Takes fuzzCase (*casefile.Case) which holds the saved probes.
// Takes route (string) which is the page's request path.
// Takes counts (*ProgramCounts) which receives the request and expected-output counts.
//
// Returns map[string]bool which maps output IDs to whether any expects a value.
// Returns map[string]int which counts the expected occurrences of each output ID.
func expectedOutputs(fuzzCase *casefile.Case, route string, counts *ProgramCounts) (map[string]bool, map[string]int) {
	expected := map[string]bool{}
	frequency := map[string]int{}
	for index := range fuzzCase.Probes {
		probe := &fuzzCase.Probes[index]
		address, _, _ := strings.Cut(probe.Path, "?")
		if !probe.Program || probe.Status != http.StatusOK || address != route {
			continue
		}
		counts.Requests++
		counts.ExpectedOutputs += len(probe.Output)
		for outputIndex := range probe.Output {
			output := &probe.Output[outputIndex]
			frequency[output.ID]++
			expected[output.ID] = expected[output.ID] || output.HasValue
		}
	}
	return expected, frequency
}

// programRoot finds the enclosing fuzz-program element of a node.
//
// Takes node (*ast_domain.TemplateNode) which is the output node.
// Takes parents (map[*ast_domain.TemplateNode]*ast_domain.TemplateNode) which maps
// visited nodes to their parents.
//
// Returns *ast_domain.TemplateNode which is the program root, or nil when the node lies
// outside one.
func programRoot(node *ast_domain.TemplateNode, parents map[*ast_domain.TemplateNode]*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	root := node
	for root != nil && attr(root, attributeID) != "fuzz-program" {
		root = parents[root]
	}
	return root
}

// advancedChange checks the modelled state change for an extended component operation.
//
// Takes transition (*casefile.ClientTransition) which holds independently expected
// component states.
// Takes op (casefile.ClientOp) which selects the component action to inspect.
// Takes key (string) which names the advanced component state field.
// Takes from (string) which is the expected initial field value.
// Takes to (string) which is the expected final field value.
//
// Returns bool which reports whether the saved interaction models the requested field
// transition.
func advancedChange(transition *casefile.ClientTransition, op casefile.ClientOp, key, from, to string) bool {
	before := transition.After
	for index := range transition.Steps {
		step := &transition.Steps[index]
		instance := step.Action.Instance
		if step.Action.Op == op && instance >= 0 && instance < len(before) && instance < len(step.After) {
			old, okOld := before[instance].Advanced[key]
			next, okNext := step.After[instance].Advanced[key]
			return okOld && okNext && old == from && next == to
		}
		before = step.After
	}
	return false
}

// advancedDirectiveOp identifies the browser operation associated with a directive.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes parent (*ast_domain.TemplateNode) which is the enclosing template node.
// Takes name (string) which is the directive name.
// Takes arg (string) which is the directive argument.
//
// Returns casefile.ClientOp which identifies the matching browser operation, or is empty
// when none is mapped.
func advancedDirectiveOp(node, parent *ast_domain.TemplateNode, name, arg string) casefile.ClientOp {
	id := attr(node, attributeID)
	switch name {
	case directiveModel:
		return modelOps[id]
	case "p-memo", directiveKey, directiveFor:
		if attr(parent, attributeID) == "extra-memo" {
			return casefile.HideMemo
		}
	case "p-show", "p-class", "p-style":
		if id == "extra-panel" || id == "extra-visual" {
			return casefile.HideMemo
		}
	case "p-ref":
		return refOps[id]
	case "p-event":
		if node.TagName == "slot" && arg == "item-picked" {
			return casefile.ChildPick
		}
	case directiveTimeline:
		if id == "extra-timeline" && arg == hideMemoKey {
			return casefile.SeekTimeline
		}
	}
	return ""
}

// markControls records browser operations supported by generated controls.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes controls (map[casefile.ClientOp]bool) which receives supported browser
// operations.
func markControls(node *ast_domain.TemplateNode, controls map[casefile.ClientOp]bool) {
	id := attr(node, attributeID)
	markExtendedControl(node, id, controls)
	markClientControl(node, id, controls)
	if node.TagName == "slot" && attr(node, "name") == "nested-leaf" {
		controls[casefile.RemountChild] = true
		events := node.GetDirectives(ast_domain.DirectiveEvent)
		for index := range events {
			if events[index].Arg == "item-picked" {
				controls[casefile.ChildPick] = true
			}
		}
	}
	markClickControls(node, id, controls)
}

// markExtendedControl records the browser operation an extended component control
// supports.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes id (string) which is the node's identity attribute.
// Takes controls (map[casefile.ClientOp]bool) which receives supported browser
// operations.
func markExtendedControl(node *ast_domain.TemplateNode, id string, controls map[casefile.ClientOp]bool) {
	model := node.HasDirective(ast_domain.DirectiveModel)
	switch id {
	case "extra-blue":
		controls[casefile.ChooseRadio] = attr(node, attributeType) == "radio" && model
	case "extra-many":
		controls[casefile.SelectMany] = node.TagName == selectTag && model
	case "extra-number":
		controls[casefile.TypeNumber] = attr(node, attributeType) == numberField && model
	case "extra-editor":
		controls[casefile.RefEdit] = node.HasDirective(ast_domain.DirectiveRef) && model
	case "extra-toggle":
		controls[casefile.HideMemo], controls[casefile.ShowRefocus] = node.HasDirective(ast_domain.DirectiveOn), node.HasDirective(ast_domain.DirectiveOn)
	case "extra-timeline":
		controls[casefile.SeekTimeline] = node.TimelineDirectives[hideMemoKey] != nil && node.HasDirective(ast_domain.DirectiveRef)
	}
}

// markClientControl records the browser operation a counter component control supports.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes id (string) which is the node's identity attribute.
// Takes controls (map[casefile.ClientOp]bool) which receives supported browser
// operations.
func markClientControl(node *ast_domain.TemplateNode, id string, controls map[casefile.ClientOp]bool) {
	model := node.HasDirective(ast_domain.DirectiveModel)
	switch id {
	case "client-label":
		controls[casefile.SetLabel] = true
	case "client-check":
		controls[casefile.ToggleCheck] = node.TagName == "input" && attr(node, attributeType) == "checkbox" && model
	case "client-select":
		controls[casefile.SelectOption] = node.TagName == selectTag && model
	case "client-input":
		controls[casefile.SetInput] = model
	}
}

// markClickControls records the browser operations a node's click handlers support.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML or template node to inspect.
// Takes id (string) which is the node's identity attribute.
// Takes controls (map[casefile.ClientOp]bool) which receives supported browser
// operations.
func markClickControls(node *ast_domain.TemplateNode, id string, controls map[casefile.ClientOp]bool) {
	directives := node.GetDirectives(ast_domain.DirectiveOn)
	for index := range directives {
		directive := &directives[index]
		if directive.Arg != clickEvent {
			continue
		}
		op, ok := clickOps[id]
		if !ok {
			continue
		}
		if op == casefile.ModifiedClick {
			controls[op] = slices.Contains(directive.EventModifiers, "prevent") && slices.Contains(directive.EventModifiers, "stop")
			continue
		}
		controls[op] = true
	}
}

// stepOracle reports whether a saved browser action has an independent state check.
//
// Takes transition (*casefile.ClientTransition) which holds independently expected
// component states.
// Takes op (casefile.ClientOp) which selects the component action to inspect.
//
// Returns bool which reports whether the action has an independent saved result.
func stepOracle(transition *casefile.ClientTransition, op casefile.ClientOp) bool {
	before := transition.After
	for index := range transition.Steps {
		step := &transition.Steps[index]
		instance := step.Action.Instance
		if step.Action.Op == op && instance >= 0 && instance < len(before) && instance < len(step.After) {
			next, previous := &step.After[instance], &before[instance]
			if next.Controls == nil || previous.Controls == nil {
				return false
			}
			if result, decided := advancedStepChecked(op, step, next.Advanced, previous.Advanced); decided {
				return result
			}
			return controlStepChecked(op, step, next, previous)
		}
		before = step.After
	}
	return false
}

// advancedStepChecked checks an extended component action against its advanced state.
//
// Takes op (casefile.ClientOp) which is the performed action.
// Takes step (*casefile.ClientStep) which is the performed step.
// Takes next (map[string]string) which is the advanced state after the step.
// Takes previous (map[string]string) which is the advanced state before the step.
//
// Returns result (bool) which reports whether the step has an independent check.
// Returns decided (bool) which reports whether op is an extended component action.
func advancedStepChecked(op casefile.ClientOp, step *casefile.ClientStep, next, previous map[string]string) (result bool, decided bool) {
	switch op {
	case casefile.ChooseRadio:
		return next["radio"] == "blue" && next["radio-model"] == "blue" && previous["radio"] == "red", true
	case casefile.SelectMany:
		return next["multi"] == "a,b" && next["multi-model"] == "a,b" && previous["multi"] == "c", true
	case casefile.TypeNumber:
		return next["number-model"] == step.Action.Value && next[numberField] == step.Action.Value && previous[numberField] != next[numberField], true
	case casefile.RefEdit, casefile.ShowRefocus:
		return next["focus"] == trueState && previous["focus"] == falseState && next["editor-identity"] == trueState, true
	case casefile.HideMemo:
		return next["shown"] == falseState && previous["shown"] == trueState && next["memo"] == "changed,b", true
	default:
		return nestedStepChecked(op, next, previous)
	}
}

// nestedStepChecked checks a nested-component or timeline action against its advanced
// state.
//
// Takes op (casefile.ClientOp) which is the performed action.
// Takes next (map[string]string) which is the advanced state after the step.
// Takes previous (map[string]string) which is the advanced state before the step.
//
// Returns result (bool) which reports whether the step has an independent check.
// Returns decided (bool) which reports whether op is one of these actions.
func nestedStepChecked(op casefile.ClientOp, next, previous map[string]string) (result bool, decided bool) {
	switch op {
	case casefile.ChildPick:
		return next["parent-picked"] == next["leaf-count"] && next["leaf-count"] != previous["leaf-count"], true
	case casefile.RemountChild:
		return next["hooks"] == "2/1" && previous["hooks"] == "1/0" && next["leaf-identity"] == trueState, true
	case casefile.SeekTimeline:
		return next["timeline-hidden"] != previous["timeline-hidden"] && next["timeline-class"] != previous["timeline-class"] &&
			next["timeline-ref"] == trueState, true
	default:
		return false, false
	}
}

// controlStepChecked checks a counter component action against its rendered controls.
//
// Takes op (casefile.ClientOp) which is the performed action.
// Takes step (*casefile.ClientStep) which is the performed step.
// Takes next (*casefile.ClientInstance) which is the instance after the step.
// Takes previous (*casefile.ClientInstance) which is the instance before the step.
//
// Returns bool which reports whether the step has an independent check.
func controlStepChecked(op casefile.ClientOp, step *casefile.ClientStep, next, previous *casefile.ClientInstance) bool {
	switch op {
	case casefile.SetLabel:
		return next.Label == step.Action.Value && next.Label != previous.Label && next.Controls.HostLabel == next.Label && next.Controls.Input == next.Label
	case casefile.SetInput:
		return next.Label == step.Action.Value && next.Label != previous.Label && next.Controls.Input == next.Label
	case casefile.ReverseRows, casefile.RemoveRow:
		return len(next.Controls.Keys) > 0 && !slices.Equal(next.Controls.Keys, previous.Controls.Keys) && !slices.Equal(next.Rows, previous.Rows)
	case casefile.ModifiedClick:
		return step.Prevented && next.Controls.Handled != previous.Controls.Handled && next.Controls.Bubbled == previous.Controls.Bubbled
	case casefile.ParentClick:
		return next.Controls.Bubbled != previous.Controls.Bubbled
	case casefile.ToggleCheck, casefile.SelectOption, casefile.ResetControls:
		return formStepChecked(op, step, next.Controls.Form, previous.Controls.Form)
	default:
		return false
	}
}

// formStepChecked checks a native form control action against its reflected state.
//
// Takes op (casefile.ClientOp) which is the performed action.
// Takes step (*casefile.ClientStep) which is the performed step.
// Takes current (*casefile.ClientForm) which is the form state after the step.
// Takes prior (*casefile.ClientForm) which is the form state before the step.
//
// Returns bool which reports whether the step has an independent check.
func formStepChecked(op casefile.ClientOp, step *casefile.ClientStep, current, prior *casefile.ClientForm) bool {
	if current == nil || prior == nil {
		return false
	}
	label := "off"
	if current.Checked {
		label = "on"
	}
	checked := current.Checked != prior.Checked && current.Checked == current.HostChecked && current.CheckedState == label
	chosen := current.Choice != prior.Choice && current.Choice == current.ChoiceState && current.Choice == current.HostChoice
	switch op {
	case casefile.ToggleCheck:
		return checked
	case casefile.SelectOption:
		return chosen && current.Choice == step.Action.Value
	default:
		return checked && chosen
	}
}
