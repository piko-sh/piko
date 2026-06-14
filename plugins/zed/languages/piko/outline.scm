(script_element
  (start_tag (tag_name) @name) @item)
(template_element
  (start_tag (tag_name) @name) @item)
(style_element
  (start_tag (tag_name) @name) @item)
(i18n_element
  (start_tag (tag_name) @name) @item)

((self_closing_tag
  name: (tag_name) @name) @item
  (#match? @name "^piko:"))
((start_tag
  name: (tag_name) @name) @item
  (#match? @name "^piko:"))
