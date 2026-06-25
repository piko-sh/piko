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

package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	t.Parallel()

	plat := platform{os: "linux", arch: "amd64", archiveExt: "tar.gz", binaryFile: "piko-lsp"}
	got := assetName("0.0.0-alpha.26", plat)
	want := "piko-lsp-0.0.0-alpha.26-linux-amd64.tar.gz"
	if got != want {
		t.Errorf("assetName() = %q, want %q", got, want)
	}
}

func TestCurrentPlatform(t *testing.T) {
	t.Parallel()

	plat, err := currentPlatform()
	if err != nil {
		t.Skipf("host platform unsupported: %v", err)
	}
	if plat.os == "" || plat.arch == "" || plat.binaryFile == "" || plat.archiveExt == "" {
		t.Errorf("incomplete platform: %+v", plat)
	}
}

func TestExtractFromTarGz(t *testing.T) {
	t.Parallel()

	content := []byte("#!/bin/sh\necho piko-lsp\n")
	archive := makeTarGz(t, "piko-lsp", content)

	got, err := extractFromTarGz(archive, "piko-lsp")
	if err != nil {
		t.Fatalf("extractFromTarGz() = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Error("extracted content does not match")
	}
}

func TestExtractFromTarGz_MissingBinary(t *testing.T) {
	t.Parallel()

	archive := makeTarGz(t, "something-else", []byte("x"))
	if _, err := extractFromTarGz(archive, "piko-lsp"); err == nil {
		t.Error("expected an error when the binary is absent")
	}
}

func TestEnsureInstalled_AlreadyOnPath(t *testing.T) {
	original := lookPath
	t.Cleanup(func() { lookPath = original })
	lookPath = func(string) (string, error) { return "/usr/local/bin/piko-lsp", nil }

	status := EnsureInstalled(context.Background(), "1.2.3")
	if !strings.Contains(status, "already on PATH") {
		t.Errorf("status = %q, want it to report the binary is already on PATH", status)
	}
}

func TestEnsureInstalled_DownloadsAndInstalls(t *testing.T) {
	plat, err := currentPlatform()
	if err != nil {
		t.Skipf("host platform unsupported: %v", err)
	}

	const version = "9.9.9"
	content := []byte("fake piko-lsp executable")

	var archive []byte
	if plat.archiveExt == "zip" {
		archive = makeZip(t, plat.binaryFile, content)
	} else {
		archive = makeTarGz(t, plat.binaryFile, content)
	}

	wantPath := "/v" + version + "/" + assetName(version, plat)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(archive)
	}))
	t.Cleanup(server.Close)

	originalURL, originalLook := releaseBaseURL, lookPath
	t.Cleanup(func() { releaseBaseURL, lookPath = originalURL, originalLook })
	releaseBaseURL = server.URL
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Join(home, ".local", "bin"))

	status := EnsureInstalled(context.Background(), version)

	installed := filepath.Join(home, ".local", "bin", plat.binaryFile)
	got, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("binary not installed (status %q): %v", status, err)
	}
	if !bytes.Equal(got, content) {
		t.Error("installed binary content does not match the archived binary")
	}
	if !strings.Contains(status, "installed piko-lsp "+version) {
		t.Errorf("status = %q, want it to confirm install of %s", status, version)
	}
}

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzw := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gzw)

	header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func makeZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)
	writer, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
