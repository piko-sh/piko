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
	"net/url"
	"strconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// reloadSnapshot reads component and form state across reload modes.
	reloadSnapshot = `(() => {
	const ids = window.__reloadNodes;
	const owner = document.getElementById("reload-owner");
	const sibling = document.getElementById("reload-sibling");
	const root = document.getElementById("reload-sample");
	const input = document.getElementById("reload-input");
	return {
		Path: location.pathname,
		Reload: {
			RootStage: root.dataset.stage, ChildStage: document.getElementById("reload-content").textContent,
			LiveOnly: root.dataset.live || "", Preserved: root.dataset.preserved,
			Draft: input.value, Default: input.defaultValue,
			Selection: [input.selectionStart, input.selectionEnd, input.selectionDirection].join(":"),
			Result: owner.shadowRoot.getElementById("reload-result").textContent,
			RootRetained: root === ids.root, InputRetained: input === ids.input,
			Focused: document.activeElement === input,
			SiblingRetained: sibling === ids.sibling,
			SiblingUntouched: sibling.shadowRoot.getElementById("reload-count").textContent === "1"
				&& sibling.shadowRoot.getElementById("reload-result").textContent === "",
			Slotted: owner === ids.owner && owner.shadowRoot.querySelector("slot").assignedElements().includes(root)
		}
	};
})()`
)

// driveReloadModes exercises the saved component and form reload modes.
//
// Takes base (*url.URL) which is the parsed application origin.
// Takes input (casefile.PartialComponentsInput) which selects the partial-component
// journey values.
// Takes run (func(string, ...browserAction) bool) which runs named browser steps and
// retains any failure.
// Takes wait (func(string) browserAction) which builds an action waiting for the selected
// reload stage.
//
// Returns error which reports a draft that cannot be encoded as a script value, or nil
// when the steps ran. run retains step failures.
func driveReloadModes(base *url.URL, input casefile.PartialComponentsInput, run func(string, ...browserAction) bool, wait func(string) browserAction) error {
	draft, err := scriptString("abc-" + input.Edited)
	if err != nil {
		return fmt.Errorf("reload draft: %w", err)
	}
	ready := `["reload-owner", "reload-sibling"].every(id => !!document.getElementById(id)?.shadowRoot?.getElementById("merge"))
		&& !!document.getElementById("reload-sample")`
	setup := `(() => {
		const byID = id => document.getElementById(id);
		window.__reloadNodes = {owner:byID("reload-owner"), sibling:byID("reload-sibling"), root:byID("reload-sample"), input:byID("reload-input")};
		window.__reloadNodes.sibling.shadowRoot.getElementById("bump").click();
	})()`
	for index, mode := range input.ReloadOrder() {
		var actions []browserAction
		if index == 0 {
			actions = append(actions,
				navigate(base.ResolveReference(&url.URL{Path: "/reload-history"}).String()),
				wait(ready), evaluate(setup, nil))
		}
		prepare := `(() => {
			const mode = ` + strconv.Quote(mode) + `;
			const {root, input, owner} = window.__reloadNodes;
			root.dataset.live = "local-" + mode;
			root.dataset.preserved = "keep-" + mode;
			input.focus();
			input.value = ` + draft + `;
			input.setSelectionRange(1, 3, "backward");
			input.dispatchEvent(new Event("input", {bubbles:true}));
			owner.shadowRoot.getElementById(mode).click();
		})()`
		actions = append(actions, evaluate(prepare, nil),
			wait("document.getElementById('reload-owner').shadowRoot.getElementById('reload-result').textContent === "+strconv.Quote(mode)))
		if !run("reload-"+mode, actions...) {
			return nil
		}
	}
	return nil
}
