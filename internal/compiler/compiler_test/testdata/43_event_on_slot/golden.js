/*
--- BEGIN AST DUMP ---

<div class="wrapper">
  <p>
    <RichText>
      "Last clicked item: "
      {{ state.lastClicked }}
    </RichText>
  </p>
  <slot [Events: p-event:item-click.="handleItemClick($event)" {P: handleItemClick($event)}] />
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
        const $$initialState = {"lastClicked": "none"};
        const state = makeReactive($$initialState, contextParam);
        function handleItemClick(event) {
            state.lastClicked = event.detail.id;
        }
        return {"state": state, "$$initialState": $$initialState, "handleItemClick": handleItemClick};
    }
    class SlotEventComponentElement extends PPElement {
        constructor () {
            super();
            this._dir_item_click_handleItemClick_evt_1 = (e) => {
                this.$$ctx.handleItemClick.call(this, e);
            };
        }
        static get propTypes () {
            return {"lastClicked": {"type": "string"}};
        }
        static get defaultProps () {
            return {"lastClicked": "none"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "wrapper"}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Last clicked item: " + String(this.$$ctx.state.lastClicked ?? ""), "r.0:0:0")), dom.el("slot", "r.0:1", {"pe:item-click": this._dir_item_click_handleItemClick_evt_1}, null)]));
        }
    }
    customElements.define("slot-event-component", SlotEventComponentElement);
})();