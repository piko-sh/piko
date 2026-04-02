/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Selected ID: "
      {{ state.selectedId }}
    </RichText>
  </p>
  <ul>
    <li [p-for: item in state.items] [p-key: item.id]>
      <span>
        <RichText>
          {{ item.text }}
        </RichText>
      </span>
      <button [Events: p-on:click.="selectItem(item.id)" {P: selectItem(item.id)}]>
        "Select"
      </button>
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
        const $$initialState = {"items": [{"id": 1, "text": "A"}, {"id": 2, "text": "B"}], "selectedId": null};
        const state = makeReactive($$initialState, contextParam);
        function selectItem(id) {
            state.selectedId = id;
        }
        return {"state": state, "$$initialState": $$initialState, "selectItem": selectItem};
    }
    class POnLoopElement extends PPElement {
        constructor () {
            super();
            this._hof_click_selectItem_evt_1 = (item) => {
                return (e) => {
                    this.$$ctx.selectItem.call(this, item.id);
                };
            };
        }
        static get propTypes () {
            return {"items": {"type": "array", "itemType": "object"}, "selectedId": {"type": "any"}};
        }
        static get defaultProps () {
            return {"items": [{"id": 1, "text": "A"}, {"id": 2, "text": "B"}], "selectedId": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Selected ID: " + String(this.$$ctx.state.selectedId ?? ""), "r.0:0:0")), dom.el("ul", "r.0:1", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:1:0." + String(item.id ?? ""), {}, dom.frag(`${"r.0:1:0." + String(item.id ?? "")}_f`, [dom.el("span", "r.0:1:0." + String(item.id ?? "") + ":0", {}, dom.txt(String(item.text ?? ""), "r.0:1:0." + String(item.id ?? "") + ":0:0")), dom.el("button", "r.0:1:0." + String(item.id ?? "") + ":1", {"onClick": this._hof_click_selectItem_evt_1(item)}, dom.txt("Select", "r.0:1:0." + String(item.id ?? "") + ":1:0"))]));
            }))]));
        }
    }
    customElements.define("p-on-loop", POnLoopElement);
})();