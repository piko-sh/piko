/*
--- BEGIN AST DUMP ---

<div class="internal-div">
  "Internal Div"
</div>
<button [Events: p-on:click.="dispatchCustomEvent" {P: dispatchCustomEvent}]>
  "Dispatch Event"
</button>
<button [Events: p-on:click.="queryInternalDiv" {P: queryInternalDiv}]>
  "Query Div"
</button>
<p>
  <RichText>
    "Message from queried div: "
    {{ state.message }}
  </RichText>
</p>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"message": "none"};
        const state = makeReactive($$initialState, contextParam);
        function dispatchCustomEvent() {
            this.dispatchEvent(new CustomEvent("my-event", {"detail": {"message": "Hello from component"}, "bubbles": true, "composed": true}));
        }
        function queryInternalDiv() {
            const internalDiv = this.shadowRoot.querySelector(".internal-div");
            state.message = internalDiv.textContent;
        }
        return {"state": state, "$$initialState": $$initialState, "dispatchCustomEvent": dispatchCustomEvent, "queryInternalDiv": queryInternalDiv};
    }
    class ThisContextElement extends PPElement {
        constructor () {
            super();
            this._dir_click_dispatchCustomEvent_evt_1 = (e) => {
                this.$$ctx.dispatchCustomEvent.call(this, e);
            };
            this._dir_click_queryInternalDiv_evt_2 = (e) => {
                this.$$ctx.queryInternalDiv.call(this, e);
            };
        }
        static get propTypes () {
            return {"message": {"type": "string"}};
        }
        static get defaultProps () {
            return {"message": "none"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.frag("root_fragment", [dom.el("div", "r.0", {"class": "internal-div"}, dom.txt("Internal Div", "r.0:0")), dom.el("button", "r.1", {"onClick": this._dir_click_dispatchCustomEvent_evt_1}, dom.txt("Dispatch Event", "r.1:0")), dom.el("button", "r.2", {"onClick": this._dir_click_queryInternalDiv_evt_2}, dom.txt("Query Div", "r.2:0")), dom.el("p", "r.3", {}, dom.txt("Message from queried div: " + String(this.$$ctx.state.message ?? ""), "r.3:0"))]);
        }
    }
    customElements.define("this-context", ThisContextElement);
})();