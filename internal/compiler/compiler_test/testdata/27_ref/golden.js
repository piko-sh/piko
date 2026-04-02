/*
--- BEGIN AST DUMP ---

<div>
  <input type="text" [p-ref: myInput] />
  <button [Events: p-on:click.="focusInput" {P: focusInput}]>
    "Focus Input"
  </button>
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
        function focusInput() {
            this.refs.myInput.focus();
        }
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState, "focusInput": focusInput};
    }
    class PRefComponentElement extends PPElement {
        constructor () {
            super();
            this._dir_click_focusInput_evt_1 = (e) => {
                this.$$ctx.focusInput.call(this, e);
            };
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("input", "r.0:0", {"_ref": "myInput", "type": "text"}, null), dom.el("button", "r.0:1", {"onClick": this._dir_click_focusInput_evt_1}, dom.txt("Focus Input", "r.0:1:0"))]));
        }
    }
    customElements.define("p-ref-component", PRefComponentElement);
})();