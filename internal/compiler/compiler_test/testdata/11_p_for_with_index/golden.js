/*
--- BEGIN AST DUMP ---

<ul>
  <li [p-for: (index, color) in state.colors] [p-key: color]>
    <RichText>
      "\n            "
      {{ index }}
      ": "
      {{ color }}
      "\n        "
    </RichText>
  </li>
</ul>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"colors": ["Red", "Green", "Blue"]};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PForIndexElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"colors": {"type": "array", "itemType": "string"}};
        }
        static get defaultProps () {
            return {"colors": ["Red", "Green", "Blue"]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("ul", "r.0", {}, (Array.isArray(this.$$ctx.state.colors) ? this.$$ctx.state.colors : this.$$ctx.state.colors && typeof this.$$ctx.state.colors === "object" ? Object.entries(this.$$ctx.state.colors) : []).map((color, index) => {
                return dom.el("li", "r.0:0." + String(color ?? ""), {}, dom.txt("\n            " + String(index ?? "") + ": " + String(color ?? "") + "\n        ", "r.0:0." + String(color ?? "") + ":0"));
            }));
        }
    }
    customElements.define("p-for-index", PForIndexElement);
})();