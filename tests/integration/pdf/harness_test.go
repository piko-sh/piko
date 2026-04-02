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

package pdf_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"piko.sh/piko"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/caller"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/generator/generator_helpers"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/shutdown"
	browserpkg "piko.sh/piko/wdk/browser"
	"piko.sh/piko/wdk/logger"
)

const pdfLayouterCSSReset = `*, *::before, *::after {
  box-sizing: border-box;
}
html, body, div, span, applet, object, iframe,
h1, h2, h3, h4, h5, h6, p, blockquote, pre,
a, abbr, acronym, address, big, cite, code,
del, dfn, em, img, ins, kbd, q, s, samp,
small, strike, strong, sub, sup, tt, var,
b, u, i, center,
dl, dt, dd, ol, ul, li,
fieldset, form, label, legend,
table, caption, tbody, tfoot, thead, tr, th, td,
article, aside, canvas, details, embed,
figure, figcaption, footer, header, hgroup,
menu, nav, output, ruby, section, summary,
time, mark, audio, video {
  margin: 0;
  padding: 0;
  border: 0;
  vertical-align: baseline;
}
body {
  line-height: 1.4;
}
img {
  max-width: 100%;
  max-height: 100%;
}
a {
  text-decoration: none;
}
`

type pdfHarness struct {
	t               *testing.T
	testCase        testCase
	spec            *pdfTestSpec
	tempDirectory   string
	sourceDirectory string
	serverCommand   *exec.Cmd
	serverPort      int
	serverURL       string
	incognitoPage   *browserpkg.IncognitoPage
	pageHelper      *browserpkg.PageHelper
}

var serverMainTemplate = template.Must(template.New("main.go").Parse(`package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"piko.sh/piko"
	"piko.sh/piko/wdk/pdf"
	_ "{{.ModuleName}}/dist"
)

const cssResetWithNotoSans = ` + "`" + `{{.CSSReset}}` + "`" + `

func main() {
	if os.Getenv("PIKO_EXTRACT_PDF") == "1" {
		extractPdf()
		return
	}

	portString := os.Getenv("PIKO_TEST_PORT")
	if portString == "" {
		portString = "8080"
	}
	port, _ := strconv.Atoi(portString)

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetPKOverride(cssResetWithNotoSans)),
	)
	server.Configure(piko.PublicConfig{
		BaseDir:        ".",
		PagesSourceDir: "pages",
		Port:           port,
	})

	fmt.Printf("Test server starting on port %s\n", portString)
	if err := server.Run(piko.RunModeProd); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func extractPdf() {
	manifestPath := os.Getenv("PIKO_MANIFEST_PATH")
	pdfPath := os.Getenv("PIKO_PDF_PATH")

	var serviceOpts []pdf.ServiceOption
	if os.Getenv("PIKO_FONT_REGULAR_ONLY") == "1" {
		serviceOpts = append(serviceOpts, pdf.WithExcludeDefaultBold())
	}
	if variableFontPath := os.Getenv("PIKO_FONT_VARIABLE_PATH"); variableFontPath != "" {
		variableFontData, readErr := os.ReadFile(variableFontPath)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "reading variable font: %v\n", readErr)
			os.Exit(1)
		}
		serviceOpts = append(serviceOpts, pdf.WithExcludeDefaultBold())
		serviceOpts = append(serviceOpts, pdf.WithVariableFont(
			"NotoSans", 100, 900, 0, variableFontData,
		))
	}

	service, err := pdf.NewServiceFromManifest(context.Background(), manifestPath, serviceOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PDF service creation failed: %v\n", err)
		os.Exit(1)
	}

	builder := service.NewRender().Template(pdfPath)
	if extraCSS := os.Getenv("PIKO_EXTRA_CSS"); extraCSS != "" {
		builder.Stylesheet(extraCSS)
	}

	userPassword := os.Getenv("PIKO_ENCRYPT_USER_PASSWORD")
	ownerPassword := os.Getenv("PIKO_ENCRYPT_OWNER_PASSWORD")
	if userPassword != "" {
		registry := pdf.NewTransformerRegistry()
		if regErr := registry.Register(pdf.NewEncryptTransformer()); regErr != nil {
			fmt.Fprintf(os.Stderr, "registering encrypt transformer: %v\n", regErr)
			os.Exit(1)
		}
		builder.Watermark("CONFIDENTIAL")
		builder.Transformations(registry, pdf.TransformConfig{
			EnabledTransformers: []string{"pdf-encrypt"},
			TransformerOptions: map[string]any{
				"pdf-encrypt": pdf.EncryptionOptions{
					Algorithm:     "aes-256",
					UserPassword:  userPassword,
					OwnerPassword: ownerPassword,
					Permissions:   0xFFFFF0C4,
				},
			},
		})
	}

	result, err := builder.Do(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "PDF rendering failed: %v\n", err)
		os.Exit(1)
	}

	if dumpPath := os.Getenv("PIKO_LAYOUT_DUMP_PATH"); dumpPath != "" && result.LayoutDump != "" {
		if writeErr := os.WriteFile(dumpPath, []byte(result.LayoutDump), 0644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "writing layout dump: %v\n", writeErr)
		}
	}

	os.Stdout.Write(result.Content)
}
`))

func buildNotoSansCSSReset() string {
	var builder strings.Builder
	builder.WriteString(render_domain.DefaultCSSResetComplete)
	builder.WriteString("@font-face {\n")
	builder.WriteString("  font-family: 'NotoSans';\n")
	builder.WriteString("  src: url('data:font/ttf;base64,")
	builder.WriteString(base64.StdEncoding.EncodeToString(fonts.NotoSansRegularTTF))
	builder.WriteString("') format('truetype');\n")
	builder.WriteString("  font-weight: 400;\n")
	builder.WriteString("  font-style: normal;\n")
	builder.WriteString("  font-display: block;\n")
	builder.WriteString("}\n")
	builder.WriteString("@font-face {\n")
	builder.WriteString("  font-family: 'NotoSans';\n")
	builder.WriteString("  src: url('data:font/ttf;base64,")
	builder.WriteString(base64.StdEncoding.EncodeToString(fonts.NotoSansBoldTTF))
	builder.WriteString("') format('truetype');\n")
	builder.WriteString("  font-weight: 700;\n")
	builder.WriteString("  font-style: normal;\n")
	builder.WriteString("  font-display: block;\n")
	builder.WriteString("}\n")
	builder.WriteString("*, *::before, *::after { font-family: 'NotoSans', sans-serif; }\n")
	builder.WriteString("body { font-family: 'NotoSans', sans-serif; }\n")
	builder.WriteString("h1, h2, h3, h4, h5, h6 { font-family: 'NotoSans', sans-serif; }\n")
	return builder.String()
}

func buildNotoSansRegularOnlyCSSReset() string {
	var builder strings.Builder
	builder.WriteString(render_domain.DefaultCSSResetComplete)
	builder.WriteString("@font-face {\n")
	builder.WriteString("  font-family: 'NotoSans';\n")
	builder.WriteString("  src: url('data:font/ttf;base64,")
	builder.WriteString(base64.StdEncoding.EncodeToString(fonts.NotoSansRegularTTF))
	builder.WriteString("') format('truetype');\n")
	builder.WriteString("  font-weight: 400;\n")
	builder.WriteString("  font-style: normal;\n")
	builder.WriteString("  font-display: block;\n")
	builder.WriteString("}\n")
	builder.WriteString("*, *::before, *::after { font-family: 'NotoSans', sans-serif; }\n")
	builder.WriteString("body { font-family: 'NotoSans', sans-serif; }\n")
	builder.WriteString("h1, h2, h3, h4, h5, h6 { font-family: 'NotoSans', sans-serif; }\n")
	return builder.String()
}

func buildNotoSansVariableCSSReset() string {
	var builder strings.Builder
	builder.WriteString(render_domain.DefaultCSSResetComplete)
	builder.WriteString("@font-face {\n")
	builder.WriteString("  font-family: 'NotoSans';\n")
	builder.WriteString("  src: url('data:font/ttf;base64,")
	builder.WriteString(base64.StdEncoding.EncodeToString(fonts.NotoSansVariableTTF))
	builder.WriteString("') format('truetype');\n")
	builder.WriteString("  font-weight: 100 900;\n")
	builder.WriteString("  font-style: normal;\n")
	builder.WriteString("  font-display: block;\n")
	builder.WriteString("}\n")
	builder.WriteString("*, *::before, *::after { font-family: 'NotoSans', sans-serif; }\n")
	builder.WriteString("body { font-family: 'NotoSans', sans-serif; }\n")
	builder.WriteString("h1, h2, h3, h4, h5, h6 { font-family: 'NotoSans', sans-serif; }\n")
	return builder.String()
}

func newPdfHarness(t *testing.T, tc testCase) *pdfHarness {
	t.Helper()

	return &pdfHarness{
		t:               t,
		testCase:        tc,
		sourceDirectory: filepath.Join(tc.Path, "src"),
	}
}

func (h *pdfHarness) setup() error {
	resetGlobalStateForTestIsolation()

	specPath := filepath.Join(h.testCase.Path, "testspec.json")
	spec, err := loadTestSpec(specPath)
	if err != nil {
		return fmt.Errorf("loading test spec: %w", err)
	}
	h.spec = spec

	h.tempDirectory = h.t.TempDir()

	return nil
}

func (h *pdfHarness) buildBinary() error {
	h.t.Helper()

	absoluteSourceDirectory, err := filepath.Abs(h.sourceDirectory)
	if err != nil {
		return fmt.Errorf("getting absolute src path: %w", err)
	}

	pikoDirectory := filepath.Join(absoluteSourceDirectory, ".piko")
	if _, err := os.Stat(pikoDirectory); err == nil {
		if err := os.RemoveAll(pikoDirectory); err != nil {
			return fmt.Errorf("removing .piko directory: %w", err)
		}
	}

	sourceDistDirectory := filepath.Join(absoluteSourceDirectory, "dist")
	if _, err := os.Stat(sourceDistDirectory); err == nil {
		if err := os.RemoveAll(sourceDistDirectory); err != nil {
			return fmt.Errorf("removing dist directory: %w", err)
		}
	}

	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := os.Chdir(absoluteSourceDirectory); err != nil {
		return fmt.Errorf("changing to source directory: %w", err)
	}

	sourceTidyCommand := exec.Command("go", "mod", "tidy")
	sourceTidyCommand.Dir = absoluteSourceDirectory
	sourceTidyCommand.Env = append(os.Environ(), "GOWORK=off")
	if output, err := sourceTidyCommand.CombinedOutput(); err != nil {
		_ = os.Chdir(originalWorkingDirectory)
		return fmt.Errorf("go mod tidy in source directory failed: %w\nOutput: %s", err, string(output))
	}

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	)
	err = server.Generate(context.Background(), piko.GenerateModeAll)
	server.Close()

	if restoreError := os.Chdir(originalWorkingDirectory); restoreError != nil {
		return fmt.Errorf("restoring working directory: %w", restoreError)
	}

	if err != nil {
		return fmt.Errorf("piko.Generate failed: %w", err)
	}

	if err := copyDirectory(absoluteSourceDirectory, h.tempDirectory); err != nil {
		return fmt.Errorf("copying source to temp: %w", err)
	}

	dependencyFile := filepath.Join(h.tempDirectory, "piko_dependency.go")
	if _, err := os.Stat(dependencyFile); err == nil {
		if err := os.Remove(dependencyFile); err != nil {
			return fmt.Errorf("removing piko_dependency.go: %w", err)
		}
	}

	pikoProjectRoot, err := findPikoProjectRoot()
	if err != nil {
		return fmt.Errorf("finding piko project root: %w", err)
	}

	goModPath := filepath.Join(h.tempDirectory, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	goModString := string(goModContent)
	replacePattern := regexp.MustCompile(`replace piko\.sh/piko => \.\./[^\n]+`)
	updatedGoModContent := replacePattern.ReplaceAllString(goModString,
		fmt.Sprintf("replace piko.sh/piko => %s", pikoProjectRoot))

	if err := os.WriteFile(goModPath, []byte(updatedGoModContent), 0644); err != nil {
		return fmt.Errorf("updating go.mod: %w", err)
	}

	goWorkContent := fmt.Sprintf("go 1.25.1\n\nuse (\n\t.\n\t%s\n)\n", pikoProjectRoot)
	if err := os.WriteFile(filepath.Join(h.tempDirectory, "go.work"), []byte(goWorkContent), 0644); err != nil {
		return fmt.Errorf("creating go.work: %w", err)
	}

	moduleName := ""
	for line := range strings.SplitSeq(updatedGoModContent, "\n") {
		if trimmed, found := strings.CutPrefix(line, "module "); found {
			moduleName = strings.TrimSpace(trimmed)
			break
		}
	}
	if moduleName == "" {
		return errors.New("could not find module name in go.mod")
	}

	mainGoPath := filepath.Join(h.tempDirectory, "main.go")
	mainGoFile, err := os.Create(mainGoPath)
	if err != nil {
		return fmt.Errorf("creating main.go: %w", err)
	}
	defer func() { _ = mainGoFile.Close() }()

	cssReset := buildNotoSansCSSReset()
	if h.spec.RegularOnly {
		cssReset = buildNotoSansRegularOnlyCSSReset()
	} else if h.spec.VariableFont {
		cssReset = buildNotoSansVariableCSSReset()
	}

	if err := serverMainTemplate.Execute(mainGoFile, map[string]string{
		"ModuleName": moduleName,
		"CSSReset":   cssReset,
	}); err != nil {
		return fmt.Errorf("executing main.go template: %w", err)
	}

	tidyCommand := exec.Command("go", "mod", "tidy")
	tidyCommand.Dir = h.tempDirectory
	tidyCommand.Env = append(os.Environ(), "GOWORK=off")
	if output, err := tidyCommand.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\nOutput: %s", err, string(output))
	}

	buildCommand := exec.Command("go", "build", "-o", "server", ".")
	buildCommand.Dir = h.tempDirectory
	buildCommand.Env = append(os.Environ(), "GOWORK=off")
	if output, err := buildCommand.CombinedOutput(); err != nil {
		return fmt.Errorf("go build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

type pdfRenderResult struct {
	pdfBytes   []byte
	layoutDump string
}

func (h *pdfHarness) renderPdf() (*pdfRenderResult, error) {
	h.t.Helper()

	manifestPath := filepath.Join(h.tempDirectory, "dist", "manifest.bin")
	layoutDumpPath := filepath.Join(h.tempDirectory, "layout_dump.txt")

	command := exec.Command(filepath.Join(h.tempDirectory, "server"))
	command.Dir = h.tempDirectory
	command.Env = append(os.Environ(),
		"PIKO_EXTRACT_PDF=1",
		"PIKO_MANIFEST_PATH="+manifestPath,
		"PIKO_PDF_PATH="+h.spec.PdfPath,
		"PIKO_LAYOUT_DUMP_PATH="+layoutDumpPath,
		"PIKO_EXTRA_CSS="+pdfLayouterCSSReset,
		"GOWORK=off",
	)
	if h.spec.RegularOnly {
		command.Env = append(command.Env, "PIKO_FONT_REGULAR_ONLY=1")
	}
	if h.spec.VariableFont {
		variableFontPath := filepath.Join(h.tempDirectory, "NotoSans-Variable.ttf")
		if writeErr := os.WriteFile(variableFontPath, fonts.NotoSansVariableTTF, 0644); writeErr != nil {
			return nil, fmt.Errorf("writing variable font file: %w", writeErr)
		}
		command.Env = append(command.Env, "PIKO_FONT_VARIABLE_PATH="+variableFontPath)
	}
	if h.spec.Encryption != nil {
		command.Env = append(command.Env,
			"PIKO_ENCRYPT_USER_PASSWORD="+h.spec.Encryption.UserPassword,
			"PIKO_ENCRYPT_OWNER_PASSWORD="+h.spec.Encryption.OwnerPassword,
		)
	}

	output, err := command.Output()
	if err != nil {
		if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, fmt.Errorf("PDF rendering failed: %w\nStderr: %s", err, string(exitError.Stderr))
		}
		return nil, fmt.Errorf("PDF rendering failed: %w", err)
	}

	var layoutDump string
	if dumpBytes, readErr := os.ReadFile(layoutDumpPath); readErr == nil {
		layoutDump = string(dumpBytes)
	}

	return &pdfRenderResult{
		pdfBytes:   output,
		layoutDump: layoutDump,
	}, nil
}

func (h *pdfHarness) startServer() error {
	h.t.Helper()

	port, err := browserpkg.FindAvailablePort()
	if err != nil {
		return fmt.Errorf("finding available port: %w", err)
	}
	h.serverPort = port
	h.serverURL = fmt.Sprintf("http://localhost:%d", h.serverPort)

	h.serverCommand = exec.Command("./server")
	h.serverCommand.Dir = h.tempDirectory
	h.serverCommand.Env = append(os.Environ(),
		fmt.Sprintf("PIKO_TEST_PORT=%d", h.serverPort),
		fmt.Sprintf("PIKO_PORT=%d", h.serverPort),
		"PIKO_RATE_LIMIT_ENABLED=true",
	)

	var stdout, stderr bytes.Buffer
	h.serverCommand.Stdout = &stdout
	h.serverCommand.Stderr = &stderr

	if err := h.serverCommand.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	if err := browserpkg.WaitForServerReady(h.serverPort, 30*time.Second); err != nil {
		return fmt.Errorf("server not ready: %w\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

func (h *pdfHarness) setupBrowser() error {
	h.t.Helper()

	incognitoPage, err := browser.NewIncognitoPage()
	if err != nil {
		return fmt.Errorf("creating incognito page: %w", err)
	}
	h.incognitoPage = incognitoPage
	h.pageHelper = browserpkg.NewPageHelper(incognitoPage.Ctx)

	return nil
}

func (h *pdfHarness) navigateToPage() error {
	h.t.Helper()

	url := h.serverURL + h.spec.ComparisonURL
	h.t.Logf("navigating to %s", url)

	return h.pageHelper.Navigate(url)
}

func (h *pdfHarness) printBrowserPdf() ([]byte, error) {
	h.t.Helper()

	regularBase64 := base64.StdEncoding.EncodeToString(fonts.NotoSansRegularTTF)

	var injectFontScript string
	if h.spec.VariableFont {
		variableBase64 := base64.StdEncoding.EncodeToString(fonts.NotoSansVariableTTF)
		injectFontScript = fmt.Sprintf(`(function() {
		var style = document.createElement('style');
		style.textContent = '@font-face { font-family: "NotoSans"; src: url("data:font/ttf;base64,%s") format("truetype"); font-weight: 100 900; font-style: normal; font-display: block; } * { font-family: "NotoSans", sans-serif !important; }';
		document.head.appendChild(style);
	})()`, variableBase64)
	} else if h.spec.RegularOnly {
		injectFontScript = fmt.Sprintf(`(function() {
		var style = document.createElement('style');
		style.textContent = '@font-face { font-family: "NotoSans"; src: url("data:font/ttf;base64,%s") format("truetype"); font-weight: 400; font-style: normal; font-display: block; } * { font-family: "NotoSans", sans-serif !important; }';
		document.head.appendChild(style);
	})()`, regularBase64)
	} else {
		boldBase64 := base64.StdEncoding.EncodeToString(fonts.NotoSansBoldTTF)
		injectFontScript = fmt.Sprintf(`(function() {
		var style = document.createElement('style');
		style.textContent = '@font-face { font-family: "NotoSans"; src: url("data:font/ttf;base64,%s") format("truetype"); font-weight: 400; font-style: normal; font-display: block; } @font-face { font-family: "NotoSans"; src: url("data:font/ttf;base64,%s") format("truetype"); font-weight: 700; font-style: normal; font-display: block; } * { font-family: "NotoSans", sans-serif !important; }';
		document.head.appendChild(style);
	})()`, regularBase64, boldBase64)
	}

	var buffer []byte

	err := chromedp.Run(h.incognitoPage.Ctx,

		chromedp.Evaluate(injectFontScript, nil),

		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exception, evalErr := cdpruntime.Evaluate(`document.fonts.ready`).WithAwaitPromise(true).Do(ctx)
			if evalErr != nil {
				return fmt.Errorf("waiting for fonts: %w", evalErr)
			}
			if exception != nil {
				return fmt.Errorf("font readiness check exception: %s", exception.Text)
			}
			return nil
		}),

		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exception, evalErr := cdpruntime.Evaluate(
				`new Promise(resolve => requestAnimationFrame(() => setTimeout(resolve, 50)))`,
			).WithAwaitPromise(true).Do(ctx)
			if evalErr != nil {
				return fmt.Errorf("waiting for reflow: %w", evalErr)
			}
			if exception != nil {
				return fmt.Errorf("reflow wait exception: %s", exception.Text)
			}
			return nil
		}),

		chromedp.ActionFunc(func(ctx context.Context) error {
			printContext, printCancel := context.WithTimeoutCause(ctx, 60*time.Second,
				errors.New("Chrome PrintToPDF exceeded 60s timeout"))
			defer printCancel()
			var printError error
			buffer, _, printError = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				Do(printContext)
			return printError
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("printing browser PDF: %w", err)
	}

	return buffer, nil
}

func (h *pdfHarness) cleanup() {
	if h.pageHelper != nil {
		h.pageHelper.Close()
	}
	if h.incognitoPage != nil {
		_ = h.incognitoPage.CloseContext()
	}
	if h.serverCommand != nil && h.serverCommand.Process != nil {
		_ = h.serverCommand.Process.Kill()
		_, _ = h.serverCommand.Process.Wait()
	}
}

func copyDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, directory os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destinationPath := filepath.Join(destination, relativePath)
		if directory.IsDir() {
			return os.MkdirAll(destinationPath, 0755)
		}
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = sourceFile.Close() }()
		destinationFile, err := os.Create(destinationPath)
		if err != nil {
			return err
		}
		defer func() { _ = destinationFile.Close() }()
		_, err = io.Copy(destinationFile, sourceFile)
		return err
	})
}

func findPikoProjectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(directory, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			content, readError := os.ReadFile(goModPath)
			if readError == nil && strings.Contains(string(content), "module piko.sh/piko\n") {
				return directory, nil
			}
		}

		parentDirectory := filepath.Dir(directory)
		if parentDirectory == directory {
			return "", errors.New("could not find piko project root")
		}
		directory = parentDirectory
	}
}

func resetGlobalStateForTestIsolation() {
	shutdown.Reset()
	logger.ResetLogger()

	config_domain.ResetGlobalFlagCoordinator()
	config_domain.ResetGlobalResolverRegistry()
	config_domain.ResetDotEnvCache()
	config_domain.ResetSecretManager()

	ast_domain.ClearExpressionCache()
	ast_domain.ResetAllPools()

	compiler_domain.ClearIdentifierRegistry()
	compiler_domain.ClearBindingRegistry()
	compiler_domain.ClearLocRefRegistry()

	collection_domain.ResetStaticCollectionRegistry()
	collection_domain.ResetRuntimeProviderRegistry()
	collection_domain.ResetHybridRegistry()

	goastutil.ResetDynamicCaches()
	ast_domain.ClearSelectorCache()
	i18n_domain.ClearExpressionCache()
	generator_helpers.ClearModulePathCaches()
	render_domain.ClearHTMLLinksCache()
	caller.ResetFrameCache()
	logger_domain.ResetCallerCache()
}
