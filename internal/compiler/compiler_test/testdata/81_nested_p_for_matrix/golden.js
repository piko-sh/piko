/*
--- BEGIN AST DUMP ---

<div>
  <table>
    <tr [p-for: (rowIndex, row) in state.matrix] [p-key: rowIndex]>
      <td [p-for: (colIndex, cell) in row] [p-key: colIndex]>
        <RichText>
          "\n                    "
          {{ cell }}
          "\n                "
        </RichText>
      </td>
    </tr>
  </table>
  <p id="total">
    <RichText>
      "Total rows: "
      {{ state.matrix.length }}
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
        const $$initialState = {"matrix": [[1, 2, 3], [4, 5, 6], [7, 8, 9]]};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class NestedPForMatrixElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"matrix": {"type": "array", "itemType": "array"}};
        }
        static get defaultProps () {
            return {"matrix": [[1, 2, 3], [4, 5, 6], [7, 8, 9]]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("table", "r.0:0", {}, (Array.isArray(this.$$ctx.state.matrix) ? this.$$ctx.state.matrix : this.$$ctx.state.matrix && typeof this.$$ctx.state.matrix === "object" ? Object.entries(this.$$ctx.state.matrix) : []).map((row, rowIndex) => {
                return dom.el("tr", "r.0:0:0." + String(rowIndex ?? ""), {}, (Array.isArray(row) ? row : row && typeof row === "object" ? Object.entries(row) : []).map((cell, colIndex) => {
                    return dom.el("td", "r.0:0:0." + String(rowIndex ?? "") + ":0." + String(colIndex ?? ""), {}, dom.txt("\n                    " + String(cell ?? "") + "\n                ", "r.0:0:0." + String(rowIndex ?? "") + ":0." + String(colIndex ?? "") + ":0"));
                }));
            })), dom.el("p", "r.0:1", {"id": "total"}, dom.txt("Total rows: " + String(this.$$ctx.state.matrix.length ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("nested-p-for-matrix", NestedPForMatrixElement);
})();