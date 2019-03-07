#pragma once
#include <libavformat/avformat.h>
#include <pthread.h>

extern int readCallBack(void*, uint8_t*, int);
extern int64_t seekCallBack(void*, int64_t, int);

// Initialize FFmpeg
void init(void);

// Initialize am AVFormatContext with the buffered file/
// input_format can be NULL.
int create_context(AVFormatContext** ctx, const char* input_format);

// Create a AVCodecContext of the desired media type
int codec_context(AVCodecContext** avcc, int* stream, AVFormatContext* avfc,
    const enum AVMediaType type);
