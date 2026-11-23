//go:build integration

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

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	eventsTestPage = `<section id="event-page" pk-ev-bound><div id="left"><div id="leaf"></div></div><div id="right"></div>
<output id="event-count">0</output><output id="event-payload"></output><output id="event-log"></output></section><script>
const shadow = (id, html) => { const root = document.getElementById(id).attachShadow({mode:'open'}); root.innerHTML = html; return root; };
const bump = node => { node.textContent = String(Number(node.textContent) + 1); };
const leaf = shadow('leaf', [0, 1, 2, 3].map(n => '<button id="emit-' + n + '">' + n + '</button>').join('') + '<span id="local">0</span><span id="serial">0</span>');
for (const n of [0, 1, 2, 3]) leaf.querySelector('#emit-' + n).onclick = () => {
  bump(leaf.querySelector('#local'));
  document.getElementById('leaf').dispatchEvent(new CustomEvent('fuzz-pick', {bubbles:true, composed:true, detail:n}));
};
for (const owner of ['left', 'right']) {
  const root = shadow(owner, '<slot></slot><button id="tick">tick</button><span id="seen">0</span><span id="serial">0</span>');
  root.querySelector('#tick').onclick = () => bump(root.querySelector('#serial'));
  document.getElementById(owner).addEventListener('fuzz-pick', () => bump(root.querySelector('#seen')));
}
document.getElementById('event-page').addEventListener('fuzz-pick', event => {
  bump(document.getElementById('event-count'));
  document.getElementById('event-payload').textContent = String(event.detail);
  document.getElementById('event-log').textContent += String(event.detail);
});
</script>`
)

func TestBrowserEventsSamplesEveryOwnerAndEmission(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/events" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(writer, eventsTestPage)
		assert.NoError(t, err)
	}))
	defer server.Close()
	ctx, session := openTestBrowser(t, eventsBudget()+20*time.Second)
	input := casefile.EventsInput{Label: "A", Rotate: 1, Delta: 1}
	trace := browserEvents(ctx, session, server.URL, input)
	require.Empty(t, trace.DriverError)
	assert.Equal(t, input, trace.Input)
	assert.Equal(t, "/events", trace.BrowserPath)
	require.Len(t, trace.States, eventsStepCount)
	steps := make([]string, 0, len(trace.States))
	for _, state := range trace.States {
		steps = append(steps, state.Step)
	}
	assert.Equal(t, []string{"mount", "tick-left", "left-flags-1", "left-flags-2", "left-flags-3", "left-flags-0",
		"move-right", "tick-right", "right-flags-1", "right-flags-2", "right-flags-3", "right-flags-0"}, steps)
	final := trace.States[len(trace.States)-1]
	assert.Equal(t, "right", final.Owner)
	assert.True(t, final.SameLeaf)
	assert.Equal(t, 4, final.Left)
	assert.Equal(t, 4, final.Right)
	assert.Equal(t, 8, final.PK)
	assert.Equal(t, 8, final.Local)
	assert.Equal(t, 1, final.LeftSerial)
	assert.Equal(t, 1, final.RightSerial)
	assert.Equal(t, "0", final.Payload)
	assert.Equal(t, "12301230", final.Log)
}
