#include "thumbnailer.h"
#include <float.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>

/**
 * Potential thumbnail lookup filter to reduce the risk of an inappropriate
 * selection (such as a black frame) we could get with an absolute seek.
 *
 * Simplified version of algorithm by Vadim Zaliva <lord@crocodile.org>.
 * http://notbrainsurgery.livejournal.com/29773.html
 *
 * Adapted by Janis Petersons <bakape@gmail.com>
 */

#define HIST_SIZE (3 * 256)
#define MAX_FRAMES 10

// Compute sum-square deviation to estimate "closeness"
static double compute_error(
    const int hist[HIST_SIZE], const double median[HIST_SIZE])
{
    double sum_sq_err = 0;
    for (int i = 0; i < HIST_SIZE; i++) {
        const double err = median[i] - (double)hist[i];
        sum_sq_err += err * err;
    }
    return sum_sq_err;
}

// Select best frame based on RGB histograms
static AVFrame* select_best_frame(AVFrame* frames[], int size)
{
    // RGB color distribution histograms of the frames
    int hists[MAX_FRAMES][HIST_SIZE] = { 0 };

    // Compute each frame's histogram
    for (int frame_i = 0; frame_i < size; frame_i++) {
        const AVFrame* f = frames[frame_i];
        uint8_t* p = f->data[0];
        for (int j = 0; j < f->height; j++) {
            for (int i = 0; i < f->width; i++) {
                for (int k = 0; k < 3; k++) {
                    hists[frame_i][k * 256 + p[i * 3 + k]]++;
                }
            }
            p += f->linesize[0];
        }
    }

    // Average all histograms
    double average[HIST_SIZE] = { 0 };
    for (int j = 0; j < size; j++) {
        for (int i = 0; i < size; i++) {
            average[j] = (double)hists[i][j];
        }
        average[j] /= size;
    }

    // Find the frame closer to the average using the sum of squared errors
    double min_sq_err = DBL_MAX;
    int best_i = 0;
    for (int i = 0; i < size; i++) {
        const double sq_err = compute_error(hists[i], average);
        if (sq_err < min_sq_err) {
            best_i = i;
            min_sq_err = sq_err;
        }
    }
    return frames[best_i];
}

// Encode frame to RGBA image
static int encode_frame(
    struct Buffer* img, AVFrame* frame, const struct Dims thumb_dims)
{
    // Maintain aspect ratio
    double scale;
    if (frame->width >= frame->height) {
        scale = (double)(frame->width) / (double)(thumb_dims.width);
    } else {
        scale = (double)(frame->height) / (double)(thumb_dims.height);
    }
    img->width = (unsigned long)((double)frame->width / scale);
    img->height = (unsigned long)((double)frame->height / scale);

    // TODO: Use scaling that works better with alpha
    struct SwsContext* ctx = sws_getContext(frame->width, frame->height,
        frame->format, img->width, img->height, AV_PIX_FMT_RGBA,
        SWS_BICUBIC | SWS_ACCURATE_RND, NULL, NULL, NULL);
    if (!ctx) {
        return AVERROR(ENOMEM);
    }

    img->size = (size_t)av_image_get_buffer_size(
        AV_PIX_FMT_RGBA, frame->width, frame->height, 1);
    uint8_t* dst_data[1];
    img->data = dst_data[0] = malloc(img->size); // RGB have one plane
    int dst_linesize[1] = { 4 * img->width }; // RGBA stride

    sws_scale(ctx, (const uint8_t* const*)frame->data, frame->linesize, 0,
        frame->height, (uint8_t* const*)dst_data, dst_linesize);

    sws_freeContext(ctx);
    return 0;
}

int generate_thumbnail(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream, const struct Dims thumb_dims)
{
    int err = 0;
    int size = 0;
    int i = 0;
    AVPacket pkt;
    AVFrame* frames[MAX_FRAMES] = { NULL };
    AVFrame* next = NULL;

    // Read up to 10 frames in 10 frame intervals
    while (1) {
        err = av_read_frame(avfc, &pkt);
        switch (err) {
        case 0:
            break;
        case -1:
            // I don't know why, but this happens for some AVI and OGG files
            // mid-read. If some frames were actually read, just silence the
            // error and select from those.
            if (size) {
                err = 0;
            }
            goto end;
        default:
            goto end;
        }

        if (pkt.stream_index == stream) {
            err = avcodec_send_packet(avcc, &pkt);
            if (err < 0) {
                goto end;
            }

            if (!next) {
                next = av_frame_alloc();
                if (!next) {
                    err = AVERROR(ENOMEM);
                    goto end;
                }
            }
            err = avcodec_receive_frame(avcc, next);
            switch (err) {
            case 0:
                // Read only every 10th frame
                if (!(i++ % 10)) {
                    frames[size++] = next;
                    next = NULL;
                    if (size == MAX_FRAMES) {
                        goto end;
                    }
                } else {
                    av_frame_free(&next);
                    next = NULL;
                }
                break;
            case AVERROR(EAGAIN):
                break;
            default:
                goto end;
            }
        }
        av_packet_unref(&pkt);
    }

end:
    if (pkt.buf) {
        av_packet_unref(&pkt);
    }
    switch (err) {
    case AVERROR_EOF:
        err = 0;
    case 0:
        if (!size) {
            err = -1;
        } else {
            err = encode_frame(
                img, select_best_frame(frames, size), thumb_dims);
        }
        break;
    }
    for (int i = 0; i < size; i++) {
        av_frame_free(&frames[i]);
    }
    if (next) {
        av_frame_free(&next);
    }
    return err;
}
