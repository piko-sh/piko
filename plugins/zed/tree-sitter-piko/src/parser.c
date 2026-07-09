#include "tree_sitter/parser.h"

#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic ignored "-Wmissing-field-initializers"
#endif

#define LANGUAGE_VERSION 14
#define STATE_COUNT 92
#define LARGE_STATE_COUNT 2
#define SYMBOL_COUNT 59
#define ALIAS_COUNT 0
#define TOKEN_COUNT 26
#define EXTERNAL_TOKEN_COUNT 3
#define FIELD_COUNT 2
#define MAX_ALIAS_SEQUENCE_LENGTH 4
#define PRODUCTION_ID_COUNT 5

enum ts_symbol_identifiers {
  anon_sym_LT = 1,
  anon_sym_GT = 2,
  anon_sym_LT_SLASH = 3,
  anon_sym_script = 4,
  anon_sym_style = 5,
  anon_sym_i18n = 6,
  anon_sym_template = 7,
  anon_sym_SLASH_GT = 8,
  sym_doctype = 9,
  sym_tag_name = 10,
  sym_text = 11,
  anon_sym_LBRACE_LBRACE = 12,
  anon_sym_RBRACE_RBRACE = 13,
  sym__interpolation_text = 14,
  anon_sym_EQ = 15,
  sym_attribute_name = 16,
  anon_sym_DQUOTE = 17,
  anon_sym_SQUOTE = 18,
  aux_sym__double_quoted_text_token1 = 19,
  aux_sym__single_quoted_text_token1 = 20,
  sym_directive_name = 21,
  sym_comment = 22,
  sym_script_raw_text = 23,
  sym_style_raw_text = 24,
  sym_i18n_raw_text = 25,
  sym_document = 26,
  sym__content = 27,
  sym_script_element = 28,
  sym_script_start_tag = 29,
  sym__script_end_tag = 30,
  sym_style_element = 31,
  sym_style_start_tag = 32,
  sym__style_end_tag = 33,
  sym_i18n_element = 34,
  sym_i18n_start_tag = 35,
  sym__i18n_end_tag = 36,
  sym__script = 37,
  sym__style = 38,
  sym__i18n = 39,
  sym_template_element = 40,
  sym_template_start_tag = 41,
  sym__template_end_tag = 42,
  sym__template = 43,
  sym_start_tag = 44,
  sym_self_closing_tag = 45,
  sym_end_tag = 46,
  sym_interpolation = 47,
  sym_attribute = 48,
  sym__plain_attribute = 49,
  sym_quoted_attribute_value = 50,
  sym__double_quoted_text = 51,
  sym__single_quoted_text = 52,
  sym_directive = 53,
  sym_directive_value = 54,
  sym__double_quoted_expr = 55,
  sym__single_quoted_expr = 56,
  aux_sym_document_repeat1 = 57,
  aux_sym_script_start_tag_repeat1 = 58,
};

static const char * const ts_symbol_names[] = {
  [ts_builtin_sym_end] = "end",
  [anon_sym_LT] = "<",
  [anon_sym_GT] = ">",
  [anon_sym_LT_SLASH] = "</",
  [anon_sym_script] = "script",
  [anon_sym_style] = "style",
  [anon_sym_i18n] = "i18n",
  [anon_sym_template] = "template",
  [anon_sym_SLASH_GT] = "/>",
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
  [sym_directive_name] = "directive_name",
  [sym_comment] = "comment",
  [sym_script_raw_text] = "raw_text",
  [sym_style_raw_text] = "raw_text",
  [sym_i18n_raw_text] = "raw_text",
  [sym_document] = "document",
  [sym__content] = "_content",
  [sym_script_element] = "script_element",
  [sym_script_start_tag] = "start_tag",
  [sym__script_end_tag] = "end_tag",
  [sym_style_element] = "style_element",
  [sym_style_start_tag] = "start_tag",
  [sym__style_end_tag] = "end_tag",
  [sym_i18n_element] = "i18n_element",
  [sym_i18n_start_tag] = "start_tag",
  [sym__i18n_end_tag] = "end_tag",
  [sym__script] = "tag_name",
  [sym__style] = "tag_name",
  [sym__i18n] = "tag_name",
  [sym_template_element] = "template_element",
  [sym_template_start_tag] = "start_tag",
  [sym__template_end_tag] = "end_tag",
  [sym__template] = "tag_name",
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
};

static const TSSymbol ts_symbol_map[] = {
  [ts_builtin_sym_end] = ts_builtin_sym_end,
  [anon_sym_LT] = anon_sym_LT,
  [anon_sym_GT] = anon_sym_GT,
  [anon_sym_LT_SLASH] = anon_sym_LT_SLASH,
  [anon_sym_script] = anon_sym_script,
  [anon_sym_style] = anon_sym_style,
  [anon_sym_i18n] = anon_sym_i18n,
  [anon_sym_template] = anon_sym_template,
  [anon_sym_SLASH_GT] = anon_sym_SLASH_GT,
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
  [sym_directive_name] = sym_directive_name,
  [sym_comment] = sym_comment,
  [sym_script_raw_text] = sym_script_raw_text,
  [sym_style_raw_text] = sym_script_raw_text,
  [sym_i18n_raw_text] = sym_script_raw_text,
  [sym_document] = sym_document,
  [sym__content] = sym__content,
  [sym_script_element] = sym_script_element,
  [sym_script_start_tag] = sym_start_tag,
  [sym__script_end_tag] = sym_end_tag,
  [sym_style_element] = sym_style_element,
  [sym_style_start_tag] = sym_start_tag,
  [sym__style_end_tag] = sym_end_tag,
  [sym_i18n_element] = sym_i18n_element,
  [sym_i18n_start_tag] = sym_start_tag,
  [sym__i18n_end_tag] = sym_end_tag,
  [sym__script] = sym_tag_name,
  [sym__style] = sym_tag_name,
  [sym__i18n] = sym_tag_name,
  [sym_template_element] = sym_template_element,
  [sym_template_start_tag] = sym_start_tag,
  [sym__template_end_tag] = sym_end_tag,
  [sym__template] = sym_tag_name,
  [sym_start_tag] = sym_start_tag,
  [sym_self_closing_tag] = sym_self_closing_tag,
  [sym_end_tag] = sym_end_tag,
  [sym_interpolation] = sym_interpolation,
  [sym_attribute] = sym_attribute,
  [sym__plain_attribute] = sym__plain_attribute,
  [sym_quoted_attribute_value] = sym_quoted_attribute_value,
  [sym__double_quoted_text] = sym__double_quoted_text,
  [sym__single_quoted_text] = sym__double_quoted_text,
  [sym_directive] = sym_directive,
  [sym_directive_value] = sym_directive_value,
  [sym__double_quoted_expr] = sym__interpolation_text,
  [sym__single_quoted_expr] = sym__interpolation_text,
  [aux_sym_document_repeat1] = aux_sym_document_repeat1,
  [aux_sym_script_start_tag_repeat1] = aux_sym_script_start_tag_repeat1,
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
  [anon_sym_LT_SLASH] = {
    .visible = true,
    .named = false,
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
  [anon_sym_template] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SLASH_GT] = {
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
  [sym_directive_name] = {
    .visible = true,
    .named = true,
  },
  [sym_comment] = {
    .visible = true,
    .named = true,
  },
  [sym_script_raw_text] = {
    .visible = true,
    .named = true,
  },
  [sym_style_raw_text] = {
    .visible = true,
    .named = true,
  },
  [sym_i18n_raw_text] = {
    .visible = true,
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
  [sym__script_end_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_style_element] = {
    .visible = true,
    .named = true,
  },
  [sym_style_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__style_end_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_i18n_element] = {
    .visible = true,
    .named = true,
  },
  [sym_i18n_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__i18n_end_tag] = {
    .visible = true,
    .named = true,
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
  [sym_template_element] = {
    .visible = true,
    .named = true,
  },
  [sym_template_start_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__template_end_tag] = {
    .visible = true,
    .named = true,
  },
  [sym__template] = {
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
  [3] = {.index = 2, .length = 2},
  [4] = {.index = 4, .length = 2},
};

static const TSFieldMapEntry ts_field_map_entries[] = {
  [0] =
    {field_name, 1},
  [1] =
    {field_name, 0},
  [2] =
    {field_name, 0, .inherited = true},
    {field_value, 0, .inherited = true},
  [4] =
    {field_name, 0},
    {field_value, 2},
};

static const TSSymbol ts_alias_sequences[PRODUCTION_ID_COUNT][MAX_ALIAS_SEQUENCE_LENGTH] = {
  [0] = {0},
};

static const uint16_t ts_non_terminal_alias_map[] = {
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
};

static bool ts_lex(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      if (eof) ADVANCE(39);
      ADVANCE_MAP(
        '"', 94,
        '\'', 95,
        '/', 18,
        ':', 35,
        '<', 40,
        '=', 88,
        '>', 42,
        '@', 33,
        'i', 54,
        'p', 53,
        's', 57,
        't', 58,
        '{', 30,
        '}', 31,
      );
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(0);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 1:
      if (lookahead == '!') ADVANCE(4);
      END_STATE();
    case 2:
      if (lookahead == '"') ADVANCE(94);
      if (lookahead == '<') ADVANCE(96);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(102);
      if (lookahead != 0) ADVANCE(103);
      END_STATE();
    case 3:
      if (lookahead == '\'') ADVANCE(95);
      if (lookahead == '<') ADVANCE(104);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(110);
      if (lookahead != 0) ADVANCE(111);
      END_STATE();
    case 4:
      if (lookahead == '-') ADVANCE(6);
      END_STATE();
    case 5:
      if (lookahead == '-') ADVANCE(6);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(19);
      END_STATE();
    case 6:
      if (lookahead == '-') ADVANCE(8);
      END_STATE();
    case 7:
      if (lookahead == '-') ADVANCE(20);
      if (lookahead != 0) ADVANCE(8);
      END_STATE();
    case 8:
      if (lookahead == '-') ADVANCE(7);
      if (lookahead != 0) ADVANCE(8);
      END_STATE();
    case 9:
      if (lookahead == '-') ADVANCE(81);
      if (lookahead == '}') ADVANCE(8);
      if (lookahead != 0) ADVANCE(83);
      END_STATE();
    case 10:
      if (lookahead == '/') ADVANCE(18);
      if (lookahead == ':') ADVANCE(35);
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == '=') ADVANCE(88);
      if (lookahead == '>') ADVANCE(42);
      if (lookahead == '@') ADVANCE(33);
      if (lookahead == 'p') ADVANCE(89);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(10);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(93);
      END_STATE();
    case 11:
      if (lookahead == '1') ADVANCE(12);
      END_STATE();
    case 12:
      if (lookahead == '8') ADVANCE(25);
      END_STATE();
    case 13:
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == 'i') ADVANCE(54);
      if (lookahead == 's') ADVANCE(57);
      if (lookahead == 't') ADVANCE(58);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(13);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 14:
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == 'i') ADVANCE(11);
      if (lookahead == 's') ADVANCE(21);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(14);
      END_STATE();
    case 15:
      if (lookahead == '<') ADVANCE(1);
      if (lookahead == 't') ADVANCE(58);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(15);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 16:
      if (lookahead == '<') ADVANCE(1);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') SKIP(16);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 17:
      if (lookahead == '<') ADVANCE(80);
      if (lookahead == '}') ADVANCE(32);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(85);
      if (lookahead != 0) ADVANCE(87);
      END_STATE();
    case 18:
      if (lookahead == '>') ADVANCE(51);
      END_STATE();
    case 19:
      if (lookahead == '>') ADVANCE(52);
      if (lookahead != 0) ADVANCE(19);
      END_STATE();
    case 20:
      if (lookahead == '>') ADVANCE(119);
      if (lookahead != 0) ADVANCE(8);
      END_STATE();
    case 21:
      if (lookahead == 'c') ADVANCE(27);
      if (lookahead == 't') ADVANCE(29);
      END_STATE();
    case 22:
      if (lookahead == 'e') ADVANCE(46);
      END_STATE();
    case 23:
      if (lookahead == 'i') ADVANCE(26);
      END_STATE();
    case 24:
      if (lookahead == 'l') ADVANCE(22);
      END_STATE();
    case 25:
      if (lookahead == 'n') ADVANCE(48);
      END_STATE();
    case 26:
      if (lookahead == 'p') ADVANCE(28);
      END_STATE();
    case 27:
      if (lookahead == 'r') ADVANCE(23);
      END_STATE();
    case 28:
      if (lookahead == 't') ADVANCE(44);
      END_STATE();
    case 29:
      if (lookahead == 'y') ADVANCE(24);
      END_STATE();
    case 30:
      if (lookahead == '{') ADVANCE(78);
      END_STATE();
    case 31:
      if (lookahead == '}') ADVANCE(79);
      END_STATE();
    case 32:
      if (lookahead == '}') ADVANCE(79);
      if (lookahead != 0) ADVANCE(87);
      END_STATE();
    case 33:
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 34:
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(113);
      END_STATE();
    case 35:
      if (('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(118);
      END_STATE();
    case 36:
      if (lookahead != 0 &&
          lookahead != '!' &&
          lookahead != '/' &&
          (lookahead < 'A' || 'Z' < lookahead) &&
          (lookahead < 'a' || 'z' < lookahead)) ADVANCE(77);
      END_STATE();
    case 37:
      if (lookahead != 0 &&
          lookahead != '}') ADVANCE(87);
      END_STATE();
    case 38:
      if (eof) ADVANCE(39);
      if (lookahead == '<') ADVANCE(41);
      if (lookahead == '{') ADVANCE(76);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(75);
      if (lookahead != 0) ADVANCE(77);
      END_STATE();
    case 39:
      ACCEPT_TOKEN(ts_builtin_sym_end);
      END_STATE();
    case 40:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '!') ADVANCE(5);
      if (lookahead == '/') ADVANCE(43);
      END_STATE();
    case 41:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '!') ADVANCE(5);
      if (lookahead == '/') ADVANCE(43);
      if (lookahead != 0 &&
          (lookahead < 'A' || 'Z' < lookahead) &&
          (lookahead < 'a' || 'z' < lookahead)) ADVANCE(77);
      END_STATE();
    case 42:
      ACCEPT_TOKEN(anon_sym_GT);
      END_STATE();
    case 43:
      ACCEPT_TOKEN(anon_sym_LT_SLASH);
      END_STATE();
    case 44:
      ACCEPT_TOKEN(anon_sym_script);
      END_STATE();
    case 45:
      ACCEPT_TOKEN(anon_sym_script);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 46:
      ACCEPT_TOKEN(anon_sym_style);
      END_STATE();
    case 47:
      ACCEPT_TOKEN(anon_sym_style);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 48:
      ACCEPT_TOKEN(anon_sym_i18n);
      END_STATE();
    case 49:
      ACCEPT_TOKEN(anon_sym_i18n);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 50:
      ACCEPT_TOKEN(anon_sym_template);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 51:
      ACCEPT_TOKEN(anon_sym_SLASH_GT);
      END_STATE();
    case 52:
      ACCEPT_TOKEN(sym_doctype);
      END_STATE();
    case 53:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '-') ADVANCE(73);
      if (('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 54:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '1') ADVANCE(55);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 55:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '8') ADVANCE(65);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 56:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'a') ADVANCE(70);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('b' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 57:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'c') ADVANCE(68);
      if (lookahead == 't') ADVANCE(71);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 58:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'e') ADVANCE(64);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 59:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'e') ADVANCE(47);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 60:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'e') ADVANCE(50);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 61:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'i') ADVANCE(66);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 62:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'l') ADVANCE(56);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 63:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'l') ADVANCE(59);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 64:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'm') ADVANCE(67);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 65:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'n') ADVANCE(49);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 66:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'p') ADVANCE(69);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 67:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'p') ADVANCE(62);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 68:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'r') ADVANCE(61);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 69:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 't') ADVANCE(45);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 70:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 't') ADVANCE(60);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 71:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == 'y') ADVANCE(63);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 72:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          lookahead == '_') ADVANCE(74);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 73:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          lookahead == '_') ADVANCE(74);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(112);
      END_STATE();
    case 74:
      ACCEPT_TOKEN(sym_tag_name);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(74);
      END_STATE();
    case 75:
      ACCEPT_TOKEN(sym_text);
      if (lookahead == '<') ADVANCE(41);
      if (lookahead == '{') ADVANCE(76);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(75);
      if (lookahead != 0) ADVANCE(77);
      END_STATE();
    case 76:
      ACCEPT_TOKEN(sym_text);
      if (lookahead == '<') ADVANCE(36);
      if (lookahead == '{') ADVANCE(78);
      if (lookahead != 0) ADVANCE(77);
      END_STATE();
    case 77:
      ACCEPT_TOKEN(sym_text);
      if (lookahead == '<') ADVANCE(36);
      if (lookahead == '{') ADVANCE(77);
      if (lookahead != 0) ADVANCE(77);
      END_STATE();
    case 78:
      ACCEPT_TOKEN(anon_sym_LBRACE_LBRACE);
      END_STATE();
    case 79:
      ACCEPT_TOKEN(anon_sym_RBRACE_RBRACE);
      END_STATE();
    case 80:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '!') ADVANCE(84);
      if (lookahead == '}') ADVANCE(37);
      if (lookahead != 0) ADVANCE(87);
      END_STATE();
    case 81:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(86);
      if (lookahead == '}') ADVANCE(9);
      if (lookahead != 0) ADVANCE(83);
      END_STATE();
    case 82:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(83);
      if (lookahead == '}') ADVANCE(37);
      if (lookahead != 0) ADVANCE(87);
      END_STATE();
    case 83:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(81);
      if (lookahead == '}') ADVANCE(9);
      if (lookahead != 0) ADVANCE(83);
      END_STATE();
    case 84:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '-') ADVANCE(82);
      if (lookahead == '}') ADVANCE(37);
      if (lookahead != 0) ADVANCE(87);
      END_STATE();
    case 85:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '<') ADVANCE(80);
      if (lookahead == '}') ADVANCE(32);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(85);
      if (lookahead != 0) ADVANCE(87);
      END_STATE();
    case 86:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '>') ADVANCE(119);
      if (lookahead == '}') ADVANCE(9);
      if (lookahead != 0) ADVANCE(83);
      END_STATE();
    case 87:
      ACCEPT_TOKEN(sym__interpolation_text);
      if (lookahead == '}') ADVANCE(37);
      if (lookahead != 0) ADVANCE(87);
      END_STATE();
    case 88:
      ACCEPT_TOKEN(anon_sym_EQ);
      END_STATE();
    case 89:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-') ADVANCE(91);
      if (lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(93);
      END_STATE();
    case 90:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          lookahead == '_') ADVANCE(93);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(117);
      END_STATE();
    case 91:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          lookahead == '_') ADVANCE(93);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(115);
      END_STATE();
    case 92:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          lookahead == '_') ADVANCE(93);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(116);
      END_STATE();
    case 93:
      ACCEPT_TOKEN(sym_attribute_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(93);
      END_STATE();
    case 94:
      ACCEPT_TOKEN(anon_sym_DQUOTE);
      END_STATE();
    case 95:
      ACCEPT_TOKEN(anon_sym_SQUOTE);
      END_STATE();
    case 96:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '!') ADVANCE(101);
      if (lookahead != 0 &&
          lookahead != '!' &&
          lookahead != '"') ADVANCE(103);
      END_STATE();
    case 97:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '"') ADVANCE(8);
      if (lookahead == '-') ADVANCE(99);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 98:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '"') ADVANCE(8);
      if (lookahead == '-') ADVANCE(97);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 99:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '"') ADVANCE(8);
      if (lookahead == '>') ADVANCE(103);
      if (lookahead != 0) ADVANCE(98);
      END_STATE();
    case 100:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '-') ADVANCE(98);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(103);
      END_STATE();
    case 101:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '-') ADVANCE(100);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(103);
      END_STATE();
    case 102:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead == '<') ADVANCE(96);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(102);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(103);
      END_STATE();
    case 103:
      ACCEPT_TOKEN(aux_sym__double_quoted_text_token1);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(103);
      END_STATE();
    case 104:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '!') ADVANCE(109);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(111);
      END_STATE();
    case 105:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '\'') ADVANCE(8);
      if (lookahead == '-') ADVANCE(107);
      if (lookahead != 0) ADVANCE(106);
      END_STATE();
    case 106:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '\'') ADVANCE(8);
      if (lookahead == '-') ADVANCE(105);
      if (lookahead != 0) ADVANCE(106);
      END_STATE();
    case 107:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '\'') ADVANCE(8);
      if (lookahead == '>') ADVANCE(111);
      if (lookahead != 0) ADVANCE(106);
      END_STATE();
    case 108:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '-') ADVANCE(106);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(111);
      END_STATE();
    case 109:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '-') ADVANCE(108);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(111);
      END_STATE();
    case 110:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead == '<') ADVANCE(104);
      if (('\t' <= lookahead && lookahead <= '\r') ||
          lookahead == ' ') ADVANCE(110);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(111);
      END_STATE();
    case 111:
      ACCEPT_TOKEN(aux_sym__single_quoted_text_token1);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(111);
      END_STATE();
    case 112:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '.') ADVANCE(34);
      if (lookahead == ':') ADVANCE(72);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(112);
      END_STATE();
    case 113:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '.') ADVANCE(34);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(113);
      END_STATE();
    case 114:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '.') ADVANCE(34);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 115:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '.') ADVANCE(92);
      if (lookahead == ':') ADVANCE(90);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(115);
      END_STATE();
    case 116:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '.') ADVANCE(92);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(116);
      END_STATE();
    case 117:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '.') ADVANCE(92);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(117);
      END_STATE();
    case 118:
      ACCEPT_TOKEN(sym_directive_name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(118);
      END_STATE();
    case 119:
      ACCEPT_TOKEN(sym_comment);
      END_STATE();
    default:
      return false;
  }
}

static const TSLexMode ts_lex_modes[STATE_COUNT] = {
  [0] = {.lex_state = 0, .external_lex_state = 1},
  [1] = {.lex_state = 38},
  [2] = {.lex_state = 38},
  [3] = {.lex_state = 38},
  [4] = {.lex_state = 38},
  [5] = {.lex_state = 38},
  [6] = {.lex_state = 13},
  [7] = {.lex_state = 10},
  [8] = {.lex_state = 10},
  [9] = {.lex_state = 10},
  [10] = {.lex_state = 10},
  [11] = {.lex_state = 10},
  [12] = {.lex_state = 10},
  [13] = {.lex_state = 10},
  [14] = {.lex_state = 10},
  [15] = {.lex_state = 10},
  [16] = {.lex_state = 10},
  [17] = {.lex_state = 10},
  [18] = {.lex_state = 38},
  [19] = {.lex_state = 38},
  [20] = {.lex_state = 38},
  [21] = {.lex_state = 38},
  [22] = {.lex_state = 38},
  [23] = {.lex_state = 38},
  [24] = {.lex_state = 38},
  [25] = {.lex_state = 38},
  [26] = {.lex_state = 38},
  [27] = {.lex_state = 38},
  [28] = {.lex_state = 38},
  [29] = {.lex_state = 38},
  [30] = {.lex_state = 38},
  [31] = {.lex_state = 38},
  [32] = {.lex_state = 38},
  [33] = {.lex_state = 38},
  [34] = {.lex_state = 38},
  [35] = {.lex_state = 38},
  [36] = {.lex_state = 38},
  [37] = {.lex_state = 38},
  [38] = {.lex_state = 10},
  [39] = {.lex_state = 10},
  [40] = {.lex_state = 38},
  [41] = {.lex_state = 10},
  [42] = {.lex_state = 10},
  [43] = {.lex_state = 10},
  [44] = {.lex_state = 10},
  [45] = {.lex_state = 10},
  [46] = {.lex_state = 10},
  [47] = {.lex_state = 10},
  [48] = {.lex_state = 10},
  [49] = {.lex_state = 0, .external_lex_state = 2},
  [50] = {.lex_state = 10},
  [51] = {.lex_state = 10},
  [52] = {.lex_state = 15},
  [53] = {.lex_state = 0, .external_lex_state = 3},
  [54] = {.lex_state = 0, .external_lex_state = 4},
  [55] = {.lex_state = 0},
  [56] = {.lex_state = 0},
  [57] = {.lex_state = 10},
  [58] = {.lex_state = 2},
  [59] = {.lex_state = 3},
  [60] = {.lex_state = 10},
  [61] = {.lex_state = 2},
  [62] = {.lex_state = 3},
  [63] = {.lex_state = 0},
  [64] = {.lex_state = 14},
  [65] = {.lex_state = 14},
  [66] = {.lex_state = 0, .external_lex_state = 3},
  [67] = {.lex_state = 0, .external_lex_state = 2},
  [68] = {.lex_state = 0, .external_lex_state = 4},
  [69] = {.lex_state = 0, .external_lex_state = 3},
  [70] = {.lex_state = 0},
  [71] = {.lex_state = 17},
  [72] = {.lex_state = 0, .external_lex_state = 2},
  [73] = {.lex_state = 0},
  [74] = {.lex_state = 0, .external_lex_state = 4},
  [75] = {.lex_state = 14},
  [76] = {.lex_state = 0},
  [77] = {.lex_state = 0},
  [78] = {.lex_state = 0},
  [79] = {.lex_state = 0},
  [80] = {.lex_state = 0},
  [81] = {.lex_state = 0},
  [82] = {.lex_state = 16},
  [83] = {.lex_state = 0},
  [84] = {.lex_state = 0},
  [85] = {.lex_state = 0},
  [86] = {.lex_state = 0},
  [87] = {.lex_state = 0},
  [88] = {.lex_state = 0},
  [89] = {.lex_state = 0},
  [90] = {.lex_state = 0},
  [91] = {.lex_state = 0},
};

static const uint16_t ts_parse_table[LARGE_STATE_COUNT][SYMBOL_COUNT] = {
  [0] = {
    [ts_builtin_sym_end] = ACTIONS(1),
    [anon_sym_LT] = ACTIONS(1),
    [anon_sym_GT] = ACTIONS(1),
    [anon_sym_LT_SLASH] = ACTIONS(1),
    [anon_sym_script] = ACTIONS(1),
    [anon_sym_style] = ACTIONS(1),
    [anon_sym_i18n] = ACTIONS(1),
    [anon_sym_template] = ACTIONS(1),
    [anon_sym_SLASH_GT] = ACTIONS(1),
    [sym_doctype] = ACTIONS(1),
    [sym_tag_name] = ACTIONS(1),
    [anon_sym_LBRACE_LBRACE] = ACTIONS(1),
    [anon_sym_RBRACE_RBRACE] = ACTIONS(1),
    [anon_sym_EQ] = ACTIONS(1),
    [anon_sym_DQUOTE] = ACTIONS(1),
    [anon_sym_SQUOTE] = ACTIONS(1),
    [sym_directive_name] = ACTIONS(1),
    [sym_comment] = ACTIONS(3),
    [sym_script_raw_text] = ACTIONS(1),
    [sym_style_raw_text] = ACTIONS(1),
    [sym_i18n_raw_text] = ACTIONS(1),
  },
  [1] = {
    [sym_document] = STATE(91),
    [sym__content] = STATE(2),
    [sym_script_element] = STATE(2),
    [sym_script_start_tag] = STATE(53),
    [sym_style_element] = STATE(2),
    [sym_style_start_tag] = STATE(49),
    [sym_i18n_element] = STATE(2),
    [sym_i18n_start_tag] = STATE(54),
    [sym_template_element] = STATE(2),
    [sym_template_start_tag] = STATE(4),
    [sym_start_tag] = STATE(2),
    [sym_self_closing_tag] = STATE(2),
    [sym_end_tag] = STATE(2),
    [sym_interpolation] = STATE(2),
    [aux_sym_document_repeat1] = STATE(2),
    [ts_builtin_sym_end] = ACTIONS(5),
    [anon_sym_LT] = ACTIONS(7),
    [anon_sym_LT_SLASH] = ACTIONS(9),
    [sym_doctype] = ACTIONS(11),
    [sym_text] = ACTIONS(11),
    [anon_sym_LBRACE_LBRACE] = ACTIONS(13),
    [sym_comment] = ACTIONS(15),
  },
};

static const uint16_t ts_small_parse_table[] = {
  [0] = 11,
    ACTIONS(7), 1,
      anon_sym_LT,
    ACTIONS(9), 1,
      anon_sym_LT_SLASH,
    ACTIONS(13), 1,
      anon_sym_LBRACE_LBRACE,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(17), 1,
      ts_builtin_sym_end,
    STATE(4), 1,
      sym_template_start_tag,
    STATE(49), 1,
      sym_style_start_tag,
    STATE(53), 1,
      sym_script_start_tag,
    STATE(54), 1,
      sym_i18n_start_tag,
    ACTIONS(19), 2,
      sym_doctype,
      sym_text,
    STATE(3), 10,
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
  [44] = 11,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(21), 1,
      ts_builtin_sym_end,
    ACTIONS(23), 1,
      anon_sym_LT,
    ACTIONS(26), 1,
      anon_sym_LT_SLASH,
    ACTIONS(32), 1,
      anon_sym_LBRACE_LBRACE,
    STATE(4), 1,
      sym_template_start_tag,
    STATE(49), 1,
      sym_style_start_tag,
    STATE(53), 1,
      sym_script_start_tag,
    STATE(54), 1,
      sym_i18n_start_tag,
    ACTIONS(29), 2,
      sym_doctype,
      sym_text,
    STATE(3), 10,
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
  [88] = 11,
    ACTIONS(7), 1,
      anon_sym_LT,
    ACTIONS(13), 1,
      anon_sym_LBRACE_LBRACE,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(35), 1,
      anon_sym_LT_SLASH,
    STATE(4), 1,
      sym_template_start_tag,
    STATE(24), 1,
      sym__template_end_tag,
    STATE(49), 1,
      sym_style_start_tag,
    STATE(53), 1,
      sym_script_start_tag,
    STATE(54), 1,
      sym_i18n_start_tag,
    ACTIONS(37), 2,
      sym_doctype,
      sym_text,
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
  [132] = 11,
    ACTIONS(7), 1,
      anon_sym_LT,
    ACTIONS(13), 1,
      anon_sym_LBRACE_LBRACE,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(35), 1,
      anon_sym_LT_SLASH,
    STATE(4), 1,
      sym_template_start_tag,
    STATE(29), 1,
      sym__template_end_tag,
    STATE(49), 1,
      sym_style_start_tag,
    STATE(53), 1,
      sym_script_start_tag,
    STATE(54), 1,
      sym_i18n_start_tag,
    ACTIONS(19), 2,
      sym_doctype,
      sym_text,
    STATE(3), 10,
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
  [176] = 10,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(39), 1,
      anon_sym_script,
    ACTIONS(41), 1,
      anon_sym_style,
    ACTIONS(43), 1,
      anon_sym_i18n,
    ACTIONS(45), 1,
      anon_sym_template,
    ACTIONS(47), 1,
      sym_tag_name,
    STATE(12), 1,
      sym__script,
    STATE(13), 1,
      sym__style,
    STATE(14), 1,
      sym__i18n,
    STATE(15), 1,
      sym__template,
  [207] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(49), 1,
      anon_sym_GT,
    ACTIONS(51), 1,
      anon_sym_SLASH_GT,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(9), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [233] = 8,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(57), 1,
      anon_sym_GT,
    ACTIONS(59), 1,
      anon_sym_SLASH_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(7), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [259] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(63), 1,
      sym_attribute_name,
    ACTIONS(66), 1,
      sym_directive_name,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    ACTIONS(61), 2,
      anon_sym_GT,
      anon_sym_SLASH_GT,
    STATE(9), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [283] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(69), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(9), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [306] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(71), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(9), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [329] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(73), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(10), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [352] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(75), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(16), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [375] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(77), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(17), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [398] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(79), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(11), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [421] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(81), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(9), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [444] = 7,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(53), 1,
      sym_attribute_name,
    ACTIONS(55), 1,
      sym_directive_name,
    ACTIONS(83), 1,
      anon_sym_GT,
    STATE(41), 1,
      sym_directive,
    STATE(42), 1,
      sym__plain_attribute,
    STATE(9), 2,
      sym_attribute,
      aux_sym_script_start_tag_repeat1,
  [467] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(85), 1,
      ts_builtin_sym_end,
    ACTIONS(87), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [481] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(89), 1,
      ts_builtin_sym_end,
    ACTIONS(91), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [495] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(93), 1,
      ts_builtin_sym_end,
    ACTIONS(95), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [509] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(97), 1,
      ts_builtin_sym_end,
    ACTIONS(99), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [523] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(101), 1,
      ts_builtin_sym_end,
    ACTIONS(103), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [537] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(105), 1,
      ts_builtin_sym_end,
    ACTIONS(107), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [551] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(109), 1,
      ts_builtin_sym_end,
    ACTIONS(111), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [565] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(113), 1,
      ts_builtin_sym_end,
    ACTIONS(115), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [579] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(117), 1,
      ts_builtin_sym_end,
    ACTIONS(119), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [593] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(121), 1,
      ts_builtin_sym_end,
    ACTIONS(123), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [607] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(125), 1,
      ts_builtin_sym_end,
    ACTIONS(127), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [621] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(129), 1,
      ts_builtin_sym_end,
    ACTIONS(131), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [635] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(133), 1,
      ts_builtin_sym_end,
    ACTIONS(135), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [649] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(137), 1,
      ts_builtin_sym_end,
    ACTIONS(139), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [663] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(141), 1,
      ts_builtin_sym_end,
    ACTIONS(143), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [677] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(145), 1,
      ts_builtin_sym_end,
    ACTIONS(147), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [691] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(149), 1,
      ts_builtin_sym_end,
    ACTIONS(151), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [705] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(153), 1,
      ts_builtin_sym_end,
    ACTIONS(155), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [719] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(157), 1,
      ts_builtin_sym_end,
    ACTIONS(159), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [733] = 2,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(161), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [744] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(165), 1,
      anon_sym_EQ,
    ACTIONS(167), 1,
      sym_attribute_name,
    ACTIONS(163), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [759] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(171), 1,
      anon_sym_EQ,
    ACTIONS(173), 1,
      sym_attribute_name,
    ACTIONS(169), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [774] = 2,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(175), 5,
      anon_sym_LT,
      anon_sym_LT_SLASH,
      sym_doctype,
      sym_text,
      anon_sym_LBRACE_LBRACE,
  [785] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(179), 1,
      sym_attribute_name,
    ACTIONS(177), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [797] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(183), 1,
      sym_attribute_name,
    ACTIONS(181), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [809] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(187), 1,
      sym_attribute_name,
    ACTIONS(185), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [821] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(191), 1,
      sym_attribute_name,
    ACTIONS(189), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [833] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(195), 1,
      sym_attribute_name,
    ACTIONS(193), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [845] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(199), 1,
      sym_attribute_name,
    ACTIONS(197), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [857] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(203), 1,
      sym_attribute_name,
    ACTIONS(201), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [869] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(207), 1,
      sym_attribute_name,
    ACTIONS(205), 3,
      anon_sym_GT,
      anon_sym_SLASH_GT,
      sym_directive_name,
  [881] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(209), 1,
      anon_sym_LT_SLASH,
    ACTIONS(211), 1,
      sym_style_raw_text,
    STATE(19), 1,
      sym__style_end_tag,
  [894] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(215), 1,
      sym_attribute_name,
    ACTIONS(213), 2,
      anon_sym_GT,
      sym_directive_name,
  [905] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(219), 1,
      sym_attribute_name,
    ACTIONS(217), 2,
      anon_sym_GT,
      sym_directive_name,
  [916] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(45), 1,
      anon_sym_template,
    ACTIONS(221), 1,
      sym_tag_name,
    STATE(79), 1,
      sym__template,
  [929] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(223), 1,
      anon_sym_LT_SLASH,
    ACTIONS(225), 1,
      sym_script_raw_text,
    STATE(18), 1,
      sym__script_end_tag,
  [942] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(227), 1,
      anon_sym_LT_SLASH,
    ACTIONS(229), 1,
      sym_i18n_raw_text,
    STATE(23), 1,
      sym__i18n_end_tag,
  [955] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(231), 1,
      anon_sym_DQUOTE,
    ACTIONS(233), 1,
      anon_sym_SQUOTE,
    STATE(43), 1,
      sym_quoted_attribute_value,
  [968] = 4,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(235), 1,
      anon_sym_DQUOTE,
    ACTIONS(237), 1,
      anon_sym_SQUOTE,
    STATE(44), 1,
      sym_directive_value,
  [981] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(241), 1,
      sym_attribute_name,
    ACTIONS(239), 2,
      anon_sym_GT,
      sym_directive_name,
  [992] = 4,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(243), 1,
      anon_sym_DQUOTE,
    ACTIONS(245), 1,
      aux_sym__double_quoted_text_token1,
    STATE(83), 1,
      sym__double_quoted_text,
  [1005] = 4,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(243), 1,
      anon_sym_SQUOTE,
    ACTIONS(247), 1,
      aux_sym__single_quoted_text_token1,
    STATE(84), 1,
      sym__single_quoted_text,
  [1018] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(251), 1,
      sym_attribute_name,
    ACTIONS(249), 2,
      anon_sym_GT,
      sym_directive_name,
  [1029] = 4,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(253), 1,
      anon_sym_DQUOTE,
    ACTIONS(255), 1,
      aux_sym__double_quoted_text_token1,
    STATE(87), 1,
      sym__double_quoted_expr,
  [1042] = 4,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(253), 1,
      anon_sym_SQUOTE,
    ACTIONS(257), 1,
      aux_sym__single_quoted_text_token1,
    STATE(90), 1,
      sym__single_quoted_expr,
  [1055] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(227), 1,
      anon_sym_LT_SLASH,
    STATE(28), 1,
      sym__i18n_end_tag,
  [1065] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(259), 1,
      anon_sym_i18n,
    STATE(78), 1,
      sym__i18n,
  [1075] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(261), 1,
      anon_sym_script,
    STATE(80), 1,
      sym__script,
  [1085] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(263), 2,
      sym_script_raw_text,
      anon_sym_LT_SLASH,
  [1093] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(265), 2,
      sym_style_raw_text,
      anon_sym_LT_SLASH,
  [1101] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(267), 2,
      sym_i18n_raw_text,
      anon_sym_LT_SLASH,
  [1109] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(269), 2,
      sym_script_raw_text,
      anon_sym_LT_SLASH,
  [1117] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(223), 1,
      anon_sym_LT_SLASH,
    STATE(25), 1,
      sym__script_end_tag,
  [1127] = 3,
    ACTIONS(15), 1,
      sym_comment,
    ACTIONS(271), 1,
      anon_sym_RBRACE_RBRACE,
    ACTIONS(273), 1,
      sym__interpolation_text,
  [1137] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(275), 2,
      sym_style_raw_text,
      anon_sym_LT_SLASH,
  [1145] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(209), 1,
      anon_sym_LT_SLASH,
    STATE(27), 1,
      sym__style_end_tag,
  [1155] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(277), 2,
      sym_i18n_raw_text,
      anon_sym_LT_SLASH,
  [1163] = 3,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(279), 1,
      anon_sym_style,
    STATE(81), 1,
      sym__style,
  [1173] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(281), 1,
      anon_sym_GT,
  [1180] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(283), 1,
      anon_sym_RBRACE_RBRACE,
  [1187] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(285), 1,
      anon_sym_GT,
  [1194] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(287), 1,
      anon_sym_GT,
  [1201] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(289), 1,
      anon_sym_GT,
  [1208] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(291), 1,
      anon_sym_GT,
  [1215] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(293), 1,
      sym_tag_name,
  [1222] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(295), 1,
      anon_sym_DQUOTE,
  [1229] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(295), 1,
      anon_sym_SQUOTE,
  [1236] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(297), 1,
      anon_sym_SQUOTE,
  [1243] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(299), 1,
      anon_sym_DQUOTE,
  [1250] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(301), 1,
      anon_sym_DQUOTE,
  [1257] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(303), 1,
      anon_sym_SQUOTE,
  [1264] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(305), 1,
      anon_sym_DQUOTE,
  [1271] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(301), 1,
      anon_sym_SQUOTE,
  [1278] = 2,
    ACTIONS(3), 1,
      sym_comment,
    ACTIONS(307), 1,
      ts_builtin_sym_end,
};

static const uint32_t ts_small_parse_table_map[] = {
  [SMALL_STATE(2)] = 0,
  [SMALL_STATE(3)] = 44,
  [SMALL_STATE(4)] = 88,
  [SMALL_STATE(5)] = 132,
  [SMALL_STATE(6)] = 176,
  [SMALL_STATE(7)] = 207,
  [SMALL_STATE(8)] = 233,
  [SMALL_STATE(9)] = 259,
  [SMALL_STATE(10)] = 283,
  [SMALL_STATE(11)] = 306,
  [SMALL_STATE(12)] = 329,
  [SMALL_STATE(13)] = 352,
  [SMALL_STATE(14)] = 375,
  [SMALL_STATE(15)] = 398,
  [SMALL_STATE(16)] = 421,
  [SMALL_STATE(17)] = 444,
  [SMALL_STATE(18)] = 467,
  [SMALL_STATE(19)] = 481,
  [SMALL_STATE(20)] = 495,
  [SMALL_STATE(21)] = 509,
  [SMALL_STATE(22)] = 523,
  [SMALL_STATE(23)] = 537,
  [SMALL_STATE(24)] = 551,
  [SMALL_STATE(25)] = 565,
  [SMALL_STATE(26)] = 579,
  [SMALL_STATE(27)] = 593,
  [SMALL_STATE(28)] = 607,
  [SMALL_STATE(29)] = 621,
  [SMALL_STATE(30)] = 635,
  [SMALL_STATE(31)] = 649,
  [SMALL_STATE(32)] = 663,
  [SMALL_STATE(33)] = 677,
  [SMALL_STATE(34)] = 691,
  [SMALL_STATE(35)] = 705,
  [SMALL_STATE(36)] = 719,
  [SMALL_STATE(37)] = 733,
  [SMALL_STATE(38)] = 744,
  [SMALL_STATE(39)] = 759,
  [SMALL_STATE(40)] = 774,
  [SMALL_STATE(41)] = 785,
  [SMALL_STATE(42)] = 797,
  [SMALL_STATE(43)] = 809,
  [SMALL_STATE(44)] = 821,
  [SMALL_STATE(45)] = 833,
  [SMALL_STATE(46)] = 845,
  [SMALL_STATE(47)] = 857,
  [SMALL_STATE(48)] = 869,
  [SMALL_STATE(49)] = 881,
  [SMALL_STATE(50)] = 894,
  [SMALL_STATE(51)] = 905,
  [SMALL_STATE(52)] = 916,
  [SMALL_STATE(53)] = 929,
  [SMALL_STATE(54)] = 942,
  [SMALL_STATE(55)] = 955,
  [SMALL_STATE(56)] = 968,
  [SMALL_STATE(57)] = 981,
  [SMALL_STATE(58)] = 992,
  [SMALL_STATE(59)] = 1005,
  [SMALL_STATE(60)] = 1018,
  [SMALL_STATE(61)] = 1029,
  [SMALL_STATE(62)] = 1042,
  [SMALL_STATE(63)] = 1055,
  [SMALL_STATE(64)] = 1065,
  [SMALL_STATE(65)] = 1075,
  [SMALL_STATE(66)] = 1085,
  [SMALL_STATE(67)] = 1093,
  [SMALL_STATE(68)] = 1101,
  [SMALL_STATE(69)] = 1109,
  [SMALL_STATE(70)] = 1117,
  [SMALL_STATE(71)] = 1127,
  [SMALL_STATE(72)] = 1137,
  [SMALL_STATE(73)] = 1145,
  [SMALL_STATE(74)] = 1155,
  [SMALL_STATE(75)] = 1163,
  [SMALL_STATE(76)] = 1173,
  [SMALL_STATE(77)] = 1180,
  [SMALL_STATE(78)] = 1187,
  [SMALL_STATE(79)] = 1194,
  [SMALL_STATE(80)] = 1201,
  [SMALL_STATE(81)] = 1208,
  [SMALL_STATE(82)] = 1215,
  [SMALL_STATE(83)] = 1222,
  [SMALL_STATE(84)] = 1229,
  [SMALL_STATE(85)] = 1236,
  [SMALL_STATE(86)] = 1243,
  [SMALL_STATE(87)] = 1250,
  [SMALL_STATE(88)] = 1257,
  [SMALL_STATE(89)] = 1264,
  [SMALL_STATE(90)] = 1271,
  [SMALL_STATE(91)] = 1278,
};

static const TSParseActionEntry ts_parse_actions[] = {
  [0] = {.entry = {.count = 0, .reusable = false}},
  [1] = {.entry = {.count = 1, .reusable = false}}, RECOVER(),
  [3] = {.entry = {.count = 1, .reusable = true}}, SHIFT_EXTRA(),
  [5] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_document, 0, 0, 0),
  [7] = {.entry = {.count = 1, .reusable = false}}, SHIFT(6),
  [9] = {.entry = {.count = 1, .reusable = false}}, SHIFT(82),
  [11] = {.entry = {.count = 1, .reusable = false}}, SHIFT(2),
  [13] = {.entry = {.count = 1, .reusable = false}}, SHIFT(71),
  [15] = {.entry = {.count = 1, .reusable = false}}, SHIFT_EXTRA(),
  [17] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_document, 1, 0, 0),
  [19] = {.entry = {.count = 1, .reusable = false}}, SHIFT(3),
  [21] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0),
  [23] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(6),
  [26] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(82),
  [29] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(3),
  [32] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_document_repeat1, 2, 0, 0), SHIFT_REPEAT(71),
  [35] = {.entry = {.count = 1, .reusable = false}}, SHIFT(52),
  [37] = {.entry = {.count = 1, .reusable = false}}, SHIFT(5),
  [39] = {.entry = {.count = 1, .reusable = false}}, SHIFT(57),
  [41] = {.entry = {.count = 1, .reusable = false}}, SHIFT(60),
  [43] = {.entry = {.count = 1, .reusable = false}}, SHIFT(50),
  [45] = {.entry = {.count = 1, .reusable = false}}, SHIFT(51),
  [47] = {.entry = {.count = 1, .reusable = false}}, SHIFT(8),
  [49] = {.entry = {.count = 1, .reusable = true}}, SHIFT(30),
  [51] = {.entry = {.count = 1, .reusable = true}}, SHIFT(31),
  [53] = {.entry = {.count = 1, .reusable = false}}, SHIFT(38),
  [55] = {.entry = {.count = 1, .reusable = true}}, SHIFT(39),
  [57] = {.entry = {.count = 1, .reusable = true}}, SHIFT(22),
  [59] = {.entry = {.count = 1, .reusable = true}}, SHIFT(21),
  [61] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_script_start_tag_repeat1, 2, 0, 0),
  [63] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_script_start_tag_repeat1, 2, 0, 0), SHIFT_REPEAT(38),
  [66] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_script_start_tag_repeat1, 2, 0, 0), SHIFT_REPEAT(39),
  [69] = {.entry = {.count = 1, .reusable = true}}, SHIFT(66),
  [71] = {.entry = {.count = 1, .reusable = true}}, SHIFT(40),
  [73] = {.entry = {.count = 1, .reusable = true}}, SHIFT(69),
  [75] = {.entry = {.count = 1, .reusable = true}}, SHIFT(72),
  [77] = {.entry = {.count = 1, .reusable = true}}, SHIFT(74),
  [79] = {.entry = {.count = 1, .reusable = true}}, SHIFT(37),
  [81] = {.entry = {.count = 1, .reusable = true}}, SHIFT(67),
  [83] = {.entry = {.count = 1, .reusable = true}}, SHIFT(68),
  [85] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_element, 2, 0, 0),
  [87] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_script_element, 2, 0, 0),
  [89] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_element, 2, 0, 0),
  [91] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_style_element, 2, 0, 0),
  [93] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_end_tag, 3, 0, 1),
  [95] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_end_tag, 3, 0, 1),
  [97] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_self_closing_tag, 3, 0, 1),
  [99] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_self_closing_tag, 3, 0, 1),
  [101] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_start_tag, 3, 0, 1),
  [103] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_start_tag, 3, 0, 1),
  [105] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_element, 2, 0, 0),
  [107] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_i18n_element, 2, 0, 0),
  [109] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_element, 2, 0, 0),
  [111] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_element, 2, 0, 0),
  [113] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_element, 3, 0, 0),
  [115] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_script_element, 3, 0, 0),
  [117] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_interpolation, 3, 0, 0),
  [119] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_interpolation, 3, 0, 0),
  [121] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_element, 3, 0, 0),
  [123] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_style_element, 3, 0, 0),
  [125] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_element, 3, 0, 0),
  [127] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_i18n_element, 3, 0, 0),
  [129] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_template_element, 3, 0, 0),
  [131] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_element, 3, 0, 0),
  [133] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_start_tag, 4, 0, 1),
  [135] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_start_tag, 4, 0, 1),
  [137] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_self_closing_tag, 4, 0, 1),
  [139] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_self_closing_tag, 4, 0, 1),
  [141] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_interpolation, 2, 0, 0),
  [143] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_interpolation, 2, 0, 0),
  [145] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__style_end_tag, 3, 0, 0),
  [147] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__style_end_tag, 3, 0, 0),
  [149] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__script_end_tag, 3, 0, 0),
  [151] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__script_end_tag, 3, 0, 0),
  [153] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__template_end_tag, 3, 0, 0),
  [155] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__template_end_tag, 3, 0, 0),
  [157] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__i18n_end_tag, 3, 0, 0),
  [159] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__i18n_end_tag, 3, 0, 0),
  [161] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_start_tag, 3, 0, 0),
  [163] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__plain_attribute, 1, 0, 2),
  [165] = {.entry = {.count = 1, .reusable = true}}, SHIFT(55),
  [167] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__plain_attribute, 1, 0, 2),
  [169] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 1, 0, 2),
  [171] = {.entry = {.count = 1, .reusable = true}}, SHIFT(56),
  [173] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 1, 0, 2),
  [175] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_template_start_tag, 4, 0, 0),
  [177] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute, 1, 0, 0),
  [179] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_attribute, 1, 0, 0),
  [181] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_attribute, 1, 0, 3),
  [183] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_attribute, 1, 0, 3),
  [185] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__plain_attribute, 3, 0, 4),
  [187] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__plain_attribute, 3, 0, 4),
  [189] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive, 3, 0, 4),
  [191] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive, 3, 0, 4),
  [193] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_quoted_attribute_value, 2, 0, 0),
  [195] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_quoted_attribute_value, 2, 0, 0),
  [197] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive_value, 2, 0, 0),
  [199] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive_value, 2, 0, 0),
  [201] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_quoted_attribute_value, 3, 0, 0),
  [203] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_quoted_attribute_value, 3, 0, 0),
  [205] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_directive_value, 3, 0, 0),
  [207] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_directive_value, 3, 0, 0),
  [209] = {.entry = {.count = 1, .reusable = true}}, SHIFT(75),
  [211] = {.entry = {.count = 1, .reusable = true}}, SHIFT(73),
  [213] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__i18n, 1, 0, 0),
  [215] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__i18n, 1, 0, 0),
  [217] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__template, 1, 0, 0),
  [219] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__template, 1, 0, 0),
  [221] = {.entry = {.count = 1, .reusable = false}}, SHIFT(76),
  [223] = {.entry = {.count = 1, .reusable = true}}, SHIFT(65),
  [225] = {.entry = {.count = 1, .reusable = true}}, SHIFT(70),
  [227] = {.entry = {.count = 1, .reusable = true}}, SHIFT(64),
  [229] = {.entry = {.count = 1, .reusable = true}}, SHIFT(63),
  [231] = {.entry = {.count = 1, .reusable = true}}, SHIFT(58),
  [233] = {.entry = {.count = 1, .reusable = true}}, SHIFT(59),
  [235] = {.entry = {.count = 1, .reusable = true}}, SHIFT(61),
  [237] = {.entry = {.count = 1, .reusable = true}}, SHIFT(62),
  [239] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__script, 1, 0, 0),
  [241] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__script, 1, 0, 0),
  [243] = {.entry = {.count = 1, .reusable = false}}, SHIFT(45),
  [245] = {.entry = {.count = 1, .reusable = false}}, SHIFT(86),
  [247] = {.entry = {.count = 1, .reusable = false}}, SHIFT(85),
  [249] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__style, 1, 0, 0),
  [251] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__style, 1, 0, 0),
  [253] = {.entry = {.count = 1, .reusable = false}}, SHIFT(46),
  [255] = {.entry = {.count = 1, .reusable = false}}, SHIFT(89),
  [257] = {.entry = {.count = 1, .reusable = false}}, SHIFT(88),
  [259] = {.entry = {.count = 1, .reusable = true}}, SHIFT(50),
  [261] = {.entry = {.count = 1, .reusable = true}}, SHIFT(57),
  [263] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_start_tag, 4, 0, 0),
  [265] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_start_tag, 4, 0, 0),
  [267] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_start_tag, 4, 0, 0),
  [269] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_script_start_tag, 3, 0, 0),
  [271] = {.entry = {.count = 1, .reusable = false}}, SHIFT(32),
  [273] = {.entry = {.count = 1, .reusable = false}}, SHIFT(77),
  [275] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_style_start_tag, 3, 0, 0),
  [277] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_i18n_start_tag, 3, 0, 0),
  [279] = {.entry = {.count = 1, .reusable = true}}, SHIFT(60),
  [281] = {.entry = {.count = 1, .reusable = true}}, SHIFT(20),
  [283] = {.entry = {.count = 1, .reusable = true}}, SHIFT(26),
  [285] = {.entry = {.count = 1, .reusable = true}}, SHIFT(36),
  [287] = {.entry = {.count = 1, .reusable = true}}, SHIFT(35),
  [289] = {.entry = {.count = 1, .reusable = true}}, SHIFT(34),
  [291] = {.entry = {.count = 1, .reusable = true}}, SHIFT(33),
  [293] = {.entry = {.count = 1, .reusable = true}}, SHIFT(76),
  [295] = {.entry = {.count = 1, .reusable = true}}, SHIFT(47),
  [297] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__single_quoted_text, 1, 0, 0),
  [299] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__double_quoted_text, 1, 0, 0),
  [301] = {.entry = {.count = 1, .reusable = true}}, SHIFT(48),
  [303] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__single_quoted_expr, 1, 0, 0),
  [305] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__double_quoted_expr, 1, 0, 0),
  [307] = {.entry = {.count = 1, .reusable = true}},  ACCEPT_INPUT(),
};

enum ts_external_scanner_symbol_identifiers {
  ts_external_token_script_raw_text = 0,
  ts_external_token_style_raw_text = 1,
  ts_external_token_i18n_raw_text = 2,
};

static const TSSymbol ts_external_scanner_symbol_map[EXTERNAL_TOKEN_COUNT] = {
  [ts_external_token_script_raw_text] = sym_script_raw_text,
  [ts_external_token_style_raw_text] = sym_style_raw_text,
  [ts_external_token_i18n_raw_text] = sym_i18n_raw_text,
};

static const bool ts_external_scanner_states[5][EXTERNAL_TOKEN_COUNT] = {
  [1] = {
    [ts_external_token_script_raw_text] = true,
    [ts_external_token_style_raw_text] = true,
    [ts_external_token_i18n_raw_text] = true,
  },
  [2] = {
    [ts_external_token_style_raw_text] = true,
  },
  [3] = {
    [ts_external_token_script_raw_text] = true,
  },
  [4] = {
    [ts_external_token_i18n_raw_text] = true,
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
