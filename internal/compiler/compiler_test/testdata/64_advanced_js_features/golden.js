/*
--- BEGIN AST DUMP ---

<div>
  <p class="result" [p-text: state.result] />
  <button class="test-dowhile" [Events: p-on:click.="testDoWhile" {P: testDoWhile}]>
    "Test Do-While"
  </button>
  <button class="test-sparse" [Events: p-on:click.="testSparseArray" {P: testSparseArray}]>
    "Test Sparse Array"
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
        const $$initialState = {"result": "Ready"};
        const state = makeReactive($$initialState, contextParam);
        function testDoWhile() {
            let count = 0;
            do {
                count++;
            } while (count < 3);
            state.result = "Do-While: " + count;
        }
        function testSparseArray() {
            const sparse = [1, , 3, , 5];
            const len = sparse.length;
            state.result = "Sparse Array Length: " + len;
        }
        return {"state": state, "$$initialState": $$initialState, "testDoWhile": testDoWhile, "testSparseArray": testSparseArray};
    }
    class AdvancedJsTestElement extends PPElement {
        constructor () {
            super();
            this._dir_click_testDoWhile_evt_1 = (e) => {
                this.$$ctx.testDoWhile.call(this, e);
            };
            this._dir_click_testSparseArray_evt_2 = (e) => {
                this.$$ctx.testSparseArray.call(this, e);
            };
        }
        static get propTypes () {
            return {"result": {"type": "string"}};
        }
        static get defaultProps () {
            return {"result": "Ready"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {"class": "result"}, [dom.txt(String(this.$$ctx.state.result), "r.0:0:txt")]), dom.el("button", "r.0:1", {"class": "test-dowhile", "onClick": this._dir_click_testDoWhile_evt_1}, dom.txt("Test Do-While", "r.0:1:0")), dom.el("button", "r.0:2", {"class": "test-sparse", "onClick": this._dir_click_testSparseArray_evt_2}, dom.txt("Test Sparse Array", "r.0:2:0"))]));
        }
        static get css () {
            return ":host{display: block}.result{font-weight: bold;margin-bottom: 10px}button{margin-right: 5px}";
        }
    }
    customElements.define("advanced-js-test", AdvancedJsTestElement);
})();