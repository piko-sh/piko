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
	"strings"
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
			_, err = fmt.Fprint(writer, `<input id="draft"><span id="retained-row">saved</span><button id="journey-bound">Bound</button><script>
			window.__journey.connected=2;
			window.__journey.pages.journey.connected=1;
			document.getElementById('journey-bound').onclick=()=>window.__journey.pages.journey.clicks++;
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
	assert.Equal(t, casefile.ExpectedLifecycle(input).States[:3], observed.States)
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
let dirty = false, leavePage = () => {};
const connectPage = key => {
  const page = window.__journey.pages[key], bound = document.getElementById(key + '-bound');
  const click = () => { page.clicks++; };
  page.connected++;
  bound.addEventListener('click', click);
  leavePage = () => { page.disconnected++; page.cleaned++; bound.removeEventListener('click', click); };
};
const show = path => {
  leavePage();
  dirty = false;
  if (path === '/journey') {
    app.innerHTML = '<input id="draft"><div id="journey-list"><span id="retained-row">saved</span></div>' +
      '<fuzz-watch id="first"></fuzz-watch><fuzz-watch id="second"></fuzz-watch>' +
      '<button id="journey-bound">Bound</button><a id="link-away" href="/journey-away">away</a>';
    document.getElementById('draft').addEventListener('input', () => { dirty = true; });
    connectPage('journey');
  } else {
    app.innerHTML = '<main id="journey-away">Editor closed<button id="away-bound">Bound</button>' +
      '<a id="link-journey" href="/journey">journey</a></main>';
    connectPage('away');
  }
  app.querySelectorAll('a').forEach(link => link.addEventListener('click', event => {
    event.preventDefault();
    window.piko.nav.navigate(link.getAttribute('href'));
  }));
};
window.piko = {
  partials: {reload: () => { document.getElementById('retained-row').textContent = 'saved'; }},
  nav: {navigate: path => {
    if (dirty && !window.confirm('Leave?')) return;
    history.pushState(null, '', path);
    show(path);
  }}
};
window.addEventListener('popstate', () => show(location.pathname));
show(location.pathname);
</script>`

	reuseListingsPage = `<div id="app"></div><script>
class ListingPager extends HTMLElement {
  connectedCallback() {
    const log = window.__journey, owner = this.getAttribute('owner');
    const filter = document.querySelector(this.getAttribute('filter_input'));
    const react = () => { log.deliveries++; log.applied.push(owner); };
    log.connected++;
    filter.addEventListener('change', react);
    this.release = () => filter.removeEventListener('change', react);
  }
  disconnectedCallback() {
    window.__journey.cleaned++;
    this.release();
  }
}
customElements.define('listing-pager', ListingPager);
const app = document.getElementById('app');
const show = path => {
  const listing = path.slice('/reuse-'.length);
  app.innerHTML = '<input id="' + listing + '-filter"><listing-pager owner="' + listing + '" filter_input="#' + listing + '-filter"></listing-pager>' +
    '<a id="link-orders" href="/reuse-orders">orders</a><a id="link-invoices" href="/reuse-invoices">invoices</a>';
  app.querySelectorAll('a').forEach(link => link.addEventListener('click', event => {
    event.preventDefault();
    history.pushState(null, '', link.getAttribute('href'));
    show(link.getAttribute('href'));
  }));
};
window.addEventListener('popstate', () => show(location.pathname));
show(location.pathname);
</script>`
)

func servePage(t *testing.T, path, page string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != path {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(writer, page)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestBrowserLifecycleMatchesTheModelInEitherOrder(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := servePage(t, "/journey", lifecycleJourneyPage)
	ctx, session := openTestBrowser(t, 2*lifecycleBudget(casefile.LifecycleInput{})+10*time.Second)
	for _, reverse := range []bool{false, true} {
		input := casefile.LifecycleInput{Draft: "draft é", Reverse: reverse}
		trace := browserLifecycle(ctx, session, server.URL, input)
		require.Empty(t, trace.DriverError, "reverse=%t failed at %s", reverse, trace.FailedStep)
		assert.Equal(t, casefile.ExpectedLifecycle(input).States, trace.States, "reverse=%t", reverse)
	}
}

func TestBrowserElementReuseMatchesTheModelWhenListingsReconnect(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := servePage(t, casefile.ReuseListingPath(casefile.ReuseFirstListing), reuseListingsPage)
	input := casefile.LifecycleInput{ElementReuse: true}
	ctx, session := openTestBrowser(t, lifecycleBudget(input)+10*time.Second)
	trace := browserLifecycle(ctx, session, server.URL, input)
	require.Empty(t, trace.DriverError, "failed at %s", trace.FailedStep)
	assert.Equal(t, casefile.ExpectedLifecycle(input).States, trace.States)
}

func TestBrowserLifecycleWaitsForADisconnectThatRunsAfterTheNextPageConnects(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	immediate := "leavePage = () => { page.disconnected++; page.cleaned++; bound.removeEventListener('click', click); };"
	late := "leavePage = () => { bound.removeEventListener('click', click); setTimeout(() => { page.disconnected++; page.cleaned++; }, 200); };"
	require.Contains(t, lifecycleJourneyPage, immediate)
	server := servePage(t, "/journey", strings.Replace(lifecycleJourneyPage, immediate, late, 1))
	input := casefile.LifecycleInput{Draft: "draft"}
	ctx, session := openTestBrowser(t, lifecycleBudget(input)+10*time.Second)
	trace := browserLifecycle(ctx, session, server.URL, input)
	require.Empty(t, trace.DriverError, "failed at %s", trace.FailedStep)
	assert.Equal(t, casefile.ExpectedLifecycle(input).States, trace.States)
}
