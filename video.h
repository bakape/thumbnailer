#pragma once
#include "ffmpeg.h"
#include "util.h"

int extract_video_image(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream);

static int encode_frame(struct Buffer* img, const AVFrame const* frame);
