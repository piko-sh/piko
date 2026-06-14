/**
 * Tree-sitter grammar for Piko single-file components (.pk / .pkc).
 *
 * Copyright 2026 PolitePixels Limited
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * This project stands against fascism, authoritarianism, and all forms of
 * oppression. We built this to empower people, not to enable those who would
 * strip others of their rights and dignity.
 */

module.exports = grammar({
  name: 'piko',

  extras: ($) => [/\s+/, $.comment],

  externals: ($) => [$.script_raw_text, $.style_raw_text, $.i18n_raw_text],

  rules: {
    document: ($) => repeat($._content),

    _content: ($) =>
      choice(
        $.script_element,
        $.style_element,
        $.i18n_element,
        $.template_element,
        $.doctype,
        $.start_tag,
        $.self_closing_tag,
        $.end_tag,
        $.interpolation,
        $.text,
      ),

    script_element: ($) =>
      seq(
        alias($.script_start_tag, $.start_tag),
        optional(alias($.script_raw_text, $.raw_text)),
        alias($._script_end_tag, $.end_tag),
      ),
    script_start_tag: ($) =>
      seq('<', alias($._script, $.tag_name), repeat($.attribute), '>'),
    _script_end_tag: ($) => seq('</', alias($._script, $.tag_name), '>'),

    style_element: ($) =>
      seq(
        alias($.style_start_tag, $.start_tag),
        optional(alias($.style_raw_text, $.raw_text)),
        alias($._style_end_tag, $.end_tag),
      ),
    style_start_tag: ($) =>
      seq('<', alias($._style, $.tag_name), repeat($.attribute), '>'),
    _style_end_tag: ($) => seq('</', alias($._style, $.tag_name), '>'),

    i18n_element: ($) =>
      seq(
        alias($.i18n_start_tag, $.start_tag),
        optional(alias($.i18n_raw_text, $.raw_text)),
        alias($._i18n_end_tag, $.end_tag),
      ),
    i18n_start_tag: ($) =>
      seq('<', alias($._i18n, $.tag_name), repeat($.attribute), '>'),
    _i18n_end_tag: ($) => seq('</', alias($._i18n, $.tag_name), '>'),

    _script: (_) => 'script',
    _style: (_) => 'style',
    _i18n: (_) => 'i18n',

    template_element: ($) =>
      seq(
        alias($.template_start_tag, $.start_tag),
        repeat($._content),
        alias($._template_end_tag, $.end_tag),
      ),
    template_start_tag: ($) =>
      seq('<', alias($._template, $.tag_name), repeat($.attribute), '>'),
    _template_end_tag: ($) => seq('</', alias($._template, $.tag_name), '>'),
    _template: (_) => 'template',

    start_tag: ($) =>
      seq('<', field('name', $.tag_name), repeat($.attribute), '>'),
    self_closing_tag: ($) =>
      seq('<', field('name', $.tag_name), repeat($.attribute), '/>'),
    end_tag: ($) => seq('</', field('name', $.tag_name), '>'),

    doctype: (_) => token(prec(1, /<![a-zA-Z][^>]*>/)),

    tag_name: (_) => /[a-zA-Z][a-zA-Z0-9:_-]*/,

    text: (_) => token(repeat1(choice(/[^<{]+/, '{', /<[^a-zA-Z!/]/))),

    interpolation: ($) =>
      seq(
        token(prec(2, '{{')),
        optional(alias($._interpolation_text, $.expression)),
        token(prec(2, '}}')),
      ),
    _interpolation_text: (_) => token(prec(-1, /([^}]|}[^}])+/)),

    attribute: ($) => choice($.directive, $._plain_attribute),

    _plain_attribute: ($) =>
      seq(
        field('name', $.attribute_name),
        optional(seq('=', field('value', $.quoted_attribute_value))),
      ),

    attribute_name: (_) => /[a-zA-Z_][a-zA-Z0-9_:.-]*/,

    quoted_attribute_value: ($) =>
      choice(
        seq('"', optional(alias($._double_quoted_text, $.attribute_value)), '"'),
        seq("'", optional(alias($._single_quoted_text, $.attribute_value)), "'"),
      ),
    _double_quoted_text: (_) => /[^"]+/,
    _single_quoted_text: (_) => /[^']+/,

    directive: ($) =>
      seq(
        field('name', $.directive_name),
        optional(seq('=', field('value', $.directive_value))),
      ),

    directive_name: (_) =>
      token(
        prec(
          1,
          choice(
            /p-[a-zA-Z][a-zA-Z0-9-]*(:[a-zA-Z][a-zA-Z0-9-]*)?(\.[a-zA-Z]+)*/,
            /:[a-zA-Z_][a-zA-Z0-9._-]*/,
            /@[a-zA-Z][a-zA-Z0-9-]*(\.[a-zA-Z]+)*/,
          ),
        ),
      ),

    directive_value: ($) =>
      choice(
        seq('"', optional(alias($._double_quoted_expr, $.expression)), '"'),
        seq("'", optional(alias($._single_quoted_expr, $.expression)), "'"),
      ),
    _double_quoted_expr: (_) => /[^"]+/,
    _single_quoted_expr: (_) => /[^']+/,

    comment: (_) => token(seq('<!--', /([^-]|-[^-]|--[^>])*/, '-->')),
  },
});
