---
title: Diagnostics
description: Every diagnostic code Piko reports, what it means, and how to fix it.
nav:
  sidebar:
    section: "reference"
    subsection: "tooling"
    order: 20
---

<!-- Generated from internal/diagnostic_docs. Edit the catalogue there, then run: make generate-diagnostics-docs -->

# Diagnostics

When Piko builds a project or renders a page, it reports problems as diagnostics. Each one
has a severity and a stable code. Codes starting with T0 come from parsing template
expressions, T1 from analysing templates, and R0 from rendering. A code keeps its meaning
and its number is never reused, so editors, scripts and CI can rely on it.

Build output follows each coded diagnostic with a link to its explanation:

```text
 ● error [T141]: Invalid operation: cannot strictly compare type 'app.Level' to 'string'
   --> pages/index.pk:6:14
  |
6 |   <span p-if="state.Level == 'error'">!</span>
  |              ^^^^^^^^^^^^^^^^^^^^^^^^^^^
  = docs: https://piko.sh/diagnostics/T141 (or run: piko explain T141)
```

Run `piko explain T141` to read an explanation in the terminal, or
`piko explain` to list every code; see the [CLI reference](cli.md). Editors
that use the Piko language server link each code to its explanation; see
[LSP setup](../how-to/lsp-setup.md).

Severities:

- **Error** stops the build, or marks a render that failed.
- **Warning** reports a likely mistake. The build continues.
- **Info** reports something worth knowing. The build continues.

Some codes are reported with more than one severity, depending on the case; each code
below says when.

SQL diagnostics, whose codes start with Q, come from the SQL generator. They are listed in
the [querier reference](querier.md#diagnostics).

## All codes {#all-codes}

### Template expressions {#template-expressions-codes}

| Code | Severity | Meaning |
|---|---|---|
| [T001](#t001) | Error | Invalid number literal |
| [T002](#t002) | Error | Invalid string literal |
| [T003](#t003) | Error | Invalid escape sequence |
| [T004](#t004) | Error | Invalid date, time or duration |
| [T005](#t005) | Error | Invalid rune literal |
| [T006](#t006) | Error | Invalid template literal |
| [T007](#t007) | Error | Unterminated interpolation |
| [T008](#t008) | Error | Unexpected token |
| [T009](#t009) | Error | Unexpected end of expression |
| [T010](#t010) | Error | Missing identifier |
| [T011](#t011) | Error | Missing operand |
| [T012](#t012) | Error | Missing closing bracket |
| [T013](#t013) | Error | Missing separator |
| [T014](#t014) | Error | Incomplete expression |
| [T015](#t015) | Error | Expression nested too deeply |
| [T016](#t016) | Error | Feature not allowed here |
| [T017](#t017) | Error | Unreadable expression |
| [T018](#t018) | Error | Unexpected text after expression |
| [T019](#t019) | Error | Unexpected comma |
| [T020](#t020) | Error | Conflicting directives |
| [T021](#t021) | Warning | Duplicate directive |
| [T022](#t022) | Error | Invalid directive value |
| [T023](#t023) | Error | p-else without p-if |
| [T024](#t024) | Warning | Directive written before p-for |
| [T025](#t025) | Error | Directive not supported on this element |
| [T026](#t026) | Warning | p-for without p-key |
| [T027](#t027) | Warning | Static and dynamic attribute conflict |
| [T028](#t028) | Warning | Directive overwrites content |
| [T029](#t029) | Error | Invalid CSS selector |
| [T030](#t030) | Error | Internal parser error |
| [T031](#t031) | Warning | Duplicate attribute |
| [T032](#t032) | Warning | Static _memo attribute |

### Template analysis {#template-analysis-codes}

| Code | Severity | Meaning |
|---|---|---|
| [T100](#t100) | Error | Undefined variable |
| [T101](#t101) | Error or warning | Type mismatch |
| [T102](#t102) | Error | Circular dependency |
| [T103](#t103) | Error | Unresolved import |
| [T104](#t104) | Warning | Email element in a web page |
| [T105](#t105) | Error | Missing is attribute |
| [T106](#t106) | Error | Invalid is attribute |
| [T107](#t107) | Error | Undefined partial alias |
| [T108](#t108) | Error | Partial could not be loaded |
| [T109](#t109) | Warning | No such slot |
| [T110](#t110) | Warning | Name shadows a built-in |
| [T111](#t111) | Warning | Script block has no language |
| [T112](#t112) | Warning | Asset not found |
| [T113](#t113) | Warning | Asset profile not found |
| [T114](#t114) | Error | CSS import failed |
| [T115](#t115) | Error, warning or info | CSS error |
| [T116](#t116) | Error | Go compilation error |
| [T117](#t117) | Error | Collection error |
| [T118](#t118) | Error | Analysis failed |
| [T119](#t119) | Error | $event or $form misused |
| [T120](#t120) | Error | Invalid p-for expression |
| [T121](#t121) | Warning | Loop variable shadows a built-in |
| [T122](#t122) | Error | Condition is not a boolean |
| [T123](#t123) | Error | p-model target is not assignable |
| [T124](#t124) | Error or warning | Binding has the wrong type |
| [T125](#t125) | Warning | Attribute value is not text |
| [T126](#t126) | Warning | Unknown prop |
| [T127](#t127) | Error | Missing required prop |
| [T128](#t128) | Error | Prop has the wrong type |
| [T129](#t129) | Warning | Prop bound twice |
| [T130](#t130) | Error or warning | Invalid prop default |
| [T131](#t131) | Error | Invalid Props struct |
| [T132](#t132) | Error or warning | Invalid query prop |
| [T133](#t133) | Warning | Handler arguments do not match |
| [T134](#t134) | Error | Invalid event handler |
| [T135](#t135) | Error | Built-in function misused |
| [T136](#t136) | Error, info or warning | Translation function misused |
| [T137](#t137) | Error | No such field or method |
| [T138](#t138) | Error | Invalid index |
| [T139](#t139) | Error | Invalid call |
| [T140](#t140) | Error | Logical operator on non-booleans |
| [T141](#t141) | Error | Invalid comparison |
| [T142](#t142) | Error | Invalid arithmetic |
| [T143](#t143) | Warning | ?? operands do not match |
| [T144](#t144) | Error | String concatenation |
| [T145](#t145) | Error or warning | Conversion failed |
| [T146](#t146) | Error or warning | Formatting function misused |
| [T147](#t147) | Error or warning | Action error |
| [T148](#t148) | Error | Entry point not found |
| [T149](#t149) | Error or warning | Client script problem |
| [T150](#t150) | Error | Partial CSS error |
| [T151](#t151) | Error | Computed property not supported |
| [T152](#t152) | Warning | Suggested prop name |
| [T153](#t153) | Warning | Collection provider not registered |
| [T154](#t154) | Warning | sitemap marker ignored |
| [T155](#t155) | Error | Collection source not imported |

### Rendering {#rendering-codes}

| Code | Severity | Meaning |
|---|---|---|
| [R001](#r001) | Error | Render returned an error |
| [R002](#r002) | Error | Partial Render returned an error |
| [R003](#r003) | Warning | Field read through nil |
| [R004](#r004) | Warning | Index into nil |
| [R005](#r005) | Warning | Index out of range |
| [R006](#r006) | Error | <piko:element> has no tag |
| [R007](#r007) | Error | <piko:element> tag not allowed |

## Template expressions {#template-expressions}

Reported while a template's expressions and directives are parsed, before any types are checked.

### T001 Invalid number literal {#t001}

**Error.** A number in a template expression is malformed.

**When it is reported.** The expression parser could not read a numeric literal, for example because it has two decimal points or a stray letter.

**How to fix it.** Write the number as a plain integer or decimal, such as 42 or 3.5.

### T002 Invalid string literal {#t002}

**Error.** A quoted string in a template expression is malformed.

**When it is reported.** The parser could not read a string literal, usually because of an unterminated quote or an invalid escape. It is also reported when an object literal's key is neither an identifier nor a valid string.

**How to fix it.** Close the string with the same quote it opened with, and escape any quote inside it.

### T003 Invalid escape sequence {#t003}

**Error.** A rune literal contains an escape sequence that is not valid.

**When it is reported.** Rune literals such as r'\n' accept the usual Go escapes. Anything else after the backslash is rejected.

**How to fix it.** Use a valid escape, such as \n, \t, \\ or \', or write the character directly.

### T004 Invalid date, time or duration {#t004}

**Error.** A date, time, datetime or duration literal does not match its expected format.

**When it is reported.** Temporal literals are parsed when the template is built. A datetime must be RFC 3339 (YYYY-MM-DDTHH:mm:ssZ), dates and times have their own fixed formats, and durations use Go's duration syntax, such as 1h30m.

**How to fix it.** Rewrite the value in the format the message names.

### T005 Invalid rune literal {#t005}

**Error.** A rune literal does not contain exactly one character.

**When it is reported.** A rune literal, written r'x', holds a single character or a single escape sequence. An empty literal or one with several characters is rejected.

**How to fix it.** Put exactly one character between the quotes, or use a string if you need more.

### T006 Invalid template literal {#t006}

**Error.** A template literal is not enclosed in backticks.

**When it is reported.** Template literals, which interpolate values with ${...}, must start and end with a backtick.

**How to fix it.** Enclose the literal in backticks.

### T007 Unterminated interpolation {#t007}

**Error.** An interpolation is opened but never closed.

**When it is reported.** Text interpolation {{ ... }} needs a closing }}, and a ${...} inside a template literal needs a closing }. The parser reached the end of the text first.

**How to fix it.** Add the missing closing braces.

```text
 ● error [T007]: Unterminated text interpolation: missing '}}'
   --> pages/index.pk:3:7
  |
3 |   <h1>{{ state.Title</h1>
  |       ^^^^^^^^^^^^^
  = docs: https://piko.sh/diagnostics/T007 (or run: piko explain T007)
```

### T008 Unexpected token {#t008}

**Error.** The parser found a token where it expected a value or a different token.

**When it is reported.** This is reported when an expression has an operator or punctuation in a position where a value belongs. It is also reported for composite type arguments such as []T, which template expressions do not support.

**How to fix it.** Check the expression around the highlighted token. For composite type arguments, declare a named type in the script block and use that instead.

### T009 Unexpected end of expression {#t009}

**Error.** The expression ended before it was complete.

**When it is reported.** The parser was still expecting a value, for example after an operator or an opening bracket, when the expression ran out.

**How to fix it.** Finish the expression, or remove the trailing operator.

### T010 Missing identifier {#t010}

**Error.** An identifier is missing after '.', '?.' or '@'.

**When it is reported.** Member access and optional chaining need a field or method name after the dot, and the @ prefix needs a name after it.

**How to fix it.** Add the missing name, or remove the dangling dot.

### T011 Missing operand {#t011}

**Error.** An operator has nothing to operate on.

**When it is reported.** A binary operator needs a value on its right, a unary operator needs a value after it, a p-for loop needs a collection after 'in', and an object key needs a value.

**How to fix it.** Add the missing value.

### T012 Missing closing bracket {#t012}

**Error.** A ']', ')' or '}' is missing.

**When it is reported.** An index expression, parenthesised expression, array literal or object literal was opened but not closed before something else appeared.

**How to fix it.** Add the closing bracket the message names.

### T013 Missing separator {#t013}

**Error.** A ',' or ':' is missing between parts of an expression.

**When it is reported.** Object literals need ':' between key and value and ',' between entries, function calls need ',' between arguments, and a ternary needs ':' between its two outcomes.

**How to fix it.** Add the separator the message names.

### T014 Incomplete expression {#t014}

**Error.** An index, function call, type argument list or ternary is not fully written.

**When it is reported.** Examples are an empty index such as items[], a call with a trailing comma such as f(a,), or a ternary with no value after '?' or ':'.

**How to fix it.** Complete the construct, or remove the stray separator.

### T015 Expression nested too deeply {#t015}

**Error.** An expression is nested more deeply than the parser allows.

**When it is reported.** The parser limits nesting so that malformed or hostile templates cannot exhaust the stack. Real templates stay far below the limit.

**How to fix it.** Move the logic into a function in the script block and call it from the template.

### T016 Feature not allowed here {#t016}

**Error.** The expression uses a feature this context does not allow.

**When it is reported.** Some places accept only a restricted kind of expression. For example, dynamic index access is not allowed where the value must be known when the template is built.

**How to fix it.** Use a simpler expression here, or compute the value in the script block.

### T017 Unreadable expression {#t017}

**Error.** The expression contains characters the lexer cannot read.

**When it is reported.** The lexer turns an expression into tokens before it is parsed. This is reported when it meets a character or sequence that is not part of the expression language.

**How to fix it.** Remove or quote the character the message points at.

### T018 Unexpected text after expression {#t018}

**Error.** Tokens remain after a complete expression.

**When it is reported.** The parser read a whole expression but more text follows it, for example two values with no operator between them.

**How to fix it.** Join the parts with an operator, or remove the extra text.

### T019 Unexpected comma {#t019}

**Error.** A comma is used where it is not an operator.

**When it is reported.** Commas only separate arguments, array items and object entries. The usual cause is a p-for loop with two variables written without parentheses.

**How to fix it.** Enclose loop variables in parentheses, such as (index, item) in collection.

```text
 ● error [T019]: Invalid syntax. A comma is not a valid operator here. If you are trying to write a for-loop, variables must be enclosed in parentheses, e.g., `(index, item) in collection`.
   --> pages/index.pk:4:19
  |
4 |   <li p-for="index, item in state.Items">{{ item }}</li>
  |                   ^
  = docs: https://piko.sh/diagnostics/T019 (or run: piko explain T019)
```

### T020 Conflicting directives {#t020}

**Error.** An element has two directives that cannot be combined.

**When it is reported.** p-text and p-html both replace an element's content, so only one can apply. p-else and p-else-if both close a conditional chain, so an element can carry only one.

**How to fix it.** Remove one of the two directives.

### T021 Duplicate directive {#t021}

**Warning.** The same directive appears more than once on an element.

**When it is reported.** Only one instance of a directive, or of a p-bind for the same attribute, takes effect. The message gives the position of the first.

**How to fix it.** Remove the duplicate, or merge the two expressions.

### T022 Invalid directive value {#t022}

**Error.** A directive's value is empty or not valid.

**When it is reported.** p-ref needs a name that is a valid JavaScript identifier, and p-slot needs a slot name.

**How to fix it.** Give the directive a valid value.

### T023 p-else without p-if {#t023}

**Error.** A p-else or p-else-if does not follow an element with p-if or p-else-if.

**When it is reported.** A conditional chain must be made of sibling elements in order. Whitespace and comments between them are ignored, but any other node breaks the chain.

**How to fix it.** Move the element directly after its p-if, or remove what separates them.

### T024 Directive written before p-for {#t024}

**Warning.** A directive is written before p-for, but p-for is evaluated first.

**When it is reported.** p-for always has the highest precedence, so another directive written before it still runs once per item. The order in the source suggests otherwise.

**How to fix it.** Write p-for before the element's other directives and dynamic attributes.

### T025 Directive not supported on this element {#t025}

**Error.** A directive is used on an element that does not support it.

**When it is reported.** p-model binds a form value, so it only works on <input>, <textarea> and <select>.

**How to fix it.** Move the directive to a supported element.

### T026 p-for without p-key {#t026}

**Warning.** A p-for loop has no p-key.

**When it is reported.** Without a key, the loop index is used to match items between renders. When items are added, removed or reordered, that can reuse the wrong element and lose its state, and it does more work.

**How to fix it.** Add a p-key bound to a value that identifies each item, such as its ID.

```text
 ● warning [T026]: Elements with 'p-for' directives should have a unique 'p-key' binding for efficient updates. The framework will fall back to using the loop index, which may cause performance issues or unexpected behaviour with stateful components.
   --> pages/index.pk:4:7
  |
4 |   <li p-for="item in state.Items">{{ item.Name }}</li>
  |       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  = docs: https://piko.sh/diagnostics/T026 (or run: piko explain T026)
```

### T027 Static and dynamic attribute conflict {#t027}

**Warning.** An attribute is set statically and also by a dynamic binding.

**When it is reported.** The dynamic :attr or p-bind:attr value overwrites the static one, so the static value never shows.

**How to fix it.** Remove the static attribute, or fold it into the binding's expression.

### T028 Directive overwrites content {#t028}

**Warning.** An element has children that p-text or p-html will replace.

**When it is reported.** p-text and p-html set the element's whole content, so its existing children are discarded at runtime and only mislead readers.

**How to fix it.** Remove the children, or drop the directive.

### T029 Invalid CSS selector {#t029}

**Error.** A CSS selector could not be parsed.

**When it is reported.** Selectors used to query the template, for example in tests, are parsed with a CSS selector parser. This is reported when the selector is not valid CSS.

**How to fix it.** Correct the selector.

### T030 Internal parser error {#t030}

**Error.** The expression parser reached a state it should not. This should not normally happen.

**When it is reported.** A defensive check in the parser fired. The expression may be unusual, but the parser should have reported a clearer error.

**How to fix it.** Open an issue on GitHub with the expression that caused it.

### T031 Duplicate attribute {#t031}

**Warning.** The same static attribute is written more than once on an element.

**When it is reported.** HTML keeps the first occurrence and silently discards the rest, so the later value never takes effect.

**How to fix it.** Remove the duplicate.

### T032 Static _memo attribute {#t032}

**Warning.** _memo is written as a static attribute instead of the p-memo directive.

**When it is reported.** A static value never changes, so the memo check always matches and the element and its children stop updating after their first render.

**How to fix it.** Use the p-memo directive with the value that should trigger a re-render, such as p-memo="item".

```text
 ● warning [T032]: The '_memo' attribute is set statically. Its value never changes, so this element and its children are skipped on every re-render after the first. Use the p-memo directive instead, for example 'p-memo="item"'.
   --> partials/row.pk:2:8
  |
2 |   <tr _memo="row">
  |        ^^^^^
  = docs: https://piko.sh/diagnostics/T032 (or run: piko explain T032)
```

## Template analysis {#template-analysis}

Reported while templates are type-checked against their Go code, and while components, props, assets and styles are linked.

### T100 Undefined variable {#t100}

**Error.** A template refers to a name that is not defined.

**When it is reported.** The name is not a field of state or props, a loop variable, an import, or a built-in. When a defined name is close, the message suggests it. It is also reported when the template calls an unexported function from the script block, which templates cannot see.

**How to fix it.** Fix the spelling, define the value in the script block, or export the function by capitalising its name.

```text
 ● error [T100]: Undefined variable: state.Titl. Did you mean 'state.Title'?
   --> pages/index.pk:2:9
  |
2 |   <h1>{{ state.Titl }}</h1>
  |         ^^^^^^^^^^
  = docs: https://piko.sh/diagnostics/T100 (or run: piko explain T100)
```

### T101 Type mismatch {#t101}

**Error or warning.** A value's type does not match the type expected where it is used.

**When it is reported.** It is an error when a function argument has the wrong type or a ternary's two outcomes have incompatible types. It is a warning when an array or object literal mixes element types, or when a field is read through a pointer that may be nil, which is then checked at runtime.

**How to fix it.** Convert the value to the expected type, or make the elements of the literal share one type.

### T102 Circular dependency {#t102}

**Error.** Components, partials or imports depend on each other in a cycle.

**When it is reported.** A component that renders a partial which, directly or through others, renders the first component again can never finish expanding. The message lists the cycle.

**How to fix it.** Break the cycle, for example by moving shared markup into a partial that neither side imports back.

### T103 Unresolved import {#t103}

**Error.** An import path for a component, partial or Go package could not be resolved.

**When it is reported.** The import does not name a file in the project or a package the module can find. CSS @import paths that cannot be found are reported as T114.

**How to fix it.** Check the path against the module name and directory layout, such as myapp/partials/card.pk.

### T104 Email element in a web page {#t104}

**Warning.** A PikoML <pml-*> element is used outside an email template.

**When it is reported.** PikoML components are built for HTML email and render table-based layouts. They belong in .pk files under the emails directory.

**How to fix it.** Use standard HTML elements or Piko components (<piko:*>) in web pages.

### T105 Missing is attribute {#t105}

**Error.** A <piko:partial> or <piko:element> has no is attribute.

**When it is reported.** <piko:partial> needs is set to the alias of an imported partial, and <piko:element> needs is set to the tag it should render.

**How to fix it.** Add the is attribute.

### T106 Invalid is attribute {#t106}

**Error.** The is attribute on <piko:partial> or <piko:element> has a value it cannot use.

**When it is reported.** <piko:partial is> must be a static string, because partials are resolved when the project is built. <piko:element is> cannot target piko:partial, piko:slot or piko:element.

**How to fix it.** Use a static partial alias, or a plain HTML tag name.

### T107 Undefined partial alias {#t107}

**Error.** A <piko:partial is> names an alias that is not imported.

**When it is reported.** Partials are imported in the Go script block under an alias, and is refers to that alias.

**How to fix it.** Import the partial under that alias, or fix the alias in is.

### T108 Partial could not be loaded {#t108}

**Error.** An imported partial's file could not be loaded.

**When it is reported.** The file may be missing, or it may not contain a <template> block.

**How to fix it.** Check that the file exists at the imported path and has a <template> block.

### T109 No such slot {#t109}

**Warning.** Content is passed to a slot the component does not have.

**When it is reported.** The component declares no <piko:slot> with that name, so the content is never rendered. When a declared slot name is close, the message suggests it.

**How to fix it.** Use one of the component's slot names, or add the slot to the component.

### T110 Name shadows a built-in {#t110}

**Warning.** A function or variable has the same name as a built-in or an import alias.

**When it is reported.** A user-defined function that shadows a Piko system symbol or global function, or a local variable that shadows an import alias, makes the template call something other than what a reader expects.

**How to fix it.** Rename the function or variable.

### T111 Script block has no language {#t111}

**Warning.** A <script> block has no lang or type attribute.

**When it is reported.** Piko needs to know whether a script is server-side Go or client-side TypeScript or JavaScript.

**How to fix it.** Add lang="go" (or type="application/x-go") for Go, lang="ts" for TypeScript, or lang="js" for JavaScript.

### T112 Asset not found {#t112}

**Warning.** An asset path could not be resolved, or an image's responsive attributes are incomplete.

**When it is reported.** The path may use a module alias that cannot be expanded or point at a file that does not exist. For responsive images, densities needs width or sizes, and each density must look like x1, x2 or 2x.

**How to fix it.** Check the path and the file, and complete the responsive attributes the message names.

### T113 Asset profile not found {#t113}

**Warning.** An asset profile named on a tag is missing from the configuration or has no settings.

**When it is reported.** The tag is still processed, using only its explicit attributes.

**How to fix it.** Define the profile in the configuration, or remove the reference.

### T114 CSS import failed {#t114}

**Error.** A CSS @import could not be resolved, or imports form a cycle.

**When it is reported.** Piko inlines @import rules when it builds a component's styles. The imported file must exist and must not import, directly or indirectly, the file that imports it.

**How to fix it.** Fix the import path, or break the cycle.

### T115 CSS error {#t115}

**Error, warning or info.** The CSS processor reported a problem in a component's styles.

**When it is reported.** Styles are parsed and minified by esbuild. Its errors, warnings and notes are reported with this code and keep esbuild's severity.

**How to fix it.** Correct the CSS at the reported position.

### T116 Go compilation error {#t116}

**Error.** The Go compiler rejected code generated from a component, and the error is mapped back to the .pk file.

**When it is reported.** Piko compiles each component's script block together with code generated from its template. Errors are reported at the matching place in the .pk source where possible.

**How to fix it.** Fix the Go code at the reported position, as you would for go build.

### T117 Collection error {#t117}

**Error.** The collection system failed while resolving a collection's types.

**When it is reported.** Templates that use p-collection or the collection functions need the collection service to read the collection's items when the project is built.

**How to fix it.** Check the collection's configuration and content, and the message for the underlying cause.

### T118 Analysis failed {#t118}

**Error.** A stage of template analysis failed and could not continue.

**When it is reported.** The build reports what it could and stops processing the affected components. The message names the stage and the underlying error.

**How to fix it.** Fix the cause in the message. If it looks like a bug, open an issue on GitHub with the component that triggers it.

### T119 $event or $form misused {#t119}

**Error.** $event or $form is used outside an event handler, or used in a way it does not support.

**When it is reported.** $event and $form only exist inside p-on and p-event handlers, and are passed whole: reading a property from them in the template is not supported. The browser's own event object is $event, not event.

**How to fix it.** Pass $event or $form whole to the handler, and read their properties inside it.

### T120 Invalid p-for expression {#t120}

**Error.** A p-for expression is not a valid loop, or loops over a type that cannot be iterated.

**When it is reported.** p-for takes the form item in items, or (index, item) in items, over a slice, array, map, string or integer.

**How to fix it.** Write the loop as item in items over an iterable value.

### T121 Loop variable shadows a built-in {#t121}

**Warning.** A p-for loop variable has the same name as a built-in symbol or global function.

**When it is reported.** Inside the loop, the name refers to the loop variable, so the built-in cannot be used there.

**How to fix it.** Rename the loop variable.

### T122 Condition is not a boolean {#t122}

**Error.** A p-if, p-else-if or p-show expression is not a boolean.

**When it is reported.** Conditions must be booleans. Piko does not treat strings, numbers or pointers as true or false implicitly.

**How to fix it.** Compare explicitly, such as len(items) > 0, or use the ~ operator for truthiness, such as ~value.

### T123 p-model target is not assignable {#t123}

**Error.** A p-model expression is not a variable that can be assigned to.

**When it is reported.** p-model writes the input's value back, so it needs a field, such as state.Name, not a call or a computed value.

**How to fix it.** Bind p-model to a field.

### T124 Binding has the wrong type {#t124}

**Error or warning.** A p-class, :class, p-style, p-key or p-context binding has a type it does not expect.

**When it is reported.** It is an error when :class is not a string, slice or map. It is a warning when p-class or p-style has another type, when p-key is a struct, map or slice, or when p-context does not resolve to a string.

**How to fix it.** Give the binding the type the message names.

### T125 Attribute value is not text {#t125}

**Warning.** A dynamic attribute's value has no defined text form.

**When it is reported.** Without String or MarshalText, Go's default formatting is used, which is rarely what you want in HTML. It is also reported for p-timeline outside .pkc files, where it is not supported.

**How to fix it.** Implement fmt.Stringer or encoding.TextMarshaler on the type, convert the value to a string, or use a server. attribute if it is only needed on the server.

### T126 Unknown prop {#t126}

**Warning.** A prop is passed to a component that does not define it.

**When it is reported.** The component's Props struct has no field for the prop, so the value is ignored. When a defined prop is close, the message suggests it.

**How to fix it.** Use the prop name the component defines, or add the field to its Props struct.

### T127 Missing required prop {#t127}

**Error.** A component is used without a prop it requires.

**When it is reported.** The prop's field in the component's Props struct is marked required.

**How to fix it.** Pass the prop, or give it a default in the component.

```text
 ● error [T127]: Missing required prop 'title' for component <card>
   --> pages/index.pk:5:4
  |
5 |   <piko:partial is="card"></piko:partial>
  |    ^^^^^^^^^^^^^^^^^^^^^^^^
  = docs: https://piko.sh/diagnostics/T127 (or run: piko explain T127)
```

### T128 Prop has the wrong type {#t128}

**Error.** The value passed to a prop does not match the prop's declared type.

**When it is reported.** When the value could be converted, the message suggests adding coerce:"true" to the prop's field.

**How to fix it.** Pass a value of the declared type, or add coerce:"true" to the field so Piko converts it.

### T129 Prop bound twice {#t129}

**Warning.** A prop is given both a standard binding and a server-only binding.

**When it is reported.** The server-only binding takes precedence, so the standard binding is ignored.

**How to fix it.** Remove one of the bindings.

### T130 Invalid prop default {#t130}

**Error or warning.** A prop's default value or default factory does not fit the prop's type.

**When it is reported.** It is an error when the default is invalid or a factory returns the wrong type. It is a warning when a default written as text cannot be parsed as the prop's type.

**How to fix it.** Correct the default so it has the prop's type.

### T131 Invalid Props struct {#t131}

**Error.** A component's Props struct is malformed.

**When it is reported.** Two fields map to the same prop name, tags conflict, or an embedded field's props cannot be collected.

**How to fix it.** Give each prop a unique name and consistent tags.

### T132 Invalid query prop {#t132}

**Error or warning.** A prop bound to a query parameter has an invalid query tag or type.

**When it is reported.** It is an error when the query tag is empty or the prop is a slice or map, which query binding does not support. It is a warning when the prop is not a string, because query parameters are strings.

**How to fix it.** Name the parameter in the tag, such as query:"page", and use a string prop or add coerce:"true".

### T133 Handler arguments do not match {#t133}

**Warning.** A handler or action call has the wrong number or types of arguments.

**When it is reported.** The call in the template does not match the exported client function's parameters or the action's input.

**How to fix it.** Pass the arguments the function or action declares.

### T134 Invalid event handler {#t134}

**Error.** An event handler is not a function call, or uses an unknown or incompatible modifier.

**When it is reported.** p-on handlers must call a function or method. The supported modifiers are .prevent, .stop, .once, .self, .passive and .capture, and .passive cannot be combined with .prevent.

**How to fix it.** Write the handler as a call, such as save($event), and use supported modifiers.

### T135 Built-in function misused {#t135}

**Error.** A built-in function such as len, cap, append, min or max is called incorrectly.

**When it is reported.** The call has the wrong number of arguments, or arguments of a type the function does not accept, such as len on a number or min with mixed types.

**How to fix it.** Call the function as Go does.

### T136 Translation function misused {#t136}

**Error, info or warning.** A translation function such as T, LT or LTC is called incorrectly, or names a missing key.

**When it is reported.** It is an error when the function has no arguments or an argument is not a string. When the key is missing from a locale's translations, it is reported as info if a fallback is given and as a warning if not.

**How to fix it.** Pass the key as a string, and add the key to the translations or give a fallback.

### T137 No such field or method {#t137}

**Error.** A field, method or package member does not exist.

**When it is reported.** The type, or the package, has no member with that name. When a member is close, the message suggests it, and it points out len() when a property such as .length is used.

**How to fix it.** Use a member the type defines.

### T138 Invalid index {#t138}

**Error.** A value is indexed when it cannot be, or with an index of the wrong type.

**When it is reported.** Only slices, arrays, maps and strings can be indexed, each with its own index type. Several values in brackets are only valid as type arguments on a generic call.

**How to fix it.** Index a collection with the right index type.

### T139 Invalid call {#t139}

**Error.** A value that is not a function is called, or a function is called with the wrong number of arguments.

**When it is reported.** The expression before the parentheses is not a function or method, or the call passes more or fewer arguments than it declares.

**How to fix it.** Call a function, with the arguments it declares.

### T140 Logical operator on non-booleans {#t140}

**Error.** An operand of &&, || or ! is not a boolean, or the operands of || are incompatible.

**When it is reported.** Piko does not treat non-boolean values as true or false implicitly.

**How to fix it.** Compare explicitly, or use the ~ operator for truthiness, such as ~value && ~other.

### T141 Invalid comparison {#t141}

**Error.** The two sides of a comparison cannot be compared.

**When it is reported.** Ordering operators (<, >, <=, >=) need ordered types of the same kind. Equality is checked strictly: a named type and a plain literal of its underlying type, such as a named string type and a string, cannot be compared directly.

**How to fix it.** Convert one side so both have the same type, or compare a field that has a comparable type.

```text
 ● error [T141]: Invalid operation: cannot strictly compare type 'app.Level' to 'string'
   --> pages/index.pk:6:14
  |
6 |   <span p-if="state.Level == 'error'">!</span>
  |              ^^^^^^^^^^^^^^^^^^^^^^^^^^^
  = docs: https://piko.sh/diagnostics/T141 (or run: piko explain T141)
```

### T142 Invalid arithmetic {#t142}

**Error.** An arithmetic operator is used with types it is not defined for.

**When it is reported.** Both operands of +, -, *, / and % must be compatible numbers, and unary - needs a number. Money only adds and subtracts Money, multiplies and divides by plain numbers, and has no %.

**How to fix it.** Convert the operands to compatible types, or use the Money methods.

### T143 ?? operands do not match {#t143}

**Warning.** The two sides of ?? have incompatible types.

**When it is reported.** a ?? b gives b when a is nil, so b should have a's type.

**How to fix it.** Give the fallback the same type as the value it replaces.

### T144 String concatenation {#t144}

Not currently reported. The number stays assigned so it is never reused.

**Error.** A string concatenation involves a value that cannot be turned into a string.

**When it is reported.** This code is not currently reported: such concatenations are reported as invalid arithmetic (T142). The number stays assigned so it is never reused.

**How to fix it.** Nothing to fix.

### T145 Conversion failed {#t145}

**Error or warning.** A value could not be converted to the type it is needed as.

**When it is reported.** It is an error when a conversion function has the wrong number of arguments or cannot convert the type. It is a warning when a static prop value cannot be converted to the prop's type, in which case the value is ignored.

**How to fix it.** Pass a value that can be converted, or change the prop's type.

### T146 Formatting function misused {#t146}

**Error or warning.** F or LF is called incorrectly, or with a value it may not format.

**When it is reported.** It is an error when F or LF does not get exactly one argument. It is a warning when the argument is not a number (int, float, Decimal, BigInt or Money), a time (time.Time or DateTime) or a string.

**How to fix it.** Pass one value of a supported type.

### T147 Action error {#t147}

**Error or warning.** An action could not be discovered, or a template calls it incorrectly.

**When it is reported.** It is an error when the actions directory cannot be read or parsed, or an action has no Call method. It is a warning when a template calls an action that does not exist, or passes a field its input does not have, misses a required field, or passes the wrong type.

**How to fix it.** Fix the action in the actions directory, or the call in the template.

### T148 Entry point not found {#t148}

**Error.** An entry point of the component dependency graph could not be resolved.

**When it is reported.** The build starts from the project's pages and other entry points. One of them could not be found.

**How to fix it.** Check the path in the message, and the project's configuration.

### T149 Client script problem {#t149}

**Error or warning.** A client script cannot be parsed, an event handler is missing from it, or an imported partial is never used.

**When it is reported.** It is an error when a client script block cannot be parsed, or when a p-on handler names a function the client script does not export. It is a warning when a partial is imported but neither rendered nor used in reloadPartial or reloadGroup.

**How to fix it.** Fix the script's syntax, export the handler from the client script, or remove the unused import.

### T150 Partial CSS error {#t150}

**Error.** A partial's CSS could not be processed when the partial was expanded.

**When it is reported.** When a partial is expanded into its parent, its styles are processed with the parent's. The message names the partial and the cause.

**How to fix it.** Fix the CSS in the named partial.

### T151 Computed property not supported {#t151}

**Error.** Member access with [...] is used where it is not supported.

**When it is reported.** Templates cannot access a package's members, or a variable's fields, by a computed name in brackets.

**How to fix it.** Use dot access, such as value.Field, or index a map instead.

### T152 Suggested prop name {#t152}

Not currently reported. The number stays assigned so it is never reused.

**Warning.** An unknown prop name is close to one the component defines.

**When it is reported.** This code is not currently reported: suggestions appear in the unknown prop message (T126) instead. The number stays assigned so it is never reused.

**How to fix it.** Nothing to fix.

### T153 Collection provider not registered {#t153}

**Warning.** A p-collection names a provider that is not registered.

**When it is reported.** Pages that use the provider will not be available until it is registered.

**How to fix it.** Register the provider as the message describes, or fix its name.

### T154 sitemap marker ignored {#t154}

**Warning.** The sitemap marker is on a tag that cannot add an image to the sitemap.

**When it is reported.** The build-only sitemap marker is honoured only on <piko:img> and <piko:picture>.

**How to fix it.** Move the marker to a <piko:img> or <piko:picture>, or remove it.

### T155 Collection source not imported {#t155}

**Error.** A p-collection-source alias matches no import in the Go script block.

**When it is reported.** p-collection-source names the import alias of the module that holds the content. Piko does not fall back to the page's local content.

**How to fix it.** Import the content module under that alias, or correct the alias.

```text
 ● error [T155]: p-collection-source "docs" does not match the name of any import in the Go script block; import the content module under that name
   --> pages/docs/{slug}.pk:1:1
  |
1 | <template p-collection="docs" p-provider="markdown" p-collection-source="docs">
  | ^^^^^^^^^^
  = docs: https://piko.sh/diagnostics/T155 (or run: piko explain T155)
```

## Rendering {#rendering}

Reported by generated code while a page renders. They appear in the server log rather than in build output, and the page still renders: a nil read or an out-of-range index gives a zero value instead of a panic.

### R001 Render returned an error {#r001}

**Error.** A page or component's Render function returned an error.

**When it is reported.** The error is recorded and logged with the page's other render diagnostics, and the request is answered with an error page.

**How to fix it.** Handle the cause in Render, or return a piko error with a safe message for the visitor.

### R002 Partial Render returned an error {#r002}

**Error.** A partial's Render function returned an error.

**When it is reported.** The partial's error is recorded with the page's render diagnostics and logged.

**How to fix it.** Handle the cause in the partial's Render function.

### R003 Field read through nil {#r003}

**Warning.** A template read a field through a pointer that was nil.

**When it is reported.** Templates guard field access through pointers that may be nil. Instead of panicking, the expression gives the field's zero value and this warning is recorded. The build reports the risk ahead of time as T101.

**How to fix it.** Check the value before using it, such as p-if="state.User != nil", or make sure Render sets it.

### R004 Index into nil {#r004}

**Warning.** A template indexed a slice or map that was nil.

**When it is reported.** Instead of panicking, the expression gives the zero value and this warning is recorded.

**How to fix it.** Make sure the slice or map is set before it is indexed, or check it first.

### R005 Index out of range {#r005}

**Warning.** A template indexed a slice outside its length.

**When it is reported.** Instead of panicking, the expression gives the zero value and this warning is recorded.

**How to fix it.** Check the length first, such as p-if="len(state.Items) > 2", or loop over the slice with p-for.

### R006 <piko:element> has no tag {#r006}

**Error.** A <piko:element :is> expression resolved to an empty tag name.

**When it is reported.** The element cannot be rendered without a tag, so it is skipped.

**How to fix it.** Make sure the is expression always gives a tag name.

### R007 <piko:element> tag not allowed {#r007}

**Error.** A <piko:element :is> expression resolved to a reserved tag.

**When it is reported.** <piko:element> cannot render piko:partial, piko:slot or piko:element, so the element is skipped.

**How to fix it.** Use a plain HTML tag name, or use <piko:partial> for partials.
