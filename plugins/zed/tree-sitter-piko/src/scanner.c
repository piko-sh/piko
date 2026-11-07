/*
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

#include <stdbool.h>
#include <stdint.h>

enum TokenType {
  SCRIPT_RAW_CHUNK,
  STYLE_RAW_CHUNK,
  I18N_RAW_CHUNK,
  SCRIPT_INNER_CLOSE,
  STYLE_INNER_CLOSE,
  I18N_INNER_CLOSE,
};

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

static bool is_tag_name_terminator(int32_t character) {
  return character == '>' || character == '/' || character == ' ' || character == '\t' ||
         character == '\n' || character == '\r' || character == '\f';
}

static bool advance_literal(TSLexer *lexer, const char *literal) {
  for (const char *cursor = literal; *cursor != '\0'; cursor++) {
    if (lexer->lookahead != (int32_t)(unsigned char)*cursor) {
      return false;
    }
    lexer->advance(lexer, false);
  }
  return true;
}

static bool advance_own_close(TSLexer *lexer, const char *tag) {
  if (lexer->lookahead != '/') {
    return false;
  }
  lexer->advance(lexer, false);
  return advance_literal(lexer, tag) && is_tag_name_terminator(lexer->lookahead);
}

static bool advance_block_opener(TSLexer *lexer) {
  bool matched = false;
  if (lexer->lookahead == 's') {
    lexer->advance(lexer, false);
    if (lexer->lookahead == 'c') {
      matched = advance_literal(lexer, "cript");
    } else if (lexer->lookahead == 't') {
      matched = advance_literal(lexer, "tyle");
    }
  } else if (lexer->lookahead == 'i') {
    matched = advance_literal(lexer, "i18n");
  }
  return matched && is_tag_name_terminator(lexer->lookahead);
}

static bool has_later_close(TSLexer *lexer, const char *tag) {
  while (!lexer->eof(lexer)) {
    if (lexer->lookahead != '<') {
      lexer->advance(lexer, false);
      continue;
    }
    lexer->advance(lexer, false);
    if (lexer->lookahead == '/') {
      if (advance_own_close(lexer, tag)) {
        return true;
      }
    } else if (advance_block_opener(lexer)) {
      return false;
    }
  }
  return false;
}

static bool scan_raw_text(TSLexer *lexer, const char *tag, enum TokenType chunk_symbol,
                          enum TokenType close_symbol, const bool *valid_symbols) {
  bool has_content = false;

  while (!lexer->eof(lexer)) {
    if (lexer->lookahead != '<') {
      lexer->advance(lexer, false);
      has_content = true;
      continue;
    }

    lexer->mark_end(lexer);
    lexer->advance(lexer, false);
    if (!advance_own_close(lexer, tag)) {
      has_content = true;
      continue;
    }

    if (has_content) {
      lexer->result_symbol = chunk_symbol;
      return valid_symbols[chunk_symbol];
    }

    if (!valid_symbols[close_symbol]) {
      return false;
    }
    lexer->mark_end(lexer);
    lexer->result_symbol = close_symbol;
    return has_later_close(lexer, tag);
  }

  lexer->mark_end(lexer);
  lexer->result_symbol = chunk_symbol;
  return has_content && valid_symbols[chunk_symbol];
}

bool tree_sitter_piko_external_scanner_scan(void *payload, TSLexer *lexer,
                                            const bool *valid_symbols) {
  (void)payload;

  if (valid_symbols[SCRIPT_RAW_CHUNK] && valid_symbols[STYLE_RAW_CHUNK] &&
      valid_symbols[I18N_RAW_CHUNK]) {
    return false;
  }

  if (valid_symbols[SCRIPT_RAW_CHUNK] || valid_symbols[SCRIPT_INNER_CLOSE]) {
    return scan_raw_text(lexer, "script", SCRIPT_RAW_CHUNK, SCRIPT_INNER_CLOSE,
                         valid_symbols);
  }
  if (valid_symbols[STYLE_RAW_CHUNK] || valid_symbols[STYLE_INNER_CLOSE]) {
    return scan_raw_text(lexer, "style", STYLE_RAW_CHUNK, STYLE_INNER_CLOSE, valid_symbols);
  }
  if (valid_symbols[I18N_RAW_CHUNK] || valid_symbols[I18N_INNER_CLOSE]) {
    return scan_raw_text(lexer, "i18n", I18N_RAW_CHUNK, I18N_INNER_CLOSE, valid_symbols);
  }

  return false;
}
