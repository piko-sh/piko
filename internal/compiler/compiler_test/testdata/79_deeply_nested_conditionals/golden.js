/*
--- BEGIN AST DUMP ---

<div>
  <button id="toggle-a" [Events: p-on:click.="toggleA" {P: toggleA}]>
    "Toggle A"
  </button>
  <button id="toggle-b" [Events: p-on:click.="toggleB" {P: toggleB}]>
    "Toggle B"
  </button>
  <button id="toggle-c" [Events: p-on:click.="toggleC" {P: toggleC}]>
    "Toggle C"
  </button>
  <div id="level1" [p-if: state.showA]>
    <p>
      "Level 1 (A visible)"
    </p>
    <div id="level2" [p-if: state.showB]>
      <p>
        "Level 2 (B visible)"
      </p>
      <div id="level3" [p-if: state.showC]>
        <p id="deepest">
          "Level 3 (All visible)"
        </p>
      </div>
      <p id="no-c" [p-else]>
        "C is hidden"
      </p>
    </div>
    <p id="no-b" [p-else]>
      "B is hidden"
    </p>
  </div>
  <p id="no-a" [p-else]>
    "A is hidden"
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
        const $$initialState = {"showA": false, "showB": false, "showC": false};
        const state = makeReactive($$initialState, contextParam);
        function toggleA() {
            state.showA = !state.showA;
        }
        function toggleB() {
            state.showB = !state.showB;
        }
        function toggleC() {
            state.showC = !state.showC;
        }
        return {"state": state, "$$initialState": $$initialState, "toggleA": toggleA, "toggleB": toggleB, "toggleC": toggleC};
    }
    class DeeplyNestedConditionalsElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggleA_evt_1 = (e) => {
                this.$$ctx.toggleA.call(this, e);
            };
            this._dir_click_toggleB_evt_2 = (e) => {
                this.$$ctx.toggleB.call(this, e);
            };
            this._dir_click_toggleC_evt_3 = (e) => {
                this.$$ctx.toggleC.call(this, e);
            };
        }
        static get propTypes () {
            return {"showA": {"type": "boolean"}, "showB": {"type": "boolean"}, "showC": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"showA": false, "showB": false, "showC": false};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "toggle-a", "onClick": this._dir_click_toggleA_evt_1}, dom.txt("Toggle A", "r.0:0:0")), dom.el("button", "r.0:1", {"id": "toggle-b", "onClick": this._dir_click_toggleB_evt_2}, dom.txt("Toggle B", "r.0:1:0")), dom.el("button", "r.0:2", {"id": "toggle-c", "onClick": this._dir_click_toggleC_evt_3}, dom.txt("Toggle C", "r.0:2:0")), this.$$ctx.state.showA ? dom.el("div", "r.0:3", {"id": "level1"}, dom.frag("r.0:3_f", [dom.el("p", "r.0:3:0", {}, dom.txt("Level 1 (A visible)", "r.0:3:0:0")), this.$$ctx.state.showB ? dom.el("div", "r.0:3:1", {"id": "level2"}, dom.frag("r.0:3:1_f", [dom.el("p", "r.0:3:1:0", {}, dom.txt("Level 2 (B visible)", "r.0:3:1:0:0")), this.$$ctx.state.showC ? dom.el("div", "r.0:3:1:1", {"id": "level3"}, dom.el("p", "r.0:3:1:1:0", {"id": "deepest"}, dom.txt("Level 3 (All visible)", "r.0:3:1:1:0:0"))) : dom.el("p", "r.0:3:1:2", {"id": "no-c"}, dom.txt("C is hidden", "r.0:3:1:2:0"))])) : dom.el("p", "r.0:3:2", {"id": "no-b"}, dom.txt("B is hidden", "r.0:3:2:0"))])) : dom.el("p", "r.0:4", {"id": "no-a"}, dom.txt("A is hidden", "r.0:4:0"))]));
        }
    }
    customElements.define("deeply-nested-conditionals", DeeplyNestedConditionalsElement);
})();