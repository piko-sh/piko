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
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// advancedSnapshot reads public component state and retained node identities.
	advancedSnapshot = `(() => {
	const q = s => root.querySelector(s);
	if (q('#extra-forms')) {
		const radios = Array.from(root.querySelectorAll('[name="colour"]'));
		return {'radio':radios.filter(r=>r.checked).map(r=>r.value).join(','),'radio-values':radios.map(r=>r.value).join(','),
			'radio-model':text('#extra-colour'),'multi':Array.from(q('#extra-many').selectedOptions,o=>o.value).join(','),
			'multi-model':text('#extra-chosen'),'number':q('#extra-number').value,'number-model':text('#extra-amount')};
	}
	const id = (name,node) => {
		const records = (globalThis.__pikoFuzzNodes ??= new WeakMap());
		if (!records.has(host)) records.set(host,new Map());
		const record=records.get(host);
		if (!record.has(name)) record.set(name,node);
		return String(node != null && record.get(name)===node);
	};
	if (q('#extra-visual')) {
		const editor=q('#extra-editor'), rows=Array.from(root.querySelectorAll('#extra-memo li'));
		const output=q('#extra-output'), markup=q('#extra-markup'), child=markup?.firstElementChild;
		const scoped=Array.from(root.querySelectorAll('#extra-context li'));
		return {branch:Array.from(root.querySelectorAll('#extra-branches span'),s=>s.textContent).join(','),
			'phase-attr':output?.getAttribute('data-phase'), 'note-attr':output?.getAttribute('data-note'),
			markup:markup && [markup.childNodes.length,child?.localName,child?.getAttribute('data-kind'),markup.textContent].join('|'),
			'context-order':scoped.map(r=>r.textContent).join(','),
			'context-identity':String(scoped.length===2 && scoped.every(r=>id('scope-'+r.textContent,r)==='true')),
			shown:String(getComputedStyle(q('#extra-panel')).display!=='none'),class:q('#extra-visual').className,
			opacity:getComputedStyle(q('#extra-visual')).opacity,value:editor.value,focus:String(root.activeElement===editor),
			'editor-identity':id('editor',editor),memo:rows.map(r=>r.textContent).join(','),
			'row-identity':String(rows.length===2 && rows.every((r,i)=>id('memo'+i,r)==='true')),'memo-stable':rows[1]?.className};
	}
	if (q('#extra-nested')) {
		const leaf=q('slot[name="nested-leaf"]').assignedElements()[0];
		return {'leaf-count':leaf.shadowRoot?.querySelector('#leaf-tally')?.textContent ?? '',
			'parent-picked':text('#extra-picked'),
			hooks:leaf.shadowRoot?.querySelector('#leaf-hooks')?.textContent ?? (String(leaf.dataset.mounts||0)+'/'+String(leaf.dataset.unmounts||0)),'leaf-identity':id('child',leaf)};
	}
	if (q('#extra-timeline')) {
		const node=q('#extra-timeline');
		return {'timeline-hidden':String(node.hasAttribute('p-timeline-hidden')),'timeline-class':String(node.classList.contains('played')),'timeline-ref':String(host.refs?.headline===node)};
	}
})()`
)

// advancedInteraction selects the browser action for an extended component operation.
//
// Takes root (string) which selects the component shadow root.
// Takes action (casefile.ClientAction) which selects the browser operation and target
// instance.
//
// Returns browserAction which is the selected browser action.
// Returns bool which reports whether the operation belongs to the extended component
// vocabulary.
// Returns error which reports a value that cannot be encoded as a script string, or nil
// on success.
func advancedInteraction(root string, action casefile.ClientAction) (browserAction, bool, error) {
	run := func(code string) browserAction {
		return evaluate("(() => { const root="+root+"; "+code+" })()", nil)
	}
	click := func(selector string) browserAction { return run("root.querySelector('" + selector + "').click()") }
	switch action.Op {
	case casefile.ChooseRadio:
		return browserActions{run("root.querySelector('#extra-blue').focus()"), typeKeys(" ")}.run, true, nil
	case casefile.SelectMany:
		return browserActions{run("root.querySelector('#extra-many').focus()"), pressKey("Home"), pressKey("ArrowDown", "Shift")}.run, true, nil
	case casefile.TypeNumber:
		return browserActions{run("const el=root.querySelector('#extra-number'); el.focus(); el.select();"), typeKeys(action.Value)}.run, true, nil
	case casefile.RefEdit:
		return browserActions{click("#extra-focus"), typeKeys(action.Value)}.run, true, nil
	case casefile.HideMemo:
		return run("root.querySelectorAll('#extra-memo li')[1].className='marked'; root.querySelector('#extra-toggle').click()"), true, nil
	case casefile.ShowRefocus:
		return browserActions{click("#extra-toggle"), click("#extra-focus")}.run, true, nil
	case casefile.ChildPick:
		return run("root.host.querySelector('fuzz-leaf').shadowRoot.querySelector('#leaf-emit').click()"), true, nil
	case casefile.RemountChild:
		return run("const leaf=root.host.querySelector('fuzz-leaf'),parent=leaf.parentNode,next=leaf.nextSibling; leaf.remove(); parent.insertBefore(leaf,next)"), true, nil
	case casefile.SeekTimeline:
		value, err := scriptString(action.Value)
		if err != nil {
			return nil, true, err
		}
		return run("root.host.setAttribute('time'," + value + scriptGroupEnd), true, nil
	default:
		return nil, false, nil
	}
}
