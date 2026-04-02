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

//go:build integration

package email_test

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/pml/pml_adapters"
	"piko.sh/piko/internal/pml/pml_components"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/pml/pml_dto"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_test/pages/mocks"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/email/email_provider_postmark"
)

type TopLevelTestSpec struct {
	Title                 string                `json:"title"`
	Description           string                `json:"description"`
	PremailerOptions      *PremailerOptionsSpec `json:"premailerOptions,omitempty"`
	Metadata              MetadataSpec          `json:"metadata"`
	ExpectedErrorContains string                `json:"expectedErrorContains,omitempty"`
	ShouldError           bool                  `json:"shouldError,omitempty"`
}

type PremailerOptionsSpec struct {
	LinkQueryParams       map[string]string `json:"linkQueryParams,omitempty"`
	Theme                 map[string]string `json:"theme,omitempty"`
	KeepBangImportant     bool              `json:"keepBangImportant,omitempty"`
	RemoveClasses         bool              `json:"removeClasses,omitempty"`
	RemoveIDs             bool              `json:"removeIDs,omitempty"`
	MakeLeftoverImportant bool              `json:"makeLeftoverImportant,omitempty"`
	ExpandShorthands      bool              `json:"expandShorthands,omitempty"`
}

type MetadataSpec struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Language    string `json:"language,omitempty"`
}

type testCase struct {
	Name      string
	Path      string
	EntryFile string
}

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")
var sendEmails = flag.Bool("send", false, "Send rendered emails via Gmail (requires .env)")

type emailConfig struct {
	ServerToken string
	FromEmail   string
	SendTo      string
}

func loadDotEnv(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)

		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		env[strings.TrimSpace(key)] = value
	}
	return env
}

func loadEmailConfig() *emailConfig {
	env := loadDotEnv(".env")
	if env == nil {
		return nil
	}

	serverToken := env["POSTMARK_SERVER_TOKEN"]
	fromEmail := env["POSTMARK_FROM_EMAIL"]
	sendTo := env["SEND_TO"]

	if serverToken == "" || fromEmail == "" || sendTo == "" {
		return nil
	}

	return &emailConfig{ServerToken: serverToken, FromEmail: fromEmail, SendTo: sendTo}
}

func sendEmail(t *testing.T, cfg *emailConfig, tc testCase, spec TopLevelTestSpec, sfcResult *sfcparser.ParseResult, combinedCSS string) {
	t.Helper()

	ctx := context.Background()

	templateAST, parseErr := ast_domain.Parse(ctx, sfcResult.Template, filepath.Join(tc.Path, "src", tc.EntryFile), nil)
	if parseErr != nil {
		t.Logf("Failed to parse template for send: %v", parseErr)
		return
	}

	mockRegistry := mocks.NewMockRegistry(t)
	mockCSRF := mocks.NewMockCSRF()

	pmlRegistry, err := pml_components.RegisterBuiltIns(ctx)
	if err != nil {
		t.Logf("Failed to register PML components for send: %v", err)
		return
	}

	mediaCollector := pml_adapters.NewMediaQueryCollector()
	msoCollector := pml_adapters.NewMSOConditionalCollector()
	transformer := pml_domain.NewTransformer(pmlRegistry, mediaCollector, msoCollector)

	orchestrator := render_domain.NewRenderOrchestrator(
		transformer,
		[]render_domain.TransformationPort{},
		mockRegistry,
		mockCSRF,
	)

	pmOpts := buildPremailerOptions(spec.PremailerOptions)

	emailMetadata := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title:       spec.Metadata.Title,
			Description: spec.Metadata.Description,
			Language:    spec.Metadata.Language,
		},
	}
	if emailMetadata.Language == "" {
		emailMetadata.Language = "en"
	}

	var rendered bytes.Buffer
	req, reqErr := http.NewRequest("GET", "/test", nil)
	if reqErr != nil {
		t.Logf("Failed to create request for send: %v", reqErr)
		return
	}

	err = orchestrator.RenderEmail(ctx, &rendered, req, render_domain.RenderEmailOptions{
		PageID:           "test-email",
		Template:         templateAST,
		Metadata:         &emailMetadata,
		Styling:          combinedCSS,
		PremailerOptions: pmOpts,
	})
	if err != nil {
		t.Logf("Failed to render email for send: %v", err)
		return
	}

	srcDir := filepath.Join(tc.Path, "src")
	assetRequests := orchestrator.GetLastEmailAssetRequests()
	var attachments []email_dto.Attachment

	for _, asset := range assetRequests {
		filePath := filepath.Join(srcDir, asset.SourcePath)
		content, readErr := os.ReadFile(filePath)
		if readErr != nil {
			t.Logf("Skipping asset %s: %v", asset.SourcePath, readErr)
			continue
		}

		attachments = append(attachments, email_dto.Attachment{
			Filename:  filepath.Base(asset.SourcePath),
			MIMEType:  "image/png",
			ContentID: asset.CID,
			Content:   content,
		})
	}

	functionalOpts := pmOpts.ToFunctionalOptions()
	pm := premailer.New(templateAST, functionalOpts...)
	premailedAST, pmErr := pm.Transform()
	if pmErr != nil {
		premailedAST = templateAST
	}
	pmlConfig := pml_dto.DefaultConfig()
	htmlAST, _, _ := transformer.Transform(ctx, premailedAST, pmlConfig)
	if htmlAST == nil {
		htmlAST = premailedAST
	}
	plainText, _ := orchestrator.RenderASTToPlainText(ctx, htmlAST)

	provider, err := email_provider_postmark.NewPostmarkProvider(ctx, email_provider_postmark.PostmarkProviderArgs{
		ServerToken: cfg.ServerToken,
		FromEmail:   cfg.FromEmail,
	})
	if err != nil {
		t.Logf("Failed to create Postmark provider: %v", err)
		return
	}
	defer provider.Close(ctx)

	err = provider.Send(ctx, &email_dto.SendParams{
		To:          []string{cfg.SendTo},
		Subject:     fmt.Sprintf("[PML Test] %s", subject(spec)),
		BodyHTML:    rendered.String(),
		BodyPlain:   plainText,
		Attachments: attachments,
	})
	if err != nil {
		t.Logf("Failed to send email: %v", err)
		return
	}

	t.Logf("Sent email to %s with %d attachments: %s", cfg.SendTo, len(attachments), spec.Title)
}

func subject(spec TopLevelTestSpec) string {
	return spec.Title
}

func runTestCase(t *testing.T, tc testCase) {
	ctx := context.Background()

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	moduleResolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	if resolveErr := moduleResolver.DetectLocalModule(ctx); resolveErr != nil && !os.IsNotExist(resolveErr) {
		require.NoError(t, resolveErr)
	}

	specPath := filepath.Join(tc.Path, "testspec.json")
	specData, err := os.ReadFile(specPath)
	require.NoError(t, err, "testspec.json is required for test case: %s", tc.Name)

	var spec TopLevelTestSpec
	err = json.Unmarshal(specData, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)

	entryPath := filepath.Join(srcDir, tc.EntryFile)
	sourceBytes, err := os.ReadFile(entryPath)
	require.NoError(t, err, "Failed to read entry file: %s", entryPath)

	sfcResult, err := sfcparser.Parse(sourceBytes)
	require.NoError(t, err, "SFC parsing failed: %s", entryPath)
	require.NotNil(t, sfcResult, "SFC parser must return a non-nil result")

	var cssBuilder strings.Builder
	for idx, block := range sfcResult.Styles {
		if idx > 0 {
			cssBuilder.WriteString("\n")
		}
		cssBuilder.WriteString(block.Content)
	}
	combinedCSS := cssBuilder.String()

	templateAST, parseErr := ast_domain.Parse(ctx, sfcResult.Template, entryPath, nil)
	require.NoError(t, parseErr, "Fatal error during template parsing for: %s", tc.Name)
	require.NotNil(t, templateAST, "Parser must always return a non-nil AST")

	if len(templateAST.Diagnostics) > 0 && !spec.ShouldError {
		diagMessages := make([]string, 0, len(templateAST.Diagnostics))
		for _, diag := range templateAST.Diagnostics {
			diagMessages = append(diagMessages, diag.Error())
		}
		t.Fatalf("Unexpected parse diagnostics for '%s':\n%s", tc.Name, strings.Join(diagMessages, "\n"))
	}

	mockRegistry := mocks.NewMockRegistry(t)
	mockCSRF := mocks.NewMockCSRF()

	pmlRegistry, err := pml_components.RegisterBuiltIns(ctx)
	require.NoError(t, err, "Failed to register PML built-in components")

	mediaCollector := pml_adapters.NewMediaQueryCollector()
	msoCollector := pml_adapters.NewMSOConditionalCollector()

	transformer := pml_domain.NewTransformer(pmlRegistry, mediaCollector, msoCollector)

	orchestrator := render_domain.NewRenderOrchestrator(
		transformer,
		[]render_domain.TransformationPort{},
		mockRegistry,
		mockCSRF,
	)

	pmOpts := buildPremailerOptions(spec.PremailerOptions)

	emailMetadata := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title:       spec.Metadata.Title,
			Description: spec.Metadata.Description,
			Language:    spec.Metadata.Language,
		},
	}
	if emailMetadata.Language == "" {
		emailMetadata.Language = "en"
	}

	var rendered bytes.Buffer
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	err = orchestrator.RenderEmail(
		ctx,
		&rendered,
		req,
		render_domain.RenderEmailOptions{
			PageID:           "test-email",
			Template:         templateAST,
			Metadata:         &emailMetadata,
			Styling:          combinedCSS,
			PremailerOptions: pmOpts,
		},
	)

	if spec.ShouldError {
		require.Error(t, err, "Expected rendering to fail, but it succeeded for: %s", tc.Name)

		if spec.ExpectedErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ExpectedErrorContains,
				"Error message did not contain expected text")
		}
		return
	}

	require.NoError(t, err, "Rendering failed unexpectedly for: %s", tc.Name)

	actualHTML := rendered.String()
	require.NotEmpty(t, actualHTML, "Rendered HTML must not be empty")

	functionalOpts := pmOpts.ToFunctionalOptions()
	pm := premailer.New(templateAST, functionalOpts...)
	premailedAST, pmErr := pm.Transform()
	if pmErr != nil {
		premailedAST = templateAST
	}

	pmlConfig := pml_dto.DefaultConfig()
	htmlAST, _, _ := transformer.Transform(ctx, premailedAST, pmlConfig)
	if htmlAST == nil {
		htmlAST = premailedAST
	}

	plainText, err := orchestrator.RenderASTToPlainText(ctx, htmlAST)
	require.NoError(t, err, "Plain text generation failed for: %s", tc.Name)

	goldenDir := filepath.Join(tc.Path, "golden")
	require.NoError(t, os.MkdirAll(goldenDir, 0755), "Failed to create golden directory")

	plainTextPath := filepath.Join(goldenDir, "actual.txt")
	require.NoError(t, os.WriteFile(plainTextPath, []byte(plainText), 0644),
		"Failed to write plain text file")

	goldenHTMLPath := filepath.Join(goldenDir, "formatted.html")

	if *updateGoldenFiles {
		prettified := prettifyHTML(actualHTML)
		require.NoError(t, os.WriteFile(goldenHTMLPath, []byte(prettified), 0644),
			"Failed to write golden file")
	}

	expectedBytes, err := os.ReadFile(goldenHTMLPath)
	require.NoError(t, err, "Failed to read golden file for '%s'. Run with -update flag to generate it.", tc.Name)

	expectedNorm := normaliseHTML(string(expectedBytes))
	actualNorm := normaliseHTML(actualHTML)

	actualPath := filepath.Join(goldenDir, "actual.html")
	_ = os.WriteFile(actualPath, []byte(actualHTML), 0644)

	assert.Equal(t, expectedNorm, actualNorm,
		"HTML output mismatch for '%s'. Run with -update to regenerate golden file.", tc.Name)

	if *sendEmails {
		cfg := loadEmailConfig()
		if cfg == nil {
			t.Log("Skipping send: .env not found or incomplete (need POSTMARK_SERVER_TOKEN, POSTMARK_FROM_EMAIL, SEND_TO)")
			return
		}
		sendEmail(t, cfg, tc, spec, sfcResult, combinedCSS)
	}
}

func buildPremailerOptions(spec *PremailerOptionsSpec) *premailer.Options {
	if spec != nil {
		return &premailer.Options{
			KeepBangImportant:     spec.KeepBangImportant,
			RemoveClasses:         spec.RemoveClasses,
			RemoveIDs:             spec.RemoveIDs,
			MakeLeftoverImportant: spec.MakeLeftoverImportant,
			ExpandShorthands:      spec.ExpandShorthands,
			LinkQueryParams:       spec.LinkQueryParams,
			Theme:                 spec.Theme,
		}
	}

	return &premailer.Options{
		ExpandShorthands:      true,
		MakeLeftoverImportant: true,
		KeepBangImportant:     false,
	}
}

func normaliseHTML(html string) string {
	normalised := strings.ReplaceAll(html, "\r\n", "\n")

	var buf strings.Builder
	insideTag := false
	prevWasTag := false

	for i := 0; i < len(normalised); i++ {
		ch := normalised[i]

		if ch == '<' {
			insideTag = true
			prevWasTag = true
			buf.WriteByte(ch)
			continue
		}

		if ch == '>' {
			insideTag = false
			buf.WriteByte(ch)
			continue
		}

		if insideTag {
			buf.WriteByte(ch)
			continue
		}

		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {

			if prevWasTag {
				continue
			}

			if i+1 < len(normalised) && (normalised[i+1] == ' ' || normalised[i+1] == '\t' || normalised[i+1] == '\n' || normalised[i+1] == '\r' || normalised[i+1] == '<') {
				continue
			}
			buf.WriteByte(' ')
		} else {
			prevWasTag = false
			buf.WriteByte(ch)
		}
	}

	return strings.TrimSpace(buf.String())
}

func prettifyHTML(html string) string {
	var buf strings.Builder
	depth := 0
	pos := 0

	blockElements := map[string]bool{
		"html": true, "head": true, "body": true, "div": true, "table": true,
		"tr": true, "td": true, "th": true, "thead": true, "tbody": true,
		"style": true, "script": true, "title": true, "meta": true,
	}

	for pos < len(html) {

		for pos < len(html) && (html[pos] == ' ' || html[pos] == '\t' || html[pos] == '\n' || html[pos] == '\r') {
			pos++
		}

		if pos >= len(html) {
			break
		}

		if html[pos] == '<' {

			end := pos + 1
			for end < len(html) && html[end] != '>' {
				end++
			}

			if end < len(html) {
				fullTag := html[pos : end+1]

				nameStart := pos + 1
				if nameStart < len(html) && html[nameStart] == '/' {
					nameStart++
				}
				nameEnd := nameStart
				for nameEnd < len(html) && html[nameEnd] != ' ' && html[nameEnd] != '>' && html[nameEnd] != '/' {
					nameEnd++
				}

				elemName := ""
				if nameEnd > nameStart {
					elemName = strings.ToLower(html[nameStart:nameEnd])
				}

				isClosingTag := len(fullTag) > 1 && fullTag[1] == '/'
				isSelfClosingTag := len(fullTag) > 1 && fullTag[len(fullTag)-2] == '/'
				isBlockElem := blockElements[elemName]

				if isClosingTag && isBlockElem {
					depth--
					if depth < 0 {
						depth = 0
					}
				}

				if isBlockElem {
					buf.WriteString("\n")
					buf.WriteString(strings.Repeat("  ", depth))
				}

				buf.WriteString(fullTag)

				if !isClosingTag && !isSelfClosingTag && isBlockElem {
					depth++
				}

				pos = end + 1
				continue
			}
		}

		textStart := pos
		for pos < len(html) && html[pos] != '<' {
			pos++
		}

		if pos > textStart {
			segment := strings.Trim(html[textStart:pos], " \t\n\r")
			if segment != "" {
				buf.WriteString(segment)
			}
		}
	}

	return buf.String()
}
