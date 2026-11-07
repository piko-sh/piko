(document
  (template_element
    .
    [
      (start_tag
        name: (tag_name) @name)
      (self_closing_tag
        name: (tag_name) @name)
    ]) @item)

(script_element
  (start_tag
    name: (tag_name) @name
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @context
        (quoted_attribute_value (attribute_value) @context)
      ])
    (#any-of? @_attr "type" "lang"))) @item

(script_element
  (start_tag
    name: (tag_name) @name .)) @item

(script_element
  (start_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (#not-any-of? @_first "type" "lang")) @item

(script_element
  (start_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (#not-any-of? @_first "type" "lang")
  (#not-any-of? @_second "type" "lang")) @item

(script_element
  (self_closing_tag
    name: (tag_name) @name
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @context
        (quoted_attribute_value (attribute_value) @context)
      ])
    (#any-of? @_attr "type" "lang"))) @item

(script_element
  (self_closing_tag
    name: (tag_name) @name .)) @item

(script_element
  (self_closing_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (#not-any-of? @_first "type" "lang")) @item

(script_element
  (self_closing_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (#not-any-of? @_first "type" "lang")
  (#not-any-of? @_second "type" "lang")) @item

(style_element
  (start_tag
    name: (tag_name) @name
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @context
        (quoted_attribute_value (attribute_value) @context)
      ])
    (#any-of? @_attr "lang"))) @item

(style_element
  (start_tag
    name: (tag_name) @name .)) @item

(style_element
  (start_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (#not-any-of? @_first "lang")) @item

(style_element
  (start_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (#not-any-of? @_first "lang")
  (#not-any-of? @_second "lang")) @item

(style_element
  (self_closing_tag
    name: (tag_name) @name
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @context
        (quoted_attribute_value (attribute_value) @context)
      ])
    (#any-of? @_attr "lang"))) @item

(style_element
  (self_closing_tag
    name: (tag_name) @name .)) @item

(style_element
  (self_closing_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (#not-any-of? @_first "lang")) @item

(style_element
  (self_closing_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (#not-any-of? @_first "lang")
  (#not-any-of? @_second "lang")) @item

(i18n_element
  (start_tag
    name: (tag_name) @name
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @context
        (quoted_attribute_value (attribute_value) @context)
      ])
    (#any-of? @_attr "lang"))) @item

(i18n_element
  (start_tag
    name: (tag_name) @name .)) @item

(i18n_element
  (start_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (#not-any-of? @_first "lang")) @item

(i18n_element
  (start_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (#not-any-of? @_first "lang")
  (#not-any-of? @_second "lang")) @item

(i18n_element
  (self_closing_tag
    name: (tag_name) @name
    (attribute
      name: (attribute_name) @_attr
      value: [
        (attribute_value) @context
        (quoted_attribute_value (attribute_value) @context)
      ])
    (#any-of? @_attr "lang"))) @item

(i18n_element
  (self_closing_tag
    name: (tag_name) @name .)) @item

(i18n_element
  (self_closing_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .)
  (#not-any-of? @_first "lang")) @item

(i18n_element
  (self_closing_tag
    name: (tag_name) @name
    .
    (attribute
      [(attribute_name) (directive_name)] @_first)
    .
    (attribute
      [(attribute_name) (directive_name)] @_second)
    .)
  (#not-any-of? @_first "lang")
  (#not-any-of? @_second "lang")) @item

((start_tag
  name: (tag_name) @name) @item
  (#match? @name "^piko:(partial|slot)$"))

((self_closing_tag
  name: (tag_name) @name) @item
  (#match? @name "^piko:(partial|slot)$"))
