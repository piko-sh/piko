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
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestClampActiveParam(t *testing.T) {
	testCases := []struct {
		name        string
		activeParam int
		paramCount  int
		want        int
	}{
		{
			name:        "negative activeParam is clamped to zero",
			activeParam: -1,
			paramCount:  3,
			want:        0,
		},
		{
			name:        "in-range activeParam is unchanged",
			activeParam: 1,
			paramCount:  3,
			want:        1,
		},
		{
			name:        "activeParam exceeding count is clamped to last index",
			activeParam: 5,
			paramCount:  3,
			want:        2,
		},
		{
			name:        "zero params returns zero",
			activeParam: 0,
			paramCount:  0,
			want:        0,
		},
		{
			name:        "activeParam at boundary with single param",
			activeParam: 0,
			paramCount:  1,
			want:        0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := clampActiveParam(tc.activeParam, tc.paramCount)
			if got != tc.want {
				t.Errorf("clampActiveParam(%d, %d) = %d, want %d",
					tc.activeParam, tc.paramCount, got, tc.want)
			}
		})
	}
}

func TestEmptySignatureHelp(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "returns non-nil SignatureHelp with empty signatures",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := emptySignatureHelp()
			if err != nil {
				t.Fatalf("emptySignatureHelp() returned unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("emptySignatureHelp() returned nil SignatureHelp")
			}
			if got.Signatures == nil {
				t.Fatal("Signatures slice is nil, want non-nil empty slice")
			}
			if len(got.Signatures) != 0 {
				t.Errorf("len(Signatures) = %d, want 0", len(got.Signatures))
			}
		})
	}
}

func TestBuildSignatureHelp(t *testing.T) {
	testCases := []struct {
		funcSig         *inspector_dto.FunctionSignature
		name            string
		calleeName      string
		wantLabelPrefix string
		activeParam     int
		wantParamCount  int
		wantActiveParam uint32
	}{
		{
			name:       "two params with activeParam zero",
			calleeName: "myFunc",
			funcSig: &inspector_dto.FunctionSignature{
				Params:  []string{"a int", "b string"},
				Results: []string{"error"},
			},
			activeParam:     0,
			wantParamCount:  2,
			wantActiveParam: 0,
			wantLabelPrefix: "myFunc",
		},
		{
			name:       "zero params with activeParam zero",
			calleeName: "noArgs",
			funcSig: &inspector_dto.FunctionSignature{
				Params:  []string{},
				Results: []string{"int"},
			},
			activeParam:     0,
			wantParamCount:  0,
			wantActiveParam: 0,
			wantLabelPrefix: "noArgs",
		},
		{
			name:       "activeParam exceeding count is clamped",
			calleeName: "twoParams",
			funcSig: &inspector_dto.FunctionSignature{
				Params:  []string{"x float64", "y float64"},
				Results: []string{"float64"},
			},
			activeParam:     3,
			wantParamCount:  2,
			wantActiveParam: 1,
			wantLabelPrefix: "twoParams",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().Build()
			got := document.buildSignatureHelp(tc.calleeName, tc.funcSig, tc.activeParam)

			if got == nil {
				t.Fatal("buildSignatureHelp() returned nil")
			}
			if len(got.Signatures) != 1 {
				t.Fatalf("len(Signatures) = %d, want 1", len(got.Signatures))
			}

			sig := got.Signatures[0]

			expectedLabel := tc.calleeName + tc.funcSig.ToSignatureString()
			if sig.Label != expectedLabel {
				t.Errorf("Label = %q, want %q", sig.Label, expectedLabel)
			}

			if len(sig.Parameters) != tc.wantParamCount {
				t.Errorf("len(Parameters) = %d, want %d", len(sig.Parameters), tc.wantParamCount)
			}

			for i, param := range sig.Parameters {
				if i < len(tc.funcSig.Params) {
					if param.Label != tc.funcSig.Params[i] {
						t.Errorf("Parameters[%d].Label = %q, want %q", i, param.Label, tc.funcSig.Params[i])
					}
				}
			}

			if got.ActiveParameter != tc.wantActiveParam {
				t.Errorf("ActiveParameter = %d, want %d", got.ActiveParameter, tc.wantActiveParam)
			}
		})
	}
}

func TestGetSignatureHelp_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult returns empty SignatureHelp",
			document: newTestDocumentBuilder().Build(),
		},
		{
			name: "nil AnnotatedAST returns empty SignatureHelp",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: nil,
				}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position := protocol.Position{Line: 0, Character: 0}
			got, err := tc.document.GetSignatureHelp(position)
			if err != nil {
				t.Fatalf("GetSignatureHelp() returned unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("GetSignatureHelp() returned nil SignatureHelp")
			}
			if got.Signatures == nil {
				t.Fatal("Signatures slice is nil, want non-nil empty slice")
			}
			if len(got.Signatures) != 0 {
				t.Errorf("len(Signatures) = %d, want 0", len(got.Signatures))
			}
		})
	}
}
