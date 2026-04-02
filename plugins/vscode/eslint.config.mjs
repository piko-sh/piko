// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

import js from "@eslint/js";
import tseslint from "@typescript-eslint/eslint-plugin";
import tsparser from "@typescript-eslint/parser";
import jsdoc from "eslint-plugin-jsdoc";
import promise from "eslint-plugin-promise";
import { createEslintConfig, NODE_GLOBALS, NODE_TEST_GLOBALS } from "../../frontend/eslint.base.mjs";

export default createEslintConfig({
  js, tseslint, tsparser, jsdoc, promise,
  globals: NODE_GLOBALS,
  testGlobals: NODE_TEST_GLOBALS,
  ignores: ["node_modules/**", "out/**", ".vscode-test/**"],
  extraConfigBlocks: [
    {
      files: ["**/*.ts"],
      rules: { "no-console": "off" },
    },
    {
      files: ["esbuild.js"],
      rules: {
        "jsdoc/require-jsdoc": "off",
        "@typescript-eslint/no-magic-numbers": "off",
      },
    },
  ],
});
