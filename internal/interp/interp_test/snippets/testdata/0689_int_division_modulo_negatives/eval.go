package main

import "fmt"

func run() string {
	result := ""

	cases := []struct {
		x, y int
	}{
		{7, 3},
		{-7, 3},
		{7, -3},
		{-7, -3},
		{1, 5},
		{-1, 5},
		{0, 7},
	}

	for _, tc := range cases {
		q := tc.x / tc.y
		r := tc.x % tc.y
		result += fmt.Sprintf("%d/%d=%d,%%=%d;", tc.x, tc.y, q, r)
	}

	int64Cases := []struct {
		x, y int64
	}{
		{-9223372036854775807, 2},
		{9223372036854775807, -1},
	}
	for _, tc := range int64Cases {
		q := tc.x / tc.y
		r := tc.x % tc.y
		result += fmt.Sprintf("%d/%d=%d,%%=%d;", tc.x, tc.y, q, r)
	}

	return result
}
