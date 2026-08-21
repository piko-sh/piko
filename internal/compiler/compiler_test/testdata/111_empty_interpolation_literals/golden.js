/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "foo"
    </RichText>
  </p>
  <p>
    <RichText>
      "Hello "
    </RichText>
  </p>
  <p>
    <RichText>
      "a"
    </RichText>
  </p>
  <p>
    <RichText>
      "a"
      "b"
    </RichText>
  </p>
  <p>
    <RichText>
      {{ state.value }}
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
        const $$initialState = {"value": "real"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class EmptyInterpolationElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"value": {"type": "string"}};
        }
        static get defaultProps () {
            return {"value": "real"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("foo", "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Hello ", "r.0:1:0")), dom.el("p", "r.0:2", {}, dom.txt("a", "r.0:2:0")), dom.el("p", "r.0:3", {}, dom.txt("a" + "b", "r.0:3:0")), dom.el("p", "r.0:4", {}, dom.txt(String(this.$$ctx.state.value ?? ""), "r.0:4:0"))]));
        }
    }
    customElements.define("empty-interpolation", EmptyInterpolationElement);
})();