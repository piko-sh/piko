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

package daemon_dto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSPTokenFromContext(t *testing.T) {
	withToken := AcquirePikoRequestCtx()
	defer ReleasePikoRequestCtx(withToken)
	withToken.CSPToken = "abc+/="

	withoutToken := AcquirePikoRequestCtx()
	defer ReleasePikoRequestCtx(withoutToken)

	testCases := []struct {
		ctx  context.Context
		name string
		want string
	}{
		{name: "the carrier holds a token", ctx: WithPikoRequestCtx(t.Context(), withToken), want: "abc+/="},
		{name: "the carrier holds no token", ctx: WithPikoRequestCtx(t.Context(), withoutToken), want: ""},
		{name: "the context has no carrier", ctx: t.Context(), want: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, CSPTokenFromContext(testCase.ctx))
		})
	}
}

func TestCSPTokenAttribute(t *testing.T) {
	testCases := []struct {
		name  string
		token string
		want  string
	}{
		{name: "no token writes nothing", token: "", want: ""},
		{name: "a token becomes an attribute", token: "abc+/=", want: ` nonce="abc+/="`},
		{name: "markup in a token is escaped", token: `a"><b`, want: ` nonce="a&#34;&gt;&lt;b"`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, CSPTokenAttribute(testCase.token))
		})
	}
}
