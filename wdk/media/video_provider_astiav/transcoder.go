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

// transcodePipeline holds the FFmpeg contexts needed for a
// transcode operation.
type transcodePipeline struct {
	// inputCtx is the input media container context.
	inputCtx *astiav.FormatContext

	// outputCtx is the output media container context.
	outputCtx *astiav.FormatContext

	// decoderCtx is the codec context used for decoding.
	decoderCtx *astiav.CodecContext

	// encoderCtx is the codec context used for encoding.
	encoderCtx *astiav.CodecContext

	// videoStreamIndex is the zero-based index of the video
	// stream being transcoded.
	videoStreamIndex int
}

// runTranscode executes the main transcode operation.
//
// Takes input (io.Reader) which provides the source video data.
// Takes output (io.Writer) which receives the transcoded video data.
// Takes spec (TranscodeSpec) which defines the transcode settings.
//
// Returns error when any transcode step fails.
func (p *Provider) runTranscode(ctx context.Context, input io.Reader, output io.Writer, spec media.TranscodeSpec) error {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "runTranscode")
	defer span.End()

	pipeline, err := p.initialiseTranscodePipeline(ctx, input, output, spec)
	if err != nil {
		l.ReportError(span, err, "Failed to initialise transcode pipeline")
		return err
	}
	defer pipeline.free(ctx)

	return p.executeTranscodePipeline(ctx, pipeline)
}

// initialiseTranscodePipeline sets up all FFmpeg contexts
// required for transcoding.
//
// Takes input (io.Reader) which provides the source video data.
// Takes output (io.Writer) which receives the transcoded data.
// Takes spec (media.TranscodeSpec) which defines the transcode
// settings.
//
// Returns *transcodePipeline which holds the initialised FFmpeg
// contexts.
// Returns error when any context creation step fails.
func (p *Provider) initialiseTranscodePipeline(
	ctx context.Context, input io.Reader, output io.Writer, spec media.TranscodeSpec,
) (*transcodePipeline, error) {
	inputCtx, err := p.createInputFormatContext(ctx, input)
	if err != nil {
		return nil, err
	}

	videoStreamIndex, err := p.findBestVideoStream(inputCtx)
	if err != nil {
		inputCtx.Free()
		return nil, err
	}

	decoderCtx, err := p.createDecoderForStream(ctx, inputCtx, videoStreamIndex)
	if err != nil {
		inputCtx.Free()
		return nil, err
	}

	encoderCtx, err := p.createEncoder(ctx, spec, decoderCtx)
	if err != nil {
		decoderCtx.Free()
		inputCtx.Free()
		return nil, err
	}

	outputCtx, err := p.createOutputFormatContext(ctx, output, spec)
	if err != nil {
		encoderCtx.Free()
		decoderCtx.Free()
		inputCtx.Free()
		return nil, err
	}

	return &transcodePipeline{
		inputCtx:         inputCtx,
		outputCtx:        outputCtx,
		decoderCtx:       decoderCtx,
		encoderCtx:       encoderCtx,
		videoStreamIndex: videoStreamIndex,
	}, nil
}

// free releases all FFmpeg contexts held by the pipeline and records metrics.
func (pl *transcodePipeline) free(ctx context.Context) {
	pl.outputCtx.Free()
	FFmpegContextFreed.Add(ctx, 1)

	pl.encoderCtx.Free()
	FFmpegContextFreed.Add(ctx, 1)

	pl.decoderCtx.Free()
	FFmpegContextFreed.Add(ctx, 1)

	pl.inputCtx.Free()
	FFmpegContextFreed.Add(ctx, 1)
	StreamsClosed.Add(ctx, 1)
}

// executeTranscodePipeline runs the decode-encode loop and
// writes the output trailer.
//
// Takes pl (*transcodePipeline) which holds the FFmpeg contexts
// for the operation.
//
// Returns error when the transcode loop or trailer write fails.
func (p *Provider) executeTranscodePipeline(ctx context.Context, pl *transcodePipeline) error {
	if err := p.runTranscodeLoop(
		ctx, pl.inputCtx, pl.outputCtx, pl.decoderCtx, pl.encoderCtx, pl.videoStreamIndex,
	); err != nil {
		return err
	}

	if err := pl.outputCtx.WriteTrailer(); err != nil {
		return fmt.Errorf("writing trailer: %w", err)
	}

	return nil
}

// runTranscodeLoop executes the main decode-encode loop.
//
// Takes inputCtx (*astiav.FormatContext) which provides the
// input media stream.
// Takes outputCtx (*astiav.FormatContext) which receives the
// transcoded output.
// Takes decoderCtx (*astiav.CodecContext) which decodes input
// packets.
// Takes encoderCtx (*astiav.CodecContext) which encodes frames
// for output.
// Takes videoStreamIndex (int) which identifies which stream
// to process.
//
// Returns error when reading, decoding, encoding, or writing
// fails.
func (p *Provider) runTranscodeLoop(
	ctx context.Context,
	inputCtx *astiav.FormatContext,
	outputCtx *astiav.FormatContext,
	decoderCtx *astiav.CodecContext,
	encoderCtx *astiav.CodecContext,
	videoStreamIndex int,
) error {
	ctx, l := logger.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "runTranscodeLoop")
	defer span.End()

	startTime := time.Now()

	frameCount, err := p.decodeEncodeAndFlush(
		ctx, inputCtx, outputCtx, decoderCtx, encoderCtx, videoStreamIndex,
	)
	if err != nil {
		return err
	}

	recordTranscodeLoopMetrics(ctx, frameCount, time.Since(startTime))

	return nil
}

// decodeEncodeAndFlush runs the packet reading, decoding,
// encoding, and flush stages.
//
// Takes inputCtx (*astiav.FormatContext) which provides the
// input media stream.
// Takes outputCtx (*astiav.FormatContext) which receives the
// transcoded output.
// Takes decoderCtx (*astiav.CodecContext) which decodes input
// packets.
// Takes encoderCtx (*astiav.CodecContext) which encodes frames
// for output.
// Takes videoStreamIndex (int) which identifies which stream to
// process.
//
// Returns int64 which is the total number of frames processed.
// Returns error when reading, decoding, encoding, or flushing
// fails.
func (p *Provider) decodeEncodeAndFlush(
	ctx context.Context,
	inputCtx *astiav.FormatContext,
	outputCtx *astiav.FormatContext,
	decoderCtx *astiav.CodecContext,
	encoderCtx *astiav.CodecContext,
	videoStreamIndex int,
) (int64, error) {
	packet := astiav.AllocPacket()
	defer packet.Free()

	frameCount := int64(0)
	processDecodedFrame := p.buildFrameProcessor(ctx, outputCtx, encoderCtx, &frameCount)

	if err := p.readAndDecodePackets(ctx, inputCtx, decoderCtx, videoStreamIndex, packet, processDecodedFrame); err != nil {
		return 0, err
	}

	if err := p.flushDecoder(ctx, decoderCtx, processDecodedFrame); err != nil {
		return 0, fmt.Errorf("flushing decoder: %w", err)
	}

	onPacket := func(encodedPacket *astiav.Packet) error {
		return writeEncodedPacket(ctx, outputCtx, encoderCtx, encodedPacket)
	}
	if err := p.flushEncoder(ctx, encoderCtx, onPacket); err != nil {
		return 0, fmt.Errorf("flushing encoder: %w", err)
	}

	return frameCount, nil
}

// buildFrameProcessor returns a closure that scales, timestamps,
// and encodes each decoded frame.
//
// Takes outputCtx (*astiav.FormatContext) which receives the
// encoded packets.
// Takes encoderCtx (*astiav.CodecContext) which encodes the
// processed frames.
// Takes frameCount (*int64) which is incremented for each frame
// processed.
//
// Returns func(*astiav.Frame) error which processes a single
// decoded frame.
func (p *Provider) buildFrameProcessor(
	ctx context.Context,
	outputCtx *astiav.FormatContext,
	encoderCtx *astiav.CodecContext,
	frameCount *int64,
) func(*astiav.Frame) error {
	return func(decodedFrame *astiav.Frame) error {
		frame, err := p.scaleFrameForEncoder(
			ctx, decodedFrame,
			encoderCtx.Width(), encoderCtx.Height(), encoderCtx.PixelFormat(),
		)
		if err != nil {
			return fmt.Errorf("scaling frame: %w", err)
		}
		defer func() {
			if frame != decodedFrame {
				frame.Free()
			}
		}()

		frame.SetPts(*frameCount)
		*frameCount++

		return p.encodeFrame(ctx, encoderCtx, frame, func(encodedPacket *astiav.Packet) error {
			return writeEncodedPacket(ctx, outputCtx, encoderCtx, encodedPacket)
		})
	}
}

// readAndDecodePackets reads packets from the input and decodes
// video packets, calling processFrame for each decoded frame.
//
// Takes inputCtx (*astiav.FormatContext) which provides the
// input media stream.
// Takes decoderCtx (*astiav.CodecContext) which decodes video
// packets.
// Takes videoStreamIndex (int) which identifies the target
// stream.
// Takes packet (*astiav.Packet) which is the reusable packet
// buffer.
// Takes processFrame (func(*astiav.Frame) error) which handles
// each decoded frame.
//
// Returns error when reading or decoding fails, or when the
// context is cancelled.
func (p *Provider) readAndDecodePackets(
	ctx context.Context,
	inputCtx *astiav.FormatContext,
	decoderCtx *astiav.CodecContext,
	videoStreamIndex int,
	packet *astiav.Packet,
	processFrame func(*astiav.Frame) error,
) error {
	_, l := logger.From(ctx, log)
	_, span, l := l.Span(ctx, "readAndDecodePackets")
	defer span.End()

	for {
		select {
		case <-ctx.Done():
			l.Warn("Transcode loop cancelled")
			return media.ErrContextCancelled
		default:
		}

		err := inputCtx.ReadFrame(packet)
		if err != nil {
			if errors.Is(err, astiav.ErrEof) {
				return nil
			}
			l.ReportError(span, err, "Failed to read frame")
			PacketErrors.Add(ctx, 1)
			return fmt.Errorf("reading frame: %w", err)
		}

		PacketsRead.Add(ctx, 1)

		if packet.StreamIndex() != videoStreamIndex {
			packet.Unref()
			PacketsDropped.Add(ctx, 1)
			continue
		}

		err = p.decodePacket(ctx, decoderCtx, packet, processFrame)

		packet.Unref()

		if err != nil {
			l.ReportError(span, err, "Error processing packet")
			return err
		}
	}
}

// createOutputFormatContext creates an output format context for writing
// transcoded video.
//
// Takes output (io.Writer) which receives the transcoded video data.
// Takes spec (media.TranscodeSpec) which defines the output format settings.
//
// Returns *astiav.FormatContext which is the configured output context.
// Returns error when allocation, IO context creation, or header writing fails.
func (*Provider) createOutputFormatContext(
	ctx context.Context, output io.Writer, spec media.TranscodeSpec,
) (*astiav.FormatContext, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "createOutputFormatContext",
		logger.String("format", spec.Format),
	)
	defer span.End()

	outputFormatContext, err := astiav.AllocOutputFormatContext(nil, spec.Format, "")
	if err != nil {
		l.ReportError(span, err, "Allocation failed")
		return nil, fmt.Errorf("allocating output format context: %w", err)
	}
	if outputFormatContext == nil {
		err := errors.New("failed to allocate output format context")
		l.ReportError(span, err, "Allocation failed")
		return nil, err
	}

	FFmpegContextAllocated.Add(ctx, 1)

	ioContext, err := astiav.AllocIOContext(
		ioContextBufferSize,
		true,
		nil,
		nil,
		func(b []byte) (n int, err error) {
			return output.Write(b)
		},
	)
	if err != nil {
		outputFormatContext.Free()
		l.ReportError(span, err, "Failed to create IO context")
		return nil, fmt.Errorf("creating output IO context: %w", err)
	}

	outputFormatContext.SetPb(ioContext)

	stream := outputFormatContext.NewStream(nil)
	if stream == nil {
		outputFormatContext.Free()
		err := errors.New("failed to create output stream")
		l.ReportError(span, err, "Stream creation failed")
		return nil, err
	}

	if err := outputFormatContext.WriteHeader(nil); err != nil {
		outputFormatContext.Free()
		l.ReportError(span, err, "Failed to write header")
		return nil, fmt.Errorf("writing header: %w", err)
	}

	return outputFormatContext, nil
}

// writeEncodedPacket writes an encoded packet to the output
// stream, rescaling timestamps and recording metrics.
//
// Takes outputCtx (*astiav.FormatContext) which receives the
// written packet.
// Takes encoderCtx (*astiav.CodecContext) which provides the
// encoder time base for timestamp rescaling.
// Takes encodedPacket (*astiav.Packet) which is the packet to
// write.
//
// Returns error when the interleaved write fails.
func writeEncodedPacket(
	ctx context.Context,
	outputCtx *astiav.FormatContext,
	encoderCtx *astiav.CodecContext,
	encodedPacket *astiav.Packet,
) error {
	encodedPacket.SetStreamIndex(0)

	outputStream := outputCtx.Streams()[0]
	encodedPacket.RescaleTs(encoderCtx.TimeBase(), outputStream.TimeBase())

	if err := outputCtx.WriteInterleavedFrame(encodedPacket); err != nil {
		return fmt.Errorf("writing interleaved frame: %w", err)
	}

	StreamBytesWritten.Add(ctx, int64(encodedPacket.Size()))
	return nil
}

// recordTranscodeLoopMetrics records frame count and FPS
// metrics after the transcode loop completes.
//
// Takes frameCount (int64) which is the number of frames
// processed.
// Takes duration (time.Duration) which is the elapsed time for
// the transcode loop.
func recordTranscodeLoopMetrics(ctx context.Context, frameCount int64, duration time.Duration) {
	fps := float64(frameCount) / duration.Seconds()
	FramesProcessedCount.Add(ctx, frameCount)
	AverageTranscodeFPS.Record(ctx, fps)
}
