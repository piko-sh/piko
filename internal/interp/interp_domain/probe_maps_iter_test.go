package interp_domain_test

import (
	"context"
	"fmt"
	"testing"

	interp_domain "piko.sh/piko/internal/interp/interp_domain"
)

func TestProbeMapsIter(t *testing.T) {
	cases := []struct{ name, code string }{
		{name: "plain_for_range_map", code: `package main
func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	count := 0
	for k := range m {
		_ = k
		count++
	}
	return count
}
func main() {}`},
		{name: "maps_keys", code: `package main
import "maps"
func run() int {
	m := map[string]int{"a": 1, "b": 2}
	count := 0
	for k := range maps.Keys(m) {
		_ = k
		count++
	}
	return count
}
func main() {}`},
		{name: "native_iter_seq", code: `package main
import "iter"
func keysIter() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < 3; i++ {
			if !yield(i) {
				return
			}
		}
	}
}
func run() int {
	count := 0
	for v := range keysIter() {
		_ = v
		count++
	}
	return count
}
func main() {}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := interp_domain.NewService()
			r, err := s.EvalFile(context.Background(), c.code, "run")
			fmt.Printf("[%s] result=%v err=%v\n", c.name, r, err)
		})
	}
}
