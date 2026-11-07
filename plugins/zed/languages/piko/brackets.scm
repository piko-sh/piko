(("<" @open
  ">" @close)
  (#set! rainbow.exclude))

(("<" @open
  "/>" @close)
  (#set! rainbow.exclude))

(("</" @open
  ">" @close)
  (#set! rainbow.exclude))

("{{" @open
  "}}" @close)

(("\"" @open
  "\"" @close)
  (#set! rainbow.exclude))

(("'" @open
  "'" @close)
  (#set! rainbow.exclude))

((template_element
  .
  (start_tag) @open
  (end_tag) @close .)
  (#set! newline.only)
  (#set! rainbow.exclude))

([
  (script_element
    (start_tag) @open
    (end_tag) @close)
  (style_element
    (start_tag) @open
    (end_tag) @close)
  (i18n_element
    (start_tag) @open
    (end_tag) @close)
]
  (#set! newline.only)
  (#set! rainbow.exclude))
