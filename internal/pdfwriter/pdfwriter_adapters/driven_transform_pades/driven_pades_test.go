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

package driven_transform_pades_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_pades"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	catalogNum  = 1
	pagesNum    = 2
	pageNum     = 3
	testKeyBits = 2048
)

type testCredentials struct {
	key       *rsa.PrivateKey
	certDER   []byte
	certChain [][]byte
}

func generateTestCredentials(t *testing.T) testCredentials {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, testKeyBits)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test Signer",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return testCredentials{
		key:       key,
		certDER:   certDER,
		certChain: [][]byte{certDER},
	}
}

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

func TestPadesTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_pades.New()
}

func TestPadesTransformer_Metadata(t *testing.T) {
	pt := driven_transform_pades.New()
	assert.Equal(t, "pades-sign", pt.Name())
	assert.Equal(t, pdfwriter_dto.TransformerSecurity, pt.Type())
	assert.Equal(t, 450, pt.Priority())
}

func TestPadesTransformer_SignProducesValidPDF(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	result, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Reason:           "Testing",
		Location:         "London",
	})
	require.NoError(t, err)

	assert.Greater(t, len(result), len(pdf))
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	assert.True(t, bytes.Contains(result, []byte("/Type /Sig")),
		"output should contain /Type /Sig")
	assert.True(t, bytes.Contains(result, []byte("/ByteRange")),
		"output should contain /ByteRange")
	assert.True(t, bytes.Contains(result, []byte("/SubFilter /ETSI.CAdES.detached")),
		"output should contain PAdES SubFilter")
	assert.True(t, bytes.Contains(result, []byte("/AcroForm")),
		"output should contain /AcroForm in the catalog")
}

func TestPadesTransformer_NilPrivateKey(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       nil,
		CertificateChain: creds.certChain,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key must not be nil")
}

func TestPadesTransformer_EmptyCertChain(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: nil,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate chain must not be empty")
}

func TestPadesTransformer_DefaultLevel(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	result, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "",
	})
	require.NoError(t, err)
	assert.Greater(t, len(result), len(pdf))
}

func TestPadesTransformer_BLTA_WithMockTSA(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	tsaServer := newMockTSA(t, creds)
	defer tsaServer.Close()

	ocspData := []byte{0x30, 0x03, 0x0a, 0x01, 0x00}
	crlData := []byte{0x30, 0x02, 0x30, 0x00}

	result, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-lta",
		TimestampURL:     tsaServer.URL,
		OCSPResponses:    [][]byte{ocspData},
		CRLs:             [][]byte{crlData},
	})
	require.NoError(t, err)
	assert.Greater(t, len(result), len(pdf))
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
	assert.True(t, bytes.Contains(result, []byte("/DSS")),
		"B-LTA output should contain /DSS dictionary")
	assert.True(t, bytes.Contains(result, []byte("/Type /DocTimeStamp")),
		"B-LTA output should contain document timestamp")
	assert.True(t, bytes.Contains(result, []byte("/SubFilter /ETSI.RFC3161")),
		"B-LTA output should contain RFC 3161 sub-filter")
}

func TestPadesTransformer_UnknownLevel(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "xyz",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown conformance level")
}

func TestPadesTransformer_PointerOptions(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	result, err := pt.Transform(context.Background(), pdf, &pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
	})
	require.NoError(t, err)
	assert.Greater(t, len(result), len(pdf))
}

func TestPadesTransformer_InvalidOptions(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)

	_, err := pt.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected PadesSignOptions")
}

func TestPadesTransformer_NilPointerOptions(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)

	_, err := pt.Transform(context.Background(), pdf, (*pdfwriter_dto.PadesSignOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestPadesTransformer_BT_RequiresTimestampURL(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-t",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TimestampURL is required")
}

func TestPadesTransformer_BLT_RequiresTimestampURL(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-lt",
		OCSPResponses:    [][]byte{{1, 2, 3}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TimestampURL is required")
}

func TestPadesTransformer_BLT_RequiresRevocationData(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-lt",
		TimestampURL:     "https://example.com/tsa",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OCSPResponses or CRLs required")
}

func TestPadesTransformer_BT_WithMockTSA(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	tsaServer := newMockTSA(t, creds)
	defer tsaServer.Close()

	result, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-t",
		TimestampURL:     tsaServer.URL,
	})
	require.NoError(t, err)
	assert.Greater(t, len(result), len(pdf))
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
	assert.True(t, bytes.Contains(result, []byte("/SubFilter /ETSI.CAdES.detached")))
}

func TestPadesTransformer_BLT_WithMockTSA(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	tsaServer := newMockTSA(t, creds)
	defer tsaServer.Close()

	ocspData := []byte{0x30, 0x03, 0x0a, 0x01, 0x00}
	crlData := []byte{0x30, 0x02, 0x30, 0x00}

	result, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-lt",
		TimestampURL:     tsaServer.URL,
		OCSPResponses:    [][]byte{ocspData},
		CRLs:             [][]byte{crlData},
	})
	require.NoError(t, err)
	assert.Greater(t, len(result), len(pdf))
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
	assert.True(t, bytes.Contains(result, []byte("/DSS")),
		"B-LT output should contain /DSS dictionary")
}

func TestPadesTransformer_BLT_OCSPOnly(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	tsaServer := newMockTSA(t, creds)
	defer tsaServer.Close()

	result, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-lt",
		TimestampURL:     tsaServer.URL,
		OCSPResponses:    [][]byte{{0x30, 0x03, 0x0a, 0x01, 0x00}},
	})
	require.NoError(t, err)
	assert.True(t, bytes.Contains(result, []byte("/DSS")))
	assert.True(t, bytes.Contains(result, []byte("/OCSPs")))
}

func TestPadesTransformer_BLT_CRLOnly(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	tsaServer := newMockTSA(t, creds)
	defer tsaServer.Close()

	result, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-lt",
		TimestampURL:     tsaServer.URL,
		CRLs:             [][]byte{{0x30, 0x02, 0x30, 0x00}},
	})
	require.NoError(t, err)
	assert.True(t, bytes.Contains(result, []byte("/DSS")))
	assert.True(t, bytes.Contains(result, []byte("/CRLs")))
}

func TestPadesTransformer_BT_TSAFailure(t *testing.T) {
	pt := driven_transform_pades.New()
	pdf := buildMinimalPDF(t)
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), pdf, pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
		Level:            "b-t",
		TimestampURL:     "http://127.0.0.1:1/nonexistent",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting timestamp")
}

func TestPadesTransformer_InvalidPDF(t *testing.T) {
	pt := driven_transform_pades.New()
	creds := generateTestCredentials(t)

	_, err := pt.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.PadesSignOptions{
		PrivateKey:       creds.key,
		CertificateChain: creds.certChain,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
}

type tsaReq struct {
	Version        int
	MessageImprint tsaMsgImprint
	CertReq        bool `asn1:"optional"`
}

type tsaMsgImprint struct {
	HashAlgorithm tsaAlgID
	HashedMessage []byte
}

type tsaAlgID struct {
	Algorithm asn1.ObjectIdentifier
}

func newMockTSA(t *testing.T, creds testCredentials) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		var req tsaReq
		if _, err := asn1.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		oidTSTInfo := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
		oidSHA256 := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
		oidSignedData := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
		oidData := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}

		type tstInfo struct {
			Version        int
			Policy         asn1.ObjectIdentifier
			MessageImprint tsaMsgImprint
			SerialNumber   *big.Int
			GenTime        time.Time `asn1:"generalized"`
		}

		tst := tstInfo{
			Version: 1,
			Policy:  asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1},
			MessageImprint: tsaMsgImprint{
				HashAlgorithm: tsaAlgID{Algorithm: oidSHA256},
				HashedMessage: req.MessageImprint.HashedMessage,
			},
			SerialNumber: big.NewInt(1),
			GenTime:      time.Now().UTC(),
		}

		tstDER, err := asn1.Marshal(tst)
		if err != nil {
			http.Error(w, "marshal error", http.StatusInternalServerError)
			return
		}

		tstDigest := sha256.Sum256(tstDER)
		sig, err := creds.key.Sign(rand.Reader, tstDigest[:], crypto.SHA256)
		if err != nil {
			http.Error(w, "sign error", http.StatusInternalServerError)
			return
		}

		cert, _ := x509.ParseCertificate(creds.certDER)

		type issuerSerial struct {
			Issuer asn1.RawValue
			Serial asn1.RawValue
		}
		type signerInfo struct {
			Version   int
			SID       issuerSerial
			DigestAlg pkix.AlgorithmIdentifier
			SigAlg    pkix.AlgorithmIdentifier
			Signature []byte
		}

		si := signerInfo{
			Version: 1,
			SID: issuerSerial{
				Issuer: asn1.RawValue{FullBytes: cert.RawIssuer},
				Serial: asn1.RawValue{FullBytes: mustMarshalBigInt(cert.SerialNumber)},
			},
			DigestAlg: pkix.AlgorithmIdentifier{Algorithm: oidSHA256},
			SigAlg:    pkix.AlgorithmIdentifier{Algorithm: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}},
			Signature: sig,
		}
		siDER, _ := asn1.Marshal(si)

		type encapContent struct {
			ContentType asn1.ObjectIdentifier
			Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
		}

		type signedData struct {
			Version      int
			DigestAlgs   []pkix.AlgorithmIdentifier `asn1:"set"`
			EncapContent encapContent
			Certs        []asn1.RawValue `asn1:"optional,set,tag:0"`
			SignerInfos  []asn1.RawValue `asn1:"set"`
		}

		sd := signedData{
			Version:    3,
			DigestAlgs: []pkix.AlgorithmIdentifier{{Algorithm: oidSHA256}},
			EncapContent: encapContent{
				ContentType: oidTSTInfo,
				Content: asn1.RawValue{
					Class:      asn1.ClassUniversal,
					Tag:        asn1.TagOctetString,
					IsCompound: false,
					Bytes:      tstDER,
				},
			},
			Certs:       []asn1.RawValue{{FullBytes: cert.Raw}},
			SignerInfos: []asn1.RawValue{{FullBytes: siDER}},
		}
		sdDER, _ := asn1.Marshal(sd)

		type contentInfo struct {
			ContentType asn1.ObjectIdentifier
			Content     asn1.RawValue `asn1:"explicit,tag:0"`
		}

		ciSD := contentInfo{
			ContentType: oidSignedData,
			Content: asn1.RawValue{
				Class:      asn1.ClassContextSpecific,
				Tag:        0,
				IsCompound: true,
				Bytes:      sdDER,
			},
		}
		tokenDER, _ := asn1.Marshal(ciSD)

		type statusInfo struct {
			Status int
		}
		type tsaResp struct {
			Status statusInfo
			Token  asn1.RawValue `asn1:"optional"`
		}

		_ = oidData
		resp := tsaResp{
			Status: statusInfo{Status: 0},
			Token:  asn1.RawValue{FullBytes: tokenDER},
		}
		respDER, _ := asn1.Marshal(resp)

		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(respDER)
	}))
}

func mustMarshalBigInt(n *big.Int) []byte {
	b, err := asn1.Marshal(n)
	if err != nil {
		panic(err)
	}
	return b
}
