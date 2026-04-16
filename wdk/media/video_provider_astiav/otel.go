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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	// log is the package-level logger for the video_provider_astiav package.
	log = logger.GetLogger("piko/media/video_provider_astiav")

	// Meter is the OpenTelemetry meter for the video_provider_astiav package.
	Meter = otel.Meter("piko/media/video_provider_astiav")

	// FFmpegInitDuration records the time taken to initialise FFmpeg operations.
	FFmpegInitDuration metric.Float64Histogram

	// FFmpegDecodeDuration records the time taken to decode media using FFmpeg.
	FFmpegDecodeDuration metric.Float64Histogram

	// FFmpegEncodeDuration records the duration of FFmpeg encoding operations.
	FFmpegEncodeDuration metric.Float64Histogram

	// FFmpegMuxDuration records the time taken for FFmpeg muxing operations.
	FFmpegMuxDuration metric.Float64Histogram

	// FFmpegDemuxDuration records the duration of FFmpeg demux operations.
	FFmpegDemuxDuration metric.Float64Histogram

	// FFmpegFlushDuration records the time taken to flush FFmpeg output buffers.
	FFmpegFlushDuration metric.Float64Histogram

	// FFmpegOperationErrors tracks the total count of FFmpeg operation failures.
	FFmpegOperationErrors metric.Int64Counter

	// FFmpegContextAllocated counts the number of FFmpeg contexts allocated.
	FFmpegContextAllocated metric.Int64Counter

	// FFmpegContextFreed counts FFmpeg contexts that have been freed.
	FFmpegContextFreed metric.Int64Counter

	// PacketsRead is a counter metric that tracks the number of packets read.
	PacketsRead metric.Int64Counter

	// PacketsWritten counts the total number of packets sent.
	PacketsWritten metric.Int64Counter

	// FramesDecoded counts the total number of frames decoded.
	FramesDecoded metric.Int64Counter

	// FramesEncoded counts the total number of video frames encoded.
	FramesEncoded metric.Int64Counter

	// PacketErrors is a counter metric that tracks the number of packet errors.
	PacketErrors metric.Int64Counter

	// FrameErrors counts the number of frame-level errors encountered.
	FrameErrors metric.Int64Counter

	// PacketsDropped is a counter that tracks the number of dropped packets.
	PacketsDropped metric.Int64Counter

	// FramesDropped counts the number of frames that have been dropped.
	FramesDropped metric.Int64Counter

	// H264EncodeOperations is a codec-specific metric that counts H.264 encoding
	// operations.
	H264EncodeOperations metric.Int64Counter

	// H265EncodeOperations counts the total number of H.265 encode operations.
	H265EncodeOperations metric.Int64Counter

	// VP9EncodeOperations counts the number of VP9 video encoding operations.
	VP9EncodeOperations metric.Int64Counter

	// H264EncodeErrors counts the total number of H264 encoding failures.
	H264EncodeErrors metric.Int64Counter

	// H265EncodeErrors counts H.265 encoding failures.
	H265EncodeErrors metric.Int64Counter

	// VP9EncodeErrors counts the number of VP9 encoding errors that have occurred.
	VP9EncodeErrors metric.Int64Counter

	// StreamsOpened is a counter metric that tracks the number of streams opened.
	StreamsOpened metric.Int64Counter

	// StreamsClosed is a counter that tracks the number of closed streams.
	StreamsClosed metric.Int64Counter

	// StreamErrors is a counter metric that tracks the number of stream errors.
	StreamErrors metric.Int64Counter

	// StreamBytesRead is a counter that tracks the total bytes read from streams.
	StreamBytesRead metric.Int64Counter

	// StreamBytesWritten is a counter metric that tracks the total bytes written
	// to streams.
	StreamBytesWritten metric.Int64Counter

	// HLSSegmentsCreated counts the number of HLS segments created.
	HLSSegmentsCreated metric.Int64Counter

	// HLSPlaylistsGenerated counts the number of HLS playlists generated.
	HLSPlaylistsGenerated metric.Int64Counter

	// HLSSegmentationDuration records the time taken to segment media into HLS
	// format.
	HLSSegmentationDuration metric.Float64Histogram

	// HLSSegmentationErrors counts errors that occur during HLS video segmentation.
	HLSSegmentationErrors metric.Int64Counter

	// ContextPoolHits counts the number of times a context is reused from the pool.
	ContextPoolHits metric.Int64Counter

	// ContextPoolMisses counts how many times a context was not found in the pool
	// and had to be created anew.
	ContextPoolMisses metric.Int64Counter

	// ContextPoolSize tracks the current number of contexts in the pool.
	ContextPoolSize metric.Int64UpDownCounter

	// BufferPoolHits counts the number of times a buffer was found in the pool.
	BufferPoolHits metric.Int64Counter

	// BufferPoolMisses counts the number of times a buffer was not available in
	// the pool and had to be allocated.
	BufferPoolMisses metric.Int64Counter

	// BufferPoolSize tracks the current size of the buffer pool.
	BufferPoolSize metric.Int64UpDownCounter

	// TranscodeLatency is a performance metric that records the time taken for
	// transcoding operations.
	TranscodeLatency metric.Float64Histogram

	// PacketProcessingTime records the time taken to process each packet.
	PacketProcessingTime metric.Float64Histogram

	// FrameProcessingTime records the duration of frame processing operations.
	FrameProcessingTime metric.Float64Histogram

	// QueueWaitTime is a histogram that records how long items wait in a queue.
	QueueWaitTime metric.Float64Histogram

	// ActiveTranscodes tracks the number of active transcode operations.
	ActiveTranscodes metric.Int64UpDownCounter

	// TranscodeCount counts the total number of transcode operations.
	TranscodeCount metric.Int64Counter

	// TranscodeErrorCount is a metric counter that tracks the number of transcode
	// errors.
	TranscodeErrorCount metric.Int64Counter

	// TranscodeDuration records the duration of transcode operations.
	TranscodeDuration metric.Float64Histogram

	// H264TranscodeCount counts H.264 transcode operations.
	H264TranscodeCount metric.Int64Counter

	// H265TranscodeCount counts H.265 transcode operations.
	H265TranscodeCount metric.Int64Counter

	// VP9TranscodeCount counts VP9 transcode operations.
	VP9TranscodeCount metric.Int64Counter

	// FramesProcessedCount is a counter that tracks the total number of frames
	// processed.
	FramesProcessedCount metric.Int64Counter

	// AverageTranscodeFPS records the average FPS achieved during transcoding.
	AverageTranscodeFPS metric.Float64Histogram
)

func init() {
	var err error

	FFmpegInitDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.ffmpeg_init_duration",
		metric.WithDescription("Duration of FFmpeg initialisation"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegDecodeDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.ffmpeg_decode_duration",
		metric.WithDescription("Duration of FFmpeg decode operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegEncodeDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.ffmpeg_encode_duration",
		metric.WithDescription("Duration of FFmpeg encode operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegMuxDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.ffmpeg_mux_duration",
		metric.WithDescription("Duration of FFmpeg mux operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegDemuxDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.ffmpeg_demux_duration",
		metric.WithDescription("Duration of FFmpeg demux operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegFlushDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.ffmpeg_flush_duration",
		metric.WithDescription("Duration of FFmpeg flush operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegOperationErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.ffmpeg_operation_errors",
		metric.WithDescription("Number of FFmpeg operation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegContextAllocated, err = Meter.Int64Counter(
		"media.video_provider_astiav.ffmpeg_context_allocated",
		metric.WithDescription("Number of FFmpeg contexts allocated"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FFmpegContextFreed, err = Meter.Int64Counter(
		"media.video_provider_astiav.ffmpeg_context_freed",
		metric.WithDescription("Number of FFmpeg contexts freed"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PacketsRead, err = Meter.Int64Counter(
		"media.video_provider_astiav.packets_read",
		metric.WithDescription("Number of packets read from input"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PacketsWritten, err = Meter.Int64Counter(
		"media.video_provider_astiav.packets_written",
		metric.WithDescription("Number of packets written to output"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FramesDecoded, err = Meter.Int64Counter(
		"media.video_provider_astiav.frames_decoded",
		metric.WithDescription("Number of frames decoded"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FramesEncoded, err = Meter.Int64Counter(
		"media.video_provider_astiav.frames_encoded",
		metric.WithDescription("Number of frames encoded"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PacketErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.packet_errors",
		metric.WithDescription("Number of packet processing errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FrameErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.frame_errors",
		metric.WithDescription("Number of frame processing errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PacketsDropped, err = Meter.Int64Counter(
		"media.video_provider_astiav.packets_dropped",
		metric.WithDescription("Number of packets dropped"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FramesDropped, err = Meter.Int64Counter(
		"media.video_provider_astiav.frames_dropped",
		metric.WithDescription("Number of frames dropped"),
	)
	if err != nil {
		otel.Handle(err)
	}

	H264EncodeOperations, err = Meter.Int64Counter(
		"media.video_provider_astiav.h264_encode_operations",
		metric.WithDescription("Number of H.264 encode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	H265EncodeOperations, err = Meter.Int64Counter(
		"media.video_provider_astiav.h265_encode_operations",
		metric.WithDescription("Number of H.265 encode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VP9EncodeOperations, err = Meter.Int64Counter(
		"media.video_provider_astiav.vp9_encode_operations",
		metric.WithDescription("Number of VP9 encode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	H264EncodeErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.h264_encode_errors",
		metric.WithDescription("Number of H.264 encode errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	H265EncodeErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.h265_encode_errors",
		metric.WithDescription("Number of H.265 encode errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VP9EncodeErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.vp9_encode_errors",
		metric.WithDescription("Number of VP9 encode errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	StreamsOpened, err = Meter.Int64Counter(
		"media.video_provider_astiav.streams_opened",
		metric.WithDescription("Number of streams opened"),
	)
	if err != nil {
		otel.Handle(err)
	}

	StreamsClosed, err = Meter.Int64Counter(
		"media.video_provider_astiav.streams_closed",
		metric.WithDescription("Number of streams closed"),
	)
	if err != nil {
		otel.Handle(err)
	}

	StreamErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.stream_errors",
		metric.WithDescription("Number of stream errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	StreamBytesRead, err = Meter.Int64Counter(
		"media.video_provider_astiav.stream_bytes_read",
		metric.WithDescription("Number of bytes read from streams"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		otel.Handle(err)
	}

	StreamBytesWritten, err = Meter.Int64Counter(
		"media.video_provider_astiav.stream_bytes_written",
		metric.WithDescription("Number of bytes written to streams"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HLSSegmentsCreated, err = Meter.Int64Counter(
		"media.video_provider_astiav.hls_segments_created",
		metric.WithDescription("Number of HLS segments created"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HLSPlaylistsGenerated, err = Meter.Int64Counter(
		"media.video_provider_astiav.hls_playlists_generated",
		metric.WithDescription("Number of HLS playlists generated"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HLSSegmentationDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.hls_segmentation_duration",
		metric.WithDescription("Duration of HLS segmentation"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HLSSegmentationErrors, err = Meter.Int64Counter(
		"media.video_provider_astiav.hls_segmentation_errors",
		metric.WithDescription("Number of HLS segmentation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ContextPoolHits, err = Meter.Int64Counter(
		"media.video_provider_astiav.context_pool_hits",
		metric.WithDescription("Number of context pool hits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ContextPoolMisses, err = Meter.Int64Counter(
		"media.video_provider_astiav.context_pool_misses",
		metric.WithDescription("Number of context pool misses"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ContextPoolSize, err = Meter.Int64UpDownCounter(
		"media.video_provider_astiav.context_pool_size",
		metric.WithDescription("Current size of context pool"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BufferPoolHits, err = Meter.Int64Counter(
		"media.video_provider_astiav.buffer_pool_hits",
		metric.WithDescription("Number of buffer pool hits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BufferPoolMisses, err = Meter.Int64Counter(
		"media.video_provider_astiav.buffer_pool_misses",
		metric.WithDescription("Number of buffer pool misses"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BufferPoolSize, err = Meter.Int64UpDownCounter(
		"media.video_provider_astiav.buffer_pool_size",
		metric.WithDescription("Current size of buffer pool"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TranscodeLatency, err = Meter.Float64Histogram(
		"media.video_provider_astiav.transcode_latency",
		metric.WithDescription("Latency of transcode operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PacketProcessingTime, err = Meter.Float64Histogram(
		"media.video_provider_astiav.packet_processing_time",
		metric.WithDescription("Time to process a single packet"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FrameProcessingTime, err = Meter.Float64Histogram(
		"media.video_provider_astiav.frame_processing_time",
		metric.WithDescription("Time to process a single frame"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	QueueWaitTime, err = Meter.Float64Histogram(
		"media.video_provider_astiav.queue_wait_time",
		metric.WithDescription("Time spent waiting in queue"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ActiveTranscodes, err = Meter.Int64UpDownCounter(
		"media.video_provider_astiav.active_transcodes",
		metric.WithDescription("Number of active transcode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TranscodeCount, err = Meter.Int64Counter(
		"media.video_provider_astiav.transcode_count",
		metric.WithDescription("Total number of transcode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TranscodeErrorCount, err = Meter.Int64Counter(
		"media.video_provider_astiav.transcode_error_count",
		metric.WithDescription("Number of transcode errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TranscodeDuration, err = Meter.Float64Histogram(
		"media.video_provider_astiav.transcode_duration",
		metric.WithDescription("Duration of transcode operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	H264TranscodeCount, err = Meter.Int64Counter(
		"media.video_provider_astiav.h264_transcode_count",
		metric.WithDescription("Number of H.264 transcode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	H265TranscodeCount, err = Meter.Int64Counter(
		"media.video_provider_astiav.h265_transcode_count",
		metric.WithDescription("Number of H.265 transcode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VP9TranscodeCount, err = Meter.Int64Counter(
		"media.video_provider_astiav.vp9_transcode_count",
		metric.WithDescription("Number of VP9 transcode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	FramesProcessedCount, err = Meter.Int64Counter(
		"media.video_provider_astiav.frames_processed_count",
		metric.WithDescription("Total number of frames processed"),
	)
	if err != nil {
		otel.Handle(err)
	}

	AverageTranscodeFPS, err = Meter.Float64Histogram(
		"media.video_provider_astiav.average_transcode_fps",
		metric.WithDescription("Average FPS achieved during transcoding"),
		metric.WithUnit("fps"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
