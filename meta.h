#pragma once
#include "ffmpeg.h"
#include <stddef.h>
#include <stdint.h>

struct Meta {
    const char const* title;
    const char const* artist;
};

struct Meta retrieve_meta(AVFormatContext* ctx);
