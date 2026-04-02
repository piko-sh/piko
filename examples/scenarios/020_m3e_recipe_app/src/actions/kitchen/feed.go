// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package kitchen

import (
	"math/rand/v2"
	"time"

	"testmodule/internal/kitchen"

	"piko.sh/piko"
)

// FeedInput is empty  - the feed takes no parameters.
type FeedInput struct{}

// FeedOutput is the non-streaming fallback response.
type FeedOutput struct {
	Active bool `json:"active"`
}

// FeedAction streams live kitchen activity events to the client via SSE.
type FeedAction struct {
	piko.ActionMetadata
}

// Call is the non-streaming fallback.
func (*FeedAction) Call(_ FeedInput) (FeedOutput, error) {
	return FeedOutput{Active: true}, nil
}

// StreamProgress draws events from two independent pools (sequences
// and standalone) and streams them to the client. Each item is removed
// from its pool after being sent, so nothing repeats until the entire
// pool is exhausted  - at which point that pool alone is recreated and
// reshuffled. The two pools refill independently.
func (*FeedAction) StreamProgress(stream *piko.SSEStream) error {
	seqs := newSequencePool()
	singles := newStandalonePool()

	// activeSeq holds the remaining events of a sequence currently
	// being played out one event at a time.
	var activeSeq kitchen.Sequence

	// standaloneGap counts how many standalone events to send
	// before the next sequence begins.
	standaloneGap := 3 + rand.IntN(3) //nolint:gosec // not security
	gapSent := 0

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		var event kitchen.Event

		switch {
		case len(activeSeq) > 0:
			// Continue playing the current sequence.
			event = activeSeq[0]
			activeSeq = activeSeq[1:]

		case gapSent < standaloneGap:
			// Send standalone events as a breather between sequences.
			if len(singles) == 0 {
				singles = newStandalonePool()
			}
			event = singles[0]
			singles = singles[1:]
			gapSent++

		default:
			// Start the next sequence.
			if len(seqs) == 0 {
				seqs = newSequencePool()
			}
			activeSeq = seqs[0]
			seqs = seqs[1:]

			event = activeSeq[0]
			activeSeq = activeSeq[1:]

			// Reset gap counter for after this sequence finishes.
			standaloneGap = 3 + rand.IntN(3) //nolint:gosec // not security
			gapSent = 0
		}

		if err := stream.Send("event", map[string]string{
			"text":     event.Text,
			"category": event.Category,
			"time":     time.Now().Format("15:04"),
		}); err != nil {
			return err
		}

		// Wait 5-12 seconds before the next event.
		delay := 5*time.Second + time.Duration(rand.IntN(7001))*time.Millisecond //nolint:gosec // not security
		timer := time.NewTimer(delay)

		select {
		case <-stream.Done():
			timer.Stop()
			return nil
		case <-heartbeat.C:
			if err := stream.SendHeartbeat(); err != nil {
				timer.Stop()
				return err
			}
		case <-timer.C:
			// Continue to next event.
		}
	}
}

// newSequencePool returns a shuffled copy of all sequence chains.
func newSequencePool() []kitchen.Sequence {
	pool := make([]kitchen.Sequence, len(kitchen.Sequences))
	for i, s := range kitchen.Sequences {
		sequence := make(kitchen.Sequence, len(s))
		copy(sequence, s)
		pool[i] = sequence
	}
	rand.Shuffle(len(pool), func(i, j int) { //nolint:gosec // not security
		pool[i], pool[j] = pool[j], pool[i]
	})
	return pool
}

// newStandalonePool returns a shuffled copy of all standalone events.
func newStandalonePool() []kitchen.Event {
	pool := make([]kitchen.Event, len(kitchen.Standalone))
	copy(pool, kitchen.Standalone)
	rand.Shuffle(len(pool), func(i, j int) { //nolint:gosec // not security
		pool[i], pool[j] = pool[j], pool[i]
	})
	return pool
}
