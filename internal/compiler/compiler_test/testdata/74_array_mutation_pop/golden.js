/*
--- BEGIN AST DUMP ---

<div>
  <button id="pop" [Events: p-on:click.="popItem" {P: popItem}]>
    "Pop Item"
  </button>
  <button id="push" [Events: p-on:click.="pushItem" {P: pushItem}]>
    "Push Item"
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
        const $$initialState = {"items": ["A", "B", "C", "D"], "counter": 0};
        const state = makeReactive($$initialState, contextParam);
        function popItem() {
            state.items.pop();
        }
        function pushItem() {
            state.items.push("New-" + state.counter++);
        }
        return {"state": state, "$$initialState": $$initialState, "popItem": popItem, "pushItem": pushItem};
    }
    class ArrayMutationPopElement extends PPElement {
        constructor () {
            super();
            this._dir_click_popItem_evt_1 = (e) => {
                this.$$ctx.popItem.call(this, e);
            };
            this._dir_click_pushItem_evt_2 = (e) => {
                this.$$ctx.pushItem.call(this, e);
            };
        }
        static get propTypes () {
            return {"counter": {"type": "number"}, "items": {"type": "array", "itemType": "string"}};
        }
        static get defaultProps () {
            return {"counter": 0, "items": ["A", "B", "C", "D"]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "pop", "onClick": this._dir_click_popItem_evt_1}, dom.txt("Pop Item", "r.0:0:0")), dom.el("button", "r.0:1", {"id": "push", "onClick": this._dir_click_pushItem_evt_2}, dom.txt("Push Item", "r.0:1:0")), dom.el("ul", "r.0:2", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:2:0." + String(item ?? ""), {}, dom.txt(String(item ?? ""), "r.0:2:0." + String(item ?? "") + ":0"));
            })), dom.el("p", "r.0:3", {"id": "count"}, dom.txt("Count: " + String(this.$$ctx.state.items.length ?? ""), "r.0:3:0"))]));
        }
    }
    customElements.define("array-mutation-pop", ArrayMutationPopElement);
})();