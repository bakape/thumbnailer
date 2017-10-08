#pragma once
#include "ffmpeg.h"

AVPacket retrieve_cover_art(AVFormatContext* ctx);
int find_cover_art(AVFormatContext* ctx);
