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

package runtime_test

import (
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
	"piko.sh/piko/wdk/runtime"
)

func TestRuntimeFacadeAPI(t *testing.T) {

	surface := apitest.Surface{

		"TemplateAST":       runtime.TemplateAST{},
		"TemplateNode":      runtime.TemplateNode{},
		"HTMLAttribute":     runtime.HTMLAttribute{},
		"Directive":         runtime.Directive{},
		"NodeType":          runtime.NodeType(0),
		"InternalMetadata":  runtime.InternalMetadata{},
		"AssetRef":          runtime.AssetRef{},
		"ActionPayload":     runtime.ActionPayload{},
		"ActionArgument":    runtime.ActionArgument{},
		"RuntimeDiagnostic": runtime.RuntimeDiagnostic{},
		"Severity":          runtime.Severity(0),

		"NodeElement":  runtime.NodeElement,
		"NodeText":     runtime.NodeText,
		"NodeComment":  runtime.NodeComment,
		"NodeFragment": runtime.NodeFragment,
		"Debug":        runtime.Debug,
		"Info":         runtime.Info,
		"Warning":      runtime.Warning,
		"Error":        runtime.Error,

		"EvaluateTruthiness":     runtime.EvaluateTruthiness,
		"EvaluateStrictEquality": runtime.EvaluateStrictEquality,
		"EvaluateLooseEquality":  runtime.EvaluateLooseEquality,

		"ValueToString": runtime.ValueToString,
		"F":             runtime.F,

		"ClassesFromString":        runtime.ClassesFromString,
		"ClassesFromSlice":         runtime.ClassesFromSlice,
		"ClassesFromMapStringBool": runtime.ClassesFromMapStringBool,
		"MergeClasses":             runtime.MergeClasses,
		"StylesFromString":         runtime.StylesFromString,
		"StylesFromStringMap":      runtime.StylesFromStringMap,
		"MergeStyles":              runtime.MergeStyles,

		"AppendDiagnostic": runtime.AppendDiagnostic,

		"RegisterASTFunc":              runtime.RegisterASTFunc,
		"RegisterCachePolicyFunc":      runtime.RegisterCachePolicyFunc,
		"RegisterMiddlewareFunc":       runtime.RegisterMiddlewareFunc,
		"RegisterSupportedLocalesFunc": runtime.RegisterSupportedLocalesFunc,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
