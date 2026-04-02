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

package io.politepixels.gen.pk;

import com.intellij.lexer.FlexLexer;
import com.intellij.psi.TokenType;
import com.intellij.psi.tree.IElementType;
import io.politepixels.piko.pk.PKTokenTypes;

%%

%public
%class PKLexer
%unicode
%implements FlexLexer
%function advance
%type IElementType
%eof{
  return;
%eof}

%state IN_STRUCTURAL_TAG_ATTRS
%state IN_STRUCTURAL_TAG_CLOSING
%state IN_GO_SCRIPT_CONTENT
%state IN_JS_SCRIPT_CONTENT
%state IN_CSS_STYLE_CONTENT
%state IN_I18N_CONTENT
%state IN_TEMPLATE_CONTENT
%state IN_HTML_TAG
%state IN_HTML_ATTR_VALUE_DQ
%state IN_HTML_ATTR_VALUE_SQ
%state IN_INTERPOLATION
%state IN_DIRECTIVE_VALUE_DQ
%state IN_DIRECTIVE_VALUE_SQ
%state IN_EXPR_STRING_DOUBLE
%state IN_EXPR_STRING_SINGLE
%state IN_EXPR_STRING_BACKTICK

%{
  private IElementType currentStructuralTagType = null;
  private String pendingScriptLang = "go";
  private boolean pendingDirective = false;
  private boolean seenTagName = false;
  private int stringReturnState = IN_INTERPOLATION;
%}

WHITE_SPACE           = [ \t\r\n\f]+
TEMPLATE_TAG_OPEN     = "<template"
SCRIPT_TAG_OPEN       = "<script"
STYLE_TAG_OPEN        = "<style"
I18N_TAG_OPEN         = "<i18n"
TEMPLATE_TAG_CLOSE    = "</template"
SCRIPT_TAG_CLOSE      = "</script"
STYLE_TAG_CLOSE       = "</style"
I18N_TAG_CLOSE        = "</i18n"
IDENTIFIER            = [a-zA-Z_][a-zA-Z0-9_]*
HTML_TAG_NAME_CHARS   = [a-zA-Z][a-zA-Z0-9\-]*
NUMBER                = [0-9]+(\.[0-9]+)?
BUILTIN               = "len"|"cap"|"make"|"new"|"append"|"copy"|"delete"|"panic"|"recover"|"print"|"println"

%%

<YYINITIAL> {
    {WHITE_SPACE}             { return TokenType.WHITE_SPACE; }
    "<!--" ~"-->"             { return PKTokenTypes.HTML_COMMENT; }

    {TEMPLATE_TAG_OPEN} {
        currentStructuralTagType = PKTokenTypes.TEMPLATE_TAG_START;
        yybegin(IN_STRUCTURAL_TAG_ATTRS);
        return PKTokenTypes.TEMPLATE_TAG_START;
    }
    {SCRIPT_TAG_OPEN} {
        currentStructuralTagType = PKTokenTypes.SCRIPT_TAG_START;
        pendingScriptLang = "go";
        yybegin(IN_STRUCTURAL_TAG_ATTRS);
        return PKTokenTypes.SCRIPT_TAG_START;
    }
    {STYLE_TAG_OPEN} {
        currentStructuralTagType = PKTokenTypes.STYLE_TAG_START;
        yybegin(IN_STRUCTURAL_TAG_ATTRS);
        return PKTokenTypes.STYLE_TAG_START;
    }
    {I18N_TAG_OPEN} {
        currentStructuralTagType = PKTokenTypes.I18N_TAG_START;
        yybegin(IN_STRUCTURAL_TAG_ATTRS);
        return PKTokenTypes.I18N_TAG_START;
    }
    [^]                       { return TokenType.BAD_CHARACTER; }
}

<IN_STRUCTURAL_TAG_ATTRS> {
    {WHITE_SPACE}             { return TokenType.WHITE_SPACE; }

    ">" {
        if (currentStructuralTagType == PKTokenTypes.TEMPLATE_TAG_START) {
            yybegin(IN_TEMPLATE_CONTENT);
        } else if (currentStructuralTagType == PKTokenTypes.SCRIPT_TAG_START) {
            if (pendingScriptLang.equals("js") || pendingScriptLang.equals("ts") ||
                pendingScriptLang.equals("javascript") || pendingScriptLang.equals("typescript")) {
                yybegin(IN_JS_SCRIPT_CONTENT);
            } else {
                yybegin(IN_GO_SCRIPT_CONTENT);
            }
        } else if (currentStructuralTagType == PKTokenTypes.STYLE_TAG_START) {
            yybegin(IN_CSS_STYLE_CONTENT);
        } else if (currentStructuralTagType == PKTokenTypes.I18N_TAG_START) {
            yybegin(IN_I18N_CONTENT);
        } else {
            yybegin(YYINITIAL);
        }
        return PKTokenTypes.TAG_END_GT;
    }

    "lang" {WHITE_SPACE}* "=" {WHITE_SPACE}* \" [^\"]* \" {
        String matched = yytext().toString();
        int start = matched.indexOf('"') + 1;
        int end = matched.lastIndexOf('"');
        if (start > 0 && end > start) {
            pendingScriptLang = matched.substring(start, end).toLowerCase();
        }
        return PKTokenTypes.ATTR_NAME;
    }
    "lang" {WHITE_SPACE}* "=" {WHITE_SPACE}* \' [^\']* \' {
        String matched = yytext().toString();
        int start = matched.indexOf('\'') + 1;
        int end = matched.lastIndexOf('\'');
        if (start > 0 && end > start) {
            pendingScriptLang = matched.substring(start, end).toLowerCase();
        }
        return PKTokenTypes.ATTR_NAME;
    }

    "type" {WHITE_SPACE}* "=" {WHITE_SPACE}* \" [^\"]* \" {
        String matched = yytext().toString();
        int start = matched.indexOf('"') + 1;
        int end = matched.lastIndexOf('"');
        if (start > 0 && end > start) {
            String typeValue = matched.substring(start, end).toLowerCase();
            if (typeValue.equals("application/x-go") || typeValue.equals("text/x-go")) {
                pendingScriptLang = "go";
            } else if (typeValue.equals("application/javascript") || typeValue.equals("text/javascript")) {
                pendingScriptLang = "js";
            } else if (typeValue.equals("application/typescript") || typeValue.equals("text/typescript")) {
                pendingScriptLang = "ts";
            }
        }
        return PKTokenTypes.ATTR_NAME;
    }
    "type" {WHITE_SPACE}* "=" {WHITE_SPACE}* \' [^\']* \' {
        String matched = yytext().toString();
        int start = matched.indexOf('\'') + 1;
        int end = matched.lastIndexOf('\'');
        if (start > 0 && end > start) {
            String typeValue = matched.substring(start, end).toLowerCase();
            if (typeValue.equals("application/x-go") || typeValue.equals("text/x-go")) {
                pendingScriptLang = "go";
            } else if (typeValue.equals("application/javascript") || typeValue.equals("text/javascript")) {
                pendingScriptLang = "js";
            } else if (typeValue.equals("application/typescript") || typeValue.equals("text/typescript")) {
                pendingScriptLang = "ts";
            }
        }
        return PKTokenTypes.ATTR_NAME;
    }

    [a-zA-Z][a-zA-Z0-9\-_:]*  { return PKTokenTypes.ATTR_NAME; }
    "="                        { return PKTokenTypes.ATTR_EQ; }
    \" [^\"]* \"               { return PKTokenTypes.ATTR_VALUE; }
    \' [^\']* \'               { return PKTokenTypes.ATTR_VALUE; }
    [^>]                       { return TokenType.BAD_CHARACTER; }
}

<IN_TEMPLATE_CONTENT> {
    {TEMPLATE_TAG_CLOSE} {
        yybegin(IN_STRUCTURAL_TAG_CLOSING);
        return PKTokenTypes.TEMPLATE_TAG_END;
    }

    "<!--" ~"-->"             { return PKTokenTypes.HTML_COMMENT; }

    "{{" {
        yybegin(IN_INTERPOLATION);
        return PKTokenTypes.INTERPOLATION_OPEN;
    }

    "</" {
        seenTagName = false;
        yybegin(IN_HTML_TAG);
        return PKTokenTypes.HTML_END_TAG_OPEN;
    }

    "<" {
        seenTagName = false;
        yybegin(IN_HTML_TAG);
        return PKTokenTypes.HTML_TAG_OPEN;
    }

    [^<{]+                    { return PKTokenTypes.TEXT_CONTENT; }
    "{" / [^{]                { return PKTokenTypes.TEXT_CONTENT; }
    "{"                       { return PKTokenTypes.TEXT_CONTENT; }
}

<IN_HTML_TAG> {
    {WHITE_SPACE}             { return TokenType.WHITE_SPACE; }

    "/>" {
        yybegin(IN_TEMPLATE_CONTENT);
        return PKTokenTypes.HTML_TAG_SELF_CLOSE;
    }

    ">" {
        yybegin(IN_TEMPLATE_CONTENT);
        return PKTokenTypes.HTML_TAG_CLOSE;
    }

    "piko:" {HTML_TAG_NAME_CHARS} {
        seenTagName = true;
        return PKTokenTypes.PIKO_TAG_NAME;
    }

    "p-bind:" {HTML_TAG_NAME_CHARS} ("." [a-zA-Z]+)* {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_NAME;
    }
    "p-on:" {HTML_TAG_NAME_CHARS} ("." [a-zA-Z]+)* {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_NAME;
    }
    "p-event:" {HTML_TAG_NAME_CHARS} ("." [a-zA-Z]+)* {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_NAME;
    }

    "p-if" | "p-else-if" | "p-else" | "p-for" | "p-show" {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_NAME;
    }

    "p-text" | "p-html" | "p-model" {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_NAME;
    }

    "p-class" | "p-style" {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_NAME;
    }

    "p-ref" | "p-slot" | "p-key" | "p-context" | "p-scaffold" {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_NAME;
    }

    ":" [a-zA-Z_][a-zA-Z0-9._\-]* {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_BIND;
    }

    "@" [a-zA-Z][a-zA-Z0-9\-]* ("." [a-zA-Z]+)* {
        pendingDirective = true;
        return PKTokenTypes.DIRECTIVE_EVENT;
    }

    {HTML_TAG_NAME_CHARS} {
        if (seenTagName) {
            pendingDirective = false;
            return PKTokenTypes.HTML_ATTR_NAME;
        } else {
            seenTagName = true;
            return PKTokenTypes.HTML_TAG_NAME;
        }
    }

    "=" {
        return PKTokenTypes.HTML_ATTR_EQ;
    }

    \" {
        if (pendingDirective) {
            pendingDirective = false;
            yybegin(IN_DIRECTIVE_VALUE_DQ);
            return PKTokenTypes.HTML_ATTR_QUOTE;
        } else {
            yybegin(IN_HTML_ATTR_VALUE_DQ);
            return PKTokenTypes.HTML_ATTR_QUOTE;
        }
    }
    \' {
        if (pendingDirective) {
            pendingDirective = false;
            yybegin(IN_DIRECTIVE_VALUE_SQ);
            return PKTokenTypes.HTML_ATTR_QUOTE;
        } else {
            yybegin(IN_HTML_ATTR_VALUE_SQ);
            return PKTokenTypes.HTML_ATTR_QUOTE;
        }
    }

    [a-zA-Z_][a-zA-Z0-9_\-:]* {
        pendingDirective = false;
        return PKTokenTypes.HTML_ATTR_NAME;
    }

    [^]                       { return TokenType.BAD_CHARACTER; }
}

<IN_HTML_ATTR_VALUE_DQ> {
    \" {
        yybegin(IN_HTML_TAG);
        return PKTokenTypes.HTML_ATTR_QUOTE;
    }
    [^\"]+ {
        return PKTokenTypes.HTML_ATTR_VALUE;
    }
}

<IN_HTML_ATTR_VALUE_SQ> {
    \' {
        yybegin(IN_HTML_TAG);
        return PKTokenTypes.HTML_ATTR_QUOTE;
    }
    [^\']+ {
        return PKTokenTypes.HTML_ATTR_VALUE;
    }
}

<IN_INTERPOLATION> {
    {WHITE_SPACE}             { return TokenType.WHITE_SPACE; }

    "}}" {
        yybegin(IN_TEMPLATE_CONTENT);
        return PKTokenTypes.INTERPOLATION_CLOSE;
    }

    "true" | "false" | "nil"  { return PKTokenTypes.EXPR_BOOLEAN; }
    {NUMBER}                  { return PKTokenTypes.EXPR_NUMBER; }

    \" {
        stringReturnState = IN_INTERPOLATION;
        yybegin(IN_EXPR_STRING_DOUBLE);
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }
    \' {
        stringReturnState = IN_INTERPOLATION;
        yybegin(IN_EXPR_STRING_SINGLE);
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }
    ` {
        stringReturnState = IN_INTERPOLATION;
        yybegin(IN_EXPR_STRING_BACKTICK);
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }

    "==" | "!=" | "<=" | ">=" { return PKTokenTypes.EXPR_OP_COMPARISON; }
    "<" | ">"                 { return PKTokenTypes.EXPR_OP_COMPARISON; }
    "&&" | "||"               { return PKTokenTypes.EXPR_OP_LOGICAL; }
    "!"                       { return PKTokenTypes.EXPR_OP_LOGICAL; }
    "+" | "-" | "*" | "/" | "%" { return PKTokenTypes.EXPR_OP_ARITHMETIC; }
    "."                       { return PKTokenTypes.EXPR_OP_DOT; }

    "("                       { return PKTokenTypes.EXPR_PAREN_OPEN; }
    ")"                       { return PKTokenTypes.EXPR_PAREN_CLOSE; }
    "["                       { return PKTokenTypes.EXPR_BRACKET_OPEN; }
    "]"                       { return PKTokenTypes.EXPR_BRACKET_CLOSE; }
    "{"                       { return PKTokenTypes.EXPR_BRACE_OPEN; }
    "}" / [^}]                { return PKTokenTypes.EXPR_BRACE_CLOSE; }
    ","                       { return PKTokenTypes.EXPR_COMMA; }
    ":"                       { return PKTokenTypes.EXPR_COLON; }

    {BUILTIN} / {WHITE_SPACE}* "(" { return PKTokenTypes.EXPR_BUILTIN; }
    {BUILTIN}                 { return PKTokenTypes.EXPR_BUILTIN; }
    "props" | "state" | "partial" { return PKTokenTypes.EXPR_CONTEXT_VAR; }
    {IDENTIFIER} / {WHITE_SPACE}* "(" { return PKTokenTypes.EXPR_FUNCTION_NAME; }
    {IDENTIFIER}              { return PKTokenTypes.EXPR_IDENTIFIER; }

    [^]                       { return TokenType.BAD_CHARACTER; }
}

<IN_DIRECTIVE_VALUE_DQ> {
    {WHITE_SPACE}             { return TokenType.WHITE_SPACE; }

    \" {
        yybegin(IN_HTML_TAG);
        return PKTokenTypes.HTML_ATTR_QUOTE;
    }

    "true" | "false" | "nil"  { return PKTokenTypes.EXPR_BOOLEAN; }
    {NUMBER}                  { return PKTokenTypes.EXPR_NUMBER; }

    \' {
        stringReturnState = IN_DIRECTIVE_VALUE_DQ;
        yybegin(IN_EXPR_STRING_SINGLE);
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }
    ` {
        stringReturnState = IN_DIRECTIVE_VALUE_DQ;
        yybegin(IN_EXPR_STRING_BACKTICK);
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }

    "==" | "!=" | "<=" | ">=" { return PKTokenTypes.EXPR_OP_COMPARISON; }
    "<" | ">"                 { return PKTokenTypes.EXPR_OP_COMPARISON; }
    "&&" | "||"               { return PKTokenTypes.EXPR_OP_LOGICAL; }
    "!"                       { return PKTokenTypes.EXPR_OP_LOGICAL; }
    "+" | "-" | "*" | "/" | "%" { return PKTokenTypes.EXPR_OP_ARITHMETIC; }
    "."                       { return PKTokenTypes.EXPR_OP_DOT; }

    "("                       { return PKTokenTypes.EXPR_PAREN_OPEN; }
    ")"                       { return PKTokenTypes.EXPR_PAREN_CLOSE; }
    "["                       { return PKTokenTypes.EXPR_BRACKET_OPEN; }
    "]"                       { return PKTokenTypes.EXPR_BRACKET_CLOSE; }
    "{"                       { return PKTokenTypes.EXPR_BRACE_OPEN; }
    "}"                       { return PKTokenTypes.EXPR_BRACE_CLOSE; }
    ","                       { return PKTokenTypes.EXPR_COMMA; }
    ":"                       { return PKTokenTypes.EXPR_COLON; }

    {BUILTIN} / {WHITE_SPACE}* "(" { return PKTokenTypes.EXPR_BUILTIN; }
    {BUILTIN}                 { return PKTokenTypes.EXPR_BUILTIN; }
    "props" | "state" | "partial" { return PKTokenTypes.EXPR_CONTEXT_VAR; }
    {IDENTIFIER} / {WHITE_SPACE}* "(" { return PKTokenTypes.EXPR_FUNCTION_NAME; }
    {IDENTIFIER}              { return PKTokenTypes.EXPR_IDENTIFIER; }

    [^]                       { return TokenType.BAD_CHARACTER; }
}

<IN_DIRECTIVE_VALUE_SQ> {
    {WHITE_SPACE}             { return TokenType.WHITE_SPACE; }

    \' {
        yybegin(IN_HTML_TAG);
        return PKTokenTypes.HTML_ATTR_QUOTE;
    }

    "true" | "false" | "nil"  { return PKTokenTypes.EXPR_BOOLEAN; }
    {NUMBER}                  { return PKTokenTypes.EXPR_NUMBER; }

    \" {
        stringReturnState = IN_DIRECTIVE_VALUE_SQ;
        yybegin(IN_EXPR_STRING_DOUBLE);
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }
    ` {
        stringReturnState = IN_DIRECTIVE_VALUE_SQ;
        yybegin(IN_EXPR_STRING_BACKTICK);
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }

    "==" | "!=" | "<=" | ">=" { return PKTokenTypes.EXPR_OP_COMPARISON; }
    "<" | ">"                 { return PKTokenTypes.EXPR_OP_COMPARISON; }
    "&&" | "||"               { return PKTokenTypes.EXPR_OP_LOGICAL; }
    "!"                       { return PKTokenTypes.EXPR_OP_LOGICAL; }
    "+" | "-" | "*" | "/" | "%" { return PKTokenTypes.EXPR_OP_ARITHMETIC; }
    "."                       { return PKTokenTypes.EXPR_OP_DOT; }

    "("                       { return PKTokenTypes.EXPR_PAREN_OPEN; }
    ")"                       { return PKTokenTypes.EXPR_PAREN_CLOSE; }
    "["                       { return PKTokenTypes.EXPR_BRACKET_OPEN; }
    "]"                       { return PKTokenTypes.EXPR_BRACKET_CLOSE; }
    "{"                       { return PKTokenTypes.EXPR_BRACE_OPEN; }
    "}"                       { return PKTokenTypes.EXPR_BRACE_CLOSE; }
    ","                       { return PKTokenTypes.EXPR_COMMA; }
    ":"                       { return PKTokenTypes.EXPR_COLON; }

    {BUILTIN} / {WHITE_SPACE}* "(" { return PKTokenTypes.EXPR_BUILTIN; }
    {BUILTIN}                 { return PKTokenTypes.EXPR_BUILTIN; }
    "props" | "state" | "partial" { return PKTokenTypes.EXPR_CONTEXT_VAR; }
    {IDENTIFIER} / {WHITE_SPACE}* "(" { return PKTokenTypes.EXPR_FUNCTION_NAME; }
    {IDENTIFIER}              { return PKTokenTypes.EXPR_IDENTIFIER; }

    [^]                       { return TokenType.BAD_CHARACTER; }
}

<IN_EXPR_STRING_DOUBLE> {
    \\[\"\\nrtbf0]            { return PKTokenTypes.EXPR_ESCAPE; }

    \" {
        if (stringReturnState == IN_INTERPOLATION) {
            yybegin(IN_INTERPOLATION);
        } else if (stringReturnState == IN_DIRECTIVE_VALUE_DQ) {
            yybegin(IN_DIRECTIVE_VALUE_DQ);
        } else if (stringReturnState == IN_DIRECTIVE_VALUE_SQ) {
            yybegin(IN_DIRECTIVE_VALUE_SQ);
        } else {
            yybegin(IN_INTERPOLATION);
        }
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }

    [^\"\\]+                  { return PKTokenTypes.EXPR_STRING; }
    \\                        { return PKTokenTypes.EXPR_STRING; }
}

<IN_EXPR_STRING_SINGLE> {
    \\[\'\\nrtbf0]            { return PKTokenTypes.EXPR_ESCAPE; }

    \' {
        if (stringReturnState == IN_INTERPOLATION) {
            yybegin(IN_INTERPOLATION);
        } else if (stringReturnState == IN_DIRECTIVE_VALUE_DQ) {
            yybegin(IN_DIRECTIVE_VALUE_DQ);
        } else if (stringReturnState == IN_DIRECTIVE_VALUE_SQ) {
            yybegin(IN_DIRECTIVE_VALUE_SQ);
        } else {
            yybegin(IN_INTERPOLATION);
        }
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }

    [^\'\\]+                  { return PKTokenTypes.EXPR_STRING; }
    \\                        { return PKTokenTypes.EXPR_STRING; }
}

<IN_EXPR_STRING_BACKTICK> {
    "${"                      { return PKTokenTypes.TEMPLATE_INTERP_OPEN; }

    ` {
        if (stringReturnState == IN_INTERPOLATION) {
            yybegin(IN_INTERPOLATION);
        } else if (stringReturnState == IN_DIRECTIVE_VALUE_DQ) {
            yybegin(IN_DIRECTIVE_VALUE_DQ);
        } else if (stringReturnState == IN_DIRECTIVE_VALUE_SQ) {
            yybegin(IN_DIRECTIVE_VALUE_SQ);
        } else {
            yybegin(IN_INTERPOLATION);
        }
        return PKTokenTypes.EXPR_STRING_QUOTE;
    }

    [^`$]+                    { return PKTokenTypes.EXPR_STRING; }
    "$" / [^{]                { return PKTokenTypes.EXPR_STRING; }
    "$"                       { return PKTokenTypes.EXPR_STRING; }
}

<IN_GO_SCRIPT_CONTENT> {
    {SCRIPT_TAG_CLOSE} {
        yybegin(IN_STRUCTURAL_TAG_CLOSING);
        return PKTokenTypes.SCRIPT_TAG_END;
    }
    [^<]+                     { return PKTokenTypes.GO_SCRIPT_CONTENT; }
    "<"                       { return PKTokenTypes.GO_SCRIPT_CONTENT; }
}

<IN_JS_SCRIPT_CONTENT> {
    {SCRIPT_TAG_CLOSE} {
        yybegin(IN_STRUCTURAL_TAG_CLOSING);
        return PKTokenTypes.SCRIPT_TAG_END;
    }
    [^<]+                     { return PKTokenTypes.JS_SCRIPT_CONTENT; }
    "<"                       { return PKTokenTypes.JS_SCRIPT_CONTENT; }
}

<IN_CSS_STYLE_CONTENT> {
    {STYLE_TAG_CLOSE} {
        yybegin(IN_STRUCTURAL_TAG_CLOSING);
        return PKTokenTypes.STYLE_TAG_END;
    }
    [^<]+                     { return PKTokenTypes.CSS_STYLE_CONTENT; }
    "<"                       { return PKTokenTypes.CSS_STYLE_CONTENT; }
}

<IN_I18N_CONTENT> {
    {I18N_TAG_CLOSE} {
        yybegin(IN_STRUCTURAL_TAG_CLOSING);
        return PKTokenTypes.I18N_TAG_END;
    }
    [^<]+                     { return PKTokenTypes.I18N_CONTENT; }
    "<"                       { return PKTokenTypes.I18N_CONTENT; }
}

<IN_STRUCTURAL_TAG_CLOSING> {
    {WHITE_SPACE}             { return TokenType.WHITE_SPACE; }
    ">" {
        yybegin(YYINITIAL);
        return PKTokenTypes.TAG_END_GT;
    }
    [^] {
        yybegin(YYINITIAL);
        return TokenType.BAD_CHARACTER;
    }
}

. { return TokenType.BAD_CHARACTER; }
