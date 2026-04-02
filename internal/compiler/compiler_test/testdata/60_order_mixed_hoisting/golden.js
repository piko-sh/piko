/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Result A: "
      {{ state.resultA }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Result B: "
      {{ state.resultB }}
    </RichText>
  </p>
  <button [Events: p-on:click.="refresh" {P: refresh}]>
    "Refresh"
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
        const arrowUsingHoisted = () => {
            return hoistedHelper() * 2;
        };
        const MULTIPLIER = 10;
        const arrowUsingConst = () => {
            return MULTIPLIER * 3;
        };
        function hoistedHelper() {
            return 21;
        }
        const COMPUTED_A = arrowUsingHoisted();
        const COMPUTED_B = arrowUsingConst();
        const laterArrow = () => {
            return hoistedHelper() + MULTIPLIER;
        };
        const $$initialState = {"resultA": COMPUTED_A, "resultB": COMPUTED_B};
        const state = makeReactive($$initialState, contextParam);
        function refresh() {
            state.resultA = laterArrow();
            state.resultB = arrowUsingHoisted() + arrowUsingConst();
        }
        function anotherHelper() {
            return 100;
        }
        return {"state": state, "$$initialState": $$initialState, "arrowUsingHoisted": arrowUsingHoisted, "arrowUsingConst": arrowUsingConst, "hoistedHelper": hoistedHelper, "laterArrow": laterArrow, "refresh": refresh, "anotherHelper": anotherHelper};
    }
    class MixedHoistingElement extends PPElement {
        constructor () {
            super();
            this._dir_click_refresh_evt_1 = (e) => {
                this.$$ctx.refresh.call(this, e);
            };
        }
        static get propTypes () {
            return {"resultA": {"type": "number"}, "resultB": {"type": "number"}};
        }
        static get defaultProps () {
            return {"resultA": null, "resultB": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Result A: " + String(this.$$ctx.state.resultA ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Result B: " + String(this.$$ctx.state.resultB ?? ""), "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_refresh_evt_1}, dom.txt("Refresh", "r.0:2:0"))]));
        }
    }
    customElements.define("mixed-hoisting", MixedHoistingElement);
})();