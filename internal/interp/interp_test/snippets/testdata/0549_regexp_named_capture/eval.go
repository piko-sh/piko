package main

import "regexp"

func run() string {
	re := regexp.MustCompile(`(?P<name>\w+)=(?P<value>\d+)`)
	m := re.FindStringSubmatch("count=42")
	if len(m) < 3 {
		return "miss"
	}
	return m[1] + "=" + m[2]
}
