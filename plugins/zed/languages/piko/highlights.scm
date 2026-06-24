(tag_name) @tag

((tag_name) @type
  (#match? @type "^piko:"))

(attribute_name) @attribute
(directive_name) @keyword

(attribute_value) @string

(doctype) @keyword

(comment) @comment

"<" @punctuation.bracket
">" @punctuation.bracket
"</" @punctuation.bracket
"/>" @punctuation.bracket

"{{" @punctuation.special
"}}" @punctuation.special

"=" @punctuation.delimiter
"\"" @punctuation.delimiter
"'" @punctuation.delimiter
