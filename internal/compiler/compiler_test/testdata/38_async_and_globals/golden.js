/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Body class: "
      {{ state.bodyClass }}
    </RichText>
  </p>
  <button [Events: p-on:click.="checkBodyClass" {P: checkBodyClass}]>
    "Check Body Class"
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
        const $$initialState = {"bodyClass": "unknown"};
        const state = makeReactive($$initialState, contextParam);
        async function checkBodyClass() {
            const body = document.querySelector("body");
            await new Promise((resolve) => {
                return setTimeout(resolve, 50);
            });
            state.bodyClass = body.className || "none";
        }
        return {"state": state, "$$initialState": $$initialState, "checkBodyClass": checkBodyClass};
    }
    class AsyncGlobalsElement extends PPElement {
        constructor () {
            super();
            this._dir_click_checkBodyClass_evt_1 = (e) => {
                this.$$ctx.checkBodyClass.call(this, e);
            };
        }
        static get propTypes () {
            return {"bodyClass": {"type": "string"}};
        }
        static get defaultProps () {
            return {"bodyClass": "unknown"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Body class: " + String(this.$$ctx.state.bodyClass ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_checkBodyClass_evt_1}, dom.txt("Check Body Class", "r.0:1:0"))]));
        }
    }
    customElements.define("async-globals", AsyncGlobalsElement);
})();