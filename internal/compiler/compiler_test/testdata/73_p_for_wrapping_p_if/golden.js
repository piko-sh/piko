/*
--- BEGIN AST DUMP ---

<div>
  <ul>
    <li [p-for: item in state.items] [p-key: item.id]>
      <span class="active" [p-if: item.active]>
        <RichText>
          {{ item.name }}
          " (Active)"
        </RichText>
      </span>
      <span class="inactive" [p-else]>
        <RichText>
          {{ item.name }}
          " (Inactive)"
        </RichText>
      </span>
    </li>
  </ul>
  <p id="active-count">
    <RichText>
      "Active: "
      {{ state.activeCount }}
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
        const $$initialState = {"items": [{"id": 1, "name": "Alice", "active": true}, {"id": 2, "name": "Bob", "active": false}, {"id": 3, "name": "Carol", "active": true}, {"id": 4, "name": "Dave", "active": false}], "activeCount": 2};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PForWrappingIfElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"activeCount": {"type": "number"}, "items": {"type": "array", "itemType": "object"}};
        }
        static get defaultProps () {
            return {"activeCount": 2, "items": [{"id": 1, "name": "Alice", "active": true}, {"id": 2, "name": "Bob", "active": false}, {"id": 3, "name": "Carol", "active": true}, {"id": 4, "name": "Dave", "active": false}]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("ul", "r.0:0", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:0:0." + String(item.id ?? ""), {}, item.active ? dom.el("span", "r.0:0:0." + String(item.id ?? "") + ":0", {"class": "active"}, dom.txt(String(item.name ?? "") + " (Active)", "r.0:0:0." + String(item.id ?? "") + ":0:0")) : dom.el("span", "r.0:0:0." + String(item.id ?? "") + ":1", {"class": "inactive"}, dom.txt(String(item.name ?? "") + " (Inactive)", "r.0:0:0." + String(item.id ?? "") + ":1:0")));
            })), dom.el("p", "r.0:1", {"id": "active-count"}, dom.txt("Active: " + String(this.$$ctx.state.activeCount ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("p-for-wrapping-if", PForWrappingIfElement);
})();