/*
--- BEGIN AST DUMP ---

<div>
  <p :id="state.elementId" {P: state.elementId}>
    "Element ID:"
    <span [p-text: state.elementId] />
  </p>
  <p>
    "Counter:"
    <span [p-text: state.counter] />
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
        const $$initialState = {"elementId": "", "counter": 0};
        const state = makeReactive($$initialState, contextParam);
        const updateCounter = () => {
            state.counter = state.counter + 1;
        };
        const testId = "test-id";
        state.elementId = state.elementId || `element-${testId}`;
        const helperValue = "helper";
        function increment() {
            updateCounter();
        }
        return {"state": state, "$$initialState": $$initialState, "updateCounter": updateCounter, "increment": increment};
    }
    class StateOrderingTestElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"counter": {"type": "number"}, "elementId": {"type": "string"}};
        }
        static get defaultProps () {
            return {"counter": 0, "elementId": ""};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {"id": (this.$$ctx.state.elementId)}, dom.frag("r.0:0_f", [dom.txt("Element ID: ", "r.0:0:0"), dom.el("span", "r.0:0:1", {}, [dom.txt(String(this.$$ctx.state.elementId), "r.0:0:1:txt")])])), dom.el("p", "r.0:1", {}, dom.frag("r.0:1_f", [dom.txt("Counter: ", "r.0:1:0"), dom.el("span", "r.0:1:1", {}, [dom.txt(String(this.$$ctx.state.counter), "r.0:1:1:txt")])]))]));
        }
    }
    customElements.define("state-ordering-test", StateOrderingTestElement);
})();