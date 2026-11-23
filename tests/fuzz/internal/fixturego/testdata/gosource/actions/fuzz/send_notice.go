package fuzz

import (
	"errors"
	"sync"

	"piko.sh/piko"
)

type Notice struct {
	ID   int
	Text string
}
type SendNoticeInput struct {
	Inspect bool
	Text    string
}
type NoticeStatus struct{ Active, Latest int }
type SendNoticeAction struct{ piko.ActionMetadata }

var notices struct {
	sync.Mutex
	next      int
	listeners map[int]chan Notice
	history   []Notice
}

func (a *SendNoticeAction) Call(in SendNoticeInput) (NoticeStatus, error) {
	notices.Lock()
	defer notices.Unlock()
	if !in.Inspect {
		if len(notices.history) >= 8 || in.Text == "" {
			return NoticeStatus{}, errors.New("bounded notice fixture is full or empty")
		}
		n := Notice{ID: len(notices.history) + 1, Text: in.Text}
		notices.history = append(notices.history, n)
		for _, queue := range notices.listeners {
			queue <- n
		}
	}
	return NoticeStatus{Active: len(notices.listeners), Latest: len(notices.history)}, nil
}
