/*
--- BEGIN AST DUMP ---

<div>
  <div [p-html: state.markup] />
  <svg viewBox="0 0 10 10" [p-html: state.shapes] />
  <piko:element is="section" [p-html: state.markup] />
  <piko:element :is="state.tag" {P: state.tag} [p-html: state.markup] />
  <span [p-text: state.label] />
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
        const $$initialState = {"markup": "<em>injected</em>", "shapes": "<circle r='4'></circle>", "tag": "section", "label": "plain"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PHtmlElementValueElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"label": {"type": "string"}, "markup": {"type": "string"}, "shapes": {"type": "string"}, "tag": {"type": "string"}};
        }
        static get defaultProps () {
            return {"label": "plain", "markup": "<em>injected</em>", "shapes": "<circle r='4'></circle>", "tag": "section"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("div", "r.0:0", {}, [], String(this.$$ctx.state.markup)), dom.el("svg", "r.0:1", {"viewBox": "0 0 10 10"}, [], String(this.$$ctx.state.shapes)), dom.el("section", "r.0:2", {}, [], String(this.$$ctx.state.markup)), dom.pikoEl(this.$$ctx.state.tag, "r.0:3", {}, [], "", String(this.$$ctx.state.markup)), dom.el("span", "r.0:4", {}, [dom.txt(String(this.$$ctx.state.label), "r.0:4:txt")])]));
        }
    }
    customElements.define("p-html-element-value", PHtmlElementValueElement);
})();