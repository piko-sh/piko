/*
--- BEGIN AST DUMP ---

<div>
  <button id="set-a" [Events: p-on:click.="setKeyA" {P: setKeyA}]>
    "Show A"
  </button>
  <button id="set-b" [Events: p-on:click.="setKeyB" {P: setKeyB}]>
    "Show B"
  </button>
  <button id="set-c" [Events: p-on:click.="setKeyC" {P: setKeyC}]>
    "Show C"
  </button>
  <p id="current-key">
    <RichText>
      "Key: "
      {{ state.currentKey }}
    </RichText>
  </p>
  <p id="current-value">
    <RichText>
      "Value: "
      {{ state.data[state.currentKey] }}
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
        const $$initialState = {"data": {"a": "Alpha", "b": "Beta", "c": "Gamma"}, "currentKey": "a"};
        const state = makeReactive($$initialState, contextParam);
        function setKeyA() {
            state.currentKey = "a";
        }
        function setKeyB() {
            state.currentKey = "b";
        }
        function setKeyC() {
            state.currentKey = "c";
        }
        return {"state": state, "$$initialState": $$initialState, "setKeyA": setKeyA, "setKeyB": setKeyB, "setKeyC": setKeyC};
    }
    class ComputedPropertyAccessElement extends PPElement {
        constructor () {
            super();
            this._dir_click_setKeyA_evt_1 = (e) => {
                this.$$ctx.setKeyA.call(this, e);
            };
            this._dir_click_setKeyB_evt_2 = (e) => {
                this.$$ctx.setKeyB.call(this, e);
            };
            this._dir_click_setKeyC_evt_3 = (e) => {
                this.$$ctx.setKeyC.call(this, e);
            };
        }
        static get propTypes () {
            return {"currentKey": {"type": "string"}, "data": {"type": "object"}};
        }
        static get defaultProps () {
            return {"currentKey": "a", "data": {"a": "Alpha", "b": "Beta", "c": "Gamma"}};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "set-a", "onClick": this._dir_click_setKeyA_evt_1}, dom.txt("Show A", "r.0:0:0")), dom.el("button", "r.0:1", {"id": "set-b", "onClick": this._dir_click_setKeyB_evt_2}, dom.txt("Show B", "r.0:1:0")), dom.el("button", "r.0:2", {"id": "set-c", "onClick": this._dir_click_setKeyC_evt_3}, dom.txt("Show C", "r.0:2:0")), dom.el("p", "r.0:3", {"id": "current-key"}, dom.txt("Key: " + String(this.$$ctx.state.currentKey ?? ""), "r.0:3:0")), dom.el("p", "r.0:4", {"id": "current-value"}, dom.txt("Value: " + String(this.$$ctx.state.data[this.$$ctx.state.currentKey] ?? ""), "r.0:4:0"))]));
        }
    }
    customElements.define("computed-property-access", ComputedPropertyAccessElement);
})();