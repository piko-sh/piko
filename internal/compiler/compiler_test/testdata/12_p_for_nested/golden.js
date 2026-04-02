/*
--- BEGIN AST DUMP ---

<div class="matrix">
  <div class="row" [p-for: (rowIndex, row) in state.matrix] [p-key: rowIndex]>
    <span class="cell" [p-for: (colIndex, cell) in row] [p-key: colIndex]>
      <RichText>
        "\n                "
        {{ rowIndex }}
        "."
        {{ colIndex }}
        ":"
        {{ cell }}
        "\n            "
      </RichText>
    </span>
  </div>
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
        const $$initialState = {"matrix": [["A", "B"], ["C", "D"]]};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PForNestedElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"matrix": {"type": "array", "itemType": "array"}};
        }
        static get defaultProps () {
            return {"matrix": [["A", "B"], ["C", "D"]]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "matrix"}, (Array.isArray(this.$$ctx.state.matrix) ? this.$$ctx.state.matrix : this.$$ctx.state.matrix && typeof this.$$ctx.state.matrix === "object" ? Object.entries(this.$$ctx.state.matrix) : []).map((row, rowIndex) => {
                return dom.el("div", "r.0:0." + String(rowIndex ?? ""), {"class": "row"}, (Array.isArray(row) ? row : row && typeof row === "object" ? Object.entries(row) : []).map((cell, colIndex) => {
                    return dom.el("span", "r.0:0." + String(rowIndex ?? "") + ":0." + String(colIndex ?? ""), {"class": "cell"}, dom.txt("\n                " + String(rowIndex ?? "") + "." + String(colIndex ?? "") + ":" + String(cell ?? "") + "\n            ", "r.0:0." + String(rowIndex ?? "") + ":0." + String(colIndex ?? "") + ":0"));
                }));
            }));
        }
    }
    customElements.define("p-for-nested", PForNestedElement);
})();