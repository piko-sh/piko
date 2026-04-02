package main

func buildTag(tag string, content string) string {
	return "<" + tag + ">" + content + "</" + tag + ">"
}
