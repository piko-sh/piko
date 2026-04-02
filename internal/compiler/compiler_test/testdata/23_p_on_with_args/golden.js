/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      {{ state.message }}
    </RichText>
  </p>
  <button [Events: p-on:click.="setMessage('Hello', state.name, $event)" {P: setMessage("Hello", state.name, $event)}]>
    "Set Message"
  </button>
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
        const $$initialState = {"message": "Initial", "name": "World"};
        const state = makeReactive($$initialState, contextParam);
        function setMessage(greeting, name, event) {
            console.log("Event target tagName:", event.target.tagName);
            state.message = `${greeting}, ${name}!`;
        }
        return {"state": state, "$$initialState": $$initialState, "setMessage": setMessage};
    }
    class POnArgsElement extends PPElement {
        constructor () {
            super();
            this._dir_click_setMessage_evt_1 = (e) => {
                this.$$ctx.setMessage.call(this, "Hello", this.$$ctx.state.name, e);
            };
        }
        static get propTypes () {
            return {"message": {"type": "string"}, "name": {"type": "string"}};
        }
        static get defaultProps () {
            return {"message": "Initial", "name": "World"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt(String(this.$$ctx.state.message ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_setMessage_evt_1}, dom.txt("Set Message", "r.0:1:0"))]));
        }
    }
    customElements.define("p-on-args", POnArgsElement);
})();