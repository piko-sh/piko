(template_element
  .
  (start_tag) @start
  (end_tag) @end .) @indent

(style_element
  (start_tag) @start
  (end_tag) @end) @indent

(start_tag
  ">" @end) @indent

(self_closing_tag
  "/>" @end) @indent
