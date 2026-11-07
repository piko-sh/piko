#include "tree_sitter/parser.h"

#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic ignored "-Wmissing-field-initializers"
#endif

#define LANGUAGE_VERSION 14
#define STATE_COUNT 110
#define LARGE_STATE_COUNT 2
#define SYMBOL_COUNT 72
#define ALIAS_COUNT 1
#define TOKEN_COUNT 33
#define EXTERNAL_TOKEN_COUNT 6
#define FIELD_COUNT 2
#define MAX_ALIAS_SEQUENCE_LENGTH 4
#define PRODUCTION_ID_COUNT 9

enum ts_symbol_identifiers {
  anon_sym_LT = 1,
  anon_sym_GT = 2,
  anon_sym_SLASH_GT = 3,
  anon_sym_LT_SLASH = 4,
  sym__script = 5,
  sym__style = 6,
  sym__i18n = 7,
  sym__template = 8,
  anon_sym_script = 9,
  anon_sym_style = 10,
  anon_sym_i18n = 11,
  sym_doctype = 12,
  sym_tag_name = 13,
  sym_text = 14,
  anon_sym_LBRACE_LBRACE = 15,
  anon_sym_RBRACE_RBRACE = 16,
  sym__interpolation_text = 17,
  anon_sym_EQ = 18,
  sym_attribute_name = 19,
  anon_sym_DQUOTE = 20,
  anon_sym_SQUOTE = 21,
  aux_sym__double_quoted_text_token1 = 22,
  aux_sym__single_quoted_text_token1 = 23,
  sym__unquoted_value = 24,
  sym_directive_name = 25,
  sym_comment = 26,
  sym__script_raw_chunk = 27,
  sym__style_raw_chunk = 28,
  sym__i18n_raw_chunk = 29,
  sym__script_inner_close = 30,
  sym__style_inner_close = 31,
  sym__i18n_inner_close = 32,
  sym_document = 33,
  sym__content = 34,
  sym_script_element = 35,
  sym_script_start_tag = 36,
  sym_script_self_closing_tag = 37,
  sym__script_end_tag = 38,
  aux_sym__script_raw_text = 39,
  sym_style_element = 40,
  sym_style_start_tag = 41,
  sym_style_self_closing_tag = 42,
  sym__style_end_tag = 43,
  aux_sym__style_raw_text = 44,
  sym_i18n_element = 45,
  sym_i18n_start_tag = 46,
  sym_i18n_self_closing_tag = 47,
  sym__i18n_end_tag = 48,
  aux_sym__i18n_raw_text = 49,
  sym__script_close_name = 50,
  sym__style_close_name = 51,
  sym__i18n_close_name = 52,
  sym_template_element = 53,
  sym_template_start_tag = 54,
  sym_template_self_closing_tag = 55,
  sym__template_end_tag = 56,
  sym_start_tag = 57,
  sym_self_closing_tag = 58,
  sym_end_tag = 59,
  sym_interpolation = 60,
  sym_attribute = 61,
  sym__plain_attribute = 62,
  sym_quoted_attribute_value = 63,
  sym__double_quoted_text = 64,
  sym__single_quoted_text = 65,
  sym_directive = 66,
  sym_directive_value = 67,
  sym__double_quoted_expr = 68,
  sym__single_quoted_expr = 69,
  aux_sym_document_repeat1 = 70,
  aux_sym_script_start_tag_repeat1 = 71,
  alias_sym_raw_text = 72,
};

static const char * const ts_symbol_names[] = {
  [ts_builtin_sym_end] = "end",
  [anon_sym_LT] = "<",
  [anon_sym_GT] = ">",
  [anon_sym_SLASH_GT] = "/>",
  [anon_sym_LT_SLASH] = "</",
  [sym__script] = "tag_name",
  [sym__style] = "tag_name",
  [sym__i18n] = "tag_name",
  [sym__template] = "tag_name",
  [anon_sym_script] = "script",
  [anon_sym_style] = "style",
  [anon_sym_i18n] = "i18n",
  [sym_doctype] = "doctype",
  [sym_tag_name] = "tag_name",
  [sym_text] = "text",
  [anon_sym_LBRACE_LBRACE] = "{{",
  [anon_sym_RBRACE_RBRACE] = "}}",
  [sym__interpolation_text] = "expression",
  [anon_sym_EQ] = "=",
  [sym_attribute_name] = "attribute_name",
  [anon_sym_DQUOTE] = "\"",
  [anon_sym_SQUOTE] = "'",
  [aux_sym__double_quoted_text_token1] = "_double_quoted_text_token1",
  [aux_sym__single_quoted_text_token1] = "_single_quoted_text_token1",
  [sym__unquoted_value] = "attribute_value",
  [sym_directive_name] = "directive_name",
  [sym_comment] = "comment",
  [sym__script_raw_chunk] = "_script_raw_chunk",
  [sym__style_raw_chunk] = "_style_raw_chunk",
  [sym__i18n_raw_chunk] = "_i18n_raw_chunk",
  [sym__script_inner_close] = "_script_inner_close",
  [sym__style_inner_close] = "_style_inner_close",
  [sym__i18n_inner_close] = "_i18n_inner_close",
  [sym_document] = "document",
  [sym__content] = "_content",
  [sym_script_element] = "script_element",
  [sym_script_start_tag] = "start_tag",
  [sym_script_self_closing_tag] = "self_closing_tag",
  [sym__script_end_tag] = "end_tag",
  [aux_sym__script_raw_text] = "_script_raw_text",
  [sym_style_element] = "style_element",
  [sym_style_start_tag] = "start_tag",
  [sym_style_self_closing_tag] = "self_closing_tag",
  [sym__style_end_tag] = "end_tag",
  [aux_sym__style_raw_text] = "_style_raw_text",
  [sym_i18n_element] = "i18n_element",
  [sym_i18n_start_tag] = "start_tag",
  [sym_i18n_self_closing_tag] = "self_closing_tag",
  [sym__i18n_end_tag] = "end_tag",
  [aux_sym__i18n_raw_text] = "_i18n_raw_text",
  [sym__script_close_name] = "tag_name",
  [sym__style_close_name] = "tag_name",
  [sym__i18n_close_name] = "tag_name",
  [sym_template_element] = "template_element",
  [sym_template_start_tag] = "start_tag",
  [sym_template_self_closing_tag] = "self_closing_tag",
  [sym__template_end_tag] = "end_tag",
  [sym_start_tag] = "start_tag",
  [sym_self_closing_tag] = "self_closing_tag",
  [sym_end_tag] = "end_tag",
  [sym_interpolation] = "interpolation",
  [sym_attribute] = "attribute",
  [sym__plain_attribute] = "_plain_attribute",
  [sym_quoted_attribute_value] = "quoted_attribute_value",
  [sym__double_quoted_text] = "attribute_value",
  [sym__single_quoted_text] = "attribute_value",
  [sym_directive] = "directive",
  [sym_directive_value] = "directive_value",
  [sym__double_quoted_expr] = "expression",
  [sym__single_quoted_expr] = "expression",
  [aux_sym_document_repeat1] = "document_repeat1",
  [aux_sym_script_start_tag_repeat1] = "script_start_tag_repeat1",
  [alias_sym_raw_text] = "raw_text",
};

static const TSSymbol ts_symbol_map[] = {
  [ts_builtin_sym_end] = ts_builtin_sym_end,
  [anon_sym_LT] = anon_sym_LT,
  [anon_sym_GT] = anon_sym_GT,
  [anon_sym_SLASH_GT] = anon_sym_SLASH_GT,
  [anon_sym_LT_SLASH] = anon_sym_LT_SLASH,
  [sym__script] = sym_tag_name,
  [sym__style] = sym_tag_name,
  [sym__i18n] = sym_tag_name,
  [sym__template] = sym_tag_name,
  [anon_sym_script] = anon_sym_script,
  [anon_sym_style] = anon_sym_style,
  [anon_sym_i18n] = anon_sym_i18n,
  [sym_doctype] = sym_doctype,
  [sym_tag_name] = sym_tag_name,
  [sym_text] = sym_text,
  [anon_sym_LBRACE_LBRACE] = anon_sym_LBRACE_LBRACE,
  [anon_sym_RBRACE_RBRACE] = anon_sym_RBRACE_RBRACE,
  [sym__interpolation_text] = sym__interpolation_text,
  [anon_sym_EQ] = anon_sym_EQ,
  [sym_attribute_name] = sym_attribute_name,
  [anon_sym_DQUOTE] = anon_sym_DQUOTE,
  [anon_sym_SQUOTE] = anon_sym_SQUOTE,
  [aux_sym__double_quoted_text_token1] = aux_sym__double_quoted_text_token1,
  [aux_sym__single_quoted_text_token1] = aux_sym__single_quoted_text_token1,
  [sym__unquoted_value] = sym__unquoted_value,
  [sym_directive_name] = sym_directive_name,
  [sym_comment] = sym_comment,
  [sym__script_raw_chunk] = sym__script_raw_chunk,
  [sym__style_raw_chunk] = sym__style_raw_chunk,
  [sym__i18n_raw_chunk] = sym__i18n_raw_chunk,
  [sym__script_inner_close] = sym__script_inner_close,
  [sym__style_inner_close] = sym__style_inner_close,
  [sym__i18n_inner_close] = sym__i18n_inner_close,
  [sym_document] = sym_document,
  [sym__content] = sym__content,
  [sym_script_element] = sym_script_element,
  [sym_script_start_tag] = sym_start_tag,
  [sym_script_self_closing_tag] = sym_self_closing_tag,
  [sym__script_end_tag] = sym_end_tag,
  [aux_sym__script_raw_text] = aux_sym__script_raw_text,
  [sym_style_element] = sym_style_element,
  [sym_style_start_tag] = sym_start_tag,
  [sym_style_self_closing_tag] = sym_self_closing_tag,
  [sym__style_end_tag] = sym_end_tag,
  [aux_sym__style_raw_text] = aux_sym__style_raw_text,
  [sym_i18n_element] = sym_i18n_element,
  [sym_i18n_start_tag] = sym_start_tag,
  [sym_i18n_self_closing_tag] = sym_self_closing_tag,
  [sym__i18n_end_tag] = sym_end_tag,
  [aux_sym__i18n_raw_text] = aux_sym__i18n_raw_text,
  [sym__script_close_name] = sym_tag_name,
  [sym__style_close_name] = sym_tag_name,
  [sym__i18n_close_name] = sym_tag_name,
  [sym_template_element] = sym_template_element,
  [sym_template_start_tag] = sym_start_tag,
  [sym_template_self_closing_tag] = sym_self_closing_tag,
  [sym__template_end_tag] = sym_end_tag,
  [sym_start_tag] = sym_start_tag,
  [sym_self_closing_tag] = sym_self_closing_tag,
  [sym_end_tag] = sym_end_tag,
  [sym_interpolation] = sym_interpolation,
  [sym_attribute] = sym_attribute,
  [sym__plain_attribute] = sym__plain_attribute,
  [sym_quoted_attribute_value] = sym_quoted_attribute_value,
  [sym__double_quoted_text] = sym__unquoted_value,
  [sym__single_quoted_text] = sym__unquoted_value,
  [sym_directive] = sym_directive,
  [sym_directive_value] = sym_directive_value,
  [sym__double_quoted_expr] = sym__interpolation_text,
  [sym__single_quoted_expr] = sym__interpolation_text,
  [aux_sym_document_repeat1] = aux_sym_document_repeat1,
  [aux_sym_script_start_tag_repeat1] = aux_sym_script_start_tag_repeat1,
  [alias_sym_raw_text] = alias_sym_raw_text,
};

static const TSSymbolMetadata ts_symbol_metadata[] = {
  [ts_builtin_sym_end] = {
    .visible = false,
    .named = true,
  },
  [anon_sym_LT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SLASH_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_SLASH] = {
    .visible = true,
    .named = false,
  },
  [sym__script] = {
    .visible = true,
    .named = true,
  },
  [sym__style] = {
    .visible = true,
    .named = true,
  },
  [sym__i18n] = {
    .visible = true,
    .named = true,
  },
  [sym__template] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_script] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_style] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_i18n] = {
    .visible = true,
    .named = false,
  },
  [sym_doctype] = {
    .visible = true,
    .named = true,
  },
  [sym_tag_name] = {
    .visible = true,
    .named = true,
  },
  [sym_text] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_LBRACE_LBRACE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RBRACE_RBRACE] = {
    .visible = true,
    .named = false,
  },
  [sym__interpolation_text] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_EQ] = {
    .visible = true,
    .named = false,
  },
  [sym_attribute_name] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_DQUOTE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SQUOTE] = {
    .visible = true,
    .named = false,
  },
  [aux_sym__double_quoted_text_token1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym__single_quoted_text_token1] = {
    .visible = false,
    .named = false,
  },
  [sym__unquoted_value] = {
    .visible = true,
    .named = true,
  },
  [sym_directive_name] = {
    .visible = true,
    .named = true,
  },
  [sym_comment] = {
    .visible = true,
    .named = true,
  },
  [sym__script_raw_chunk] = {
    .visible = false,
    .named = true,
  },
  [sym__style_raw_chunk] = {
    .visible = false,
    .named = true,
  },
  [sym__i18n_raw_chunk] = {
    .visible = false,
    .named = true,
  },
  [sym__script_inner_close] = {
    .visible = false,
    .named = true,
  },
  [sym__style_inner_close] = {
    .visible = false,
    .named = true,
  },
  [sym__i18n_inner_close] = {
    .visible = false,
    .named = true,
  },
  [sym_document] = {
    .visible = true,
    .named = true,
  },
  [sym__content] = {
    .visible = false,
    .named = true,
  },
  [sym_script_element] = {
    .visible = true,
    .named = true,
  },
  [sym_script_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_script_self_closing_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__script_end_tag] = {
    .visible = true,
    .named = true,
  },
  [aux_sym__script_raw_text] = {
    .visible = false,
    .named = false,
  },
  [sym_style_element] = {
    .visible = true,
    .named = true,
  },
  [sym_style_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_style_self_closing_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__style_end_tag] = {
    .visible = true,
    .named = true,
  },
  [aux_sym__style_raw_text] = {
    .visible = false,
    .named = false,
  },
  [sym_i18n_element] = {
    .visible = true,
    .named = true,
  },
  [sym_i18n_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_i18n_self_closing_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__i18n_end_tag] = {
    .visible = true,
    .named = true,
  },
  [aux_sym__i18n_raw_text] = {
    .visible = false,
    .named = false,
  },
  [sym__script_close_name] = {
    .visible = true,
    .named = true,
  },
  [sym__style_close_name] = {
    .visible = true,
    .named = true,
  },
  [sym__i18n_close_name] = {
    .visible = true,
    .named = true,
  },
  [sym_template_element] = {
    .visible = true,
    .named = true,
  },
  [sym_template_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_template_self_closing_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__template_end_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_self_closing_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_end_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_interpolation] = {
    .visible = true,
    .named = true,
  },
  [sym_attribute] = {
    .visible = true,
    .named = true,
  },
  [sym__plain_attribute] = {
    .visible = false,
    .named = true,
  },
  [sym_quoted_attribute_value] = {
    .visible = true,
    .named = true,
  },
  [sym__double_quoted_text] = {
    .visible = true,
    .named = true,
  },
  [sym__single_quoted_text] = {
    .visible = true,
    .named = true,
  },
  [sym_directive] = {
    .visible = true,
    .named = true,
  },
  [sym_directive_value] = {
    .visible = true,
    .named = true,
  },
  [sym__double_quoted_expr] = {
    .visible = true,
    .named = true,
  },
  [sym__single_quoted_expr] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_document_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_script_start_tag_repeat1] = {
    .visible = false,
    .named = false,
  },
  [alias_sym_raw_text] = {
    .visible = true,
    .named = true,
  },
};

enum ts_field_identifiers {
  field_name = 1,
  field_value = 2,
};

static const char * const ts_field_names[] = {
  [0] = NULL,
  [field_name] = "name",
  [field_value] = "value",
};

static const TSFieldMapSlice ts_field_map_slices[PRODUCTION_ID_COUNT] = {
  [1] = {.index = 0, .length = 1},
  [2] = {.index = 1, .length = 1},
  [3] = {.index = 2, .length = 1},
  [4] = {.index = 3, .length = 2},
  [5] = {.index = 5, .length = 1},
  [6] = {.index = 5, .length = 1},
  [7] = {.index = 6, .length = 2},
};

static const TSFieldMapEntry ts_field_map_entries[] = {
  [0] =
    {field_name, 1, .inherited = true},
  [1] =
    {field_name, 1},
  [2] =
    {field_name, 0},
  [3] =
    {field_name, 0, .inherited = true},
    {field_value, 0, .inherited = true},
  [5] =
    {field_name, 2, .inherited = true},
  [6] =
    {field_name, 0},
    {field_value, 2},
};

static const TSSymbol ts_alias_sequences[PRODUCTION_ID_COUNT][MAX_ALIAS_SEQUENCE_LENGTH] = {
  [0] = {0},
  [5] = {
    [1] = alias_sym_raw_text,
  },
  [8] = {
    [0] = sym__interpolation_text,
  },
};

static const uint16_t ts_non_terminal_alias_map[] = {
  aux_sym__script_raw_text, 2,
    aux_sym__script_raw_text,
    alias_sym_raw_text,
  aux_sym__style_raw_text, 2,
    aux_sym__style_raw_text,
    alias_sym_raw_text,
  aux_sym__i18n_raw_text, 2,
    aux_sym__i18n_raw_text,
    alias_sym_raw_text,
  0,
};

static const TSStateId ts_primary_state_ids[STATE_COUNT] = {
  [0] = 0,
  [1] = 1,
  [2] = 2,
  [3] = 3,
  [4] = 4,
  [5] = 5,
  [6] = 6,
  [7] = 7,
  [8] = 8,
  [9] = 9,
  [10] = 10,
  [11] = 11,
  [12] = 12,
  [13] = 13,
  [14] = 14,
  [15] = 15,
  [16] = 16,
  [17] = 17,
  [18] = 18,
  [19] = 19,
  [20] = 20,
  [21] = 21,
  [22] = 22,
  [23] = 23,
  [24] = 24,
  [25] = 25,
  [26] = 26,
  [27] = 27,
  [28] = 28,
  [29] = 29,
  [30] = 30,
  [31] = 31,
  [32] = 32,
  [33] = 33,
  [34] = 34,
  [35] = 35,
  [36] = 36,
  [37] = 37,
  [38] = 38,
  [39] = 39,
  [40] = 40,
  [41] = 41,
  [42] = 42,
  [43] = 43,
  [44] = 44,
  [45] = 45,
  [46] = 46,
  [47] = 47,
  [48] = 48,
  [49] = 49,
  [50] = 50,
  [51] = 51,
  [52] = 52,
  [53] = 53,
  [54] = 54,
  [55] = 55,
  [56] = 56,
  [57] = 57,
  [58] = 58,
  [59] = 59,
  [60] = 60,
  [61] = 61,
  [62] = 62,
  [63] = 63,
  [64] = 64,
  [65] = 65,
  [66] = 66,
  [67] = 67,
  [68] = 68,
  [69] = 69,
  [70] = 70,
  [71] = 71,
  [72] = 72,
  [73] = 73,
  [74] = 74,
  [75] = 75,
  [76] = 76,
  [77] = 77,
  [78] = 78,
  [79] = 79,
  [80] = 80,
  [81] = 81,
  [82] = 82,
  [83] = 83,
  [84] = 84,
  [85] = 85,
  [86] = 86,
  [87] = 87,
  [88] = 88,
  [89] = 89,
  [90] = 90,
  [91] = 91,
  [92] = 92,
  [93] = 93,
  [94] = 94,
  [95] = 95,
  [96] = 96,
  [97] = 97,
  [98] = 98,
  [99] = 99,
  [100] = 100,
  [101] = 101,
  [102] = 102,
  [103] = 103,
  [104] = 104,
  [105] = 105,
  [106] = 106,
  [107] = 107,
  [108] = 108,
  [109] = 109,
};

static bool ts_lex(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      if (eof) ADVANCE(38);
      ADVANCE_MAP(
        '"', 103,
        '\'', 104,
        '/', 20,
        ':', 34,
        '<', 39,
        '=', 99,
        '>', 41,
        'I', 56,
        'S', 70,
        'i', 57,
        'p', 55,
        's', 60,
        '{', 31,
        '}', 32,
        'T', 71,
        't', 71,
      );
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(0);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 1:
      if (lookahead == '!') ADVANCE(5);
      END_STATE();
    case 2:
      if (lookahead == '"') ADVANCE(103);
      if (lookahead == '\'') ADVANCE(104);
      if (lookahead == '<') ADVANCE(1);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(2);
      if (lookahead != 0 &&
          (lookahead < '<' || '>' < lookahead) &&
          lookahead != '`') ADVANCE(121);
      END_STATE();
    case 3:
      if (lookahead == '"') ADVANCE(103);
      if (lookahead == '<') ADVANCE(105);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(111);
      if (lookahead != 0) ADVANCE(112);
      END_STATE();
    case 4:
      if (lookahead == '\'') ADVANCE(104);
      if (lookahead == '<') ADVANCE(113);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(119);
      if (lookahead != 0) ADVANCE(120);
      END_STATE();
    case 5:
      if (lookahead == '-') ADVANCE(7);
      END_STATE();
    case 6:
      if (lookahead == '-') ADVANCE(7);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(21);
      END_STATE();
    case 7:
      if (lookahead == '-') ADVANCE(10);
      END_STATE();
    case 8:
      if (lookahead == '-') ADVANCE(8);
      if (lookahead == '>') ADVANCE(124);
      if (lookahead != 0) ADVANCE(10);
      END_STATE();
    case 9:
      if (lookahead == '-') ADVANCE(8);
      if (lookahead != 0) ADVANCE(10);
      END_STATE();
    case 10:
      if (lookahead == '-') ADVANCE(9);
      if (lookahead != 0) ADVANCE(10);
      END_STATE();
    case 11:
      if (lookahead == '-') ADVANCE(93);
      if (lookahead == '}') ADVANCE(10);
      if (lookahead != 0) ADVANCE(95);
      END_STATE();
    case 12:
      if (lookahead == '/') ADVANCE(20);
      if (lookahead == ':') ADVANCE(34);
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == '=') ADVANCE(99);
      if (lookahead == '>') ADVANCE(41);
      if (lookahead == 'p') ADVANCE(100);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(12);
      if (('@' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(102);
      END_STATE();
    case 13:
      if (lookahead == '1') ADVANCE(14);
      END_STATE();
    case 14:
      if (lookahead == '8') ADVANCE(26);
      END_STATE();
    case 15:
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == 'i') ADVANCE(13);
      if (lookahead == 's') ADVANCE(22);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(15);
      END_STATE();
    case 16:
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == 'I' ||
          lookahead == 'i') ADVANCE(56);
      if (lookahead == 'S' ||
          lookahead == 's') ADVANCE(70);
      if (lookahead == 'T' ||
          lookahead == 't') ADVANCE(71);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(16);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 17:
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == 'T' ||
          lookahead == 't') ADVANCE(71);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(17);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 18:
      if (lookahead == '<') ADVANCE(1);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(18);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 19:
      if (lookahead == '<') ADVANCE(91);
      if (lookahead == '}') ADVANCE(33);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(97);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 20:
      if (lookahead == '>') ADVANCE(42);
      END_STATE();
    case 21:
      if (lookahead == '>') ADVANCE(54);
      if (lookahead != 0) ADVANCE(21);
      END_STATE();
    case 22:
      if (lookahead == 'c') ADVANCE(28);
      if (lookahead == 't') ADVANCE(30);
      END_STATE();
    case 23:
      if (lookahead == 'e') ADVANCE(50);
      END_STATE();
    case 24:
      if (lookahead == 'i') ADVANCE(27);
      END_STATE();
    case 25:
      if (lookahead == 'l') ADVANCE(23);
      END_STATE();
    case 26:
      if (lookahead == 'n') ADVANCE(52);
      END_STATE();
    case 27:
      if (lookahead == 'p') ADVANCE(29);
      END_STATE();
    case 28:
      if (lookahead == 'r') ADVANCE(24);
      END_STATE();
    case 29:
      if (lookahead == 't') ADVANCE(48);
      END_STATE();
    case 30:
      if (lookahead == 'y') ADVANCE(25);
      END_STATE();
    case 31:
      if (lookahead == '{') ADVANCE(89);
      END_STATE();
    case 32:
      if (lookahead == '}') ADVANCE(90);
      END_STATE();
    case 33:
      if (lookahead == '}') ADVANCE(90);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 34:
      if (('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(123);
      END_STATE();
    case 35:
      if (lookahead != 0 &&
          lookahead != '!' &&
          lookahead != '/' &&
          (lookahead < 'A' || 'Z' < lookahead) &&
          (lookahead < 'a' || 'z' < lookahead)) ADVANCE(88);
      END_STATE();
    case 36:
      if (lookahead != 0 &&
          lookahead != '}') ADVANCE(98);
      END_STATE();
    case 37:
      if (eof) ADVANCE(38);
      if (lookahead == '<') ADVANCE(40);
      if (lookahead == '{') ADVANCE(87);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(37);
      if (lookahead != 0) ADVANCE(88);
      END_STATE();
    case 38:
      ACCEPT_TOKEN(ts_builtin_sym_end);
      END_STATE();
    case 39:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '!') ADVANCE(6);
      if (lookahead == '/') ADVANCE(43);
      END_STATE();
    case 40:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '!') ADVANCE(6);
      if (lookahead == '/') ADVANCE(43);
      if (lookahead != 0 &&
          (lookahead < 'A' || 'Z' < lookahead) &&
          (lookahead < 'a' || 'z' < lookahead)) ADVANCE(88);
      END_STATE();
    case 41:
      ACCEPT_TOKEN(anon_sym_GT);
      END_STATE();
    case 42:
      ACCEPT_TOKEN(anon_sym_SLASH_GT);
      END_STATE();
    case 43:
      ACCEPT_TOKEN(anon_sym_LT_SLASH);
      END_STATE();
    case 44:
      ACCEPT_TOKEN(sym__script);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 45:
      ACCEPT_TOKEN(sym__style);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 46:
      ACCEPT_TOKEN(sym__i18n);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 47:
      ACCEPT_TOKEN(sym__template);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 48:
      ACCEPT_TOKEN(anon_sym_script);
      END_STATE();
    case 49:
      ACCEPT_TOKEN(anon_sym_script);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 50:
      ACCEPT_TOKEN(anon_sym_style);
      END_STATE();
    case 51:
      ACCEPT_TOKEN(anon_sym_style);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 52:
      ACCEPT_TOKEN(anon_sym_i18n);
      END_STATE();
    case 53:
      ACCEPT_TOKEN(anon_sym_i18n);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 54:
      ACCEPT_TOKEN(sym_doctype);
      END_STATE();
    case 55:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '-') ADVANCE(85);
      if (('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 56:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '1') ADVANCE(58);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 57:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '1') ADVANCE(59);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 58:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '8') ADVANCE(78);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 59:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '8') ADVANCE(64);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 60:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'C') ADVANCE(81);
      if (lookahead == 'T') ADVANCE(84);
      if (lookahead == 'c') ADVANCE(66);
      if (lookahead == 't') ADVANCE(68);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 61:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'E') ADVANCE(45);
      if (lookahead == 'e') ADVANCE(51);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 62:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'I') ADVANCE(79);
      if (lookahead == 'i') ADVANCE(65);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 63:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'L') ADVANCE(72);
      if (lookahead == 'l') ADVANCE(61);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 64:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'N') ADVANCE(46);
      if (lookahead == 'n') ADVANCE(53);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 65:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'P') ADVANCE(82);
      if (lookahead == 'p') ADVANCE(67);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 66:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'R') ADVANCE(74);
      if (lookahead == 'r') ADVANCE(62);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 67:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'T') ADVANCE(44);
      if (lookahead == 't') ADVANCE(49);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 68:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'Y') ADVANCE(76);
      if (lookahead == 'y') ADVANCE(63);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 69:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'A' ||
          lookahead == 'a') ADVANCE(83);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('B' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('b' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 70:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'C' ||
          lookahead == 'c') ADVANCE(81);
      if (lookahead == 'T' ||
          lookahead == 't') ADVANCE(84);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 71:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'E' ||
          lookahead == 'e') ADVANCE(77);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 72:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'E' ||
          lookahead == 'e') ADVANCE(45);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 73:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'E' ||
          lookahead == 'e') ADVANCE(47);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 74:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'I' ||
          lookahead == 'i') ADVANCE(79);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 75:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'L' ||
          lookahead == 'l') ADVANCE(69);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 76:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'L' ||
          lookahead == 'l') ADVANCE(72);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 77:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'M' ||
          lookahead == 'm') ADVANCE(80);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 78:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'N' ||
          lookahead == 'n') ADVANCE(46);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 79:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'P' ||
          lookahead == 'p') ADVANCE(82);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 80:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'P' ||
          lookahead == 'p') ADVANCE(75);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 81:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'R' ||
          lookahead == 'r') ADVANCE(74);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 82:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'T' ||
          lookahead == 't') ADVANCE(44);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 83:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'T' ||
          lookahead == 't') ADVANCE(73);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 84:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'Y' ||
          lookahead == 'y') ADVANCE(76);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 85:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          lookahead == '_') ADVANCE(86);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(122);
      END_STATE();
    case 86:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(86);
      END_STATE();
    case 87:
      ACCEPT_TOKEN(sym_text);
      if (lookahead == '<') ADVANCE(35);
      if (lookahead == '{') ADVANCE(89);
      if (lookahead != 0) ADVANCE(88);
      END_STATE();
    case 88:
      ACCEPT_TOKEN(sym_text);
      if (lookahead == '<') ADVANCE(35);
      if (lookahead == '{') ADVANCE(88);
      if (lookahead != 0) ADVANCE(88);
      END_STATE();
    case 89:
      ACCEPT_TOKEN(anon_sym_LBRACE_LBRACE);
      END_STATE();
    case 90:
      ACCEPT_TOKEN(anon_sym_RBRACE_RBRACE);
      END_STATE();
    case 91:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '!') ADVANCE(96);
      if (lookahead == '}') ADVANCE(36);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 92:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(92);
      if (lookahead == '>') ADVANCE(124);
      if (lookahead == '}') ADVANCE(11);
      if (lookahead != 0) ADVANCE(95);
      END_STATE();
    case 93:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(92);
      if (lookahead == '}') ADVANCE(11);
      if (lookahead != 0) ADVANCE(95);
      END_STATE();
    case 94:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(95);
      if (lookahead == '}') ADVANCE(36);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 95:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(93);
      if (lookahead == '}') ADVANCE(11);
      if (lookahead != 0) ADVANCE(95);
      END_STATE();
    case 96:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(94);
      if (lookahead == '}') ADVANCE(36);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 97:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '<') ADVANCE(91);
      if (lookahead == '}') ADVANCE(33);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(97);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 98:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '}') ADVANCE(36);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 99:
      ACCEPT_TOKEN(anon_sym_EQ);
      END_STATE();
    case 100:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-') ADVANCE(101);
      if (lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('@' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(102);
      END_STATE();
    case 101:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          lookahead == '@' ||
          lookahead == '_') ADVANCE(102);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(123);
      END_STATE();
    case 102:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('@' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(102);
      END_STATE();
    case 103:
      ACCEPT_TOKEN(anon_sym_DQUOTE);
      END_STATE();
    case 104:
      ACCEPT_TOKEN(anon_sym_SQUOTE);
      END_STATE();
    case 105:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '!') ADVANCE(110);
      if (lookahead != 0 &&
          lookahead != '!' &&
          lookahead != '"') ADVANCE(112);
      END_STATE();
    case 106:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '"') ADVANCE(10);
      if (lookahead == '-') ADVANCE(106);
      if (lookahead == '>') ADVANCE(112);
      if (lookahead != 0) ADVANCE(108);
      END_STATE();
    case 107:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '"') ADVANCE(10);
      if (lookahead == '-') ADVANCE(106);
      if (lookahead != 0) ADVANCE(108);
      END_STATE();
    case 108:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '"') ADVANCE(10);
      if (lookahead == '-') ADVANCE(107);
      if (lookahead != 0) ADVANCE(108);
      END_STATE();
    case 109:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '-') ADVANCE(108);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(112);
      END_STATE();
    case 110:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '-') ADVANCE(109);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(112);
      END_STATE();
    case 111:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '<') ADVANCE(105);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(111);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(112);
      END_STATE();
    case 112:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(112);
      END_STATE();
    case 113:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '!') ADVANCE(118);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(120);
      END_STATE();
    case 114:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '\'') ADVANCE(10);
      if (lookahead == '-') ADVANCE(114);
      if (lookahead == '>') ADVANCE(120);
      if (lookahead != 0) ADVANCE(116);
      END_STATE();
    case 115:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '\'') ADVANCE(10);
      if (lookahead == '-') ADVANCE(114);
      if (lookahead != 0) ADVANCE(116);
      END_STATE();
    case 116:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '\'') ADVANCE(10);
      if (lookahead == '-') ADVANCE(115);
      if (lookahead != 0) ADVANCE(116);
      END_STATE();
    case 117:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '-') ADVANCE(116);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(120);
      END_STATE();
    case 118:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '-') ADVANCE(117);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(120);
      END_STATE();
    case 119:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '<') ADVANCE(113);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(119);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(120);
      END_STATE();
    case 120:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(120);
      END_STATE();
    case 121:
      ACCEPT_TOKEN(sym__unquoted_value);
      if (lookahead != 0 &&
          (lookahead < '\t' || '\r' < lookahead) &&
          lookahead != ' ' &&
          lookahead != '"' &&
          lookahead != '\'' &&
          (lookahead < '<' || '>' < lookahead) &&
          lookahead != '`') ADVANCE(121);
      END_STATE();
    case 122:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '.') ADVANCE(123);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(122);
      END_STATE();
    case 123:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(123);
      END_STATE();
    case 124:
      ACCEPT_TOKEN(sym_comment);
      END_STATE();
    default:
      return false;
  }
}

static const TSLexMode ts_lex_modes[STATE_COUNT] = {
  [0] = {.lex_state = 0, .external_lex_state = 1},
  [1] = {.lex_state = 37},
  [2] = {.lex_state = 37},
  [3] = {.lex_state = 37},
  [4] = {.lex_state = 37},
  [5] = {.lex_state = 37},
  [6] = {.lex_state = 12},
  [7] = {.lex_state = 12},
  [8] = {.lex_state = 12},
  [9] = {.lex_state = 12},
  [10] = {.lex_state = 12},
  [11] = {.lex_state = 12},
  [12] = {.lex_state = 12},
  [13] = {.lex_state = 12},
  [14] = {.lex_state = 12},
  [15] = {.lex_state = 12},
  [16] = {.lex_state = 12},
  [17] = {.lex_state = 37},
  [18] = {.lex_state = 37},
  [19] = {.lex_state = 37},
  [20] = {.lex_state = 37},
  [21] = {.lex_state = 37},
  [22] = {.lex_state = 37},
  [23] = {.lex_state = 37},
  [24] = {.lex_state = 37},
  [25] = {.lex_state = 37},
  [26] = {.lex_state = 37},
  [27] = {.lex_state = 37},
  [28] = {.lex_state = 37},
  [29] = {.lex_state = 37},
  [30] = {.lex_state = 37},
  [31] = {.lex_state = 37},
  [32] = {.lex_state = 37},
  [33] = {.lex_state = 37},
  [34] = {.lex_state = 37},
  [35] = {.lex_state = 37},
  [36] = {.lex_state = 37},
  [37] = {.lex_state = 37},
  [38] = {.lex_state = 37},
  [39] = {.lex_state = 37},
  [40] = {.lex_state = 37},
  [41] = {.lex_state = 37},
  [42] = {.lex_state = 37},
  [43] = {.lex_state = 37},
  [44] = {.lex_state = 37},
  [45] = {.lex_state = 37},
  [46] = {.lex_state = 37},
  [47] = {.lex_state = 37},
  [48] = {.lex_state = 0, .external_lex_state = 2},
  [49] = {.lex_state = 0, .external_lex_state = 3},
  [50] = {.lex_state = 37},
  [51] = {.lex_state = 0, .external_lex_state = 2},
  [52] = {.lex_state = 0, .external_lex_state = 4},
  [53] = {.lex_state = 0, .external_lex_state = 4},
  [54] = {.lex_state = 16},
  [55] = {.lex_state = 0, .external_lex_state = 3},
  [56] = {.lex_state = 12},
  [57] = {.lex_state = 37},
  [58] = {.lex_state = 12},
  [59] = {.lex_state = 12},
  [60] = {.lex_state = 0, .external_lex_state = 2},
  [61] = {.lex_state = 0, .external_lex_state = 3},
  [62] = {.lex_state = 0, .external_lex_state = 4},
  [63] = {.lex_state = 2},
  [64] = {.lex_state = 2},
  [65] = {.lex_state = 12},
  [66] = {.lex_state = 12},
  [67] = {.lex_state = 12},
  [68] = {.lex_state = 12},
  [69] = {.lex_state = 12},
  [70] = {.lex_state = 12},
  [71] = {.lex_state = 12},
  [72] = {.lex_state = 12},
  [73] = {.lex_state = 0, .external_lex_state = 3},
  [74] = {.lex_state = 0, .external_lex_state = 3},
  [75] = {.lex_state = 0, .external_lex_state = 3},
  [76] = {.lex_state = 0, .external_lex_state = 4},
  [77] = {.lex_state = 0, .external_lex_state = 2},
  [78] = {.lex_state = 0, .external_lex_state = 4},
  [79] = {.lex_state = 0, .external_lex_state = 4},
  [80] = {.lex_state = 3},
  [81] = {.lex_state = 0, .external_lex_state = 2},
  [82] = {.lex_state = 0, .external_lex_state = 2},
  [83] = {.lex_state = 4},
  [84] = {.lex_state = 3},
  [85] = {.lex_state = 4},
  [86] = {.lex_state = 15},
  [87] = {.lex_state = 15},
  [88] = {.lex_state = 17},
  [89] = {.lex_state = 15},
  [90] = {.lex_state = 19},
  [91] = {.lex_state = 0},
  [92] = {.lex_state = 0},
  [93] = {.lex_state = 0},
  [94] = {.lex_state = 0},
  [95] = {.lex_state = 0},
  [96] = {.lex_state = 0},
  [97] = {.lex_state = 18},
  [98] = {.lex_state = 0},
  [99] = {.lex_state = 0},
  [100] = {.lex_state = 0},
  [101] = {.lex_state = 0},
  [102] = {.lex_state = 0},
  [103] = {.lex_state = 0},
  [104] = {.lex_state = 0},
  [105] = {.lex_state = 0},
  [106] = {.lex_state = 0},
  [107] = {.lex_state = 0},
  [108] = {.lex_state = 0},
  [109] = {.lex_state = 0},
};

static const uint16_t ts_parse_table[LARGE_STATE_COUNT][SYMBOL_COUNT] = {
  [0] = {
    [ts_builtin_sym_end] = ACTIONS(1),
    [anon_sym_LT] = ACTIONS(1),
    [anon_sym_GT] = ACTIONS(1),
    [anon_sym_SLASH_GT] = ACTIONS(1),
    [anon_sym_LT_SLASH] = ACTIONS(1),
    [sym__script] = ACTIONS(1),
    [sym__style] = ACTIONS(1),
    [sym__i18n] = ACTIONS(1),
    [sym__template] = ACTIONS(1),
    [anon_sym_script] = ACTIONS(1),
    [anon_sym_style] = ACTIONS(1),
    [anon_sym_i18n] = ACTIONS(1),
    [sym_doctype] = ACTIONS(1),
    [sym_tag_name] = ACTIONS(1),
    [anon_sym_LBRACE_LBRACE] = ACTIONS(1),
    [anon_sym_RBRACE_RBRACE] = ACTIONS(1),
    [anon_sym_EQ] = ACTIONS(1),
    [anon_sym_DQUOTE] = ACTIONS(1),
    [anon_sym_SQUOTE] = ACTIONS(1),
    [sym_directive_name] = ACTIONS(1),
    [sym_comment] = ACTIONS(3),
    [sym__script_raw_chunk] = ACTIONS(1),
    [sym__style_raw_chunk] = ACTIONS(1),
    [sym__i18n_raw_chunk] = ACTIONS(1),
    [sym__script_inner_close] = ACTIONS(1),
    [sym__style_inner_close] = ACTIONS(1),
    [sym__i18n_inner_close] = ACTIONS(1),
  },
  [1] = {
    [sym_document] = STATE(109),
    [sym__content] = STATE(3),
    [sym_script_element] = STATE(3),
    [sym_script_start_tag] = STATE(49),
    [sym_script_self_closing_tag] = STATE(17),
    [sym_style_element] = STATE(3),
    [sym_style_start_tag] = STATE(48),
    [sym_style_self_closing_tag] = STATE(18),
    [sym_i18n_element] = STATE(3),
    [sym_i18n_start_tag] = STATE(52),
    [sym_i18n_self_closing_tag] = STATE(24),
    [sym_template_element] = STATE(3),
    [sym_template_start_tag] = STATE(2),
    [sym_template_self_closing_tag] = STATE(27),
    [sym_start_tag] = STATE(3),
    [sym_self_closing_tag] = STATE(3),
    [sym_end_tag] = STATE(3),
    [sym_interpolation] = STATE(3),
    [aux_sym_document_repeat1] = STATE(3),
    [ts_builtin_sym_end] = ACTIONS(5),
    [anon_sym_LT] = ACTIONS(7),
    [anon_sym_LT_SLASH] = ACTIONS(9),
    [sym_doctype] = ACTIONS(11),
    [sym_text] = ACTIONS(13),
    [anon_sym_LBRACE_LBRACE] = ACTIONS(15),
    [sym_comment] = ACTIONS(3),
  },
};

static const uint16_t ts_small_parse_table[] = {
  [0] = 16,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(7), 1,
      anon_sym_LT,
    ACTIONS(15), 1,
      anon_sym_LBRACE_LBRACE,
    ACTIONS(17), 1,
      anon_sym_LT_SLASH,
    ACTIONS(19), 1,
      sym_doctype,
    ACTIONS(21), 1,
      sym_text,
    STATE(2), 1,
      sym_template_start_tag,
    STATE(17), 1,
      sym_script_self_closing_tag,
    STATE(18), 1,
      sym_style_self_closing_tag,
    STATE(23), 1,
      sym__template_end_tag,
    STATE(24), 1,
      sym_i18n_self_closing_tag,
    STATE(27), 1,
      sym_template_self_closing_tag,
    STATE(48), 1,
      sym_style_start_tag,
    STATE(49), 1,
      sym_script_start_tag,
    STATE(52), 1,
      sym_i18n_start_tag,
    STATE(4), 10,
      sym__content,
      sym_script_element,
      sym_style_element,
      sym_i18n_element,
      sym_template_element,
      sym_start_tag,
      sym_self_closing_tag,
      sym_end_tag,
      sym_interpolation,
      aux_sym_document_repeat1,
  [58] = 16,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(7), 1,
      anon_sym_LT,
    ACTIONS(9), 1,
      anon_sym_LT_SLASH,
    ACTIONS(15), 1,
      anon_sym_LBRACE_LBRACE,
    ACTIONS(23), 1,
      ts_builtin_sym_end,
    ACTIONS(25), 1,
      sym_doctype,
    ACTIONS(27), 1,
      sym_text,
    STATE(2), 1,
      sym_template_start_tag,
    STATE(17), 1,
      sym_script_self_closing_tag,
    STATE(18), 1,
      sym_style_self_closing_tag,
    STATE(24), 1,
      sym_i18n_self_closing_tag,
    STATE(27), 1,
      sym_template_self_closing_tag,
    STATE(48), 1,
      sym_style_start_tag,
    STATE(49), 1,
      sym_script_start_tag,
    STATE(52), 1,
      sym_i18n_start_tag,
    STATE(5), 10,
      sym__content,
      sym_script_element,
      sym_style_element,
      sym_i18n_element,
      sym_template_element,
      sym_start_tag,
      sym_self_closing_tag,
      sym_end_tag,
      sym_interpolation,
      aux_sym_document_repeat1,
  [116] = 16,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(7), 1,
      anon_sym_LT,
    ACTIONS(15), 1,
      anon_sym_LBRACE_LBRACE,
    ACTIONS(17), 1,
      anon_sym_LT_SLASH,
    ACTIONS(25), 1,
      sym_doctype,
    ACTIONS(27), 1,
      sym_text,
    STATE(2), 1,
      sym_template_start_tag,
    STATE(17), 1,
      sym_script_self_closing_tag,
    STATE(18), 1,
      sym_style_self_closing_tag,
    STATE(24), 1,
      sym_i18n_self_closing_tag,
    STATE(27), 1,
      sym_template_self_closing_tag,
    STATE(37), 1,
      sym__template_end_tag,
    STATE(48), 1,
      sym_style_start_tag,
    STATE(49), 1,
      sym_script_start_tag,
    STATE(52), 1,
      sym_i18n_start_tag,
    STATE(5), 10,
      sym__content,
      sym_script_element,
      sym_style_element,
      sym_i18n_element,
      sym_template_element,
      sym_start_tag,
      sym_self_closing_tag,
      sym_end_tag,
      sym_interpolation,
      aux_sym_document_repeat1,
  [174] = 16,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(29), 1,
      ts_builtin_sym_end,
    ACTIONS(31), 1,
      anon_sym_LT,
    ACTIONS(34), 1,
      anon_sym_LT_SLASH,
    ACTIONS(37), 1,
      sym_doctype,
    ACTIONS(40), 1,
      sym_text,
    ACTIONS(43), 1,
      anon_sym_LBRACE_LBRACE,
    STATE(2), 1,
      sym_template_start_tag,
    STATE(17), 1,
      sym_script_self_closing_tag,
    STATE(18), 1,
      sym_style_self_closing_tag,
    STATE(24), 1,
      sym_i18n_self_closing_tag,
    STATE(27), 1,
      sym_template_self_closing_tag,
    STATE(48), 1,
      sym_style_start_tag,
    STATE(49), 1,
      sym_script_start_tag,
    STATE(52), 1,
      sym_i18n_start_tag,
    STATE(5), 10,
      sym__content,
      sym_script_element,
      sym_style_element,
      sym_i18n_element,
      sym_template_element,
      sym_start_tag,
      sym_self_closing_tag,
      sym_end_tag,
      sym_interpolation,
      aux_sym_document_repeat1,
  [232] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(46), 1,
      anon_sym_GT,
    ACTIONS(48), 1,
      anon_sym_SLASH_GT,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(13), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [258] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(54), 1,
      anon_sym_GT,
    ACTIONS(56), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(12), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [284] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(58), 1,
      anon_sym_GT,
    ACTIONS(60), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(11), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [310] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(62), 1,
      anon_sym_GT,
    ACTIONS(64), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(14), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [336] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(66), 1,
      anon_sym_GT,
    ACTIONS(68), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(15), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [362] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(70), 1,
      anon_sym_GT,
    ACTIONS(72), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(16), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [388] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(74), 1,
      anon_sym_GT,
    ACTIONS(76), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(16), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [414] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(78), 1,
      anon_sym_GT,
    ACTIONS(80), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(16), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [440] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(82), 1,
      anon_sym_GT,
    ACTIONS(84), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(16), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [466] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(50), 1,
      sym_attribute_name,
    ACTIONS(52), 1,
      sym_directive_name,
    ACTIONS(86), 1,
      anon_sym_GT,
    ACTIONS(88), 1,
      anon_sym_SLASH_GT,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    STATE(16), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [492] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(92), 1,
      sym_attribute_name,
    ACTIONS(95), 1,
      sym_directive_name,
    STATE(65), 1,
      sym__plain_attribute,
    STATE(66), 1,
      sym_directive,
    ACTIONS(90), 2,
      anon_sym_GT,
      anon_sym_SLASH_GT,
    STATE(16), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [516] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(100), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(98), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [530] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(104), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(102), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [544] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(108), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(106), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [558] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(112), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(110), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [572] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(116), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(114), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [586] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(120), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(118), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [600] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(124), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(122), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [614] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(128), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(126), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [628] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(132), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(130), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [642] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(136), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(134), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [656] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(140), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(138), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [670] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(144), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(142), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [684] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(148), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(146), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [698] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(152), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(150), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [712] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(156), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(154), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [726] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(160), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(158), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [740] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(164), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(162), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [754] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(168), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(166), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [768] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(172), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(170), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [782] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(176), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(174), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [796] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(180), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(178), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [810] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(184), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(182), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [824] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(188), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(186), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [838] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(192), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(190), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [852] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(196), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(194), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [866] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(200), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(198), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [880] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(204), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(202), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [894] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(208), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(206), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [908] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(212), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(210), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [922] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(216), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(214), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [936] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(220), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(218), 4,
      ts_builtin_sym_end,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [950] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(222), 1,
      anon_sym_LT_SLASH,
    STATE(21), 1,
      sym__style_end_tag,
    STATE(51), 1,
      aux_sym__style_raw_text,
    ACTIONS(224), 2,
      sym__style_raw_chunk,
      sym__style_inner_close,
  [967] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(226), 1,
      anon_sym_LT_SLASH,
    STATE(20), 1,
      sym__script_end_tag,
    STATE(55), 1,
      aux_sym__script_raw_text,
    ACTIONS(228), 2,
      sym__script_raw_chunk,
      sym__script_inner_close,
  [984] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(230), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(232), 3,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [997] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(222), 1,
      anon_sym_LT_SLASH,
    STATE(35), 1,
      sym__style_end_tag,
    STATE(60), 1,
      aux_sym__style_raw_text,
    ACTIONS(224), 2,
      sym__style_raw_chunk,
      sym__style_inner_close,
  [1014] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(234), 1,
      anon_sym_LT_SLASH,
    STATE(22), 1,
      sym__i18n_end_tag,
    STATE(53), 1,
      aux_sym__i18n_raw_text,
    ACTIONS(236), 2,
      sym__i18n_raw_chunk,
      sym__i18n_inner_close,
  [1031] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(234), 1,
      anon_sym_LT_SLASH,
    STATE(36), 1,
      sym__i18n_end_tag,
    STATE(62), 1,
      aux_sym__i18n_raw_text,
    ACTIONS(236), 2,
      sym__i18n_raw_chunk,
      sym__i18n_inner_close,
  [1048] = 6,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(238), 1,
      sym__script,
    ACTIONS(240), 1,
      sym__style,
    ACTIONS(242), 1,
      sym__i18n,
    ACTIONS(244), 1,
      sym__template,
    ACTIONS(246), 1,
      sym_tag_name,
  [1067] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(226), 1,
      anon_sym_LT_SLASH,
    STATE(34), 1,
      sym__script_end_tag,
    STATE(61), 1,
      aux_sym__script_raw_text,
    ACTIONS(228), 2,
      sym__script_raw_chunk,
      sym__script_inner_close,
  [1084] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(250), 1,
      anon_sym_EQ,
    ACTIONS(252), 1,
      sym_attribute_name,
    ACTIONS(248), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1099] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(254), 2,
      anon_sym_LT,
      sym_text,
    ACTIONS(256), 3,
      anon_sym_LT_SLASH,
      sym_doctype,
      anon_sym_LBRACE_LBRACE,
  [1112] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(260), 1,
      anon_sym_EQ,
    ACTIONS(262), 1,
      sym_attribute_name,
    ACTIONS(258), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1127] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(266), 1,
      sym_attribute_name,
    ACTIONS(264), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1139] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(268), 1,
      anon_sym_LT_SLASH,
    STATE(60), 1,
      aux_sym__style_raw_text,
    ACTIONS(270), 2,
      sym__style_raw_chunk,
      sym__style_inner_close,
  [1153] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(273), 1,
      anon_sym_LT_SLASH,
    STATE(61), 1,
      aux_sym__script_raw_text,
    ACTIONS(275), 2,
      sym__script_raw_chunk,
      sym__script_inner_close,
  [1167] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(278), 1,
      anon_sym_LT_SLASH,
    STATE(62), 1,
      aux_sym__i18n_raw_text,
    ACTIONS(280), 2,
      sym__i18n_raw_chunk,
      sym__i18n_inner_close,
  [1181] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(283), 1,
      anon_sym_DQUOTE,
    ACTIONS(285), 1,
      anon_sym_SQUOTE,
    ACTIONS(287), 1,
      sym__unquoted_value,
    STATE(67), 1,
      sym_quoted_attribute_value,
  [1197] = 5,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(289), 1,
      anon_sym_DQUOTE,
    ACTIONS(291), 1,
      anon_sym_SQUOTE,
    ACTIONS(293), 1,
      sym__unquoted_value,
    STATE(69), 1,
      sym_directive_value,
  [1213] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(297), 1,
      sym_attribute_name,
    ACTIONS(295), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1225] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(301), 1,
      sym_attribute_name,
    ACTIONS(299), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1237] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(305), 1,
      sym_attribute_name,
    ACTIONS(303), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1249] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(309), 1,
      sym_attribute_name,
    ACTIONS(307), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1261] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(313), 1,
      sym_attribute_name,
    ACTIONS(311), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1273] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(317), 1,
      sym_attribute_name,
    ACTIONS(315), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1285] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(321), 1,
      sym_attribute_name,
    ACTIONS(319), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1297] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(325), 1,
      sym_attribute_name,
    ACTIONS(323), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [1309] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(327), 3,
      sym__script_raw_chunk,
      sym__script_inner_close,
      anon_sym_LT_SLASH,
  [1318] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(329), 3,
      sym__script_raw_chunk,
      sym__script_inner_close,
      anon_sym_LT_SLASH,
  [1327] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(331), 3,
      sym__script_raw_chunk,
      sym__script_inner_close,
      anon_sym_LT_SLASH,
  [1336] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(333), 3,
      sym__i18n_raw_chunk,
      sym__i18n_inner_close,
      anon_sym_LT_SLASH,
  [1345] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(335), 3,
      sym__style_raw_chunk,
      sym__style_inner_close,
      anon_sym_LT_SLASH,
  [1354] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(337), 3,
      sym__i18n_raw_chunk,
      sym__i18n_inner_close,
      anon_sym_LT_SLASH,
  [1363] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(339), 3,
      sym__i18n_raw_chunk,
      sym__i18n_inner_close,
      anon_sym_LT_SLASH,
  [1372] = 4,
    ACTIONS(341), 1,
      anon_sym_DQUOTE,
    ACTIONS(343), 1,
      aux_sym__double_quoted_text_token1,
    ACTIONS(345), 1,
      sym_comment,
    STATE(100), 1,
      sym__double_quoted_text,
  [1385] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(347), 3,
      sym__style_raw_chunk,
      sym__style_inner_close,
      anon_sym_LT_SLASH,
  [1394] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(349), 3,
      sym__style_raw_chunk,
      sym__style_inner_close,
      anon_sym_LT_SLASH,
  [1403] = 4,
    ACTIONS(341), 1,
      anon_sym_SQUOTE,
    ACTIONS(345), 1,
      sym_comment,
    ACTIONS(351), 1,
      aux_sym__single_quoted_text_token1,
    STATE(91), 1,
      sym__single_quoted_text,
  [1416] = 4,
    ACTIONS(345), 1,
      sym_comment,
    ACTIONS(353), 1,
      anon_sym_DQUOTE,
    ACTIONS(355), 1,
      aux_sym__double_quoted_text_token1,
    STATE(105), 1,
      sym__double_quoted_expr,
  [1429] = 4,
    ACTIONS(345), 1,
      sym_comment,
    ACTIONS(353), 1,
      anon_sym_SQUOTE,
    ACTIONS(357), 1,
      aux_sym__single_quoted_text_token1,
    STATE(107), 1,
      sym__single_quoted_expr,
  [1442] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(359), 1,
      anon_sym_script,
    STATE(94), 1,
      sym__script_close_name,
  [1452] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(361), 1,
      anon_sym_i18n,
    STATE(96), 1,
      sym__i18n_close_name,
  [1462] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(363), 1,
      sym__template,
    ACTIONS(365), 1,
      sym_tag_name,
  [1472] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(367), 1,
      anon_sym_style,
    STATE(98), 1,
      sym__style_close_name,
  [1482] = 3,
    ACTIONS(345), 1,
      sym_comment,
    ACTIONS(369), 1,
      anon_sym_RBRACE_RBRACE,
    ACTIONS(371), 1,
      sym__interpolation_text,
  [1492] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(373), 1,
      anon_sym_SQUOTE,
  [1499] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(375), 1,
      anon_sym_GT,
  [1506] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(377), 1,
      anon_sym_GT,
  [1513] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(379), 1,
      anon_sym_GT,
  [1520] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(381), 1,
      anon_sym_GT,
  [1527] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(383), 1,
      anon_sym_GT,
  [1534] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(385), 1,
      sym_tag_name,
  [1541] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(387), 1,
      anon_sym_GT,
  [1548] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(389), 1,
      anon_sym_DQUOTE,
  [1555] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(373), 1,
      anon_sym_DQUOTE,
  [1562] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(391), 1,
      anon_sym_SQUOTE,
  [1569] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(393), 1,
      anon_sym_RBRACE_RBRACE,
  [1576] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(395), 1,
      anon_sym_GT,
  [1583] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(397), 1,
      anon_sym_DQUOTE,
  [1590] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(399), 1,
      anon_sym_DQUOTE,
  [1597] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(401), 1,
      anon_sym_SQUOTE,
  [1604] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(399), 1,
      anon_sym_SQUOTE,
  [1611] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(403), 1,
      anon_sym_GT,
  [1618] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(405), 1,
      ts_builtin_sym_end,
};

static const uint32_t ts_small_parse_table_map[] = {
  [SMALL_STATE(2)] = 0,
  [SMALL_STATE(3)] = 58,
  [SMALL_STATE(4)] = 116,
  [SMALL_STATE(5)] = 174,
  [SMALL_STATE(6)] = 232,
  [SMALL_STATE(7)] = 258,
  [SMALL_STATE(8)] = 284,
  [SMALL_STATE(9)] = 310,
  [SMALL_STATE(10)] = 336,
  [SMALL_STATE(11)] = 362,
  [SMALL_STATE(12)] = 388,
  [SMALL_STATE(13)] = 414,
  [SMALL_STATE(14)] = 440,
  [SMALL_STATE(15)] = 466,
  [SMALL_STATE(16)] = 492,
  [SMALL_STATE(17)] = 516,
  [SMALL_STATE(18)] = 530,
  [SMALL_STATE(19)] = 544,
  [SMALL_STATE(20)] = 558,
  [SMALL_STATE(21)] = 572,
  [SMALL_STATE(22)] = 586,
  [SMALL_STATE(23)] = 600,
  [SMALL_STATE(24)] = 614,
  [SMALL_STATE(25)] = 628,
  [SMALL_STATE(26)] = 642,
  [SMALL_STATE(27)] = 656,
  [SMALL_STATE(28)] = 670,
  [SMALL_STATE(29)] = 684,
  [SMALL_STATE(30)] = 698,
  [SMALL_STATE(31)] = 712,
  [SMALL_STATE(32)] = 726,
  [SMALL_STATE(33)] = 740,
  [SMALL_STATE(34)] = 754,
  [SMALL_STATE(35)] = 768,
  [SMALL_STATE(36)] = 782,
  [SMALL_STATE(37)] = 796,
  [SMALL_STATE(38)] = 810,
  [SMALL_STATE(39)] = 824,
  [SMALL_STATE(40)] = 838,
  [SMALL_STATE(41)] = 852,
  [SMALL_STATE(42)] = 866,
  [SMALL_STATE(43)] = 880,
  [SMALL_STATE(44)] = 894,
  [SMALL_STATE(45)] = 908,
  [SMALL_STATE(46)] = 922,
  [SMALL_STATE(47)] = 936,
  [SMALL_STATE(48)] = 950,
  [SMALL_STATE(49)] = 967,
  [SMALL_STATE(50)] = 984,
  [SMALL_STATE(51)] = 997,
  [SMALL_STATE(52)] = 1014,
  [SMALL_STATE(53)] = 1031,
  [SMALL_STATE(54)] = 1048,
  [SMALL_STATE(55)] = 1067,
  [SMALL_STATE(56)] = 1084,
  [SMALL_STATE(57)] = 1099,
  [SMALL_STATE(58)] = 1112,
  [SMALL_STATE(59)] = 1127,
  [SMALL_STATE(60)] = 1139,
  [SMALL_STATE(61)] = 1153,
  [SMALL_STATE(62)] = 1167,
  [SMALL_STATE(63)] = 1181,
  [SMALL_STATE(64)] = 1197,
  [SMALL_STATE(65)] = 1213,
  [SMALL_STATE(66)] = 1225,
  [SMALL_STATE(67)] = 1237,
  [SMALL_STATE(68)] = 1249,
  [SMALL_STATE(69)] = 1261,
  [SMALL_STATE(70)] = 1273,
  [SMALL_STATE(71)] = 1285,
  [SMALL_STATE(72)] = 1297,
  [SMALL_STATE(73)] = 1309,
  [SMALL_STATE(74)] = 1318,
  [SMALL_STATE(75)] = 1327,
  [SMALL_STATE(76)] = 1336,
  [SMALL_STATE(77)] = 1345,
  [SMALL_STATE(78)] = 1354,
  [SMALL_STATE(79)] = 1363,
  [SMALL_STATE(80)] = 1372,
  [SMALL_STATE(81)] = 1385,
  [SMALL_STATE(82)] = 1394,
  [SMALL_STATE(83)] = 1403,
  [SMALL_STATE(84)] = 1416,
  [SMALL_STATE(85)] = 1429,
  [SMALL_STATE(86)] = 1442,
  [SMALL_STATE(87)] = 1452,
  [SMALL_STATE(88)] = 1462,
  [SMALL_STATE(89)] = 1472,
  [SMALL_STATE(90)] = 1482,
  [SMALL_STATE(91)] = 1492,
  [SMALL_STATE(92)] = 1499,
  [SMALL_STATE(93)] = 1506,
  [SMALL_STATE(94)] = 1513,
  [SMALL_STATE(95)] = 1520,
  [SMALL_STATE(96)] = 1527,
  [SMALL_STATE(97)] = 1534,
  [SMALL_STATE(98)] = 1541,
  [SMALL_STATE(99)] = 1548,
  [SMALL_STATE(100)] = 1555,
  [SMALL_STATE(101)] = 1562,
  [SMALL_STATE(102)] = 1569,
  [SMALL_STATE(103)] = 1576,
  [SMALL_STATE(104)] = 1583,
  [SMALL_STATE(105)] = 1590,
  [SMALL_STATE(106)] = 1597,
  [SMALL_STATE(107)] = 1604,
  [SMALL_STATE(108)] = 1611,
  [SMALL_STATE(109)] = 1618,
};

static const TSParseActionEntry ts_parse_actions[] = {
  [0] = {.entry = {.count = 0, .reusable = false}},
  [1] = {.entry = {.count = 1, .reusable = false}}, RECOVER(),
  [3] = {.entry = {.count = 1, .reusable = true}}, SHIFT_EXTRA(),
  [5] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_document, 0, 0, 0),
  [7] = {.entry = {.count = 1, .reusable = false}}, SHIFT(54),
  [9] = {.entry = {.count = 1, .reusable = true}}, SHIFT(97),
  [11] = {.entry = {.count = 1, .reusable = true}}, SHIFT(3),
  [13] = {.entry = {.count = 1, .reusable = false}}, SHIFT(3),
  [15] = {.entry = {.count = 1, .reusable = true}}, SHIFT(90),
  [17] = {.entry = {.count = 1, .reusable = true}}, SHIFT(88),
  [19] = {.entry = {.count = 1, .reusable = true}}, SHIFT(4),
  [21] = {.entry = {.count = 1, .reusable = false}}, SHIFT(4),
  [23] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_document, 1, 0, 0),
  [25] = {.entry = {.count = 1, .reusable = true}}, SHIFT(5),
  [27] = {.entry = {.count = 1, .reusable = false}}, SHIFT(5),
  [29] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0),
  [31] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(54),
  [34] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(97),
  [37] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(5),
  [40] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(5),
  [43] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(90),
  [46] = {.entry = {.count = 1, .reusable = true}}, SHIFT(76),
  [48] = {.entry = {.count = 1, .reusable = true}}, SHIFT(28),
  [50] = {.entry = {.count = 1, .reusable = false}}, SHIFT(56),
  [52] = {.entry = {.count = 1, .reusable = true}}, SHIFT(58),
  [54] = {.entry = {.count = 1, .reusable = true}}, SHIFT(81),
  [56] = {.entry = {.count = 1, .reusable = true}}, SHIFT(26),
  [58] = {.entry = {.count = 1, .reusable = true}}, SHIFT(73),
  [60] = {.entry = {.count = 1, .reusable = true}}, SHIFT(25),
  [62] = {.entry = {.count = 1, .reusable = true}}, SHIFT(50),
  [64] = {.entry = {.count = 1, .reusable = true}}, SHIFT(29),
  [66] = {.entry = {.count = 1, .reusable = true}}, SHIFT(30),
  [68] = {.entry = {.count = 1, .reusable = true}}, SHIFT(31),
  [70] = {.entry = {.count = 1, .reusable = true}}, SHIFT(75),
  [72] = {.entry = {.count = 1, .reusable = true}}, SHIFT(38),
  [74] = {.entry = {.count = 1, .reusable = true}}, SHIFT(77),
  [76] = {.entry = {.count = 1, .reusable = true}}, SHIFT(39),
  [78] = {.entry = {.count = 1, .reusable = true}}, SHIFT(79),
  [80] = {.entry = {.count = 1, .reusable = true}}, SHIFT(40),
  [82] = {.entry = {.count = 1, .reusable = true}}, SHIFT(57),
  [84] = {.entry = {.count = 1, .reusable = true}}, SHIFT(41),
  [86] = {.entry = {.count = 1, .reusable = true}}, SHIFT(42),
  [88] = {.entry = {.count = 1, .reusable = true}}, SHIFT(43),
  [90] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_script_start_tag_repeat1, 2, 0, 0),
  [92] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_script_start_tag_repeat1, 2, 0, 0), SHIFT_REPEAT(56),
  [95] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_script_start_tag_repeat1, 2, 0, 0), SHIFT_REPEAT(58),
  [98] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_element, 1, 0, 0),
  [100] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_script_element, 1, 0, 0),
  [102] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_element, 1, 0, 0),
  [104] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_style_element, 1, 0, 0),
  [106] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_interpolation, 2, 0, 0),
  [108] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_interpolation, 2, 0, 0),
  [110] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_element, 2, 0, 1),
  [112] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_script_element, 2, 0, 1),
  [114] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_element, 2, 0, 1),
  [116] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_style_element, 2, 0, 1),
  [118] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_element, 2, 0, 1),
  [120] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_i18n_element, 2, 0, 1),
  [122] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_element, 2, 0, 1),
  [124] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_element, 2, 0, 1),
  [126] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_element, 1, 0, 0),
  [128] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_i18n_element, 1, 0, 0),
  [130] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_self_closing_tag, 3, 0, 2),
  [132] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_script_self_closing_tag, 3, 0, 2),
  [134] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_self_closing_tag, 3, 0, 2),
  [136] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_style_self_closing_tag, 3, 0, 2),
  [138] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_element, 1, 0, 0),
  [140] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_element, 1, 0, 0),
  [142] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_self_closing_tag, 3, 0, 2),
  [144] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_i18n_self_closing_tag, 3, 0, 2),
  [146] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_self_closing_tag, 3, 0, 2),
  [148] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_self_closing_tag, 3, 0, 2),
  [150] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_start_tag, 3, 0, 2),
  [152] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_start_tag, 3, 0, 2),
  [154] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_self_closing_tag, 3, 0, 2),
  [156] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_self_closing_tag, 3, 0, 2),
  [158] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_end_tag, 3, 0, 2),
  [160] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_end_tag, 3, 0, 2),
  [162] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_interpolation, 3, 0, 0),
  [164] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_interpolation, 3, 0, 0),
  [166] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_element, 3, 0, 5),
  [168] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_script_element, 3, 0, 5),
  [170] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_element, 3, 0, 5),
  [172] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_style_element, 3, 0, 5),
  [174] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_element, 3, 0, 5),
  [176] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_i18n_element, 3, 0, 5),
  [178] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_element, 3, 0, 6),
  [180] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_element, 3, 0, 6),
  [182] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_self_closing_tag, 4, 0, 2),
  [184] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_script_self_closing_tag, 4, 0, 2),
  [186] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_self_closing_tag, 4, 0, 2),
  [188] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_style_self_closing_tag, 4, 0, 2),
  [190] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_self_closing_tag, 4, 0, 2),
  [192] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_i18n_self_closing_tag, 4, 0, 2),
  [194] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_self_closing_tag, 4, 0, 2),
  [196] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_self_closing_tag, 4, 0, 2),
  [198] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_start_tag, 4, 0, 2),
  [200] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_start_tag, 4, 0, 2),
  [202] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_self_closing_tag, 4, 0, 2),
  [204] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_self_closing_tag, 4, 0, 2),
  [206] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__script_end_tag, 3, 0, 2),
  [208] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__script_end_tag, 3, 0, 2),
  [210] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__style_end_tag, 3, 0, 2),
  [212] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__style_end_tag, 3, 0, 2),
  [214] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__i18n_end_tag, 3, 0, 2),
  [216] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__i18n_end_tag, 3, 0, 2),
  [218] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__template_end_tag, 3, 0, 2),
  [220] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__template_end_tag, 3, 0, 2),
  [222] = {.entry = {.count = 1, .reusable = true}}, SHIFT(89),
  [224] = {.entry = {.count = 1, .reusable = true}}, SHIFT(82),
  [226] = {.entry = {.count = 1, .reusable = true}}, SHIFT(86),
  [228] = {.entry = {.count = 1, .reusable = true}}, SHIFT(74),
  [230] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_start_tag, 3, 0, 2),
  [232] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_start_tag, 3, 0, 2),
  [234] = {.entry = {.count = 1, .reusable = true}}, SHIFT(87),
  [236] = {.entry = {.count = 1, .reusable = true}}, SHIFT(78),
  [238] = {.entry = {.count = 1, .reusable = false}}, SHIFT(8),
  [240] = {.entry = {.count = 1, .reusable = false}}, SHIFT(7),
  [242] = {.entry = {.count = 1, .reusable = false}}, SHIFT(6),
  [244] = {.entry = {.count = 1, .reusable = false}}, SHIFT(9),
  [246] = {.entry = {.count = 1, .reusable = false}}, SHIFT(10),
  [248] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__plain_attribute, 1, 0, 3),
  [250] = {.entry = {.count = 1, .reusable = true}}, SHIFT(63),
  [252] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__plain_attribute, 1, 0, 3),
  [254] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_start_tag, 4, 0, 2),
  [256] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_start_tag, 4, 0, 2),
  [258] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 1, 0, 3),
  [260] = {.entry = {.count = 1, .reusable = true}}, SHIFT(64),
  [262] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 1, 0, 3),
  [264] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive_value, 3, 0, 0),
  [266] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive_value, 3, 0, 0),
  [268] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__style_raw_text, 2, 0, 0),
  [270] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym__style_raw_text, 2, 0, 0), SHIFT_REPEAT(82),
  [273] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__script_raw_text, 2, 0, 0),
  [275] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym__script_raw_text, 2, 0, 0), SHIFT_REPEAT(74),
  [278] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__i18n_raw_text, 2, 0, 0),
  [280] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym__i18n_raw_text, 2, 0, 0), SHIFT_REPEAT(78),
  [283] = {.entry = {.count = 1, .reusable = true}}, SHIFT(80),
  [285] = {.entry = {.count = 1, .reusable = true}}, SHIFT(83),
  [287] = {.entry = {.count = 1, .reusable = true}}, SHIFT(67),
  [289] = {.entry = {.count = 1, .reusable = true}}, SHIFT(84),
  [291] = {.entry = {.count = 1, .reusable = true}}, SHIFT(85),
  [293] = {.entry = {.count = 1, .reusable = true}}, SHIFT(68),
  [295] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute, 1, 0, 4),
  [297] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_attribute, 1, 0, 4),
  [299] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute, 1, 0, 0),
  [301] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_attribute, 1, 0, 0),
  [303] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__plain_attribute, 3, 0, 7),
  [305] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__plain_attribute, 3, 0, 7),
  [307] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive_value, 1, 0, 8),
  [309] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive_value, 1, 0, 8),
  [311] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 3, 0, 7),
  [313] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 3, 0, 7),
  [315] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_quoted_attribute_value, 2, 0, 0),
  [317] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_quoted_attribute_value, 2, 0, 0),
  [319] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive_value, 2, 0, 0),
  [321] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive_value, 2, 0, 0),
  [323] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_quoted_attribute_value, 3, 0, 0),
  [325] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_quoted_attribute_value, 3, 0, 0),
  [327] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_start_tag, 3, 0, 2),
  [329] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__script_raw_text, 1, 0, 0),
  [331] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_start_tag, 4, 0, 2),
  [333] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_start_tag, 3, 0, 2),
  [335] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_start_tag, 4, 0, 2),
  [337] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__i18n_raw_text, 1, 0, 0),
  [339] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_start_tag, 4, 0, 2),
  [341] = {.entry = {.count = 1, .reusable = false}}, SHIFT(70),
  [343] = {.entry = {.count = 1, .reusable = false}}, SHIFT(99),
  [345] = {.entry = {.count = 1, .reusable = false}}, SHIFT_EXTRA(),
  [347] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_start_tag, 3, 0, 2),
  [349] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__style_raw_text, 1, 0, 0),
  [351] = {.entry = {.count = 1, .reusable = false}}, SHIFT(101),
  [353] = {.entry = {.count = 1, .reusable = false}}, SHIFT(71),
  [355] = {.entry = {.count = 1, .reusable = false}}, SHIFT(104),
  [357] = {.entry = {.count = 1, .reusable = false}}, SHIFT(106),
  [359] = {.entry = {.count = 1, .reusable = true}}, SHIFT(92),
  [361] = {.entry = {.count = 1, .reusable = true}}, SHIFT(93),
  [363] = {.entry = {.count = 1, .reusable = false}}, SHIFT(103),
  [365] = {.entry = {.count = 1, .reusable = false}}, SHIFT(108),
  [367] = {.entry = {.count = 1, .reusable = true}}, SHIFT(95),
  [369] = {.entry = {.count = 1, .reusable = false}}, SHIFT(19),
  [371] = {.entry = {.count = 1, .reusable = false}}, SHIFT(102),
  [373] = {.entry = {.count = 1, .reusable = true}}, SHIFT(72),
  [375] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__script_close_name, 1, 0, 0),
  [377] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__i18n_close_name, 1, 0, 0),
  [379] = {.entry = {.count = 1, .reusable = true}}, SHIFT(44),
  [381] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__style_close_name, 1, 0, 0),
  [383] = {.entry = {.count = 1, .reusable = true}}, SHIFT(46),
  [385] = {.entry = {.count = 1, .reusable = true}}, SHIFT(108),
  [387] = {.entry = {.count = 1, .reusable = true}}, SHIFT(45),
  [389] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__double_quoted_text, 1, 0, 0),
  [391] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__single_quoted_text, 1, 0, 0),
  [393] = {.entry = {.count = 1, .reusable = true}}, SHIFT(33),
  [395] = {.entry = {.count = 1, .reusable = true}}, SHIFT(47),
  [397] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__double_quoted_expr, 1, 0, 0),
  [399] = {.entry = {.count = 1, .reusable = true}}, SHIFT(59),
  [401] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__single_quoted_expr, 1, 0, 0),
  [403] = {.entry = {.count = 1, .reusable = true}}, SHIFT(32),
  [405] = {.entry = {.count = 1, .reusable = true}},  ACCEPT_INPUT(),
};

enum ts_external_scanner_symbol_identifiers {
  ts_external_token__script_raw_chunk = 0,
  ts_external_token__style_raw_chunk = 1,
  ts_external_token__i18n_raw_chunk = 2,
  ts_external_token__script_inner_close = 3,
  ts_external_token__style_inner_close = 4,
  ts_external_token__i18n_inner_close = 5,
};

static const TSSymbol ts_external_scanner_symbol_map[EXTERNAL_TOKEN_COUNT] = {
  [ts_external_token__script_raw_chunk] = sym__script_raw_chunk,
  [ts_external_token__style_raw_chunk] = sym__style_raw_chunk,
  [ts_external_token__i18n_raw_chunk] = sym__i18n_raw_chunk,
  [ts_external_token__script_inner_close] = sym__script_inner_close,
  [ts_external_token__style_inner_close] = sym__style_inner_close,
  [ts_external_token__i18n_inner_close] = sym__i18n_inner_close,
};

static const bool ts_external_scanner_states[5][EXTERNAL_TOKEN_COUNT] = {
  [1] = {
    [ts_external_token__script_raw_chunk] = true,
    [ts_external_token__style_raw_chunk] = true,
    [ts_external_token__i18n_raw_chunk] = true,
    [ts_external_token__script_inner_close] = true,
    [ts_external_token__style_inner_close] = true,
    [ts_external_token__i18n_inner_close] = true,
  },
  [2] = {
    [ts_external_token__style_raw_chunk] = true,
    [ts_external_token__style_inner_close] = true,
  },
  [3] = {
    [ts_external_token__script_raw_chunk] = true,
    [ts_external_token__script_inner_close] = true,
  },
  [4] = {
    [ts_external_token__i18n_raw_chunk] = true,
    [ts_external_token__i18n_inner_close] = true,
  },
};

#ifdef __cplusplus
extern "C" {
#endif
void *tree_sitter_piko_external_scanner_create(void);
void tree_sitter_piko_external_scanner_destroy(void *);
bool tree_sitter_piko_external_scanner_scan(void *, TSLexer *, const bool *);
unsigned tree_sitter_piko_external_scanner_serialize(void *, char *);
void tree_sitter_piko_external_scanner_deserialize(void *, const char *, unsigned);

#ifdef TREE_SITTER_HIDE_SYMBOLS
#define TS_PUBLIC
#elif defined(_WIN32)
#define TS_PUBLIC __declspec(dllexport)
#else
#define TS_PUBLIC __attribute__((visibility("default")))
#endif

TS_PUBLIC const TSLanguage *tree_sitter_piko(void) {
  static const TSLanguage language = {
    .version = LANGUAGE_VERSION,
    .symbol_count = SYMBOL_COUNT,
    .alias_count = ALIAS_COUNT,
    .token_count = TOKEN_COUNT,
    .external_token_count = EXTERNAL_TOKEN_COUNT,
    .state_count = STATE_COUNT,
    .large_state_count = LARGE_STATE_COUNT,
    .production_id_count = PRODUCTION_ID_COUNT,
    .field_count = FIELD_COUNT,
    .max_alias_sequence_length = MAX_ALIAS_SEQUENCE_LENGTH,
    .parse_table = &ts_parse_table[0][0],
    .small_parse_table = ts_small_parse_table,
    .small_parse_table_map = ts_small_parse_table_map,
    .parse_actions = ts_parse_actions,
    .symbol_names = ts_symbol_names,
    .field_names = ts_field_names,
    .field_map_slices = ts_field_map_slices,
    .field_map_entries = ts_field_map_entries,
    .symbol_metadata = ts_symbol_metadata,
    .public_symbol_map = ts_symbol_map,
    .alias_map = ts_non_terminal_alias_map,
    .alias_sequences = &ts_alias_sequences[0][0],
    .lex_modes = ts_lex_modes,
    .lex_fn = ts_lex,
    .external_scanner = {
      &ts_external_scanner_states[0][0],
      ts_external_scanner_symbol_map,
      tree_sitter_piko_external_scanner_create,
      tree_sitter_piko_external_scanner_destroy,
      tree_sitter_piko_external_scanner_scan,
      tree_sitter_piko_external_scanner_serialize,
      tree_sitter_piko_external_scanner_deserialize,
    },
    .primary_state_ids = ts_primary_state_ids,
  };
  return &language;
}
#ifdef __cplusplus
}
#endif
