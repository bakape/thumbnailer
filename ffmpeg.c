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

// Initialize am AVFormatContext with the buffered file
int create_context(AVFormatContext** ctx)
{
    unsigned char* buf = malloc(bufSize);
    AVFormatContext* c = *ctx;

    c->pb = avio_alloc_context(
        buf, bufSize, 0, c, readCallBack, NULL, seekCallBack);
    c->flags |= AVFMT_FLAG_CUSTOM_IO;

    int err = avformat_open_input(ctx, NULL, NULL, NULL);
    if (err < 0) {
        return err;
    }

    // Calls avcodec_open2 internally, so needs locking
    pthread_mutex_lock(&codecMu);
    err = avformat_find_stream_info(*ctx, NULL);
    pthread_mutex_unlock(&codecMu);
    return err;
}

void destroy(AVFormatContext* ctx)
{
    av_free(ctx->pb->buffer);
    ctx->pb->buffer = NULL;
    av_free(ctx->pb);
    av_free(ctx);
}

// Create a AVCodecContext of the desired media type
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
        return -1;
    }
    err = avcodec_parameters_to_context(*avcc, st->codecpar);
    if (err < 0) {
        return err;
    }

    // Not thread safe. Needs lock.
    pthread_mutex_lock(&codecMu);
    err = avcodec_open2(*avcc, codec, NULL);
    if (err < 0) {
        avcodec_free_context(avcc);
        *avcc = NULL;
    }
    pthread_mutex_unlock(&codecMu);
    return err;
}

// Format ffmpeg error code to string message
char* format_error(const int code)
{
    char* buf = malloc(1024);
    av_strerror(code, buf, 1024);
    return buf;
}
