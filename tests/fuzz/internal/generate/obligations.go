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

package generate

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// ArithmeticMatrix names the finite arithmetic-expression matrix.
	ArithmeticMatrix = "arithmetic trees and inputs"

	// TypedMatrix names the finite typed-value matrix.
	TypedMatrix = "typed values and expressions"

	// AuthPageMatrix names the finite authentication-page matrix.
	AuthPageMatrix = "authentication on page requests"

	// AuthActionMatrix names the finite publication-action matrix.
	AuthActionMatrix = "publication action and ledger"

	// FormShimMatrix names the finite form-client matrix.
	FormShimMatrix = "protected form submissions"

	// AuthClaimMatrix names the finite authentication-claim matrix.
	AuthClaimMatrix = "page role claim representation"

	// matrixServer names the server-only matrix environment.
	matrixServer = "server"

	// matrixBrowser names the browser matrix environment.
	matrixBrowser = "browser"

	// arithmeticCells is the arithmetic matrix's shape and operand-triple count.
	arithmeticCells = 486

	// authClaimCells is the number of role claim representations.
	authClaimCells = 4

	// authPageCells is the number of distinct authenticated page steps.
	authPageCells = 16

	// authActionCells is the number of distinct publication action steps.
	authActionCells = 4

	// formShimCells is the number of distinct protected form submissions.
	formShimCells = 7

	// graphCells is the number of partial graph shape and edit combinations.
	graphCells = 32

	// diagnosticCells is the number of paired diagnostic sources.
	diagnosticCells = 4

	// partialComponentCells is the number of partial owner, locale and reload orders.
	partialComponentCells = 8
)

// Matrix describes one deliberately finite generation contract.
type Matrix struct {
	// Name identifies the finite matrix.
	Name string

	// Environment names the host environment the matrix requires.
	Environment string

	// Cells lists the matrix cells represented by this case.
	Cells []string

	// Feasible is the finite matrix's number of possible cells.
	Feasible int
}

// Matrices lists finite generation cells represented by the saved case.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns []Matrix which lists matrix values in their recorded order.
func Matrices(c *casefile.Case) []Matrix {
	var result []Matrix
	switch c.Target {
	case "pages", "http-actions", "components":
		arithmetic, typed := programCells(c.Probes)
		result = append(result,
			newMatrix(ArithmeticMatrix, matrixServer, arithmeticCells, arithmetic),
			newMatrix(TypedMatrix, matrixServer, len(typedModes)*len(typedFields), typed))
	}
	return append(result, targetMatrices(c)...)
}

// targetMatrices lists the finite matrices specific to the case's target.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns []Matrix which lists the target's matrices, or nil when it has none.
func targetMatrices(c *casefile.Case) []Matrix {
	switch c.Target {
	case "auth-contracts":
		return authMatrices(c.Expected.Auth)
	case "form-shim-contracts":
		return formShimMatrices(c.Expected.FormShim)
	case "markup", "firefox-markup":
		return []Matrix{markupMatrix(c.Expected.DOM)}
	case "graph-rebuild":
		return []Matrix{graphMatrix(c.Features)}
	case "components":
		return []Matrix{componentMatrix(c.Features)}
	case "source-diagnostics":
		if diagnostic := c.Expected.Diagnostic; diagnostic != nil {
			return []Matrix{newMatrix("single-edit diagnostics", "package", diagnosticCells, []string{diagnostic.ID})}
		}
	default:
		return clientMatrices(c)
	}
	return nil
}

// clientMatrices lists the finite matrices of the browser-client targets.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns []Matrix which lists the target's matrices, or nil when it has none.
func clientMatrices(c *casefile.Case) []Matrix {
	client := c.Expected.Client
	if client == nil {
		return nil
	}
	switch {
	case c.Target == "partial-components" && client.PartialComponents != nil:
		input := client.PartialComponents.Input
		cell := fmt.Sprintf("%s/second=%t/reverse=%t", input.Locale, input.EditSecond, input.ReverseReload)
		return []Matrix{newMatrix("partial owner, locale and reload order", matrixBrowser, partialComponentCells, []string{cell})}
	case c.Target == "browser-events" && client.Events != nil:
		return []Matrix{newMatrix("event flag rotation", matrixBrowser, browserEventChoices, []string{strconv.Itoa(client.Events.Input.Rotate)})}
	case c.Target == "browser-lifecycle" && client.Lifecycle != nil:
		return []Matrix{newMatrix("request release order", matrixBrowser, 2, []string{strconv.FormatBool(client.Lifecycle.Input.Reverse)})}
	}
	return nil
}

// programCells lists the arithmetic and typed matrix cells covered by program outputs.
//
// Takes probes ([]casefile.Probe) which lists the case's requests.
//
// Returns arithmetic ([]string) which lists the arithmetic matrix cells.
// Returns typed ([]string) which lists the typed matrix cells.
func programCells(probes []casefile.Probe) (arithmetic, typed []string) {
	for index := range probes {
		probe := &probes[index]
		for _, output := range probe.Output {
			name, id := ProgramMatrixCell(*probe, output)
			switch name {
			case ArithmeticMatrix:
				arithmetic = append(arithmetic, id)
			case TypedMatrix:
				typed = append(typed, id)
			}
		}
	}
	return arithmetic, typed
}

// authMatrices lists the authentication claim, page and action matrices.
//
// Takes model (*casefile.AuthContract) which holds the modelled authentication contract,
// or nil.
//
// Returns []Matrix which lists the authentication matrices, or nil without a model.
func authMatrices(model *casefile.AuthContract) []Matrix {
	if model == nil {
		return nil
	}
	pages := make([]string, 0, len(model.Steps))
	for index := range model.Steps {
		pages = append(pages, model.Steps[index].Name)
	}
	actions := make([]string, 0, len(model.Actions))
	for index := range model.Actions {
		actions = append(actions, model.Actions[index].Name)
	}
	var result []Matrix
	if model.RoleClaim != "" {
		result = append(result, newMatrix(AuthClaimMatrix, matrixServer, authClaimCells, []string{model.RoleClaim}))
	}
	return append(result,
		newMatrix(AuthPageMatrix, matrixServer, authPageCells, pages),
		newMatrix(AuthActionMatrix, matrixServer, authActionCells, actions))
}

// formShimMatrices lists the protected form submission matrix.
//
// Takes model (*casefile.FormShimContract) which holds the modelled form contract, or
// nil.
//
// Returns []Matrix which lists the form matrix, or nil without a model.
func formShimMatrices(model *casefile.FormShimContract) []Matrix {
	if model == nil {
		return nil
	}
	cells := make([]string, 0, len(model.Steps))
	for index := range model.Steps {
		cells = append(cells, model.Steps[index].Name)
	}
	return []Matrix{newMatrix(FormShimMatrix, matrixServer, formShimCells, cells)}
}

// markupMatrix lists the HTML content families measured by the markup journey.
//
// Takes model (*casefile.DOMPlan) which holds the markup journey, or nil.
//
// Returns Matrix which holds the measured content families.
func markupMatrix(model *casefile.DOMPlan) Matrix {
	var cells []string
	if model != nil {
		for index := range model.Steps {
			if model.Steps[index].ID != "content-models" {
				continue
			}
			for _, field := range model.Steps[index].Fields {
				if family, ok := strings.CutSuffix(field.ID, "-parent"); ok {
					cells = append(cells, family)
				}
			}
		}
	}
	return newMatrix("HTML content families", matrixBrowser, len(markupFamilies), cells)
}

// graphMatrix lists the partial graph shape and edit covered by a rebuild case.
//
// Takes features ([]string) which lists the case's recorded features.
//
// Returns Matrix which holds the covered shape and edit cell.
func graphMatrix(features []string) Matrix {
	var shape, version string
	for _, feature := range features {
		if strings.HasPrefix(feature, "partial-graph-") && (strings.HasSuffix(feature, "-reverse") || strings.HasSuffix(feature, "-forward")) {
			shape = strings.TrimPrefix(feature, "partial-graph-")
		}
		if snapshot, ok := strings.CutPrefix(feature, "graph-snapshot-"); ok {
			version = snapshot
		}
	}
	var cells []string
	if version != "" && shape != "" {
		cells = append(cells, shape+"/"+version)
	}
	return newMatrix("partial graph and edit", matrixServer, graphCells, cells)
}

// componentMatrix lists the component recipe and family covered by a components case.
//
// Takes features ([]string) which lists the case's recorded features.
//
// Returns Matrix which holds the covered recipe and family cell.
func componentMatrix(features []string) Matrix {
	var recipe, family string
	for _, feature := range features {
		if value, ok := strings.CutPrefix(feature, "client-recipe-"); ok {
			recipe = value
		}
		if value, ok := strings.CutPrefix(feature, "client-extra-"); ok {
			family = value
		}
	}
	var cells []string
	if recipe != "" && family != "" {
		cells = append(cells, recipe+"/"+family)
	}
	return newMatrix("PKC action order and family", matrixBrowser, clientVariants*advancedFamilies, cells)
}

// newMatrix builds a matrix with its cells sorted and de-duplicated.
//
// Takes name (string) which identifies the finite matrix.
// Takes environment (string) which names the host environment the matrix requires.
// Takes feasible (int) which is the matrix's number of possible cells.
// Takes cells ([]string) which lists the covered cells and is sorted in place.
//
// Returns Matrix which holds the matrix and its distinct covered cells.
func newMatrix(name, environment string, feasible int, cells []string) Matrix {
	slices.Sort(cells)
	return Matrix{Name: name, Environment: environment, Cells: slices.Compact(cells), Feasible: feasible}
}

// ProgramMatrixCell identifies the finite matrix cell owning a program output.
//
// Takes probe (casefile.Probe) which is the request owning the expected output.
// Takes output (casefile.ProgramOutput) which is the expected output row whose matrix
// identity is inspected.
//
// Returns string which identifies the recognised matrix, or is empty for an unrelated
// output.
// Returns string which identifies the recognised cell, or is empty for an unrelated
// output.
func ProgramMatrixCell(probe casefile.Probe, output casefile.ProgramOutput) (name, id string) {
	switch {
	case probe.Program && probe.Path == "/arithmetic":
		return ArithmeticMatrix, output.ID + "/" + output.Value
	case probe.Program && strings.HasPrefix(probe.Path, "/typed?"):
		return TypedMatrix, probe.Path + "/" + output.ID
	default:
		return "", ""
	}
}
