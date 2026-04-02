//go:build ffmpeg

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

package video_provider_astiav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/asticode/go-astiav"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/media"
)

// Config holds configuration for the astiav video provider.
type Config struct {
	media.VideoServiceConfig

	// MaxConcurrentTranscodes sets the maximum number of transcode operations
	// that can run at the same time. Defaults to 10.
	MaxConcurrentTranscodes int

	// EnableHWAccel enables hardware acceleration when supported.
	EnableHWAccel bool
}

// Provider implements VideoTranscoderPort and StreamingTranscoderPort using
// FFmpeg through go-astiav bindings.
type Provider struct {
	// codecRegistry provides codec lookup for encoding and decoding.
	codecRegistry *CodecRegistry

	// semaphore limits concurrent transcoding operations.
	semaphore chan struct{}

	// maxConcurrent is the maximum number of concurrent transcoding operations.
	maxConcurrent int
}

var _ media.VideoTranscoderPort = (*Provider)(nil)
var _ media.StreamingTranscoderPort = (*Provider)(nil)

var errNoFramesInVideo = errors.New("no frames in video")

// NewProvider creates a new FFmpeg-based video transcoding provider.
//
// Takes config (Config) which sets the provider's behaviour.
//
// Returns *Provider which is the configured provider ready for use.
// Returns error when setup fails.
func NewProvider(config Config) (*Provider, error) {
	maxConcurrent := config.MaxConcurrentTranscodes
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrent
	}

	p := &Provider{
		codecRegistry: NewCodecRegistry(),
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}

	_, l := logger.From(context.Background(), log)
	l.Internal("Initialised astiav video provider",
		logger.Int("maxConcurrent", p.maxConcurrent),
		logger.String("supportedCodecs", fmt.Sprintf("%v", p.codecRegistry.SupportedCodecs())),
	)

	return p, nil
}

// Transcode converts video from one format/codec to another.
// Delegates the actual transcoding work to startTranscodeGoroutine
// after validating the spec and acquiring a semaphore slot.
//
// Takes input (io.Reader) which provides the source video data.
// Takes spec (TranscodeSpec) which defines the output format,
// codec, and encoding parameters.
//
// Returns io.ReadCloser which streams the transcoded video data.
// The caller must close this reader when finished.
// Returns error when the context is cancelled before starting,
// when defaults cannot be applied, or when the spec is invalid.
func (p *Provider) Transcode(ctx context.Context, input io.Reader, spec media.TranscodeSpec) (io.ReadCloser, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "Provider.Transcode",
		logger.String("codec", spec.Codec),
		logger.Int("width", spec.Width),
		logger.Int("height", spec.Height),
		logger.String("preset", spec.Preset),
	)
	defer span.End()

	if err := p.acquireSemaphore(ctx); err != nil {
		l.Warn("Transcode cancelled before starting")
		return nil, err
	}
	defer func() { <-p.semaphore }()

	ActiveTranscodes.Add(ctx, 1)
	defer ActiveTranscodes.Add(ctx, -1)

	if err := p.validateTranscodeSpec(ctx, &spec); err != nil {
		l.ReportError(span, err, "Transcode spec validation failed")
		return nil, err
	}

	return p.startTranscodeGoroutine(ctx, input, spec, l)
}

// ExtractCapabilities analyses input video and returns metadata.
//
// Takes input (io.Reader) which provides the video data to analyse.
//
// Returns media.VideoCapabilities which contains the extracted metadata
// including dimensions, codec, duration, bitrate, and framerate.
// Returns error when the input format context cannot be created or no video
// stream is found.
func (p *Provider) ExtractCapabilities(ctx context.Context, input io.Reader) (media.VideoCapabilities, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "Provider.ExtractCapabilities")
	defer span.End()

	caps := media.VideoCapabilities{
		SupportedCodecs:  p.codecRegistry.SupportedCodecs(),
		SupportedFormats: []string{"mp4", "webm", "mkv"},
		SupportsHLS:      true,
		SupportsDASH:     false,
	}

	inputCtx, err := p.createInputFormatContext(ctx, input)
	if err != nil {
		l.ReportError(span, err, "Failed to create input format context")
		return caps, err
	}
	defer inputCtx.Free()

	videoStreamIndex := findFirstVideoStreamIndex(inputCtx)
	if videoStreamIndex == -1 {
		return caps, media.ErrInvalidStream
	}

	videoStream := inputCtx.Streams()[videoStreamIndex]
	codecParams := videoStream.CodecParameters()

	caps.Width = codecParams.Width()
	caps.Height = codecParams.Height()
	caps.VideoCodec = codecParams.CodecID().Name()

	if videoStream.Duration() != astiav.NoPtsValue {
		timeBase := videoStream.TimeBase()
		durationSeconds := float64(videoStream.Duration()) * timeBase.Float64()
		caps.Duration = time.Duration(durationSeconds * float64(time.Second))
	}

	if codecParams.BitRate() > 0 {
		caps.Bitrate = codecParams.BitRate()
	}

	frameRate := videoStream.AvgFrameRate()
	if frameRate.Num() > 0 && frameRate.Den() > 0 {
		caps.Framerate = float64(frameRate.Num()) / float64(frameRate.Den())
	}

	return caps, nil
}

// SupportsCodec checks if a codec is supported.
//
// Takes codec (string) which is the codec name to check.
//
// Returns bool which is true if the codec is supported.
func (p *Provider) SupportsCodec(codec string) bool {
	return p.codecRegistry.SupportsCodec(codec)
}

// TranscodeHLS generates HLS manifest and segments.
// This will be implemented in Phase 3.
//
// Returns media.HLSResult which contains the manifest and segment paths.
// Returns error when the transcoding fails.
func (*Provider) TranscodeHLS(ctx context.Context, _ io.Reader, _ media.HLSSpec) (media.HLSResult, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "Provider.TranscodeHLS")
	defer span.End()

	l.Warn("HLS transcoding not yet implemented (Phase 3)")
	return media.HLSResult{}, errors.New("HLS transcoding not yet implemented")
}

// ExtractThumbnail extracts a single frame from a video and encodes it as an
// image.
//
// Takes input (io.Reader) which provides the source video data.
// Takes spec (ThumbnailSpec) which defines the extraction parameters.
//
// Returns io.ReadCloser which streams the encoded image data.
// Returns error when extraction fails.
func (p *Provider) ExtractThumbnail(ctx context.Context, input io.Reader, spec media.ThumbnailSpec) (io.ReadCloser, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "Provider.ExtractThumbnail",
		logger.String("format", spec.Format),
		logger.Int("width", spec.Width),
		logger.Int("height", spec.Height),
	)
	defer span.End()

	if err := p.acquireSemaphore(ctx); err != nil {
		l.Warn("Thumbnail extraction cancelled before starting")
		return nil, err
	}
	defer func() { <-p.semaphore }()

	inputCtx, err := p.createInputFormatContext(ctx, input)
	if err != nil {
		l.ReportError(span, err, "Failed to create input format context")
		return nil, err
	}
	defer inputCtx.Free()

	frame, err := p.decodeThumbnailFrame(ctx, inputCtx, spec)
	if err != nil {
		l.ReportError(span, err, "Failed to decode thumbnail frame")
		return nil, err
	}
	defer frame.Free()

	return p.scaleThumbnailAndEncode(ctx, frame, spec)
}

// acquireSemaphore attempts to acquire a transcode semaphore
// slot, blocking until a slot is available or the context is
// cancelled.
//
// Returns error when the context is cancelled before a slot
// becomes available.
func (p *Provider) acquireSemaphore(ctx context.Context) error {
	select {
	case p.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return media.ErrContextCancelled
	}
}

// validateTranscodeSpec applies codec defaults and validates
// the transcode specification.
//
// Takes spec (*media.TranscodeSpec) which is the specification
// to validate and populate with defaults.
//
// Returns error when defaults cannot be applied or the spec
// is invalid.
func (p *Provider) validateTranscodeSpec(ctx context.Context, spec *media.TranscodeSpec) error {
	if err := p.codecRegistry.ApplyDefaults(spec); err != nil {
		return err
	}

	if err := p.codecRegistry.ValidateSpec(spec); err != nil {
		TranscodeErrorCount.Add(ctx, 1)
		return err
	}

	return nil
}

// startTranscodeGoroutine spawns the asynchronous transcode
// pipeline and returns a reader for consuming the output.
//
// Takes input (io.Reader) which provides the source video data.
// Takes spec (media.TranscodeSpec) which defines the transcode
// settings.
// Takes l (logger.Logger) which is used for logging within the
// spawned goroutine.
//
// Returns io.ReadCloser which streams the transcoded output.
// Returns error when the pipe cannot be created.
//
// Safe for concurrent use; the spawned goroutine writes to an
// io.Pipe independently of the caller.
func (p *Provider) startTranscodeGoroutine(
	ctx context.Context, input io.Reader, spec media.TranscodeSpec, l logger.Logger,
) (io.ReadCloser, error) {
	startTime := time.Now()
	outputReader, outputWriter := io.Pipe()

	go func() {
		defer outputWriter.Close()

		if err := p.runTranscode(ctx, input, outputWriter, spec); err != nil {
			l.Error("Transcoding failed", logger.Error(err))
			TranscodeErrorCount.Add(ctx, 1)
			_ = outputWriter.CloseWithError(err)
			return
		}

		recordTranscodeMetrics(ctx, spec.Codec, time.Since(startTime))
	}()

	return outputReader, nil
}

// decodeThumbnailFrame locates the video stream, optionally
// seeks to the requested timestamp, and decodes the first
// available frame.
//
// Takes inputCtx (*astiav.FormatContext) which provides access
// to the input media container.
// Takes spec (media.ThumbnailSpec) which defines the timestamp
// and extraction parameters.
//
// Returns *astiav.Frame which contains the decoded video frame.
// Returns error when no video stream is found or decoding fails.
func (p *Provider) decodeThumbnailFrame(
	ctx context.Context, inputCtx *astiav.FormatContext, spec media.ThumbnailSpec,
) (*astiav.Frame, error) {
	ctx, l := logger.From(ctx, log)

	videoStreamIndex := findFirstVideoStreamIndex(inputCtx)
	if videoStreamIndex == -1 {
		return nil, media.ErrInvalidStream
	}

	videoStream := inputCtx.Streams()[videoStreamIndex]

	decoder, err := p.createDecoder(ctx, videoStream)
	if err != nil {
		return nil, err
	}
	defer decoder.Free()

	if spec.Timestamp > 0 {
		timestampPts := p.durationToPts(spec.Timestamp, videoStream.TimeBase())
		if err := inputCtx.SeekFrame(
			videoStreamIndex, timestampPts,
			astiav.NewSeekFlags(astiav.SeekFlagBackward),
		); err != nil {
			l.Trace("Seek failed, using first frame", logger.Error(err))
		}
	}

	return p.decodeFirstFrame(ctx, inputCtx, decoder, videoStreamIndex)
}

// scaleThumbnailAndEncode optionally scales the frame and
// encodes it to the requested image format.
//
// Takes frame (*astiav.Frame) which contains the decoded video
// frame to process.
// Takes spec (media.ThumbnailSpec) which defines the desired
// dimensions and output format.
//
// Returns io.ReadCloser which streams the encoded image data.
// Returns error when scaling or encoding fails.
func (p *Provider) scaleThumbnailAndEncode(
	ctx context.Context, frame *astiav.Frame, spec media.ThumbnailSpec,
) (io.ReadCloser, error) {
	outputFrame := frame
	if spec.Width > 0 || spec.Height > 0 {
		scaledFrame, err := p.scaleFrame(ctx, frame, spec.Width, spec.Height)
		if err != nil {
			return nil, err
		}
		defer scaledFrame.Free()
		outputFrame = scaledFrame
	}

	imageData, err := p.encodeFrameToImage(ctx, outputFrame, spec.Format, spec.Quality)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(io.NewSectionReader(
		readerAtFromBytes(imageData), 0, int64(len(imageData)),
	)), nil
}

// readerAtFromBytes wraps a byte slice to implement io.ReaderAt.
type readerAtFromBytes []byte

// ReadAt reads len(p) bytes from the byte slice starting at byte offset off.
//
// Takes p ([]byte) which is the buffer to read data into.
// Takes off (int64) which is the byte offset to start reading from.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the offset is beyond the end of the data.
func (r readerAtFromBytes) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(r)) {
		return 0, io.EOF
	}
	n = copy(p, r[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// durationToPts converts a time.Duration to a PTS value.
//
// Takes d (time.Duration) which is the duration to convert.
// Takes timeBase (astiav.Rational) which is the time base for the conversion.
//
// Returns int64 which is the presentation timestamp in time base units.
func (*Provider) durationToPts(d time.Duration, timeBase astiav.Rational) int64 {
	seconds := float64(d) / float64(time.Second)
	return int64(seconds / timeBase.Float64())
}

// createDecoder creates a decoder for the given video stream.
//
// Takes stream (*astiav.Stream) which provides the stream to decode.
//
// Returns *astiav.CodecContext which is the configured decoder context.
// Returns error when the decoder is not found or initialisation fails.
func (*Provider) createDecoder(ctx context.Context, stream *astiav.Stream) (*astiav.CodecContext, error) {
	ctx, l := logger.From(ctx, log)
	_, span, _ := l.Span(ctx, "createDecoder")
	defer span.End()

	codecParams := stream.CodecParameters()
	codec := astiav.FindDecoder(codecParams.CodecID())
	if codec == nil {
		return nil, fmt.Errorf("decoder not found for codec: %s", codecParams.CodecID().Name())
	}

	codecCtx := astiav.AllocCodecContext(codec)
	if codecCtx == nil {
		return nil, errors.New("failed to allocate codec context")
	}

	if err := codecParams.ToCodecContext(codecCtx); err != nil {
		codecCtx.Free()
		return nil, fmt.Errorf("copying codec params: %w", err)
	}

	if err := codecCtx.Open(codec, nil); err != nil {
		codecCtx.Free()
		return nil, fmt.Errorf("opening codec: %w", err)
	}

	return codecCtx, nil
}

// decodeFirstFrame reads and decodes the first available video frame.
//
// Takes inputCtx (*astiav.FormatContext) which provides access to the input
// media container.
// Takes decoder (*astiav.CodecContext) which decodes the video stream.
// Takes videoStreamIndex (int) which identifies which stream to decode.
//
// Returns *astiav.Frame which contains the first decoded video frame.
// Returns error when packet allocation fails, no frames exist, or decoding
// fails.
func (*Provider) decodeFirstFrame(
	ctx context.Context,
	inputCtx *astiav.FormatContext,
	decoder *astiav.CodecContext,
	videoStreamIndex int,
) (*astiav.Frame, error) {
	ctx, l := logger.From(ctx, log)
	_, span, _ := l.Span(ctx, "decodeFirstFrame")
	defer span.End()

	packet := astiav.AllocPacket()
	if packet == nil {
		return nil, errors.New("failed to allocate packet")
	}
	defer packet.Free()

	frame := astiav.AllocFrame()
	if frame == nil {
		return nil, errors.New("failed to allocate frame")
	}

	for {
		if err := inputCtx.ReadFrame(packet); err != nil {
			frame.Free()
			return nil, readFrameError(err)
		}

		result, err := tryDecodeVideoPacket(decoder, packet, frame, videoStreamIndex)
		packet.Unref()

		if err != nil {
			frame.Free()
			return nil, err
		}

		if result {
			return frame, nil
		}
	}
}

// scaleFrame scales a frame to the given size.
//
// Takes source (*astiav.Frame) which is the source frame to
// scale.
// Takes targetWidth (int) which is the desired output width.
// Set to zero to work out the width from the height while
// keeping the aspect ratio.
// Takes targetHeight (int) which is the desired output height.
// Set to zero to work out the height from the width while
// keeping the aspect ratio.
//
// Returns *astiav.Frame which is the scaled frame in RGB24
// format.
// Returns error when the scaler cannot be created or scaling
// fails.
func (*Provider) scaleFrame(ctx context.Context, source *astiav.Frame, targetWidth, targetHeight int) (*astiav.Frame, error) {
	ctx, l := logger.From(ctx, log)
	_, span, _ := l.Span(ctx, "scaleFrame")
	defer span.End()

	srcWidth := source.Width()
	srcHeight := source.Height()

	outWidth, outHeight := targetWidth, targetHeight
	if outWidth == 0 && outHeight > 0 {
		outWidth = srcWidth * outHeight / srcHeight
	} else if outHeight == 0 && outWidth > 0 {
		outHeight = srcHeight * outWidth / srcWidth
	}

	swsCtx, err := astiav.CreateSoftwareScaleContext(
		srcWidth, srcHeight, source.PixelFormat(),
		outWidth, outHeight, astiav.PixelFormatRgb24,
		astiav.NewSoftwareScaleContextFlags(astiav.SoftwareScaleContextFlagBilinear),
	)
	if err != nil {
		return nil, fmt.Errorf("creating scaler: %w", err)
	}
	defer swsCtx.Free()

	destination := astiav.AllocFrame()
	if destination == nil {
		return nil, errors.New("failed to allocate destination frame")
	}

	destination.SetWidth(outWidth)
	destination.SetHeight(outHeight)
	destination.SetPixelFormat(astiav.PixelFormatRgb24)

	if err := destination.AllocBuffer(1); err != nil {
		destination.Free()
		return nil, fmt.Errorf("allocating buffer: %w", err)
	}

	if err := swsCtx.ScaleFrame(source, destination); err != nil {
		destination.Free()
		return nil, fmt.Errorf("scaling: %w", err)
	}

	return destination, nil
}

// encodeFrameToImage encodes a frame to JPEG, PNG, or WebP format.
//
// Takes frame (*astiav.Frame) which contains the video frame to encode.
// Takes format (string) which specifies the output format (jpeg, png, webp).
// Takes quality (int) which sets the encoding quality level.
//
// Returns []byte which contains the encoded image data.
// Returns error when colour conversion or encoding fails.
func (*Provider) encodeFrameToImage(ctx context.Context, frame *astiav.Frame, format string, quality int) ([]byte, error) {
	ctx, l := logger.From(ctx, log)
	_, span, _ := l.Span(ctx, "encodeFrameToImage")
	defer span.End()

	width := frame.Width()
	height := frame.Height()

	var rgbFrame *astiav.Frame
	if frame.PixelFormat() != astiav.PixelFormatRgb24 {
		swsCtx, err := astiav.CreateSoftwareScaleContext(
			width, height, frame.PixelFormat(),
			width, height, astiav.PixelFormatRgb24,
			astiav.NewSoftwareScaleContextFlags(astiav.SoftwareScaleContextFlagBilinear),
		)
		if err != nil {
			return nil, fmt.Errorf("creating RGB converter: %w", err)
		}
		defer swsCtx.Free()

		rgbFrame = astiav.AllocFrame()
		if rgbFrame == nil {
			return nil, errors.New("failed to allocate RGB frame")
		}
		defer rgbFrame.Free()

		rgbFrame.SetWidth(width)
		rgbFrame.SetHeight(height)
		rgbFrame.SetPixelFormat(astiav.PixelFormatRgb24)

		if err := rgbFrame.AllocBuffer(1); err != nil {
			return nil, fmt.Errorf("allocating RGB buffer: %w", err)
		}

		if err := swsCtx.ScaleFrame(frame, rgbFrame); err != nil {
			return nil, fmt.Errorf("converting to RGB: %w", err)
		}
	} else {
		rgbFrame = frame
	}

	imageData, err := encodeRGBFrameToImage(rgbFrame, format, quality)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}

// createInputFormatContext creates an FFmpeg input format context from an
// io.Reader.
//
// Takes input (io.Reader) which provides the media data to be read.
//
// Returns *astiav.FormatContext which is the configured input format context.
// Returns error when allocation fails, IO context creation fails, or stream
// info cannot be found.
func (*Provider) createInputFormatContext(ctx context.Context, input io.Reader) (*astiav.FormatContext, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "createInputFormatContext")
	defer span.End()

	inputFormatContext := astiav.AllocFormatContext()
	if inputFormatContext == nil {
		err := errors.New("failed to allocate input format context")
		l.ReportError(span, err, "Allocation failed")
		return nil, err
	}

	FFmpegContextAllocated.Add(ctx, 1)

	ioContext, err := astiav.AllocIOContext(
		ioContextBufferSize,
		false,
		func(b []byte) (n int, err error) {
			return input.Read(b)
		},
		nil,
		nil,
	)
	if err != nil {
		inputFormatContext.Free()
		l.ReportError(span, err, "Failed to create IO context")
		return nil, fmt.Errorf("creating IO context: %w", err)
	}

	inputFormatContext.SetPb(ioContext)

	if err := inputFormatContext.OpenInput("", nil, nil); err != nil {
		inputFormatContext.Free()
		l.ReportError(span, err, "Failed to open input")
		return nil, fmt.Errorf("opening input: %w", err)
	}

	if err := inputFormatContext.FindStreamInfo(nil); err != nil {
		inputFormatContext.Free()
		l.ReportError(span, err, "Failed to find stream info")
		return nil, fmt.Errorf("finding stream info: %w", err)
	}

	StreamsOpened.Add(ctx, 1)

	return inputFormatContext, nil
}

// recordTranscodeMetrics records duration and codec-specific
// counters after a successful transcode.
//
// Takes codec (string) which identifies which codec counter
// to increment.
// Takes duration (time.Duration) which is the elapsed
// transcoding time to record.
func recordTranscodeMetrics(ctx context.Context, codec string, duration time.Duration) {
	TranscodeDuration.Record(ctx, float64(duration.Milliseconds()))
	TranscodeCount.Add(ctx, 1)

	switch codec {
	case "h264":
		H264TranscodeCount.Add(ctx, 1)
	case "h265":
		H265TranscodeCount.Add(ctx, 1)
	case "vp9":
		VP9TranscodeCount.Add(ctx, 1)
	}
}

// findFirstVideoStreamIndex returns the index of the first
// video stream, or -1 if none is found.
//
// Takes inputCtx (*astiav.FormatContext) which provides access
// to the media container's streams.
//
// Returns int which is the zero-based stream index, or -1 when
// no video stream exists.
func findFirstVideoStreamIndex(inputCtx *astiav.FormatContext) int {
	for i := range inputCtx.NbStreams() {
		stream := inputCtx.Streams()[i]
		if stream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
			return i
		}
	}
	return -1
}

// readFrameError converts a ReadFrame error into a user-facing
// error, translating EOF into a descriptive message.
//
// Takes err (error) which is the original ReadFrame error.
//
// Returns error which wraps the original or replaces EOF with
// a descriptive message.
func readFrameError(err error) error {
	if errors.Is(err, astiav.ErrEof) {
		return errNoFramesInVideo
	}
	return fmt.Errorf("reading frame: %w", err)
}

// tryDecodeVideoPacket attempts to decode a single packet for
// the target video stream.
//
// Takes decoder (*astiav.CodecContext) which decodes the video
// packet.
// Takes packet (*astiav.Packet) which is the packet to decode.
// Takes frame (*astiav.Frame) which receives the decoded frame
// data.
// Takes videoStreamIndex (int) which identifies the target
// stream.
//
// Returns bool which is true when a frame was successfully
// decoded, false when the packet was skipped or more data is
// needed.
// Returns error when decoding fails.
func tryDecodeVideoPacket(
	decoder *astiav.CodecContext, packet *astiav.Packet,
	frame *astiav.Frame, videoStreamIndex int,
) (bool, error) {
	if packet.StreamIndex() != videoStreamIndex {
		return false, nil
	}

	if err := decoder.SendPacket(packet); err != nil {
		return false, nil
	}

	if err := decoder.ReceiveFrame(frame); err != nil {
		if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
			return false, nil
		}
		return false, fmt.Errorf("receiving frame: %w", err)
	}

	return true, nil
}
