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

package compiler_domain

import (
	"fmt"

	parsejs "github.com/tdewolff/parse/v2/js"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// convertStatement converts an esbuild statement to a tdewolff statement.
// It uses specialised converters based on the type of statement.
//
// Takes statement (js_ast.Stmt) which is the esbuild statement to convert.
//
// Returns parsejs.IStmt which is the converted tdewolff statement.
// Returns error when the conversion fails.
func (c *ASTConverter) convertStatement(statement js_ast.Stmt) (parsejs.IStmt, error) {
	if statement.Data == nil {
		return nil, nil
	}

	if result := c.tryConvertSimpleStmt(statement); result != nil {
		return result, nil
	}

	if result, handled, err := c.tryConvertDeclStmt(statement); handled {
		return result, err
	}

	if result, handled, err := c.tryConvertControlFlowStmt(statement); handled {
		return result, err
	}

	return c.convertExpressionStmt(statement)
}

// tryConvertSimpleStmt converts simple statements that need no extra handling.
//
// Takes statement (js_ast.Stmt) which is the statement to convert.
//
// Returns parsejs.IStmt which is the converted statement, or nil if the
// statement type is not simple.
func (c *ASTConverter) tryConvertSimpleStmt(statement js_ast.Stmt) parsejs.IStmt {
	switch s := statement.Data.(type) {
	case *js_ast.SBreak:
		return c.convertSBreak(s)
	case *js_ast.SContinue:
		return c.convertSContinue(s)
	case *js_ast.SEmpty:
		return &parsejs.EmptyStmt{}
	case *js_ast.SDebugger:
		return &parsejs.DebuggerStmt{}
	default:
		return nil
	}
}

// convertSBreak converts a break statement to a branch statement.
//
// Takes s (*js_ast.SBreak) which is the break statement to convert. If the
// statement has a label, the label name is resolved and included.
//
// Returns parsejs.IStmt which is the converted branch statement.
func (c *ASTConverter) convertSBreak(s *js_ast.SBreak) parsejs.IStmt {
	var label []byte
	if s.Label != nil {
		labelName := c.resolveRef(s.Label.Ref)
		if labelName != "" {
			label = []byte(labelName)
		}
	}
	return &parsejs.BranchStmt{Type: parsejs.BreakToken, Label: label}
}

// convertSContinue converts a continue statement to a branch statement.
//
// Takes s (*js_ast.SContinue) which is the continue statement to convert.
//
// Returns parsejs.IStmt which is the branch statement with an optional label.
func (c *ASTConverter) convertSContinue(s *js_ast.SContinue) parsejs.IStmt {
	var label []byte
	if s.Label != nil {
		labelName := c.resolveRef(s.Label.Ref)
		if labelName != "" {
			label = []byte(labelName)
		}
	}
	return &parsejs.BranchStmt{Type: parsejs.ContinueToken, Label: label}
}

// tryConvertDeclStmt handles declaration statements (functions, classes,
// variables, imports).
//
// Takes statement (js_ast.Stmt) which is the statement to convert.
//
// Returns parsejs.IStmt which is the converted statement, or nil if not a
// declaration.
// Returns bool which indicates whether the statement was a declaration type.
// Returns error when the conversion fails.
func (c *ASTConverter) tryConvertDeclStmt(statement js_ast.Stmt) (parsejs.IStmt, bool, error) {
	switch s := statement.Data.(type) {
	case *js_ast.SLocal:
		r, err := c.convertSLocal(s)
		return r, true, err
	case *js_ast.SFunction:
		r, err := c.convertSFunction(s)
		return r, true, err
	case *js_ast.SClass:
		r, err := c.convertSClass(s)
		return r, true, err
	case *js_ast.SImport:
		r, err := c.convertSImport(s)
		return r, true, err
	case *js_ast.SExportDefault:
		r, err := c.convertSExportDefault(s)
		return r, true, err
	default:
		return nil, false, nil
	}
}

// tryConvertControlFlowStmt handles control flow statements (if, for, while,
// switch, try).
//
// Takes statement (js_ast.Stmt) which is the statement to convert.
//
// Returns parsejs.IStmt which is the converted control flow statement.
// Returns bool which indicates whether the statement was a control flow type.
// Returns error when the conversion fails.
func (c *ASTConverter) tryConvertControlFlowStmt(statement js_ast.Stmt) (parsejs.IStmt, bool, error) {
	switch s := statement.Data.(type) {
	case *js_ast.SIf:
		r, err := c.convertSIf(s)
		return r, true, err
	case *js_ast.SFor:
		r, err := c.convertSFor(s)
		return r, true, err
	case *js_ast.SForIn:
		r, err := c.convertSForIn(s)
		return r, true, err
	case *js_ast.SForOf:
		r, err := c.convertSForOf(s)
		return r, true, err
	case *js_ast.SWhile:
		r, err := c.convertSWhile(s)
		return r, true, err
	case *js_ast.SDoWhile:
		r, err := c.convertSDoWhile(s)
		return r, true, err
	case *js_ast.SSwitch:
		r, err := c.convertSSwitch(s)
		return r, true, err
	case *js_ast.STry:
		r, err := c.convertSTry(s)
		return r, true, err
	case *js_ast.SLabel:
		r, err := c.convertSLabel(s)
		return r, true, err
	default:
		return nil, false, nil
	}
}

// convertExpressionStmt handles expression-based statements such as expr,
// return, block, and throw.
//
// Takes statement (js_ast.Stmt) which is the statement to convert.
//
// Returns parsejs.IStmt which is the converted statement representation.
// Returns error when conversion of the underlying expression fails.
func (c *ASTConverter) convertExpressionStmt(statement js_ast.Stmt) (parsejs.IStmt, error) {
	var result parsejs.IStmt
	var err error

	switch s := statement.Data.(type) {
	case *js_ast.SExpr:
		result, err = c.convertSExpr(s)
	case *js_ast.SReturn:
		result, err = c.convertSReturn(s)
	case *js_ast.SBlock:
		result, err = c.convertSBlock(s)
	case *js_ast.SThrow:
		result, err = c.convertSThrow(s)
	default:
		return &parsejs.EmptyStmt{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("converting %T statement: %w", statement.Data, err)
	}
	return result, nil
}

// convertSExpr converts an expression statement to its internal form.
//
// Takes s (*js_ast.SExpr) which provides the expression statement to convert.
//
// Returns parsejs.IStmt which is the converted statement.
// Returns error when the expression conversion fails.
func (c *ASTConverter) convertSExpr(s *js_ast.SExpr) (parsejs.IStmt, error) {
	expression, err := c.convertExpression(s.Value)
	if err != nil {
		return nil, fmt.Errorf("converting expression statement value: %w", err)
	}
	return &parsejs.ExprStmt{Value: expression}, nil
}

// convertSReturn converts a return statement from the JS AST.
//
// Takes s (*js_ast.SReturn) which is the return statement to convert.
//
// Returns parsejs.IStmt which is the converted return statement.
// Returns error when the return value cannot be converted.
func (c *ASTConverter) convertSReturn(s *js_ast.SReturn) (parsejs.IStmt, error) {
	if s.ValueOrNil.Data == nil {
		return &parsejs.ReturnStmt{}, nil
	}
	expression, err := c.convertExpression(s.ValueOrNil)
	if err != nil {
		return nil, fmt.Errorf("converting return statement value: %w", err)
	}
	return &parsejs.ReturnStmt{Value: expression}, nil
}

// convertSBlock converts a block statement to the internal format.
//
// Takes s (*js_ast.SBlock) which is the block statement to convert.
//
// Returns parsejs.IStmt which is the converted block statement.
// Returns error when a statement in the block fails to convert.
func (c *ASTConverter) convertSBlock(s *js_ast.SBlock) (parsejs.IStmt, error) {
	statements := make([]parsejs.IStmt, 0, len(s.Stmts))
	for i, statement := range s.Stmts {
		converted, err := c.convertStatement(statement)
		if err != nil {
			return nil, fmt.Errorf("converting block statement %d: %w", i, err)
		}
		if converted != nil {
			statements = append(statements, converted)
		}
	}
	return &parsejs.BlockStmt{List: statements}, nil
}

// convertSIf converts an esbuild if statement to a parsejs if statement.
//
// Takes s (*js_ast.SIf) which is the if statement to convert.
//
// Returns parsejs.IStmt which is the converted if statement.
// Returns error when the condition or body conversion fails.
func (c *ASTConverter) convertSIf(s *js_ast.SIf) (parsejs.IStmt, error) {
	test, err := c.convertExpression(s.Test)
	if err != nil {
		return nil, fmt.Errorf("converting if condition: %w", err)
	}
	yes, err := c.convertStatement(s.Yes)
	if err != nil {
		return nil, fmt.Errorf("converting if body: %w", err)
	}

	result := &parsejs.IfStmt{
		Cond: test,
		Body: yes,
	}

	if s.NoOrNil.Data != nil {
		no, err := c.convertStatement(s.NoOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting else branch: %w", err)
		}
		result.Else = no
	}

	return result, nil
}

// convertSWhile converts a while statement to the internal format.
//
// Takes s (*js_ast.SWhile) which is the while statement to convert.
//
// Returns parsejs.IStmt which is the converted while statement.
// Returns error when converting the test expression or body fails.
func (c *ASTConverter) convertSWhile(s *js_ast.SWhile) (parsejs.IStmt, error) {
	test, err := c.convertExpression(s.Test)
	if err != nil {
		return nil, fmt.Errorf("converting while condition: %w", err)
	}
	body, err := c.convertStatement(s.Body)
	if err != nil {
		return nil, fmt.Errorf("converting while body: %w", err)
	}
	return &parsejs.WhileStmt{Cond: test, Body: body}, nil
}

// convertSDoWhile converts a do-while statement.
//
// Takes s (*js_ast.SDoWhile) which is the do-while statement to convert.
//
// Returns parsejs.IStmt which is the converted do-while statement.
// Returns error when the body or test expression conversion fails.
func (c *ASTConverter) convertSDoWhile(s *js_ast.SDoWhile) (parsejs.IStmt, error) {
	body, err := c.convertStatement(s.Body)
	if err != nil {
		return nil, fmt.Errorf("converting do-while body: %w", err)
	}
	test, err := c.convertExpression(s.Test)
	if err != nil {
		return nil, fmt.Errorf("converting do-while condition: %w", err)
	}
	return &parsejs.DoWhileStmt{Cond: test, Body: body}, nil
}

// convertSLabel converts a labelled statement, such as "loop: for(...)" or
// "outer: while(...)".
//
// Takes s (*js_ast.SLabel) which is the labelled statement to convert.
//
// Returns parsejs.IStmt which is the converted labelled statement.
// Returns error when the inner statement cannot be converted.
func (c *ASTConverter) convertSLabel(s *js_ast.SLabel) (parsejs.IStmt, error) {
	labelName := c.resolveRef(s.Name.Ref)
	if labelName == "" {
		labelName = "label"
	}

	statement, err := c.convertStatement(s.Stmt)
	if err != nil {
		return nil, fmt.Errorf("converting labelled statement %q: %w", labelName, err)
	}

	return &parsejs.LabelledStmt{
		Label: []byte(labelName),
		Value: statement,
	}, nil
}

// convertSThrow converts a throw statement.
//
// Takes s (*js_ast.SThrow) which contains the throw statement to convert.
//
// Returns parsejs.IStmt which is the converted throw statement.
// Returns error when the value expression cannot be converted.
func (c *ASTConverter) convertSThrow(s *js_ast.SThrow) (parsejs.IStmt, error) {
	expression, err := c.convertExpression(s.Value)
	if err != nil {
		return nil, fmt.Errorf("converting throw statement value: %w", err)
	}
	return &parsejs.ThrowStmt{Value: expression}, nil
}

// convertSLocal converts a local variable declaration.
//
// Takes s (*js_ast.SLocal) which is the local variable statement to convert.
//
// Returns parsejs.IStmt which is the converted variable declaration.
// Returns error when binding or expression conversion fails.
func (c *ASTConverter) convertSLocal(s *js_ast.SLocal) (parsejs.IStmt, error) {
	tokenType := getLocalTokenType(s.Kind)

	bindings := make([]parsejs.BindingElement, 0, len(s.Decls))
	for _, declaration := range s.Decls {
		binding, err := c.convertBinding(declaration.Binding)
		if err != nil {
			return nil, fmt.Errorf("converting local declaration binding: %w", err)
		}

		var defaultExpr parsejs.IExpr
		if declaration.ValueOrNil.Data != nil {
			defaultExpr, err = c.convertExpression(declaration.ValueOrNil)
			if err != nil {
				return nil, fmt.Errorf("converting local declaration value: %w", err)
			}
		}

		bindings = append(bindings, parsejs.BindingElement{
			Binding: binding,
			Default: defaultExpr,
		})
	}

	return &parsejs.VarDecl{
		TokenType: tokenType,
		List:      bindings,
	}, nil
}

// convertSFunction converts a function statement to a function declaration.
//
// Takes s (*js_ast.SFunction) which is the function statement to convert.
//
// Returns parsejs.IStmt which is the converted function declaration.
// Returns error when parameter or body conversion fails.
func (c *ASTConverter) convertSFunction(s *js_ast.SFunction) (parsejs.IStmt, error) {
	params, err := c.convertParams(s.Fn.Args)
	if err != nil {
		return nil, fmt.Errorf("converting function parameters: %w", err)
	}

	body, err := c.convertFunctionBody(s.Fn.Body)
	if err != nil {
		return nil, fmt.Errorf("converting function body: %w", err)
	}

	name := "anonymous"
	if s.Fn.Name != nil {
		var regName string
		if c.registry != nil {
			regName = c.registry.LookupLocRefName(s.Fn.Name)
		}
		if regName != "" {
			name = regName
		} else if resolved := c.resolveRef(s.Fn.Name.Ref); resolved != "" {
			name = resolved
		}
	}

	return &parsejs.FuncDecl{
		Async:     s.Fn.IsAsync,
		Generator: s.Fn.IsGenerator,
		Name:      &parsejs.Var{Data: []byte(name)},
		Params:    params,
		Body:      *body,
	}, nil
}

// convertSClass converts a class statement to a class declaration.
//
// Takes s (*js_ast.SClass) which contains the class statement to convert.
//
// Returns parsejs.IStmt which is the converted class declaration.
// Returns error when the extends clause or class properties cannot be
// converted.
func (c *ASTConverter) convertSClass(s *js_ast.SClass) (parsejs.IStmt, error) {
	var extends parsejs.IExpr
	if s.Class.ExtendsOrNil.Data != nil {
		var err error
		extends, err = c.convertExpression(s.Class.ExtendsOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting class extends clause: %w", err)
		}
	}

	elements := make([]parsejs.ClassElement, 0, len(s.Class.Properties))
	for _, prop := range s.Class.Properties {
		element, err := c.convertClassProperty(prop)
		if err != nil {
			return nil, fmt.Errorf("converting class property: %w", err)
		}
		if element != nil {
			elements = append(elements, *element)
		}
	}

	name := "AnonymousClass"
	if s.Class.Name != nil {
		var regName string
		if c.registry != nil {
			regName = c.registry.LookupLocRefName(s.Class.Name)
		}
		if regName != "" {
			name = regName
		} else if resolved := c.resolveRef(s.Class.Name.Ref); resolved != "" {
			name = resolved
		}
	}

	return &parsejs.ClassDecl{
		Name:    &parsejs.Var{Data: []byte(name)},
		Extends: extends,
		List:    elements,
	}, nil
}

// convertSFor converts a for statement.
//
// Takes s (*js_ast.SFor) which is the esbuild for statement to convert.
//
// Returns parsejs.IStmt which is the converted for statement.
// Returns error when the test or update expression conversion fails.
func (c *ASTConverter) convertSFor(s *js_ast.SFor) (parsejs.IStmt, error) {
	init := c.convertForInit(s.InitOrNil)

	var test parsejs.IExpr
	if s.TestOrNil.Data != nil {
		var err error
		test, err = c.convertExpression(s.TestOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting for-loop test: %w", err)
		}
	}

	var update parsejs.IExpr
	if s.UpdateOrNil.Data != nil {
		var err error
		update, err = c.convertExpression(s.UpdateOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting for-loop update: %w", err)
		}
	}

	body, err := c.convertForBody(s.Body)
	if err != nil {
		return nil, fmt.Errorf("converting for-loop body: %w", err)
	}

	return &parsejs.ForStmt{
		Init: init,
		Cond: test,
		Post: update,
		Body: body,
	}, nil
}

// convertForInit converts the init part of a for loop.
//
// Takes initStmt (js_ast.Stmt) which is the initialisation statement to
// convert.
//
// Returns parsejs.IExpr which is the converted expression, or nil if the
// statement cannot be converted.
func (c *ASTConverter) convertForInit(initStmt js_ast.Stmt) parsejs.IExpr {
	if initStmt.Data == nil {
		return nil
	}

	switch init := initStmt.Data.(type) {
	case *js_ast.SLocal:
		varDecl, err := c.convertSLocal(init)
		if err != nil {
			return nil
		}
		if vd, ok := varDecl.(*parsejs.VarDecl); ok {
			return vd
		}
	case *js_ast.SExpr:
		expression, err := c.convertExpression(init.Value)
		if err != nil {
			return nil
		}
		return expression
	}
	return nil
}

// convertForBody converts the body of a for loop.
//
// Takes bodyStmt (js_ast.Stmt) which is the statement forming the loop body.
//
// Returns *parsejs.BlockStmt which wraps the converted body as a block.
// Returns error when the body statement cannot be converted.
func (c *ASTConverter) convertForBody(bodyStmt js_ast.Stmt) (*parsejs.BlockStmt, error) {
	converted, err := c.convertStatement(bodyStmt)
	if err != nil {
		return nil, fmt.Errorf("converting for-loop body statement: %w", err)
	}

	if block, ok := converted.(*parsejs.BlockStmt); ok {
		return block, nil
	}
	if converted != nil {
		return &parsejs.BlockStmt{List: []parsejs.IStmt{converted}}, nil
	}
	return nil, nil
}

// convertSForIn converts a for-in statement.
//
// Takes s (*js_ast.SForIn) which is the esbuild for-in statement to convert.
//
// Returns parsejs.IStmt which is the converted for-in statement.
// Returns error when the value or body expression cannot be converted.
func (c *ASTConverter) convertSForIn(s *js_ast.SForIn) (parsejs.IStmt, error) {
	init := c.convertForInit(s.Init)

	value, err := c.convertExpression(s.Value)
	if err != nil {
		return nil, fmt.Errorf("converting for-in value: %w", err)
	}

	body, err := c.convertForBody(s.Body)
	if err != nil {
		return nil, fmt.Errorf("converting for-in body: %w", err)
	}

	return &parsejs.ForInStmt{
		Init:  init,
		Value: value,
		Body:  body,
	}, nil
}

// convertSForOf converts a for-of statement.
//
// Takes s (*js_ast.SForOf) which is the esbuild for-of statement to convert.
//
// Returns parsejs.IStmt which is the converted for-of statement.
// Returns error when the value expression or body conversion fails.
func (c *ASTConverter) convertSForOf(s *js_ast.SForOf) (parsejs.IStmt, error) {
	init := c.convertForInit(s.Init)

	value, err := c.convertExpression(s.Value)
	if err != nil {
		return nil, fmt.Errorf("converting for-of value: %w", err)
	}

	body, err := c.convertForBody(s.Body)
	if err != nil {
		return nil, fmt.Errorf("converting for-of body: %w", err)
	}

	return &parsejs.ForOfStmt{
		Await: s.Await.Len > 0,
		Init:  init,
		Value: value,
		Body:  body,
	}, nil
}

// convertSTry converts a try statement to the internal representation.
//
// Takes s (*js_ast.STry) which is the try statement to convert.
//
// Returns parsejs.IStmt which is the converted try statement.
// Returns error when the body, catch, or finally block conversion fails.
func (c *ASTConverter) convertSTry(s *js_ast.STry) (parsejs.IStmt, error) {
	body, err := c.convertFunctionBody(js_ast.FnBody{Block: s.Block})
	if err != nil {
		return nil, fmt.Errorf("converting try body: %w", err)
	}

	result := &parsejs.TryStmt{
		Body: body,
	}

	if s.Catch != nil {
		catchBody, err := c.convertFunctionBody(js_ast.FnBody{Block: s.Catch.Block})
		if err != nil {
			return nil, fmt.Errorf("converting catch body: %w", err)
		}

		var binding parsejs.IBinding
		if s.Catch.BindingOrNil.Data != nil {
			binding, err = c.convertBinding(s.Catch.BindingOrNil)
			if err != nil {
				return nil, fmt.Errorf("converting catch binding: %w", err)
			}
		}

		result.Catch = catchBody
		result.Binding = binding
	}

	if s.Finally != nil {
		finallyBody, err := c.convertFunctionBody(js_ast.FnBody{Block: s.Finally.Block})
		if err != nil {
			return nil, fmt.Errorf("converting finally block: %w", err)
		}
		result.Finally = finallyBody
	}

	return result, nil
}

// convertSSwitch converts a switch statement.
//
// Takes s (*js_ast.SSwitch) which contains the switch test expression and
// cases.
//
// Returns parsejs.IStmt which is the converted switch statement.
// Returns error when converting the test expression or any case clause fails.
func (c *ASTConverter) convertSSwitch(s *js_ast.SSwitch) (parsejs.IStmt, error) {
	test, err := c.convertExpression(s.Test)
	if err != nil {
		return nil, fmt.Errorf("converting switch test: %w", err)
	}

	cases := make([]parsejs.CaseClause, 0, len(s.Cases))
	for i, caseItem := range s.Cases {
		clause, err := c.convertCaseClause(caseItem)
		if err != nil {
			return nil, fmt.Errorf("converting switch case %d: %w", i, err)
		}
		cases = append(cases, clause)
	}

	return &parsejs.SwitchStmt{
		Init: test,
		List: cases,
	}, nil
}

// convertCaseClause converts a switch case clause.
//
// Takes caseItem (js_ast.Case) which is the case clause to convert.
//
// Returns parsejs.CaseClause which is the converted case clause.
// Returns error when expression or statement conversion fails.
func (c *ASTConverter) convertCaseClause(caseItem js_ast.Case) (parsejs.CaseClause, error) {
	var condition parsejs.IExpr
	var err error

	if caseItem.ValueOrNil.Data != nil {
		condition, err = c.convertExpression(caseItem.ValueOrNil)
		if err != nil {
			return parsejs.CaseClause{}, fmt.Errorf("converting case condition: %w", err)
		}
	}

	var statements []parsejs.IStmt
	for i, statement := range caseItem.Body {
		converted, err := c.convertStatement(statement)
		if err != nil {
			return parsejs.CaseClause{}, fmt.Errorf("converting case body statement %d: %w", i, err)
		}
		if converted != nil {
			statements = append(statements, converted)
		}
	}

	return parsejs.CaseClause{
		Cond: condition,
		List: statements,
	}, nil
}

// convertSImport converts a JavaScript import statement to the internal form.
//
// Takes s (*js_ast.SImport) which contains the import statement to convert.
//
// Returns parsejs.IStmt which is the converted import statement.
// Returns error when the namespace import cannot be built.
func (c *ASTConverter) convertSImport(s *js_ast.SImport) (parsejs.IStmt, error) {
	importList := c.buildImportList(s)
	defaultName := c.getDefaultImportName(s)
	modulePath := c.getModulePath(s)

	if c.isNamespaceImport(s) {
		return c.buildNamespaceImport(s, modulePath)
	}

	return &parsejs.ImportStmt{
		Default: defaultName,
		List:    importList,
		Module:  []byte(modulePath),
	}, nil
}

// buildImportList builds the list of named imports.
//
// For imports, esbuild's ClauseItem fields mean:
//   - Alias: the name exported by the source module (e.g., "add")
//   - OriginalName: the local binding name (e.g., "addNumbers")
//   - Name.Ref: symbol reference for the local binding
//
// For tdewolff's Alias struct:
//   - Name: printed before "as" (the module's export name)
//   - Binding: printed after "as" (the local binding name)
//
// Takes s (*js_ast.SImport) which contains the import statement to process.
//
// Returns []parsejs.Alias which contains the resolved import aliases, or nil
// if there are no named imports.
func (c *ASTConverter) buildImportList(s *js_ast.SImport) []parsejs.Alias {
	if s.Items == nil {
		return nil
	}

	importList := make([]parsejs.Alias, 0, len(*s.Items))
	for _, item := range *s.Items {
		localName := item.OriginalName
		if localName == "" {
			localName = c.resolveRef(item.Name.Ref)
		}
		if localName == "" {
			localName = item.Alias
		}
		if localName == "" {
			localName = "import"
		}

		alias := parsejs.Alias{
			Binding: []byte(localName),
		}

		if item.Alias != "" && item.Alias != localName {
			alias.Name = []byte(item.Alias)
		}

		importList = append(importList, alias)
	}

	return importList
}

// getDefaultImportName gets the default import name if present.
//
// Takes s (*js_ast.SImport) which is the import statement to examine.
//
// Returns []byte which is the default import name, or nil if not present.
func (c *ASTConverter) getDefaultImportName(s *js_ast.SImport) []byte {
	if s.DefaultName == nil {
		return nil
	}
	name := c.resolveRef(s.DefaultName.Ref)
	if name == "" {
		return nil
	}
	return []byte(name)
}

// getModulePath gets the module path from import records.
//
// Takes s (*js_ast.SImport) which provides the import statement to look up.
//
// Returns string which is the quoted module path, or "unknown" if not found.
func (c *ASTConverter) getModulePath(s *js_ast.SImport) string {
	modulePath := ""
	if c.importRecords != nil && int(s.ImportRecordIndex) < len(c.importRecords) {
		modulePath = c.importRecords[s.ImportRecordIndex].Path.Text
	}
	if modulePath == "" {
		modulePath = "unknown"
	}
	return fmtQuotedStrValue(modulePath)
}

// isNamespaceImport checks if this is a namespace import.
//
// Takes s (*js_ast.SImport) which is the import statement to check.
//
// Returns bool which is true when the import uses namespace syntax.
func (*ASTConverter) isNamespaceImport(s *js_ast.SImport) bool {
	return s.NamespaceRef.InnerIndex != 0 || s.StarNameLoc != nil
}

// buildNamespaceImport builds a namespace import statement.
//
// Takes s (*js_ast.SImport) which provides the import statement to convert.
// Takes modulePath (string) which specifies the module being imported.
//
// Returns parsejs.IStmt which is the converted import statement.
// Returns error when the conversion fails.
func (c *ASTConverter) buildNamespaceImport(s *js_ast.SImport, modulePath string) (parsejs.IStmt, error) {
	nsName := c.resolveRef(s.NamespaceRef)
	if nsName == "" {
		return &parsejs.ImportStmt{Module: []byte(modulePath)}, nil
	}

	return &parsejs.ImportStmt{
		Default: nil,
		List:    []parsejs.Alias{{Name: []byte("*"), Binding: []byte(nsName)}},
		Module:  []byte(modulePath),
	}, nil
}

// convertSExportDefault converts an export default statement.
//
// Takes s (*js_ast.SExportDefault) which is the export default statement to
// convert.
//
// Returns parsejs.IStmt which is the converted statement.
// Returns error when the conversion fails.
func (c *ASTConverter) convertSExportDefault(s *js_ast.SExportDefault) (parsejs.IStmt, error) {
	switch v := s.Value.Data.(type) {
	case *js_ast.SClass:
		return c.convertExportDefaultClass(v)
	case *js_ast.SFunction:
		return c.convertExportDefaultFunction(v)
	case *js_ast.SExpr:
		return c.convertExportDefaultExpr(v)
	default:
		return &parsejs.EmptyStmt{}, nil
	}
}

// convertExportDefaultClass converts an export default class statement.
//
// Takes v (*js_ast.SClass) which contains the class to convert.
//
// Returns parsejs.IStmt which is the converted export statement.
// Returns error when the extends clause or class properties cannot be
// converted.
func (c *ASTConverter) convertExportDefaultClass(v *js_ast.SClass) (parsejs.IStmt, error) {
	var extends parsejs.IExpr
	if v.Class.ExtendsOrNil.Data != nil {
		var err error
		extends, err = c.convertExpression(v.Class.ExtendsOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting export default class extends: %w", err)
		}
	}

	elements := make([]parsejs.ClassElement, 0, len(v.Class.Properties))
	for _, prop := range v.Class.Properties {
		element, err := c.convertClassProperty(prop)
		if err != nil {
			return nil, fmt.Errorf("converting export default class property: %w", err)
		}
		if element != nil {
			elements = append(elements, *element)
		}
	}

	className := "AnonymousClass"
	if v.Class.Name != nil {
		if resolved := c.resolveRef(v.Class.Name.Ref); resolved != "" {
			className = resolved
		}
	}

	classExpr := &parsejs.ClassDecl{
		Name:    &parsejs.Var{Data: []byte(className)},
		Extends: extends,
		List:    elements,
	}

	return &parsejs.ExportStmt{
		Default: true,
		Decl:    classExpr,
	}, nil
}

// convertExportDefaultFunction converts an export default function statement.
//
// Takes v (*js_ast.SFunction) which contains the function to convert.
//
// Returns parsejs.IStmt which is the converted export statement.
// Returns error when parameter or body conversion fails.
func (c *ASTConverter) convertExportDefaultFunction(v *js_ast.SFunction) (parsejs.IStmt, error) {
	params, err := c.convertParams(v.Fn.Args)
	if err != nil {
		return nil, fmt.Errorf("converting export default function parameters: %w", err)
	}

	body, err := c.convertFunctionBody(v.Fn.Body)
	if err != nil {
		return nil, fmt.Errorf("converting export default function body: %w", err)
	}

	name := "anonymous"
	if v.Fn.Name != nil {
		if resolved := c.resolveRef(v.Fn.Name.Ref); resolved != "" {
			name = resolved
		}
	}

	funcExpr := &parsejs.FuncDecl{
		Async:     v.Fn.IsAsync,
		Generator: v.Fn.IsGenerator,
		Name:      &parsejs.Var{Data: []byte(name)},
		Params:    params,
		Body:      *body,
	}

	return &parsejs.ExportStmt{
		Default: true,
		Decl:    funcExpr,
	}, nil
}

// convertExportDefaultExpr converts an export default expression.
//
// Takes v (*js_ast.SExpr) which contains the expression to export.
//
// Returns parsejs.IStmt which is the converted export statement.
// Returns error when the expression conversion fails.
func (c *ASTConverter) convertExportDefaultExpr(v *js_ast.SExpr) (parsejs.IStmt, error) {
	expression, err := c.convertExpression(v.Value)
	if err != nil {
		return nil, fmt.Errorf("converting export default expression: %w", err)
	}
	return &parsejs.ExportStmt{
		Default: true,
		Decl:    expression,
	}, nil
}

// getLocalTokenType returns the token type for a local declaration kind.
//
// Takes kind (js_ast.LocalKind) which specifies the declaration kind (const,
// let, or var).
//
// Returns parsejs.TokenType which is the corresponding token type.
func getLocalTokenType(kind js_ast.LocalKind) parsejs.TokenType {
	switch kind {
	case js_ast.LocalConst:
		return parsejs.ConstToken
	case js_ast.LocalLet:
		return parsejs.LetToken
	default:
		return parsejs.VarToken
	}
}

// fmtQuotedStrValue formats a string as a quoted JavaScript string.
//
// Takes s (string) which is the value to wrap in quotes.
//
// Returns string which is the input wrapped in double quotes.
func fmtQuotedStrValue(s string) string {
	return "\"" + s + "\""
}
