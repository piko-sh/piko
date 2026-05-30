package main

type Ordered interface {
	~int | ~int64 | ~float64 | ~string
}
