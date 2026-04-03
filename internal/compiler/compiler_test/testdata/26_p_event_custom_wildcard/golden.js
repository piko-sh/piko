/*
--- BEGIN AST DUMP ---

<div>
  <child-component [Events: p-event:item-selected.="handleSelection($event)" {P: handleSelection($event)}] />
  <p>
    <RichText>
      "Parent selected: "
      {{ state.selectedItem }}
    </RichText>
  </p>
</div>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"selectedItem": "None"};
        const state = makeReactive($$initialState, contextParam);
        function handleSelection(event) {
            state.selectedItem = event.detail.name;
        }
        return {"state": state, "$$initialState": $$initialState, "handleSelection": handleSelection};
    }
    class ParentComponentElement extends PPElement {
        constructor () {
            super();
            this._dir_item_selected_handleSelection_evt_1 = (e) => {
                this.$$ctx.handleSelection.call(this, e);
            };
        }
        static get propTypes () {
            return {"selectedItem": {"type": "string"}};
        }
        static get defaultProps () {
            return {"selectedItem": "None"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("child-component", "r.0:0", {"pe:item-selected": this._dir_item_selected_handleSelection_evt_1}, null), dom.el("p", "r.0:1", {}, dom.txt("Parent selected: " + String(this.$$ctx.state.selectedItem ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("parent-component", ParentComponentElement);
})();