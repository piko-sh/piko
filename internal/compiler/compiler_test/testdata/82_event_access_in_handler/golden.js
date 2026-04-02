/*
--- BEGIN AST DUMP ---

<div>
  <input id="input" type="text" [Events: p-on:input.="onInput($event)" {P: onInput($event)}] />
  <p id="value">
    <RichText>
      "Value: "
      {{ state.inputValue }}
    </RichText>
  </p>
  <p id="type">
    <RichText>
      "Event type: "
      {{ state.lastEventType }}
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
        const $$initialState = {"inputValue": "", "lastEventType": "none"};
        const state = makeReactive($$initialState, contextParam);
        function onInput(event) {
            state.inputValue = event.target.value;
            state.lastEventType = event.type;
        }
        return {"state": state, "$$initialState": $$initialState, "onInput": onInput};
    }
    class EventAccessInHandlerElement extends PPElement {
        constructor () {
            super();
            this._dir_input_onInput_evt_1 = (e) => {
                this.$$ctx.onInput.call(this, e);
            };
        }
        static get propTypes () {
            return {"inputValue": {"type": "string"}, "lastEventType": {"type": "string"}};
        }
        static get defaultProps () {
            return {"inputValue": "", "lastEventType": "none"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("input", "r.0:0", {"id": "input", "onInput": this._dir_input_onInput_evt_1, "type": "text"}, null), dom.el("p", "r.0:1", {"id": "value"}, dom.txt("Value: " + String(this.$$ctx.state.inputValue ?? ""), "r.0:1:0")), dom.el("p", "r.0:2", {"id": "type"}, dom.txt("Event type: " + String(this.$$ctx.state.lastEventType ?? ""), "r.0:2:0"))]));
        }
    }
    customElements.define("event-access-in-handler", EventAccessInHandlerElement);
})();