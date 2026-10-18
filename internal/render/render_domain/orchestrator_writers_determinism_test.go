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

package render_domain

import (
	"bytes"
	"testing"

	qt "github.com/valyala/quicktemplate"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	eventDirectiveDeterminismRuns = 32
)

func TestWriteEventDirectives_WritesEventsInNameOrder(t *testing.T) {
	t.Parallel()

	events := map[string][]ast_domain.Directive{
		"submit":    {{RawExpression: "onSubmit", Modifier: "prevent"}},
		"click":     {{RawExpression: "onClick"}, {RawExpression: "onClickAgain", Modifier: "stop"}},
		"blur":      {{RawExpression: "onBlur"}},
		"focus":     {{RawExpression: "onFocus"}},
		"keydown":   {{RawExpression: "onKeydown"}},
		"mouseover": {{RawExpression: "onMouseover"}},
		"input":     {{RawExpression: "onInput"}},
		"change":    {{RawExpression: "onChange"}},
	}
	want := ` p-on:blur="onBlur"` +
		` p-on:change="onChange"` +
		` p-on:click="onClick"` +
		` p-on:click.stop="onClickAgain"` +
		` p-on:focus="onFocus"` +
		` p-on:input="onInput"` +
		` p-on:keydown="onKeydown"` +
		` p-on:mouseover="onMouseover"` +
		` p-on:submit.prevent="onSubmit"`

	for range eventDirectiveDeterminismRuns {
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)

		writeEventDirectives(events, pOnPrefix, qw)
		qt.ReleaseWriter(qw)

		require.Equal(t, want, buffer.String(), "event attributes must be written in name order on every run")
	}
}
