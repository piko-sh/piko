/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      {{ getMessage() }}
    </RichText>
  </p>
  <button [Events: p-on:click.="updateName" {P: updateName}]>
    "Update"
  </button>
</div>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
import { format } from "./utils.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"name": "World"};
        const state = makeReactive($$initialState, contextParam);
        function getMessage() {
            return format(state.name);
        }
        function updateName() {
            state.name = "Universe";
        }
        return {"state": state, "$$initialState": $$initialState, "getMessage": getMessage, "updateName": updateName};
    }
    class ImportsComponentElement extends PPElement {
        constructor () {
            super();
            this._dir_click_updateName_evt_1 = (e) => {
                this.$$ctx.updateName.call(this, e);
            };
        }
        static get propTypes () {
            return {"name": {"type": "string"}};
        }
        static get defaultProps () {
            return {"name": "World"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt(String(this.$$ctx.getMessage() ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_updateName_evt_1}, dom.txt("Update", "r.0:1:0"))]));
        }
    }
    customElements.define("imports-component", ImportsComponentElement);
})();