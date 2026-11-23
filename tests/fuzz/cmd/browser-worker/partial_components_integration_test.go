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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	recordListPage = `<html lang="en"><body><svg hidden><symbol id="icon" viewBox="0 0 10 10"><path d="M0 0L10 0"/></symbol></svg>
<div id="record-rows" data-revision="0"></div><output id="record-action" data-revision="0"></output><output id="record-log"></output>
<button id="record-edit">edit</button><button id="record-reorder">reorder</button>
<button id="record-remove">remove</button><button id="record-restore">restore</button><script>
class RecordChip extends HTMLElement {
  connectedCallback() {
    if (this.shadowRoot) return;
    const root = this.attachShadow({mode:'open'});
    root.innerHTML = '<button id="pick">pick</button><span id="label"></span><span id="score"></span><span id="count">0</span><slot name="caption"></slot>';
    root.querySelector('#label').textContent = this.getAttribute('label');
    root.querySelector('#score').textContent = this.getAttribute('score');
    root.querySelector('#pick').onclick = () => {
      const count = root.querySelector('#count');
      count.textContent = String(Number(count.textContent) + 1);
      this.dispatchEvent(new CustomEvent('record-picked', {bubbles:true, composed:true, detail:{label:this.getAttribute('label')}}));
    };
  }
}
customElements.define('record-chip', RecordChip);
let records = [{key:'amber', label:'Amber'}, {key:'blue', label:'Blue'}];
let removed;
const holder = document.getElementById('record-rows'), action = document.getElementById('record-action');
const render = () => {
  holder.innerHTML = records.map(record => '<article data-record="' + record.key + '"><record-chip label="' + record.label + '" score="3">' +
    '<span slot="caption">' + record.label + '</span></record-chip><a href="/record/' + record.key + '" aria-label="' + record.label + '">' +
    '<svg viewBox="0 0 10 10"><use href="#icon"></use></svg></a></article>').join('');
  holder.dataset.revision = String(Number(holder.dataset.revision) + 1);
};
const mutate = change => { change(); render(); action.dataset.revision = String(Number(action.dataset.revision) + 1); };
document.getElementById('record-edit').onclick = () => mutate(() => { records[0].label += ' edited'; });
document.getElementById('record-reorder').onclick = () => mutate(() => records.reverse());
document.getElementById('record-remove').onclick = () => mutate(() => { removed = records.pop(); });
document.getElementById('record-restore').onclick = () => mutate(() => records.push(removed));
document.addEventListener('record-picked', event => { document.getElementById('record-log').textContent += event.detail.label + ';'; });
document.addEventListener('click', event => {
  const link = event.target.closest('a');
  if (!link) return;
  event.preventDefault();
  fetch(link.href).then(response => response.text()).then(body => {
    history.pushState(null, '', link.href);
    document.body.insertAdjacentHTML('beforeend', new DOMParser().parseFromString(body, 'text/html').body.innerHTML);
  });
});
window.addEventListener('popstate', () => document.getElementById('record-detail')?.remove());
render();
</script></body></html>`
)

func TestBrowserPartialComponentsCompletesTheJourney(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		var err error
		switch request.URL.Path {
		case "/record-list":
			_, err = fmt.Fprint(writer, recordListPage)
		case "/record/amber", "/record/blue":
			_, err = fmt.Fprintf(writer, `<html lang="en"><main id="record-detail" data-revision="0"><p id="summary">%s</p></main></html>`, request.URL.Path[len("/record/"):])
		default:
			http.NotFound(writer, request)
		}
		assert.NoError(t, err)
	}))
	defer server.Close()
	input := casefile.PartialComponentsInput{Labels: [2]string{"Amber", "Blue"}, Edited: "edited"}
	ctx, session := openTestBrowser(t, partialComponentsBudget(input))
	trace := browserPartialComponents(ctx, session, server.URL, input)
	require.Empty(t, trace.DriverError)
	assert.Equal(t, input, trace.Input)
	steps := make([]string, 0, len(trace.States))
	for _, state := range trace.States {
		steps = append(steps, state.Step)
	}
	assert.Equal(t, []string{"mount", "select", "edit", "select-edited", "reorder", "select-sibling", "remove", "detached-event",
		"restore", "select-restored", "detail", "back"}, steps)
	mount := trace.States[0]
	assert.Equal(t, "/record-list", mount.Path)
	assert.Equal(t, "en", mount.Locale)
	require.Len(t, mount.Rows, 2)
	row := mount.Rows[0]
	assert.Equal(t, "amber", row.Key)
	assert.Equal(t, "Amber", row.Shadow)
	assert.Equal(t, 3, row.ShadowScore)
	assert.Equal(t, "Amber", row.Caption)
	assert.Equal(t, "/record/amber", row.Link)
	assert.Contains(t, row.Icon, "#icon|0 0 10 10|0 0 10 10|0,0|10,0")

	edited := trace.States[2]
	assert.Equal(t, revisionEdit, edited.ActionRevision)
	assert.Equal(t, "Amber edited", edited.Rows[0].Label)
	assert.Len(t, trace.States[6].Rows, 1)
	assert.Equal(t, "Amber;Amber edited;Blue;", trace.States[6].Log)
	assert.Equal(t, revisionRestore, trace.States[8].ActionRevision)
	detail := trace.States[10]
	assert.Equal(t, "blue", detail.Detail)
	assert.Equal(t, -1, detail.ActionRevision)
	assert.Len(t, trace.States[11].Rows, 2)
}
