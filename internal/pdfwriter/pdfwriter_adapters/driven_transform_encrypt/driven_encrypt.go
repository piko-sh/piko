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

package driven_transform_encrypt

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"

	"golang.org/x/text/secure/precis"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the encrypt transformer.
	defaultPriority = 400

	// transformerName identifies this transformer for registry lookups and
	// validation prefixes.
	transformerName = "pdf-encrypt"

	// algorithmAES256 is the algorithm identifier for AES-256 encryption.
	algorithmAES256 = "aes-256"

	// algorithmAES128 is the algorithm identifier for AES-128 encryption.
	algorithmAES128 = "aes-128"

	// algorithmRC4128 is the algorithm identifier for RC4-128 encryption.
	algorithmRC4128 = "rc4-128"

	// encryptionKeyLength is the AES-256 key size in bytes.
	encryptionKeyLength = 32

	// saltLength is the length of validation and key salts in bytes.
	saltLength = 8

	// uValueLength is the total length of the /U and /O values:
	// hash(32) + validation_salt(8) + key_salt(8).
	uValueLength = 48

	// aesBlockSize is the AES block size in bytes.
	aesBlockSize = 16

	// permsPlaintextLength is the length of the Perms plaintext before
	// AES-ECB encryption.
	permsPlaintextLength = 16

	// permsNoEncryptMetadataStart is the byte offset where the
	// 0xFFFFFFFF "no EncryptMetadata restriction" field begins.
	permsNoEncryptMetadataStart = 4

	// permsNoEncryptMetadataEnd is the byte offset after the last byte
	// of the no-EncryptMetadata field.
	permsNoEncryptMetadataEnd = 8

	// permsNoEncryptMetadataFill is the byte value 0xFF used for each
	// byte of the no-EncryptMetadata field.
	permsNoEncryptMetadataFill = 0xFF

	// permsEncryptMetadataOffset is the byte offset for the
	// EncryptMetadata flag ('T' for true).
	permsEncryptMetadataOffset = 8

	// permsMarkerOffset is the byte offset where the "adb" marker begins.
	permsMarkerOffset = 9

	// permsRandomStart is the byte offset where random padding begins.
	permsRandomStart = 12

	// permsMarkerByte0 is the first byte of the "adb" marker.
	permsMarkerByte0 = 'a'

	// permsMarkerByte1 is the second byte of the "adb" marker.
	permsMarkerByte1 = 'd'

	// permsMarkerByte2 is the third byte of the "adb" marker.
	permsMarkerByte2 = 'b'

	// encryptDictV is the /V value for AES-256 encryption (PDF 2.0).
	encryptDictV = 5

	// encryptDictR is the /R value for AES-256 encryption (PDF 2.0).
	encryptDictR = 6

	// encryptDictKeyBits is the /Length value in bits for AES-256.
	encryptDictKeyBits = 256

	// cfmAESV3 is the /CFM value for AES-256 crypt filters.
	cfmAESV3 = "AESV3"

	// stdCF is the name of the standard crypt filter.
	stdCF = "StdCF"

	// documentIDLength is the length of the /ID byte strings in the trailer.
	documentIDLength = 16

	// algorithm2BRepeatCount is the number of times the K1 block is repeated
	// in algorithm 2.B before AES-128-CBC encryption.
	algorithm2BRepeatCount = 64

	// algorithm2BHashPrefixLen is the number of bytes from the encrypted
	// output used to determine the next hash function in algorithm 2.B.
	algorithm2BHashPrefixLen = 16

	// algorithm2BHashModulus selects the next hash function in algorithm 2.B.
	algorithm2BHashModulus = 3

	// algorithm2BMaxRounds caps Algorithm 2.B's outer loop as a defence
	// against pathological inputs that would otherwise loop unbounded. The
	// spec terminates with overwhelming probability well before 1024
	// iterations; qpdf carries the same safety net.
	algorithm2BMaxRounds = 1024

	// algorithm2BMinRounds is the minimum 1-based round number before the
	// termination condition in algorithm 2.B is evaluated. The PDF spec
	// uses 1-based round counting (round 1 is the first AES iteration).
	algorithm2BMinRounds = 64

	// algorithm2BRoundOffset is the offset subtracted from the 1-based round
	// number in the algorithm 2.B termination condition: terminate when the
	// last byte of E is <= round_number - algorithm2BRoundOffset.
	algorithm2BRoundOffset = 32

	// algorithm2BOutputLen is the length of the final hash output in bytes
	// returned by algorithm 2.B.
	algorithm2BOutputLen = 32

	// defaultMaxObjectNestingDepth caps recursion when encrypting nested PDF
	// dictionaries and arrays.
	defaultMaxObjectNestingDepth = 256

	// passwordMaxBytes is the maximum byte length passed to Algorithm 2.B
	// per ISO 32000-2 section 7.6.4.3.2; longer prepared passwords are
	// truncated. Truncation is by bytes, not runes, to match qpdf and
	// other reference readers.
	passwordMaxBytes = 127
)

// ErrObjectNestingTooDeep is returned when a PDF object's nested
// dictionary/array structure exceeds the configured depth limit.
var ErrObjectNestingTooDeep = errors.New("encrypt: PDF object nesting exceeds depth limit")

// ErrAlgorithm2BNotConverging is returned when Algorithm 2.B fails to
// reach its termination condition within algorithm2BMaxRounds rounds.
var ErrAlgorithm2BNotConverging = errors.New("encrypt: algorithm 2.B exceeded round limit without terminating")

// EncryptTransformer applies AES-256 encryption to PDF documents per
// ISO 32000-2 section 7.6.
//
// It generates a random 32-byte file encryption key, computes the /U, /UE,
// /O, /OE, and /Perms values from the owner and user passwords, encrypts all
// string and stream objects with AES-256-CBC, and adds the /Encrypt dictionary
// to the trailer. Only the "aes-256" algorithm is currently implemented;
// "aes-128" and "rc4-128" return an unsupported algorithm error.
type EncryptTransformer struct {
	// randomSource supplies cryptographic randomness. Defaults to
	// crypto/rand.Reader.
	randomSource io.Reader

	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int

	// maxObjectNestingDepth caps recursion when walking nested
	// dictionaries and arrays during encryption.
	maxObjectNestingDepth int
}

var _ pdfwriter_domain.PdfTransformerPort = (*EncryptTransformer)(nil)

// Option configures an EncryptTransformer at construction time.
type Option func(*EncryptTransformer)

// WithRandomSource overrides the cryptographic randomness source.
//
// Takes r (io.Reader) which is the randomness source.
//
// Returns Option which the EncryptTransformer applies at construction.
func WithRandomSource(r io.Reader) Option {
	return func(t *EncryptTransformer) { t.randomSource = r }
}

// WithMaxObjectNestingDepth overrides the maximum recursion depth permitted
// when encrypting nested PDF dictionaries and arrays. A non-positive value
// is ignored; callers that want to disable the limit must pass math.MaxInt.
//
// Takes depth (int) which is the maximum recursion depth.
//
// Returns Option which the EncryptTransformer applies at construction.
func WithMaxObjectNestingDepth(depth int) Option {
	return func(t *EncryptTransformer) {
		if depth > 0 {
			t.maxObjectNestingDepth = depth
		}
	}
}

// New creates a new encrypt transformer with default name and priority.
//
// Takes opts (...Option) which override defaults; callers wanting standard
// production behaviour can omit them.
//
// Returns *EncryptTransformer which is the initialised transformer.
func New(opts ...Option) *EncryptTransformer {
	t := &EncryptTransformer{
		name:                  transformerName,
		priority:              defaultPriority,
		randomSource:          rand.Reader,
		maxObjectNestingDepth: defaultMaxObjectNestingDepth,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *EncryptTransformer) Name() string { return t.name }

// Type returns TransformerSecurity.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a security
// transformer.
func (*EncryptTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerSecurity
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *EncryptTransformer) Priority() int { return t.priority }

// Transform applies AES-256 encryption to the PDF. Options must be
// EncryptionOptions or *EncryptionOptions.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be EncryptionOptions or *EncryptionOptions.
//
// Returns []byte which is the encrypted PDF.
// Returns error when the PDF cannot be parsed or encryption fails.
func (t *EncryptTransformer) Transform(ctx context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}

	if err := validateOptions(&opts); err != nil {
		return nil, err
	}

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("encrypt: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("encrypt: creating writer: %w", err)
	}

	fileKey, err := t.generateFileKey()
	if err != nil {
		return nil, fmt.Errorf("encrypt: generating file key: %w", err)
	}

	userPassword, err := preparePassword(opts.UserPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypt: preparing user password: %w", err)
	}
	ownerPassword, err := preparePassword(opts.OwnerPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypt: preparing owner password: %w", err)
	}

	uValue, ueValue, err := t.computeUserValues(fileKey, userPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypt: computing U/UE: %w", err)
	}

	oValue, oeValue, err := t.computeOwnerValues(fileKey, ownerPassword, uValue)
	if err != nil {
		return nil, fmt.Errorf("encrypt: computing O/OE: %w", err)
	}

	permsValue, err := t.computePerms(fileKey, opts.Permissions)
	if err != nil {
		return nil, fmt.Errorf("encrypt: computing Perms: %w", err)
	}

	encryptObjNum := addEncryptDict(writer, uValue, ueValue, oValue, oeValue, permsValue, opts.Permissions)

	if err := t.encryptObjects(ctx, writer, doc, fileKey); err != nil {
		return nil, fmt.Errorf("encrypt: encrypting objects: %w", err)
	}

	return t.finaliseEncryptedPDF(writer, encryptObjNum)
}

// finaliseEncryptedPDF sets the /Encrypt reference in the trailer, ensures a
// valid /ID array exists, and serialises the PDF.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes encryptObjNum (int) which is the object number of the /Encrypt dictionary.
//
// Returns []byte which is the finalised encrypted PDF.
// Returns error when ID generation or PDF writing fails.
func (t *EncryptTransformer) finaliseEncryptedPDF(writer *pdfparse.Writer, encryptObjNum int) ([]byte, error) {
	trailer := writer.Trailer()
	trailer.Set("Encrypt", pdfparse.RefObj(encryptObjNum, 0))

	if !trailer.Has("ID") {
		docID := make([]byte, documentIDLength)
		if _, idErr := io.ReadFull(t.randomSource, docID); idErr != nil {
			return nil, fmt.Errorf("encrypt: generating document ID: %w", idErr)
		}
		trailer.Set("ID", pdfparse.Arr(
			pdfparse.HexStr(string(docID)),
			pdfparse.HexStr(string(docID)),
		))
	}

	writer.SetTrailer(trailer)

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("encrypt: writing PDF: %w", err)
	}
	return output, nil
}

// castOptions extracts EncryptionOptions from the generic options.
//
// Takes options (any) which must be EncryptionOptions or *EncryptionOptions.
//
// Returns pdfwriter_dto.EncryptionOptions which holds the extracted options.
// Returns error when the options type is invalid or nil.
func castOptions(options any) (pdfwriter_dto.EncryptionOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.EncryptionOptions:
		return v, nil
	case *pdfwriter_dto.EncryptionOptions:
		if v == nil {
			return pdfwriter_dto.EncryptionOptions{}, errors.New("encrypt: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.EncryptionOptions{}, fmt.Errorf("encrypt: expected EncryptionOptions, got %T", options)
	}
}

// validateOptions checks that the encryption options are valid.
//
// An empty Algorithm defaults to "aes-256". Only "aes-256" is currently
// supported; "aes-128" and "rc4-128" return an unsupported algorithm error.
//
// Takes opts (*pdfwriter_dto.EncryptionOptions) which holds the options to validate.
//
// Returns error when the algorithm is unsupported or the owner password is empty.
func validateOptions(opts *pdfwriter_dto.EncryptionOptions) error {
	if opts.Algorithm == "" {
		opts.Algorithm = algorithmAES256
	}

	switch opts.Algorithm {
	case algorithmAES256:

	case algorithmAES128:
		return fmt.Errorf("encrypt: algorithm %q is not yet supported; use %q", algorithmAES128, algorithmAES256)
	case algorithmRC4128:
		return fmt.Errorf("encrypt: algorithm %q is not yet supported; use %q", algorithmRC4128, algorithmAES256)
	default:
		return fmt.Errorf("encrypt: unknown algorithm %q", opts.Algorithm)
	}

	if opts.OwnerPassword == "" {
		return errors.New("encrypt: owner password must not be empty")
	}

	return nil
}

// generateFileKey generates a cryptographically random 32-byte file
// encryption key.
//
// Returns []byte which is the generated encryption key.
// Returns error when random byte generation fails.
func (t *EncryptTransformer) generateFileKey() ([]byte, error) {
	key := make([]byte, encryptionKeyLength)
	if _, err := io.ReadFull(t.randomSource, key); err != nil {
		return nil, fmt.Errorf("reading random bytes: %w", err)
	}
	return key, nil
}

// preparePassword normalises and truncates a password for use in PDF 2.0
// revision 6 encryption per ISO 32000-2 section 7.6.4.3.2.
//
// The PRECIS OpaqueString profile applies SASLprep-style Unicode
// normalisation and rejects disallowed code points, which keeps generated
// PDFs interoperable with conformant readers (qpdf, Acrobat). The result
// is encoded as UTF-8 and truncated to passwordMaxBytes; truncation is by
// bytes (not runes) so a multi-byte rune sitting on the boundary is
// dropped, matching the qpdf reference implementation. Empty passwords
// bypass PRECIS preparation: the spec permits them (open without prompt)
// and PRECIS treats the empty string as invalid input.
//
// Takes password (string) which is the caller-supplied password.
//
// Returns []byte which is the prepared, byte-truncated password.
// Returns error when PRECIS preparation rejects the input.
func preparePassword(password string) ([]byte, error) {
	if password == "" {
		return []byte{}, nil
	}
	prepared, err := precis.OpaqueString.String(password)
	if err != nil {
		return nil, fmt.Errorf("preparing password: %w", err)
	}
	bytesOut := []byte(prepared)
	if len(bytesOut) > passwordMaxBytes {
		bytesOut = bytesOut[:passwordMaxBytes]
	}
	return bytesOut, nil
}

// algorithm2B implements the ISO 32000-2 section 7.6.4.3.4 "Algorithm 2.B"
// hash computation used by revision 6 encryption. It takes a SHA-256 hash as
// input and iteratively applies AES-128-CBC encryption followed by SHA-256,
// SHA-384, or SHA-512 hashing based on the encrypted output, until a
// termination condition is met.
//
// Takes input ([]byte) which is the initial data to hash (password + salt, or
// password + salt + U for owner values).
// Takes password ([]byte) which is the prepared UTF-8 password (max 127 bytes
// per ISO 32000-2 section 7.6.4.3.2).
// Takes userKey ([]byte) which is the 48-byte /U value for owner password
// computations, or nil for user password computations.
//
// Returns []byte which is the 32-byte hash result.
// Returns error when AES cipher construction fails or the round limit is
// exceeded without convergence.
func algorithm2B(input, password, userKey []byte) ([]byte, error) {
	k := sha256Hash(input)

	for round := 1; round <= algorithm2BMaxRounds; round++ {
		single := make([]byte, 0, len(password)+len(k)+len(userKey))
		single = append(single, password...)
		single = append(single, k...)
		single = append(single, userKey...)

		k1 := make([]byte, 0, len(single)*algorithm2BRepeatCount)
		for range algorithm2BRepeatCount {
			k1 = append(k1, single...)
		}

		block, err := aes.NewCipher(k[:16])
		if err != nil {
			return nil, fmt.Errorf("creating AES cipher in algorithm 2.B: %w", err)
		}
		mode := cipher.NewCBCEncrypter(block, k[16:32])
		encrypted := make([]byte, len(k1))
		mode.CryptBlocks(encrypted, k1)

		var byteSum int
		for _, b := range encrypted[:algorithm2BHashPrefixLen] {
			byteSum += int(b)
		}

		hasher, err := hasherForByteSum(byteSum % algorithm2BHashModulus)
		if err != nil {
			return nil, err
		}

		_, _ = hasher.Write(encrypted)
		k = hasher.Sum(nil)

		lastByte := encrypted[len(encrypted)-1]
		if round >= algorithm2BMinRounds && int(lastByte) <= (round-algorithm2BRoundOffset) {
			return k[:algorithm2BOutputLen], nil
		}
	}

	return nil, ErrAlgorithm2BNotConverging
}

// hasherForByteSum returns the algorithm 2.B hash function selected by the
// modulo-3 residue of the first 16 bytes of E. The spec defines the
// selection as the big-endian integer mod 3; summing the bytes mod 3 is
// equivalent because 256 mod 3 == 1.
//
// Takes residue (int) which must be 0, 1, or 2.
//
// Returns hash.Hash which is the selected hasher.
// Returns error when residue is outside {0, 1, 2}.
func hasherForByteSum(residue int) (hash.Hash, error) {
	switch residue {
	case 0:
		return sha256.New(), nil
	case 1:
		return sha512.New384(), nil
	case 2:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("encrypt: unexpected algorithm 2.B residue %d", residue)
	}
}

// sha256Hash returns the SHA-256 digest of data.
//
// Takes data ([]byte) which is the input to hash.
//
// Returns []byte which is the 32-byte SHA-256 digest.
func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// computeUserValues computes the /U and /UE values per ISO 32000-2
// section 7.6.4.3.3.
//
// U is 48 bytes: SHA-256(password + validation_salt) (32 bytes) +
// validation_salt (8 bytes) + key_salt (8 bytes).
//
// UE is the file encryption key encrypted with AES-256-CBC using
// SHA-256(password + key_salt) as the key, with a zero IV.
//
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
// Takes userPassword ([]byte) which is the user password bytes.
//
// Returns []byte which is the 48-byte /U value.
// Returns []byte which is the encrypted /UE value.
// Returns error when random generation or encryption fails.
func (t *EncryptTransformer) computeUserValues(fileKey, userPassword []byte) (uValue []byte, ueValue []byte, err error) {
	validationSalt := make([]byte, saltLength)
	keySalt := make([]byte, saltLength)
	if _, err = io.ReadFull(t.randomSource, validationSalt); err != nil {
		return nil, nil, fmt.Errorf("generating user validation salt: %w", err)
	}
	if _, err = io.ReadFull(t.randomSource, keySalt); err != nil {
		return nil, nil, fmt.Errorf("generating user key salt: %w", err)
	}

	validationInput := make([]byte, 0, len(userPassword)+saltLength)
	validationInput = append(validationInput, userPassword...)
	validationInput = append(validationInput, validationSalt...)
	validationHash, err := algorithm2B(validationInput, userPassword, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("computing user validation hash: %w", err)
	}

	uValue = make([]byte, 0, uValueLength)
	uValue = append(uValue, validationHash...)
	uValue = append(uValue, validationSalt...)
	uValue = append(uValue, keySalt...)

	keyInput := make([]byte, 0, len(userPassword)+saltLength)
	keyInput = append(keyInput, userPassword...)
	keyInput = append(keyInput, keySalt...)
	keyHash, err := algorithm2B(keyInput, userPassword, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("computing user key hash: %w", err)
	}
	ueValue, err = aes256CBCEncryptZeroIV(keyHash, fileKey)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypting UE: %w", err)
	}

	return uValue, ueValue, nil
}

// computeOwnerValues computes the /O and /OE values per ISO 32000-2
// section 7.6.4.3.3.
//
// O is 48 bytes: SHA-256(password + validation_salt + U) (32 bytes) +
// validation_salt (8 bytes) + key_salt (8 bytes).
//
// OE is the file encryption key encrypted with AES-256-CBC using
// SHA-256(password + key_salt + U) as the key, with a zero IV.
//
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
// Takes ownerPassword ([]byte) which is the owner password bytes.
// Takes uBytes ([]byte) which is the 48-byte /U value.
//
// Returns []byte which is the 48-byte /O value.
// Returns []byte which is the encrypted /OE value.
// Returns error when random generation or encryption fails.
func (t *EncryptTransformer) computeOwnerValues(fileKey, ownerPassword, uBytes []byte) (oValue []byte, oeValue []byte, err error) {
	validationSalt := make([]byte, saltLength)
	keySalt := make([]byte, saltLength)
	if _, err = io.ReadFull(t.randomSource, validationSalt); err != nil {
		return nil, nil, fmt.Errorf("generating owner validation salt: %w", err)
	}
	if _, err = io.ReadFull(t.randomSource, keySalt); err != nil {
		return nil, nil, fmt.Errorf("generating owner key salt: %w", err)
	}

	validationInput := make([]byte, 0, len(ownerPassword)+saltLength+len(uBytes))
	validationInput = append(validationInput, ownerPassword...)
	validationInput = append(validationInput, validationSalt...)
	validationInput = append(validationInput, uBytes...)
	validationHash, err := algorithm2B(validationInput, ownerPassword, uBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("computing owner validation hash: %w", err)
	}

	oValue = make([]byte, 0, uValueLength)
	oValue = append(oValue, validationHash...)
	oValue = append(oValue, validationSalt...)
	oValue = append(oValue, keySalt...)

	keyInput := make([]byte, 0, len(ownerPassword)+saltLength+len(uBytes))
	keyInput = append(keyInput, ownerPassword...)
	keyInput = append(keyInput, keySalt...)
	keyInput = append(keyInput, uBytes...)
	keyHash, err := algorithm2B(keyInput, ownerPassword, uBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("computing owner key hash: %w", err)
	}

	oeValue, err = aes256CBCEncryptZeroIV(keyHash, fileKey)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypting OE: %w", err)
	}

	return oValue, oeValue, nil
}

// computePerms computes the /Perms value per ISO 32000-2 section 7.6.4.3.3.
// The Perms value is a 16-byte AES-256-ECB encryption of a plaintext block
// containing the permissions integer and the "adb" marker.
//
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
// Takes permissions (uint32) which is the PDF permission flags bitmask.
//
// Returns []byte which is the 16-byte encrypted /Perms value.
// Returns error when encryption fails.
func (t *EncryptTransformer) computePerms(fileKey []byte, permissions uint32) ([]byte, error) {
	plaintext := make([]byte, permsPlaintextLength)

	binary.LittleEndian.PutUint32(plaintext[0:permsNoEncryptMetadataStart], permissions)

	for i := permsNoEncryptMetadataStart; i < permsNoEncryptMetadataEnd; i++ {
		plaintext[i] = permsNoEncryptMetadataFill
	}

	plaintext[permsEncryptMetadataOffset] = 'T'

	plaintext[permsMarkerOffset] = permsMarkerByte0
	plaintext[permsMarkerOffset+1] = permsMarkerByte1
	plaintext[permsMarkerOffset+2] = permsMarkerByte2

	if _, err := io.ReadFull(t.randomSource, plaintext[permsRandomStart:permsPlaintextLength]); err != nil {
		return nil, fmt.Errorf("generating random Perms padding: %w", err)
	}

	block, err := aes.NewCipher(fileKey)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher for Perms: %w", err)
	}
	encrypted := make([]byte, aesBlockSize)
	block.Encrypt(encrypted, plaintext)

	return encrypted, nil
}

// addEncryptDict creates the /Encrypt dictionary and adds it to the
// writer.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes uValue ([]byte) which is the 48-byte /U value.
// Takes ueValue ([]byte) which is the encrypted /UE value.
// Takes oValue ([]byte) which is the 48-byte /O value.
// Takes oeValue ([]byte) which is the encrypted /OE value.
// Takes permsValue ([]byte) which is the 16-byte encrypted /Perms value.
// Takes permissions (uint32) which is the PDF permission flags bitmask.
//
// Returns int which is the object number of the new /Encrypt dictionary.
func addEncryptDict(
	writer *pdfparse.Writer,
	uValue, ueValue, oValue, oeValue, permsValue []byte,
	permissions uint32,
) int {
	stdCFDict := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("CryptFilter")},
		{Key: "CFM", Value: pdfparse.Name(cfmAESV3)},
		{Key: "Length", Value: pdfparse.Int(int64(encryptionKeyLength))},
		{Key: "AuthEvent", Value: pdfparse.Name("DocOpen")},
	}}

	cfDict := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: stdCF, Value: pdfparse.DictObj(stdCFDict)},
	}}

	encryptDict := pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Encrypt")},
		{Key: "Filter", Value: pdfparse.Name("Standard")},
		{Key: "V", Value: pdfparse.Int(encryptDictV)},
		{Key: "R", Value: pdfparse.Int(encryptDictR)},
		{Key: "Length", Value: pdfparse.Int(encryptDictKeyBits)},
		{Key: "CF", Value: pdfparse.DictObj(cfDict)},
		{Key: "StmF", Value: pdfparse.Name(stdCF)},
		{Key: "StrF", Value: pdfparse.Name(stdCF)},
		{Key: "O", Value: pdfparse.HexStr(string(oValue))},
		{Key: "U", Value: pdfparse.HexStr(string(uValue))},
		{Key: "OE", Value: pdfparse.HexStr(string(oeValue))},
		{Key: "UE", Value: pdfparse.HexStr(string(ueValue))},
		{Key: "P", Value: pdfparse.Int(permissionsToSigned(permissions))},
		{Key: "Perms", Value: pdfparse.HexStr(string(permsValue))},
		{Key: "EncryptMetadata", Value: pdfparse.Bool(true)},
	}}

	return writer.AddObject(pdfparse.DictObj(encryptDict))
}

// encryptObjects encrypts all string and stream objects in the PDF using
// AES-256-CBC with the file encryption key.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which provides the object numbers to iterate.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
//
// Returns error when encryption of any object fails or context is cancelled.
func (t *EncryptTransformer) encryptObjects(ctx context.Context, writer *pdfparse.Writer, doc *pdfparse.Document, fileKey []byte) error {
	for _, objNum := range doc.ObjectNumbers() {
		if ctx.Err() != nil {
			return fmt.Errorf("encrypting objects: %w", ctx.Err())
		}

		obj := writer.GetObject(objNum)
		if obj.IsNull() {
			continue
		}

		encrypted, err := t.encryptObject(obj, fileKey)
		if err != nil {
			return fmt.Errorf("object %d: %w", objNum, err)
		}
		writer.SetObject(objNum, encrypted)
	}
	return nil
}

// encryptObject encrypts a single PDF object.
//
// For stream objects, the stream data is encrypted with AES-256-CBC. For
// dictionary objects, string values within the dictionary are encrypted.
// Other object types are returned unchanged.
//
// Takes obj (pdfparse.Object) which is the object to encrypt.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
//
// Returns pdfparse.Object which is the encrypted object.
// Returns error when encryption fails.
func (t *EncryptTransformer) encryptObject(obj pdfparse.Object, fileKey []byte) (pdfparse.Object, error) {
	switch obj.Type {
	case pdfparse.ObjectStream:
		return t.encryptStreamObject(obj, fileKey)
	case pdfparse.ObjectDictionary:
		dict, ok := obj.Value.(pdfparse.Dict)
		if !ok {
			return obj, nil
		}
		encryptedDict, err := t.encryptDict(dict, fileKey, 0)
		if err != nil {
			return obj, err
		}
		return pdfparse.DictObj(encryptedDict), nil
	default:
		return obj, nil
	}
}

// encryptStreamObject encrypts a stream object's data with AES-256-CBC
// and also encrypts any string values in its dictionary.
//
// Takes obj (pdfparse.Object) which is the stream object to encrypt.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
//
// Returns pdfparse.Object which is the encrypted stream object.
// Returns error when encryption of the stream data or dictionary fails.
func (t *EncryptTransformer) encryptStreamObject(obj pdfparse.Object, fileKey []byte) (pdfparse.Object, error) {
	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return obj, nil
	}

	streamData := obj.StreamData

	encryptedData, err := t.aes256CBCEncrypt(fileKey, streamData)
	if err != nil {
		return obj, fmt.Errorf("encrypting stream data: %w", err)
	}

	encryptedDict, err := t.encryptDict(dict, fileKey, 0)
	if err != nil {
		return obj, fmt.Errorf("encrypting stream dictionary: %w", err)
	}

	return pdfparse.StreamObj(encryptedDict, encryptedData), nil
}

// encryptDict encrypts all string and hex string values in a dictionary.
//
// Takes dict (pdfparse.Dict) which is the dictionary to encrypt.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
// Takes depth (int) which is the current recursion depth; values exceeding
// the configured cap return ErrObjectNestingTooDeep.
//
// Returns pdfparse.Dict which is the dictionary with encrypted values.
// Returns error when encryption of any value fails.
func (t *EncryptTransformer) encryptDict(dict pdfparse.Dict, fileKey []byte, depth int) (pdfparse.Dict, error) {
	if depth > t.maxObjectNestingDepth {
		return dict, fmt.Errorf("%w (depth %d, limit %d)", ErrObjectNestingTooDeep, depth, t.maxObjectNestingDepth)
	}
	result := pdfparse.Dict{Pairs: make([]pdfparse.DictPair, len(dict.Pairs))}
	for i, pair := range dict.Pairs {
		encrypted, err := t.encryptValue(pair.Value, fileKey, depth+1)
		if err != nil {
			return dict, fmt.Errorf("key %q: %w", pair.Key, err)
		}
		result.Pairs[i] = pdfparse.DictPair{Key: pair.Key, Value: encrypted}
	}
	return result, nil
}

// encryptValue encrypts a single PDF object value.
//
// Strings and hex strings are encrypted with AES-256-CBC. Dictionaries and
// arrays are processed recursively.
//
// Takes obj (pdfparse.Object) which is the value to encrypt.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
// Takes depth (int) which is the current recursion depth.
//
// Returns pdfparse.Object which is the encrypted value.
// Returns error when encryption fails.
func (t *EncryptTransformer) encryptValue(obj pdfparse.Object, fileKey []byte, depth int) (pdfparse.Object, error) {
	switch obj.Type {
	case pdfparse.ObjectString, pdfparse.ObjectHexString:
		return t.encryptStringValue(obj, fileKey)
	case pdfparse.ObjectDictionary:
		return t.encryptDictValue(obj, fileKey, depth)
	case pdfparse.ObjectArray:
		return t.encryptArrayValue(obj, fileKey, depth)
	default:
		return obj, nil
	}
}

// encryptStringValue encrypts a literal or hex string value with
// AES-256-CBC and returns it as a hex string.
//
// Takes obj (pdfparse.Object) which is the string value to encrypt.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
//
// Returns pdfparse.Object which is the encrypted hex string.
// Returns error when encryption fails.
func (t *EncryptTransformer) encryptStringValue(obj pdfparse.Object, fileKey []byte) (pdfparse.Object, error) {
	s, ok := obj.Value.(string)
	if !ok {
		return obj, nil
	}
	encrypted, err := t.aes256CBCEncrypt(fileKey, []byte(s))
	if err != nil {
		return obj, fmt.Errorf("encrypting string value: %w", err)
	}
	return pdfparse.HexStr(string(encrypted)), nil
}

// encryptDictValue encrypts all string values within a dictionary object.
//
// Takes obj (pdfparse.Object) which is the dictionary object to encrypt.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
// Takes depth (int) which is the current recursion depth.
//
// Returns pdfparse.Object which is the dictionary with encrypted values.
// Returns error when encryption of any value fails.
func (t *EncryptTransformer) encryptDictValue(obj pdfparse.Object, fileKey []byte, depth int) (pdfparse.Object, error) {
	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return obj, nil
	}
	encryptedDict, err := t.encryptDict(dict, fileKey, depth)
	if err != nil {
		return obj, err
	}
	return pdfparse.DictObj(encryptedDict), nil
}

// encryptArrayValue encrypts all elements within an array object.
//
// Takes obj (pdfparse.Object) which is the array object to encrypt.
// Takes fileKey ([]byte) which is the 32-byte file encryption key.
// Takes depth (int) which is the current recursion depth.
//
// Returns pdfparse.Object which is the array with encrypted elements.
// Returns error when encryption of any element fails.
func (t *EncryptTransformer) encryptArrayValue(obj pdfparse.Object, fileKey []byte, depth int) (pdfparse.Object, error) {
	if depth > t.maxObjectNestingDepth {
		return obj, fmt.Errorf("%w (depth %d, limit %d)", ErrObjectNestingTooDeep, depth, t.maxObjectNestingDepth)
	}
	items, ok := obj.Value.([]pdfparse.Object)
	if !ok {
		return obj, nil
	}
	encrypted := make([]pdfparse.Object, len(items))
	for i, item := range items {
		enc, err := t.encryptValue(item, fileKey, depth+1)
		if err != nil {
			return obj, err
		}
		encrypted[i] = enc
	}
	return pdfparse.Arr(encrypted...), nil
}

// aes256CBCEncrypt encrypts plaintext with AES-256-CBC using a random IV
// prepended to the ciphertext.
//
// Takes key ([]byte) which is the 32-byte AES-256 encryption key.
// Takes plaintext ([]byte) which is the data to encrypt.
//
// Returns []byte which is the IV followed by the encrypted ciphertext.
// Returns error when cipher creation or IV generation fails.
func (t *EncryptTransformer) aes256CBCEncrypt(key, plaintext []byte) ([]byte, error) {
	padded := pkcs7Pad(plaintext, aesBlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}

	ciphertext := make([]byte, aesBlockSize+len(padded))
	iv := ciphertext[:aesBlockSize]
	if _, err := io.ReadFull(t.randomSource, iv); err != nil {
		return nil, fmt.Errorf("generating IV: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aesBlockSize:], padded)

	return ciphertext, nil
}

// aes256CBCEncryptZeroIV encrypts plaintext with AES-256-CBC using a
// zero IV for /UE and /OE computation per the PDF spec.
//
// Takes key ([]byte) which is the 32-byte AES-256 encryption key.
// Takes plaintext ([]byte) which is the data to encrypt.
//
// Returns []byte which is the encrypted ciphertext without IV prefix.
// Returns error when cipher creation fails.
func aes256CBCEncryptZeroIV(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}

	if len(plaintext)%aesBlockSize != 0 {
		return nil, fmt.Errorf("plaintext length %d is not a multiple of AES block size", len(plaintext))
	}

	iv := make([]byte, aesBlockSize)
	ciphertext := make([]byte, len(plaintext))

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext, nil
}

// permissionsToSigned reinterprets the uint32 permissions bitmask as a
// signed int64 value for the PDF /P entry.
//
// Takes p (uint32) which is the permissions bitmask.
//
// Returns int64 which is the signed 32-bit interpretation of the bitmask.
func permissionsToSigned(p uint32) int64 {
	return int64(int32(p)) //nolint:gosec // PDF spec requires signed 32-bit
}

// pkcs7Pad pads data to a multiple of blockSize using PKCS#7 padding.
//
// Takes data ([]byte) which is the input bytes to pad.
// Takes blockSize (int) which is the target alignment size in bytes.
//
// Returns []byte which is the padded data.
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	padByte := byte(padding) //nolint:gosec // blockSize is aesBlockSize (16)
	for i := len(data); i < len(padded); i++ {
		padded[i] = padByte
	}
	return padded
}
