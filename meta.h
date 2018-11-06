#pragma once
#include "ffmpeg.h"
#include <stddef.h>
#include <stdint.h>

struct Meta {
    char const* title;
    char const* artist;
};

struct Meta retrieve_meta(AVFormatContext* ctx);
