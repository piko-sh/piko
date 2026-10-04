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

package quickpackages

import (
	"bufio"
	"bytes"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/gcexportdata"
)

func buildTestArchive(memberName, body string) string {
	header := []byte(strings.Repeat(" ", archiveHeaderSize))
	copy(header[0:16], memberName)
	copy(header[48:58], strconv.Itoa(len(body)))
	return archiveSignature + string(header) + body
}

func TestExportDataReaderReadsCompilerArchive(t *testing.T) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain not available: %v", err)
	}

	output, err := exec.Command(goBinary, "list", "-export", "-f", "{{.Export}}", "strings").Output()
	require.NoError(t, err)
	exportFile := strings.TrimSpace(string(output))
	require.NotEmpty(t, exportFile, "go list -export returned no export file for strings")

	file, err := os.Open(exportFile)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	reader, err := exportDataReader(bufio.NewReader(file))
	require.NoError(t, err)

	pkg, err := gcexportdata.Read(reader, token.NewFileSet(), make(map[string]*types.Package), "strings")
	require.NoError(t, err)
	assert.Equal(t, "strings", pkg.Name())
	assert.NotNil(t, pkg.Scope().Lookup("Builder"), "the decoded package must expose strings.Builder")
}

func TestExportDataReaderReturnsOnlyExportData(t *testing.T) {
	body := "go object linux amd64 go1.27\nbuild id \"x\"\n" + exportDataStart + "udata" + exportDataEnd
	archive := buildTestArchive(packageDefinitionName, body) + "trailing member bytes"

	reader, err := exportDataReader(bufio.NewReader(strings.NewReader(archive)))
	require.NoError(t, err)

	var data bytes.Buffer
	_, err = data.ReadFrom(reader)
	require.NoError(t, err)
	assert.Equal(t, "udata", data.String(), "markers and later archive members must not be returned")
}

func TestExportDataReaderRejectsMalformedInput(t *testing.T) {
	testCases := []struct {
		wantErr error
		name    string
		input   string
	}{
		{name: "not an archive", input: "package strings\n", wantErr: errNotGoArchive},
		{name: "truncated member header", input: archiveSignature + packageDefinitionName, wantErr: errNoPackageDefinition},
		{name: "first member is not the package definition", input: buildTestArchive("_go_.o", "x"), wantErr: errNoPackageDefinition},
		{
			name:    "text export section",
			input:   buildTestArchive(packageDefinitionName, "go object linux amd64 go1.27\n$$\nold\n$$\n"),
			wantErr: errNoBinaryExportData,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := exportDataReader(bufio.NewReader(strings.NewReader(testCase.input)))
			require.ErrorIs(t, err, testCase.wantErr)
		})
	}
}
