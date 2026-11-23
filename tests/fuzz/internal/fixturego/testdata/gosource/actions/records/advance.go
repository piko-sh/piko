package records

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"example.org/pikofuzzcase/pkg/recordboard"
	"piko.sh/piko"
)

var mu sync.Mutex

type AdvanceInput struct {
	Op string `json:"op"`
}
type AdvanceAction struct{ piko.ActionMetadata }
type AdvanceResult struct {
	Revision int `json:"revision"`
}

func (a *AdvanceAction) Call(input AdvanceInput) (AdvanceResult, error) {
	mu.Lock()
	defer mu.Unlock()
	current, err := recordboard.Revision()
	if err != nil {
		return AdvanceResult{}, err
	}
	planned := [...]string{"edit", "reorder", "remove", "restore"}
	if current < 0 || current >= len(planned) || planned[current] != input.Op {
		return AdvanceResult{}, fmt.Errorf("unexpected operation %q at revision %d", input.Op, current)
	}
	next := current + 1
	if err := os.WriteFile("records-revision.txt", []byte(strconv.Itoa(next)), 0600); err != nil {
		return AdvanceResult{}, err
	}
	return AdvanceResult{Revision: next}, nil
}
