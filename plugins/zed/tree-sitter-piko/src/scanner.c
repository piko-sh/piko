/*
 * External scanner for tree-sitter-piko.
 *
 * It scans the body of a raw-text single-file block (script, style, i18n) up to
 * that block's specific close tag, so a "</" appearing inside embedded Go or
 * JavaScript (for example a string literal "</strong>") does not terminate the
 * block early.
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

#include "tree_sitter/parser.h"

#include <stddef.h>

enum TokenType {
  SCRIPT_RAW_TEXT,
  STYLE_RAW_TEXT,
  I18N_RAW_TEXT,
};

static char ascii_tolower(int32_t character) {
  if (character >= 'A' && character <= 'Z') {
    return (char)(character + ('a' - 'A'));
  }
  return (char)character;
}

void *tree_sitter_piko_external_scanner_create(void) { return NULL; }

void tree_sitter_piko_external_scanner_destroy(void *payload) { (void)payload; }

unsigned tree_sitter_piko_external_scanner_serialize(void *payload, char *buffer) {
  (void)payload;
  (void)buffer;
  return 0;
}

void tree_sitter_piko_external_scanner_deserialize(void *payload, const char *buffer,
                                                   unsigned length) {
  (void)payload;
  (void)buffer;
  (void)length;
}

static bool match_close_tag_name(TSLexer *lexer, const char *tag) {
  for (const char *cursor = tag; *cursor != '\0'; cursor++) {
    if (ascii_tolower(lexer->lookahead) != *cursor) {
      return false;
    }
    lexer->advance(lexer, false);
  }
  return true;
}

static bool at_close_tag(TSLexer *lexer, const char *tag) {
  lexer->mark_end(lexer);
  lexer->advance(lexer, false);
  if (lexer->lookahead != '/') {
    return false;
  }
  lexer->advance(lexer, false);
  return match_close_tag_name(lexer, tag);
}

static bool scan_raw_text(TSLexer *lexer, const char *tag) {
  bool has_content = false;

  while (!lexer->eof(lexer)) {
    if (lexer->lookahead == '<') {
      if (at_close_tag(lexer, tag)) {
        return has_content;
      }
    } else {
      lexer->advance(lexer, false);
    }
    has_content = true;
  }

  lexer->mark_end(lexer);
  return has_content;
}

bool tree_sitter_piko_external_scanner_scan(void *payload, TSLexer *lexer,
                                            const bool *valid_symbols) {
  (void)payload;

  if (valid_symbols[SCRIPT_RAW_TEXT]) {
    lexer->result_symbol = SCRIPT_RAW_TEXT;
    return scan_raw_text(lexer, "script");
  }
  if (valid_symbols[STYLE_RAW_TEXT]) {
    lexer->result_symbol = STYLE_RAW_TEXT;
    return scan_raw_text(lexer, "style");
  }
  if (valid_symbols[I18N_RAW_TEXT]) {
    lexer->result_symbol = I18N_RAW_TEXT;
    return scan_raw_text(lexer, "i18n");
  }

  return false;
}
