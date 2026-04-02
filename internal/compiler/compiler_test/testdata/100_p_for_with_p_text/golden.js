/*
--- BEGIN AST DUMP ---

<select>
  <option :value="opt.value" {P: opt.value} [p-for: (i, opt) in state.options] [p-text: opt.label] />
</select>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"options": []};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PForPTextTestElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"options": {"type": "valuestring"}};
        }
        static get defaultProps () {
            return {"options": []};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("select", "r.0", {}, (Array.isArray(this.$$ctx.state.options) ? this.$$ctx.state.options : this.$$ctx.state.options && typeof this.$$ctx.state.options === "object" ? Object.entries(this.$$ctx.state.options) : []).map((opt, i) => {
                return dom.el("option", "r.0:0." + String(i ?? ""), {"value": (opt.value)}, [dom.txt(String(opt.label), "r.0:0." + String(i ?? "") + ":txt")]);
            }));
        }
    }
    customElements.define("p-for-p-text-test", PForPTextTestElement);
})();