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

	"github.com/asticode/go-astiav"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/media"
)

// createEncoder creates and initialises an encoder based on the transcode spec.
//
// Takes spec (TranscodeSpec) which defines the target codec and
// encoding parameters.
// Takes decoderCtx (*astiav.CodecContext) which provides the source video
// properties for encoder configuration.
//
// Returns *astiav.CodecContext which is the configured and opened encoder
// context ready for encoding frames.
// Returns error when the codec is not found, encoder allocation fails, or the
// encoder cannot be opened with the specified options.
func (p *Provider) createEncoder(
	ctx context.Context, spec media.TranscodeSpec, decoderCtx *astiav.CodecContext,
) (*astiav.CodecContext, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "createEncoder",
		logger.String("codec", spec.Codec),
	)
	defer span.End()

	encoderCtx, encoder, err := p.allocateEncoderContext(ctx, spec)
	if err != nil {
		l.ReportError(span, err, "Failed to allocate encoder context")
		return nil, err
	}

	if err := p.configureAndOpenEncoder(ctx, encoderCtx, encoder, spec, decoderCtx); err != nil {
		encoderCtx.Free()
		l.ReportError(span, err, "Failed to configure and open encoder")
		return nil, err
	}

	recordEncoderMetric(ctx, spec.Codec)

	return encoderCtx, nil
}

// allocateEncoderContext looks up the codec, finds the FFmpeg
// encoder, and allocates a codec context for it.
//
// Takes spec (TranscodeSpec) which identifies the codec to
// allocate.
//
// Returns *astiav.CodecContext which is the allocated encoder
// context.
// Returns *astiav.Codec which is the resolved FFmpeg encoder.
// Returns error when the codec is not found or allocation
// fails.
func (p *Provider) allocateEncoderContext(
	ctx context.Context, spec media.TranscodeSpec,
) (*astiav.CodecContext, *astiav.Codec, error) {
	codecConfig, err := p.codecRegistry.GetCodec(spec.Codec)
	if err != nil {
		return nil, nil, err
	}

	encoder := astiav.FindEncoderByName(codecConfig.EncoderName)
	if encoder == nil {
		return nil, nil, fmt.Errorf("encoder not found: %s", codecConfig.EncoderName)
	}

	encoderCtx := astiav.AllocCodecContext(encoder)
	if encoderCtx == nil {
		return nil, nil, errors.New("failed to allocate encoder context")
	}

	FFmpegContextAllocated.Add(ctx, 1)

	return encoderCtx, encoder, nil
}

// configureAndOpenEncoder configures the encoder parameters
// and opens it with codec-specific options. The caller must
// free encoderCtx on error.
//
// Takes encoderCtx (*astiav.CodecContext) which is the
// encoder context to configure.
// Takes encoder (*astiav.Codec) which is the FFmpeg encoder.
// Takes spec (TranscodeSpec) which provides the encoding
// settings.
// Takes decoderCtx (*astiav.CodecContext) which supplies
// fallback values from the input stream.
//
// Returns error when configuration or opening fails.
func (p *Provider) configureAndOpenEncoder(
	ctx context.Context,
	encoderCtx *astiav.CodecContext,
	encoder *astiav.Codec,
	spec media.TranscodeSpec,
	decoderCtx *astiav.CodecContext,
) error {
	codecConfig, err := p.codecRegistry.GetCodec(spec.Codec)
	if err != nil {
		return err
	}

	if err := p.configureEncoder(ctx, encoderCtx, spec, codecConfig, decoderCtx); err != nil {
		return fmt.Errorf("configuring encoder: %w", err)
	}

	optionsDict, err := p.buildEncoderOptionsDict(&spec)
	if err != nil {
		return err
	}

	if err := encoderCtx.Open(encoder, optionsDict); err != nil {
		return fmt.Errorf("opening encoder: %w", err)
	}

	return nil
}

// buildEncoderOptionsDict builds an astiav dictionary from
// codec-specific encoder options.
//
// Takes spec (*media.TranscodeSpec) which specifies the codec
// and encoding parameters.
//
// Returns *astiav.Dictionary which contains the encoder
// option key-value pairs.
// Returns error when option retrieval or dictionary
// population fails.
func (p *Provider) buildEncoderOptionsDict(spec *media.TranscodeSpec) (*astiav.Dictionary, error) {
	encoderOptions, err := p.codecRegistry.GetEncoderOptions(spec)
	if err != nil {
		return nil, fmt.Errorf("getting encoder options: %w", err)
	}

	optionsDict := astiav.NewDictionary()
	for key, value := range encoderOptions {
		if err := optionsDict.Set(key, value, 0); err != nil {
			return nil, fmt.Errorf("setting encoder option %s=%s: %w", key, value, err)
		}
	}

	return optionsDict, nil
}

// configureEncoder configures the encoder context with the transcode spec
// parameters.
//
// Takes encoderCtx (*astiav.CodecContext) which is the encoder to configure.
// Takes spec (TranscodeSpec) which specifies the transcode settings.
// Takes codecConfig (*CodecConfig) which provides codec-specific configuration.
// Takes decoderCtx (*astiav.CodecContext) which supplies fallback values from
// the input stream.
//
// Returns error when configuration fails.
func (*Provider) configureEncoder(
	ctx context.Context,
	encoderCtx *astiav.CodecContext,
	spec media.TranscodeSpec,
	codecConfig *CodecConfig,
	decoderCtx *astiav.CodecContext,
) error {
	ctx, l := logger.From(ctx, log)
	_, span, _ := l.Span(ctx, "configureEncoder")
	defer span.End()

	if spec.Width > 0 && spec.Height > 0 {
		encoderCtx.SetWidth(spec.Width)
		encoderCtx.SetHeight(spec.Height)
	} else {
		encoderCtx.SetWidth(decoderCtx.Width())
		encoderCtx.SetHeight(decoderCtx.Height())
	}

	encoderCtx.SetPixelFormat(codecConfig.PixelFormat)

	if spec.Framerate > 0 {
		encoderCtx.SetTimeBase(astiav.NewRational(1, int(spec.Framerate)))
		encoderCtx.SetFramerate(astiav.NewRational(int(spec.Framerate), 1))
	} else {
		encoderCtx.SetTimeBase(decoderCtx.TimeBase())
		encoderCtx.SetFramerate(decoderCtx.Framerate())
	}

	if spec.Bitrate > 0 {
		encoderCtx.SetBitRate(int64(spec.Bitrate))
	}

	gopSize := defaultGOPSize
	if spec.Framerate > 0 {
		gopSize = int(spec.Framerate * 2)
	} else if decoderCtx.Framerate().Num() > 0 {
		fps := float64(decoderCtx.Framerate().Num()) / float64(decoderCtx.Framerate().Den())
		gopSize = int(fps * 2)
	}
	encoderCtx.SetGopSize(gopSize)

	encoderCtx.SetMaxBFrames(2)

	encoderCtx.SetThreadCount(0)

	switch spec.Codec {
	case "h264":
		encoderCtx.SetProfile(astiav.ProfileH264Main)

	case "h265":
		encoderCtx.SetProfile(astiav.ProfileHevcMain)

	case "vp9":
	}

	return nil
}

// encodeFrame encodes a frame into packets.
//
// Takes encoderCtx (*astiav.CodecContext) which provides the encoder settings.
// Takes frame (*astiav.Frame) which is the frame to encode.
// Takes onPacket (func(*astiav.Packet) error) which handles each encoded
// packet.
//
// Returns error when encoding fails or onPacket returns an error.
func (*Provider) encodeFrame(
	ctx context.Context,
	encoderCtx *astiav.CodecContext,
	frame *astiav.Frame,
	onPacket func(*astiav.Packet) error,
) error {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "encodeFrame")
	defer span.End()

	if err := encoderCtx.SendFrame(frame); err != nil {
		if !errors.Is(err, astiav.ErrEof) && !errors.Is(err, astiav.ErrEagain) {
			l.ReportError(span, err, "Failed to send frame to encoder")
			FrameErrors.Add(ctx, 1)
			return fmt.Errorf("sending frame to encoder: %w", err)
		}
	}

	packet := astiav.AllocPacket()
	defer packet.Free()

	for {
		err := encoderCtx.ReceivePacket(packet)
		if err != nil {
			if errors.Is(err, astiav.ErrEagain) || errors.Is(err, astiav.ErrEof) {
				break
			}
			l.ReportError(span, err, "Failed to receive packet from encoder")
			PacketErrors.Add(ctx, 1)
			return fmt.Errorf("receiving packet from encoder: %w", err)
		}

		FramesEncoded.Add(ctx, 1)
		PacketsWritten.Add(ctx, 1)

		if err := onPacket(packet); err != nil {
			return err
		}

		packet.Unref()
	}

	return nil
}

// flushEncoder sends any remaining packets from the encoder.
//
// Takes encoderCtx (*astiav.CodecContext) which is the encoder to flush.
// Takes onPacket (func(*astiav.Packet) error) which handles each flushed
// packet.
//
// Returns error when flushing fails or packet handling returns an error.
func (*Provider) flushEncoder(
	ctx context.Context,
	encoderCtx *astiav.CodecContext,
	onPacket func(*astiav.Packet) error,
) error {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "flushEncoder")
	defer span.End()

	startTime := astiav.NoPtsValue

	if err := encoderCtx.SendFrame(nil); err != nil {
		if !errors.Is(err, astiav.ErrEof) {
			l.ReportError(span, err, "Failed to flush encoder")
			return fmt.Errorf("flushing encoder: %w", err)
		}
	}

	packet := astiav.AllocPacket()
	defer packet.Free()

	for {
		err := encoderCtx.ReceivePacket(packet)
		if err != nil {
			if errors.Is(err, astiav.ErrEof) {
				break
			}
			if errors.Is(err, astiav.ErrEagain) {
				break
			}
			l.ReportError(span, err, "Error receiving packet during flush")
			return fmt.Errorf("receiving packet during flush: %w", err)
		}

		PacketsWritten.Add(ctx, 1)

		if err := onPacket(packet); err != nil {
			return err
		}

		packet.Unref()
	}

	duration := astiav.NoPtsValue - startTime
	FFmpegFlushDuration.Record(ctx, float64(duration))

	return nil
}

// scaleFrameForEncoder scales a frame to match encoder dimensions if needed.
//
// Takes srcFrame (*astiav.Frame) which is the source frame to scale.
// Takes dstWidth (int) which specifies the target width in pixels.
// Takes dstHeight (int) which specifies the target height in pixels.
// Takes dstPixFmt (astiav.PixelFormat) which specifies the target pixel format.
//
// Returns *astiav.Frame which is the scaled frame, or the original if no
// scaling was needed.
// Returns error when the scaler context cannot be created, buffer allocation
// fails, or the scaling operation fails.
func (*Provider) scaleFrameForEncoder(
	ctx context.Context,
	srcFrame *astiav.Frame,
	dstWidth, dstHeight int,
	dstPixFmt astiav.PixelFormat,
) (*astiav.Frame, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "scaleFrameForEncoder")
	defer span.End()

	if srcFrame.Width() == dstWidth && srcFrame.Height() == dstHeight && srcFrame.PixelFormat() == dstPixFmt {
		return srcFrame, nil
	}

	swsCtx, err := astiav.CreateSoftwareScaleContext(
		srcFrame.Width(), srcFrame.Height(), srcFrame.PixelFormat(),
		dstWidth, dstHeight, dstPixFmt,
		astiav.NewSoftwareScaleContextFlags(astiav.SoftwareScaleContextFlagBilinear),
	)
	if err != nil {
		l.ReportError(span, err, "Failed to create scaler")
		return nil, fmt.Errorf("creating software scale context: %w", err)
	}
	defer swsCtx.Free()

	dstFrame := astiav.AllocFrame()
	dstFrame.SetWidth(dstWidth)
	dstFrame.SetHeight(dstHeight)
	dstFrame.SetPixelFormat(dstPixFmt)

	if err := dstFrame.AllocBuffer(0); err != nil {
		dstFrame.Free()
		l.ReportError(span, err, "Failed to allocate frame buffer")
		return nil, fmt.Errorf("allocating frame buffer: %w", err)
	}

	if err := swsCtx.ScaleFrame(srcFrame, dstFrame); err != nil {
		dstFrame.Free()
		l.ReportError(span, err, "Failed to scale frame")
		return nil, fmt.Errorf("scaling frame: %w", err)
	}

	dstFrame.SetPictureType(srcFrame.PictureType())
	dstFrame.SetPts(srcFrame.Pts())
	dstFrame.SetKeyFrame(srcFrame.KeyFrame())

	return dstFrame, nil
}

// recordEncoderMetric increments the codec-specific encode
// operation counter.
//
// Takes codec (string) which identifies which codec counter
// to increment.
func recordEncoderMetric(ctx context.Context, codec string) {
	switch codec {
	case "h264":
		H264EncodeOperations.Add(ctx, 1)
	case "h265":
		H265EncodeOperations.Add(ctx, 1)
	case "vp9":
		VP9EncodeOperations.Add(ctx, 1)
	}
}
