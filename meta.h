#pragma once
#include "ffmpeg.h"
#include <stddef.h>
#include <stdint.h>

struct Meta {
    char* title;
    char* artist;
};

struct Meta retrieve_meta(AVFormatContext* ctx);
