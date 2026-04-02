/*
--- BEGIN AST DUMP ---

<div>
  <p class="status">
    <RichText>
      "Status: "
      {{ state.status }}
    </RichText>
  </p>
  <p class="attr-result">
    <RichText>
      "Attr: "
      {{ state.attrResult }}
    </RichText>
  </p>
  <slot name="items" />
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
        const $$initialState = {"itemRefs": [], "status": "waiting", "attrResult": ""};
        const state = makeReactive($$initialState, contextParam);
        pkc.onConnected(() => {
            pkc.attachSlotListener("items", (assignedEls) => {
                state.itemRefs = assignedEls;
                try {
                    const attrs = [];
                    state.itemRefs.forEach((el) => {
                        const dataId = el.getAttribute("data-id");
                        if (dataId) {
                            attrs.push(dataId);
                        }
                        el.classList.add("processed");
                    });
                    state.attrResult = attrs.join(",");
                    state.status = "success";
                    console.log("DOM_METHODS_SUCCESS");
                } catch(err) {
                    state.status = "error: " + err.message;
                    console.error("DOM_METHODS_FAILED", err.message);
                }
            });
        });
        return {"state": state, "$$initialState": $$initialState};
    }
    class ReactiveDomTestElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"attrResult": {"type": "string"}, "itemRefs": {"type": "array", "itemType": "element"}, "status": {"type": "string"}};
        }
        static get defaultProps () {
            return {"attrResult": "", "itemRefs": [], "status": "waiting"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {"class": "status"}, dom.txt("Status: " + String(this.$$ctx.state.status ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {"class": "attr-result"}, dom.txt("Attr: " + String(this.$$ctx.state.attrResult ?? ""), "r.0:1:0")), dom.el("slot", "r.0:2", {"name": "items"}, null)]));
        }
        static get css () {
            return ":host{display: block}.processed{color: green}";
        }
    }
    customElements.define("reactive-dom-test", ReactiveDomTestElement);
})();