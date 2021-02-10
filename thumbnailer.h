#pragma once
#include "ffmpeg.h"

struct Buffer {
    uint8_t* data;
    size_t size;
    uint64_t width, height;
};

struct Dims {
    uint64_t width, height;
};

// Writes RGBA thumbnail buffer to img
int generate_thumbnail(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream, const struct Dims thumb_dims);
