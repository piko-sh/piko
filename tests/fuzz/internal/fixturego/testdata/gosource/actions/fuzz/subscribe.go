package fuzz

import (
	"errors"
	"strconv"

	"piko.sh/piko"
)

type SubscribeAction struct{ piko.ActionMetadata }
type SubscribeInput struct{}

func (a *SubscribeAction) Call(_ SubscribeInput) (NoticeStatus, error) {
	notices.Lock()
	defer notices.Unlock()
	return NoticeStatus{Active: len(notices.listeners), Latest: len(notices.history)}, nil
}

func (a *SubscribeAction) StreamProgress(stream *piko.SSEStream) error {
	last := 0
	if v := stream.LastEventID(); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		last = n
	}
	notices.Lock()
	if notices.listeners == nil {
		notices.listeners = make(map[int]chan Notice)
	}
	if len(notices.listeners) >= 2 {
		notices.Unlock()
		return errors.New("two subscriber fixture limit")
	}
	notices.next++
	owner := notices.next
	queue := make(chan Notice, 8)
	notices.listeners[owner] = queue
	replay := append([]Notice(nil), notices.history...)
	notices.Unlock()
	defer func() { notices.Lock(); delete(notices.listeners, owner); notices.Unlock() }()
	send := func(n Notice) error { return stream.SendWithID(strconv.Itoa(n.ID), "notice", n) }
	for _, n := range replay {
		if n.ID > last {
			if err := send(n); err != nil {
				return err
			}
		}
	}
	if err := stream.Send("ready", true); err != nil {
		return err
	}
	for {
		select {
		case <-stream.Done():
			return nil
		case v := <-queue:
			if err := send(v); err != nil {
				return err
			}
		}
	}
}
