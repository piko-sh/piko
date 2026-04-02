/*
--- BEGIN AST DUMP ---

<div>
  <div [p-class: ["base-class", state.dynamicClass, {"conditional": state.isConditional}]]>
    "Sample Text"
  </div>
  <button [Events: p-on:click.="changeClass" {P: changeClass}]>
    "Change Class"
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
        const $$initialState = {"dynamicClass": "theme-dark", "isConditional": true};
        const state = makeReactive($$initialState, contextParam);
        function changeClass() {
            state.dynamicClass = "theme-light";
            state.isConditional = false;
        }
        return {"state": state, "$$initialState": $$initialState, "changeClass": changeClass};
    }
    class PClassArrayElement extends PPElement {
        constructor () {
            super();
            this._dir_click_changeClass_evt_1 = (e) => {
                this.$$ctx.changeClass.call(this, e);
            };
        }
        static get propTypes () {
            return {"dynamicClass": {"type": "string"}, "isConditional": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"dynamicClass": "theme-dark", "isConditional": true};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("div", "r.0:0", {"_class": ["base-class", this.$$ctx.state.dynamicClass, {"conditional": this.$$ctx.state.isConditional}]}, dom.txt(" Sample Text ", "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_changeClass_evt_1}, dom.txt("Change Class", "r.0:1:0"))]));
        }
    }
    customElements.define("p-class-array", PClassArrayElement);
})();