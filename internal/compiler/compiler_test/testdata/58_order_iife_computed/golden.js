/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Config Value: "
      {{ state.configValue }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Computed: "
      {{ state.computed }}
    </RichText>
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
        const CONFIG = (() => {
            const base = 10;
            const multiplier = 2;
            return {"value": base * multiplier, "name": "computed"};
        })();
        const CONFIG_VALUE = CONFIG.value;
        const CONFIG_NAME = CONFIG.name;
        const COMPUTED_SUM = (() => {
            let sum = 0;
            sum += 10;
            sum += 20;
            sum += 30;
            return sum;
        })();
        const $$initialState = {"configValue": CONFIG_VALUE, "computed": COMPUTED_SUM};
        const state = makeReactive($$initialState, contextParam);
        function doubleConfig() {
            state.configValue = CONFIG.value * 2;
        }
        return {"state": state, "$$initialState": $$initialState, "doubleConfig": doubleConfig};
    }
    class IifeComputedElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"computed": {"type": "number"}, "configValue": {"type": "number"}};
        }
        static get defaultProps () {
            return {"computed": null, "configValue": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Config Value: " + String(this.$$ctx.state.configValue ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Computed: " + String(this.$$ctx.state.computed ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("iife-computed", IifeComputedElement);
})();