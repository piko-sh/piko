/*
--- BEGIN AST DUMP ---

<div>
  <button :disabled="state.isButtonDisabled" {P: state.isButtonDisabled}>
    "Click me"
  </button>
  <input type="text" :readonly="!state.isEditable" {P: !state.isEditable} />
  <button [Events: p-on:click.="toggle" {P: toggle}]>
    "Toggle"
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
        const $$initialState = {"isButtonDisabled": true, "isEditable": false};
        const state = makeReactive($$initialState, contextParam);
        function toggle() {
            state.isButtonDisabled = !state.isButtonDisabled;
            state.isEditable = !state.isEditable;
        }
        return {"state": state, "$$initialState": $$initialState, "toggle": toggle};
    }
    class BoolAttrElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggle_evt_1 = (e) => {
                this.$$ctx.toggle.call(this, e);
            };
        }
        static get propTypes () {
            return {"isButtonDisabled": {"type": "boolean"}, "isEditable": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"isButtonDisabled": true, "isEditable": false};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"?disabled": (this.$$ctx.state.isButtonDisabled)}, dom.txt("Click me", "r.0:0:0")), dom.el("input", "r.0:1", {"?readonly": (!this.$$ctx.state.isEditable), "type": "text"}, null), dom.el("button", "r.0:2", {"onClick": this._dir_click_toggle_evt_1}, dom.txt("Toggle", "r.0:2:0"))]));
        }
    }
    customElements.define("bool-attr", BoolAttrElement);
})();