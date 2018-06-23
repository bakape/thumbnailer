#pragma once

extern "C" {
#include "util.h"
}
#include <Magick++.h>

// Lossy PNG compression. img is reused and can be set to NULL after call in
// case of error.
void compress_png(Magick::Image& img, struct Thumbnail* thumb,
    const struct CompressionRange quality);
