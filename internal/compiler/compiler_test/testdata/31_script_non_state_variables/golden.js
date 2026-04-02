/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      {{ getGreeting() }}
    </RichText>
  </p>
  <button [Events: p-on:click.="forceRerender" {P: forceRerender}]>
    "Rerender"
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
        const GREETING = "Hello";
        let callCounter = 0;
        const $$initialState = {"name": "Component", "rerenderTrigger": 0};
        const state = makeReactive($$initialState, contextParam);
        function getGreeting() {
            callCounter++;
            return `${GREETING}, ${state.name}! Call count: ${callCounter}`;
        }
        function forceRerender() {
            state.rerenderTrigger++;
        }
        return {"state": state, "$$initialState": $$initialState, "getGreeting": getGreeting, "forceRerender": forceRerender};
    }
    class NonStateVarsElement extends PPElement {
        constructor () {
            super();
            this._dir_click_forceRerender_evt_1 = (e) => {
                this.$$ctx.forceRerender.call(this, e);
            };
        }
        static get propTypes () {
            return {"name": {"type": "string"}, "rerenderTrigger": {"type": "number"}};
        }
        static get defaultProps () {
            return {"name": "Component", "rerenderTrigger": 0};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt(String(this.$$ctx.getGreeting() ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_forceRerender_evt_1}, dom.txt("Rerender", "r.0:1:0"))]));
        }
    }
    customElements.define("non-state-vars", NonStateVarsElement);
})();