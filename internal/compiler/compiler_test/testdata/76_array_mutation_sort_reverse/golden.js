/*
--- BEGIN AST DUMP ---

<div>
  <button id="sort" [Events: p-on:click.="sortItems" {P: sortItems}]>
    "Sort"
  </button>
  <button id="reverse" [Events: p-on:click.="reverseItems" {P: reverseItems}]>
    "Reverse"
  </button>
  <button id="reset" [Events: p-on:click.="resetItems" {P: resetItems}]>
    "Reset"
  </button>
  <ul>
    <li [p-for: item in state.items] [p-key: item]>
      <RichText>
        {{ item }}
      </RichText>
    </li>
  </ul>
  <p id="first">
    <RichText>
      "First: "
      {{ state.items[0] }}
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
        const $$initialState = {"items": ["C", "A", "B"]};
        const state = makeReactive($$initialState, contextParam);
        function sortItems() {
            state.items.sort();
        }
        function reverseItems() {
            state.items.reverse();
        }
        function resetItems() {
            state.items = ["C", "A", "B"];
        }
        return {"state": state, "$$initialState": $$initialState, "sortItems": sortItems, "reverseItems": reverseItems, "resetItems": resetItems};
    }
    class ArrayMutationSortReverseElement extends PPElement {
        constructor () {
            super();
            this._dir_click_sortItems_evt_1 = (e) => {
                this.$$ctx.sortItems.call(this, e);
            };
            this._dir_click_reverseItems_evt_2 = (e) => {
                this.$$ctx.reverseItems.call(this, e);
            };
            this._dir_click_resetItems_evt_3 = (e) => {
                this.$$ctx.resetItems.call(this, e);
            };
        }
        static get propTypes () {
            return {"items": {"type": "array", "itemType": "string"}};
        }
        static get defaultProps () {
            return {"items": ["C", "A", "B"]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "sort", "onClick": this._dir_click_sortItems_evt_1}, dom.txt("Sort", "r.0:0:0")), dom.el("button", "r.0:1", {"id": "reverse", "onClick": this._dir_click_reverseItems_evt_2}, dom.txt("Reverse", "r.0:1:0")), dom.el("button", "r.0:2", {"id": "reset", "onClick": this._dir_click_resetItems_evt_3}, dom.txt("Reset", "r.0:2:0")), dom.el("ul", "r.0:3", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:3:0." + String(item ?? ""), {}, dom.txt(String(item ?? ""), "r.0:3:0." + String(item ?? "") + ":0"));
            })), dom.el("p", "r.0:4", {"id": "first"}, dom.txt("First: " + String(this.$$ctx.state.items[0] ?? ""), "r.0:4:0"))]));
        }
    }
    customElements.define("array-mutation-sort-reverse", ArrayMutationSortReverseElement);
})();