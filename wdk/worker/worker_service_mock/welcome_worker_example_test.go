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

package worker_service_mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/worker"
	"piko.sh/piko/wdk/worker/worker_service_mock"
)

type sendWelcomeArgs struct {
	UserID string `json:"user_id"`
}

func (sendWelcomeArgs) Kind() string { return "email:send_welcome" }

var (
	errPermanentBounce = errors.New("mail: permanent bounce")
)

type mailerSpy struct {
	failErr error
	sent    []string
}

func (m *mailerSpy) send(userID string) error {
	if m.failErr != nil {
		return m.failErr
	}
	m.sent = append(m.sent, userID)
	return nil
}

type sendWelcomeWorker struct {
	mailer *mailerSpy
}

func (w sendWelcomeWorker) Work(_ context.Context, job worker.Job[sendWelcomeArgs]) error {
	if job.Args.UserID == "" {
		return worker.Fatal(errors.New("send_welcome: empty user id"))
	}
	if err := w.mailer.send(job.Args.UserID); err != nil {
		if errors.Is(err, errPermanentBounce) {
			return worker.Fatal(err)
		}
		return errors.New("sending welcome email: " + err.Error())
	}
	return nil
}

func TestSendWelcomeWorker_FailureVocabulary(t *testing.T) {
	testCases := []struct {
		name      string
		args      sendWelcomeArgs
		mailerErr error
		wantErr   bool
		wantFatal bool
		wantSent  bool
	}{
		{name: "happy path", args: sendWelcomeArgs{UserID: "u1"}, wantSent: true},
		{name: "empty id is fatal", args: sendWelcomeArgs{UserID: ""}, wantErr: true, wantFatal: true},
		{name: "permanent bounce is fatal", args: sendWelcomeArgs{UserID: "u2"}, mailerErr: errPermanentBounce, wantErr: true, wantFatal: true},
		{name: "transient is retryable", args: sendWelcomeArgs{UserID: "u3"}, mailerErr: errors.New("smtp reset"), wantErr: true, wantFatal: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			spy := &mailerSpy{failErr: testCase.mailerErr}
			welcome := sendWelcomeWorker{mailer: spy}
			job := worker_service_mock.Job(worker_service_mock.JobConfig[sendWelcomeArgs]{Args: testCase.args, Attempt: 1, MaxAttempts: 5})

			err := welcome.Work(context.Background(), job)
			require.Equal(t, testCase.wantErr, err != nil, "Work error presence")
			require.Equal(t, testCase.wantFatal, worker.IsFatal(err), "IsFatal")
			require.Equal(t, testCase.wantSent, len(spy.sent) > 0, "sent")
		})
	}
}

func TestSendWelcome_EnqueueAndDrain(t *testing.T) {
	mock := worker_service_mock.NewMockWorkerService()
	worker.Register[sendWelcomeArgs](mock, sendWelcomeWorker{mailer: &mailerSpy{}})

	args := sendWelcomeArgs{UserID: "u1"}
	handle, err := worker.Enqueue[sendWelcomeArgs](context.Background(), mock, args)
	require.NoError(t, err, "Enqueue")
	require.NotEmpty(t, handle.ID)
	mock.AssertEnqueued(t, args)
	mock.AssertEnqueuedKind(t, "email:send_welcome", 1)

	state, err := mock.WorkOne(context.Background())
	require.NoError(t, err, "WorkOne")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.True(t, state.IsTerminal(), "drained state is terminal")

	_, err = mock.WorkOne(context.Background())
	require.ErrorIs(t, err, worker_service_mock.ErrNoJobs, "empty-queue WorkOne")
}
