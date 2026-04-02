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

package local_aes_gcm

import (
	"bytes"
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// chunkLengthBytes is the number of bytes used to store a chunk length prefix.
	chunkLengthBytes = 4

	// maxChunkSizeBytes is the maximum allowed chunk size (10MB) for sanity checking.
	maxChunkSizeBytes = 10 * 1024 * 1024

	// maxHeaderSizeBytes is the largest header size allowed, which is one megabyte.
	maxHeaderSizeBytes = 1024 * 1024
)

// encryptingWriter implements io.WriteCloser for streaming encryption.
// It chunks the plaintext, encrypts each chunk with a unique IV, and writes
// the encrypted chunks to the destination writer.
//
// Memory usage: O(chunk_size) - typically 64KB, regardless of total data size.
type encryptingWriter struct {
	// dst is the writer where encrypted chunks are written.
	dst io.Writer

	// aead is the AES-GCM cipher that encrypts chunks.
	aead cipher.AEAD

	// baseIV is the 12-byte base initialisation vector for the stream. It is
	// combined with chunk numbers to create a unique IV for each chunk.
	baseIV []byte

	// buffer holds plaintext data until a full chunk is ready for encryption.
	buffer []byte

	// position is the current write position in the buffer.
	position int

	// chunkNum is the current chunk number used to derive the IV.
	chunkNum uint32

	// closed indicates whether the writer has been closed.
	closed bool

	// chunkSize is the size of each plaintext chunk in bytes.
	chunkSize int
}

// Write accumulates plaintext data and encrypts it in chunks.
//
// Takes p ([]byte) which contains the plaintext data to encrypt.
//
// Returns n (int) which is the number of bytes consumed from p.
// Returns err (error) when flushing an encrypted chunk fails.
func (w *encryptingWriter) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, errors.New("write to closed encryptingWriter")
	}

	n = len(p)
	offset := 0

	for offset < len(p) {
		copied := copy(w.buffer[w.position:], p[offset:])
		w.position += copied
		offset += copied

		if w.position == w.chunkSize {
			if err := w.flushChunk(); err != nil {
				return offset, fmt.Errorf("flushing chunk: %w", err)
			}
		}
	}

	return n, nil
}

// Close flushes any remaining data and completes the stream.
//
// Returns error when flushing the final chunk fails.
func (w *encryptingWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	if w.position > 0 {
		if err := w.flushChunk(); err != nil {
			return fmt.Errorf("flushing final chunk: %w", err)
		}
	}

	return nil
}

// flushChunk encrypts the current buffer contents and writes them to the
// output.
//
// Returns error when writing the chunk length or encrypted data fails.
func (w *encryptingWriter) flushChunk() error {
	iv := deriveIV(w.baseIV, w.chunkNum)

	plaintext := w.buffer[:w.position]
	//nolint:gosec // IV from crypto/rand + counter
	ciphertext := w.aead.Seal(nil, iv, plaintext, nil)

	chunkLen := make([]byte, chunkLengthBytes)
	binary.BigEndian.PutUint32(chunkLen, safeconv.IntToUint32(len(ciphertext)))

	if _, err := w.dst.Write(chunkLen); err != nil {
		return fmt.Errorf("failed to write chunk length: %w", err)
	}

	if _, err := w.dst.Write(ciphertext); err != nil {
		return fmt.Errorf("failed to write encrypted chunk: %w", err)
	}

	w.position = 0

	if w.chunkNum == math.MaxUint32 {
		return errors.New("chunk counter overflow: stream exceeds maximum of 2^32 chunks")
	}
	w.chunkNum++

	return nil
}

// decryptingReader implements io.ReadCloser for streaming decryption.
// It reads encrypted chunks, decrypts them with the appropriate IV, and
// returns the plaintext to the caller.
//
// Memory usage: O(chunk_size) - typically 64KB, regardless of total data size.
type decryptingReader struct {
	// src is the source of encrypted data to decrypt.
	src io.Reader

	// aead is the cipher used to decrypt and verify each data chunk.
	aead cipher.AEAD

	// buffer holds decrypted data that is waiting to be read.
	buffer *bytes.Buffer

	// baseIV is the starting value used to create a unique IV for each chunk.
	baseIV []byte

	// chunkNum is the current chunk number used to derive the IV for each chunk.
	chunkNum uint32

	// eof indicates whether the end of the encrypted stream has been reached.
	eof bool

	// closed indicates whether the reader has been closed.
	closed bool
}

// Read decrypts data and returns it to the caller.
//
// Takes p ([]byte) which is the buffer to fill with decrypted data.
//
// Returns n (int) which is the number of bytes written to p.
// Returns err (error) when decryption fails or the stream ends
// (io.EOF).
func (r *decryptingReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, errors.New("read from closed decryptingReader")
	}

	if r.buffer.Len() > 0 {
		return r.buffer.Read(p)
	}

	if r.eof {
		return 0, io.EOF
	}

	if err := r.readChunk(); err != nil {
		if errors.Is(err, io.EOF) {
			r.eof = true
			if r.buffer.Len() > 0 {
				return r.buffer.Read(p)
			}
			return 0, io.EOF
		}
		return 0, fmt.Errorf("reading chunk: %w", err)
	}

	return r.buffer.Read(p)
}

// Close releases resources held by the reader.
//
// Returns error when the underlying reader cannot be closed.
func (r *decryptingReader) Close() error {
	r.closed = true
	return nil
}

// readChunk reads one encrypted chunk, decrypts it, and stores it in the
// buffer.
//
// Returns error when the chunk length is invalid, reading fails, or decryption
// fails.
func (r *decryptingReader) readChunk() error {
	chunkLenBytes := make([]byte, chunkLengthBytes)
	if _, err := io.ReadFull(r.src, chunkLenBytes); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return io.EOF
		}
		return fmt.Errorf("reading chunk length: %w", err)
	}

	chunkLen := binary.BigEndian.Uint32(chunkLenBytes)
	if chunkLen == 0 || chunkLen > maxChunkSizeBytes {
		return fmt.Errorf("invalid chunk length: %d", chunkLen)
	}

	ciphertext := make([]byte, chunkLen)
	if _, err := io.ReadFull(r.src, ciphertext); err != nil {
		return fmt.Errorf("failed to read encrypted chunk: %w", err)
	}

	iv := deriveIV(r.baseIV, r.chunkNum)

	plaintext, err := r.aead.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt chunk %d: %w", r.chunkNum, err)
	}

	_, _ = r.buffer.Write(plaintext)

	if r.chunkNum == math.MaxUint32 {
		return errors.New("chunk counter overflow: stream exceeds maximum of 2^32 chunks")
	}
	r.chunkNum++

	return nil
}

// EncryptStream implements streaming encryption for the local AES-GCM provider.
//
// Takes output (io.Writer) which receives the encrypted data stream.
//
// Returns io.WriteCloser which encrypts data written to it using
// AES-256-GCM with a freshly generated IV.
// Returns error when IV generation or header writing fails.
func (p *provider) EncryptStream(_ context.Context, output io.Writer, _ *crypto_dto.EncryptRequest) (io.WriteCloser, error) {
	baseIV := make([]byte, IVSize)
	if _, err := io.ReadFull(rand.Reader, baseIV); err != nil {
		return nil, fmt.Errorf("failed to generate base IV: %w", err)
	}

	header := &crypto_dto.StreamingHeader{
		Version:   2,
		KeyID:     p.keyID,
		Provider:  string(p.Type()),
		IV:        base64.StdEncoding.EncodeToString(baseIV),
		Algorithm: "AES-256-GCM",
	}

	if err := WriteStreamingHeader(output, header); err != nil {
		return nil, fmt.Errorf("writing streaming header: %w", err)
	}

	return &encryptingWriter{
		dst:       output,
		aead:      p.aead,
		baseIV:    baseIV,
		buffer:    make([]byte, crypto_dto.DefaultChunkSize),
		position:  0,
		chunkNum:  0,
		closed:    false,
		chunkSize: crypto_dto.DefaultChunkSize,
	}, nil
}

// DecryptStream reads encrypted data and returns a reader for the plain text.
//
// Takes input (io.Reader) which provides the encrypted data stream.
//
// Returns io.ReadCloser which yields the decrypted data as it is read.
// Returns error when the stream header is malformed or the IV is invalid.
func (p *provider) DecryptStream(_ context.Context, input io.Reader) (io.ReadCloser, error) {
	header, err := ReadStreamingHeader(input)
	if err != nil {
		return nil, fmt.Errorf("reading streaming header: %w", err)
	}

	baseIV, err := base64.StdEncoding.DecodeString(header.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	if len(baseIV) != IVSize {
		return nil, fmt.Errorf("invalid IV length: expected %d, got %d", IVSize, len(baseIV))
	}

	return &decryptingReader{
		src:      input,
		aead:     p.aead,
		baseIV:   baseIV,
		buffer:   new(bytes.Buffer),
		chunkNum: 0,
		eof:      false,
		closed:   false,
	}, nil
}

// WriteStreamingHeader writes the v2 streaming envelope header to the output.
// Format: [Version (1 byte)] [Header Length (4 bytes)] [JSON Header].
//
// This function is exported so that other providers (AWS KMS, GCP KMS) can
// reuse the same envelope format for consistency.
//
// Takes output (io.Writer) which receives the encoded header bytes.
// Takes header (*crypto_dto.StreamingHeader) which contains the envelope
// metadata to encode.
//
// Returns error when the header cannot be marshalled to JSON or when writing
// to the output fails.
func WriteStreamingHeader(output io.Writer, header *crypto_dto.StreamingHeader) error {
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("failed to marshal streaming header: %w", err)
	}

	if _, err := output.Write([]byte{crypto_dto.StreamingEnvelopeVersion}); err != nil {
		return fmt.Errorf("failed to write version byte: %w", err)
	}

	headerLen := make([]byte, chunkLengthBytes)
	binary.BigEndian.PutUint32(headerLen, safeconv.IntToUint32(len(headerBytes)))
	if _, err := output.Write(headerLen); err != nil {
		return fmt.Errorf("failed to write header length: %w", err)
	}

	if _, err := output.Write(headerBytes); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	return nil
}

// ReadStreamingHeader reads and parses the v2 streaming envelope header.
//
// This function is exported so that other providers (AWS KMS, GCP KMS) can
// reuse the same envelope format for consistency.
//
// Takes input (io.Reader) which provides the encrypted stream to read from.
//
// Returns *crypto_dto.StreamingHeader which contains the parsed header data.
// Returns error when the format is invalid, the version is unsupported, or
// the header cannot be read.
func ReadStreamingHeader(input io.Reader) (*crypto_dto.StreamingHeader, error) {
	versionByte := make([]byte, 1)
	if _, err := io.ReadFull(input, versionByte); err != nil {
		return nil, fmt.Errorf("failed to read version byte: %w", err)
	}

	version := versionByte[0]
	if version != crypto_dto.StreamingEnvelopeVersion {
		return nil, fmt.Errorf("unsupported ciphertext version: expected %d, got %d",
			crypto_dto.StreamingEnvelopeVersion, version)
	}

	headerLenBytes := make([]byte, chunkLengthBytes)
	if _, err := io.ReadFull(input, headerLenBytes); err != nil {
		return nil, fmt.Errorf("failed to read header length: %w", err)
	}

	headerLen := binary.BigEndian.Uint32(headerLenBytes)
	if headerLen == 0 || headerLen > maxHeaderSizeBytes {
		return nil, fmt.Errorf("invalid header length: %d", headerLen)
	}

	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(input, headerBytes); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var header crypto_dto.StreamingHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to parse streaming header: %w", err)
	}

	return &header, nil
}

// NewEncryptingWriter creates a new encrypting writer for streaming
// encryption. This is exported to allow other providers (AWS KMS, GCP KMS)
// to use local AES-GCM streaming with their own envelope encryption.
//
// Takes dst (io.Writer) which receives the encrypted output.
// Takes aead (cipher.AEAD) which is the AES-GCM cipher instance.
// Takes baseIV ([]byte) which is the base IV for the stream (12 bytes).
// Takes chunkSize (int) which is the size of each plaintext chunk.
//
// Returns io.WriteCloser which encrypts data written to it.
//
// Panics if baseIV length does not equal IVSize.
func NewEncryptingWriter(dst io.Writer, aead cipher.AEAD, baseIV []byte, chunkSize int) io.WriteCloser {
	if len(baseIV) != IVSize {
		panic(fmt.Sprintf("invalid base IV length: expected %d, got %d", IVSize, len(baseIV)))
	}
	return &encryptingWriter{
		dst:       dst,
		aead:      aead,
		baseIV:    baseIV,
		buffer:    make([]byte, chunkSize),
		position:  0,
		chunkNum:  0,
		closed:    false,
		chunkSize: chunkSize,
	}
}

// NewDecryptingReader creates a new decrypting reader for streaming
// decryption. This is exported to allow other providers (AWS KMS, GCP KMS)
// to use local AES-GCM streaming with their own envelope encryption.
//
// Takes src (io.Reader) which provides the encrypted input stream.
// Takes aead (cipher.AEAD) which is the AES-GCM cipher instance.
// Takes baseIV ([]byte) which is the base IV for the stream (12 bytes).
//
// Returns io.ReadCloser which decrypts data as it is read.
//
// Panics if baseIV is not exactly IVSize (12) bytes.
func NewDecryptingReader(src io.Reader, aead cipher.AEAD, baseIV []byte) io.ReadCloser {
	if len(baseIV) != IVSize {
		panic(fmt.Sprintf("invalid base IV length: expected %d, got %d", IVSize, len(baseIV)))
	}
	return &decryptingReader{
		src:      src,
		aead:     aead,
		baseIV:   baseIV,
		buffer:   new(bytes.Buffer),
		chunkNum: 0,
		eof:      false,
		closed:   false,
	}
}

// GenerateIV creates a random initialisation vector for AES-GCM encryption.
// This is exported so other providers can create IVs using the same method.
//
// Returns []byte which contains the generated IV of IVSize bytes.
// Returns error when reading from the random source fails.
func GenerateIV() ([]byte, error) {
	iv := make([]byte, IVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}
	return iv, nil
}

// deriveIV creates a unique IV for a given chunk by combining the base IV
// with the chunk number. Each chunk needs a unique IV for AES-GCM security.
//
// Format: baseIV[0:8] || chunkNum (as 4 bytes, big-endian)
//
// The scheme supports up to 2^32 chunks per stream (up to 256TB with 64KB chunks).
//
// Takes baseIV ([]byte) which provides the base IV to derive from.
// Takes chunkNum (uint32) which specifies the chunk index.
//
// Returns []byte which is the derived IV unique to this chunk.
//
// Panics if baseIV length does not equal IVSize.
func deriveIV(baseIV []byte, chunkNum uint32) []byte {
	if len(baseIV) != IVSize {
		panic(fmt.Sprintf("invalid base IV length: expected %d, got %d", IVSize, len(baseIV)))
	}

	iv := make([]byte, IVSize)
	copy(iv, baseIV[:8])
	binary.BigEndian.PutUint32(iv[8:], chunkNum)

	return iv
}
