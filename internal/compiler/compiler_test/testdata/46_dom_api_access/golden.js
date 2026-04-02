/*
--- BEGIN AST DUMP ---

<p>
  <RichText>
    "Body Width: "
    {{ state.width }}
  </RichText>
</p>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"width": 0};
        const state = makeReactive($$initialState, contextParam);
        pkc.onConnected(() => {
            requestAnimationFrame(() => {
                const el = document.querySelector("body");
                if (el) {
                    state.width = el.clientWidth;
                }
            });
        });
        return {"state": state, "$$initialState": $$initialState};
    }
    class DomApiElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"width": {"type": "number"}};
        }
        static get defaultProps () {
            return {"width": 0};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("p", "r.0", {}, dom.txt("Body Width: " + String(this.$$ctx.state.width ?? ""), "r.0:0"));
        }
    }
    customElements.define("dom-api", DomApiElement);
})();