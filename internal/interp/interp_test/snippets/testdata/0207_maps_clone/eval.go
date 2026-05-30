package main

import "maps"

func run() string {
	m := map[string]string{"key": "original"}
	c := maps.Clone(m)
	c["key"] = "modified"
	return m["key"]
}
