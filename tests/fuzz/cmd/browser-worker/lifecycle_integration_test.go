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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestLifecycleWorkerSavesPartialBrowserReceipt(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		var err error
		switch request.URL.Path {
		case "/":
			_, err = fmt.Fprint(writer, `<test-counter></test-counter><script>
			const root=document.querySelector('test-counter').attachShadow({mode:'open'});
			root.innerHTML='<output id="count">1</output><button id="increment">add</button>';
			root.querySelector('button').onclick=()=>root.querySelector('output').textContent='2';
			</script>`)
		case "/journey":
			_, err = fmt.Fprint(writer, `<input id="draft"><span id="retained-row">saved</span><script>
			window.__journey.connected=2;
			window.piko={partials:{reload:()=>document.getElementById('retained-row').textContent='saved'},
				nav:{navigate:()=>{throw new Error('local navigation interrupted')}}};
			</script>`)
		default:
			http.NotFound(writer, request)
		}
		assert.NoError(t, err)
	}))
	defer server.Close()
	input := casefile.LifecycleInput{Draft: "typed draft"}
	recipe, err := json.Marshal(input)
	require.NoError(t, err)
	t.Setenv(envLifecycle, string(recipe))
	result := filepath.Join(t.TempDir(), "counter.json")
	require.NoError(t, probe(t.Context(), server.URL, "test-counter", result, nil))
	data, err := os.ReadFile(result)
	require.NoError(t, err)
	var saved casefile.ClientTransition
	require.NoError(t, json.Unmarshal(data, &saved))
	observed := saved.Lifecycle
	require.NotNil(t, observed)
	assert.Equal(t, "cancel-navigation", observed.FailedStep)
	assert.Contains(t, observed.DriverError, "local navigation interrupted")
	assert.Equal(t, casefile.ExpectedLifecycle(input).States[:2], observed.States)
}

const (
	lifecycleJourneyPage = `<div id="app"></div><script>
class FuzzWatch extends HTMLElement {
  connectedCallback() {
    const log = window.__journey, root = this.shadowRoot || this.attachShadow({mode:'open'});
    root.innerHTML = '<output id="received"></output>';
    log.connected++;
    this.active = true;
    this.listen = () => {
      log.deliveries++;
      if (log.started) return;
      const owner = this.id;
      const gate = new Promise(resolve => { log.release[owner] = resolve; });
      log.ready.push(owner);
      gate.then(() => {
        if (this.active) { root.querySelector('#received').textContent = owner; log.applied.push(owner); }
        log.finished.push(owner);
      });
    };
    window.addEventListener('journey-request', this.listen);
  }
  disconnectedCallback() {
    this.active = false;
    window.__journey.cleaned++;
    window.removeEventListener('journey-request', this.listen);
  }
}
customElements.define('fuzz-watch', FuzzWatch);
const app = document.getElementById('app');
const mount = () => {
  app.innerHTML = '<input id="draft"><div id="journey-list"><span id="retained-row">saved</span></div>' +
    '<fuzz-watch id="first"></fuzz-watch><fuzz-watch id="second"></fuzz-watch>';
};
window.piko = {
  partials: {reload: () => { document.getElementById('retained-row').textContent = 'saved'; }},
  nav: {navigate: path => {
    if (!window.confirm('Leave?')) return;
    history.pushState(null, '', path);
    app.innerHTML = '<main id="journey-away">Editor closed</main>';
  }}
};
window.addEventListener('popstate', () => { if (location.pathname === '/journey') mount(); });
mount();
</script>`
)

func TestBrowserLifecycleMatchesTheModelInEitherOrder(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/journey" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(writer, lifecycleJourneyPage)
		assert.NoError(t, err)
	}))
	defer server.Close()
	ctx, session := openTestBrowser(t, 2*lifecycleBudget()+10*time.Second)
	for _, reverse := range []bool{false, true} {
		input := casefile.LifecycleInput{Draft: "draft é", Reverse: reverse}
		trace := browserLifecycle(ctx, session, server.URL, input)
		require.Empty(t, trace.DriverError, "reverse=%t failed at %s", reverse, trace.FailedStep)
		assert.Equal(t, casefile.ExpectedLifecycle(input).States, trace.States, "reverse=%t", reverse)
	}
}
