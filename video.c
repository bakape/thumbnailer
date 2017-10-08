#include "video.h"
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>

const size_t frameBufferSize = 1 << 17;

int extract_video_image(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream)
{
    int err;
    AVPacket pkt;
    AVFrame* frame = av_frame_alloc();

    for (;;) {
        err = av_read_frame(avfc, &pkt);
        if (err < 0) {
            break;
        }

        if (pkt.stream_index == stream) {
            err = avcodec_send_packet(avcc, &pkt);
            if (err < 0) {
                break;
            }

            err = avcodec_receive_frame(avcc, frame);
            if (err == AVERROR(EAGAIN)) {
                av_packet_unref(&pkt);
                continue;
            } else if (err < 0) {
                break;
            }

            err = encode_frame(img, frame);
            break;
        }
    }

    av_packet_unref(&pkt);
    av_frame_free(&frame);
    return err;
}

// Encode frame to RGBA image
static int encode_frame(struct Buffer* img, const AVFrame const* frame)
{
    struct SwsContext* ctx;
    uint8_t* dst_data[1];
    int dst_linesize[1];

    ctx = sws_getContext(frame->width, frame->height, frame->format,
        frame->width, frame->height, AV_PIX_FMT_RGBA,
        SWS_BICUBIC | SWS_ACCURATE_RND, NULL, NULL, NULL);
    if (!ctx) {
        return -1;
    }

    img->width = (unsigned long)frame->width;
    img->height = (unsigned long)frame->height;
    img->size = (size_t)av_image_get_buffer_size(
        AV_PIX_FMT_RGBA, frame->width, frame->height, 1);
    img->data = dst_data[0] = malloc(img->size); // RGB have one plane
    dst_linesize[0] = 4 * img->width; // RGBA stride

    sws_scale(ctx, (const uint8_t* const*)frame->data, frame->linesize, 0,
        frame->height, (uint8_t * const*)dst_data, dst_linesize);

    sws_freeContext(ctx);
    return 0;
}
