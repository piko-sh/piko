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

package driven_transform_encrypt_test

import (
	"bytes"
	"context"
	"errors"
	mathrand "math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_encrypt"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	catalogNum = 1
	pagesNum   = 2
	pageNum    = 3
)

func buildMinimalPDF(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
	}}))

	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(pageNum, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))

	w.SetObject(pageNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "Parent", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792),
		)},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func TestEncryptTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_encrypt.New()
}

func TestEncryptTransformer_Metadata(t *testing.T) {
	enc := driven_transform_encrypt.New()
	assert.Equal(t, "pdf-encrypt", enc.Name())
	assert.Equal(t, pdfwriter_dto.TransformerSecurity, enc.Type())
	assert.Equal(t, 400, enc.Priority())
}

func TestEncryptTransformer_EncryptProducesValidPDF(t *testing.T) {
	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	result, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner123",
		UserPassword:  "user456",
		Permissions:   0xFFFFF0C4,
	})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")), "output should start with %%PDF-")

	assert.False(t, bytes.Equal(pdf, result), "encrypted output should differ from input")

	assert.True(t, bytes.Contains(result, []byte("/Encrypt")), "output should contain /Encrypt")
}

func TestEncryptTransformer_EmptyOwnerPassword(t *testing.T) {
	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	_, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "",
		UserPassword:  "user",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner password must not be empty")
}

func TestEncryptTransformer_UnsupportedAlgorithm(t *testing.T) {
	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	_, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "rc4-128",
		OwnerPassword: "owner",
		UserPassword:  "user",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet supported")
	assert.Contains(t, err.Error(), "rc4-128")
}

func TestEncryptTransformer_DefaultsToAES256(t *testing.T) {
	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	result, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "",
		OwnerPassword: "owner123",
		UserPassword:  "user456",
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
	assert.True(t, bytes.Contains(result, []byte("/Encrypt")))
}

func TestEncryptTransformer_PointerOptions(t *testing.T) {
	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	result, err := enc.Transform(context.Background(), pdf, &pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner123",
		UserPassword:  "user456",
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
	assert.True(t, bytes.Contains(result, []byte("/Encrypt")))
}

func TestEncryptTransformer_InvalidOptions(t *testing.T) {
	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	_, err := enc.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected EncryptionOptions")
}

func TestEncryptTransformer_NilPointerOptions(t *testing.T) {
	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	_, err := enc.Transform(context.Background(), pdf, (*pdfwriter_dto.EncryptionOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestEncryptTransformer_InvalidPDF(t *testing.T) {
	enc := driven_transform_encrypt.New()

	_, err := enc.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner",
		UserPassword:  "user",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
}

func TestEncryptTransformer_UserPasswordValidation(t *testing.T) {

	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	userPass := "hello"
	ownerPass := "owner-secret"

	result, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: ownerPass,
		UserPassword:  userPass,
		Permissions:   0xFFFFF0C4,
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	trailer := doc.Trailer()
	encRef := trailer.GetRef("Encrypt")
	require.NotEqual(t, 0, encRef.Number, "expected /Encrypt reference in trailer")

	encObj, err := doc.GetObject(encRef.Number)
	require.NoError(t, err)
	require.Equal(t, pdfparse.ObjectDictionary, encObj.Type, "expected /Encrypt to be a dict")

	encDict, ok := encObj.Value.(pdfparse.Dict)
	require.True(t, ok)

	uObj := encDict.Get("U")
	require.NotEqual(t, pdfparse.ObjectNull, uObj.Type, "expected /U in encrypt dict")

	uRaw, ok := uObj.Value.(string)
	require.True(t, ok, "expected /U value to be string, got %T", uObj.Value)
	uBytes := []byte(uRaw)

	t.Logf("/U length: %d (expected 48)", len(uBytes))
	require.Equal(t, 48, len(uBytes), "/U must be exactly 48 bytes")

	if _, lookErr := exec.LookPath("qpdf"); lookErr == nil {
		tmpDir := t.TempDir()
		unencryptedPath := filepath.Join(tmpDir, "plain.pdf")
		ourPath := filepath.Join(tmpDir, "ours.pdf")
		qpdfPath := filepath.Join(tmpDir, "qpdf_ref.pdf")

		require.NoError(t, os.WriteFile(unencryptedPath, pdf, 0o644))
		require.NoError(t, os.WriteFile(ourPath, result, 0o644))

		refCmd := exec.Command("qpdf", "--encrypt", userPass, ownerPass, "256", "--", unencryptedPath, qpdfPath)
		refOut, refErr := refCmd.CombinedOutput()
		require.NoError(t, refErr, "qpdf encrypt should succeed: %s", string(refOut))

		checkRef := exec.Command("qpdf", "--password="+userPass, "--check", qpdfPath)
		checkRefOut, _ := checkRef.CombinedOutput()
		t.Logf("qpdf ref check: %s", string(checkRefOut))
		require.NotContains(t, string(checkRefOut), "invalid password", "qpdf reference should accept password")

		checkOurs := exec.Command("qpdf", "--password="+userPass, "--check", ourPath)
		checkOursOut, _ := checkOurs.CombinedOutput()
		t.Logf("our check: %s", string(checkOursOut))

		for name, path := range map[string]string{"qpdf_ref": qpdfPath, "ours": ourPath} {
			dumpCmd := exec.Command("qpdf", "--show-encryption", "--password="+userPass, path)
			dumpOut, _ := dumpCmd.CombinedOutput()
			t.Logf("%s encryption:\n%s", name, string(dumpOut))
		}

		rawPDF, _ := os.ReadFile(ourPath)
		if encIdx := bytes.Index(rawPDF, []byte("/Encrypt")); encIdx >= 0 {
			start := encIdx
			end := min(encIdx+500, len(rawPDF))
			t.Logf("raw PDF around /Encrypt:\n%s", string(rawPDF[start:end]))
		}

		assert.NotContains(t, string(checkOursOut), "invalid password", "our encrypted PDF should be accepted by qpdf")
	}
}

func TestEncryptTransformer_QpdfRoundtrip(t *testing.T) {
	qpdfPath, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed, skipping roundtrip test")
	}
	_ = qpdfPath

	enc := driven_transform_encrypt.New()
	pdf := buildMinimalPDF(t)

	userPass := "hello"
	ownerPass := "owner-secret"

	result, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: ownerPass,
		UserPassword:  userPass,
		Permissions:   0xFFFFF0C4,
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	encryptedPath := filepath.Join(tmpDir, "encrypted.pdf")
	decryptedPath := filepath.Join(tmpDir, "decrypted.pdf")
	require.NoError(t, os.WriteFile(encryptedPath, result, 0o644))

	cmd := exec.Command("qpdf", "--password="+userPass, "--decrypt", encryptedPath, decryptedPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qpdf decrypt with user password failed: %v\noutput: %s", err, string(output))
	}

	decrypted, err := os.ReadFile(decryptedPath)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(decrypted, []byte("%PDF-")), "decrypted output should be a valid PDF")
}

func TestEncryptTransformer_WithRandomSourceProducesDeterministicOutput(t *testing.T) {
	pdf := buildMinimalPDF(t)
	options := pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner-secret",
		UserPassword:  "user-secret",
		Permissions:   0xFFFFF0C4,
	}

	seed := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	first := driven_transform_encrypt.New(driven_transform_encrypt.WithRandomSource(mathrand.NewChaCha8(seed)))
	resultA, err := first.Transform(context.Background(), pdf, options)
	require.NoError(t, err)

	second := driven_transform_encrypt.New(driven_transform_encrypt.WithRandomSource(mathrand.NewChaCha8(seed)))
	resultB, err := second.Transform(context.Background(), pdf, options)
	require.NoError(t, err)

	assert.True(t, bytes.Equal(resultA, resultB),
		"deterministic random source should produce byte-identical output across calls")
}

func TestEncryptTransformer_WithRandomSourceDiffersFromDefault(t *testing.T) {
	pdf := buildMinimalPDF(t)
	options := pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner-secret",
		UserPassword:  "user-secret",
		Permissions:   0xFFFFF0C4,
	}

	seed := [32]byte{99}
	seeded := driven_transform_encrypt.New(driven_transform_encrypt.WithRandomSource(mathrand.NewChaCha8(seed)))
	defaulted := driven_transform_encrypt.New()

	resultSeeded, err := seeded.Transform(context.Background(), pdf, options)
	require.NoError(t, err)
	resultDefault, err := defaulted.Transform(context.Background(), pdf, options)
	require.NoError(t, err)

	assert.False(t, bytes.Equal(resultSeeded, resultDefault),
		"seeded source should produce different output from crypto/rand-backed default")
}

func TestEncryptTransformer_RejectsExcessivelyNestedObjects(t *testing.T) {
	w := pdfparse.NewWriter()

	deeplyNested := pdfparse.Arr(pdfparse.Int(0))
	for range 300 {
		deeplyNested = pdfparse.Arr(deeplyNested)
	}

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "Hostile", Value: deeplyNested},
	}}))
	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr()},
		{Key: "Count", Value: pdfparse.Int(0)},
	}}))
	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	pdf, err := w.Write()
	require.NoError(t, err)

	enc := driven_transform_encrypt.New()
	_, err = enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner",
		UserPassword:  "user",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, driven_transform_encrypt.ErrObjectNestingTooDeep),
		"deep nesting must surface ErrObjectNestingTooDeep, got: %v", err)
}

func TestEncryptTransformer_WithMaxObjectNestingDepthOverride(t *testing.T) {
	w := pdfparse.NewWriter()

	nested := pdfparse.Arr(pdfparse.Int(0))
	for range 50 {
		nested = pdfparse.Arr(nested)
	}

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "Nested", Value: nested},
	}}))
	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr()},
		{Key: "Count", Value: pdfparse.Int(0)},
	}}))
	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	pdf, err := w.Write()
	require.NoError(t, err)

	enc := driven_transform_encrypt.New(driven_transform_encrypt.WithMaxObjectNestingDepth(10))
	_, err = enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner",
		UserPassword:  "user",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, driven_transform_encrypt.ErrObjectNestingTooDeep),
		"depth 50 must exceed configured limit 10, got: %v", err)
}

func TestEncryptTransformer_EncryptsStringStreamAndNestedDictValues(t *testing.T) {
	const contentNum = 4

	w := pdfparse.NewWriter()

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "Title", Value: pdfparse.Str("Confidential Memo")},
		{Key: "Metadata", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Author", Value: pdfparse.Str("Alice")},
			{Key: "Tags", Value: pdfparse.Arr(pdfparse.Str("draft"), pdfparse.Str("internal"))},
		}})},
	}}))
	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(pageNum, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))
	w.SetObject(pageNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "Parent", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "Contents", Value: pdfparse.RefObj(contentNum, 0)},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792),
		)},
	}}))
	streamPayload := []byte("BT /F1 12 Tf 100 700 Td (hello world) Tj ET")
	w.SetObject(contentNum, pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Length", Value: pdfparse.Int(int64(len(streamPayload)))},
		}},
		streamPayload,
	))
	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	pdf, err := w.Write()
	require.NoError(t, err)

	enc := driven_transform_encrypt.New()
	result, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
	assert.True(t, bytes.Contains(result, []byte("/Encrypt")))
	assert.False(t, bytes.Contains(result, streamPayload),
		"plaintext stream payload must not survive encryption")
	assert.False(t, bytes.Contains(result, []byte("Confidential Memo")),
		"plaintext string values must not survive encryption")
	assert.False(t, bytes.Contains(result, []byte("Alice")),
		"plaintext nested-dict values must not survive encryption")
}

func TestEncryptTransformer_WithMaxObjectNestingDepthIgnoresNonPositive(t *testing.T) {
	enc := driven_transform_encrypt.New(
		driven_transform_encrypt.WithMaxObjectNestingDepth(0),
		driven_transform_encrypt.WithMaxObjectNestingDepth(-5),
	)
	pdf := buildMinimalPDF(t)

	result, err := enc.Transform(context.Background(), pdf, pdfwriter_dto.EncryptionOptions{
		Algorithm:     "aes-256",
		OwnerPassword: "owner",
		UserPassword:  "user",
	})
	require.NoError(t, err, "non-positive overrides must be ignored, leaving the default in force")
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
}
