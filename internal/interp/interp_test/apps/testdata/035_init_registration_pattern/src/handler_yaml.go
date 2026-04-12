package main

func init() {
	register("yaml", func() string { return "ok-yaml" })
}
