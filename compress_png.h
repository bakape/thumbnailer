#pragma once
#include "thumbnailer.h"

// Lossy PNG compression
char* compress_png(
    Image* img, struct Thumbnail* thumb, const struct CompressionRange quality);
