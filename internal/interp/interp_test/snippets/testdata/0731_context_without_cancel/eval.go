package main

import (
	"context"
	"fmt"
)

func run() string {
	result := ""
	parent, cancel := context.WithCancel(context.Background())
	detached := context.WithoutCancel(parent)

	cancel()

	result += fmt.Sprintf("parent_err=%v;", parent.Err())
	result += fmt.Sprintf("detached_err=%v;", detached.Err())

	select {
	case <-parent.Done():
		result += "parent_done;"
	default:
		result += "parent_undone;"
	}

	select {
	case <-detached.Done():
		result += "detached_done"
	default:
		result += "detached_undone"
	}

	return result
}
