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

package browser

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
)

type firefoxServer struct {
	conn net.Conn
	done chan struct{}
	t    *testing.T
}

func newPipedFirefoxSession(t *testing.T, maxMessageBytes int64) (*FirefoxSession, *firefoxServer) {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	session := &FirefoxSession{conn: client, maxMessageBytes: maxMessageBytes, cancel: func(error) {}}
	return session, &firefoxServer{conn: server, done: make(chan struct{}), t: t}
}

func (f *firefoxServer) serve(script func(*firefoxServer)) {
	go func() {
		defer close(f.done)
		script(f)
	}()
}

func (f *firefoxServer) wait() {
	f.t.Helper()
	select {
	case <-f.done:
	case <-time.After(5 * time.Second):
		require.FailNow(f.t, "scripted Firefox server did not finish")
	}
}

func (f *firefoxServer) readCommand() (int, string, bool) {
	payload, err := wsutil.ReadClientText(f.conn)
	if !assert.NoError(f.t, err) {
		return 0, "", false
	}
	var command struct {
		ID int `json:"id"`
	}
	if !assert.NoError(f.t, json.Unmarshal(payload, &command)) {
		return 0, "", false
	}
	return command.ID, string(payload), true
}

func (f *firefoxServer) reply(text string) bool {
	return assert.NoError(f.t, wsutil.WriteServerText(f.conn, []byte(text)))
}

func TestFirefoxCommandSkipsEventsAndAnswersPings(t *testing.T) {
	session, server := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	server.serve(func(server *firefoxServer) {
		id, _, ok := server.readCommand()
		if !ok || !assert.NoError(t, wsutil.WriteServerMessage(server.conn, ws.OpPing, []byte("ping"))) {
			return
		}
		frame, err := ws.ReadFrame(server.conn)
		if !assert.NoError(t, err) {
			return
		}
		frame = ws.UnmaskFrameInPlace(frame)
		assert.Equal(t, ws.OpPong, frame.Header.OpCode)
		assert.Equal(t, []byte("ping"), frame.Payload)
		if server.reply(`{"type":"event","method":"log.entryAdded"}`) {
			server.reply(`{"type":"success","id":` + itoa(id) + `,"result":{"type":"success","result":{"value":"ready"}}}`)
		}
	})

	value, err := session.EvaluateString(t.Context(), "'ready'")

	require.NoError(t, err)
	assert.Equal(t, "ready", value)
	server.wait()
}

func TestFirefoxOversizedReplyBreaksTheSession(t *testing.T) {
	session, server := newPipedFirefoxSession(t, 64)
	server.serve(func(server *firefoxServer) {
		id, _, ok := server.readCommand()
		if ok {
			large := `{"type":"success","id":` + itoa(id) + `,"result":{"value":"` + strings.Repeat("x", 256) + `"}}`
			_ = wsutil.WriteServerText(server.conn, []byte(large))
		}
	})

	_, err := session.EvaluateString(t.Context(), "'x'.repeat(256)")
	require.ErrorIs(t, err, ErrFirefoxMessageTooLarge)
	require.ErrorIs(t, err, ErrFirefoxSessionBroken)

	err = session.NavigateURL(t.Context(), "about:blank")
	require.ErrorIs(t, err, ErrFirefoxSessionBroken, "a broken session fails fast")
	server.wait()
}

func TestFirefoxEvaluationErrors(t *testing.T) {
	session, server := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	server.serve(func(server *firefoxServer) {
		replies := []string{
			`"result":{"type":"exception","exceptionDetails":{"text":"ReferenceError: nope"}}}`,
			`"result":{"type":"exception"}}`,
			`"error":"no such frame","message":"gone"}`,
		}
		for index, text := range replies {
			id, _, ok := server.readCommand()
			if !ok {
				return
			}
			kind := "success"
			if index == len(replies)-1 {
				kind = "error"
			}
			if !server.reply(`{"type":"` + kind + `","id":` + itoa(id) + `,` + text) {
				return
			}
		}
	})

	_, err := session.EvaluateString(t.Context(), "nope")
	require.ErrorIs(t, err, errFirefoxEvaluation)
	assert.Contains(t, err.Error(), "ReferenceError: nope")

	_, err = session.EvaluateString(t.Context(), "throw undefined")
	require.ErrorIs(t, err, errFirefoxEvaluation, "an exception without text still reports a failure")

	err = session.NavigateURL(t.Context(), "about:blank")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such frame")
	assert.NotErrorIs(t, err, ErrFirefoxSessionBroken, "a protocol error leaves the connection usable")
	server.wait()
}

func TestFirefoxOutOfOrderReplyBreaksTheSession(t *testing.T) {
	session, server := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	server.serve(func(server *firefoxServer) {
		if id, _, ok := server.readCommand(); ok {
			server.reply(`{"type":"success","id":` + itoa(id+7) + `,"result":{}}`)
		}
	})

	err := session.NavigateURL(t.Context(), "about:blank")

	require.ErrorIs(t, err, errFirefoxOutOfOrder)
	require.ErrorIs(t, err, ErrFirefoxSessionBroken)
	server.wait()
}

func TestFirefoxCommandEncodesTypedParams(t *testing.T) {
	session, server := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	session.tab = "tab-1"
	commands := make(chan string, 4)
	server.serve(func(server *firefoxServer) {
		for range 4 {
			id, text, ok := server.readCommand()
			if !ok {
				return
			}
			commands <- text
			result := `{}`
			if strings.Contains(text, "script.evaluate") {
				result = `{"type":"success","result":{"value":"ok"}}`
			}
			if !server.reply(`{"type":"success","id":` + itoa(id) + `,"result":` + result + `}`) {
				return
			}
		}
	})

	require.NoError(t, session.NavigateURL(t.Context(), "http://127.0.0.1/"))
	require.NoError(t, session.SetViewport(t.Context(), 800, 600))
	require.NoError(t, session.Key(t.Context(), "aé"))
	_, err := session.EvaluateString(t.Context(), "'ok'")
	require.NoError(t, err)
	server.wait()

	assert.JSONEq(t, `{"id":1,"method":"browsingContext.navigate","params":{"context":"tab-1","url":"http://127.0.0.1/","wait":"complete"}}`, <-commands)
	assert.JSONEq(t, `{"id":2,"method":"browsingContext.setViewport","params":{"context":"tab-1","viewport":{"width":800,"height":600}}}`, <-commands)
	assert.JSONEq(t, `{"id":3,"method":"input.performActions","params":{"context":"tab-1","actions":[{"type":"key","id":"keyboard","actions":[`+
		`{"type":"keyDown","value":"a"},{"type":"keyUp","value":"a"},{"type":"keyDown","value":"é"},{"type":"keyUp","value":"é"}]}]}}`, <-commands)
	assert.JSONEq(t, `{"id":4,"method":"script.evaluate","params":{"expression":"'ok'","target":{"context":"tab-1"},"awaitPromise":true}}`, <-commands)
}

func TestFirefoxCommandDeadlineBreaksTheSession(t *testing.T) {
	session, server := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	server.serve(func(server *firefoxServer) {
		server.readCommand()
	})
	cause := errors.New("command budget spent")
	ctx, cancel := context.WithTimeoutCause(t.Context(), 30*time.Millisecond, cause)
	defer cancel()

	_, err := session.EvaluateString(ctx, "new Promise(() => {})")

	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, err, ErrFirefoxSessionBroken, "an unread reply leaves the stream out of step")
	server.wait()
}

func TestFirefoxCommandCancellationBreaksTheSession(t *testing.T) {
	session, server := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	cause := errors.New("caller gave up")
	ctx, cancel := context.WithCancelCause(t.Context())
	server.serve(func(server *firefoxServer) {
		if _, _, ok := server.readCommand(); ok {
			cancel(cause)
		}
	})

	err := session.NavigateURL(ctx, "about:blank")

	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, err, ErrFirefoxSessionBroken)
	server.wait()
}

func TestFirefoxCommandRestoresTheSessionDeadline(t *testing.T) {
	session, server := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	server.serve(func(server *firefoxServer) {
		for range 2 {
			id, _, ok := server.readCommand()
			if !ok || !server.reply(`{"type":"success","id":`+itoa(id)+`,"result":{}}`) {
				return
			}
		}
	})
	short, cancel := context.WithTimeoutCause(t.Context(), 50*time.Millisecond, errors.New("short command"))
	defer cancel()

	require.NoError(t, session.NavigateURL(short, "about:blank"))
	<-short.Done()
	require.NoError(t, session.NavigateURL(t.Context(), "about:blank"), "a finished command's deadline does not outlive it")
	server.wait()
}

func TestFirefoxCommandRefusesADoneContext(t *testing.T) {
	session, _ := newPipedFirefoxSession(t, defaultFirefoxMaxMessageBytes)
	cause := errors.New("already finished")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)

	err := session.NavigateURL(ctx, "about:blank")

	require.ErrorIs(t, err, cause)
	assert.NotErrorIs(t, err, ErrFirefoxSessionBroken, "nothing was sent, so the stream is still in step")
}

func TestFirefoxPressKeyRejectsUnknownNames(t *testing.T) {
	session := &FirefoxSession{}

	require.ErrorIs(t, session.PressKey(t.Context(), "Hyper"), errFirefoxUnsupportedKey)
	require.NoError(t, session.Key(t.Context(), ""), "empty text sends nothing")
}

func TestFirefoxConnectRefusesEarlyFrames(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	served := make(chan struct{})
	go func() {
		defer close(served)
		conn, err := listener.Accept()
		if !assert.NoError(t, err) {
			return
		}
		defer func() { _ = conn.Close() }()
		var handshake bytes.Buffer
		_, err = ws.Upgrade(struct {
			io.Reader
			io.Writer
		}{Reader: conn, Writer: &handshake})
		if !assert.NoError(t, err) {
			return
		}
		assert.NoError(t, ws.WriteFrame(&handshake, ws.NewTextFrame([]byte(`{"type":"event"}`))))
		_, err = conn.Write(handshake.Bytes())
		assert.NoError(t, err)
		_, _ = conn.Read(make([]byte, 1))
	}()
	session := &FirefoxSession{}

	err = session.connect(t.Context(), "ws://"+listener.Addr().String()+"/session")

	require.ErrorIs(t, err, errFirefoxEarlyFrames)
	assert.Nil(t, session.conn)
	<-served
}

func TestParseFirefoxEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		want      string
		wantFound bool
		wantErr   error
	}{
		{name: "local endpoint", line: firefoxListeningMarker + "ws://127.0.0.1:9222\n", want: "ws://127.0.0.1:9222/session", wantFound: true, wantErr: nil},
		{name: "localhost endpoint", line: "1 INFO " + firefoxListeningMarker + "ws://localhost:1\n", want: "ws://localhost:1/session", wantFound: true, wantErr: nil},
		{name: "remote host refused", line: firefoxListeningMarker + "ws://203.0.113.9:9222\n", want: "", wantFound: true, wantErr: errFirefoxEndpoint},
		{name: "secure scheme refused", line: firefoxListeningMarker + "wss://127.0.0.1:9222\n", want: "", wantFound: true, wantErr: errFirefoxEndpoint},
		{name: "other line", line: "Marionette listening\n", want: "", wantFound: false, wantErr: nil},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, found, err := parseFirefoxEndpoint(testCase.line)
			assert.Equal(t, testCase.wantFound, found)
			assert.Equal(t, testCase.want, got)
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWithFirefoxMaxMessageBytes(t *testing.T) {
	session := &FirefoxSession{maxMessageBytes: defaultFirefoxMaxMessageBytes}

	WithFirefoxMaxMessageBytes(0)(session)
	assert.Equal(t, int64(defaultFirefoxMaxMessageBytes), session.maxMessageBytes)

	WithFirefoxMaxMessageBytes(4096)(session)
	assert.Equal(t, int64(4096), session.maxMessageBytes)
}

func TestRawJSONKeepsACopy(t *testing.T) {
	var raw rawJSON
	source := []byte(`{"a":1}`)
	require.NoError(t, raw.UnmarshalJSON(source))
	source[2] = 'b'

	assert.True(t, bytes.Equal([]byte(`{"a":1}`), raw))
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
