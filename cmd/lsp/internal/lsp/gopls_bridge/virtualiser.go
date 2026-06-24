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

package gopls_bridge

import (
	"cmp"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	protocol "github.com/politepixels/golang-language-server"
)

const (
	// overlayDirName is the in-module directory holding the synthetic Go overlays.
	//
	// It MUST NOT begin with "." or "_": gopls excludes such directories from workspace
	// package loading, so a dot-prefixed name makes gopls silently withhold diagnostics for
	// the overlay even though it type-checks it for hover and completion. That left the
	// bridge unable to surface Go errors and, since the overlay-analysed signal is derived
	// from a diagnostics publish, intermittently unable to forward interactive requests at
	// all.
	overlayDirName = "piko-lsp"

	// OverlayPathMarker is the separator-delimited path segment that uniquely identifies a
	// synthetic overlay file in a URI or diagnostic message, so the lsp_domain layer can
	// recognise (and drop or window) results that point at the bridge's own overlays.
	OverlayPathMarker = "/" + overlayDirName + "/"
)

// VirtualDoc is the synthetic Go file presented to gopls for one .pk Go block, together
// with the mapper that relates its positions back to the .pk file.
type VirtualDoc struct {
	// Mapper relates positions in the synthetic Go file back to the .pk file.
	Mapper *Mapper

	// Content holds the synthetic Go source presented to gopls.
	Content []byte
}

// VirtualDocInput carries the primitives needed to build a virtual document for a .pk Go
// block. The caller (lsp_domain) extracts these from the parsed file and the annotator's
// VirtualModule, so the gopls_bridge package stays free of those types.
type VirtualDocInput struct {
	// AliasToCanonical maps each .pk import alias used in the block to the canonical Go
	// package path the annotator assigned its component, so the rewritten imports resolve
	// against the satellite overlays.
	AliasToCanonical map[string]string

	// ModuleRoot is the absolute Go module root the gopls child is rooted at.
	ModuleRoot string

	// HashedName is the component's stable identifier, used to place the virtual file in its
	// own synthetic package directory so sibling pages never collide.
	HashedName string

	// BlockContent is the verbatim Go source of the <script type="application/x-go"> block.
	BlockContent string

	// ContentLine is the 1-based .pk line where the block content starts.
	ContentLine int

	// ContentColumn is the 1-based .pk column where the block content starts.
	ContentColumn int
}

// BuildVirtualDoc produces the position-faithful virtual Go document for a .pk Go block:
// the block bytes verbatim with only .pk import paths rewritten in place, placed at a
// synthetic in-module path in its own package directory.
//
// Takes realURI (protocol.DocumentURI) which is the .pk file the block belongs to.
// Takes input (VirtualDocInput) which carries the import aliases, module root and block
// content used to build the overlay.
//
// Returns *VirtualDoc which pairs the synthetic Go content with its position mapper.
func BuildVirtualDoc(realURI protocol.DocumentURI, input VirtualDocInput) *VirtualDoc {
	rewritten := RewriteBlock(input.BlockContent, input.AliasToCanonical)
	virtualPath := filepath.Join(input.ModuleRoot, overlayDirName, sanitisePackageDir(input.HashedName), "source.pk.go")
	virtualURI := fileURI(virtualPath)

	return &VirtualDoc{
		Content: []byte(rewritten),
		Mapper:  NewMapper(realURI, virtualURI, input.ContentLine, input.ContentColumn),
	}
}

// RewriteBlock rewrites .pk import paths in a Go block to their canonical Go package
// paths, preserving the block's line count and every byte outside the rewritten path
// literals. Imports that are not Piko imports, and blocks that fail to parse, are
// returned unchanged (best effort).
//
// Takes block (string) which is the verbatim Go source of the .pk block.
// Takes aliasToCanonical (map[string]string) which maps each .pk import alias to the
// canonical Go package path to substitute.
//
// Returns string which is the block with .pk import paths rewritten in place.
func RewriteBlock(block string, aliasToCanonical map[string]string) string {
	if len(aliasToCanonical) == 0 {
		return block
	}

	fileSet := token.NewFileSet()
	parsed, parseErr := parser.ParseFile(fileSet, "block.go", block, parser.ImportsOnly)
	if parseErr != nil || parsed == nil {
		return block
	}

	type replacement struct {
		text  string
		start int
		end   int
	}

	var replacements []replacement
	for _, importSpec := range parsed.Imports {
		unquoted, unquoteErr := strconv.Unquote(importSpec.Path.Value)
		if unquoteErr != nil || !strings.HasSuffix(unquoted, ".pk") {
			continue
		}
		canonical, ok := aliasToCanonical[importAlias(importSpec)]
		if !ok {
			continue
		}
		replacements = append(replacements, replacement{
			text:  strconv.Quote(canonical),
			start: fileSet.Position(importSpec.Path.ValuePos).Offset,
			end:   fileSet.Position(importSpec.Path.End()).Offset,
		})
	}
	if len(replacements) == 0 {
		return block
	}

	slices.SortFunc(replacements, func(a, b replacement) int { return cmp.Compare(a.start, b.start) })

	var builder strings.Builder
	builder.Grow(len(block))
	cursor := 0
	for _, item := range replacements {
		if item.start < cursor || item.end > len(block) || item.start > item.end {
			continue
		}
		builder.WriteString(block[cursor:item.start])
		builder.WriteString(item.text)
		cursor = item.end
	}
	builder.WriteString(block[cursor:])
	return builder.String()
}

// importAlias returns the local name a Go import is bound to: the explicit alias when
// present, otherwise the path's base with any .pk suffix removed.
//
// Takes importSpec (*ast.ImportSpec) which is the parsed import to inspect.
//
// Returns the local name the import is bound to.
func importAlias(importSpec *ast.ImportSpec) string {
	if importSpec.Name != nil {
		return importSpec.Name.Name
	}
	unquoted, unquoteErr := strconv.Unquote(importSpec.Path.Value)
	if unquoteErr != nil {
		return ""
	}
	return strings.TrimSuffix(path.Base(unquoted), ".pk")
}

// sanitisePackageDir makes a hashed component name safe to use as a directory segment,
// falling back to a constant when empty.
//
// Takes hashedName (string) which is the component's stable identifier.
//
// Returns the sanitised directory segment, or "block" when empty.
func sanitisePackageDir(hashedName string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, hashedName)
	if cleaned == "" {
		return "block"
	}
	return cleaned
}
