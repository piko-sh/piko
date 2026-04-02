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

package lsp_domain

import (
	"context"
	"path/filepath"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// httpPrefixLen is the length of "http://".
	httpPrefixLen = 7

	// httpsPrefixLen is the length of the "https://" prefix.
	httpsPrefixLen = 8

	// dataPrefixLen is the length of the "data:" prefix for data URI checks.
	dataPrefixLen = 5
)

// GetDocumentLinks finds and returns clickable links within the document.
// This includes partial component imports (is="..." attributes) and asset
// references.
//
// Takes ctx (context.Context) which carries tracing values for resolver calls.
//
// Returns []protocol.DocumentLink which contains all links found in the
// document.
// Returns error when link extraction fails.
func (d *document) GetDocumentLinks(ctx context.Context) ([]protocol.DocumentLink, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return []protocol.DocumentLink{}, nil
	}

	links := []protocol.DocumentLink{}

	d.AnnotationResult.AnnotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		links = append(links, d.extractLinksFromNode(ctx, node)...)
		return true
	})

	return links, nil
}

// extractLinksFromNode extracts document links from a single template node's
// attributes.
//
// Takes ctx (context.Context) which carries tracing values for the resolver.
// Takes node (*ast_domain.TemplateNode) which is the node to extract links
// from.
//
// Returns []protocol.DocumentLink which contains the extracted document links.
func (d *document) extractLinksFromNode(ctx context.Context, node *ast_domain.TemplateNode) []protocol.DocumentLink {
	links := []protocol.DocumentLink{}

	for i := range node.Attributes {
		attribute := &node.Attributes[i]
		if link := d.tryCreateLinkFromAttribute(ctx, node, attribute); link != nil {
			links = append(links, *link)
		}
	}

	return links
}

// tryCreateLinkFromAttribute attempts to create a document link from an
// attribute.
//
// Takes ctx (context.Context) which carries tracing values for the resolver.
// Takes node (*ast_domain.TemplateNode) which provides the template context.
// Takes attribute (*ast_domain.HTMLAttribute) which contains the
// attribute to check.
//
// Returns *protocol.DocumentLink which is the created link, or nil if the
// attribute cannot be resolved to a link.
func (d *document) tryCreateLinkFromAttribute(ctx context.Context, node *ast_domain.TemplateNode, attribute *ast_domain.HTMLAttribute) *protocol.DocumentLink {
	if attribute.Value == "" {
		return nil
	}

	switch attribute.Name {
	case "is":
		return d.resolvePartialLink(ctx, node, attribute.Value, attribute.NameLocation)
	case "src", "href":
		return d.createAssetLink(ctx, attribute.Value, attribute.NameLocation)
	default:
		return nil
	}
}

// resolvePartialLink resolves a partial component alias to a document link.
//
// Takes ctx (context.Context) which carries tracing values for the resolver.
// Takes node (*ast_domain.TemplateNode) which provides the template node
// containing the partial reference.
// Takes alias (string) which specifies the partial component alias to resolve.
// Takes attributeLocation (ast_domain.Location) which defines
// the location for the link range.
//
// Returns *protocol.DocumentLink which links to the partial component file, or
// nil if the alias cannot be resolved.
func (d *document) resolvePartialLink(ctx context.Context, node *ast_domain.TemplateNode, alias string, attributeLocation ast_domain.Location) *protocol.DocumentLink {
	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return nil
	}

	invokerComp := d.getInvokerComponent(node)
	if invokerComp == nil {
		return nil
	}

	partialPath := findPartialImportPath(invokerComp, alias)
	if partialPath == "" {
		return nil
	}

	targetURI := d.resolvePartialToURI(ctx, partialPath, invokerComp.Source.SourcePath)
	if targetURI == "" {
		return nil
	}

	return buildDocumentLink(alias, attributeLocation, targetURI, "Go to partial component: "+alias)
}

// getInvokerComponent retrieves the VirtualComponent for the node's invoker.
//
// Takes node (*ast_domain.TemplateNode) which specifies the template node to
// look up.
//
// Returns *annotator_dto.VirtualComponent which is the component for the
// node's invoker, or nil if the node has no annotations or the invoker
// cannot be found.
func (d *document) getInvokerComponent(node *ast_domain.TemplateNode) *annotator_dto.VirtualComponent {
	if node.GoAnnotations == nil || node.GoAnnotations.OriginalPackageAlias == nil {
		return nil
	}

	invokerHashedName := *node.GoAnnotations.OriginalPackageAlias
	if invokerHashedName == "" {
		return nil
	}

	invokerComp, ok := d.AnnotationResult.VirtualModule.ComponentsByHash[invokerHashedName]
	if !ok {
		return nil
	}
	return invokerComp
}

// resolvePartialToURI resolves a partial import path to a document URI.
//
// Takes ctx (context.Context) which carries tracing values for the resolver.
// Takes partialPath (string) which is the partial import path to resolve.
// Takes invokerSourcePath (string) which is the source path of the invoker.
//
// Returns protocol.DocumentURI which is the resolved URI, using the resolver
// if available or falling back to the virtual module path.
func (d *document) resolvePartialToURI(ctx context.Context, partialPath, invokerSourcePath string) protocol.DocumentURI {
	if d.Resolver != nil {
		resolvedPath, err := d.Resolver.ResolvePKPath(ctx, partialPath, invokerSourcePath)
		if err == nil {
			return uri.File(resolvedPath)
		}
	}

	return d.resolvePartialViaVirtualModule(partialPath)
}

// resolvePartialViaVirtualModule resolves a partial path using the
// VirtualModule graph.
//
// Takes partialPath (string) which is the path to look up in the graph.
//
// Returns protocol.DocumentURI which is the resolved file URI, or empty if
// the path cannot be found.
func (d *document) resolvePartialViaVirtualModule(partialPath string) protocol.DocumentURI {
	partialHashedName, ok := d.AnnotationResult.VirtualModule.Graph.PathToHashedName[partialPath]
	if !ok {
		return ""
	}

	partialComp, ok := d.AnnotationResult.VirtualModule.ComponentsByHash[partialHashedName]
	if !ok {
		return ""
	}
	return protocol.DocumentURI("file://" + partialComp.Source.SourcePath)
}

// createAssetLink creates a document link for an asset reference such as a
// src or href attribute.
//
// Takes ctx (context.Context) which carries tracing values for the resolver.
// Takes assetPath (string) which is the path from the attribute value.
// Takes attributeLocation (ast_domain.Location) which is the
// source location of the attribute value.
//
// Returns *protocol.DocumentLink which is the resolved link, or nil if the
// path is an absolute URL, data URI, or cannot be resolved.
func (d *document) createAssetLink(ctx context.Context, assetPath string, attributeLocation ast_domain.Location) *protocol.DocumentLink {
	if startsWithHTTP(assetPath) || startsWithData(assetPath) {
		return nil
	}

	var targetURI protocol.DocumentURI
	if d.Resolver != nil {
		containingDir := filepath.Dir(d.URI.Filename())

		resolvedPath, err := d.Resolver.ResolveCSSPath(ctx, assetPath, containingDir)
		if err == nil && resolvedPath != "" {
			targetURI = uri.File(resolvedPath)
		}
	}

	if targetURI == "" {
		return nil
	}

	valueStartLine := safeconv.IntToUint32(attributeLocation.Line - 1)
	valueStartChar := safeconv.IntToUint32(attributeLocation.Column - 1)
	valueEndChar := valueStartChar + safeconv.IntToUint32(len(assetPath))

	return &protocol.DocumentLink{
		Range: protocol.Range{
			Start: protocol.Position{Line: valueStartLine, Character: valueStartChar},
			End:   protocol.Position{Line: valueStartLine, Character: valueEndChar},
		},
		Target:  targetURI,
		Tooltip: "Go to asset: " + assetPath,
	}
}

// findPartialImportPath finds the import path for a partial alias in a
// component.
//
// Takes comp (*annotator_dto.VirtualComponent) which contains the source
// imports to search.
// Takes alias (string) which is the partial alias to match.
//
// Returns string which is the matching import path, or empty if not found.
func findPartialImportPath(comp *annotator_dto.VirtualComponent, alias string) string {
	for _, imp := range comp.Source.PikoImports {
		if imp.Alias == alias {
			return imp.Path
		}
	}
	return ""
}

// buildDocumentLink creates a document link for an attribute value.
//
// Takes value (string) which is the text to create the link for.
// Takes attributeLocation (ast_domain.Location) which specifies
// the source location.
// Takes targetURI (protocol.DocumentURI) which is the link destination.
// Takes tooltip (string) which provides hover text for the link.
//
// Returns *protocol.DocumentLink which is the configured link ready for use.
func buildDocumentLink(value string, attributeLocation ast_domain.Location, targetURI protocol.DocumentURI, tooltip string) *protocol.DocumentLink {
	valueStartLine := safeconv.IntToUint32(attributeLocation.Line - 1)
	valueStartChar := safeconv.IntToUint32(attributeLocation.Column - 1)
	valueEndChar := valueStartChar + safeconv.IntToUint32(len(value))

	return &protocol.DocumentLink{
		Range: protocol.Range{
			Start: protocol.Position{Line: valueStartLine, Character: valueStartChar},
			End:   protocol.Position{Line: valueStartLine, Character: valueEndChar},
		},
		Target:  targetURI,
		Tooltip: tooltip,
	}
}

// startsWithHTTP checks if a string starts with http:// or https://.
//
// Takes s (string) which is the string to check for an HTTP prefix.
//
// Returns bool which is true if the string starts with http:// or https://.
func startsWithHTTP(s string) bool {
	return len(s) >= httpPrefixLen && (s[:httpPrefixLen] == "http://" || (len(s) >= httpsPrefixLen && s[:httpsPrefixLen] == "https://"))
}

// startsWithData checks if a string starts with the "data:" prefix.
//
// Takes s (string) which is the string to check.
//
// Returns bool which is true if the string starts with "data:", false
// otherwise.
func startsWithData(s string) bool {
	return len(s) >= dataPrefixLen && s[:dataPrefixLen] == "data:"
}
