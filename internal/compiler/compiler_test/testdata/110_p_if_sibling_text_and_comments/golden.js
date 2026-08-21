/*
--- BEGIN AST DUMP ---

<div>
  <span>
    <i [p-if: state.flag]>
      "A"
    </i>
    <RichText>
      {{ state.after }}
    </RichText>
    <i class="tail" />
  </span>
  <span>
    <i [p-if: state.flag]>
      "A"
    </i>
    <RichText>
      {{ state.after }}
    </RichText>
  </span>
  <p>
    <b [p-if: state.flag]>
      "Y"
    </b>
    <!--  kept  -->
    <i class="tail" />
  </p>
  <p>
    <b [p-if: state.flag]>
      "Y"
    </b>
    <!--  absorbed  -->
    <b [p-else]>
      "N"
    </b>
    <RichText>
      {{ state.after }}
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
        const $$initialState = {"flag": true, "after": "tail text"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PIfSiblingsElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"after": {"type": "string"}, "flag": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"after": "tail text", "flag": true};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("span", "r.0:0", {}, dom.frag("r.0:0_f", [this.$$ctx.state.flag ? dom.el("i", "r.0:0:0", {}, dom.txt("A", "r.0:0:0:0")) : null, dom.txt(String(this.$$ctx.state.after ?? ""), "r.0:0:1"), dom.el("i", "r.0:0:2", {"class": "tail"}, null)])), dom.el("span", "r.0:1", {}, dom.frag("r.0:1_f", [this.$$ctx.state.flag ? dom.el("i", "r.0:1:0", {}, dom.txt("A", "r.0:1:0:0")) : null, dom.txt(String(this.$$ctx.state.after ?? ""), "r.0:1:1")])), dom.el("p", "r.0:2", {}, dom.frag("r.0:2_f", [this.$$ctx.state.flag ? dom.el("b", "r.0:2:0", {}, dom.txt("Y", "r.0:2:0:0")) : null, dom.cmt(" kept ", "r.0:2:1"), dom.el("i", "r.0:2:2", {"class": "tail"}, null)])), dom.el("p", "r.0:3", {}, dom.frag("r.0:3_f", [this.$$ctx.state.flag ? dom.el("b", "r.0:3:0", {}, dom.txt("Y", "r.0:3:0:0")) : dom.el("b", "r.0:3:2", {}, dom.txt("N", "r.0:3:2:0")), dom.txt(String(this.$$ctx.state.after ?? ""), "r.0:3:3")]))]));
        }
    }
    customElements.define("p-if-siblings", PIfSiblingsElement);
})();