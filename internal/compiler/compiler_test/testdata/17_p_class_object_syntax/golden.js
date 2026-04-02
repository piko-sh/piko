/*
--- BEGIN AST DUMP ---

<div>
  <div class="static" [p-class: {"active": state.isActive, "text-danger": state.hasError}]>
    "Sample Text"
  </div>
  <button [Events: p-on:click.="toggleActive" {P: toggleActive}]>
    "Toggle Active"
  </button>
  <button [Events: p-on:click.="toggleError" {P: toggleError}]>
    "Toggle Error"
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
        const $$initialState = {"isActive": true, "hasError": false};
        const state = makeReactive($$initialState, contextParam);
        function toggleActive() {
            state.isActive = !state.isActive;
        }
        function toggleError() {
            state.hasError = !state.hasError;
        }
        return {"state": state, "$$initialState": $$initialState, "toggleActive": toggleActive, "toggleError": toggleError};
    }
    class PClassObjectElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggleActive_evt_1 = (e) => {
                this.$$ctx.toggleActive.call(this, e);
            };
            this._dir_click_toggleError_evt_2 = (e) => {
                this.$$ctx.toggleError.call(this, e);
            };
        }
        static get propTypes () {
            return {"hasError": {"type": "boolean"}, "isActive": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"hasError": false, "isActive": true};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("div", "r.0:0", {"_class": {"active": this.$$ctx.state.isActive, "text-danger": this.$$ctx.state.hasError}, "class": "static"}, dom.txt(" Sample Text ", "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_toggleActive_evt_1}, dom.txt("Toggle Active", "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_toggleError_evt_2}, dom.txt("Toggle Error", "r.0:2:0"))]));
        }
    }
    customElements.define("p-class-object", PClassObjectElement);
})();