/*
--- BEGIN AST DUMP ---

<div>
  <button id="toggle" [Events: p-on:click.="toggle" {P: toggle}]>
    "Toggle List"
  </button>
  <div [p-if: state.showList]>
    <ul>
      <li [p-for: item in state.items] [p-key: item.id]>
        <RichText>
          {{ item.name }}
        </RichText>
      </li>
    </ul>
  </div>
  <p [p-else]>
    "List is hidden"
  </p>
  <p id="count">
    <RichText>
      "Items: "
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
        const $$initialState = {"showList": false, "items": [{"id": 1, "name": "First"}, {"id": 2, "name": "Second"}, {"id": 3, "name": "Third"}]};
        const state = makeReactive($$initialState, contextParam);
        function toggle() {
            state.showList = !state.showList;
        }
        return {"state": state, "$$initialState": $$initialState, "toggle": toggle};
    }
    class PIfWrappingForElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggle_evt_1 = (e) => {
                this.$$ctx.toggle.call(this, e);
            };
        }
        static get propTypes () {
            return {"items": {"type": "array", "itemType": "object"}, "showList": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"items": [{"id": 1, "name": "First"}, {"id": 2, "name": "Second"}, {"id": 3, "name": "Third"}], "showList": false};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "toggle", "onClick": this._dir_click_toggle_evt_1}, dom.txt("Toggle List", "r.0:0:0")), this.$$ctx.state.showList ? dom.el("div", "r.0:1", {}, dom.el("ul", "r.0:1:0", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:1:0:0." + String(item.id ?? ""), {}, dom.txt(String(item.name ?? ""), "r.0:1:0:0." + String(item.id ?? "") + ":0"));
            }))) : dom.el("p", "r.0:2", {}, dom.txt("List is hidden", "r.0:2:0")), dom.el("p", "r.0:3", {"id": "count"}, dom.txt("Items: " + String(this.$$ctx.state.items.length ?? ""), "r.0:3:0"))]));
        }
    }
    customElements.define("p-if-wrapping-for", PIfWrappingForElement);
})();