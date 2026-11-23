package fuzz

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko"
)

type RendezvousInput struct {
	Label  string
	Number int64
}
type RendezvousResult struct {
	Label, Before, After, Cookie string
	Number, Receiver             int64
}
type RendezvousAction struct {
	piko.ActionMetadata
	receiver atomic.Int64
}

var errRendezvousTimeout = errors.New("rendezvous partner did not arrive")

var actors struct {
	sync.Mutex
	waiting chan struct{}
}

func meetPartner(ctx context.Context) error {
	actors.Lock()
	if partner := actors.waiting; partner != nil {
		actors.waiting = nil
		close(partner)
		actors.Unlock()
		return nil
	}
	gate := make(chan struct{})
	actors.waiting = gate
	actors.Unlock()
	select {
	case <-gate:
		return nil
	case <-ctx.Done():
		actors.Lock()
		if actors.waiting == gate {
			actors.waiting = nil
		}
		actors.Unlock()
		return context.Cause(ctx)
	}
}

func (a *RendezvousAction) Call(input RendezvousInput) (RendezvousResult, error) {
	ctx, cancel := context.WithTimeoutCause(a.Ctx(), 4*time.Second, errRendezvousTimeout)
	defer cancel()
	before := a.Request().Headers.Get("X-Fuzz-Actor")
	a.receiver.Store(input.Number)
	if err := meetPartner(ctx); err != nil {
		return RendezvousResult{}, err
	}
	cookie, err := a.Request().RawRequest.Cookie("case-actor")
	if err != nil {
		return RendezvousResult{}, err
	}
	return RendezvousResult{Label: input.Label, Number: input.Number, Receiver: a.receiver.Load(), Before: before, After: a.Request().Headers.Get("X-Fuzz-Actor"), Cookie: cookie.Value}, nil
}
