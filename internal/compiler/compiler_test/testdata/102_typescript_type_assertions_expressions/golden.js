/*
--- BEGIN AST DUMP ---

<div>
  <p>
    "Type assertion test"
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
        const $$initialState = {"items": []};
        const state = makeReactive($$initialState, contextParam);
        const getItems = () => {
            const slot = pkc.shadowRoot?.querySelector("slot");
            if (!slot) return [];
            const assignedElements = slot.assignedElements({"flatten": true});
            return assignedElements.filter((el) => {
                return el instanceof HTMLElement;
            });
        };
        const processItems = () => {
            const result = new Map();
            const items = getItems();
            items.forEach((item) => {
                const id = item.getAttribute("data-id");
                if (id) result.set(id, item);
            });
            return result;
        };
        const filterElements = (elements) => {
            return elements.filter((el) => {
                return el.hasAttribute("data-filter");
            });
        };
        pkc.onConnected(() => {
            const items = getItems();
            console.log("Found items:", items.length);
        });
        return {"state": state, "$$initialState": $$initialState, "getItems": getItems, "processItems": processItems, "filterElements": filterElements};
    }
    class TsTypeAssertionsElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"items": {"type": "array", "itemType": "string"}};
        }
        static get defaultProps () {
            return {"items": []};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.el("p", "r.0:0", {}, dom.txt("Type assertion test", "r.0:0:0")));
        }
        static get css () {
            return ":host{display: block}";
        }
    }
    customElements.define("ts-type-assertions", TsTypeAssertionsElement);
})();