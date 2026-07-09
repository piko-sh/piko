(script_element
  (start_tag
    (attribute
      value: (quoted_attribute_value (attribute_value) @_lang)))
  (raw_text) @injection.content
  (#match? @_lang "(^|/)x-go$|(^|/)go$")
  (#set! injection.language "go"))

(script_element
  (start_tag
    (attribute
      value: (quoted_attribute_value (attribute_value) @_lang)))
  (raw_text) @injection.content
  (#match? @_lang "javascript|(^|/)js$")
  (#set! injection.language "javascript"))

(script_element
  (start_tag
    (attribute
      value: (quoted_attribute_value (attribute_value) @_lang)))
  (raw_text) @injection.content
  (#match? @_lang "typescript|(^|/)ts$")
  (#set! injection.language "typescript"))

(style_element
  (raw_text) @injection.content
  (#set! injection.language "css"))

(i18n_element
  (raw_text) @injection.content
  (#set! injection.language "json"))

(interpolation
  (expression) @injection.content
  (#set! injection.language "go"))

(directive_value
  (expression) @injection.content
  (#set! injection.language "go"))
