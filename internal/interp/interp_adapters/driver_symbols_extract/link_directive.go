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

package driver_symbols_extract

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/packages"
)

const (
	// linkDirectivePrefix is the //piko:link comment prefix scanned on
	// the doc comments of exported generic functions.
	linkDirectivePrefix = "//piko:link"

	// maxLinkDirectiveCommentBytes caps the length of an individual doc
	// comment line examined for a //piko:link directive. Go doc
	// comments are typically well under 1KB; anything larger is either
	// malformed or an intentional attack, and scanning it wastes time.
	maxLinkDirectiveCommentBytes = 1024

	// maxLinkDirectiveDocLines caps how many comment lines a single
	// function's doc block may contain before the directive parser
	// gives up. Protects extract from pathological source files with
	// enormous comment headers.
	maxLinkDirectiveDocLines = 512

	// maxLinkDirectiveTargetLength caps the accepted length of the
	// sibling identifier. Real Go identifiers are rarely more than 64
	// characters.
	maxLinkDirectiveTargetLength = 256

	// maxLinkReturnWalkDepth caps the recursion used while deciding
	// whether a return type is parametric, guarding against the same
	// descriptor depth-bomb class covered at the runtime boundary.
	maxLinkReturnWalkDepth = 64
)

var (
	// errLinkDirectiveMalformed reports a //piko:link line that does
	// not parse as "//piko:link <identifier>".
	errLinkDirectiveMalformed = errors.New("malformed //piko:link directive")

	// errLinkDirectiveDuplicate reports more than one //piko:link line
	// on the same function.
	errLinkDirectiveDuplicate = errors.New("duplicate //piko:link directive")

	// errLinkTargetMissing reports a directive whose target identifier
	// does not resolve to a function in the same package.
	errLinkTargetMissing = errors.New("//piko:link target not found in package")

	// errLinkTargetNotFunc reports a directive whose target identifier
	// resolves to something other than a function.
	errLinkTargetNotFunc = errors.New("//piko:link target is not a function")

	// errLinkTargetArity reports a directive whose target function has
	// the wrong number of parameters for the generic it links to.
	errLinkTargetArity = errors.New("//piko:link target has wrong arity")

	// errLinkDirectiveOnNonGeneric reports a directive attached to a
	// non-generic function.
	errLinkDirectiveOnNonGeneric = errors.New("//piko:link may only be used on generic functions")

	// errLinkDirectiveDocTooLong reports a function whose doc block
	// exceeds maxLinkDirectiveDocLines. Truncating silently could hide
	// a directive placed near the end of a pathological comment block.
	errLinkDirectiveDocTooLong = errors.New("//piko:link doc block exceeds line limit")

	// errLinkDirectiveCommentTooLong reports a single doc line that
	// exceeds maxLinkDirectiveCommentBytes. Silent skip would hide a
	// directive embedded in a long line.
	errLinkDirectiveCommentTooLong = errors.New("//piko:link doc line exceeds byte limit")

	// errLinkTargetTypePrefix reports a sibling whose leading
	// parameters are not reflect.Type.
	errLinkTargetTypePrefix = errors.New("//piko:link target has wrong leading parameter shape")

	// errLinkTargetReturnShape reports a sibling whose returns do not
	// match the generic's return shape.
	errLinkTargetReturnShape = errors.New("//piko:link target has wrong return shape")
)

// LinkDirective records a parsed //piko:link annotation attached to a
// generic function declaration.
type LinkDirective struct {
	// GenericName is the exported name of the annotated generic
	// function.
	GenericName string

	// LinkTarget is the identifier the directive points at. The target
	// must be declared in the same package; it need not be exported.
	LinkTarget string
}

// linkCollectorState accumulates directives and per-package duplicate
// detection across the two-level declaration walk without inflating
// collectLinkDirectives' cognitive complexity.
type linkCollectorState struct {
	// seen maps generic names already linked to the position of their
	// directive, used to detect duplicates across the file set.
	seen map[string]token.Pos

	// directives collects every valid directive as it is parsed.
	directives []LinkDirective

	// errs accumulates parse errors from malformed directives so the
	// caller can report them all in a single errors.Join.
	errs []error
}

// visitDecl handles a single top-level declaration, parsing any
// //piko:link directive and recording duplicates or errors on the
// receiver's state.
//
// Takes fset (*token.FileSet) which resolves comment positions.
// Takes decl (ast.Decl) which is the declaration to inspect.
func (s *linkCollectorState) visitDecl(fset *token.FileSet, decl ast.Decl) {
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok || funcDecl.Doc == nil || funcDecl.Recv != nil {
		return
	}
	link, ok, err := parseFuncLinkDirective(funcDecl, fset)
	if err != nil {
		s.errs = append(s.errs, err)
		return
	}
	if !ok {
		return
	}
	if prev, duplicate := s.seen[link.GenericName]; duplicate {
		s.errs = append(s.errs, fmt.Errorf("%w: %s already linked (previous at %s)",
			errLinkDirectiveDuplicate, link.GenericName, fset.Position(prev)))
		return
	}
	s.seen[link.GenericName] = funcDecl.Pos()
	s.directives = append(s.directives, link)
}

// collectLinkDirectives walks the AST files of a loaded package and
// gathers //piko:link directives attached to function declarations.
// Invalid directives surface as errors so misuse fails extract rather
// than silently producing broken generated code.
//
// Takes pkg (*packages.Package) which is the loaded package. The
// caller must have passed packages.NeedSyntax so pkg.Syntax is
// populated.
//
// Returns a slice of LinkDirective values sorted by GenericName and
// any parse error encountered along the way.
func collectLinkDirectives(pkg *packages.Package) ([]LinkDirective, error) {
	if pkg == nil {
		return nil, nil
	}
	state := linkCollectorState{seen: make(map[string]token.Pos)}
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			state.visitDecl(pkg.Fset, decl)
		}
	}
	if err := errors.Join(state.errs...); err != nil {
		return nil, err
	}
	return state.directives, nil
}

// parseFuncLinkDirective extracts a //piko:link directive from the
// given function declaration's doc comment.
//
// Takes funcDecl (*ast.FuncDecl) which carries the doc comment.
// Takes fset (*token.FileSet) which resolves comment positions for
// error messages.
//
// Returns the parsed directive, a bool indicating whether any directive
// was found, and any parse error. A missing directive is not an error.
func parseFuncLinkDirective(funcDecl *ast.FuncDecl, fset *token.FileSet) (LinkDirective, bool, error) {
	var (
		target string
		found  bool
	)
	docLines := funcDecl.Doc.List
	if len(docLines) > maxLinkDirectiveDocLines {
		return LinkDirective{}, false, fmt.Errorf("%w: %s has %d doc lines, limit is %d",
			errLinkDirectiveDocTooLong, funcDecl.Name.Name, len(docLines), maxLinkDirectiveDocLines)
	}
	for _, comment := range docLines {
		if len(comment.Text) > maxLinkDirectiveCommentBytes {
			return LinkDirective{}, false, fmt.Errorf("%w: %s at %s (%d bytes, limit %d)",
				errLinkDirectiveCommentTooLong, funcDecl.Name.Name,
				fset.Position(comment.Pos()), len(comment.Text), maxLinkDirectiveCommentBytes)
		}
		text := strings.TrimSpace(comment.Text)
		if !strings.HasPrefix(text, linkDirectivePrefix) {
			continue
		}
		remainder := strings.TrimSpace(strings.TrimPrefix(text, linkDirectivePrefix))
		fields := strings.Fields(remainder)
		if len(fields) != 1 || utf8.RuneCountInString(fields[0]) > maxLinkDirectiveTargetLength || !isValidIdentifier(fields[0]) {
			return LinkDirective{}, false, fmt.Errorf("%w on %s at %s: %q",
				errLinkDirectiveMalformed, funcDecl.Name.Name,
				fset.Position(comment.Pos()), text)
		}
		if found {
			return LinkDirective{}, false, fmt.Errorf("%w on %s at %s",
				errLinkDirectiveDuplicate, funcDecl.Name.Name,
				fset.Position(comment.Pos()))
		}
		target = fields[0]
		found = true
	}
	if !found {
		return LinkDirective{}, false, nil
	}
	return LinkDirective{
		GenericName: funcDecl.Name.Name,
		LinkTarget:  target,
	}, true, nil
}

// validateLinkDirectives ensures each directive's generic exists as
// an exported generic function in the package, and each target exists
// as a function whose arity matches TypeArgCount + len(generic params).
//
// Takes pkg (*packages.Package) which is the loaded package.
// Takes directives ([]LinkDirective) which are the parsed directives.
//
// Returns the subset of directives that passed validation, plus any
// validation errors joined together.
func validateLinkDirectives(pkg *packages.Package, directives []LinkDirective) ([]LinkDirective, error) {
	if len(directives) == 0 {
		return nil, nil
	}
	scope := pkg.Types.Scope()
	var (
		valid []LinkDirective
		errs  []error
	)
	for _, link := range directives {
		if err := validateOneLinkDirective(scope, link); err != nil {
			errs = append(errs, err)
			continue
		}
		valid = append(valid, link)
	}
	return valid, errors.Join(errs...)
}

// validateOneLinkDirective checks that a single directive resolves to
// a generic function with a sibling of matching arity.
//
// Takes scope (*types.Scope) which is the package-level scope.
// Takes link (LinkDirective) which is the directive to validate.
//
// Returns nil when the directive is valid, or an error describing the
// first violation encountered.
func validateOneLinkDirective(scope *types.Scope, link LinkDirective) error {
	genericSig, err := resolveGenericSignature(scope, link)
	if err != nil {
		return err
	}
	targetSig, err := resolveLinkTargetSignature(scope, link)
	if err != nil {
		return err
	}

	typeArgCount := genericSig.TypeParams().Len()
	wantParams := typeArgCount + genericSig.Params().Len()
	if targetSig.Params().Len() != wantParams {
		return fmt.Errorf("%w: %s -> %s expects %d parameters, sibling has %d",
			errLinkTargetArity, link.GenericName, link.LinkTarget, wantParams, targetSig.Params().Len())
	}
	wantResults := genericSig.Results().Len()
	if targetSig.Results().Len() != wantResults {
		return fmt.Errorf("%w: %s -> %s returns %d values, sibling returns %d",
			errLinkTargetArity, link.GenericName, link.LinkTarget, wantResults, targetSig.Results().Len())
	}
	if err := verifyLinkTargetTypePrefix(link, targetSig, typeArgCount); err != nil {
		return err
	}
	return verifyLinkTargetReturnShape(link, genericSig, targetSig)
}

// verifyLinkTargetTypePrefix confirms the sibling's first TypeArgCount
// parameters are declared as reflect.Type.
//
// Takes link (LinkDirective) which names the generic and sibling.
// Takes targetSig (*types.Signature) which is the sibling's
// signature.
// Takes typeArgCount (int) which is the number of type parameters
// the generic declares.
//
// Returns nil when the prefix matches, or errLinkTargetTypePrefix
// wrapping the first offending position.
func verifyLinkTargetTypePrefix(link LinkDirective, targetSig *types.Signature, typeArgCount int) error {
	if typeArgCount <= 0 {
		return nil
	}
	params := targetSig.Params()
	for position := range typeArgCount {
		paramType := params.At(position).Type()
		if !isReflectType(paramType) {
			return fmt.Errorf("%w: %s -> %s parameter %d must be reflect.Type, got %s",
				errLinkTargetTypePrefix, link.GenericName, link.LinkTarget,
				position, paramType.String())
		}
	}
	return nil
}

// verifyLinkTargetReturnShape confirms each sibling return either
// matches the generic's corresponding return exactly, or is
// reflect.Value when the generic's return mentions a type parameter.
//
// Takes link (LinkDirective) which names the generic and sibling.
// Takes genericSig (*types.Signature) which is the generic's
// signature.
// Takes targetSig (*types.Signature) which is the sibling's
// signature.
//
// Returns nil when the returns line up, or errLinkTargetReturnShape
// wrapping the first offending position.
func verifyLinkTargetReturnShape(link LinkDirective, genericSig, targetSig *types.Signature) error {
	paramSet := typeParamSet(genericSig)
	genericResults := genericSig.Results()
	targetResults := targetSig.Results()
	for position := range genericResults.Len() {
		genericType := genericResults.At(position).Type()
		targetType := targetResults.At(position).Type()
		if typeMentionsParam(genericType, paramSet) {
			if !isReflectValue(targetType) {
				return fmt.Errorf("%w: %s -> %s return %d must be reflect.Value (generic returns %s), got %s",
					errLinkTargetReturnShape, link.GenericName, link.LinkTarget,
					position, genericType.String(), targetType.String())
			}
			continue
		}
		if !types.Identical(genericType, targetType) {
			return fmt.Errorf("%w: %s -> %s return %d type mismatch: generic=%s sibling=%s",
				errLinkTargetReturnShape, link.GenericName, link.LinkTarget,
				position, genericType.String(), targetType.String())
		}
	}
	return nil
}

// typeParamSet collects the generic's declared type parameters into a
// lookup set used by typeMentionsParam.
//
// Takes genericSig (*types.Signature) which is the generic's
// signature.
//
// Returns a set keyed by *types.TypeParam pointer.
func typeParamSet(genericSig *types.Signature) map[*types.TypeParam]struct{} {
	params := genericSig.TypeParams()
	if params == nil || params.Len() == 0 {
		return nil
	}
	set := make(map[*types.TypeParam]struct{}, params.Len())
	for typeParam := range params.TypeParams() {
		set[typeParam] = struct{}{}
	}
	return set
}

// typeMentionsParam reports whether t references any *types.TypeParam
// in params, walking composite types.
//
// Takes t (types.Type) which is the type to inspect.
// Takes params (map[*types.TypeParam]struct{}) which is the owning
// generic's parameter set.
//
// Returns true when any component of t is one of the parameters.
func typeMentionsParam(t types.Type, params map[*types.TypeParam]struct{}) bool {
	return typeMentionsParamAtDepth(t, params, 0)
}

// typeMentionsParamAtDepth is the bounded implementation of
// typeMentionsParam. Recursion beyond maxLinkReturnWalkDepth returns
// false rather than walking further.
//
// Takes t (types.Type) which is the type to inspect.
// Takes params (map[*types.TypeParam]struct{}) which is the owning
// generic's parameter set.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns true when any component of t is one of the parameters.
func typeMentionsParamAtDepth(t types.Type, params map[*types.TypeParam]struct{}, depth int) bool {
	if depth >= maxLinkReturnWalkDepth || t == nil {
		return false
	}
	if typeParam, ok := t.(*types.TypeParam); ok {
		_, found := params[typeParam]
		return found
	}
	if element, ok := singleElementInner(t); ok {
		return typeMentionsParamAtDepth(element, params, depth+1)
	}
	if mapType, ok := t.(*types.Map); ok {
		return typeMentionsParamAtDepth(mapType.Key(), params, depth+1) ||
			typeMentionsParamAtDepth(mapType.Elem(), params, depth+1)
	}
	if named, ok := t.(*types.Named); ok {
		return namedMentionsParam(named, params, depth)
	}
	if alias, ok := t.(*types.Alias); ok {
		return typeMentionsParamAtDepth(types.Unalias(alias), params, depth+1)
	}
	return false
}

// singleElementInner returns the element type of a Pointer, Slice,
// Array, or Chan wrapper. These share the same "wraps a single inner
// type" shape and collapsing them lets typeMentionsParamAtDepth stay
// below the cognitive-complexity limit without losing any cases.
//
// Takes t (types.Type) which is the candidate wrapper type.
//
// Returns the inner type and true for a single-element wrapper; nil
// and false otherwise.
func singleElementInner(t types.Type) (types.Type, bool) {
	switch typeValue := t.(type) {
	case *types.Pointer:
		return typeValue.Elem(), true
	case *types.Slice:
		return typeValue.Elem(), true
	case *types.Array:
		return typeValue.Elem(), true
	case *types.Chan:
		return typeValue.Elem(), true
	}
	return nil, false
}

// namedMentionsParam reports whether any of the named type's instantiated
// type arguments mention a generic parameter from params.
//
// Takes named (*types.Named) which is the named type being inspected.
// Takes params (map[*types.TypeParam]struct{}) which is the owning
// generic's parameter set.
// Takes depth (int) which tracks the current recursion depth so the
// overall walk stays bounded.
//
// Returns true when any component of any type argument is one of
// the parameters.
func namedMentionsParam(named *types.Named, params map[*types.TypeParam]struct{}, depth int) bool {
	args := named.TypeArgs()
	if args == nil {
		return false
	}
	for typeArg := range args.Types() {
		if typeMentionsParamAtDepth(typeArg, params, depth+1) {
			return true
		}
	}
	return false
}

// isReflectValue reports whether t is the reflect.Value struct.
//
// Takes t (types.Type) which is the candidate type.
//
// Returns true when t names reflect.Value.
func isReflectValue(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "reflect" && obj.Name() == "Value"
}

// isReflectType reports whether t is the reflect.Type interface. The
// directive validator runs from extract's own process, so a single
// identity check against the package path + type name is sufficient;
// we do not need to instantiate reflect.TypeFor here.
//
// Takes t (types.Type) which is the candidate parameter type.
//
// Returns true when t names reflect.Type.
func isReflectType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "reflect" && obj.Name() == "Type"
}

// resolveGenericSignature looks up the annotated generic function and
// confirms it actually has type parameters.
//
// Takes scope (*types.Scope) which is the package-level scope.
// Takes link (LinkDirective) which names the generic.
//
// Returns the generic function's signature and nil on success, or a
// wrapped sentinel error when the symbol is missing or not generic.
func resolveGenericSignature(scope *types.Scope, link LinkDirective) (*types.Signature, error) {
	genericObj := scope.Lookup(link.GenericName)
	if genericObj == nil {
		return nil, fmt.Errorf("%w: %s referenced by //piko:link has no declaration",
			errLinkTargetMissing, link.GenericName)
	}
	genericFunc, ok := genericObj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not a function",
			errLinkDirectiveOnNonGeneric, link.GenericName)
	}
	genericSig, ok := genericFunc.Type().(*types.Signature)
	if !ok || genericSig.TypeParams() == nil || genericSig.TypeParams().Len() == 0 {
		return nil, fmt.Errorf("%w: %s", errLinkDirectiveOnNonGeneric, link.GenericName)
	}
	return genericSig, nil
}

// resolveLinkTargetSignature looks up the sibling function and ensures
// it has a usable signature.
//
// Takes scope (*types.Scope) which is the package-level scope.
// Takes link (LinkDirective) which names the sibling.
//
// Returns the sibling's signature and nil on success, or a wrapped
// sentinel error when the sibling is missing or not a function.
func resolveLinkTargetSignature(scope *types.Scope, link LinkDirective) (*types.Signature, error) {
	targetObj := scope.Lookup(link.LinkTarget)
	if targetObj == nil {
		return nil, fmt.Errorf("%w: %s -> %s",
			errLinkTargetMissing, link.GenericName, link.LinkTarget)
	}
	targetFunc, ok := targetObj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%w: %s -> %s",
			errLinkTargetNotFunc, link.GenericName, link.LinkTarget)
	}
	targetSig, ok := targetFunc.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("%w: %s -> %s cannot be type-checked",
			errLinkTargetNotFunc, link.GenericName, link.LinkTarget)
	}
	return targetSig, nil
}

// isValidIdentifier reports whether name is a syntactically valid Go
// identifier.
//
// The Go spec permits any Unicode letter as the leading rune and any
// Unicode letter or digit thereafter, so the check uses unicode rather
// than restricting to ASCII.
//
// Takes name (string) which is the candidate identifier.
//
// Returns true when name parses as a Go identifier.
func isValidIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for index, runeValue := range name {
		if index == 0 {
			if !isIdentifierStart(runeValue) {
				return false
			}
			continue
		}
		if !isIdentifierContinue(runeValue) {
			return false
		}
	}
	return true
}

// isIdentifierStart returns true for characters permitted as the first
// rune of a Go identifier: an underscore or any Unicode letter.
//
// Takes runeValue (rune) which is the candidate rune.
//
// Returns true when runeValue is a letter or underscore.
func isIdentifierStart(runeValue rune) bool {
	return runeValue == '_' || unicode.IsLetter(runeValue)
}

// isIdentifierContinue returns true for characters permitted after the
// first rune of a Go identifier: an identifier-start rune, or any
// Unicode digit.
//
// Takes runeValue (rune) which is the candidate rune.
//
// Returns true when runeValue may appear in an identifier body.
func isIdentifierContinue(runeValue rune) bool {
	if isIdentifierStart(runeValue) {
		return true
	}
	return unicode.IsDigit(runeValue)
}
