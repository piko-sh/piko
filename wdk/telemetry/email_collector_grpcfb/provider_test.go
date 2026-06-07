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

package email_collector_grpcfb

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/wdk/email"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

type sink struct {
	batches []*telemetry_grpcfb.Batch
	mu      sync.Mutex
}

func (s *sink) Ingest(stream telemetry_grpcfb.IngestStream) error {
	for {
		b, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.batches = append(s.batches, b)
		s.mu.Unlock()
	}
	return stream.SendAndClose(&telemetry_grpcfb.IngestAck{OK: true})
}

func (s *sink) emails() []telemetry_grpcfb.EmailEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []telemetry_grpcfb.EmailEvent
	for _, b := range s.batches {
		out = append(out, b.Emails...)
	}
	return out
}

type fakeProvider struct {
	sendErr      error
	bulkErr      error
	closeErr     error
	sent         []*email.SendParams
	bulk         [][]*email.SendParams
	supportsBulk bool
	closed       bool
}

func (f *fakeProvider) Send(_ context.Context, params *email.SendParams) error {
	f.sent = append(f.sent, params)
	return f.sendErr
}

func (f *fakeProvider) SendBulk(_ context.Context, emails []*email.SendParams) error {
	f.bulk = append(f.bulk, emails)
	return f.bulkErr
}

func (f *fakeProvider) SupportsBulkSending() bool { return f.supportsBulk }

func (f *fakeProvider) Close(_ context.Context) error {
	f.closed = true
	return f.closeErr
}

func newTestClient(t *testing.T) (*telemetry_grpcfb.Client, *sink, func()) {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()

	conn := dial(t, lis)

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{
		SiteID:        "site-x",
		APIKey:        "key-x",
		FlushSize:     1,
		FlushInterval: time.Hour,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client.Start(ctx)

	drain := func() {
		defer cancel()
		require.NoError(t, client.Close(ctx))
		conn.Close()
		srv.Stop()
	}
	return client, snk, drain
}

func dial(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)
	return conn
}

func TestSendEmitsEventOnSuccess(t *testing.T) {
	client, snk, drain := newTestClient(t)
	inner := &fakeProvider{}
	p := Wrap("smtp", inner, client)

	ctx := context.Background()
	params := &email.SendParams{
		To:              []string{"alice@example.com"},
		Subject:         "Welcome",
		ProviderOptions: map[string]any{"template": "welcome.pk"},
	}
	require.NoError(t, p.Send(ctx, params))

	drain()

	require.Len(t, inner.sent, 1, "Send must delegate to the inner provider")

	events := snk.emails()
	require.Len(t, events, 1)
	ev := events[0]
	assert.Equal(t, "a***@example.com", ev.Recipient)
	assert.Equal(t, "Welcome", ev.Subject)
	assert.Equal(t, "sent", ev.Event)
	assert.Equal(t, "ok", ev.Status)
	assert.Empty(t, ev.Error)
	assert.Equal(t, "smtp", ev.Provider)
	assert.Equal(t, "welcome.pk", ev.Template)
	assert.NotEmpty(t, ev.MessageID)
}

func TestSendEmitsEventOnFailure(t *testing.T) {
	client, snk, drain := newTestClient(t)
	wantErr := errors.New("upstream refused the connection")
	inner := &fakeProvider{sendErr: wantErr}
	p := Wrap("ses", inner, client)

	err := p.Send(context.Background(), &email.SendParams{
		To:      []string{"bob@example.com"},
		Subject: "Receipt",
	})
	require.ErrorIs(t, err, wantErr)

	drain()

	events := snk.emails()
	require.Len(t, events, 1)
	ev := events[0]
	assert.Equal(t, "b***@example.com", ev.Recipient)
	assert.Equal(t, "failed", ev.Event)
	assert.Equal(t, "error", ev.Status)
	assert.Equal(t, wantErr.Error(), ev.Error)
	assert.Equal(t, "ses", ev.Provider)
	assert.NotEmpty(t, ev.MessageID)
}

func TestSendTruncatesOverlongSubjectAndError(t *testing.T) {
	client, snk, drain := newTestClient(t)
	longSubject := strings.Repeat("é", maxSubjectLen)
	longErr := errors.New(strings.Repeat("x", maxErrorLen*2))
	inner := &fakeProvider{sendErr: longErr}
	p := Wrap("smtp", inner, client)

	err := p.Send(context.Background(), &email.SendParams{
		To:      []string{"carol@example.com"},
		Subject: longSubject,
	})
	require.Error(t, err)

	drain()

	events := snk.emails()
	require.Len(t, events, 1)
	ev := events[0]

	assert.LessOrEqual(t, len(ev.Subject), maxSubjectLen)
	assert.True(t, utf8.ValidString(ev.Subject), "truncated subject must stay valid UTF-8")
	assert.LessOrEqual(t, len(ev.Error), maxErrorLen)
	assert.True(t, utf8.ValidString(ev.Error), "truncated error must stay valid UTF-8")
}

func TestSendBulkEmitsEventPerMessage(t *testing.T) {
	client, snk, drain := newTestClient(t)
	inner := &fakeProvider{}
	p := Wrap("smtp", inner, client)

	batch := []*email.SendParams{
		{To: []string{"alice@example.com"}, Subject: "One"},
		{To: []string{"bob@example.com"}, Subject: "Two"},
		{To: []string{"carol@example.com"}, Subject: "Three"},
	}
	require.NoError(t, p.SendBulk(context.Background(), batch))

	drain()

	require.Len(t, inner.bulk, 1, "SendBulk must delegate to the inner provider exactly once")
	require.Len(t, inner.bulk[0], len(batch))

	events := snk.emails()
	require.Len(t, events, len(batch))
	gotSubjects := map[string]bool{}
	for _, ev := range events {
		assert.Equal(t, "sent", ev.Event)
		assert.Equal(t, "smtp", ev.Provider)
		gotSubjects[ev.Subject] = true
	}
	assert.Equal(t, map[string]bool{"One": true, "Two": true, "Three": true}, gotSubjects)
}

func TestSendBulkAttributesPerMessageOnPartialFailure(t *testing.T) {
	client, snk, drain := newTestClient(t)
	bobErr := errors.New("bob mailbox full")
	batch := []*email.SendParams{
		{To: []string{"alice@example.com"}, Subject: "One"},
		{To: []string{"bob@example.com"}, Subject: "Two"},
		{To: []string{"carol@example.com"}, Subject: "Three"},
	}
	inner := &fakeProvider{bulkErr: &email.MultiError{
		Errors: []email_domain.EmailError{
			{Email: email.SendParams{To: []string{"bob@example.com"}, Subject: "Two"}, Error: bobErr},
		},
	}}
	p := Wrap("smtp", inner, client)

	err := p.SendBulk(context.Background(), batch)
	require.Error(t, err)

	drain()

	bySubject := map[string]telemetry_grpcfb.EmailEvent{}
	for _, ev := range snk.emails() {
		bySubject[ev.Subject] = ev
	}
	require.Len(t, bySubject, len(batch))

	assert.Equal(t, "sent", bySubject["One"].Event, "delivered message must not be marked failed")
	assert.Equal(t, "ok", bySubject["One"].Status)
	assert.Equal(t, "sent", bySubject["Three"].Event, "delivered message must not be marked failed")

	assert.Equal(t, "failed", bySubject["Two"].Event)
	assert.Equal(t, "error", bySubject["Two"].Status)
	assert.Equal(t, bobErr.Error(), bySubject["Two"].Error)
}

func TestSendBulkMarksUnknownOnOpaqueError(t *testing.T) {
	client, snk, drain := newTestClient(t)
	opaque := errors.New("transport reset")
	batch := []*email.SendParams{
		{To: []string{"alice@example.com"}, Subject: "One"},
		{To: []string{"bob@example.com"}, Subject: "Two"},
	}
	inner := &fakeProvider{bulkErr: opaque}
	p := Wrap("smtp", inner, client)

	err := p.SendBulk(context.Background(), batch)
	require.ErrorIs(t, err, opaque)

	drain()

	events := snk.emails()
	require.Len(t, events, len(batch))
	for _, ev := range events {
		assert.Equal(t, "bulk", ev.Event, "opaque bulk error must not assert per-message failure")
		assert.Equal(t, "unknown", ev.Status)
		assert.Empty(t, ev.Error)
	}
}

func TestSupportsBulkSendingDelegatesToInner(t *testing.T) {
	for name, want := range map[string]bool{"supported": true, "unsupported": false} {
		t.Run(name, func(t *testing.T) {
			p := Wrap("smtp", &fakeProvider{supportsBulk: want}, nil)
			assert.Equal(t, want, p.SupportsBulkSending())
		})
	}
}

func TestCloseDelegatesToInner(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inner := &fakeProvider{}
		p := Wrap("smtp", inner, nil)
		require.NoError(t, p.Close(context.Background()))
		assert.True(t, inner.closed)
	})

	t.Run("error is surfaced", func(t *testing.T) {
		wantErr := errors.New("close failed")
		inner := &fakeProvider{closeErr: wantErr}
		p := Wrap("smtp", inner, nil)
		assert.ErrorIs(t, p.Close(context.Background()), wantErr)
	})
}

func TestMaskRecipient(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "ascii local part", in: "alice@example.com", want: "a***@example.com"},
		{name: "multi-byte local part", in: "éve@example.com", want: "é***@example.com"},
		{name: "single rune local part is fully masked", in: "a@x.com", want: "***@x.com"},
		{name: "two rune local part is fully masked", in: "ab@example.com", want: "***@example.com"},
		{name: "two multi-byte rune local part is fully masked", in: "éé@example.com", want: "***@example.com"},
		{name: "three rune local part keeps first", in: "abc@example.com", want: "a***@example.com"},
		{name: "empty stays empty", in: "", want: ""},
		{name: "no at sign is fully masked", in: "noat", want: "***"},
		{name: "leading at sign is fully masked", in: "@example.com", want: "***"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := maskRecipient(tc.in)
			assert.Equal(t, tc.want, got)
			assert.True(t, utf8.ValidString(got), "masked recipient must stay valid UTF-8")
		})
	}
}

func TestMessageIDStable(t *testing.T) {
	a := messageID("a@b.com", "Hi", 1000)
	b := messageID("a@b.com", "Hi", 1000)
	assert.Equal(t, a, b, "messageID must be stable for identical inputs")
	assert.NotEqual(t, a, messageID("a@b.com", "Hi", 1001), "messageID must vary with timestamp")
	assert.NotEmpty(t, a)
}

func TestEmitIsNilSafe(t *testing.T) {
	assert.NotPanics(t, func() {
		var p *Provider
		p.emit(context.Background(), &email.SendParams{To: []string{"a@b.com"}}, nil)
	})
	assert.NotPanics(t, func() {
		p := Wrap("smtp", &fakeProvider{}, nil)
		p.emit(context.Background(), &email.SendParams{To: []string{"a@b.com"}}, nil)
	})
	assert.NotPanics(t, func() {
		p := Wrap("smtp", &fakeProvider{}, nil)
		p.emit(context.Background(), nil, nil)
	})
}
