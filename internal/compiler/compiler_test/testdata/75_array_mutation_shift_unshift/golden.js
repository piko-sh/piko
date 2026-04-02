/*
--- BEGIN AST DUMP ---

<div>
  <button id="shift" [Events: p-on:click.="shiftItem" {P: shiftItem}]>
    "Shift Item"
  </button>
  <button id="unshift" [Events: p-on:click.="unshiftItem" {P: unshiftItem}]>
    "Unshift Item"
  </button>
  <ul>
    <li [p-for: item in state.items] [p-key: item]>
      <RichText>
        {{ item }}
      </RichText>
    </li>
  </ul>
  <p id="count">
    <RichText>
      "Count: "
      {{ state.items.length }}
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
        const $$initialState = {"items": ["A", "B", "C"], "counter": 0};
        const state = makeReactive($$initialState, contextParam);
        function shiftItem() {
            state.items.shift();
        }
        function unshiftItem() {
            state.items.unshift("New-" + state.counter++);
        }
        return {"state": state, "$$initialState": $$initialState, "shiftItem": shiftItem, "unshiftItem": unshiftItem};
    }
    class ArrayMutationShiftUnshiftElement extends PPElement {
        constructor () {
            super();
            this._dir_click_shiftItem_evt_1 = (e) => {
                this.$$ctx.shiftItem.call(this, e);
            };
            this._dir_click_unshiftItem_evt_2 = (e) => {
                this.$$ctx.unshiftItem.call(this, e);
            };
        }
        static get propTypes () {
            return {"counter": {"type": "number"}, "items": {"type": "array", "itemType": "string"}};
        }
        static get defaultProps () {
            return {"counter": 0, "items": ["A", "B", "C"]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "shift", "onClick": this._dir_click_shiftItem_evt_1}, dom.txt("Shift Item", "r.0:0:0")), dom.el("button", "r.0:1", {"id": "unshift", "onClick": this._dir_click_unshiftItem_evt_2}, dom.txt("Unshift Item", "r.0:1:0")), dom.el("ul", "r.0:2", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:2:0." + String(item ?? ""), {}, dom.txt(String(item ?? ""), "r.0:2:0." + String(item ?? "") + ":0"));
            })), dom.el("p", "r.0:3", {"id": "count"}, dom.txt("Count: " + String(this.$$ctx.state.items.length ?? ""), "r.0:3:0"))]));
        }
    }
    customElements.define("array-mutation-shift-unshift", ArrayMutationShiftUnshiftElement);
})();