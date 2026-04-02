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

// createDecoderForStream creates and sets up a decoder for the given video
// stream.
//
// Takes inputCtx (*astiav.FormatContext) which provides the input format
// context that contains the streams.
// Takes streamIndex (int) which specifies which stream to decode.
//
// Returns *astiav.CodecContext which is the set up decoder ready for use.
// Returns error when the stream index is invalid, the decoder cannot be found,
// or the decoder fails to set up.
func (*Provider) createDecoderForStream(ctx context.Context, inputCtx *astiav.FormatContext, streamIndex int) (*astiav.CodecContext, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "createDecoderForStream",
		logger.Int("streamIndex", streamIndex),
	)
	defer span.End()

	if streamIndex < 0 || streamIndex >= inputCtx.NbStreams() {
		err := fmt.Errorf("invalid stream index: %d", streamIndex)
		l.ReportError(span, err, "Invalid stream index")
		return nil, err
	}

	stream := inputCtx.Streams()[streamIndex]
	codecParams := stream.CodecParameters()

	decoder := astiav.FindDecoder(codecParams.CodecID())
	if decoder == nil {
		err := fmt.Errorf("decoder not found for codec: %s", codecParams.CodecID().Name())
		l.ReportError(span, err, "Decoder not found")
		return nil, err
	}

	decoderCtx := astiav.AllocCodecContext(decoder)
	if decoderCtx == nil {
		err := errors.New("failed to allocate decoder context")
		l.ReportError(span, err, "Allocation failed")
		return nil, err
	}

	FFmpegContextAllocated.Add(ctx, 1)

	if err := decoderCtx.FromCodecParameters(codecParams); err != nil {
		decoderCtx.Free()
		l.ReportError(span, err, "Failed to copy codec parameters")
		return nil, fmt.Errorf("copying codec parameters to decoder: %w", err)
	}

	decoderCtx.SetThreadCount(0)

	if err := decoderCtx.Open(decoder, nil); err != nil {
		decoderCtx.Free()
		l.ReportError(span, err, "Failed to open decoder")
		return nil, fmt.Errorf("opening decoder: %w", err)
	}

	return decoderCtx, nil
}

// findBestVideoStream finds the video stream with the highest resolution.
//
// Takes inputCtx (*astiav.FormatContext) which contains the streams to search.
//
// Returns int which is the index of the best video stream.
// Returns error when no video stream is found.
func (*Provider) findBestVideoStream(inputCtx *astiav.FormatContext) (int, error) {
	bestStreamIndex := -1
	var bestStream *astiav.Stream

	for i := range inputCtx.NbStreams() {
		stream := inputCtx.Streams()[i]
		codecParams := stream.CodecParameters()

		if codecParams.MediaType() == astiav.MediaTypeVideo {
			if bestStreamIndex == -1 {
				bestStreamIndex = i
				bestStream = stream
			} else {
				currentBest := bestStream.CodecParameters()
				if codecParams.Width()*codecParams.Height() > currentBest.Width()*currentBest.Height() {
					bestStreamIndex = i
					bestStream = stream
				}
			}
		}
	}

	if bestStreamIndex == -1 {
		return -1, media.ErrInvalidStream
	}

	return bestStreamIndex, nil
}

// decodePacket decodes a packet into frames.
//
// Takes decoderCtx (*astiav.CodecContext) which holds the decoder state.
// Takes packet (*astiav.Packet) which holds the encoded data to decode.
// Takes onFrame (func(*astiav.Frame) error) which handles each decoded frame.
//
// Returns error when sending the packet fails or frame handling fails.
func (*Provider) decodePacket(ctx context.Context, decoderCtx *astiav.CodecContext, packet *astiav.Packet, onFrame func(*astiav.Frame) error) error {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "decodePacket")
	defer span.End()

	if err := decoderCtx.SendPacket(packet); err != nil {
		if !errors.Is(err, astiav.ErrEof) && !errors.Is(err, astiav.ErrEagain) {
			l.ReportError(span, err, "Failed to send packet to decoder")
			PacketErrors.Add(ctx, 1)
			return fmt.Errorf("sending packet to decoder: %w", err)
		}
	}

	frame := astiav.AllocFrame()
	defer frame.Free()

	for {
		err := decoderCtx.ReceiveFrame(frame)
		if err != nil {
			if errors.Is(err, astiav.ErrEagain) || errors.Is(err, astiav.ErrEof) {
				break
			}
			l.ReportError(span, err, "Failed to receive frame from decoder")
			FrameErrors.Add(ctx, 1)
			return fmt.Errorf("receiving frame from decoder: %w", err)
		}

		FramesDecoded.Add(ctx, 1)

		if err := onFrame(frame); err != nil {
			return err
		}

		frame.Unref()
	}

	return nil
}

// flushDecoder drains any remaining frames from the decoder.
//
// Takes decoderCtx (*astiav.CodecContext) which is the decoder to drain.
// Takes onFrame (func(*astiav.Frame) error) which handles each output frame.
//
// Returns error when draining fails or frame handling returns an error.
func (*Provider) flushDecoder(ctx context.Context, decoderCtx *astiav.CodecContext, onFrame func(*astiav.Frame) error) error {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "flushDecoder")
	defer span.End()

	if err := decoderCtx.SendPacket(nil); err != nil {
		if !errors.Is(err, astiav.ErrEof) {
			l.ReportError(span, err, "Failed to flush decoder")
			return fmt.Errorf("flushing decoder: %w", err)
		}
	}

	frame := astiav.AllocFrame()
	defer frame.Free()

	for {
		err := decoderCtx.ReceiveFrame(frame)
		if err != nil {
			if errors.Is(err, astiav.ErrEof) {
				break
			}
			if errors.Is(err, astiav.ErrEagain) {
				break
			}
			l.ReportError(span, err, "Error receiving frame during flush")
			return fmt.Errorf("receiving frame during flush: %w", err)
		}

		FramesDecoded.Add(ctx, 1)

		if err := onFrame(frame); err != nil {
			return err
		}

		frame.Unref()
	}

	return nil
}
