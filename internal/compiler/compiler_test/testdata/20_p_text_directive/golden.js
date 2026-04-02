/*
--- BEGIN AST DUMP ---

<div>
  <p [p-text: state.message]>
    "This will be replaced."
  </p>
  <button [Events: p-on:click.="updateMessage" {P: updateMessage}]>
    "Update"
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
        const $$initialState = {"message": "Initial message"};
        const state = makeReactive($$initialState, contextParam);
        function updateMessage() {
            state.message = "Updated message with <strong>HTML</strong>";
        }
        return {"state": state, "$$initialState": $$initialState, "updateMessage": updateMessage};
    }
    class PTextDirElement extends PPElement {
        constructor () {
            super();
            this._dir_click_updateMessage_evt_1 = (e) => {
                this.$$ctx.updateMessage.call(this, e);
            };
        }
        static get propTypes () {
            return {"message": {"type": "string"}};
        }
        static get defaultProps () {
            return {"message": "Initial message"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, [dom.txt(String(this.$$ctx.state.message), "r.0:0:txt")]), dom.el("button", "r.0:1", {"onClick": this._dir_click_updateMessage_evt_1}, dom.txt("Update", "r.0:1:0"))]));
        }
    }
    customElements.define("p-text-dir", PTextDirElement);
})();