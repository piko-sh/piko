package main

import "strings"

func wrapTag(tag string, content string) string {
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	b.WriteString(">")
	b.WriteString(content)
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
	return b.String()
}

func joinAll(items []string) string {
	return strings.Join(items, "")
}
