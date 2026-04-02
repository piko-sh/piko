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

package ast_domain

// Sanitises AST structures by removing transient data and normalising paths for serialisation and caching.
// Strips diagnostics, converts absolute paths to relative paths, and prepares ASTs for storage or transmission.

import (
	"path/filepath"
	"reflect"
)

// SanitiseForEncoding creates a deep clone of the AST and prepares it
// for stable, portable serialisation. This is the primary entry point for
// this functionality.
//
// The sanitisation process includes:
//   - Removing all diagnostics, which are transient and not part of the
//     core tree structure.
//   - Converting all absolute OriginalSourcePath fields in every annotation
//     to be relative to the provided basePath. This makes the serialised
//     output independent of the machine it was generated on.
//   - Converting all absolute GeneratedSourcePath fields in every annotation
//     to be relative to the provided basePath. This keeps LSP-related
//     paths are also portable.
//
// Takes tree (*TemplateAST) which is the AST to sanitise.
// Takes basePath (string) which is the base path for making paths relative.
//
// Returns *TemplateAST which is a sanitised deep clone of the input tree.
func SanitiseForEncoding(tree *TemplateAST, basePath string) *TemplateAST {
	if tree == nil {
		return nil
	}

	sanitisedAST := tree.DeepClone()
	sanitisedAST.Diagnostics = nil

	if basePath == "" {
		return sanitisedAST
	}

	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return sanitisedAST
	}

	sanitisedAST.Walk(func(node *TemplateNode) bool {
		sanitiseItem(node, absBasePath)
		return true
	})

	return sanitisedAST
}

// isNil checks whether a value is nil, including typed nil values.
//
// Takes i (any) which is the value to check.
//
// Returns bool which is true if i is nil or holds a nil pointer, map, slice,
// interface, or function.
func isNil(i any) bool {
	if i == nil {
		return true
	}
	reflectValue := reflect.ValueOf(i)
	switch reflectValue.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func:
		return reflectValue.IsNil()
	default:
		return false
	}
}

// sanitiseItem cleans the annotations and fields of an AST element.
//
// When item is nil, returns without action.
//
// Takes item (any) which is the AST element to clean.
// Takes basePath (string) which is the path prefix for annotation processing.
func sanitiseItem(item any, basePath string) {
	if isNil(item) {
		return
	}

	sanitiseAnnotations(item, basePath)
	sanitiseFields(item, basePath)
}

// sanitiseAnnotations clears diagnostic data and makes file paths relative
// for annotations in different item types.
//
// Takes item (any) which is the element that holds annotations to clean.
// Takes basePath (string) which is the path prefix to remove from file paths.
func sanitiseAnnotations(item any, basePath string) {
	switch v := item.(type) {
	case Expression:
		sanitiseItem(v.GetGoAnnotation(), basePath)
	case *TemplateNode:
		v.Diagnostics = nil
		sanitiseItem(v.GoAnnotations, basePath)
	case *Directive:
		sanitiseItem(v.GoAnnotations, basePath)
	case *DynamicAttribute:
		sanitiseItem(v.GoAnnotations, basePath)
	case *TextPart:
		sanitiseItem(v.GoAnnotations, basePath)
	case *PropValue:
		sanitiseItem(v.InvokerAnnotation, basePath)
	case *GoGeneratorAnnotation:
		sanitiseGoAnnotationPath(v, basePath)
	}
}

// sanitiseGoAnnotationPath changes absolute paths in a Go generator
// annotation to relative paths with forward slashes.
//
// Takes v (*GoGeneratorAnnotation) which is the annotation to change in place.
// Takes basePath (string) which is the base folder for working out relative
// paths.
func sanitiseGoAnnotationPath(v *GoGeneratorAnnotation, basePath string) {
	if v.OriginalSourcePath != nil {
		absPath := *v.OriginalSourcePath
		if filepath.IsAbs(absPath) {
			relPath, err := filepath.Rel(basePath, absPath)
			if err == nil {
				v.OriginalSourcePath = new(filepath.ToSlash(relPath))
			}
		}
	}

	if v.GeneratedSourcePath != nil {
		absPath := *v.GeneratedSourcePath
		if filepath.IsAbs(absPath) {
			relPath, err := filepath.Rel(basePath, absPath)
			if err == nil {
				v.GeneratedSourcePath = new(filepath.ToSlash(relPath))
			}
		}
	}
}

// sanitiseFields processes path-related fields within the given item.
//
// It handles different AST node types by calling the correct type-specific
// sanitise function. Types that are not supported are ignored.
//
// Takes item (any) which is the AST node or value to sanitise.
// Takes basePath (string) which is the base path used to resolve relative
// paths.
func sanitiseFields(item any, basePath string) {
	switch v := item.(type) {
	case *TemplateNode:
		sanitiseTemplateNode(v, basePath)
	case *Directive:
		sanitiseDirective(v, basePath)
	case *DynamicAttribute:
		sanitiseItem(v.Expression, basePath)
	case *TextPart:
		sanitiseItem(v.Expression, basePath)
	case *GoGeneratorAnnotation:
		sanitiseItem(v.PartialInfo, basePath)
	case *PartialInvocationInfo:
		sanitisePartialInfo(v, basePath)
	case *PropValue:
		sanitiseItem(v.Expression, basePath)
	case Expression:
		sanitiseExpression(v, basePath)
	}
}

// sanitiseTemplateNode cleans a template node by processing its key,
// directives, and collections.
//
// Takes v (*TemplateNode) which is the node to clean.
// Takes basePath (string) which is the base path for resolving references.
func sanitiseTemplateNode(v *TemplateNode, basePath string) {
	sanitiseItem(v.Key, basePath)
	sanitiseNodeDirectives(v, basePath)
	sanitiseNodeCollections(v, basePath)
}

// sanitiseNodeDirectives cleans all directive fields on a template node.
//
// Takes v (*TemplateNode) which is the node whose directives will be cleaned.
// Takes basePath (string) which is the base path used for cleaning.
func sanitiseNodeDirectives(v *TemplateNode, basePath string) {
	sanitiseItem(v.DirIf, basePath)
	sanitiseItem(v.DirElseIf, basePath)
	sanitiseItem(v.DirElse, basePath)
	sanitiseItem(v.DirFor, basePath)
	sanitiseItem(v.DirShow, basePath)
	sanitiseItem(v.DirModel, basePath)
	sanitiseItem(v.DirRef, basePath)
	sanitiseItem(v.DirSlot, basePath)
	sanitiseItem(v.DirClass, basePath)
	sanitiseItem(v.DirStyle, basePath)
	sanitiseItem(v.DirText, basePath)
	sanitiseItem(v.DirHTML, basePath)
	sanitiseItem(v.DirKey, basePath)
	sanitiseItem(v.DirContext, basePath)
	sanitiseItem(v.DirScaffold, basePath)
}

// sanitiseNodeCollections cleans paths in all collections within a template
// node, including dynamic attributes, rich text, events, and binds.
//
// Takes v (*TemplateNode) which is the node whose collections will be cleaned.
// Takes basePath (string) which is the base path used for cleaning.
func sanitiseNodeCollections(v *TemplateNode, basePath string) {
	for i := range v.DynamicAttributes {
		sanitiseItem(&v.DynamicAttributes[i], basePath)
	}
	for i := range v.RichText {
		sanitiseItem(&v.RichText[i], basePath)
	}
	for _, directives := range v.OnEvents {
		for i := range directives {
			sanitiseItem(&directives[i], basePath)
		}
	}
	for _, directives := range v.CustomEvents {
		for i := range directives {
			sanitiseItem(&directives[i], basePath)
		}
	}
	for _, directive := range v.Binds {
		sanitiseItem(directive, basePath)
	}
}

// sanitiseDirective updates the paths in a directive to be relative to the
// base path.
//
// Takes v (*Directive) which is the directive to update.
// Takes basePath (string) which is the base path for making paths relative.
func sanitiseDirective(v *Directive, basePath string) {
	sanitiseItem(v.Expression, basePath)
	sanitiseItem(v.ChainKey, basePath)
}

// sanitisePartialInfo removes base path prefixes from a partial invocation
// info.
//
// Takes v (*PartialInvocationInfo) which holds the data to clean.
// Takes basePath (string) which is the prefix to remove from values.
func sanitisePartialInfo(v *PartialInvocationInfo, basePath string) {
	for _, propVal := range v.PassedProps {
		sanitiseItem(&propVal, basePath)
	}
	for _, propVal := range v.RequestOverrides {
		sanitiseItem(&propVal, basePath)
	}
}

// sanitiseExpression walks an expression tree and applies the base path to all
// file positions within nested items.
//
// Takes v (Expression) which is the expression node to process.
// Takes basePath (string) which is the base path to apply to file positions.
func sanitiseExpression(v Expression, basePath string) {
	switch e := v.(type) {
	case *MemberExpression:
		sanitiseItem(e.Base, basePath)
		sanitiseItem(e.Property, basePath)
	case *IndexExpression:
		sanitiseItem(e.Base, basePath)
		sanitiseItem(e.Index, basePath)
	case *CallExpression:
		sanitiseItem(e.Callee, basePath)
		for _, argument := range e.Args {
			sanitiseItem(argument, basePath)
		}
	case *BinaryExpression:
		sanitiseItem(e.Left, basePath)
		sanitiseItem(e.Right, basePath)
	case *UnaryExpression:
		sanitiseItem(e.Right, basePath)
	case *TernaryExpression:
		sanitiseItem(e.Condition, basePath)
		sanitiseItem(e.Consequent, basePath)
		sanitiseItem(e.Alternate, basePath)
	case *ForInExpression:
		sanitiseItem(e.Collection, basePath)
		sanitiseItem(e.IndexVariable, basePath)
		sanitiseItem(e.ItemVariable, basePath)
	case *ObjectLiteral:
		for _, pair := range e.Pairs {
			sanitiseItem(pair, basePath)
		}
	case *ArrayLiteral:
		for _, element := range e.Elements {
			sanitiseItem(element, basePath)
		}
	case *TemplateLiteral:
		for i := range e.Parts {
			sanitiseItem(e.Parts[i].Expression, basePath)
		}
	}
}
