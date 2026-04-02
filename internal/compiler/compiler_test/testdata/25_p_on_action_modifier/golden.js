/*
--- BEGIN AST DUMP ---

<button [Events: p-on:click.="doSomethingOnServer(state.id, 'static')" {P: doSomethingOnServer(state.id, "static")}]>
  "Run Action"
</button>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"id": 42};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class POnActionElement extends PPElement {
        constructor () {
            super();
            this._dir_click_doSomethingOnServer_evt_1 = (e) => {
                e.preventDefault();
                this.$$ctx.doSomethingOnServer.call(this, this.$$ctx.state.id, "static");
            };
        }
        static get propTypes () {
            return {"id": {"type": "number"}};
        }
        static get defaultProps () {
            return {"id": 42};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("button", "r.0", {"onClick": this._dir_click_doSomethingOnServer_evt_1}, dom.txt(" Run Action ", "r.0:0"));
        }
    }
    customElements.define("p-on-action", POnActionElement);
})();