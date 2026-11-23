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
	"go/ast"
	"go/token"
	"strconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// subscriberRole is the reader identity prefix and the constant role in list claims.
	subscriberRole = "subscriber"
)

// authProvider renders the authentication fixture provider for one edition.
//
// Takes edition (string) which suffixes every ticket and identity literal.
// Takes paths ([]string) which lists the public request paths.
// Takes shape (string) which selects the publication role claim representation.
//
// Returns string which is the rendered provider source.
// Returns error which reports an unknown role shape or a stale fixture edit, or nil on
// success.
func authProvider(edition string, paths []string, shape string) (string, error) {
	claim, err := roleClaim(shape)
	if err != nil {
		return "", err
	}
	fields := []string{"roles", "PublicPaths", "reader-ticket", "editor-ticket", subscriberRole, "publisher"}
	return editFixture("internal/authfixture/provider.go", fields, func(node ast.Node) string {
		switch typed := node.(type) {
		case *ast.CaseClause:
			return editAuthCase(typed, claim, edition)
		case *ast.KeyValueExpr:
			return editAuthField(typed, paths, edition)
		}
		return ""
	})
}

// editAuthCase rewrites a provider switch case that names a ticket or the role claim.
//
// Takes clause (*ast.CaseClause) which is the candidate switch case.
// Takes claim (ast.Expr) which replaces the returned role claim.
// Takes edition (string) which replaces each ticket's sample suffix.
//
// Returns string which is the edit label, or empty when the case is not edited.
func editAuthCase(clause *ast.CaseClause, claim ast.Expr, edition string) string {
	if len(clause.List) != 1 {
		return ""
	}
	for _, prefix := range []string{"reader-ticket", "editor-ticket"} {
		if replaceEdition(clause.List[0], prefix, edition) {
			return prefix
		}
	}
	if len(clause.Body) != 1 {
		return ""
	}
	literal, isLiteral := clause.List[0].(*ast.BasicLit)
	statement, returns := clause.Body[0].(*ast.ReturnStmt)
	if isLiteral && literal.Value == `"roles"` && returns && len(statement.Results) == 1 {
		statement.Results[0] = claim
		return "roles"
	}
	return ""
}

// editAuthField rewrites a provider field that lists public paths or names an identity.
//
// Takes pair (*ast.KeyValueExpr) which is the candidate keyed field.
// Takes paths ([]string) which replaces the public path list.
// Takes edition (string) which replaces each identity's sample suffix.
//
// Returns string which is the edit label, or empty when the field is not edited.
func editAuthField(pair *ast.KeyValueExpr, paths []string, edition string) string {
	if named(pair.Key, "PublicPaths") {
		pair.Value = stringSlice(paths)
		return "PublicPaths"
	}
	if !named(pair.Key, "id") {
		return ""
	}
	for _, prefix := range []string{subscriberRole, "publisher"} {
		if replaceEdition(pair.Value, prefix, edition) {
			return prefix
		}
	}
	return ""
}

// checkoutPlace renders the checkout place action with the drawn SKU and unit price.
//
// Takes input (casefile.CheckoutInput) which supplies the SKU and unit price.
//
// Returns string which is the rendered action source.
// Returns error which reports a stale fixture edit, or nil on success.
func checkoutPlace(input casefile.CheckoutInput) (string, error) {
	return editFixture("actions/shop/place.go", []string{"sku", "unit", "total"}, func(node ast.Node) string {
		switch typed := node.(type) {
		case *ast.BinaryExpr:
			if _, literal := typed.Y.(*ast.BasicLit); literal && typed.Op == token.NEQ && selector(typed.X, "input", "SKU") {
				typed.Y = stringLiteral(input.SKU)
				return "sku"
			}
			if typed.Op == token.MUL && selector(typed.X, "input", "Quantity") {
				typed.Y = integerLiteral(input.Unit)
				return "total"
			}
		case *ast.KeyValueExpr:
			if named(typed.Key, "Unit") {
				typed.Value = integerLiteral(input.Unit)
				return "unit"
			}
		}
		return ""
	})
}

// formShimProvider renders the form fixture provider with the drawn quota, score and
// action name.
//
// Takes quota (int) which is the accepted submission quota.
// Takes score (int) which is the captcha score percentage.
// Takes action (string) which is the expected captcha action name.
//
// Returns string which is the rendered provider source.
// Returns error which reports a stale fixture edit, or nil on success.
func formShimProvider(quota, score int, action string) (string, error) {
	return editFixture("internal/formfixture/provider.go", []string{"Quota", "Score", "ActionName"}, func(node ast.Node) string {
		specification, ok := node.(*ast.ValueSpec)
		if !ok || len(specification.Names) != 1 || len(specification.Values) != 1 {
			return ""
		}
		switch name := specification.Names[0].Name; name {
		case "Quota":
			specification.Values[0] = integerLiteral(quota)
			return name
		case "Score":
			specification.Values[0] = integerLiteral(score)
			return name
		case "ActionName":
			specification.Values[0] = stringLiteral(action)
			return name
		}
		return ""
	})
}

// recordboardStore renders the record board model with the drawn labels and edits.
//
// Takes input (casefile.PartialComponentsInput) which supplies labels, edits, query and
// score.
// Takes selected (string) which is the initially selected record key.
//
// Returns string which is the rendered model source.
// Returns error which reports a stale fixture edit, or nil on success.
func recordboardStore(input casefile.PartialComponentsInput, selected string) (string, error) {
	return editFixture("pkg/recordboard/model.go", []string{"labels", "selected", "edited", "query", "score"}, func(node ast.Node) string {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			return editRecordboardAssignment(typed, input, selected)
		case *ast.CallExpr:
			if selector(typed.Fun, "url", "QueryEscape") && len(typed.Args) == 1 {
				typed.Args[0] = stringLiteral(input.Query)
				return "query"
			}
		}
		return ""
	})
}

// editRecordboardAssignment rewrites a record board assignment that seeds labels,
// selection, edits or scores.
//
// Takes assignment (*ast.AssignStmt) which is the candidate assignment.
// Takes input (casefile.PartialComponentsInput) which supplies labels, edits and score.
// Takes selected (string) which is the initially selected record key.
//
// Returns string which is the edit label, or empty when the assignment is not edited.
func editRecordboardAssignment(assignment *ast.AssignStmt, input casefile.PartialComponentsInput, selected string) string {
	if len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return ""
	}
	target := assignment.Lhs[0]
	switch {
	case assignment.Tok == token.DEFINE && named(target, "labels"):
		assignment.Rhs[0] = stringSlice(input.Labels[:])
		return "labels"
	case assignment.Tok == token.DEFINE && named(target, "selected"):
		assignment.Rhs[0] = stringLiteral(selected)
		return "selected"
	case assignment.Tok == token.ASSIGN && named(target, "label"):
		assignment.Rhs[0] = stringLiteral(input.Edited)
		return "edited"
	case assignment.Tok == token.DEFINE && named(target, "score"):
		if sum, ok := assignment.Rhs[0].(*ast.BinaryExpr); ok && sum.Op == token.ADD && named(sum.Y, "i") {
			sum.X = integerLiteral(input.Score)
			return "score"
		}
	}
	return ""
}

// editFixture renders a fixture after editing each intended source location exactly once.
//
// Takes path (string) which names the fixture within the fixture tree.
// Takes fields ([]string) which lists the edit labels that must each match once.
// Takes edit (func(ast.Node) string) which rewrites a node and returns its edit label, or
// an empty string when the node is not edited.
//
// Returns string which is the rendered fixture source.
// Returns error which reports a missing fixture or an edit that did not match exactly
// once, or nil on success.
func editFixture(path string, fields []string, edit func(ast.Node) string) (string, error) {
	source, err := fixturego.Render(path, func(_ *token.FileSet, file *ast.File) error {
		count := make(map[string]int)
		ast.Inspect(file, func(node ast.Node) bool {
			if field := edit(node); field != "" {
				count[field]++
			}
			return true
		})
		for _, field := range fields {
			if count[field] != 1 {
				return fmt.Errorf("%s: expected one edit, found %d", field, count[field])
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("rendering fixture %s: %w", path, err)
	}
	return source, nil
}

// named reports whether an expression is the identifier with the given name.
//
// Takes expression (ast.Expr) which is the expression to inspect.
// Takes name (string) which is the expected identifier name.
//
// Returns bool which reports whether the expression is that identifier.
func named(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

// selector reports whether an expression selects the given member of a named receiver.
//
// Takes expression (ast.Expr) which is the expression to inspect.
// Takes receiver (string) which is the expected receiver identifier.
// Takes member (string) which is the expected selected member.
//
// Returns bool which reports whether the expression is receiver.member.
func selector(expression ast.Expr, receiver, member string) bool {
	selection, ok := expression.(*ast.SelectorExpr)
	return ok && named(selection.X, receiver) && selection.Sel.Name == member
}

// stringLiteral builds a quoted Go string literal.
//
// Takes value (string) which is the literal's value.
//
// Returns *ast.BasicLit which is the quoted string literal.
func stringLiteral(value string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(value)}
}

// integerLiteral builds a Go integer literal, negated when the value is negative.
//
// Takes number (int) which is the literal's value.
//
// Returns ast.Expr which is the integer literal or its negation.
func integerLiteral(number int) ast.Expr {
	value := strconv.Itoa(number)
	if number < 0 {
		return &ast.UnaryExpr{Op: token.SUB, X: &ast.BasicLit{Kind: token.INT, Value: value[1:]}}
	}
	return &ast.BasicLit{Kind: token.INT, Value: value}
}

// sliceLiteral builds a slice composite literal of the named element type.
//
// Takes element (string) which names the slice element type.
// Takes values (...ast.Expr) which are the slice elements.
//
// Returns *ast.CompositeLit which is the slice literal.
func sliceLiteral(element string, values ...ast.Expr) *ast.CompositeLit {
	return &ast.CompositeLit{Type: &ast.ArrayType{Elt: ast.NewIdent(element)}, Elts: values}
}

// stringSlice builds a string slice literal from the given values.
//
// Takes values ([]string) which are the slice elements.
//
// Returns *ast.CompositeLit which is the string slice literal.
func stringSlice(values []string) *ast.CompositeLit {
	items := make([]ast.Expr, len(values))
	for position, value := range values {
		items[position] = stringLiteral(value)
	}
	return sliceLiteral("string", items...)
}

// roleClaim builds the publication role claim expression for a claim shape.
//
// Takes shape (string) which selects the claim representation.
//
// Returns ast.Expr which is the role claim expression.
// Returns error which reports an unknown claim shape, or nil on success.
func roleClaim(shape string) (ast.Expr, error) {
	role := &ast.SelectorExpr{X: ast.NewIdent("i"), Sel: ast.NewIdent("role")}
	converted := &ast.CallExpr{Fun: ast.NewIdent("publicationRole"), Args: []ast.Expr{role}}
	switch shape {
	case "string":
		return role, nil
	case "string-list":
		return sliceLiteral("string", stringLiteral(subscriberRole), role), nil
	case "stringer":
		return converted, nil
	case "mixed-list":
		return sliceLiteral("any", stringLiteral(subscriberRole), sliceLiteral("any", converted)), nil
	default:
		return nil, fmt.Errorf("unknown publication role shape %q", shape)
	}
}

// replaceEdition rewrites a "<prefix>-sample" string literal to "<prefix>-<edition>".
//
// Takes expression (ast.Expr) which is the candidate literal.
// Takes prefix (string) which is the literal's expected prefix.
// Takes edition (string) which replaces the sample suffix.
//
// Returns bool which reports whether the literal matched and was rewritten.
func replaceEdition(expression ast.Expr, prefix, edition string) bool {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING || literal.Value != strconv.Quote(prefix+"-sample") {
		return false
	}
	literal.Value = strconv.Quote(prefix + "-" + edition)
	return true
}
