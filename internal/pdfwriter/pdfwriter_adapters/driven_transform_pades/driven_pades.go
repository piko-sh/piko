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

package driven_transform_pades

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the PAdES transformer.
	defaultPriority = 450

	// transformerName is the identifier used by the registry and validation.
	transformerName = "pades-sign"

	// levelBB is the PAdES B-B (Basic) conformance level.
	levelBB = "b-b"

	// levelBT is the PAdES B-T (with Timestamp) conformance level.
	levelBT = "b-t"

	// levelBLT is the PAdES B-LT (Long-Term validation) conformance level.
	levelBLT = "b-lt"

	// levelBLTA is the PAdES B-LTA (Long-Term Archival) conformance level.
	levelBLTA = "b-lta"

	// signatureContentsSizeBB is the number of bytes reserved for the
	// hex-encoded CMS signature in the /Contents placeholder for B-B.
	// 8192 bytes of DER becomes 16384 hex characters, sufficient for
	// most certificate chains.
	signatureContentsSizeBB = 8192

	// signatureContentsSizeBT is the larger reservation for B-T and
	// above. The embedded timestamp token adds 3-5 KB of DER.
	signatureContentsSizeBT = 16384

	// contentsPlaceholderTag is the marker used to locate the /Contents
	// hex string placeholder in the serialised PDF so that the byte
	// range can be computed and the signature inserted.
	contentsPlaceholderTag = "/Contents <"

	// annotsKey is the PDF dictionary key for page annotations.
	annotsKey = "Annots"

	// annotFlagHiddenPrint is the annotation flags bitmask for an
	// invisible signature widget (Hidden + Print = 128 + 4 = 132).
	annotFlagHiddenPrint = 132

	// sigFlagsValue is the /SigFlags bitmask for signature fields:
	// SignaturesExist (1) | AppendOnly (2) = 3.
	sigFlagsValue = 3

	// byteRangeSize is the number of elements in a /ByteRange array.
	byteRangeSize = 4

	// asn1TagSequence is the ASN.1 tag byte for SEQUENCE.
	asn1TagSequence = 0x30

	// asn1TagSet is the ASN.1 tag byte for SET.
	asn1TagSet = 0x31

	// asn1TagImplicitConstructed0 is the ASN.1 tag byte for IMPLICIT
	// [0] CONSTRUCTED context-specific.
	asn1TagImplicitConstructed0 = 0xa0

	// asn1TagImplicitConstructed1 is the ASN.1 tag byte for IMPLICIT
	// [1] CONSTRUCTED context-specific (unsigned attributes).
	asn1TagImplicitConstructed1 = 0xa1
)

var (
	// oidData is the ASN.1 OID for PKCS#7 id-data content type.
	oidData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}

	// oidSignedData is the ASN.1 OID for PKCS#7 id-signedData content type.
	oidSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}

	// oidSHA256 is the ASN.1 OID for the SHA-256 hash algorithm.
	oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}

	// oidContentType is the ASN.1 OID for the CMS content-type signed attribute.
	oidContentType = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}

	// oidMessageDigest is the ASN.1 OID for the CMS message-digest signed attribute.
	oidMessageDigest = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}

	// oidSigningTime is the ASN.1 OID for the CMS signing-time signed attribute.
	oidSigningTime = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}

	// oidECDSAWithSHA256 is the OID for ecdsa-with-SHA256.
	oidECDSAWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}

	// oidSHA256WithRSA is the OID for sha256WithRSAEncryption.
	oidSHA256WithRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}

	// oidTimeStampToken is the id-smime-aa-timeStampToken OID used to
	// embed an RFC 3161 timestamp token as an unsigned attribute in CMS
	// SignerInfo (RFC 5816).
	oidTimeStampToken = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14}
)

var (
	// errSignatureDictNotFound is returned when the signature dictionary
	// object cannot be located in the serialised PDF.
	errSignatureDictNotFound = errors.New("signature dictionary object not found")

	// errContentsPlaceholderNotFound is returned when the /Contents hex
	// placeholder cannot be found in the signature dictionary.
	errContentsPlaceholderNotFound = errors.New("/Contents placeholder not found in signature dictionary")

	// errContentsHexNotTerminated is returned when the /Contents hex string
	// is missing its closing angle bracket.
	errContentsHexNotTerminated = errors.New("/Contents hex string not terminated")
)

// PadesTransformer adds a PAdES CMS digital signature to a PDF document.
type PadesTransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*PadesTransformer)(nil)

// New creates a new PAdES signature transformer with default name and
// priority.
//
// Returns *PadesTransformer which is the initialised transformer.
func New() *PadesTransformer {
	return &PadesTransformer{
		name:     transformerName,
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *PadesTransformer) Name() string { return t.name }

// Type returns TransformerSecurity.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a
// security transformer.
func (*PadesTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerSecurity
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *PadesTransformer) Priority() int { return t.priority }

// Transform applies a PAdES digital signature to the PDF. Options must be
// PadesSignOptions or *PadesSignOptions.
//
// The implementation follows the PAdES B-B profile: it adds a signature
// dictionary with /SubFilter /ETSI.CAdES.detached, reserves space for
// the CMS signature, computes the SHA-256 hash over the byte ranges
// (everything except the /Contents hex string value), constructs a CMS
// SignedData structure, and embeds the DER-encoded signature.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be PadesSignOptions or *PadesSignOptions.
//
// Returns []byte which is the signed PDF.
// Returns error when validation fails, the PDF cannot be parsed, or
// signing fails.
func (*PadesTransformer) Transform(ctx context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}

	if err := validateOptions(&opts); err != nil {
		return nil, err
	}

	applyDefaults(&opts)

	switch opts.Level {
	case levelBB, levelBT, levelBLT:
		return signPDF(ctx, pdf, &opts)
	case levelBLTA:
		return signPDFWithDocTimestamp(ctx, pdf, &opts)
	default:
		return nil, fmt.Errorf("pades-sign: unknown conformance level %q", opts.Level)
	}
}

// contentsSize returns the appropriate /Contents reservation size for
// the given conformance level.
//
// Takes level (string) which specifies the PAdES conformance level.
//
// Returns int which is the reservation size in bytes.
func contentsSize(level string) int {
	switch level {
	case levelBT, levelBLT, levelBLTA:
		return signatureContentsSizeBT
	default:
		return signatureContentsSizeBB
	}
}

// signPDFWithDocTimestamp performs the B-LTA signing operation.
//
// It first signs the PDF at B-LT level (CMS signature with timestamp
// and DSS), then appends a document timestamp as an incremental
// update. The document timestamp covers the entire signed document
// including the DSS data, ensuring long-term archival validity.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pdf ([]byte) which is the input PDF document.
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds the signing
// configuration.
//
// Returns []byte which is the signed PDF with document timestamp.
// Returns error when signing or timestamping fails.
func signPDFWithDocTimestamp(ctx context.Context, pdf []byte, opts *pdfwriter_dto.PadesSignOptions) ([]byte, error) {
	signedPDF, err := signPDF(ctx, pdf, opts)
	if err != nil {
		return nil, err
	}

	result, err := appendDocumentTimestamp(ctx, signedPDF, opts.TimestampURL)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// signPDF performs the PAdES signing operation: parsing, adding
// signature structure (and DSS for B-LT), serialising, hashing,
// signing, optionally timestamping, and embedding the CMS signature.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pdf ([]byte) which is the input PDF document.
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds the signing
// configuration.
//
// Returns []byte which is the signed PDF.
// Returns error when any step of the signing pipeline fails.
func signPDF(ctx context.Context, pdf []byte, opts *pdfwriter_dto.PadesSignOptions) ([]byte, error) {
	certs, err := parseCertificateChain(opts.CertificateChain)
	if err != nil {
		return nil, fmt.Errorf("pades-sign: parsing certificate chain: %w", err)
	}

	pdfBytes, sigDictObjNum, err := preparePDFStructure(pdf, opts)
	if err != nil {
		return nil, err
	}

	byteRange, contentsStart, contentsEnd, err := locateContentsPlaceholder(pdfBytes, sigDictObjNum)
	if err != nil {
		return nil, fmt.Errorf("pades-sign: locating signature placeholder: %w", err)
	}

	pdfBytes = patchByteRange(pdfBytes, byteRange)
	digest := computeByteRangeDigest(pdfBytes, byteRange)

	cmsData, err := buildSignatureWithTimestamp(ctx, digest, opts, certs)
	if err != nil {
		return nil, err
	}

	return embedSignature(pdfBytes, cmsData, contentsStart, contentsEnd)
}

// preparePDFStructure parses the PDF, adds the signature dictionary
// and optionally a DSS dictionary, then serialises.
//
// Takes pdf ([]byte) which is the input PDF document.
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds the signing
// configuration.
//
// Returns []byte which is the serialised PDF with signature structure.
// Returns int which is the signature dictionary object number.
// Returns error when parsing, writing, or adding structures fails.
func preparePDFStructure(pdf []byte, opts *pdfwriter_dto.PadesSignOptions) ([]byte, int, error) {
	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: creating writer: %w", err)
	}

	sigDictObjNum, err := addSignatureStructure(writer, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: adding signature structure: %w", err)
	}

	if opts.Level == levelBLT || opts.Level == levelBLTA {
		if err := addDSSDictionary(writer, opts); err != nil {
			return nil, 0, fmt.Errorf("pades-sign: adding DSS dictionary: %w", err)
		}
	}

	pdfBytes, err := writer.Write()
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: writing PDF: %w", err)
	}

	return pdfBytes, sigDictObjNum, nil
}

// buildSignatureWithTimestamp creates the CMS SignedData, optionally
// requesting a timestamp from the TSA and rebuilding the CMS with the
// timestamp token as an unsigned attribute.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes digest ([]byte) which is the SHA-256 hash of the byte ranges.
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds the signing
// configuration including the TSA URL.
// Takes certs ([]*x509.Certificate) which is the parsed certificate chain.
//
// Returns []byte which is the DER-encoded CMS ContentInfo.
// Returns error when signing or timestamping fails.
func buildSignatureWithTimestamp(
	ctx context.Context,
	digest []byte,
	opts *pdfwriter_dto.PadesSignOptions,
	certs []*x509.Certificate,
) ([]byte, error) {
	cmsData, err := buildCMSSignedData(digest, opts.PrivateKey, certs, time.Now().UTC(), nil)
	if err != nil {
		return nil, fmt.Errorf("pades-sign: building CMS signature: %w", err)
	}

	needsTimestamp := opts.Level == levelBT || opts.Level == levelBLT || opts.Level == levelBLTA
	if !needsTimestamp {
		return cmsData, nil
	}

	timestampToken, err := requestTimestamp(ctx, opts.TimestampURL, extractSignatureValue(cmsData))
	if err != nil {
		return nil, fmt.Errorf("pades-sign: requesting timestamp: %w", err)
	}

	cmsData, err = buildCMSSignedData(digest, opts.PrivateKey, certs, time.Now().UTC(), timestampToken)
	if err != nil {
		return nil, fmt.Errorf("pades-sign: rebuilding CMS with timestamp: %w", err)
	}

	return cmsData, nil
}

// extractSignatureValue extracts the raw signature bytes from a
// DER-encoded CMS ContentInfo.
//
// It navigates the ASN.1 structure: ContentInfo -> SignedData ->
// SignerInfos -> first SignerInfo -> signature field.
//
// Takes cmsData ([]byte) which is the DER-encoded CMS ContentInfo.
//
// Returns []byte which holds the raw signature value, or the original
// cmsData if the structure cannot be parsed.
func extractSignatureValue(cmsData []byte) []byte {
	var ci struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}
	if _, err := asn1.Unmarshal(cmsData, &ci); err != nil {
		return cmsData
	}

	var sd struct { //nolint:govet // ASN.1 field order
		Version          int
		DigestAlgorithms asn1.RawValue
		EncapContentInfo asn1.RawValue
		Rest             asn1.RawValue `asn1:"optional"`
	}
	rest, err := asn1.Unmarshal(ci.Content.Bytes, &sd)
	if err != nil {
		return cmsData
	}

	data := rest
	if len(sd.Rest.FullBytes) > 0 {
		data = sd.Rest.FullBytes
	}

	for len(data) > 0 {
		var raw asn1.RawValue
		remaining, err := asn1.Unmarshal(data, &raw)
		if err != nil {
			break
		}

		if raw.Tag == asn1.TagSet && raw.Class == asn1.ClassUniversal {
			var siRaw asn1.RawValue
			if _, err := asn1.Unmarshal(raw.Bytes, &siRaw); err == nil {
				return extractLastOctetString(siRaw.Bytes)
			}
		}
		data = remaining
	}

	return cmsData
}

// extractLastOctetString walks ASN.1 elements and returns the last
// OCTET STRING value found.
//
// Takes data ([]byte) which is the raw ASN.1 bytes to search.
//
// Returns []byte which holds the last OCTET STRING value found.
func extractLastOctetString(data []byte) []byte {
	var last []byte
	for len(data) > 0 {
		var raw asn1.RawValue
		remaining, err := asn1.Unmarshal(data, &raw)
		if err != nil {
			break
		}
		if raw.Tag == asn1.TagOctetString && raw.Class == asn1.ClassUniversal {
			last = raw.Bytes
		}
		data = remaining
	}
	return last
}

// castOptions extracts PadesSignOptions from the generic options.
//
// Takes options (any) which must be PadesSignOptions or
// *PadesSignOptions.
//
// Returns pdfwriter_dto.PadesSignOptions which is the extracted value.
// Returns error when the type assertion fails or the pointer is nil.
func castOptions(options any) (pdfwriter_dto.PadesSignOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.PadesSignOptions:
		return v, nil
	case *pdfwriter_dto.PadesSignOptions:
		if v == nil {
			return pdfwriter_dto.PadesSignOptions{}, errors.New("pades-sign: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.PadesSignOptions{}, fmt.Errorf("pades-sign: expected PadesSignOptions, got %T", options)
	}
}

// validateOptions checks that required fields are present for the
// requested conformance level.
//
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds the options
// to validate.
//
// Returns error when required fields are missing for the requested level.
func validateOptions(opts *pdfwriter_dto.PadesSignOptions) error {
	if opts.PrivateKey == nil {
		return errors.New("pades-sign: private key must not be nil")
	}
	if len(opts.CertificateChain) == 0 {
		return errors.New("pades-sign: certificate chain must not be empty")
	}

	level := opts.Level
	if level == "" {
		level = levelBB
	}

	switch level {
	case levelBT:
		if opts.TimestampURL == "" {
			return errors.New("pades-sign: TimestampURL is required for B-T level")
		}
	case levelBLT, levelBLTA:
		if opts.TimestampURL == "" {
			return errors.New("pades-sign: TimestampURL is required for " + level + " level")
		}
		if len(opts.OCSPResponses) == 0 && len(opts.CRLs) == 0 {
			return errors.New("pades-sign: OCSPResponses or CRLs required for " + level + " level")
		}
	}

	return nil
}

// applyDefaults fills zero-value fields with sensible defaults.
//
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds the options
// to populate with defaults.
func applyDefaults(opts *pdfwriter_dto.PadesSignOptions) {
	if opts.Level == "" {
		opts.Level = levelBB
	}
}

// parseCertificateChain decodes DER-encoded certificates into x509
// certificate objects.
//
// Takes chain ([][]byte) which holds the DER-encoded certificates.
//
// Returns []*x509.Certificate which is the parsed certificate chain.
// Returns error when any certificate cannot be parsed.
func parseCertificateChain(chain [][]byte) ([]*x509.Certificate, error) {
	certs := make([]*x509.Certificate, 0, len(chain))
	for i, der := range chain {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("certificate %d: %w", i, err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// addSignatureStructure creates the signature dictionary, widget
// annotation, and AcroForm entry in the PDF writer.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds the signing
// configuration.
//
// Returns int which is the signature dictionary object number.
// Returns error when the annotation or AcroForm cannot be added.
func addSignatureStructure(writer *pdfparse.Writer, opts *pdfwriter_dto.PadesSignOptions) (int, error) {
	placeholderHex := string(bytes.Repeat([]byte("0"), contentsSize(opts.Level)*2))

	sigDict := buildSigDict(placeholderHex, opts)
	sigDictObjNum := writer.AddObject(pdfparse.DictObj(sigDict))

	widgetObjNum := addWidgetAnnotation(writer, sigDictObjNum, "Signature1")

	if err := addAnnotToFirstPage(writer, widgetObjNum); err != nil {
		return 0, fmt.Errorf("adding annotation to first page: %w", err)
	}

	if err := ensureAcroForm(writer, widgetObjNum); err != nil {
		return 0, fmt.Errorf("ensuring AcroForm: %w", err)
	}

	return sigDictObjNum, nil
}

// buildBaseSigDict creates the common fields shared by all signature
// dictionaries.
//
// Takes typeName (string) which specifies the /Type value.
// Takes subFilter (string) which specifies the /SubFilter value.
// Takes placeholderHex (string) which is the hex placeholder for
// /Contents.
//
// Returns pdfparse.Dict which holds the base signature dictionary.
func buildBaseSigDict(typeName, subFilter, placeholderHex string) pdfparse.Dict {
	return pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name(typeName)},
		{Key: "Filter", Value: pdfparse.Name("Adobe.PPKLite")},
		{Key: "SubFilter", Value: pdfparse.Name(subFilter)},
		{Key: "ByteRange", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0),
			pdfparse.Int(0), pdfparse.Int(0),
		)},
		{Key: "Contents", Value: pdfparse.HexStr(placeholderHex)},
		{Key: "M", Value: pdfparse.Str(time.Now().UTC().Format("D:20060102150405Z"))},
	}}
}

// buildSigDict creates the PDF signature dictionary with the appropriate
// PAdES fields.
//
// Takes placeholderHex (string) which is the hex placeholder for
// /Contents.
// Takes opts (*pdfwriter_dto.PadesSignOptions) which holds optional
// reason, location, and contact info.
//
// Returns pdfparse.Dict which holds the complete signature dictionary.
func buildSigDict(placeholderHex string, opts *pdfwriter_dto.PadesSignOptions) pdfparse.Dict {
	sigDict := buildBaseSigDict("Sig", "ETSI.CAdES.detached", placeholderHex)

	if opts.Reason != "" {
		sigDict.Set("Reason", pdfparse.Str(opts.Reason))
	}
	if opts.Location != "" {
		sigDict.Set("Location", pdfparse.Str(opts.Location))
	}
	if opts.ContactInfo != "" {
		sigDict.Set("ContactInfo", pdfparse.Str(opts.ContactInfo))
	}

	return sigDict
}

// addWidgetAnnotation creates an invisible widget annotation for a
// signature field and adds it to the writer.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes sigDictObjNum (int) which is the object number of the signature
// dictionary.
// Takes fieldName (string) which distinguishes the primary signature
// from the document timestamp.
//
// Returns int which is the widget annotation object number.
func addWidgetAnnotation(writer *pdfparse.Writer, sigDictObjNum int, fieldName string) int {
	widgetDict := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Annot")},
		{Key: "Subtype", Value: pdfparse.Name("Widget")},
		{Key: "FT", Value: pdfparse.Name("Sig")},
		{Key: "T", Value: pdfparse.Str(fieldName)},
		{Key: "V", Value: pdfparse.RefObj(sigDictObjNum, 0)},
		{Key: "Rect", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(0),
		)},
		{Key: "F", Value: pdfparse.Int(annotFlagHiddenPrint)},
	}}
	return writer.AddObject(pdfparse.DictObj(widgetDict))
}

// addAnnotToFirstPage appends a widget annotation reference to the first
// page's /Annots array.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes widgetObjNum (int) which is the widget annotation object number.
//
// Returns error when the PDF structure cannot be traversed to locate
// the first page.
func addAnnotToFirstPage(writer *pdfparse.Writer, widgetObjNum int) error {
	trailer := writer.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return errors.New("pades-sign: trailer has no /Root reference")
	}

	catalogObj := writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return fmt.Errorf("pades-sign: catalog object %d is not a dictionary", rootRef.Number)
	}

	pagesRef := catalogDict.GetRef("Pages")
	if pagesRef.Number == 0 {
		return errors.New("pades-sign: catalog has no /Pages reference")
	}

	pagesObj := writer.GetObject(pagesRef.Number)
	pagesDict, ok := pagesObj.Value.(pdfparse.Dict)
	if !ok {
		return fmt.Errorf("pades-sign: pages object %d is not a dictionary", pagesRef.Number)
	}

	kids := pagesDict.GetArray("Kids")
	if len(kids) == 0 {
		return errors.New("pades-sign: pages /Kids array is empty")
	}

	firstPageRef, ok := kids[0].Value.(pdfparse.Ref)
	if !ok {
		return errors.New("pades-sign: first /Kids entry is not a reference")
	}

	pageObj := writer.GetObject(firstPageRef.Number)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return fmt.Errorf("pades-sign: page object %d is not a dictionary", firstPageRef.Number)
	}

	newRef := pdfparse.RefObj(widgetObjNum, 0)
	existingAnnots := pageDict.Get(annotsKey)

	switch existingAnnots.Type {
	case pdfparse.ObjectArray:
		items, ok := existingAnnots.Value.([]pdfparse.Object)
		if ok {
			items = append(items, newRef)
			pageDict.Set(annotsKey, pdfparse.Arr(items...))
		} else {
			pageDict.Set(annotsKey, pdfparse.Arr(newRef))
		}
	default:
		pageDict.Set(annotsKey, pdfparse.Arr(newRef))
	}

	writer.SetObject(firstPageRef.Number, pdfparse.DictObj(pageDict))
	return nil
}

// ensureAcroForm adds or updates the /AcroForm dictionary in the document
// catalog to include the signature field and set the /SigFlags.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes widgetObjNum (int) which is the widget annotation object number
// to add to the /Fields array.
//
// Returns error when the catalog cannot be located or is not a
// dictionary.
func ensureAcroForm(writer *pdfparse.Writer, widgetObjNum int) error {
	trailer := writer.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return errors.New("pades-sign: trailer has no /Root reference")
	}

	catalogObj := writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return fmt.Errorf("pades-sign: catalog object %d is not a dictionary", rootRef.Number)
	}

	acroForm := catalogDict.GetDict("AcroForm")
	existingFields := acroForm.GetArray("Fields")
	fieldRef := pdfparse.RefObj(widgetObjNum, 0)

	if existingFields != nil {
		existingFields = append(existingFields, fieldRef)
		acroForm.Set("Fields", pdfparse.Arr(existingFields...))
	} else {
		acroForm.Set("Fields", pdfparse.Arr(fieldRef))
	}

	acroForm.Set("SigFlags", pdfparse.Int(sigFlagsValue))
	catalogDict.Set("AcroForm", pdfparse.DictObj(acroForm))

	writer.SetObject(rootRef.Number, pdfparse.DictObj(catalogDict))
	return nil
}

// locateContentsPlaceholder finds the /Contents hex string in the
// serialised PDF and returns the byte range array values along with the
// start and end offsets of the hex string (including angle brackets).
//
// Takes pdf ([]byte) which is the serialised PDF bytes.
// Takes sigDictObjNum (int) which is the signature dictionary object
// number.
//
// Returns [byteRangeSize]int64 which holds the computed byte range.
// Returns int which is the start offset of the hex string.
// Returns int which is the end offset of the hex string.
// Returns error when the signature dictionary or placeholder cannot
// be found.
func locateContentsPlaceholder(pdf []byte, sigDictObjNum int) (
	byteRange [byteRangeSize]int64, contentsStart int, contentsEnd int, err error,
) {
	objHeader := fmt.Appendf(nil, "%d 0 obj", sigDictObjNum)
	objStart := bytes.Index(pdf, objHeader)
	if objStart < 0 {
		return [byteRangeSize]int64{}, 0, 0, errSignatureDictNotFound
	}

	searchStart := objStart
	contentsIdx := bytes.Index(pdf[searchStart:], []byte(contentsPlaceholderTag))
	if contentsIdx < 0 {
		return [byteRangeSize]int64{}, 0, 0, errContentsPlaceholderNotFound
	}

	hexStart := searchStart + contentsIdx + len("/Contents ")
	hexEndRel := bytes.IndexByte(pdf[hexStart:], '>')
	if hexEndRel < 0 {
		return [byteRangeSize]int64{}, 0, 0, errContentsHexNotTerminated
	}
	hexEnd := hexStart + hexEndRel + 1

	byteRange = [byteRangeSize]int64{
		0,
		int64(hexStart),
		int64(hexEnd),
		int64(len(pdf) - hexEnd),
	}

	return byteRange, hexStart, hexEnd, nil
}

// patchByteRange replaces the /ByteRange array in the serialised PDF with
// the actual computed values.
//
// Takes pdf ([]byte) which is the serialised PDF bytes.
// Takes byteRange ([byteRangeSize]int64) which holds the computed byte
// range values.
//
// Returns []byte which is the PDF with the patched /ByteRange array.
func patchByteRange(pdf []byte, byteRange [byteRangeSize]int64) []byte {
	brTag := []byte("/ByteRange [")
	brIdx := bytes.Index(pdf, brTag)
	if brIdx < 0 {
		return pdf
	}

	brStart := brIdx + len(brTag)
	brEnd := bytes.IndexByte(pdf[brStart:], ']')
	if brEnd < 0 {
		return pdf
	}
	brEnd += brStart

	newBR := fmt.Sprintf("%-*s",
		brEnd-brStart,
		fmt.Sprintf("%d %d %d %d", byteRange[0], byteRange[1], byteRange[2], byteRange[3]),
	)

	result := make([]byte, len(pdf))
	copy(result, pdf)
	copy(result[brStart:brEnd], newBR)
	return result
}

// computeByteRangeDigest computes the SHA-256 hash over the byte ranges
// defined by the /ByteRange array.
//
// Takes pdf ([]byte) which is the serialised PDF bytes.
// Takes byteRange ([byteRangeSize]int64) which defines the ranges to hash.
//
// Returns []byte which is the SHA-256 digest.
func computeByteRangeDigest(pdf []byte, byteRange [byteRangeSize]int64) []byte {
	h := sha256.New()
	_, _ = h.Write(pdf[byteRange[0] : byteRange[0]+byteRange[1]])
	_, _ = h.Write(pdf[byteRange[2] : byteRange[2]+byteRange[3]])
	return h.Sum(nil)
}

// embedSignature inserts the DER-encoded CMS signature into the /Contents
// hex string placeholder.
//
// Takes pdf ([]byte) which is the serialised PDF bytes.
// Takes cmsData ([]byte) which is the DER-encoded CMS signature.
// Takes contentsStart (int) which is the start offset of the hex string.
// Takes contentsEnd (int) which is the end offset of the hex string.
//
// Returns []byte which is the PDF with the embedded signature.
// Returns error when the signature exceeds the reserved space.
func embedSignature(pdf, cmsData []byte, contentsStart, contentsEnd int) ([]byte, error) {
	hexSig := hex.EncodeToString(cmsData)

	available := contentsEnd - contentsStart - 2
	if len(hexSig) > available {
		return nil, fmt.Errorf("CMS signature (%d hex chars) exceeds reserved space (%d)", len(hexSig), available)
	}

	padded := hexSig + string(bytes.Repeat([]byte("0"), available-len(hexSig)))

	result := make([]byte, len(pdf))
	copy(result, pdf)
	copy(result[contentsStart+1:contentsEnd-1], padded)
	return result, nil
}

// buildCMSSignedData constructs a CMS SignedData structure for PAdES.
//
// Takes digest ([]byte) which is the SHA-256 hash of the byte ranges.
// Takes signer (crypto.Signer) which is the private key.
// Takes certs ([]*x509.Certificate) which is the certificate chain.
// Takes signingTime (time.Time) which is the signing timestamp.
// Takes timestampToken ([]byte) which is an optional DER-encoded
// RFC 3161 TimeStampToken. Nil for B-B.
//
// Returns []byte which is the DER-encoded CMS ContentInfo.
// Returns error when signing or ASN.1 encoding fails.
func buildCMSSignedData(
	digest []byte,
	signer crypto.Signer,
	certs []*x509.Certificate,
	signingTime time.Time,
	timestampToken []byte,
) ([]byte, error) {
	signedAttrs, err := buildSignedAttributes(digest, signingTime)
	if err != nil {
		return nil, fmt.Errorf("encoding signed attributes: %w", err)
	}

	attrDigest := sha256.Sum256(signedAttrs)

	sig, err := signer.Sign(rand.Reader, attrDigest[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("signing: %w", err)
	}

	signerInfoDER, err := buildSignerInfo(certs[0], signedAttrs, sig, timestampToken)
	if err != nil {
		return nil, fmt.Errorf("encoding signer info: %w", err)
	}

	var rawCerts []asn1.RawValue
	for _, cert := range certs {
		rawCerts = append(rawCerts, asn1.RawValue{
			FullBytes: cert.Raw,
		})
	}

	signedData := signedDataContent{
		Version: 1,
		DigestAlgorithms: []pkix.AlgorithmIdentifier{
			{Algorithm: oidSHA256},
		},
		EncapContentInfo: encapContentInfo{
			ContentType: oidData,
		},
		Certificates: rawCerts,
		SignerInfos:  []asn1.RawValue{{FullBytes: signerInfoDER}},
	}

	signedDataDER, err := asn1.Marshal(signedData)
	if err != nil {
		return nil, fmt.Errorf("encoding SignedData: %w", err)
	}

	ci := contentInfo{
		ContentType: oidSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      signedDataDER,
		},
	}

	return asn1.Marshal(ci)
}

// buildSignedAttributes constructs the DER-encoded signed attributes SET
// for CMS.
//
// Takes digest ([]byte) which is the SHA-256 message digest.
// Takes signingTime (time.Time) which is the signing timestamp.
//
// Returns []byte which is the DER-encoded SET of signed attributes.
// Returns error when ASN.1 encoding fails.
func buildSignedAttributes(digest []byte, signingTime time.Time) ([]byte, error) {
	contentTypeAttr := cmsAttribute{
		Type:   oidContentType,
		Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true},
	}
	ctVal, err := asn1.Marshal(oidData)
	if err != nil {
		return nil, err
	}
	contentTypeAttr.Values.Bytes = ctVal

	digestAttr := cmsAttribute{
		Type:   oidMessageDigest,
		Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true},
	}
	digestVal, err := asn1.Marshal(digest)
	if err != nil {
		return nil, err
	}
	digestAttr.Values.Bytes = digestVal

	timeAttr := cmsAttribute{
		Type:   oidSigningTime,
		Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true},
	}
	timeVal, err := asn1.Marshal(signingTime)
	if err != nil {
		return nil, err
	}
	timeAttr.Values.Bytes = timeVal

	attrs := []cmsAttribute{contentTypeAttr, digestAttr, timeAttr}

	encoded, err := asn1.Marshal(attrs)
	if err != nil {
		return nil, err
	}

	if len(encoded) > 0 && encoded[0] == asn1TagSequence {
		encoded[0] = asn1TagSet
	}

	return encoded, nil
}

// buildSignerInfo constructs a DER-encoded SignerInfo structure.
//
// When timestampToken is non-nil, it is added as an unsigned attribute.
//
// Takes cert (*x509.Certificate) which identifies the signer.
// Takes signedAttrs ([]byte) which is the DER-encoded signed attributes.
// Takes signature ([]byte) which is the raw signature value.
// Takes timestampToken ([]byte) which is an optional DER-encoded
// timestamp token.
//
// Returns []byte which is the DER-encoded SignerInfo.
// Returns error when ASN.1 encoding fails.
func buildSignerInfo(
	cert *x509.Certificate,
	signedAttrs []byte,
	signature []byte,
	timestampToken []byte,
) ([]byte, error) {
	issuerRaw := asn1.RawValue{FullBytes: cert.RawIssuer}
	serialDER, err := marshalASN1(cert.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("encoding certificate serial number: %w", err)
	}
	serialRaw := asn1.RawValue{FullBytes: serialDER}

	implicitAttrs := make([]byte, len(signedAttrs))
	copy(implicitAttrs, signedAttrs)
	if len(implicitAttrs) > 0 {
		implicitAttrs[0] = asn1TagImplicitConstructed0
	}

	si := cmsSignerInfo{
		Version: 1,
		SID: issuerAndSerial{
			Issuer: issuerRaw,
			Serial: serialRaw,
		},
		DigestAlgorithm: pkix.AlgorithmIdentifier{Algorithm: oidSHA256},
		SignedAttrs:     asn1.RawValue{FullBytes: implicitAttrs},
		SignatureAlgorithm: pkix.AlgorithmIdentifier{
			Algorithm: sigAlgorithmOID(cert),
		},
		Signature: signature,
	}

	if timestampToken != nil {
		unsignedAttrs, err := buildUnsignedAttributes(timestampToken)
		if err != nil {
			return nil, fmt.Errorf("encoding unsigned attributes: %w", err)
		}
		si.UnsignedAttrs = asn1.RawValue{FullBytes: unsignedAttrs}
	}

	return asn1.Marshal(si)
}

// buildUnsignedAttributes constructs the DER-encoded unsigned
// attributes for a SignerInfo.
//
// Takes timestampToken ([]byte) which is the DER-encoded timestamp
// token to embed.
//
// Returns []byte which is the DER-encoded unsigned attributes.
// Returns error when ASN.1 encoding fails.
func buildUnsignedAttributes(timestampToken []byte) ([]byte, error) {
	tsAttr := cmsAttribute{
		Type: oidTimeStampToken,
		Values: asn1.RawValue{
			Class:      asn1.ClassUniversal,
			Tag:        asn1.TagSet,
			IsCompound: true,
			Bytes:      timestampToken,
		},
	}

	attrs := []cmsAttribute{tsAttr}

	encoded, err := asn1.Marshal(attrs)
	if err != nil {
		return nil, err
	}

	if len(encoded) > 0 && encoded[0] == asn1TagSequence {
		encoded[0] = asn1TagImplicitConstructed1
	}

	return encoded, nil
}

// sigAlgorithmOID returns the appropriate signature algorithm OID based
// on the certificate's public key algorithm.
//
// Takes cert (*x509.Certificate) which provides the public key algorithm.
//
// Returns asn1.ObjectIdentifier which is the signature algorithm OID.
func sigAlgorithmOID(cert *x509.Certificate) asn1.ObjectIdentifier {
	switch cert.PublicKeyAlgorithm {
	case x509.ECDSA:
		return oidECDSAWithSHA256
	default:
		return oidSHA256WithRSA
	}
}

// marshalASN1 marshals a value to ASN.1 DER encoding.
//
// Takes val (any) which is the value to encode.
//
// Returns []byte which is the DER-encoded value.
// Returns error when encoding fails.
func marshalASN1(val any) ([]byte, error) {
	b, err := asn1.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("asn1.Marshal: %w", err)
	}
	return b, nil
}

// contentInfo is the top-level CMS ContentInfo structure (RFC 5652 s3).
type contentInfo struct {
	// ContentType holds the OID identifying the content type.
	ContentType asn1.ObjectIdentifier

	// Content holds the wrapped content value.
	Content asn1.RawValue `asn1:"explicit,tag:0"`
}

// signedDataContent is the CMS SignedData structure (RFC 5652 s5.1).
// Field order matches the ASN.1 SEQUENCE definition and must not be
// reordered.
//
//nolint:govet // ASN.1 field order
type signedDataContent struct {
	// Version holds the CMS syntax version number.
	Version int

	// DigestAlgorithms holds the set of digest algorithm identifiers.
	DigestAlgorithms []pkix.AlgorithmIdentifier `asn1:"set"`

	// EncapContentInfo holds the encapsulated content info.
	EncapContentInfo encapContentInfo

	// Certificates holds the DER-encoded certificate chain.
	Certificates []asn1.RawValue `asn1:"optional,set,tag:0"`

	// SignerInfos holds the set of per-signer information.
	SignerInfos []asn1.RawValue `asn1:"set"`
}

// encapContentInfo is the CMS EncapsulatedContentInfo (RFC 5652 s5.2).
// For detached signatures, eContent is absent.
type encapContentInfo struct {
	// ContentType holds the OID identifying the encapsulated content type.
	ContentType asn1.ObjectIdentifier
}

// issuerAndSerial identifies a certificate by issuer name and serial
// number (RFC 5652 s10.2.4).
type issuerAndSerial struct {
	// Issuer holds the DER-encoded issuer distinguished name.
	Issuer asn1.RawValue

	// Serial holds the DER-encoded certificate serial number.
	Serial asn1.RawValue
}

// cmsSignerInfo is the CMS SignerInfo structure (RFC 5652 s5.3).
// Field order matches the ASN.1 SEQUENCE definition and must not be
// reordered.
//
//nolint:govet // ASN.1 field order
type cmsSignerInfo struct {
	// Version holds the CMS SignerInfo syntax version number.
	Version int

	// SID holds the signer identifier (issuer and serial number).
	SID issuerAndSerial

	// DigestAlgorithm holds the digest algorithm used by the signer.
	DigestAlgorithm pkix.AlgorithmIdentifier

	// SignedAttrs holds the optional IMPLICIT [0] signed attributes.
	SignedAttrs asn1.RawValue `asn1:"optional"`

	// SignatureAlgorithm holds the signature algorithm identifier.
	SignatureAlgorithm pkix.AlgorithmIdentifier

	// Signature holds the raw signature value bytes.
	Signature []byte

	// UnsignedAttrs holds the optional IMPLICIT [1] unsigned attributes.
	UnsignedAttrs asn1.RawValue `asn1:"optional"`
}

// cmsAttribute is a single CMS attribute (type + values set).
type cmsAttribute struct {
	// Type holds the attribute type OID.
	Type asn1.ObjectIdentifier

	// Values holds the SET OF attribute values.
	Values asn1.RawValue
}
