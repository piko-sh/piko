/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      {{ state.status }}
    </RichText>
  </p>
  <button [Events: p-on:click.="fetchData" {P: fetchData}]>
    "Fetch"
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
        const $$initialState = {"status": "Idle"};
        const state = makeReactive($$initialState, contextParam);
        async function fetchData() {
            state.status = "Loading...";
            await new Promise((resolve) => {
                return setTimeout(resolve, 500);
            });
            state.status = "Loaded!";
        }
        return {"state": state, "$$initialState": $$initialState, "fetchData": fetchData};
    }
    class AsyncMethodElement extends PPElement {
        constructor () {
            super();
            this._dir_click_fetchData_evt_1 = (e) => {
                this.$$ctx.fetchData.call(this, e);
            };
        }
        static get propTypes () {
            return {"status": {"type": "string"}};
        }
        static get defaultProps () {
            return {"status": "Idle"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt(String(this.$$ctx.state.status ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_fetchData_evt_1}, dom.txt("Fetch", "r.0:1:0"))]));
        }
    }
    customElements.define("async-method", AsyncMethodElement);
})();