/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Item count: "
      {{ state.count }}
    </RichText>
  </p>
  <slot />
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
        const $$initialState = {"count": 0};
        const state = makeReactive($$initialState, contextParam);
        pkc.onConnected(() => {
            pkc.attachSlotListener("", (elements) => {
                state.count = elements.length;
            });
        });
        return {"state": state, "$$initialState": $$initialState};
    }
    class SlotListenerComponentElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"count": {"type": "number"}};
        }
        static get defaultProps () {
            return {"count": 0};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Item count: " + String(this.$$ctx.state.count ?? ""), "r.0:0:0")), dom.el("slot", "r.0:1", {}, null)]));
        }
    }
    customElements.define("slot-listener-component", SlotListenerComponentElement);
})();