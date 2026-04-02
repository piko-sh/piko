/*
--- BEGIN AST DUMP ---

<div>
  <button [Events: p-on:click.="nextStatus" {P: nextStatus}]>
    "Next"
  </button>
  <p>
    <RichText>
      "Status: "
      {{ state.status }}
    </RichText>
  </p>
  <p class="content" [p-if: (state.status == "a")]>
    "Block A"
  </p>
  <p class="content" [p-else-if: (state.status == "b")]>
    "Block B"
  </p>
  <p class="content" [p-else]>
    "Block C"
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
        const $$initialState = {"status": "a"};
        const state = makeReactive($$initialState, contextParam);
        const statuses = ["a", "b", "c"];
        function nextStatus() {
            const currentIndex = statuses.indexOf(state.status);
            state.status = statuses[(currentIndex + 1) % statuses.length];
        }
        return {"state": state, "$$initialState": $$initialState, "nextStatus": nextStatus};
    }
    class PIfElseChainElement extends PPElement {
        constructor () {
            super();
            this._dir_click_nextStatus_evt_1 = (e) => {
                this.$$ctx.nextStatus.call(this, e);
            };
        }
        static get propTypes () {
            return {"status": {"type": "string"}};
        }
        static get defaultProps () {
            return {"status": "a"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"onClick": this._dir_click_nextStatus_evt_1}, dom.txt("Next", "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Status: " + String(this.$$ctx.state.status ?? ""), "r.0:1:0")), this.$$ctx.state.status === "a" ? dom.el("p", "r.0:2", {"class": "content"}, dom.txt("Block A", "r.0:2:0")) : this.$$ctx.state.status === "b" ? dom.el("p", "r.0:3", {"class": "content"}, dom.txt("Block B", "r.0:3:0")) : dom.el("p", "r.0:4", {"class": "content"}, dom.txt("Block C", "r.0:4:0"))]));
        }
    }
    customElements.define("p-if-else-chain", PIfElseChainElement);
})();