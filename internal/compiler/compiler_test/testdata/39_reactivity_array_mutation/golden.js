/*
--- BEGIN AST DUMP ---

<div>
  <button id="add" [Events: p-on:click.="addItem" {P: addItem}]>
    "Add"
  </button>
  <button id="remove" [Events: p-on:click.="removeItem" {P: removeItem}]>
    "Remove"
  </button>
  <ul>
    <li [p-for: item in state.items] [p-key: item]>
      <RichText>
        {{ item }}
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
        const $$initialState = {"items": ["A", "B"]};
        const state = makeReactive($$initialState, contextParam);
        function addItem() {
            state.items.push(`Item-${state.items.length}`);
        }
        function removeItem() {
            if (state.items.length > 0) {
                state.items.splice(0, 1);
            }
        }
        return {"state": state, "$$initialState": $$initialState, "addItem": addItem, "removeItem": removeItem};
    }
    class ArrayMutationElement extends PPElement {
        constructor () {
            super();
            this._dir_click_addItem_evt_1 = (e) => {
                this.$$ctx.addItem.call(this, e);
            };
            this._dir_click_removeItem_evt_2 = (e) => {
                this.$$ctx.removeItem.call(this, e);
            };
        }
        static get propTypes () {
            return {"items": {"type": "array", "itemType": "string"}};
        }
        static get defaultProps () {
            return {"items": ["A", "B"]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "add", "onClick": this._dir_click_addItem_evt_1}, dom.txt("Add", "r.0:0:0")), dom.el("button", "r.0:1", {"id": "remove", "onClick": this._dir_click_removeItem_evt_2}, dom.txt("Remove", "r.0:1:0")), dom.el("ul", "r.0:2", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:2:0." + String(item ?? ""), {}, dom.txt(String(item ?? ""), "r.0:2:0." + String(item ?? "") + ":0"));
            }))]));
        }
    }
    customElements.define("array-mutation", ArrayMutationElement);
})();