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

export const BROWSER_GLOBALS = {
  document: "readonly",
  window: "readonly",
  console: "readonly",
  HTMLElement: "readonly",
  Element: "readonly",
  Node: "readonly",
  NodeList: "readonly",
  Event: "readonly",
  CustomEvent: "readonly",
  MutationObserver: "readonly",
  requestAnimationFrame: "readonly",
  cancelAnimationFrame: "readonly",
  setTimeout: "readonly",
  clearTimeout: "readonly",
  setInterval: "readonly",
  clearInterval: "readonly",
  fetch: "readonly",
  URL: "readonly",
  URLSearchParams: "readonly",
  FormData: "readonly",
  Blob: "readonly",
  File: "readonly",
  FileReader: "readonly",
  localStorage: "readonly",
  sessionStorage: "readonly",
  navigator: "readonly",
  location: "readonly",
  history: "readonly",
  performance: "readonly",
  queueMicrotask: "readonly",
  ResizeObserver: "readonly",
  IntersectionObserver: "readonly",
  DOMParser: "readonly",
  XMLSerializer: "readonly",
  AbortController: "readonly",
  AbortSignal: "readonly",
  Headers: "readonly",
  Request: "readonly",
  Response: "readonly",
  TextEncoder: "readonly",
  TextDecoder: "readonly",
  Map: "readonly",
  Set: "readonly",
  WeakMap: "readonly",
  WeakSet: "readonly",
  Promise: "readonly",
  Proxy: "readonly",
  Reflect: "readonly",
  Symbol: "readonly",
  atob: "readonly",
  btoa: "readonly",
  __DEV__: "readonly",
};

export const BROWSER_TEST_GLOBALS = {
  describe: "readonly",
  it: "readonly",
  expect: "readonly",
  beforeEach: "readonly",
  afterEach: "readonly",
  beforeAll: "readonly",
  afterAll: "readonly",
  vi: "readonly",
  test: "readonly",
  document: "readonly",
  window: "readonly",
  console: "readonly",
  HTMLElement: "readonly",
  HTMLDivElement: "readonly",
  HTMLInputElement: "readonly",
  HTMLButtonElement: "readonly",
  HTMLFormElement: "readonly",
  HTMLSlotElement: "readonly",
  Element: "readonly",
  Node: "readonly",
  NodeList: "readonly",
  Event: "readonly",
  CustomEvent: "readonly",
  MutationObserver: "readonly",
  requestAnimationFrame: "readonly",
  cancelAnimationFrame: "readonly",
  setTimeout: "readonly",
  clearTimeout: "readonly",
  setInterval: "readonly",
  clearInterval: "readonly",
  fetch: "readonly",
  URL: "readonly",
  URLSearchParams: "readonly",
  FormData: "readonly",
  DOMParser: "readonly",
  history: "readonly",
  location: "readonly",
  customElements: "readonly",
  getComputedStyle: "readonly",
  btoa: "readonly",
  atob: "readonly",
  global: "readonly",
  Text: "readonly",
  Comment: "readonly",
  DocumentFragment: "readonly",
  ShadowRoot: "readonly",
  Range: "readonly",
  Selection: "readonly",
  __DEV__: "readonly",
};

export const NODE_GLOBALS = {
  console: "readonly",
  process: "readonly",
  Buffer: "readonly",
  __dirname: "readonly",
  __filename: "readonly",
  module: "readonly",
  require: "readonly",
  exports: "readonly",
  setTimeout: "readonly",
  clearTimeout: "readonly",
  setInterval: "readonly",
  clearInterval: "readonly",
  setImmediate: "readonly",
  clearImmediate: "readonly",
  URL: "readonly",
  URLSearchParams: "readonly",
  Map: "readonly",
  Set: "readonly",
  WeakMap: "readonly",
  WeakSet: "readonly",
  Promise: "readonly",
  Proxy: "readonly",
  Reflect: "readonly",
  Symbol: "readonly",
  TextEncoder: "readonly",
  TextDecoder: "readonly",
};

export const NODE_TEST_GLOBALS = {
  describe: "readonly",
  it: "readonly",
  expect: "readonly",
  beforeEach: "readonly",
  afterEach: "readonly",
  beforeAll: "readonly",
  afterAll: "readonly",
  test: "readonly",
  console: "readonly",
  process: "readonly",
  Buffer: "readonly",
  __dirname: "readonly",
  __filename: "readonly",
  module: "readonly",
  require: "readonly",
};

const SHARED_RULES = {
  "no-unused-vars": "off",
  "no-undef": "off",

  "max-len": [
    "warn",
    {
      code: 200,
      ignoreUrls: true,
      ignoreStrings: true,
      ignoreTemplateLiterals: true,
      ignoreRegExpLiterals: true,
    },
  ],
  "max-lines": ["warn", { max: 1000, skipBlankLines: true, skipComments: true }],
  "max-lines-per-function": ["error", { max: 80, skipBlankLines: true, skipComments: true }],
  "max-statements": ["warn", 60],
  "max-params": ["warn", 7],
  complexity: ["warn", 15],
  "max-depth": ["warn", 3],

  "no-multiple-empty-lines": ["warn", { max: 1, maxEOF: 0, maxBOF: 0 }],
  "no-else-return": ["warn", { allowElseIf: false }],
  "no-useless-return": "warn",
  "no-useless-constructor": "off",
  "@typescript-eslint/no-useless-constructor": "warn",
  "no-var": "error",
  "prefer-const": "warn",
  "no-plusplus": "off",
  "no-unneeded-ternary": ["warn", { defaultAssignment: false }],
  "no-empty": ["warn", { allowEmptyCatch: true }],
  "no-duplicate-imports": "warn",

  "@typescript-eslint/naming-convention": [
    "warn",
    { selector: "variable", format: ["camelCase", "UPPER_CASE", "PascalCase"], leadingUnderscore: "allow" },
    { selector: "function", format: ["camelCase", "PascalCase"], leadingUnderscore: "allow" },
    { selector: "parameter", format: ["camelCase"], leadingUnderscore: "allow", filter: { regex: "^__", match: false } },
    { selector: "parameter", format: null, filter: { regex: "^__", match: true } },
    { selector: "classProperty", format: ["camelCase", "UPPER_CASE", "PascalCase"], leadingUnderscore: "allow" },
    { selector: "classMethod", format: ["camelCase"], leadingUnderscore: "allow" },
    { selector: "objectLiteralProperty", format: null },
    { selector: "objectLiteralMethod", format: null },
    { selector: "typeProperty", format: ["camelCase", "UPPER_CASE", "PascalCase"], leadingUnderscore: "allow", filter: { regex: "^__", match: false } },
    { selector: "typeProperty", format: null, filter: { regex: "^__", match: true } },
    { selector: "property", format: ["camelCase", "UPPER_CASE", "PascalCase"], leadingUnderscore: "allow", filter: { regex: "^__", match: false } },
    { selector: "property", format: null, filter: { regex: "^__", match: true } },
    { selector: "method", format: ["camelCase"], leadingUnderscore: "allow" },
    { selector: "typeLike", format: ["PascalCase"] },
    { selector: "enumMember", format: ["PascalCase", "UPPER_CASE"] },
  ],

  "jsdoc/require-jsdoc": [
    "warn",
    {
      publicOnly: false,
      require: {
        FunctionDeclaration: false,
        MethodDefinition: false,
        ClassDeclaration: false,
        ArrowFunctionExpression: false,
        FunctionExpression: false,
      },
      contexts: [
        "ExportNamedDeclaration > FunctionDeclaration",
        "ExportNamedDeclaration > ClassDeclaration",
        "ExportNamedDeclaration > VariableDeclaration > VariableDeclarator > ArrowFunctionExpression",
        "ExportDefaultDeclaration > FunctionDeclaration",
        "ExportDefaultDeclaration > ClassDeclaration",
        "ExportNamedDeclaration > TSTypeAliasDeclaration",
        "ExportNamedDeclaration > TSInterfaceDeclaration",
        "ExportNamedDeclaration > TSEnumDeclaration",
      ],
    },
  ],
  "jsdoc/require-description": ["warn", { contexts: ["any"] }],
  "jsdoc/require-param-description": "warn",
  "jsdoc/require-returns-description": "warn",
  "jsdoc/no-types": "warn",
  "jsdoc/require-hyphen-before-param-description": ["warn", "always"],

  "@typescript-eslint/no-floating-promises": "error",
  "@typescript-eslint/no-misused-promises": "error",
  "promise/catch-or-return": "warn",
  "promise/always-return": "off",
  "promise/no-return-wrap": "warn",
  "@typescript-eslint/prefer-promise-reject-errors": "warn",
  "no-restricted-imports": "off",
  "no-unreachable": "error",
  "no-shadow-restricted-names": "error",
  "@typescript-eslint/no-shadow": "warn",
  "@typescript-eslint/consistent-type-assertions": ["warn", { assertionStyle: "as", objectLiteralTypeAssertions: "allow-as-parameter" }],
  "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrorsIgnorePattern: "^_" }],
  "no-unsafe-finally": "error",

  "no-fallthrough": "error",
  "no-constant-binary-expression": "warn",
  "no-constant-condition": ["warn", { checkLoops: false }],

  "@typescript-eslint/no-magic-numbers": [
    "warn",
    {
      ignore: [0, 1, -1, 2],
      ignoreArrayIndexes: true,
      ignoreDefaultValues: true,
      ignoreEnums: true,
      ignoreNumericLiteralTypes: true,
      ignoreReadonlyClassProperties: true,
      ignoreTypeIndexes: true,
    },
  ],
  "@typescript-eslint/explicit-function-return-type": [
    "warn",
    {
      allowExpressions: true,
      allowTypedFunctionExpressions: true,
      allowHigherOrderFunctions: true,
      allowDirectConstAssertionInArrowFunctions: true,
      allowConciseArrowFunctionExpressionsStartingWithVoid: true,
      allowFunctionsWithoutTypeParameters: true,
      allowIIFEs: true,
    },
  ],

  "@typescript-eslint/no-explicit-any": "warn",
  "@typescript-eslint/explicit-member-accessibility": "off",
  "@typescript-eslint/no-non-null-assertion": "warn",
  "@typescript-eslint/prefer-nullish-coalescing": "warn",
  "@typescript-eslint/prefer-optional-chain": "warn",
  "@typescript-eslint/strict-boolean-expressions": "off",
  "@typescript-eslint/no-unnecessary-condition": "warn",
  "@typescript-eslint/no-unsafe-argument": "warn",
  "@typescript-eslint/no-unsafe-assignment": "warn",
  "@typescript-eslint/no-unsafe-call": "warn",
  "@typescript-eslint/no-unsafe-member-access": "warn",
  "@typescript-eslint/no-unsafe-return": "warn",

  "no-lonely-if": "warn",
  "prefer-template": "warn",
  "consistent-return": "off",
  "object-shorthand": ["warn", "always"],
  "arrow-body-style": ["warn", "as-needed"],
  "no-console": ["warn", { allow: ["warn", "error"] }],
  "prefer-arrow-callback": "warn",
  "no-nested-ternary": "warn",
  curly: ["warn", "all"],
  "brace-style": ["warn", "1tbs", { allowSingleLine: true }],
  "no-implied-eval": "error",
  "no-script-url": "error",
  "@typescript-eslint/no-unused-expressions": "warn",
};

const TEST_RULE_OVERRIDES = {
  "no-undef": "off",
  "no-unused-vars": "off",
  "max-lines-per-function": "off",
  "max-statements": "off",
  "max-lines": "off",
  "@typescript-eslint/no-magic-numbers": "off",
  "@typescript-eslint/no-explicit-any": "off",
  "@typescript-eslint/no-non-null-assertion": "off",
  "@typescript-eslint/no-unsafe-argument": "off",
  "@typescript-eslint/no-unsafe-assignment": "off",
  "@typescript-eslint/no-unsafe-call": "off",
  "@typescript-eslint/no-unsafe-member-access": "off",
  "@typescript-eslint/no-unsafe-return": "off",
  "@typescript-eslint/no-unused-vars": "off",
  "jsdoc/require-jsdoc": "off",
  "no-console": "off",
  complexity: "off",
};

/**
 * Creates a standardised ESLint flat config array.
 *
 * Plugins are passed in from each project so that Node.js resolves them from
 * the project's own node_modules (ESM resolves imports relative to the file
 * location, not the caller).
 *
 * @param options.js - The @eslint/js module.
 * @param options.tseslint - The @typescript-eslint/eslint-plugin module.
 * @param options.tsparser - The @typescript-eslint/parser module.
 * @param options.jsdoc - The eslint-plugin-jsdoc module.
 * @param options.promise - The eslint-plugin-promise module.
 * @param options.globals - Global variables for source files.
 * @param options.testGlobals - Global variables for test files.
 * @param options.ignores - Ignore patterns (e.g. ["out/**"]).
 * @param options.extraConfigBlocks - Additional config objects appended to the array.
 */
export function createEslintConfig({
  js,
  tseslint,
  tsparser,
  jsdoc,
  promise,
  globals = BROWSER_GLOBALS,
  testGlobals = BROWSER_TEST_GLOBALS,
  ignores = ["node_modules/**", "dist/**"],
  extraConfigBlocks = [],
}) {
  return [
    js.configs.recommended,

    {
      files: ["**/*.ts", "**/*.tsx"],
      ignores: [...ignores, "**/*.spec.ts", "**/test/**"],
      languageOptions: {
        parser: tsparser,
        parserOptions: {
          ecmaVersion: "latest",
          sourceType: "module",
          project: "./tsconfig.json",
        },
        globals,
      },
      plugins: {
        "@typescript-eslint": tseslint,
        jsdoc,
        promise,
      },
      rules: { ...SHARED_RULES },
    },

    {
      files: ["**/*.spec.ts", "**/test/**/*.ts"],
      languageOptions: {
        parser: tsparser,
        parserOptions: {
          ecmaVersion: "latest",
          sourceType: "module",
          project: "./tsconfig.json",
        },
        globals: testGlobals,
      },
      plugins: {
        "@typescript-eslint": tseslint,
      },
      rules: { ...TEST_RULE_OVERRIDES },
    },

    {
      files: ["*.config.ts", "*.config.mjs", "*.config.js"],
      rules: {
        "jsdoc/require-jsdoc": "off",
        "@typescript-eslint/no-magic-numbers": "off",
      },
    },

    {
      ignores,
    },

    ...extraConfigBlocks,
  ];
}
