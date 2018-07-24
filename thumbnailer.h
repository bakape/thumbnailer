#pragma once
#include "util.h"
#include <magick/api.h>
#include <stdbool.h>

struct Thumbnail {
    bool isPNG;
    struct Buffer img;
};

struct CompressionRange {
    uint8_t min, max;
};

struct Options {
    uint8_t JPEGCompression;
    struct CompressionRange PNGCompression;
    struct Dims maxSrcDims, thumbDims;
};

char* thumbnail(
    struct Buffer* src, struct Thumbnail* thumb, const struct Options opts);
