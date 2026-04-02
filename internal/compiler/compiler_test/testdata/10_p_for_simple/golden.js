/*
--- BEGIN AST DUMP ---

<div>
  <button [Events: p-on:click.="addItem" {P: addItem}]>
    "Add Item"
  </button>
  <ul>
    <li [p-for: item in state.items] [p-key: item.id]>
      <RichText>
        "\n                "
        {{ item.text }}
        "\n            "
      </RichText>
    </li>
  </ul>
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
        const $$initialState = {"items": [{"id": 1, "text": "First"}, {"id": 2, "text": "Second"}], "nextId": 3};
        const state = makeReactive($$initialState, contextParam);
        function addItem() {
            state.items.push({"id": state.nextId, "text": `Item ${state.nextId}`});
            state.nextId++;
        }
        return {"state": state, "$$initialState": $$initialState, "addItem": addItem};
    }
    class PForSimpleElement extends PPElement {
        constructor () {
            super();
            this._dir_click_addItem_evt_1 = (e) => {
                this.$$ctx.addItem.call(this, e);
            };
        }
        static get propTypes () {
            return {"items": {"type": "array", "itemType": "object"}, "nextId": {"type": "number"}};
        }
        static get defaultProps () {
            return {"items": [{"id": 1, "text": "First"}, {"id": 2, "text": "Second"}], "nextId": 3};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"onClick": this._dir_click_addItem_evt_1}, dom.txt("Add Item", "r.0:0:0")), dom.el("ul", "r.0:1", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:1:0." + String(item.id ?? ""), {}, dom.txt("\n                " + String(item.text ?? "") + "\n            ", "r.0:1:0." + String(item.id ?? "") + ":0"));
            }))]));
        }
    }
    customElements.define("p-for-simple", PForSimpleElement);
})();