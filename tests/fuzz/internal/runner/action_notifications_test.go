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

package runner

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestNoticeStreamJoinsDataLinesOfOneEvent(t *testing.T) {
	stream := newNoticeStream(io.NopCloser(strings.NewReader("event: ready\ndata: tr\ndata: ue\n\n")))
	var frames []NoticeFrame

	err := stream.read(&frames)

	require.Error(t, err, "joined ready data is \"tr\\nue\", not \"true\"")
	assert.ErrorContains(t, err, `"tr\nue"`)
}

func TestNoticeStreamDecodesNoticeFrames(t *testing.T) {
	body := ": comment\nevent: ready\ndata: true\n\nevent: notice\nid: 1\ndata: {\"ID\":1,\ndata: \"Text\":\"label-1\"}\n\n"
	stream := newNoticeStream(io.NopCloser(strings.NewReader(body)))
	var frames []NoticeFrame

	require.NoError(t, stream.read(&frames))
	require.NoError(t, stream.read(&frames))

	assert.Equal(t, []NoticeFrame{
		{Event: "ready"},
		{Event: "notice", ID: "1", Text: "label-1", Sequence: 1},
	}, frames)
	assert.ErrorIs(t, stream.read(&frames), errNoticeStreamEnded)
}

func TestNoticeStreamBoundsEachFrameSeparately(t *testing.T) {
	frame := "event: ready\ndata: true\n: " + strings.Repeat("p", noticeFrameMax/2) + "\n\n"
	stream := newNoticeStream(io.NopCloser(strings.NewReader(strings.Repeat(frame, 4))))
	var frames []NoticeFrame
	for range 4 {
		require.NoError(t, stream.read(&frames), "a long stream of small frames must not be truncated")
	}
	assert.Len(t, frames, 4)

	oversized := "event: ready\n: " + strings.Repeat("p", noticeFrameMax) + "\ndata: true\n\n"
	stream = newNoticeStream(io.NopCloser(strings.NewReader(oversized)))
	assert.ErrorIs(t, stream.read(&frames), errNoticeFrameOversize)

	split := "event: ready\n" + strings.Repeat(": "+strings.Repeat("p", 1000)+"\n", noticeFrameMax/1000+1) + "data: true\n\n"
	stream = newNoticeStream(io.NopCloser(strings.NewReader(split)))
	assert.ErrorIs(t, stream.read(&frames), errNoticeFrameOversize)
}

func TestCheckNotificationsAcceptsModel(t *testing.T) {
	input := casefile.ActionValue{Label: "label", Quantity: 2}
	trace := expectedNotices(input)
	require.NoError(t, checkNotifications(input, trace))
	trace.Statuses[0].Active++
	assert.Error(t, checkNotifications(input, trace))
}
