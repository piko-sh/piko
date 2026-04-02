package main

type Props struct {
	Article string `prop:"article"`
	Count   int    `prop:"count"`
}

func run() string {
	p := Props{Article: "hello", Count: 3}
	return p.Article
}
