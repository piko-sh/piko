package shop

import "piko.sh/piko"

type ReleaseAction struct{ piko.ActionMetadata }
type ReleaseInput struct{ Approve bool }

var release = make(chan bool, 1)

func (a *ReleaseAction) Call(input ReleaseInput) (bool, error) {
	select {
	case release <- input.Approve:
		return true, nil
	case <-a.Ctx().Done():
		return false, a.Ctx().Err()
	}
}
