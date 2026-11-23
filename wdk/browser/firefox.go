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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"piko.sh/json"
	"piko.sh/safedisk"
)

const (
	// defaultFirefoxMaxMessageBytes caps one WebDriver BiDi message from Firefox.
	defaultFirefoxMaxMessageBytes = 1 << 20

	// firefoxStartupPollInterval sets how often startup checks the WebDriver log.
	firefoxStartupPollInterval = 35 * time.Millisecond

	// firefoxCloseTimeout bounds the graceful browser.close request and the wait for Firefox
	// to exit before it is killed.
	firefoxCloseTimeout = 2 * time.Second

	// firefoxProfilePrefix names the private profile directory.
	firefoxProfilePrefix = "firefox-"

	// firefoxLogName is the WebDriver log inside the profile.
	firefoxLogName = "webdriver.log"

	// firefoxListeningMarker precedes the endpoint in the WebDriver log.
	firefoxListeningMarker = "WebDriver BiDi listening on "

	// executablePermissionBits are the permission bits that make a file executable.
	executablePermissionBits = 0o111
)

var (
	// ErrFirefoxSessionBroken reports a failed connection that requires session closure.
	ErrFirefoxSessionBroken = errors.New("firefox session connection is broken")

	// ErrFirefoxMessageTooLarge is returned when a reply exceeds the configured size.
	ErrFirefoxMessageTooLarge = errors.New("firefox message exceeds the size limit")

	// errFirefoxEndpoint is returned when Firefox announces an endpoint that is not a local
	// WebSocket.
	errFirefoxEndpoint = errors.New("unexpected Firefox endpoint")

	// errFirefoxCapabilities is returned when Firefox does not negotiate the expected
	// capabilities.
	errFirefoxCapabilities = errors.New("firefox capability negotiation failed")

	// errFirefoxOutOfOrder is returned when a reply answers a different command.
	errFirefoxOutOfOrder = errors.New("out-of-order WebDriver response")

	// errFirefoxEvaluation is returned when a script throws without a message.
	errFirefoxEvaluation = errors.New("firefox script evaluation failed")

	// errFirefoxLogFile is returned when the profile sandbox cannot supply a real file for
	// Firefox to write its log to.
	errFirefoxLogFile = errors.New("firefox log is not backed by a file")

	// errFirefoxUnsupportedKey is returned for a key name without a WebDriver code.
	errFirefoxUnsupportedKey = errors.New("unsupported key")

	// errFirefoxEarlyFrames reports frames received during the WebSocket handshake.
	errFirefoxEarlyFrames = errors.New("firefox sent frames during the WebSocket handshake")

	// errFirefoxSessionClosed is the cause recorded when Close ends the session.
	errFirefoxSessionClosed = errors.New("firefox session closed")

	// firefoxKeyActionTypes are the key actions sent for each typed character.
	firefoxKeyActionTypes = [...]string{"keyDown", "keyUp"}

	// firefoxNamedKeys maps supported key names to WebDriver codes.
	firefoxNamedKeys = map[string]string{"End": "\ue010", "Home": "\ue011", "Enter": "\ue007", "Escape": "\ue00c", "Tab": "\ue004"}
)

// FirefoxSession drives an isolated headless Firefox tab using WebDriver BiDi.
//
// Commands are serialised, as each reply must be read before the next command. Each
// command is bounded by its own context and by the context that opened the session.
// Cancelling a command while waiting for its reply can break the session.
type FirefoxSession struct {
	// conn is the WebDriver BiDi WebSocket connection.
	conn net.Conn

	// processErr is the Firefox process exit error, read after done closes.
	processErr error

	// closeErr is the result of the first Close.
	closeErr error

	// failure is the first connection error, after which commands fail fast.
	failure error

	// sessionDeadline is the deadline of the context that opened the session, or zero.
	sessionDeadline time.Time

	// cancel kills Firefox when the session ends.
	cancel context.CancelCauseFunc

	// done closes when the Firefox process has exited.
	done chan struct{}

	// stopConn unregisters the hook that closes the connection on cancellation.
	stopConn func() bool

	// profileFiles confines the session's own file access to the private profile.
	profileFiles safedisk.Sandbox

	// tab is the browsing context the session drives.
	tab string

	// version is the negotiated browser version.
	version string

	// profile is the private profile directory removed on Close.
	profile string

	// next is the identifier of the most recent command.
	next int

	// maxMessageBytes caps one reply.
	maxMessageBytes int64

	// commands serialises commands and their replies.
	commands sync.Mutex

	// closeOnce makes Close idempotent.
	closeOnce sync.Once
}

// FirefoxOption configures a Firefox session.
type FirefoxOption func(*FirefoxSession)

// firefoxReply is one WebDriver BiDi message from Firefox.
type firefoxReply struct {
	// Type is "success", "error" or "event".
	Type string `json:"type"`

	// Error is the protocol error code of a failed command.
	Error string `json:"error"`

	// Message describes a failed command.
	Message string `json:"message"`

	// Result is the command's undecoded result.
	Result rawJSON `json:"result"`

	// ID is the command the reply answers.
	ID int `json:"id"`
}

// rawJSON keeps a JSON value undecoded until its type is known.
type rawJSON []byte

// firefoxCommand is one WebDriver BiDi command sent to Firefox.
type firefoxCommand struct {
	// Params supplies the protocol arguments.
	Params any `json:"params"`

	// Method names the protocol operation.
	Method string `json:"method"`

	// ID correlates the command with its reply.
	ID int `json:"id"`
}

// firefoxContextTarget names the browsing context a command addresses.
type firefoxContextTarget struct {
	// Context is the browsing context identifier.
	Context string `json:"context"`
}

// firefoxNavigateParams are the arguments of browsingContext.navigate.
type firefoxNavigateParams struct {
	// Context is the browsing context to navigate.
	Context string `json:"context"`

	// URL is the address to load.
	URL string `json:"url"`

	// Wait is the readiness state to wait for.
	Wait string `json:"wait"`
}

// firefoxViewport is a viewport size in CSS pixels.
type firefoxViewport struct {
	// Width is the viewport width.
	Width int `json:"width"`

	// Height is the viewport height.
	Height int `json:"height"`
}

// firefoxViewportParams are the arguments of browsingContext.setViewport.
type firefoxViewportParams struct {
	// Context is the browsing context to resize.
	Context string `json:"context"`

	// Viewport is the new viewport size.
	Viewport firefoxViewport `json:"viewport"`
}

// firefoxKeyAction is one key press or release.
type firefoxKeyAction struct {
	// Type is keyDown or keyUp.
	Type string `json:"type"`

	// Value is the key's code point.
	Value string `json:"value"`
}

// firefoxKeySource is the keyboard input source of input.performActions.
type firefoxKeySource struct {
	// Type is the input source type, always key.
	Type string `json:"type"`

	// ID identifies the input source.
	ID string `json:"id"`

	// Actions lists the key actions in order.
	Actions []firefoxKeyAction `json:"actions"`
}

// firefoxActionsParams are the arguments of input.performActions.
type firefoxActionsParams struct {
	// Context is the browsing context that receives the input.
	Context string `json:"context"`

	// Actions lists the input sources and their actions.
	Actions []firefoxKeySource `json:"actions"`
}

// firefoxEvaluateParams are the arguments of script.evaluate.
type firefoxEvaluateParams struct {
	// Expression is the JavaScript to evaluate.
	Expression string `json:"expression"`

	// Target names the browsing context to evaluate in.
	Target firefoxContextTarget `json:"target"`

	// AwaitPromise waits for a returned promise to settle.
	AwaitPromise bool `json:"awaitPromise"`
}

// firefoxCreateParams are the arguments of browsingContext.create.
type firefoxCreateParams struct {
	// Type is the kind of browsing context to create.
	Type string `json:"type"`
}

// firefoxCapabilities is the capability request of session.new.
type firefoxCapabilities struct{}

// firefoxSessionParams are the arguments of session.new.
type firefoxSessionParams struct {
	// Capabilities requests session capabilities.
	Capabilities firefoxCapabilities `json:"capabilities"`
}

// firefoxEmptyParams are the arguments of a command that takes none.
type firefoxEmptyParams struct{}

// firefoxEvaluation is the result of script.evaluate.
type firefoxEvaluation struct {
	// Type is "success" or "exception".
	Type string `json:"type"`

	// ExceptionDetails describes a thrown exception.
	ExceptionDetails struct {
		// Text is the exception message.
		Text string `json:"text"`
	} `json:"exceptionDetails"`

	// Result is the evaluated value.
	Result struct {
		// Value is the string value.
		Value string `json:"value"`
	} `json:"result"`
}

// UnmarshalJSON keeps a copy of the undecoded value.
//
// Takes data ([]byte) which is the JSON value.
//
// Returns error which is always nil.
func (r *rawJSON) UnmarshalJSON(data []byte) error {
	*r = append((*r)[:0], data...)
	return nil
}

// Close stops Firefox and removes its private profile.
//
// Firefox has a short grace period to exit and is terminated if the time expires.
// Repeated calls wait for the first call and return its result.
//
// Returns error when cleaning up the profile fails.
func (b *FirefoxSession) Close() error {
	b.closeOnce.Do(func() {
		b.closeGracefully()
		if b.stopConn != nil {
			b.stopConn()
		}
		if b.conn != nil {
			_ = b.conn.Close()
		}
		b.cancel(errFirefoxSessionClosed)
		if b.done != nil {
			<-b.done
		}
		var sandboxErr error
		if b.profileFiles != nil {
			sandboxErr = b.profileFiles.Close()
		}
		if err := errors.Join(sandboxErr, os.RemoveAll(b.profile)); err != nil {
			b.closeErr = fmt.Errorf("removing Firefox profile: %w", err)
		}
	})
	return b.closeErr
}

// Version reports the browser version negotiated by the session.
//
// Returns string which is the Firefox version.
func (b *FirefoxSession) Version() string { return b.version }

// NavigateURL loads a URL and waits for the document to complete.
//
// Takes raw (string) which is the URL.
//
// Returns error when navigation fails.
func (b *FirefoxSession) NavigateURL(ctx context.Context, raw string) error {
	return b.command(ctx, "browsingContext.navigate", firefoxNavigateParams{Context: b.tab, URL: raw, Wait: "complete"}, nil)
}

// SetViewport changes viewport dimensions in CSS pixels.
//
// Takes width (int) which is the viewport width.
// Takes height (int) which is the viewport height.
//
// Returns error when the viewport cannot be set.
func (b *FirefoxSession) SetViewport(ctx context.Context, width, height int) error {
	params := firefoxViewportParams{Context: b.tab, Viewport: firefoxViewport{Width: width, Height: height}}
	return b.command(ctx, "browsingContext.setViewport", params, nil)
}

// Key sends literal text one code point at a time through native keyboard events.
//
// Takes text (string) which is the text to type.
//
// Returns error when the key actions fail.
func (b *FirefoxSession) Key(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}
	keys := make([]firefoxKeyAction, 0, len(firefoxKeyActionTypes)*utf8.RuneCountInString(text))
	for _, character := range text {
		for _, kind := range firefoxKeyActionTypes {
			keys = append(keys, firefoxKeyAction{Type: kind, Value: string(character)})
		}
	}
	params := firefoxActionsParams{Context: b.tab, Actions: []firefoxKeySource{{Type: "key", ID: "keyboard", Actions: keys}}}
	return b.command(ctx, "input.performActions", params, nil)
}

// PressKey sends a named WebDriver key to the focused element.
//
// Takes name (string) which is a key name such as Home or Enter.
//
// Returns error when the key is unsupported or the key action fails.
func (b *FirefoxSession) PressKey(ctx context.Context, name string) error {
	key, ok := firefoxNamedKeys[name]
	if !ok {
		return fmt.Errorf("%w %q", errFirefoxUnsupportedKey, name)
	}
	return b.Key(ctx, key)
}

// EvaluateString evaluates a JavaScript expression, awaits promises and returns its
// string result.
//
// The reply must fit the message size limit set by WithFirefoxMaxMessageBytes, which
// defaults to 1 MiB.
//
// Takes expr (string) which is the DOM-reading expression.
//
// Returns string which is the decoded JavaScript string result.
// Returns error which reports a protocol or JavaScript evaluation failure.
func (b *FirefoxSession) EvaluateString(ctx context.Context, expr string) (string, error) {
	var evaluation firefoxEvaluation
	params := firefoxEvaluateParams{Expression: expr, Target: firefoxContextTarget{Context: b.tab}, AwaitPromise: true}
	if err := b.command(ctx, "script.evaluate", params, &evaluation); err != nil {
		return "", err
	}
	if evaluation.Type != "success" {
		if evaluation.ExceptionDetails.Text == "" {
			return "", errFirefoxEvaluation
		}
		return "", fmt.Errorf("%w: %s", errFirefoxEvaluation, evaluation.ExceptionDetails.Text)
	}
	return evaluation.Result.Value, nil
}

// closeGracefully gives Firefox time to exit before the profile is removed.
//
// Concurrency: Sets a deadline on an in-flight command before waiting for the command
// lock to send browser.close.
func (b *FirefoxSession) closeGracefully() {
	if b.conn == nil || b.done == nil {
		return
	}
	deadline := time.Now().Add(firefoxCloseTimeout)
	if err := b.conn.SetDeadline(deadline); err != nil {
		return
	}
	b.commands.Lock()
	if b.failure == nil && b.conn.SetDeadline(deadline) == nil {
		_ = b.exchange("browser.close", firefoxEmptyParams{}, nil)
	}
	b.commands.Unlock()
	timer := time.NewTimer(firefoxCloseTimeout)
	defer timer.Stop()
	select {
	case <-b.done:
	case <-timer.C:
	}
}

// start launches Firefox and negotiates the session and its tab.
//
// The profile sandbox must supply an OS file that Firefox can write after the parent
// closes its handle.
//
// Takes binary (string) which is the Firefox executable.
//
// Returns error when Firefox cannot start or negotiation fails.
//
// Concurrency: A background waiter records the exit error before closing done.
func (b *FirefoxSession) start(ctx context.Context, binary string) error {
	logFile, err := b.profileFiles.Create(firefoxLogName)
	if err != nil {
		return fmt.Errorf("creating Firefox log: %w", err)
	}
	arguments := []string{"--headless", "--no-remote", "--profile", b.profile, "--remote-debugging-port", "0", "about:blank"}
	command := exec.CommandContext(ctx, binary, arguments...) //nolint:gosec // validated executable, fixed argv
	file, ok := logFile.(interface{ OSFile() *os.File })
	if !ok || file.OSFile() == nil {
		return errors.Join(errFirefoxLogFile, logFile.Close())
	}
	command.Stdout, command.Stderr = file.OSFile(), file.OSFile()
	err = command.Start()
	closeErr := logFile.Close()
	if err != nil {
		return fmt.Errorf("starting Firefox: %w", err)
	}
	b.done = make(chan struct{})
	go func() {
		b.processErr = command.Wait()
		close(b.done)
	}()
	if closeErr != nil {
		return fmt.Errorf("closing Firefox log: %w", closeErr)
	}
	endpoint, err := b.endpoint(ctx)
	if err != nil {
		return err
	}
	if err := b.connect(ctx, endpoint); err != nil {
		return err
	}
	return b.negotiate(ctx)
}

// connect opens the WebSocket connection and ties it to ctx.
//
// Takes endpoint (string) which is the local WebDriver BiDi URL.
//
// Returns error when the connection fails.
func (b *FirefoxSession) connect(ctx context.Context, endpoint string) error {
	conn, buffered, _, err := ws.Dial(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("connecting to Firefox: %w", err)
	}
	if buffered != nil {
		ws.PutReader(buffered)
		return errors.Join(errFirefoxEarlyFrames, conn.Close())
	}
	b.conn = conn
	b.stopConn = context.AfterFunc(ctx, func() { _ = conn.Close() })
	if deadline, ok := ctx.Deadline(); ok {
		b.sessionDeadline = deadline
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("setting Firefox deadline: %w", err)
		}
	}
	return nil
}

// negotiate creates the WebDriver session and the tab it drives.
//
// Returns error when negotiation fails.
func (b *FirefoxSession) negotiate(ctx context.Context) error {
	var session struct {
		Capabilities struct {
			BrowserName    string `json:"browserName"`
			BrowserVersion string `json:"browserVersion"`
		} `json:"capabilities"`
	}
	if err := b.command(ctx, "session.new", firefoxSessionParams{Capabilities: firefoxCapabilities{}}, &session); err != nil {
		return err
	}
	if session.Capabilities.BrowserName != "firefox" || session.Capabilities.BrowserVersion == "" {
		return errFirefoxCapabilities
	}
	b.version = session.Capabilities.BrowserVersion
	var tab struct {
		Context string `json:"context"`
	}
	if err := b.command(ctx, "browsingContext.create", firefoxCreateParams{Type: "tab"}, &tab); err != nil {
		return err
	}
	b.tab = tab.Context
	return nil
}

// endpoint waits for Firefox to log its WebDriver BiDi endpoint.
//
// Returns string which is the session endpoint URL.
// Returns error when Firefox exits, ctx ends or the endpoint is not local.
func (b *FirefoxSession) endpoint(ctx context.Context) (string, error) {
	logs, err := b.profileFiles.Open(firefoxLogName)
	if err != nil {
		return "", fmt.Errorf("opening Firefox log: %w", err)
	}
	defer func() { _ = logs.Close() }()
	reader := bufio.NewReader(logs)
	var line strings.Builder
	for {
		part, readErr := reader.ReadString('\n')
		line.WriteString(part)
		if strings.HasSuffix(part, "\n") {
			if endpoint, found, err := parseFirefoxEndpoint(line.String()); found {
				return endpoint, err
			}
			line.Reset()
		}
		if readErr == nil {
			continue
		}
		if !errors.Is(readErr, io.EOF) {
			return "", fmt.Errorf("reading Firefox log: %w", readErr)
		}
		if err := b.waitForLog(ctx); err != nil {
			return "", err
		}
	}
}

// waitForLog waits before the WebDriver log is read again.
//
// Returns error when ctx ends or Firefox exits first.
func (b *FirefoxSession) waitForLog(ctx context.Context) error {
	timer := time.NewTimer(firefoxStartupPollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting for Firefox WebDriver: %w", context.Cause(ctx))
	case <-b.done:
		return fmt.Errorf("firefox exited during startup: %w", b.processErr)
	case <-timer.C:
		return nil
	}
}

// command sends one WebDriver BiDi command bounded by ctx and decodes its result.
//
// If the socket times out at the caller's deadline, command waits for the context to
// record its cancellation cause.
//
// Takes method (string) which names the protocol operation.
// Takes params (any) which supplies the protocol arguments.
// Takes destination (any) which receives the decoded result, or nil.
//
// Returns error which wraps ErrFirefoxSessionBroken after a connection failure, joined
// with the cancellation cause when ctx interrupted the command, or reports a protocol or
// encoding error.
//
// Concurrency: Serialises exchanges so each reply reaches the command that sent it.
func (b *FirefoxSession) command(ctx context.Context, method string, params, destination any) error {
	b.commands.Lock()
	defer b.commands.Unlock()
	if b.failure != nil {
		return fmt.Errorf("%s: %w", method, b.failure)
	}
	if ctx.Err() != nil {
		return fmt.Errorf("%s: %w", method, context.Cause(ctx))
	}
	release, ctxDeadline, err := b.boundCommand(ctx)
	if err != nil {
		return b.fail(method, err)
	}
	err = b.exchange(method, params, destination)
	if restoreErr := release(); restoreErr != nil && err == nil {
		err = b.fail(method, restoreErr)
	}
	if err == nil {
		return nil
	}
	if ctxDeadline && errors.Is(err, os.ErrDeadlineExceeded) {
		<-ctx.Done()
	}
	if ctx.Err() != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	return err
}

// boundCommand applies a command's deadline and cancellation to the connection.
//
// The earlier of the command and session deadlines applies. The caller must hold the
// command lock until it calls release.
//
// Returns release (func() error) which stops watching ctx and restores the session
// deadline.
// Returns ctxDeadline (bool) which is true when ctx's deadline is the one applied.
// Returns err (error) when the deadline cannot be set.
func (b *FirefoxSession) boundCommand(ctx context.Context) (release func() error, ctxDeadline bool, err error) {
	deadline := b.sessionDeadline
	if commandDeadline, ok := ctx.Deadline(); ok && (deadline.IsZero() || commandDeadline.Before(deadline)) {
		deadline = commandDeadline
		ctxDeadline = true
	}
	if err := b.conn.SetDeadline(deadline); err != nil {
		return nil, false, fmt.Errorf("setting Firefox command deadline: %w", err)
	}
	interrupted := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(interrupted)
		_ = b.conn.SetDeadline(time.Unix(1, 0))
	})
	return func() error {
		if !stop() {
			<-interrupted
		}
		return b.conn.SetDeadline(b.sessionDeadline)
	}, ctxDeadline, nil
}

// exchange writes one command and reads its reply under the connection's current
// deadline.
//
// The caller must hold the command lock for the whole exchange.
//
// Takes method (string) which names the protocol operation.
// Takes params (any) which supplies the protocol arguments.
// Takes destination (any) which receives the decoded result, or nil.
//
// Returns error which wraps ErrFirefoxSessionBroken after a connection failure, or
// reports a browser-protocol failure.
func (b *FirefoxSession) exchange(method string, params, destination any) error {
	b.next++
	data, err := json.Marshal(firefoxCommand{ID: b.next, Method: method, Params: params})
	if err != nil {
		return fmt.Errorf("encoding %s: %w", method, err)
	}
	if err := wsutil.WriteClientText(b.conn, data); err != nil {
		return b.fail(method, err)
	}
	reply, err := b.readReply()
	if err != nil {
		return b.fail(method, err)
	}
	if reply.ID != b.next {
		return b.fail(method, errFirefoxOutOfOrder)
	}
	if reply.Type != "success" {
		return fmt.Errorf("%s: %s %s", method, reply.Error, reply.Message)
	}
	if destination != nil {
		if err := json.Unmarshal(reply.Result, destination); err != nil {
			return fmt.Errorf("decoding %s result: %w", method, err)
		}
	}
	return nil
}

// readReply reads messages until the reply to a command, skipping events and answering
// control frames.
//
// The caller must hold the command lock while reading.
//
// Returns firefoxReply which is the reply.
// Returns error when reading fails or a message is too large.
func (b *FirefoxSession) readReply() (firefoxReply, error) {
	for {
		payload, err := b.readMessage()
		if err != nil {
			return firefoxReply{}, err
		}
		if payload == nil {
			continue
		}
		var reply firefoxReply
		if err := json.Unmarshal(payload, &reply); err != nil {
			return firefoxReply{}, fmt.Errorf("decoding BiDi message: %w", err)
		}
		if reply.Type != "event" {
			return reply, nil
		}
	}
}

// readMessage reads the next frame, answering a control frame and returning the payload
// of a text message.
//
// The caller must hold the command lock while reading.
//
// Returns []byte which is the text payload, or nil after a control frame.
// Returns error when reading fails, the frame is not text or the message is too large.
func (b *FirefoxSession) readMessage() ([]byte, error) {
	controlHandler := wsutil.ControlFrameHandler(b.conn, ws.StateClientSide)
	reader := wsutil.NewReader(b.conn, ws.StateClientSide)
	reader.MaxFrameSize = b.maxMessageBytes
	reader.OnIntermediate = controlHandler
	header, err := reader.NextFrame()
	if errors.Is(err, wsutil.ErrFrameTooLarge) {
		return nil, ErrFirefoxMessageTooLarge
	}
	if err != nil {
		return nil, err
	}
	if header.OpCode.IsControl() {
		return nil, controlHandler(header, reader)
	}
	if header.OpCode != ws.OpText {
		return nil, fmt.Errorf("unexpected BiDi frame %d", header.OpCode)
	}
	payload, err := io.ReadAll(io.LimitReader(reader, b.maxMessageBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > b.maxMessageBytes {
		return nil, ErrFirefoxMessageTooLarge
	}
	return payload, nil
}

// fail records a connection failure and closes the connection, as the stream can no
// longer be read in step with the commands.
//
// The caller must hold the command lock while recording the failure.
//
// Takes method (string) which names the failed command.
// Takes cause (error) which is the failure.
//
// Returns error which wraps ErrFirefoxSessionBroken and the cause.
func (b *FirefoxSession) fail(method string, cause error) error {
	b.failure = fmt.Errorf("%w: %w", ErrFirefoxSessionBroken, cause)
	_ = b.conn.Close()
	return fmt.Errorf("%s: %w", method, b.failure)
}

// WithFirefoxMaxMessageBytes caps the size of a WebDriver BiDi message.
//
// An oversized message breaks the session.
//
// Takes limit (int64) which is the byte limit, or below one for the 1 MiB default.
//
// Returns FirefoxOption which applies the limit.
func WithFirefoxMaxMessageBytes(limit int64) FirefoxOption {
	return func(session *FirefoxSession) {
		if limit > 0 {
			session.maxMessageBytes = limit
		}
	}
}

// OpenFirefoxSession opens a headless Firefox tab with a private profile.
//
// The caller's context bounds startup and the session lifetime. Call Close to release the
// connection and remove the profile.
//
// Takes opts (...FirefoxOption) which adjust the session.
//
// Returns *FirefoxSession which owns the browser process.
// Returns error when Firefox is missing or cannot be started.
func OpenFirefoxSession(ctx context.Context, opts ...FirefoxOption) (*FirefoxSession, error) {
	binary, err := findFirefox()
	if err != nil {
		return nil, err
	}
	profile, err := os.MkdirTemp("", firefoxProfilePrefix)
	if err != nil {
		return nil, fmt.Errorf("creating Firefox profile: %w", err)
	}
	profileFiles, err := safedisk.NewSandboxWithOptions(profile, safedisk.ModeReadWrite, safedisk.WithNoSymlinks())
	if err != nil {
		return nil, errors.Join(fmt.Errorf("confining Firefox profile: %w", err), os.RemoveAll(profile))
	}
	ctx, cancel := context.WithCancelCause(ctx)
	var session FirefoxSession
	session.profile = profile
	session.profileFiles = profileFiles
	session.cancel = cancel
	session.maxMessageBytes = defaultFirefoxMaxMessageBytes
	for _, opt := range opts {
		opt(&session)
	}
	if err := session.start(ctx, binary); err != nil {
		return nil, errors.Join(err, session.Close())
	}
	return &session, nil
}

// findFirefox resolves the Firefox executable from PATH and checks that it is an
// absolute, executable file.
//
// Returns string which is the executable's absolute path.
// Returns error when Firefox is missing or not executable.
func findFirefox() (string, error) {
	binary, err := exec.LookPath("firefox")
	if err != nil {
		return "", fmt.Errorf("finding Firefox executable: %w", err)
	}
	binary, err = filepath.Abs(binary)
	if err != nil {
		return "", fmt.Errorf("resolving Firefox executable: %w", err)
	}
	info, err := os.Stat(binary)
	if err != nil {
		return "", fmt.Errorf("checking Firefox executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&executablePermissionBits == 0 {
		return "", fmt.Errorf("firefox executable %q is not an executable file", binary)
	}
	return binary, nil
}

// parseFirefoxEndpoint finds the WebDriver BiDi endpoint in one log line.
//
// Takes line (string) which is a complete log line.
//
// Returns string which is the session endpoint URL.
// Returns bool which is true when the line announces an endpoint.
// Returns error when the announced endpoint is not a local WebSocket.
func parseFirefoxEndpoint(line string) (string, bool, error) {
	_, announced, found := strings.Cut(line, firefoxListeningMarker)
	if !found {
		return "", false, nil
	}
	endpoint, err := url.Parse(strings.TrimSpace(announced))
	if err != nil {
		return "", true, fmt.Errorf("%w: %w", errFirefoxEndpoint, err)
	}
	host := endpoint.Hostname()
	if endpoint.Scheme != "ws" || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		return "", true, errFirefoxEndpoint
	}
	endpoint.Path = "/session"
	return endpoint.String(), true, nil
}
