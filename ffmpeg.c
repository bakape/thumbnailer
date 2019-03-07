#include "ffmpeg.h"

static const int bufSize = 1 << 12;

static pthread_mutex_t codecMu = PTHREAD_MUTEX_INITIALIZER;

void init(void)
{
#if LIBAVCODEC_VERSION_INT < AV_VERSION_INT(58, 9, 100)
    av_register_all();
#endif
#if LIBAVCODEC_VERSION_INT < AV_VERSION_INT(58, 10, 100)
    avcodec_register_all();
#endif
    av_log_set_level(16);
}

int create_context(AVFormatContext** ctx, const char* input_format)
{
    unsigned char* buf = malloc(bufSize);
    AVFormatContext* c = *ctx;

    c->pb = avio_alloc_context(
        buf, bufSize, 0, c, readCallBack, NULL, seekCallBack);
    c->flags |= AVFMT_FLAG_CUSTOM_IO | AVFMT_FLAG_DISCARD_CORRUPT;

    AVInputFormat* avif = NULL;
    if (input_format) {
        avif = av_find_input_format(input_format);
    }
    int err = avformat_open_input(ctx, NULL, avif, NULL);
    if (err < 0) {
        return err;
    }

    // Calls avcodec_open2 internally, so needs locking
    pthread_mutex_lock(&codecMu);
    err = avformat_find_stream_info(*ctx, NULL);
    pthread_mutex_unlock(&codecMu);
    return err;
}

int codec_context(AVCodecContext** avcc, int* stream, AVFormatContext* avfc,
    const enum AVMediaType type)
{
    int err;
    AVStream* st = NULL;
    AVCodec* codec = NULL;

    *stream = av_find_best_stream(avfc, type, -1, -1, NULL, 0);
    if (*stream < 0) {
        return *stream;
    }
    st = avfc->streams[*stream];

    // ffvp8/9 doesn't support alpha channel so force libvpx.
    switch (st->codecpar->codec_id) {
    case AV_CODEC_ID_VP8:
        codec = avcodec_find_decoder_by_name("libvpx");
        break;
    case AV_CODEC_ID_VP9:
        codec = avcodec_find_decoder_by_name("libvpx-vp9");
        break;
    }
    if (!codec) {
        codec = avcodec_find_decoder(st->codecpar->codec_id);
        if (!codec) {
            return -1;
        }
    }

    *avcc = avcodec_alloc_context3(codec);
    if (!*avcc) {
        goto end;
    }
    err = avcodec_parameters_to_context(*avcc, st->codecpar);
    if (err < 0) {
        goto end;
    }

    // Not thread safe. Needs lock.
    pthread_mutex_lock(&codecMu);
    err = avcodec_open2(*avcc, codec, NULL);
    pthread_mutex_unlock(&codecMu);

end:
    if (err < 0 && *avcc) {
        avcodec_free_context(avcc);
        *avcc = NULL;
    }
    return err;
}
