/*
--- BEGIN AST DUMP ---

<div>
  <h2>
    "TypeScript Mixed (Explicit + Inferred) Test"
  </h2>
  <p>
    "Explicit:"
    <span [p-text: state.explicit] />
  </p>
  <p>
    "Inferred:"
    <span [p-text: state.inferred] />
  </p>
  <p>
    <RichText>
      "Explicit Array: "
      {{ state.explicitArray.length }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Inferred Array: "
      {{ state.inferredArray.length }}
    </RichText>
  </p>
  <button :disabled="state.explicitBool" {P: state.explicitBool}>
    "Explicit Boolean"
  </button>
  <button :disabled="state.inferredBool" {P: state.inferredBool}>
    "Inferred Boolean"
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
        const $$initialState = {"explicit": 0, "explicitStr": "test", "explicitBool": false, "explicitArray": [], "inferred": 100, "inferredStr": "hello", "inferredBool": true, "inferredArray": [1, 2, 3]};
        const state = makeReactive($$initialState, contextParam);
        function update() {
            state.explicit++;
            state.inferred++;
        }
        return {"state": state, "$$initialState": $$initialState, "update": update};
    }
    class TsMixedTestElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"explicit": {"type": "number"}, "explicitArray": {"type": "array", "itemType": "string"}, "explicitBool": {"type": "boolean"}, "explicitStr": {"type": "string"}, "inferred": {"type": "number"}, "inferredArray": {"type": "array", "itemType": "number"}, "inferredBool": {"type": "boolean"}, "inferredStr": {"type": "string"}};
        }
        static get defaultProps () {
            return {"explicit": 0, "explicitArray": [], "explicitBool": false, "explicitStr": "test", "inferred": 100, "inferredArray": [1, 2, 3], "inferredBool": true, "inferredStr": "hello"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("h2", "r.0:0", {}, dom.txt("TypeScript Mixed (Explicit + Inferred) Test", "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.frag("r.0:1_f", [dom.txt("Explicit: ", "r.0:1:0"), dom.el("span", "r.0:1:1", {}, [dom.txt(String(this.$$ctx.state.explicit), "r.0:1:1:txt")])])), dom.el("p", "r.0:2", {}, dom.frag("r.0:2_f", [dom.txt("Inferred: ", "r.0:2:0"), dom.el("span", "r.0:2:1", {}, [dom.txt(String(this.$$ctx.state.inferred), "r.0:2:1:txt")])])), dom.el("p", "r.0:3", {}, dom.txt("Explicit Array: " + String(this.$$ctx.state.explicitArray.length ?? ""), "r.0:3:0")), dom.el("p", "r.0:4", {}, dom.txt("Inferred Array: " + String(this.$$ctx.state.inferredArray.length ?? ""), "r.0:4:0")), dom.el("button", "r.0:5", {"?disabled": (this.$$ctx.state.explicitBool)}, dom.txt("Explicit Boolean", "r.0:5:0")), dom.el("button", "r.0:6", {"?disabled": (this.$$ctx.state.inferredBool)}, dom.txt("Inferred Boolean", "r.0:6:0"))]));
        }
        static get css () {
            return "h2{color: #666}";
        }
    }
    customElements.define("ts-mixed-test", TsMixedTestElement);
})();