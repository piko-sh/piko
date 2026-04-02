/*
--- BEGIN AST DUMP ---

<div>
  <button [Events: p-on:click.="toggle" {P: toggle}]>
    "Toggle"
  </button>
  <p [p-show: state.show]>
    "Visible"
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
        const $$initialState = {"show": true};
        const state = makeReactive($$initialState, contextParam);
        function toggle() {
            state.show = !state.show;
        }
        return {"state": state, "$$initialState": $$initialState, "toggle": toggle};
    }
    class PShowSimpleElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggle_evt_1 = (e) => {
                this.$$ctx.toggle.call(this, e);
            };
        }
        static get propTypes () {
            return {"show": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"show": true};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"onClick": this._dir_click_toggle_evt_1}, dom.txt("Toggle", "r.0:0:0")), dom.el("p", "r.0:1", {"_s": this.$$ctx.state.show}, dom.txt("Visible", "r.0:1:0"))]));
        }
    }
    customElements.define("p-show-simple", PShowSimpleElement);
})();