(script_element
  (start_tag
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @_lang
        (quoted_attribute_value (attribute_value) @_lang)
      ])
    (#any-of? @_attr "type" "lang")
    (#match? @_lang "^(application/x-go|application/go|go|golang)$"))
  (raw_text) @injection.content
  (#set! injection.language "go"))

(script_element
  (start_tag
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @_lang
        (quoted_attribute_value (attribute_value) @_lang)
      ])
    (#any-of? @_attr "type" "lang")
    (#match? @_lang "(?i)^(ts|typescript|application/typescript)$"))
  (raw_text) @injection.content
  (#set! injection.language "typescript"))

(script_element
  (start_tag
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @_lang
        (quoted_attribute_value (attribute_value) @_lang)
      ])
    (#any-of? @_attr "type" "lang")
    (#match? @_lang "(?i)^(js|javascript|application/javascript|text/javascript|module)$"))
  (raw_text) @injection.content
  (#set! injection.language "javascript"))

(script_element
  (start_tag
    name: (tag_name) .)
  (raw_text) @injection.content
  (#set! injection.language "javascript"))

(script_element
  (start_tag
    name: (tag_name)
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (raw_text) @injection.content
  (#not-any-of? @_first "type" "lang")
  (#set! injection.language "javascript"))

(script_element
  (start_tag
    name: (tag_name)
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (raw_text) @injection.content
  (#not-any-of? @_first "type" "lang")
  (#not-any-of? @_second "type" "lang")
  (#set! injection.language "javascript"))

(style_element
  (start_tag
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @_lang
        (quoted_attribute_value (attribute_value) @_lang)
      ])
    (#any-of? @_attr "lang")
    (#match? @_lang "(?i)^scss$"))
  (raw_text) @injection.content
  (#set! injection.language "scss"))

(style_element
  (start_tag
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @_lang
        (quoted_attribute_value (attribute_value) @_lang)
      ])
    (#any-of? @_attr "lang")
    (#match? @_lang "(?i)^css$"))
  (raw_text) @injection.content
  (#set! injection.language "css"))

(style_element
  (start_tag
    name: (tag_name) .)
  (raw_text) @injection.content
  (#set! injection.language "css"))

(style_element
  (start_tag
    name: (tag_name)
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (raw_text) @injection.content
  (#not-any-of? @_first "type" "lang")
  (#set! injection.language "css"))

(style_element
  (start_tag
    name: (tag_name)
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (raw_text) @injection.content
  (#not-any-of? @_first "type" "lang")
  (#not-any-of? @_second "type" "lang")
  (#set! injection.language "css"))

(i18n_element
  (start_tag
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @_lang
        (quoted_attribute_value (attribute_value) @_lang)
      ])
    (#any-of? @_attr "lang")
    (#match? @_lang "(?i)^(yaml|yml)$"))
  (raw_text) @injection.content
  (#set! injection.language "yaml"))

(i18n_element
  (start_tag
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @_lang
        (quoted_attribute_value (attribute_value) @_lang)
      ])
    (#any-of? @_attr "lang")
    (#match? @_lang "(?i)^json$"))
  (raw_text) @injection.content
  (#set! injection.language "json"))

(i18n_element
  (start_tag
    name: (tag_name) .)
  (raw_text) @injection.content
  (#set! injection.language "json"))

(i18n_element
  (start_tag
    name: (tag_name)
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (raw_text) @injection.content
  (#not-any-of? @_first "type" "lang")
  (#set! injection.language "json"))

(i18n_element
  (start_tag
    name: (tag_name)
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (raw_text) @injection.content
  (#not-any-of? @_first "type" "lang")
  (#not-any-of? @_second "type" "lang")
  (#set! injection.language "json"))

(interpolation
  (expression) @injection.content
  (#set! injection.language "javascript"))

(directive_value
  (expression) @injection.content
  (#set! injection.language "javascript"))

((comment) @injection.content
  (#set! injection.language "comment"))
