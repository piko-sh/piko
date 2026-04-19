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

// Parses component template files into structured AST representations by extracting template, script, and style blocks.
// Handles Go script parsing, validates template structure, and produces source objects ready for further compilation stages.

import (
	"context"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/sfcparser"
)

const (
	// i18nLangJSON is the expected lang attribute value for i18n blocks.
	i18nLangJSON = "json"

	// typeNameProps is the name used to find the Props type in parsed scripts.
	typeNameProps = "Props"

	// functionNameRender is the name of the template render method.
	functionNameRender = "Render"

	// functionNameMiddlewares is the expected name for the middleware chain function.
	functionNameMiddlewares = "Middlewares"

	// functionNameCachePolicy is the name of the CachePolicy lifecycle hook function.
	functionNameCachePolicy = "CachePolicy"

	// functionNameLocales is the name of the function that returns supported locales.
	functionNameLocales = "SupportedLocales"

	// functionNameAuthPolicy is the name of the AuthPolicy lifecycle hook function.
	functionNameAuthPolicy = "AuthPolicy"

	// functionNamePreview is the name of the Preview convention function for
	// dev-mode component previewing.
	functionNamePreview = "Preview"
)

// Sources holds the separated source code blocks from a .pk file.
// Intended for debugging.
type Sources struct {
	// TemplateSource is the raw template content used for parsing.
	TemplateSource string

	// ScriptSource holds the Go script content from the component file.
	ScriptSource string

	// StyleBlocks holds parsed style data from the source.
	StyleBlocks []sfcparser.Style
}

// scriptBlockParseError represents a fatal syntax error within a Go script
// block. It implements the error interface.
type scriptBlockParseError struct {
	// reason describes why the script block could not be parsed.
	reason string
}

// Error returns the reason for the script parsing failure.
//
// Returns string which is the formatted error message.
func (e *scriptBlockParseError) Error() string {
	return "cannot parse <script> snippet: " + e.reason
}

// ParsePK parses an SFC file and extracts its template, script, and i18n
// blocks.
//
// Takes data ([]byte) which contains the raw SFC file content.
// Takes sourcePath (string) which identifies the file path for error messages.
//
// Returns *annotator_dto.ParsedComponent which contains the parsed template,
// script, and i18n blocks.
// Returns Sources which provides the separated source blocks for each section.
// Returns error when the script or i18n blocks cannot be parsed. Template
// errors are returned with a partial component to support LSP features.
func ParsePK(ctx context.Context, data []byte, sourcePath string) (*annotator_dto.ParsedComponent, Sources, error) {
	sfcResult, srcs, err := parseAndSeparateSFC(data)
	if err != nil {
		return nil, srcs, err
	}

	templateStartLocation := ast_domain.Location{Line: sfcResult.TemplateContentLocation.Line, Column: sfcResult.TemplateContentLocation.Column, Offset: 0}

	parsedTemplate, templateErr := parseTemplateBlock(ctx, srcs.TemplateSource, sourcePath, templateStartLocation)

	scriptBlockDiagnostics := validateScriptBlocks(sfcResult, sourcePath)
	if len(scriptBlockDiagnostics) > 0 {
		if parsedTemplate != nil {
			parsedTemplate.Diagnostics = append(parsedTemplate.Diagnostics, scriptBlockDiagnostics...)
		}
	}

	var scriptStartLocation ast_domain.Location
	if goScript, ok := sfcResult.GoScript(); ok {
		scriptStartLocation = ast_domain.Location{
			Line:   goScript.ContentLocation.Line,
			Column: goScript.ContentLocation.Column,
			Offset: 0,
		}
	}

	parsedScript, pikoImports, scriptErr := analyseGoScript(srcs.ScriptSource, scriptStartLocation)

	localTranslations, i18nErr := parseI18nBlocks(sfcResult, sourcePath)
	if i18nErr != nil {
		return nil, srcs, i18nErr
	}

	if _, hasGoScriptTag := sfcResult.GoScript(); hasGoScriptTag && parsedScript == nil {
		parsedScript = createEmptyParsedScript()
	}

	component := buildParsedComponent(parsedTemplate, parsedScript, localTranslations, sourcePath, sfcResult, pikoImports)

	if templateErr != nil {
		return component, srcs, templateErr
	}
	if scriptErr != nil {
		return component, srcs, scriptErr
	}

	return component, srcs, nil
}

// parseTemplateBlock handles only the parsing of the <template> block into
// the custom AST.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes templateSource (string) which contains the raw template content.
// Takes sourcePath (string) which identifies the file for error reporting.
// Takes templateContentLocation (ast_domain.Location) which specifies the
// position of the template content within the source file.
//
// Returns *ast_domain.TemplateAST which contains the parsed template tree.
// Returns error when the template contains parse errors or diagnostics.
func parseTemplateBlock(ctx context.Context, templateSource, sourcePath string, templateContentLocation ast_domain.Location) (*ast_domain.TemplateAST, error) {
	if strings.TrimSpace(templateSource) == "" {
		return nil, nil
	}

	parsedTemplate, tmplErr := ast_domain.Parse(ctx, templateSource, sourcePath, &templateContentLocation)
	if tmplErr != nil {
		return nil, tmplErr
	}
	if ast_domain.HasErrors(parsedTemplate.Diagnostics) {
		return parsedTemplate, &ParseDiagnosticError{
			Diagnostics:    parsedTemplate.Diagnostics,
			SourcePath:     sourcePath,
			TemplateSource: templateSource,
		}
	}
	return parsedTemplate, nil
}

// parseAndSeparateSFC parses a .pk file and splits it into its parts.
//
// Takes data ([]byte) which contains the raw .pk file content.
//
// Returns *sfcparser.ParseResult which contains the full parse result.
// Returns Sources which holds the template, script, and style blocks.
// Returns error when parsing fails.
func parseAndSeparateSFC(data []byte) (*sfcparser.ParseResult, Sources, error) {
	sfcResult, err := sfcparser.Parse(data)
	if err != nil {
		return nil, Sources{}, err
	}

	srcs := Sources{
		TemplateSource: sfcResult.Template,
		ScriptSource:   "",
		StyleBlocks:    sfcResult.Styles,
	}

	if goScript, ok := sfcResult.GoScript(); ok {
		srcs.ScriptSource = goScript.Content
	}

	return sfcResult, srcs, nil
}

// validateScriptBlocks checks all script blocks for missing or unrecognised
// lang/type attributes and returns diagnostics for any problems found.
//
// Valid lang and type combinations cover Go, JavaScript, and TypeScript. See
// the implementation for the recognised lang and MIME type strings accepted
// for each language.
//
// Script blocks without a lang or type attribute will trigger a warning.
//
// Takes sfcResult (*sfcparser.ParseResult) which contains the parsed script
// blocks.
// Takes sourcePath (string) which is the file path for diagnostic reporting.
//
// Returns []*ast_domain.Diagnostic which contains warnings for script blocks
// with missing or unrecognised attributes.
func validateScriptBlocks(sfcResult *sfcparser.ParseResult, sourcePath string) []*ast_domain.Diagnostic {
	diagnostics := make([]*ast_domain.Diagnostic, 0, len(sfcResult.Scripts))

	for _, script := range sfcResult.Scripts {
		if script.HasRecognizedScriptType() {
			continue
		}

		location := ast_domain.Location{
			Line:   script.Location.Line,
			Column: script.Location.Column,
		}

		var message string
		if lang, ok := script.Attributes["lang"]; ok && lang != "" {
			message = fmt.Sprintf("Unrecognised script block attribute lang=%q. Valid options are: "+
				"lang=\"go\", lang=\"ts\", lang=\"js\", type=\"application/x-go\", type=\"application/javascript\", "+
				"type=\"application/typescript\", type=\"module\"", lang)
		} else if scriptType, ok := script.Attributes["type"]; ok && scriptType != "" {
			message = fmt.Sprintf("Unrecognised script block attribute type=%q. Valid options are: "+
				"lang=\"go\", lang=\"ts\", lang=\"js\", type=\"application/x-go\", type=\"application/javascript\", "+
				"type=\"application/typescript\", type=\"module\"", scriptType)
		} else {
			message = "Script block is missing a lang or type attribute. " +
				"Add lang=\"go\" for Go, lang=\"ts\" for TypeScript, or lang=\"js\" for JavaScript."
		}

		diagnostic := ast_domain.NewDiagnosticWithCode(
			ast_domain.Warning,
			message,
			"<script>",
			annotator_dto.CodeScriptMissingLang,
			location,
			sourcePath,
		)
		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics
}

// buildParsedComponent builds a ParsedComponent from parsed SFC parts.
//
// Takes parsedTemplate (*ast_domain.TemplateAST) which is the parsed template
// AST.
// Takes parsedScript (*annotator_dto.ParsedScript) which is the parsed Go
// script.
// Takes localTranslations (i18n_domain.Translations) which contains the local
// translation strings.
// Takes sourcePath (string) which is the path to the source file.
// Takes sfcResult (*sfcparser.ParseResult) which is the SFC parse result
// including styles and client script.
// Takes pikoImports ([]annotator_dto.PikoImport) which lists the Piko
// component imports.
//
// Returns *annotator_dto.ParsedComponent which is the assembled component
// ready for further processing.
func buildParsedComponent(
	parsedTemplate *ast_domain.TemplateAST,
	parsedScript *annotator_dto.ParsedScript,
	localTranslations i18n_domain.Translations,
	sourcePath string,
	sfcResult *sfcparser.ParseResult,
	pikoImports []annotator_dto.PikoImport,
) *annotator_dto.ParsedComponent {
	var clientScript string
	if clientScriptBlock, ok := sfcResult.ClientScript(); ok {
		clientScript = clientScriptBlock.Content
	}

	component := &annotator_dto.ParsedComponent{
		Template:            parsedTemplate,
		Script:              parsedScript,
		LocalTranslations:   localTranslations,
		SourcePath:          sourcePath,
		ModuleImportPath:    "",
		IsExternal:          false,
		StyleBlocks:         sfcResult.Styles,
		PikoImports:         pikoImports,
		ComponentType:       "",
		HasCollection:       false,
		CollectionName:      "",
		CollectionProvider:  "",
		CollectionParamName: "",
		ClientScript:        clientScript,
		ContentModulePath:   "",
	}

	if sfcResult.HasCollectionDirective() {
		component.HasCollection = true
		component.CollectionName = sfcResult.GetCollectionName()
		component.CollectionProvider = sfcResult.GetCollectionProvider()
		component.CollectionParamName = sfcResult.GetCollectionParamName()

		if sfcResult.HasCollectionSource() {
			alias := sfcResult.GetCollectionSource()
			var goImports []*goast.ImportSpec
			if parsedScript != nil && parsedScript.AST != nil {
				goImports = parsedScript.AST.Imports
			}
			component.ContentModulePath = resolveCollectionSourceAlias(alias, goImports)
		}
	}

	if sfcResult.HasPublicDirective() {
		component.VisibilityOverride = new(true)
	} else if sfcResult.HasPrivateDirective() {
		component.VisibilityOverride = new(false)
	}

	return component
}

// resolveCollectionSourceAlias finds the full import path for a given alias.
// Resolves p-collection-source attributes that reference Go imports for external
// markdown content.
//
// Takes alias (string) which is the import alias to look up.
// Takes goImports ([]*goast.ImportSpec) which contains the parsed Go imports.
//
// Returns string which is the full import path, or an empty string if not
// found.
func resolveCollectionSourceAlias(alias string, goImports []*goast.ImportSpec) string {
	for _, imp := range goImports {
		if imp == nil || imp.Path == nil {
			continue
		}

		pathVal := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil && imp.Name.Name == alias {
			return pathVal
		}

		segments := strings.Split(pathVal, "/")
		if len(segments) > 0 && segments[len(segments)-1] == alias {
			return pathVal
		}
	}
	return ""
}

// createEmptyParsedScript creates a default ParsedScript for empty script blocks.
//
// Returns *annotator_dto.ParsedScript which contains a minimal valid script
// structure with default values.
func createEmptyParsedScript() *annotator_dto.ParsedScript {
	const defaultPackageName = "piko_default"
	return &annotator_dto.ParsedScript{
		AST:                        &goast.File{Name: goast.NewIdent(defaultPackageName)},
		GoPackageName:              defaultPackageName,
		ScriptStartLocation:        ast_domain.Location{},
		PropsTypeExpression:        nil,
		RenderReturnTypeExpression: nil,
		Fset:                       nil,
		ProvisionalGoPackagePath:   "",
		MiddlewaresFuncName:        "",
		CachePolicyFuncName:        "",
		SupportedLocalesFuncName:   "",
		HasMiddleware:              false,
		HasCachePolicy:             false,
		HasSupportedLocales:        false,
	}
}

// analyseGoScript parses Go source code and extracts script metadata.
//
// When the script source is empty or contains only whitespace, returns an
// empty parsed script with no imports.
//
// Takes scriptSource (string) which is the Go source code to parse.
// Takes scriptStartLocation (ast_domain.Location) which specifies where the
// script begins in the original file.
//
// Returns *annotator_dto.ParsedScript which contains the AST and metadata.
// Returns []annotator_dto.PikoImport which lists any Piko-specific imports.
// Returns error when the Go source code has parse errors. In fault-tolerant
// mode, a partial script is still returned along with the error.
func analyseGoScript(scriptSource string, scriptStartLocation ast_domain.Location) (*annotator_dto.ParsedScript, []annotator_dto.PikoImport, error) {
	if isEffectivelyEmpty(scriptSource) {
		return createEmptyParsedScript(), nil, nil
	}

	fset := token.NewFileSet()
	file, parseErr := parser.ParseFile(fset, "", scriptSource, parser.AllErrors)

	if file == nil {
		if parseErr != nil {
			return nil, nil, &scriptBlockParseError{reason: parseErr.Error()}
		}
		return createEmptyParsedScript(), nil, nil
	}

	parsedScript := &annotator_dto.ParsedScript{
		PropsTypeExpression:        nil,
		RenderReturnTypeExpression: nil,
		AST:                        file,
		Fset:                       fset,
		ProvisionalGoPackagePath:   "",
		GoPackageName:              "",
		MiddlewaresFuncName:        "",
		CachePolicyFuncName:        "",
		SupportedLocalesFuncName:   "",
		ScriptStartLocation:        scriptStartLocation,
		HasMiddleware:              false,
		HasCachePolicy:             false,
		HasSupportedLocales:        false,
	}
	if file.Name != nil {
		parsedScript.GoPackageName = file.Name.Name
	} else {
		parsedScript.GoPackageName = "piko_default"
	}

	pikoImports, goImports, otherDecls := separateImports(file.Decls, fset)
	parsedScript.AST.Imports = goImports
	parsedScript.AST.Decls = otherDecls

	if len(goImports) > 0 {
		reconstructImportBlock(parsedScript.AST)
	}

	inspectDeclarations(parsedScript)

	if parseErr != nil {
		return parsedScript, pikoImports, &scriptBlockParseError{reason: parseErr.Error()}
	}
	return parsedScript, pikoImports, nil
}

// separateImports splits declarations into piko imports, standard Go imports,
// and other declarations.
//
// Takes decls ([]goast.Decl) which contains the declarations to separate.
// Takes fset (*token.FileSet) which provides source position data.
//
// Returns []annotator_dto.PikoImport which contains the piko imports found.
// Returns []*goast.ImportSpec which contains the standard Go imports.
// Returns []goast.Decl which contains all other declarations.
func separateImports(decls []goast.Decl, fset *token.FileSet) ([]annotator_dto.PikoImport, []*goast.ImportSpec, []goast.Decl) {
	var pikoImports []annotator_dto.PikoImport
	var goImports []*goast.ImportSpec
	var otherDecls []goast.Decl

	for _, declaration := range decls {
		genDecl, isImport := declaration.(*goast.GenDecl)
		if !isImport || genDecl.Tok != token.IMPORT {
			otherDecls = append(otherDecls, declaration)
			continue
		}

		categoriseImportDecl(genDecl, fset, &pikoImports, &goImports)
	}
	return pikoImports, goImports, otherDecls
}

// categoriseImportDecl processes a single import declaration and sorts its
// specs into Piko imports or standard Go imports.
//
// Takes genDecl (*goast.GenDecl) which is the import declaration to process.
// Takes fset (*token.FileSet) which provides position information.
// Takes pikoImports (*[]annotator_dto.PikoImport) which collects Piko imports.
// Takes goImports (*[]*goast.ImportSpec) which collects standard Go imports.
func categoriseImportDecl(genDecl *goast.GenDecl, fset *token.FileSet, pikoImports *[]annotator_dto.PikoImport, goImports *[]*goast.ImportSpec) {
	for _, spec := range genDecl.Specs {
		impSpec, ok := spec.(*goast.ImportSpec)
		if !ok || impSpec.Path == nil {
			continue
		}

		categoriseImportSpec(impSpec, fset, pikoImports, goImports)
	}
}

// categoriseImportSpec sorts a single import into either the Piko imports list
// or the standard Go imports list.
//
// Takes impSpec (*goast.ImportSpec) which is the import to check.
// Takes fset (*token.FileSet) which provides position details.
// Takes pikoImports (*[]annotator_dto.PikoImport) which collects Piko imports.
// Takes goImports (*[]*goast.ImportSpec) which collects standard Go imports.
func categoriseImportSpec(impSpec *goast.ImportSpec, fset *token.FileSet, pikoImports *[]annotator_dto.PikoImport, goImports *[]*goast.ImportSpec) {
	pathVal := strings.Trim(impSpec.Path.Value, `"`)

	if isPikoImport(pathVal) {
		pikoImport := createPikoImport(impSpec, fset, pathVal)
		*pikoImports = append(*pikoImports, pikoImport)
	} else {
		*goImports = append(*goImports, impSpec)
	}
}

// isPikoImport checks whether an import path is a Piko template import.
//
// Takes path (string) which is the import path to check.
//
// Returns bool which is true if the path ends with the .pk extension.
func isPikoImport(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".pk")
}

// createPikoImport builds a PikoImport from an import statement.
//
// Takes impSpec (*goast.ImportSpec) which provides the import details.
// Takes fset (*token.FileSet) which maps positions to line and column numbers.
// Takes pathVal (string) which is the import path.
//
// Returns annotator_dto.PikoImport which holds the alias, path, and location.
func createPikoImport(impSpec *goast.ImportSpec, fset *token.FileSet, pathVal string) annotator_dto.PikoImport {
	var alias string
	if impSpec.Name != nil {
		alias = impSpec.Name.Name
	}

	position := fset.Position(impSpec.Path.Pos())
	location := ast_domain.Location{
		Line:   position.Line,
		Column: position.Column,
		Offset: position.Offset,
	}

	return annotator_dto.PikoImport{
		Alias:    alias,
		Path:     pathVal,
		Location: location,
	}
}

// reconstructImportBlock adds a single, tidy import block back to the AST
// if one is needed.
//
// Takes file (*goast.File) which is the AST to modify.
func reconstructImportBlock(file *goast.File) {
	importDecl := &goast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
		Specs:  make([]goast.Spec, len(file.Imports)),
	}
	for i, impSpec := range file.Imports {
		importDecl.Specs[i] = impSpec
	}
	file.Decls = append([]goast.Decl{importDecl}, file.Decls...)
}

// inspectDeclarations walks the AST and passes each node to functions that
// handle specific declaration types.
//
// Takes result (*annotator_dto.ParsedScript) which holds the parsed AST and
// collects the declarations found during the walk.
func inspectDeclarations(result *annotator_dto.ParsedScript) {
	inspector := func(n goast.Node) bool {
		switch node := n.(type) {
		case *goast.TypeSpec:
			inspectTypeSpec(node, result)
		case *goast.FuncDecl:
			inspectFuncDecl(node, result)
		}
		return true
	}
	goast.Inspect(result.AST, inspector)
}

// inspectTypeSpec checks a type declaration and stores it if it is a Props
// type.
//
// Takes node (*goast.TypeSpec) which is the type declaration to check.
// Takes result (*annotator_dto.ParsedScript) which stores the found Props type.
func inspectTypeSpec(node *goast.TypeSpec, result *annotator_dto.ParsedScript) {
	if node.Name.Name == typeNameProps {
		if result.PropsTypeExpression == nil {
			result.PropsTypeExpression = node.Name
		}
	}
}

// inspectFuncDecl checks a function declaration for Render and lifecycle
// functions.
//
// Takes node (*goast.FuncDecl) which is the function declaration to check.
// Takes result (*annotator_dto.ParsedScript) which stores the parsed data.
func inspectFuncDecl(node *goast.FuncDecl, result *annotator_dto.ParsedScript) {
	if isRenderFuncSignature(node) {
		result.RenderReturnTypeExpression = node.Type.Results.List[0].Type
		if node.Type.Params.NumFields() >= 2 {
			propsField := node.Type.Params.List[1]
			result.PropsTypeExpression = propsField.Type
		}
	} else {
		findAndAnalyseLifecycleFuncs(node, result)
	}
}

// isRenderFuncSignature checks whether a function declaration matches the
// expected signature for a Render function.
//
// Takes node (*goast.FuncDecl) which is the function declaration to check.
//
// Returns bool which is true if the function is named Render and has at least
// one parameter and one return value.
func isRenderFuncSignature(node *goast.FuncDecl) bool {
	return node.Name.Name == functionNameRender &&
		node.Type.Params != nil &&
		node.Type.Params.NumFields() > 0 &&
		node.Type.Results != nil &&
		node.Type.Results.NumFields() > 0
}

// findAndAnalyseLifecycleFuncs checks for special lifecycle functions and
// records their presence in the parsed script result. It looks for functions
// named Middlewares, Locales, and CachePolicy.
//
// Takes node (*goast.FuncDecl) which is the function declaration to check.
// Takes result (*annotator_dto.ParsedScript) which stores the analysis results.
func findAndAnalyseLifecycleFuncs(node *goast.FuncDecl, result *annotator_dto.ParsedScript) {
	hasOneResult := node.Type.Results != nil && node.Type.Results.NumFields() == 1
	if !hasOneResult {
		return
	}

	switch node.Name.Name {
	case functionNameLocales:
		result.HasSupportedLocales = true
		result.SupportedLocalesFuncName = functionNameLocales
	case functionNameMiddlewares:
		result.HasMiddleware = true
		result.MiddlewaresFuncName = functionNameMiddlewares
	case functionNameCachePolicy:
		result.HasCachePolicy = true
		result.CachePolicyFuncName = functionNameCachePolicy
	case functionNameAuthPolicy:
		result.HasAuthPolicy = true
		result.AuthPolicyFuncName = functionNameAuthPolicy
	case functionNamePreview:
		result.HasPreview = true
		result.PreviewFuncName = functionNamePreview
	}
}

// parseI18nBlocks extracts and flattens translation data from i18n blocks.
//
// Takes sfcResult (*sfcparser.ParseResult) which contains the parsed SFC with
// i18n blocks to process.
// Takes sourcePath (string) which identifies the source file for error
// messages.
//
// Returns i18n_domain.Translations which contains the flattened translations
// grouped by locale.
// Returns error when the i18n JSON block cannot be parsed.
func parseI18nBlocks(sfcResult *sfcparser.ParseResult, sourcePath string) (i18n_domain.Translations, error) {
	if len(sfcResult.I18nBlocks) == 0 {
		return nil, nil
	}
	localTranslations := make(i18n_domain.Translations)
	for _, block := range sfcResult.I18nBlocks {
		lang, ok := block.Attributes["lang"]
		if !ok || lang != i18nLangJSON {
			continue
		}

		var translationsForLangs map[string]map[string]any
		if err := json.UnmarshalString(block.Content, &translationsForLangs); err != nil {
			return nil, fmt.Errorf("failed to parse i18n JSON block in %s: %w", sourcePath, err)
		}

		for locale, nestedKeyValues := range translationsForLangs {
			if _, exists := localTranslations[locale]; !exists {
				localTranslations[locale] = make(map[string]string)
			}
			i18n_domain.FlattenTranslations(nestedKeyValues, "", localTranslations[locale])
		}
	}
	return localTranslations, nil
}

// isEffectivelyEmpty checks whether Go source code contains only whitespace
// and comments.
//
// Takes source (string) which is the Go source code to check.
//
// Returns bool which is true if the source has no code apart from whitespace
// and comments.
func isEffectivelyEmpty(source string) bool {
	if strings.TrimSpace(source) == "" {
		return true
	}

	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(source))
	var s scanner.Scanner
	s.Init(file, []byte(source), nil, scanner.ScanComments)

	for {
		_, scannedToken, _ := s.Scan()
		if scannedToken == token.EOF {
			return true
		}
		if scannedToken != token.COMMENT {
			return false
		}
	}
}
