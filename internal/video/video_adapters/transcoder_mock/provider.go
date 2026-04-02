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

package transcoder_mock

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"slices"
	"sync"
	"time"

	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/internal/video/video_dto"
)

const (
	// defaultMockDuration is the default video duration used in mock responses.
	defaultMockDuration = 30 * time.Second

	// defaultMockBitrate is the default bitrate in bits per second for mock video.
	defaultMockBitrate = 2000000

	// defaultMockWidth is the default video width in pixels for mock data.
	defaultMockWidth = 1920

	// defaultMockHeight is the default height in pixels for mock video
	// capabilities.
	defaultMockHeight = 1080

	// defaultMockFramerate is the default frame rate for mock video capabilities.
	defaultMockFramerate = 30.0

	// defaultMockCodec is the default video codec used in mock responses.
	defaultMockCodec = "h264"
)

var (
	// mockThumbnailJPEG is a minimal 1x1 pixel JPEG image used as placeholder
	// thumbnail data.
	//
	//go:embed mock_thumbnail.jpg
	mockThumbnailJPEG []byte

	_ video_domain.TranscoderPort = (*Provider)(nil)

	_ video_domain.StreamingTranscoderPort = (*Provider)(nil)
)

// TranscodeCall records the parameters passed to a single call of the
// Transcode method. It allows tests to inspect what the service is asking
// the transcoder to do.
type TranscodeCall struct {
	// InputData is a copy of the raw video bytes passed to the transcoder.
	InputData []byte

	// Spec is the transcoding specification that was requested.
	Spec video_dto.TranscodeSpec
}

// ExtractCapabilitiesCall records calls to ExtractCapabilities.
type ExtractCapabilitiesCall struct {
	// InputData is a copy of the raw video bytes passed to extract capabilities.
	InputData []byte
}

// TranscodeHLSCall records the arguments passed to a TranscodeHLS call.
type TranscodeHLSCall struct {
	// InputData is a copy of the raw video bytes passed to the transcoder.
	InputData []byte

	// Spec is the HLS specification that was requested.
	Spec video_dto.HLSSpec
}

// ExtractThumbnailCall records the arguments passed to ExtractThumbnail.
type ExtractThumbnailCall struct {
	// InputData is a copy of the raw video bytes used to extract the thumbnail.
	InputData []byte

	// Spec is the thumbnail specification that was requested.
	Spec video_dto.ThumbnailSpec
}

// Provider is a thread-safe, in-memory mock implementation of TranscoderPort
// and StreamingTranscoderPort. It is designed for unit and integration testing
// of the video service, allowing for call inspection and simulation of both
// successful transcodes and errors.
type Provider struct {
	// errToReturn is the error to return from Transform and Transcode calls.
	errToReturn error

	// transcodeCalls records Transcode calls for test assertions.
	transcodeCalls []TranscodeCall

	// extractCapabilitiesCalls records all calls made to ExtractCapabilities.
	extractCapabilitiesCalls []ExtractCapabilitiesCall

	// transcodeHLSCalls records all HLS transcoding calls made to the mock
	// provider.
	transcodeHLSCalls []TranscodeHLSCall

	// extractThumbnailCalls records all calls made to ExtractThumbnail for test
	// verification.
	extractThumbnailCalls []ExtractThumbnailCall

	// outputDataToReturn holds the byte data to write to output during
	// Transform or Transcode calls; nil means no data is returned.
	outputDataToReturn []byte

	// thumbnailDataToReturn holds the byte data to return from ExtractThumbnail;
	// nil means use default behaviour.
	thumbnailDataToReturn []byte

	// supportedCodecs lists the video codecs this provider can process.
	supportedCodecs []string

	// hlsResultToReturn is the HLS result to return from TranscodeHLS;
	// empty MasterPlaylistContent means use default behaviour.
	hlsResultToReturn video_dto.HLSResult

	// capabilitiesToReturn holds the video capabilities to return from
	// ExtractCapabilities; empty VideoCodec means use actual extraction.
	capabilitiesToReturn video_dto.VideoCapabilities

	// slowTranscodeDuration is the delay duration for simulated slow transcodes.
	slowTranscodeDuration time.Duration

	// mu guards concurrent access to Provider's mutable fields.
	mu sync.RWMutex

	// simulateSlowTranscode enables artificial delay during transcoding for
	// testing.
	simulateSlowTranscode bool
}

// NewProvider creates a new, initialised mock video transcoder.
//
// Returns *Provider which is ready to use with default test values.
func NewProvider() *Provider {
	return &Provider{
		errToReturn:              nil,
		transcodeCalls:           make([]TranscodeCall, 0),
		extractCapabilitiesCalls: make([]ExtractCapabilitiesCall, 0),
		transcodeHLSCalls:        make([]TranscodeHLSCall, 0),
		extractThumbnailCalls:    make([]ExtractThumbnailCall, 0),
		outputDataToReturn:       nil,
		thumbnailDataToReturn:    nil,
		supportedCodecs:          []string{defaultMockCodec, "h265", "vp9"},
		hlsResultToReturn:        video_dto.HLSResult{},
		capabilitiesToReturn:     video_dto.VideoCapabilities{},
		slowTranscodeDuration:    0,
		mu:                       sync.RWMutex{},
		simulateSlowTranscode:    false,
	}
}

// Transcode simulates a video transcode operation. It records the call and
// returns either a configured error/result or, by default, passes the input
// through to output.
//
// Takes input (io.Reader) which provides the video data to transcode.
// Takes spec (video_dto.TranscodeSpec) which specifies the transcode settings.
//
// Returns io.ReadCloser which provides the transcoded output data.
// Returns error when reading input fails or when configured to return an error.
//
// Safe for concurrent use; protects internal state with a mutex.
func (p *Provider) Transcode(_ context.Context, input io.Reader, spec video_dto.TranscodeSpec) (io.ReadCloser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inBytes, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	dataCopy := make([]byte, len(inBytes))
	copy(dataCopy, inBytes)
	p.transcodeCalls = append(p.transcodeCalls, TranscodeCall{
		InputData: dataCopy,
		Spec:      spec,
	})

	if p.simulateSlowTranscode {
		time.Sleep(p.slowTranscodeDuration)
	}

	if p.errToReturn != nil {
		return nil, p.errToReturn
	}

	if p.outputDataToReturn != nil {
		return io.NopCloser(bytes.NewReader(p.outputDataToReturn)), nil
	}

	return io.NopCloser(bytes.NewReader(dataCopy)), nil
}

// ExtractCapabilities simulates video capability extraction.
// Returns configured capabilities or sensible defaults for testing.
//
// Takes input (io.Reader) which provides the video data to analyse.
//
// Returns video_dto.VideoCapabilities which contains the extracted or mock
// capabilities.
// Returns error when reading input fails or a configured error is set.
//
// Safe for concurrent use; protects internal state with a mutex.
func (p *Provider) ExtractCapabilities(_ context.Context, input io.Reader) (video_dto.VideoCapabilities, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inBytes, err := io.ReadAll(input)
	if err != nil {
		return video_dto.VideoCapabilities{}, err
	}

	dataCopy := make([]byte, len(inBytes))
	copy(dataCopy, inBytes)
	p.extractCapabilitiesCalls = append(p.extractCapabilitiesCalls, ExtractCapabilitiesCall{
		InputData: dataCopy,
	})

	if p.errToReturn != nil {
		return video_dto.VideoCapabilities{}, p.errToReturn
	}

	if p.capabilitiesToReturn.VideoCodec != "" {
		return p.capabilitiesToReturn, nil
	}

	return video_dto.VideoCapabilities{
		Duration:         defaultMockDuration,
		Bitrate:          defaultMockBitrate,
		FileSize:         int64(len(inBytes)),
		Format:           "mov",
		Width:            defaultMockWidth,
		Height:           defaultMockHeight,
		Framerate:        defaultMockFramerate,
		VideoCodec:       defaultMockCodec,
		AudioCodec:       "aac",
		SupportedCodecs:  p.supportedCodecs,
		SupportedFormats: []string{"mp4", "webm", "mkv"},
		SupportsHLS:      true,
		SupportsDASH:     false,
	}, nil
}

// SupportsCodec checks if a codec is supported by the mock transcoder.
//
// Takes codec (string) which is the codec name to check.
//
// Returns bool which is true if the codec is supported.
//
// Safe for concurrent use.
func (p *Provider) SupportsCodec(codec string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return slices.Contains(p.supportedCodecs, codec)
}

// TranscodeHLS simulates HLS transcode operation.
//
// Takes input (io.Reader) which provides the video data to transcode.
// Takes spec (video_dto.HLSSpec) which defines the HLS output settings.
//
// Returns video_dto.HLSResult which contains the mock HLS output data.
// Returns error when reading input fails or an error has been configured.
//
// Safe for concurrent use. The method is protected by a mutex.
func (p *Provider) TranscodeHLS(_ context.Context, input io.Reader, spec video_dto.HLSSpec) (video_dto.HLSResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inBytes, err := io.ReadAll(input)
	if err != nil {
		return video_dto.HLSResult{}, err
	}

	dataCopy := make([]byte, len(inBytes))
	copy(dataCopy, inBytes)
	p.transcodeHLSCalls = append(p.transcodeHLSCalls, TranscodeHLSCall{
		InputData: dataCopy,
		Spec:      spec,
	})

	if p.errToReturn != nil {
		return video_dto.HLSResult{}, p.errToReturn
	}

	if p.hlsResultToReturn.MasterPlaylistContent != "" {
		return p.hlsResultToReturn, nil
	}

	return video_dto.HLSResult{
		MasterPlaylist:        io.NopCloser(bytes.NewReader([]byte("#EXTM3U\n#EXT-X-VERSION:3\n"))),
		MasterPlaylistContent: "#EXTM3U\n#EXT-X-VERSION:3\n",
		Variants: []video_dto.HLSVariant{
			{
				Playlist:        io.NopCloser(bytes.NewReader([]byte("#EXTM3U\n"))),
				PlaylistContent: "#EXTM3U\n",
				Segments:        []video_dto.HLSSegment{},
				Bitrate:         defaultMockBitrate,
				Resolution:      "1920x1080",
				Codec:           defaultMockCodec,
			},
		},
		TotalSegments: 0,
		TotalDuration: defaultMockDuration,
	}, nil
}

// ExtractThumbnail simulates thumbnail extraction from a video.
// Returns configured thumbnail data or a minimal placeholder image.
//
// Takes input (io.Reader) which provides the video data to extract from.
// Takes spec (video_dto.ThumbnailSpec) which specifies the thumbnail settings.
//
// Returns io.ReadCloser which contains the thumbnail image data.
// Returns error when reading the input fails or an error has been configured.
//
// Safe for concurrent use; protects internal state with a mutex.
func (p *Provider) ExtractThumbnail(_ context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inBytes, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	dataCopy := make([]byte, len(inBytes))
	copy(dataCopy, inBytes)
	p.extractThumbnailCalls = append(p.extractThumbnailCalls, ExtractThumbnailCall{
		InputData: dataCopy,
		Spec:      spec,
	})

	if p.errToReturn != nil {
		return nil, p.errToReturn
	}

	if p.thumbnailDataToReturn != nil {
		return io.NopCloser(bytes.NewReader(p.thumbnailDataToReturn)), nil
	}

	return io.NopCloser(bytes.NewReader(mockThumbnailJPEG)), nil
}

// SetError configures the mock to return the specified error on the next call.
//
// Takes err (error) which is the error to return, or nil to clear any error.
//
// Safe for concurrent use.
func (p *Provider) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errToReturn = err
}

// SetTranscodeResult configures the mock to return specific transcoded output
// data on the next Transcode call.
//
// Takes data ([]byte) which specifies the output data to return.
//
// Safe for concurrent use.
func (p *Provider) SetTranscodeResult(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.outputDataToReturn = data
}

// SetCapabilitiesResult configures the mock to return specific capabilities
// on the next ExtractCapabilities call.
//
// Takes caps (video_dto.VideoCapabilities) which specifies the capabilities
// to return.
//
// Safe for concurrent use.
func (p *Provider) SetCapabilitiesResult(caps video_dto.VideoCapabilities) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.capabilitiesToReturn = caps
}

// SetHLSResult configures the mock to return a specific HLS result on the next
// TranscodeHLS call.
//
// Takes result (video_dto.HLSResult) which specifies the HLS result to return.
//
// Safe for concurrent use.
func (p *Provider) SetHLSResult(result video_dto.HLSResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hlsResultToReturn = result
}

// SetThumbnailResult configures the mock to return specific thumbnail data
// on the next ExtractThumbnail call.
//
// Takes data ([]byte) which specifies the thumbnail data to return.
//
// Safe for concurrent use.
func (p *Provider) SetThumbnailResult(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.thumbnailDataToReturn = data
}

// SetSupportedCodecs configures which codecs the mock reports as supported.
//
// Takes codecs ([]string) which specifies the list of codec names to report.
//
// Safe for concurrent use.
func (p *Provider) SetSupportedCodecs(codecs []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.supportedCodecs = codecs
}

// SimulateSlowTranscode configures the mock to sleep for the specified duration
// during Transcode calls, simulating a slow transcode operation.
//
// Takes duration (time.Duration) which specifies how long Transcode should
// sleep.
//
// Safe for concurrent use.
func (p *Provider) SimulateSlowTranscode(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.simulateSlowTranscode = true
	p.slowTranscodeDuration = duration
}

// GetTranscodeCalls returns a copy of all recorded calls made to the
// Transcode method. Returning a copy prevents the test from modifying the
// mock's internal state.
//
// Returns []TranscodeCall which contains copies of all recorded transcode
// calls.
//
// Safe for concurrent use; uses a read lock to protect access.
func (p *Provider) GetTranscodeCalls() []TranscodeCall {
	p.mu.RLock()
	defer p.mu.RUnlock()

	callsCopy := make([]TranscodeCall, len(p.transcodeCalls))
	copy(callsCopy, p.transcodeCalls)
	return callsCopy
}

// GetExtractCapabilitiesCalls returns a copy of all recorded
// ExtractCapabilities calls.
//
// Returns []ExtractCapabilitiesCall which contains a copy of all recorded
// calls.
//
// Safe for concurrent use; protected by a read lock.
func (p *Provider) GetExtractCapabilitiesCalls() []ExtractCapabilitiesCall {
	p.mu.RLock()
	defer p.mu.RUnlock()

	callsCopy := make([]ExtractCapabilitiesCall, len(p.extractCapabilitiesCalls))
	copy(callsCopy, p.extractCapabilitiesCalls)
	return callsCopy
}

// GetTranscodeHLSCalls returns a copy of all recorded TranscodeHLS calls.
//
// Returns []TranscodeHLSCall which contains a snapshot of all recorded calls.
//
// Safe for concurrent use; acquires a read lock during access.
func (p *Provider) GetTranscodeHLSCalls() []TranscodeHLSCall {
	p.mu.RLock()
	defer p.mu.RUnlock()

	callsCopy := make([]TranscodeHLSCall, len(p.transcodeHLSCalls))
	copy(callsCopy, p.transcodeHLSCalls)
	return callsCopy
}

// GetExtractThumbnailCalls returns a copy of all recorded ExtractThumbnail
// calls.
//
// Returns []ExtractThumbnailCall which contains a copy of all recorded calls.
//
// Safe for concurrent use. Uses a read lock to protect access to the call
// history.
func (p *Provider) GetExtractThumbnailCalls() []ExtractThumbnailCall {
	p.mu.RLock()
	defer p.mu.RUnlock()

	callsCopy := make([]ExtractThumbnailCall, len(p.extractThumbnailCalls))
	copy(callsCopy, p.extractThumbnailCalls)
	return callsCopy
}

// GetCallCounts returns the number of calls made to each method.
// Useful for quick assertions without inspecting full call details.
//
// Returns transcode (int) which is the number of Transcode calls.
// Returns extractCaps (int) which is the number of ExtractCapabilities calls.
// Returns hlsTranscode (int) which is the number of TranscodeHLS calls.
// Returns extractThumbnail (int) which is the number of ExtractThumbnail calls.
//
// Safe for concurrent use; protected by a read lock.
func (p *Provider) GetCallCounts() (transcode, extractCaps, hlsTranscode, extractThumbnail int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.transcodeCalls), len(p.extractCapabilitiesCalls), len(p.transcodeHLSCalls), len(p.extractThumbnailCalls)
}

// Reset clears all recorded calls and configured return values/errors,
// preparing the mock for a new, isolated test case.
//
// Safe for concurrent use.
func (p *Provider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.transcodeCalls = make([]TranscodeCall, 0)
	p.extractCapabilitiesCalls = make([]ExtractCapabilitiesCall, 0)
	p.transcodeHLSCalls = make([]TranscodeHLSCall, 0)
	p.extractThumbnailCalls = make([]ExtractThumbnailCall, 0)
	p.outputDataToReturn = nil
	p.thumbnailDataToReturn = nil
	p.capabilitiesToReturn = video_dto.VideoCapabilities{}
	p.hlsResultToReturn = video_dto.HLSResult{}
	p.errToReturn = nil
	p.supportedCodecs = []string{"h264", "h265", "vp9"}
	p.simulateSlowTranscode = false
	p.slowTranscodeDuration = 0
}
