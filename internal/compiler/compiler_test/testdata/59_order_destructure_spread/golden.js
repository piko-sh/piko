/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Theme: "
      {{ state.currentTheme }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Items: "
      {{ state.itemCount }}
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
        const defaults = {"theme": "dark", "size": "medium", "enabled": true};
        const {"theme": theme, "size": size} = defaults;
        const extended = {...defaults, "extra": "value", "overridden": "new"};
        const baseItems = ["a", "b", "c"];
        const moreItems = ["d", "e"];
        const allItems = [...baseItems, ...moreItems];
        const reversedItems = [...allItems].reverse();
        const ITEM_COUNT = allItems.length;
        const $$initialState = {"currentTheme": theme, "itemCount": ITEM_COUNT};
        const state = makeReactive($$initialState, contextParam);
        function cycleTheme() {
            const themes = ["dark", "light", "system"];
            const idx = themes.indexOf(state.currentTheme);
            state.currentTheme = themes[(idx + 1) % themes.length];
        }
        return {"state": state, "$$initialState": $$initialState, "cycleTheme": cycleTheme};
    }
    class DestructureSpreadElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"currentTheme": {"type": "string"}, "itemCount": {"type": "number"}};
        }
        static get defaultProps () {
            return {"currentTheme": null, "itemCount": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Theme: " + String(this.$$ctx.state.currentTheme ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Items: " + String(this.$$ctx.state.itemCount ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("destructure-spread", DestructureSpreadElement);
})();