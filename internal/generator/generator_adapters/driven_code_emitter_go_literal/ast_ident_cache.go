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

package driven_code_emitter_go_literal

import (
	goast "go/ast"

	"piko.sh/piko/internal/goastutil"
)

// cachedIdent returns a cached identifier node for the given name.
// This is a thin wrapper around goastutil.CachedIdent for use within this
// package.
//
// Takes name (string) which specifies the identifier name to look up or cache.
//
// Returns *goast.Ident which is the cached identifier node.
func cachedIdent(name string) *goast.Ident {
	return goastutil.CachedIdent(name)
}

func init() {
	domainIdentifiers := []string{
		runtimePackageName,
		facadePackageName,

		RequestVarName,
		DiagnosticsVarName,
		OkVarName,
		"rootAST", "propsData",
		arenaVarName,

		fieldNodeType,
		fieldTagName,
		fieldTextContent,
		fieldChildren,
		fieldAttributes,
		"RichText",
		"RootNodes",
		"TextContentWriter",
		"InnerHTML",
		"RuntimeAnnotations",
		"NeedsCSRF",

		FieldNameName,
		FieldNameValue,

		"TemplateAST", "TemplateNode", "HTMLAttribute", "RuntimeDiagnostic",
		"InternalMetadata", "AssetRef", "ActionArgument", "ActionPayload",
		"RequestData", NoPropsTypeName, "Metadata",

		"GetNode", "GetTemplateAST", "GetRootNodesSlice", "GetDirectWriter",
		"GetRuntimeAnnotation", "GetStaticCollectionItem", "AppendDiagnostic",
		"EvaluateTruthiness", "ValueToString", "SetLocalStoreFromMap",
		"WithCollectionData", "CollectionData", "URL", "Path",
		"GetArena", "SetArena",

		"Error", "Warning", "Info",

		BlankIdentifier,

		"sortedKeys", "k", "sort", "Strings", "DeepClone",
		"pageData", "renderErr",
		"rootMap", "pageVal", "json", "base64",
		"strconv", "interface{}",
	}

	for _, name := range domainIdentifiers {
		goastutil.RegisterIdent(name)
	}
}
