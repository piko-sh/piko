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

package email_domain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/premailer"
)

func newTestService(provider EmailProviderPort) *service {
	if provider == nil {
		provider = &mockProvider{}
	}
	emailService := NewServiceWithProvider(context.Background(), provider)
	s, ok := emailService.(*service)
	if !ok {
		panic("expected service to be *service")
	}
	return s
}

func TestEmailBuilder_To(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().To("a@x.com").To("b@x.com", "c@x.com")
	if len(b.params.To) != 3 {
		t.Errorf("expected 3 To addresses, got %d", len(b.params.To))
	}
}

func TestEmailBuilder_Cc(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().Cc("cc@x.com")
	if len(b.params.Cc) != 1 || b.params.Cc[0] != "cc@x.com" {
		t.Errorf("unexpected Cc: %v", b.params.Cc)
	}
}

func TestEmailBuilder_Bcc(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().Bcc("bcc@x.com")
	if len(b.params.Bcc) != 1 || b.params.Bcc[0] != "bcc@x.com" {
		t.Errorf("unexpected Bcc: %v", b.params.Bcc)
	}
}

func TestEmailBuilder_From(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().From("sender@x.com")
	if b.params.From == nil || *b.params.From != "sender@x.com" {
		t.Error("expected From to be set")
	}
}

func TestEmailBuilder_Subject(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().Subject("hello")
	if b.params.Subject != "hello" {
		t.Errorf("expected subject 'hello', got %q", b.params.Subject)
	}
}

func TestEmailBuilder_BodyHTML(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().BodyHTML("<p>hi</p>")
	if b.params.BodyHTML != "<p>hi</p>" {
		t.Errorf("unexpected BodyHTML: %q", b.params.BodyHTML)
	}
}

func TestEmailBuilder_BodyPlain(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().BodyPlain("plain text")
	if b.params.BodyPlain != "plain text" {
		t.Errorf("unexpected BodyPlain: %q", b.params.BodyPlain)
	}
}

func TestEmailBuilder_Attachment(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().Attachment("file.pdf", "application/pdf", []byte("data"))
	if len(b.params.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(b.params.Attachments))
	}
	if b.params.Attachments[0].Filename != "file.pdf" {
		t.Error("unexpected filename")
	}
}

func TestEmailBuilder_Provider(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().Provider("ses")
	if b.providerName != "ses" {
		t.Errorf("expected provider 'ses', got %q", b.providerName)
	}
}

func TestEmailBuilder_Immediate(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().Immediate()
	if !b.immediateSend {
		t.Error("expected immediateSend = true")
	}
}

func TestEmailBuilder_ProviderOption(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().ProviderOption("key1", "val1").ProviderOption("key2", 42)
	if b.params.ProviderOptions == nil {
		t.Fatal("expected ProviderOptions to be initialised")
	}
	if b.params.ProviderOptions["key1"] != "val1" {
		t.Error("key1 not set")
	}
	if b.params.ProviderOptions["key2"] != 42 {
		t.Error("key2 not set")
	}
}

func TestEmailBuilder_Do_Success_ViaDispatcher(t *testing.T) {
	queued := false
	disp := &mockDispatcher{
		QueueFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			queued = true
			return nil
		},
	}
	s := newTestService(&mockProvider{})
	s.dispatcher = disp

	err := s.NewEmail().
		To("user@example.com").
		Subject("test").
		BodyPlain("hello").
		Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !queued {
		t.Error("expected email to be queued via dispatcher")
	}
}

func TestEmailBuilder_Do_ImmediateSend(t *testing.T) {
	sent := false
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			sent = true
			return nil
		},
	}
	disp := &mockDispatcher{
		QueueFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			t.Error("should not queue when Immediate is set")
			return nil
		},
	}
	s := newTestService(provider)
	s.dispatcher = disp

	err := s.NewEmail().
		To("user@example.com").
		Subject("test").
		BodyPlain("hello").
		Immediate().
		Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sent {
		t.Error("expected immediate send via provider")
	}
}

func TestEmailBuilder_Do_NamedProvider_BypassesDispatcher(t *testing.T) {
	sent := false
	namedProv := &mockProvider{
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			sent = true
			return nil
		},
	}
	s := newTestService(&mockProvider{})
	_ = s.RegisterProvider(context.Background(), "ses", namedProv)
	s.dispatcher = &mockDispatcher{}

	err := s.NewEmail().
		To("user@example.com").
		Subject("test").
		BodyPlain("hello").
		Provider("ses").
		Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sent {
		t.Error("expected named provider to send directly")
	}
}

func TestEmailBuilder_Do_NoProvider(t *testing.T) {
	emailService := NewService(context.Background())
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	err := s.NewEmail().
		To("user@example.com").
		Subject("test").
		BodyPlain("hello").
		Do(context.Background())
	if err == nil {
		t.Error("expected error when no provider configured")
	}
}

func TestEmailBuilder_Do_ValidationError(t *testing.T) {
	s := newTestService(&mockProvider{})
	err := s.NewEmail().
		Subject("test").
		BodyPlain("hello").
		Do(context.Background())
	if err == nil {
		t.Error("expected validation error for missing recipients")
	}
}

func TestEmailBuilder_Do_ProviderError(t *testing.T) {
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			return errors.New("send failed")
		},
	}
	s := newTestService(provider)
	err := s.NewEmail().
		To("user@example.com").
		Subject("test").
		BodyPlain("hello").
		Do(context.Background())
	if err == nil {
		t.Error("expected error from provider")
	}
}

func TestEmailBuilder_Build(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().
		To("a@x.com").
		Subject("test").
		BodyPlain("hello").
		ProviderOption("k", "v")
	params := b.Build()
	if params.Subject != "test" {
		t.Error("expected subject in built params")
	}
	if len(params.To) != 1 {
		t.Error("expected To in built params")
	}
	if params.ProviderOptions["k"] != "v" {
		t.Error("expected ProviderOptions in built params")
	}
}

func TestEmailBuilder_Build_IndependentCopy(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().To("a@x.com").Subject("original")
	params := b.Build()
	params.Subject = "modified"
	if b.params.Subject != "original" {
		t.Error("modifying built params should not affect builder")
	}
}

func TestEmailBuilder_Clone(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().To("a@x.com").Subject("original").BodyPlain("body")
	cloned := b.Clone()

	cloned.To("b@x.com")
	cloned.Subject("cloned")

	if len(b.params.To) != 1 {
		t.Error("clone modification should not affect original To")
	}
	if b.params.Subject != "original" {
		t.Error("clone modification should not affect original Subject")
	}
}

func TestEmailBuilder_Clone_PreservesProviderAndImmediate(t *testing.T) {
	s := newTestService(nil)
	b := s.NewEmail().Provider("ses").Immediate()
	cloned := b.Clone()
	if cloned.providerName != "ses" {
		t.Errorf("expected provider 'ses', got %q", cloned.providerName)
	}
	if !cloned.immediateSend {
		t.Error("expected immediateSend to be preserved")
	}
}

func TestTemplatedEmailBuilder_Props(t *testing.T) {
	type TestProps struct{ Name string }
	s := newTestService(&mockProvider{})
	s.templater = &mockTemplater{}
	builder, err := NewTemplatedEmail[TestProps](s)
	require.NoError(t, err)
	b := builder.Props(TestProps{Name: "Alice"})
	if b.templateProps.Name != "Alice" {
		t.Errorf("expected props.Name='Alice', got %q", b.templateProps.Name)
	}
}

func TestTemplatedEmailBuilder_BodyTemplate(t *testing.T) {
	s := newTestService(&mockProvider{})
	s.templater = &mockTemplater{}
	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	b := builder.BodyTemplate("emails/welcome.pk")
	if b.templatePath != "emails/welcome.pk" {
		t.Error("expected template path to be set")
	}
	if !b.useTemplate {
		t.Error("expected useTemplate = true")
	}
}

func TestTemplatedEmailBuilder_PremailerOptions(t *testing.T) {
	s := newTestService(&mockProvider{})
	s.templater = &mockTemplater{}
	opts := premailer.Options{RemoveClasses: true}
	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	b := builder.PremailerOptions(opts)
	if b.premailerOptions == nil {
		t.Fatal("expected premailer options to be set")
	}
	if !b.premailerOptions.RemoveClasses {
		t.Error("expected RemoveClasses=true")
	}
}

func TestTemplatedEmailBuilder_Do_Success(t *testing.T) {
	sent := false
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			sent = true
			return nil
		},
	}
	template := &mockTemplater{
		RenderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options) (*RenderedEmail, error) {
			return &RenderedEmail{HTML: "<p>hello</p>", PlainText: "hello"}, nil
		},
	}
	s := newTestService(provider)
	s.templater = template

	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	err = builder.
		To("user@example.com").
		Subject("welcome").
		BodyTemplate("emails/welcome.pk").
		Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sent {
		t.Error("expected email to be sent")
	}
}

func TestTemplatedEmailBuilder_Do_NoTemplater(t *testing.T) {
	s := newTestService(&mockProvider{})

	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	err = builder.
		To("user@example.com").
		Subject("welcome").
		BodyTemplate("emails/welcome.pk").
		Do(context.Background())
	if err == nil {
		t.Error("expected error when templater is nil")
	}
}

func TestTemplatedEmailBuilder_Do_RenderError(t *testing.T) {
	template := &mockTemplater{
		RenderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options) (*RenderedEmail, error) {
			return nil, errors.New("render failed")
		},
	}
	s := newTestService(&mockProvider{})
	s.templater = template

	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	err = builder.
		To("user@example.com").
		Subject("welcome").
		BodyTemplate("emails/welcome.pk").
		Do(context.Background())
	if err == nil {
		t.Error("expected render error")
	}
}

func TestTemplatedEmailBuilder_Do_PreservesUserPlainText(t *testing.T) {
	var capturedParams *email_dto.SendParams
	provider := &mockProvider{
		SendFunc: func(_ context.Context, p *email_dto.SendParams) error {
			capturedParams = p
			return nil
		},
	}
	template := &mockTemplater{
		RenderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options) (*RenderedEmail, error) {
			return &RenderedEmail{HTML: "<p>template</p>", PlainText: "template plain"}, nil
		},
	}
	s := newTestService(provider)
	s.templater = template

	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	err = builder.
		To("user@example.com").
		Subject("welcome").
		BodyPlain("custom plain text").
		BodyTemplate("emails/welcome.pk").
		Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedParams.BodyPlain != "custom plain text" {
		t.Errorf("expected user plain text preserved, got %q", capturedParams.BodyPlain)
	}
	if capturedParams.BodyHTML != "<p>template</p>" {
		t.Error("expected HTML from template")
	}
}

func TestTemplatedEmailBuilder_Do_WithAssets(t *testing.T) {
	var capturedParams *email_dto.SendParams
	provider := &mockProvider{
		SendFunc: func(_ context.Context, p *email_dto.SendParams) error {
			capturedParams = p
			return nil
		},
	}
	template := &mockTemplater{
		RenderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options) (*RenderedEmail, error) {
			return &RenderedEmail{
				HTML:      "<img src='cid:logo'>",
				PlainText: "text",
				AttachmentRequests: []*email_dto.EmailAssetRequest{
					{SourcePath: "assets/logo.png", CID: "logo"},
				},
			}, nil
		},
	}
	resolver := &mockAssetResolver{
		ResolveAssetsFunc: func(_ context.Context, reqs []*email_dto.EmailAssetRequest) ([]*email_dto.Attachment, []error) {
			attachments := make([]*email_dto.Attachment, len(reqs))
			errs := make([]error, len(reqs))
			for i, request := range reqs {
				attachments[i] = &email_dto.Attachment{
					Filename:  request.SourcePath,
					ContentID: request.CID,
					Content:   []byte("png data"),
				}
			}
			return attachments, errs
		},
	}
	s := newTestService(provider)
	s.templater = template
	s.assetResolver = resolver

	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	err = builder.
		To("user@example.com").
		Subject("welcome").
		BodyTemplate("emails/welcome.pk").
		Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedParams.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(capturedParams.Attachments))
	}
	if capturedParams.Attachments[0].ContentID != "logo" {
		t.Errorf("expected ContentID 'logo', got %q", capturedParams.Attachments[0].ContentID)
	}
}

func TestTemplatedEmailBuilder_Do_AssetResolutionErrors(t *testing.T) {
	provider := &mockProvider{}
	template := &mockTemplater{
		RenderFunc: func(_ context.Context, _ *http.Request, _ string, _ any, _ *premailer.Options) (*RenderedEmail, error) {
			return &RenderedEmail{
				HTML:      "<p>hi</p>",
				PlainText: "hi",
				AttachmentRequests: []*email_dto.EmailAssetRequest{
					{SourcePath: "assets/missing.png", CID: "img1"},
				},
			}, nil
		},
	}
	resolver := &mockAssetResolver{
		ResolveAssetsFunc: func(_ context.Context, reqs []*email_dto.EmailAssetRequest) ([]*email_dto.Attachment, []error) {
			return nil, []error{errors.New("not found")}
		},
	}
	s := newTestService(provider)
	s.templater = template
	s.assetResolver = resolver

	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	err = builder.
		To("user@example.com").
		Subject("welcome").
		BodyTemplate("emails/welcome.pk").
		Do(context.Background())

	if err != nil {
		t.Fatalf("expected email to be sent despite asset errors, got %v", err)
	}
}

func TestTemplatedEmailBuilder_Clone(t *testing.T) {
	s := newTestService(&mockProvider{})
	s.templater = &mockTemplater{}
	opts := premailer.Options{RemoveClasses: true}
	builder, err := NewTemplatedEmail[struct{}](s)
	require.NoError(t, err)
	b := builder.
		To("a@x.com").
		Subject("orig").
		BodyTemplate("emails/welcome.pk").
		PremailerOptions(opts)

	cloned := b.Clone()
	cloned.To("b@x.com")
	cloned.Subject("cloned")

	if len(b.params.To) != 1 {
		t.Error("clone modification should not affect original To")
	}
	if b.params.Subject != "orig" {
		t.Error("clone modification should not affect original Subject")
	}
	if cloned.premailerOptions == nil || !cloned.premailerOptions.RemoveClasses {
		t.Error("cloned builder should preserve premailer options")
	}
}

func TestNewTemplatedEmail_NonServiceImpl(t *testing.T) {
	type fakeService struct{ Service }
	builder, err := NewTemplatedEmail[struct{}](&fakeService{})
	require.Error(t, err, "expected error for non-*service implementation")
	require.ErrorIs(t, err, ErrUnsupportedServiceImpl)
	require.Nil(t, builder)
}

func TestNewTemplatedEmail_ReturnsErrorOnInvalidInput(t *testing.T) {
	type fakeService struct{ Service }
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewTemplatedEmail panicked instead of returning error: %v", r)
		}
	}()
	builder, err := NewTemplatedEmail[struct{}](&fakeService{})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnsupportedServiceImpl)
	require.Nil(t, builder)
}
