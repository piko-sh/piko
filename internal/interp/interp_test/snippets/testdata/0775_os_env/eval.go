package main

import (
	"fmt"
	"os"
)

func run() string {
	const key = "PIKO_TEST_VAR_775"
	_ = os.Unsetenv(key)
	result := ""

	_, present1 := os.LookupEnv(key)
	result += fmt.Sprintf("before:present=%t;", present1)

	_ = os.Setenv(key, "hello")
	v, present2 := os.LookupEnv(key)
	result += fmt.Sprintf("after:v=%s,present=%t;", v, present2)

	got := os.Getenv(key)
	result += fmt.Sprintf("getenv=%s;", got)

	_ = os.Unsetenv(key)
	got2 := os.Getenv(key)
	result += fmt.Sprintf("unset:getenv=%q", got2)

	return result
}
